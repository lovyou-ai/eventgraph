package intelligence_test

import (
	"context"
	"strings"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/intelligence"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// ════════════════════════════════════════════════════════════════════════
// Code Graph integration tests: agents describe, update, and build from specs
// ════════════════════════════════════════════════════════════════════════

func codegraphRuntime(t *testing.T) *intelligence.AgentRuntime {
	t.Helper()
	p := newMockProvider("ok")
	agentID := types.MustActorID("actor_00000000000000000000000000000001")
	rt, err := intelligence.NewRuntime(context.Background(), intelligence.RuntimeConfig{
		AgentID:  agentID,
		Provider: p,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	return rt
}

// TestCodeGraphDescribeProject verifies that an agent can describe a project
// by recording codegraph events: entity definitions, state machines, views.
func TestCodeGraphDescribeProject(t *testing.T) {
	rt := codegraphRuntime(t)
	agentID := rt.ID()
	s := rt.Store()

	// 1. Define the Task entity
	ev1, err := emitCodeGraph(rt, event.CodeGraphEntityDefinedContent{
		EntityType: "Task",
		EntityID:   agentID,
	})
	if err != nil {
		t.Fatalf("emit entity.defined: %v", err)
	}
	if ev1.Type().Value() != "codegraph.entity.defined" {
		t.Errorf("type = %q, want codegraph.entity.defined", ev1.Type().Value())
	}

	// 2. Define the state machine
	ev2, err := emitCodeGraph(rt, event.CodeGraphStateTransitionedContent{
		EntityType: "Task",
		EntityID:   agentID,
		From:       "todo",
		To:         "doing",
	})
	if err != nil {
		t.Fatalf("emit state.transitioned: %v", err)
	}
	if ev2.Type().Value() != "codegraph.state.transitioned" {
		t.Errorf("type = %q, want codegraph.state.transitioned", ev2.Type().Value())
	}

	// 3. Define a view
	ev3, err := emitCodeGraph(rt, event.CodeGraphViewRenderedContent{
		ViewName: "TaskBoard",
		Actor:    agentID,
	})
	if err != nil {
		t.Fatalf("emit view.rendered: %v", err)
	}

	// Verify all events are on the graph
	page, err := s.ByType(types.MustEventType("codegraph.entity.defined"), 10, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType: %v", err)
	}
	if len(page.Items()) != 1 {
		t.Errorf("entity.defined events = %d, want 1", len(page.Items()))
	}

	// Verify causal chain: ev3 → ev2 → ev1 → bootstrap
	if len(ev2.Causes()) == 0 || ev2.Causes()[0] != ev1.ID() {
		t.Errorf("ev2 should cause-link to ev1")
	}
	if len(ev3.Causes()) == 0 || ev3.Causes()[0] != ev2.ID() {
		t.Errorf("ev3 should cause-link to ev2")
	}
}

// TestCodeGraphUpdateDesign verifies that an agent can modify an existing spec
// and the changes are recorded as events with causal links to the original.
func TestCodeGraphUpdateDesign(t *testing.T) {
	rt := codegraphRuntime(t)
	agentID := rt.ID()
	s := rt.Store()

	// Original: define entity
	evDefine, err := emitCodeGraph(rt, event.CodeGraphEntityDefinedContent{
		EntityType: "Task",
		EntityID:   agentID,
	})
	if err != nil {
		t.Fatalf("define: %v", err)
	}

	// Modification: add a property
	evModify, err := emitCodeGraph(rt, event.CodeGraphEntityModifiedContent{
		EntityType: "Task",
		EntityID:   agentID,
		Property:   "priority",
		Previous:   types.None[string](),
		Current:    "high|medium|low",
	})
	if err != nil {
		t.Fatalf("modify: %v", err)
	}

	// The modification causally follows the definition
	if evModify.Causes()[0] != evDefine.ID() {
		t.Errorf("modify should cause-link to define")
	}

	// Second modification: change the property
	evModify2, err := emitCodeGraph(rt, event.CodeGraphEntityModifiedContent{
		EntityType: "Task",
		EntityID:   agentID,
		Property:   "priority",
		Previous:   types.Some("high|medium|low"),
		Current:    "critical|high|medium|low",
	})
	if err != nil {
		t.Fatalf("modify2: %v", err)
	}

	// Query all modifications
	page, err := s.ByType(types.MustEventType("codegraph.entity.modified"), 10, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType: %v", err)
	}
	if len(page.Items()) != 2 {
		t.Errorf("entity.modified events = %d, want 2", len(page.Items()))
	}

	// Chain: modify2 → modify → define → bootstrap
	if evModify2.Causes()[0] != evModify.ID() {
		t.Errorf("modify2 should cause-link to modify")
	}
}

// TestCodeGraphAgentGeneratesCode verifies the end-to-end flow: agent reads
// a spec from the graph, evaluates it with LLM, and produces code.
func TestCodeGraphAgentGeneratesCode(t *testing.T) {
	// Use a mock provider that returns "React code" for any prompt
	p := newMockProvider("function TaskBoard() { return <div>Board</div>; }")
	agentID := types.MustActorID("actor_00000000000000000000000000000001")
	rt, err := intelligence.NewRuntime(context.Background(), intelligence.RuntimeConfig{
		AgentID:  agentID,
		Provider: p,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	ctx := context.Background()

	// 1. Agent records the spec as codegraph events
	_, err = emitCodeGraph(rt, event.CodeGraphEntityDefinedContent{
		EntityType: "Task",
		EntityID:   agentID,
	})
	if err != nil {
		t.Fatalf("define entity: %v", err)
	}
	_, err = emitCodeGraph(rt, event.CodeGraphViewRenderedContent{
		ViewName: "TaskBoard",
		Actor:    agentID,
	})
	if err != nil {
		t.Fatalf("define view: %v", err)
	}

	// 2. Agent reads its memory (spec events) and generates code
	code, err := rt.CodeWrite(ctx, "Implement TaskBoard view from the Code Graph spec", "typescript")
	if err != nil {
		t.Fatalf("CodeWrite: %v", err)
	}
	if !strings.Contains(code, "TaskBoard") {
		t.Errorf("generated code should mention TaskBoard, got: %s", code)
	}

	// 3. Verify the act event was recorded
	page, err := rt.EventsByType("agent.acted", 10)
	if err != nil {
		t.Fatalf("EventsByType: %v", err)
	}
	if len(page) < 1 {
		t.Fatal("expected at least one agent.acted event")
	}
	// The acted event comes after the codegraph events in the chain
	actedEvent := page[len(page)-1]
	if actedEvent.Type().Value() != "agent.acted" {
		t.Errorf("type = %q, want agent.acted", actedEvent.Type().Value())
	}
}

// TestCodeGraphHashChainIntegrity verifies the full hash chain across
// codegraph events mixed with agent events.
func TestCodeGraphHashChainIntegrity(t *testing.T) {
	rt := codegraphRuntime(t)
	agentID := rt.ID()
	ctx := context.Background()

	// Mix codegraph and agent events, collecting each for chain verification
	var chain []event.Event

	ev, _ := emitCodeGraph(rt, event.CodeGraphEntityDefinedContent{
		EntityType: "Task", EntityID: agentID,
	})
	chain = append(chain, ev)

	ev, _ = rt.Observe(ctx, 1)
	chain = append(chain, ev)

	ev, _ = emitCodeGraph(rt, event.CodeGraphStateTransitionedContent{
		EntityType: "Task", EntityID: agentID, From: "todo", To: "doing",
	})
	chain = append(chain, ev)

	ev, _ = emitCodeGraph(rt, event.CodeGraphViewRenderedContent{
		ViewName: "Board", Actor: agentID,
	})
	chain = append(chain, ev)

	ev, _ = rt.Observe(ctx, 2)
	chain = append(chain, ev)

	// Verify chain: each event's PrevHash should match prior event's Hash
	for i := 1; i < len(chain); i++ {
		if chain[i].PrevHash() != chain[i-1].Hash() {
			t.Errorf("event %d PrevHash != event %d Hash (chain broken at %s → %s)",
				i, i-1, chain[i].Type().Value(), chain[i-1].Type().Value())
		}
	}
}

// TestCodeGraphQueryByType verifies that codegraph events are queryable
// by their specific event types.
func TestCodeGraphQueryByType(t *testing.T) {
	rt := codegraphRuntime(t)
	agentID := rt.ID()
	s := rt.Store()

	// Record various codegraph events
	_, _ = emitCodeGraph(rt, event.CodeGraphEntityDefinedContent{
		EntityType: "Task", EntityID: agentID,
	})
	_, _ = emitCodeGraph(rt, event.CodeGraphEntityDefinedContent{
		EntityType: "User", EntityID: agentID,
	})
	_, _ = emitCodeGraph(rt, event.CodeGraphViewRenderedContent{
		ViewName: "Board", Actor: agentID,
	})
	_, _ = emitCodeGraph(rt, event.CodeGraphCommandExecutedContent{
		Command: "create", EntityType: "Task", EntityID: agentID, Actor: agentID,
	})

	// Query entities only
	entities, err := s.ByType(types.MustEventType("codegraph.entity.defined"), 10, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType entity.defined: %v", err)
	}
	if len(entities.Items()) != 2 {
		t.Errorf("entity.defined = %d, want 2", len(entities.Items()))
	}

	// Query views only
	views, err := s.ByType(types.MustEventType("codegraph.ui.view.rendered"), 10, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType view.rendered: %v", err)
	}
	if len(views.Items()) != 1 {
		t.Errorf("view.rendered = %d, want 1", len(views.Items()))
	}

	// Query commands
	cmds, err := s.ByType(types.MustEventType("codegraph.io.command.executed"), 10, types.None[types.Cursor]())
	if err != nil {
		t.Fatalf("ByType command.executed: %v", err)
	}
	if len(cmds.Items()) != 1 {
		t.Errorf("command.executed = %d, want 1", len(cmds.Items()))
	}
}

// TestCodeGraphRealLLMDescribeProject uses a real LLM to describe a project
// in Code Graph vocabulary, then queries the resulting events.
func TestCodeGraphRealLLMDescribeProject(t *testing.T) {
	skipWithoutClaudeCli(t)

	p := agentProvider(t)
	agentID := types.MustActorID("actor_00000000000000000000000000000001")
	rt, err := intelligence.NewRuntime(context.Background(), intelligence.RuntimeConfig{
		AgentID:  agentID,
		Provider: p,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	ctx := context.Background()

	// 1. Agent records spec events
	_, _ = emitCodeGraph(rt, event.CodeGraphEntityDefinedContent{
		EntityType: "Task", EntityID: agentID,
	})
	_, _ = emitCodeGraph(rt, event.CodeGraphStateTransitionedContent{
		EntityType: "Task", EntityID: agentID, From: "todo", To: "doing",
	})
	_, _ = emitCodeGraph(rt, event.CodeGraphViewRenderedContent{
		ViewName: "TaskBoard", Actor: agentID,
	})

	// 2. Agent evaluates the spec
	_, evaluation, err := rt.Evaluate(ctx, "codegraph_spec",
		"You have a Code Graph spec with: Entity(Task) with states todo→doing→done, and View(TaskBoard).\n"+
			"Evaluate: what components would a React implementation need? List them in 2-3 bullet points.")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(evaluation) == 0 {
		t.Fatal("evaluation should not be empty")
	}
	t.Logf("Evaluation: %s", evaluation)

	// 3. Agent generates code
	code, err := rt.CodeWrite(ctx,
		"Implement a React TaskBoard component. It shows tasks grouped by state (todo, doing, done). "+
			"Each task has a title and can be dragged between columns.",
		"typescript")
	if err != nil {
		t.Fatalf("CodeWrite: %v", err)
	}
	if len(code) == 0 {
		t.Fatal("code should not be empty")
	}
	t.Logf("Generated code (%d chars): %s", len(code), code[:min(200, len(code))])

	// 4. Verify all events are on the graph
	memory, _ := rt.Memory(20)
	if len(memory) < 6 {
		t.Errorf("expected >= 6 events (3 codegraph + evaluate + 2 code_write), got %d", len(memory))
	}

	// Verify codegraph events exist
	entities, _ := rt.EventsByType("codegraph.entity.defined", 10)
	if len(entities) != 1 {
		t.Errorf("entity.defined = %d, want 1", len(entities))
	}
}

// ════════════════════════════════════════════════════════════════════════
// Helper: emit a code graph event via the runtime's store
// ════════════════════════════════════════════════════════════════════════

// emitCodeGraph records a code graph event on the runtime's event graph.
// This uses the runtime's internal emit mechanism via the public API.
func emitCodeGraph(rt *intelligence.AgentRuntime, content event.EventContent) (event.Event, error) {
	return rt.Emit(content)
}
