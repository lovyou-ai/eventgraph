package event

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// EventContent is the interface for all typed event content.
// Every event type has a corresponding content struct.
type EventContent interface {
	// EventTypeName returns the event type string this content belongs to.
	EventTypeName() string
	// Accept dispatches to the appropriate visitor method.
	Accept(EventContentVisitor)
}

// EventContentVisitor provides exhaustive dispatch over event content types.
type EventContentVisitor interface {
	// Trust
	VisitTrustUpdated(TrustUpdatedContent)
	VisitTrustScore(TrustScoreContent)
	VisitTrustDecayed(TrustDecayedContent)
	// Authority
	VisitAuthorityRequested(AuthorityRequestContent)
	VisitAuthorityResolved(AuthorityResolvedContent)
	VisitAuthorityDelegated(AuthorityDelegatedContent)
	VisitAuthorityRevoked(AuthorityRevokedContent)
	VisitAuthorityTimeout(AuthorityTimeoutContent)
	// Actor
	VisitActorRegistered(ActorRegisteredContent)
	VisitActorSuspended(ActorSuspendedContent)
	VisitActorMemorial(ActorMemorialContent)
	// Edge
	VisitEdgeCreated(EdgeCreatedContent)
	VisitEdgeSuperseded(EdgeSupersededContent)
	// Integrity
	VisitViolationDetected(ViolationDetectedContent)
	VisitChainVerified(ChainVerifiedContent)
	VisitChainBroken(ChainBrokenContent)
	// System
	VisitBootstrap(BootstrapContent)
	VisitClockTick(ClockTickContent)
	VisitHealthReport(HealthReportContent)
	// Decision tree
	VisitBranchProposed(BranchProposedContent)
	VisitBranchInserted(BranchInsertedContent)
	VisitCostReport(CostReportContent)
	// Social grammar
	VisitGrammarEmit(GrammarEmitContent)
	VisitGrammarRespond(GrammarRespondContent)
	VisitGrammarDerive(GrammarDeriveContent)
	VisitGrammarExtend(GrammarExtendContent)
	VisitGrammarRetract(GrammarRetractContent)
	VisitGrammarAnnotate(GrammarAnnotateContent)
	VisitGrammarMerge(GrammarMergeContent)
	VisitGrammarConsent(GrammarConsentContent)
	// EGIP
	VisitEGIPHelloSent(EGIPHelloSentContent)
	VisitEGIPHelloReceived(EGIPHelloReceivedContent)
	VisitEGIPMessageSent(EGIPMessageSentContent)
	VisitEGIPMessageReceived(EGIPMessageReceivedContent)
	VisitEGIPReceiptSent(EGIPReceiptSentContent)
	VisitEGIPReceiptReceived(EGIPReceiptReceivedContent)
	VisitEGIPProofRequested(EGIPProofRequestedContent)
	VisitEGIPProofReceived(EGIPProofReceivedContent)
	VisitEGIPTreatyProposed(EGIPTreatyProposedContent)
	VisitEGIPTreatyActive(EGIPTreatyActiveContent)
	VisitEGIPTrustUpdated(EGIPTrustUpdatedContent)
}

// --- Trust content ---

// TrustUpdatedContent is emitted when trust between actors changes.
type TrustUpdatedContent struct {
	Actor    types.ActorID     `json:"Actor"`
	Previous types.Score       `json:"Previous"`
	Current  types.Score       `json:"Current"`
	Domain   types.DomainScope `json:"Domain"`
	Cause    types.EventID     `json:"Cause"`
}

func (c TrustUpdatedContent) EventTypeName() string    { return "trust.updated" }
func (c TrustUpdatedContent) Accept(v EventContentVisitor) { v.VisitTrustUpdated(c) }

// TrustScoreContent is emitted when a trust score snapshot is recorded.
type TrustScoreContent struct {
	Actor   types.ActorID `json:"Actor"`
	Metrics TrustMetrics  `json:"Metrics"`
}

func (c TrustScoreContent) EventTypeName() string    { return "trust.score" }
func (c TrustScoreContent) Accept(v EventContentVisitor) { v.VisitTrustScore(c) }

