package agent

import (
	"fmt"
	"strings"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/primitive"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

var agentLayer = types.MustLayer(1)
var cadence1 = types.MustCadence(1)

// ════════════════════════════════════════════════════════════════════════
// STRUCTURAL PRIMITIVES (11) — Define what an agent IS
// ════════════════════════════════════════════════════════════════════════

// IdentityPrimitive — ActorID + keys + type + chain of custody.
// The unforgeable "who."
type IdentityPrimitive struct{}

func NewIdentityPrimitive() *IdentityPrimitive { return &IdentityPrimitive{} }

func (p *IdentityPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Identity") }
func (p *IdentityPrimitive) Layer() types.Layer              { return agentLayer }
func (p *IdentityPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *IdentityPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *IdentityPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.identity.*"),
		types.MustSubscriptionPattern("actor.registered"),
	}
}

func (p *IdentityPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	created := 0
	rotated := 0
	for _, ev := range events {
		t := ev.Type().Value()
		switch {
		case t == "agent.identity.created" || t == "actor.registered":
			created++
		case t == "agent.identity.rotated":
			rotated++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "identitiesCreated", Value: created},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "keysRotated", Value: rotated},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// SoulPrimitive — The agent's values and ethical constraints.
// Immutable after imprint.
type SoulPrimitive struct{}

func NewSoulPrimitive() *SoulPrimitive { return &SoulPrimitive{} }

func (p *SoulPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Soul") }
func (p *SoulPrimitive) Layer() types.Layer              { return agentLayer }
func (p *SoulPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *SoulPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *SoulPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.soul.*"),
		types.MustSubscriptionPattern("agent.refused"),
	}
}

func (p *SoulPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	imprints := 0
	refusals := 0
	for _, ev := range events {
		t := ev.Type().Value()
		switch {
		case t == "agent.soul.imprinted":
			imprints++
		case t == "agent.refused":
			refusals++
		}
	}
	// Soul is set once — track whether imprinted
	mutations := []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}
	if imprints > 0 {
		mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "imprinted", Value: true})
	}
	if refusals > 0 {
		mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "soulRefusals", Value: refusals})
	}
	return mutations, nil
}

// ModelPrimitive — The IIntelligence binding.
// Which reasoning engine, what capabilities, what cost tier.
type ModelPrimitive struct{}

func NewModelPrimitive() *ModelPrimitive { return &ModelPrimitive{} }

func (p *ModelPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Model") }
func (p *ModelPrimitive) Layer() types.Layer              { return agentLayer }
func (p *ModelPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *ModelPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *ModelPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.model.*"),
	}
}

