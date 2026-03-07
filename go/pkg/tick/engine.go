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
	currentTick types.Tick
}

// NewEngine creates a tick engine.
func NewEngine(
	registry *primitive.Registry,
	s store.Store,
	actorStore actor.IActorStore,
	factory *event.EventFactory,
	signer event.Signer,
	config Config,
) *Engine {
	return &Engine{
		registry:    registry,
		store:       s,
		actorStore:  actorStore,
		factory:     factory,
		signer:      signer,
		config:      config,
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
	invokedThisTick := make(map[types.PrimitiveID]bool)

	for wavesRun < e.config.MaxWavesPerTick {
		waveMutations, waveErrs := e.runWave(tick, wavesRun, waveEvents, snapshot, invokedThisTick)
		allMutationErrors = append(allMutationErrors, waveErrs...)
		wavesRun++

		if len(waveMutations) == 0 {
			break // quiescence
		}

		totalMutations += len(waveMutations)

		// Eagerly persist AddEvent mutations so subsequent waves get real events.
		// Non-AddEvent mutations are deferred to end of tick.
		newEvents, deferred, errs := e.applyAndExtractNewEvents(waveMutations)
		deferredMutations = append(deferredMutations, deferred...)
		allMutationErrors = append(allMutationErrors, errs...)

		if len(newEvents) == 0 {
			break // quiescence — mutations but no new events
		}

		waveEvents = newEvents
		snapshot.PendingEvents = waveEvents
	}

	quiesced := wavesRun < e.config.MaxWavesPerTick

	// 3. Apply deferred (non-AddEvent) mutations
	deferredErrors := e.applyMutations(deferredMutations, tick)
	allMutationErrors = append(allMutationErrors, deferredErrors...)

	return Result{
		Tick:           tick,
		Waves:          wavesRun,
		Mutations:      totalMutations,
		Duration:       time.Since(start),
		Quiesced:       quiesced,
		MutationErrors: allMutationErrors,
	}, nil
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

	return primitive.Snapshot{
		Tick:          tick,
		Primitives:    e.registry.AllStates(),
		PendingEvents: pending,
		RecentEvents:  recent,
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

				// Transition: Active → Processing
				e.registry.SetLifecycle(pw.prim.ID(), types.LifecycleProcessing)

				mutations, err := pw.prim.Process(tick, pw.events, snapshot)
				results[idx] = primResult{
					id:        pw.prim.ID(),
					mutations: mutations,
					err:       err,
				}

				// Transition: Processing → Active (or Emitting → Active if mutations exist)
				if len(mutations) > 0 {
					e.registry.SetLifecycle(pw.prim.ID(), types.LifecycleEmitting)
					e.registry.SetLifecycle(pw.prim.ID(), types.LifecycleActive)
				} else {
					e.registry.SetLifecycle(pw.prim.ID(), types.LifecycleActive)
				}

				// Record last tick
				e.registry.SetLastTick(pw.prim.ID(), tick)
			}(i, w)
		}
		wg.Wait()

		// Mark all invoked primitives (after goroutines complete)
		for _, w := range work {
			invokedThisTick[w.prim.ID()] = true
		}

		// Collect mutations from this layer
		for _, r := range results {
			if r.err != nil {
				waveErrors = append(waveErrors, fmt.Errorf("primitive %s: %w", r.id.Value(), r.err))
				continue
			}
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

		// Cadence gating — only on first invocation per tick
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
				convID = types.MustConversationID("conv_tick_000000000000000000000001")
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
	panic("unreachable: AddEvent should have been handled by applyAndExtractNewEvents")
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

	// Need a cause — use head of chain (causality invariant: no event without causes)
	headOpt, err := a.engine.store.Head()
	if err != nil {
		a.err = err
		return
	}
	if !headOpt.IsSome() {
		a.err = fmt.Errorf("cannot create edge event: store has no head event (causality invariant)")
		return
	}
	causes := []types.EventID{headOpt.Unwrap().ID()}

	ev, err := a.engine.factory.Create(
		event.EventTypeEdgeCreated, m.From, content, causes,
		tickConversationID,
		a.engine.store, a.engine.signer,
	)
	if err != nil {
		a.err = err
		return
	}
	if _, err := a.engine.store.Append(ev); err != nil {
		a.err = err
	}
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
