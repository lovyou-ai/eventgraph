// Package tick implements the ripple-wave processor — the system's heartbeat.
// Each tick processes pending events through eligible primitives, collecting
// mutations and applying them atomically. New events from mutations trigger
// further waves within the same tick until quiescence or the wave limit.
package tick

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/primitive"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// tickConversationID is the default conversation ID for engine-generated events
// when primitives don't specify one.
var tickConversationID = types.MustConversationID("conv_tick_000000000000000000000001")

// Config controls tick engine behaviour.
type Config struct {
	MaxWavesPerTick int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxWavesPerTick: 10,
	}
}

// Result is the outcome of a single tick.
type Result struct {
	Tick           types.Tick
	Waves          int
	Mutations      int
	Duration       time.Duration
	Quiesced       bool
	MutationErrors []error // errors from individual mutation applications
}

// EventPublisher is called after events are persisted to notify subscribers.
// The Graph layer wires this to its EventBus.Publish.
type EventPublisher func(ev event.Event)

// Engine is the ripple-wave tick processor.
// Not safe for concurrent Tick() calls — callers must serialise externally
// or rely on the internal mutex.
type Engine struct {
	mu          sync.Mutex
	registry   *primitive.Registry
	store      store.Store
	actorStore actor.IActorStore
	factory    *event.EventFactory
	config     Config
	signer     event.Signer
	publisher  EventPublisher
	currentTick types.Tick
}

// NewEngine creates a tick engine.
// publisher is optional — if non-nil, it is called after each event is persisted.
func NewEngine(
	registry *primitive.Registry,
	s store.Store,
	actorStore actor.IActorStore,
	factory *event.EventFactory,
	signer event.Signer,
	config Config,
	publisher EventPublisher,
) *Engine {
	return &Engine{
		registry:    registry,
		store:       s,
		actorStore:  actorStore,
		factory:     factory,
		signer:      signer,
		config:      config,
		publisher:   publisher,
		currentTick: types.MustTick(0),
	}
}

// Tick runs a single tick. Returns the result.
func (e *Engine) Tick(pendingEvents []event.Event) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	start := time.Now()

	// Advance tick counter
	nextVal := e.currentTick.Value() + 1
	tick, err := types.NewTick(nextVal)
	if err != nil {
		return Result{}, fmt.Errorf("tick overflow: %w", err)
	}
	e.currentTick = tick

	// 1. Snapshot
	snapshot := e.buildSnapshot(tick, pendingEvents)

	// 2. Ripple-wave loop
	var deferredMutations []primitive.Mutation
	var allMutationErrors []error
	totalMutations := 0
	waveEvents := pendingEvents
	wavesRun := 0
	quiesced := false
	invokedThisTick := make(map[types.PrimitiveID]bool)

	for wavesRun < e.config.MaxWavesPerTick {
		waveMutations, waveErrs := e.runWave(tick, wavesRun, waveEvents, snapshot, invokedThisTick)
		allMutationErrors = append(allMutationErrors, waveErrs...)
		wavesRun++

		if len(waveMutations) == 0 {
			quiesced = true
			break
		}

		// Eagerly persist AddEvent mutations so subsequent waves get real events.
		// Non-AddEvent mutations are deferred to end of tick.
		newEvents, deferred, errs := e.applyAndExtractNewEvents(waveMutations)
		allMutationErrors = append(allMutationErrors, errs...)
		// Only keep deferred mutations if no AddEvent mutations failed.
		// Deferred mutations (AddEdge, etc.) may reference cause IDs from
		// failed AddEvent mutations in the same wave, which would violate
		// the causality invariant when applied.
		if len(errs) == 0 {
			deferredMutations = append(deferredMutations, deferred...)
		}

		// Count successfully persisted events now; deferred mutations are counted
		// after application at end of tick (some may fail).
		totalMutations += len(newEvents)

		if len(newEvents) == 0 {
			// Only mark as quiesced if there were no errors — if all AddEvent
			// mutations failed, this is an error state, not stable quiescence.
			if len(errs) == 0 {
				quiesced = true
			}
			break
		}

		waveEvents = newEvents
		pendingCopy := make([]event.Event, len(waveEvents))
		copy(pendingCopy, waveEvents)
		snapshot.PendingEvents = pendingCopy

		// Refresh snapshot so subsequent waves see state changes from this wave
		snapshot.Primitives = e.registry.AllStates()

		recentPage, recentErr := e.store.Recent(100, types.None[types.Cursor]())
		if recentErr != nil {
			allMutationErrors = append(allMutationErrors, fmt.Errorf("inter-wave snapshot refresh: %w", recentErr))
		} else if recentPage.Items() != nil {
			recentCopy := make([]event.Event, len(recentPage.Items()))
			copy(recentCopy, recentPage.Items())
			snapshot.RecentEvents = recentCopy
		}
	}

	// 3. Apply deferred (non-AddEvent) mutations
	deferredErrors := e.applyMutations(deferredMutations, tick)
	allMutationErrors = append(allMutationErrors, deferredErrors...)
	totalMutations += len(deferredMutations) - len(deferredErrors)

	return Result{
		Tick:           tick,
		Waves:          wavesRun,
		Mutations:      totalMutations,
		Duration:       time.Since(start),
		Quiesced:       quiesced,
		MutationErrors: allMutationErrors,
	}, nil
}

