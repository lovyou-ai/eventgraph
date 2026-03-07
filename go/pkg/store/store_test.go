package store_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/store/storetest"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestInMemoryStoreConformance runs the shared conformance test suite.
func TestInMemoryStoreConformance(t *testing.T) {
	storetest.RunConformanceSuite(t, func() store.Store {
		return store.NewInMemoryStore()
	})
}

// --- Test helper: deterministic signer ---

type testSigner struct{}

func (s testSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data)
	return types.MustSignature(sig), nil
}

// --- Test helper: build events with correct chain ---

func makeBootstrapEvent(t *testing.T) event.Event {
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

type headFromStore struct{ s store.Store }

func (h headFromStore) Head() (types.Option[event.Event], error) { return h.s.Head() }

func makeEvent(t *testing.T, s store.Store, eventType types.EventType, causes []types.EventID) event.Event {
	t.Helper()
	registry := event.DefaultRegistry()
	factory := event.NewEventFactory(registry)
	ev, err := factory.Create(
		eventType,
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

// --- Store error tests ---

func TestStoreErrorTypes(t *testing.T) {
	eventID := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	hash := types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")

	errs := []error{
		&store.EventNotFoundError{ID: eventID},
		&store.ActorNotFoundError{ID: actorID},
		&store.EdgeNotFoundError{From: actorID, To: actorID, EdgeType: event.EdgeTypeTrust},
		&store.DuplicateEventError{ID: eventID},
		&store.CausalLinkMissingError{EventID: eventID, MissingCause: eventID},
		&store.ChainIntegrityViolationError{Position: 0, Expected: hash, Actual: hash},
		&store.HashMismatchError{EventID: eventID, Computed: hash, Stored: hash},
		&store.SignatureInvalidError{EventID: eventID, Signer: actorID},
		&store.ActorSuspendedError{ID: actorID},
		&store.ActorMemorialError{ID: actorID},
		&store.RateLimitExceededError{Actor: actorID, Limit: 10, Window: "1m"},
		&store.StoreUnavailableError{Reason: "test"},
	}

	for _, err := range errs {
		if err.Error() == "" {
			t.Errorf("error %T has empty message", err)
		}
	}
}

func TestStoreErrorsAsDispatch(t *testing.T) {
	err := error(&store.EventNotFoundError{ID: types.MustEventID("019462a0-0000-7000-8000-000000000001")})
	var notFound *store.EventNotFoundError
	if !errors.As(err, &notFound) {
		t.Error("errors.As should match EventNotFoundError")
	}
}

// --- Store error visitor test ---

type storeErrorCollector struct{ visited string }

func (c *storeErrorCollector) VisitEventNotFound(*store.EventNotFoundError)                   { c.visited = "EventNotFound" }
func (c *storeErrorCollector) VisitActorNotFound(*store.ActorNotFoundError)                   { c.visited = "ActorNotFound" }
func (c *storeErrorCollector) VisitActorKeyNotFound(*store.ActorKeyNotFoundError)             { c.visited = "ActorKeyNotFound" }
func (c *storeErrorCollector) VisitEdgeNotFound(*store.EdgeNotFoundError)                     { c.visited = "EdgeNotFound" }
func (c *storeErrorCollector) VisitDuplicateEvent(*store.DuplicateEventError)                 { c.visited = "DuplicateEvent" }
func (c *storeErrorCollector) VisitCausalLinkMissing(*store.CausalLinkMissingError)           { c.visited = "CausalLinkMissing" }
func (c *storeErrorCollector) VisitChainIntegrityViolation(*store.ChainIntegrityViolationError) { c.visited = "ChainIntegrityViolation" }
func (c *storeErrorCollector) VisitHashMismatch(*store.HashMismatchError)                     { c.visited = "HashMismatch" }
func (c *storeErrorCollector) VisitSignatureInvalid(*store.SignatureInvalidError)              { c.visited = "SignatureInvalid" }
func (c *storeErrorCollector) VisitActorSuspended(*store.ActorSuspendedError)                 { c.visited = "ActorSuspended" }
func (c *storeErrorCollector) VisitActorMemorial(*store.ActorMemorialError)                   { c.visited = "ActorMemorial" }
func (c *storeErrorCollector) VisitRateLimitExceeded(*store.RateLimitExceededError)            { c.visited = "RateLimitExceeded" }
func (c *storeErrorCollector) VisitStoreUnavailable(*store.StoreUnavailableError)              { c.visited = "StoreUnavailable" }
func (c *storeErrorCollector) VisitInvalidCursor(*store.InvalidCursorError)                    { c.visited = "InvalidCursor" }

func TestStoreErrorVisitor(t *testing.T) {
	eventID := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	hash := types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")

	tests := []struct {
		err      store.VisitableStoreError
		expected string
	}{
		{&store.EventNotFoundError{ID: eventID}, "EventNotFound"},
		{&store.ActorNotFoundError{ID: actorID}, "ActorNotFound"},
		{&store.ActorKeyNotFoundError{KeyHex: "deadbeef"}, "ActorKeyNotFound"},
		{&store.EdgeNotFoundError{From: actorID, To: actorID, EdgeType: event.EdgeTypeTrust}, "EdgeNotFound"},
		{&store.DuplicateEventError{ID: eventID}, "DuplicateEvent"},
		{&store.CausalLinkMissingError{EventID: eventID, MissingCause: eventID}, "CausalLinkMissing"},
		{&store.ChainIntegrityViolationError{Position: 0, Expected: hash, Actual: hash}, "ChainIntegrityViolation"},
		{&store.HashMismatchError{EventID: eventID, Computed: hash, Stored: hash}, "HashMismatch"},
		{&store.SignatureInvalidError{EventID: eventID, Signer: actorID}, "SignatureInvalid"},
		{&store.ActorSuspendedError{ID: actorID}, "ActorSuspended"},
		{&store.ActorMemorialError{ID: actorID}, "ActorMemorial"},
		{&store.RateLimitExceededError{Actor: actorID, Limit: 10, Window: "1m"}, "RateLimitExceeded"},
		{&store.StoreUnavailableError{Reason: "test"}, "StoreUnavailable"},
		{&store.InvalidCursorError{Cursor: "bad"}, "InvalidCursor"},
	}

	for _, tt := range tests {
		c := &storeErrorCollector{}
		tt.err.Accept(c)
		if c.visited != tt.expected {
			t.Errorf("visitor dispatched to %q, want %q", c.visited, tt.expected)
		}
	}
}

// --- InMemoryStore tests ---

func TestAppendAndGet(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)

	stored, err := s.Append(ev)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if stored.ID() != ev.ID() {
		t.Errorf("stored ID %v != original %v", stored.ID(), ev.ID())
	}

	got, err := s.Get(ev.ID())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID() != ev.ID() {
		t.Errorf("Get returned wrong event")
	}
}

func TestGetNotFound(t *testing.T) {
	s := store.NewInMemoryStore()
	_, err := s.Get(types.MustEventID("019462a0-0000-7000-8000-000000000099"))
	if err == nil {
		t.Fatal("expected EventNotFoundError")
	}
	var notFound *store.EventNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected EventNotFoundError, got %T", err)
	}
}

func TestAppendIdempotent(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)

	_, err := s.Append(ev)
	if err != nil {
		t.Fatalf("first Append failed: %v", err)
	}

	// Same event again — should return existing, not error
	stored, err := s.Append(ev)
	if err != nil {
		t.Fatalf("idempotent Append failed: %v", err)
	}
	if stored.ID() != ev.ID() {
		t.Error("idempotent Append returned wrong event")
	}

	count, _ := s.Count()
	if count != 1 {
		t.Errorf("count should be 1 after idempotent append, got %d", count)
	}
}

