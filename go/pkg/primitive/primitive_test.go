package primitive_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/primitive"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// testPrimitive is a minimal Primitive for testing.
type testPrimitive struct {
	id            types.PrimitiveID
	layer         types.Layer
	subscriptions []types.SubscriptionPattern
	cadence       types.Cadence
	lifecycle     types.LifecycleState
	processFunc   func(types.Tick, []event.Event, primitive.Snapshot) ([]primitive.Mutation, error)
}

func (p *testPrimitive) ID() types.PrimitiveID                { return p.id }
func (p *testPrimitive) Layer() types.Layer                    { return p.layer }
func (p *testPrimitive) Subscriptions() []types.SubscriptionPattern { return p.subscriptions }
func (p *testPrimitive) Cadence() types.Cadence                { return p.cadence }
func (p *testPrimitive) Lifecycle() types.LifecycleState       { return p.lifecycle }
func (p *testPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	if p.processFunc != nil {
		return p.processFunc(tick, events, snap)
	}
	return nil, nil
}

func newTestPrimitive(name string, layer int) *testPrimitive {
	return &testPrimitive{
		id:        types.MustPrimitiveID(name),
		layer:     types.MustLayer(layer),
		cadence:   types.MustCadence(1),
		lifecycle: types.LifecycleActive,
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
	}
}

// --- Mutation visitor tests ---

func TestMutationVisitorDispatch(t *testing.T) {
	type visited struct{ name string }
	v := &testMutationVisitor{}

	mutations := []primitive.Mutation{
		primitive.AddEvent{Type: event.EventTypeTrustUpdated},
		primitive.AddEdge{EdgeType: event.EdgeTypeTrust},
		primitive.UpdateState{Key: "k"},
		primitive.UpdateActivation{Level: types.MustActivation(0.5)},
		primitive.UpdateLifecycle{State: types.LifecycleActive},
	}

	expected := []string{"AddEvent", "AddEdge", "UpdateState", "UpdateActivation", "UpdateLifecycle"}
	for i, m := range mutations {
		v.last = ""
		m.Accept(v)
		if v.last != expected[i] {
			t.Errorf("mutation %d: visited %q, want %q", i, v.last, expected[i])
		}
	}
}

type testMutationVisitor struct{ last string }

func (v *testMutationVisitor) VisitAddEvent(primitive.AddEvent)             { v.last = "AddEvent" }
func (v *testMutationVisitor) VisitAddEdge(primitive.AddEdge)               { v.last = "AddEdge" }
func (v *testMutationVisitor) VisitUpdateState(primitive.UpdateState)       { v.last = "UpdateState" }
func (v *testMutationVisitor) VisitUpdateActivation(primitive.UpdateActivation) { v.last = "UpdateActivation" }
func (v *testMutationVisitor) VisitUpdateLifecycle(primitive.UpdateLifecycle)   { v.last = "UpdateLifecycle" }

// --- Registry tests ---

func TestRegistryRegisterAndGet(t *testing.T) {
	r := primitive.NewRegistry()
	p := newTestPrimitive("test_prim", 0)

	if err := r.Register(p); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if r.Count() != 1 {
		t.Errorf("Count = %d, want 1", r.Count())
	}

	got, ok := r.Get(p.ID())
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.ID() != p.ID() {
		t.Errorf("ID = %v, want %v", got.ID(), p.ID())
	}
}

func TestRegistryDuplicateRegister(t *testing.T) {
	r := primitive.NewRegistry()
	p := newTestPrimitive("test_prim", 0)

	r.Register(p)
	if err := r.Register(p); err == nil {
		t.Fatal("expected error for duplicate register")
	}
}

func TestRegistryAllStates(t *testing.T) {
	r := primitive.NewRegistry()
	p := newTestPrimitive("test_prim", 0)
	r.Register(p)

	states := r.AllStates()
	if len(states) != 1 {
		t.Fatalf("AllStates len = %d, want 1", len(states))
	}

	ps := states[p.ID()]
	if ps.Lifecycle != types.LifecycleDormant {
		t.Errorf("Lifecycle = %v, want Dormant", ps.Lifecycle)
	}
	if ps.Activation.Value() != 0.0 {
		t.Errorf("Activation = %v, want 0.0", ps.Activation.Value())
	}
}

