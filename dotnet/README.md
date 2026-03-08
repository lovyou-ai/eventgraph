# LovYou.EventGraph

A hash-chained, append-only, causal event graph. The foundation for building systems where every action is signed, auditable, and causally linked.

## Install

```bash
dotnet add package LovYou.EventGraph
```

## Quick Start

```csharp
using EventGraph;

// Create store and bootstrap
var store = new InMemoryStore();
var signer = new NoopSigner();
var source = new ActorId("actor_alice");
var boot = EventFactory.CreateBootstrap(source, signer);
store.Append(boot);

// Record an event — hash-chained and causally linked
var ev = EventFactory.CreateEvent(
    new EventType("trust.updated"),
    source,
    new Dictionary<string, object?> { ["score"] = 0.85, ["domain"] = "code_review" },
    new List<EventId> { boot.Id },
    new ConversationId("conv_1"),
    boot.Hash,
    signer);
store.Append(ev);

// Verify chain integrity
var result = store.VerifyChain();
Console.WriteLine(result.Valid); // True
```

## What's Included

- **Types** — Always-valid domain models: Score [0,1], Weight [-1,1], Activation [0,1], Layer [0,13], Cadence [1,+inf), typed IDs (EventId, ActorId, Hash, etc.)
- **Event** — Immutable events with canonical form, SHA-256 hash chain, and causal links
- **Store** — IStore interface and InMemoryStore with chain integrity enforcement
- **Bus** — Pub/sub event bus with pattern-based subscriptions
- **Primitive** — IPrimitive interface, lifecycle state machine, and registry
- **Tick Engine** — Ripple-wave processor with cadence control and quiescence detection

## Conformance

This package produces identical SHA-256 hashes to the Go reference implementation for the same canonical form inputs. 94 tests including conformance hash vectors.

## Links

- [GitHub](https://github.com/lovyou-ai/eventgraph)
- [Documentation](https://github.com/lovyou-ai/eventgraph/tree/main/docs)
- [Go reference implementation](https://github.com/lovyou-ai/eventgraph/tree/main/go)

## License

[BSL 1.1](https://github.com/lovyou-ai/eventgraph/blob/main/LICENSE) converting to Apache 2.0 on 26 February 2030.
