use std::collections::{BTreeMap, HashMap};

use eventgraph::agent::*;
use eventgraph::event::{create_event, new_event_id, NoopSigner};
use eventgraph::primitive::{Mutation, Registry, Snapshot};
use eventgraph::types::*;

// ── Helper: create an event with a given type ───────────────────────────

fn make_event(event_type_str: &str) -> eventgraph::event::Event {
    let et = EventType::new(event_type_str).unwrap();
    let source = ActorId::new("test-agent").unwrap();
    let conv = ConversationId::new("test-conv").unwrap();
    let cause = new_event_id();
    create_event(
        et, source, BTreeMap::new(),
        vec![cause], conv,
        Hash::zero(), &NoopSigner, 1,
    )
}

fn empty_snapshot() -> Snapshot {
    Snapshot {
        tick: 1,
        primitives: HashMap::new(),
        pending_events: vec![],
        recent_events: vec![],
    }
}

// ══════════════════════════════════════════════════════════════════════════
// OperationalState FSM tests
// ══════════════════════════════════════════════════════════════════════════

#[test]
fn test_state_valid_transitions() {
    use OperationalState::*;

    // Idle -> Processing
    assert_eq!(Idle.transition_to(Processing).unwrap(), Processing);
    // Idle -> Suspended
    assert_eq!(Idle.transition_to(Suspended).unwrap(), Suspended);
    // Idle -> Retiring
    assert_eq!(Idle.transition_to(Retiring).unwrap(), Retiring);

    // Processing -> Idle
    assert_eq!(Processing.transition_to(Idle).unwrap(), Idle);
    // Processing -> Waiting
    assert_eq!(Processing.transition_to(Waiting).unwrap(), Waiting);
    // Processing -> Escalating
    assert_eq!(Processing.transition_to(Escalating).unwrap(), Escalating);
    // Processing -> Refusing
    assert_eq!(Processing.transition_to(Refusing).unwrap(), Refusing);
    // Processing -> Retiring
    assert_eq!(Processing.transition_to(Retiring).unwrap(), Retiring);

    // Waiting -> Processing
    assert_eq!(Waiting.transition_to(Processing).unwrap(), Processing);
    // Waiting -> Idle
    assert_eq!(Waiting.transition_to(Idle).unwrap(), Idle);
    // Waiting -> Retiring
    assert_eq!(Waiting.transition_to(Retiring).unwrap(), Retiring);

    // Escalating -> Waiting
    assert_eq!(Escalating.transition_to(Waiting).unwrap(), Waiting);
    // Escalating -> Idle
    assert_eq!(Escalating.transition_to(Idle).unwrap(), Idle);

    // Refusing -> Idle
    assert_eq!(Refusing.transition_to(Idle).unwrap(), Idle);

    // Suspended -> Idle
    assert_eq!(Suspended.transition_to(Idle).unwrap(), Idle);
    // Suspended -> Retiring
    assert_eq!(Suspended.transition_to(Retiring).unwrap(), Retiring);

    // Retiring -> Retired
    assert_eq!(Retiring.transition_to(Retired).unwrap(), Retired);
}

#[test]
fn test_state_invalid_transitions() {
    use OperationalState::*;

    // Idle cannot go to Waiting directly
    assert!(Idle.transition_to(Waiting).is_err());
    // Idle cannot go to Retired directly
    assert!(Idle.transition_to(Retired).is_err());
    // Processing cannot go to Suspended
    assert!(Processing.transition_to(Suspended).is_err());
    // Retired is terminal — cannot go anywhere
    assert!(Retired.transition_to(Idle).is_err());
    assert!(Retired.transition_to(Processing).is_err());
    assert!(Retired.transition_to(Retiring).is_err());
    // Refusing cannot go to Processing
    assert!(Refusing.transition_to(Processing).is_err());
    // Escalating cannot go to Retired
    assert!(Escalating.transition_to(Retired).is_err());
}

#[test]
fn test_is_terminal() {
    use OperationalState::*;
    assert!(Retired.is_terminal());
    assert!(!Idle.is_terminal());
    assert!(!Processing.is_terminal());
    assert!(!Waiting.is_terminal());
    assert!(!Escalating.is_terminal());
    assert!(!Refusing.is_terminal());
    assert!(!Suspended.is_terminal());
    assert!(!Retiring.is_terminal());
}

