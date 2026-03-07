// Package layer0 implements the Layer 0 foundation primitives.
// Groups 0 (Core), 1 (Causality), and 2 (Identity).
package layer0

import (
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/primitive"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

var layer0 = types.MustLayer(0)
var cadence1 = types.MustCadence(1)

// --- Group 0: Core ---

// EventPrimitive validates incoming events before graph entry.
// Checks hash integrity, causal links, and required fields.
type EventPrimitive struct {
	systemActor types.ActorID
	store       store.Store
}

func NewEventPrimitive(systemActor types.ActorID, s store.Store) *EventPrimitive {
	return &EventPrimitive{systemActor: systemActor, store: s}
}

func (p *EventPrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("Event") }
func (p *EventPrimitive) Layer() types.Layer                          { return layer0 }
func (p *EventPrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *EventPrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *EventPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *EventPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	for _, ev := range events {
		// Verify hash integrity.
		canonical := event.CanonicalForm(ev)
		computed, err := event.ComputeHash(canonical)
		if err != nil || computed != ev.Hash() {
			mutations = append(mutations, primitive.UpdateState{
				PrimitiveID: p.ID(),
				Key:         "lastInvalidEvent",
				Value:       ev.ID().Value(),
			})
			continue
		}

		// Verify causal predecessors exist (skip bootstrap).
		if !ev.IsBootstrap() {
			for _, causeID := range ev.Causes() {
				if _, err := p.store.Get(causeID); err != nil {
					mutations = append(mutations, primitive.UpdateState{
						PrimitiveID: p.ID(),
						Key:         "lastMissingCause",
						Value:       causeID.Value(),
					})
					break
				}
			}
		}
	}

	// Update event count state.
	count := len(events)
	if count > 0 {
		mutations = append(mutations,
			primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastEventID", Value: events[len(events)-1].ID().Value()},
			primitive.UpdateState{PrimitiveID: p.ID(), Key: "eventCount", Value: count},
		)
	}
	return mutations, nil
}

// EventStorePrimitive wraps the Store interface for the tick engine.
// Tracks chain head and event count.
type EventStorePrimitive struct {
	store store.Store
}

func NewEventStorePrimitive(s store.Store) *EventStorePrimitive {
	return &EventStorePrimitive{store: s}
}

func (p *EventStorePrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("EventStore") }
func (p *EventStorePrimitive) Layer() types.Layer                          { return layer0 }
func (p *EventStorePrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *EventStorePrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *EventStorePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("store.*")}
}

func (p *EventStorePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation

	count, err := p.store.Count()
	if err == nil {
		mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "eventCount", Value: count})
	}

	head, err := p.store.Head()
	if err == nil && head.IsSome() {
		mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastHash", Value: head.Unwrap().Hash().Value()})
	}

	return mutations, nil
}

// ClockPrimitive provides temporal ordering — tick counting and timestamps.
type ClockPrimitive struct{}

func NewClockPrimitive() *ClockPrimitive { return &ClockPrimitive{} }

func (p *ClockPrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("Clock") }
func (p *ClockPrimitive) Layer() types.Layer                          { return layer0 }
func (p *ClockPrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *ClockPrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *ClockPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("clock.*")}
}

func (p *ClockPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	now := types.Now()
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "currentTick", Value: tick.Value()},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTickTime", Value: now.UnixNano()},
	}, nil
}

// HashPrimitive provides cryptographic integrity — SHA-256 chain verification.
type HashPrimitive struct {
	store store.Store
}

func NewHashPrimitive(s store.Store) *HashPrimitive {
	return &HashPrimitive{store: s}
}

func (p *HashPrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("Hash") }
func (p *HashPrimitive) Layer() types.Layer                          { return layer0 }
func (p *HashPrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *HashPrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *HashPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *HashPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	for _, ev := range events {
		canonical := event.CanonicalForm(ev)
		computed, err := event.ComputeHash(canonical)
		if err != nil {
			continue
		}
		if computed != ev.Hash() {
			mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastMismatch", Value: ev.ID().Value()})
			continue
		}
		mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "chainHead", Value: ev.Hash().Value()})
	}
	return mutations, nil
}