// TrustDecayedContent is emitted when trust decays over time.
type TrustDecayedContent struct {
	Actor    types.ActorID  `json:"Actor"`
	Previous types.Score    `json:"Previous"`
	Current  types.Score    `json:"Current"`
	Elapsed  types.Duration `json:"Elapsed"`
	Rate     types.Score    `json:"Rate"`
}

func (c TrustDecayedContent) EventTypeName() string    { return "trust.decayed" }
func (c TrustDecayedContent) Accept(v EventContentVisitor) { v.VisitTrustDecayed(c) }

// --- Authority content ---

// AuthorityRequestContent is emitted when an authority request is made.
type AuthorityRequestContent struct {
	Action        string                       `json:"Action"`
	Actor         types.ActorID                `json:"Actor"`
	Level         AuthorityLevel               `json:"Level"`
	Justification string                       `json:"Justification"`
	Causes        types.NonEmpty[types.EventID] `json:"Causes"`
}

func (c AuthorityRequestContent) EventTypeName() string    { return "authority.requested" }
func (c AuthorityRequestContent) Accept(v EventContentVisitor) { v.VisitAuthorityRequested(c) }

// AuthorityResolvedContent is emitted when an authority request is resolved.
type AuthorityResolvedContent struct {
	RequestID types.EventID        `json:"RequestID"`
	Approved  bool                 `json:"Approved"`
	Resolver  types.ActorID        `json:"Resolver"`
	Reason    types.Option[string] `json:"Reason"`
}

func (c AuthorityResolvedContent) EventTypeName() string    { return "authority.resolved" }
func (c AuthorityResolvedContent) Accept(v EventContentVisitor) { v.VisitAuthorityResolved(c) }

// AuthorityDelegatedContent is emitted when authority is delegated.
type AuthorityDelegatedContent struct {
	From      types.ActorID                `json:"From"`
	To        types.ActorID                `json:"To"`
	Scope     types.DomainScope            `json:"Scope"`
	Weight    types.Score                  `json:"Weight"`
	ExpiresAt types.Option[types.Timestamp] `json:"ExpiresAt"`
}

func (c AuthorityDelegatedContent) EventTypeName() string    { return "authority.delegated" }
func (c AuthorityDelegatedContent) Accept(v EventContentVisitor) { v.VisitAuthorityDelegated(c) }

// AuthorityRevokedContent is emitted when authority is revoked.
type AuthorityRevokedContent struct {
	From   types.ActorID     `json:"From"`
	To     types.ActorID     `json:"To"`
	Scope  types.DomainScope `json:"Scope"`
	Reason types.EventID     `json:"Reason"`
}

func (c AuthorityRevokedContent) EventTypeName() string    { return "authority.revoked" }
func (c AuthorityRevokedContent) Accept(v EventContentVisitor) { v.VisitAuthorityRevoked(c) }

// AuthorityTimeoutContent is emitted when an authority request times out.
type AuthorityTimeoutContent struct {
	RequestID types.EventID  `json:"RequestID"`
	Level     AuthorityLevel `json:"Level"`
	Duration  types.Duration `json:"Duration"`
}

func (c AuthorityTimeoutContent) EventTypeName() string    { return "authority.timeout" }
func (c AuthorityTimeoutContent) Accept(v EventContentVisitor) { v.VisitAuthorityTimeout(c) }

// --- Actor content ---

// ActorRegisteredContent is emitted when a new actor is registered.
type ActorRegisteredContent struct {
	ActorID   types.ActorID  `json:"ActorID"`
	PublicKey types.PublicKey `json:"PublicKey"`
	Type      ActorType      `json:"Type"`
}

func (c ActorRegisteredContent) EventTypeName() string    { return "actor.registered" }
func (c ActorRegisteredContent) Accept(v EventContentVisitor) { v.VisitActorRegistered(c) }

// ActorSuspendedContent is emitted when an actor is suspended.
type ActorSuspendedContent struct {
	ActorID types.ActorID `json:"ActorID"`
	Reason  types.EventID `json:"Reason"`
}

