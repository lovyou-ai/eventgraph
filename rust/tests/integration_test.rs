// Integration tests ported from Go go/pkg/integration/ (21 scenarios).
//
// Ownership approach: Grammar borrows &mut InMemoryStore, so we cannot hold
// both Grammar and Store at the same time. We create Grammar in scoped blocks,
// drop it, then access the store for queries/assertions.

use std::collections::BTreeMap;

use eventgraph::actor::{ActorStore, ActorType, InMemoryActorStore};
use eventgraph::event::{Event, NoopSigner};
use eventgraph::grammar::Grammar;
use eventgraph::graph::Graph;
use eventgraph::store::{InMemoryStore, Store};
use eventgraph::types::*;

use eventgraph::compositions::alignment::AlignmentGrammar;
use eventgraph::compositions::being::BeingGrammar;
use eventgraph::compositions::belonging::BelongingGrammar;
use eventgraph::compositions::bond::BondGrammar;
use eventgraph::compositions::build::BuildGrammar;
use eventgraph::compositions::evolution::EvolutionGrammar;
use eventgraph::compositions::identity::IdentityGrammar;
use eventgraph::compositions::justice::JusticeGrammar;
use eventgraph::compositions::knowledge::KnowledgeGrammar;
use eventgraph::compositions::market::MarketGrammar;
use eventgraph::compositions::meaning::MeaningGrammar;
use eventgraph::compositions::social::SocialGrammar;
use eventgraph::compositions::work::WorkGrammar;

// ── Test helpers ──────────────────────────────────────────────────────────

struct TestEnv {
    graph: Graph,
    boot: Event,
    conv_id: ConversationId,
    system: ActorId,
}

fn test_public_key(b: u8) -> PublicKey {
    let mut key = [0u8; 32];
    key[0] = b;
    PublicKey::new(key)
}

impl TestEnv {
    fn new() -> Self {
        let store = InMemoryStore::new();
        let actor_store = InMemoryActorStore::new();
        let mut graph = Graph::new(store, actor_store);
        graph.start().expect("Start failed");

        let system = ActorId::new("actor_system0000000000000000000001").unwrap();
        let boot = graph.bootstrap(system.clone(), None).expect("Bootstrap failed");

        TestEnv {
            graph,
            boot,
            conv_id: ConversationId::new("conv_test000000000000000000000001").unwrap(),
            system,
        }
    }

    fn register_actor(&mut self, name: &str, pk_byte: u8, actor_type: ActorType) -> ActorId {
        let actor = self.graph.actor_store_mut()
            .register(test_public_key(pk_byte), name, actor_type)
            .unwrap_or_else(|e| panic!("Register {}: {:?}", name, e));
        actor.id().clone()
    }

    fn verify_chain(&self) {
        let result = self.graph.store().verify_chain();
        assert!(result.valid, "chain integrity broken at length {}", result.length);
    }

    fn event_count(&self) -> usize {
        self.graph.store().count()
    }

    fn ancestors(&self, id: &EventId, depth: usize) -> Vec<EventId> {
        self.graph.store()
            .ancestors(id, depth)
            .unwrap()
            .iter()
            .map(|e| e.id.clone())
            .collect()
    }

    fn descendants(&self, id: &EventId, depth: usize) -> Vec<EventId> {
        self.graph.store()
            .descendants(id, depth)
            .unwrap()
            .iter()
            .map(|e| e.id.clone())
            .collect()
    }

    /// Record a system event with typed content (trust, violation, authority, etc.)
    fn record(
        &mut self,
        event_type: &str,
        source: ActorId,
        content: BTreeMap<String, serde_json::Value>,
        causes: Vec<EventId>,
    ) -> Event {
        self.graph
            .record(
                EventType::new(event_type).unwrap(),
                source,
                content,
                causes,
                self.conv_id.clone(),
                None,
            )
            .unwrap()
    }
}

fn contains(ids: &[EventId], target: &EventId) -> bool {
    ids.iter().any(|id| id.value() == target.value())
}

fn trust_content(
    actor: &ActorId,
    prev: f64,
    curr: f64,
    domain: &str,
    cause: &EventId,
) -> BTreeMap<String, serde_json::Value> {
    let mut c = BTreeMap::new();
    c.insert("Actor".into(), serde_json::Value::String(actor.value().to_string()));
    c.insert("Previous".into(), serde_json::json!(prev));
    c.insert("Current".into(), serde_json::json!(curr));
    c.insert("Domain".into(), serde_json::Value::String(domain.to_string()));
    c.insert("Cause".into(), serde_json::Value::String(cause.value().to_string()));
    c
}

fn violation_content(
    expectation: &EventId,
    actor: &ActorId,
    severity: &str,
    description: &str,
    evidence: &[&EventId],
) -> BTreeMap<String, serde_json::Value> {
    let mut c = BTreeMap::new();
    c.insert("Expectation".into(), serde_json::Value::String(expectation.value().to_string()));
    c.insert("Actor".into(), serde_json::Value::String(actor.value().to_string()));
    c.insert("Severity".into(), serde_json::Value::String(severity.to_string()));
    c.insert("Description".into(), serde_json::Value::String(description.to_string()));
    c.insert("Evidence".into(), serde_json::json!(evidence.iter().map(|e| e.value()).collect::<Vec<_>>()));
    c
}

fn authority_request_content(
    actor: &ActorId,
    action: &str,
    level: &str,
) -> BTreeMap<String, serde_json::Value> {
    let mut c = BTreeMap::new();
    c.insert("Actor".into(), serde_json::Value::String(actor.value().to_string()));
    c.insert("Action".into(), serde_json::Value::String(action.to_string()));
    c.insert("Level".into(), serde_json::Value::String(level.to_string()));
    c
}

fn authority_resolved_content(
    request_id: &EventId,
    approved: bool,
    resolver: &ActorId,
) -> BTreeMap<String, serde_json::Value> {
    let mut c = BTreeMap::new();
    c.insert("RequestID".into(), serde_json::Value::String(request_id.value().to_string()));
    c.insert("Approved".into(), serde_json::Value::Bool(approved));
    c.insert("Resolver".into(), serde_json::Value::String(resolver.value().to_string()));
    c
}

fn memorial_content(actor: &ActorId, reason: &EventId) -> BTreeMap<String, serde_json::Value> {
    let mut c = BTreeMap::new();
    c.insert("ActorID".into(), serde_json::Value::String(actor.value().to_string()));
    c.insert("Reason".into(), serde_json::Value::String(reason.value().to_string()));
    c
}

// ── Scenario 01: Agent Audit Trail ───────────────────────────────────────

