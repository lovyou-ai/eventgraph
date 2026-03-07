// Package layer0 implements the Layer 0 foundation primitives.
// Groups 0-10 (Core, Causality, Identity, Expectations, Trust, Confidence,
// Instrumentation, Query, Integrity, Deception, Health).
package layer0

import (
	"strings"
	"time"

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

// --- Group 3: Expectations ---

// ExpectationPrimitive tracks expectations set by events and monitors for their fulfilment.
type ExpectationPrimitive struct {
	store store.Store
}

func NewExpectationPrimitive(s store.Store) *ExpectationPrimitive {
	return &ExpectationPrimitive{store: s}
}

func (p *ExpectationPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Expectation") }
func (p *ExpectationPrimitive) Layer() types.Layer               { return layer0 }
func (p *ExpectationPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *ExpectationPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *ExpectationPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *ExpectationPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	pending := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeAuthorityRequested {
			pending++
		}
	}
	mutations = append(mutations,
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "pendingExpectations", Value: pending},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastTick", Value: tick.Value()},
	)
	return mutations, nil
}

// TimeoutPrimitive monitors for expired expectations based on deadlines.
type TimeoutPrimitive struct{}

func NewTimeoutPrimitive() *TimeoutPrimitive { return &TimeoutPrimitive{} }

func (p *TimeoutPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Timeout") }
func (p *TimeoutPrimitive) Layer() types.Layer               { return layer0 }
func (p *TimeoutPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *TimeoutPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *TimeoutPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("authority.*")}
}

func (p *TimeoutPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	timeouts := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeAuthorityTimeout {
			timeouts++
		}
	}
	mutations = append(mutations,
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "timeoutCount", Value: timeouts},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastCheck", Value: time.Now().UnixNano()},
	)
	return mutations, nil
}

// ViolationPrimitive detects and records when expectations are not met.
type ViolationPrimitive struct{}

func NewViolationPrimitive() *ViolationPrimitive { return &ViolationPrimitive{} }

func (p *ViolationPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Violation") }
func (p *ViolationPrimitive) Layer() types.Layer               { return layer0 }
func (p *ViolationPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *ViolationPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *ViolationPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("violation.*")}
}

func (p *ViolationPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	detected := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeViolationDetected {
			detected++
		}
	}
	mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "violationCount", Value: detected})
	return mutations, nil
}

// SeverityPrimitive classifies events by severity level.
type SeverityPrimitive struct{}

func NewSeverityPrimitive() *SeverityPrimitive { return &SeverityPrimitive{} }

func (p *SeverityPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Severity") }
func (p *SeverityPrimitive) Layer() types.Layer               { return layer0 }
func (p *SeverityPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *SeverityPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *SeverityPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("violation.*")}
}

func (p *SeverityPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	for _, ev := range events {
		if ev.Type() == event.EventTypeViolationDetected {
			if vc, ok := ev.Content().(event.ViolationDetectedContent); ok {
				mutations = append(mutations, primitive.UpdateState{
					PrimitiveID: p.ID(), Key: "lastSeverity", Value: string(vc.Severity),
				})
			}
		}
	}
	return mutations, nil
}

// --- Group 4: Trust ---

// TrustScorePrimitive monitors trust score snapshots.
type TrustScorePrimitive struct{}

func NewTrustScorePrimitive() *TrustScorePrimitive { return &TrustScorePrimitive{} }

func (p *TrustScorePrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("TrustScore") }
func (p *TrustScorePrimitive) Layer() types.Layer               { return layer0 }
func (p *TrustScorePrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *TrustScorePrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *TrustScorePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("trust.*")}
}

func (p *TrustScorePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	scores := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeTrustScore {
			scores++
		}
	}
	mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "scoreSnapshots", Value: scores})
	return mutations, nil
}

// TrustUpdatePrimitive tracks trust changes between actors.
type TrustUpdatePrimitive struct{}

func NewTrustUpdatePrimitive() *TrustUpdatePrimitive { return &TrustUpdatePrimitive{} }

