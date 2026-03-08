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
    // Wave 1 with no events, counter still processes (wave > 0 check), mutations = 1
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
    // tick 1: last=0, 1-0=1 < 3 → skip (cadence not met)
    assert_eq!(r1.mutations, 0);

    let _r2 = engine.tick(None); // tick 2: 2-0=2 < 3 → skip
    assert_eq!(_r2.mutations, 0);

    let r3 = engine.tick(None); // tick 3: 3-0=3 >= 3 → runs
    assert!(r3.mutations > 0);
}
