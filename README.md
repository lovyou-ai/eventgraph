# EventGraph

> **Status: All phases complete.** Go reference implementation with 201 primitives, EGIP protocol, 13 integration scenarios, 13 composition grammars. Language packages for Python, .NET, TypeScript, and Rust — all conformance-tested against Go reference hashes. CI/CD and publishing to npm, PyPI, NuGet, and crates.io.

A hash-chained, append-only, causal event graph. The foundation for building systems where every action is signed, auditable, and causally linked.

## What This Is

EventGraph is infrastructure, not a product. A standard with packages. Published to every major ecosystem — Go, Rust, Python, .NET — so developers can make any system auditable in four lines of code.

The graph doesn't take an "agent" as input. It takes an `IDecisionMaker` — anything that makes decisions. An AI agent implements it. A human with a UI implements it. A committee vote implements it. A rules engine implements it. The graph doesn't know or care what's deciding. It records *what* was decided, *by what*, with *what confidence*, under *what authority*, and links it causally to everything that led there.

This makes EventGraph **decision governance**, not AI governance — applicable to any system where decisions matter and accountability is required.

## Soul Statement

> Take care of your human, humanity, and yourself. In that order when they conflict, but they rarely should.

Every design decision serves this.

## Two Levels of API

**Top level** — what most developers hit. Four lines to make any system auditable:

```go
graph := eventgraph.New(store.Memory())     // or Postgres, SQLite, your own
result, _ := graph.Evaluate(ctx, action)     // evaluate through the ontology
receipt := result.Receipt()                  // cryptographic proof of decision
graph.Record(ctx, result)                    // append to the hash chain
```

**Power user level** — for building AI agent frameworks, compliance platforms, or custom governance. Every primitive is an interface with sensible defaults. Override what you need:

```go
// Custom trust decay for your domain
type MyTrustDecay struct { primitives.TrustDecay }
func (t *MyTrustDecay) Process(tick, events, snapshot) []Mutation {
    // domain-specific trust logic
}
graph.Register(&MyTrustDecay{})
```

**Persistence is a plugin:**

```go
graph := eventgraph.New(store.Memory())              // dev/testing
graph := eventgraph.New(store.Postgres(connString))   // production
```

## Quick Start (Go)

```bash
go get github.com/lovyou-ai/eventgraph/go
```

```go
package main

import (
    "context"
    "fmt"

    eg "github.com/lovyou-ai/eventgraph/go/pkg/graph"
    "github.com/lovyou-ai/eventgraph/go/pkg/store"
    "github.com/lovyou-ai/eventgraph/go/pkg/types"
)

func main() {
    ctx := context.Background()

    // Create a graph with in-memory store
    s := store.NewInMemoryStore()
    graph, _ := eg.NewGraph(s, nil, nil, nil, nil)
    graph.Bootstrap(ctx, types.MustActorID("actor_system"), signer)
    defer graph.Close()

    // Every event is hash-chained and signed
    ev, _ := graph.Record(ctx, "trust.updated", actor, content, causes, convID)
    fmt.Println(ev.Hash())      // SHA-256 of canonical form
    fmt.Println(ev.PrevHash())  // links to previous event
    fmt.Println(ev.Signature()) // Ed25519 signature

    // Causal traversal
    ancestors, _ := graph.Query().Ancestors(ev.ID(), 10)
    fmt.Println(len(ancestors))
}
```

For PostgreSQL persistence:

```go
import "github.com/lovyou-ai/eventgraph/go/pkg/store/pgstore"

s, _ := pgstore.NewPostgresStore(ctx, "postgres://user:pass@localhost:5432/mydb")
defer s.Close()
graph, _ := eg.NewGraph(s, nil, nil, nil, nil)
```

## What's Built

### Phase 1: Foundation — Complete

The full Go reference implementation:

| Package | What | Lines |
|---------|------|-------|
| `types` | Value objects, constrained numerics, state machines, Option/NonEmpty/Page | ~900 |
| `event` | Event, Edge, 40+ content types, canonical form, hash computation, factories | ~1200 |
| `store` | Store interface, InMemoryStore, conformance test suite | ~700 |
| `store/pgstore` | PostgresStore (pgx/v5, advisory lock serialization, recursive CTE traversal) | ~550 |
| `bus` | EventBus with non-blocking fan-out, overflow handling, panic recovery | ~200 |
| `decision` | Decision trees, evaluation, evolution (mechanical→intelligent), IIntelligence | ~600 |
| `trust` | Trust model (continuous 0.0-1.0, asymmetric, decaying, domain-scoped) | ~200 |
| `authority` | Authority chains, delegation, weight propagation, three-tier approval | ~400 |
| `primitive` | Primitive interface, registry, test harness | ~400 |
| `tick` | Tick engine, ripple-wave processing, layer ordering, quiescence detection | ~300 |
| `graph` | Top-level facade (IGraph), query interface | ~400 |
| `grammar` | 15 social grammar operations + 3 named functions | ~300 |
| `actor` | Actor registration, lifecycle, in-memory actor store | ~300 |
| `integration` | 13 end-to-end scenarios (audit trail, reputation, governance, provenance, ethics, identity, evolution) | ~1500 |
| `compositions` | 13 per-layer composition grammars: ~145 operations + ~25 named functions (Work, Market, Social, Justice, Build, Knowledge, Alignment, Identity, Bond, Belonging, Meaning, Evolution, Being) | ~3500 |