func (p *TrustUpdatePrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("TrustUpdate") }
func (p *TrustUpdatePrimitive) Layer() types.Layer               { return layer0 }
func (p *TrustUpdatePrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *TrustUpdatePrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *TrustUpdatePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("trust.*")}
}

func (p *TrustUpdatePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	updates := 0
	decays := 0
	for _, ev := range events {
		switch ev.Type() {
		case event.EventTypeTrustUpdated:
			updates++
		case event.EventTypeTrustDecayed:
			decays++
		}
	}
	mutations = append(mutations,
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "trustUpdates", Value: updates},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "trustDecays", Value: decays},
	)
	return mutations, nil
}

// CorroborationPrimitive detects when multiple sources agree, strengthening trust.
type CorroborationPrimitive struct{}

func NewCorroborationPrimitive() *CorroborationPrimitive { return &CorroborationPrimitive{} }

func (p *CorroborationPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Corroboration") }
func (p *CorroborationPrimitive) Layer() types.Layer               { return layer0 }
func (p *CorroborationPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *CorroborationPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *CorroborationPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("trust.*")}
}

func (p *CorroborationPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	// Track unique sources that updated trust in the same direction.
	sources := make(map[string]bool)
	for _, ev := range events {
		if ev.Type() == event.EventTypeTrustUpdated {
			sources[ev.Source().Value()] = true
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "uniqueSources", Value: len(sources)},
	}, nil
}

// ContradictionPrimitive detects when sources disagree, weakening confidence.
type ContradictionPrimitive struct{}

func NewContradictionPrimitive() *ContradictionPrimitive { return &ContradictionPrimitive{} }

func (p *ContradictionPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Contradiction") }
func (p *ContradictionPrimitive) Layer() types.Layer               { return layer0 }
func (p *ContradictionPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *ContradictionPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *ContradictionPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("trust.*")}
}

func (p *ContradictionPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	// Detect trust updates that move in opposite directions for the same actor.
	increases := 0
	decreases := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeTrustUpdated {
			if tc, ok := ev.Content().(event.TrustUpdatedContent); ok {
				if tc.Current.Value() > tc.Previous.Value() {
					increases++
				} else if tc.Current.Value() < tc.Previous.Value() {
					decreases++
				}
			}
		}
	}
	contradictions := 0
	if increases > 0 && decreases > 0 {
		contradictions = 1
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "contradictions", Value: contradictions},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "increases", Value: increases},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "decreases", Value: decreases},
	}, nil
}

// --- Group 5: Confidence ---

// ConfidencePrimitive tracks confidence levels across decisions.
type ConfidencePrimitive struct{}

func NewConfidencePrimitive() *ConfidencePrimitive { return &ConfidencePrimitive{} }

func (p *ConfidencePrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Confidence") }
func (p *ConfidencePrimitive) Layer() types.Layer               { return layer0 }
func (p *ConfidencePrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *ConfidencePrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *ConfidencePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("decision.*")}
}

func (p *ConfidencePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	decisions := 0
	for _, ev := range events {
		if strings.HasPrefix(ev.Type().Value(), "decision.") {
			decisions++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "decisionsThisTick", Value: decisions},
	}, nil
}

// EvidencePrimitive tracks evidence chains supporting decisions.
type EvidencePrimitive struct {
	store store.Store
}

func NewEvidencePrimitive(s store.Store) *EvidencePrimitive {
	return &EvidencePrimitive{store: s}
}

func (p *EvidencePrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Evidence") }
func (p *EvidencePrimitive) Layer() types.Layer               { return layer0 }
func (p *EvidencePrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *EvidencePrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *EvidencePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *EvidencePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	totalCauses := 0
	for _, ev := range events {
		totalCauses += len(ev.Causes())
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "evidenceLinks", Value: totalCauses},
	}, nil
}

// RevisionPrimitive tracks grammar retractions (content corrections).
type RevisionPrimitive struct{}

func NewRevisionPrimitive() *RevisionPrimitive { return &RevisionPrimitive{} }

