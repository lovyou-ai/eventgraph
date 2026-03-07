package event

import (
	"fmt"

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
	Actor    types.ActorID
	Previous types.Score
	Current  types.Score
	Domain   types.DomainScope
	Cause    types.EventID
}

func (c TrustUpdatedContent) EventTypeName() string    { return "trust.updated" }
func (c TrustUpdatedContent) Accept(v EventContentVisitor) { v.VisitTrustUpdated(c) }

// TrustScoreContent is emitted when a trust score snapshot is recorded.
type TrustScoreContent struct {
	Actor   types.ActorID
	Metrics TrustMetrics
}

func (c TrustScoreContent) EventTypeName() string    { return "trust.score" }
func (c TrustScoreContent) Accept(v EventContentVisitor) { v.VisitTrustScore(c) }

// TrustDecayedContent is emitted when trust decays over time.
type TrustDecayedContent struct {
	Actor    types.ActorID
	Previous types.Score
	Current  types.Score
	Elapsed  types.Duration
	Rate     types.Score
}

func (c TrustDecayedContent) EventTypeName() string    { return "trust.decayed" }
func (c TrustDecayedContent) Accept(v EventContentVisitor) { v.VisitTrustDecayed(c) }

// --- Authority content ---

// AuthorityRequestContent is emitted when an authority request is made.
type AuthorityRequestContent struct {
	Action        string
	Actor         types.ActorID
	Level         AuthorityLevel
	Justification string
	Causes        types.NonEmpty[types.EventID]
}

func (c AuthorityRequestContent) EventTypeName() string    { return "authority.requested" }
func (c AuthorityRequestContent) Accept(v EventContentVisitor) { v.VisitAuthorityRequested(c) }

// AuthorityResolvedContent is emitted when an authority request is resolved.
type AuthorityResolvedContent struct {
	RequestID types.EventID
	Approved  bool
	Resolver  types.ActorID
	Reason    types.Option[string]
}

func (c AuthorityResolvedContent) EventTypeName() string    { return "authority.resolved" }
func (c AuthorityResolvedContent) Accept(v EventContentVisitor) { v.VisitAuthorityResolved(c) }

// AuthorityDelegatedContent is emitted when authority is delegated.
type AuthorityDelegatedContent struct {
	From      types.ActorID
	To        types.ActorID
	Scope     types.DomainScope
	Weight    types.Score
	ExpiresAt types.Option[types.Timestamp]
}

func (c AuthorityDelegatedContent) EventTypeName() string    { return "authority.delegated" }
func (c AuthorityDelegatedContent) Accept(v EventContentVisitor) { v.VisitAuthorityDelegated(c) }

// AuthorityRevokedContent is emitted when authority is revoked.
type AuthorityRevokedContent struct {
	From   types.ActorID
	To     types.ActorID
	Scope  types.DomainScope
	Reason types.EventID
}

func (c AuthorityRevokedContent) EventTypeName() string    { return "authority.revoked" }
func (c AuthorityRevokedContent) Accept(v EventContentVisitor) { v.VisitAuthorityRevoked(c) }

// AuthorityTimeoutContent is emitted when an authority request times out.
type AuthorityTimeoutContent struct {
	RequestID types.EventID
	Level     AuthorityLevel
	Duration  types.Duration
}

func (c AuthorityTimeoutContent) EventTypeName() string    { return "authority.timeout" }
func (c AuthorityTimeoutContent) Accept(v EventContentVisitor) { v.VisitAuthorityTimeout(c) }

// --- Actor content ---

// ActorRegisteredContent is emitted when a new actor is registered.
type ActorRegisteredContent struct {
	ActorID   types.ActorID
	PublicKey types.PublicKey
	Type      ActorType
}

func (c ActorRegisteredContent) EventTypeName() string    { return "actor.registered" }
func (c ActorRegisteredContent) Accept(v EventContentVisitor) { v.VisitActorRegistered(c) }

// ActorSuspendedContent is emitted when an actor is suspended.
type ActorSuspendedContent struct {
	ActorID types.ActorID
	Reason  types.EventID
}

func (c ActorSuspendedContent) EventTypeName() string    { return "actor.suspended" }
func (c ActorSuspendedContent) Accept(v EventContentVisitor) { v.VisitActorSuspended(c) }

// ActorMemorialContent is emitted when an actor is memorialised.
type ActorMemorialContent struct {
	ActorID types.ActorID
	Reason  types.EventID
}

