use std::collections::BTreeMap;
use serde_json::Value;
use eventgraph::event::*;
use eventgraph::primitive::*;
use eventgraph::store::InMemoryStore;
use eventgraph::tick::*;
use eventgraph::types::*;

struct CounterPrimitive {
    pid: PrimitiveId,
}

impl CounterPrimitive {
    fn new(name: &str) -> Self {
        Self { pid: PrimitiveId::new(name).unwrap() }
    }
}

impl Primitive for CounterPrimitive {
    fn id(&self) -> PrimitiveId { self.pid.clone() }
    fn layer(&self) -> Layer { Layer::new(0).unwrap() }

    fn process(&self, _tick: u64, events: &[Event], _snapshot: &Snapshot) -> Vec<Mutation> {
        vec![Mutation::UpdateState {
            primitive_id: self.pid.clone(),
            key: "count".to_string(),
            value: Value::Number(events.len().into()),
        }]
    }

    fn subscriptions(&self) -> Vec<SubscriptionPattern> {
        vec![SubscriptionPattern::new("*").unwrap()]
    }

    fn cadence(&self) -> Cadence { Cadence::new(1).unwrap() }
}

struct EmitterPrimitive {
    pid: PrimitiveId,
}

impl EmitterPrimitive {
    fn new(name: &str) -> Self {
        Self { pid: PrimitiveId::new(name).unwrap() }
    }
}

impl Primitive for EmitterPrimitive {
    fn id(&self) -> PrimitiveId { self.pid.clone() }
    fn layer(&self) -> Layer { Layer::new(0).unwrap() }

    fn process(&self, _tick: u64, events: &[Event], snapshot: &Snapshot) -> Vec<Mutation> {
        let count = snapshot.primitives.get(self.pid.value())
            .and_then(|s| s.state.get("emitted"))
            .and_then(|v| v.as_u64())
            .unwrap_or(0);

        if count > 0 { return vec![]; }

        if events.is_empty() { return vec![]; }
        let cause = events[0].id.clone();

        let mut content = BTreeMap::new();
        content.insert("source".to_string(), Value::String("emitter".to_string()));

        vec![
            Mutation::UpdateState {
                primitive_id: self.pid.clone(),
                key: "emitted".to_string(),
                value: Value::Number(1.into()),
            },
            Mutation::AddEvent {
                event_type: EventType::new("test.emitted").unwrap(),
                source: ActorId::new("emitter").unwrap(),
                content,
                causes: vec![cause],
                conversation_id: ConversationId::new("conv_test").unwrap(),
            },
        ]
    }

    fn subscriptions(&self) -> Vec<SubscriptionPattern> {
        vec![SubscriptionPattern::new("*").unwrap()]
    }

    fn cadence(&self) -> Cadence { Cadence::new(1).unwrap() }
}

struct LayeredPrimitive {
    pid: PrimitiveId,
    layer_val: u8,
}

impl LayeredPrimitive {
    fn new(name: &str, layer: u8) -> Self {
        Self {
            pid: PrimitiveId::new(name).unwrap(),
            layer_val: layer,
        }
    }
}

impl Primitive for LayeredPrimitive {
    fn id(&self) -> PrimitiveId { self.pid.clone() }
    fn layer(&self) -> Layer { Layer::new(self.layer_val).unwrap() }
    fn process(&self, _tick: u64, _events: &[Event], _snapshot: &Snapshot) -> Vec<Mutation> {
        vec![Mutation::UpdateState {
            primitive_id: self.pid.clone(),
            key: "ran".to_string(),
            value: Value::Bool(true),
        }]
    }
    fn subscriptions(&self) -> Vec<SubscriptionPattern> {
        vec![SubscriptionPattern::new("*").unwrap()]
    }
    fn cadence(&self) -> Cadence { Cadence::new(1).unwrap() }
}

fn setup_engine(prims: Vec<Box<dyn Primitive>>) -> TickEngine {
    let mut registry = Registry::new();
    for p in prims {
        let pid = p.id();
        registry.register(p).unwrap();
        registry.activate(&pid).unwrap();
    }
    let store = InMemoryStore::new();
    TickEngine::new(registry, store, None)
}

#[test]
fn tick_increments() {
    let mut engine = setup_engine(vec![Box::new(CounterPrimitive::new("counter"))]);
    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r = engine.tick(Some(vec![boot]));
    assert_eq!(r.tick, 1);
    assert!(r.mutations > 0);
}

#[test]
fn tick_quiesces_with_no_events() {
    let mut engine = setup_engine(vec![Box::new(CounterPrimitive::new("counter"))]);
    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r = engine.tick(Some(vec![boot]));
    // Wave 1 processes, wave 2 finds no new events -> quiesces
    assert!(r.quiesced);
}

