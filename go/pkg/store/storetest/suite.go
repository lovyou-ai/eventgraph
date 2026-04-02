// Package storetest provides a conformance test suite for Store implementations.
// Any Store implementation can import and run this suite to verify correctness.
package storetest

import (
	"errors"
	"sync"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

type testSigner struct{}

func (testSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data)
	return types.MustSignature(sig), nil
}

type headFromStore struct{ s store.Store }

func (h headFromStore) Head() (types.Option[event.Event], error) { return h.s.Head() }

func makeBootstrap(t *testing.T) event.Event {
	t.Helper()
	registry := event.DefaultRegistry()
	factory := event.NewBootstrapFactory(registry)
	ev, err := factory.Init(
		types.MustActorID("actor_00000000000000000000000000000001"),
		testSigner{},
	)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}
	return ev
}

func makeChainedEvent(t *testing.T, s store.Store, causes []types.EventID) event.Event {
	t.Helper()
	registry := event.DefaultRegistry()
	factory := event.NewEventFactory(registry)
	ev, err := factory.Create(
		event.EventTypeTrustUpdated,
		types.MustActorID("actor_00000000000000000000000000000001"),
		event.TrustUpdatedContent{
			Actor:    types.MustActorID("actor_00000000000000000000000000000002"),
			Previous: types.MustScore(0.5),
			Current:  types.MustScore(0.6),
			Domain:   types.MustDomainScope("test"),
			Cause:    causes[0],
		},
		causes,
		types.MustConversationID("conv_00000000000000000000000000000001"),
		headFromStore{s},
		testSigner{},
	)
	if err != nil {
		t.Fatalf("create event failed: %v", err)
	}
	return ev
}

