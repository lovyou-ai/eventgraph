package pgactor_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/actor/pgactor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

func testPublicKey(b byte) types.PublicKey {
	key := make([]byte, 32)
	key[0] = b
	return types.MustPublicKey(key)
}

func setup(t *testing.T) *pgactor.PostgresActorStore {
	t.Helper()
	connStr := os.Getenv("EVENTGRAPH_POSTGRES_URL")
	if connStr == "" {
		t.Skip("EVENTGRAPH_POSTGRES_URL not set; skipping pgactor tests")
	}
	ctx := context.Background()
	s, err := pgactor.NewPostgresActorStore(ctx, connStr)
	if err != nil {
		t.Fatalf("NewPostgresActorStore: %v", err)
	}
	// Clean slate for each test.
	if err := s.Truncate(ctx); err != nil {
		t.Fatalf("Truncate: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestRegister(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)

	a, err := s.Register(pk, "Alice", event.ActorTypeHuman)
	if err != nil {
		t.Fatalf("Register: %v", err)
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
	if a.ID().Value() == "" {
		t.Error("ID should not be empty")
	}
}

func TestRegisterIdempotent(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)

	a1, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	a2, _ := s.Register(pk, "Alice Again", event.ActorTypeHuman)

	if a1.ID() != a2.ID() {
		t.Error("idempotent register should return same actor")
	}
	// Display name should NOT change on re-register.
	if a2.DisplayName() != "Alice" {
		t.Errorf("DisplayName should remain Alice, got %q", a2.DisplayName())
	}
}

func TestGet(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)

	got, err := s.Get(a.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID() != a.ID() {
		t.Error("Get returned wrong actor")
	}
	if got.DisplayName() != "Alice" {
		t.Errorf("DisplayName = %q, want Alice", got.DisplayName())
	}
}

func TestGetNotFound(t *testing.T) {
	s := setup(t)
	_, err := s.Get(types.MustActorID("actor_nonexistent00000000000000001"))
	if err == nil {
		t.Fatal("expected error")
	}
	var notFound *store.ActorNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected ActorNotFoundError, got %T: %v", err, err)
	}
}

func TestGetByPublicKey(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)

	got, err := s.GetByPublicKey(pk)
	if err != nil {
		t.Fatalf("GetByPublicKey: %v", err)
	}
	if got.ID() != a.ID() {
		t.Error("GetByPublicKey returned wrong actor")
	}
}

func TestGetByPublicKeyNotFound(t *testing.T) {
	s := setup(t)
	_, err := s.GetByPublicKey(testPublicKey(99))
	if err == nil {
		t.Fatal("expected error")
	}
	var notFound *store.ActorKeyNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected ActorKeyNotFoundError, got %T: %v", err, err)
	}
}

