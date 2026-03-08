// Multi-system EGIP example — two sovereign event graphs communicating via
// signed envelopes, treaties, and cross-graph event references (CGERs).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/grammar"
	"github.com/lovyou-ai/eventgraph/go/pkg/graph"
	"github.com/lovyou-ai/eventgraph/go/pkg/protocol/egip"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

type noopSigner struct{}

func (noopSigner) Sign(data []byte) (types.Signature, error) {
	return types.MustSignature(make([]byte, 64)), nil
}

var sign = noopSigner{}

// system holds all components for one sovereign event graph.
type system struct {
	name     string
	identity *egip.SystemIdentity
	store    store.Store
	graph    *graph.Graph
	grammar  *grammar.Grammar
	handler  *egip.Handler
	peers    *egip.PeerStore
	treaties *egip.TreatyStore
}

func main() {
	ctx := context.Background()

	// --- Create two sovereign systems ---
	alpha := newSystem("Alpha", "eg://alpha.example.com")
	defer alpha.graph.Close()

	beta := newSystem("Beta", "eg://beta.example.com")
	defer beta.graph.Close()

	fmt.Println("=== Two Sovereign Event Graphs ===")
	fmt.Printf("System A: %s (%s)\n", alpha.name, alpha.identity.SystemURI().Value())
	fmt.Printf("System B: %s (%s)\n\n", beta.name, beta.identity.SystemURI().Value())

	// --- Step 1: HELLO handshake ---
	fmt.Println("--- Step 1: HELLO Handshake ---")

	// Simulate HELLO by directly processing envelopes (no network needed).
	helloA := makeHello(alpha)
	if err := beta.handler.HandleIncoming(ctx, helloA); err != nil {
		log.Fatal(err)
	}
	helloB := makeHello(beta)
	if err := alpha.handler.HandleIncoming(ctx, helloB); err != nil {
		log.Fatal(err)
	}

	peerA, _ := beta.peers.Get(alpha.identity.SystemURI())
	peerB, _ := alpha.peers.Get(beta.identity.SystemURI())
	fmt.Printf("Alpha knows Beta: version=%d, trust=%.2f\n", peerB.NegotiatedVersion, peerB.Trust.Value())
	fmt.Printf("Beta knows Alpha: version=%d, trust=%.2f\n\n", peerA.NegotiatedVersion, peerA.Trust.Value())

	// --- Step 2: Establish a treaty ---
	fmt.Println("--- Step 2: Treaty ---")

	treatyID := types.MustTreatyID("a0000001-b002-4003-8004-c00000000005")
	terms := []egip.TreatyTerm{{
		Scope:     types.MustDomainScope("data_sharing"),
		Policy:    "Both systems share event summaries and integrity proofs.",
		Symmetric: true,
	}}

	// Alpha proposes.
	alpha.treaties.Put(egip.NewTreaty(treatyID, alpha.identity.SystemURI(), beta.identity.SystemURI(), terms))
	proposeTreaty := makeTreaty(alpha, beta.identity.SystemURI(), treatyID, event.TreatyActionPropose, terms)
	if err := beta.handler.HandleIncoming(ctx, proposeTreaty); err != nil {
		log.Fatal(err)
	}

	// Beta accepts.
	beta.treaties.Apply(treatyID, func(t *egip.Treaty) error { return t.ApplyAction(event.TreatyActionAccept) })
	acceptTreaty := makeTreaty(beta, alpha.identity.SystemURI(), treatyID, event.TreatyActionAccept, nil)
	if err := alpha.handler.HandleIncoming(ctx, acceptTreaty); err != nil {
		log.Fatal(err)
	}

	at, _ := alpha.treaties.Get(treatyID)
	fmt.Printf("Treaty %s: status=%s, scope=%s\n\n", treatyID.Value()[:8]+"…", at.Status, at.Terms[0].Scope.Value())

	// --- Step 3: Record events locally ---
	fmt.Println("--- Step 3: Local Events ---")

	conv := types.MustConversationID("conv_cross_system")
	alphaActor := types.MustActorID("actor_alpha_main")
	betaActor := types.MustActorID("actor_beta_main")

	boot, _ := alpha.store.Head()
	bootEvt := boot.Unwrap()

	ev1, err := alpha.grammar.Emit(ctx, alphaActor, "Research findings on topic X",
		conv, []types.EventID{bootEvt.ID()}, sign)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Alpha recorded: %s (type: %s)\n", ev1.ID().Value()[:13]+"…", ev1.Type().Value())

	bootB, _ := beta.store.Head()
	bootBEvt := bootB.Unwrap()

	ev2, err := beta.grammar.Emit(ctx, betaActor, "Related analysis from our perspective",
		conv, []types.EventID{bootBEvt.ID()}, sign)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Beta recorded:  %s (type: %s)\n\n", ev2.ID().Value()[:13]+"…", ev2.Type().Value())

	// --- Step 4: Send a message with CGER ---
	fmt.Println("--- Step 4: Cross-Graph Message (CGER) ---")

	var betaReceived *egip.MessagePayloadContent
	beta.handler.OnMessage = func(_ types.SystemURI, msg *egip.MessagePayloadContent) error {
		betaReceived = msg
		return nil
	}

	msgEnv := makeMessage(alpha, beta.identity.SystemURI(), "research.findings",
		`{"topic":"X","confidence":0.92}`,
		[]egip.CGER{{
			LocalEventID:  ev1.ID(),
			RemoteSystem:  alpha.identity.SystemURI(),
			RemoteEventID: ev1.ID().Value(),
			RemoteHash:    ev1.Hash(),
			Relationship:  event.CGERRelationshipCausedBy,
		}})
	if err := beta.handler.HandleIncoming(ctx, msgEnv); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Beta received message: type=%s, CGERs=%d\n", betaReceived.ContentType.Value(), len(betaReceived.CGERs))
	fmt.Printf("CGER references: system=%s, event=%s…\n\n",
		betaReceived.CGERs[0].RemoteSystem.Value(), betaReceived.CGERs[0].RemoteEventID[:13])

	// --- Step 5: Send a proof ---
	fmt.Println("--- Step 5: Chain Summary Proof ---")

	headA, _ := alpha.store.Head()
	headEvt := headA.Unwrap()
	countA, _ := alpha.store.Count()

	proofEnv := makeProof(alpha, beta.identity.SystemURI(), &egip.ProofPayload{
		ProofType: event.ProofTypeChainSummary,
		Data: &egip.ChainSummaryProof{
			Length:    countA,
			HeadHash: headEvt.Hash(),
			Timestamp: time.Now(),
		},
	})
	if err := beta.handler.HandleIncoming(ctx, proofEnv); err != nil {
		log.Fatal(err)
	}

	// Check trust increased after valid proof.
	peerAAfter, _ := beta.peers.Get(alpha.identity.SystemURI())
	fmt.Printf("Alpha trust on Beta: %.4f (was %.4f before proof)\n\n", peerAAfter.Trust.Value(), peerA.Trust.Value())

	// --- Summary ---
	fmt.Println("=== Summary ===")
	resultA, _ := alpha.store.VerifyChain()
	resultB, _ := beta.store.VerifyChain()
	cntA, _ := alpha.store.Count()
	cntB, _ := beta.store.Count()
	fmt.Printf("Alpha: %d events, chain valid=%v\n", cntA, resultA.Valid)
	fmt.Printf("Beta:  %d events, chain valid=%v\n", cntB, resultB.Valid)
	fmt.Printf("Treaty: %s\n", at.Status)
	fmt.Printf("Trust is asymmetric, non-transitive, and earned through interaction.\n")
}

