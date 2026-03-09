// Package statestore provides durable key-value storage for primitive state,
// agent memory, trust scores, and any other state that needs to survive restarts.
// One table, all concerns. Scope separates namespaces.
package statestore

import (
	"encoding/json"
	"sync"
)

// IStateStore is the persistence interface for primitive and agent state.
// Scope separates namespaces (e.g., "trust:actor_xxx", "agent:actor_xxx", "primitive:pattern").
// Implementations must be safe for concurrent access.
type IStateStore interface {
	// Get retrieves a value by scope and key. Returns nil, nil if not found.
	Get(scope, key string) (json.RawMessage, error)

	// Put stores a value by scope and key. Upserts.
	Put(scope, key string, value json.RawMessage) error

	// Delete removes a value by scope and key. No error if not found.
	Delete(scope, key string) error

	// List returns all key-value pairs for a scope.
	List(scope string) (map[string]json.RawMessage, error)

	// ListScopes returns all scopes matching a prefix (e.g., "trust:" returns all trust scopes).
	ListScopes(prefix string) ([]string, error)
}

// InMemoryStateStore implements IStateStore with in-memory storage.
// Safe for concurrent access.
type InMemoryStateStore struct {
	mu   sync.RWMutex
	data map[storeKey]json.RawMessage
}

type storeKey struct {
	scope string
	key   string
}

// NewInMemoryStateStore creates a new empty in-memory state store.
func NewInMemoryStateStore() *InMemoryStateStore {
	return &InMemoryStateStore{
		data: make(map[storeKey]json.RawMessage),
	}
}

func (s *InMemoryStateStore) Get(scope, key string) (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.data[storeKey{scope, key}]
	if !ok {
		return nil, nil
	}
	cp := make(json.RawMessage, len(v))
	copy(cp, v)
	return cp, nil
}

func (s *InMemoryStateStore) Put(scope, key string, value json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := make(json.RawMessage, len(value))
	copy(cp, value)
	s.data[storeKey{scope, key}] = cp
	return nil
}

func (s *InMemoryStateStore) Delete(scope, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, storeKey{scope, key})
	return nil
}

func (s *InMemoryStateStore) List(scope string) (map[string]json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]json.RawMessage)
	for k, v := range s.data {
		if k.scope == scope {
			cp := make(json.RawMessage, len(v))
			copy(cp, v)
			result[k.key] = cp
		}
	}
	return result, nil
}

func (s *InMemoryStateStore) ListScopes(prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	for k := range s.data {
		if len(k.scope) >= len(prefix) && k.scope[:len(prefix)] == prefix {
			seen[k.scope] = true
		}
	}
	result := make([]string, 0, len(seen))
	for scope := range seen {
		result = append(result, scope)
	}
	return result, nil
}
