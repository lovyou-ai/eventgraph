package pgactor_test

import (
	"context"
	"os"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/actor/pgactor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/statestore/pgstate"
	"github.com/lovyou-ai/eventgraph/go/pkg/trust"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestRestartPersistence simulates a hive restart: create actors and trust state
// with one store instance, close it, open a new instance, verify state survived.
func TestRestartPersistence(t *testing.T) {
	connStr := os.Getenv("EVENTGRAPH_POSTGRES_URL")
	if connStr == "" {
		t.Skip("EVENTGRAPH_POSTGRES_URL not set; skipping restart test")
	}
	ctx := context.Background()

	// --- First session: create state ---

	actors1, err := pgactor.NewPostgresActorStore(ctx, connStr)
	if err != nil {
		t.Fatalf("session 1 actor store: %v", err)
	}
	actors1.Truncate(ctx) // clean slate

	states1, err := pgstate.NewPostgresStateStore(ctx, connStr)
	if err != nil {
		t.Fatalf("session 1 state store: %v", err)
	}
	states1.Truncate(ctx)

	// Register actors.
	pk1 := testPublicKey(10)
	pk2 := testPublicKey(20)
	human, _ := actors1.Register(pk1, "Matt", event.ActorTypeHuman)
	agent, _ := actors1.Register(pk2, "CTO", event.ActorTypeAI)

	// Build trust state.
	trustModel1 := trust.NewDefaultTrustModel()
	memActors := actor.NewInMemoryActorStore()
	humanMem, _ := memActors.Register(pk1, "Matt", event.ActorTypeHuman)
	agentMem, _ := memActors.Register(pk2, "CTO", event.ActorTypeAI)

	// Give CTO some trust.
	evID, _ := types.NewEventIDFromNew()
	ev := event.NewEvent(1, evID, event.EventTypeTrustUpdated, types.Now(),
		humanMem.ID(),
		event.TrustUpdatedContent{
			Actor:    agentMem.ID(),
			Previous: types.MustScore(0.0),
			Current:  types.MustScore(0.5),
			Domain:   types.MustDomainScope("general"),
			Cause:    types.MustEventID("019462a0-0000-7000-8000-000000000001"),
		},
		[]types.EventID{types.MustEventID("019462a0-0000-7000-8000-000000000001")},
		types.MustConversationID("conv_restart_test00000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)
	trustModel1.Update(ctx, agentMem, ev)

	// Persist trust.
	trustJSON, _ := trustModel1.ExportJSON()
	states1.Put("trust", "model", trustJSON)

	// Close everything (simulate shutdown).
	actors1.Close()
	states1.Close()

	// --- Second session: verify state survived ---

	actors2, err := pgactor.NewPostgresActorStore(ctx, connStr)
	if err != nil {
		t.Fatalf("session 2 actor store: %v", err)
	}
	defer actors2.Close()

	states2, err := pgstate.NewPostgresStateStore(ctx, connStr)
	if err != nil {
		t.Fatalf("session 2 state store: %v", err)
	}
	defer states2.Close()

	// Verify actors survived.
	humanRestored, err := actors2.Get(human.ID())
	if err != nil {
		t.Fatalf("human not found after restart: %v", err)
	}
	if humanRestored.DisplayName() != "Matt" {
		t.Errorf("human name = %q, want Matt", humanRestored.DisplayName())
	}

	agentRestored, err := actors2.Get(agent.ID())
	if err != nil {
		t.Fatalf("agent not found after restart: %v", err)
	}
	if agentRestored.DisplayName() != "CTO" {
		t.Errorf("agent name = %q, want CTO", agentRestored.DisplayName())
	}
	if agentRestored.Type() != event.ActorTypeAI {
		t.Errorf("agent type = %v, want AI", agentRestored.Type())
	}

	// Verify trust survived.
	trustData, err := states2.Get("trust", "model")
	if err != nil {
		t.Fatalf("trust state not found after restart: %v", err)
	}
	if trustData == nil {
		t.Fatal("trust state is nil after restart")
	}

	trustModel2 := trust.NewDefaultTrustModel()
	if err := trustModel2.ImportJSON(trustData); err != nil {
		t.Fatalf("ImportJSON: %v", err)
	}

	metrics, _ := trustModel2.Score(ctx, agentMem)
	if metrics.Overall().Value() <= 0.0 {
		t.Errorf("CTO trust should be > 0 after restart, got %v", metrics.Overall().Value())
	}
	t.Logf("CTO trust after restart: %.4f", metrics.Overall().Value())
}