func (c ActorMemorialContent) EventTypeName() string    { return "actor.memorial" }
func (c ActorMemorialContent) Accept(v EventContentVisitor) { v.VisitActorMemorial(c) }

// --- Edge content ---

// EdgeCreatedContent is emitted when a new edge is created.
type EdgeCreatedContent struct {
	From      types.ActorID
	To        types.ActorID
	EdgeType  EdgeType
	Weight    types.Weight
	Direction EdgeDirection
	Scope     types.Option[types.DomainScope]
	ExpiresAt types.Option[types.Timestamp]
}

func (c EdgeCreatedContent) EventTypeName() string    { return "edge.created" }
func (c EdgeCreatedContent) Accept(v EventContentVisitor) { v.VisitEdgeCreated(c) }

// EdgeSupersededContent is emitted when an edge is superseded.
type EdgeSupersededContent struct {
	PreviousEdge types.EdgeID
	NewEdge      types.EdgeID
	Reason       types.EventID
}

func (c EdgeSupersededContent) EventTypeName() string    { return "edge.superseded" }
func (c EdgeSupersededContent) Accept(v EventContentVisitor) { v.VisitEdgeSuperseded(c) }

// --- Integrity content ---

// ViolationDetectedContent is emitted when a violation is detected.
type ViolationDetectedContent struct {
	Expectation types.EventID
	Actor       types.ActorID
	Severity    SeverityLevel
	Description string
	Evidence    types.NonEmpty[types.EventID]
}

func (c ViolationDetectedContent) EventTypeName() string    { return "violation.detected" }
func (c ViolationDetectedContent) Accept(v EventContentVisitor) { v.VisitViolationDetected(c) }

// ChainVerifiedContent is emitted after chain verification completes.
type ChainVerifiedContent struct {
	Valid    bool
	Length   int
	Duration types.Duration
}

func (c ChainVerifiedContent) EventTypeName() string    { return "chain.verified" }
func (c ChainVerifiedContent) Accept(v EventContentVisitor) { v.VisitChainVerified(c) }

// ChainBrokenContent is emitted when a chain break is detected.
type ChainBrokenContent struct {
	Position int
	Expected types.Hash
	Actual   types.Hash
}

func (c ChainBrokenContent) EventTypeName() string    { return "chain.broken" }
func (c ChainBrokenContent) Accept(v EventContentVisitor) { v.VisitChainBroken(c) }

// --- System content ---

// BootstrapContent is emitted for the genesis event.
type BootstrapContent struct {
	ActorID      types.ActorID
	ChainGenesis types.Hash
	Timestamp    types.Timestamp
}

func (c BootstrapContent) EventTypeName() string    { return "system.bootstrapped" }
func (c BootstrapContent) Accept(v EventContentVisitor) { v.VisitBootstrap(c) }

// ClockTickContent is emitted for each tick.
type ClockTickContent struct {
	Tick      types.Tick
	Timestamp types.Timestamp
	Elapsed   types.Duration
}

func (c ClockTickContent) EventTypeName() string    { return "clock.tick" }
func (c ClockTickContent) Accept(v EventContentVisitor) { v.VisitClockTick(c) }

// HealthReportContent is emitted for health checks.
type HealthReportContent struct {
	Overall         types.Score
	ChainIntegrity  bool
	PrimitiveHealth map[types.PrimitiveID]types.Score
	ActiveActors    int
	EventRate       float64
}

func (c HealthReportContent) EventTypeName() string    { return "health.report" }
func (c HealthReportContent) Accept(v EventContentVisitor) { v.VisitHealthReport(c) }

// --- Decision tree content ---

// BranchProposedContent is emitted when a decision tree branch is proposed.
type BranchProposedContent struct {
	PrimitiveID types.PrimitiveID
	TreeVersion int
	Condition   Condition
	Outcome     DecisionOutcome
	Accuracy    types.Score
	SampleSize  int
}

func (c BranchProposedContent) EventTypeName() string    { return "decision.branch.proposed" }
func (c BranchProposedContent) Accept(v EventContentVisitor) { v.VisitBranchProposed(c) }

// BranchInsertedContent is emitted when a decision tree branch is inserted.
type BranchInsertedContent struct {
	PrimitiveID types.PrimitiveID
	TreeVersion int
	Path        []PathStep
	Outcome     DecisionOutcome
	Confidence  types.Score
}

func (c BranchInsertedContent) EventTypeName() string    { return "decision.branch.inserted" }
func (c BranchInsertedContent) Accept(v EventContentVisitor) { v.VisitBranchInserted(c) }

