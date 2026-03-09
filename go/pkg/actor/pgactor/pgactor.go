// Package pgactor implements a PostgreSQL-backed IActorStore.
package pgactor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

const schema = `
CREATE TABLE IF NOT EXISTS actors (
	id TEXT PRIMARY KEY,
	public_key BYTEA UNIQUE NOT NULL,
	display_name TEXT NOT NULL,
	actor_type TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'Active',
	metadata_json JSONB NOT NULL DEFAULT '{}',
	created_at_nanos BIGINT NOT NULL,
	seq BIGSERIAL
);

CREATE INDEX IF NOT EXISTS idx_actors_status ON actors(status);
CREATE INDEX IF NOT EXISTS idx_actors_type ON actors(actor_type);
CREATE INDEX IF NOT EXISTS idx_actors_seq ON actors(seq);
`

// PostgresActorStore implements actor.IActorStore backed by PostgreSQL.
type PostgresActorStore struct {
	pool *pgxpool.Pool
}

// NewPostgresActorStore creates a PostgresActorStore connected to the given Postgres instance.
// It creates the schema if it doesn't exist.
func NewPostgresActorStore(ctx context.Context, connString string) (*PostgresActorStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("connect: %v", err)}
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("schema: %v", err)}
	}
	return &PostgresActorStore{pool: pool}, nil
}

// NewPostgresActorStoreFromPool creates a PostgresActorStore from an existing connection pool.
// It creates the schema if it doesn't exist.
func NewPostgresActorStoreFromPool(ctx context.Context, pool *pgxpool.Pool) (*PostgresActorStore, error) {
	if _, err := pool.Exec(ctx, schema); err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("schema: %v", err)}
	}
	return &PostgresActorStore{pool: pool}, nil
}

func (s *PostgresActorStore) Register(publicKey types.PublicKey, displayName string, actorType event.ActorType) (actor.IActor, error) {
	ctx := context.Background()

	// Check for existing actor by public key.
	existing, err := s.getByPublicKey(ctx, publicKey)
	if err == nil {
		return existing, nil
	}

	// Derive actor ID from public key (same algorithm as InMemoryActorStore).
	id := deriveActorID(publicKey)
	now := types.Now()

	metadataJSON, _ := json.Marshal(map[string]any{})

	_, err = s.pool.Exec(ctx,
		`INSERT INTO actors (id, public_key, display_name, actor_type, status, metadata_json, created_at_nanos)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (public_key) DO NOTHING`,
		id.Value(), publicKey.Bytes(), displayName, string(actorType),
		string(types.ActorStatusActive), metadataJSON, now.UnixNano(),
	)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("insert actor: %v", err)}
	}

	// Re-read to handle race (ON CONFLICT DO NOTHING means we might not have inserted).
	return s.getByPublicKey(ctx, publicKey)
}

func (s *PostgresActorStore) Get(id types.ActorID) (actor.IActor, error) {
	ctx := context.Background()
	row := s.pool.QueryRow(ctx,
		`SELECT id, public_key, display_name, actor_type, status, metadata_json, created_at_nanos
		 FROM actors WHERE id = $1`, id.Value())
	a, err := scanActor(row)
	if err != nil {
		if isNotFound(err) {
			return nil, &store.ActorNotFoundError{ID: id}
		}
		return nil, err
	}
	return a, nil
}

func (s *PostgresActorStore) GetByPublicKey(publicKey types.PublicKey) (actor.IActor, error) {
	return s.getByPublicKey(context.Background(), publicKey)
}