#[test]
fn test_can_act() {
    use OperationalState::*;
    assert!(Processing.can_act());
    assert!(!Idle.can_act());
    assert!(!Waiting.can_act());
    assert!(!Escalating.can_act());
    assert!(!Refusing.can_act());
    assert!(!Suspended.can_act());
    assert!(!Retiring.can_act());
    assert!(!Retired.can_act());
}

#[test]
fn test_state_display() {
    use OperationalState::*;
    assert_eq!(Idle.to_string(), "Idle");
    assert_eq!(Processing.to_string(), "Processing");
    assert_eq!(Waiting.to_string(), "Waiting");
    assert_eq!(Escalating.to_string(), "Escalating");
    assert_eq!(Refusing.to_string(), "Refusing");
    assert_eq!(Suspended.to_string(), "Suspended");
    assert_eq!(Retiring.to_string(), "Retiring");
    assert_eq!(Retired.to_string(), "Retired");
}

#[test]
fn test_full_lifecycle() {
    use OperationalState::*;
    // Boot -> process -> wait -> resume -> retire
    let s = Idle;
    let s = s.transition_to(Processing).unwrap();
    let s = s.transition_to(Waiting).unwrap();
    let s = s.transition_to(Processing).unwrap();
    let s = s.transition_to(Idle).unwrap();
    let s = s.transition_to(Retiring).unwrap();
    let s = s.transition_to(Retired).unwrap();
    assert!(s.is_terminal());
    assert!(!s.can_act());
}

// ══════════════════════════════════════════════════════════════════════════
// Primitives tests
// ══════════════════════════════════════════════════════════════════════════

#[test]
fn test_all_primitives_count() {
    let prims = all_primitives();
    assert_eq!(prims.len(), 28, "Expected exactly 28 agent primitives");
}

#[test]
fn test_all_primitives_unique_ids() {
    let prims = all_primitives();
    let mut ids: Vec<String> = prims.iter().map(|p| p.id().value().to_string()).collect();
    let total = ids.len();
    ids.sort();
    ids.dedup();
    assert_eq!(ids.len(), total, "All agent primitive IDs must be unique");
}

#[test]
fn test_all_primitives_layer_1() {
    let prims = all_primitives();
    for p in &prims {
        assert_eq!(p.layer().value(), 1, "Agent primitive {} should be layer 1", p.id().value());
    }
}

#[test]
fn test_all_primitives_cadence_1() {
    let prims = all_primitives();
    for p in &prims {
        assert_eq!(p.cadence().value(), 1, "Agent primitive {} should have cadence 1", p.id().value());
    }
}

#[test]
fn test_all_primitives_have_subscriptions() {
    let prims = all_primitives();
    for p in &prims {
        let subs = p.subscriptions();
        assert!(!subs.is_empty(), "Agent primitive {} has no subscriptions", p.id().value());
    }
}

#[test]
fn test_all_primitives_are_agent_primitives() {
    let prims = all_primitives();
    for p in &prims {
        assert!(is_agent_primitive(&p.id()), "{} should be identified as agent primitive", p.id().value());
    }
}

#[test]
fn test_all_primitives_process_empty_events() {
    let prims = all_primitives();
    let snap = empty_snapshot();
    for p in &prims {
        let mutations = p.process(1, &[], &snap);
        // Every primitive should produce at least a lastTick mutation
        let has_last_tick = mutations.iter().any(|m| matches!(m,
            Mutation::UpdateState { key, .. } if key == "lastTick"
        ));
        assert!(has_last_tick, "Primitive {} should produce lastTick mutation", p.id().value());
    }
}

#[test]
fn test_identity_process_with_events() {
    let prims = all_primitives();
    let identity = prims.iter().find(|p| p.id().value() == "agent.Identity").unwrap();
    let snap = empty_snapshot();

    let events = vec![
        make_event(AGENT_IDENTITY_CREATED),
        make_event(AGENT_IDENTITY_CREATED),
        make_event(AGENT_IDENTITY_ROTATED),
    ];

    let mutations = identity.process(5, &events, &snap);
    // Should have identitiesCreated=2, keysRotated=1, lastTick=5
    let created = mutations.iter().find(|m| matches!(m,
        Mutation::UpdateState { key, .. } if key == "identitiesCreated"
    )).unwrap();
    if let Mutation::UpdateState { value, .. } = created {
        assert_eq!(value.as_u64().unwrap(), 2);
    }
}