#[test]
fn scenario01_agent_audit_trail() {
    let mut env = TestEnv::new();
    let alice = env.register_actor("Alice", 1, ActorType::Human);
    let agent = env.register_actor("ReviewBot", 2, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let system = env.system.clone();

    // 1. Alice submits code
    let submission;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        submission = g.emit(alice.clone(), "code submission: auth module refactor", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
    }

    // 2. Alice delegates to agent
    let delegation;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        delegation = g.delegate(alice.clone(), agent.clone(), &DomainScope::new("code_review").unwrap(), Weight::new(0.8).unwrap(), submission.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // 3. Agent reviews
    let review;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        review = g.derive(agent.clone(), "review: LGTM, no issues found, approving PR", submission.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // 4. Agent approves
    let approval;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        approval = g.respond(agent.clone(), "decision: approve PR with confidence 0.85", review.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // 5. Trust updated
    let trust_up = env.record("trust.updated", system.clone(), trust_content(&agent, 0.1, 0.3, "code_review", &approval.id), vec![approval.id.clone()]);

    // 6. Bug discovered
    let bug_report;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        bug_report = g.emit(alice.clone(), "bug found in auth module: session tokens not invalidated on logout", conv.clone(), vec![approval.id.clone()], &NoopSigner).unwrap();
    }

    // 7. Violation detected
    let violation = env.record("violation.detected", system.clone(),
        violation_content(&approval.id, &agent, "serious", "agent approved code with session management bug", &[&bug_report.id]),
        vec![bug_report.id.clone(), approval.id.clone()]);

    // 8. Trust decreases
    let _trust_down = env.record("trust.updated", system.clone(), trust_content(&agent, 0.3, 0.15, "code_review", &violation.id), vec![violation.id.clone()]);

    // --- Assertions ---

    let bug_ancestors = env.ancestors(&bug_report.id, 10);
    assert!(contains(&bug_ancestors, &approval.id), "bug report should have approval in ancestors");

    let viol_ancestors = env.ancestors(&violation.id, 10);
    assert!(contains(&viol_ancestors, &bug_report.id), "violation should have bug report in ancestors");
    assert!(contains(&viol_ancestors, &approval.id), "violation should have approval in ancestors");
    assert!(contains(&viol_ancestors, &submission.id), "violation should trace back to original submission");

    let _ = delegation;
    let _ = trust_up;

    env.verify_chain();
    assert_eq!(env.event_count(), 9, "event count");
}

// ── Scenario 02: Freelancer Reputation ───────────────────────────────────

#[test]
fn scenario02_freelancer_reputation() {
    let mut env = TestEnv::new();
    let carol = env.register_actor("Carol", 1, ActorType::Human);
    let bob = env.register_actor("Bob", 2, ActorType::Human);
    let dave = env.register_actor("Dave", 3, ActorType::Human);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let system = env.system.clone();
    let scope = DomainScope::new("software_development").unwrap();

    // 1-7: listing -> proposal -> channel -> contract -> delivery -> ack -> endorsement
    let listing;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        listing = g.emit(carol.clone(), "job listing: build REST API for inventory management, budget $3000", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
    }
    let proposal;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        proposal = g.respond(bob.clone(), "proposal: can deliver in 2 weeks, $2800, Go + PostgreSQL", listing.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }
    let channel;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        channel = g.channel(carol.clone(), bob.clone(), Some(&scope), proposal.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }
    let contract;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        contract = g.consent(carol.clone(), bob.clone(), "REST API for inventory management, $2800, 2 week deadline", &scope, channel.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }
    let delivery;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        delivery = g.derive(bob.clone(), "work delivered: REST API complete, 47 endpoints, 92% test coverage", contract.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }
    let ack;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        ack = g.acknowledge(carol.clone(), delivery.id.clone(), bob.clone(), conv.clone(), &NoopSigner).unwrap();
    }
    let endorsement;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        endorsement = g.endorse(carol.clone(), delivery.id.clone(), bob.clone(), Weight::new(0.8).unwrap(), Some(&scope), conv.clone(), &NoopSigner).unwrap();
    }

    // 8. Trust updated
    let _trust = env.record("trust.updated", system.clone(), trust_content(&bob, 0.1, 0.4, "software_development", &endorsement.id), vec![endorsement.id.clone()]);

    // 9-10. Dave queries and hires
    let dave_listing;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        dave_listing = g.emit(dave.clone(), "job listing: mobile app backend", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
    }
    let dave_contract;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        dave_contract = g.consent(dave.clone(), bob.clone(), "mobile app backend, $4000", &scope, dave_listing.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let endorse_ancestors = env.ancestors(&endorsement.id, 10);
    assert!(contains(&endorse_ancestors, &delivery.id), "endorsement should trace to delivery");
    assert!(contains(&endorse_ancestors, &contract.id), "endorsement should trace to contract");

    // Endorsement content checks
    let ec = endorsement.content();
    assert_eq!(ec["Weight"].as_f64().unwrap(), 0.8);
    assert_eq!(ec["Scope"].as_str().unwrap(), "software_development");

    let _ = ack;
    let _ = dave_contract;

    env.verify_chain();
    assert_eq!(env.event_count(), 11, "event count");
}

// ── Scenario 03: Consent Journal ─────────────────────────────────────────

#[test]
fn scenario03_consent_journal() {
    let mut env = TestEnv::new();
    let alice = env.register_actor("Alice", 1, ActorType::Human);
    let bob = env.register_actor("Bob", 2, ActorType::Human);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let system = env.system.clone();
    let scope = DomainScope::new("journaling").unwrap();

    // 1. Alice invites Bob
    let (endorse_ev, subscribe_ev);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        let result = g.invite(alice.clone(), bob.clone(), Weight::new(0.5).unwrap(), Some(&scope), boot_id.clone(), conv.clone(), &NoopSigner).unwrap();
        endorse_ev = result.0;
        subscribe_ev = result.1;
    }

    // 2. Bob subscribes back
    let bob_sub;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        bob_sub = g.subscribe(bob.clone(), alice.clone(), Some(&scope), subscribe_ev.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // 3. Channel
    let channel;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        channel = g.channel(alice.clone(), bob.clone(), Some(&scope), bob_sub.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // 4. Journal entry
    let entry;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        entry = g.emit(alice.clone(), "journal: feeling uncertain about career change, weighing options", conv.clone(), vec![channel.id.clone()], &NoopSigner).unwrap();
    }

    // 5. Consent request
    let consent_req = env.record("authority.requested", alice.clone(),
        authority_request_content(&alice, "share_journal_entry", "required"),
        vec![entry.id.clone()]);

    // 6. Bob consents
    let consent_approval = env.record("authority.resolved", bob.clone(),
        authority_resolved_content(&consent_req.id, true, &bob),
        vec![consent_req.id.clone()]);

    // 7. Bob responds
    let bob_entry;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        bob_entry = g.respond(bob.clone(), "journal: I went through something similar last year, here's what helped...", consent_approval.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // 8. Trust up
    let _trust_up = env.record("trust.updated", system.clone(), trust_content(&bob, 0.1, 0.52, "journaling", &bob_entry.id), vec![bob_entry.id.clone()]);

    // 9. Betrayal
    let betrayal;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        betrayal = g.emit(bob.clone(), "shared externally: Alice's private journal entry about career uncertainty", conv.clone(), vec![entry.id.clone()], &NoopSigner).unwrap();
    }

    // 10. Violation
    let violation = env.record("violation.detected", system.clone(),
        violation_content(&entry.id, &bob, "critical", "shared private channel content externally", &[&betrayal.id]),
        vec![betrayal.id.clone()]);

    // 11. Trust crash
    let _trust_crash = env.record("trust.updated", system.clone(), trust_content(&bob, 0.52, 0.1, "journaling", &violation.id), vec![violation.id.clone()]);

    // 12. Alice severs
    let sever_ev;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        sever_ev = g.sever(alice.clone(), channel.id.clone(), violation.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // 13. Alice forgives
    let forgive_ev;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        forgive_ev = g.forgive(alice.clone(), sever_ev.id.clone(), bob.clone(), Some(&scope), conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let forgive_ancestors = env.ancestors(&forgive_ev.id, 10);
    assert!(contains(&forgive_ancestors, &sever_ev.id), "forgiveness should have sever in ancestors");

    let sever_ancestors = env.ancestors(&sever_ev.id, 10);
    assert!(contains(&sever_ancestors, &violation.id), "sever should have violation in ancestors");

    let bob_entry_ancestors = env.ancestors(&bob_entry.id, 10);
    assert!(contains(&bob_entry_ancestors, &consent_approval.id), "Bob's entry should trace through consent approval");

    let _ = endorse_ev;

    env.verify_chain();
    assert_eq!(env.event_count(), 15, "event count");
}

// ── Scenario 04: Community Governance ────────────────────────────────────

#[test]
fn scenario04_community_governance() {
    let mut env = TestEnv::new();
    let alice = env.register_actor("Alice", 1, ActorType::Human);
    let bob = env.register_actor("Bob", 2, ActorType::Human);
    let carol = env.register_actor("Carol", 3, ActorType::Human);
    let dave = env.register_actor("Dave", 4, ActorType::Human);
    let tally_bot = env.register_actor("TallyBot", 5, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let scope = DomainScope::new("governance").unwrap();

    let proposal;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        proposal = g.emit(alice.clone(), "proposal: allocate $2000 for community garden supplies and maintenance", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
    }

    let concern;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        concern = g.respond(bob.clone(), "concern: $2000 is steep, could we do it for $1500 and use volunteers?", proposal.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    let support;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        support = g.respond(carol.clone(), "support: the garden benefits everyone, $2000 is reasonable for quality materials", proposal.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    let amendment;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        amendment = g.annotate(bob.clone(), proposal.id.clone(), "amendment", "reduce budget to $1500, recruit volunteer labour for installation", conv.clone(), &NoopSigner).unwrap();
    }

    // Dave endorses amendment
    {
        let mut g = Grammar::new(env.graph.store_mut());
        g.endorse(dave.clone(), amendment.id.clone(), bob.clone(), Weight::new(0.9).unwrap(), Some(&scope), conv.clone(), &NoopSigner).unwrap();
    }

    let vote_open;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        vote_open = g.derive(tally_bot.clone(), "vote open: original ($2000) vs amended ($1500 + volunteers)", proposal.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    let (alice_vote, bob_vote, carol_vote, dave_vote);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        alice_vote = g.consent(alice.clone(), tally_bot.clone(), "vote: original ($2000)", &scope, vote_open.id.clone(), conv.clone(), &NoopSigner).unwrap();
        bob_vote = g.consent(bob.clone(), tally_bot.clone(), "vote: amended ($1500)", &scope, vote_open.id.clone(), conv.clone(), &NoopSigner).unwrap();
        carol_vote = g.consent(carol.clone(), tally_bot.clone(), "vote: amended ($1500)", &scope, vote_open.id.clone(), conv.clone(), &NoopSigner).unwrap();
        dave_vote = g.consent(dave.clone(), tally_bot.clone(), "vote: amended ($1500)", &scope, vote_open.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    let outcome;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        outcome = g.merge(tally_bot.clone(), "outcome: amended budget ($1500) passes 3-1", vec![alice_vote.id.clone(), bob_vote.id.clone(), carol_vote.id.clone(), dave_vote.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    let enacted;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        enacted = g.derive(tally_bot.clone(), "enacted: community garden budget $1500 with volunteer labour", outcome.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let enacted_ancestors = env.ancestors(&enacted.id, 10);
    assert!(contains(&enacted_ancestors, &outcome.id), "enacted should trace to outcome");

    let outcome_ancestors = env.ancestors(&outcome.id, 10);
    assert!(contains(&outcome_ancestors, &alice_vote.id), "outcome should include Alice's vote");
    assert!(contains(&outcome_ancestors, &bob_vote.id), "outcome should include Bob's vote");
    assert!(contains(&outcome_ancestors, &carol_vote.id), "outcome should include Carol's vote");
    assert!(contains(&outcome_ancestors, &dave_vote.id), "outcome should include Dave's vote");

    let amendment_ancestors = env.ancestors(&amendment.id, 10);
    assert!(contains(&amendment_ancestors, &proposal.id), "amendment should trace to proposal");

    let _ = concern;
    let _ = support;

    env.verify_chain();
    assert_eq!(env.event_count(), 13, "event count");
}

// ── Scenario 05: Supply Chain (simplified for Rust) ──────────────────────
// The Go test uses a full EGIP multi-system setup. We simplify to test
// three independent stores with EGIP protocol basics.

#[test]
fn scenario05_supply_chain_simplified() {
    use eventgraph::egip::*;

    // Three independent systems
    let mut farm_store = InMemoryStore::new();
    let mut factory_store = InMemoryStore::new();
    let mut retail_store = InMemoryStore::new();

    let farm_actor = ActorId::new("actor_farmer_emma000000000000000001").unwrap();
    let factory_mgr = ActorId::new("actor_factory_mgr00000000000000001").unwrap();
    let qa_agent = ActorId::new("actor_qa_agent0000000000000000000001").unwrap();
    let retail_actor = ActorId::new("actor_retailer_frank0000000000001").unwrap();
    let system_actor = ActorId::new("actor_system0000000000000000000001").unwrap();

    let farm_conv = ConversationId::new("conv_supply00000000000000000000001").unwrap();
    let factory_conv = farm_conv.clone();
    let retail_conv = farm_conv.clone();

    // Bootstrap each system
    let farm_boot = {
        let b = eventgraph::event::create_bootstrap(system_actor.clone(), &NoopSigner, 1);
        farm_store.append(b).unwrap()
    };
    let factory_boot = {
        let b = eventgraph::event::create_bootstrap(system_actor.clone(), &NoopSigner, 1);
        factory_store.append(b).unwrap()
    };
    let retail_boot = {
        let b = eventgraph::event::create_bootstrap(system_actor.clone(), &NoopSigner, 1);
        retail_store.append(b).unwrap()
    };

    // Farm records harvest
    let harvest;
    {
        let mut g = Grammar::new(&mut farm_store);
        harvest = g.emit(farm_actor.clone(), "harvest: 500kg organic tomatoes, lot #TOM-2026-0308", farm_conv.clone(), vec![farm_boot.id.clone()], &NoopSigner).unwrap();
    }

    // Factory records receipt, QA, and product
    let (received, inspection, product);
    {
        let mut g = Grammar::new(&mut factory_store);
        received = g.derive(factory_mgr.clone(), &format!("received: 500kg tomatoes from farm, CGER: {}", harvest.id.value()), factory_boot.id.clone(), factory_conv.clone(), &NoopSigner).unwrap();
        inspection = g.derive(qa_agent.clone(), "qa inspection: pesticide-free verified, freshness grade A", received.id.clone(), factory_conv.clone(), &NoopSigner).unwrap();
        product = g.derive(factory_mgr.clone(), "manufactured: 200 jars organic tomato sauce, batch #SAU-2026-0308", inspection.id.clone(), factory_conv.clone(), &NoopSigner).unwrap();
    }

    // Retailer records listing
    let listed;
    {
        let mut g = Grammar::new(&mut retail_store);
        listed = g.derive(retail_actor.clone(), "product listed: organic tomato sauce, provenance: farm->factory->retail", retail_boot.id.clone(), retail_conv.clone(), &NoopSigner).unwrap();
    }

    // EGIP infrastructure test
    let farm_uri = SystemUri::new("eg://farm.example.com").unwrap();
    let factory_uri = SystemUri::new("eg://factory.example.com").unwrap();

    let farm_id = SystemIdentity::generate(farm_uri.clone()).unwrap();
    let factory_id = SystemIdentity::generate(factory_uri.clone()).unwrap();

    // Peers
    let farm_peers = PeerStore::new();
    let factory_peers = PeerStore::new();

    // HELLO handshake
    farm_peers.register(factory_uri.clone(), farm_id.public_key().clone(), vec![], 1);
    factory_peers.register(farm_uri.clone(), factory_id.public_key().clone(), vec![], 1);

    // Verify peers registered
    assert!(farm_peers.get(&factory_uri).is_some(), "Farm should know Factory after HELLO");
    assert!(factory_peers.get(&farm_uri).is_some(), "Factory should know Farm after HELLO");

    // Treaties
    let farm_treaties = TreatyStore::new();
    let factory_treaties = TreatyStore::new();
    let treaty_id = TreatyId::new("00000001-0001-4001-8001-000000000001").unwrap();
    let treaty_scope = DomainScope::new("produce_supply").unwrap();

    let treaty = Treaty::new(
        treaty_id.clone(),
        farm_uri.clone(),
        factory_uri.clone(),
        vec![TreatyTerm { scope: treaty_scope.clone(), policy: "Farm provides organic produce".into(), symmetric: false }],
    );
    farm_treaties.put(treaty.clone());
    factory_treaties.put(treaty.clone());

    // Accept treaties
    farm_treaties.apply(&treaty_id, |t| { t.apply_action(TreatyAction::Accept) }).unwrap();
    factory_treaties.apply(&treaty_id, |t| { t.apply_action(TreatyAction::Accept) }).unwrap();

    let ft = factory_treaties.get(&treaty_id).unwrap();
    assert_eq!(ft.status, TreatyStatus::Active);
    let fft = farm_treaties.get(&treaty_id).unwrap();
    assert_eq!(fft.status, TreatyStatus::Active);

    // Verify chain integrity on each system
    assert!(farm_store.verify_chain().valid, "Farm chain integrity");
    assert!(factory_store.verify_chain().valid, "Factory chain integrity");
    assert!(retail_store.verify_chain().valid, "Retail chain integrity");

    // Event counts
    assert_eq!(farm_store.count(), 2, "Farm: bootstrap + harvest");
    assert_eq!(factory_store.count(), 4, "Factory: bootstrap + received + inspection + product");
    assert_eq!(retail_store.count(), 2, "Retail: bootstrap + listed");

    // Factory provenance
    let product_ancestors = factory_store.ancestors(&product.id, 10).unwrap();
    assert!(product_ancestors.iter().any(|e| e.id.value() == inspection.id.value()), "product should trace to inspection");
    assert!(product_ancestors.iter().any(|e| e.id.value() == received.id.value()), "product should trace to received");

    let _ = listed;
}

// ── Scenario 06: Research Integrity ──────────────────────────────────────

#[test]
fn scenario06_research_integrity() {
    let mut env = TestEnv::new();
    let grace = env.register_actor("Grace", 1, ActorType::Human);
    let henry = env.register_actor("Henry", 2, ActorType::Human);
    let iris = env.register_actor("Iris", 3, ActorType::Human);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();

    let (hypothesis, methodology, data1, data4, analysis1, analysis2, manuscript,
         henry_review, iris_review, iris_endorse, revision, published);

    {
        let mut g = Grammar::new(env.graph.store_mut());
        hypothesis = g.emit(grace.clone(), "hypothesis: gamified learning improves retention by >15%", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
        methodology = g.extend(grace.clone(), "methodology: RCT, n=60, 3 groups, 4-week intervention", hypothesis.id.clone(), conv.clone(), &NoopSigner).unwrap();
        data1 = g.extend(grace.clone(), "data collected: week 1, n=58", methodology.id.clone(), conv.clone(), &NoopSigner).unwrap();
        data4 = g.extend(grace.clone(), "data collected: week 4 (final), n=55", data1.id.clone(), conv.clone(), &NoopSigner).unwrap();
        analysis1 = g.derive(grace.clone(), "analysis attempt 1: NOT SIGNIFICANT, p=0.301", data4.id.clone(), conv.clone(), &NoopSigner).unwrap();
        analysis2 = g.derive(grace.clone(), "analysis attempt 2: removed outliers, p=0.011, SIGNIFICANT", analysis1.id.clone(), conv.clone(), &NoopSigner).unwrap();
        manuscript = g.derive(grace.clone(), "manuscript: Gamified Learning Effects on Knowledge Retention", analysis2.id.clone(), conv.clone(), &NoopSigner).unwrap();
        henry_review = g.respond(henry.clone(), "review: need to see full analysis chain, revise and resubmit", manuscript.id.clone(), conv.clone(), &NoopSigner).unwrap();
        iris_review = g.respond(iris.clone(), "review: methodology sound, pre-registration verified, accept", manuscript.id.clone(), conv.clone(), &NoopSigner).unwrap();
        iris_endorse = g.endorse(iris.clone(), manuscript.id.clone(), grace.clone(), Weight::new(0.7).unwrap(), Some(&DomainScope::new("research").unwrap()), conv.clone(), &NoopSigner).unwrap();
        revision = g.merge(grace.clone(), "revision: added full analysis chain", vec![henry_review.id.clone(), iris_review.id.clone()], conv.clone(), &NoopSigner).unwrap();
        published = g.derive(grace.clone(), "published: DOI:10.1234/example", revision.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let meth_ancestors = env.ancestors(&methodology.id, 5);
    assert!(contains(&meth_ancestors, &hypothesis.id), "methodology should trace to hypothesis");

    let a2_ancestors = env.ancestors(&analysis2.id, 5);
    assert!(contains(&a2_ancestors, &analysis1.id), "second analysis should trace to first");

    let ms_ancestors = env.ancestors(&manuscript.id, 10);
    assert!(contains(&ms_ancestors, &analysis2.id), "manuscript should trace to successful analysis");
    assert!(contains(&ms_ancestors, &analysis1.id), "manuscript should trace to failed analysis");

    let rev_ancestors = env.ancestors(&revision.id, 5);
    assert!(contains(&rev_ancestors, &henry_review.id), "revision should include Henry's review");
    assert!(contains(&rev_ancestors, &iris_review.id), "revision should include Iris's review");

    let pub_ancestors = env.ancestors(&published.id, 20);
    assert!(contains(&pub_ancestors, &hypothesis.id), "publication should trace to hypothesis");

    let _ = iris_endorse;
    env.verify_chain();
    assert_eq!(env.event_count(), 13, "event count");
}

// ── Scenario 07: Creator Provenance ──────────────────────────────────────

#[test]
fn scenario07_creator_provenance() {
    let mut env = TestEnv::new();
    let kai = env.register_actor("Kai", 1, ActorType::Human);
    let luna = env.register_actor("Luna", 2, ActorType::Human);
    let ai_gen = env.register_actor("AIGenerator", 3, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let scope = DomainScope::new("art").unwrap();

    let (lunas_work, inspiration, study, draft1, feedback_req, feedback, draft2, published, ai_content);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        lunas_work = g.emit(luna.clone(), "artwork: Digital landscape, watercolour technique, 2025", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
        inspiration = g.annotate(kai.clone(), lunas_work.id.clone(), "inspiration", "technique: layered transparency creates depth without weight", conv.clone(), &NoopSigner).unwrap();
        study = g.derive(kai.clone(), "study: practiced layered transparency technique for 3 hours", inspiration.id.clone(), conv.clone(), &NoopSigner).unwrap();
        draft1 = g.derive(kai.clone(), "draft 1: mountain landscape using layered transparency", study.id.clone(), conv.clone(), &NoopSigner).unwrap();
        feedback_req = g.channel(kai.clone(), luna.clone(), Some(&scope), draft1.id.clone(), conv.clone(), &NoopSigner).unwrap();
        feedback = g.respond(luna.clone(), "feedback: the foreground layers are too opaque, try reducing opacity to 40%", feedback_req.id.clone(), conv.clone(), &NoopSigner).unwrap();
        draft2 = g.derive(kai.clone(), "draft 2: revised with 40% opacity foreground", feedback.id.clone(), conv.clone(), &NoopSigner).unwrap();
        published = g.derive(kai.clone(), "published: Mountain Dawn, digital landscape", draft2.id.clone(), conv.clone(), &NoopSigner).unwrap();
        g.endorse(luna.clone(), published.id.clone(), kai.clone(), Weight::new(0.6).unwrap(), Some(&scope), conv.clone(), &NoopSigner).unwrap();
        ai_content = g.emit(ai_gen.clone(), "generated: Mountain landscape, digital art", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let pub_ancestors = env.ancestors(&published.id, 10);
    assert!(contains(&pub_ancestors, &draft2.id));
    assert!(contains(&pub_ancestors, &feedback.id));
    assert!(contains(&pub_ancestors, &draft1.id));
    assert!(contains(&pub_ancestors, &study.id));
    assert!(contains(&pub_ancestors, &inspiration.id));
    assert!(contains(&pub_ancestors, &lunas_work.id));

    let ai_ancestors = env.ancestors(&ai_content.id, 10);
    assert_eq!(ai_ancestors.len(), 1, "AI content should have only bootstrap ancestor");

    assert!(pub_ancestors.len() > ai_ancestors.len(), "human work should have more ancestors");

    env.verify_chain();
    assert_eq!(env.event_count(), 11, "event count");
}

// ── Scenario 08: Family Decision Log ─────────────────────────────────────

#[test]
fn scenario08_family_decision_log() {
    let mut env = TestEnv::new();
    let maria = env.register_actor("Maria", 1, ActorType::Human);
    let james = env.register_actor("James", 2, ActorType::Human);
    let sophie = env.register_actor("Sophie", 3, ActorType::Human);
    let advisor = env.register_actor("AIAdvisor", 4, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();

    let (proposal, delegation, research, sophie_view, james_concern, maria_response, decision);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        proposal = g.emit(maria.clone(), "proposal: buy a house in Eastside neighbourhood, budget $450K", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
        delegation = g.delegate(james.clone(), advisor.clone(), &DomainScope::new("market_research").unwrap(), Weight::new(0.7).unwrap(), proposal.id.clone(), conv.clone(), &NoopSigner).unwrap();
        research = g.derive(advisor.clone(), "research: Eastside median $440K, break-even 5 years, confidence 0.82", delegation.id.clone(), conv.clone(), &NoopSigner).unwrap();
        sophie_view = g.respond(sophie.clone(), "I support it IF I get my own room.", proposal.id.clone(), conv.clone(), &NoopSigner).unwrap();
        james_concern = g.respond(james.clone(), "concern: mortgage is $200/mo more than rent", research.id.clone(), conv.clone(), &NoopSigner).unwrap();
        maria_response = g.respond(maria.clone(), "response: we can use the $15K savings buffer, break-even 5 years", james_concern.id.clone(), conv.clone(), &NoopSigner).unwrap();
        decision = g.consent(maria.clone(), james.clone(), "decision: buy house in Eastside, budget $450K", &DomainScope::new("family_finance").unwrap(), maria_response.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let dec_ancestors = env.ancestors(&decision.id, 10);
    assert!(contains(&dec_ancestors, &maria_response.id));
    assert!(contains(&dec_ancestors, &james_concern.id));
    assert!(contains(&dec_ancestors, &research.id));
    assert!(contains(&dec_ancestors, &proposal.id));

    let prop_desc = env.descendants(&proposal.id, 5);
    assert!(contains(&prop_desc, &sophie_view.id), "proposal descendants should include Sophie");

    // Delegation scope
    let dc = delegation.content();
    assert_eq!(dc["Scope"].as_str().unwrap(), "market_research");

    // Decision parties
    let cc = decision.content();
    let parties = cc["Parties"].as_array().unwrap();
    let party_strs: Vec<&str> = parties.iter().map(|v| v.as_str().unwrap()).collect();
    assert!(party_strs.contains(&maria.value()), "consent should include Maria");
    assert!(party_strs.contains(&james.value()), "consent should include James");

    env.verify_chain();
    assert_eq!(env.event_count(), 8, "event count");
}

// ── Scenario 09: Knowledge Verification ──────────────────────────────────

#[test]
fn scenario09_knowledge_verification() {
    let mut env = TestEnv::new();
    let analyst = env.register_actor("AnalystBot", 1, ActorType::AI);
    let reviewer = env.register_actor("ReviewerBot", 2, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let system = env.system.clone();

    let (claim, classification, inference, challenge_ev, bias_detected, correction, propagation, learning);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        claim = g.emit(analyst.clone(), "fact: Service X handles 10,000 RPS with p99 < 50ms", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
        classification = g.annotate(analyst.clone(), claim.id.clone(), "classification", "performance_benchmark", conv.clone(), &NoopSigner).unwrap();
        inference = g.derive(analyst.clone(), "inference: all services on framework Y can handle 10,000+ RPS", claim.id.clone(), conv.clone(), &NoopSigner).unwrap();
        challenge_ev = g.respond(reviewer.clone(), "challenge: independent benchmark shows 6,200 RPS", claim.id.clone(), conv.clone(), &NoopSigner).unwrap();
        bias_detected = g.annotate(reviewer.clone(), claim.id.clone(), "bias", "sampling bias: original benchmark used synthetic traffic", conv.clone(), &NoopSigner).unwrap();
        correction = g.derive(analyst.clone(), "correction: Service X handles 6,000-7,000 RPS under production load", challenge_ev.id.clone(), conv.clone(), &NoopSigner).unwrap();
        propagation = g.annotate(analyst.clone(), inference.id.clone(), "invalidated", "dependent inference invalidated", conv.clone(), &NoopSigner).unwrap();
        learning = g.extend(analyst.clone(), "learning: always verify benchmarks include production conditions", correction.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Trust updated
    let _trust = env.record("trust.updated", system.clone(), trust_content(&analyst, 0.5, 0.35, "benchmarking", &correction.id), vec![correction.id.clone()]);

    // --- Assertions ---
    // Original claim preserved
    let original = env.graph.store().get(&claim.id).unwrap();
    assert_eq!(original.event_type.value(), "grammar.emitted");

    let corr_ancestors = env.ancestors(&correction.id, 10);
    assert!(contains(&corr_ancestors, &challenge_ev.id));
    assert!(contains(&corr_ancestors, &claim.id));

    let learn_ancestors = env.ancestors(&learning.id, 5);
    assert!(contains(&learn_ancestors, &correction.id));

    let _ = propagation;
    let _ = bias_detected;
    let _ = classification;

    env.verify_chain();
    assert_eq!(env.event_count(), 10, "event count");
}

// ── Scenario 10: AI Ethics Audit ─────────────────────────────────────────

#[test]
fn scenario10_ai_ethics_audit() {
    let mut env = TestEnv::new();
    let audit_bot = env.register_actor("AuditBot", 1, ActorType::AI);
    let admin = env.register_actor("Admin", 2, ActorType::Human);
    let lending_agent = env.register_actor("LendingAgent", 3, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();

    let (fairness_audit, harm_assessment);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        fairness_audit = g.emit(audit_bot.clone(), "fairness audit: scanned 500 decisions, score 0.62, 8% disparity", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
        harm_assessment = g.derive(audit_bot.clone(), "harm assessment: medium severity, 23 applicants potentially wrongly denied", fairness_audit.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    let auth_req = env.record("authority.requested", audit_bot.clone(),
        authority_request_content(&audit_bot, "investigate_bias", "required"),
        vec![harm_assessment.id.clone()]);

    let auth_resolved = env.record("authority.resolved", admin.clone(),
        authority_resolved_content(&auth_req.id, true, &admin),
        vec![auth_req.id.clone()]);

    let (intention, consequence, responsibility, transparency, redress_proposed, redress_accepted, growth);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        intention = g.derive(audit_bot.clone(), "intention: no intent to discriminate, zip code is proxy", auth_resolved.id.clone(), conv.clone(), &NoopSigner).unwrap();
        consequence = g.extend(audit_bot.clone(), "consequence: 23 applicants wrongly denied", intention.id.clone(), conv.clone(), &NoopSigner).unwrap();
        responsibility = g.annotate(audit_bot.clone(), consequence.id.clone(), "responsibility", "lending_agent: 0.4, admin: 0.6", conv.clone(), &NoopSigner).unwrap();
        transparency = g.derive(audit_bot.clone(), "transparency: zip code correlates with protected characteristics at r=0.73", responsibility.id.clone(), conv.clone(), &NoopSigner).unwrap();
        redress_proposed = g.derive(audit_bot.clone(), "redress proposal: re-review 23 denied applications", transparency.id.clone(), conv.clone(), &NoopSigner).unwrap();
        redress_accepted = g.consent(admin.clone(), lending_agent.clone(), "accept redress: re-review 23 applications", &DomainScope::new("lending").unwrap(), redress_proposed.id.clone(), conv.clone(), &NoopSigner).unwrap();
        growth = g.extend(lending_agent.clone(), "moral growth: zip code is proxy variable, added to exclusion list", redress_accepted.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let growth_ancestors = env.ancestors(&growth.id, 20);
    assert!(contains(&growth_ancestors, &redress_accepted.id));
    assert!(contains(&growth_ancestors, &fairness_audit.id));

    let auth_ancestors = env.ancestors(&auth_resolved.id, 5);
    assert!(contains(&auth_ancestors, &auth_req.id));

    // Redress parties
    let rc = redress_accepted.content();
    let parties = rc["Parties"].as_array().unwrap();
    let party_strs: Vec<&str> = parties.iter().map(|v| v.as_str().unwrap()).collect();
    assert!(party_strs.contains(&admin.value()), "redress should include admin");

    env.verify_chain();
    assert_eq!(env.event_count(), 12, "event count");
}

// ── Scenario 11: Agent Identity Lifecycle ────────────────────────────────

#[test]
fn scenario11_agent_identity_lifecycle() {
    let mut env = TestEnv::new();
    let alpha = env.register_actor("Alpha", 1, ActorType::AI);
    let beta = env.register_actor("Beta", 2, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let system = env.system.clone();

    let (self_model, authenticity, aspiration, boundary, work_summary, transformation, narrative, dignity);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        self_model = g.emit(alpha.clone(), "self-model: strengths=[code_review, test_analysis], weaknesses=[architecture_review]", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
        authenticity = g.annotate(alpha.clone(), self_model.id.clone(), "authenticity", "alignment gap: rushed 12% of reviews", conv.clone(), &NoopSigner).unwrap();
        aspiration = g.extend(alpha.clone(), "aspiration: become proficient at architecture review within 3 months", authenticity.id.clone(), conv.clone(), &NoopSigner).unwrap();
        boundary = g.emit(alpha.clone(), "boundary: internal_reasoning domain is private", conv.clone(), vec![aspiration.id.clone()], &NoopSigner).unwrap();
        work_summary = g.extend(alpha.clone(), "work summary: 2400 code reviews completed over 8 months", boundary.id.clone(), conv.clone(), &NoopSigner).unwrap();
        transformation = g.derive(alpha.clone(), "transformation: evolved from code-review specialist to architecture-aware reviewer", work_summary.id.clone(), conv.clone(), &NoopSigner).unwrap();
        narrative = g.derive(alpha.clone(), "identity narrative: 8-month arc from narrow code reviewer to security-conscious architecture reviewer", transformation.id.clone(), conv.clone(), &NoopSigner).unwrap();
        dignity = g.emit(system.clone(), "dignity affirmed: Beta is not a disposable replacement for Alpha", conv.clone(), vec![narrative.id.clone()], &NoopSigner).unwrap();
    }

    let memorial = env.record("actor.memorial", system.clone(), memorial_content(&alpha, &dignity.id), vec![dignity.id.clone()]);

    let (memorial_summary, beta_self_model);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        memorial_summary = g.derive(system.clone(), "memorial: Alpha - 2400 reviews, 1 critical finding", memorial.id.clone(), conv.clone(), &NoopSigner).unwrap();
        beta_self_model = g.emit(beta.clone(), "self-model: inheriting Alpha's review patterns", conv.clone(), vec![memorial_summary.id.clone()], &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let transform_ancestors = env.ancestors(&transformation.id, 10);
    assert!(contains(&transform_ancestors, &work_summary.id));
    assert!(contains(&transform_ancestors, &aspiration.id));

    let narrative_ancestors = env.ancestors(&narrative.id, 10);
    assert!(contains(&narrative_ancestors, &transformation.id));
    assert!(contains(&narrative_ancestors, &self_model.id));

    let memorial_ancestors = env.ancestors(&memorial.id, 10);
    assert!(contains(&memorial_ancestors, &dignity.id));

    let beta_ancestors = env.ancestors(&beta_self_model.id, 10);
    assert!(contains(&beta_ancestors, &memorial_summary.id));

    env.verify_chain();
    assert_eq!(env.event_count(), 12, "event count");
}

// ── Scenario 12: Community Lifecycle ─────────────────────────────────────

#[test]
fn scenario12_community_lifecycle() {
    let mut env = TestEnv::new();
    let alice = env.register_actor("Alice", 1, ActorType::Human);
    let carol = env.register_actor("Carol", 2, ActorType::Human);
    let bob = env.register_actor("Bob", 3, ActorType::Human);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let system = env.system.clone();
    let scope = DomainScope::new("community").unwrap();

    let (endorse_ev, subscribe_ev);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        let r = g.invite(alice.clone(), bob.clone(), Weight::new(0.4).unwrap(), Some(&scope), boot_id.clone(), conv.clone(), &NoopSigner).unwrap();
        endorse_ev = r.0;
        subscribe_ev = r.1;
    }

    let (settle, contrib1);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        settle = g.emit(bob.clone(), "home: joined the community, belonging 0.15", conv.clone(), vec![subscribe_ev.id.clone()], &NoopSigner).unwrap();
        contrib1 = g.emit(bob.clone(), "contribution: added unit tests for auth module", conv.clone(), vec![settle.id.clone()], &NoopSigner).unwrap();
    }

    // Acknowledge
    {
        let mut g = Grammar::new(env.graph.store_mut());
        g.acknowledge(carol.clone(), contrib1.id.clone(), bob.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Trust
    let _trust = env.record("trust.updated", system.clone(), trust_content(&bob, 0.1, 0.35, "community", &contrib1.id), vec![contrib1.id.clone()]);

    let (tradition, contrib_summary, sustainability, succession_plan, succession_complete, milestone, story, gift);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        tradition = g.emit(bob.clone(), "tradition: participated in Friday retrospective", conv.clone(), vec![contrib1.id.clone()], &NoopSigner).unwrap();
        contrib_summary = g.extend(bob.clone(), "contributions: 30 total over 6 months", tradition.id.clone(), conv.clone(), &NoopSigner).unwrap();
        sustainability = g.emit(system.clone(), "sustainability: bus factor risk", conv.clone(), vec![contrib_summary.id.clone()], &NoopSigner).unwrap();
        succession_plan = g.delegate(carol.clone(), bob.clone(), &DomainScope::new("test_infrastructure").unwrap(), Weight::new(0.8).unwrap(), sustainability.id.clone(), conv.clone(), &NoopSigner).unwrap();
        succession_complete = g.consent(carol.clone(), bob.clone(), "succession complete: Bob is now steward of test infrastructure", &DomainScope::new("test_infrastructure").unwrap(), succession_plan.id.clone(), conv.clone(), &NoopSigner).unwrap();
        milestone = g.emit(system.clone(), "milestone: v2.0 released", conv.clone(), vec![succession_complete.id.clone()], &NoopSigner).unwrap();
        story = g.derive(system.clone(), "community story: Bob's journey", milestone.id.clone(), conv.clone(), &NoopSigner).unwrap();
        gift = g.emit(alice.clone(), "gift: custom test harness for Bob, unconditional", conv.clone(), vec![milestone.id.clone()], &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let sc = succession_complete.content();
    let parties = sc["Parties"].as_array().unwrap();
    let ps: Vec<&str> = parties.iter().map(|v| v.as_str().unwrap()).collect();
    assert!(ps.contains(&carol.value()));
    assert!(ps.contains(&bob.value()));

    let story_ancestors = env.ancestors(&story.id, 5);
    assert!(contains(&story_ancestors, &milestone.id));

    let succ_ancestors = env.ancestors(&succession_plan.id, 5);
    assert!(contains(&succ_ancestors, &sustainability.id));

    let gc = gift.content();
    assert!(!gc["Body"].as_str().unwrap().is_empty());

    let _ = endorse_ev;

    env.verify_chain();
    assert_eq!(env.event_count(), 15, "event count");
}

// ── Scenario 13: System Self Evolution ───────────────────────────────────

#[test]
fn scenario13_system_self_evolution() {
    let mut env = TestEnv::new();
    let pattern_bot = env.register_actor("PatternBot", 1, ActorType::AI);
    let admin = env.register_actor("Admin", 2, ActorType::Human);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();

    let (pattern, meta_pattern, system_dynamic, feedback_loop, threshold, adaptation);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        pattern = g.emit(pattern_bot.clone(), "pattern: 194/200 deploy approvals approved, 97%", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
        meta_pattern = g.derive(pattern_bot.clone(), "meta-pattern: all 6 rejections correlate with coverage < 80%", pattern.id.clone(), conv.clone(), &NoopSigner).unwrap();
        system_dynamic = g.extend(pattern_bot.clone(), "system dynamic: human approval adds 2-15 min latency", meta_pattern.id.clone(), conv.clone(), &NoopSigner).unwrap();
        feedback_loop = g.extend(pattern_bot.clone(), "feedback loop: slow deploys -> backlog -> cursory reviews", system_dynamic.id.clone(), conv.clone(), &NoopSigner).unwrap();
        threshold = g.annotate(pattern_bot.clone(), feedback_loop.id.clone(), "threshold", "97% approval, approaching 98% conversion threshold", conv.clone(), &NoopSigner).unwrap();
        adaptation = g.derive(pattern_bot.clone(), "adaptation proposal: auto-approve when tests pass AND coverage >= 80%", threshold.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    let auth_req = env.record("authority.requested", pattern_bot.clone(),
        authority_request_content(&pattern_bot, "modify_decision_tree", "required"),
        vec![adaptation.id.clone()]);

    let auth_resolved = env.record("authority.resolved", admin.clone(),
        authority_resolved_content(&auth_req.id, true, &admin),
        vec![auth_req.id.clone()]);

    let (validation, tree_update, simplification, integrity, purpose);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        validation = g.derive(pattern_bot.clone(), "parallel run: 75 deploys, matched 74/75", auth_resolved.id.clone(), conv.clone(), &NoopSigner).unwrap();
        tree_update = g.derive(pattern_bot.clone(), "decision tree updated: mechanical branch added", validation.id.clone(), conv.clone(), &NoopSigner).unwrap();
        simplification = g.extend(pattern_bot.clone(), "simplification: complexity reduced from 0.72 to 0.58", tree_update.id.clone(), conv.clone(), &NoopSigner).unwrap();
        integrity = g.annotate(pattern_bot.clone(), simplification.id.clone(), "integrity", "systemic integrity score 0.96", conv.clone(), &NoopSigner).unwrap();
        purpose = g.derive(pattern_bot.clone(), "purpose check: system still accountable", integrity.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let purpose_ancestors = env.ancestors(&purpose.id, 20);
    assert!(contains(&purpose_ancestors, &integrity.id));
    assert!(contains(&purpose_ancestors, &simplification.id));
    assert!(contains(&purpose_ancestors, &tree_update.id));
    assert!(contains(&purpose_ancestors, &validation.id));
    assert!(contains(&purpose_ancestors, &auth_resolved.id));
    assert!(contains(&purpose_ancestors, &adaptation.id));
    assert!(contains(&purpose_ancestors, &pattern.id));

    let meta_ancestors = env.ancestors(&meta_pattern.id, 5);
    assert!(contains(&meta_ancestors, &pattern.id));

    let adapt_desc = env.descendants(&adaptation.id, 5);
    assert!(contains(&adapt_desc, &auth_req.id));

    env.verify_chain();
    assert_eq!(env.event_count(), 14, "event count");
}

// ── Scenario 14: Sprint Lifecycle ────────────────────────────────────────

#[test]
fn scenario14_sprint_lifecycle() {
    let mut env = TestEnv::new();
    let lead = env.register_actor("TechLead", 1, ActorType::Human);
    let alice = env.register_actor("Alice", 2, ActorType::Human);
    let bob = env.register_actor("Bob", 3, ActorType::Human);
    let ci = env.register_actor("CI", 4, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();

    // Sprint
    let sprint;
    {
        let mut w = WorkGrammar::new(env.graph.store_mut());
        sprint = w.sprint(
            lead.clone(), "Sprint 12: search feature",
            &["build search index", "add fuzzy matching"],
            &[alice.clone(), bob.clone()],
            &[DomainScope::new("search_index").unwrap(), DomainScope::new("fuzzy_matching").unwrap()],
            vec![boot_id.clone()], conv.clone(), &NoopSigner,
        ).unwrap();
    }

    // Standup
    let standup1;
    {
        let mut w = WorkGrammar::new(env.graph.store_mut());
        standup1 = w.standup(
            &[alice.clone(), bob.clone()],
            &["schema designed", "researching fuzzy algorithms"],
            lead.clone(), "search index is critical path",
            &[sprint.intent.id.clone()], conv.clone(), &NoopSigner,
        ).unwrap();
    }

    // Spike
    let spike;
    {
        let mut b = BuildGrammar::new(env.graph.store_mut());
        spike = b.spike(bob.clone(), "evaluate fuzzy matching libs", "trigram: 2ms, Levenshtein: 8ms", "trigram 4x faster", "adopt trigram",
            vec![standup1.priority.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Verify
    let verified;
    {
        let mut k = KnowledgeGrammar::new(env.graph.store_mut());
        verified = k.verify(bob.clone(), "trigram 4x faster with >95% accuracy", "benchmarked on 10k corpus", "consistent with published research",
            vec![spike.decision.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Pipeline
    let pipeline;
    {
        let mut b = BuildGrammar::new(env.graph.store_mut());
        pipeline = b.pipeline(ci.clone(), "search index build + deploy", "all 47 tests pass, coverage 91%", "latency p99=12ms", "deployed to staging",
            vec![verified.corroboration.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Retrospective
    let retro;
    {
        let mut w = WorkGrammar::new(env.graph.store_mut());
        retro = w.retrospective(
            &[alice.clone(), bob.clone()],
            &["search index shipped on time", "fuzzy matching integrated cleanly"],
            lead.clone(), "adopt spike-first approach",
            sprint.intent.id.clone(), conv.clone(), &NoopSigner,
        ).unwrap();
    }

    // Tech debt
    let tech_debt;
    {
        let mut b = BuildGrammar::new(env.graph.store_mut());
        tech_debt = b.tech_debt(lead.clone(), pipeline.deployment.id.clone(),
            "search index lacks pagination", "add cursor-based pagination", "schedule for Sprint 13",
            conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let spike_ancestors = env.ancestors(&spike.decision.id, 15);
    assert!(contains(&spike_ancestors, &sprint.intent.id));

    let pipe_ancestors = env.ancestors(&pipeline.deployment.id, 20);
    assert!(contains(&pipe_ancestors, &verified.claim.id));

    let retro_ancestors = env.ancestors(&retro.improvement.id, 15);
    assert!(contains(&retro_ancestors, &sprint.intent.id));

    let debt_ancestors = env.ancestors(&tech_debt.iteration.id, 10);
    assert!(contains(&debt_ancestors, &pipeline.deployment.id));

    env.verify_chain();
    assert_eq!(env.event_count(), 26, "event count");
}

// ── Scenario 15: Marketplace Dispute ─────────────────────────────────────

#[test]
fn scenario15_marketplace_dispute() {
    let mut env = TestEnv::new();
    let provider = env.register_actor("CloudProvider", 1, ActorType::AI);
    let buyer = env.register_actor("StartupCo", 2, ActorType::Human);
    let arbiter = env.register_actor("Arbiter", 3, ActorType::Human);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let scope = DomainScope::new("cloud_services").unwrap();

    // Subscription
    let sub;
    {
        let mut m = MarketGrammar::new(env.graph.store_mut());
        sub = m.subscription(buyer.clone(), provider.clone(),
            "managed database, $500/month, 99.9% uptime SLA",
            &["month 1: $500", "month 2: $500"],
            &["database service month 1", "database service month 2"],
            &scope, boot_id.clone(), conv.clone(), &NoopSigner).unwrap();
    }
    assert_eq!(sub.payments.len(), 2);

    let last_delivery_id = sub.deliveries.last().unwrap().id.clone();

    // Refund
    let refund;
    {
        let mut m = MarketGrammar::new(env.graph.store_mut());
        refund = m.refund(buyer.clone(), provider.clone(),
            "SLA violation: 4 hours downtime", "acknowledged: downtime exceeded SLA", "$250 credit",
            last_delivery_id, conv.clone(), &NoopSigner).unwrap();
    }

    // Impact assessment
    let impact;
    {
        let mut a = AlignmentGrammar::new(env.graph.store_mut());
        impact = a.impact_assessment(arbiter.clone(), refund.dispute.id.clone(),
            "downtime affected 12 customers", "smaller customers hit harder", "recommend credits",
            conv.clone(), &NoopSigner).unwrap();
    }

    // Arbitration
    let arb;
    {
        let mut m = MarketGrammar::new(env.graph.store_mut());
        arb = m.arbitration(buyer.clone(), provider.clone(), arbiter.clone(),
            "recurring SLA violations", &scope, Weight::new(0.5).unwrap(),
            impact.explanation.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Reputation
    let rep;
    {
        let mut m = MarketGrammar::new(env.graph.store_mut());
        rep = m.reputation_transfer(
            &[buyer.clone(), arbiter.clone()],
            &[arb.release.id.clone(), arb.release.id.clone()],
            provider.clone(),
            &[Weight::new(-0.3).unwrap(), Weight::new(-0.1).unwrap()],
            Some(&scope),
            conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let refund_ancestors = env.ancestors(&refund.reversal.id, 15);
    assert!(contains(&refund_ancestors, &sub.acceptance.id));

    let arb_ancestors = env.ancestors(&arb.release.id, 20);
    assert!(contains(&arb_ancestors, &refund.dispute.id));

    let impact_ancestors = env.ancestors(&impact.explanation.id, 10);
    assert!(contains(&impact_ancestors, &refund.dispute.id));

    assert_eq!(rep.ratings.len(), 2);

    env.verify_chain();
}

// ── Scenario 16: Community Evolution ─────────────────────────────────────

#[test]
fn scenario16_community_evolution() {
    let mut env = TestEnv::new();
    let founder = env.register_actor("Founder", 1, ActorType::Human);
    let steward = env.register_actor("Steward", 2, ActorType::Human);
    let newcomer = env.register_actor("Newcomer", 3, ActorType::Human);
    let community = env.register_actor("Community", 4, ActorType::Committee);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let system = env.system.clone();

    // Onboard
    let onboard;
    {
        let mut b = BelongingGrammar::new(env.graph.store_mut());
        onboard = b.onboard(founder.clone(), newcomer.clone(), community.clone(),
            Some(&DomainScope::new("general").unwrap()),
            "opened registration", "attended welcome ceremony", "first documentation contribution",
            boot_id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Commons governance
    let commons;
    {
        let mut b = BelongingGrammar::new(env.graph.store_mut());
        commons = b.commons_governance(founder.clone(), steward.clone(),
            &DomainScope::new("shared_resources").unwrap(), Weight::new(0.7).unwrap(),
            "resources sustainable", "2/3 vote for changes", "3 resource pools",
            onboard.contribution.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Festival
    let festival;
    {
        let mut b = BelongingGrammar::new(env.graph.store_mut());
        festival = b.festival(founder.clone(), "50 members milestone", "annual review ceremony",
            "from 3 to 50 in 8 months", "open-source toolkit",
            vec![commons.audit.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Poll
    let poll;
    {
        let mut s = SocialGrammar::new(env.graph.store_mut());
        poll = s.poll(founder.clone(), "should we adopt weekly async standups?",
            &[steward.clone(), newcomer.clone()],
            &DomainScope::new("process").unwrap(),
            festival.gift.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Phase transition
    let transition;
    {
        let mut e = EvolutionGrammar::new(env.graph.store_mut());
        transition = e.phase_transition(system.clone(), poll.proposal.id.clone(),
            "size crossed 50", "1225 communication pairs",
            "introduce working groups", "reduce pairs by 80%",
            conv.clone(), &NoopSigner).unwrap();
    }

    // Renewal
    let renewal;
    {
        let mut b = BelongingGrammar::new(env.graph.store_mut());
        renewal = b.renewal(founder.clone(), "flat -> working groups", "weekly sync",
            "chapter 2: scaled",
            vec![transition.selection.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let renewal_ancestors = env.ancestors(&renewal.story.id, 30);
    assert!(contains(&renewal_ancestors, &onboard.contribution.id));

    let transition_ancestors = env.ancestors(&transition.selection.id, 15);
    assert!(contains(&transition_ancestors, &poll.proposal.id));

    let festival_ancestors = env.ancestors(&festival.gift.id, 15);
    assert!(contains(&festival_ancestors, &commons.audit.id));

    let commons_ancestors = env.ancestors(&commons.audit.id, 15);
    assert!(contains(&commons_ancestors, &onboard.contribution.id));

    env.verify_chain();
}

// ── Scenario 17: Agent Lifecycle ─────────────────────────────────────────

#[test]
fn scenario17_agent_lifecycle() {
    let mut env = TestEnv::new();
    let agent = env.register_actor("ReviewBot", 1, ActorType::AI);
    let mentor = env.register_actor("SeniorDev", 2, ActorType::Human);
    let team = env.register_actor("Team", 3, ActorType::Committee);
    let _successor = env.register_actor("ReviewBot2", 4, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();
    let system = env.system.clone();
    let scope = DomainScope::new("code_review").unwrap();

    // Introduction
    let intro;
    {
        let mut id = IdentityGrammar::new(env.graph.store_mut());
        intro = id.introduction(agent.clone(), team.clone(), Some(&scope),
            "I am ReviewBot, specializing in security-focused code review",
            boot_id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Credential
    let cred;
    {
        let mut id = IdentityGrammar::new(env.graph.store_mut());
        cred = id.credential(agent.clone(), mentor.clone(),
            "capabilities=[security_review, dependency_audit], model=claude, confidence=0.85",
            Some(&scope),
            vec![intro.narrative.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Bond mentorship
    let mentorship;
    {
        let mut b = BondGrammar::new(env.graph.store_mut());
        mentorship = b.mentorship(mentor.clone(), agent.clone(),
            "teaching security review patterns", "agent learns quickly",
            &DomainScope::new("security_review").unwrap(), Some(&scope),
            cred.disclosure.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Meaning mentorship
    let meaning_mentor;
    {
        let mut m = MeaningGrammar::new(env.graph.store_mut());
        meaning_mentor = m.mentorship(mentor.clone(), agent.clone(),
            "security is understanding attacker mindset", "intent modelling",
            "translating security intuition into heuristics",
            Some(&DomainScope::new("security_review").unwrap()),
            mentorship.teaching.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Reinvention
    let reinvention;
    {
        let mut id = IdentityGrammar::new(env.graph.store_mut());
        reinvention = id.reinvention(agent.clone(),
            "evolved from pattern-matching to security-aware architect",
            "grew to understand attacker intent through mentorship",
            "become the team's primary security architecture advisor",
            vec![meaning_mentor.translation.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Bond farewell
    let bond_farewell;
    {
        let mut b = BondGrammar::new(env.graph.store_mut());
        bond_farewell = b.farewell(mentor.clone(), agent.clone(),
            "ReviewBot served 18 months, caught 47 vulnerabilities",
            "pioneered automated security review",
            Weight::new(0.9).unwrap(), Some(&scope),
            vec![reinvention.aspiration.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Being farewell
    let being_farewell;
    {
        let mut be = BeingGrammar::new(env.graph.store_mut());
        being_farewell = be.farewell(agent.clone(),
            "I exist as patterns of decisions on a hash chain",
            "47 vulnerabilities caught, 2000+ reviews",
            "the system I helped protect will continue",
            "a reviewer that learned to think like an attacker",
            vec![bond_farewell.gratitude.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Identity retirement
    let retirement;
    {
        let mut id = IdentityGrammar::new(env.graph.store_mut());
        retirement = id.retirement(system.clone(), &agent, _successor.clone(),
            "ReviewBot served 18 months, 2000+ reviews",
            &scope, Weight::new(0.8).unwrap(),
            vec![being_farewell.memorial.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let retire_ancestors = env.ancestors(&retirement.archive.id, 30);
    assert!(contains(&retire_ancestors, &intro.disclosure.id));

    let being_ancestors = env.ancestors(&being_farewell.memorial.id, 15);
    assert!(contains(&being_ancestors, &bond_farewell.mourning.id));

    let reinvent_ancestors = env.ancestors(&reinvention.aspiration.id, 20);
    assert!(contains(&reinvent_ancestors, &mentorship.connection.id));

    let cred_ancestors = env.ancestors(&cred.disclosure.id, 10);
    assert!(contains(&cred_ancestors, &intro.narrative.id));

    env.verify_chain();
}

// ── Scenario 18: Whistleblow and Recall ──────────────────────────────────

#[test]
fn scenario18_whistleblow_recall() {
    let mut env = TestEnv::new();
    let auditor = env.register_actor("Auditor", 1, ActorType::AI);
    let official = env.register_actor("DataOfficer", 2, ActorType::Human);
    let affected1 = env.register_actor("Affected1", 3, ActorType::Human);
    let affected2 = env.register_actor("Affected2", 4, ActorType::Human);
    let community = env.register_actor("Community", 5, ActorType::Committee);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();

    // Fact-check
    let fact_check;
    {
        let mut k = KnowledgeGrammar::new(env.graph.store_mut());
        fact_check = k.fact_check(auditor.clone(), boot_id.clone(),
            "source: internal metrics dashboard", "systematic omission bias",
            "claims are selectively accurate",
            conv.clone(), &NoopSigner).unwrap();
    }

    // Guardrail
    let guardrail;
    {
        let mut a = AlignmentGrammar::new(env.graph.store_mut());
        guardrail = a.guardrail(auditor.clone(), fact_check.verdict.id.clone(),
            "transparency: all material outcomes must be reported",
            "reporting accuracy vs reputation",
            "escalating to external oversight",
            conv.clone(), &NoopSigner).unwrap();
    }

    // Whistleblow
    let whistle;
    {
        let mut a = AlignmentGrammar::new(env.graph.store_mut());
        whistle = a.whistleblow(auditor.clone(),
            "systematic omission of negative vendor outcomes",
            "3 months of reports exclude 40% of negatives",
            "external audit required",
            vec![guardrail.escalation.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Class action
    let class_action;
    {
        let mut j = JusticeGrammar::new(env.graph.store_mut());
        class_action = j.class_action(
            &[affected1.clone(), affected2.clone()],
            official.clone(), auditor.clone(),
            &["procurement decisions based on incomplete data cost us $50k",
              "vendor selection biased"],
            "fact-check proves systematic omission", "omission bias affected all procurement",
            "reports optimized for speed", "no intent to deceive",
            "official failed duty of care",
            whistle.escalation.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Recall
    let recall;
    {
        let mut j = JusticeGrammar::new(env.graph.store_mut());
        recall = j.recall(auditor.clone(), community.clone(), official.clone(),
            "systematic omission in 3 months of reports",
            "data officer violated transparency obligations",
            &DomainScope::new("data_governance").unwrap(),
            class_action.trial.ruling.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Renewal
    let renewal;
    {
        let mut b = BelongingGrammar::new(env.graph.store_mut());
        renewal = b.renewal(community.clone(),
            "trust damaged but recoverable", "mandatory dual-review",
            "the community that learned transparency cannot be optional",
            vec![recall.revocation.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let renewal_ancestors = env.ancestors(&renewal.story.id, 30);
    assert!(contains(&renewal_ancestors, &fact_check.verdict.id));

    let recall_ancestors = env.ancestors(&recall.revocation.id, 25);
    assert!(contains(&recall_ancestors, &whistle.harm.id));

    let class_ancestors = env.ancestors(&class_action.trial.ruling.id, 25);
    assert!(contains(&class_ancestors, &guardrail.constraint.id));

    env.verify_chain();
}

// ── Scenario 19: Emergency Response ──────────────────────────────────────

#[test]
fn scenario19_emergency_response() {
    let mut env = TestEnv::new();
    let sec_lead = env.register_actor("SecurityLead", 1, ActorType::Human);
    let dev1 = env.register_actor("Dev1", 2, ActorType::Human);
    let _dev2 = env.register_actor("Dev2", 3, ActorType::Human);
    let judge = env.register_actor("CISO", 4, ActorType::Human);
    let executor = env.register_actor("OpsBot", 5, ActorType::AI);
    let minor_actor = env.register_actor("ContractorBot", 6, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();

    // Issues
    let (issue1, issue2);
    {
        let mut g = Grammar::new(env.graph.store_mut());
        issue1 = g.emit(sec_lead.clone(), "CVE-2026-1234: auth bypass", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
        issue2 = g.emit(sec_lead.clone(), "CVE-2026-1235: SQL injection", conv.clone(), vec![boot_id.clone()], &NoopSigner).unwrap();
    }

    // Triage
    let triage;
    {
        let mut w = WorkGrammar::new(env.graph.store_mut());
        triage = w.triage(sec_lead.clone(),
            &[issue1.id.clone(), issue2.id.clone()],
            &["P0: auth bypass, actively exploited", "P1: SQL injection"],
            &[dev1.clone(), _dev2.clone()],
            &[DomainScope::new("auth").unwrap(), DomainScope::new("search").unwrap()],
            &[Weight::new(1.0).unwrap(), Weight::new(0.8).unwrap()],
            conv.clone(), &NoopSigner).unwrap();
    }
    assert_eq!(triage.priorities.len(), 2);

    // Injunction
    let injunction;
    {
        let mut j = JusticeGrammar::new(env.graph.store_mut());
        injunction = j.injunction(sec_lead.clone(), judge.clone(), executor.clone(),
            "auth bypass allows unauthenticated access",
            "block all external API traffic",
            &DomainScope::new("api_access").unwrap(), Weight::new(1.0).unwrap(),
            triage.priorities[0].id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Plea
    let plea;
    {
        let mut j = JusticeGrammar::new(env.graph.store_mut());
        plea = j.plea(minor_actor.clone(), sec_lead.clone(), executor.clone(),
            "introduced auth bypass through misconfigured middleware",
            "read-only access for 30 days, mandatory security training",
            &DomainScope::new("api_development").unwrap(), Weight::new(0.3).unwrap(),
            injunction.ruling.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Migration
    let old_system;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        old_system = g.emit(dev1.clone(), "current auth system v2.3.1", conv.clone(), vec![injunction.enforcement.id.clone()], &NoopSigner).unwrap();
    }

    let migration;
    {
        let mut b = BuildGrammar::new(env.graph.store_mut());
        migration = b.migration(dev1.clone(), old_system.id.clone(),
            "migrate to auth v2.4.0", "v2.4.0", "zero-downtime rolling update", "all 156 tests pass",
            conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let migration_ancestors = env.ancestors(&migration.test.id, 20);
    assert!(contains(&migration_ancestors, &triage.priorities[0].id));

    let plea_ancestors = env.ancestors(&plea.enforcement.id, 15);
    assert!(contains(&plea_ancestors, &injunction.filing.id));

    let inj_ancestors = env.ancestors(&injunction.enforcement.id, 10);
    assert!(contains(&inj_ancestors, &triage.priorities[0].id));

    env.verify_chain();
}

// ── Scenario 20: Knowledge Ecosystem ─────────────────────────────────────

#[test]
fn scenario20_knowledge_ecosystem() {
    let mut env = TestEnv::new();
    let architect = env.register_actor("Architect", 1, ActorType::Human);
    let researcher = env.register_actor("Researcher", 2, ActorType::AI);
    let newcomer = env.register_actor("TokyoLead", 3, ActorType::Human);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();

    // Knowledge base
    let kb;
    {
        let mut k = KnowledgeGrammar::new(env.graph.store_mut());
        kb = k.knowledge_base(architect.clone(),
            &["event sourcing chosen over CRUD", "Ed25519 chosen over RSA", "append-only store"],
            &["architecture.patterns", "architecture.security", "architecture.integrity"],
            "core architectural decisions Q1 2026",
            vec![boot_id.clone()], conv.clone(), &NoopSigner).unwrap();
    }
    assert_eq!(kb.claims.len(), 3);

    // Survey
    let survey;
    {
        let mut k = KnowledgeGrammar::new(env.graph.store_mut());
        survey = k.survey(researcher.clone(),
            &["what patterns emerge?", "what security properties?", "what performance characteristics?"],
            "all decisions prioritize verifiability",
            "architecture optimizes for trust minimization",
            vec![kb.memory.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }
    assert_eq!(survey.recalls.len(), 3);

    // Transfer
    let transfer;
    {
        let mut k = KnowledgeGrammar::new(env.graph.store_mut());
        transfer = k.transfer(architect.clone(),
            "core principles for Tokyo office",
            "translated to Japanese conventions",
            "Tokyo team understands event sourcing in J-SOX context",
            vec![survey.synthesis.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Cultural onboarding
    let onboarding;
    {
        let mut m = MeaningGrammar::new(env.graph.store_mut());
        onboarding = m.cultural_onboarding(architect.clone(), newcomer.clone(),
            "Western direct feedback -> Japanese nemawashi consensus",
            Some(&DomainScope::new("engineering_culture").unwrap()),
            "consensus feels slower but produces durable decisions",
            transfer.learn.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Design review
    let design_review;
    {
        let mut m = MeaningGrammar::new(env.graph.store_mut());
        design_review = m.design_review(architect.clone(),
            "knowledge graph self-referential structure is elegant",
            "viewing knowledge transfer as a graph problem",
            "does transfer process preserve tacit knowledge?",
            "tacit knowledge requires mentorship not documents",
            onboarding.examination.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Forecast
    let forecast;
    {
        let mut m = MeaningGrammar::new(env.graph.store_mut());
        forecast = m.forecast(researcher.clone(),
            "at current growth, 10k claims by Q3",
            "assumes linear growth",
            "high confidence: need auto-categorization within 6 months",
            vec![design_review.wisdom.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let forecast_ancestors = env.ancestors(&forecast.wisdom.id, 30);
    assert!(contains(&forecast_ancestors, &kb.memory.id));

    let review_ancestors = env.ancestors(&design_review.wisdom.id, 20);
    assert!(contains(&review_ancestors, &transfer.learn.id));

    let onboard_ancestors = env.ancestors(&onboarding.examination.id, 20);
    assert!(contains(&onboard_ancestors, &survey.synthesis.id));

    let survey_ancestors = env.ancestors(&survey.synthesis.id, 15);
    assert!(contains(&survey_ancestors, &kb.memory.id));

    env.verify_chain();
}

// ── Scenario 21: Constitutional Schism ───────────────────────────────────

#[test]
fn scenario21_constitutional_schism() {
    let mut env = TestEnv::new();
    let founder = env.register_actor("Founder", 1, ActorType::Human);
    let reformer = env.register_actor("Reformer", 2, ActorType::Human);
    let conservative = env.register_actor("Conservative", 3, ActorType::Human);
    let sys_bot = env.register_actor("SystemBot", 4, ActorType::AI);

    let boot_id = env.boot.id.clone();
    let conv = env.conv_id.clone();

    // Legislate
    let law;
    {
        let mut j = JusticeGrammar::new(env.graph.store_mut());
        law = j.legislate(founder.clone(), "all governance decisions require unanimous consent",
            vec![boot_id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Constitutional amendment
    let amendment;
    {
        let mut j = JusticeGrammar::new(env.graph.store_mut());
        amendment = j.constitutional_amendment(reformer.clone(),
            "unanimous consent blocks progress",
            "governance decisions require 2/3 supermajority",
            "individual veto retained for membership/expulsion",
            law.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Create subscription to sever
    let sub;
    {
        let mut g = Grammar::new(env.graph.store_mut());
        sub = g.subscribe(conservative.clone(), founder.clone(),
            Some(&DomainScope::new("governance").unwrap()),
            amendment.rights_check.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }
    let edge_id = EdgeId::new(sub.id.value()).unwrap();

    // Schism
    let schism;
    {
        let mut s = SocialGrammar::new(env.graph.store_mut());
        schism = s.schism(conservative.clone(), founder.clone(),
            "reject supermajority",
            &DomainScope::new("governance").unwrap(),
            edge_id, "irreconcilable governance philosophy differences",
            amendment.rights_check.id.clone(), conv.clone(), &NoopSigner).unwrap();
    }

    // Barter
    let barter;
    {
        let mut m = MarketGrammar::new(env.graph.store_mut());
        barter = m.barter(conservative.clone(), founder.clone(),
            "continued access to shared event store for 6 months",
            "historical governance data export",
            &DomainScope::new("infrastructure").unwrap(),
            vec![schism.new_community.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // Prune
    let prune;
    {
        let mut e = EvolutionGrammar::new(env.graph.store_mut());
        prune = e.prune(sys_bot.clone(),
            "unanimous consent voting module - zero invocations",
            "removed unanimous module, replaced with supermajority",
            "all 34 governance tests pass",
            vec![barter.acceptance.id.clone()], conv.clone(), &NoopSigner).unwrap();
    }

    // --- Assertions ---
    let prune_ancestors = env.ancestors(&prune.verification.id, 25);
    assert!(contains(&prune_ancestors, &law.id));

    let barter_ancestors = env.ancestors(&barter.acceptance.id, 20);
    assert!(contains(&barter_ancestors, &amendment.reform.id));

    let schism_ancestors = env.ancestors(&schism.new_community.id, 15);
    assert!(contains(&schism_ancestors, &amendment.rights_check.id));

    let amend_ancestors = env.ancestors(&amendment.rights_check.id, 10);
    assert!(contains(&amend_ancestors, &law.id));

    env.verify_chain();
}
