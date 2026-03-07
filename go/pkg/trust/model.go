package trust

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// ITrustModel calculates, updates, and decays trust.
type ITrustModel interface {
	Score(ctx context.Context, a actor.IActor) (event.TrustMetrics, error)
	ScoreInDomain(ctx context.Context, a actor.IActor, domain types.DomainScope) (event.TrustMetrics, error)
	Update(ctx context.Context, a actor.IActor, evidence event.Event) (event.TrustMetrics, error)
	UpdateBetween(ctx context.Context, from actor.IActor, to actor.IActor, evidence event.Event) (event.TrustMetrics, error)
	Decay(ctx context.Context, a actor.IActor, elapsed time.Duration) (event.TrustMetrics, error)
	Between(ctx context.Context, from actor.IActor, to actor.IActor) (event.TrustMetrics, error)
}

// DefaultConfig holds default trust model parameters.
type DefaultConfig struct {
	InitialTrust  types.Score  // default: 0.0
	DecayRate     types.Score  // per day, default: 0.01
	MaxAdjustment types.Weight // single event max, default: 0.1
}

// DefaultTrustModel implements ITrustModel with linear decay and equal weighting.
type DefaultTrustModel struct {
	config   DefaultConfig
	mu       sync.RWMutex
	scores   map[trustKey]*trustState
	directed map[directedTrustKey]*trustState
}

type trustKey struct {
	actor  string
	domain string
}

// directedTrustKey is a collision-free key for directional trust (from→to).
// Uses separate fields instead of string concatenation to avoid collisions
// when actor IDs contain the separator.
type directedTrustKey struct {
	from string
	to   string
}

type trustState struct {
	score       types.Score
	byDomain    map[types.DomainScope]types.Score
	evidence    []types.EventID
	lastUpdated types.Timestamp
	trend       types.Weight
}

// NewDefaultTrustModel creates a DefaultTrustModel with sensible defaults.
func NewDefaultTrustModel() *DefaultTrustModel {
	return &DefaultTrustModel{
		config: DefaultConfig{
			InitialTrust:  types.MustScore(0.0),
			DecayRate:     types.MustScore(0.01),
			MaxAdjustment: types.MustWeight(0.1),
		},
		scores:   make(map[trustKey]*trustState),
		directed: make(map[directedTrustKey]*trustState),
	}
}

// NewDefaultTrustModelWithConfig creates a DefaultTrustModel with custom config.
func NewDefaultTrustModelWithConfig(config DefaultConfig) *DefaultTrustModel {
	return &DefaultTrustModel{
		config:   config,
		scores:   make(map[trustKey]*trustState),
		directed: make(map[directedTrustKey]*trustState),
	}
}

// getOrCreate returns the trust state for the actor, creating it if absent.
// Caller must hold m.mu.Lock().
func (m *DefaultTrustModel) getOrCreate(actorID types.ActorID) *trustState {
	key := trustKey{actor: actorID.Value()}
	if state, ok := m.scores[key]; ok {
		return state
	}
	state := &trustState{
		score:       m.config.InitialTrust,
		byDomain:    make(map[types.DomainScope]types.Score),
		lastUpdated: types.Now(),
		trend:       types.MustWeight(0.0),
	}
	m.scores[key] = state
	return state
}

// getOrDefault returns the trust state for the actor, or a default if absent.
// Caller must hold at least m.mu.RLock(). Does not mutate m.scores.
func (m *DefaultTrustModel) getOrDefault(actorID types.ActorID) trustState {
	key := trustKey{actor: actorID.Value()}
	if state, ok := m.scores[key]; ok {
		return *state
	}
	return trustState{
		score:       m.config.InitialTrust,
		byDomain:    make(map[types.DomainScope]types.Score),
		lastUpdated: types.Now(),
		trend:       types.MustWeight(0.0),
	}
}

func (m *DefaultTrustModel) Score(_ context.Context, a actor.IActor) (event.TrustMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := m.getOrDefault(a.ID())
	return m.buildMetrics(a.ID(), &state), nil
}

func (m *DefaultTrustModel) ScoreInDomain(_ context.Context, a actor.IActor, domain types.DomainScope) (event.TrustMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := m.getOrDefault(a.ID())

	// If domain-specific score exists, return metrics with that score
	if domainScore, ok := state.byDomain[domain]; ok {
		return m.buildDomainMetrics(a.ID(), &state, domainScore), nil
	}

	// Fall back to global score with lower confidence
	return m.buildMetrics(a.ID(), &state), nil
}

