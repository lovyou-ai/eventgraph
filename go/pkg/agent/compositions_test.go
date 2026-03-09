package agent_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/agent"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

func TestAllCompositionsCount(t *testing.T) {
	comps := agent.AllCompositions()
	if len(comps) != 8 {
		t.Errorf("AllCompositions() returned %d, want 8", len(comps))
	}
}

func TestCompositionNames(t *testing.T) {
	expected := []string{"Boot", "Imprint", "Task", "Supervise", "Collaborate", "Crisis", "Retire", "Whistleblow"}
	comps := agent.AllCompositions()
	for i, comp := range comps {
		if comp.Name != expected[i] {
			t.Errorf("composition[%d].Name = %q, want %q", i, comp.Name, expected[i])
		}
	}
}

func TestBootComposition(t *testing.T) {
	comp := agent.Boot()
	if comp.Name != "Boot" {
		t.Errorf("Name = %q, want Boot", comp.Name)
	}
	if len(comp.Primitives) != 5 {
		t.Errorf("Boot uses %d primitives, want 5", len(comp.Primitives))
	}
	if len(comp.Events) != 5 {
		t.Errorf("Boot emits %d event types, want 5", len(comp.Events))
	}
	// Verify expected primitives
	expectedPrims := []string{"agent.Identity", "agent.Soul", "agent.Model", "agent.Authority", "agent.State"}
	for i, p := range comp.Primitives {
		if p != expectedPrims[i] {
			t.Errorf("Boot.Primitives[%d] = %q, want %q", i, p, expectedPrims[i])
		}
	}
}

func TestImprintComposition(t *testing.T) {
	comp := agent.Imprint()
	if comp.Name != "Imprint" {
		t.Errorf("Name = %q, want Imprint", comp.Name)
	}
	// Imprint = Boot + Observe + Learn + Goal = 5 + 3 = 8 primitives
	if len(comp.Primitives) != 8 {
		t.Errorf("Imprint uses %d primitives, want 8", len(comp.Primitives))
	}
	// Boot events + 3 more
	if len(comp.Events) != 8 {
		t.Errorf("Imprint emits %d event types, want 8", len(comp.Events))
	}
}

func TestTaskComposition(t *testing.T) {
	comp := agent.Task()
	if comp.Name != "Task" {
		t.Errorf("Name = %q, want Task", comp.Name)
	}
	expectedPrims := []string{"agent.Observe", "agent.Evaluate", "agent.Decide", "agent.Act", "agent.Learn"}
	if len(comp.Primitives) != len(expectedPrims) {
		t.Fatalf("Task uses %d primitives, want %d", len(comp.Primitives), len(expectedPrims))
	}
	for i, p := range comp.Primitives {
		if p != expectedPrims[i] {
			t.Errorf("Task.Primitives[%d] = %q, want %q", i, p, expectedPrims[i])
		}
	}
}

func TestSuperviseComposition(t *testing.T) {
	comp := agent.Supervise()
	if comp.Name != "Supervise" {
		t.Errorf("Name = %q, want Supervise", comp.Name)
	}
	if len(comp.Primitives) != 5 {
		t.Errorf("Supervise uses %d primitives, want 5", len(comp.Primitives))
	}
}

func TestCollaborateComposition(t *testing.T) {
	comp := agent.Collaborate()
	if comp.Name != "Collaborate" {
		t.Errorf("Name = %q, want Collaborate", comp.Name)
	}
	if len(comp.Primitives) != 5 {
		t.Errorf("Collaborate uses %d primitives, want 5", len(comp.Primitives))
	}
	// Collaborate has 6 events (consent has requested + granted)
	if len(comp.Events) != 6 {
		t.Errorf("Collaborate emits %d event types, want 6", len(comp.Events))
	}
}

func TestCrisisComposition(t *testing.T) {
	comp := agent.Crisis()
	if comp.Name != "Crisis" {
		t.Errorf("Name = %q, want Crisis", comp.Name)
	}
	if len(comp.Primitives) != 5 {
		t.Errorf("Crisis uses %d primitives, want 5", len(comp.Primitives))
	}
}

func TestRetireComposition(t *testing.T) {
	comp := agent.Retire()
	if comp.Name != "Retire" {
		t.Errorf("Name = %q, want Retire", comp.Name)
	}
	if len(comp.Primitives) != 4 {
		t.Errorf("Retire uses %d primitives, want 4", len(comp.Primitives))
	}
}

func TestWhistleblowComposition(t *testing.T) {
	comp := agent.Whistleblow()
	if comp.Name != "Whistleblow" {
		t.Errorf("Name = %q, want Whistleblow", comp.Name)
	}
	if len(comp.Primitives) != 5 {
		t.Errorf("Whistleblow uses %d primitives, want 5", len(comp.Primitives))
	}
	// Verify it includes Refuse (the dignity invariant)
	hasRefuse := false
	for _, p := range comp.Primitives {
		if p == "agent.Refuse" {
			hasRefuse = true
		}
	}
	if !hasRefuse {
		t.Error("Whistleblow must include agent.Refuse (dignity invariant)")
	}
}

func TestBootEvents(t *testing.T) {
	agentID := types.MustActorID("actor_00000000000000000000000000000099")
	grantor := types.MustActorID("actor_00000000000000000000000000000001")
	scope := types.MustDomainScope("test")
	values := []string{"Take care of your human"}

	contents := agent.BootEvents(agentID, "ai", "claude-opus-4", "premium", values, scope, grantor)
	if len(contents) != 5 {
		t.Fatalf("BootEvents returned %d contents, want 5", len(contents))
	}

	// Verify event type names match Boot composition
	expectedTypes := []string{
		"agent.identity.created",
		"agent.soul.imprinted",
		"agent.model.bound",
		"agent.authority.granted",
		"agent.state.changed",
	}
	for i, c := range contents {
		if c.EventTypeName() != expectedTypes[i] {
			t.Errorf("content[%d].EventTypeName() = %q, want %q", i, c.EventTypeName(), expectedTypes[i])
		}
	}
}

func TestAllCompositionsHaveEvents(t *testing.T) {
	for _, comp := range agent.AllCompositions() {
		if len(comp.Events) == 0 {
			t.Errorf("%q: no events", comp.Name)
		}
	}
}

func TestAllCompositionsHavePrimitives(t *testing.T) {
	for _, comp := range agent.AllCompositions() {
		if len(comp.Primitives) == 0 {
			t.Errorf("%q: no primitives", comp.Name)
		}
	}
}

func TestAllCompositionEventsRegistered(t *testing.T) {
	registry := event.DefaultRegistry()
	for _, comp := range agent.AllCompositions() {
		for _, et := range comp.Events {
			if !registry.IsRegistered(et) {
				t.Errorf("%q: event type %q not registered", comp.Name, et.Value())
			}
		}
	}
}

func TestAllAgentEventTypesRegistered(t *testing.T) {
	registry := event.DefaultRegistry()
	for _, et := range event.AllAgentEventTypes() {
		if !registry.IsRegistered(et) {
			t.Errorf("agent event type %q not registered in DefaultRegistry", et.Value())
		}
	}
}

func TestAgentEventTypeCount(t *testing.T) {
	types := event.AllAgentEventTypes()
	// 19 structural + 15 operational + 9 relational + 2 modal = 45
	if len(types) != 45 {
		t.Errorf("AllAgentEventTypes() returned %d, want 45", len(types))
	}
}