func (p *ModelPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	bindings := 0
	changes := 0
	for _, ev := range events {
		switch ev.Type().Value() {
		case "agent.model.bound":
			bindings++
		case "agent.model.changed":
			changes++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "bindings", Value: bindings},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "modelChanges", Value: changes},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// MemoryPrimitive — Persistent state across ticks.
// What the agent has learned and remembers.
type MemoryPrimitive struct{}

func NewMemoryPrimitive() *MemoryPrimitive { return &MemoryPrimitive{} }

func (p *MemoryPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Memory") }
func (p *MemoryPrimitive) Layer() types.Layer              { return agentLayer }
func (p *MemoryPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *MemoryPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *MemoryPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.memory.*"),
		types.MustSubscriptionPattern("agent.learned"),
	}
}

func (p *MemoryPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	updates := 0
	for _, ev := range events {
		t := ev.Type().Value()
		if t == "agent.memory.updated" || t == "agent.learned" {
			updates++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "memoryUpdates", Value: updates},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// StatePrimitive — Current operational state: idle, processing, waiting, suspended.
// The finite state machine.
type StatePrimitive struct{}

func NewStatePrimitive() *StatePrimitive { return &StatePrimitive{} }

func (p *StatePrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.State") }
func (p *StatePrimitive) Layer() types.Layer              { return agentLayer }
func (p *StatePrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *StatePrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *StatePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.state.*"),
	}
}

func (p *StatePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	transitions := 0
	var lastState string
	for _, ev := range events {
		if ev.Type().Value() == "agent.state.changed" {
			transitions++
			lastState = "changed"
		}
	}
	mutations := []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "transitions", Value: transitions},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}
	if lastState != "" {
		mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTransition", Value: lastState})
	}
	return mutations, nil
}

// AuthorityPrimitive — What this agent is permitted to do.
// Received from above, scoped, revocable.
type AuthorityPrimitive struct{}

func NewAuthorityPrimitive() *AuthorityPrimitive { return &AuthorityPrimitive{} }

func (p *AuthorityPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Authority") }
func (p *AuthorityPrimitive) Layer() types.Layer              { return agentLayer }
func (p *AuthorityPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *AuthorityPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *AuthorityPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.authority.*"),
		types.MustSubscriptionPattern("authority.*"),
	}
}

func (p *AuthorityPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	granted := 0
	revoked := 0
	for _, ev := range events {
		t := ev.Type().Value()
		switch {
		case t == "agent.authority.granted":
			granted++
		case t == "agent.authority.revoked":
			revoked++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "authorityGrants", Value: granted},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "authorityRevocations", Value: revoked},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// TrustPrimitive — Trust scores this agent holds toward other actors.
// Asymmetric, non-transitive, decaying.
type TrustPrimitive struct{}

func NewTrustPrimitive() *TrustPrimitive { return &TrustPrimitive{} }

func (p *TrustPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Trust") }
func (p *TrustPrimitive) Layer() types.Layer              { return agentLayer }
func (p *TrustPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *TrustPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *TrustPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.trust.*"),
		types.MustSubscriptionPattern("trust.*"),
	}
}

func (p *TrustPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	assessments := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.trust.assessed" {
			assessments++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "trustAssessments", Value: assessments},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// BudgetPrimitive — Resource constraints: token budget, API calls, time limits, cost ceiling.
type BudgetPrimitive struct{}

func NewBudgetPrimitive() *BudgetPrimitive { return &BudgetPrimitive{} }

func (p *BudgetPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Budget") }
func (p *BudgetPrimitive) Layer() types.Layer              { return agentLayer }
func (p *BudgetPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *BudgetPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *BudgetPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.budget.*"),
	}
}

func (p *BudgetPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	allocated := 0
	exhausted := 0
	for _, ev := range events {
		switch ev.Type().Value() {
		case "agent.budget.allocated":
			allocated++
		case "agent.budget.exhausted":
			exhausted++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "allocations", Value: allocated},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "exhaustions", Value: exhausted},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// RolePrimitive — Named function within a team: Builder, Reviewer, Guardian, CTO.
// Determines subscription patterns.
type RolePrimitive struct{}

func NewRolePrimitive() *RolePrimitive { return &RolePrimitive{} }

func (p *RolePrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Role") }
func (p *RolePrimitive) Layer() types.Layer              { return agentLayer }
func (p *RolePrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *RolePrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *RolePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.role.*"),
	}
}

func (p *RolePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	assignments := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.role.assigned" {
			assignments++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "roleAssignments", Value: assignments},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// LifespanPrimitive — Birth, expected duration, graceful shutdown conditions.
// When and how the agent ends.
type LifespanPrimitive struct{}

func NewLifespanPrimitive() *LifespanPrimitive { return &LifespanPrimitive{} }

func (p *LifespanPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Lifespan") }
func (p *LifespanPrimitive) Layer() types.Layer              { return agentLayer }
func (p *LifespanPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *LifespanPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *LifespanPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.lifespan.*"),
	}
}

func (p *LifespanPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	started := 0
	ended := 0
	for _, ev := range events {
		switch ev.Type().Value() {
		case "agent.lifespan.started":
			started++
		case "agent.lifespan.ended":
			ended++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "agentsStarted", Value: started},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "agentsEnded", Value: ended},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// GoalPrimitive — Current objective hierarchy.
// What the agent is trying to accomplish. Mutable as tasks arrive.
type GoalPrimitive struct{}

func NewGoalPrimitive() *GoalPrimitive { return &GoalPrimitive{} }

func (p *GoalPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Goal") }
func (p *GoalPrimitive) Layer() types.Layer              { return agentLayer }
func (p *GoalPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *GoalPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *GoalPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.goal.*"),
	}
}

func (p *GoalPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	set := 0
	completed := 0
	abandoned := 0
	for _, ev := range events {
		switch ev.Type().Value() {
		case "agent.goal.set":
			set++
		case "agent.goal.completed":
			completed++
		case "agent.goal.abandoned":
			abandoned++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "goalsSet", Value: set},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "goalsCompleted", Value: completed},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "goalsAbandoned", Value: abandoned},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// ════════════════════════════════════════════════════════════════════════
// OPERATIONAL PRIMITIVES (13) — Define what an agent DOES
// ════════════════════════════════════════════════════════════════════════

// ObservePrimitive — Passive perception. Events arrive via subscriptions.
type ObservePrimitive struct{}

func NewObservePrimitive() *ObservePrimitive { return &ObservePrimitive{} }

func (p *ObservePrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Observe") }
func (p *ObservePrimitive) Layer() types.Layer              { return agentLayer }
func (p *ObservePrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *ObservePrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *ObservePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.observed"),
		types.MustSubscriptionPattern("agent.*"),
	}
}

func (p *ObservePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	observed := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.observed" {
			observed++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "eventsObserved", Value: observed},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "totalEventsReceived", Value: len(events)},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// ProbePrimitive — Active perception. The agent queries the graph, stores, other agents.
type ProbePrimitive struct{}

func NewProbePrimitive() *ProbePrimitive { return &ProbePrimitive{} }

func (p *ProbePrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Probe") }
func (p *ProbePrimitive) Layer() types.Layer              { return agentLayer }
func (p *ProbePrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *ProbePrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *ProbePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.probed"),
	}
}

func (p *ProbePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	probes := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.probed" {
			probes++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "probesExecuted", Value: probes},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// EvaluatePrimitive — One-shot judgment. Assess a situation, produce a score/classification.
type EvaluatePrimitive struct{}

func NewEvaluatePrimitive() *EvaluatePrimitive { return &EvaluatePrimitive{} }

func (p *EvaluatePrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Evaluate") }
func (p *EvaluatePrimitive) Layer() types.Layer              { return agentLayer }
func (p *EvaluatePrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *EvaluatePrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *EvaluatePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.evaluated"),
	}
}

func (p *EvaluatePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	evaluations := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.evaluated" {
			evaluations++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "evaluations", Value: evaluations},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// DecidePrimitive — Commit to an action. Takes evaluation output, produces a Decision.
type DecidePrimitive struct{}

func NewDecidePrimitive() *DecidePrimitive { return &DecidePrimitive{} }

func (p *DecidePrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Decide") }
func (p *DecidePrimitive) Layer() types.Layer              { return agentLayer }
func (p *DecidePrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *DecidePrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *DecidePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.decided"),
		types.MustSubscriptionPattern("agent.evaluated"),
	}
}

func (p *DecidePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	decisions := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.decided" {
			decisions++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "decisions", Value: decisions},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// ActPrimitive — Execute a decision. Emit events, create edges, modify graph state.
type ActPrimitive struct{}

func NewActPrimitive() *ActPrimitive { return &ActPrimitive{} }

func (p *ActPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Act") }
func (p *ActPrimitive) Layer() types.Layer              { return agentLayer }
func (p *ActPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *ActPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *ActPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.acted"),
		types.MustSubscriptionPattern("agent.decided"),
	}
}

func (p *ActPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	actions := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.acted" {
			actions++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "actionsExecuted", Value: actions},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// DelegatePrimitive — Assign work to another agent. Transfer a goal with authority and constraints.
type DelegatePrimitive struct{}

func NewDelegatePrimitive() *DelegatePrimitive { return &DelegatePrimitive{} }

func (p *DelegatePrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Delegate") }
func (p *DelegatePrimitive) Layer() types.Layer              { return agentLayer }
func (p *DelegatePrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *DelegatePrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *DelegatePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.delegated"),
	}
}

func (p *DelegatePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	delegations := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.delegated" {
			delegations++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "delegations", Value: delegations},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// EscalatePrimitive — Pass upward. "I can't handle this." Capability-limited.
type EscalatePrimitive struct{}

func NewEscalatePrimitive() *EscalatePrimitive { return &EscalatePrimitive{} }

func (p *EscalatePrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Escalate") }
func (p *EscalatePrimitive) Layer() types.Layer              { return agentLayer }
func (p *EscalatePrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *EscalatePrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *EscalatePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.escalated"),
	}
}

func (p *EscalatePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	escalations := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.escalated" {
			escalations++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "escalations", Value: escalations},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// RefusePrimitive — Decline to act. "I won't do this." Values-limited.
// Emits refusal event with reason.
type RefusePrimitive struct{}

func NewRefusePrimitive() *RefusePrimitive { return &RefusePrimitive{} }

func (p *RefusePrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Refuse") }
func (p *RefusePrimitive) Layer() types.Layer              { return agentLayer }
func (p *RefusePrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *RefusePrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *RefusePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.refused"),
	}
}

func (p *RefusePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	refusals := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.refused" {
			refusals++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "refusals", Value: refusals},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// LearnPrimitive — Update Memory based on outcomes. Self-mutating.
type LearnPrimitive struct{}

func NewLearnPrimitive() *LearnPrimitive { return &LearnPrimitive{} }

func (p *LearnPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Learn") }
func (p *LearnPrimitive) Layer() types.Layer              { return agentLayer }
func (p *LearnPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *LearnPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *LearnPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.learned"),
		types.MustSubscriptionPattern("agent.goal.completed"),
		types.MustSubscriptionPattern("agent.goal.abandoned"),
	}
}

func (p *LearnPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	lessons := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.learned" {
			lessons++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lessonsLearned", Value: lessons},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// IntrospectPrimitive — Read own State and Soul. Self-observation without mutation.
type IntrospectPrimitive struct{}

func NewIntrospectPrimitive() *IntrospectPrimitive { return &IntrospectPrimitive{} }

func (p *IntrospectPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Introspect") }
func (p *IntrospectPrimitive) Layer() types.Layer              { return agentLayer }
func (p *IntrospectPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *IntrospectPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *IntrospectPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.introspected"),
	}
}

func (p *IntrospectPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	introspections := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.introspected" {
			introspections++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "introspections", Value: introspections},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// CommunicatePrimitive — Send a message to another agent or channel.
type CommunicatePrimitive struct{}

func NewCommunicatePrimitive() *CommunicatePrimitive { return &CommunicatePrimitive{} }

func (p *CommunicatePrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Communicate") }
func (p *CommunicatePrimitive) Layer() types.Layer              { return agentLayer }
func (p *CommunicatePrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *CommunicatePrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *CommunicatePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.communicated"),
		types.MustSubscriptionPattern("agent.channel.*"),
	}
}

func (p *CommunicatePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	messages := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.communicated" {
			messages++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "messagesSent", Value: messages},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// RepairPrimitive — Fix a prior Act. Changes both graph state AND relationship state.
type RepairPrimitive struct{}

func NewRepairPrimitive() *RepairPrimitive { return &RepairPrimitive{} }

func (p *RepairPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Repair") }
func (p *RepairPrimitive) Layer() types.Layer              { return agentLayer }
func (p *RepairPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *RepairPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *RepairPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.repaired"),
	}
}

func (p *RepairPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	repairs := 0
	for _, ev := range events {
		if ev.Type().Value() == "agent.repaired" {
			repairs++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "repairs", Value: repairs},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// ExpectPrimitive — Create a persistent monitoring condition.
// "Watch for X and alert me." Continuous, unlike one-shot Evaluate.
type ExpectPrimitive struct{}

func NewExpectPrimitive() *ExpectPrimitive { return &ExpectPrimitive{} }

func (p *ExpectPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Expect") }
func (p *ExpectPrimitive) Layer() types.Layer              { return agentLayer }
func (p *ExpectPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *ExpectPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *ExpectPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.expectation.*"),
	}
}

func (p *ExpectPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	set := 0
	met := 0
	expired := 0
	for _, ev := range events {
		switch ev.Type().Value() {
		case "agent.expectation.set":
			set++
		case "agent.expectation.met":
			met++
		case "agent.expectation.expired":
			expired++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "expectationsSet", Value: set},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "expectationsMet", Value: met},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "expectationsExpired", Value: expired},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// ════════════════════════════════════════════════════════════════════════
// RELATIONAL PRIMITIVES (3) — Define how agents relate
// ════════════════════════════════════════════════════════════════════════

// ConsentPrimitive — Bilateral agreement. Both parties must agree.
type ConsentPrimitive struct{}

func NewConsentPrimitive() *ConsentPrimitive { return &ConsentPrimitive{} }

func (p *ConsentPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Consent") }
func (p *ConsentPrimitive) Layer() types.Layer              { return agentLayer }
func (p *ConsentPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *ConsentPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *ConsentPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.consent.*"),
	}
}

func (p *ConsentPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	requested := 0
	granted := 0
	denied := 0
	for _, ev := range events {
		switch ev.Type().Value() {
		case "agent.consent.requested":
			requested++
		case "agent.consent.granted":
			granted++
		case "agent.consent.denied":
			denied++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "consentRequested", Value: requested},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "consentGranted", Value: granted},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "consentDenied", Value: denied},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// ChannelPrimitive — Persistent bidirectional communication link between agents.
type ChannelPrimitive struct{}

func NewChannelPrimitive() *ChannelPrimitive { return &ChannelPrimitive{} }

func (p *ChannelPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Channel") }
func (p *ChannelPrimitive) Layer() types.Layer              { return agentLayer }
func (p *ChannelPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *ChannelPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *ChannelPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.channel.*"),
	}
}

func (p *ChannelPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	opened := 0
	closed := 0
	for _, ev := range events {
		switch ev.Type().Value() {
		case "agent.channel.opened":
			opened++
		case "agent.channel.closed":
			closed++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "channelsOpened", Value: opened},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "channelsClosed", Value: closed},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// CompositionPrimitive — Form a group. Multiple agents become a unit.
type CompositionPrimitive struct{}

func NewCompositionPrimitive() *CompositionPrimitive { return &CompositionPrimitive{} }

func (p *CompositionPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Composition") }
func (p *CompositionPrimitive) Layer() types.Layer              { return agentLayer }
func (p *CompositionPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *CompositionPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *CompositionPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.composition.*"),
	}
}

func (p *CompositionPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	formed := 0
	dissolved := 0
	joined := 0
	left := 0
	for _, ev := range events {
		switch ev.Type().Value() {
		case "agent.composition.formed":
			formed++
		case "agent.composition.dissolved":
			dissolved++
		case "agent.composition.joined":
			joined++
		case "agent.composition.left":
			left++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "groupsFormed", Value: formed},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "groupsDissolved", Value: dissolved},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "membersJoined", Value: joined},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "membersLeft", Value: left},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// ════════════════════════════════════════════════════════════════════════
// MODAL PRIMITIVE (1) — Modifies how other primitives operate
// ════════════════════════════════════════════════════════════════════════

// AttenuationPrimitive — Reduce scope, confidence, or authority.
// "Do less, be more careful." Applied to any operation.
type AttenuationPrimitive struct{}

func NewAttenuationPrimitive() *AttenuationPrimitive { return &AttenuationPrimitive{} }

func (p *AttenuationPrimitive) ID() types.PrimitiveID          { return types.MustPrimitiveID("agent.Attenuation") }
func (p *AttenuationPrimitive) Layer() types.Layer              { return agentLayer }
func (p *AttenuationPrimitive) Lifecycle() types.LifecycleState { return types.LifecycleActive }
func (p *AttenuationPrimitive) Cadence() types.Cadence          { return cadence1 }
func (p *AttenuationPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("agent.attenuated"),
		types.MustSubscriptionPattern("agent.attenuation.*"),
		types.MustSubscriptionPattern("agent.budget.exhausted"),
	}
}

func (p *AttenuationPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	attenuated := 0
	lifted := 0
	budgetTriggered := 0
	for _, ev := range events {
		t := ev.Type().Value()
		switch {
		case t == "agent.attenuated":
			attenuated++
		case t == "agent.attenuation.lifted":
			lifted++
		case t == "agent.budget.exhausted":
			budgetTriggered++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "attenuations", Value: attenuated},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lifts", Value: lifted},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "budgetTriggered", Value: budgetTriggered},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	}, nil
}

// ════════════════════════════════════════════════════════════════════════
// REGISTRATION
// ════════════════════════════════════════════════════════════════════════

// AllPrimitives returns all 28 agent primitives.
func AllPrimitives() []primitive.Primitive {
	return []primitive.Primitive{
		// Structural (11)
		NewIdentityPrimitive(),
		NewSoulPrimitive(),
		NewModelPrimitive(),
		NewMemoryPrimitive(),
		NewStatePrimitive(),
		NewAuthorityPrimitive(),
		NewTrustPrimitive(),
		NewBudgetPrimitive(),
		NewRolePrimitive(),
		NewLifespanPrimitive(),
		NewGoalPrimitive(),
		// Operational (13)
		NewObservePrimitive(),
		NewProbePrimitive(),
		NewEvaluatePrimitive(),
		NewDecidePrimitive(),
		NewActPrimitive(),
		NewDelegatePrimitive(),
		NewEscalatePrimitive(),
		NewRefusePrimitive(),
		NewLearnPrimitive(),
		NewIntrospectPrimitive(),
		NewCommunicatePrimitive(),
		NewRepairPrimitive(),
		NewExpectPrimitive(),
		// Relational (3)
		NewConsentPrimitive(),
		NewChannelPrimitive(),
		NewCompositionPrimitive(),
		// Modal (1)
		NewAttenuationPrimitive(),
	}
}

// RegisterAll registers all 28 agent primitives with the given registry.
func RegisterAll(reg *primitive.Registry) error {
	for _, p := range AllPrimitives() {
		if err := reg.Register(p); err != nil {
			return fmt.Errorf("register agent primitive %q: %w", p.ID().Value(), err)
		}
		if err := reg.Activate(p.ID()); err != nil {
			return fmt.Errorf("activate agent primitive %q: %w", p.ID().Value(), err)
		}
	}
	return nil
}

// IsAgentPrimitive returns true if the primitive ID belongs to the agent layer.
func IsAgentPrimitive(id types.PrimitiveID) bool {
	return strings.HasPrefix(id.Value(), "agent.")
}
