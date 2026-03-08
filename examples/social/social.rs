// Social Grammar example -- demonstrates the 15 social grammar operations
// and 4 named functions on an event graph.
//
// To run, add this as an example in rust/Cargo.toml:
//
//   [[example]]
//   name = "social"
//   path = "../examples/social/social.rs"
//
// Then:  cargo run --example social
//
// Or compile standalone (from the repo root):
//   rustc --edition 2024 -L target/debug/deps --extern eventgraph=target/debug/libeventgraph.rlib \
//         examples/social/social.rs -o social && ./social

use eventgraph::actor::InMemoryActorStore;
use eventgraph::event::{Event, NoopSigner};
use eventgraph::grammar::Grammar;
use eventgraph::graph::Graph;
use eventgraph::store::{InMemoryStore, Store};
use eventgraph::types::{ActorId, ConversationId, DomainScope, Weight};

fn show(op: &str, ev: &Event) {
    println!("{:<20} -> {} ({}...)", op, ev.event_type, &ev.id.value()[..13]);
}

fn main() {
    // --- Setup ---
    let store = InMemoryStore::new();
    let actor_store = InMemoryActorStore::new();
    let mut g = Graph::new(store, actor_store);
    g.start().expect("failed to start graph");

    let system_actor = ActorId::new("actor_system").unwrap();
    let boot = g
        .bootstrap(system_actor, Some(&NoopSigner))
        .expect("failed to bootstrap");

    let signer = NoopSigner;
    let conv = ConversationId::new("conv_social_demo").unwrap();
    let alice = ActorId::new("actor_alice").unwrap();
    let bob = ActorId::new("actor_bob").unwrap();

    // Get a mutable reference to the store for Grammar.
    let store_mut = g.store_mut();
    let mut gr = Grammar::new(store_mut);

    // --- 15 Social Grammar Operations ---

    // 1. Emit -- Alice starts a conversation (independent content, requires causes).
    let emit = gr
        .emit(
            alice.clone(),
            "I have an idea for a new project",
            conv.clone(),
            vec![boot.id.clone()],
            &signer,
        )
        .expect("Emit failed");
    show("Emit", &emit);

    // 2. Respond -- Bob replies (causally dependent, subordinate content).
    let respond = gr
        .respond(
            bob.clone(),
            "That sounds great, tell me more",
            emit.id.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Respond failed");
    show("Respond", &respond);

    // 3. Derive -- Alice builds on her idea (causally dependent, independent content).
    let derive = gr
        .derive(
            alice.clone(),
            "Here's the detailed spec",
            emit.id.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Derive failed");
    show("Derive", &derive);

    // 4. Extend -- Bob adds to Alice's spec (sequential content from same thread).
    let extend = gr
        .extend(
            bob.clone(),
            "Adding performance requirements",
            derive.id.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Extend failed");
    show("Extend", &extend);

    // 5. Retract -- Alice retracts her initial emit (only author can retract).
    let retract = gr
        .retract(
            alice.clone(),
            emit.id.clone(),
            "Superseded by the spec",
            conv.clone(),
            &signer,
        )
        .expect("Retract failed");
    show("Retract", &retract);

    // 6. Annotate -- Alice adds metadata to the spec.
    let annotate = gr
        .annotate(
            alice.clone(),
            derive.id.clone(),
            "priority",
            "high",
            conv.clone(),
            &signer,
        )
        .expect("Annotate failed");
    show("Annotate", &annotate);

    // 7. Acknowledge -- Bob acknowledges Alice's spec (content-free centripetal edge).
    let ack = gr
        .acknowledge(
            bob.clone(),
            derive.id.clone(),
            alice.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Acknowledge failed");
    show("Acknowledge", &ack);

    // 8. Propagate -- Alice propagates the spec to Bob (centrifugal reference edge).
    let propagate = gr
        .propagate(
            alice.clone(),
            derive.id.clone(),
            bob.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Propagate failed");
    show("Propagate", &propagate);

    // 9. Endorse -- Bob endorses Alice's spec (reputation-staked centripetal edge).
    let innovation_scope = DomainScope::new("innovation").unwrap();
    let endorse = gr
        .endorse(
            bob.clone(),
            derive.id.clone(),
            alice.clone(),
            Weight::new(0.9).unwrap(),
            Some(&innovation_scope),
            conv.clone(),
            &signer,
        )
        .expect("Endorse failed");
    show("Endorse", &endorse);

    // 10. Subscribe -- Bob subscribes to Alice's updates (persistent future-oriented edge).
    let projects_scope = DomainScope::new("projects").unwrap();
    let subscribe = gr
        .subscribe(
            bob.clone(),
            alice.clone(),
            Some(&projects_scope),
            boot.id.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Subscribe failed");
    show("Subscribe", &subscribe);

    // 11. Channel -- Alice creates a private channel with Bob (bidirectional edge).
    let project_alpha_scope = DomainScope::new("project_alpha").unwrap();
    let channel = gr
        .channel(
            alice.clone(),
            bob.clone(),
            Some(&project_alpha_scope),
            emit.id.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Channel failed");
    show("Channel", &channel);

    // 12. Delegate -- Alice delegates a task to Bob (authority-granting centrifugal edge).
    let engineering_scope = DomainScope::new("engineering").unwrap();
    let delegate_ev = gr
        .delegate(
            alice.clone(),
            bob.clone(),
            &engineering_scope,
            Weight::new(0.8).unwrap(),
            derive.id.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Delegate failed");
    show("Delegate", &delegate_ev);

    // 13. Consent -- Alice and Bob establish mutual consent.
    let collaboration_scope = DomainScope::new("collaboration").unwrap();
    let consent = gr
        .consent(
            alice.clone(),
            bob.clone(),
            "Both parties agree to collaborate on project alpha",
            &collaboration_scope,
            derive.id.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Consent failed");
    show("Consent", &consent);

    // 14. Sever -- Alice severs the delegation (only parties can sever).
    //     In Rust, sever takes the edge's EventId directly (not a separate EdgeId type).
    let sever = gr
        .sever(
            alice.clone(),
            delegate_ev.id.clone(),
            consent.id.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Sever failed");
    show("Sever", &sever);

    // 15. Merge -- Alice merges her spec with Bob's extension (joins subtrees).
    let merge = gr
        .merge(
            alice.clone(),
            "Final spec with all requirements",
            vec![derive.id.clone(), extend.id.clone()],
            conv.clone(),
            &signer,
        )
        .expect("Merge failed");
    show("Merge", &merge);

    // --- Named Functions (compositions of base operations) ---

    println!("\n--- Named Functions ---");

    // Challenge = Respond + dispute annotation.
    let (challenge_resp, challenge_flag) = gr
        .challenge(
            bob.clone(),
            "I disagree with this approach",
            derive.id.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Challenge failed");
    show("Challenge (resp)", &challenge_resp);
    show("Challenge (flag)", &challenge_flag);

    // Recommend = Propagate + Channel: directed sharing.
    let (rec_prop, rec_chan) = gr
        .recommend(
            alice.clone(),
            derive.id.clone(),
            bob.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Recommend failed");
    show("Recommend (prop)", &rec_prop);
    show("Recommend (chan)", &rec_chan);

    // Invite = Endorse + Subscribe: trust-staked introduction.
    let (inv_endorse, inv_subscribe) = gr
        .invite(
            alice.clone(),
            bob.clone(),
            Weight::new(0.7).unwrap(),
            Some(&engineering_scope),
            derive.id.clone(),
            conv.clone(),
            &signer,
        )
        .expect("Invite failed");
    show("Invite (endorse)", &inv_endorse);
    show("Invite (sub)", &inv_subscribe);

    // Forgive = Subscribe after Sever: reconciliation with history.
    let forgive = gr
        .forgive(
            alice.clone(),
            sever.id.clone(),
            bob.clone(),
            Some(&engineering_scope),
            conv.clone(),
            &signer,
        )
        .expect("Forgive failed");
    show("Forgive", &forgive);

    // --- Verify integrity ---

    // Drop Grammar to release the mutable borrow on the store.
    drop(gr);

    let result = g.store().verify_chain();
    let count = g.store().count();
    println!("\nChain: {} events, valid={}", count, result.valid);

    g.close();
}
