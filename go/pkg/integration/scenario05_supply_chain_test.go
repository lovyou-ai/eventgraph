package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/graph"
	"github.com/lovyou-ai/eventgraph/go/pkg/grammar"
	"github.com/lovyou-ai/eventgraph/go/pkg/protocol/egip"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// egipSystem represents one sovereign system in the supply chain.
type egipSystem struct {
	name      string
	identity  *egip.SystemIdentity
	store     store.Store
	graph     *graph.Graph
	grammar   *grammar.Grammar
	handler   *egip.Handler
	peers     *egip.PeerStore
	treaties  *egip.TreatyStore
	transport *routingTransport
	boot      event.Event
	convID    types.ConversationID
}

// routingTransport is a mock transport that delivers envelopes between systems
// via an in-memory network. When Send is called, it looks up the target system
// and calls HandleIncoming on its handler.
type routingTransport struct {
	mu       sync.Mutex
	network  *egipNetwork
	self     types.SystemURI
	sent     []*egip.Envelope
	incoming chan egip.IncomingEnvelope
}

func (t *routingTransport) Send(ctx context.Context, to types.SystemURI, env *egip.Envelope) (*egip.ReceiptPayload, error) {
	t.mu.Lock()
	t.sent = append(t.sent, env)
	t.mu.Unlock()

	// Route to the target system's handler.
	target, ok := t.network.systems[to.Value()]
	if !ok {
		return nil, &egip.TransportFailureError{To: to, Reason: "system not found"}
	}

	// Process incoming on the target (simulates network delivery).
	err := target.handler.HandleIncoming(ctx, env)
	if err != nil {
		return &egip.ReceiptPayload{
			EnvelopeID: env.ID,
			Status:     event.ReceiptStatusRejected,
			Reason:     types.Some(err.Error()),
		}, nil
	}

	return &egip.ReceiptPayload{
		EnvelopeID: env.ID,
		Status:     event.ReceiptStatusDelivered,
	}, nil
}

func (t *routingTransport) Listen(_ context.Context) <-chan egip.IncomingEnvelope {
	return t.incoming
}

// egipNetwork connects multiple systems for envelope routing.
type egipNetwork struct {
	systems map[string]*egipSystem
}

func newEGIPNetwork() *egipNetwork {
	return &egipNetwork{systems: make(map[string]*egipSystem)}
}

func (n *egipNetwork) addSystem(t *testing.T, name, uri string) *egipSystem {
	t.Helper()

	sysURI := types.MustSystemURI(uri)
	identity, err := egip.GenerateIdentity(sysURI)
	if err != nil {
		t.Fatalf("GenerateIdentity %s: %v", name, err)
	}

	s := store.NewInMemoryStore()
	g := graph.New(s, nil)
	if err := g.Start(); err != nil {
		t.Fatalf("Start %s: %v", name, err)
	}

	systemActor := types.MustActorID("actor_system0000000000000000000001")
	boot, err := g.Bootstrap(systemActor, signer)
	if err != nil {
		t.Fatalf("Bootstrap %s: %v", name, err)
	}

	transport := &routingTransport{
		network:  n,
		self:     sysURI,
		incoming: make(chan egip.IncomingEnvelope, 10),
	}

	peers := egip.NewPeerStore()
	treaties := egip.NewTreatyStore()
	handler := egip.NewHandler(identity, transport, peers, treaties)
	handler.ChainLength = func() (int, error) { return s.Count() }

	sys := &egipSystem{
		name:      name,
		identity:  identity,
		store:     s,
		graph:     g,
		grammar:   grammar.New(g),
		handler:   handler,
		peers:     peers,
		treaties:  treaties,
		transport: transport,
		boot:      boot,
		convID:    types.MustConversationID("conv_supply00000000000000000000001"),
	}

	n.systems[uri] = sys
	t.Cleanup(func() { g.Close() })
	return sys
}

