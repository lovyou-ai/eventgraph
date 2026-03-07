package authority_test

import (
	"context"
	"testing"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/authority"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/trust"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

type chainTestSigner struct{}

func (chainTestSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data)
	return types.MustSignature(sig), nil
}

type headFromStore struct{ s store.Store }

func (h headFromStore) Head() (types.Option[event.Event], error) { return h.s.Head() }

func bootstrapStore(t *testing.T) store.Store {
	t.Helper()
	s := store.NewInMemoryStore()
	registry := event.DefaultRegistry()
	factory := event.NewBootstrapFactory(registry)
	ev, err := factory.Init(
		types.MustActorID("actor_00000000000000000000000000000001"),
		chainTestSigner{},
	)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	s.Append(ev)
	return s
}

func addAuthorityEdge(t *testing.T, s store.Store, from, to types.ActorID, weight float64, scope types.DomainScope, expiresAt types.Option[types.Timestamp]) types.EventID {
	t.Helper()
	registry := event.DefaultRegistry()
	factory := event.NewEventFactory(registry)

	head, _ := s.Head()
	var causes []types.EventID
	if head.IsSome() {
		causes = []types.EventID{head.Unwrap().ID()}
	}

	edgeWeight := types.MustWeight(weight*2 - 1)
	ev, err := factory.Create(
		event.EventTypeEdgeCreated, from,
		event.EdgeCreatedContent{
			From:      from,
			To:        to,
			EdgeType:  event.EdgeTypeAuthority,
			Weight:    edgeWeight,
			Direction: event.EdgeDirectionCentrifugal,
			Scope:     types.Some(scope),
			ExpiresAt: expiresAt,
		},
		causes,
		types.MustConversationID("conv_00000000000000000000000000000001"),
		headFromStore{s},
		chainTestSigner{},
	)
	if err != nil {
		t.Fatalf("create edge event: %v", err)
	}
	s.Append(ev)
	return ev.ID()
}

func TestDelegationChainNoDelegation(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	s := bootstrapStore(t)
	chain := authority.NewDelegationChain(model, s)

	a := testActor(t, "Alice", 1)
	result, err := chain.Evaluate(context.Background(), a, "test.action")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.Level != event.AuthorityLevelNotification {
		t.Errorf("Level = %v, want Notification", result.Level)
	}
	if result.Weight.Value() != 1.0 {
		t.Errorf("Weight = %v, want 1.0", result.Weight.Value())
	}
	if len(result.Chain) != 1 {
		t.Errorf("Chain length = %d, want 1", len(result.Chain))
	}
}

func TestDelegationChainSingleHop(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	s := bootstrapStore(t)
	chain := authority.NewDelegationChain(model, s)

	root := testActor(t, "Root", 1)
	delegate := testActor(t, "Delegate", 2)

	addAuthorityEdge(t, s, root.ID(), delegate.ID(), 0.8, types.MustDomainScope("code_review"), types.None[types.Timestamp]())

	result, err := chain.Evaluate(context.Background(), delegate, "test.action")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(result.Chain) != 2 {
		t.Fatalf("Chain length = %d, want 2", len(result.Chain))
	}
	if result.Chain[0].Actor != root.ID() {
		t.Errorf("Chain[0].Actor = %v, want root", result.Chain[0].Actor)
	}
	if result.Chain[1].Actor != delegate.ID() {
		t.Errorf("Chain[1].Actor = %v, want delegate", result.Chain[1].Actor)
	}
	if result.Chain[1].Weight.Value() != 0.8 {
		t.Errorf("Chain[1].Weight = %v, want 0.8", result.Chain[1].Weight.Value())
	}
}

func TestDelegationChainTwoHops(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	s := bootstrapStore(t)
	chain := authority.NewDelegationChain(model, s)

	root := testActor(t, "Root", 1)
	mid := testActor(t, "Mid", 2)
	leaf := testActor(t, "Leaf", 3)

	addAuthorityEdge(t, s, root.ID(), mid.ID(), 0.9, types.MustDomainScope("deploy"), types.None[types.Timestamp]())
	addAuthorityEdge(t, s, mid.ID(), leaf.ID(), 0.8, types.MustDomainScope("deploy"), types.None[types.Timestamp]())

	result, err := chain.Evaluate(context.Background(), leaf, "deploy.prod")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(result.Chain) != 3 {
		t.Fatalf("Chain length = %d, want 3", len(result.Chain))
	}

	// Weight propagates multiplicatively: 1.0 * 0.9 * 0.8 = 0.72
	expected := 1.0 * 0.9 * 0.8
	if abs(result.Weight.Value()-expected) > 0.001 {
		t.Errorf("Weight = %v, want %v", result.Weight.Value(), expected)
	}
}

