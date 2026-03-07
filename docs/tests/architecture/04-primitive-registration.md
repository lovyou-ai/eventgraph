# Architecture Test 4: Primitive Registration

Verify that custom primitives can be registered, their lifecycle is honored, and they integrate cleanly with the tick engine.

## Purpose

SDK users must be able to create custom primitives for domain-specific intelligence. This test verifies the full primitive lifecycle: registration, activation, subscription, processing, state management, and deregistration.

## Setup

```
custom_primitive: TestPrimitive {
    Layer: 1
    Subscriptions: ["test.input"]
    Cadence: 1
    State: { count: 0 }
    Process: increments count, emits "test.output"
}
```

## Test Cases

### TC-4.1: Register Custom Primitive

**Input:** Register TestPrimitive with the tick engine.
**Assertions:**
- Primitive appears in the primitive registry
- Lifecycle state is `Dormant`
- Activation is 0.0
- Subscriptions are registered with the bus

### TC-4.2: Lifecycle State Machine

**Input:** Transition TestPrimitive through all valid states.
**Assertions:**
- `Dormant → Activating → Active → Processing → Active → Deactivating → Dormant` all succeed
- Invalid transitions (e.g., `Dormant → Processing`) return `Err(ValidationError)`
- Each transition emits a lifecycle event
- State machine is enforced, not advisory

### TC-4.3: Subscription Delivery

**Input:** Emit "test.input" events while TestPrimitive is Active.
**Assertions:**
- TestPrimitive.Process() is called with the events
- Events delivered match the subscription prefix
- Events NOT matching the prefix are NOT delivered

### TC-4.4: Cadence Enforcement

**Input:** Set TestPrimitive cadence to 3. Run 10 ticks.
**Assertions:**
- Process() is called on ticks 1, 4, 7, 10 (every 3 ticks)
- Process() is NOT called on ticks 2, 3, 5, 6, 8, 9

### TC-4.5: State Isolation

**Input:** Two custom primitives running simultaneously.
**Assertions:**
- Primitive A cannot read or modify Primitive B's state
- Snapshot passed to Process() is `Frozen<Snapshot>` (deeply immutable)
- Attempting to mutate the snapshot panics or returns error (language-dependent)

### TC-4.6: Mutation Application

**Input:** TestPrimitive returns mutations: [AddEvent, UpdateState, UpdateActivation].
**Assertions:**
- AddEvent: new event appears in Store with correct causes and hash
- UpdateState: primitive state updated atomically after tick
- UpdateActivation: activation value updated within [0.0, 1.0]
- All mutations applied atomically (all or nothing)

### TC-4.7: Layer Constraint

**Input:** Register Layer-2 primitive and Layer-1 primitive. Layer 1 is not yet stable.
**Assertions:**
- Layer-2 primitive does NOT process until Layer-1 primitives are stable
- "Stable" means: all Layer-1 primitives are Active and produced no mutations in the current wave

### TC-4.8: Deregistration

**Input:** Deregister TestPrimitive while it's Active.
**Assertions:**
- Primitive transitions to Deactivating, then Dormant
- Subscriptions removed from bus
- Primitive no longer receives events
- State is preserved (can be re-registered)

### TC-4.9: Multiple Custom Primitives Same Layer

**Input:** Register 5 custom primitives all on Layer 1.
**Assertions:**
- All 5 process in the same tick wave
- No ordering dependency between same-layer primitives
- All 5 share the same snapshot

### TC-4.10: Custom Primitive Emits Custom Event Type

**Input:** TestPrimitive emits an event with type "custom.my_event".
**Assertions:**
- Event type must be registered in EventTypeRegistry first
- If registered, event is stored and delivered to subscribers
- If not registered, mutation returns `Err(ValidationError)`

## Error Cases

| Case | Input | Expected |
|------|-------|----------|
| Duplicate primitive ID | Register same ID twice | `Err(ValidationError)` |
| Invalid layer | Register with Layer(14) | `Err(ValidationError)` at construction |
| Invalid cadence | Register with Cadence(0) | `Err(ValidationError)` at construction |
| Process panic | Primitive panics in Process() | Caught, primitive deactivated, error event emitted |

## Reference

- `docs/interfaces.md` — Primitive interface specification
- `docs/tick-engine.md` — Tick engine processing model
- `docs/primitives.md` — Built-in primitive specifications
