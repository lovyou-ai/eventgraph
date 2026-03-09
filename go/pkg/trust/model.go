package trust

import (
	"context"
	"encoding/json"
	"fmt"
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
	InitialTrust       types.Score  // default: 0.0
	DecayRate          types.Score  // per day, default: 0.01
	MaxAdjustment      types.Weight // single event max, default: 0.1
	ObservedEventDelta float64      // trust boost for any non-trust event, default: 0.01
	TrendDecayRate     float64      // trend decay per day toward zero, default: 0.01
}

// DefaultTrustModel implements ITrustModel with linear decay and equal weighting.
type DefaultTrustModel struct {
	config   DefaultConfig
	mu       sync.RWMutex
	scores   map[trustKey]*trustState
	directed map[directedTrustKey]*trustState
}

type trustKey struct {
	actor string
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
			InitialTrust:       types.MustScore(0.0),
			DecayRate:          types.MustScore(0.01),
			MaxAdjustment:      types.MustWeight(0.1),
			ObservedEventDelta: 0.01,
			TrendDecayRate:     0.01,
		},
		scores:   make(map[trustKey]*trustState),
		directed: make(map[directedTrustKey]*trustState),
	}
}

// NewDefaultTrustModelWithConfig creates a DefaultTrustModel with custom config.
// All config values are used as-is — zero values are respected (no implicit defaults).
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

	// Fall back to global score with halved confidence (no domain-specific data)
	evidenceCount := len(state.evidence)
	globalConfidence := math.Min(1.0, float64(evidenceCount)/50.0)
	return event.NewTrustMetrics(
		a.ID(),
		state.score,
		state.byDomain,
		types.MustScore(globalConfidence*0.5),
		state.trend,
		state.evidence,
		state.lastUpdated,
		m.config.DecayRate,
	), nil
}

func (m *DefaultTrustModel) Update(_ context.Context, a actor.IActor, evidence event.Event) (event.TrustMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.getOrCreate(a.ID())

	// Deduplicate: if this evidence was already applied, return current metrics unchanged.
	for _, id := range state.evidence {
		if id == evidence.ID() {
			return m.buildMetrics(a.ID(), state), nil
		}
	}

	// Extract trust delta from the evidence event, relative to current score
	delta := m.extractDeltaForScore(evidence, state.score.Value())

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

	// Deduplicate: if this evidence was already applied, return current metrics unchanged.
	for _, id := range state.evidence {
		if id == evidence.ID() {
			return m.buildMetrics(to.ID(), state), nil
		}
	}

	delta := m.extractDeltaForScore(evidence, state.score.Value())

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

	return m.buildMetrics(to.ID(), state), nil
}

func (m *DefaultTrustModel) Decay(_ context.Context, a actor.IActor, elapsed time.Duration) (event.TrustMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Guard against negative durations (clock skew, repeated ticks) which would
	// invert the decay and silently inflate trust scores.
	if elapsed <= 0 {
		state := m.getOrDefault(a.ID())
		return m.buildMetrics(a.ID(), &state), nil
	}

	days := elapsed.Hours() / 24
	decayAmount := m.config.DecayRate.Value() * days
	trendDecay := m.config.TrendDecayRate * days

	// Decay undirected trust
	key := trustKey{actor: a.ID().Value()}
	state, ok := m.scores[key]
	if ok {
		m.decayState(state, decayAmount, trendDecay)
	}

	// Decay directed trust where actor is the trust holder (from).
	// Only the trust-holder's perspective decays — the target's trust from
	// other actors is decayed when Decay is called for those actors.
	for dkey, dstate := range m.directed {
		if dkey.from == a.ID().Value() {
			m.decayState(dstate, decayAmount, trendDecay)
		}
	}

	if !ok {
		// No undirected trust state — return defaults without creating state
		defaults := trustState{
			score:       m.config.InitialTrust,
			byDomain:    make(map[types.DomainScope]types.Score),
			lastUpdated: types.Now(),
			trend:       types.MustWeight(0.0),
		}
		return m.buildMetrics(a.ID(), &defaults), nil
	}

	return m.buildMetrics(a.ID(), state), nil
}

// decayState applies linear decay to a trust state's score, domain scores, and trend.
func (m *DefaultTrustModel) decayState(state *trustState, decayAmount, trendDecay float64) {
	state.score = types.MustScore(math.Max(0, state.score.Value()-decayAmount))

	for domain, ds := range state.byDomain {
		state.byDomain[domain] = types.MustScore(math.Max(0, ds.Value()-decayAmount))
	}

	if state.trend.Value() > 0 {
		state.trend = types.MustWeight(math.Max(0, state.trend.Value()-trendDecay))
	} else if state.trend.Value() < 0 {
		state.trend = types.MustWeight(math.Min(0, state.trend.Value()+trendDecay))
	}

	state.lastUpdated = types.Now()
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

// extractDeltaForScore computes the trust adjustment from an evidence event
// relative to the given current score. For TrustUpdatedContent, the delta moves
// the score toward tc.Current (absolute target), not by tc.Current - tc.Previous
// (which would double-apply if the model's current score has diverged from tc.Previous).
func (m *DefaultTrustModel) extractDeltaForScore(ev event.Event, currentScore float64) float64 {
	if tc, ok := ev.Content().(event.TrustUpdatedContent); ok {
		return tc.Current.Value() - currentScore
	}
	// Small positive trust boost for any observed (non-trust) event
	return m.config.ObservedEventDelta
}

// --- Export/Import for persistence ---

// TrustStateExport is a serializable snapshot of one trust relationship.
type TrustStateExport struct {
	Score       float64                    `json:"score"`
	ByDomain    map[string]float64         `json:"by_domain,omitempty"`
	Evidence    []string                   `json:"evidence,omitempty"`
	LastUpdated int64                      `json:"last_updated_nanos"`
	Trend       float64                    `json:"trend"`
}

// TrustExport is a serializable snapshot of the entire trust model.
type TrustExport struct {
	Scores   map[string]TrustStateExport `json:"scores"`            // key: "actor_id" or "actor_id:domain"
	Directed map[string]TrustStateExport `json:"directed,omitempty"` // key: "from→to"
}

// Export returns a serializable snapshot of all trust state.
func (m *DefaultTrustModel) Export() TrustExport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	export := TrustExport{
		Scores:   make(map[string]TrustStateExport, len(m.scores)),
		Directed: make(map[string]TrustStateExport, len(m.directed)),
	}

	for key, state := range m.scores {
		export.Scores[key.actor] = exportState(state)
	}

	for key, state := range m.directed {
		dirKey := encodeDirectedKey(key.from, key.to)
		export.Directed[dirKey] = exportState(state)
	}

	return export
}