#[test]
fn test_goal_process_with_events() {
    let prims = all_primitives();
    let goal = prims.iter().find(|p| p.id().value() == "agent.Goal").unwrap();
    let snap = empty_snapshot();

    let events = vec![
        make_event(AGENT_GOAL_SET),
        make_event(AGENT_GOAL_COMPLETED),
        make_event(AGENT_GOAL_SET),
        make_event(AGENT_GOAL_ABANDONED),
    ];

    let mutations = goal.process(10, &events, &snap);
    let set = mutations.iter().find(|m| matches!(m,
        Mutation::UpdateState { key, .. } if key == "goalsSet"
    )).unwrap();
    if let Mutation::UpdateState { value, .. } = set {
        assert_eq!(value.as_u64().unwrap(), 2);
    }
    let completed = mutations.iter().find(|m| matches!(m,
        Mutation::UpdateState { key, .. } if key == "goalsCompleted"
    )).unwrap();
    if let Mutation::UpdateState { value, .. } = completed {
        assert_eq!(value.as_u64().unwrap(), 1);
    }
}

#[test]
fn test_soul_process_imprint() {
    let prims = all_primitives();
    let soul = prims.iter().find(|p| p.id().value() == "agent.Soul").unwrap();
    let snap = empty_snapshot();

    // No imprint events — should not have imprinted key
    let mutations = soul.process(1, &[], &snap);
    let has_imprinted = mutations.iter().any(|m| matches!(m,
        Mutation::UpdateState { key, .. } if key == "imprinted"
    ));
    assert!(!has_imprinted, "Soul should not have imprinted key without imprint event");

    // With imprint event
    let events = vec![make_event(AGENT_SOUL_IMPRINTED)];
    let mutations = soul.process(2, &events, &snap);
    let has_imprinted = mutations.iter().any(|m| matches!(m,
        Mutation::UpdateState { key, value, .. } if key == "imprinted" && *value == serde_json::Value::Bool(true)
    ));
    assert!(has_imprinted, "Soul should have imprinted=true after imprint event");
}

#[test]
fn test_register_all() {
    let mut registry = Registry::new();
    register_all(&mut registry).expect("register_all should succeed");
    assert_eq!(registry.count(), 28, "Registry should contain 28 agent primitives");

    // All should be active
    let prims = all_primitives();
    for p in &prims {
        assert_eq!(
            registry.get_lifecycle(&p.id()),
            LifecycleState::Active,
            "Primitive {} should be active after register_all",
            p.id().value()
        );
    }
}

#[test]
fn test_register_all_no_duplicates() {
    let mut registry = Registry::new();
    register_all(&mut registry).expect("first register_all should succeed");
    // Second registration should fail due to duplicate IDs
    assert!(register_all(&mut registry).is_err(), "Duplicate registration should fail");
}

// ══════════════════════════════════════════════════════════════════════════
// Compositions tests
// ══════════════════════════════════════════════════════════════════════════

#[test]
fn test_all_compositions_count() {
    let comps = all_compositions();
    assert_eq!(comps.len(), 8, "Expected exactly 8 compositions");
}

#[test]
fn test_composition_names() {
    let comps = all_compositions();
    let names: Vec<&str> = comps.iter().map(|c| c.name).collect();
    assert_eq!(names, vec![
        "Boot", "Imprint", "Task", "Supervise",
        "Collaborate", "Crisis", "Retire", "Whistleblow",
    ]);
}

#[test]
fn test_boot_composition() {
    let b = boot();
    assert_eq!(b.name, "Boot");
    assert_eq!(b.primitives.len(), 5);
    assert!(b.primitives.contains(&"agent.Identity"));
    assert!(b.primitives.contains(&"agent.Soul"));
    assert!(b.primitives.contains(&"agent.Model"));
    assert!(b.primitives.contains(&"agent.Authority"));
    assert!(b.primitives.contains(&"agent.State"));
    assert_eq!(b.events.len(), 5);
    assert!(b.events.contains(&AGENT_IDENTITY_CREATED));
    assert!(b.events.contains(&AGENT_STATE_CHANGED));
}