// --- System setup ---

func newSystem(name, uri string) *system {
	sysURI := types.MustSystemURI(uri)
	identity, err := egip.GenerateIdentity(sysURI)
	if err != nil {
		log.Fatalf("GenerateIdentity %s: %v", name, err)
	}

	s := store.NewInMemoryStore()
	g := graph.New(s, nil)
	if err := g.Start(); err != nil {
		log.Fatalf("Start %s: %v", name, err)
	}

	systemActor := types.MustActorID("actor_system")
	if _, err := g.Bootstrap(systemActor, sign); err != nil {
		log.Fatalf("Bootstrap %s: %v", name, err)
	}

	peers := egip.NewPeerStore()
	treaties := egip.NewTreatyStore()
	// No real transport — envelopes are passed directly via HandleIncoming.
	handler := egip.NewHandler(identity, &nullTransport{}, peers, treaties)
	handler.ChainLength = func() (int, error) { return s.Count() }

	return &system{
		name:     name,
		identity: identity,
		store:    s,
		graph:    g,
		grammar:  grammar.New(g),
		handler:  handler,
		peers:    peers,
		treaties: treaties,
	}
}

// nullTransport is used when envelopes are passed directly (no network).
type nullTransport struct{}

func (nullTransport) Send(context.Context, types.SystemURI, *egip.Envelope) (*egip.ReceiptPayload, error) {
	return nil, nil
}
func (nullTransport) Listen(context.Context) <-chan egip.IncomingEnvelope {
	return make(chan egip.IncomingEnvelope)
}

