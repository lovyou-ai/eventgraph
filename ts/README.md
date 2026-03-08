# @lovyou-ai/eventgraph

A hash-chained, append-only, causal event graph. The foundation for building systems where every action is signed, auditable, and causally linked.

## Install

```bash
npm install @lovyou-ai/eventgraph
```

## Quick Start

```typescript
import {
  ActorId, EventType, ConversationId, Hash,
  createBootstrap, createEvent, NoopSigner, InMemoryStore,
} from "@lovyou-ai/eventgraph";

// Create store and bootstrap
const store = new InMemoryStore();
const signer = new NoopSigner();
const source = new ActorId("actor_alice");
const boot = createBootstrap(source, signer);
store.append(boot);

// Record an event — hash-chained and causally linked
const ev = createEvent(
  new EventType("trust.updated"),
  source,
  { score: 0.85, domain: "code_review" },
  [boot.id],
  new ConversationId("conv_1"),
  boot.hash,
  signer,
);
store.append(ev);

// Verify chain integrity
const result = store.verifyChain();
console.log(result.valid); // true
```

## What's Included

- **Types** — Always-valid domain models: Score [0,1], Weight [-1,1], Activation [0,1], Layer [0,13], Cadence [1,+inf), typed IDs (EventId, ActorId, Hash, etc.)
- **Event** — Immutable events with canonical form, SHA-256 hash chain, and causal links
- **Store** — Store interface and InMemoryStore with chain integrity enforcement
- **Bus** — Pub/sub event bus with pattern-based subscriptions
- **Primitive** — Primitive interface, lifecycle state machine, and registry
- **Tick Engine** — Ripple-wave processor with cadence control and quiescence detection

## Conformance

This package produces identical SHA-256 hashes to the Go reference implementation for the same canonical form inputs.

## Links

- [GitHub](https://github.com/lovyou-ai/eventgraph)
- [Documentation](https://github.com/lovyou-ai/eventgraph/tree/main/docs)
- [Go reference implementation](https://github.com/lovyou-ai/eventgraph/tree/main/go)

## License

[BSL 1.1](https://github.com/lovyou-ai/eventgraph/blob/main/LICENSE) converting to Apache 2.0 on 26 February 2030.