#[test]
fn test_imprint_composition() {
    let i = imprint();
    assert_eq!(i.name, "Imprint");
    assert_eq!(i.primitives.len(), 8);
    // Includes all of Boot plus Observe, Learn, Goal
    assert!(i.primitives.contains(&"agent.Observe"));
    assert!(i.primitives.contains(&"agent.Learn"));
    assert!(i.primitives.contains(&"agent.Goal"));
    assert_eq!(i.events.len(), 8);
}

#[test]
fn test_task_composition() {
    let t = task();
    assert_eq!(t.name, "Task");
    assert_eq!(t.primitives.len(), 5);
    assert_eq!(t.events.len(), 5);
}

#[test]
fn test_supervise_composition() {
    let s = supervise();
    assert_eq!(s.name, "Supervise");
    assert_eq!(s.primitives.len(), 5);
    assert!(s.primitives.contains(&"agent.Delegate"));
    assert!(s.primitives.contains(&"agent.Expect"));
}

#[test]
fn test_collaborate_composition() {
    let c = collaborate();
    assert_eq!(c.name, "Collaborate");
    assert_eq!(c.primitives.len(), 5);
    assert!(c.primitives.contains(&"agent.Channel"));
    assert!(c.primitives.contains(&"agent.Consent"));
    assert!(c.primitives.contains(&"agent.Composition"));
}

#[test]
fn test_crisis_composition() {
    let c = crisis();
    assert_eq!(c.name, "Crisis");
    assert_eq!(c.primitives.len(), 5);
    assert!(c.primitives.contains(&"agent.Attenuation"));
    assert!(c.primitives.contains(&"agent.Escalate"));
}

#[test]
fn test_retire_composition() {
    let r = retire();
    assert_eq!(r.name, "Retire");
    assert_eq!(r.primitives.len(), 4);
    assert!(r.primitives.contains(&"agent.Introspect"));
    assert!(r.primitives.contains(&"agent.Lifespan"));
}

#[test]
fn test_whistleblow_composition() {
    let w = whistleblow();
    assert_eq!(w.name, "Whistleblow");
    assert_eq!(w.primitives.len(), 5);
    assert!(w.primitives.contains(&"agent.Refuse"));
    assert!(w.primitives.contains(&"agent.Escalate"));
    assert!(w.primitives.contains(&"agent.Communicate"));
}

#[test]
fn test_composition_primitives_are_valid() {
    let all_prim_ids: Vec<String> = all_primitives()
        .iter()
        .map(|p| p.id().value().to_string())
        .collect();

    for comp in all_compositions() {
        for prim_id in &comp.primitives {
            assert!(
                all_prim_ids.contains(&prim_id.to_string()),
                "Composition {} references unknown primitive {}",
                comp.name, prim_id
            );
        }
    }
}

#[test]
fn test_composition_events_are_valid() {
    let all_events = all_agent_event_types();
    for comp in all_compositions() {
        for event in &comp.events {
            assert!(
                all_events.contains(event),
                "Composition {} references unknown event type {}",
                comp.name, event
            );
        }
    }
}

// ══════════════════════════════════════════════════════════════════════════
// Event type constants tests
// ══════════════════════════════════════════════════════════════════════════

#[test]
fn test_all_agent_event_types_count() {
    let types = all_agent_event_types();
    assert_eq!(types.len(), 45, "Expected 45 agent event types");
}

#[test]
fn test_all_agent_event_types_valid() {
    for et_str in all_agent_event_types() {
        assert!(
            EventType::new(et_str).is_ok(),
            "Event type '{}' should be a valid EventType",
            et_str
        );
    }
}

#[test]
fn test_all_agent_event_types_start_with_agent() {
    for et_str in all_agent_event_types() {
        assert!(
            et_str.starts_with("agent."),
            "Agent event type '{}' should start with 'agent.'",
            et_str
        );
    }
}
