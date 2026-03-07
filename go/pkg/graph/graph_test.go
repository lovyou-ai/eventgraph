package graph_test

import (
	"context"
	"testing"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/authority"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/graph"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

type testSigner struct{}

func (testSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data[:min(64, len(data))])
	return types.MustSignature(sig), nil
}

func testPublicKey(b byte) types.PublicKey {
	key := make([]byte, 32)
	key[0] = b
	return types.MustPublicKey(key)
}

func newTestGraph(t *testing.T) (*graph.Graph, types.ActorID) {
	t.Helper()
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	g := graph.New(s, as)
	g.Start()

	actorID := types.MustActorID("actor_system0000000000000000000001")
	return g, actorID
}

func TestNewGraph(t *testing.T) {
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	g := graph.New(s, as)
	defer g.Close()

	if g.Store() != s {
		t.Error("Store() should return the wrapped store")
	}
	if g.ActorStore() != as {
		t.Error("ActorStore() should return the wrapped actor store")
	}
	if g.Bus() == nil {
		t.Error("Bus() should not be nil")
	}
	if g.Registry() == nil {
		t.Error("Registry() should not be nil")
	}
}

func TestBootstrap(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	ev, err := g.Bootstrap(actorID, testSigner{})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if ev.Type().Value() != "system.bootstrapped" {
		t.Errorf("Type = %v, want system.bootstrapped", ev.Type().Value())
	}
	if ev.Source() != actorID {
		t.Error("Source should be the system actor")
	}

	count, _ := g.Store().Count()
	if count != 1 {
		t.Errorf("Count = %d, want 1", count)
	}
}

func TestRecord(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	bootstrap, _ := g.Bootstrap(actorID, testSigner{})

	content := event.TrustUpdatedContent{
		Actor:    actorID,
		Previous: types.MustScore(0.0),
		Current:  types.MustScore(0.5),
		Domain:   types.MustDomainScope("general"),
		Cause:    bootstrap.ID(),
	}

	ev, err := g.Record(
		event.EventTypeTrustUpdated,
		actorID,
		content,
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		testSigner{},
	)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if ev.Type().Value() != "trust.updated" {
		t.Errorf("Type = %v, want trust.updated", ev.Type().Value())
	}

	count, _ := g.Store().Count()
	if count != 2 {
		t.Errorf("Count = %d, want 2", count)
	}
}

func TestRecordAfterClose(t *testing.T) {
	g, actorID := newTestGraph(t)
	g.Bootstrap(actorID, testSigner{})
	g.Close()

	_, err := g.Record(
		event.EventTypeTrustUpdated,
		actorID,
		event.TrustUpdatedContent{
			Actor:    actorID,
			Previous: types.MustScore(0.0),
			Current:  types.MustScore(0.5),
			Domain:   types.MustDomainScope("general"),
			Cause:    types.MustEventID("019462a0-0000-7000-8000-000000000001"),
		},
		nil,
		types.MustConversationID("conv_test000000000000000000000001"),
		testSigner{},
	)
	if err == nil {
		t.Fatal("expected error recording after close")
	}
}

func TestEvaluate(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	as := g.ActorStore()
	pk := testPublicKey(1)
	a, _ := as.Register(pk, "Alice", event.ActorTypeHuman)

	result, err := g.Evaluate(context.Background(), a, "test.action")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	// Default authority chain returns Notification for unknown actions
	if result.Level != event.AuthorityLevelNotification {
		t.Errorf("Level = %v, want Notification", result.Level)
	}
	_ = actorID
}

func TestEvaluateAfterClose(t *testing.T) {
	g, _ := newTestGraph(t)
	as := g.ActorStore()
	pk := testPublicKey(1)
	a, _ := as.Register(pk, "Alice", event.ActorTypeHuman)

	g.Close()

	_, err := g.Evaluate(context.Background(), a, "test")
	if err == nil {
		t.Fatal("expected error evaluating after close")
	}
}

func TestQuery(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	g.Bootstrap(actorID, testSigner{})

	q, _ := g.Query()

	// EventCount
	count, err := q.EventCount()
	if err != nil {
		t.Fatalf("EventCount: %v", err)
	}
	if count != 1 {
		t.Errorf("EventCount = %d, want 1", count)
	}

	// Recent
	page, err := q.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Errorf("Recent items = %d, want 1", len(page.Items()))
	}
}