// RunConformanceSuite runs the full store conformance test suite against
// the provided factory function. Each test gets a fresh store instance.
func RunConformanceSuite(t *testing.T, newStore func() store.Store) {
	t.Run("AppendAndGet", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)

		stored, err := s.Append(ev)
		if err != nil {
			t.Fatalf("Append: %v", err)
		}
		if stored.ID() != ev.ID() {
			t.Errorf("stored ID %v != original %v", stored.ID(), ev.ID())
		}

		got, err := s.Get(ev.ID())
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.ID() != ev.ID() {
			t.Error("Get returned wrong event")
		}
	})

	t.Run("GetNotFound", func(t *testing.T) {
		s := newStore()
		_, err := s.Get(types.MustEventID("019462a0-0000-7000-8000-000000000099"))
		if err == nil {
			t.Fatal("expected EventNotFoundError")
		}
		var notFound *store.EventNotFoundError
		if !errors.As(err, &notFound) {
			t.Errorf("expected EventNotFoundError, got %T", err)
		}
	})

	t.Run("Idempotent", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)

		stored, err := s.Append(ev)
		if err != nil {
			t.Fatalf("idempotent Append: %v", err)
		}
		if stored.ID() != ev.ID() {
			t.Error("idempotent Append returned wrong event")
		}

		count, _ := s.Count()
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("HashChainIntegrity", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)

		ev2 := makeChainedEvent(t, s, []types.EventID{ev.ID()})
		_, err := s.Append(ev2)
		if err != nil {
			t.Fatalf("Append chained: %v", err)
		}

		count, _ := s.Count()
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	t.Run("ChainHeadConflict", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)

		wrongHash := types.MustHash("0000000000000000000000000000000000000000000000000000000000000000")
		badEv := event.NewEvent(1,
			types.MustEventID("019462a0-0000-7000-8000-000000000099"),
			event.EventTypeTrustUpdated,
			types.Now(),
			types.MustActorID("actor_00000000000000000000000000000001"),
			event.TrustUpdatedContent{},
			[]types.EventID{ev.ID()},
			types.MustConversationID("conv_00000000000000000000000000000001"),
			wrongHash, wrongHash,
			types.MustSignature(make([]byte, 64)),
		)
		_, err := s.Append(badEv)
		if err == nil {
			t.Fatal("expected chain integrity violation")
		}
	})

	t.Run("Head", func(t *testing.T) {
		s := newStore()

		head, _ := s.Head()
		if head.IsSome() {
			t.Error("Head should be None for empty store")
		}

		ev := makeBootstrap(t)
		s.Append(ev)
		head, _ = s.Head()
		if !head.IsSome() {
			t.Fatal("Head should be Some after append")
		}
		if head.Unwrap().ID() != ev.ID() {
			t.Error("Head should be the last appended event")
		}
	})

	t.Run("Recent", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)
		ev2 := makeChainedEvent(t, s, []types.EventID{ev.ID()})
		s.Append(ev2)

		page, _ := s.Recent(10, types.None[types.Cursor]())
		if len(page.Items()) != 2 {
			t.Fatalf("expected 2 items, got %d", len(page.Items()))
		}
		if page.Items()[0].ID() != ev2.ID() {
			t.Error("most recent should be first")
		}
	})

	t.Run("RecentPagination", func(t *testing.T) {
		s := newStore()
		ev0 := makeBootstrap(t)
		s.Append(ev0)
		prev := ev0
		for i := 0; i < 3; i++ {
			ev := makeChainedEvent(t, s, []types.EventID{prev.ID()})
			s.Append(ev)
			prev = ev
		}

		page1, _ := s.Recent(2, types.None[types.Cursor]())
		if len(page1.Items()) != 2 {
			t.Fatalf("page1: %d items, want 2", len(page1.Items()))
		}
		if !page1.HasMore() {
			t.Error("page1 should have more")
		}

		page2, _ := s.Recent(2, page1.Cursor())
		if len(page2.Items()) != 2 {
			t.Fatalf("page2: %d items, want 2", len(page2.Items()))
		}
	})

	t.Run("ByType", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)
		ev2 := makeChainedEvent(t, s, []types.EventID{ev.ID()})
		s.Append(ev2)

		page, _ := s.ByType(event.EventTypeTrustUpdated, 10, types.None[types.Cursor]())
		if len(page.Items()) != 1 {
			t.Errorf("expected 1 trust.updated, got %d", len(page.Items()))
		}
	})

	t.Run("BySource", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)

		page, _ := s.BySource(ev.Source(), 10, types.None[types.Cursor]())
		if len(page.Items()) != 1 {
			t.Errorf("expected 1 event from source, got %d", len(page.Items()))
		}
	})

	t.Run("ByConversation", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)

		page, _ := s.ByConversation(ev.ConversationID(), 10, types.None[types.Cursor]())
		if len(page.Items()) != 1 {
			t.Errorf("expected 1 event in conversation, got %d", len(page.Items()))
		}
	})

	t.Run("Since", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)
		ev2 := makeChainedEvent(t, s, []types.EventID{ev.ID()})
		s.Append(ev2)

		page, _ := s.Since(ev.ID(), 10)
		if len(page.Items()) != 1 {
			t.Fatalf("expected 1 event after bootstrap, got %d", len(page.Items()))
		}
		if page.Items()[0].ID() != ev2.ID() {
			t.Error("Since should return events after the given ID")
		}
	})

	t.Run("SinceNotFound", func(t *testing.T) {
		s := newStore()
		_, err := s.Since(types.MustEventID("019462a0-0000-7000-8000-000000000099"), 10)
		if err == nil {
			t.Fatal("expected error for Since with nonexistent ID")
		}
	})

	t.Run("SincePagination", func(t *testing.T) {
		s := newStore()
		ev0 := makeBootstrap(t)
		s.Append(ev0)
		prev := ev0
		for i := 0; i < 5; i++ {
			ev := makeChainedEvent(t, s, []types.EventID{prev.ID()})
			s.Append(ev)
			prev = ev
		}

		page, _ := s.Since(ev0.ID(), 2)
		if len(page.Items()) != 2 {
			t.Errorf("expected 2 items, got %d", len(page.Items()))
		}
		if !page.HasMore() {
			t.Error("should have more")
		}
	})

	t.Run("Ancestors", func(t *testing.T) {
		s := newStore()
		ev0 := makeBootstrap(t)
		s.Append(ev0)
		ev1 := makeChainedEvent(t, s, []types.EventID{ev0.ID()})
		s.Append(ev1)
		ev2 := makeChainedEvent(t, s, []types.EventID{ev1.ID()})
		s.Append(ev2)

		ancestors, err := s.Ancestors(ev2.ID(), 10)
		if err != nil {
			t.Fatalf("Ancestors: %v", err)
		}
		if len(ancestors) < 1 {
			t.Error("expected at least 1 ancestor")
		}
	})

	t.Run("AncestorsNotFound", func(t *testing.T) {
		s := newStore()
		_, err := s.Ancestors(types.MustEventID("019462a0-0000-7000-8000-000000000099"), 10)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("MultiCauseRoundTrip", func(t *testing.T) {
		s := newStore()
		ev0 := makeBootstrap(t)
		s.Append(ev0)

		// Create two independent events that will both be causes.
		ev1 := makeChainedEvent(t, s, []types.EventID{ev0.ID()})
		stored1, err := s.Append(ev1)
		if err != nil {
			t.Fatalf("Append ev1: %v", err)
		}
		ev2 := makeChainedEvent(t, s, []types.EventID{stored1.ID()})
		stored2, err := s.Append(ev2)
		if err != nil {
			t.Fatalf("Append ev2: %v", err)
		}

		// Create an event with two causes.
		ev3 := makeChainedEvent(t, s, []types.EventID{stored1.ID(), stored2.ID()})
		stored3, err := s.Append(ev3)
		if err != nil {
			t.Fatalf("Append ev3 (multi-cause): %v", err)
		}

		// Verify via Get.
		got, err := s.Get(stored3.ID())
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if len(got.Causes()) != 2 {
			t.Errorf("expected 2 causes, got %d", len(got.Causes()))
		}

		// Verify causes contain both expected IDs (order may vary).
		causeSet := make(map[types.EventID]bool)
		for _, c := range got.Causes() {
			causeSet[c] = true
		}
		if !causeSet[stored1.ID()] {
			t.Error("missing cause: stored1.ID()")
		}
		if !causeSet[stored2.ID()] {
			t.Error("missing cause: stored2.ID()")
		}

		// Verify via ByType (batch path).
		page, err := s.ByType(event.EventTypeTrustUpdated, 100, types.None[types.Cursor]())
		if err != nil {
			t.Fatalf("ByType: %v", err)
		}
		var found bool
		for _, item := range page.Items() {
			if item.ID() == stored3.ID() {
				found = true
				if len(item.Causes()) != 2 {
					t.Errorf("ByType: expected 2 causes, got %d", len(item.Causes()))
				}
			}
		}
		if !found {
			t.Error("multi-cause event not found in ByType results")
		}
	})

	t.Run("Descendants", func(t *testing.T) {
		s := newStore()
		ev0 := makeBootstrap(t)
		s.Append(ev0)
		ev1 := makeChainedEvent(t, s, []types.EventID{ev0.ID()})
		s.Append(ev1)

		descendants, err := s.Descendants(ev0.ID(), 10)
		if err != nil {
			t.Fatalf("Descendants: %v", err)
		}
		if len(descendants) != 1 {
			t.Errorf("expected 1 descendant, got %d", len(descendants))
		}
	})

	t.Run("DescendantsNotFound", func(t *testing.T) {
		s := newStore()
		_, err := s.Descendants(types.MustEventID("019462a0-0000-7000-8000-000000000099"), 10)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Count", func(t *testing.T) {
		s := newStore()
		count, _ := s.Count()
		if count != 0 {
			t.Errorf("empty store count = %d, want 0", count)
		}

		ev := makeBootstrap(t)
		s.Append(ev)
		count, _ = s.Count()
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("VerifyChain", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)
		ev2 := makeChainedEvent(t, s, []types.EventID{ev.ID()})
		s.Append(ev2)

		result, err := s.VerifyChain()
		if err != nil {
			t.Fatalf("VerifyChain: %v", err)
		}
		if !result.Valid {
			t.Error("chain should be valid")
		}
		if result.Length != 2 {
			t.Errorf("chain length = %d, want 2", result.Length)
		}
	})

	t.Run("VerifyChainEmpty", func(t *testing.T) {
		s := newStore()
		result, err := s.VerifyChain()
		if err != nil {
			t.Fatalf("VerifyChain: %v", err)
		}
		if !result.Valid {
			t.Error("empty chain should be valid")
		}
	})

	t.Run("EdgeQueries", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)

		from := types.MustActorID("actor_00000000000000000000000000000001")
		to := types.MustActorID("actor_00000000000000000000000000000002")

		edges, _ := s.EdgesFrom(from, event.EdgeTypeTrust)
		if len(edges) != 0 {
			t.Errorf("expected 0 edges from, got %d", len(edges))
		}

		edges, _ = s.EdgesTo(to, event.EdgeTypeTrust)
		if len(edges) != 0 {
			t.Errorf("expected 0 edges to, got %d", len(edges))
		}

		opt, _ := s.EdgeBetween(from, to, event.EdgeTypeTrust)
		if opt.IsSome() {
			t.Error("expected no edge between actors")
		}
	})

	t.Run("EdgeIndexing", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)

		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		from := types.MustActorID("actor_00000000000000000000000000000001")
		to := types.MustActorID("actor_00000000000000000000000000000002")

		edgeEv, err := factory.Create(
			event.EventTypeEdgeCreated, from,
			event.EdgeCreatedContent{
				From:      from,
				To:        to,
				EdgeType:  event.EdgeTypeTrust,
				Weight:    types.MustWeight(0.8),
				Direction: event.EdgeDirectionCentripetal,
				Scope:     types.Some(types.MustDomainScope("test")),
				ExpiresAt: types.None[types.Timestamp](),
			},
			[]types.EventID{ev.ID()},
			types.MustConversationID("conv_00000000000000000000000000000001"),
			headFromStore{s},
			testSigner{},
		)
		if err != nil {
			t.Fatalf("create edge event: %v", err)
		}
		s.Append(edgeEv)

		edgesFrom, _ := s.EdgesFrom(from, event.EdgeTypeTrust)
		if len(edgesFrom) != 1 {
			t.Errorf("expected 1 edge from, got %d", len(edgesFrom))
		}

		edgesTo, _ := s.EdgesTo(to, event.EdgeTypeTrust)
		if len(edgesTo) != 1 {
			t.Errorf("expected 1 edge to, got %d", len(edgesTo))
		}

		between, _ := s.EdgeBetween(from, to, event.EdgeTypeTrust)
		if !between.IsSome() {
			t.Error("expected edge between actors")
		}
	})

	t.Run("ConcurrentAppend", func(t *testing.T) {
		s := newStore()
		ev := makeBootstrap(t)
		s.Append(ev)

		var events []event.Event
		for i := 0; i < 5; i++ {
			e := makeChainedEvent(t, s, []types.EventID{ev.ID()})
			events = append(events, e)
		}

		var wg sync.WaitGroup
		successes := make(chan bool, len(events))
		for _, e := range events {
			wg.Add(1)
			go func(ev event.Event) {
				defer wg.Done()
				_, err := s.Append(ev)
				successes <- (err == nil)
			}(e)
		}
		wg.Wait()
		close(successes)

		successCount := 0
		for s := range successes {
			if s {
				successCount++
			}
		}
		if successCount < 1 {
			t.Error("at least one concurrent append should succeed")
		}
	})

	t.Run("Close", func(t *testing.T) {
		s := newStore()
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
}