// publish notifies the publisher (if set) of a persisted event.
func (e *Engine) publish(ev event.Event) {
	if e.publisher != nil {
		e.publisher(ev)
	}
}

// CurrentTick returns the current tick counter.
func (e *Engine) CurrentTick() types.Tick {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.currentTick
}

func (e *Engine) buildSnapshot(tick types.Tick, pending []event.Event) primitive.Snapshot {
	// Get recent events for context
	recentPage, _ := e.store.Recent(100, types.None[types.Cursor]())
	var recent []event.Event
	if recentPage.Items() != nil {
		recent = recentPage.Items()
	}

	// Get active actors
	var activeActors []actor.IActor
	actorPage, err := e.actorStore.List(actor.ActorFilter{
		Status: types.Some(types.ActorStatusActive),
		Limit:  1000,
	})
	if err == nil {
		activeActors = actorPage.Items()
	}

	// Defensive copy of event slices — primitives must not share backing arrays
	// (Frozen<Snapshot> invariant: deeply immutable views).
	pendingCopy := make([]event.Event, len(pending))
	copy(pendingCopy, pending)
	recentCopy := make([]event.Event, len(recent))
	copy(recentCopy, recent)

	return primitive.Snapshot{
		Tick:          tick,
		Primitives:    e.registry.AllStates(),
		PendingEvents: pendingCopy,
		RecentEvents:  recentCopy,
		ActiveActors:  activeActors,
	}
}