func TestRegistryActivate(t *testing.T) {
	r := primitive.NewRegistry()
	p := newTestPrimitive("test_prim", 0)
	r.Register(p)

	if err := r.Activate(p.ID()); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if r.Lifecycle(p.ID()) != types.LifecycleActive {
		t.Errorf("Lifecycle = %v, want Active", r.Lifecycle(p.ID()))
	}
}

func TestRegistrySetActivation(t *testing.T) {
	r := primitive.NewRegistry()
	p := newTestPrimitive("test_prim", 0)
	r.Register(p)

	if err := r.SetActivation(p.ID(), types.MustActivation(0.75)); err != nil {
		t.Fatalf("SetActivation: %v", err)
	}
	states := r.AllStates()
	if states[p.ID()].Activation.Value() != 0.75 {
		t.Errorf("Activation = %v, want 0.75", states[p.ID()].Activation.Value())
	}
}

func TestRegistryUpdateState(t *testing.T) {
	r := primitive.NewRegistry()
	p := newTestPrimitive("test_prim", 0)
	r.Register(p)

	if err := r.UpdateState(p.ID(), "counter", 42); err != nil {
		t.Fatalf("UpdateState: %v", err)
	}
	states := r.AllStates()
	if states[p.ID()].State()["counter"] != 42 {
		t.Errorf("State[counter] = %v, want 42", states[p.ID()].State()["counter"])
	}
}

func TestRegistrySetLastTick(t *testing.T) {
	r := primitive.NewRegistry()
	p := newTestPrimitive("test_prim", 0)
	r.Register(p)

	r.SetLastTick(p.ID(), types.MustTick(5))
	if r.LastTick(p.ID()).Value() != 5 {
		t.Errorf("LastTick = %d, want 5", r.LastTick(p.ID()).Value())
	}
}

func TestRegistryInvalidLifecycleTransition(t *testing.T) {
	r := primitive.NewRegistry()
	p := newTestPrimitive("test_prim", 0)
	r.Register(p)

	// Dormant → Active is invalid (must go through Activating)
	if err := r.SetLifecycle(p.ID(), types.LifecycleActive); err == nil {
		t.Fatal("expected error for invalid transition Dormant → Active")
	}
}

func TestRegistryAllOrderedByLayer(t *testing.T) {
	r := primitive.NewRegistry()
	p2 := newTestPrimitive("prim_b", 2)
	p0 := newTestPrimitive("prim_a", 0)
	p1 := newTestPrimitive("prim_c", 1)

	r.Register(p2)
	r.Register(p0)
	r.Register(p1)

	all := r.All()
	if len(all) != 3 {
		t.Fatalf("All len = %d, want 3", len(all))
	}
	if all[0].Layer().Value() != 0 {
		t.Errorf("first primitive layer = %d, want 0", all[0].Layer().Value())
	}
	if all[1].Layer().Value() != 1 {
		t.Errorf("second primitive layer = %d, want 1", all[1].Layer().Value())
	}
	if all[2].Layer().Value() != 2 {
		t.Errorf("third primitive layer = %d, want 2", all[2].Layer().Value())
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	r := primitive.NewRegistry()
	_, ok := r.Get(types.MustPrimitiveID("nonexistent"))
	if ok {
		t.Error("expected Get to return false for nonexistent primitive")
	}
}

func TestRegistrySetActivationNotFound(t *testing.T) {
	r := primitive.NewRegistry()
	err := r.SetActivation(types.MustPrimitiveID("nonexistent"), types.MustActivation(0.5))
	if err == nil {
		t.Error("expected error for nonexistent primitive")
	}
}

func TestRegistryUpdateStateNotFound(t *testing.T) {
	r := primitive.NewRegistry()
	err := r.UpdateState(types.MustPrimitiveID("nonexistent"), "k", "v")
	if err == nil {
		t.Error("expected error for nonexistent primitive")
	}
}

func TestRegistrySetLifecycleNotFound(t *testing.T) {
	r := primitive.NewRegistry()
	err := r.SetLifecycle(types.MustPrimitiveID("nonexistent"), types.LifecycleActive)
	if err == nil {
		t.Error("expected error for nonexistent primitive")
	}
}

// --- Snapshot tests ---

func TestSnapshotIsValueType(t *testing.T) {
	snap := primitive.Snapshot{
		Tick:       types.MustTick(1),
		Primitives: map[types.PrimitiveID]primitive.PrimitiveState{},
	}
	if snap.Tick.Value() != 1 {
		t.Errorf("Tick = %d, want 1", snap.Tick.Value())
	}
}
