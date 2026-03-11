// Package pgstore implements a PostgreSQL-backed Store.
package pgstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

const schema = `
CREATE TABLE IF NOT EXISTS events (
	seq BIGSERIAL PRIMARY KEY,
	id TEXT UNIQUE NOT NULL,
	version INT NOT NULL,
	event_type TEXT NOT NULL,
	timestamp_nanos BIGINT NOT NULL,
	source TEXT NOT NULL,
	conversation_id TEXT NOT NULL,
	hash TEXT NOT NULL,
	prev_hash TEXT NOT NULL,
	signature BYTEA NOT NULL,
	content_json JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_source ON events(source);
CREATE INDEX IF NOT EXISTS idx_events_conversation ON events(conversation_id);

CREATE TABLE IF NOT EXISTS event_causes (
	event_id TEXT NOT NULL,
	cause_id TEXT NOT NULL,
	PRIMARY KEY (event_id, cause_id)
);

CREATE INDEX IF NOT EXISTS idx_event_causes_cause ON event_causes(cause_id);

CREATE TABLE IF NOT EXISTS edges (
	id TEXT PRIMARY KEY,
	from_actor TEXT NOT NULL,
	to_actor TEXT NOT NULL,
	edge_type TEXT NOT NULL,
	weight DOUBLE PRECISION NOT NULL,
	direction TEXT NOT NULL,
	scope TEXT,
	created_at_nanos BIGINT NOT NULL,
	expires_at_nanos BIGINT
);

CREATE INDEX IF NOT EXISTS idx_edges_from_type ON edges(from_actor, edge_type);
CREATE INDEX IF NOT EXISTS idx_edges_to_type ON edges(to_actor, edge_type);
`

// PostgresStore implements store.Store backed by PostgreSQL.
type PostgresStore struct {
	pool     *pgxpool.Pool
	ownsPool bool // true when we created the pool, false when borrowed via FromPool
}

// NewPostgresStore creates a PostgresStore connected to the given Postgres instance.
// It creates the schema if it doesn't exist.
func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("connect: %v", err)}
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("schema: %v", err)}
	}
	return &PostgresStore{pool: pool, ownsPool: true}, nil
}

// NewPostgresStoreFromPool creates a PostgresStore from an existing connection pool.
// It creates the schema if it doesn't exist.
// The caller retains ownership of the pool — Close() will not close it.
func NewPostgresStoreFromPool(ctx context.Context, pool *pgxpool.Pool) (*PostgresStore, error) {
	if _, err := pool.Exec(ctx, schema); err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("schema: %v", err)}
	}
	return &PostgresStore{pool: pool, ownsPool: false}, nil
}