// ExportJSON returns the trust state as JSON bytes.
func (m *DefaultTrustModel) ExportJSON() (json.RawMessage, error) {
	return json.Marshal(m.Export())
}

// Import restores trust state from a snapshot. Replaces all current state.
func (m *DefaultTrustModel) Import(export TrustExport) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	scores := make(map[trustKey]*trustState, len(export.Scores))
	for actorID, se := range export.Scores {
		state, err := importState(se)
		if err != nil {
			return fmt.Errorf("score for %s: %w", actorID, err)
		}
		scores[trustKey{actor: actorID}] = state
	}

	directed := make(map[directedTrustKey]*trustState, len(export.Directed))
	for dirKey, se := range export.Directed {
		from, to := decodeDirectedKey(dirKey)
		if from == "" || to == "" {
			return fmt.Errorf("invalid directed key: %q", dirKey)
		}
		state, err := importState(se)
		if err != nil {
			return fmt.Errorf("directed %s: %w", dirKey, err)
		}
		directed[directedTrustKey{from: from, to: to}] = state
	}

	// Only replace state after all entries parse successfully (atomic swap).
	m.scores = scores
	m.directed = directed
	return nil
}

// ImportJSON restores trust state from JSON bytes.
func (m *DefaultTrustModel) ImportJSON(data json.RawMessage) error {
	var export TrustExport
	if err := json.Unmarshal(data, &export); err != nil {
		return err
	}
	return m.Import(export)
}

func exportState(state *trustState) TrustStateExport {
	byDomain := make(map[string]float64, len(state.byDomain))
	for d, s := range state.byDomain {
		byDomain[d.Value()] = s.Value()
	}
	evidence := make([]string, len(state.evidence))
	for i, e := range state.evidence {
		evidence[i] = e.Value()
	}
	return TrustStateExport{
		Score:       state.score.Value(),
		ByDomain:    byDomain,
		Evidence:    evidence,
		LastUpdated: state.lastUpdated.UnixNano(),
		Trend:       state.trend.Value(),
	}
}

func importState(se TrustStateExport) (*trustState, error) {
	score, err := types.NewScore(se.Score)
	if err != nil {
		return nil, fmt.Errorf("invalid score %v: %w", se.Score, err)
	}
	trend, err := types.NewWeight(se.Trend)
	if err != nil {
		return nil, fmt.Errorf("invalid trend %v: %w", se.Trend, err)
	}

	byDomain := make(map[types.DomainScope]types.Score, len(se.ByDomain))
	for d, s := range se.ByDomain {
		ds, err := types.NewDomainScope(d)
		if err != nil {
			return nil, fmt.Errorf("invalid domain scope %q: %w", d, err)
		}
		sv, err := types.NewScore(s)
		if err != nil {
			return nil, fmt.Errorf("invalid domain score %v for %q: %w", s, d, err)
		}
		byDomain[ds] = sv
	}
	evidence := make([]types.EventID, len(se.Evidence))
	for i, e := range se.Evidence {
		eid, err := types.NewEventID(e)
		if err != nil {
			return nil, fmt.Errorf("invalid event ID %q: %w", e, err)
		}
		evidence[i] = eid
	}
	return &trustState{
		score:       score,
		byDomain:    byDomain,
		evidence:    evidence,
		lastUpdated: types.NewTimestamp(time.Unix(0, se.LastUpdated)),
		trend:       trend,
	}, nil
}

// encodeDirectedKey encodes a from→to pair as a JSON array string.
// This is collision-free regardless of actor ID content.
func encodeDirectedKey(from, to string) string {
	b, _ := json.Marshal([]string{from, to})
	return string(b)
}

// decodeDirectedKey decodes a directed trust key.
// Supports both new JSON array format and legacy "→" separator for backwards compatibility.
func decodeDirectedKey(key string) (string, string) {
	// Try JSON array format first
	var pair []string
	if err := json.Unmarshal([]byte(key), &pair); err == nil && len(pair) == 2 {
		return pair[0], pair[1]
	}
	// Legacy: split on "→" (3-byte UTF-8 sequence)
	for i := 0; i < len(key)-2; i++ {
		if key[i] == 0xe2 && key[i+1] == 0x86 && key[i+2] == 0x92 {
			return key[:i], key[i+3:]
		}
	}
	return "", ""
}
