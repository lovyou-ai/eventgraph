# EventGraph for Python

A hash-chained, append-only, causal event graph. The foundation for building systems where every action is signed, auditable, and causally linked.

## Install

```bash
pip install lovyou-eventgraph
```

## Quick Start

```python
from eventgraph import (
    ActorID, EventType, ConversationID, NoopSigner,
    create_bootstrap, create_event, InMemoryStore,
)

# Create store and bootstrap
store = InMemoryStore()
signer = NoopSigner()
source = ActorID("actor_alice")
boot = create_bootstrap(source, signer)
store.append(boot)

# Record an event — hash-chained and causally linked
ev = create_event(
    event_type=EventType("trust.updated"),
    source=source,
    content={"score": 0.85, "domain": "code_review"},
    causes=[boot.id],
    conversation_id=ConversationID("conv_1"),
    prev_hash=boot.hash,
    signer=signer,
)
store.append(ev)

# Verify chain integrity
result = store.verify_chain()
assert result.valid
```

## What's Included

- **Types** — Always-valid domain models: Score [0,1], Weight [-1,1], Activation [0,1], Layer [0,13], Cadence [1,+inf), typed IDs (EventId, ActorId, Hash, etc.)
- **Event** — Immutable events with canonical form, SHA-256 hash chain, and causal links
- **Store** — Store protocol and InMemoryStore with chain integrity enforcement
- **Bus** — Pub/sub event bus with pattern-based subscriptions
- **Primitive** — Primitive protocol, lifecycle state machine, and registry
- **Tick Engine** — Ripple-wave processor with cadence control and quiescence detection

## Conformance

This package produces identical SHA-256 hashes to the Go reference implementation for the same canonical form inputs. 135 tests including conformance vectors.

## Links

- [GitHub](https://github.com/lovyou-ai/eventgraph)
- [Documentation](https://github.com/lovyou-ai/eventgraph/tree/main/docs)
- [Go reference implementation](https://github.com/lovyou-ai/eventgraph/tree/main/go)

## License

[BSL 1.1](https://github.com/lovyou-ai/eventgraph/blob/main/LICENSE) converting to Apache 2.0 on 26 February 2030.
