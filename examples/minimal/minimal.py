"""Minimal EventGraph example -- bootstrap a graph, record two events, verify the chain.

Mirrors examples/minimal/main.go using the Python package.
Run: cd eventgraph && PYTHONPATH=python python examples/minimal/minimal.py
"""

import sys
import os

# Add the python/ package directory to the import path.
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "python"))

from eventgraph import (
    ActorID,
    ConversationID,
    Grammar,
    InMemoryActorStore,
    InMemoryStore,
    Graph,
    NoopSigner,
)

def main() -> None:
    # 1. Create in-memory store and graph.
    store = InMemoryStore()
    actor_store = InMemoryActorStore()
    g = Graph(store=store, actor_store=actor_store)
    g.start()

    signer = NoopSigner()

    # 2. Bootstrap the hash chain.
    system_actor = ActorID("actor_system")
    boot = g.bootstrap(system_actor, signer)
    print(f"Bootstrap: {boot.id.value} (hash: {boot.hash.value[:16]}...)")

    # 3. Record events using social grammar.
    #    Grammar wraps the store directly (not the Graph facade).
    gr = Grammar(store)
    conv = ConversationID("conv_example")
    alice = ActorID("actor_alice")

    # Emit -- Alice starts a conversation, caused by bootstrap.
    ev1 = gr.emit(alice, "Hello, EventGraph!", conv, [boot.id], signer)
    print(f"Event 1:   {ev1.id.value} (type: {ev1.type.value})")

    # Derive -- Alice builds on her first event.
    ev2 = gr.derive(alice, "A derived thought", ev1.id, conv, signer)
    print(f"Event 2:   {ev2.id.value} (type: {ev2.type.value})")

    # 4. Verify chain integrity.
    result = store.verify_chain()
    print(f"Chain:     {result.length} events, valid={result.valid}")

    g.close()


if __name__ == "__main__":
    main()