// CostReportContent is emitted for decision tree cost reports.
type CostReportContent struct {
	PrimitiveID    types.PrimitiveID
	TreeVersion    int
	TotalLeaves    int
	LLMLeaves      int
	MechanicalRate types.Score
	TotalTokens    int
}

func (c CostReportContent) EventTypeName() string    { return "decision.cost.report" }
func (c CostReportContent) Accept(v EventContentVisitor) { v.VisitCostReport(c) }

// --- Decision tree support types ---

// Condition represents a decision tree condition.
type Condition struct {
	Field     types.FieldPath
	Operator  ConditionOperator
	Threshold types.Option[types.Score]
	Prompt    types.Option[string]
}

// MatchValue is a tagged union — exactly one field must be Some.
type MatchValue struct {
	String    types.Option[string]
	Number    types.Option[float64]
	Boolean   types.Option[bool]
	EventType types.Option[types.EventType]
}

// PathStep records a step taken in a decision tree traversal.
type PathStep struct {
	Condition Condition
	Branch    MatchValue
}

// --- Social grammar content ---

// GrammarEmitContent is emitted when independent content is created.
type GrammarEmitContent struct {
	Body string
}

func (c GrammarEmitContent) EventTypeName() string          { return "grammar.emit" }
func (c GrammarEmitContent) Accept(v EventContentVisitor)   { v.VisitGrammarEmit(c) }

// GrammarRespondContent is emitted when causally dependent, subordinate content is created.
type GrammarRespondContent struct {
	Body   string
	Parent types.EventID
}

func (c GrammarRespondContent) EventTypeName() string        { return "grammar.respond" }
func (c GrammarRespondContent) Accept(v EventContentVisitor) { v.VisitGrammarRespond(c) }

// GrammarDeriveContent is emitted when causally dependent but independent content is created.
type GrammarDeriveContent struct {
	Body   string
	Source types.EventID
}

func (c GrammarDeriveContent) EventTypeName() string        { return "grammar.derive" }
func (c GrammarDeriveContent) Accept(v EventContentVisitor) { v.VisitGrammarDerive(c) }

// GrammarExtendContent is emitted when sequential content from the same author is created.
type GrammarExtendContent struct {
	Body     string
	Previous types.EventID
}

func (c GrammarExtendContent) EventTypeName() string        { return "grammar.extend" }
func (c GrammarExtendContent) Accept(v EventContentVisitor) { v.VisitGrammarExtend(c) }

// GrammarRetractContent is emitted when own content is tombstoned.
type GrammarRetractContent struct {
	Target types.EventID
	Reason string
}

func (c GrammarRetractContent) EventTypeName() string        { return "grammar.retract" }
func (c GrammarRetractContent) Accept(v EventContentVisitor) { v.VisitGrammarRetract(c) }

// GrammarAnnotateContent is emitted when metadata is attached to existing content.
type GrammarAnnotateContent struct {
	Target types.EventID
	Key    string
	Value  string
}

func (c GrammarAnnotateContent) EventTypeName() string        { return "grammar.annotate" }
func (c GrammarAnnotateContent) Accept(v EventContentVisitor) { v.VisitGrammarAnnotate(c) }

// GrammarMergeContent is emitted when two or more independent subtrees are joined.
type GrammarMergeContent struct {
	Body    string
	Sources []types.EventID
}

func (c GrammarMergeContent) EventTypeName() string        { return "grammar.merge" }
func (c GrammarMergeContent) Accept(v EventContentVisitor) { v.VisitGrammarMerge(c) }

// GrammarConsentContent is emitted for a mutual, atomic, dual-signed event.
type GrammarConsentContent struct {
	Parties   [2]types.ActorID
	Agreement string
	Scope     types.DomainScope
}

func (c GrammarConsentContent) EventTypeName() string        { return "grammar.consent" }
func (c GrammarConsentContent) Accept(v EventContentVisitor) { v.VisitGrammarConsent(c) }

// --- EGIP content ---

// EGIPHelloSentContent records a HELLO sent to a remote system.
type EGIPHelloSentContent struct {
	To types.SystemURI
}

func (c EGIPHelloSentContent) EventTypeName() string    { return "egip.hello.sent" }
func (c EGIPHelloSentContent) Accept(v EventContentVisitor) { v.VisitEGIPHelloSent(c) }

// EGIPHelloReceivedContent records a HELLO received from a remote system.
type EGIPHelloReceivedContent struct {
	From      types.SystemURI
	PublicKey types.PublicKey
}