func (p *RevisionPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Revision") }
func (p *RevisionPrimitive) Layer() types.Layer               { return layer0 }
func (p *RevisionPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *RevisionPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *RevisionPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("grammar.*")}
}

func (p *RevisionPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	retractions := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeGrammarRetract {
			retractions++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "retractions", Value: retractions},
	}, nil
}

// UncertaintyPrimitive monitors for low-confidence decisions that may need escalation.
type UncertaintyPrimitive struct{}

func NewUncertaintyPrimitive() *UncertaintyPrimitive { return &UncertaintyPrimitive{} }

func (p *UncertaintyPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Uncertainty") }
func (p *UncertaintyPrimitive) Layer() types.Layer               { return layer0 }
func (p *UncertaintyPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *UncertaintyPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *UncertaintyPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("decision.*")}
}

func (p *UncertaintyPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	escalations := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeAuthorityRequested {
			escalations++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "escalations", Value: escalations},
	}, nil
}

// --- Group 6: Instrumentation ---

// InstrumentationSpecPrimitive defines what should be measured in the graph.
type InstrumentationSpecPrimitive struct{}

func NewInstrumentationSpecPrimitive() *InstrumentationSpecPrimitive {
	return &InstrumentationSpecPrimitive{}
}

func (p *InstrumentationSpecPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("InstrumentationSpec") }
func (p *InstrumentationSpecPrimitive) Layer() types.Layer               { return layer0 }
func (p *InstrumentationSpecPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *InstrumentationSpecPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *InstrumentationSpecPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("health.*")}
}

func (p *InstrumentationSpecPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "primitivesTracked", Value: len(snap.Primitives)},
	}, nil
}

// CoverageCheckPrimitive verifies that all event types have subscribers.
type CoverageCheckPrimitive struct{}

func NewCoverageCheckPrimitive() *CoverageCheckPrimitive { return &CoverageCheckPrimitive{} }

func (p *CoverageCheckPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("CoverageCheck") }
func (p *CoverageCheckPrimitive) Layer() types.Layer               { return layer0 }
func (p *CoverageCheckPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *CoverageCheckPrimitive) Cadence() types.Cadence           { return types.MustCadence(5) }
func (p *CoverageCheckPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("health.*")}
}

func (p *CoverageCheckPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	active := 0
	for _, ps := range snap.Primitives {
		if ps.Lifecycle == types.LifecycleActive {
			active++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "activePrimitives", Value: active},
	}, nil
}

// GapPrimitive detects gaps in event coverage — time periods with no events.
type GapPrimitive struct{}

func NewGapPrimitive() *GapPrimitive { return &GapPrimitive{} }

func (p *GapPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Gap") }
func (p *GapPrimitive) Layer() types.Layer               { return layer0 }
func (p *GapPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *GapPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *GapPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *GapPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	gap := 0
	if len(events) == 0 {
		gap = 1
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "gapDetected", Value: gap},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "eventsInTick", Value: len(events)},
	}, nil
}

// BlindPrimitive detects blind spots — areas of the graph with no instrumentation.
type BlindPrimitive struct{}

func NewBlindPrimitive() *BlindPrimitive { return &BlindPrimitive{} }

func (p *BlindPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Blind") }
func (p *BlindPrimitive) Layer() types.Layer               { return layer0 }
func (p *BlindPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *BlindPrimitive) Cadence() types.Cadence           { return types.MustCadence(5) }
func (p *BlindPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("health.*")}
}

func (p *BlindPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	dormant := 0
	for _, ps := range snap.Primitives {
		if ps.Lifecycle == types.LifecycleDormant {
			dormant++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "dormantPrimitives", Value: dormant},
	}, nil
}

// --- Group 7: Query ---

// PathQueryPrimitive supports querying paths between events in the causal DAG.
type PathQueryPrimitive struct {
	store store.Store
}

func NewPathQueryPrimitive(s store.Store) *PathQueryPrimitive {
	return &PathQueryPrimitive{store: s}
}

func (p *PathQueryPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("PathQuery") }
func (p *PathQueryPrimitive) Layer() types.Layer               { return layer0 }
func (p *PathQueryPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *PathQueryPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *PathQueryPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("query.*")}
}

