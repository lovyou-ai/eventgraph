package authority

import (
	"context"
	"sync"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/trust"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// AuthorityResult is the result of evaluating authority for an action.
type AuthorityResult struct {
	Level  event.AuthorityLevel
	Weight types.Score
	Chain  []event.AuthorityLink
}

// AuthorityPolicy defines the authority requirements for an action.
type AuthorityPolicy struct {
	Action   string
	Level    event.AuthorityLevel
	MinTrust types.Option[types.Score]
	Scope    types.Option[types.DomainScope]
}

// IAuthorityChain evaluates authority. Returns weighted authority, not binary permission.
type IAuthorityChain interface {
	Evaluate(ctx context.Context, a actor.IActor, action string) (AuthorityResult, error)
	Chain(ctx context.Context, a actor.IActor, action string) ([]event.AuthorityLink, error)
	Grant(ctx context.Context, from actor.IActor, to actor.IActor, scope types.DomainScope, weight types.Score) (event.Edge, error)
	Revoke(ctx context.Context, from actor.IActor, to actor.IActor, scope types.DomainScope) error
}

// DefaultAuthorityChain is a flat authority model — no delegation chain.
// All actions default to Notification unless a policy says otherwise.
type DefaultAuthorityChain struct {
	mu         sync.RWMutex
	policies   []AuthorityPolicy
	trustModel trust.ITrustModel
	store      store.Store
	factory    *event.EventFactory
	signer     event.Signer
}

// NewDefaultAuthorityChain creates a flat authority chain.
func NewDefaultAuthorityChain(trustModel trust.ITrustModel, s store.Store, factory *event.EventFactory, signer event.Signer) *DefaultAuthorityChain {
	return &DefaultAuthorityChain{
		trustModel: trustModel,
		store:      s,
		factory:    factory,
		signer:     signer,
	}
}

// AddPolicy registers an authority policy. Policies are checked in order; first match wins.
func (c *DefaultAuthorityChain) AddPolicy(policy AuthorityPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.policies = append(c.policies, policy)
}

func (c *DefaultAuthorityChain) Evaluate(ctx context.Context, a actor.IActor, action string) (AuthorityResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	policy := c.findPolicy(action)
	level := policy.Level

	// If actor has high enough trust, downgrade Required -> Recommended
	if level == event.AuthorityLevelRequired && policy.MinTrust.IsSome() {
		metrics, err := c.trustModel.Score(ctx, a)
		if err == nil && metrics.Overall().Value() >= policy.MinTrust.Unwrap().Value() {
			level = event.AuthorityLevelRecommended
		}
	}

	link := event.AuthorityLink{
		Actor:  a.ID(),
		Level:  level,
		Weight: types.MustScore(1.0),
	}

	return AuthorityResult{
		Level:  level,
		Weight: types.MustScore(1.0),
		Chain:  []event.AuthorityLink{link},
	}, nil
}

func (c *DefaultAuthorityChain) Chain(_ context.Context, a actor.IActor, action string) ([]event.AuthorityLink, error) {
	c.mu.RLock()
	policy := c.findPolicy(action)
	c.mu.RUnlock()

	return []event.AuthorityLink{
		{
			Actor:  a.ID(),
			Level:  policy.Level,
			Weight: types.MustScore(1.0),
		},
	}, nil
}

// Grant creates an authority edge and persists it to the store as an edge.created event.
func (c *DefaultAuthorityChain) Grant(_ context.Context, from actor.IActor, to actor.IActor, scope types.DomainScope, weight types.Score) (event.Edge, error) {
	return grantAndPersist(c.store, c.factory, c.signer, from, to, scope, weight)
}

func (c *DefaultAuthorityChain) Revoke(_ context.Context, _ actor.IActor, _ actor.IActor, _ types.DomainScope) error {
	// In the flat model, revocation is a no-op — would emit a supersede event in full impl
	return nil
}

func (c *DefaultAuthorityChain) findPolicy(action string) AuthorityPolicy {
	for _, p := range c.policies {
		if matchesAction(p.Action, action) {
			return p
		}
	}
	// Default: Notification level
	return AuthorityPolicy{
		Action: "*",
		Level:  event.AuthorityLevelNotification,
	}
}

func matchesAction(pattern, action string) bool {
	if pattern == "*" {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(action) >= len(prefix) && action[:len(prefix)] == prefix
	}
	return pattern == action
}
