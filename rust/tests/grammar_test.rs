use eventgraph::event::*;
use eventgraph::grammar::Grammar;
use eventgraph::store::*;
use eventgraph::types::*;

fn setup() -> (InMemoryStore, Event) {
    let mut store = InMemoryStore::new();
    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let boot = store.append(boot).unwrap();
    (store, boot)
}

// ── emit ────────────────────────────────────────────────────────────────

#[test]
fn emit_creates_emitted_event() {
    let (mut store, boot) = setup();
    let mut g = Grammar::new(&mut store);

    let ev = g
        .emit(
            ActorId::new("alice").unwrap(),
            "hello world",
            ConversationId::new("conv_1").unwrap(),
            vec![boot.id.clone()],
            &NoopSigner,
        )
        .unwrap();

    assert_eq!(ev.event_type.value(), "grammar.emitted");
    assert_eq!(ev.source.value(), "alice");
    let content = ev.content();
    assert_eq!(content["Body"], "hello world");
}

#[test]
fn emit_empty_causes_fails() {
    let (mut store, _boot) = setup();
    let mut g = Grammar::new(&mut store);

    let result = g.emit(
        ActorId::new("alice").unwrap(),
        "hello",
        ConversationId::new("conv_1").unwrap(),
        vec![],
        &NoopSigner,
    );

    assert!(result.is_err());
}

// ── respond ─────────────────────────────────────────────────────────────

#[test]
fn respond_creates_responded_event() {
    let (mut store, boot) = setup();
    let mut g = Grammar::new(&mut store);

    let ev = g
        .respond(
            ActorId::new("bob").unwrap(),
            "I agree",
            boot.id.clone(),
            ConversationId::new("conv_1").unwrap(),
            &NoopSigner,
        )
        .unwrap();

    assert_eq!(ev.event_type.value(), "grammar.responded");
    let content = ev.content();
    assert_eq!(content["Body"], "I agree");
    assert_eq!(content["Parent"], boot.id.value());
}

// ── derive ──────────────────────────────────────────────────────────────

#[test]
fn derive_creates_derived_event() {
    let (mut store, boot) = setup();
    let mut g = Grammar::new(&mut store);

    let ev = g
        .derive(
            ActorId::new("alice").unwrap(),
            "derived insight",
            boot.id.clone(),
            ConversationId::new("conv_1").unwrap(),
            &NoopSigner,
        )
        .unwrap();

    assert_eq!(ev.event_type.value(), "grammar.derived");
    let content = ev.content();
    assert_eq!(content["Body"], "derived insight");
    assert_eq!(content["Source"], boot.id.value());
}

// ── extend ──────────────────────────────────────────────────────────────

#[test]
fn extend_creates_extended_event() {
    let (mut store, boot) = setup();
    let mut g = Grammar::new(&mut store);

    let ev = g
        .extend(
            ActorId::new("alice").unwrap(),
            "more detail",
            boot.id.clone(),
            ConversationId::new("conv_1").unwrap(),
            &NoopSigner,
        )
        .unwrap();

    assert_eq!(ev.event_type.value(), "grammar.extended");
    let content = ev.content();
    assert_eq!(content["Body"], "more detail");
    assert_eq!(content["Previous"], boot.id.value());
}

// ── retract ─────────────────────────────────────────────────────────────

#[test]
fn retract_creates_retracted_event() {
    let (mut store, boot) = setup();
    let mut g = Grammar::new(&mut store);

    // First emit something as alice (bootstrap is also by alice)
    let emitted = g
        .emit(
            ActorId::new("alice").unwrap(),
            "oops",
            ConversationId::new("conv_1").unwrap(),
            vec![boot.id.clone()],
            &NoopSigner,
        )
        .unwrap();

    let ev = g
        .retract(
            ActorId::new("alice").unwrap(),
            emitted.id.clone(),
            "mistake",
            ConversationId::new("conv_1").unwrap(),
            &NoopSigner,
        )
        .unwrap();

    assert_eq!(ev.event_type.value(), "grammar.retracted");
    let content = ev.content();
    assert_eq!(content["Target"], emitted.id.value());
    assert_eq!(content["Reason"], "mistake");
}