func (s *PostgresStore) Append(ev event.Event) (event.Event, error) {
	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("begin tx: %v", err)}
	}
	defer tx.Rollback(ctx)

	// Advisory lock to serialize chain writes (lock ID 1 for the event chain).
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(1)"); err != nil {
		return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("advisory lock: %v", err)}
	}

	// Idempotency: if same ID exists, verify hash matches and return it.
	var existingHash string
	err = tx.QueryRow(ctx, "SELECT hash FROM events WHERE id = $1", ev.ID().Value()).Scan(&existingHash)
	if err == nil {
		if existingHash != ev.Hash().Value() {
			return event.Event{}, &store.HashMismatchError{
				EventID:  ev.ID(),
				Computed: ev.Hash(),
				Stored:   types.MustHash(existingHash),
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("commit (idempotent): %v", err)}
		}
		return s.getEvent(ctx, ev.ID())
	}
	if err != pgx.ErrNoRows {
		return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("check existing: %v", err)}
	}

	// Verify PrevHash matches chain head.
	var headHash string
	var headExists bool
	err = tx.QueryRow(ctx, "SELECT hash FROM events ORDER BY seq DESC LIMIT 1").Scan(&headHash)
	if err == pgx.ErrNoRows {
		headExists = false
	} else if err != nil {
		return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("head query: %v", err)}
	} else {
		headExists = true
	}

	if !headExists {
		if ev.PrevHash() != types.ZeroHash() {
			return event.Event{}, &store.ChainIntegrityViolationError{
				Position: 0,
				Expected: types.ZeroHash(),
				Actual:   ev.PrevHash(),
			}
		}
	} else {
		if ev.PrevHash().Value() != headHash {
			count := 0
			tx.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&count)
			return event.Event{}, &store.ChainIntegrityViolationError{
				Position: count,
				Expected: types.MustHash(headHash),
				Actual:   ev.PrevHash(),
			}
		}
	}

	// Recompute hash and verify.
	canonical := event.CanonicalForm(ev)
	computed, err := event.ComputeHash(canonical)
	if err != nil {
		return event.Event{}, err
	}
	if computed != ev.Hash() {
		return event.Event{}, &store.HashMismatchError{
			EventID:  ev.ID(),
			Computed: computed,
			Stored:   ev.Hash(),
		}
	}

	// Verify causal predecessors exist.
	if !ev.IsBootstrap() {
		for _, causeID := range ev.Causes() {
			var exists bool
			err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM events WHERE id = $1)", causeID.Value()).Scan(&exists)
			if err != nil {
				return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("cause check: %v", err)}
			}
			if !exists {
				return event.Event{}, &store.CausalLinkMissingError{
					EventID:      ev.ID(),
					MissingCause: causeID,
				}
			}
		}
	}

	// Validate edge content before inserting.
	var edge event.Edge
	var hasEdge bool
	if ev.Type() == event.EventTypeEdgeCreated {
		ec, ok := ev.Content().(event.EdgeCreatedContent)
		if !ok {
			return event.Event{}, &store.EdgeIndexError{
				EventID: ev.ID(),
				Reason:  fmt.Sprintf("wrong content type %T", ev.Content()),
			}
		}
		edgeID, edgeIDErr := types.NewEdgeID(ev.ID().Value())
		if edgeIDErr != nil {
			return event.Event{}, &store.EdgeIndexError{
				EventID: ev.ID(),
				Reason:  fmt.Sprintf("derive edge ID: %v", edgeIDErr),
			}
		}
		var newEdgeErr error
		edge, newEdgeErr = event.NewEdge(
			edgeID, ec.From, ec.To, ec.EdgeType, ec.Weight, ec.Direction,
			ec.Scope, nil, ev.Timestamp(), ec.ExpiresAt,
		)
		if newEdgeErr != nil {
			return event.Event{}, &store.EdgeIndexError{
				EventID: ev.ID(),
				Reason:  fmt.Sprintf("construct edge: %v", newEdgeErr),
			}
		}
		hasEdge = true
	}

	var supersededPrevID string
	if ev.Type() == event.EventTypeEdgeSuperseded {
		ec, ok := ev.Content().(event.EdgeSupersededContent)
		if !ok {
			return event.Event{}, &store.EdgeIndexError{
				EventID: ev.ID(),
				Reason:  fmt.Sprintf("wrong content type for edge.superseded: %T", ev.Content()),
			}
		}
		supersededPrevID = ec.PreviousEdge.Value()
	}

	// Serialize content to JSON.
	contentJSON, err := json.Marshal(ev.Content())
	if err != nil {
		return event.Event{}, fmt.Errorf("marshal content: %w", err)
	}

	// Insert the event.
	_, err = tx.Exec(ctx,
		`INSERT INTO events (id, version, event_type, timestamp_nanos, source, conversation_id, hash, prev_hash, signature, content_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		ev.ID().Value(), ev.Version(), ev.Type().Value(), ev.Timestamp().UnixNano(),
		ev.Source().Value(), ev.ConversationID().Value(), ev.Hash().Value(), ev.PrevHash().Value(),
		ev.Signature().Bytes(), contentJSON,
	)
	if err != nil {
		return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("insert event: %v", err)}
	}

	// Insert causes.
	for _, causeID := range ev.Causes() {
		_, err = tx.Exec(ctx,
			"INSERT INTO event_causes (event_id, cause_id) VALUES ($1, $2)",
			ev.ID().Value(), causeID.Value(),
		)
		if err != nil {
			return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("insert cause: %v", err)}
		}
	}

	// Insert edge if applicable.
	if hasEdge {
		var scopeVal *string
		if edge.Scope().IsSome() {
			s := edge.Scope().Unwrap().Value()
			scopeVal = &s
		}
		var expiresNanos *int64
		if edge.ExpiresAt().IsSome() {
			n := edge.ExpiresAt().Unwrap().UnixNano()
			expiresNanos = &n
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO edges (id, from_actor, to_actor, edge_type, weight, direction, scope, created_at_nanos, expires_at_nanos)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			edge.ID().Value(), edge.From().Value(), edge.To().Value(),
			string(edge.Type()), edge.Weight().Value(), string(edge.Direction()),
			scopeVal, edge.CreatedAt().UnixNano(), expiresNanos,
		)
		if err != nil {
			return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("insert edge: %v", err)}
		}
	}

	// Handle edge supersession.
	if supersededPrevID != "" {
		_, err = tx.Exec(ctx, "DELETE FROM edges WHERE id = $1", supersededPrevID)
		if err != nil {
			return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("delete superseded edge: %v", err)}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("commit: %v", err)}
	}
	return ev, nil
}

func (s *PostgresStore) Get(id types.EventID) (event.Event, error) {
	ev, err := s.getEvent(context.Background(), id)
	if err != nil {
		return event.Event{}, err
	}
	return ev, nil
}

func (s *PostgresStore) Head() (types.Option[event.Event], error) {
	ctx := context.Background()
	row := s.pool.QueryRow(ctx,
		`SELECT id, version, event_type, timestamp_nanos, source, conversation_id,
		        hash, prev_hash, signature, content_json
		 FROM events ORDER BY seq DESC LIMIT 1`)
	ev, err := scanEvent(ctx, s.pool, row)
	if err == pgx.ErrNoRows {
		return types.None[event.Event](), nil
	}
	if err != nil {
		return types.None[event.Event](), err
	}
	return types.Some(ev), nil
}

func (s *PostgresStore) Recent(limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 100
	}
	return s.paginateReverse(ctx, "", nil, limit, after)
}

func (s *PostgresStore) ByType(eventType types.EventType, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 100
	}
	return s.paginateReverse(ctx, "event_type = $%d", []any{eventType.Value()}, limit, after)
}

func (s *PostgresStore) BySource(source types.ActorID, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 100
	}
	return s.paginateReverse(ctx, "source = $%d", []any{source.Value()}, limit, after)
}

func (s *PostgresStore) ByConversation(id types.ConversationID, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 100
	}
	return s.paginateReverse(ctx, "conversation_id = $%d", []any{id.Value()}, limit, after)
}

func (s *PostgresStore) Since(afterID types.EventID, limit int) (types.Page[event.Event], error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 100
	}

	// Find the seq of the afterID event.
	var afterSeq int64
	err := s.pool.QueryRow(ctx, "SELECT seq FROM events WHERE id = $1", afterID.Value()).Scan(&afterSeq)
	if err == pgx.ErrNoRows {
		return types.Page[event.Event]{}, &store.EventNotFoundError{ID: afterID}
	}
	if err != nil {
		return types.Page[event.Event]{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("since lookup: %v", err)}
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, version, event_type, timestamp_nanos, source, conversation_id,
		        hash, prev_hash, signature, content_json
		 FROM events WHERE seq > $1 ORDER BY seq ASC LIMIT $2`,
		afterSeq, limit)
	if err != nil {
		return types.Page[event.Event]{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("since query: %v", err)}
	}
	defer rows.Close()

	var items []event.Event
	for rows.Next() {
		ev, err := scanEventFromRows(ctx, s.pool, rows)
		if err != nil {
			return types.Page[event.Event]{}, err
		}
		items = append(items, ev)
	}
	if err := rows.Err(); err != nil {
		return types.Page[event.Event]{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("since rows: %v", err)}
	}

	// Check hasMore.
	var totalAfter int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM events WHERE seq > $1", afterSeq).Scan(&totalAfter); err != nil {
		return types.Page[event.Event]{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("since count: %v", err)}
	}
	hasMore := totalAfter > len(items)

	return types.NewPage(items, types.None[types.Cursor](), hasMore), nil
}