func (c ActorSuspendedContent) EventTypeName() string    { return "actor.suspended" }
func (c ActorSuspendedContent) Accept(v EventContentVisitor) { v.VisitActorSuspended(c) }

// ActorMemorialContent is emitted when an actor is memorialised.
type ActorMemorialContent struct {
	ActorID types.ActorID `json:"ActorID"`
	Reason  types.EventID `json:"Reason"`
}

func (c ActorMemorialContent) EventTypeName() string    { return "actor.memorial" }
func (c ActorMemorialContent) Accept(v EventContentVisitor) { v.VisitActorMemorial(c) }

// --- Edge content ---

// EdgeCreatedContent is emitted when a new edge is created.
type EdgeCreatedContent struct {
	From      types.ActorID                `json:"From"`
	To        types.ActorID                `json:"To"`
	EdgeType  EdgeType                     `json:"EdgeType"`
	Weight    types.Weight                 `json:"Weight"`
	Direction EdgeDirection                `json:"Direction"`
	Scope     types.Option[types.DomainScope]  `json:"Scope"`
	ExpiresAt types.Option[types.Timestamp]    `json:"ExpiresAt"`
}

func (c EdgeCreatedContent) EventTypeName() string    { return "edge.created" }
func (c EdgeCreatedContent) Accept(v EventContentVisitor) { v.VisitEdgeCreated(c) }

// EdgeSupersededContent is emitted when an edge is superseded.
type EdgeSupersededContent struct {
	PreviousEdge types.EdgeID              `json:"PreviousEdge"`
	NewEdge      types.Option[types.EdgeID] `json:"NewEdge"`
	Reason       types.EventID             `json:"Reason"`
}

func (c EdgeSupersededContent) EventTypeName() string    { return "edge.superseded" }
func (c EdgeSupersededContent) Accept(v EventContentVisitor) { v.VisitEdgeSuperseded(c) }

// --- Integrity content ---

// ViolationDetectedContent is emitted when a violation is detected.
type ViolationDetectedContent struct {
	Expectation types.EventID                `json:"Expectation"`
	Actor       types.ActorID                `json:"Actor"`
	Severity    SeverityLevel                `json:"Severity"`
	Description string                       `json:"Description"`
	Evidence    types.NonEmpty[types.EventID] `json:"Evidence"`
}

func (c ViolationDetectedContent) EventTypeName() string    { return "violation.detected" }
func (c ViolationDetectedContent) Accept(v EventContentVisitor) { v.VisitViolationDetected(c) }

// ChainVerifiedContent is emitted after chain verification completes.
type ChainVerifiedContent struct {
	Valid    bool           `json:"Valid"`
	Length   int            `json:"Length"`
	Duration types.Duration `json:"Duration"`
}

func (c ChainVerifiedContent) EventTypeName() string    { return "chain.verified" }
func (c ChainVerifiedContent) Accept(v EventContentVisitor) { v.VisitChainVerified(c) }

// ChainBrokenContent is emitted when a chain break is detected.
type ChainBrokenContent struct {
	Position int        `json:"Position"`
	Expected types.Hash `json:"Expected"`
	Actual   types.Hash `json:"Actual"`
}

func (c ChainBrokenContent) EventTypeName() string    { return "chain.broken" }
func (c ChainBrokenContent) Accept(v EventContentVisitor) { v.VisitChainBroken(c) }

// --- System content ---

// BootstrapContent is emitted for the genesis event.
type BootstrapContent struct {
	ActorID      types.ActorID   `json:"ActorID"`
	ChainGenesis types.Hash      `json:"ChainGenesis"`
	Timestamp    types.Timestamp `json:"Timestamp"`
}

func (c BootstrapContent) EventTypeName() string    { return "system.bootstrapped" }
func (c BootstrapContent) Accept(v EventContentVisitor) { v.VisitBootstrap(c) }

// ClockTickContent is emitted for each tick.
type ClockTickContent struct {
	Tick      types.Tick      `json:"Tick"`
	Timestamp types.Timestamp `json:"Timestamp"`
	Elapsed   types.Duration  `json:"Elapsed"`
}

