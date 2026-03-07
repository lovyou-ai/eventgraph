package tick_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/primitive"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/tick"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

type testSigner struct{}

func (testSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data[:min(64, len(data))])
	return types.MustSignature(sig), nil
}

// testPrimitive is a configurable Primitive for testing.
type testPrimitive struct {
	id            types.PrimitiveID
	layer         types.Layer
	subscriptions []types.SubscriptionPattern
	cadence       types.Cadence
	processFunc   func(types.Tick, []event.Event, primitive.Snapshot) ([]primitive.Mutation, error)
}

func (p *testPrimitive) ID() types.PrimitiveID                { return p.id }
func (p *testPrimitive) Layer() types.Layer                    { return p.layer }
func (p *testPrimitive) Subscriptions() []types.SubscriptionPattern { return p.subscriptions }
func (p *testPrimitive) Cadence() types.Cadence                { return p.cadence }
func (p *testPrimitive) Lifecycle() types.LifecycleState       { return types.LifecycleActive }
func (p *testPrimitive) Process(t types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
	if p.processFunc != nil {
		return p.processFunc(t, events, snap)
	}
	return nil, nil
}

func newEngine(t *testing.T, prims ...*testPrimitive) (*tick.Engine, store.Store, event.Event) {
	t.Helper()
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	registry := primitive.NewRegistry()
	signer := testSigner{}

	for _, p := range prims {
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register: %v", err)
		}
		if err := registry.Activate(p.ID()); err != nil {
			t.Fatalf("Activate: %v", err)
		}
	}

	eventRegistry := event.DefaultRegistry()
	factory := event.NewEventFactory(eventRegistry)

	// Bootstrap the store
	bf := event.NewBootstrapFactory(eventRegistry)
	bootstrap, err := bf.Init(types.MustActorID("actor_system0000000000000000001"), signer)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if _, err := s.Append(bootstrap); err != nil {
		t.Fatalf("Append bootstrap: %v", err)
	}

	e := tick.NewEngine(registry, s, as, factory, signer, tick.DefaultConfig(), nil)
	return e, s, bootstrap
}

// --- Basic tick tests ---

func TestTickEmpty(t *testing.T) {
	e, _, _ := newEngine(t)

	result, err := e.Tick(nil)
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if result.Tick.Value() != 1 {
		t.Errorf("Tick = %d, want 1", result.Tick.Value())
	}
	if !result.Quiesced {
		t.Error("expected quiescence with no primitives")
	}
	if result.Mutations != 0 {
		t.Errorf("Mutations = %d, want 0", result.Mutations)
	}
}

func TestTickNoPrimitives(t *testing.T) {
	e, _, bootstrap := newEngine(t)

	// Create a test event
	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	result, err := e.Tick([]event.Event{ev})
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if !result.Quiesced {
		t.Error("expected quiescence")
	}
}

func TestTickWithPrimitive(t *testing.T) {
	var invoked atomic.Int32

	p := &testPrimitive{
		id:      types.MustPrimitiveID("counter"),
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			invoked.Add(1)
			return nil, nil
		},
	}

	e, _, bootstrap := newEngine(t, p)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	result, err := e.Tick([]event.Event{ev})
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if invoked.Load() != 1 {
		t.Errorf("invoked = %d, want 1", invoked.Load())
	}
	if result.Waves != 1 {
		t.Errorf("Waves = %d, want 1", result.Waves)
	}
}

func TestTickAdvancesCounter(t *testing.T) {
	e, _, _ := newEngine(t)

	r1, _ := e.Tick(nil)
	r2, _ := e.Tick(nil)
	r3, _ := e.Tick(nil)

	if r1.Tick.Value() != 1 || r2.Tick.Value() != 2 || r3.Tick.Value() != 3 {
		t.Errorf("Ticks = %d, %d, %d, want 1, 2, 3",
			r1.Tick.Value(), r2.Tick.Value(), r3.Tick.Value())
	}
}

