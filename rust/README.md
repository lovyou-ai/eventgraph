# eventgraph

A hash-chained, append-only, causal event graph. The foundation for building systems where every action is signed, auditable, and causally linked.

## Install

```bash
cargo add eventgraph
```

## Quick Start

```rust
use std::collections::BTreeMap;
use serde_json::Value;
use eventgraph::{
    event::{create_bootstrap, create_event, NoopSigner},
    store::{InMemoryStore, Store},
    types::{ActorId, ConversationId, EventType},
};

// Create store and bootstrap
let mut store = InMemoryStore::new();
let source = ActorId::new("actor_alice").unwrap();
let boot = create_bootstrap(source.clone(), &NoopSigner, 1);
let boot = store.append(boot).unwrap();

// Record an event — hash-chained and causally linked
let mut content = BTreeMap::new();
content.insert("score".into(), Value::Number(serde_json::Number::from_f64(0.85).unwrap()));
content.insert("domain".into(), Value::String("code_review".into()));

let ev = create_event(
    EventType::new("trust.updated").unwrap(),
    ActorId::new("actor_alice").unwrap(),
    content,
    vec![boot.id.clone()],
    ConversationId::new("conv_1").unwrap(),
    boot.hash.clone(),
    &NoopSigner,
    1,
);
store.append(ev).unwrap();

// Verify chain integrity
let result = store.verify_chain();
assert!(result.valid);
```

## What's Included

- **Types** — Always-valid domain models: Score [0,1], Weight [-1,1], Activation [0,1], Layer [0,13], Cadence [1,+inf), typed IDs (EventId, ActorId, Hash, etc.)
- **Event** — Immutable events with canonical form, SHA-256 hash chain, and causal links
- **Store** — Store trait and InMemoryStore with chain integrity enforcement
- **Bus** — Pub/sub event bus with pattern-based subscriptions
- **Primitive** — Primitive trait, lifecycle state machine, and registry
- **Tick Engine** — Ripple-wave processor with cadence control and quiescence detection

## Conformance

This crate produces identical SHA-256 hashes to the Go reference implementation for the same canonical form inputs. 73 tests including 3 conformance hash vectors.

## Links

- [GitHub](https://github.com/lovyou-ai/eventgraph)
- [Documentation](https://github.com/lovyou-ai/eventgraph/tree/main/docs)
- [Go reference implementation](https://github.com/lovyou-ai/eventgraph/tree/main/go)

## License

[BSL 1.1](https://github.com/lovyou-ai/eventgraph/blob/main/LICENSE) converting to Apache 2.0 on 26 February 2030.