func (p *PathQueryPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	queries := len(events)
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "queriesProcessed", Value: queries},
	}, nil
}

// SubgraphExtractPrimitive extracts subgraphs around events of interest.
type SubgraphExtractPrimitive struct {
	store store.Store
}

func NewSubgraphExtractPrimitive(s store.Store) *SubgraphExtractPrimitive {
	return &SubgraphExtractPrimitive{store: s}
}

func (p *SubgraphExtractPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("SubgraphExtract") }
func (p *SubgraphExtractPrimitive) Layer() types.Layer               { return layer0 }
func (p *SubgraphExtractPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *SubgraphExtractPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *SubgraphExtractPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("query.*")}
}

func (p *SubgraphExtractPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	// For each query event, count ancestors + descendants to measure subgraph size.
	totalSize := 0
	for _, ev := range events {
		anc, _ := p.store.Ancestors(ev.ID(), 5)
		desc, _ := p.store.Descendants(ev.ID(), 5)
		totalSize += len(anc) + len(desc) + 1
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastSubgraphSize", Value: totalSize},
	}, nil
}

// AnnotatePrimitive tracks annotation events on the graph.
type AnnotatePrimitive struct{}

func NewAnnotatePrimitive() *AnnotatePrimitive { return &AnnotatePrimitive{} }

func (p *AnnotatePrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Annotate") }
func (p *AnnotatePrimitive) Layer() types.Layer               { return layer0 }
func (p *AnnotatePrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *AnnotatePrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *AnnotatePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("grammar.*")}
}

func (p *AnnotatePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	annotations := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeGrammarAnnotate {
			annotations++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "annotations", Value: annotations},
	}, nil
}

// TimelinePrimitive provides chronological views of event sequences.
type TimelinePrimitive struct {
	store store.Store
}

func NewTimelinePrimitive(s store.Store) *TimelinePrimitive {
	return &TimelinePrimitive{store: s}
}

func (p *TimelinePrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Timeline") }
func (p *TimelinePrimitive) Layer() types.Layer               { return layer0 }
func (p *TimelinePrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *TimelinePrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *TimelinePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("query.*")}
}

func (p *TimelinePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	count, _ := p.store.Count()
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "totalEvents", Value: count},
	}, nil
}

// --- Group 8: Integrity ---

// HashChainPrimitive maintains and monitors the hash chain.
type HashChainPrimitive struct {
	store store.Store
}

func NewHashChainPrimitive(s store.Store) *HashChainPrimitive {
	return &HashChainPrimitive{store: s}
}

func (p *HashChainPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("HashChain") }
func (p *HashChainPrimitive) Layer() types.Layer               { return layer0 }
func (p *HashChainPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *HashChainPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *HashChainPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *HashChainPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	var mutations []primitive.Mutation
	head, err := p.store.Head()
	if err == nil && head.IsSome() {
		mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "chainHead", Value: head.Unwrap().Hash().Value()})
	}
	count, _ := p.store.Count()
	mutations = append(mutations, primitive.UpdateState{PrimitiveID: p.ID(), Key: "chainLength", Value: count})
	return mutations, nil
}

// ChainVerifyPrimitive periodically verifies hash chain integrity.
type ChainVerifyPrimitive struct {
	store store.Store
}

func NewChainVerifyPrimitive(s store.Store) *ChainVerifyPrimitive {
	return &ChainVerifyPrimitive{store: s}
}

func (p *ChainVerifyPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("ChainVerify") }
func (p *ChainVerifyPrimitive) Layer() types.Layer               { return layer0 }
func (p *ChainVerifyPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *ChainVerifyPrimitive) Cadence() types.Cadence           { return types.MustCadence(10) }
func (p *ChainVerifyPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("chain.*")}
}

func (p *ChainVerifyPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	result, err := p.store.VerifyChain()
	if err != nil {
		return []primitive.Mutation{
			primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastVerifyError", Value: err.Error()},
		}, nil
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "chainValid", Value: result.Valid},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "chainLength", Value: result.Length},
	}, nil
}