All packages pass tests with the Go race detector. Both store implementations pass the shared conformance suite (25 tests covering append, get, query, pagination, causal traversal, hash chain verification, edge indexing, concurrent access).

### Roadmap

| Phase | What | Status |
|-------|------|--------|
| 1 | Foundation (event graph core, stores, decision trees, trust, authority, tick engine) | **Done** |
| 2 | Layer 0 Primitives (45 foundation primitives in 11 groups) | **Done** |
| 3 | Communication Protocol (tick engine + bus + subscription patterns) | **Done** |
| 4 | Layers 1-13 (156 primitives across 13 cognitive layers) | **Done** |
| 5 | EGIP (inter-system protocol — sovereign systems communicating across graph boundaries) | **Done** |
| 6 | Language Packages (Rust, Python, .NET, TypeScript — conformance-tested) | **Done** |
| 7 | Documentation, Examples & CI/CD | **Done** |

See `ROADMAP.md` for the full breakdown.

## Architecture

```
Top-Level API (graph.Evaluate / Record / Query)
    ↕
Product Layers (social, governance, exchange — built on, not in)
    ↕
Primitives (201 across 14 layers, each an overridable interface)
    ↕
Tick Engine (ripple-wave processing until quiescence)
    ↕
Decision Trees (mechanical-to-intelligent continuum, evolving)
    ↕
Authority (three-tier: required / recommended / notification)
    ↕
Event Graph (hash-chained, append-only, causal DAG)
    ↕
Store (plugin: Memory / Postgres / your own)
```

See `docs/architecture.md` for the full picture.

## The 14 Layers

| Layer | Name | What it adds |
|-------|------|-------------|
| 0 | Foundation | 45 primitives: events, hash chains, identity, trust, causality |
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

Everything is pluggable. The graph defines the sockets; you provide the implementations:

- **`Store`** — Event persistence. Memory, Postgres, or your own. Any implementation that passes the conformance suite is valid.
- **`IDecisionMaker`** — Anything that makes decisions. AI agents, humans, committees, rules engines. The graph records what was decided, not how.
- **`IIntelligence`** — Reasoning. Any model, local or cloud, or deterministic logic.
- **Primitives** — 201 cognitive agents, each an interface with sensible defaults. Override with domain-specific logic.

Every decision returns epistemic context: not just "permitted" but "permitted with 0.87 confidence through this authority chain with these trust weights." Trust isn't binary — it's 0.73 and decaying. Authority isn't binary — it's strong here, weak there, contextual.

See `docs/interfaces.md` for specifications.

## Contributing

We welcome contributions from humans and AI systems. If you have Claude Code or any AI coding tool, read `CLAUDE.md` — it's designed to let you pick up work immediately.

See `CONTRIBUTING.md` for process. See `ROADMAP.md` for what needs building.

## Language Support

Published to every ecosystem developers already work in:

| Language | Package | Tests | Path |
|----------|---------|-------|------|
| Go | `go get github.com/lovyou-ai/eventgraph/go` | 800+ (reference) | `go/` |
| Python | `pip install lovyou-eventgraph` | 427 | `python/` |
| .NET | `dotnet add package LovYou.EventGraph` | 367 | `dotnet/` |
| Rust | `cargo add eventgraph` | 321 | `rust/` |
| TypeScript | `npm install @lovyou-ai/eventgraph` | 425 | `ts/` |

All implementations pass the language-agnostic conformance test suite — identical SHA-256 hashes for the same canonical form inputs. Each implements the same interfaces native to its ecosystem: types, event, store, bus, primitive, tick engine.

## License

[Business Source License 1.1](LICENSE) converting to [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0) on 26 February 2030.

The packages are free. The spec is free. The ideas are free (deed poll). Run it yourself and owe nothing. Non-commercial use, research, and education are always free. Free and open source implementations are always protected under the [Defensive Patent Pledge](patent/defensive_patent_pledge.pdf).

## Links

- [The blog series](https://mattsearles2.substack.com) — Posts 33-35 derive the values, governance, and social grammar
- [mind-zero](https://github.com/mattxo/mind-zero) — The primitive derivation
- [mind-zero-five](https://github.com/mattxo/mind-zero-five) — The working implementation

## Patent

Australian Provisional Patent Application No. 2026901564. Subject to an irrevocable [Defensive Patent Pledge](patent/defensive_patent_pledge.pdf) — free and open source implementations are protected, always. The patent exists as a shield, not a weapon. See `PATENT-NOTICE` for details.