func TestAppendChainIntegrity(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	_, err := s.Append(ev)
	if err != nil {
		t.Fatalf("Append bootstrap failed: %v", err)
	}

	// Append another event that chains correctly
	ev2 := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{ev.ID()})
	_, err = s.Append(ev2)
	if err != nil {
		t.Fatalf("Append chained event failed: %v", err)
	}

	count, _ := s.Count()
	if count != 2 {
		t.Errorf("count should be 2, got %d", count)
	}
}

func TestHead(t *testing.T) {
	s := store.NewInMemoryStore()

	// Empty store
	head, err := s.Head()
	if err != nil {
		t.Fatalf("Head failed: %v", err)
	}
	if head.IsSome() {
		t.Error("Head should be None for empty store")
	}

	// After bootstrap
	ev := makeBootstrapEvent(t)
	s.Append(ev)
	head, err = s.Head()
	if err != nil {
		t.Fatalf("Head failed: %v", err)
	}
	if !head.IsSome() {
		t.Fatal("Head should be Some after append")
	}
	if head.Unwrap().ID() != ev.ID() {
		t.Error("Head should be the last appended event")
	}
}

func TestRecent(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	ev2 := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{ev.ID()})
	s.Append(ev2)

	page, err := s.Recent(10, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("Recent failed: %v", err)
	}
	items := page.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// Most recent first
	if items[0].ID() != ev2.ID() {
		t.Error("most recent event should be first")
	}
}

