package intelligence_test

import (
	"context"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/decision"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/intelligence"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// ════════════════════════════════════════════════════════════════════════
// Mock provider
// ════════════════════════════════════════════════════════════════════════

type mockProvider struct {
	response string
}

func newMockProvider(response string) *mockProvider {
	return &mockProvider{response: response}
}

func (m *mockProvider) Name() string  { return "mock" }
func (m *mockProvider) Model() string { return "mock-model" }
func (m *mockProvider) Reason(_ context.Context, _ string, _ []event.Event) (decision.Response, error) {
	confidence, _ := types.NewScore(0.8)
	return decision.NewResponse(m.response, confidence, 10), nil
}

var _ intelligence.Provider = (*mockProvider)(nil)

// ════════════════════════════════════════════════════════════════════════
// Runtime unit tests
// ════════════════════════════════════════════════════════════════════════

func TestRuntimeRequiresProvider(t *testing.T) {
	_, err := intelligence.NewRuntime(context.Background(), intelligence.RuntimeConfig{
		AgentID: types.MustActorID("actor_00000000000000000000000000000001"),
	})
	if err == nil {
		t.Fatal("expected error when provider is nil")
	}
}

func TestRuntimeBootstrapsGraph(t *testing.T) {
	p := newMockProvider("ok")
	agentID := types.MustActorID("actor_00000000000000000000000000000001")

	rt, err := intelligence.NewRuntime(context.Background(), intelligence.RuntimeConfig{
		AgentID:  agentID,
		Provider: p,
	})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}

	head, err := rt.Store().Head()
	if err != nil {
		t.Fatalf("Head failed: %v", err)
	}
	if !head.IsSome() {
		t.Fatal("store should have bootstrap event")
	}
	if head.Unwrap().Type().Value() != "system.bootstrapped" {
		t.Errorf("head type = %q, want system.bootstrapped", head.Unwrap().Type().Value())
	}
}

func TestRuntimeBootComposition(t *testing.T) {
	rt := mustRuntime(t)
	grantor := types.MustActorID("actor_00000000000000000000000000000099")
	scope := types.MustDomainScope("test")

	events, err := rt.Boot("ai", "claude-sonnet-4-6", "standard", []string{"Take care"}, scope, grantor)
	if err != nil {
		t.Fatalf("Boot failed: %v", err)
	}

	if len(events) != 5 {
		t.Fatalf("Boot emitted %d events, want 5", len(events))
	}

	expectedTypes := []string{
		"agent.identity.created",
		"agent.soul.imprinted",
		"agent.model.bound",
		"agent.authority.granted",
		"agent.state.changed",
	}
	for i, ev := range events {
		if ev.Type().Value() != expectedTypes[i] {
			t.Errorf("event[%d] type = %q, want %q", i, ev.Type().Value(), expectedTypes[i])
		}
	}
}

func TestRuntimeObserve(t *testing.T) {
	rt := mustRuntime(t)
	ev, err := rt.Observe(context.Background(), 5)
	if err != nil {
		t.Fatalf("Observe failed: %v", err)
	}
	if ev.Type().Value() != "agent.observed" {
		t.Errorf("type = %q, want agent.observed", ev.Type().Value())
	}
}

func TestRuntimeEvaluate(t *testing.T) {
	rt := mustRuntime(t)
	ev, result, err := rt.Evaluate(context.Background(), "test-subject", "evaluate this")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if ev.Type().Value() != "agent.evaluated" {
		t.Errorf("type = %q, want agent.evaluated", ev.Type().Value())
	}
	if result != "ok" {
		t.Errorf("result = %q, want ok", result)
	}
}

func TestRuntimeDecide(t *testing.T) {
	rt := mustRuntime(t)
	ev, result, err := rt.Decide(context.Background(), "test-action", "decide this")
	if err != nil {
		t.Fatalf("Decide failed: %v", err)
	}
	if ev.Type().Value() != "agent.decided" {
		t.Errorf("type = %q, want agent.decided", ev.Type().Value())
	}
	if result != "ok" {
		t.Errorf("result = %q, want ok", result)
	}
}

func TestRuntimeAct(t *testing.T) {
	rt := mustRuntime(t)
	ev, err := rt.Act(context.Background(), "write_code", "main.go")
	if err != nil {
		t.Fatalf("Act failed: %v", err)
	}
	if ev.Type().Value() != "agent.acted" {
		t.Errorf("type = %q, want agent.acted", ev.Type().Value())
	}
}