func TestTickCadenceGating(t *testing.T) {
	var invoked atomic.Int32

	p := &testPrimitive{
		id:      types.MustPrimitiveID("slow_prim"),
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(3), // every 3 ticks
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			invoked.Add(1)
			return nil, nil
		},
	}

	e, _, _ := newEngine(t, p)

	// Tick 1: eligible (0 + 3 <= 1? No. elapsed = 1 - 0 = 1 < 3, skip)
	// Actually: lastTick=0, cadence=3, tick=1 → elapsed=1 < 3 → skip
	// Tick 2: elapsed=2 < 3 → skip
	// Tick 3: elapsed=3 >= 3 → invoke
	e.Tick(nil)
	e.Tick(nil)
	e.Tick(nil)

	if invoked.Load() != 1 {
		t.Errorf("invoked = %d, want 1 (cadence=3)", invoked.Load())
	}

	// Tick 4,5: skip. Tick 6: invoke
	e.Tick(nil)
	e.Tick(nil)
	e.Tick(nil)

	if invoked.Load() != 2 {
		t.Errorf("invoked = %d, want 2 after 6 ticks", invoked.Load())
	}
}

func TestTickSubscriptionFiltering(t *testing.T) {
	var receivedCount atomic.Int32

	p := &testPrimitive{
		id:      types.MustPrimitiveID("trust_watcher"),
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("trust.*"),
		},
		processFunc: func(tick types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			receivedCount.Store(int32(len(events)))
			return nil, nil
		},
	}

	e, _, bootstrap := newEngine(t, p)

	// Send trust event and edge event — primitive should only see trust
	trustEv := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000098"),
		event.EventTypeTrustUpdated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	edgeEv := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeEdgeCreated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.EdgeCreatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	e.Tick([]event.Event{trustEv, edgeEv})

	if receivedCount.Load() != 1 {
		t.Errorf("received %d events, want 1 (only trust.*)", receivedCount.Load())
	}
}

func TestTickMutationProducesEvents(t *testing.T) {
	var wavesSeen atomic.Int32
	actorID := types.MustActorID("actor_system0000000000000000001")

	p := &testPrimitive{
		id:      types.MustPrimitiveID("event_emitter"),
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			wave := wavesSeen.Add(1)
			if wave == 1 {
				// First invocation: emit an event
				return []primitive.Mutation{
					primitive.AddEvent{
						Type:    event.EventTypeTrustUpdated,
						Source:  actorID,
						Content: event.TrustUpdatedContent{},
						Causes:  []types.EventID{events[0].ID()},
					},
				}, nil
			}
			// Second invocation (wave 2): no more mutations → quiesce
			return nil, nil
		},
	}

	e, _, bootstrap := newEngine(t, p)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		actorID, event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	result, err := e.Tick([]event.Event{ev})
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}

	// Wave 0: processes ev, emits AddEvent mutation
	// Wave 1: processes the new event, emits nothing → quiesce
	if result.Waves != 2 {
		t.Errorf("Waves = %d, want 2", result.Waves)
	}
	if !result.Quiesced {
		t.Error("expected quiescence")
	}
}

func TestTickLayerOrdering(t *testing.T) {
	var order []string
	var orderMu sync.Mutex

	makeLayerPrim := func(name string, layer int) *testPrimitive {
		return &testPrimitive{
			id:      types.MustPrimitiveID(name),
			layer:   types.MustLayer(layer),
			cadence: types.MustCadence(1),
			subscriptions: []types.SubscriptionPattern{
				types.MustSubscriptionPattern("*"),
			},
			processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
				orderMu.Lock()
				order = append(order, name)
				orderMu.Unlock()
				return nil, nil
			},
		}
	}

	p0 := makeLayerPrim("layer0", 0)
	p1 := makeLayerPrim("layer1", 1)

	e, _, bootstrap := newEngine(t, p0, p1)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	// Tick 1: Layer 0 runs, Layer 1 blocked (Layer 0 never invoked before)
	e.Tick([]event.Event{ev})
	if len(order) != 1 || order[0] != "layer0" {
		t.Fatalf("tick 1: order = %v, want [layer0]", order)
	}

	// Tick 2: Layer 0 stable (was invoked), Layer 1 now eligible
	order = nil
	e.Tick([]event.Event{ev})
	if len(order) != 2 {
		t.Fatalf("tick 2: invoked %d primitives, want 2", len(order))
	}
	if order[0] != "layer0" {
		t.Errorf("tick 2: first = %q, want layer0", order[0])
	}
	if order[1] != "layer1" {
		t.Errorf("tick 2: second = %q, want layer1", order[1])
	}
}

