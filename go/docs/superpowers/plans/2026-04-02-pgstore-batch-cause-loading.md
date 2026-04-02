# pgstore Batch Cause Loading Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the N+1 cause-loading query in pgstore by replacing per-event `SELECT cause_id` with a single batch `WHERE event_id = ANY($1)` query.

**Architecture:** Split the scan/reconstruct pipeline into two phases: (1) scan raw row data into an intermediate struct, (2) batch-load all causes in one query, then reconstruct events. `reconstructEvent` becomes a pure function with no DB access.

**Tech Stack:** Go 1.24, pgx/v5, PostgreSQL

**Spec:** `go/docs/superpowers/specs/2026-04-02-pgstore-batch-cause-loading-design.md`

---

### Task 1: Add multi-cause conformance test to storetest

**Files:**
- Modify: `pkg/store/storetest/suite.go`

- [ ] **Step 1: Write the multi-cause test**

Add this test inside `RunConformanceSuite`, after the existing `Ancestors` test:

```go
t.Run("MultiCauseRoundTrip", func(t *testing.T) {
    s := newStore()
    ev0 := makeBootstrap(t)
    s.Append(ev0)

    // Create two independent events that will both be causes.
    ev1 := makeChainedEvent(t, s, []types.EventID{ev0.ID()})
    stored1, err := s.Append(ev1)
    if err != nil {
        t.Fatalf("Append ev1: %v", err)
    }
    ev2 := makeChainedEvent(t, s, []types.EventID{stored1.ID()})
    stored2, err := s.Append(ev2)
    if err != nil {
        t.Fatalf("Append ev2: %v", err)
    }

    // Create an event with two causes.
    ev3 := makeChainedEvent(t, s, []types.EventID{stored1.ID(), stored2.ID()})
    stored3, err := s.Append(ev3)
    if err != nil {
        t.Fatalf("Append ev3 (multi-cause): %v", err)
    }

    // Verify via Get.
    got, err := s.Get(stored3.ID())
    if err != nil {
        t.Fatalf("Get: %v", err)
    }
    if len(got.Causes()) != 2 {
        t.Errorf("expected 2 causes, got %d", len(got.Causes()))
    }

    // Verify causes contain both expected IDs (order may vary).
    causeSet := make(map[types.EventID]bool)
    for _, c := range got.Causes() {
        causeSet[c] = true
    }
    if !causeSet[stored1.ID()] {
        t.Error("missing cause: stored1.ID()")
    }
    if !causeSet[stored2.ID()] {
        t.Error("missing cause: stored2.ID()")
    }

    // Verify via ByType (batch path).
    page, err := s.ByType(event.EventTypeTrustUpdated, 100, types.None[types.Cursor]())
    if err != nil {
        t.Fatalf("ByType: %v", err)
    }
    var found bool
    for _, item := range page.Items() {
        if item.ID() == stored3.ID() {
            found = true
            if len(item.Causes()) != 2 {
                t.Errorf("ByType: expected 2 causes, got %d", len(item.Causes()))
            }
        }
    }
    if !found {
        t.Error("multi-cause event not found in ByType results")
    }
})
```

- [ ] **Step 2: Run the test to verify it passes with current code**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go test ./pkg/store/storetest/... -v -run MultiCause`

This test should pass against the current N+1 implementation (it tests behavior, not performance). This establishes the baseline.

Also run the in-memory store tests to confirm: `go test ./pkg/store/... -v -run MultiCause`

- [ ] **Step 3: Commit**

```bash
git add pkg/store/storetest/suite.go
git commit -m "test: add multi-cause round-trip conformance test"
```

---

### Task 2: Add `scannedEvent` struct and `scanRawEvent`/`scanRawSingleEvent` helpers

**Files:**
- Modify: `pkg/store/pgstore/pgstore.go`

- [ ] **Step 1: Add the `scannedEvent` struct and scan helpers**

Add after the `PostgresStore` struct definition (after line 65), before any methods:

```go
// scannedEvent holds raw columns scanned from a database row.
// Used as an intermediate representation between scanning and reconstruction
// to enable batch cause loading.
type scannedEvent struct {
	id             string
	version        int
	eventType      string
	timestampNanos int64
	source         string
	conversationID string
	hash           string
	prevHash       string
	signature      []byte
	contentJSON    []byte
}