// TestScenario05_SupplyChain exercises multi-system supply chain provenance
// across three sovereign event graphs communicating via EGIP.
//
// Farm → Factory → Retailer, with treaties, signed envelopes, CGERs,
// trust accumulation, and cross-system proof verification.
func TestScenario05_SupplyChain(t *testing.T) {
	ctx := context.Background()
	net := newEGIPNetwork()

	farm := net.addSystem(t, "Farm", "eg://farm.example.com")
	factory := net.addSystem(t, "Factory", "eg://factory.example.com")
	retail := net.addSystem(t, "Retailer", "eg://retail.example.com")

	// Track messages received by each system.
	var (
		factoryMessages []*egip.MessagePayloadContent
		retailMessages  []*egip.MessagePayloadContent
		farmMessages    []*egip.MessagePayloadContent
	)
	factory.handler.OnMessage = func(_ types.SystemURI, msg *egip.MessagePayloadContent) error {
		factoryMessages = append(factoryMessages, msg)
		return nil
	}
	retail.handler.OnMessage = func(_ types.SystemURI, msg *egip.MessagePayloadContent) error {
		retailMessages = append(retailMessages, msg)
		return nil
	}
	farm.handler.OnMessage = func(_ types.SystemURI, msg *egip.MessagePayloadContent) error {
		farmMessages = append(farmMessages, msg)
		return nil
	}

	// ----------------------------------------------------------------
	// Step 1: HELLO handshakes — Farm↔Factory, Factory↔Retailer
	// ----------------------------------------------------------------
	t.Run("HELLO handshakes", func(t *testing.T) {
		// Farm → Factory
		if err := farm.handler.Hello(ctx, factory.identity.SystemURI()); err != nil {
			t.Fatalf("Farm→Factory HELLO: %v", err)
		}
		// Factory → Farm (reciprocal)
		if err := factory.handler.Hello(ctx, farm.identity.SystemURI()); err != nil {
			t.Fatalf("Factory→Farm HELLO: %v", err)
		}
		// Factory → Retailer
		if err := factory.handler.Hello(ctx, retail.identity.SystemURI()); err != nil {
			t.Fatalf("Factory→Retailer HELLO: %v", err)
		}
		// Retailer → Factory (reciprocal)
		if err := retail.handler.Hello(ctx, factory.identity.SystemURI()); err != nil {
			t.Fatalf("Retailer→Factory HELLO: %v", err)
		}

		// Verify peers are registered.
		if _, ok := farm.peers.Get(factory.identity.SystemURI()); !ok {
			t.Error("Farm should know Factory after HELLO")
		}
		if _, ok := factory.peers.Get(farm.identity.SystemURI()); !ok {
			t.Error("Factory should know Farm after HELLO")
		}
		if _, ok := factory.peers.Get(retail.identity.SystemURI()); !ok {
			t.Error("Factory should know Retailer after HELLO")
		}
		if _, ok := retail.peers.Get(factory.identity.SystemURI()); !ok {
			t.Error("Retailer should know Factory after HELLO")
		}

		// Non-transitive: Retailer does NOT know Farm.
		if _, ok := retail.peers.Get(farm.identity.SystemURI()); ok {
			t.Error("Retailer should NOT know Farm (non-transitive)")
		}
	})

	// ----------------------------------------------------------------
	// Step 2: Treaty between Farm and Factory
	// ----------------------------------------------------------------
	treatyAB := types.MustTreatyID("00000001-0001-4001-8001-000000000001")

	t.Run("Treaty Farm-Factory", func(t *testing.T) {
		terms := []egip.TreatyTerm{
			{
				Scope:     types.MustDomainScope("produce_supply"),
				Policy:    "Farm provides organic produce with harvest records. Factory provides processing records.",
				Symmetric: false,
			},
		}

		// Farm proposes treaty — store locally and send to Factory.
		farm.treaties.Put(egip.NewTreaty(treatyAB,
			farm.identity.SystemURI(), factory.identity.SystemURI(), terms))

		proposeEnv := makeTreatyEnvelope(t, farm, factory.identity.SystemURI(),
			treatyAB, event.TreatyActionPropose, terms)
		if err := factory.handler.HandleIncoming(ctx, proposeEnv); err != nil {
			t.Fatalf("Factory handle treaty propose: %v", err)
		}

		// Factory accepts treaty — send accept back to Farm.
		acceptEnv := makeTreatyEnvelope(t, factory, farm.identity.SystemURI(),
			treatyAB, event.TreatyActionAccept, nil)

		// Accept on Factory's local copy.
		if err := factory.treaties.Apply(treatyAB, func(tr *egip.Treaty) error {
			return tr.ApplyAction(event.TreatyActionAccept)
		}); err != nil {
			t.Fatalf("Factory local accept: %v", err)
		}

		// Farm receives the accept.
		if err := farm.handler.HandleIncoming(ctx, acceptEnv); err != nil {
			t.Fatalf("Farm handle treaty accept: %v", err)
		}

		// Verify treaty is active on both sides.
		ft, ok := factory.treaties.Get(treatyAB)
		if !ok {
			t.Fatal("Factory should have treaty")
		}
		if ft.Status != event.TreatyStatusActive {
			t.Errorf("Factory treaty status = %s, want Active", ft.Status)
		}

		fft, ok := farm.treaties.Get(treatyAB)
		if !ok {
			t.Fatal("Farm should have treaty after accept")
		}
		if fft.Status != event.TreatyStatusActive {
			t.Errorf("Farm treaty status = %s, want Active", fft.Status)
		}
	})

	// ----------------------------------------------------------------
	// Step 3: Treaty between Factory and Retailer
	// ----------------------------------------------------------------
	treatyBC := types.MustTreatyID("00000002-0002-4002-8002-000000000002")

	t.Run("Treaty Factory-Retailer", func(t *testing.T) {
		terms := []egip.TreatyTerm{
			{
				Scope:     types.MustDomainScope("product_supply"),
				Policy:    "Factory provides manufactured products with full provenance. Retailer provides sales records.",
				Symmetric: false,
			},
		}

		// Factory proposes — store locally and send.
		factory.treaties.Put(egip.NewTreaty(treatyBC,
			factory.identity.SystemURI(), retail.identity.SystemURI(), terms))

		proposeEnv := makeTreatyEnvelope(t, factory, retail.identity.SystemURI(),
			treatyBC, event.TreatyActionPropose, terms)
		if err := retail.handler.HandleIncoming(ctx, proposeEnv); err != nil {
			t.Fatalf("Retailer handle treaty propose: %v", err)
		}

		// Retailer accepts — apply locally and send back.
		if err := retail.treaties.Apply(treatyBC, func(tr *egip.Treaty) error {
			return tr.ApplyAction(event.TreatyActionAccept)
		}); err != nil {
			t.Fatalf("Retailer local accept: %v", err)
		}

		acceptEnv := makeTreatyEnvelope(t, retail, factory.identity.SystemURI(),
			treatyBC, event.TreatyActionAccept, nil)
		if err := factory.handler.HandleIncoming(ctx, acceptEnv); err != nil {
			t.Fatalf("Factory handle treaty accept: %v", err)
		}

		rt, ok := retail.treaties.Get(treatyBC)
		if !ok {
			t.Fatal("Retailer should have treaty")
		}
		if rt.Status != event.TreatyStatusActive {
			t.Errorf("Retailer treaty status = %s, want Active", rt.Status)
		}
	})

	// ----------------------------------------------------------------
	// Step 4: Farm records harvest on its local graph
	// ----------------------------------------------------------------
	farmActor := registerActorOn(t, farm, "farmer_emma", 1, event.ActorTypeHuman)

	harvest, err := farm.grammar.Emit(ctx, farmActor.ID(),
		"harvest: 500kg organic tomatoes, lot #TOM-2026-0308, field B3, method: organic no pesticides",
		farm.convID, []types.EventID{farm.boot.ID()}, signer)
	if err != nil {
		t.Fatalf("Emit harvest: %v", err)
	}

	// ----------------------------------------------------------------
	// Step 5: Farm sends harvest record to Factory via EGIP MESSAGE with CGER
	// ----------------------------------------------------------------
	t.Run("Farm sends harvest to Factory", func(t *testing.T) {
		msgEnv := makeMessageEnvelope(t, farm, factory.identity.SystemURI(),
			"produce.harvested",
			`{"product":"Organic Tomatoes","quantity":500,"location":"Farm A, Plot 7"}`,
			[]egip.CGER{
				{
					LocalEventID:  harvest.ID(),
					RemoteSystem:  farm.identity.SystemURI(),
					RemoteEventID: harvest.ID().Value(),
					RemoteHash:    harvest.Hash(),
					Relationship:  event.CGERRelationshipCausedBy,
				},
			})
		receipt, err := farm.transport.Send(ctx, factory.identity.SystemURI(), msgEnv)
		if err != nil {
			t.Fatalf("Send harvest message: %v", err)
		}
		if receipt.Status != event.ReceiptStatusDelivered {
			t.Errorf("receipt status = %s, want Delivered", receipt.Status)
		}

		if len(factoryMessages) != 1 {
			t.Fatalf("Factory should have received 1 message, got %d", len(factoryMessages))
		}
		if len(factoryMessages[0].CGERs) != 1 {
			t.Fatalf("Message should have 1 CGER, got %d", len(factoryMessages[0].CGERs))
		}
		if factoryMessages[0].CGERs[0].RemoteEventID != harvest.ID().Value() {
			t.Error("CGER should reference harvest event ID")
		}
	})

	// ----------------------------------------------------------------
	// Step 6: Factory records receipt, QA, and manufacturing on its graph
	// ----------------------------------------------------------------
	factoryMgr := registerActorOn(t, factory, "factory_mgr", 2, event.ActorTypeHuman)
	qaAgent := registerActorOn(t, factory, "qa_agent", 3, event.ActorTypeAI)

	received, err := factory.grammar.Derive(ctx, factoryMgr.ID(),
		"received: 500kg tomatoes from farm.example.com, lot #TOM-2026-0308, CGER: "+harvest.ID().Value(),
		factory.boot.ID(), factory.convID, signer)
	if err != nil {
		t.Fatalf("Derive received: %v", err)
	}

	inspection, err := factory.grammar.Derive(ctx, qaAgent.ID(),
		"qa inspection: pesticide-free verified, freshness grade A, confidence 0.92",
		received.ID(), factory.convID, signer)
	if err != nil {
		t.Fatalf("Derive inspection: %v", err)
	}

	product, err := factory.grammar.Derive(ctx, factoryMgr.ID(),
		"manufactured: 200 jars organic tomato sauce, batch #SAU-2026-0308",
		inspection.ID(), factory.convID, signer)
	if err != nil {
		t.Fatalf("Derive product: %v", err)
	}

	// ----------------------------------------------------------------
	// Step 7: Factory endorses farm produce quality (EGIP message back to Farm)
	// ----------------------------------------------------------------
	t.Run("Factory endorses Farm quality", func(t *testing.T) {
		endorseEnv := makeMessageEnvelope(t, factory, farm.identity.SystemURI(),
			"endorsement",
			`{"endorser":"eg://factory.example.com","subject":"`+harvest.ID().Value()+`","quality":0.9,"domain":"produce_quality"}`,
			nil)
		receipt, err := factory.transport.Send(ctx, farm.identity.SystemURI(), endorseEnv)
		if err != nil {
			t.Fatalf("Send endorsement: %v", err)
		}
		if receipt.Status != event.ReceiptStatusDelivered {
			t.Errorf("endorsement receipt status = %s, want Delivered", receipt.Status)
		}
		if len(farmMessages) != 1 {
			t.Fatalf("Farm should have received 1 message, got %d", len(farmMessages))
		}
	})

	// ----------------------------------------------------------------
	// Step 8: Factory sends product to Retailer with chained CGERs
	// ----------------------------------------------------------------
	t.Run("Factory sends product to Retailer with CGERs", func(t *testing.T) {
		msgEnv := makeMessageEnvelope(t, factory, retail.identity.SystemURI(),
			"product.manufactured",
			`{"product":"Organic Tomato Sauce","batch_id":"SAU-2026-0308"}`,
			[]egip.CGER{
				{
					LocalEventID:  product.ID(),
					RemoteSystem:  factory.identity.SystemURI(),
					RemoteEventID: product.ID().Value(),
					RemoteHash:    product.Hash(),
					Relationship:  event.CGERRelationshipCausedBy,
				},
				{
					LocalEventID:  harvest.ID(),
					RemoteSystem:  farm.identity.SystemURI(),
					RemoteEventID: harvest.ID().Value(),
					RemoteHash:    harvest.Hash(),
					Relationship:  event.CGERRelationshipReferences,
				},
			})
		receipt, err := factory.transport.Send(ctx, retail.identity.SystemURI(), msgEnv)
		if err != nil {
			t.Fatalf("Send product message: %v", err)
		}
		if receipt.Status != event.ReceiptStatusDelivered {
			t.Errorf("receipt status = %s, want Delivered", receipt.Status)
		}

		if len(retailMessages) != 1 {
			t.Fatalf("Retailer should have received 1 message, got %d", len(retailMessages))
		}
		if len(retailMessages[0].CGERs) != 2 {
			t.Fatalf("Retailer message should have 2 CGERs (factory + farm), got %d", len(retailMessages[0].CGERs))
		}

		// Verify transitive provenance — CGERs reference both factory and farm.
		var factoryCGER, farmCGER bool
		for _, cger := range retailMessages[0].CGERs {
			if cger.RemoteSystem.Value() == "eg://factory.example.com" {
				factoryCGER = true
			}
			if cger.RemoteSystem.Value() == "eg://farm.example.com" {
				farmCGER = true
			}
		}
		if !factoryCGER {
			t.Error("CGERs should reference factory system")
		}
		if !farmCGER {
			t.Error("CGERs should reference farm system (transitive provenance)")
		}
	})

	// ----------------------------------------------------------------
	// Step 9: Retailer records product listing on its graph
	// ----------------------------------------------------------------
	retailActor := registerActorOn(t, retail, "retailer_frank", 4, event.ActorTypeHuman)

	listed, err := retail.grammar.Derive(ctx, retailActor.ID(),
		"product listed: organic tomato sauce, batch #SAU-2026-0308, price $8.99, provenance: farm→factory→retail",
		retail.boot.ID(), retail.convID, signer)
	if err != nil {
		t.Fatalf("Derive listed: %v", err)
	}

	// ----------------------------------------------------------------
	// Step 10: Proof request — Retailer requests event existence from Factory
	// ----------------------------------------------------------------
	t.Run("Proof request from Retailer to Factory", func(t *testing.T) {
		proofEnv := makeProofEnvelope(t, retail, factory.identity.SystemURI(),
			&egip.ProofPayload{
				ProofType: event.ProofTypeChainSummary,
				Data: &egip.ChainSummaryProof{
					Length:    3,
					HeadHash: product.Hash(),
					Timestamp: time.Now(),
				},
			})
		receipt, err := retail.transport.Send(ctx, factory.identity.SystemURI(), proofEnv)
		if err != nil {
			t.Fatalf("Send proof request: %v", err)
		}
		if receipt.Status != event.ReceiptStatusDelivered {
			t.Errorf("proof receipt status = %s, want Delivered", receipt.Status)
		}
	})

	// ----------------------------------------------------------------
	// Assertions
	// ----------------------------------------------------------------

	t.Run("Non-transitive trust", func(t *testing.T) {
		// Retailer trusts Factory (via HELLO + messages).
		retailFactoryPeer, ok := retail.peers.Get(factory.identity.SystemURI())
		if !ok {
			t.Fatal("Retailer should know Factory")
		}
		if retailFactoryPeer.Trust.Value() <= 0 {
			t.Error("Retailer should have positive trust in Factory")
		}

		// But Retailer does NOT trust Farm — trust is non-transitive.
		_, farmKnown := retail.peers.Get(farm.identity.SystemURI())
		if farmKnown {
			t.Error("Retailer should NOT know Farm directly (non-transitive)")
		}
	})

	t.Run("Trust accumulation from messages", func(t *testing.T) {
		// Factory's trust in Farm should have increased from HELLO + message receipt.
		factoryFarmPeer, ok := factory.peers.Get(farm.identity.SystemURI())
		if !ok {
			t.Fatal("Factory should know Farm")
		}
		if factoryFarmPeer.Trust.Value() <= 0 {
			t.Error("Factory should have positive trust in Farm after message exchange")
		}
	})

	t.Run("Treaty governance", func(t *testing.T) {
		// Both treaties should be active.
		abTreaty, ok := factory.treaties.Get(treatyAB)
		if !ok {
			t.Fatal("Factory should have Farm treaty")
		}
		if abTreaty.Status != event.TreatyStatusActive {
			t.Errorf("Farm-Factory treaty = %s, want Active", abTreaty.Status)
		}
		if len(abTreaty.Terms) != 1 {
			t.Errorf("Farm-Factory treaty terms = %d, want 1", len(abTreaty.Terms))
		}
		if abTreaty.Terms[0].Scope.Value() != "produce_supply" {
			t.Errorf("treaty scope = %s, want produce_supply", abTreaty.Terms[0].Scope.Value())
		}

		bcTreaty, ok := retail.treaties.Get(treatyBC)
		if !ok {
			t.Fatal("Retailer should have Factory treaty")
		}
		if bcTreaty.Status != event.TreatyStatusActive {
			t.Errorf("Factory-Retailer treaty = %s, want Active", bcTreaty.Status)
		}
	})

	t.Run("Each system has independent hash chain", func(t *testing.T) {
		for _, sys := range []*egipSystem{farm, factory, retail} {
			result, err := sys.store.VerifyChain()
			if err != nil {
				t.Fatalf("%s VerifyChain: %v", sys.name, err)
			}
			if !result.Valid {
				t.Errorf("%s chain integrity broken", sys.name)
			}
		}
	})

	t.Run("Event counts per system", func(t *testing.T) {
		// Farm: bootstrap(1) + harvest(1) = 2
		farmCount, _ := farm.store.Count()
		if farmCount != 2 {
			t.Errorf("Farm events = %d, want 2", farmCount)
		}

		// Factory: bootstrap(1) + received(1) + inspection(1) + product(1) = 4
		factoryCount, _ := factory.store.Count()
		if factoryCount != 4 {
			t.Errorf("Factory events = %d, want 4", factoryCount)
		}

		// Retailer: bootstrap(1) + listed(1) = 2
		retailCount, _ := retail.store.Count()
		if retailCount != 2 {
			t.Errorf("Retailer events = %d, want 2", retailCount)
		}
	})

	t.Run("Local provenance on Factory graph", func(t *testing.T) {
		q, err := factory.graph.Query()
		if err != nil {
			t.Fatalf("Query: %v", err)
		}
		ancestors, err := q.Ancestors(product.ID(), 10)
		if err != nil {
			t.Fatalf("Ancestors: %v", err)
		}

		if !containsEvent(ancestors, inspection.ID()) {
			t.Error("product should trace to inspection")
		}
		if !containsEvent(ancestors, received.ID()) {
			t.Error("product should trace to received")
		}
	})

	t.Run("CGER hash integrity", func(t *testing.T) {
		// The CGERs sent to Retailer should contain verifiable hashes.
		for _, cger := range retailMessages[0].CGERs {
			if cger.RemoteHash == (types.Hash{}) {
				t.Errorf("CGER for %s has empty hash", cger.RemoteSystem.Value())
			}
			if cger.RemoteEventID == "" {
				t.Errorf("CGER for %s has empty event ID", cger.RemoteSystem.Value())
			}
		}
	})

	t.Run("Signed envelope verification", func(t *testing.T) {
		// Every envelope sent by Farm should be verifiable with Farm's public key.
		farm.transport.mu.Lock()
		for _, env := range farm.transport.sent {
			valid, err := farm.identity.Verify(farm.identity.PublicKey(), mustCanonical(t, env), env.Signature)
			if err != nil {
				t.Fatalf("Verify Farm envelope: %v", err)
			}
			if !valid {
				t.Error("Farm envelope signature should be valid")
			}
		}
		farm.transport.mu.Unlock()
	})

	_ = listed // used in Derive above
}

