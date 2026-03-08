"""Social Grammar example -- demonstrates all 15 social grammar operations + 2 named functions.

Mirrors examples/social/main.go using the Python package.
Run: cd eventgraph && PYTHONPATH=python python examples/social/social.py
"""

import sys
import os

# Add the python/ package directory to the import path.
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "python"))

from eventgraph import (
    ActorID,
    ConversationID,
    DomainScope,
    EdgeID,
    Grammar,
    InMemoryActorStore,
    InMemoryStore,
    Graph,
    NoopSigner,
    Option,
    Weight,
)
from eventgraph.event import Event


def show(op: str, ev: Event) -> None:
    """Print an operation's result: name, event type, and truncated ID."""
    print(f"{op:<12} -> {ev.type.value} ({ev.id.value[:13]}...)")


def main() -> None:
    store = InMemoryStore()
    actor_store = InMemoryActorStore()
    g = Graph(store=store, actor_store=actor_store)
    g.start()

    signer = NoopSigner()

    # Bootstrap the chain.
    system_actor = ActorID("actor_system")
    boot = g.bootstrap(system_actor, signer)

    # Grammar wraps the store directly.
    gr = Grammar(store)
    conv = ConversationID("conv_social_demo")

    alice = ActorID("actor_alice")
    bob = ActorID("actor_bob")

    # ── 15 Social Grammar Operations ──────────────────────────────────

    # 1. Emit -- Alice starts a conversation.
    emit = gr.emit(alice, "I have an idea for a new project", conv, [boot.id], signer)
    show("Emit", emit)

    # 2. Respond -- Bob replies.
    respond = gr.respond(bob, "That sounds great, tell me more", emit.id, conv, signer)
    show("Respond", respond)

    # 3. Derive -- Alice builds on her idea.
    derive = gr.derive(alice, "Here's the detailed spec", emit.id, conv, signer)
    show("Derive", derive)

    # 4. Extend -- Bob adds to Alice's spec.
    extend = gr.extend(bob, "Adding performance requirements", derive.id, conv, signer)
    show("Extend", extend)

    # 5. Retract -- Alice retracts her initial emit (only author can retract).
    retract = gr.retract(alice, emit.id, "Superseded by the spec", conv, signer)
    show("Retract", retract)

    # 6. Annotate -- Alice adds metadata to the spec.
    annotate = gr.annotate(alice, derive.id, "priority", "high", conv, signer)
    show("Annotate", annotate)

    # 7. Acknowledge -- Bob acknowledges Alice's spec.
    ack = gr.acknowledge(bob, derive.id, alice, conv, signer)
    show("Acknowledge", ack)

    # 8. Propagate -- Alice propagates the spec to Bob.
    propagate = gr.propagate(alice, derive.id, bob, conv, signer)
    show("Propagate", propagate)

    # 9. Endorse -- Bob endorses Alice's spec (reputation-staked).
    endorse = gr.endorse(
        bob, derive.id, alice,
        Weight(0.9), Option.some(DomainScope("innovation")),
        conv, signer,
    )
    show("Endorse", endorse)

    # 10. Subscribe -- Bob subscribes to Alice's updates.
    subscribe = gr.subscribe(
        bob, alice,
        Option.some(DomainScope("projects")),
        boot.id, conv, signer,
    )
    show("Subscribe", subscribe)

    # 11. Channel -- Alice creates a private channel with Bob.
    channel = gr.channel(
        alice, bob,
        Option.some(DomainScope("project_alpha")),
        emit.id, conv, signer,
    )
    show("Channel", channel)

    # 12. Delegate -- Alice delegates a task to Bob.
    delegate = gr.delegate(
        alice, bob,
        DomainScope("engineering"), Weight(0.8),
        derive.id, conv, signer,
    )
    show("Delegate", delegate)

    # 13. Consent -- Alice and Bob establish mutual consent.
    consent = gr.consent(
        alice, bob,
        "Both parties agree to collaborate on project alpha",
        DomainScope("collaboration"),
        derive.id, conv, signer,
    )
    show("Consent", consent)

    # 14. Sever -- Alice severs the delegation (only parties can sever).
    delegate_edge_id = EdgeID(delegate.id.value)
    sever = gr.sever(alice, delegate_edge_id, consent.id, conv, signer)
    show("Sever", sever)

    # 15. Merge -- Alice merges her spec with Bob's extension.
    merge = gr.merge(
        alice, "Final spec with all requirements",
        [derive.id, extend.id], conv, signer,
    )
    show("Merge", merge)

    # ── Named Functions (compositions of operations) ──────────────────

    print()
    print("Named functions:")

    # Challenge = Respond + dispute annotation.
    challenge_resp, challenge_flag = gr.challenge(
        bob, "I disagree with this approach", derive.id, conv, signer,
    )
    show("Challenge[0]", challenge_resp)
    show("Challenge[1]", challenge_flag)

    # Recommend = Propagate + Channel.
    rec_prop, rec_chan = gr.recommend(
        alice, derive.id, bob, conv, signer,
    )
    show("Recommend[0]", rec_prop)
    show("Recommend[1]", rec_chan)

    # ── Verify integrity ──────────────────────────────────────────────

    result = store.verify_chain()
    count = store.count()
    print(f"\nChain: {count} events, valid={result.valid}")

    g.close()


if __name__ == "__main__":
    main()