// WitnessPrimitive records witnessing of events for third-party verification.
type WitnessPrimitive struct{}

func NewWitnessPrimitive() *WitnessPrimitive { return &WitnessPrimitive{} }

func (p *WitnessPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Witness") }
func (p *WitnessPrimitive) Layer() types.Layer               { return layer0 }
func (p *WitnessPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *WitnessPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *WitnessPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *WitnessPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "witnessed", Value: len(events)},
	}, nil
}

// IntegrityViolationPrimitive detects and records chain integrity violations.
type IntegrityViolationPrimitive struct{}

func NewIntegrityViolationPrimitive() *IntegrityViolationPrimitive {
	return &IntegrityViolationPrimitive{}
}

func (p *IntegrityViolationPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("IntegrityViolation") }
func (p *IntegrityViolationPrimitive) Layer() types.Layer               { return layer0 }
func (p *IntegrityViolationPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *IntegrityViolationPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *IntegrityViolationPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("chain.*")}
}

func (p *IntegrityViolationPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	broken := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeChainBroken {
			broken++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "chainBreaks", Value: broken},
	}, nil
}

// --- Group 9: Deception ---

// PatternPrimitive detects recurring patterns in event sequences.
type PatternPrimitive struct{}

func NewPatternPrimitive() *PatternPrimitive { return &PatternPrimitive{} }

func (p *PatternPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Pattern") }
func (p *PatternPrimitive) Layer() types.Layer               { return layer0 }
func (p *PatternPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *PatternPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *PatternPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *PatternPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	// Track event type frequency distribution for pattern detection.
	typeCounts := make(map[string]int)
	for _, ev := range events {
		typeCounts[ev.Type().Value()]++
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "uniqueTypes", Value: len(typeCounts)},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "totalEvents", Value: len(events)},
	}, nil
}

// DeceptionIndicatorPrimitive watches for signs of deceptive behaviour.
type DeceptionIndicatorPrimitive struct{}

func NewDeceptionIndicatorPrimitive() *DeceptionIndicatorPrimitive {
	return &DeceptionIndicatorPrimitive{}
}

func (p *DeceptionIndicatorPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("DeceptionIndicator") }
func (p *DeceptionIndicatorPrimitive) Layer() types.Layer               { return layer0 }
func (p *DeceptionIndicatorPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *DeceptionIndicatorPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *DeceptionIndicatorPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{
		types.MustSubscriptionPattern("trust.*"),
		types.MustSubscriptionPattern("violation.*"),
	}
}

func (p *DeceptionIndicatorPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	indicators := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeViolationDetected || ev.Type() == event.EventTypeTrustDecayed {
			indicators++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "indicators", Value: indicators},
	}, nil
}

// SuspicionPrimitive tracks actors with declining trust or repeated violations.
type SuspicionPrimitive struct{}

func NewSuspicionPrimitive() *SuspicionPrimitive { return &SuspicionPrimitive{} }

func (p *SuspicionPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Suspicion") }
func (p *SuspicionPrimitive) Layer() types.Layer               { return layer0 }
func (p *SuspicionPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *SuspicionPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *SuspicionPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("trust.*")}
}

func (p *SuspicionPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	// Track actors whose trust decreased.
	suspectActors := make(map[string]bool)
	for _, ev := range events {
		if ev.Type() == event.EventTypeTrustDecayed {
			if tc, ok := ev.Content().(event.TrustDecayedContent); ok {
				suspectActors[tc.Actor.Value()] = true
			}
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "suspectedActors", Value: len(suspectActors)},
	}, nil
}

// QuarantinePrimitive manages actor quarantine for suspicious behaviour.
type QuarantinePrimitive struct{}

func NewQuarantinePrimitive() *QuarantinePrimitive { return &QuarantinePrimitive{} }

func (p *QuarantinePrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Quarantine") }
func (p *QuarantinePrimitive) Layer() types.Layer               { return layer0 }
func (p *QuarantinePrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *QuarantinePrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *QuarantinePrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("actor.*")}
}

