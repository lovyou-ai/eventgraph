# Tutorial: Build Your First Primitive

A primitive is a software agent that embodies a specific domain of intelligence. This tutorial builds a `Counter` primitive that counts events by type.

## What You'll Build

A primitive that:
1. Subscribes to all events
2. Counts how many events of each type it has seen
3. Emits a summary event every 5 ticks

## Prerequisites

- Go 1.22+ installed
- EventGraph cloned: `git clone https://github.com/lovyou-ai/eventgraph.git`

## Step 1: Define the Primitive

Create `counter.go`:

```go
package main

import (
    "github.com/lovyou-ai/eventgraph/go/pkg/event"
    "github.com/lovyou-ai/eventgraph/go/pkg/primitive"
    "github.com/lovyou-ai/eventgraph/go/pkg/types"
)

type Counter struct {
    id types.PrimitiveID
}

func NewCounter() *Counter {
    return &Counter{id: types.MustPrimitiveID("counter")}
}

func (c *Counter) ID() types.PrimitiveID              { return c.id }
func (c *Counter) Layer() types.Layer                  { return types.MustLayer(0) }
func (c *Counter) Cadence() types.Cadence              { return types.MustCadence(1) }
func (c *Counter) Lifecycle() types.LifecycleState     { return types.LifecycleActive }
func (c *Counter) Subscriptions() []types.SubscriptionPattern {
    return []types.SubscriptionPattern{types.MustSubscriptionPattern("*")}
}
```

Every primitive implements 6 methods:
- **ID** — unique identifier
- **Layer** — position in the ontological hierarchy (0 = foundation)
- **Cadence** — minimum ticks between invocations (1 = every tick)
- **Lifecycle** — current state (Dormant → Activating → Active → ...)
- **Subscriptions** — which event types to receive (`*` = all)

## Step 2: Implement Process

The `Process` method receives events and returns mutations:

```go
func (c *Counter) Process(
    tick types.Tick,
    events []event.Event,
    snapshot primitive.Snapshot,
) ([]primitive.Mutation, error) {
    // Read current counts from state
    counts := make(map[string]int)
    if state := snapshot.Primitives[c.id]; state.State() != nil {
        for k, v := range state.State() {
            if n, ok := v.(int); ok {
                counts[k] = n
            }
        }
    }

    // Count new events
    for _, ev := range events {
        counts[ev.Type().Value()]++
    }

    // Build mutations: update state for each event type
    var mutations []primitive.Mutation
    for eventType, count := range counts {
        mutations = append(mutations, primitive.UpdateState{
            PrimitiveID: c.id,
            Key:         eventType,
            Value:       count,
        })
    }

    return mutations, nil
}
```

Key concepts:
- **Snapshot** is read-only — you cannot modify other primitives' state
- **Mutations** are declarative — you return what should change, the tick engine applies it
- Four mutation types: `AddEvent`, `UpdateState`, `UpdateActivation`, `UpdateLifecycle`

## Step 3: Register and Run

```go
package main

import (
    "fmt"

    "github.com/lovyou-ai/eventgraph/go/pkg/graph"
)

func main() {
    g, _ := graph.New(graph.DefaultConfig())
    defer g.Close()

    // Register the primitive
    g.Registry().Register(NewCounter())
    g.Registry().Activate(NewCounter().ID())

    // Bootstrap the graph
    g.Bootstrap()

    // Record some events
    g.Record("trust.updated", map[string]any{"score": 0.8})
    g.Record("trust.updated", map[string]any{"score": 0.9})
    g.Record("system.health", map[string]any{"status": "ok"})

    // Check the counter's state
    states := g.Registry().AllStates()
    state := states[NewCounter().ID()]
    fmt.Printf("Counter state: %v\n", state.State())
    // Output: Counter state: map[system.bootstrapped:1 system.health:1 trust.updated:2]
}
```

## Step 4: Test with the Harness

The `PrimitiveTestHarness` lets you test primitives in isolation:

```go
func TestCounter(t *testing.T) {
    harness := primitive.NewTestHarness(NewCounter())

    // Simulate events
    harness.AddEvent(event.Event{/* ... */})
    harness.Tick()

    // Check mutations
    mutations := harness.Mutations()
    assert.Len(t, mutations, 1)
}
```

## What's Next

- Read `docs/primitives.md` for the full list of 201 primitives
- Read `docs/layers/00-foundation.md` for Layer 0 specifications
- Look at `go/pkg/primitive/layer0/` for real primitive implementations