func (e *Engine) runWave(tick types.Tick, wave int, events []event.Event, snapshot primitive.Snapshot, invokedThisTick map[types.PrimitiveID]bool) ([]primitive.Mutation, []error) {
	// 1. Determine eligible primitives
	eligible := e.eligiblePrimitives(tick, snapshot, invokedThisTick)

	// 2. Group by layer
	byLayer := make(map[int][]primitive.Primitive)
	for _, p := range eligible {
		l := p.Layer().Value()
		byLayer[l] = append(byLayer[l], p)
	}

	layers := make([]int, 0, len(byLayer))
	for l := range byLayer {
		layers = append(layers, l)
	}
	sort.Ints(layers)

	// 3. Process layer by layer (sequential between layers, parallel within)
	var allMutations []primitive.Mutation
	var waveErrors []error

	for _, layer := range layers {
		prims := byLayer[layer]

		// Match events to subscribers
		type primEvents struct {
			prim   primitive.Primitive
			events []event.Event
		}
		var work []primEvents
		for _, p := range prims {
			matching := matchEvents(p, events)
			// On subsequent waves (re-invocation), only invoke primitives that
			// have matching events. This prevents cadence-bypassed primitives
			// from re-firing with empty event slices on every wave.
			// First-wave primitives are always invoked — they may do work
			// based on the snapshot alone even with no matching events.
			if len(matching) == 0 && invokedThisTick[p.ID()] {
				continue
			}
			work = append(work, primEvents{prim: p, events: matching})
		}

		// Process primitives within the same layer concurrently
		type primResult struct {
			id        types.PrimitiveID
			mutations []primitive.Mutation
			err       error
		}

		results := make([]primResult, len(work))
		var wg sync.WaitGroup
		for i, w := range work {
			wg.Add(1)
			go func(idx int, pw primEvents) {
				defer wg.Done()

				pid := pw.prim.ID()

				// Transition: Active → Processing
				if err := e.registry.SetLifecycle(pid, types.LifecycleProcessing); err != nil {
					results[idx] = primResult{id: pid, err: fmt.Errorf("lifecycle Active→Processing: %w", err)}
					return
				}

				// Recover panics from Process() — restore lifecycle and record the error.
				// Without this, a panicking primitive would crash the process and leave
				// the primitive permanently stuck in Processing.
				var mutations []primitive.Mutation
				var processErr error
				func() {
					defer func() {
						if r := recover(); r != nil {
							processErr = fmt.Errorf("primitive %s panicked: %v", pid.Value(), r)
						}
					}()
					mutations, processErr = pw.prim.Process(tick, pw.events, snapshot)
				}()

				results[idx] = primResult{
					id:        pid,
					mutations: mutations,
					err:       processErr,
				}

				// Transition: Processing → Active (or Emitting → Active if mutations exist and no error).
				// Only transition through Emitting when Process succeeded with mutations,
				// since Emitting implies mutations will be applied.
				// Preserve the original processErr if lifecycle restoration also fails.
				var lcErr error
				if len(mutations) > 0 && processErr == nil {
					if err := e.registry.SetLifecycle(pid, types.LifecycleEmitting); err != nil {
						lcErr = fmt.Errorf("lifecycle Processing→Emitting: %w", err)
					} else if err := e.registry.SetLifecycle(pid, types.LifecycleActive); err != nil {
						// Emitting→Active failed — primitive is stuck in Emitting with no
						// valid exit. Force-reset to Active to prevent permanent lockout.
						// This is a defensive recovery; the error is still reported.
						_ = e.registry.ForceLifecycle(pid, types.LifecycleActive)
						lcErr = fmt.Errorf("lifecycle Emitting→Active: %w (force-recovered)", err)
					}
				} else {
					if err := e.registry.SetLifecycle(pid, types.LifecycleActive); err != nil {
						lcErr = fmt.Errorf("lifecycle Processing→Active: %w", err)
					}
				}
				if processErr != nil && lcErr != nil {
					results[idx].err = fmt.Errorf("%w; additionally: %v", processErr, lcErr)
				} else if lcErr != nil {
					results[idx].err = lcErr
				}

				// Record last tick only on success
				if results[idx].err == nil {
					e.registry.SetLastTick(pid, tick)
				}
			}(i, w)
		}
		wg.Wait()

		// Collect mutations from this layer; only mark successful primitives as invoked
		for _, r := range results {
			if r.err != nil {
				waveErrors = append(waveErrors, fmt.Errorf("primitive %s: %w", r.id.Value(), r.err))
				continue
			}
			invokedThisTick[r.id] = true
			allMutations = append(allMutations, r.mutations...)
		}
	}

	return allMutations, waveErrors
}

func (e *Engine) eligiblePrimitives(tick types.Tick, snapshot primitive.Snapshot, invokedThisTick map[types.PrimitiveID]bool) []primitive.Primitive {
	all := e.registry.All()
	eligible := make([]primitive.Primitive, 0, len(all))

	for _, p := range all {
		// Must be Active
		if e.registry.Lifecycle(p.ID()) != types.LifecycleActive {
			continue
		}

		// Cadence gating — applied on first invocation per tick.
		// Re-invocation within the same tick (subsequent waves) is permitted
		// because new events from earlier waves may need processing.
		// The event-matching filter in runWave ensures primitives only fire
		// when they have subscribed events to process.
		if !invokedThisTick[p.ID()] {
			lastTick := e.registry.LastTick(p.ID())
			elapsed := tick.Value() - lastTick.Value()
			if elapsed < p.Cadence().Value() {
				continue
			}
		}

		// Layer constraint
		if !layerStable(p.Layer(), snapshot) {
			continue
		}

		eligible = append(eligible, p)
	}

	return eligible
}