func TestUpdate(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)

	updated, err := s.Update(a.ID(), actor.ActorUpdate{
		DisplayName: types.Some("Alice Updated"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.DisplayName() != "Alice Updated" {
		t.Errorf("DisplayName = %q, want Alice Updated", updated.DisplayName())
	}

	// Verify persistence.
	got, _ := s.Get(a.ID())
	if got.DisplayName() != "Alice Updated" {
		t.Errorf("persisted DisplayName = %q, want Alice Updated", got.DisplayName())
	}
}

func TestUpdateMetadataMerge(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)

	// Set initial metadata.
	s.Update(a.ID(), actor.ActorUpdate{
		Metadata: types.Some(map[string]any{"role": "builder"}),
	})

	// Merge additional metadata.
	updated, err := s.Update(a.ID(), actor.ActorUpdate{
		Metadata: types.Some(map[string]any{"team": "core"}),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
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
	s := setup(t)
	_, err := s.Update(types.MustActorID("actor_nonexistent00000000000000001"), actor.ActorUpdate{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSuspend(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	suspended, err := s.Suspend(a.ID(), reason)
	if err != nil {
		t.Fatalf("Suspend: %v", err)
	}
	if suspended.Status() != types.ActorStatusSuspended {
		t.Errorf("Status = %v, want Suspended", suspended.Status())
	}

	// Verify persistence.
	got, _ := s.Get(a.ID())
	if got.Status() != types.ActorStatusSuspended {
		t.Errorf("persisted Status = %v, want Suspended", got.Status())
	}
}

func TestSuspendNotFound(t *testing.T) {
	s := setup(t)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	_, err := s.Suspend(types.MustActorID("actor_nonexistent00000000000000001"), reason)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSuspendAndReactivate(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	s.Suspend(a.ID(), reason)

	got, _ := s.Get(a.ID())
	if got.Status() != types.ActorStatusSuspended {
		t.Errorf("Status after Suspend = %v, want Suspended", got.Status())
	}

	reactivated, err := s.Reactivate(a.ID(), reason)
	if err != nil {
		t.Fatalf("Reactivate: %v", err)
	}
	if reactivated.Status() != types.ActorStatusActive {
		t.Errorf("Status after Reactivate = %v, want Active", reactivated.Status())
	}
}

func TestReactivateNotFound(t *testing.T) {
	s := setup(t)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	_, err := s.Reactivate(types.MustActorID("actor_nonexistent00000000000000001"), reason)
	if err == nil {
		t.Error("expected error for nonexistent actor")
	}
}

func TestReactivateFromActiveIsError(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	_, err := s.Reactivate(a.ID(), reason)
	if err == nil {
		t.Error("expected error when reactivating already-active actor")
	}
}

func TestMemorial(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	memorial, err := s.Memorial(a.ID(), reason)
	if err != nil {
		t.Fatalf("Memorial: %v", err)
	}
	if memorial.Status() != types.ActorStatusMemorial {
		t.Errorf("Status = %v, want Memorial", memorial.Status())
	}
}

func TestMemorialIsTerminal(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	s.Memorial(a.ID(), reason)

	_, err := s.Suspend(a.ID(), reason)
	if err == nil {
		t.Fatal("expected error transitioning from Memorial")
	}
}

func TestMemorialReactivateIsError(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	s.Memorial(a.ID(), reason)

	_, err := s.Reactivate(a.ID(), reason)
	if err == nil {
		t.Fatal("expected error reactivating a memorialised actor")
	}
}

func TestMemorialNotFound(t *testing.T) {
	s := setup(t)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	_, err := s.Memorial(types.MustActorID("actor_nonexistent00000000000000001"), reason)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestList(t *testing.T) {
	s := setup(t)
	for i := byte(1); i <= 5; i++ {
		s.Register(testPublicKey(i), "Actor", event.ActorTypeHuman)
	}

	page, err := s.List(actor.ActorFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items()) != 5 {
		t.Errorf("expected 5 actors, got %d", len(page.Items()))
	}
}

func TestListWithStatusFilter(t *testing.T) {
	s := setup(t)
	reason := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	a1, _ := s.Register(testPublicKey(1), "Active1", event.ActorTypeHuman)
	s.Register(testPublicKey(2), "Active2", event.ActorTypeHuman)
	s.Register(testPublicKey(3), "ToBeSuspended", event.ActorTypeHuman)

	// Suspend actor 3. List all, find 3rd, suspend.
	all, _ := s.List(actor.ActorFilter{Limit: 10})
	var thirdID types.ActorID
	for _, a := range all.Items() {
		if a.ID() != a1.ID() && a.DisplayName() != "Active2" {
			thirdID = a.ID()
			break
		}
	}
	if thirdID.Value() != "" {
		s.Suspend(thirdID, reason)
	}

	activePage, err := s.List(actor.ActorFilter{
		Status: types.Some(types.ActorStatusActive),
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("List with filter: %v", err)
	}
	if len(activePage.Items()) != 2 {
		t.Errorf("expected 2 active actors, got %d", len(activePage.Items()))
	}
}

func TestListWithTypeFilter(t *testing.T) {
	s := setup(t)
	s.Register(testPublicKey(1), "Human", event.ActorTypeHuman)
	s.Register(testPublicKey(2), "AI", event.ActorTypeAI)
	s.Register(testPublicKey(3), "System", event.ActorTypeSystem)

	page, err := s.List(actor.ActorFilter{
		Type:  types.Some(event.ActorTypeAI),
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("List with type filter: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Errorf("expected 1 AI actor, got %d", len(page.Items()))
	}
}

func TestListPagination(t *testing.T) {
	s := setup(t)
	for i := byte(1); i <= 5; i++ {
		s.Register(testPublicKey(i), "Actor", event.ActorTypeHuman)
	}

	// Page 1.
	page1, err := s.List(actor.ActorFilter{Limit: 2})
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if len(page1.Items()) != 2 {
		t.Fatalf("expected 2 items in page 1, got %d", len(page1.Items()))
	}
	if !page1.HasMore() {
		t.Error("page 1 should have more")
	}

	// Page 2.
	page2, err := s.List(actor.ActorFilter{Limit: 2, After: page1.Cursor()})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(page2.Items()) != 2 {
		t.Fatalf("expected 2 items in page 2, got %d", len(page2.Items()))
	}

	// Page 3 — last page.
	page3, err := s.List(actor.ActorFilter{Limit: 2, After: page2.Cursor()})
	if err != nil {
		t.Fatalf("List page 3: %v", err)
	}
	if len(page3.Items()) != 1 {
		t.Errorf("expected 1 item in page 3, got %d", len(page3.Items()))
	}
	if page3.HasMore() {
		t.Error("page 3 should not have more")
	}
}

func TestActorIDConsistency(t *testing.T) {
	// Verify pgactor derives the same ActorID as InMemoryActorStore.
	s := setup(t)
	memStore := actor.NewInMemoryActorStore()
	pk := testPublicKey(42)

	pgActor, err := s.Register(pk, "Test", event.ActorTypeHuman)
	if err != nil {
		t.Fatalf("pg Register: %v", err)
	}

	memActor, err := memStore.Register(pk, "Test", event.ActorTypeHuman)
	if err != nil {
		t.Fatalf("mem Register: %v", err)
	}

	if pgActor.ID() != memActor.ID() {
		t.Errorf("ID mismatch: pg=%s mem=%s", pgActor.ID().Value(), memActor.ID().Value())
	}
}

func TestPublicKeyRoundTrip(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(42)
	a, _ := s.Register(pk, "Test", event.ActorTypeHuman)

	got, _ := s.Get(a.ID())
	if !bytesEqual(got.PublicKey().Bytes(), pk.Bytes()) {
		t.Error("public key should round-trip through Postgres")
	}
}

func TestMetadataRoundTrip(t *testing.T) {
	s := setup(t)
	pk := testPublicKey(1)
	a, _ := s.Register(pk, "Alice", event.ActorTypeHuman)

	s.Update(a.ID(), actor.ActorUpdate{
		Metadata: types.Some(map[string]any{
			"role":  "builder",
			"level": float64(3),
			"tags":  []any{"go", "rust"},
		}),
	})

	got, _ := s.Get(a.ID())
	md := got.Metadata()
	if md["role"] != "builder" {
		t.Errorf("role = %v, want builder", md["role"])
	}
	if md["level"] != float64(3) {
		t.Errorf("level = %v, want 3", md["level"])
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
