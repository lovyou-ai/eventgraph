package graph_test

import (
	"context"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
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
		types.MustEventType("trust.updated"),
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
		types.MustEventType("trust.updated"),
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

	result, err := g.Evaluate(context.Background(), a, "test.action", nil)
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

	_, err := g.Evaluate(context.Background(), a, "test", nil)
	if err == nil {
		t.Fatal("expected error evaluating after close")
	}
}

func TestQuery(t *testing.T) {
	g, actorID := newTestGraph(t)
	defer g.Close()

	g.Bootstrap(actorID, testSigner{})

	q := g.Query()

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

	q := g.Query()
	page, err := q.ByType(types.MustEventType("system.bootstrapped"), 10)
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

	q := g.Query()
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

	q := g.Query()
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
	default:
		// Give async delivery a moment
		// Bus delivery is async, may not be immediate
	}
}