func (c ClockTickContent) EventTypeName() string    { return "clock.tick" }
func (c ClockTickContent) Accept(v EventContentVisitor) { v.VisitClockTick(c) }

// HealthReportContent is emitted for health checks.
// Use NewHealthReportContent to ensure defensive copying of PrimitiveHealth.
type HealthReportContent struct {
	Overall         types.Score                       `json:"Overall"`
	ChainIntegrity  bool                              `json:"ChainIntegrity"`
	primitiveHealth map[types.PrimitiveID]types.Score
	ActiveActors    int                               `json:"ActiveActors"`
	EventRate       float64                           `json:"EventRate"`
}

// PrimitiveHealth returns a defensive copy of the primitive health map.
func (c HealthReportContent) PrimitiveHealth() map[types.PrimitiveID]types.Score {
	if c.primitiveHealth == nil {
		return nil
	}
	cp := make(map[types.PrimitiveID]types.Score, len(c.primitiveHealth))
	for k, v := range c.primitiveHealth {
		cp[k] = v
	}
	return cp
}

// MarshalJSON implements json.Marshaler, including the unexported primitiveHealth field.
func (c HealthReportContent) MarshalJSON() ([]byte, error) {
	type alias struct {
		Overall         types.Score                       `json:"Overall"`
		ChainIntegrity  bool                              `json:"ChainIntegrity"`
		PrimitiveHealth map[types.PrimitiveID]types.Score `json:"PrimitiveHealth,omitempty"`
		ActiveActors    int                               `json:"ActiveActors"`
		EventRate       float64                           `json:"EventRate"`
	}
	return json.Marshal(alias{
		Overall:         c.Overall,
		ChainIntegrity:  c.ChainIntegrity,
		PrimitiveHealth: c.primitiveHealth,
		ActiveActors:    c.ActiveActors,
		EventRate:       c.EventRate,
	})
}

