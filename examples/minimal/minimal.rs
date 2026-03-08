// Minimal EventGraph example -- bootstrap a graph, record two events, verify the chain.
//
// To run, add this as an example in rust/Cargo.toml:
//
//   [[example]]
//   name = "minimal"
//   path = "../examples/minimal/minimal.rs"
//
// Then:  cargo run --example minimal
//
// Or compile standalone (from the repo root):
//   rustc --edition 2024 -L target/debug/deps --extern eventgraph=target/debug/libeventgraph.rlib \
//         examples/minimal/minimal.rs -o minimal && ./minimal

use eventgraph::actor::InMemoryActorStore;
use eventgraph::event::NoopSigner;
use eventgraph::grammar::Grammar;
use eventgraph::graph::Graph;
use eventgraph::store::{InMemoryStore, Store};
use eventgraph::types::{ActorId, ConversationId};

fn main() {
    // 1. Create in-memory store, actor store, and graph.
    let store = InMemoryStore::new();
    let actor_store = InMemoryActorStore::new();
    let mut g = Graph::new(store, actor_store);
    g.start().expect("failed to start graph");

    // 2. Bootstrap the hash chain.
    let system_actor = ActorId::new("actor_system").unwrap();
    let boot = g
        .bootstrap(system_actor, Some(&NoopSigner))
        .expect("failed to bootstrap");
    println!(
        "Bootstrap: {} (hash: {}...)",
        boot.id,
        &boot.hash.value()[..16]
    );

    // 3. Record events using social grammar.
    //    Grammar operates directly on the mutable store.
    let signer = NoopSigner;
    let conv = ConversationId::new("conv_example").unwrap();
    let alice = ActorId::new("actor_alice").unwrap();

    let store_mut = g.store_mut();
    let mut gr = Grammar::new(store_mut);

    // Emit -- creates independent content with at least one cause.
    let ev1 = gr
        .emit(
            alice.clone(),
            "Hello, EventGraph!",
            conv.clone(),
            vec![boot.id.clone()],
            &signer,
        )
        .expect("emit failed");
    println!("Event 1:   {} (type: {})", ev1.id, ev1.event_type);

    // Derive -- creates causally dependent but independent content.
    let ev2 = gr
        .derive(
            alice,
            "A derived thought",
            ev1.id,
            conv,
            &signer,
        )
        .expect("derive failed");
    println!("Event 2:   {} (type: {})", ev2.id, ev2.event_type);

    // 4. Verify chain integrity.
    //    We need to drop the Grammar (which borrows store_mut) first,
    //    then access the store through the graph again.
    drop(gr);

    let result = g.store().verify_chain();
    println!(
        "Chain:     {} events, valid={}",
        result.length, result.valid
    );

    // 5. Clean up.
    g.close();
}
