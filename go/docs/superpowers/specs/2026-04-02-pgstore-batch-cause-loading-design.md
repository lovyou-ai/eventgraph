# pgstore: Batch Cause Loading

**Version:** 0.1.0
**Last Updated:** 2026-04-02
**Author:** Michael Saucier, Claude (Anthropic)
**Status:** Approved
**Scope:** `pkg/store/pgstore/pgstore.go` — eliminate N+1 cause-loading queries

### Changelog

| Version | Date | Changes |
|---------|------|---------|
| v0.1.0 | 2026-04-02 | Initial design spec |

---

## Problem

`reconstructEvent` in `pgstore.go` issues a separate `SELECT cause_id FROM event_causes WHERE event_id = $1` query for every event row returned by any read operation. For a `ByType` call returning 100 events, this produces 100 additional round-trips to the database. The `batchStatus` method in lovyou-ai-work calls `ByType` 4 times, resulting in up to 400 sequential database round-trips per request.

This N+1 pattern is the dominant performance bottleneck for all pgstore read operations.

## Solution

Replace the per-event cause query in `reconstructEvent` with a single batch query issued after all event rows are scanned. The batch query uses `WHERE event_id = ANY($1)` with the existing `event_causes(event_id, cause_id)` primary key index.

The change is internal to pgstore. No schema changes, no Store interface changes, no changes to other store implementations or consumer code.

## Values and Risks

### Upsides

1. **50-100x reduction in database round-trips** for multi-event read operations — the primary bottleneck
2. **No schema migration required** — uses existing tables and indexes as-is
3. **No interface changes** — purely internal refactoring, invisible to consumers
4. **Proven pattern** — the MySQL store already avoids N+1 by storing causes inline; this achieves the same result without schema changes
5. **Improves every read path** — `ByType`, `Recent`, `BySource`, `ByConversation`, `Since`, `Ancestors`, `Descendants`, `VerifyChain` all benefit

### Downsides

1. **Increases memory usage slightly** — all scanned row data is held in memory before reconstruction instead of being processed one at a time. For the current limit of 1000 events this is negligible (tens of KB)
2. **Refactoring complexity** — the scan/reconstruct pipeline is split into two phases, introducing an intermediate struct and changing several call sites
3. **Single-event fetches (`Get`, `Head`) become 1+1 queries instead of 1+1** — no change in round-trips, but the code path changes

### Risk Assessment

Low risk. The change is scoped to a single file with no public API changes. The existing `storetest` conformance suite validates correctness across all Store interface methods.

---

## Design

### New Private Types

**`scannedEvent`** — holds raw scanned columns between the scan and reconstruct phases:

```go
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
```

### New Helper Functions

**`batchLoadCauses`** — single batch query for all cause IDs:

```go
func batchLoadCauses(ctx context.Context, pool *pgxpool.Pool, eventIDs []string) (map[string][]types.EventID, error)
```

Query: `SELECT event_id, cause_id FROM event_causes WHERE event_id = ANY($1)`

Returns a map from event ID string to its list of cause EventIDs. Events with no causes get an empty/nil slice (map miss). Uses the existing primary key index `(event_id, cause_id)`.

**`scanRawEvent`** — scans a row into `scannedEvent` without any DB calls:

```go
func scanRawEvent(rows pgx.Rows) (scannedEvent, error)
```

**`scanRawSingleEvent`** — same for `pgx.Row` (single-row variant):

```go
func scanRawSingleEvent(row pgx.Row) (scannedEvent, error)
```

### Modified Functions

**`reconstructEvent`** — signature changes to accept causes instead of loading them:

```go
// Before:
func reconstructEvent(ctx context.Context, pool *pgxpool.Pool,
    id string, version int, ...) (event.Event, error)

// After:
func reconstructEvent(
    id string, version int, ...,
    causes []types.EventID) (event.Event, error)
```

No `ctx` or `pool` parameters needed — this becomes a pure transformation function.

**`scanEvent`** (single-row, used by `getEvent` and `Head`) — scans raw columns, loads causes for 1 event ID via `batchLoadCauses`, then calls `reconstructEvent`:

```go
func scanEvent(ctx context.Context, pool *pgxpool.Pool, row pgx.Row) (event.Event, error) {
    raw, err := scanRawSingleEvent(row)
    if err != nil {
        return event.Event{}, err
    }
    causesMap, err := batchLoadCauses(ctx, pool, []string{raw.id})
    if err != nil {
        return event.Event{}, err
    }
    return reconstructEvent(raw.id, raw.version, ..., causesMap[raw.id])
}
```

**`scanEventFromRows`** — removed. Replaced by the two-phase pattern at each call site.

### Call Site Changes

Every function that previously iterated with `scanEventFromRows` switches to:

```go
// Phase 1: Scan all rows into raw structs.
var raws []scannedEvent
for rows.Next() {
    raw, err := scanRawEvent(rows)
    if err != nil {
        return ..., err
    }
    raws = append(raws, raw)
}

// Phase 2: Batch load causes.
ids := make([]string, len(raws))
for i, r := range raws {
    ids[i] = r.id
}
causesMap, err := batchLoadCauses(ctx, pool, ids)
if err != nil {
    return ..., err
}

// Phase 3: Reconstruct events.
events := make([]event.Event, 0, len(raws))
for _, r := range raws {
    ev, err := reconstructEvent(r.id, r.version, ..., causesMap[r.id])
    if err != nil {
        return ..., err
    }
    events = append(events, ev)
}
```

Affected functions:
- `paginateReverse` — used by `ByType`, `Recent`, `BySource`, `ByConversation`
- `Since`
- `Ancestors`
- `Descendants`
- `VerifyChain`

### `VerifyChain` Special Consideration

`VerifyChain` currently processes events one at a time in chain order, checking each event's hash against the previous. The two-phase pattern still works — scan all rows first, batch load causes, then iterate through the reconstructed events sequentially for verification. The verification logic itself does not change.

### Empty Result Optimization

If `len(raws) == 0` after scanning, skip the `batchLoadCauses` call entirely. No query issued for empty result sets.

## Testing Strategy

- The `storetest` conformance suite (`pkg/store/storetest/suite.go`) validates all Store interface methods and runs against pgstore
- `pgstore_test.go` runs the conformance suite against a real Postgres instance
- No new tests needed — behavior is unchanged; only the number of queries changes
- Verification: `go test ./pkg/store/pgstore/...` for correctness

## What Doesn't Change

- No schema changes — `event_causes` table, indexes, and `events` table unchanged
- No `Store` interface changes
- No changes to `Append` — cause insertion during writes stays per-cause within a transaction
- No changes to other store implementations (memory, mysql, sqlite)
- No changes to consumer code (lovyou-ai-work, lovyou-ai-agent, etc.)
- No behavioral changes visible to callers