func (s *PostgresStore) Ancestors(id types.EventID, maxDepth int) ([]event.Event, error) {
	ctx := context.Background()

	// Verify the event exists.
	var exists bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM events WHERE id = $1)", id.Value()).Scan(&exists)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("ancestors check: %v", err)}
	}
	if !exists {
		return nil, &store.EventNotFoundError{ID: id}
	}

	// Recursive CTE for ancestor traversal.
	rows, err := s.pool.Query(ctx,
		`WITH RECURSIVE ancestors AS (
			SELECT cause_id AS id, 1 AS depth
			FROM event_causes WHERE event_id = $1
		  UNION
			SELECT ec.cause_id, a.depth + 1
			FROM ancestors a
			JOIN event_causes ec ON ec.event_id = a.id
			WHERE a.depth < $2
		)
		SELECT DISTINCT e.id, e.version, e.event_type, e.timestamp_nanos, e.source,
		       e.conversation_id, e.hash, e.prev_hash, e.signature, e.content_json
		FROM ancestors a
		JOIN events e ON e.id = a.id`,
		id.Value(), maxDepth)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("ancestors query: %v", err)}
	}
	defer rows.Close()

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
}

func (s *PostgresStore) Descendants(id types.EventID, maxDepth int) ([]event.Event, error) {
	ctx := context.Background()

	var exists bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM events WHERE id = $1)", id.Value()).Scan(&exists)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("descendants check: %v", err)}
	}
	if !exists {
		return nil, &store.EventNotFoundError{ID: id}
	}

	// Recursive CTE for descendant traversal.
	rows, err := s.pool.Query(ctx,
		`WITH RECURSIVE descendants AS (
			SELECT event_id AS id, 1 AS depth
			FROM event_causes WHERE cause_id = $1
		  UNION
			SELECT ec.event_id, d.depth + 1
			FROM descendants d
			JOIN event_causes ec ON ec.cause_id = d.id
			WHERE d.depth < $2
		)
		SELECT DISTINCT e.id, e.version, e.event_type, e.timestamp_nanos, e.source,
		       e.conversation_id, e.hash, e.prev_hash, e.signature, e.content_json
		FROM descendants d
		JOIN events e ON e.id = d.id`,
		id.Value(), maxDepth)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("descendants query: %v", err)}
	}
	defer rows.Close()

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
}

