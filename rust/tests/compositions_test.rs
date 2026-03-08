use eventgraph::event::*;
use eventgraph::store::*;
use eventgraph::types::*;

fn setup() -> (InMemoryStore, Event) {
    let mut store = InMemoryStore::new();
    let boot = create_bootstrap(ActorId::new("alice").unwrap(), &NoopSigner, 1);
    let boot = store.append(boot).unwrap();
    (store, boot)
}

fn actor(name: &str) -> ActorId {
    ActorId::new(name).unwrap()
}

fn conv() -> ConversationId {
    ConversationId::new("conv_1").unwrap()
}

fn scope(s: &str) -> DomainScope {
    DomainScope::new(s).unwrap()
}

// ── Grammar edge operations ──────────────────────────────────────────

#[test]
fn acknowledge_creates_edge_event() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let ev = g
        .acknowledge(actor("alice"), boot.id.clone(), actor("bob"), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(ev.event_type.value(), "edge.created");
    let content = ev.content();
    assert_eq!(content["EdgeType"], "Acknowledgement");
    assert_eq!(content["Direction"], "Centripetal");
    assert_eq!(content["From"], "alice");
    assert_eq!(content["To"], "bob");
}

#[test]
fn propagate_creates_edge_event() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let ev = g
        .propagate(actor("alice"), boot.id.clone(), actor("bob"), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(ev.event_type.value(), "edge.created");
    let content = ev.content();
    assert_eq!(content["EdgeType"], "Reference");
    assert_eq!(content["Direction"], "Centrifugal");
}

#[test]
fn endorse_creates_weighted_edge() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let w = Weight::new(0.8).unwrap();
    let s = scope("code.review");
    let ev = g
        .endorse(actor("alice"), boot.id.clone(), actor("bob"), w, Some(&s), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(ev.event_type.value(), "edge.created");
    let content = ev.content();
    assert_eq!(content["EdgeType"], "Endorsement");
    assert_eq!(content["Scope"], "code.review");
    // Weight should be 0.8
    assert_eq!(content["Weight"].as_f64().unwrap(), 0.8);
}

#[test]
fn subscribe_creates_subscription_edge() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let ev = g
        .subscribe(actor("alice"), actor("bob"), None, boot.id.clone(), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(ev.event_type.value(), "edge.created");
    let content = ev.content();
    assert_eq!(content["EdgeType"], "Subscription");
}

#[test]
fn channel_creates_channel_edge() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let ev = g
        .channel(actor("alice"), actor("bob"), None, boot.id.clone(), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(ev.event_type.value(), "edge.created");
    let content = ev.content();
    assert_eq!(content["EdgeType"], "Channel");
}

#[test]
fn delegate_creates_delegation_edge() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let s = scope("task.management");
    let w = Weight::new(0.5).unwrap();
    let ev = g
        .delegate(actor("alice"), actor("bob"), &s, w, boot.id.clone(), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(ev.event_type.value(), "edge.created");
    let content = ev.content();
    assert_eq!(content["EdgeType"], "Delegation");
    assert_eq!(content["Direction"], "Centrifugal");
    assert_eq!(content["Scope"], "task.management");
}

#[test]
fn consent_creates_consented_event() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let s = scope("governance");
    let ev = g
        .consent(actor("alice"), actor("bob"), "we agree", &s, boot.id.clone(), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(ev.event_type.value(), "grammar.consented");
    let content = ev.content();
    assert_eq!(content["Agreement"], "we agree");
    assert_eq!(content["Scope"], "governance");
    // Parties should be sorted: alice < bob
    let parties = content["Parties"].as_array().unwrap();
    assert_eq!(parties[0], "alice");
    assert_eq!(parties[1], "bob");
}

#[test]
fn consent_sorts_parties() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let s = scope("governance");
    // Pass bob first, alice second -- should still sort alice first
    let ev = g
        .consent(actor("bob"), actor("alice"), "agreement", &s, boot.id.clone(), conv(), &NoopSigner)
        .unwrap();

    let content = ev.content();
    let parties = content["Parties"].as_array().unwrap();
    assert_eq!(parties[0], "alice");
    assert_eq!(parties[1], "bob");
}

#[test]
fn sever_supersedes_severable_edge() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    // Create a subscription edge first
    let sub = g
        .subscribe(actor("alice"), actor("bob"), None, boot.id.clone(), conv(), &NoopSigner)
        .unwrap();

    // Now sever it
    let sever_ev = g
        .sever(actor("alice"), sub.id.clone(), boot.id.clone(), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(sever_ev.event_type.value(), "edge.superseded");
    let content = sever_ev.content();
    assert_eq!(content["PreviousEdge"], sub.id.value());
}