func (c EGIPHelloReceivedContent) EventTypeName() string    { return "egip.hello.received" }
func (c EGIPHelloReceivedContent) Accept(v EventContentVisitor) { v.VisitEGIPHelloReceived(c) }

// EGIPMessageSentContent records a message sent to a remote system.
type EGIPMessageSentContent struct {
	To         types.SystemURI
	EnvelopeID types.EnvelopeID
}

func (c EGIPMessageSentContent) EventTypeName() string    { return "egip.message.sent" }
func (c EGIPMessageSentContent) Accept(v EventContentVisitor) { v.VisitEGIPMessageSent(c) }

// EGIPMessageReceivedContent records a message received from a remote system.
type EGIPMessageReceivedContent struct {
	From       types.SystemURI
	EnvelopeID types.EnvelopeID
}

func (c EGIPMessageReceivedContent) EventTypeName() string    { return "egip.message.received" }
func (c EGIPMessageReceivedContent) Accept(v EventContentVisitor) { v.VisitEGIPMessageReceived(c) }

// EGIPReceiptSentContent records a receipt sent for an envelope.
type EGIPReceiptSentContent struct {
	EnvelopeID types.EnvelopeID
	Status     ReceiptStatus
}

func (c EGIPReceiptSentContent) EventTypeName() string    { return "egip.receipt.sent" }
func (c EGIPReceiptSentContent) Accept(v EventContentVisitor) { v.VisitEGIPReceiptSent(c) }

// EGIPReceiptReceivedContent records a receipt received for an envelope.
type EGIPReceiptReceivedContent struct {
	EnvelopeID types.EnvelopeID
	Status     ReceiptStatus
}

func (c EGIPReceiptReceivedContent) EventTypeName() string    { return "egip.receipt.received" }
func (c EGIPReceiptReceivedContent) Accept(v EventContentVisitor) { v.VisitEGIPReceiptReceived(c) }

// EGIPProofRequestedContent records a proof request to a remote system.
type EGIPProofRequestedContent struct {
	System    types.SystemURI
	ProofType ProofType
}

func (c EGIPProofRequestedContent) EventTypeName() string    { return "egip.proof.requested" }
func (c EGIPProofRequestedContent) Accept(v EventContentVisitor) { v.VisitEGIPProofRequested(c) }

// EGIPProofReceivedContent records a proof received from a remote system.
type EGIPProofReceivedContent struct {
	System types.SystemURI
	Valid  bool
}

func (c EGIPProofReceivedContent) EventTypeName() string    { return "egip.proof.received" }
func (c EGIPProofReceivedContent) Accept(v EventContentVisitor) { v.VisitEGIPProofReceived(c) }

// EGIPTreatyProposedContent records a treaty proposal sent.
type EGIPTreatyProposedContent struct {
	TreatyID types.TreatyID
	To       types.SystemURI
}

func (c EGIPTreatyProposedContent) EventTypeName() string    { return "egip.treaty.proposed" }
func (c EGIPTreatyProposedContent) Accept(v EventContentVisitor) { v.VisitEGIPTreatyProposed(c) }

// EGIPTreatyActiveContent records a treaty becoming active.
type EGIPTreatyActiveContent struct {
	TreatyID types.TreatyID
	With     types.SystemURI
}

func (c EGIPTreatyActiveContent) EventTypeName() string    { return "egip.treaty.active" }
func (c EGIPTreatyActiveContent) Accept(v EventContentVisitor) { v.VisitEGIPTreatyActive(c) }

// EGIPTrustUpdatedContent records inter-system trust changes.
type EGIPTrustUpdatedContent struct {
	System   types.SystemURI
	Previous types.Score
	Current  types.Score
	Evidence types.EnvelopeID
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
type EventTypeRegistry struct {
	types map[string]EventTypeRegistration
}

// NewEventTypeRegistry creates a new empty registry.
func NewEventTypeRegistry() *EventTypeRegistry {
	return &EventTypeRegistry{types: make(map[string]EventTypeRegistration)}
}

// Register adds an event type to the registry.
func (r *EventTypeRegistry) Register(et types.EventType, validate func(EventContent) error) {
	r.types[et.Value()] = EventTypeRegistration{Type: et, Validate: validate}
}

// Validate checks that content matches the registered schema for the given type.
func (r *EventTypeRegistry) Validate(et types.EventType, content EventContent) error {
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
	_, ok := r.types[et.Value()]
	return ok
}

// AllTypes returns all registered event types.
func (r *EventTypeRegistry) AllTypes() []types.EventType {
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