// scanRawEvent scans the current row from pgx.Rows into a scannedEvent.
func scanRawEvent(rows pgx.Rows) (scannedEvent, error) {
	var raw scannedEvent
	err := rows.Scan(&raw.id, &raw.version, &raw.eventType, &raw.timestampNanos, &raw.source,
		&raw.conversationID, &raw.hash, &raw.prevHash, &raw.signature, &raw.contentJSON)
	if err != nil {
		return scannedEvent{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("scan event: %v", err)}
	}
	return raw, nil
}

// scanRawSingleEvent scans a single pgx.Row into a scannedEvent.
func scanRawSingleEvent(row pgx.Row) (scannedEvent, error) {
	var raw scannedEvent
	err := row.Scan(&raw.id, &raw.version, &raw.eventType, &raw.timestampNanos, &raw.source,
		&raw.conversationID, &raw.hash, &raw.prevHash, &raw.signature, &raw.contentJSON)
	if err != nil {
		return scannedEvent{}, err
	}
	return raw, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go build ./pkg/store/pgstore/...`

Expected: compiles with no errors (new code is unused but compiles).

- [ ] **Step 3: Commit**

```bash
git add pkg/store/pgstore/pgstore.go
git commit -m "feat: add scannedEvent struct and raw scan helpers"
```

---

### Task 3: Add `batchLoadCauses` helper

**Files:**
- Modify: `pkg/store/pgstore/pgstore.go`

- [ ] **Step 1: Add `batchLoadCauses` function**

Add after the scan helpers from Task 2:

```go
// batchLoadCauses loads causes for multiple events in a single query.
// Returns a map from event ID string to its cause EventIDs.
func batchLoadCauses(ctx context.Context, pool *pgxpool.Pool, eventIDs []string) (map[string][]types.EventID, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}
	rows, err := pool.Query(ctx,
		"SELECT event_id, cause_id FROM event_causes WHERE event_id = ANY($1)", eventIDs)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("batch load causes: %v", err)}
	}
	defer rows.Close()

	result := make(map[string][]types.EventID)
	for rows.Next() {
		var eventID, causeID string
		if err := rows.Scan(&eventID, &causeID); err != nil {
			return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("scan cause: %v", err)}
		}
		result[eventID] = append(result[eventID], types.MustEventID(causeID))
	}
	if err := rows.Err(); err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("batch causes rows: %v", err)}
	}
	return result, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go build ./pkg/store/pgstore/...`

- [ ] **Step 3: Commit**

```bash
git add pkg/store/pgstore/pgstore.go
git commit -m "feat: add batchLoadCauses helper for batch cause queries"
```

---

### Task 4: Refactor `reconstructEvent` to accept causes as parameter

**Files:**
- Modify: `pkg/store/pgstore/pgstore.go`

- [ ] **Step 1: Change `reconstructEvent` signature and remove DB access**

Replace the existing `reconstructEvent` function (lines 765-811) with:

```go
// reconstructEvent rebuilds an Event from database columns and pre-loaded causes.
func reconstructEvent(
	raw scannedEvent,
	causes []types.EventID,
) (event.Event, error) {
	evID := types.MustEventID(raw.id)
	evType := types.MustEventType(raw.eventType)
	ts := types.NewTimestamp(time.Unix(0, raw.timestampNanos))
	src := types.MustActorID(raw.source)
	convID := types.MustConversationID(raw.conversationID)
	h := types.MustHash(raw.hash)
	ph := types.MustHash(raw.prevHash)
	sig := types.MustSignature(raw.signature)

	content, err := unmarshalContent(raw.eventType, raw.contentJSON)
	if err != nil {
		return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("unmarshal content: %v", err)}
	}

	if evType == event.EventTypeSystemBootstrapped {
		bc, ok := content.(event.BootstrapContent)
		if !ok {
			return event.Event{}, &store.StoreUnavailableError{Reason: "bootstrap content type mismatch"}
		}
		return event.NewBootstrapEvent(raw.version, evID, evType, ts, src, bc, convID, h, sig), nil
	}

	return event.NewEvent(raw.version, evID, evType, ts, src, content, causes, convID, h, ph, sig), nil
}
```

- [ ] **Step 2: Update `scanEvent` to use the new pattern**

Replace the existing `scanEvent` function (lines 719-740) with:

```go
// scanEvent scans a single row into an Event. Loads causes via batch helper.
func scanEvent(ctx context.Context, pool *pgxpool.Pool, row pgx.Row) (event.Event, error) {
	raw, err := scanRawSingleEvent(row)
	if err != nil {
		return event.Event{}, err
	}
	causesMap, err := batchLoadCauses(ctx, pool, []string{raw.id})
	if err != nil {
		return event.Event{}, err
	}
	return reconstructEvent(raw, causesMap[raw.id])
}
```

**Note:** Do NOT remove `scanEventFromRows` yet — call sites still reference it. It will be removed in Task 8 after all call sites are updated, keeping every commit compilable.

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go build ./pkg/store/pgstore/...`

