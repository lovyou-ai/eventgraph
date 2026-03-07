package primitive

import (
	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Harness is a test utility for invoking primitives in isolation.
// Mutations are captured but not applied, enabling assertion without side effects.
type Harness struct {
	registry      *Registry
	pendingEvents []event.Event
	recentEvents  []event.Event
	activeActors  []actor.IActor
	tick          types.Tick
	mutations     []Mutation
}

// NewHarness creates a new PrimitiveTestHarness with sensible defaults.
func NewHarness() *Harness {
	return &Harness{
		registry: NewRegistry(),
		tick:     types.MustTick(1),
	}
}

// WithEvents sets the pending events for the snapshot.
func (h *Harness) WithEvents(events []event.Event) *Harness {
	h.pendingEvents = events
	return h
}

// WithRecentEvents sets the recent events for snapshot context.
func (h *Harness) WithRecentEvents(events []event.Event) *Harness {
	h.recentEvents = events
	return h
}

// WithActors sets the active actors for the snapshot.
func (h *Harness) WithActors(actors []actor.IActor) *Harness {
	h.activeActors = actors
	return h
}

// WithTick sets the tick counter for processing.
func (h *Harness) WithTick(tick types.Tick) *Harness {
	h.tick = tick
	return h
}

// Process invokes a primitive's Process method and captures mutations.
// The primitive is auto-registered and activated if not already in the registry.
// Mutations are captured, not applied — enabling assertion without side effects.
func (h *Harness) Process(p Primitive, events []event.Event) ([]Mutation, error) {
	if _, ok := h.registry.Get(p.ID()); !ok {
		if err := h.registry.Register(p); err != nil {
			return nil, err
		}
		if err := h.registry.Activate(p.ID()); err != nil {
			return nil, err
		}
	}

	snapshot := Snapshot{
		Tick:          h.tick,
		Primitives:    h.registry.AllStates(),
		PendingEvents: h.pendingEvents,
		RecentEvents:  h.recentEvents,
		ActiveActors:  h.activeActors,
	}

	mutations, err := p.Process(h.tick, events, snapshot)
	if err != nil {
		return nil, err
	}

	h.mutations = append(h.mutations, mutations...)
	return mutations, nil
}

// Reset clears accumulated mutations, allowing per-invocation assertions.
// Call between Process invocations to isolate each call's output.
func (h *Harness) Reset() {
	h.mutations = nil
}

// Mutations returns all accumulated mutations from Process calls.
func (h *Harness) Mutations() []Mutation {
	return h.mutations
}

// EmittedEvents returns AddEvent mutations from all Process calls.
func (h *Harness) EmittedEvents() []AddEvent {
	var result []AddEvent
	for _, m := range h.mutations {
		if ae, ok := m.(AddEvent); ok {
			result = append(result, ae)
		}
	}
	return result
}

// StateChanges returns UpdateState mutations as a key-value map.
func (h *Harness) StateChanges() map[string]any {
	result := make(map[string]any)
	for _, m := range h.mutations {
		if us, ok := m.(UpdateState); ok {
			result[us.Key] = us.Value
		}
	}
	return result
}

// ActivationChanges returns UpdateActivation mutations from all Process calls.
func (h *Harness) ActivationChanges() []UpdateActivation {
	var result []UpdateActivation
	for _, m := range h.mutations {
		if ua, ok := m.(UpdateActivation); ok {
			result = append(result, ua)
		}
	}
	return result
}

// EdgeMutations returns AddEdge mutations from all Process calls.
func (h *Harness) EdgeMutations() []AddEdge {
	var result []AddEdge
	for _, m := range h.mutations {
		if ae, ok := m.(AddEdge); ok {
			result = append(result, ae)
		}
	}
	return result
}

// LifecycleChanges returns UpdateLifecycle mutations from all Process calls.
func (h *Harness) LifecycleChanges() []UpdateLifecycle {
	var result []UpdateLifecycle
	for _, m := range h.mutations {
		if ul, ok := m.(UpdateLifecycle); ok {
			result = append(result, ul)
		}
	}
	return result
}
