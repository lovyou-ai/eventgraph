// Package primitive defines the Primitive interface, Mutation types,
// Snapshot, PrimitiveState, and PrimitiveRegistry. These are the building
// blocks that the tick engine processes.
package primitive

import (
	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Primitive is a software agent that embodies a specific domain of intelligence.
// Each primitive has identity, state, lifecycle, subscriptions, and cadence.
// The tick engine invokes Process() when matching events exist and cadence allows.
type Primitive interface {
	ID() types.PrimitiveID
	Layer() types.Layer
	Process(tick types.Tick, events []event.Event, snapshot Snapshot) ([]Mutation, error)
	Subscriptions() []types.SubscriptionPattern
	Cadence() types.Cadence
	Lifecycle() types.LifecycleState
}

// PrimitiveState is the read-only view of a primitive's state used in snapshots.
// The state map is unexported to enforce the Frozen<Snapshot> invariant —
// callers receive a defensive copy via the State() method.
type PrimitiveState struct {
	ID         types.PrimitiveID
	Layer      types.Layer
	Lifecycle  types.LifecycleState
	Activation types.Activation
	Cadence    types.Cadence
	state      map[string]any
	LastTick   types.Tick
}

// State returns a deep defensive copy of the primitive's key-value state.
// Returns nil if the state is nil, preserving nil/empty distinction.
func (ps PrimitiveState) State() map[string]any {
	return deepCopyState(ps.state)
}

// Snapshot is the frozen, read-only view passed to primitives during processing.
// No primitive can mutate another's state through the snapshot.
type Snapshot struct {
	Tick          types.Tick
	Primitives    map[types.PrimitiveID]PrimitiveState
	PendingEvents []event.Event
	RecentEvents  []event.Event
	ActiveActors  []actor.IActor
}

// --- Mutation types ---

// Mutation is a declarative change produced by a primitive's Process() call.
// The tick engine collects and applies mutations atomically after each wave.
type Mutation interface {
	Accept(MutationVisitor)
	isMutation()
}

// MutationVisitor provides exhaustive dispatch over mutation types.
type MutationVisitor interface {
	VisitAddEvent(AddEvent)
	VisitAddEdge(AddEdge)
	VisitUpdateState(UpdateState)
	VisitUpdateActivation(UpdateActivation)
	VisitUpdateLifecycle(UpdateLifecycle)
}

// AddEvent requests the tick engine to create and append a new event.
type AddEvent struct {
	Type           types.EventType
	Source         types.ActorID
	Content        event.EventContent
	Causes         []types.EventID
	ConversationID types.ConversationID // zero value uses engine default
}

func (m AddEvent) Accept(v MutationVisitor) { v.VisitAddEvent(m) }
func (m AddEvent) isMutation()              {}

// AddEdge requests the tick engine to create a new edge (via an edge.created event).
type AddEdge struct {
	From     types.ActorID
	To       types.ActorID
	EdgeType event.EdgeType
	Weight   types.Weight
	Scope    types.Option[types.DomainScope]
	Causes   []types.EventID // optional — if empty, engine uses chain head
}

func (m AddEdge) Accept(v MutationVisitor) { v.VisitAddEdge(m) }
func (m AddEdge) isMutation()              {}

// UpdateState updates a key-value pair in the primitive's internal state.
type UpdateState struct {
	PrimitiveID types.PrimitiveID
	Key         string
	Value       any
}

func (m UpdateState) Accept(v MutationVisitor) { v.VisitUpdateState(m) }
func (m UpdateState) isMutation()              {}

// UpdateActivation changes the primitive's activation level.
type UpdateActivation struct {
	PrimitiveID types.PrimitiveID
	Level       types.Activation
}

func (m UpdateActivation) Accept(v MutationVisitor) { v.VisitUpdateActivation(m) }
func (m UpdateActivation) isMutation()              {}

// UpdateLifecycle requests a lifecycle state transition. Must be a valid transition.
type UpdateLifecycle struct {
	PrimitiveID types.PrimitiveID
	State       types.LifecycleState
}

func (m UpdateLifecycle) Accept(v MutationVisitor) { v.VisitUpdateLifecycle(m) }
func (m UpdateLifecycle) isMutation()              {}
