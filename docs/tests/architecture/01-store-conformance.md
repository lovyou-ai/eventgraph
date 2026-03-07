# Architecture Test 1: Store Conformance

Every `Store` implementation must pass this identical test suite. The suite is parameterized — implementations provide a factory, the suite provides the cases.

## Purpose

Verify that **any Store implementation** (memory, SQLite, Postgres, custom) maintains the same behavioral contract. If you can swap Store implementations without changing any other code, the plugin architecture works.

## Setup

```
factory: func() Store  // provided by the implementation under test
bootstrap: Event       // created via BootstrapFactory.Init()
actors: [actor_a, actor_b, actor_c]
```

## Test Cases

### TC-1.1: Append and Retrieve

**Input:** Append a valid event to the store.
**Assertions:**
- `Store.Get(event.ID)` returns the event with identical content
- `Store.Get(event.ID).Hash` matches recomputed hash
- `Store.Get(event.ID).PrevHash` links to the previous event's hash

### TC-1.2: Hash Chain Integrity

**Input:** Append 100 events sequentially.
**Assertions:**
- `Store.VerifyChain()` returns no errors
- Each event's `PrevHash` equals the previous event's `Hash`
- Chain is unbroken from bootstrap to latest

### TC-1.3: Hash Chain Tamper Detection

**Input:** Append 10 events, then directly corrupt event #5's content in storage (bypassing Store API).
**Assertions:**
- `Store.VerifyChain()` returns error identifying event #5
- Error includes both expected and actual hash

### TC-1.4: Causal Query — Ancestors

**Input:** Create a chain: A → B → C → D (each caused by the previous).
**Assertions:**
- `Store.Ancestors(D, depth=3)` returns [C, B, A] in causal order
- `Store.Ancestors(D, depth=1)` returns [C] only
- `Store.Ancestors(A, depth=10)` returns [bootstrap] only

### TC-1.5: Causal Query — Descendants

**Input:** Same chain A → B → C → D, plus A → E (branch).
**Assertions:**
- `Store.Descendants(A, depth=3)` includes B, C, D, E
- `Store.Descendants(A, depth=1)` returns [B, E]
- Results respect causal ordering

### TC-1.6: Query by Type

**Input:** Append events with types "work.submitted", "work.completed", "trust.updated".
**Assertions:**
- `Store.Query(type="work.*")` returns only work events
- `Store.Query(type="trust.*")` returns only trust events
- Results are ordered by timestamp

### TC-1.7: Query by Source

**Input:** Append events from actor_a, actor_b, actor_c.
**Assertions:**
- `Store.Query(source=actor_a)` returns only actor_a's events
- Count matches expected

### TC-1.8: Pagination

**Input:** Append 50 events.
**Assertions:**
- `Store.Query(limit=10)` returns `Page<Event>` with 10 events and `Some(cursor)`
- `Store.Query(limit=10, after=cursor)` returns next 10
- Five pages of 10 exhaust all events, final page has `None` cursor
- No duplicates across pages
- No gaps across pages

### TC-1.9: Empty Store

**Input:** Fresh store, no events.
**Assertions:**
- `Store.Query()` returns empty `Page<Event>` with `None` cursor
- `Store.Get(random_id)` returns `Err(StoreError.NotFound)`
- `Store.VerifyChain()` returns no errors (vacuously true)

### TC-1.10: Concurrent Append

**Input:** Append 100 events concurrently from 10 goroutines (10 each).
**Assertions:**
- All 100 events are stored
- `Store.VerifyChain()` returns no errors
- No duplicate hashes
- Hash chain is linear (no forks)

### TC-1.11: Event Immutability

**Input:** Append event, retrieve it, attempt to modify retrieved event.
**Assertions:**
- Retrieved event is a copy (modifying it doesn't affect the store)
- Re-retrieving returns the original, unmodified event

### TC-1.12: NonEmpty Causes Enforcement

**Input:** Attempt to append event with empty causes (non-bootstrap).
**Assertions:**
- Returns `Err(ValidationError)` — not stored
- Store state unchanged

### TC-1.13: Duplicate ID Rejection

**Input:** Append event, then attempt to append a different event with the same ID.
**Assertions:**
- Second append returns `Err(StoreError.DuplicateID)`
- Original event is unchanged

### TC-1.14: Edge Append and Query

**Input:** Create edges between events with various EdgeTypes.
**Assertions:**
- `Store.GetEdges(event_id)` returns edges connected to that event
- Edges have correct types, weights, and source/target
- Edge creation emits `edge.created` event

### TC-1.15: Subgraph Extract

**Input:** Create a subtree of 10 events rooted at event_a.
**Assertions:**
- `Store.SubgraphExtract(event_a, depth=5)` returns the complete subtree
- No events outside the subtree are included
- Causal ordering is preserved within the subgraph

## Error Cases

| Case | Input | Expected |
|------|-------|----------|
| Invalid EventID format | `Store.Get("not-a-uuid")` | `Err(ValidationError)` |
| Null content | Append event with nil Content | `Err(ValidationError)` |
| Future timestamp | Event with timestamp 1 year in future | `Err(ValidationError)` |
| Invalid hash | Event with incorrect Hash field | `Err(ValidationError)` |
| Invalid signature | Event with mismatched Signature | `Err(ValidationError)` |

## Implementation Pattern

```go
// Each Store implementation registers itself
func TestMemoryStore(t *testing.T) {
    RunStoreConformance(t, func() Store { return NewMemoryStore() })
}

func TestSQLiteStore(t *testing.T) {
    RunStoreConformance(t, func() Store { return NewSQLiteStore(":memory:") })
}

func TestPostgresStore(t *testing.T) {
    RunStoreConformance(t, func() Store { return NewPostgresStore(testDSN) })
}
```

## Reference

- `docs/interfaces.md` — Store interface specification
- `docs/conformance/` — Language-agnostic conformance vectors