func TestByType(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	ev2 := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{ev.ID()})
	s.Append(ev2)

	page, err := s.ByType(event.EventTypeTrustUpdated, 10, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType failed: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Errorf("expected 1 trust.updated event, got %d", len(page.Items()))
	}
}

func TestBySource(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	page, err := s.BySource(ev.Source(), 10, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("BySource failed: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Errorf("expected 1 event from source, got %d", len(page.Items()))
	}
}

func TestByConversation(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	page, err := s.ByConversation(ev.ConversationID(), 10, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByConversation failed: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Errorf("expected 1 event in conversation, got %d", len(page.Items()))
	}
}

func TestSince(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	ev2 := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{ev.ID()})
	s.Append(ev2)

	page, err := s.Since(ev.ID(), 10)
	if err != nil {
		t.Fatalf("Since failed: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Fatalf("expected 1 event after bootstrap, got %d", len(page.Items()))
	}
	if page.Items()[0].ID() != ev2.ID() {
		t.Error("Since should return events after the given ID")
	}
}

func TestSinceNotFound(t *testing.T) {
	s := store.NewInMemoryStore()
	_, err := s.Since(types.MustEventID("019462a0-0000-7000-8000-000000000099"), 10)
	if err == nil {
		t.Fatal("expected error for Since with nonexistent ID")
	}
}

func TestAncestors(t *testing.T) {
	s := store.NewInMemoryStore()
	ev0 := makeBootstrapEvent(t)
	s.Append(ev0)

	ev1 := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{ev0.ID()})
	s.Append(ev1)

	ev2 := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{ev1.ID()})
	s.Append(ev2)

	ancestors, err := s.Ancestors(ev2.ID(), 10)
	if err != nil {
		t.Fatalf("Ancestors failed: %v", err)
	}
	if len(ancestors) < 1 {
		t.Error("expected at least 1 ancestor")
	}
}

func TestAncestorsNotFound(t *testing.T) {
	s := store.NewInMemoryStore()
	_, err := s.Ancestors(types.MustEventID("019462a0-0000-7000-8000-000000000099"), 10)
	if err == nil {
		t.Fatal("expected error for Ancestors with nonexistent ID")
	}
}

func TestDescendants(t *testing.T) {
	s := store.NewInMemoryStore()
	ev0 := makeBootstrapEvent(t)
	s.Append(ev0)

	ev1 := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{ev0.ID()})
	s.Append(ev1)

	descendants, err := s.Descendants(ev0.ID(), 10)
	if err != nil {
		t.Fatalf("Descendants failed: %v", err)
	}
	if len(descendants) != 1 {
		t.Errorf("expected 1 descendant, got %d", len(descendants))
	}
}

func TestDescendantsNotFound(t *testing.T) {
	s := store.NewInMemoryStore()
	_, err := s.Descendants(types.MustEventID("019462a0-0000-7000-8000-000000000099"), 10)
	if err == nil {
		t.Fatal("expected error for Descendants with nonexistent ID")
	}
}

func TestCount(t *testing.T) {
	s := store.NewInMemoryStore()
	count, _ := s.Count()
	if count != 0 {
		t.Errorf("empty store count should be 0, got %d", count)
	}

	ev := makeBootstrapEvent(t)
	s.Append(ev)
	count, _ = s.Count()
	if count != 1 {
		t.Errorf("count should be 1, got %d", count)
	}
}

func TestVerifyChain(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	ev2 := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{ev.ID()})
	s.Append(ev2)

	result, err := s.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}
	if !result.Valid {
		t.Error("chain should be valid")
	}
	if result.Length != 2 {
		t.Errorf("chain length should be 2, got %d", result.Length)
	}
}

func TestVerifyChainEmpty(t *testing.T) {
	s := store.NewInMemoryStore()
	result, err := s.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}
	if !result.Valid {
		t.Error("empty chain should be valid")
	}
}