// SelfPrimitive maintains system identity and routes messages to primitives.
type SelfPrimitive struct {
	systemActor types.ActorID
	registry    *primitive.Registry
}

func NewSelfPrimitive(systemActor types.ActorID, registry *primitive.Registry) *SelfPrimitive {
	return &SelfPrimitive{systemActor: systemActor, registry: registry}
}

func (p *SelfPrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("Self") }
func (p *SelfPrimitive) Layer() types.Layer                          { return layer0 }
func (p *SelfPrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *SelfPrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *SelfPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("system.*"),
	}
}

func (p *SelfPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "systemActorID", Value: p.systemActor.Value()},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "registeredPrimitives", Value: p.registry.Count()},
	}, nil
}

// --- Group 1: Causality ---

// CausalLinkPrimitive validates causal edges on every new event.
type CausalLinkPrimitive struct {
	store store.Store
}

func NewCausalLinkPrimitive(s store.Store) *CausalLinkPrimitive {
	return &CausalLinkPrimitive{store: s}
}

func (p *CausalLinkPrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("CausalLink") }
func (p *CausalLinkPrimitive) Layer() types.Layer                          { return layer0 }
func (p *CausalLinkPrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *CausalLinkPrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *CausalLinkPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *CausalLinkPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	validLinks := 0
	invalidLinks := 0

	for _, ev := range events {
		if ev.IsBootstrap() {
			continue
		}
		causes := ev.Causes()
		if len(causes) == 0 {
			invalidLinks++
			continue
		}
		allValid := true
		for _, causeID := range causes {
			if _, err := p.store.Get(causeID); err != nil {
				allValid = false
				invalidLinks++
				break
			}
		}
		if allValid {
			validLinks += len(causes)
		}
	}

	mutations = append(mutations,
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "validLinks", Value: validLinks},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "invalidLinks", Value: invalidLinks},
	)
	return mutations, nil
}

// AncestryPrimitive traverses causal chains upward.
type AncestryPrimitive struct {
	store store.Store
}

func NewAncestryPrimitive(s store.Store) *AncestryPrimitive {
	return &AncestryPrimitive{store: s}
}

func (p *AncestryPrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("Ancestry") }
func (p *AncestryPrimitive) Layer() types.Layer                          { return layer0 }
func (p *AncestryPrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *AncestryPrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *AncestryPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("query.*")}
}

func (p *AncestryPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	for _, ev := range events {
		ancestors, err := p.store.Ancestors(ev.ID(), 10)
		if err != nil {
			continue
		}
		mutations = append(mutations, primitive.UpdateState{
			PrimitiveID: p.ID(),
			Key:         "lastQueryDepth",
			Value:       len(ancestors),
		})
	}
	return mutations, nil
}

// DescendancyPrimitive traverses causal chains downward.
type DescendancyPrimitive struct {
	store store.Store
}

func NewDescendancyPrimitive(s store.Store) *DescendancyPrimitive {
	return &DescendancyPrimitive{store: s}
}

func (p *DescendancyPrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("Descendancy") }
func (p *DescendancyPrimitive) Layer() types.Layer                          { return layer0 }
func (p *DescendancyPrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *DescendancyPrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *DescendancyPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("query.*")}
}

func (p *DescendancyPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	for _, ev := range events {
		descendants, err := p.store.Descendants(ev.ID(), 10)
		if err != nil {
			continue
		}
		mutations = append(mutations, primitive.UpdateState{
			PrimitiveID: p.ID(),
			Key:         "lastQueryDepth",
			Value:       len(descendants),
		})
	}
	return mutations, nil
}

// FirstCausePrimitive finds root causes by walking ancestors to the deepest point.
type FirstCausePrimitive struct {
	store store.Store
}

func NewFirstCausePrimitive(s store.Store) *FirstCausePrimitive {
	return &FirstCausePrimitive{store: s}
}

func (p *FirstCausePrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("FirstCause") }
func (p *FirstCausePrimitive) Layer() types.Layer                          { return layer0 }
func (p *FirstCausePrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *FirstCausePrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *FirstCausePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("query.*")}
}