// --- Envelope builders ---

var envCounter int

func nextEnvID() types.EnvelopeID {
	envCounter++
	return types.MustEnvelopeID(fmt.Sprintf("a0000000-0000-4000-8000-%012d", envCounter))
}

func makeHello(sys *system) *egip.Envelope {
	count, _ := sys.store.Count()
	env := &egip.Envelope{
		ProtocolVersion: egip.CurrentProtocolVersion,
		ID:              nextEnvID(),
		From:            sys.identity.SystemURI(),
		To:              sys.identity.SystemURI(), // overridden by target
		Type:            event.MessageTypeHello,
		Payload: egip.HelloPayload{
			SystemURI:        sys.identity.SystemURI(),
			PublicKey:        sys.identity.PublicKey(),
			ProtocolVersions: []int{egip.CurrentProtocolVersion},
			Capabilities:     []string{"treaty", "proof"},
			ChainLength:      count,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}
	signed, err := egip.SignEnvelope(env, sys.identity)
	if err != nil {
		log.Fatalf("sign hello: %v", err)
	}
	return signed
}

func makeTreaty(sys *system, to types.SystemURI, treatyID types.TreatyID, action event.TreatyAction, terms []egip.TreatyTerm) *egip.Envelope {
	env := &egip.Envelope{
		ProtocolVersion: egip.CurrentProtocolVersion,
		ID:              nextEnvID(),
		From:            sys.identity.SystemURI(),
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
	signed, err := egip.SignEnvelope(env, sys.identity)
	if err != nil {
		log.Fatalf("sign treaty: %v", err)
	}
	return signed
}

func makeMessage(sys *system, to types.SystemURI, contentType, contentJSON string, cgers []egip.CGER) *egip.Envelope {
	env := &egip.Envelope{
		ProtocolVersion: egip.CurrentProtocolVersion,
		ID:              nextEnvID(),
		From:            sys.identity.SystemURI(),
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
	signed, err := egip.SignEnvelope(env, sys.identity)
	if err != nil {
		log.Fatalf("sign message: %v", err)
	}
	return signed
}

func makeProof(sys *system, to types.SystemURI, proof *egip.ProofPayload) *egip.Envelope {
	env := &egip.Envelope{
		ProtocolVersion: egip.CurrentProtocolVersion,
		ID:              nextEnvID(),
		From:            sys.identity.SystemURI(),
		To:              to,
		Type:            event.MessageTypeProof,
		Payload:         *proof,
		Timestamp:       time.Now(),
		InReplyTo:       types.None[types.EnvelopeID](),
	}
	signed, err := egip.SignEnvelope(env, sys.identity)
	if err != nil {
		log.Fatalf("sign proof: %v", err)
	}
	return signed
}