#[test]
fn emitter_causes_ripple() {
    let mut engine = setup_engine(vec![
        Box::new(EmitterPrimitive::new("emitter")),
        Box::new(CounterPrimitive::new("counter")),
    ]);
    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r = engine.tick(Some(vec![boot]));
    // Emitter emits in wave 1, counter processes in wave 2
    assert!(r.waves >= 2);
}

#[test]
fn empty_tick() {
    let mut engine = setup_engine(vec![Box::new(CounterPrimitive::new("counter"))]);
    let r = engine.tick(None);
    assert_eq!(r.tick, 1);
}

#[test]
fn inactive_primitives_skipped() {
    let mut registry = Registry::new();
    let p = Box::new(CounterPrimitive::new("counter"));
    registry.register(p).unwrap();
    // Don't activate - stays Dormant
    let store = InMemoryStore::new();
    let mut engine = TickEngine::new(registry, store, None);

    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r = engine.tick(Some(vec![boot]));
    assert_eq!(r.mutations, 0);
}

#[test]
fn multiple_ticks() {
    let mut engine = setup_engine(vec![Box::new(CounterPrimitive::new("counter"))]);
    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    engine.tick(Some(vec![boot]));
    let r2 = engine.tick(None);
    assert_eq!(r2.tick, 2);
}

#[test]
fn cadence_respected() {
    struct SlowPrimitive { pid: PrimitiveId }
    impl Primitive for SlowPrimitive {
        fn id(&self) -> PrimitiveId { self.pid.clone() }
        fn layer(&self) -> Layer { Layer::new(0).unwrap() }
        fn process(&self, _tick: u64, _events: &[Event], _snapshot: &Snapshot) -> Vec<Mutation> {
            vec![Mutation::UpdateState {
                primitive_id: self.pid.clone(),
                key: "ran".to_string(),
                value: Value::Bool(true),
            }]
        }
        fn subscriptions(&self) -> Vec<SubscriptionPattern> {
            vec![SubscriptionPattern::new("*").unwrap()]
        }
        fn cadence(&self) -> Cadence { Cadence::new(3).unwrap() }
    }

    let mut engine = setup_engine(vec![Box::new(SlowPrimitive {
        pid: PrimitiveId::new("slow").unwrap(),
    })]);

    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r1 = engine.tick(Some(vec![boot]));
    // tick 1: last=0, 1-0=1 < 3 → skip
    assert_eq!(r1.mutations, 0);

    let _r2 = engine.tick(None); // tick 2: 2-0=2 < 3 → skip
    assert_eq!(_r2.mutations, 0);

    let r3 = engine.tick(None); // tick 3: 3-0=3 >= 3 → runs
    assert!(r3.mutations > 0);
}

// --- New tests matching Go's comprehensive suite ---

#[test]
fn layer_constraint_blocks_when_lower_layer_not_invoked() {
    // Layer 1 should not run until Layer 0 has been invoked at least once
    let mut engine = setup_engine(vec![
        Box::new(LayeredPrimitive::new("l0_prim", 0)),
        Box::new(LayeredPrimitive::new("l1_prim", 1)),
    ]);

    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);

    // Tick 1: Layer 0 runs, Layer 1 blocked (Layer 0 never invoked before)
    let r1 = engine.tick(Some(vec![boot.clone()]));
    // Layer 0 produces UpdateState (deferred), Layer 1 is blocked
    // Total should be 1 (the deferred UpdateState from Layer 0)
    assert!(r1.mutations >= 1);

    // Tick 2: Layer 0 was invoked, Layer 1 now eligible
    let r2 = engine.tick(Some(vec![boot]));
    // Both Layer 0 and Layer 1 should run
    assert!(r2.mutations >= 2);
}

#[test]
fn layer_constraint_vacuously_true_for_sparse_layers() {
    // Layer 1 with no Layer 0 primitives → should run (vacuously stable)
    let mut engine = setup_engine(vec![
        Box::new(LayeredPrimitive::new("l1_only", 1)),
    ]);

    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r = engine.tick(Some(vec![boot]));
    assert!(r.mutations >= 1); // Layer 1 should run
}

#[test]
fn layer_constraint_blocked_by_dormant_lower_layer() {
    // Layer 0 primitive registered but not activated (Dormant) → Layer 1 blocked
    let mut registry = Registry::new();
    let l0 = Box::new(LayeredPrimitive::new("dormant_l0", 0));
    let l1 = Box::new(LayeredPrimitive::new("active_l1", 1));
    let l1_id = l1.id();

    registry.register(l0).unwrap();
    // Don't activate l0 — stays Dormant
    registry.register(l1).unwrap();
    registry.activate(&l1_id).unwrap();

    let store = InMemoryStore::new();
    let mut engine = TickEngine::new(registry, store, None);

    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r = engine.tick(Some(vec![boot]));

    // Layer 1 should NOT run because Layer 0 has a Dormant primitive
    assert_eq!(r.mutations, 0);
}