func (s *PostgresStore) EdgesFrom(id types.ActorID, edgeType event.EdgeType) ([]event.Edge, error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx,
		`SELECT id, from_actor, to_actor, edge_type, weight, direction, scope, created_at_nanos, expires_at_nanos
		 FROM edges WHERE from_actor = $1 AND edge_type = $2`,
		id.Value(), string(edgeType))
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("edges from: %v", err)}
	}
	defer rows.Close()
	return scanEdges(rows)
}

func (s *PostgresStore) EdgesTo(id types.ActorID, edgeType event.EdgeType) ([]event.Edge, error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx,
		`SELECT id, from_actor, to_actor, edge_type, weight, direction, scope, created_at_nanos, expires_at_nanos
		 FROM edges WHERE to_actor = $1 AND edge_type = $2`,
		id.Value(), string(edgeType))
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("edges to: %v", err)}
	}
	defer rows.Close()
	return scanEdges(rows)
}

func (s *PostgresStore) EdgeBetween(from types.ActorID, to types.ActorID, edgeType event.EdgeType) (types.Option[event.Edge], error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx,
		`SELECT id, from_actor, to_actor, edge_type, weight, direction, scope, created_at_nanos, expires_at_nanos
		 FROM edges WHERE from_actor = $1 AND to_actor = $2 AND edge_type = $3
		 ORDER BY created_at_nanos DESC LIMIT 1`,
		from.Value(), to.Value(), string(edgeType))
	if err != nil {
		return types.None[event.Edge](), &store.StoreUnavailableError{Reason: fmt.Sprintf("edge between: %v", err)}
	}
	defer rows.Close()

	edges, err := scanEdges(rows)
	if err != nil {
		return types.None[event.Edge](), err
	}
	if len(edges) == 0 {
		return types.None[event.Edge](), nil
	}
	return types.Some(edges[0]), nil
}

func (s *PostgresStore) Count() (int, error) {
	ctx := context.Background()
	var count int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil {
		return 0, &store.StoreUnavailableError{Reason: fmt.Sprintf("count: %v", err)}
	}
	return count, nil
}

func (s *PostgresStore) VerifyChain() (event.ChainVerifiedContent, error) {
	ctx := context.Background()
	start := time.Now()

	rows, err := s.pool.Query(ctx,
		`SELECT id, version, event_type, timestamp_nanos, source, conversation_id,
		        hash, prev_hash, signature, content_json
		 FROM events ORDER BY seq ASC`)
	if err != nil {
		return event.ChainVerifiedContent{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("verify chain: %v", err)}
	}
	defer rows.Close()

	var prevHash string
	i := 0
	for rows.Next() {
		ev, err := scanEventFromRows(ctx, s.pool, rows)
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
		i++
	}

	ns := time.Since(start).Nanoseconds()
	if ns < 0 {
		ns = 0
	}
	dur := types.MustDuration(ns)
	return event.ChainVerifiedContent{
		Valid:    true,
		Length:   i,
		Duration: dur,
	}, nil
}

