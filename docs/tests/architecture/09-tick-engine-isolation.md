# Architecture Test 9: Tick Engine Isolation

Verify that the tick engine's snapshot model, wave processing, and atomic mutation application maintain isolation guarantees.

## Purpose

The tick engine is where all primitives execute. Its correctness guarantees — snapshot immutability, atomic mutations, wave limits, layer ordering — are the foundation of the entire system's safety. These tests verify those guarantees are real, not just documented.

## Setup

```
engine: TickEngine { maxWaves: 10 }
primitives: [
    prim_a: Layer 0, emits "a.output" on "a.input"
    prim_b: Layer 0, emits "b.output" on "a.output" (chained)
    prim_c: Layer 1, subscribes to "a.output"
]
```

## Test Cases

### TC-9.1: Snapshot Immutability

**Input:** During prim_a.Process(), attempt to modify the snapshot.
**Assertions:**
- Modification fails (compile-time in Go via `Frozen<T>`, runtime in dynamic languages)
- Primitive's own state is not affected
- Other primitives see the original snapshot

### TC-9.2: Atomic Mutation Application

**Input:** prim_a returns 3 mutations: [AddEvent, UpdateState, UpdateActivation].
**Assertions:**
- All 3 are applied together (not one at a time)
- If any mutation fails validation, ALL are rolled back
- Partial application never occurs

### TC-9.3: Wave Processing

**Input:** prim_a emits "a.output" → prim_b subscribes to "a.output" → emits "b.output".
**Assertions:**
- Wave 1: prim_a processes, emits "a.output"
- Wave 2: prim_b processes (received "a.output"), emits "b.output"
- Wave 3: if something subscribes to "b.output", it processes
- Each wave is a complete snapshot-process-apply cycle

### TC-9.4: Max Wave Limit

**Input:** Create a cycle: prim_a → prim_b → prim_a (each emits events the other subscribes to).
**Assertions:**
- Processing stops at maxWaves (10)
- Warning event emitted about quiescence not reached
- All events from all waves are valid and stored
- No infinite loop

### TC-9.5: Layer Ordering

**Input:** prim_c (Layer 1) subscribes to "a.output" from prim_a (Layer 0).
**Assertions:**
- prim_c does NOT process until prim_a (and all Layer 0) have quiesced
- "Quiesced" = no more mutations produced by any Layer 0 primitive in a wave
- prim_c receives all Layer 0 events in its first processing call

### TC-9.6: Concurrent Primitive Processing

**Input:** 10 primitives all on Layer 0, all active, all processing.
**Assertions:**
- All 10 receive the same snapshot
- All 10 can process concurrently
- Mutations from all 10 are collected and applied atomically
- No data races (verified with `-race` flag in Go)

### TC-9.7: Cadence Interaction with Waves

**Input:** prim_a has cadence=1, prim_b has cadence=3. Run tick 1 with 3 waves.
**Assertions:**
- prim_a processes in all 3 waves (cadence applies to ticks, not waves)
- prim_b processes in wave 1 only (its cadence is for ticks; within a tick it processes all waves if it has events)
- Actually: cadence controls which ticks invoke the primitive, not which waves within a tick

### TC-9.8: Empty Tick

**Input:** Run a tick with no pending events.
**Assertions:**
- Tick completes with 0 waves (or 1 empty wave)
- No errors
- Primitive states unchanged
- Tick event still recorded (heartbeat)

### TC-9.9: State Persistence Across Ticks

**Input:** prim_a updates state in tick 1. Tick 2 begins.
**Assertions:**
- prim_a sees its updated state in tick 2's snapshot
- State survived the tick boundary
- Hash of state is part of the system's integrity

## Reference

- `docs/tick-engine.md` — Ripple-wave processing model
- `docs/interfaces.md` — Frozen<T>, Mutation types