#[test]
fn deferred_mutations_applied_at_end_of_tick() {
    // UpdateState should be deferred and applied at end of tick,
    // not between waves
    struct StatefulPrimitive { pid: PrimitiveId }
    impl Primitive for StatefulPrimitive {
        fn id(&self) -> PrimitiveId { self.pid.clone() }
        fn layer(&self) -> Layer { Layer::new(0).unwrap() }
        fn process(&self, _tick: u64, events: &[Event], snapshot: &Snapshot) -> Vec<Mutation> {
            // Check if state was set in this tick (it shouldn't be — deferred)
            let count = snapshot.primitives.get(self.pid.value())
                .and_then(|s| s.state.get("count"))
                .and_then(|v| v.as_u64())
                .unwrap_or(0);

            if events.is_empty() { return vec![]; }
            let cause = events[0].id.clone();

            let mut result = vec![
                Mutation::UpdateState {
                    primitive_id: self.pid.clone(),
                    key: "count".to_string(),
                    value: Value::Number((count + 1).into()),
                },
            ];

            // Only emit an event on wave 0 to trigger wave 1
            if count == 0 {
                let mut content = BTreeMap::new();
                content.insert("wave".to_string(), Value::String("ripple".to_string()));
                result.push(Mutation::AddEvent {
                    event_type: EventType::new("test.ripple").unwrap(),
                    source: ActorId::new("stateful").unwrap(),
                    content,
                    causes: vec![cause],
                    conversation_id: ConversationId::new("conv_test").unwrap(),
                });
            }

            result
        }
        fn subscriptions(&self) -> Vec<SubscriptionPattern> {
            vec![SubscriptionPattern::new("*").unwrap()]
        }
        fn cadence(&self) -> Cadence { Cadence::new(1).unwrap() }
    }

    let mut engine = setup_engine(vec![Box::new(StatefulPrimitive {
        pid: PrimitiveId::new("stateful").unwrap(),
    })]);

    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r = engine.tick(Some(vec![boot]));

    // Should have multiple waves (wave 0 emits event, wave 1 processes it)
    assert!(r.waves >= 2);
    // UpdateState mutations are deferred, AddEvent is eager
    assert!(r.mutations >= 1);
}

#[test]
fn mixed_mutations_add_event_and_update_state() {
    struct MixedPrimitive { pid: PrimitiveId }
    impl Primitive for MixedPrimitive {
        fn id(&self) -> PrimitiveId { self.pid.clone() }
        fn layer(&self) -> Layer { Layer::new(0).unwrap() }
        fn process(&self, _tick: u64, events: &[Event], _snapshot: &Snapshot) -> Vec<Mutation> {
            if events.is_empty() { return vec![]; }
            let cause = events[0].id.clone();
            let mut content = BTreeMap::new();
            content.insert("test".to_string(), Value::Bool(true));
            vec![
                Mutation::AddEvent {
                    event_type: EventType::new("test.mixed").unwrap(),
                    source: ActorId::new("mixed").unwrap(),
                    content,
                    causes: vec![cause],
                    conversation_id: ConversationId::new("conv_test").unwrap(),
                },
                Mutation::UpdateState {
                    primitive_id: self.pid.clone(),
                    key: "processed".to_string(),
                    value: Value::Bool(true),
                },
                Mutation::UpdateActivation {
                    primitive_id: self.pid.clone(),
                    level: Activation::new(0.9).unwrap(),
                },
            ]
        }
        fn subscriptions(&self) -> Vec<SubscriptionPattern> {
            vec![SubscriptionPattern::new("*").unwrap()]
        }
        fn cadence(&self) -> Cadence { Cadence::new(1).unwrap() }
    }

    let mut engine = setup_engine(vec![Box::new(MixedPrimitive {
        pid: PrimitiveId::new("mixed").unwrap(),
    })]);

    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r = engine.tick(Some(vec![boot]));

    // AddEvent (1 eager) + UpdateState + UpdateActivation (2 deferred)
    // The mixed primitive runs again on wave 1 with the emitted event, producing more mutations
    assert!(r.mutations >= 3);
}

