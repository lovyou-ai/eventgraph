// Social Grammar example — demonstrates all 15 social grammar operations
// plus 2 named functions on an event graph.
// TypeScript mirror of examples/social/main.go

import { InMemoryStore } from "../../ts/src/store.js";
import { InMemoryActorStore } from "../../ts/src/actor.js";
import { Graph } from "../../ts/src/graph.js";
import { Grammar } from "../../ts/src/grammar.js";
import { NoopSigner, type Event } from "../../ts/src/event.js";
import {
  ActorId, ConversationId, DomainScope, EdgeId, Option, Weight,
} from "../../ts/src/types.js";

// ── helpers ─────────────────────────────────────────────────────────────

const signer = new NoopSigner();

function show(op: string, ev: Event): void {
  console.log(`${op.padEnd(12)} -> ${ev.type.value} (${ev.id.value.slice(0, 13)}...)`);
}

// ── setup ───────────────────────────────────────────────────────────────

const store = new InMemoryStore();
const actorStore = new InMemoryActorStore();
const g = new Graph(store, actorStore, { signer });

const systemActor = new ActorId("actor_system");
const boot = g.bootstrap(systemActor, signer);

g.start();

const gr = new Grammar(store);
const conv = new ConversationId("conv_social_demo");
const alice = new ActorId("actor_alice");
const bob = new ActorId("actor_bob");

// ── 15 social grammar operations ────────────────────────────────────────

// 1. Emit — Alice starts a conversation.
const emit = gr.emit(alice, "I have an idea for a new project", conv, [boot.id], signer);
show("Emit", emit);

// 2. Respond — Bob replies.
const respond = gr.respond(bob, "That sounds great, tell me more", emit.id, conv, signer);
show("Respond", respond);

// 3. Derive — Alice builds on her idea.
const derive = gr.derive(alice, "Here's the detailed spec", emit.id, conv, signer);
show("Derive", derive);

// 4. Extend — Bob adds to Alice's spec.
const extend = gr.extend(bob, "Adding performance requirements", derive.id, conv, signer);
show("Extend", extend);

// 5. Retract — Alice retracts her initial emit (only author can retract).
const retract = gr.retract(alice, emit.id, "Superseded by the spec", conv, signer);
show("Retract", retract);

// 6. Annotate — Alice adds metadata to the spec.
const annotate = gr.annotate(alice, derive.id, "priority", "high", conv, signer);
show("Annotate", annotate);

// 7. Acknowledge — Bob acknowledges Alice's spec.
const ack = gr.acknowledge(bob, derive.id, alice, conv, signer);
show("Acknowledge", ack);

// 8. Propagate — Alice propagates the spec to Bob.
const propagate = gr.propagate(alice, derive.id, bob, conv, signer);
show("Propagate", propagate);

// 9. Endorse — Bob endorses Alice's spec (reputation-staked).
const endorse = gr.endorse(
  bob, derive.id, alice,
  new Weight(0.9), Option.some(new DomainScope("innovation")),
  conv, signer,
);
show("Endorse", endorse);

// 10. Subscribe — Bob subscribes to Alice's updates.
const subscribe = gr.subscribe(
  bob, alice,
  Option.some(new DomainScope("projects")),
  boot.id, conv, signer,
);
show("Subscribe", subscribe);

// 11. Channel — Alice creates a private channel with Bob.
const channel = gr.channel(
  alice, bob,
  Option.some(new DomainScope("project_alpha")),
  emit.id, conv, signer,
);
show("Channel", channel);

// 12. Delegate — Alice delegates a task to Bob.
const delegate = gr.delegate(
  alice, bob,
  new DomainScope("engineering"), new Weight(0.8),
  derive.id, conv, signer,
);
show("Delegate", delegate);

// 13. Consent — Alice and Bob establish mutual consent.
const consent = gr.consent(
  alice, bob,
  "Both parties agree to collaborate on project alpha",
  new DomainScope("collaboration"),
  derive.id, conv, signer,
);
show("Consent", consent);

// 14. Sever — Alice severs the delegation (only parties can sever).
const delegateEdgeId = new EdgeId(delegate.id.value);
const sever = gr.sever(alice, delegateEdgeId, consent.id, conv, signer);
show("Sever", sever);

// 15. Merge — Alice merges her spec with Bob's extension.
const merge = gr.merge(
  alice, "Final spec with all requirements",
  [derive.id, extend.id], conv, signer,
);
show("Merge", merge);

// ── named functions (compositions of primitives) ────────────────────────

// Challenge = Respond + dispute flag
console.log("\n--- Named Functions ---");
const { response: challengeResp, disputeFlag } = gr.challenge(
  bob, "I disagree with section 3", derive.id, conv, signer,
);
show("Challenge", challengeResp);
show("  +flag", disputeFlag);

// Recommend = Propagate + Channel
const { propagateEv, channelEv } = gr.recommend(
  alice, derive.id, bob, conv, signer,
);
show("Recommend", propagateEv);
show("  +channel", channelEv);

// ── verify integrity ────────────────────────────────────────────────────

const result = store.verifyChain();
const count = store.count();
console.log(`\nChain: ${count} events, valid=${result.valid}`);

// Clean up.
g.close();
