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
	factory    *event.EventFactory
	signer     event.Signer
	mu         sync.RWMutex
	policies   []AuthorityPolicy
}

// NewDelegationChain creates a delegation-aware authority chain.
func NewDelegationChain(trustModel trust.ITrustModel, s store.Store, factory *event.EventFactory, signer event.Signer) *DelegationChain {
	return &DelegationChain{
		trustModel: trustModel,
		store:      s,
		factory:    factory,
		signer:     signer,
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

	policy := c.findPolicy(action)
	chain, err := c.walkChain(ctx, a.ID(), policy, nil, 0)
	if err != nil {
		return AuthorityResult{}, err
	}

	if len(chain) == 0 {
		// No delegation chain — use direct authority
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

	// Level comes from the resolved policy
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

	policy := c.findPolicy(action)
	chain, err := c.walkChain(ctx, a.ID(), policy, nil, 0)
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

// Grant creates an authority edge and persists it to the store as an edge.created event.
func (c *DelegationChain) Grant(_ context.Context, from actor.IActor, to actor.IActor, scope types.DomainScope, weight types.Score) (event.Edge, error) {
	return grantAndPersist(c.store, c.factory, c.signer, from, to, scope, weight)
}

// Revoke is a stub — full implementation would record an edge.superseded event.
func (c *DelegationChain) Revoke(_ context.Context, _ actor.IActor, _ actor.IActor, _ types.DomainScope) error {
	return nil
}

// walkChain recursively walks Authority edges from the given actor, building
// the delegation chain. Weight propagates multiplicatively through the chain.
// depth tracks current path depth (not total actors visited across branches).
// policy is resolved once by the caller and passed through to avoid redundant lookups.
func (c *DelegationChain) walkChain(ctx context.Context, actorID types.ActorID, policy AuthorityPolicy, visited map[types.ActorID]bool, depth int) ([]event.AuthorityLink, error) {
	if visited == nil {
		visited = make(map[types.ActorID]bool)
	}

	if visited[actorID] {
		return nil, nil // cycle detected
	}
	if depth >= MaxChainDepth {
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

		// Check scope match: if policy requires a scope, the edge must have
		// a matching scope. Un-scoped edges do not satisfy scoped policies.
		if policy.Scope.IsSome() {
			if !e.Scope().IsSome() || policy.Scope.Unwrap() != e.Scope().Unwrap() {
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
		return []event.AuthorityLink{
			{
				Actor:  actorID,
				Level:  policy.Level,
				Weight: types.MustScore(1.0),
			},
		}, nil
	}

	// Walk up the chain from the delegator
	parentChain, err := c.walkChain(ctx, bestEdge.From(), policy, visited, depth+1)
	if err != nil {
		return nil, err
	}

	// Append this actor's link with delegated weight
	link := event.AuthorityLink{
		Actor:  actorID,
		Level:  policy.Level,
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
