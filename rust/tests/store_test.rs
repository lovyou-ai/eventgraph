use eventgraph::event::*;
use eventgraph::store::*;
use eventgraph::types::*;

fn bootstrap() -> Event {
    create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1)
}

#[test]
fn append_and_get() {
    let mut store = InMemoryStore::new();
    let ev = bootstrap();
    let id = ev.id.clone();
    store.append(ev).unwrap();
    assert_eq!(store.get(&id).unwrap().id, id);
}

#[test]
fn head_returns_latest() {
    let mut store = InMemoryStore::new();
    assert!(store.head().is_none());
    let ev = bootstrap();
    store.append(ev).unwrap();
    assert!(store.head().is_some());
}

#[test]
fn count_increments() {
    let mut store = InMemoryStore::new();
    assert_eq!(store.count(), 0);
    store.append(bootstrap()).unwrap();
    assert_eq!(store.count(), 1);
}

#[test]
fn chain_integrity_enforced() {
    let mut store = InMemoryStore::new();
    let boot = bootstrap();
    store.append(boot).unwrap();

    // Create a second bootstrap (wrong prev_hash)
    let bad = bootstrap();
    assert!(store.append(bad).is_err());
}

#[test]
fn verify_chain_valid() {
    let mut store = InMemoryStore::new();
    let boot = bootstrap();
    store.append(boot).unwrap();
    let v = store.verify_chain();
    assert!(v.valid);
    assert_eq!(v.length, 1);
}

#[test]
fn get_nonexistent_fails() {
    let store = InMemoryStore::new();
    let id = EventId::new("019462a0-0000-7000-8000-000000000001").unwrap();
    assert!(store.get(&id).is_err());
}

#[test]
fn chained_append() {
    let signer = NoopSigner;
    let mut store = InMemoryStore::new();
    let boot = bootstrap();
    let prev_hash = boot.hash.clone();
    let boot_id = boot.id.clone();
    store.append(boot).unwrap();

    let mut content = std::collections::BTreeMap::new();
    content.insert("score".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(0.5).unwrap()));

    let ev = create_event(
        EventType::new("trust.updated").unwrap(),
        ActorId::new("alice").unwrap(),
        content,
        vec![boot_id],
        ConversationId::new("conv_1").unwrap(),
        prev_hash,
        &signer,
        1,
    );
    store.append(ev).unwrap();
    assert_eq!(store.count(), 2);
    let v = store.verify_chain();
    assert!(v.valid);
    assert_eq!(v.length, 2);
}

#[test]
fn recent_returns_reverse_order() {
    let mut store = InMemoryStore::new();
    store.append(bootstrap()).unwrap();
    let recent = store.recent(10);
    assert_eq!(recent.len(), 1);
}