Expected: compiles successfully. `scanEventFromRows` is still present (unused by new code but still called by old call sites).

- [ ] **Step 4: Commit**

```bash
git add pkg/store/pgstore/pgstore.go
git commit -m "refactor: make reconstructEvent a pure function, update scanEvent"
```

---

### Task 5: Update `paginateReverse` to two-phase pattern

**Files:**
- Modify: `pkg/store/pgstore/pgstore.go`

- [ ] **Step 1: Rewrite the row iteration in `paginateReverse`**

In `paginateReverse` (starts at line 640), replace the row iteration block:

```go
// OLD (lines 696-703):
var items []event.Event
for rows.Next() {
    ev, err := scanEventFromRows(ctx, s.pool, rows)
    if err != nil {
        return types.Page[event.Event]{}, err
    }
    items = append(items, ev)
}
```

With the two-phase pattern:

```go
// Phase 1: Scan raw rows.
var raws []scannedEvent
for rows.Next() {
    raw, err := scanRawEvent(rows)
    if err != nil {
        return types.Page[event.Event]{}, err
    }
    raws = append(raws, raw)
}
if err := rows.Err(); err != nil {
    return types.Page[event.Event]{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("paginate rows: %v", err)}
}

// Phase 2: Batch load causes.
ids := make([]string, len(raws))
for i, r := range raws {
    ids[i] = r.id
}
causesMap, err := batchLoadCauses(ctx, s.pool, ids)
if err != nil {
    return types.Page[event.Event]{}, err
}

// Phase 3: Reconstruct events.
items := make([]event.Event, 0, len(raws))
for _, r := range raws {
    ev, err := reconstructEvent(r, causesMap[r.id])
    if err != nil {
        return types.Page[event.Event]{}, err
    }
    items = append(items, ev)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go build ./pkg/store/pgstore/...`

Expected: compiles (old `scanEventFromRows` call sites still use it until Tasks 6-8 update them).

- [ ] **Step 3: Commit**

```bash
git add pkg/store/pgstore/pgstore.go
git commit -m "refactor: update paginateReverse to two-phase batch cause loading"
```

---

### Task 6: Update `Since` to two-phase pattern

**Files:**
- Modify: `pkg/store/pgstore/pgstore.go`

- [ ] **Step 1: Rewrite the row iteration in `Since`**

In `Since` (starts at line 355), replace:

```go
// OLD (lines 381-388):
var items []event.Event
for rows.Next() {
    ev, err := scanEventFromRows(ctx, s.pool, rows)
    if err != nil {
        return types.Page[event.Event]{}, err
    }
    items = append(items, ev)
}
```

With:

```go
// Phase 1: Scan raw rows.
var raws []scannedEvent
for rows.Next() {
    raw, err := scanRawEvent(rows)
    if err != nil {
        return types.Page[event.Event]{}, err
    }
    raws = append(raws, raw)
}
if err := rows.Err(); err != nil {
    return types.Page[event.Event]{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("since rows: %v", err)}
}

// Phase 2: Batch load causes.
ids := make([]string, len(raws))
for i, r := range raws {
    ids[i] = r.id
}
causesMap, err := batchLoadCauses(ctx, s.pool, ids)
if err != nil {
    return types.Page[event.Event]{}, err
}

// Phase 3: Reconstruct events.
var items []event.Event
for _, r := range raws {
    ev, err := reconstructEvent(r, causesMap[r.id])
    if err != nil {
        return types.Page[event.Event]{}, err
    }
    items = append(items, ev)
}
```

