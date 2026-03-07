package actor_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

func testPublicKey(b byte) types.PublicKey {
	key := make([]byte, 32)
	key[0] = b
	return types.MustPublicKey(key)
}

func TestRegister(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)

	a, err := s.Register(pk, "Alice", event.ActorTypeHuman)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if a.DisplayName() != "Alice" {
		t.Errorf("DisplayName = %q, want Alice", a.DisplayName())
	}
	if a.Type() != event.ActorTypeHuman {
		t.Errorf("Type = %v, want Human", a.Type())
	}
	if a.Status() != types.ActorStatusActive {
		t.Errorf("Status = %v, want Active", a.Status())
	}
}

func TestRegisterIdempotent(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)

	a1, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	a2, _ := s.Register(pk, "Alice Again", event.ActorTypeHuman)

	if a1.ID() != a2.ID() {
		t.Error("idempotent register should return same actor")
	}
	if s.ActorCount() != 1 {
		t.Errorf("should have 1 actor, got %d", s.ActorCount())
	}
}

func TestGet(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)

	got, err := s.Get(a.ID())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID() != a.ID() {
		t.Error("Get returned wrong actor")
	}
}

func TestGetNotFound(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	_, err := s.Get(types.MustActorID("actor_nonexistent"))
	if err == nil {
		t.Fatal("expected error")
	}
	var notFound *store.ActorNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected ActorNotFoundError, got %T", err)
	}
}

func TestGetByPublicKey(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)

	got, err := s.GetByPublicKey(pk)
	if err != nil {
		t.Fatalf("GetByPublicKey failed: %v", err)
	}
	if got.ID() != a.ID() {
		t.Error("GetByPublicKey returned wrong actor")
	}
}