func TestQueryByType(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	g.Bootstrap(actorID, testSigner{})

	q, _ := g.Query()
	page, err := q.ByType(event.EventTypeSystemBootstrapped, 10)
	if err != nil {
		t.Fatalf("ByType: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Errorf("ByType items = %d, want 1", len(page.Items()))
	}
}

func TestQueryBySource(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	g.Bootstrap(actorID, testSigner{})

	q, _ := g.Query()
	page, err := q.BySource(actorID, 10)
	if err != nil {
		t.Fatalf("BySource: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Errorf("BySource items = %d, want 1", len(page.Items()))
	}
}

func TestQueryTrust(t *testing.T) {
	g, _ := newTestGraph(t)
	defer g.Close()

	as := g.ActorStore()
	pk := testPublicKey(1)
	a, _ := as.Register(pk, "Alice", event.ActorTypeHuman)

	q, _ := g.Query()
	metrics, err := q.TrustScore(context.Background(), a)
	if err != nil {
		t.Fatalf("TrustScore: %v", err)
	}
	if metrics.Overall().Value() != 0.0 {
		t.Errorf("initial trust = %v, want 0.0", metrics.Overall().Value())
	}
}

func TestStartIdempotent(t *testing.T) {
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	g := graph.New(s, as)
	defer g.Close()

	if err := g.Start(); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	if err := g.Start(); err != nil {
		t.Fatalf("second Start: %v", err)
	}
}

func TestCloseIdempotent(t *testing.T) {
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	g := graph.New(s, as)

	if err := g.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := g.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestWithOptions(t *testing.T) {
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()

	config := graph.Config{
		SubscriberBufferSize: 512,
		FallbackToMechanical: false,
	}

	g := graph.New(s, as, graph.WithConfig(config))
	defer g.Close()

	if g.Store() != s {
		t.Error("Store should be the one provided")
	}
}

func TestBusReceivesPublishedEvents(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	received := make(chan event.Event, 1)
	g.Bus().Subscribe(types.MustSubscriptionPattern("*"), func(ev event.Event) {
		received <- ev
	})

	g.Bootstrap(actorID, testSigner{})

	select {
	case ev := <-received:
		if ev.Type().Value() != "system.bootstrapped" {
			t.Errorf("received event type = %v, want system.bootstrapped", ev.Type().Value())
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bus delivery")
	}
}

// --- mock types for option tests ---

type mockTrustModel struct{ called bool }

func (m *mockTrustModel) zeroMetrics(id types.ActorID) event.TrustMetrics {
	return event.NewTrustMetrics(
		id,
		types.MustScore(0.0),
		nil,
		types.MustScore(0.0),
		types.MustWeight(0.0),
		nil,
		types.Now(),
		types.MustScore(0.0),
	)
}

func (m *mockTrustModel) Score(_ context.Context, a actor.IActor) (event.TrustMetrics, error) {
	m.called = true
	return m.zeroMetrics(a.ID()), nil
}
func (m *mockTrustModel) ScoreInDomain(_ context.Context, a actor.IActor, _ types.DomainScope) (event.TrustMetrics, error) {
	return m.zeroMetrics(a.ID()), nil
}
func (m *mockTrustModel) Update(_ context.Context, a actor.IActor, _ event.Event) (event.TrustMetrics, error) {
	return m.zeroMetrics(a.ID()), nil
}
func (m *mockTrustModel) UpdateBetween(_ context.Context, _ actor.IActor, to actor.IActor, _ event.Event) (event.TrustMetrics, error) {
	return m.zeroMetrics(to.ID()), nil
}
func (m *mockTrustModel) Decay(_ context.Context, a actor.IActor, _ time.Duration) (event.TrustMetrics, error) {
	return m.zeroMetrics(a.ID()), nil
}
func (m *mockTrustModel) Between(_ context.Context, from actor.IActor, _ actor.IActor) (event.TrustMetrics, error) {
	m.called = true
	return m.zeroMetrics(from.ID()), nil
}

type mockAuthorityChain struct{ evaluated bool }

func (m *mockAuthorityChain) Evaluate(ctx context.Context, a actor.IActor, action string) (authority.AuthorityResult, error) {
	m.evaluated = true
	return authority.AuthorityResult{
		Level:  event.AuthorityLevelNotification,
		Weight: types.MustScore(1.0),
	}, nil
}
func (m *mockAuthorityChain) Chain(ctx context.Context, a actor.IActor, action string) ([]event.AuthorityLink, error) {
	return nil, nil
}
func (m *mockAuthorityChain) Grant(ctx context.Context, from actor.IActor, to actor.IActor, scope types.DomainScope, weight types.Score) (event.Edge, error) {
	return event.Edge{}, nil
}
func (m *mockAuthorityChain) Revoke(ctx context.Context, from actor.IActor, to actor.IActor, scope types.DomainScope) error {
	return nil
}

// --- option tests ---

func TestWithTrustModel(t *testing.T) {
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	tm := &mockTrustModel{}

	g := graph.New(s, as, graph.WithTrustModel(tm))
	defer g.Close()
	g.Start()

	pk := testPublicKey(1)
	a, _ := as.Register(pk, "Alice", event.ActorTypeHuman)

	q, _ := g.Query()
	_, err := q.TrustScore(context.Background(), a)
	if err != nil {
		t.Fatalf("TrustScore: %v", err)
	}
	if !tm.called {
		t.Error("expected custom trust model to be called")
	}
}

func TestWithAuthorityChain(t *testing.T) {
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	ac := &mockAuthorityChain{}

	g := graph.New(s, as, graph.WithAuthorityChain(ac))
	defer g.Close()
	g.Start()

	pk := testPublicKey(1)
	a, _ := as.Register(pk, "Alice", event.ActorTypeHuman)

	_, err := g.Evaluate(context.Background(), a, "test.action")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !ac.evaluated {
		t.Error("expected custom authority chain to be called")
	}
}

func TestWithDecisionMaker(t *testing.T) {
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()

	// Just test that the option is accepted and doesn't panic.
	g := graph.New(s, as, graph.WithDecisionMaker(nil))
	defer g.Close()

	if g.Store() != s {
		t.Error("Store should be the one provided")
	}
}

// --- query tests ---

func TestQueryByConversation(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	bootstrap, err := g.Bootstrap(actorID, testSigner{})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	convID := types.MustConversationID("conv_test000000000000000000000001")

	// Record an event with a known conversation ID.
	_, err = g.Record(
		event.EventTypeTrustUpdated,
		actorID,
		event.TrustUpdatedContent{
			Actor:    actorID,
			Previous: types.MustScore(0.0),
			Current:  types.MustScore(0.5),
			Domain:   types.MustDomainScope("general"),
			Cause:    bootstrap.ID(),
		},
		[]types.EventID{bootstrap.ID()},
		convID,
		testSigner{},
	)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	q, _ := g.Query()
	page, err := q.ByConversation(convID, 10)
	if err != nil {
		t.Fatalf("ByConversation: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Errorf("ByConversation items = %d, want 1", len(page.Items()))
	}
}

func TestQueryAncestors(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	bootstrap, err := g.Bootstrap(actorID, testSigner{})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	convID := types.MustConversationID("conv_test000000000000000000000001")

	// Record a child event caused by bootstrap.
	child, err := g.Record(
		event.EventTypeTrustUpdated,
		actorID,
		event.TrustUpdatedContent{
			Actor:    actorID,
			Previous: types.MustScore(0.0),
			Current:  types.MustScore(0.5),
			Domain:   types.MustDomainScope("general"),
			Cause:    bootstrap.ID(),
		},
		[]types.EventID{bootstrap.ID()},
		convID,
		testSigner{},
	)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	q, _ := g.Query()
	ancestors, err := q.Ancestors(child.ID(), 10)
	if err != nil {
		t.Fatalf("Ancestors: %v", err)
	}
	if len(ancestors) < 1 {
		t.Fatalf("Ancestors count = %d, want >= 1", len(ancestors))
	}
	found := false
	for _, a := range ancestors {
		if a.ID() == bootstrap.ID() {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected bootstrap event in ancestors")
	}
}

func TestQueryDescendants(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	bootstrap, err := g.Bootstrap(actorID, testSigner{})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	convID := types.MustConversationID("conv_test000000000000000000000001")

	child, err := g.Record(
		event.EventTypeTrustUpdated,
		actorID,
		event.TrustUpdatedContent{
			Actor:    actorID,
			Previous: types.MustScore(0.0),
			Current:  types.MustScore(0.5),
			Domain:   types.MustDomainScope("general"),
			Cause:    bootstrap.ID(),
		},
		[]types.EventID{bootstrap.ID()},
		convID,
		testSigner{},
	)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	q, _ := g.Query()
	descendants, err := q.Descendants(bootstrap.ID(), 10)
	if err != nil {
		t.Fatalf("Descendants: %v", err)
	}
	if len(descendants) < 1 {
		t.Fatalf("Descendants count = %d, want >= 1", len(descendants))
	}
	found := false
	for _, d := range descendants {
		if d.ID() == child.ID() {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected child event in descendants")
	}
}

func TestQueryTrustBetween(t *testing.T) {
	g, _ := newTestGraph(t)
	defer g.Close()

	as := g.ActorStore()
	pk1 := testPublicKey(1)
	pk2 := testPublicKey(2)
	alice, _ := as.Register(pk1, "Alice", event.ActorTypeHuman)
	bob, _ := as.Register(pk2, "Bob", event.ActorTypeHuman)

	q, _ := g.Query()
	metrics, err := q.TrustBetween(context.Background(), alice, bob)
	if err != nil {
		t.Fatalf("TrustBetween: %v", err)
	}
	if metrics.Overall().Value() != 0.0 {
		t.Errorf("initial trust between = %v, want 0.0", metrics.Overall().Value())
	}
}

func TestQueryActor(t *testing.T) {
	g, _ := newTestGraph(t)
	defer g.Close()

	as := g.ActorStore()
	pk := testPublicKey(3)
	registered, _ := as.Register(pk, "Charlie", event.ActorTypeHuman)

	q, _ := g.Query()
	found, err := q.Actor(registered.ID())
	if err != nil {
		t.Fatalf("Actor: %v", err)
	}
	if found.ID() != registered.ID() {
		t.Errorf("Actor ID = %v, want %v", found.ID(), registered.ID())
	}
	if found.DisplayName() != "Charlie" {
		t.Errorf("Actor DisplayName = %v, want Charlie", found.DisplayName())
	}
}