func (m *DefaultTrustModel) Update(_ context.Context, a actor.IActor, evidence event.Event) (event.TrustMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.getOrCreate(a.ID())

	// Extract trust delta from the evidence event
	delta := m.extractDelta(evidence)

	// Clamp to MaxAdjustment
	maxAdj := m.config.MaxAdjustment.Value()
	if delta > maxAdj {
		delta = maxAdj
	}
	if delta < -maxAdj {
		delta = -maxAdj
	}

	// Apply delta, clamp to [0, 1]
	newScore := state.score.Value() + delta
	newScore = math.Max(0, math.Min(1, newScore))
	state.score = types.MustScore(newScore)

	// Update domain-specific score if evidence carries a domain
	if tc, ok := evidence.Content().(event.TrustUpdatedContent); ok {
		var domainScore float64
		if existing, has := state.byDomain[tc.Domain]; has {
			domainScore = math.Max(0, math.Min(1, existing.Value()+delta))
		} else {
			domainScore = math.Max(0, math.Min(1, m.config.InitialTrust.Value()+delta))
		}
		state.byDomain[tc.Domain] = types.MustScore(domainScore)
	}

	// Update trend
	if delta > 0 {
		state.trend = types.MustWeight(math.Min(1, state.trend.Value()+0.1))
	} else if delta < 0 {
		state.trend = types.MustWeight(math.Max(-1, state.trend.Value()-0.1))
	}

	// Track evidence
	state.evidence = append(state.evidence, evidence.ID())
	if len(state.evidence) > 100 {
		state.evidence = state.evidence[len(state.evidence)-100:]
	}

	state.lastUpdated = types.Now()

	return m.buildMetrics(a.ID(), state), nil
}

// UpdateBetween updates directional trust from one actor toward another based on evidence.
func (m *DefaultTrustModel) UpdateBetween(_ context.Context, from actor.IActor, to actor.IActor, evidence event.Event) (event.TrustMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := directedTrustKey{from: from.ID().Value(), to: to.ID().Value()}
	state, ok := m.directed[key]
	if !ok {
		state = &trustState{
			score:       m.config.InitialTrust,
			byDomain:    make(map[types.DomainScope]types.Score),
			lastUpdated: types.Now(),
			trend:       types.MustWeight(0.0),
		}
		m.directed[key] = state
	}

	delta := m.extractDelta(evidence)

	maxAdj := m.config.MaxAdjustment.Value()
	if delta > maxAdj {
		delta = maxAdj
	}
	if delta < -maxAdj {
		delta = -maxAdj
	}

	newScore := state.score.Value() + delta
	newScore = math.Max(0, math.Min(1, newScore))
	state.score = types.MustScore(newScore)

	if delta > 0 {
		state.trend = types.MustWeight(math.Min(1, state.trend.Value()+0.1))
	} else if delta < 0 {
		state.trend = types.MustWeight(math.Max(-1, state.trend.Value()-0.1))
	}

	state.evidence = append(state.evidence, evidence.ID())
	if len(state.evidence) > 100 {
		state.evidence = state.evidence[len(state.evidence)-100:]
	}

	state.lastUpdated = types.Now()

	return m.buildMetrics(to.ID(), state), nil
}

func (m *DefaultTrustModel) Decay(_ context.Context, a actor.IActor, elapsed time.Duration) (event.TrustMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.getOrCreate(a.ID())

	days := elapsed.Hours() / 24
	decayAmount := m.config.DecayRate.Value() * days
	newScore := math.Max(0, state.score.Value()-decayAmount)
	state.score = types.MustScore(newScore)

	// Decay domain-specific scores
	for domain, ds := range state.byDomain {
		state.byDomain[domain] = types.MustScore(math.Max(0, ds.Value()-decayAmount))
	}

	// Decay trend toward zero
	if state.trend.Value() > 0 {
		state.trend = types.MustWeight(math.Max(0, state.trend.Value()-0.01*days))
	} else if state.trend.Value() < 0 {
		state.trend = types.MustWeight(math.Min(0, state.trend.Value()+0.01*days))
	}

	state.lastUpdated = types.Now()

	return m.buildMetrics(a.ID(), state), nil
}

func (m *DefaultTrustModel) Between(_ context.Context, from actor.IActor, to actor.IActor) (event.TrustMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := directedTrustKey{from: from.ID().Value(), to: to.ID().Value()}
	state, ok := m.directed[key]
	if !ok {
		// No direct trust relationship
		return event.NewTrustMetrics(
			to.ID(),
			m.config.InitialTrust,
			nil,
			types.MustScore(0.0), // low confidence
			types.MustWeight(0.0),
			nil,
			types.Now(),
			m.config.DecayRate,
		), nil
	}
	return m.buildMetrics(to.ID(), state), nil
}

func (m *DefaultTrustModel) buildDomainMetrics(actorID types.ActorID, state *trustState, domainScore types.Score) event.TrustMetrics {
	evidenceCount := len(state.evidence)
	confidence := math.Min(1.0, float64(evidenceCount)/50.0)

	return event.NewTrustMetrics(
		actorID,
		domainScore,
		state.byDomain,
		types.MustScore(confidence),
		state.trend,
		state.evidence,
		state.lastUpdated,
		m.config.DecayRate,
	)
}

func (m *DefaultTrustModel) buildMetrics(actorID types.ActorID, state *trustState) event.TrustMetrics {
	// Confidence is based on evidence count
	evidenceCount := len(state.evidence)
	confidence := math.Min(1.0, float64(evidenceCount)/50.0)

	return event.NewTrustMetrics(
		actorID,
		state.score,
		state.byDomain,
		types.MustScore(confidence),
		state.trend,
		state.evidence,
		state.lastUpdated,
		m.config.DecayRate,
	)
}

func (m *DefaultTrustModel) extractDelta(ev event.Event) float64 {
	// Extract trust delta from TrustUpdatedContent
	if tc, ok := ev.Content().(event.TrustUpdatedContent); ok {
		return tc.Current.Value() - tc.Previous.Value()
	}
	// Default small positive for any observed event
	return 0.01
}