// UnmarshalJSON implements json.Unmarshaler, populating the unexported primitiveHealth field.
func (c *HealthReportContent) UnmarshalJSON(data []byte) error {
	type alias struct {
		Overall         types.Score                       `json:"Overall"`
		ChainIntegrity  bool                              `json:"ChainIntegrity"`
		PrimitiveHealth map[types.PrimitiveID]types.Score `json:"PrimitiveHealth,omitempty"`
		ActiveActors    int                               `json:"ActiveActors"`
		EventRate       float64                           `json:"EventRate"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	c.Overall = a.Overall
	c.ChainIntegrity = a.ChainIntegrity
	c.primitiveHealth = a.PrimitiveHealth
	c.ActiveActors = a.ActiveActors
	c.EventRate = a.EventRate
	return nil
}

// NewHealthReportContent creates a HealthReportContent with a defensive copy of the health map.
// Panics if eventRate is NaN or Inf — these are not valid JSON numbers and would
// break canonical form hashing.
func NewHealthReportContent(overall types.Score, chainIntegrity bool, primitiveHealth map[types.PrimitiveID]types.Score, activeActors int, eventRate float64) HealthReportContent {
	if math.IsNaN(eventRate) || math.IsInf(eventRate, 0) {
		panic(fmt.Sprintf("NewHealthReportContent: eventRate must be finite, got %v", eventRate))
	}
	// Normalize: empty or nil map → nil, so that canonical form is consistent
	// (omitempty omits nil maps; an empty non-nil map would emit "PrimitiveHealth":{}
	// producing a different hash for semantically equivalent content).
	var ph map[types.PrimitiveID]types.Score
	if len(primitiveHealth) > 0 {
		ph = make(map[types.PrimitiveID]types.Score, len(primitiveHealth))
		for k, v := range primitiveHealth {
			ph[k] = v
		}
	}
	return HealthReportContent{
		Overall:         overall,
		ChainIntegrity:  chainIntegrity,
		primitiveHealth: ph,
		ActiveActors:    activeActors,
		EventRate:       eventRate,
	}
}

func (c HealthReportContent) EventTypeName() string    { return "health.report" }
func (c HealthReportContent) Accept(v EventContentVisitor) { v.VisitHealthReport(c) }

// --- Decision tree content ---

// BranchProposedContent is emitted when a decision tree branch is proposed.
type BranchProposedContent struct {
	PrimitiveID types.PrimitiveID `json:"PrimitiveID"`
	TreeVersion int               `json:"TreeVersion"`
	Condition   Condition         `json:"Condition"`
	Outcome     DecisionOutcome   `json:"Outcome"`
	Accuracy    types.Score       `json:"Accuracy"`
	SampleSize  int               `json:"SampleSize"`
}

func (c BranchProposedContent) EventTypeName() string    { return "decision.branch.proposed" }
func (c BranchProposedContent) Accept(v EventContentVisitor) { v.VisitBranchProposed(c) }

// BranchInsertedContent is emitted when a decision tree branch is inserted.
type BranchInsertedContent struct {
	PrimitiveID types.PrimitiveID `json:"PrimitiveID"`
	TreeVersion int               `json:"TreeVersion"`
	Path        []PathStep        `json:"Path"`
	Outcome     DecisionOutcome   `json:"Outcome"`
	Confidence  types.Score       `json:"Confidence"`
}

func (c BranchInsertedContent) EventTypeName() string    { return "decision.branch.inserted" }
func (c BranchInsertedContent) Accept(v EventContentVisitor) { v.VisitBranchInserted(c) }

// CostReportContent is emitted for decision tree cost reports.
type CostReportContent struct {
	PrimitiveID    types.PrimitiveID `json:"PrimitiveID"`
	TreeVersion    int               `json:"TreeVersion"`
	TotalLeaves    int               `json:"TotalLeaves"`
	LLMLeaves      int               `json:"LLMLeaves"`
	MechanicalRate types.Score       `json:"MechanicalRate"`
	TotalTokens    int               `json:"TotalTokens"`
}

func (c CostReportContent) EventTypeName() string    { return "decision.cost.report" }
func (c CostReportContent) Accept(v EventContentVisitor) { v.VisitCostReport(c) }

// --- Decision tree support types ---

// Condition represents a decision tree condition.
type Condition struct {
	Field     types.FieldPath          `json:"Field"`
	Operator  ConditionOperator        `json:"Operator"`
	Threshold types.Option[types.Score] `json:"Threshold"`
	Prompt    types.Option[string]     `json:"Prompt"`
}

// MatchValue is a tagged union — exactly one field must be Some.
type MatchValue struct {
	String    types.Option[string]         `json:"String"`
	Number    types.Option[float64]        `json:"Number"`
	Boolean   types.Option[bool]           `json:"Boolean"`
	EventType types.Option[types.EventType] `json:"EventType"`
}

// MarshalJSON validates that Number is finite before serializing.
// Inf/NaN cannot be represented in JSON and would cause canonicalContentJSON to panic.
func (mv MatchValue) MarshalJSON() ([]byte, error) {
	if mv.Number.IsSome() {
		n := mv.Number.Unwrap()
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, fmt.Errorf("MatchValue.Number must be finite, got %v", n)
		}
	}
	// Use alias to avoid infinite recursion
	type matchValueAlias MatchValue
	return json.Marshal(matchValueAlias(mv))
}

// PathStep records a step taken in a decision tree traversal.
type PathStep struct {
	Condition Condition  `json:"Condition"`
	Branch    MatchValue `json:"Branch"`
}

// --- Social grammar content ---

// GrammarEmitContent is emitted when independent content is created.
type GrammarEmitContent struct {
	Body string `json:"Body"`
}

func (c GrammarEmitContent) EventTypeName() string          { return "grammar.emit" }
func (c GrammarEmitContent) Accept(v EventContentVisitor)   { v.VisitGrammarEmit(c) }

// GrammarRespondContent is emitted when causally dependent, subordinate content is created.
type GrammarRespondContent struct {
	Body   string        `json:"Body"`
	Parent types.EventID `json:"Parent"`
}

func (c GrammarRespondContent) EventTypeName() string        { return "grammar.respond" }
func (c GrammarRespondContent) Accept(v EventContentVisitor) { v.VisitGrammarRespond(c) }

// GrammarDeriveContent is emitted when causally dependent but independent content is created.
type GrammarDeriveContent struct {
	Body   string        `json:"Body"`
	Source types.EventID `json:"Source"`
}

func (c GrammarDeriveContent) EventTypeName() string        { return "grammar.derive" }
func (c GrammarDeriveContent) Accept(v EventContentVisitor) { v.VisitGrammarDerive(c) }

// GrammarExtendContent is emitted when sequential content from the same author is created.
type GrammarExtendContent struct {
	Body     string        `json:"Body"`
	Previous types.EventID `json:"Previous"`
}

func (c GrammarExtendContent) EventTypeName() string        { return "grammar.extend" }
func (c GrammarExtendContent) Accept(v EventContentVisitor) { v.VisitGrammarExtend(c) }

// GrammarRetractContent is emitted when own content is tombstoned.
type GrammarRetractContent struct {
	Target types.EventID `json:"Target"`
	Reason string        `json:"Reason"`
}

func (c GrammarRetractContent) EventTypeName() string        { return "grammar.retract" }
func (c GrammarRetractContent) Accept(v EventContentVisitor) { v.VisitGrammarRetract(c) }

// GrammarAnnotateContent is emitted when metadata is attached to existing content.
type GrammarAnnotateContent struct {
	Target types.EventID `json:"Target"`
	Key    string        `json:"Key"`
	Value  string        `json:"Value"`
}

func (c GrammarAnnotateContent) EventTypeName() string        { return "grammar.annotate" }
func (c GrammarAnnotateContent) Accept(v EventContentVisitor) { v.VisitGrammarAnnotate(c) }

// GrammarMergeContent is emitted when two or more independent subtrees are joined.
// Sources are sorted lexicographically for deterministic hashing.
// Use NewGrammarMergeContent to ensure sorting.
type GrammarMergeContent struct {
	Body    string          `json:"Body"`
	Sources []types.EventID `json:"Sources"`
}

// NewGrammarMergeContent creates a GrammarMergeContent with Sources sorted
// lexicographically. This ensures semantically equivalent merges produce
// identical hashes regardless of the order Sources are provided.
func NewGrammarMergeContent(body string, sources []types.EventID) GrammarMergeContent {
	sorted := make([]types.EventID, len(sources))
	copy(sorted, sources)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value() < sorted[j].Value()
	})
	return GrammarMergeContent{Body: body, Sources: sorted}
}

func (c GrammarMergeContent) EventTypeName() string        { return "grammar.merge" }
func (c GrammarMergeContent) Accept(v EventContentVisitor) { v.VisitGrammarMerge(c) }

// GrammarConsentContent is emitted for a mutual, atomic, dual-signed event.
// Parties are sorted lexicographically for deterministic hashing.
// Use NewGrammarConsentContent to ensure sorting.
type GrammarConsentContent struct {
	Parties   [2]types.ActorID  `json:"Parties"`
	Agreement string            `json:"Agreement"`
	Scope     types.DomainScope `json:"Scope"`
}

// NewGrammarConsentContent creates a GrammarConsentContent with Parties sorted
// lexicographically. This ensures consent(A,B) and consent(B,A) produce
// identical hashes for the same agreement.
func NewGrammarConsentContent(partyA, partyB types.ActorID, agreement string, scope types.DomainScope) GrammarConsentContent {
	parties := [2]types.ActorID{partyA, partyB}
	if partyA.Value() > partyB.Value() {
		parties[0], parties[1] = partyB, partyA
	}
	return GrammarConsentContent{Parties: parties, Agreement: agreement, Scope: scope}
}

func (c GrammarConsentContent) EventTypeName() string        { return "grammar.consent" }
func (c GrammarConsentContent) Accept(v EventContentVisitor) { v.VisitGrammarConsent(c) }

// --- EGIP content ---

// EGIPHelloSentContent records a HELLO sent to a remote system.
type EGIPHelloSentContent struct {
	To types.SystemURI `json:"To"`
}

func (c EGIPHelloSentContent) EventTypeName() string    { return "egip.hello.sent" }
func (c EGIPHelloSentContent) Accept(v EventContentVisitor) { v.VisitEGIPHelloSent(c) }

// EGIPHelloReceivedContent records a HELLO received from a remote system.
type EGIPHelloReceivedContent struct {
	From      types.SystemURI `json:"From"`
	PublicKey types.PublicKey  `json:"PublicKey"`
}

func (c EGIPHelloReceivedContent) EventTypeName() string    { return "egip.hello.received" }
func (c EGIPHelloReceivedContent) Accept(v EventContentVisitor) { v.VisitEGIPHelloReceived(c) }

// EGIPMessageSentContent records a message sent to a remote system.
type EGIPMessageSentContent struct {
	To         types.SystemURI  `json:"To"`
	EnvelopeID types.EnvelopeID `json:"EnvelopeID"`
}

func (c EGIPMessageSentContent) EventTypeName() string    { return "egip.message.sent" }
func (c EGIPMessageSentContent) Accept(v EventContentVisitor) { v.VisitEGIPMessageSent(c) }

// EGIPMessageReceivedContent records a message received from a remote system.
type EGIPMessageReceivedContent struct {
	From       types.SystemURI  `json:"From"`
	EnvelopeID types.EnvelopeID `json:"EnvelopeID"`
}

func (c EGIPMessageReceivedContent) EventTypeName() string    { return "egip.message.received" }
func (c EGIPMessageReceivedContent) Accept(v EventContentVisitor) { v.VisitEGIPMessageReceived(c) }

// EGIPReceiptSentContent records a receipt sent for an envelope.
type EGIPReceiptSentContent struct {
	EnvelopeID types.EnvelopeID `json:"EnvelopeID"`
	Status     ReceiptStatus    `json:"Status"`
}

func (c EGIPReceiptSentContent) EventTypeName() string    { return "egip.receipt.sent" }
func (c EGIPReceiptSentContent) Accept(v EventContentVisitor) { v.VisitEGIPReceiptSent(c) }

// EGIPReceiptReceivedContent records a receipt received for an envelope.
type EGIPReceiptReceivedContent struct {
	EnvelopeID types.EnvelopeID `json:"EnvelopeID"`
	Status     ReceiptStatus    `json:"Status"`
}

func (c EGIPReceiptReceivedContent) EventTypeName() string    { return "egip.receipt.received" }
func (c EGIPReceiptReceivedContent) Accept(v EventContentVisitor) { v.VisitEGIPReceiptReceived(c) }

// EGIPProofRequestedContent records a proof request to a remote system.
type EGIPProofRequestedContent struct {
	System    types.SystemURI `json:"System"`
	ProofType ProofType       `json:"ProofType"`
}

func (c EGIPProofRequestedContent) EventTypeName() string    { return "egip.proof.requested" }
func (c EGIPProofRequestedContent) Accept(v EventContentVisitor) { v.VisitEGIPProofRequested(c) }

// EGIPProofReceivedContent records a proof received from a remote system.
type EGIPProofReceivedContent struct {
	System types.SystemURI `json:"System"`
	Valid  bool            `json:"Valid"`
}

func (c EGIPProofReceivedContent) EventTypeName() string    { return "egip.proof.received" }
func (c EGIPProofReceivedContent) Accept(v EventContentVisitor) { v.VisitEGIPProofReceived(c) }

// EGIPTreatyProposedContent records a treaty proposal sent.
type EGIPTreatyProposedContent struct {
	TreatyID types.TreatyID  `json:"TreatyID"`
	To       types.SystemURI `json:"To"`
}

func (c EGIPTreatyProposedContent) EventTypeName() string    { return "egip.treaty.proposed" }
func (c EGIPTreatyProposedContent) Accept(v EventContentVisitor) { v.VisitEGIPTreatyProposed(c) }

// EGIPTreatyActiveContent records a treaty becoming active.
type EGIPTreatyActiveContent struct {
	TreatyID types.TreatyID  `json:"TreatyID"`
	With     types.SystemURI `json:"With"`
}

func (c EGIPTreatyActiveContent) EventTypeName() string    { return "egip.treaty.active" }
func (c EGIPTreatyActiveContent) Accept(v EventContentVisitor) { v.VisitEGIPTreatyActive(c) }

// EGIPTrustUpdatedContent records inter-system trust changes.
type EGIPTrustUpdatedContent struct {
	System   types.SystemURI  `json:"System"`
	Previous types.Score      `json:"Previous"`
	Current  types.Score      `json:"Current"`
	Evidence types.EnvelopeID `json:"Evidence"`
}

func (c EGIPTrustUpdatedContent) EventTypeName() string    { return "egip.trust.updated" }
func (c EGIPTrustUpdatedContent) Accept(v EventContentVisitor) { v.VisitEGIPTrustUpdated(c) }

// --- Event Type Registry ---

// EventTypeRegistration holds a registered event type and its validator.
type EventTypeRegistration struct {
	Type     types.EventType
	Validate func(EventContent) error
}

// EventTypeRegistry maps event type strings to their content schemas.
// Thread-safe for concurrent access.
type EventTypeRegistry struct {
	mu    sync.RWMutex
	types map[string]EventTypeRegistration
}

// NewEventTypeRegistry creates a new empty registry.
func NewEventTypeRegistry() *EventTypeRegistry {
	return &EventTypeRegistry{types: make(map[string]EventTypeRegistration)}
}

// Register adds an event type to the registry.
func (r *EventTypeRegistry) Register(et types.EventType, validate func(EventContent) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types[et.Value()] = EventTypeRegistration{Type: et, Validate: validate}
}

// Validate checks that content matches the registered schema for the given type.
func (r *EventTypeRegistry) Validate(et types.EventType, content EventContent) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.types[et.Value()]
	if !ok {
		return fmt.Errorf("unregistered event type: %s", et.Value())
	}
	if content.EventTypeName() != et.Value() {
		return fmt.Errorf("content type %q does not match event type %q", content.EventTypeName(), et.Value())
	}
	if reg.Validate != nil {
		return reg.Validate(content)
	}
	return nil
}

// IsRegistered returns true if the event type is registered.
func (r *EventTypeRegistry) IsRegistered(et types.EventType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.types[et.Value()]
	return ok
}

// AllTypes returns all registered event types.
func (r *EventTypeRegistry) AllTypes() []types.EventType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]types.EventType, 0, len(r.types))
	for _, reg := range r.types {
		result = append(result, reg.Type)
	}
	return result
}

// DefaultRegistry returns a registry with all standard event types registered.
func DefaultRegistry() *EventTypeRegistry {
	r := NewEventTypeRegistry()
	for _, et := range []types.EventType{
		EventTypeTrustUpdated, EventTypeTrustScore, EventTypeTrustDecayed,
		EventTypeAuthorityRequested, EventTypeAuthorityResolved, EventTypeAuthorityDelegated,
		EventTypeAuthorityRevoked, EventTypeAuthorityTimeout,
		EventTypeActorRegistered, EventTypeActorSuspended, EventTypeActorMemorial,
		EventTypeEdgeCreated, EventTypeEdgeSuperseded,
		EventTypeViolationDetected, EventTypeChainVerified, EventTypeChainBroken,
		EventTypeSystemBootstrapped, EventTypeClockTick, EventTypeHealthReport,
		EventTypeDecisionBranchProposed, EventTypeDecisionBranchInserted, EventTypeDecisionCostReport,
		EventTypeGrammarEmit, EventTypeGrammarRespond, EventTypeGrammarDerive,
		EventTypeGrammarExtend, EventTypeGrammarRetract, EventTypeGrammarAnnotate,
		EventTypeGrammarMerge, EventTypeGrammarConsent,
		EventTypeEGIPHelloSent, EventTypeEGIPHelloReceived,
		EventTypeEGIPMessageSent, EventTypeEGIPMessageReceived,
		EventTypeEGIPReceiptSent, EventTypeEGIPReceiptReceived,
		EventTypeEGIPProofRequested, EventTypeEGIPProofReceived,
		EventTypeEGIPTreatyProposed, EventTypeEGIPTreatyActive,
		EventTypeEGIPTrustUpdated,
	} {
		r.Register(et, nil)
	}
	return r
}
