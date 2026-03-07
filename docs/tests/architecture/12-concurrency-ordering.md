# Architecture Test 12: Concurrency & Ordering

Verify that the system maintains correctness under concurrent access.

## Purpose

Multiple actors, agents, and primitives operate concurrently. The hash chain must remain linear, events must be ordered consistently, and no data races can occur. These tests verify correctness under concurrent load.

## Setup

```
store: any Store implementation
actors: [actor_1 through actor_10]
bus: EventBus
engine: TickEngine
```

## Test Cases

### TC-12.1: Concurrent Event Append

**Input:** 10 actors append events simultaneously (100 events total).
**Assertions:**
- All 100 events stored
- Hash chain is linear (no forks)
- `Store.VerifyChain()` passes
- No duplicate event IDs
- Total ordering is deterministic (re-running produces same order)

### TC-12.2: Concurrent Read During Write

**Input:** Actor writes events while another actor queries.
**Assertions:**
- Reader sees a consistent snapshot (no partial events)
- Reader may or may not see in-flight writes (implementation-dependent)
- No data races (verified with `-race` flag)

### TC-12.3: Concurrent Trust Updates

**Input:** 5 actors trigger trust updates for the same target simultaneously.
**Assertions:**
- Final trust score is deterministic
- All trust update events are stored
- Trust score is within [0.0, 1.0]
- No lost updates

### TC-12.4: Bus Concurrent Publish

**Input:** 10 publishers, 10 subscribers, 1000 events.
**Assertions:**
- Every subscriber receives all events matching its subscription
- No duplicates
- No missed events
- Ordering within a single publisher is preserved

### TC-12.5: Tick Engine Concurrent Primitives

**Input:** 20 primitives processing simultaneously in a tick.
**Assertions:**
- All primitives receive the same snapshot
- No primitive sees another's in-progress mutations
- All mutations applied atomically after all primitives complete
- No data races

### TC-12.6: Hash Chain Under Contention

**Input:** High-contention scenario: 100 goroutines each appending 100 events.
**Assertions:**
- 10,000 events stored
- Single linear hash chain
- Chain verification passes
- Performance degrades gracefully (not exponentially)

### TC-12.7: Causal Ordering Consistency

**Input:** Event A causes Event B. Both appended by different actors.
**Assertions:**
- B's causes include A
- B's timestamp ≥ A's timestamp
- Traversing B's ancestors always includes A
- This holds regardless of append ordering

### TC-12.8: Shutdown During Processing

**Input:** Initiate shutdown while a tick is in progress.
**Assertions:**
- Current tick completes (not aborted mid-wave)
- All mutations from current tick are applied
- Store is in a consistent state after shutdown
- Restart can resume from the last completed tick

## Reference

- `docs/interfaces.md` — Store, Bus concurrency requirements
- `docs/tick-engine.md` — Tick engine processing guarantees
