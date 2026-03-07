package graph

import (
	"context"
	"fmt"
	"sync"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/authority"
	"github.com/lovyou-ai/eventgraph/go/pkg/bus"
	"github.com/lovyou-ai/eventgraph/go/pkg/decision"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/trust"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Config controls graph behaviour.
type Config struct {
	SubscriberBufferSize int
	FallbackToMechanical bool
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		SubscriberBufferSize: 256,
		FallbackToMechanical: true,
	}
}

// Graph is the top-level facade — IGraph.
type Graph struct {
	mu           sync.RWMutex
	store        store.Store
	actorStore   actor.IActorStore
	bus          *bus.EventBus
	registry     *event.EventTypeRegistry
	factory      *event.EventFactory
	trustModel   trust.ITrustModel
	authChain    authority.IAuthorityChain
	decisionMaker decision.IDecisionMaker
	config       Config
	started      bool
	closed       bool
}

// Option configures a Graph.
type Option func(*Graph)

// WithTrustModel sets the trust model.
func WithTrustModel(m trust.ITrustModel) Option { return func(g *Graph) { g.trustModel = m } }

// WithAuthorityChain sets the authority chain.
func WithAuthorityChain(c authority.IAuthorityChain) Option {
	return func(g *Graph) { g.authChain = c }
}

// WithDecisionMaker sets the decision maker.
func WithDecisionMaker(dm decision.IDecisionMaker) Option {
	return func(g *Graph) { g.decisionMaker = dm }
}

// WithConfig sets the graph config.
func WithConfig(c Config) Option { return func(g *Graph) { g.config = c } }

// New creates a new Graph with the given store, actor store, and options.
func New(s store.Store, as actor.IActorStore, opts ...Option) *Graph {
	registry := event.DefaultRegistry()
	config := DefaultConfig()

	g := &Graph{
		store:      s,
		actorStore: as,
		registry:   registry,
		factory:    event.NewEventFactory(registry),
		config:     config,
	}

	for _, opt := range opts {
		opt(g)
	}

	// Defaults
	if g.trustModel == nil {
		g.trustModel = trust.NewDefaultTrustModel()
	}
	if g.authChain == nil {
		g.authChain = authority.NewDefaultAuthorityChain(g.trustModel, s)
	}

	g.bus = bus.NewEventBus(s, g.config.SubscriberBufferSize)

	return g
}

// Start initializes the graph. Must be called before Evaluate/Record.
func (g *Graph) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.started {
		return nil
	}
	g.started = true
	return nil
}

// Close performs graceful shutdown.
func (g *Graph) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.closed {
		return nil
	}
	g.closed = true
	g.bus.Close()
	g.store.Close()
	return nil
}

// Record creates and persists an event, then notifies the bus.
func (g *Graph) Record(
	eventType types.EventType,
	source types.ActorID,
	content event.EventContent,
	causes []types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.closed {
		return event.Event{}, fmt.Errorf("graph is closed")
	}

	ev, err := g.factory.Create(eventType, source, content, causes, conversationID, g.store, signer)
	if err != nil {
		return event.Event{}, err
	}

	stored, err := g.store.Append(ev)
	if err != nil {
		return event.Event{}, err
	}

	g.bus.Publish(stored)
	return stored, nil
}

// Bootstrap initializes the graph with a genesis event.
func (g *Graph) Bootstrap(systemActor types.ActorID, signer event.Signer) (event.Event, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.closed {
		return event.Event{}, fmt.Errorf("graph is closed")
	}

	bf := event.NewBootstrapFactory(g.registry)
	ev, err := bf.Init(systemActor, signer)
	if err != nil {
		return event.Event{}, err
	}

	stored, err := g.store.Append(ev)
	if err != nil {
		return event.Event{}, err
	}

	g.bus.Publish(stored)
	return stored, nil
}

// Evaluate runs an action through authority evaluation.
// In the bootstrap phase, this is simplified — no tick engine or full decision tree.
func (g *Graph) Evaluate(ctx context.Context, a actor.IActor, action string, evalContext map[string]any) (authority.AuthorityResult, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.closed {
		return authority.AuthorityResult{}, fmt.Errorf("graph is closed")
	}

	return g.authChain.Evaluate(ctx, a, action)
}

// Query returns a query builder for the graph.
func (g *Graph) Query() *Query {
	return &Query{
		store:      g.store,
		actorStore: g.actorStore,
		trustModel: g.trustModel,
	}
}

// Store returns the underlying event store.
func (g *Graph) Store() store.Store { return g.store }

// ActorStore returns the underlying actor store.
func (g *Graph) ActorStore() actor.IActorStore { return g.actorStore }

// Bus returns the event bus.
func (g *Graph) Bus() *bus.EventBus { return g.bus }

// Registry returns the event type registry.
func (g *Graph) Registry() *event.EventTypeRegistry { return g.registry }
