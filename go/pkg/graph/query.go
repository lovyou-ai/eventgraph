package graph

import (
	"context"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/trust"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Query provides a query interface for the graph.
type Query struct {
	store      store.Store
	actorStore actor.IActorStore
	trustModel trust.ITrustModel
}

// Recent returns the most recent events.
func (q *Query) Recent(limit int) (types.Page[event.Event], error) {
	return q.store.Recent(limit, types.None[types.Cursor]())
}

// ByType returns events of a given type.
func (q *Query) ByType(eventType types.EventType, limit int) (types.Page[event.Event], error) {
	return q.store.ByType(eventType, limit, types.None[types.Cursor]())
}

// BySource returns events from a given source.
func (q *Query) BySource(source types.ActorID, limit int) (types.Page[event.Event], error) {
	return q.store.BySource(source, limit, types.None[types.Cursor]())
}

// ByConversation returns events in a conversation.
func (q *Query) ByConversation(id types.ConversationID, limit int) (types.Page[event.Event], error) {
	return q.store.ByConversation(id, limit, types.None[types.Cursor]())
}

// Ancestors returns causal ancestors of an event.
func (q *Query) Ancestors(id types.EventID, maxDepth int) ([]event.Event, error) {
	return q.store.Ancestors(id, maxDepth)
}

// Descendants returns causal descendants of an event.
func (q *Query) Descendants(id types.EventID, maxDepth int) ([]event.Event, error) {
	return q.store.Descendants(id, maxDepth)
}

// TrustScore returns trust metrics for an actor.
func (q *Query) TrustScore(ctx context.Context, a actor.IActor) (event.TrustMetrics, error) {
	return q.trustModel.Score(ctx, a)
}

// TrustBetween returns trust from one actor toward another.
func (q *Query) TrustBetween(ctx context.Context, from actor.IActor, to actor.IActor) (event.TrustMetrics, error) {
	return q.trustModel.Between(ctx, from, to)
}

// Actor returns an actor by ID.
func (q *Query) Actor(id types.ActorID) (actor.IActor, error) {
	return q.actorStore.Get(id)
}

// EventCount returns the total number of events.
func (q *Query) EventCount() (int, error) {
	return q.store.Count()
}
