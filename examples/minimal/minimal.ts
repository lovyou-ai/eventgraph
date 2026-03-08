// Minimal EventGraph example — bootstrap a graph, record two events, verify the chain.
// TypeScript mirror of examples/minimal/main.go

import { InMemoryStore } from "../../ts/src/store.js";
import { InMemoryActorStore } from "../../ts/src/actor.js";
import { Graph } from "../../ts/src/graph.js";
import { Grammar } from "../../ts/src/grammar.js";
import { NoopSigner } from "../../ts/src/event.js";
import { ActorId, ConversationId } from "../../ts/src/types.js";

// 1. Create in-memory store and graph.
const store = new InMemoryStore();
const actorStore = new InMemoryActorStore();
const signer = new NoopSigner();
const g = new Graph(store, actorStore, { signer });

// 2. Bootstrap the hash chain.
const systemActor = new ActorId("actor_system");
const boot = g.bootstrap(systemActor, signer);
console.log(`Bootstrap: ${boot.id.value} (hash: ${boot.hash.value.slice(0, 16)}...)`);

// Start the graph — required before recording events.
g.start();

// 3. Record events using social grammar.
const gr = new Grammar(store);
const convId = new ConversationId("conv_example");
const alice = new ActorId("actor_alice");

// Emit — Alice starts with an independent thought.
const ev1 = gr.emit(alice, "Hello, EventGraph!", convId, [boot.id], signer);
console.log(`Event 1:   ${ev1.id.value} (type: ${ev1.type.value})`);

// Derive — Alice builds on her first event.
const ev2 = gr.derive(alice, "A derived thought", ev1.id, convId, signer);
console.log(`Event 2:   ${ev2.id.value} (type: ${ev2.type.value})`);

// 4. Verify chain integrity.
const result = store.verifyChain();
console.log(`Chain:     ${result.length} events, valid=${result.valid}`);

// Clean up.
g.close();
