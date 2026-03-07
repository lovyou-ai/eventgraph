package event

import (
	"encoding/json"

	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Decision represents what IDecisionMaker.Decide() returns. Immutable.
type Decision struct {
	outcome        DecisionOutcome
	confidence     types.Score
	authorityChain types.NonEmpty[AuthorityLink]
	trustWeights   []TrustWeight
	evidence       types.NonEmpty[types.EventID]
	receipt        Receipt
	needsHuman     bool
}

// NewDecision creates a new immutable Decision.
func NewDecision(
	outcome DecisionOutcome,
	confidence types.Score,
	authorityChain types.NonEmpty[AuthorityLink],
	trustWeights []TrustWeight,
	evidence types.NonEmpty[types.EventID],
	receipt Receipt,
	needsHuman bool,
) (Decision, error) {
	if !outcome.IsValid() {
		return Decision{}, &types.InvalidFormatError{
			Field: "DecisionOutcome", Value: string(outcome), Expected: "Permit, Deny, Defer, or Escalate",
		}
	}
	tw := make([]TrustWeight, len(trustWeights))
	copy(tw, trustWeights)
	return Decision{
		outcome:        outcome,
		confidence:     confidence,
		authorityChain: authorityChain,
		trustWeights:   tw,
		evidence:       evidence,
		receipt:        receipt,
		needsHuman:     needsHuman,
	}, nil
}

func (d Decision) Outcome() DecisionOutcome              { return d.outcome }
func (d Decision) Confidence() types.Score               { return d.confidence }
func (d Decision) AuthorityChain() types.NonEmpty[AuthorityLink] { return d.authorityChain }
func (d Decision) TrustWeights() []TrustWeight {
	tw := make([]TrustWeight, len(d.trustWeights))
	copy(tw, d.trustWeights)
	return tw
}
func (d Decision) Evidence() types.NonEmpty[types.EventID] { return d.evidence }
func (d Decision) Receipt() Receipt                        { return d.receipt }
func (d Decision) NeedsHuman() bool                        { return d.needsHuman }

// AuthorityLink represents a link in an authority chain.
type AuthorityLink struct {
	Actor  types.ActorID
	Level  AuthorityLevel
	Weight types.Score
}

// TrustWeight represents a trust score for an actor in a domain.
type TrustWeight struct {
	Actor  types.ActorID
	Score  types.Score
	Domain types.DomainScope
}

// Receipt is a cryptographic proof of a decision.
type Receipt struct {
	hash      types.Hash
	timestamp types.Timestamp
	signedBy  types.ActorID
	signature types.Signature
	inputHash types.Hash
	chainPos  types.EventID
}

// NewReceipt creates a new immutable Receipt.
func NewReceipt(
	hash types.Hash,
	timestamp types.Timestamp,
	signedBy types.ActorID,
	signature types.Signature,
	inputHash types.Hash,
	chainPos types.EventID,
) Receipt {
	return Receipt{
		hash:      hash,
		timestamp: timestamp,
		signedBy:  signedBy,
		signature: signature,
		inputHash: inputHash,
		chainPos:  chainPos,
	}
}

func (r Receipt) Hash() types.Hash         { return r.hash }
func (r Receipt) Timestamp() types.Timestamp { return r.timestamp }
func (r Receipt) SignedBy() types.ActorID   { return r.signedBy }
func (r Receipt) Signature() types.Signature { return r.signature }
func (r Receipt) InputHash() types.Hash    { return r.inputHash }
func (r Receipt) ChainPos() types.EventID  { return r.chainPos }

// TrustMetrics represents a trust query result. Immutable.
type TrustMetrics struct {
	actor       types.ActorID
	overall     types.Score
	byDomain    map[types.DomainScope]types.Score
	confidence  types.Score
	trend       types.Weight
	evidence    []types.EventID
	lastUpdated types.Timestamp
	decayRate   types.Score
}

// NewTrustMetrics creates a new immutable TrustMetrics.
func NewTrustMetrics(
	actor types.ActorID,
	overall types.Score,
	byDomain map[types.DomainScope]types.Score,
	confidence types.Score,
	trend types.Weight,
	evidence []types.EventID,
	lastUpdated types.Timestamp,
	decayRate types.Score,
) TrustMetrics {
	// Normalize empty collections to nil so omitempty omits them from JSON.
	// This matches the HealthReportContent pattern and ensures cross-language
	// canonical form consistency (empty = absent, not empty object/array).
	var bd map[types.DomainScope]types.Score
	if len(byDomain) > 0 {
		bd = make(map[types.DomainScope]types.Score, len(byDomain))
		for k, v := range byDomain {
			bd[k] = v
		}
	}
	var ev []types.EventID
	if len(evidence) > 0 {
		ev = make([]types.EventID, len(evidence))
		copy(ev, evidence)
	}
	return TrustMetrics{
		actor:       actor,
		overall:     overall,
		byDomain:    bd,
		confidence:  confidence,
		trend:       trend,
		evidence:    ev,
		lastUpdated: lastUpdated,
		decayRate:   decayRate,
	}
}

// trustMetricsJSON is the serialization form for TrustMetrics.
// TrustMetrics has all unexported fields (immutability), so we need explicit JSON support.
type trustMetricsJSON struct {
	Actor       types.ActorID                      `json:"Actor"`
	Overall     types.Score                        `json:"Overall"`
	ByDomain    map[types.DomainScope]types.Score  `json:"ByDomain,omitempty"`
	Confidence  types.Score                        `json:"Confidence"`
	Trend       types.Weight                       `json:"Trend"`
	Evidence    []types.EventID                    `json:"Evidence,omitempty"`
	LastUpdated types.Timestamp                    `json:"LastUpdated"`
	DecayRate   types.Score                        `json:"DecayRate"`
}

// MarshalJSON serializes TrustMetrics to JSON with all fields exposed.
func (m TrustMetrics) MarshalJSON() ([]byte, error) {
	return json.Marshal(trustMetricsJSON{
		Actor:       m.actor,
		Overall:     m.overall,
		ByDomain:    m.byDomain,
		Confidence:  m.confidence,
		Trend:       m.trend,
		Evidence:    m.evidence,
		LastUpdated: m.lastUpdated,
		DecayRate:   m.decayRate,
	})
}

// UnmarshalJSON deserializes TrustMetrics from JSON.
func (m *TrustMetrics) UnmarshalJSON(b []byte) error {
	var raw trustMetricsJSON
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*m = NewTrustMetrics(
		raw.Actor, raw.Overall, raw.ByDomain, raw.Confidence,
		raw.Trend, raw.Evidence, raw.LastUpdated, raw.DecayRate,
	)
	return nil
}

func (m TrustMetrics) Actor() types.ActorID   { return m.actor }
func (m TrustMetrics) Overall() types.Score   { return m.overall }
func (m TrustMetrics) ByDomain() map[types.DomainScope]types.Score {
	bd := make(map[types.DomainScope]types.Score, len(m.byDomain))
	for k, v := range m.byDomain {
		bd[k] = v
	}
	return bd
}
func (m TrustMetrics) Confidence() types.Score { return m.confidence }
func (m TrustMetrics) Trend() types.Weight     { return m.trend }
func (m TrustMetrics) Evidence() []types.EventID {
	ev := make([]types.EventID, len(m.evidence))
	copy(ev, m.evidence)
	return ev
}
func (m TrustMetrics) LastUpdated() types.Timestamp { return m.lastUpdated }
func (m TrustMetrics) DecayRate() types.Score { return m.decayRate }

// Expectation represents what should happen after an event. Immutable.
type Expectation struct {
	id          types.EventID
	trigger     types.EventID
	description string
	deadline    types.Timestamp
	severity    SeverityLevel
	status      ExpectationStatus
}

// NewExpectation creates a new immutable Expectation.
func NewExpectation(
	id types.EventID,
	trigger types.EventID,
	description string,
	deadline types.Timestamp,
	severity SeverityLevel,
	status ExpectationStatus,
) (Expectation, error) {
	if !severity.IsValid() {
		return Expectation{}, &types.InvalidFormatError{
			Field: "SeverityLevel", Value: string(severity), Expected: "Info, Warning, Serious, or Critical",
		}
	}
	if !status.IsValid() {
		return Expectation{}, &types.InvalidFormatError{
			Field: "ExpectationStatus", Value: string(status), Expected: "Pending, Met, Violated, or Expired",
		}
	}
	return Expectation{
		id:          id,
		trigger:     trigger,
		description: description,
		deadline:    deadline,
		severity:    severity,
		status:      status,
	}, nil
}

func (e Expectation) ID() types.EventID         { return e.id }
func (e Expectation) Trigger() types.EventID     { return e.trigger }
func (e Expectation) Description() string        { return e.description }
func (e Expectation) Deadline() types.Timestamp   { return e.deadline }
func (e Expectation) Severity() SeverityLevel    { return e.severity }
func (e Expectation) Status() ExpectationStatus  { return e.status }

// ViolationRecord records when an expectation is not met. Immutable.
type ViolationRecord struct {
	id          types.EventID
	expectation types.EventID
	severity    SeverityLevel
	actor       types.ActorID
	description string
	evidence    types.NonEmpty[types.EventID]
}

// NewViolationRecord creates a new immutable ViolationRecord.
func NewViolationRecord(
	id types.EventID,
	expectation types.EventID,
	severity SeverityLevel,
	actor types.ActorID,
	description string,
	evidence types.NonEmpty[types.EventID],
) (ViolationRecord, error) {
	if !severity.IsValid() {
		return ViolationRecord{}, &types.InvalidFormatError{
			Field: "SeverityLevel", Value: string(severity), Expected: "Info, Warning, Serious, or Critical",
		}
	}
	return ViolationRecord{
		id:          id,
		expectation: expectation,
		severity:    severity,
		actor:       actor,
		description: description,
		evidence:    evidence,
	}, nil
}

func (v ViolationRecord) ID() types.EventID                    { return v.id }
func (v ViolationRecord) Expectation() types.EventID           { return v.expectation }
func (v ViolationRecord) Severity() SeverityLevel              { return v.severity }
func (v ViolationRecord) Actor() types.ActorID                 { return v.actor }
func (v ViolationRecord) Description() string                  { return v.description }
func (v ViolationRecord) Evidence() types.NonEmpty[types.EventID] { return v.evidence }

// DecisionInput is the input to IDecisionMaker.Decide().
type DecisionInput struct {
	Action  string
	Actor   types.ActorID
	Context map[string]any
	Causes  types.NonEmpty[types.EventID]
}
