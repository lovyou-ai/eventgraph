use std::collections::BTreeMap;
use serde_json::Value;
use eventgraph::event::*;
use eventgraph::types::*;

#[test]
fn canonical_form_sorted_keys_no_whitespace() {
    let mut content = BTreeMap::new();
    content.insert("b".to_string(), Value::Number(1.into()));
    content.insert("a".to_string(), Value::Number(2.into()));
    let json = canonical_content_json(&content);
    assert!(!json.contains(' '));
    assert!(json.starts_with("{\"a\""));
}

#[test]
fn canonical_form_pipe_separated() {
    let canon = canonical_form(
        1, &"0".repeat(64),
        &["c2", "c1"],
        "eid", "trust.updated", "actor_alice", "conv_1",
        123456789, "{\"key\":\"val\"}",
    );
    let parts: Vec<&str> = canon.split('|').collect();
    assert_eq!(parts[0], "1");
    assert_eq!(parts[1], &"0".repeat(64));
    assert_eq!(parts[2], "c1,c2"); // sorted
    assert_eq!(parts[3], "eid");
    assert_eq!(parts[4], "trust.updated");
}

#[test]
fn canonical_form_empty_causes() {
    let canon = canonical_form(1, "", &[], "eid", "system.bootstrapped", "s", "c", 0, "{}");
    let parts: Vec<&str> = canon.split('|').collect();
    assert_eq!(parts[2], "");
}

#[test]
fn compute_hash_deterministic() {
    let h1 = compute_hash("hello");
    let h2 = compute_hash("hello");
    assert_eq!(h1.value(), h2.value());
}

#[test]
fn compute_hash_different_input() {
    let h1 = compute_hash("hello");
    let h2 = compute_hash("world");
    assert_ne!(h1.value(), h2.value());
}

#[test]
fn compute_hash_returns_64_hex() {
    let h = compute_hash("test");
    assert_eq!(h.value().len(), 64);
}

#[test]
fn new_event_id_is_valid_v7() {
    let eid = new_event_id();
    assert_eq!(eid.value().len(), 36);
    assert_eq!(eid.value().chars().nth(14), Some('7'));
}

#[test]
fn create_bootstrap_valid() {
    let signer = NoopSigner;
    let source = ActorId::new("actor_alice").unwrap();
    let ev = create_bootstrap(source, &signer, 1);

    assert_eq!(ev.version, 1);
    assert_eq!(ev.event_type.value(), "system.bootstrapped");
    assert_eq!(ev.source.value(), "actor_alice");
    assert!(ev.prev_hash.is_zero());
    assert_eq!(ev.causes.len(), 1);
}

#[test]
fn create_event_valid() {
    let signer = NoopSigner;
    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &signer, 1);

    let mut content = BTreeMap::new();
    content.insert("score".to_string(), Value::Number(serde_json::Number::from_f64(0.8).unwrap()));

    let ev = create_event(
        EventType::new("trust.updated").unwrap(),
        ActorId::new("alice").unwrap(),
        content,
        vec![boot.id.clone()],
        ConversationId::new("conv_1").unwrap(),
        boot.hash.clone(),
        &signer,
        1,
    );

    assert_eq!(ev.event_type.value(), "trust.updated");
    assert_eq!(ev.prev_hash.value(), boot.hash.value());
    assert_eq!(ev.causes.len(), 1);
}

#[test]
fn content_is_defensive_copy() {
    let signer = NoopSigner;
    let ev = create_bootstrap(ActorId::new("alice").unwrap(), &signer, 1);
    let c1 = ev.content();
    let c2 = ev.content();
    // Both are equal but separate allocations
    assert_eq!(c1, c2);
}

// ── Conformance tests ──────────────────────────────────────────────────

#[test]
fn conformance_bootstrap_hash() {
    let mut content = BTreeMap::new();
    content.insert("ActorID".to_string(), Value::String("actor_00000000000000000000000000000001".to_string()));
    content.insert("ChainGenesis".to_string(), Value::String("0".repeat(64)));
    content.insert("Timestamp".to_string(), Value::String("2023-11-14T22:13:20Z".to_string()));
    let content_json = canonical_content_json(&content);

    let canon = canonical_form(
        1, "", &[],
        "019462a0-0000-7000-8000-000000000001",
        "system.bootstrapped",
        "actor_00000000000000000000000000000001",
        "conv_00000000000000000000000000000001",
        1700000000000000000, &content_json,
    );

    assert!(canon.starts_with("1|||"));
    let hash = compute_hash(&canon);
    assert_eq!(hash.value(), "f7cae7ae11c1232a932c64f2302432c0e304dffce80f3935e688980dfbafeb75");
}

#[test]
fn conformance_trust_updated_hash() {
    let mut content = BTreeMap::new();
    content.insert("Actor".to_string(), Value::String("actor_00000000000000000000000000000002".to_string()));
    content.insert("Cause".to_string(), Value::String("019462a0-0000-7000-8000-000000000001".to_string()));
    content.insert("Current".to_string(), Value::Number(serde_json::Number::from_f64(0.85).unwrap()));
    content.insert("Domain".to_string(), Value::String("code_review".to_string()));
    content.insert("Previous".to_string(), Value::Number(serde_json::Number::from_f64(0.8).unwrap()));
    let content_json = canonical_content_json(&content);

    let canon = canonical_form(
        1, "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
        &["019462a0-0000-7000-8000-000000000001"],
        "019462a0-0000-7000-8000-000000000002",
        "trust.updated",
        "actor_00000000000000000000000000000001",
        "conv_00000000000000000000000000000001",
        1700000001000000000, &content_json,
    );

    let hash = compute_hash(&canon);
    assert_eq!(hash.value(), "b2fbcd2684868f0b0d07d2f5136b52f14b8e749da7b4b7bae2a22f67147152b7");
}

#[test]
fn conformance_edge_created_key_ordering_hash() {
    let mut content = BTreeMap::new();
    content.insert("Weight".to_string(), Value::Number(serde_json::Number::from_f64(0.5).unwrap()));
    content.insert("From".to_string(), Value::String("actor_00000000000000000000000000000001".to_string()));
    content.insert("To".to_string(), Value::String("actor_00000000000000000000000000000002".to_string()));
    content.insert("EdgeType".to_string(), Value::String("Trust".to_string()));
    content.insert("Direction".to_string(), Value::String("Centripetal".to_string()));
    let content_json = canonical_content_json(&content);

    let canon = canonical_form(
        1, "b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3",
        &["019462a0-0000-7000-8000-000000000001"],
        "019462a0-0000-7000-8000-000000000003",
        "edge.created",
        "actor_00000000000000000000000000000001",
        "conv_00000000000000000000000000000001",
        1700000002000000000, &content_json,
    );

    let hash = compute_hash(&canon);
    assert_eq!(hash.value(), "4e5c6710ca9325676663b4a66d2e82114fcd8fb49dbe5705795051e0b0be374c");
}