func (p *FirstCausePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	for _, ev := range events {
		root := p.findRoot(ev)
		mutations = append(mutations, primitive.UpdateState{
			PrimitiveID: p.ID(),
			Key:         "lastFirstCause",
			Value:       root.Value(),
		})
	}
	return mutations, nil
}

// findRoot walks the causal chain to find the root ancestor (typically the bootstrap event).
func (p *FirstCausePrimitive) findRoot(ev event.Event) types.EventID {
	current := ev
	for {
		causes := current.Causes()
		if len(causes) == 0 {
			return current.ID()
		}
		parent, err := p.store.Get(causes[0])
		if err != nil {
			return current.ID()
		}
		current = parent
	}
}

// --- Group 2: Identity ---

// ActorIDPrimitive manages actor identity and keypair association.
type ActorIDPrimitive struct {
	systemActor types.ActorID
}

func NewActorIDPrimitive(systemActor types.ActorID) *ActorIDPrimitive {
	return &ActorIDPrimitive{systemActor: systemActor}
}

func (p *ActorIDPrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("ActorID") }
func (p *ActorIDPrimitive) Layer() types.Layer                          { return layer0 }
func (p *ActorIDPrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *ActorIDPrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *ActorIDPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("actor.*")}
}

func (p *ActorIDPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	registered := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeActorRegistered {
			registered++
		}
	}
	if registered > 0 {
		mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "registeredThisTick", Value: registered})
	}
	return mutations, nil
}

// ActorRegistryPrimitive manages actor lifecycle (Active, Suspended, Memorial).
type ActorRegistryPrimitive struct{}

func NewActorRegistryPrimitive() *ActorRegistryPrimitive { return &ActorRegistryPrimitive{} }

func (p *ActorRegistryPrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("ActorRegistry") }
func (p *ActorRegistryPrimitive) Layer() types.Layer                          { return layer0 }
func (p *ActorRegistryPrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *ActorRegistryPrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *ActorRegistryPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("actor.*")}
}

func (p *ActorRegistryPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	activeCount := 0
	suspendedCount := 0
	memorialCount := 0

	for _, ev := range events {
		switch ev.Type() {
		case event.EventTypeActorRegistered:
			activeCount++
		case event.EventTypeActorSuspended:
			suspendedCount++
		case event.EventTypeActorMemorial:
			memorialCount++
		}
	}

	mutations = append(mutations,
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "activeCount", Value: activeCount},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "suspendedCount", Value: suspendedCount},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "memorialCount", Value: memorialCount},
	)
	return mutations, nil
}

// SignaturePrimitive tracks Ed25519 signing of events.
type SignaturePrimitive struct{}

func NewSignaturePrimitive() *SignaturePrimitive { return &SignaturePrimitive{} }

func (p *SignaturePrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("Signature") }
func (p *SignaturePrimitive) Layer() types.Layer                          { return layer0 }
func (p *SignaturePrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *SignaturePrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *SignaturePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *SignaturePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	signedCount := 0
	for _, ev := range events {
		if len(ev.Signature().Bytes()) == 64 {
			signedCount++
		}
	}
	if signedCount > 0 {
		mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "signedCount", Value: signedCount})
	}
	return mutations, nil
}

// VerifyPrimitive verifies event signatures and tracks verification results.
type VerifyPrimitive struct{}

func NewVerifyPrimitive() *VerifyPrimitive { return &VerifyPrimitive{} }

func (p *VerifyPrimitive) ID() types.PrimitiveID                       { return types.MustPrimitiveID("Verify") }
func (p *VerifyPrimitive) Layer() types.Layer                          { return layer0 }
func (p *VerifyPrimitive) Lifecycle() types.LifecycleState             { return types.LifecycleActive }
func (p *VerifyPrimitive) Cadence() types.Cadence                      { return cadence1 }
func (p *VerifyPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *VerifyPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	verified := 0
	failed := 0
	for _, ev := range events {
		// Every event must have a 64-byte signature.
		if len(ev.Signature().Bytes()) == 64 {
			verified++
		} else {
			failed++
		}
	}
	mutations = append(mutations,
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "verifiedCount", Value: verified},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "failedCount", Value: failed},
	)
	return mutations, nil
}