func TestGetByPublicKeyNotFound(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	_, err := s.GetByPublicKey(testPublicKey(99))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)

	updated, err := s.Update(a.ID(), actor.ActorUpdate{
		DisplayName: types.Some("Alice Updated"),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.DisplayName() != "Alice Updated" {
		t.Errorf("DisplayName = %q, want Alice Updated", updated.DisplayName())
	}
}

func TestUpdateMetadataMerge(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)

	// Set initial metadata
	s.Update(a.ID(), actor.ActorUpdate{
		Metadata: types.Some(map[string]any{"role": "builder"}),
	})

	// Merge additional metadata
	updated, err := s.Update(a.ID(), actor.ActorUpdate{
		Metadata: types.Some(map[string]any{"team": "core"}),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	md := updated.Metadata()
	if md["role"] != "builder" {
		t.Error("existing metadata should be preserved")
	}
	if md["team"] != "core" {
		t.Error("new metadata should be added")
	}
}

func TestUpdateNotFound(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	_, err := s.Update(types.MustActorID("actor_nonexistent"), actor.ActorUpdate{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSuspend(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	suspended, err := s.Suspend(a.ID(), reason)
	if err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}
	if suspended.Status() != types.ActorStatusSuspended {
		t.Errorf("Status = %v, want Suspended", suspended.Status())
	}
}

func TestSuspendNotFound(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	_, err := s.Suspend(types.MustActorID("actor_nonexistent"), reason)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMemorial(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	memorial, err := s.Memorial(a.ID(), reason)
	if err != nil {
		t.Fatalf("Memorial failed: %v", err)
	}
	if memorial.Status() != types.ActorStatusMemorial {
		t.Errorf("Status = %v, want Memorial", memorial.Status())
	}
}

func TestMemorialIsTerminal(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	s.Memorial(a.ID(), reason)

	// Try to reactivate — should fail
	_, err := s.Suspend(a.ID(), reason)
	if err == nil {
		t.Fatal("expected error transitioning from Memorial")
	}
}

func TestMemorialNotFound(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	_, err := s.Memorial(types.MustActorID("actor_nonexistent"), reason)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSuspendAndReactivate(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	s.Suspend(a.ID(), reason)

	got, _ := s.Get(a.ID())
	if got.Status() != types.ActorStatusSuspended {
		t.Errorf("Status after Suspend = %v, want Suspended", got.Status())
	}

	// Reactivate — Suspended → Active
	reactivated, err := s.Reactivate(a.ID(), reason)
	if err != nil {
		t.Fatalf("Reactivate: %v", err)
	}
	if reactivated.Status() != types.ActorStatusActive {
		t.Errorf("Status after Reactivate = %v, want Active", reactivated.Status())
	}
}

func TestReactivateNotFound(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	_, err := s.Reactivate(types.MustActorID("actor_nonexistent0000000000000001"), reason)
	if err == nil {
		t.Error("expected error for nonexistent actor")
	}
}

func TestMemorialReactivateIsError(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	s.Memorial(a.ID(), reason)

	_, err := s.Reactivate(a.ID(), reason)
	if err == nil {
		t.Fatal("expected error reactivating a memorialised actor")
	}
}

func TestReactivateFromActiveIsError(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	_, err := s.Reactivate(a.ID(), reason)
	if err == nil {
		t.Error("expected error when reactivating already-active actor")
	}
}

func TestList(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	for i := byte(1); i <= 5; i++ {
		s.Register(testPublicKey(i), "Actor", event.ActorTypeHuman)
	}

	page, err := s.List(actor.ActorFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(page.Items()) != 5 {
		t.Errorf("expected 5 actors, got %d", len(page.Items()))
	}
}

func TestListWithStatusFilter(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	a1, _ := s.Register(testPublicKey(1), "Active", event.ActorTypeHuman)
	s.Register(testPublicKey(2), "Active2", event.ActorTypeHuman)
	s.Register(testPublicKey(3), "Suspended", event.ActorTypeHuman)

	// Suspend the third
	a3, _ := s.Get(a1.ID()) // we need actor 3's ID
	_ = a3
	// Register and suspend actor 3
	actors, _ := s.List(actor.ActorFilter{Limit: 10})
	if len(actors.Items()) >= 3 {
		s.Suspend(actors.Items()[2].ID(), reason)
	}

	// Filter by active only
	activePage, err := s.List(actor.ActorFilter{
		Status: types.Some(types.ActorStatusActive),
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("List with filter failed: %v", err)
	}
	if len(activePage.Items()) != 2 {
		t.Errorf("expected 2 active actors, got %d", len(activePage.Items()))
	}
}

func TestListWithTypeFilter(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	s.Register(testPublicKey(1), "Human", event.ActorTypeHuman)
	s.Register(testPublicKey(2), "AI", event.ActorTypeAI)
	s.Register(testPublicKey(3), "System", event.ActorTypeSystem)

	page, err := s.List(actor.ActorFilter{
		Type:  types.Some(event.ActorTypeAI),
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("List with type filter failed: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Errorf("expected 1 AI actor, got %d", len(page.Items()))
	}
}

func TestListPagination(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	for i := byte(1); i <= 5; i++ {
		s.Register(testPublicKey(i), "Actor", event.ActorTypeHuman)
	}

	// Page 1
	page1, _ := s.List(actor.ActorFilter{Limit: 2})
	if len(page1.Items()) != 2 {
		t.Fatalf("expected 2 items in page 1, got %d", len(page1.Items()))
	}
	if !page1.HasMore() {
		t.Error("page 1 should have more")
	}

	// Page 2
	page2, _ := s.List(actor.ActorFilter{Limit: 2, After: page1.Cursor()})
	if len(page2.Items()) != 2 {
		t.Fatalf("expected 2 items in page 2, got %d", len(page2.Items()))
	}
}

func TestConcurrentRegister(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	var wg sync.WaitGroup
	for i := byte(1); i <= 10; i++ {
		wg.Add(1)
		go func(b byte) {
			defer wg.Done()
			s.Register(testPublicKey(b), "Actor", event.ActorTypeHuman)
		}(i)
	}
	wg.Wait()

	if s.ActorCount() != 10 {
		t.Errorf("expected 10 actors, got %d", s.ActorCount())
	}
}

func TestActorGetters(t *testing.T) {
	s := actor.NewInMemoryActorStore()
	pk := testPublicKey(42)
	a, _ := s.Register(pk, "Alice", event.ActorTypeAI)

	if a.ID().Value() == "" {
		t.Error("ID should not be empty")
	}
	_ = a.PublicKey()
	_ = a.CreatedAt()
	_ = a.Metadata()
}
