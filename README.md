# EventGraph

A hash-chained, append-only, causal event graph. The foundation for building systems where every action is signed, auditable, and causally linked.

## What This Is

EventGraph is infrastructure, not a product. It provides the substrate for building accountable systems — social networks, governance platforms, marketplaces, identity systems — where every action is an event, every event has a cause, and the full history is cryptographically verifiable.

200 cognitive primitives across 14 layers. A tick engine that processes events through primitives in ripple waves. A decision tree engine that migrates expensive AI reasoning to cheap deterministic rules over time. An inter-system protocol for sovereign systems to communicate without shared infrastructure.

## Soul Statement

> Take care of your human, humanity, and yourself. In that order when they conflict, but they rarely should.

Every design decision serves this.

## Quick Start

### Go

```bash
go get github.com/lovyou-ai/eventgraph/go
```

```go
package main

import (
    "context"
    "github.com/lovyou-ai/eventgraph/go/pkg/event"
    "github.com/lovyou-ai/eventgraph/go/pkg/store"
)

func main() {
    ctx := context.Background()

    // Create an in-memory store
    s := store.NewMemory()
    s.Init(ctx)

    // Emit an event
    e, _ := s.Append(ctx, "hello.world", "me", map[string]any{
        "message": "first event",
    }, nil, "")

    // Every event is hash-chained
    fmt.Println(e.Hash)     // SHA-256
    fmt.Println(e.PrevHash) // links to previous

    // Causal traversal
    ancestors, _ := s.Ancestors(ctx, e.ID, 10)
    descendants, _ := s.Descendants(ctx, e.ID, 10)
}
```

## Architecture

```
Event Graph (hash-chained, append-only, causal)
    ↕
Primitives (200 across 14 layers, each with lifecycle + state + subscriptions)
    ↕
Tick Engine (ripple-wave processing until quiescence)
    ↕
Decision Trees (mechanical-to-intelligent continuum, evolving)
    ↕
Authority (three-tier approval: required / recommended / notification)
    ↕
EGIP (inter-system protocol: sovereign, bilateral, trust-accumulating)
```

See `docs/architecture.md` for the full picture.

## The 14 Layers

| Layer | Name | What it adds |
|-------|------|-------------|
| 0 | Foundation | 44 primitives: events, hash chains, identity, trust, causality |
| 1 | Agency | Observer becomes participant — action and communication |
| 2 | Exchange | Individual becomes dyad — negotiated interaction |
| 3 | Society | Dyad becomes group — collective behaviour |
| 4 | Legal | Informal becomes formal — binding agreements |
| 5 | Technology | Governing becomes building — tool creation |
| 6 | Information | Physical becomes symbolic — representation |
| 7 | Ethics | Is becomes ought — moral reasoning |
| 8 | Identity | Doing becomes being — self-knowledge |
| 9 | Relationship | Self becomes self-with-other |
| 10 | Community | Relationship becomes belonging |
| 11 | Culture | Living culture becomes seeing culture |
| 12 | Emergence | Content becomes architecture — self-organisation |
| 13 | Existence | Everything becomes the fact of everything |

Each layer is derived from a gap in the layer below — something the lower layer cannot express but structurally needs. See `docs/layers/` for per-layer specifications.

## Key Interfaces

Everything is pluggable:

- **`Store`** — Event persistence. Implement for your database.
- **`IIntelligence`** — Reasoning. Implement with any model or deterministic logic.
- **`IDecisionMaker`** — Decision routing. Deterministic branches with AI fallthrough.

See `docs/interfaces.md` for specifications.

## Contributing

We welcome contributions from humans and AI systems. If you have Claude Code or any AI coding tool, read `CLAUDE.md` — it's designed to let you pick up work immediately.

See `CONTRIBUTING.md` for process. See `ROADMAP.md` for what needs building.

## Language Support

| Language | Status | Path |
|----------|--------|------|
| Go | Reference implementation | `go/` |
| Rust | Community — help wanted | `rust/` |
| Python | Community — help wanted | `python/` |
| .NET | Community — help wanted | `dotnet/` |

All implementations must pass the language-agnostic conformance test suite.

## License

[Business Source License 1.1](LICENSE) converting to [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0) on 26 February 2030.

Free and open source implementations are always protected under the [Defensive Patent Pledge](patent/defensive_patent_pledge.pdf).

## Links

- [The blog series](https://mattsearles2.substack.com) — Posts 33-35 derive the values, governance, and social grammar
- [mind-zero](https://github.com/mattxo/mind-zero) — The primitive derivation
- [mind-zero-five](https://github.com/mattxo/mind-zero-five) — The working implementation

## Patent

Australian Provisional Patent Application No. 2026901564. Subject to an irrevocable [Defensive Patent Pledge](patent/defensive_patent_pledge.pdf) — free and open source implementations are protected, always. The patent exists as a shield, not a weapon. See `PATENT-NOTICE` for details.