func TestTickLayerConstraint(t *testing.T) {
	// Layer 1 should not run if Layer 0 has never been invoked
	var layer1Invoked atomic.Int32

	p1 := &testPrimitive{
		id:      types.MustPrimitiveID("layer1_prim"),
		layer:   types.MustLayer(1),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			layer1Invoked.Add(1)
			return nil, nil
		},
	}

	// Register only layer 1, no layer 0 primitives registered
	// Layer constraint: Layer 1 requires Layer 0 to be stable
	// With no Layer 0 primitives, there's nothing unstable, so it should still run
	e, _, bootstrap := newEngine(t, p1)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	e.Tick([]event.Event{ev})

	// Layer 1 should run — no Layer 0 primitives means nothing blocks it
	if layer1Invoked.Load() != 1 {
		t.Errorf("layer1 invoked = %d, want 1", layer1Invoked.Load())
	}
}

func TestTickPrimitiveError(t *testing.T) {
	p := &testPrimitive{
		id:      types.MustPrimitiveID("error_prim"),
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			return nil, fmt.Errorf("something went wrong")
		},
	}

	e, _, bootstrap := newEngine(t, p)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	// Should not crash — errors are logged, not fatal
	result, err := e.Tick([]event.Event{ev})
	if err != nil {
		t.Fatalf("Tick should not fail on primitive error: %v", err)
	}
	if result.Mutations != 0 {
		t.Errorf("Mutations = %d, want 0 (error primitive)", result.Mutations)
	}
}

func TestTickUpdateStateMutation(t *testing.T) {
	primID := types.MustPrimitiveID("stateful")
	p := &testPrimitive{
		id:      primID,
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			return []primitive.Mutation{
				primitive.UpdateState{PrimitiveID: primID, Key: "count", Value: 1},
				primitive.UpdateActivation{PrimitiveID: primID, Level: types.MustActivation(0.8)},
			}, nil
		},
	}

	e, _, bootstrap := newEngine(t, p)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	result, err := e.Tick([]event.Event{ev})
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if result.Mutations != 2 {
		t.Errorf("Mutations = %d, want 2", result.Mutations)
	}
}

func TestTickWaveLimitPreventsInfiniteLoop(t *testing.T) {
	actorID := types.MustActorID("actor_system0000000000000000001")

	p := &testPrimitive{
		id:      types.MustPrimitiveID("infinite_emitter"),
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			// Always emit a new event — would loop forever without wave limit
			// Use a dummy cause — the factory will validate it
			causeID := types.MustEventID("019462a0-0000-7000-8000-000000000099")
			if len(events) > 0 {
				causeID = events[0].ID()
			}
			return []primitive.Mutation{
				primitive.AddEvent{
					Type:    event.EventTypeTrustUpdated,
					Source:  actorID,
					Content: event.TrustUpdatedContent{},
					Causes:  []types.EventID{causeID},
				},
			}, nil
		},
	}

	config := tick.Config{MaxWavesPerTick: 3}
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	registry := primitive.NewRegistry()
	registry.Register(p)
	registry.Activate(p.ID())

	eventRegistry := event.DefaultRegistry()
	factory := event.NewEventFactory(eventRegistry)
	signer := testSigner{}

	bf := event.NewBootstrapFactory(eventRegistry)
	bootstrap, _ := bf.Init(actorID, signer)
	s.Append(bootstrap)

	e := tick.NewEngine(registry, s, as, factory, signer, config, nil)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		actorID, event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	result, err := e.Tick([]event.Event{ev})
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if result.Quiesced {
		t.Error("expected NOT quiesced (hit wave limit)")
	}
	if result.Waves > 3 {
		t.Errorf("Waves = %d, should not exceed MaxWavesPerTick=3", result.Waves)
	}
}