func TestClose(t *testing.T) {
	s := store.NewInMemoryStore()
	if err := s.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestEdgeQueries(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	// EdgesFrom with no edges
	edges, err := s.EdgesFrom(types.MustActorID("actor_00000000000000000000000000000001"), event.EdgeTypeTrust)
	if err != nil {
		t.Fatalf("EdgesFrom failed: %v", err)
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(edges))
	}

	// EdgesTo with no edges
	edges, err = s.EdgesTo(types.MustActorID("actor_00000000000000000000000000000002"), event.EdgeTypeTrust)
	if err != nil {
		t.Fatalf("EdgesTo failed: %v", err)
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(edges))
	}

	// EdgeBetween with no edges
	opt, err := s.EdgeBetween(
		types.MustActorID("actor_00000000000000000000000000000001"),
		types.MustActorID("actor_00000000000000000000000000000002"),
		event.EdgeTypeTrust,
	)
	if err != nil {
		t.Fatalf("EdgeBetween failed: %v", err)
	}
	if opt.IsSome() {
		t.Error("expected no edge between actors")
	}
}

func TestConcurrentAppend(t *testing.T) {
	// Concurrent appends to the same store — only one should succeed per chain position
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	// Build multiple events that all point to the same head
	var events []event.Event
	for i := 0; i < 5; i++ {
		ev := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{ev.ID()})
		events = append(events, ev)
	}

	var wg sync.WaitGroup
	successes := make(chan bool, len(events))
	for _, ev := range events {
		wg.Add(1)
		go func(e event.Event) {
			defer wg.Done()
			_, err := s.Append(e)
			successes <- (err == nil)
		}(ev)
	}
	wg.Wait()
	close(successes)

	successCount := 0
	for s := range successes {
		if s {
			successCount++
		}
	}
	// At least one should succeed (the first one), others may fail due to chain head conflict
	if successCount < 1 {
		t.Error("at least one concurrent append should succeed")
	}
}

// Test that factory refuses empty causes
func TestFactoryRefusesEmptyCauses(t *testing.T) {
	registry := event.DefaultRegistry()
	factory := event.NewEventFactory(registry)
	s := store.NewInMemoryStore()

	_, err := factory.Create(
		event.EventTypeTrustUpdated,
		types.MustActorID("actor_00000000000000000000000000000001"),
		event.TrustUpdatedContent{},
		nil, // empty causes
		types.MustConversationID("conv_00000000000000000000000000000001"),
		headFromStore{s},
		testSigner{},
	)
	if err == nil {
		t.Fatal("expected error for empty causes")
	}
}

// Test bootstrap factory
func TestBootstrapFactory(t *testing.T) {
	registry := event.DefaultRegistry()
	factory := event.NewBootstrapFactory(registry)
	ev, err := factory.Init(
		types.MustActorID("actor_00000000000000000000000000000001"),
		testSigner{},
	)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}
	if !ev.IsBootstrap() {
		t.Error("should be a bootstrap event")
	}
	if ev.Version() != 1 {
		t.Errorf("version should be 1, got %d", ev.Version())
	}
	if ev.Type().Value() != "system.bootstrapped" {
		t.Errorf("type should be system.bootstrapped, got %s", ev.Type().Value())
	}
}

// Test factory with invalid event type
func TestFactoryInvalidEventType(t *testing.T) {
	registry := event.DefaultRegistry()
	factory := event.NewEventFactory(registry)
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	_, err := factory.Create(
		event.EventTypeTrustUpdated,
		types.MustActorID("actor_00000000000000000000000000000001"),
		event.BootstrapContent{}, // wrong content for trust.updated
		[]types.EventID{ev.ID()},
		types.MustConversationID("conv_00000000000000000000000000000001"),
		headFromStore{s},
		testSigner{},
	)
	if err == nil {
		t.Fatal("expected error for mismatched content type")
	}
}

// Test pagination with Recent
func TestRecentPagination(t *testing.T) {
	s := store.NewInMemoryStore()
	ev0 := makeBootstrapEvent(t)
	s.Append(ev0)

	// Add a few more events
	prev := ev0
	for i := 0; i < 3; i++ {
		ev := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{prev.ID()})
		s.Append(ev)
		prev = ev
	}

	// Get first page (limit 2)
	page1, err := s.Recent(2, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("Recent page 1 failed: %v", err)
	}
	if len(page1.Items()) != 2 {
		t.Fatalf("expected 2 items in page 1, got %d", len(page1.Items()))
	}
	if !page1.HasMore() {
		t.Error("page 1 should have more")
	}

	// Get second page
	page2, err := s.Recent(2, page1.Cursor())
	if err != nil {
		t.Fatalf("Recent page 2 failed: %v", err)
	}
	if len(page2.Items()) != 2 {
		t.Fatalf("expected 2 items in page 2, got %d", len(page2.Items()))
	}
}