// --- Helpers for EGIP envelope construction ---

func registerActorOn(t *testing.T, sys *egipSystem, name string, pkByte byte, actorType event.ActorType) interface{ ID() types.ActorID } {
	t.Helper()
	// Systems without an actor store use the graph directly.
	// Since we passed nil for actor store, use the farm/factory/retail graph's store.
	// We just need actor IDs — we don't actually need IActorStore for this test.
	return &simpleActor{id: types.MustActorID("actor_" + padTo(name, 30))}
}

type simpleActor struct{ id types.ActorID }

func (a *simpleActor) ID() types.ActorID { return a.id }

func padTo(s string, length int) string {
	for len(s) < length {
		s += "0"
	}
	return s[:length]
}

func makeTreatyEnvelope(t *testing.T, from *egipSystem, to types.SystemURI, treatyID types.TreatyID, action event.TreatyAction, terms []egip.TreatyTerm) *egip.Envelope {
	t.Helper()

	env := &egip.Envelope{
		ProtocolVersion: egip.CurrentProtocolVersion,
		ID:              mustEnvelopeID(t),
		From:            from.identity.SystemURI(),
		To:              to,
		Type:            event.MessageTypeTreaty,
		Payload: egip.TreatyPayload{
			TreatyID: treatyID,
			Action:   action,
			Terms:    terms,
			Reason:   types.None[string](),
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, err := egip.SignEnvelope(env, from.identity)
	if err != nil {
		t.Fatalf("Sign treaty envelope: %v", err)
	}
	return signed
}

func makeMessageEnvelope(t *testing.T, from *egipSystem, to types.SystemURI, contentType, contentJSON string, cgers []egip.CGER) *egip.Envelope {
	t.Helper()

	env := &egip.Envelope{
		ProtocolVersion: egip.CurrentProtocolVersion,
		ID:              mustEnvelopeID(t),
		From:            from.identity.SystemURI(),
		To:              to,
		Type:            event.MessageTypeMessage,
		Payload: egip.MessagePayloadContent{
			ContentJSON:    json.RawMessage(contentJSON),
			ContentType:    types.MustEventType(contentType),
			ConversationID: types.None[types.ConversationID](),
			CGERs:          cgers,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, err := egip.SignEnvelope(env, from.identity)
	if err != nil {
		t.Fatalf("Sign message envelope: %v", err)
	}
	return signed
}

func makeProofEnvelope(t *testing.T, from *egipSystem, to types.SystemURI, proof *egip.ProofPayload) *egip.Envelope {
	t.Helper()

	env := &egip.Envelope{
		ProtocolVersion: egip.CurrentProtocolVersion,
		ID:              mustEnvelopeID(t),
		From:            from.identity.SystemURI(),
		To:              to,
		Type:            event.MessageTypeProof,
		Payload:         *proof,
		Timestamp:       time.Now(),
		InReplyTo:       types.None[types.EnvelopeID](),
	}

	signed, err := egip.SignEnvelope(env, from.identity)
	if err != nil {
		t.Fatalf("Sign proof envelope: %v", err)
	}
	return signed
}

var envelopeCounter atomic.Int64

func mustEnvelopeID(t *testing.T) types.EnvelopeID {
	t.Helper()
	n := envelopeCounter.Add(1)
	id := fmt.Sprintf("00000000-0000-4000-8000-%012d", n)
	return types.MustEnvelopeID(id)
}

func mustCanonical(t *testing.T, env *egip.Envelope) []byte {
	t.Helper()
	canonical, err := env.CanonicalForm()
	if err != nil {
		t.Fatalf("CanonicalForm: %v", err)
	}
	return []byte(canonical)
}