func TestTickCurrentTick(t *testing.T) {
	e, _, _ := newEngine(t)

	if e.CurrentTick().Value() != 0 {
		t.Errorf("initial tick = %d, want 0", e.CurrentTick().Value())
	}

	e.Tick(nil)
	if e.CurrentTick().Value() != 1 {
		t.Errorf("after first tick = %d, want 1", e.CurrentTick().Value())
	}
}

func TestTickDurationNonNegative(t *testing.T) {
	e, _, _ := newEngine(t)
	result, _ := e.Tick(nil)
	if result.Duration < 0 {
		t.Error("Duration should be non-negative")
	}
}

func TestTickAddEdgeMutation(t *testing.T) {
	actorID := types.MustActorID("actor_system0000000000000000001")
	targetID := types.MustActorID("actor_system0000000000000000002")

	p := &testPrimitive{
		id:      types.MustPrimitiveID("edge_creator"),
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			return []primitive.Mutation{
				primitive.AddEdge{
					From:     actorID,
					To:       targetID,
					EdgeType: event.EdgeTypeTrust,
					Weight:   types.MustWeight(0.5),
					Scope:    types.None[types.DomainScope](),
				},
			}, nil
		},
	}

	e, s, bootstrap := newEngine(t, p)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		actorID, event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	result, err := e.Tick([]event.Event{ev})
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if result.Mutations != 1 {
		t.Errorf("Mutations = %d, want 1", result.Mutations)
	}

	// Verify the edge.created event was stored
	page, _ := s.Recent(10, types.None[types.Cursor]())
	found := false
	for _, item := range page.Items() {
		if item.Type() == event.EventTypeEdgeCreated {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected edge.created event in store")
	}
}

func TestTickUpdateLifecycleMutation(t *testing.T) {
	primID := types.MustPrimitiveID("lifecycle_updater")
	var invoked atomic.Int32

	p := &testPrimitive{
		id:      primID,
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			invoked.Add(1)
			// Request transition to Deactivating (Active → Deactivating is valid)
			return []primitive.Mutation{
				primitive.UpdateLifecycle{PrimitiveID: primID, State: types.LifecycleDeactivating},
			}, nil
		},
	}

	e, _, bootstrap := newEngine(t, p)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	result, err := e.Tick([]event.Event{ev})
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if result.Mutations != 1 {
		t.Errorf("Mutations = %d, want 1", result.Mutations)
	}

	// Tick 2: primitive is now Deactivating, should NOT be eligible
	result2, err := e.Tick([]event.Event{ev})
	if err != nil {
		t.Fatalf("Tick 2: %v", err)
	}
	if invoked.Load() != 1 {
		t.Errorf("invoked = %d, want 1 (should not run when Deactivating)", invoked.Load())
	}
	if result2.Mutations != 0 {
		t.Errorf("Tick 2 Mutations = %d, want 0", result2.Mutations)
	}
}

