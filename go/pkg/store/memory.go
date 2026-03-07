package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// InMemoryStore implements Store with in-memory storage.
// Safe for concurrent access. Chain head is locked during Append.
type InMemoryStore struct {
	mu     sync.RWMutex
	events []event.Event                      // ordered by insertion
	byID   map[types.EventID]int              // eventID → index in events
	byType map[string][]int                   // eventType → indices
	bySrc  map[types.ActorID][]int            // source → indices
	byConv map[types.ConversationID][]int     // conversationID → indices
	edges  []event.Edge                       // all edges
}

// NewInMemoryStore creates a new empty InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		byID:   make(map[types.EventID]int),
		byType: make(map[string][]int),
		bySrc:  make(map[types.ActorID][]int),
		byConv: make(map[types.ConversationID][]int),
	}
}

func (s *InMemoryStore) Append(ev event.Event) (event.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Idempotency: if same ID exists, return it
	if idx, ok := s.byID[ev.ID()]; ok {
		return s.events[idx], nil
	}

	// Verify PrevHash matches chain head
	if len(s.events) == 0 {
		if ev.PrevHash() != types.ZeroHash() {
			return event.Event{}, &ChainIntegrityViolationError{
				Position: 0,
				Expected: types.ZeroHash(),
				Actual:   ev.PrevHash(),
			}
		}
	} else {
		headHash := s.events[len(s.events)-1].Hash()
		if ev.PrevHash() != headHash {
			return event.Event{}, &ChainIntegrityViolationError{
				Position: len(s.events),
				Expected: headHash,
				Actual:   ev.PrevHash(),
			}
		}
	}

	// Recompute hash and verify
	canonical := event.CanonicalForm(ev)
	computed, err := event.ComputeHash(canonical)
	if err != nil {
		return event.Event{}, err
	}
	if computed != ev.Hash() {
		return event.Event{}, &HashMismatchError{
			EventID:  ev.ID(),
			Computed: computed,
			Stored:   ev.Hash(),
		}
	}

	// Store the event
	idx := len(s.events)
	s.events = append(s.events, ev)
	s.byID[ev.ID()] = idx
	s.byType[ev.Type().Value()] = append(s.byType[ev.Type().Value()], idx)
	s.bySrc[ev.Source()] = append(s.bySrc[ev.Source()], idx)
	s.byConv[ev.ConversationID()] = append(s.byConv[ev.ConversationID()], idx)

	// Index edge-creating events
	if ev.Type() == event.EventTypeEdgeCreated {
		if ec, ok := ev.Content().(event.EdgeCreatedContent); ok {
			edgeID := types.MustEdgeID(ev.ID().Value())
			// metadata is nil — EdgeCreatedContent does not carry typed metadata.
			// Callers must nil-check Edge.Metadata() before calling methods on it.
			edge, edgeErr := event.NewEdge(
				edgeID, ec.From, ec.To, ec.EdgeType, ec.Weight, ec.Direction,
				ec.Scope, nil, ev.Timestamp(), ec.ExpiresAt,
			)
			if edgeErr != nil {
				return event.Event{}, fmt.Errorf("index edge: %w", edgeErr)
			}
			s.edges = append(s.edges, edge)
		}
	}

	return ev, nil
}

func (s *InMemoryStore) Get(id types.EventID) (event.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, ok := s.byID[id]
	if !ok {
		return event.Event{}, &EventNotFoundError{ID: id}
	}
	return s.events[idx], nil
}

func (s *InMemoryStore) Head() (types.Option[event.Event], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.events) == 0 {
		return types.None[event.Event](), nil
	}
	return types.Some(s.events[len(s.events)-1]), nil
}

func (s *InMemoryStore) Recent(limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.paginateReverse(s.allIndices(), limit, after)
}

func (s *InMemoryStore) ByType(eventType types.EventType, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indices := s.byType[eventType.Value()]
	return s.paginateReverse(indices, limit, after)
}

func (s *InMemoryStore) BySource(source types.ActorID, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indices := s.bySrc[source]
	return s.paginateReverse(indices, limit, after)
}

func (s *InMemoryStore) ByConversation(id types.ConversationID, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indices := s.byConv[id]
	return s.paginateReverse(indices, limit, after)
}

func (s *InMemoryStore) Since(afterID types.EventID, limit int) (types.Page[event.Event], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	startIdx, ok := s.byID[afterID]
	if !ok {
		return types.Page[event.Event]{}, &EventNotFoundError{ID: afterID}
	}

	var items []event.Event
	for i := startIdx + 1; i < len(s.events) && len(items) < limit; i++ {
		items = append(items, s.events[i])
	}

	hasMore := startIdx+1+limit < len(s.events)
	var cursor types.Option[types.Cursor]
	if hasMore && len(items) > 0 {
		c := types.MustCursor(items[len(items)-1].ID().Value())
		cursor = types.Some(c)
	}

	return types.NewPage(items, cursor, hasMore), nil
}

