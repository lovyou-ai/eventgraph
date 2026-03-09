package pgstate_test

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/statestore/pgstate"
)

func setup(t *testing.T) *pgstate.PostgresStateStore {
	t.Helper()
	connStr := os.Getenv("EVENTGRAPH_POSTGRES_URL")
	if connStr == "" {
		t.Skip("EVENTGRAPH_POSTGRES_URL not set; skipping pgstate tests")
	}
	ctx := context.Background()
	s, err := pgstate.NewPostgresStateStore(ctx, connStr)
	if err != nil {
		t.Fatalf("NewPostgresStateStore: %v", err)
	}
	if err := s.Truncate(ctx); err != nil {
		t.Fatalf("Truncate: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestPutAndGet(t *testing.T) {
	s := setup(t)
	value := json.RawMessage(`{"score": 0.7}`)

	if err := s.Put("trust:actor_1", "score", value); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := s.Get("trust:actor_1", "score")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("Get = %s, want %s", got, value)
	}
}

func TestGetNotFound(t *testing.T) {
	s := setup(t)
	got, err := s.Get("nonexistent", "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for not found, got %s", got)
	}
}

func TestPutUpsert(t *testing.T) {
	s := setup(t)

	s.Put("scope", "key", json.RawMessage(`"v1"`))
	s.Put("scope", "key", json.RawMessage(`"v2"`))

	got, _ := s.Get("scope", "key")
	if string(got) != `"v2"` {
		t.Errorf("Upsert failed: got %s", got)
	}
}

func TestDelete(t *testing.T) {
	s := setup(t)

	s.Put("scope", "key", json.RawMessage(`"value"`))
	if err := s.Delete("scope", "key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, _ := s.Get("scope", "key")
	if got != nil {
		t.Errorf("expected nil after delete, got %s", got)
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := setup(t)
	// Should not error.
	if err := s.Delete("nonexistent", "key"); err != nil {
		t.Fatalf("Delete not-found: %v", err)
	}
}

func TestList(t *testing.T) {
	s := setup(t)

	s.Put("trust:actor_1", "score", json.RawMessage(`0.7`))
	s.Put("trust:actor_1", "trend", json.RawMessage(`0.1`))
	s.Put("trust:actor_2", "score", json.RawMessage(`0.3`))

	items, err := s.List("trust:actor_1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	if string(items["score"]) != "0.7" {
		t.Errorf("score = %s, want 0.7", items["score"])
	}
	if string(items["trend"]) != "0.1" {
		t.Errorf("trend = %s, want 0.1", items["trend"])
	}
}

func TestListScopes(t *testing.T) {
	s := setup(t)

	s.Put("trust:actor_1", "score", json.RawMessage(`0.7`))
	s.Put("trust:actor_2", "score", json.RawMessage(`0.3`))
	s.Put("agent:actor_1", "narrative", json.RawMessage(`"working on market graph"`))

	scopes, err := s.ListScopes("trust:")
	if err != nil {
		t.Fatalf("ListScopes: %v", err)
	}
	sort.Strings(scopes)
	if len(scopes) != 2 {
		t.Errorf("expected 2 trust scopes, got %d: %v", len(scopes), scopes)
	}

	agentScopes, _ := s.ListScopes("agent:")
	if len(agentScopes) != 1 {
		t.Errorf("expected 1 agent scope, got %d", len(agentScopes))
	}
}

func TestComplexJSON(t *testing.T) {
	s := setup(t)

	value := json.RawMessage(`{
		"score": 0.7,
		"by_domain": {"code_review": 0.8, "deployment": 0.5},
		"evidence": ["evt_1", "evt_2"],
		"trend": 0.1
	}`)

	s.Put("trust:actor_1", "state", value)

	got, _ := s.Get("trust:actor_1", "state")

	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed["score"] != 0.7 {
		t.Errorf("score = %v, want 0.7", parsed["score"])
	}
}