func TestDelegationChainExpiredEdge(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	s := bootstrapStore(t)
	chain := authority.NewDelegationChain(model, s)

	root := testActor(t, "Root", 1)
	delegate := testActor(t, "Delegate", 2)

	expired := types.NewTimestamp(time.Now().Add(-1 * time.Hour))
	addAuthorityEdge(t, s, root.ID(), delegate.ID(), 0.9, types.MustDomainScope("code_review"), types.Some(expired))

	result, err := chain.Evaluate(context.Background(), delegate, "test")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(result.Chain) != 1 {
		t.Errorf("Chain length = %d, want 1 (expired delegation ignored)", len(result.Chain))
	}
}

func TestDelegationChainNotExpired(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	s := bootstrapStore(t)
	chain := authority.NewDelegationChain(model, s)

	root := testActor(t, "Root", 1)
	delegate := testActor(t, "Delegate", 2)

	future := types.NewTimestamp(time.Now().Add(1 * time.Hour))
	addAuthorityEdge(t, s, root.ID(), delegate.ID(), 0.9, types.MustDomainScope("code_review"), types.Some(future))

	result, err := chain.Evaluate(context.Background(), delegate, "test")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(result.Chain) != 2 {
		t.Errorf("Chain length = %d, want 2 (valid delegation)", len(result.Chain))
	}
}

func TestDelegationChainWithPolicy(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	s := bootstrapStore(t)
	chain := authority.NewDelegationChain(model, s)

	chain.AddPolicy(authority.AuthorityPolicy{
		Action: "deploy.*",
		Level:  event.AuthorityLevelRequired,
	})

	a := testActor(t, "Alice", 1)
	result, err := chain.Evaluate(context.Background(), a, "deploy.prod")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.Level != event.AuthorityLevelRequired {
		t.Errorf("Level = %v, want Required", result.Level)
	}
}

func TestDelegationChainMethod(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	s := bootstrapStore(t)
	chain := authority.NewDelegationChain(model, s)

	a := testActor(t, "Alice", 1)
	links, err := chain.Chain(context.Background(), a, "test")
	if err != nil {
		t.Fatalf("Chain: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("chain length = %d, want 1", len(links))
	}
}

func TestDelegationChainGrant(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	s := bootstrapStore(t)
	chain := authority.NewDelegationChain(model, s)

	from := testActor(t, "Alice", 1)
	to := testActor(t, "Bob", 2)

	edge, err := chain.Grant(context.Background(), from, to, types.MustDomainScope("code_review"), types.MustScore(0.8))
	if err != nil {
		t.Fatalf("Grant: %v", err)
	}
	if edge.Type() != event.EdgeTypeAuthority {
		t.Errorf("edge type = %v, want Authority", edge.Type())
	}
}

func TestDelegationChainRevoke(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	s := bootstrapStore(t)
	chain := authority.NewDelegationChain(model, s)

	from := testActor(t, "Alice", 1)
	to := testActor(t, "Bob", 2)

	err := chain.Revoke(context.Background(), from, to, types.MustDomainScope("code_review"))
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
}

func TestDelegationChainBestWeight(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	s := bootstrapStore(t)
	chain := authority.NewDelegationChain(model, s)

	root1 := testActor(t, "Root1", 1)
	root2 := testActor(t, "Root2", 2)
	delegate := testActor(t, "Delegate", 3)

	addAuthorityEdge(t, s, root1.ID(), delegate.ID(), 0.5, types.MustDomainScope("any"), types.None[types.Timestamp]())
	addAuthorityEdge(t, s, root2.ID(), delegate.ID(), 0.9, types.MustDomainScope("any"), types.None[types.Timestamp]())

	result, err := chain.Evaluate(context.Background(), delegate, "test")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(result.Chain) < 2 {
		t.Fatalf("Chain length = %d, want >= 2", len(result.Chain))
	}
	if result.Chain[len(result.Chain)-1].Weight.Value() != 0.9 {
		t.Errorf("delegate Weight = %v, want 0.9", result.Chain[len(result.Chain)-1].Weight.Value())
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