func (s *PostgresStore) Close() error {
	if s.ownsPool {
		s.pool.Close()
	}
	return nil
}

// --- internal helpers ---

// getEvent fetches and reconstructs a single event by ID.
func (s *PostgresStore) getEvent(ctx context.Context, id types.EventID) (event.Event, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, version, event_type, timestamp_nanos, source, conversation_id,
		        hash, prev_hash, signature, content_json
		 FROM events WHERE id = $1`, id.Value())
	ev, err := scanEvent(ctx, s.pool, row)
	if err == pgx.ErrNoRows {
		return event.Event{}, &store.EventNotFoundError{ID: id}
	}
	return ev, err
}

// paginateReverse returns events in reverse chronological order (most recent first).
// filterClause is a SQL WHERE clause fragment like "event_type = $%d" where %d is
// replaced with the next parameter index. filterArgs are the values for the filter.
func (s *PostgresStore) paginateReverse(ctx context.Context, filterClause string, filterArgs []any, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	args := make([]any, 0, len(filterArgs)+2)
	paramIdx := 1

	// Build WHERE clauses.
	var whereParts []string

	if filterClause != "" {
		clause := fmt.Sprintf(filterClause, paramIdx)
		whereParts = append(whereParts, clause)
		args = append(args, filterArgs...)
		paramIdx += len(filterArgs)
	}

	if after.IsSome() {
		cursor := after.Unwrap()
		// Find the seq of the cursor event within the filtered set.
		var cursorSeq int64
		lookupQuery := "SELECT seq FROM events WHERE id = $1"
		err := s.pool.QueryRow(ctx, lookupQuery, cursor.Value()).Scan(&cursorSeq)
		if err == pgx.ErrNoRows {
			return types.NewPage[event.Event](nil, types.None[types.Cursor](), false),
				&store.InvalidCursorError{Cursor: cursor.Value()}
		}
		if err != nil {
			return types.Page[event.Event]{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("cursor lookup: %v", err)}
		}
		whereParts = append(whereParts, fmt.Sprintf("seq < $%d", paramIdx))
		args = append(args, cursorSeq)
		paramIdx++
	}

	whereSQL := ""
	if len(whereParts) > 0 {
		whereSQL = "WHERE "
		for i, part := range whereParts {
			if i > 0 {
				whereSQL += " AND "
			}
			whereSQL += part
		}
	}

	// Query limit+1 to determine hasMore.
	query := fmt.Sprintf(
		`SELECT id, version, event_type, timestamp_nanos, source, conversation_id,
		        hash, prev_hash, signature, content_json
		 FROM events %s ORDER BY seq DESC LIMIT $%d`, whereSQL, paramIdx)
	args = append(args, limit+1)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return types.Page[event.Event]{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("paginate query: %v", err)}
	}
	defer rows.Close()

	var items []event.Event
	for rows.Next() {
		ev, err := scanEventFromRows(ctx, s.pool, rows)
		if err != nil {
			return types.Page[event.Event]{}, err
		}
		items = append(items, ev)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	var cursorOpt types.Option[types.Cursor]
	if hasMore && len(items) > 0 {
		c := types.MustCursor(items[len(items)-1].ID().Value())
		cursorOpt = types.Some(c)
	}

	return types.NewPage(items, cursorOpt, hasMore), nil
}

// scanEvent scans a single row into an Event. Loads causes from event_causes table.
func scanEvent(ctx context.Context, pool *pgxpool.Pool, row pgx.Row) (event.Event, error) {
	var (
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
	)
	err := row.Scan(&id, &version, &eventType, &timestampNanos, &source,
		&conversationID, &hash, &prevHash, &signature, &contentJSON)
	if err != nil {
		return event.Event{}, err
	}
	return reconstructEvent(ctx, pool, id, version, eventType, timestampNanos, source,
		conversationID, hash, prevHash, signature, contentJSON)
}

// scanEventFromRows scans the current row from pgx.Rows into an Event.
func scanEventFromRows(ctx context.Context, pool *pgxpool.Pool, rows pgx.Rows) (event.Event, error) {
	var (
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
	)
	err := rows.Scan(&id, &version, &eventType, &timestampNanos, &source,
		&conversationID, &hash, &prevHash, &signature, &contentJSON)
	if err != nil {
		return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("scan event: %v", err)}
	}
	return reconstructEvent(ctx, pool, id, version, eventType, timestampNanos, source,
		conversationID, hash, prevHash, signature, contentJSON)
}

// reconstructEvent rebuilds an Event from database columns.
func reconstructEvent(
	ctx context.Context, pool *pgxpool.Pool,
	id string, version int, eventType string, timestampNanos int64,
	source, conversationID, hash, prevHash string,
	signature, contentJSON []byte,
) (event.Event, error) {
	evID := types.MustEventID(id)
	evType := types.MustEventType(eventType)
	ts := types.NewTimestamp(time.Unix(0, timestampNanos))
	src := types.MustActorID(source)
	convID := types.MustConversationID(conversationID)
	h := types.MustHash(hash)
	ph := types.MustHash(prevHash)
	sig := types.MustSignature(signature)

	content, err := unmarshalContent(eventType, contentJSON)
	if err != nil {
		return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("unmarshal content: %v", err)}
	}

	// Load causes from event_causes table.
	causeRows, err := pool.Query(ctx, "SELECT cause_id FROM event_causes WHERE event_id = $1", id)
	if err != nil {
		return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("load causes: %v", err)}
	}
	defer causeRows.Close()

	var causes []types.EventID
	for causeRows.Next() {
		var causeID string
		if err := causeRows.Scan(&causeID); err != nil {
			return event.Event{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("scan cause: %v", err)}
		}
		causes = append(causes, types.MustEventID(causeID))
	}

	if evType == event.EventTypeSystemBootstrapped {
		bc, ok := content.(event.BootstrapContent)
		if !ok {
			return event.Event{}, &store.StoreUnavailableError{Reason: "bootstrap content type mismatch"}
		}
		return event.NewBootstrapEvent(version, evID, evType, ts, src, bc, convID, h, sig), nil
	}

	return event.NewEvent(version, evID, evType, ts, src, content, causes, convID, h, ph, sig), nil
}

// unmarshalContent deserializes JSON into the correct EventContent type.
// Delegates to the event package's data-driven registry.
func unmarshalContent(eventType string, data []byte) (event.EventContent, error) {
	return event.UnmarshalContent(eventType, data)
}

// scanEdges scans rows into Edge slices.
func scanEdges(rows pgx.Rows) ([]event.Edge, error) {
	var result []event.Edge
	for rows.Next() {
		var (
			id           string
			fromActor    string
			toActor      string
			edgeType     string
			weight       float64
			direction    string
			scope        *string
			createdNanos int64
			expiresNanos *int64
		)
		if err := rows.Scan(&id, &fromActor, &toActor, &edgeType, &weight, &direction, &scope, &createdNanos, &expiresNanos); err != nil {
			return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("scan edge: %v", err)}
		}

		edgeID := types.MustEdgeID(id)
		from := types.MustActorID(fromActor)
		to := types.MustActorID(toActor)
		et := event.EdgeType(edgeType)
		w := types.MustWeight(weight)
		dir := event.EdgeDirection(direction)

		var scopeOpt types.Option[types.DomainScope]
		if scope != nil {
			scopeOpt = types.Some(types.MustDomainScope(*scope))
		}

		createdAt := types.NewTimestamp(time.Unix(0, createdNanos))
		var expiresOpt types.Option[types.Timestamp]
		if expiresNanos != nil {
			expiresOpt = types.Some(types.NewTimestamp(time.Unix(0, *expiresNanos)))
		}

		edge, err := event.NewEdge(edgeID, from, to, et, w, dir, scopeOpt, nil, createdAt, expiresOpt)
		if err != nil {
			return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("reconstruct edge: %v", err)}
		}
		result = append(result, edge)
	}
	return result, nil
}

// Truncate removes all data from the store tables. Used for testing.
func (s *PostgresStore) Truncate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, "TRUNCATE events, event_causes, edges RESTART IDENTITY CASCADE")
	return err
}
