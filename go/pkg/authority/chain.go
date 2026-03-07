package authority

import (
	"context"
	"fmt"
	"sync"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/trust"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// MaxChainDepth is the maximum delegation chain depth to prevent infinite loops.
const MaxChainDepth = 10

// ErrChainDepthExceeded is returned when a delegation chain exceeds MaxChainDepth.
var ErrChainDepthExceeded = fmt.Errorf("delegation chain depth exceeds %d", MaxChainDepth)

// DelegationChain walks Authority edges in the store to build delegation chains.
// Unlike DefaultAuthorityChain (flat model), this supports multi-hop delegation
// with weight propagation and expiry handling.
type DelegationChain struct {
	trustModel trust.ITrustModel
	store      store.Store
	mu         sync.RWMutex
	policies   []AuthorityPolicy
}

// NewDelegationChain creates a delegation-aware authority chain.
func NewDelegationChain(trustModel trust.ITrustModel, s store.Store) *DelegationChain {
	return &DelegationChain{
		trustModel: trustModel,
		store:      s,
	}
}

// AddPolicy registers an authority policy.
func (c *DelegationChain) AddPolicy(policy AuthorityPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.policies = append(c.policies, policy)
}

func (c *DelegationChain) Evaluate(ctx context.Context, a actor.IActor, action string) (AuthorityResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	chain, err := c.walkChain(ctx, a.ID(), action, nil)
	if err != nil {
		return AuthorityResult{}, err
	}

	if len(chain) == 0 {
		// No delegation chain — use direct authority
		policy := c.findPolicy(action)
		link := event.AuthorityLink{
			Actor:  a.ID(),
			Level:  policy.Level,
			Weight: types.MustScore(1.0),
		}
		return AuthorityResult{
			Level:  policy.Level,
			Weight: types.MustScore(1.0),
			Chain:  []event.AuthorityLink{link},
		}, nil
	}

	// The final weight is the product of all weights in the chain
	weight := 1.0
	for _, link := range chain {
		weight *= link.Weight.Value()
	}
	weightScore := types.MustScore(clamp(weight, 0.0, 1.0))

	// Level comes from the policy for this action
	policy := c.findPolicy(action)
	level := policy.Level

	return AuthorityResult{
		Level:  level,
		Weight: weightScore,
		Chain:  chain,
	}, nil
}

func (c *DelegationChain) Chain(ctx context.Context, a actor.IActor, action string) ([]event.AuthorityLink, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	chain, err := c.walkChain(ctx, a.ID(), action, nil)
	if err != nil {
		return nil, err
	}
	if len(chain) == 0 {
		return []event.AuthorityLink{
			{
				Actor:  a.ID(),
				Level:  event.AuthorityLevelNotification,
				Weight: types.MustScore(1.0),
			},
		}, nil
	}
	return chain, nil
}

// Grant creates an authority edge. The returned edge is not persisted —
// the caller must persist it by recording an edge.created event via the store.
// This separation follows the pattern: factory creates, store persists.
func (c *DelegationChain) Grant(_ context.Context, from actor.IActor, to actor.IActor, scope types.DomainScope, weight types.Score) (event.Edge, error) {
	eid, err := types.NewEventIDFromNew()
	if err != nil {
		return event.Edge{}, err
	}
	edgeID := types.MustEdgeID(eid.Value())
	edge, err := event.NewEdge(
		edgeID,
		from.ID(),
		to.ID(),
		event.EdgeTypeAuthority,
		types.MustWeight(weight.Value()*2-1),
		event.EdgeDirectionCentrifugal,
		types.Some(scope),
		nil,
		types.Now(),
		types.None[types.Timestamp](),
	)
	if err != nil {
		return event.Edge{}, err
	}
	return edge, nil
}

// Revoke is a stub — full implementation would record an edge.superseded event.
func (c *DelegationChain) Revoke(_ context.Context, _ actor.IActor, _ actor.IActor, _ types.DomainScope) error {
	return nil
}

// walkChain recursively walks Authority edges from the given actor, building
// the delegation chain. Weight propagates multiplicatively through the chain.
func (c *DelegationChain) walkChain(ctx context.Context, actorID types.ActorID, action string, visited map[types.ActorID]bool) ([]event.AuthorityLink, error) {
	if visited == nil {
		visited = make(map[types.ActorID]bool)
	}

	if visited[actorID] {
		return nil, nil // cycle detected
	}
	if len(visited) >= MaxChainDepth {
		return nil, ErrChainDepthExceeded
	}
	visited[actorID] = true

	// Find Authority edges pointing TO this actor (delegations granted to them)
	edges, err := c.store.EdgesTo(actorID, event.EdgeTypeAuthority)
	if err != nil {
		return nil, err
	}

	now := types.Now()

	// Filter for valid (non-expired) edges with matching scope
	var bestEdge *event.Edge
	var bestWeight float64

	for i := range edges {
		e := edges[i]

		// Check expiry
		if e.ExpiresAt().IsSome() {
			if e.ExpiresAt().Unwrap().Value().Before(now.Value()) {
				continue
			}
		}

		// Check scope match if policy has a scope
		policy := c.findPolicy(action)
		if policy.Scope.IsSome() && e.Scope().IsSome() {
			if policy.Scope.Unwrap() != e.Scope().Unwrap() {
				continue
			}
		}

		// Convert edge weight [-1,1] back to authority weight [0,1]
		w := (e.Weight().Value() + 1.0) / 2.0
		if w > bestWeight {
			bestWeight = w
			edge := e
			bestEdge = &edge
		}
	}

	if bestEdge == nil {
		// No delegation — just the actor itself
		policy := c.findPolicy(action)
		return []event.AuthorityLink{
			{
				Actor:  actorID,
				Level:  policy.Level,
				Weight: types.MustScore(1.0),
			},
		}, nil
	}

	// Walk up the chain from the delegator
	parentChain, err := c.walkChain(ctx, bestEdge.From(), action, visited)
	if err != nil {
		return nil, err
	}

	// Append this actor's link with delegated weight
	link := event.AuthorityLink{
		Actor:  actorID,
		Level:  c.findPolicy(action).Level,
		Weight: types.MustScore(clamp(bestWeight, 0.0, 1.0)),
	}

	return append(parentChain, link), nil
}

func (c *DelegationChain) findPolicy(action string) AuthorityPolicy {
	for _, p := range c.policies {
		if matchesAction(p.Action, action) {
			return p
		}
	}
	return AuthorityPolicy{
		Action: "*",
		Level:  event.AuthorityLevelNotification,
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