// layerStable returns true if all registered Layer N-1 primitives are Active and
// have been invoked at least once. Vacuously true when no Layer N-1 primitives
// are registered — sparse layer configurations are permitted.
func layerStable(layer types.Layer, snapshot primitive.Snapshot) bool {
	if layer.Value() == 0 {
		return true // Layer 0 always eligible
	}

	targetLayer := layer.Value() - 1
	for _, ps := range snapshot.Primitives {
		if ps.Layer.Value() == targetLayer {
			if ps.Lifecycle != types.LifecycleActive {
				return false
			}
			if ps.LastTick.Value() == 0 {
				return false // never invoked
			}
		}
	}
	return true
}

func matchEvents(p primitive.Primitive, events []event.Event) []event.Event {
	subs := p.Subscriptions()
	if len(subs) == 0 {
		return nil
	}

	var matched []event.Event
	for _, ev := range events {
		for _, pattern := range subs {
			if pattern.Matches(ev.Type()) {
				matched = append(matched, ev)
				break
			}
		}
	}
	return matched
}

// applyAndExtractNewEvents eagerly persists AddEvent mutations between waves
// so that subsequent waves receive real events with valid IDs and hashes.
// Non-AddEvent mutations are returned for deferred application.
func (e *Engine) applyAndExtractNewEvents(mutations []primitive.Mutation) (newEvents []event.Event, deferred []primitive.Mutation, errs []error) {
	for _, m := range mutations {
		if ae, ok := m.(primitive.AddEvent); ok {
			convID := ae.ConversationID
			if convID == (types.ConversationID{}) {
				convID = tickConversationID
			}
			ev, err := e.factory.Create(
				ae.Type, ae.Source, ae.Content, ae.Causes,
				convID, e.store, e.signer,
			)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			stored, err := e.store.Append(ev)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			e.publish(stored)
			newEvents = append(newEvents, stored)
		} else {
			deferred = append(deferred, m)
		}
	}
	return
}

func (e *Engine) applyMutations(mutations []primitive.Mutation, tick types.Tick) []error {
	applier := &mutationApplier{
		engine: e,
		tick:   tick,
	}

	var errs []error
	for _, m := range mutations {
		m.Accept(applier)
		if applier.err != nil {
			errs = append(errs, applier.err)
			applier.err = nil
		}
	}
	return errs
}

type mutationApplier struct {
	engine *Engine
	tick   types.Tick
	err    error
}

func (a *mutationApplier) VisitAddEvent(_ primitive.AddEvent) {
	// AddEvent mutations are handled eagerly by applyAndExtractNewEvents between waves.
	// If this is reached, it means the split logic has a bug.
	a.err = fmt.Errorf("invariant violation: AddEvent reached applyMutations (should have been handled by applyAndExtractNewEvents)")
}

func (a *mutationApplier) VisitAddEdge(m primitive.AddEdge) {
	content := event.EdgeCreatedContent{
		From:      m.From,
		To:        m.To,
		EdgeType:  m.EdgeType,
		Weight:    m.Weight,
		Direction: event.EdgeDirectionCentripetal,
		Scope:     m.Scope,
	}

	// Use declared causes if provided, otherwise fall back to chain head.
	causes := m.Causes
	if len(causes) == 0 {
		headOpt, err := a.engine.store.Head()
		if err != nil {
			a.err = err
			return
		}
		if !headOpt.IsSome() {
			a.err = fmt.Errorf("cannot create edge event: store has no head event (causality invariant)")
			return
		}
		causes = []types.EventID{headOpt.Unwrap().ID()}
	}

	ev, err := a.engine.factory.Create(
		event.EventTypeEdgeCreated, m.From, content, causes,
		tickConversationID,
		a.engine.store, a.engine.signer,
	)
	if err != nil {
		a.err = err
		return
	}
	stored, err := a.engine.store.Append(ev)
	if err != nil {
		a.err = err
		return
	}
	a.engine.publish(stored)
}

func (a *mutationApplier) VisitUpdateState(m primitive.UpdateState) {
	if err := a.engine.registry.UpdateState(m.PrimitiveID, m.Key, m.Value); err != nil {
		a.err = err
	}
}

func (a *mutationApplier) VisitUpdateActivation(m primitive.UpdateActivation) {
	if err := a.engine.registry.SetActivation(m.PrimitiveID, m.Level); err != nil {
		a.err = err
	}
}

func (a *mutationApplier) VisitUpdateLifecycle(m primitive.UpdateLifecycle) {
	if err := a.engine.registry.SetLifecycle(m.PrimitiveID, m.State); err != nil {
		a.err = err
	}
}