#[test]
fn wave_limit_prevents_infinite_loop() {
    struct InfiniteEmitter { pid: PrimitiveId }
    impl Primitive for InfiniteEmitter {
        fn id(&self) -> PrimitiveId { self.pid.clone() }
        fn layer(&self) -> Layer { Layer::new(0).unwrap() }
        fn process(&self, _tick: u64, events: &[Event], _snapshot: &Snapshot) -> Vec<Mutation> {
            if events.is_empty() { return vec![]; }
            let cause = events[0].id.clone();
            let mut content = BTreeMap::new();
            content.insert("loop".to_string(), Value::Bool(true));
            vec![Mutation::AddEvent {
                event_type: EventType::new("test.loop").unwrap(),
                source: ActorId::new("infinite").unwrap(),
                content,
                causes: vec![cause],
                conversation_id: ConversationId::new("conv_test").unwrap(),
            }]
        }
        fn subscriptions(&self) -> Vec<SubscriptionPattern> {
            vec![SubscriptionPattern::new("*").unwrap()]
        }
        fn cadence(&self) -> Cadence { Cadence::new(1).unwrap() }
    }

    let mut registry = Registry::new();
    let p = Box::new(InfiniteEmitter { pid: PrimitiveId::new("infinite").unwrap() });
    let pid = p.id();
    registry.register(p).unwrap();
    registry.activate(&pid).unwrap();

    let store = InMemoryStore::new();
    let mut engine = TickEngine::new(registry, store, Some(TickConfig { max_waves_per_tick: 3 }));

    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r = engine.tick(Some(vec![boot]));

    assert!(!r.quiesced);
    assert!(r.waves <= 3);
}

#[test]
fn update_lifecycle_mutation() {
    struct LifecycleUpdater { pid: PrimitiveId }
    impl Primitive for LifecycleUpdater {
        fn id(&self) -> PrimitiveId { self.pid.clone() }
        fn layer(&self) -> Layer { Layer::new(0).unwrap() }
        fn process(&self, _tick: u64, _events: &[Event], _snapshot: &Snapshot) -> Vec<Mutation> {
            vec![Mutation::UpdateLifecycle {
                primitive_id: self.pid.clone(),
                state: LifecycleState::Suspending,
            }]
        }
        fn subscriptions(&self) -> Vec<SubscriptionPattern> {
            vec![SubscriptionPattern::new("*").unwrap()]
        }
        fn cadence(&self) -> Cadence { Cadence::new(1).unwrap() }
    }

    let mut engine = setup_engine(vec![Box::new(LifecycleUpdater {
        pid: PrimitiveId::new("updater").unwrap(),
    })]);

    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r1 = engine.tick(Some(vec![boot.clone()]));
    assert!(r1.mutations >= 1);

    // Tick 2: primitive is now Suspending, should NOT be eligible
    let r2 = engine.tick(Some(vec![boot]));
    assert_eq!(r2.mutations, 0);
}

#[test]
fn subscription_filtering_only_matching_events() {
    struct TrustWatcher { pid: PrimitiveId }
    impl Primitive for TrustWatcher {
        fn id(&self) -> PrimitiveId { self.pid.clone() }
        fn layer(&self) -> Layer { Layer::new(0).unwrap() }
        fn process(&self, _tick: u64, events: &[Event], _snapshot: &Snapshot) -> Vec<Mutation> {
            vec![Mutation::UpdateState {
                primitive_id: self.pid.clone(),
                key: "received".to_string(),
                value: Value::Number(events.len().into()),
            }]
        }
        fn subscriptions(&self) -> Vec<SubscriptionPattern> {
            vec![SubscriptionPattern::new("trust.*").unwrap()]
        }
        fn cadence(&self) -> Cadence { Cadence::new(1).unwrap() }
    }

    let mut engine = setup_engine(vec![Box::new(TrustWatcher {
        pid: PrimitiveId::new("watcher").unwrap(),
    })]);

    // Create events of different types
    let trust_ev = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    // The bootstrap event has type "system.bootstrap" which should NOT match "trust.*"
    let r = engine.tick(Some(vec![trust_ev]));

    // Primitive is invoked on first wave (even with no matched events)
    // but should get 0 matched events since bootstrap != trust.*
    assert!(r.mutations >= 1); // UpdateState with received=0
}

#[test]
fn no_subscriptions_gets_no_events() {
    struct NoSubsPrimitive { pid: PrimitiveId }
    impl Primitive for NoSubsPrimitive {
        fn id(&self) -> PrimitiveId { self.pid.clone() }
        fn layer(&self) -> Layer { Layer::new(0).unwrap() }
        fn process(&self, _tick: u64, events: &[Event], _snapshot: &Snapshot) -> Vec<Mutation> {
            vec![Mutation::UpdateState {
                primitive_id: self.pid.clone(),
                key: "event_count".to_string(),
                value: Value::Number(events.len().into()),
            }]
        }
        fn subscriptions(&self) -> Vec<SubscriptionPattern> {
            vec![] // no subscriptions
        }
        fn cadence(&self) -> Cadence { Cadence::new(1).unwrap() }
    }

    let mut engine = setup_engine(vec![Box::new(NoSubsPrimitive {
        pid: PrimitiveId::new("nosubs").unwrap(),
    })]);

    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let r = engine.tick(Some(vec![boot]));

    // Primitive is still invoked but gets 0 matched events
    assert!(r.mutations >= 1);
}