func TestRuntimeLearn(t *testing.T) {
	rt := mustRuntime(t)
	ev, err := rt.Learn(context.Background(), "Always check deps", "task")
	if err != nil {
		t.Fatalf("Learn failed: %v", err)
	}
	if ev.Type().Value() != "agent.learned" {
		t.Errorf("type = %q, want agent.learned", ev.Type().Value())
	}
}

func TestRuntimeRefuse(t *testing.T) {
	rt := mustRuntime(t)
	ev, err := rt.Refuse(context.Background(), "delete_logs", "violates invariant")
	if err != nil {
		t.Fatalf("Refuse failed: %v", err)
	}
	if ev.Type().Value() != "agent.refused" {
		t.Errorf("type = %q, want agent.refused", ev.Type().Value())
	}
}

func TestRuntimeEscalate(t *testing.T) {
	rt := mustRuntime(t)
	authority := types.MustActorID("actor_00000000000000000000000000000099")
	ev, err := rt.Escalate(context.Background(), authority, "needs human")
	if err != nil {
		t.Fatalf("Escalate failed: %v", err)
	}
	if ev.Type().Value() != "agent.escalated" {
		t.Errorf("type = %q, want agent.escalated", ev.Type().Value())
	}
}

func TestRuntimeMemory(t *testing.T) {
	rt := mustRuntime(t)
	ctx := context.Background()

	rt.Observe(ctx, 1)
	rt.Act(ctx, "test", "target")
	rt.Learn(ctx, "lesson", "source")

	events, err := rt.Memory(10)
	if err != nil {
		t.Fatalf("Memory failed: %v", err)
	}
	// Bootstrap + 3 agent events = 4 minimum.
	if len(events) < 4 {
		t.Errorf("Memory returned %d events, want >= 4", len(events))
	}
}

func TestRuntimeEventsByType(t *testing.T) {
	rt := mustRuntime(t)
	ctx := context.Background()

	rt.Observe(ctx, 1)
	rt.Observe(ctx, 2)
	rt.Act(ctx, "test", "target")

	events, err := rt.EventsByType("agent.observed", 10)
	if err != nil {
		t.Fatalf("EventsByType failed: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("got %d agent.observed events, want 2", len(events))
	}
}

func TestRuntimeHashChainIntegrity(t *testing.T) {
	rt := mustRuntime(t)
	ctx := context.Background()

	rt.Observe(ctx, 1)
	rt.Act(ctx, "test", "target")
	rt.Learn(ctx, "lesson", "source")

	page, err := rt.Store().Recent(100, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("Recent failed: %v", err)
	}

	events := page.Items()
	// Recent returns newest-first; reverse for chain order.
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	for i := 1; i < len(events); i++ {
		if events[i].PrevHash() != events[i-1].Hash() {
			t.Errorf("hash chain broken at event %d", i)
		}
	}
	t.Logf("Hash chain intact: %d events", len(events))
}

func TestRuntimeRunTask(t *testing.T) {
	rt := mustRuntime(t)
	ctx := context.Background()

	result, err := rt.RunTask(ctx, "Review the authentication module")
	if err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}

	if len(result.Events) != 5 {
		t.Errorf("RunTask emitted %d events, want 5", len(result.Events))
	}
	if result.Evaluation == "" {
		t.Error("evaluation is empty")
	}
	if result.Decision == "" {
		t.Error("decision is empty")
	}
	if result.Lesson == "" {
		t.Error("lesson is empty")
	}
}

func TestRuntimeCustomStore(t *testing.T) {
	p := newMockProvider("ok")
	agentID := types.MustActorID("actor_00000000000000000000000000000001")
	s := store.NewInMemoryStore()

	rt, err := intelligence.NewRuntime(context.Background(), intelligence.RuntimeConfig{
		AgentID:  agentID,
		Provider: p,
		Store:    s,
	})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}

	if rt.Store() != s {
		t.Error("runtime should use the provided store")
	}
}

// ════════════════════════════════════════════════════════════════════════
// Helper
// ════════════════════════════════════════════════════════════════════════

func mustRuntime(t *testing.T) *intelligence.AgentRuntime {
	t.Helper()
	p := newMockProvider("ok")
	agentID := types.MustActorID("actor_00000000000000000000000000000001")
	rt, err := intelligence.NewRuntime(context.Background(), intelligence.RuntimeConfig{
		AgentID:  agentID,
		Provider: p,
	})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	return rt
}