func (s *PostgresActorStore) getByPublicKey(ctx context.Context, publicKey types.PublicKey) (actor.IActor, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, public_key, display_name, actor_type, status, metadata_json, created_at_nanos
		 FROM actors WHERE public_key = $1`, publicKey.Bytes())
	a, err := scanActor(row)
	if err != nil {
		var notFound *store.ActorNotFoundError
		if isNotFound(err) {
			keyHex := hex.EncodeToString(publicKey.Bytes())
			return nil, &store.ActorKeyNotFoundError{KeyHex: keyHex}
		}
		_ = notFound
		return nil, err
	}
	return a, nil
}

func (s *PostgresActorStore) Update(id types.ActorID, updates actor.ActorUpdate) (actor.IActor, error) {
	ctx := context.Background()

	// Read current state.
	current, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	displayName := current.DisplayName()
	if updates.DisplayName.IsSome() {
		displayName = updates.DisplayName.Unwrap()
	}

	metadata := current.Metadata()
	if updates.Metadata.IsSome() {
		for k, v := range updates.Metadata.Unwrap() {
			metadata[k] = v
		}
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("marshal metadata: %v", err)}
	}

	_, err = s.pool.Exec(ctx,
		`UPDATE actors SET display_name = $1, metadata_json = $2 WHERE id = $3`,
		displayName, metadataJSON, id.Value())
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("update actor: %v", err)}
	}

	return s.Get(id)
}

func (s *PostgresActorStore) List(filter actor.ActorFilter) (types.Page[actor.IActor], error) {
	ctx := context.Background()

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	var whereParts []string
	var args []any
	paramIdx := 1

	if filter.Status.IsSome() {
		whereParts = append(whereParts, fmt.Sprintf("status = $%d", paramIdx))
		args = append(args, string(filter.Status.Unwrap()))
		paramIdx++
	}

	if filter.Type.IsSome() {
		whereParts = append(whereParts, fmt.Sprintf("actor_type = $%d", paramIdx))
		args = append(args, string(filter.Type.Unwrap()))
		paramIdx++
	}

	if filter.After.IsSome() {
		cursor := filter.After.Unwrap()
		// Find the seq of the cursor actor.
		var cursorSeq int64
		err := s.pool.QueryRow(ctx, "SELECT seq FROM actors WHERE id = $1", cursor.Value()).Scan(&cursorSeq)
		if err == pgx.ErrNoRows {
			return types.NewPage[actor.IActor](nil, types.None[types.Cursor](), false),
				&store.InvalidCursorError{Cursor: cursor.Value()}
		}
		if err != nil {
			return types.Page[actor.IActor]{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("cursor lookup: %v", err)}
		}
		whereParts = append(whereParts, fmt.Sprintf("seq > $%d", paramIdx))
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
		`SELECT id, public_key, display_name, actor_type, status, metadata_json, created_at_nanos
		 FROM actors %s ORDER BY seq ASC LIMIT $%d`, whereSQL, paramIdx)
	args = append(args, limit+1)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return types.Page[actor.IActor]{}, &store.StoreUnavailableError{Reason: fmt.Sprintf("list query: %v", err)}
	}
	defer rows.Close()

	var items []actor.IActor
	for rows.Next() {
		a, err := scanActorFromRows(rows)
		if err != nil {
			return types.Page[actor.IActor]{}, err
		}
		items = append(items, a)
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

func (s *PostgresActorStore) Suspend(id types.ActorID, reason types.EventID) (actor.IActor, error) {
	return s.transitionStatus(id, types.ActorStatusSuspended, reason)
}

func (s *PostgresActorStore) Reactivate(id types.ActorID, reason types.EventID) (actor.IActor, error) {
	return s.transitionStatus(id, types.ActorStatusActive, reason)
}

func (s *PostgresActorStore) Memorial(id types.ActorID, reason types.EventID) (actor.IActor, error) {
	return s.transitionStatus(id, types.ActorStatusMemorial, reason)
}

func (s *PostgresActorStore) transitionStatus(id types.ActorID, target types.ActorStatus, reason types.EventID) (actor.IActor, error) {
	ctx := context.Background()

	// Read current state to validate transition.
	current, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	newStatus, err := current.Status().TransitionTo(target)
	if err != nil {
		return nil, err
	}

	_, err = s.pool.Exec(ctx,
		`UPDATE actors SET status = $1 WHERE id = $2`,
		string(newStatus), id.Value())
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("update status: %v", err)}
	}
	_ = reason // recorded on the event graph, not stored here

	return s.Get(id)
}

// Close closes the connection pool.
func (s *PostgresActorStore) Close() {
	s.pool.Close()
}

// Truncate removes all data from the actors table. Used for testing.
func (s *PostgresActorStore) Truncate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, "TRUNCATE actors RESTART IDENTITY CASCADE")
	return err
}

// --- internal helpers ---

func scanActor(row pgx.Row) (actor.IActor, error) {
	var (
		id             string
		publicKey      []byte
		displayName    string
		actorType      string
		status         string
		metadataJSON   []byte
		createdAtNanos int64
	)
	err := row.Scan(&id, &publicKey, &displayName, &actorType, &status, &metadataJSON, &createdAtNanos)
	if err == pgx.ErrNoRows {
		return nil, &store.ActorNotFoundError{ID: types.MustActorID("actor_unknown")}
	}
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("scan actor: %v", err)}
	}
	return reconstructActor(id, publicKey, displayName, actorType, status, metadataJSON, createdAtNanos)
}

func scanActorFromRows(rows pgx.Rows) (actor.IActor, error) {
	var (
		id             string
		publicKey      []byte
		displayName    string
		actorType      string
		status         string
		metadataJSON   []byte
		createdAtNanos int64
	)
	err := rows.Scan(&id, &publicKey, &displayName, &actorType, &status, &metadataJSON, &createdAtNanos)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("scan actor: %v", err)}
	}
	return reconstructActor(id, publicKey, displayName, actorType, status, metadataJSON, createdAtNanos)
}

func reconstructActor(
	id string, publicKey []byte, displayName, actorType, status string,
	metadataJSON []byte, createdAtNanos int64,
) (actor.IActor, error) {
	actorID := types.MustActorID(id)
	pk := types.MustPublicKey(publicKey)
	ts := types.NewTimestamp(time.Unix(0, createdAtNanos))

	at, err := parseActorType(actorType)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("invalid actor type: %s", actorType)}
	}

	st, err := parseActorStatus(status)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("invalid actor status: %s", status)}
	}

	var metadata map[string]any
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("unmarshal metadata: %v", err)}
	}

	return actor.NewActor(actorID, pk, displayName, at, metadata, ts, st), nil
}

func parseActorType(s string) (event.ActorType, error) {
	switch event.ActorType(s) {
	case event.ActorTypeHuman:
		return event.ActorTypeHuman, nil
	case event.ActorTypeAI:
		return event.ActorTypeAI, nil
	case event.ActorTypeSystem:
		return event.ActorTypeSystem, nil
	case event.ActorTypeCommittee:
		return event.ActorTypeCommittee, nil
	case event.ActorTypeRulesEngine:
		return event.ActorTypeRulesEngine, nil
	default:
		return "", fmt.Errorf("unknown actor type: %s", s)
	}
}

func parseActorStatus(s string) (types.ActorStatus, error) {
	status, err := types.NewActorStatus(s)
	if err != nil {
		return "", fmt.Errorf("unknown actor status: %s", s)
	}
	return status, nil
}

func isNotFound(err error) bool {
	_, ok := err.(*store.ActorNotFoundError)
	return ok
}

// deriveActorID derives an ActorID from a public key using SHA-256.
// Same algorithm as InMemoryActorStore for consistency.
func deriveActorID(pk types.PublicKey) types.ActorID {
	h := sha256.Sum256(pk.Bytes())
	id := fmt.Sprintf("actor_%s", hex.EncodeToString(h[:16]))
	return types.MustActorID(id)
}