func (s *InMemoryStore) Ancestors(id types.EventID, maxDepth int) ([]event.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, ok := s.byID[id]
	if !ok {
		return nil, &EventNotFoundError{ID: id}
	}

	visited := make(map[types.EventID]bool)
	var result []event.Event
	s.collectAncestors(idx, maxDepth, 0, visited, &result)
	return result, nil
}

func (s *InMemoryStore) collectAncestors(idx int, maxDepth, depth int, visited map[types.EventID]bool, result *[]event.Event) {
	if depth >= maxDepth {
		return
	}
	ev := s.events[idx]
	for _, causeID := range ev.Causes() {
		if visited[causeID] {
			continue
		}
		causeIdx, ok := s.byID[causeID]
		if !ok {
			continue
		}
		visited[causeID] = true
		*result = append(*result, s.events[causeIdx])
		s.collectAncestors(causeIdx, maxDepth, depth+1, visited, result)
	}
}

func (s *InMemoryStore) Descendants(id types.EventID, maxDepth int) ([]event.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.byID[id]; !ok {
		return nil, &EventNotFoundError{ID: id}
	}

	visited := make(map[types.EventID]bool)
	var result []event.Event
	s.collectDescendants(id, maxDepth, 0, visited, &result)
	return result, nil
}

func (s *InMemoryStore) collectDescendants(id types.EventID, maxDepth, depth int, visited map[types.EventID]bool, result *[]event.Event) {
	if depth >= maxDepth {
		return
	}
	for _, ev := range s.events {
		if visited[ev.ID()] {
			continue
		}
		for _, causeID := range ev.Causes() {
			if causeID == id {
				visited[ev.ID()] = true
				*result = append(*result, ev)
				s.collectDescendants(ev.ID(), maxDepth, depth+1, visited, result)
				break
			}
		}
	}
}

func (s *InMemoryStore) EdgesFrom(id types.ActorID, edgeType event.EdgeType) ([]event.Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []event.Edge
	for _, e := range s.edges {
		if e.From() == id && e.Type() == edgeType {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *InMemoryStore) EdgesTo(id types.ActorID, edgeType event.EdgeType) ([]event.Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []event.Edge
	for _, e := range s.edges {
		if e.To() == id && e.Type() == edgeType {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *InMemoryStore) EdgeBetween(from types.ActorID, to types.ActorID, edgeType event.EdgeType) (types.Option[event.Edge], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := len(s.edges) - 1; i >= 0; i-- {
		e := s.edges[i]
		if e.From() == from && e.To() == to && e.Type() == edgeType {
			return types.Some(e), nil
		}
	}
	return types.None[event.Edge](), nil
}

func (s *InMemoryStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events), nil
}

func (s *InMemoryStore) VerifyChain() (event.ChainVerifiedContent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Now()

	for i, ev := range s.events {
		// Verify prev_hash
		if i == 0 {
			if ev.PrevHash() != types.ZeroHash() {
				return event.ChainVerifiedContent{Valid: false, Length: i}, nil
			}
		} else {
			if ev.PrevHash() != s.events[i-1].Hash() {
				return event.ChainVerifiedContent{Valid: false, Length: i}, nil
			}
		}

		// Verify hash
		canonical := event.CanonicalForm(ev)
		computed, err := event.ComputeHash(canonical)
		if err != nil {
			return event.ChainVerifiedContent{Valid: false, Length: i}, nil
		}
		if computed != ev.Hash() {
			return event.ChainVerifiedContent{Valid: false, Length: i}, nil
		}
	}

	elapsed := time.Since(start)
	dur := types.MustDuration(elapsed.Nanoseconds())
	return event.ChainVerifiedContent{
		Valid:    true,
		Length:   len(s.events),
		Duration: dur,
	}, nil
}

func (s *InMemoryStore) Close() error {
	return nil
}

// --- pagination helpers ---

func (s *InMemoryStore) allIndices() []int {
	indices := make([]int, len(s.events))
	for i := range indices {
		indices[i] = i
	}
	return indices
}

func (s *InMemoryStore) paginateReverse(indices []int, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error) {
	if limit <= 0 {
		limit = 100
	}

	// Reverse order (most recent first)
	startPos := len(indices) - 1
	if after.IsSome() {
		cursor := after.Unwrap()
		found := false
		for i := len(indices) - 1; i >= 0; i-- {
			if s.events[indices[i]].ID().Value() == cursor.Value() {
				startPos = i - 1
				found = true
				break
			}
		}
		if !found {
			return types.NewPage[event.Event](nil, types.None[types.Cursor](), false),
				fmt.Errorf("invalid cursor: event %q not found", cursor.Value())
		}
	}

	var items []event.Event
	for i := startPos; i >= 0 && len(items) < limit; i-- {
		items = append(items, s.events[indices[i]])
	}

	hasMore := startPos-limit >= 0
	var cursor types.Option[types.Cursor]
	if hasMore && len(items) > 0 {
		c := types.MustCursor(items[len(items)-1].ID().Value())
		cursor = types.Some(c)
	}

	return types.NewPage(items, cursor, hasMore), nil
}