#[test]
fn retract_by_non_author_fails() {
    let (mut store, boot) = setup();
    let mut g = Grammar::new(&mut store);

    // boot was authored by alice; bob cannot retract it
    let result = g.retract(
        ActorId::new("bob").unwrap(),
        boot.id.clone(),
        "nope",
        ConversationId::new("conv_1").unwrap(),
        &NoopSigner,
    );

    assert!(result.is_err());
}

// ── annotate ────────────────────────────────────────────────────────────

#[test]
fn annotate_creates_annotated_event() {
    let (mut store, boot) = setup();
    let mut g = Grammar::new(&mut store);

    let ev = g
        .annotate(
            ActorId::new("alice").unwrap(),
            boot.id.clone(),
            "sentiment",
            "positive",
            ConversationId::new("conv_1").unwrap(),
            &NoopSigner,
        )
        .unwrap();

    assert_eq!(ev.event_type.value(), "grammar.annotated");
    let content = ev.content();
    assert_eq!(content["Target"], boot.id.value());
    assert_eq!(content["Key"], "sentiment");
    assert_eq!(content["Value"], "positive");
}

// ── merge ───────────────────────────────────────────────────────────────

#[test]
fn merge_creates_merged_event() {
    let (mut store, boot) = setup();
    let mut g = Grammar::new(&mut store);

    let ev1 = g
        .emit(
            ActorId::new("alice").unwrap(),
            "point one",
            ConversationId::new("conv_1").unwrap(),
            vec![boot.id.clone()],
            &NoopSigner,
        )
        .unwrap();

    let ev2 = g
        .emit(
            ActorId::new("alice").unwrap(),
            "point two",
            ConversationId::new("conv_1").unwrap(),
            vec![ev1.id.clone()],
            &NoopSigner,
        )
        .unwrap();

    let merged = g
        .merge(
            ActorId::new("alice").unwrap(),
            "synthesis",
            vec![ev1.id.clone(), ev2.id.clone()],
            ConversationId::new("conv_1").unwrap(),
            &NoopSigner,
        )
        .unwrap();

    assert_eq!(merged.event_type.value(), "grammar.merged");
    let content = merged.content();
    assert_eq!(content["Body"], "synthesis");
    let sources = content["Sources"].as_array().unwrap();
    assert_eq!(sources.len(), 2);
    assert_eq!(sources[0], ev1.id.value());
    assert_eq!(sources[1], ev2.id.value());
}

#[test]
fn merge_with_fewer_than_two_sources_fails() {
    let (mut store, boot) = setup();
    let mut g = Grammar::new(&mut store);

    let result = g.merge(
        ActorId::new("alice").unwrap(),
        "lone merge",
        vec![boot.id.clone()],
        ConversationId::new("conv_1").unwrap(),
        &NoopSigner,
    );

    assert!(result.is_err());
}

// ── chain integrity ─────────────────────────────────────────────────────

#[test]
fn grammar_operations_maintain_hash_chain() {
    let (mut store, boot) = setup();
    let mut g = Grammar::new(&mut store);

    let e1 = g
        .emit(
            ActorId::new("alice").unwrap(),
            "first",
            ConversationId::new("conv_1").unwrap(),
            vec![boot.id.clone()],
            &NoopSigner,
        )
        .unwrap();

    let e2 = g
        .respond(
            ActorId::new("bob").unwrap(),
            "reply",
            e1.id.clone(),
            ConversationId::new("conv_1").unwrap(),
            &NoopSigner,
        )
        .unwrap();

    let _e3 = g
        .annotate(
            ActorId::new("alice").unwrap(),
            e2.id.clone(),
            "tag",
            "important",
            ConversationId::new("conv_1").unwrap(),
            &NoopSigner,
        )
        .unwrap();

    // Drop grammar to regain access to store
    drop(g);

    let v = store.verify_chain();
    assert!(v.valid);
    assert_eq!(v.length, 4); // boot + 3 grammar events
}