func TestTickMixedMutations(t *testing.T) {
	// Test that a primitive can emit multiple mutation types in one tick
	primID := types.MustPrimitiveID("mixed_prim")
	actorID := types.MustActorID("actor_system0000000000000000001")
	targetID := types.MustActorID("actor_system0000000000000000002")

	p := &testPrimitive{
		id:      primID,
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			return []primitive.Mutation{
				primitive.AddEvent{
					Type:    event.EventTypeTrustUpdated,
					Source:  actorID,
					Content: event.TrustUpdatedContent{},
					Causes:  []types.EventID{events[0].ID()},
				},
				primitive.AddEdge{
					From:     actorID,
					To:       targetID,
					EdgeType: event.EdgeTypeTrust,
					Weight:   types.MustWeight(0.3),
					Scope:    types.None[types.DomainScope](),
				},
				primitive.UpdateState{PrimitiveID: primID, Key: "processed", Value: true},
				primitive.UpdateActivation{PrimitiveID: primID, Level: types.MustActivation(0.9)},
			}, nil
		},
	}

	e, _, bootstrap := newEngine(t, p)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		actorID, event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	result, err := e.Tick([]event.Event{ev})
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	// Wave 0 produces 4 mutations (AddEvent + AddEdge + UpdateState + UpdateActivation)
	// Wave 1 processes the new event from AddEvent, but the second invocation returns nil
	// because processFunc always returns the same mutations regardless of wave
	// Actually the processFunc will run again with the new event and emit more mutations
	// So we expect at least 4 mutations from wave 0
	if result.Mutations < 4 {
		t.Errorf("Mutations = %d, want >= 4", result.Mutations)
	}
}

func TestTickLayerBlockedByNonActivePrimitive(t *testing.T) {
	// Layer 1 should be blocked when a Layer 0 primitive exists but is not Active (e.g., Dormant)
	var layer1Invoked atomic.Int32

	p0 := &testPrimitive{
		id:      types.MustPrimitiveID("dormant_l0"),
		layer:   types.MustLayer(0),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
	}
	p1 := &testPrimitive{
		id:      types.MustPrimitiveID("active_l1"),
		layer:   types.MustLayer(1),
		cadence: types.MustCadence(1),
		subscriptions: []types.SubscriptionPattern{
			types.MustSubscriptionPattern("*"),
		},
		processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			layer1Invoked.Add(1)
			return nil, nil
		},
	}

	// Manual setup: register p0 as Dormant (don't activate), activate p1
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	registry := primitive.NewRegistry()
	signer := testSigner{}

	registry.Register(p0) // stays Dormant
	registry.Register(p1)
	registry.Activate(p1.ID())

	eventRegistry := event.DefaultRegistry()
	factory := event.NewEventFactory(eventRegistry)
	bf := event.NewBootstrapFactory(eventRegistry)
	bootstrap, _ := bf.Init(types.MustActorID("actor_system0000000000000000001"), signer)
	s.Append(bootstrap)

	e := tick.NewEngine(registry, s, as, factory, signer, tick.DefaultConfig(), nil)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	e.Tick([]event.Event{ev})

	// Layer 1 should NOT run because Layer 0 has a Dormant primitive
	if layer1Invoked.Load() != 0 {
		t.Errorf("layer1 invoked = %d, want 0 (blocked by dormant L0)", layer1Invoked.Load())
	}
}

func TestTickNoSubscriptionsSkipsEvents(t *testing.T) {
	var invoked atomic.Int32

	p := &testPrimitive{
		id:            types.MustPrimitiveID("no_subs"),
		layer:         types.MustLayer(0),
		cadence:       types.MustCadence(1),
		subscriptions: nil, // no subscriptions
		processFunc: func(tk types.Tick, events []event.Event, snap primitive.Snapshot) ([]primitive.Mutation, error) {
			invoked.Add(1)
			return nil, nil
		},
	}

	e, _, bootstrap := newEngine(t, p)

	ev := event.NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000099"),
		event.EventTypeTrustUpdated, types.Now(),
		types.MustActorID("actor_system0000000000000000001"),
		event.TrustUpdatedContent{},
		[]types.EventID{bootstrap.ID()},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(), types.ZeroHash(),
		types.MustSignature(make([]byte, 64)),
	)

	e.Tick([]event.Event{ev})

	// Primitive is still invoked (it's eligible), but gets no matched events
	if invoked.Load() != 1 {
		t.Errorf("invoked = %d, want 1", invoked.Load())
	}
}
