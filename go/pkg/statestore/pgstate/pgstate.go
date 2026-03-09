// Package pgstate implements IStateStore backed by PostgreSQL.
package pgstate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/statestore"
)

// Compile-time interface check.
var _ statestore.IStateStore = (*PostgresStateStore)(nil)

const schema = `
CREATE TABLE IF NOT EXISTS state (
	scope TEXT NOT NULL,
	key TEXT NOT NULL,
	value_json JSONB NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (scope, key)
);

CREATE INDEX IF NOT EXISTS idx_state_scope ON state(scope);
`

// PostgresStateStore implements statestore.IStateStore backed by PostgreSQL.
type PostgresStateStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStateStore creates a PostgresStateStore connected to the given Postgres instance.
func NewPostgresStateStore(ctx context.Context, connString string) (*PostgresStateStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("connect: %v", err)}
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("schema: %v", err)}
	}
	return &PostgresStateStore{pool: pool}, nil
}

// NewPostgresStateStoreFromPool creates a PostgresStateStore from an existing connection pool.
func NewPostgresStateStoreFromPool(ctx context.Context, pool *pgxpool.Pool) (*PostgresStateStore, error) {
	if _, err := pool.Exec(ctx, schema); err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("schema: %v", err)}
	}
	return &PostgresStateStore{pool: pool}, nil
}

func (s *PostgresStateStore) Get(scope, key string) (json.RawMessage, error) {
	ctx := context.Background()
	var value []byte
	err := s.pool.QueryRow(ctx,
		`SELECT value_json FROM state WHERE scope = $1 AND key = $2`,
		scope, key).Scan(&value)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("get: %v", err)}
	}
	return json.RawMessage(value), nil
}

func (s *PostgresStateStore) Put(scope, key string, value json.RawMessage) error {
	ctx := context.Background()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO state (scope, key, value_json)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (scope, key)
		 DO UPDATE SET value_json = $3, updated_at = NOW()`,
		scope, key, []byte(value))
	if err != nil {
		return &store.StoreUnavailableError{Reason: fmt.Sprintf("put: %v", err)}
	}
	return nil
}

func (s *PostgresStateStore) Delete(scope, key string) error {
	ctx := context.Background()
	_, err := s.pool.Exec(ctx,
		`DELETE FROM state WHERE scope = $1 AND key = $2`,
		scope, key)
	if err != nil {
		return &store.StoreUnavailableError{Reason: fmt.Sprintf("delete: %v", err)}
	}
	return nil
}

func (s *PostgresStateStore) List(scope string) (map[string]json.RawMessage, error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx,
		`SELECT key, value_json FROM state WHERE scope = $1`, scope)
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("list: %v", err)}
	}
	defer rows.Close()

	result := make(map[string]json.RawMessage)
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("scan: %v", err)}
		}
		result[key] = json.RawMessage(value)
	}
	return result, nil
}

func (s *PostgresStateStore) ListScopes(prefix string) ([]string, error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT scope FROM state WHERE scope LIKE $1`,
		prefix+"%")
	if err != nil {
		return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("list scopes: %v", err)}
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var scope string
		if err := rows.Scan(&scope); err != nil {
			return nil, &store.StoreUnavailableError{Reason: fmt.Sprintf("scan scope: %v", err)}
		}
		result = append(result, scope)
	}
	return result, nil
}

// Close closes the connection pool.
func (s *PostgresStateStore) Close() {
	s.pool.Close()
}

// Truncate removes all state. For testing.
func (s *PostgresStateStore) Truncate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, "TRUNCATE state")
	return err
}
