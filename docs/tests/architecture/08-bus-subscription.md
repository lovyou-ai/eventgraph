# Architecture Test 8: Bus & Subscription

Verify that the event bus delivers events correctly, respects subscriptions, and maintains ordering guarantees.

## Purpose

The bus is how primitives communicate. It must deliver the right events to the right subscribers in the right order, and it must be extensible — custom subscribers should work identically to built-in ones.

## Setup

```
bus: EventBus
subscribers: [
    sub_a: subscribes to "trust.*"
    sub_b: subscribes to "work.*"
    sub_c: subscribes to "*" (wildcard)
    sub_d: subscribes to "trust.updated" (exact)
]
```

## Test Cases

### TC-8.1: Prefix Matching

**Input:** Publish "trust.updated" event.
**Assertions:**
- sub_a receives it (prefix "trust." matches)
- sub_b does NOT receive it
- sub_c receives it (wildcard)
- sub_d receives it (exact match)

### TC-8.2: No Duplicate Delivery

**Input:** Publish event matching multiple subscription patterns of the same subscriber.
**Assertions:**
- Subscriber receives the event exactly once, not once per matching pattern

### TC-8.3: Ordering Within Source

**Input:** Publish 10 events from the same source.
**Assertions:**
- All subscribers receive events in emission order
- No reordering within a single source

### TC-8.4: Custom Subscriber

**Input:** Register a custom subscriber (not a primitive) that counts events.
**Assertions:**
- Custom subscriber receives events matching its subscription
- Removal works (unsubscribe stops delivery)
- Custom subscriber doesn't interfere with primitive delivery

### TC-8.5: Dynamic Subscription

**Input:** sub_a subscribes, receives events, unsubscribes, more events published.
**Assertions:**
- Events before unsubscribe are delivered
- Events after unsubscribe are NOT delivered
- Re-subscribing works

### TC-8.6: Backpressure

**Input:** Publish 10,000 events rapidly while a slow subscriber is processing.
**Assertions:**
- No events lost
- Slow subscriber eventually receives all events
- Fast subscribers are not blocked by the slow one

### TC-8.7: Bus Isolation

**Input:** Two separate bus instances.
**Assertions:**
- Events on bus A are not delivered to bus B subscribers
- Each bus is fully independent

### TC-8.8: Event Filtering

**Input:** Subscriber with a filter predicate (e.g., "only events with Score > 0.8").
**Assertions:**
- Only events passing the filter are delivered
- Filter is applied after type matching, before delivery

## Reference

- `docs/interfaces.md` — Bus interface specification
- `docs/tick-engine.md` — How the bus integrates with the tick engine