#[test]
fn sever_rejects_non_severable_edge() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    // Create an endorsement edge (not severable)
    let endorse = g
        .endorse(actor("alice"), boot.id.clone(), actor("bob"), Weight::new(0.5).unwrap(), None, conv(), &NoopSigner)
        .unwrap();

    let result = g.sever(actor("alice"), endorse.id.clone(), boot.id.clone(), conv(), &NoopSigner);
    assert!(result.is_err());
}

#[test]
fn sever_rejects_non_party() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let sub = g
        .subscribe(actor("alice"), actor("bob"), None, boot.id.clone(), conv(), &NoopSigner)
        .unwrap();

    // charlie is not a party to the edge
    let result = g.sever(actor("charlie"), sub.id.clone(), boot.id.clone(), conv(), &NoopSigner);
    assert!(result.is_err());
}

// ── Named functions ──────────────────────────────────────────────────

#[test]
fn challenge_creates_response_and_flag() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let (response, flag) = g
        .challenge(actor("alice"), "I disagree", boot.id.clone(), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(response.event_type.value(), "grammar.responded");
    assert_eq!(flag.event_type.value(), "grammar.annotated");
    let flag_content = flag.content();
    assert_eq!(flag_content["Key"], "dispute");
    assert_eq!(flag_content["Value"], "challenged");
}

#[test]
fn recommend_creates_propagate_and_channel() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let (prop, chan) = g
        .recommend(actor("alice"), boot.id.clone(), actor("bob"), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(prop.event_type.value(), "edge.created");
    assert_eq!(prop.content()["EdgeType"], "Reference");
    assert_eq!(chan.event_type.value(), "edge.created");
    assert_eq!(chan.content()["EdgeType"], "Channel");
}

#[test]
fn invite_creates_endorse_and_subscribe() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let w = Weight::new(0.3).unwrap();
    let (endorse_ev, sub_ev) = g
        .invite(actor("alice"), actor("bob"), w, None, boot.id.clone(), conv(), &NoopSigner)
        .unwrap();

    assert_eq!(endorse_ev.event_type.value(), "edge.created");
    assert_eq!(endorse_ev.content()["EdgeType"], "Endorsement");
    assert_eq!(sub_ev.event_type.value(), "edge.created");
    assert_eq!(sub_ev.content()["EdgeType"], "Subscription");
}

#[test]
fn forgive_creates_subscription() {
    let (mut store, boot) = setup();
    let mut g = eventgraph::grammar::Grammar::new(&mut store);

    let ev = g
        .forgive(actor("alice"), boot.id.clone(), actor("bob"), None, conv(), &NoopSigner)
        .unwrap();

    assert_eq!(ev.event_type.value(), "edge.created");
    assert_eq!(ev.content()["EdgeType"], "Subscription");
}

// ── Work Grammar compositions ────────────────────────────────────────

#[test]
fn work_intend_creates_emit() {
    let (mut store, boot) = setup();
    let mut w = eventgraph::compositions::work::WorkGrammar::new(&mut store);

    let ev = w.intend(actor("alice"), "build something", vec![boot.id.clone()], conv(), &NoopSigner).unwrap();
    assert_eq!(ev.event_type.value(), "grammar.emitted");
    assert!(ev.content()["Body"].as_str().unwrap().starts_with("intend:"));
}

#[test]
fn work_escalate_creates_block_and_handoff() {
    let (mut store, boot) = setup();
    let mut w = eventgraph::compositions::work::WorkGrammar::new(&mut store);

    let s = scope("tasks");
    let result = w.escalate(actor("alice"), "stuck on X", boot.id.clone(), actor("boss"), &s, conv(), &NoopSigner).unwrap();

    assert_eq!(result.block_event.event_type.value(), "grammar.annotated");
    assert_eq!(result.handoff_event.event_type.value(), "grammar.consented");
}

#[test]
fn work_sprint_creates_intent_subtasks_assignments() {
    let (mut store, boot) = setup();
    let mut w = eventgraph::compositions::work::WorkGrammar::new(&mut store);

    let s = scope("dev");
    let result = w.sprint(
        actor("alice"), "build v1",
        &["task A", "task B"],
        &[actor("bob"), actor("charlie")],
        &[s.clone(), s.clone()],
        vec![boot.id.clone()], conv(), &NoopSigner,
    ).unwrap();

    assert_eq!(result.intent.event_type.value(), "grammar.emitted");
    assert_eq!(result.subtasks.len(), 2);
    assert_eq!(result.assignments.len(), 2);
}