Note: remove the old `rows.Err()` check that came after the old loop (it's now inside Phase 1).

- [ ] **Step 2: Commit**

```bash
git add pkg/store/pgstore/pgstore.go
git commit -m "refactor: update Since to two-phase batch cause loading"
```

---

### Task 7: Update `Ancestors` and `Descendants` to two-phase pattern

**Files:**
- Modify: `pkg/store/pgstore/pgstore.go`

- [ ] **Step 1: Rewrite `Ancestors` row iteration**

In `Ancestors` (starts at line 403), replace:

```go
// OLD (lines 437-444):
var result []event.Event
for rows.Next() {
    ev, err := scanEventFromRows(ctx, s.pool, rows)
    if err != nil {
        return nil, err
    }
    result = append(result, ev)
}
if err := rows.Err(); err != nil {
    return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("ancestors rows: %v", err)}
}
return result, nil
```

With:

```go
var raws []scannedEvent
for rows.Next() {
    raw, err := scanRawEvent(rows)
    if err != nil {
        return nil, err
    }
    raws = append(raws, raw)
}
if err := rows.Err(); err != nil {
    return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("ancestors rows: %v", err)}
}

ids := make([]string, len(raws))
for i, r := range raws {
    ids[i] = r.id
}
causesMap, err := batchLoadCauses(ctx, s.pool, ids)
if err != nil {
    return nil, err
}

result := make([]event.Event, 0, len(raws))
for _, r := range raws {
    ev, err := reconstructEvent(r, causesMap[r.id])
    if err != nil {
        return nil, err
    }
    result = append(result, ev)
}
return result, nil
```

- [ ] **Step 2: Rewrite `Descendants` row iteration**

Same pattern — in `Descendants` (starts at line 451), replace:

```go
// OLD (lines 484-493):
var result []event.Event
for rows.Next() {
    ev, err := scanEventFromRows(ctx, s.pool, rows)
    if err != nil {
        return nil, err
    }
    result = append(result, ev)
}
if err := rows.Err(); err != nil {
    return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("descendants rows: %v", err)}
}
return result, nil
```

With:

```go
var raws []scannedEvent
for rows.Next() {
    raw, err := scanRawEvent(rows)
    if err != nil {
        return nil, err
    }
    raws = append(raws, raw)
}
if err := rows.Err(); err != nil {
    return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("descendants rows: %v", err)}
}

ids := make([]string, len(raws))
for i, r := range raws {
    ids[i] = r.id
}
causesMap, err := batchLoadCauses(ctx, s.pool, ids)
if err != nil {
    return nil, err
}

result := make([]event.Event, 0, len(raws))
for _, r := range raws {
    ev, err := reconstructEvent(r, causesMap[r.id])
    if err != nil {
        return nil, err
    }
    result = append(result, ev)
}
return result, nil
```

- [ ] **Step 3: Commit**

```bash
git add pkg/store/pgstore/pgstore.go
git commit -m "refactor: update Ancestors and Descendants to two-phase batch cause loading"
```

---

### Task 8: Update `VerifyChain` to two-phase pattern with preserved error semantics

**Files:**
- Modify: `pkg/store/pgstore/pgstore.go`

- [ ] **Step 1: Rewrite `VerifyChain` row iteration**

In `VerifyChain` (starts at line 556), replace the row scan loop and verification loop. The current code iterates rows one at a time, scanning and reconstructing each event then verifying it inline.

Replace the block from `var prevHash string` through the end of the `for rows.Next()` loop with:

```go
// Phase 1: Scan raw rows.
var raws []scannedEvent
for rows.Next() {
    raw, err := scanRawEvent(rows)
    if err != nil {
        return event.ChainVerifiedContent{Valid: false, Length: len(raws)}, nil
    }
    raws = append(raws, raw)
}
if rows.Err() != nil {
    return event.ChainVerifiedContent{Valid: false, Length: len(raws)}, nil
}

// Phase 2: Batch load causes.
ids := make([]string, len(raws))
for i, r := range raws {
    ids[i] = r.id
}
causesMap, err := batchLoadCauses(ctx, s.pool, ids)
if err != nil {
    return event.ChainVerifiedContent{Valid: false, Length: len(raws)}, nil
}

// Phase 3: Reconstruct and verify chain.
var prevHash string
for i, r := range raws {
    ev, err := reconstructEvent(r, causesMap[r.id])
    if err != nil {
        return event.ChainVerifiedContent{Valid: false, Length: i}, nil
    }

    if i == 0 {
        if !ev.IsBootstrap() {
            return event.ChainVerifiedContent{Valid: false, Length: i}, nil
        }
        if ev.PrevHash() != types.ZeroHash() {
            return event.ChainVerifiedContent{Valid: false, Length: i}, nil
        }
    } else {
        if ev.PrevHash().Value() != prevHash {
            return event.ChainVerifiedContent{Valid: false, Length: i}, nil
        }
    }

    canonical := event.CanonicalForm(ev)
    computed, cerr := event.ComputeHash(canonical)
    if cerr != nil {
        return event.ChainVerifiedContent{Valid: false, Length: i}, nil
    }
    if computed != ev.Hash() {
        return event.ChainVerifiedContent{Valid: false, Length: i}, nil
    }

    prevHash = ev.Hash().Value()
}
```

Then update the final return to use `len(raws)` instead of `i`:

```go
ns := time.Since(start).Nanoseconds()
if ns < 0 {
    ns = 0
}
dur := types.MustDuration(ns)
return event.ChainVerifiedContent{
    Valid:    true,
    Length:   len(raws),
    Duration: dur,
}, nil
```

**Critical:** Error semantics are preserved — `reconstructEvent` errors return `(invalid, nil)` not `(zero, error)`.

- [ ] **Step 2: Remove `scanEventFromRows`**

Delete the `scanEventFromRows` function entirely. All call sites have been updated in Tasks 5-8. This is the last reference to it.

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go build ./pkg/store/pgstore/...`

Expected: compiles successfully. All `scanEventFromRows` references are now removed.

- [ ] **Step 4: Run `go vet`**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go vet ./pkg/store/pgstore/...`

Expected: no issues.

- [ ] **Step 5: Commit**

```bash
git add pkg/store/pgstore/pgstore.go
git commit -m "refactor: update VerifyChain to two-phase batch cause loading, remove scanEventFromRows

Preserves error semantics: reconstruction failures return
ChainVerifiedContent{Valid: false} with nil error.
Removes scanEventFromRows — all call sites now use two-phase pattern."
```

---

### Task 9: Run full test suite

**Files:**
- None (verification only)

- [ ] **Step 1: Run in-memory store tests**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go test ./pkg/store/... -v -count=1`

Expected: all tests pass, including the new `MultiCauseRoundTrip` test.

- [ ] **Step 2: Run pgstore tests (if Postgres available)**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && EVENTGRAPH_POSTGRES_URL="postgres://localhost:5432/eventgraph_test?sslmode=disable" go test ./pkg/store/pgstore/... -v -count=1`

If `EVENTGRAPH_POSTGRES_URL` is not configured, the tests will be skipped. That's acceptable — the conformance suite runs against in-memory which validates the multi-cause test, and the pgstore-specific changes are verified by compilation + vet.

- [ ] **Step 3: Run all package tests**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go test ./... -count=1`

Expected: all pass.

- [ ] **Step 4: Run go vet across the module**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go vet ./...`

Expected: no issues.

---

### Task 10: Final commit and cleanup

**Files:**
- None (git only)

- [ ] **Step 1: Review the full diff**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph && git diff main --stat`

Verify:
- `pkg/store/pgstore/pgstore.go` — modified (the main change)
- `pkg/store/storetest/suite.go` — modified (multi-cause test)
- No other files changed

- [ ] **Step 2: Verify no unused imports or dead code**

Run: `cd /home/transpara/transpara-ai/repos/lovyou-ai-eventgraph/go && go vet ./pkg/store/pgstore/...`

Ensure the old `scanEventFromRows` is fully removed and no orphaned imports remain.