// Test edge indexing via edge.created event
func TestEdgeIndexing(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	// Create an edge.created event
	registry := event.DefaultRegistry()
	factory := event.NewEventFactory(registry)
	from := types.MustActorID("actor_00000000000000000000000000000001")
	to := types.MustActorID("actor_00000000000000000000000000000002")

	edgeEv, err := factory.Create(
		event.EventTypeEdgeCreated,
		from,
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
		t.Fatalf("create edge event failed: %v", err)
	}
	_, err = s.Append(edgeEv)
	if err != nil {
		t.Fatalf("append edge event failed: %v", err)
	}

	// Query edges
	edgesFrom, err := s.EdgesFrom(from, event.EdgeTypeTrust)
	if err != nil {
		t.Fatalf("EdgesFrom failed: %v", err)
	}
	if len(edgesFrom) != 1 {
		t.Errorf("expected 1 edge from, got %d", len(edgesFrom))
	}

	edgesTo, err := s.EdgesTo(to, event.EdgeTypeTrust)
	if err != nil {
		t.Fatalf("EdgesTo failed: %v", err)
	}
	if len(edgesTo) != 1 {
		t.Errorf("expected 1 edge to, got %d", len(edgesTo))
	}

	edgeBetween, err := s.EdgeBetween(from, to, event.EdgeTypeTrust)
	if err != nil {
		t.Fatalf("EdgeBetween failed: %v", err)
	}
	if !edgeBetween.IsSome() {
		t.Error("expected edge between actors")
	}
}

// Test chain integrity violation on wrong prev_hash
func TestAppendWrongPrevHash(t *testing.T) {
	s := store.NewInMemoryStore()
	ev := makeBootstrapEvent(t)
	s.Append(ev)

	// Create an event manually with wrong prev_hash
	wrongHash := types.MustHash("0000000000000000000000000000000000000000000000000000000000000000")
	badEv := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated,
		types.Now(),
		types.MustActorID("actor_00000000000000000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{ev.ID()},
		types.MustConversationID("conv_00000000000000000000000000000001"),
		wrongHash, // wrong hash
		wrongHash, // wrong prev_hash — doesn't match chain head
		types.MustSignature(make([]byte, 64)),
	)

	_, err := s.Append(badEv)
	if err == nil {
		t.Fatal("expected chain integrity violation")
	}
}

// Test Since with pagination cursor
func TestSincePagination(t *testing.T) {
	s := store.NewInMemoryStore()
	ev0 := makeBootstrapEvent(t)
	s.Append(ev0)

	prev := ev0
	for i := 0; i < 5; i++ {
		ev := makeEvent(t, s, event.EventTypeTrustUpdated, []types.EventID{prev.ID()})
		s.Append(ev)
		prev = ev
	}

	page, err := s.Since(ev0.ID(), 2)
	if err != nil {
		t.Fatalf("Since failed: %v", err)
	}
	if len(page.Items()) != 2 {
		t.Errorf("expected 2 items, got %d", len(page.Items()))
	}
	if !page.HasMore() {
		t.Error("should have more")
	}
}

// Test marker interface methods for coverage
func TestStoreErrorMarkerInterface(t *testing.T) {
	// These are no-op marker methods but we call them for coverage
	errs := []store.StoreError{
		&store.EventNotFoundError{ID: types.MustEventID("019462a0-0000-7000-8000-000000000001")},
		&store.ActorNotFoundError{ID: types.MustActorID("actor_test")},
		&store.EdgeNotFoundError{},
		&store.DuplicateEventError{},
		&store.CausalLinkMissingError{},
		&store.ChainIntegrityViolationError{},
		&store.HashMismatchError{},
		&store.SignatureInvalidError{},
		&store.ActorSuspendedError{},
		&store.ActorMemorialError{},
		&store.RateLimitExceededError{},
		&store.StoreUnavailableError{},
	}
	for _, err := range errs {
		_ = err.Error()
	}
}