// ── Being Grammar compositions ───────────────────────────────────────

#[test]
fn being_contemplation_creates_four_events() {
    let (mut store, boot) = setup();
    let mut b = eventgraph::compositions::being::BeingGrammar::new(&mut store);

    let result = b.contemplation(
        actor("alice"), "seasons change", "the unknowable", "the cosmos", "why?",
        vec![boot.id.clone()], conv(), &NoopSigner,
    ).unwrap();

    assert!(result.change.content()["Body"].as_str().unwrap().contains("change:"));
    assert!(result.mystery.content()["Body"].as_str().unwrap().contains("mystery:"));
    assert!(result.awe.content()["Body"].as_str().unwrap().contains("marvel:"));
    assert!(result.wonder.content()["Body"].as_str().unwrap().contains("wonder:"));
}

// ── Evolution Grammar compositions ───────────────────────────────────

#[test]
fn evolution_self_evolve_creates_four_events() {
    let (mut store, boot) = setup();
    let mut e = eventgraph::compositions::evolution::EvolutionGrammar::new(&mut store);

    let result = e.self_evolve(
        actor("alice"), "repeated pattern", "proposed change", "kept", "removed cruft",
        vec![boot.id.clone()], conv(), &NoopSigner,
    ).unwrap();

    assert!(result.pattern.content()["Body"].as_str().unwrap().contains("pattern:"));
    assert!(result.adaptation.content()["Body"].as_str().unwrap().contains("adapt:"));
    assert!(result.selection.content()["Body"].as_str().unwrap().contains("select:"));
    assert!(result.simplification.content()["Body"].as_str().unwrap().contains("simplify:"));
}

// ── Chain integrity across compositions ──────────────────────────────

#[test]
fn compositions_maintain_hash_chain() {
    let (mut store, boot) = setup();

    {
        let mut w = eventgraph::compositions::work::WorkGrammar::new(&mut store);
        let _intent = w.intend(actor("alice"), "goal", vec![boot.id.clone()], conv(), &NoopSigner).unwrap();
    }

    let v = store.verify_chain();
    assert!(v.valid);
    assert_eq!(v.length, 2); // boot + intend
}

// ── Market Grammar ───────────────────────────────────────────────────

#[test]
fn market_barter_creates_three_events() {
    let (mut store, boot) = setup();
    let mut m = eventgraph::compositions::market::MarketGrammar::new(&mut store);

    let s = scope("trade");
    let result = m.barter(
        actor("alice"), actor("bob"), "apples", "oranges", &s,
        vec![boot.id.clone()], conv(), &NoopSigner,
    ).unwrap();

    assert!(result.listing.content()["Body"].as_str().unwrap().contains("list:"));
    assert!(result.counter_offer.content()["Body"].as_str().unwrap().contains("bid:"));
    assert_eq!(result.acceptance.event_type.value(), "grammar.consented");
}

// ── Bond Grammar ─────────────────────────────────────────────────────

#[test]
fn bond_connect_creates_mutual_subscriptions() {
    let (mut store, boot) = setup();
    let mut b = eventgraph::compositions::bond::BondGrammar::new(&mut store);

    let (sub1, sub2) = b.connect(
        actor("alice"), actor("bob"), None,
        boot.id.clone(), conv(), &NoopSigner,
    ).unwrap();

    assert_eq!(sub1.event_type.value(), "edge.created");
    assert_eq!(sub1.content()["EdgeType"], "Subscription");
    assert_eq!(sub1.content()["From"], "alice");
    assert_eq!(sub1.content()["To"], "bob");

    assert_eq!(sub2.event_type.value(), "edge.created");
    assert_eq!(sub2.content()["EdgeType"], "Subscription");
    assert_eq!(sub2.content()["From"], "bob");
    assert_eq!(sub2.content()["To"], "alice");
}

// ── Knowledge Grammar ────────────────────────────────────────────────

#[test]
fn knowledge_fact_check_creates_three_events() {
    let (mut store, boot) = setup();
    let mut k = eventgraph::compositions::knowledge::KnowledgeGrammar::new(&mut store);

    let claim_ev = k.claim(actor("alice"), "the sky is blue", vec![boot.id.clone()], conv(), &NoopSigner).unwrap();
    let result = k.fact_check(
        actor("alice"), claim_ev.id.clone(), "observation", "none detected", "confirmed",
        conv(), &NoopSigner,
    ).unwrap();

    assert_eq!(result.provenance.event_type.value(), "grammar.annotated");
    assert_eq!(result.bias_check.event_type.value(), "grammar.annotated");
    assert_eq!(result.verdict.event_type.value(), "grammar.merged");
}
