package actor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// InMemoryActorStore implements IActorStore with in-memory storage.
// Safe for concurrent access.
type InMemoryActorStore struct {
	mu       sync.RWMutex
	actors   map[types.ActorID]Actor
	byKey    map[string]types.ActorID // hex(publicKey) → ActorID
	ordered  []types.ActorID          // insertion order for pagination
}

// NewInMemoryActorStore creates a new empty InMemoryActorStore.
func NewInMemoryActorStore() *InMemoryActorStore {
	return &InMemoryActorStore{
		actors:  make(map[types.ActorID]Actor),
		byKey:   make(map[string]types.ActorID),
		ordered: nil,
	}
}

func pubKeyHex(pk types.PublicKey) string {
	return hex.EncodeToString(pk.Bytes())
}

func deriveActorID(pk types.PublicKey) types.ActorID {
	h := sha256.Sum256(pk.Bytes())
	id := fmt.Sprintf("actor_%s", hex.EncodeToString(h[:16]))
	aid, _ := types.NewActorID(id)
	return aid
}

// Register creates a new actor or returns the existing one if the public key is already registered.
// Idempotent on public key: re-registration returns the existing actor without updating displayName or actorType.
// Use Update to modify actor properties after registration.
func (s *InMemoryActorStore) Register(publicKey types.PublicKey, displayName string, actorType event.ActorType) (IActor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyHex := pubKeyHex(publicKey)
	if existingID, ok := s.byKey[keyHex]; ok {
		a := s.actors[existingID]
		return a, nil
	}

	id := deriveActorID(publicKey)
	a := NewActor(id, publicKey, displayName, actorType, nil, types.Now(), types.ActorStatusActive)
	s.actors[id] = a
	s.byKey[keyHex] = id
	s.ordered = append(s.ordered, id)
	return a, nil
}

func (s *InMemoryActorStore) Get(id types.ActorID) (IActor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.actors[id]
	if !ok {
		return nil, &store.ActorNotFoundError{ID: id}
	}
	return a, nil
}

func (s *InMemoryActorStore) GetByPublicKey(publicKey types.PublicKey) (IActor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keyHex := pubKeyHex(publicKey)
	id, ok := s.byKey[keyHex]
	if !ok {
		return nil, &store.ActorKeyNotFoundError{KeyHex: keyHex}
	}
	return s.actors[id], nil
}

func (s *InMemoryActorStore) Update(id types.ActorID, updates ActorUpdate) (IActor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.actors[id]
	if !ok {
		return nil, &store.ActorNotFoundError{ID: id}
	}
	updated := a.withUpdates(updates)
	s.actors[id] = updated
	return updated, nil
}

func (s *InMemoryActorStore) List(filter ActorFilter) (types.Page[IActor], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	// Find start position
	startIdx := 0
	if filter.After.IsSome() {
		cursor := filter.After.Unwrap()
		found := false
		for i, id := range s.ordered {
			if id.Value() == cursor.Value() {
				startIdx = i + 1
				found = true
				break
			}
		}
		if !found {
			return types.NewPage[IActor](nil, types.None[types.Cursor](), false),
				&store.InvalidCursorError{Cursor: cursor.Value()}
		}
	}

	var items []IActor
	for i := startIdx; i < len(s.ordered) && len(items) < limit; i++ {
		a := s.actors[s.ordered[i]]
		if filter.Status.IsSome() && a.Status() != filter.Status.Unwrap() {
			continue
		}
		if filter.Type.IsSome() && a.Type() != filter.Type.Unwrap() {
			continue
		}
		items = append(items, a)
	}

	hasMore := false
	var cursor types.Option[types.Cursor]
	if len(items) == limit {
		// Check if there are more matching items
		lastActor := items[len(items)-1]
		lastIdx := 0
		for i, id := range s.ordered {
			if id == lastActor.ID() {
				lastIdx = i
				break
			}
		}
		for i := lastIdx + 1; i < len(s.ordered); i++ {
			a := s.actors[s.ordered[i]]
			if filter.Status.IsSome() && a.Status() != filter.Status.Unwrap() {
				continue
			}
			if filter.Type.IsSome() && a.Type() != filter.Type.Unwrap() {
				continue
			}
			hasMore = true
			break
		}
		if hasMore {
			c := types.MustCursor(lastActor.ID().Value())
			cursor = types.Some(c)
		}
	}

	return types.NewPage(items, cursor, hasMore), nil
}

func (s *InMemoryActorStore) Suspend(id types.ActorID, reason types.EventID) (IActor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.actors[id]
	if !ok {
		return nil, &store.ActorNotFoundError{ID: id}
	}
	newStatus, err := a.Status().TransitionTo(types.ActorStatusSuspended)
	if err != nil {
		return nil, err
	}
	updated := a.withStatus(newStatus)
	s.actors[id] = updated
	_ = reason // recorded on the event graph, not stored here
	return updated, nil
}

func (s *InMemoryActorStore) Reactivate(id types.ActorID, reason types.EventID) (IActor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.actors[id]
	if !ok {
		return nil, &store.ActorNotFoundError{ID: id}
	}
	newStatus, err := a.Status().TransitionTo(types.ActorStatusActive)
	if err != nil {
		return nil, err
	}
	updated := a.withStatus(newStatus)
	s.actors[id] = updated
	_ = reason // recorded on the event graph, not stored here
	return updated, nil
}

func (s *InMemoryActorStore) Memorial(id types.ActorID, reason types.EventID) (IActor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.actors[id]
	if !ok {
		return nil, &store.ActorNotFoundError{ID: id}
	}
	newStatus, err := a.Status().TransitionTo(types.ActorStatusMemorial)
	if err != nil {
		return nil, err
	}
	updated := a.withStatus(newStatus)
	s.actors[id] = updated
	_ = reason
	return updated, nil
}

// ActorCount returns the number of registered actors. For testing.
func (s *InMemoryActorStore) ActorCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.actors)
}

// AllActorIDs returns all actor IDs sorted alphabetically by ID value. For testing.
func (s *InMemoryActorStore) AllActorIDs() []types.ActorID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]types.ActorID, len(s.ordered))
	copy(result, s.ordered)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Value() < result[j].Value()
	})
	return result
}
