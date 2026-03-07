# Architecture Test 2: Intelligence Pluggability

Verify that `IIntelligence` implementations can be swapped without affecting the system's behavioral contract.

## Purpose

The system must work with **any** intelligence backend: Claude, GPT, local models, deterministic logic, or a test stub. Swapping the implementation changes the *quality* of reasoning but not the *structure* of the system.

## Setup

```
implementations: [
    StubIntelligence        // returns fixed responses
    DeterministicIntelligence  // rule-based, no randomness
    MockLLMIntelligence     // simulates model calls with latency
]
store: MemoryStore
primitives: [TrustScore, Pattern, Salience]  // primitives that use IIntelligence
```

## Test Cases

### TC-2.1: Stub Intelligence Produces Valid Events

**Input:** Process events through primitives using StubIntelligence (returns Score(0.5) for everything).
**Assertions:**
- Primitives still produce valid events (correct types, causes, signatures)
- Hash chain integrity maintained
- All event content passes validation
- Trust scores are valid Score values

### TC-2.2: Swap Intelligence Mid-Run

**Input:** Process 10 ticks with StubIntelligence, swap to DeterministicIntelligence, process 10 more.
**Assertions:**
- No errors during swap
- Events from both phases form a valid hash chain
- Primitives that were mid-process complete correctly after swap

### TC-2.3: Intelligence Failure Handling

**Input:** Use a FailingIntelligence that returns errors on every call.
**Assertions:**
- Primitives that depend on IIntelligence emit `primitive.error` events (not panic)
- Primitives that are "Both" (mechanical + intelligent) fall back to mechanical behavior
- Primitives that are "Intelligent" only degrade gracefully (reduce activation, retry on next tick)
- System continues processing non-intelligent primitives normally

### TC-2.4: Intelligence Timeout

**Input:** Use a SlowIntelligence that takes 30 seconds per call.
**Assertions:**
- System enforces timeout (configurable)
- Timeout produces a typed error, not a hang
- Timed-out primitive reduces activation and retries next tick

### TC-2.5: Intelligence Result Validation

**Input:** Use a BadIntelligence that returns invalid Score values (> 1.0, negative, NaN).
**Assertions:**
- Results are validated before being used
- Invalid values produce `Err(ValidationError)`, not silent corruption
- Primitive falls back to previous state

### TC-2.6: Deterministic Reproducibility

**Input:** Process the same event sequence twice with DeterministicIntelligence.
**Assertions:**
- Both runs produce identical events (same types, same content, same causes)
- Both runs produce identical trust score updates
- Hash chains are identical (except for timestamps)

### TC-2.7: Intelligence Interface Completeness

**Input:** Register a MinimalIntelligence implementing only required methods.
**Assertions:**
- System runs without error
- Optional methods (if any) have sensible defaults
- No method is called that isn't in the interface

## Error Cases

| Case | Input | Expected |
|------|-------|----------|
| Nil intelligence | Set IIntelligence to nil | `Err(ConfigError)` at startup |
| Wrong return type | Intelligence returns string instead of Score | `Err(ValidationError)` at call site |
| Concurrent calls | 10 primitives call IIntelligence simultaneously | All calls complete, no data races |

## Reference

- `docs/interfaces.md` — IIntelligence interface specification
- `docs/decision-trees.md` — How intelligence integrates with decision trees