func (p *QuarantinePrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	quarantined := 0
	for _, ev := range events {
		if ev.Type() == event.EventTypeActorSuspended {
			quarantined++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "quarantinedThisTick", Value: quarantined},
	}, nil
}

// --- Group 10: Health ---

// GraphHealthPrimitive monitors overall graph health.
type GraphHealthPrimitive struct {
	store store.Store
}

func NewGraphHealthPrimitive(s store.Store) *GraphHealthPrimitive {
	return &GraphHealthPrimitive{store: s}
}

func (p *GraphHealthPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("GraphHealth") }
func (p *GraphHealthPrimitive) Layer() types.Layer               { return layer0 }
func (p *GraphHealthPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *GraphHealthPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *GraphHealthPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("health.*")}
}

func (p *GraphHealthPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	count, _ := p.store.Count()
	activePrims := 0
	for _, ps := range snap.Primitives {
		if ps.Lifecycle == types.LifecycleActive {
			activePrims++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "eventCount", Value: count},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "activePrimitives", Value: activePrims},
	}, nil
}

// InvariantPrimitive defines system invariants that must hold.
type InvariantPrimitive struct{}

func NewInvariantPrimitive() *InvariantPrimitive { return &InvariantPrimitive{} }

func (p *InvariantPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Invariant") }
func (p *InvariantPrimitive) Layer() types.Layer               { return layer0 }
func (p *InvariantPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *InvariantPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *InvariantPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}

func (p *InvariantPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	// Check: every non-bootstrap event must have causes.
	violations := 0
	for _, ev := range events {
		if !ev.IsBootstrap() && len(ev.Causes()) == 0 {
			violations++
		}
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "invariantViolations", Value: violations},
	}, nil
}

// InvariantCheckPrimitive runs periodic checks on system invariants.
type InvariantCheckPrimitive struct {
	store store.Store
}

func NewInvariantCheckPrimitive(s store.Store) *InvariantCheckPrimitive {
	return &InvariantCheckPrimitive{store: s}
}

func (p *InvariantCheckPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("InvariantCheck") }
func (p *InvariantCheckPrimitive) Layer() types.Layer               { return layer0 }
func (p *InvariantCheckPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *InvariantCheckPrimitive) Cadence() types.Cadence           { return types.MustCadence(10) }
func (p *InvariantCheckPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("chain.*")}
}

func (p *InvariantCheckPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	result, err := p.store.VerifyChain()
	if err != nil {
		return []primitive.Mutation{
			primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastCheckError", Value: err.Error()},
		}, nil
	}
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "chainIntact", Value: result.Valid},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "lastCheckTick", Value: tick.Value()},
	}, nil
}

// BootstrapPrimitive monitors system bootstrap status and ensures proper initialization.
type BootstrapPrimitive struct {
	store store.Store
}

func NewBootstrapPrimitive(s store.Store) *BootstrapPrimitive {
	return &BootstrapPrimitive{store: s}
}

func (p *BootstrapPrimitive) ID() types.PrimitiveID           { return types.MustPrimitiveID("Bootstrap") }
func (p *BootstrapPrimitive) Layer() types.Layer               { return layer0 }
func (p *BootstrapPrimitive) Lifecycle() types.LifecycleState  { return types.LifecycleActive }
func (p *BootstrapPrimitive) Cadence() types.Cadence           { return cadence1 }
func (p *BootstrapPrimitive) Subscriptions() []types.SubscriptionPattern {
	return []types.SubscriptionPattern{types.MustSubscriptionPattern("system.*")}
}

func (p *BootstrapPrimitive) Process(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	bootstrapped := false
	for _, ev := range events {
		if ev.Type() == event.EventTypeSystemBootstrapped {
			bootstrapped = true
		}
	}
	count, _ := p.store.Count()
	return []primitive.Mutation{
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "bootstrapped", Value: bootstrapped},
		primitive.UpdateState{PrimitiveID: p.ID(), Key: "eventCount", Value: count},
	}, nil
}
