package event

import "fmt"

// EdgeType represents the type of relationship in the graph.
type EdgeType string

const (
	EdgeTypeTrust        EdgeType = "Trust"
	EdgeTypeAuthority    EdgeType = "Authority"
	EdgeTypeSubscription EdgeType = "Subscription"
	EdgeTypeEndorsement  EdgeType = "Endorsement"
	EdgeTypeDelegation   EdgeType = "Delegation"
	EdgeTypeCausation    EdgeType = "Causation"
	EdgeTypeReference    EdgeType = "Reference"
	EdgeTypeChannel      EdgeType = "Channel"
	EdgeTypeAnnotation   EdgeType = "Annotation"
)

var validEdgeTypes = map[EdgeType]bool{
	EdgeTypeTrust: true, EdgeTypeAuthority: true, EdgeTypeSubscription: true,
	EdgeTypeEndorsement: true, EdgeTypeDelegation: true, EdgeTypeCausation: true,
	EdgeTypeReference: true, EdgeTypeChannel: true, EdgeTypeAnnotation: true,
}

// IsValid returns true if the edge type is a known type.
func (t EdgeType) IsValid() bool { return validEdgeTypes[t] }

// EdgeTypeVisitor provides exhaustive dispatch over edge types.
type EdgeTypeVisitor interface {
	VisitTrust()
	VisitAuthority()
	VisitSubscription()
	VisitEndorsement()
	VisitDelegation()
	VisitCausation()
	VisitReference()
	VisitChannel()
	VisitAnnotation()
}

// Accept dispatches to the appropriate visitor method.
func (t EdgeType) Accept(v EdgeTypeVisitor) {
	switch t {
	case EdgeTypeTrust:
		v.VisitTrust()
	case EdgeTypeAuthority:
		v.VisitAuthority()
	case EdgeTypeSubscription:
		v.VisitSubscription()
	case EdgeTypeEndorsement:
		v.VisitEndorsement()
	case EdgeTypeDelegation:
		v.VisitDelegation()
	case EdgeTypeCausation:
		v.VisitCausation()
	case EdgeTypeReference:
		v.VisitReference()
	case EdgeTypeChannel:
		v.VisitChannel()
	case EdgeTypeAnnotation:
		v.VisitAnnotation()
	default:
		panic(fmt.Sprintf("EdgeType.Accept: unrecognized EdgeType %q", t))
	}
}

// AuthorityLevel represents the approval level required for an action.
type AuthorityLevel string

const (
	AuthorityLevelRequired     AuthorityLevel = "Required"
	AuthorityLevelRecommended  AuthorityLevel = "Recommended"
	AuthorityLevelNotification AuthorityLevel = "Notification"
)

var validAuthorityLevels = map[AuthorityLevel]bool{
	AuthorityLevelRequired: true, AuthorityLevelRecommended: true, AuthorityLevelNotification: true,
}

// IsValid returns true if the authority level is a known level.
func (l AuthorityLevel) IsValid() bool { return validAuthorityLevels[l] }

// AuthorityLevelVisitor provides exhaustive dispatch over authority levels.
type AuthorityLevelVisitor interface {
	VisitRequired()
	VisitRecommended()
	VisitNotification()
}

// Accept dispatches to the appropriate visitor method.
func (l AuthorityLevel) Accept(v AuthorityLevelVisitor) {
	switch l {
	case AuthorityLevelRequired:
		v.VisitRequired()
	case AuthorityLevelRecommended:
		v.VisitRecommended()
	case AuthorityLevelNotification:
		v.VisitNotification()
	default:
		panic(fmt.Sprintf("AuthorityLevel.Accept: unrecognized AuthorityLevel %q", l))
	}
}

// DecisionOutcome represents the result of a decision.
type DecisionOutcome string

const (
	DecisionOutcomePermit   DecisionOutcome = "Permit"
	DecisionOutcomeDeny     DecisionOutcome = "Deny"
	DecisionOutcomeDefer    DecisionOutcome = "Defer"
	DecisionOutcomeEscalate DecisionOutcome = "Escalate"
)

var validDecisionOutcomes = map[DecisionOutcome]bool{
	DecisionOutcomePermit: true, DecisionOutcomeDeny: true,
	DecisionOutcomeDefer: true, DecisionOutcomeEscalate: true,
}

// IsValid returns true if the decision outcome is a known outcome.
func (o DecisionOutcome) IsValid() bool { return validDecisionOutcomes[o] }

// DecisionOutcomeVisitor provides exhaustive dispatch over decision outcomes.
type DecisionOutcomeVisitor interface {
	VisitPermit()
	VisitDeny()
	VisitDefer()
	VisitEscalate()
}

// Accept dispatches to the appropriate visitor method.
func (o DecisionOutcome) Accept(v DecisionOutcomeVisitor) {
	switch o {
	case DecisionOutcomePermit:
		v.VisitPermit()
	case DecisionOutcomeDeny:
		v.VisitDeny()
	case DecisionOutcomeDefer:
		v.VisitDefer()
	case DecisionOutcomeEscalate:
		v.VisitEscalate()
	default:
		panic(fmt.Sprintf("DecisionOutcome.Accept: unrecognized DecisionOutcome %q", o))
	}
}

// ActorType represents what kind of decision-maker an actor is.
type ActorType string

const (
	ActorTypeHuman      ActorType = "Human"
	ActorTypeAI         ActorType = "AI"
	ActorTypeSystem     ActorType = "System"
	ActorTypeCommittee  ActorType = "Committee"
	ActorTypeRulesEngine ActorType = "RulesEngine"
)

var validActorTypes = map[ActorType]bool{
	ActorTypeHuman: true, ActorTypeAI: true, ActorTypeSystem: true,
	ActorTypeCommittee: true, ActorTypeRulesEngine: true,
}

// IsValid returns true if the actor type is a known type.
func (t ActorType) IsValid() bool { return validActorTypes[t] }

// EdgeDirection represents the direction of an edge from the social grammar.
type EdgeDirection string

const (
	EdgeDirectionCentripetal EdgeDirection = "Centripetal"
	EdgeDirectionCentrifugal EdgeDirection = "Centrifugal"
)

var validEdgeDirections = map[EdgeDirection]bool{
	EdgeDirectionCentripetal: true, EdgeDirectionCentrifugal: true,
}

// IsValid returns true if the edge direction is a known direction.
func (d EdgeDirection) IsValid() bool { return validEdgeDirections[d] }

// ExpectationStatus represents the state of an expectation.
type ExpectationStatus string

const (
	ExpectationStatusPending  ExpectationStatus = "Pending"
	ExpectationStatusMet      ExpectationStatus = "Met"
	ExpectationStatusViolated ExpectationStatus = "Violated"
	ExpectationStatusExpired  ExpectationStatus = "Expired"
)

var validExpectationStatuses = map[ExpectationStatus]bool{
	ExpectationStatusPending: true, ExpectationStatusMet: true,
	ExpectationStatusViolated: true, ExpectationStatusExpired: true,
}

// IsValid returns true if the expectation status is a known status.
func (s ExpectationStatus) IsValid() bool { return validExpectationStatuses[s] }

// SeverityLevel represents the severity of a violation.
type SeverityLevel string

const (
	SeverityLevelInfo     SeverityLevel = "Info"
	SeverityLevelWarning  SeverityLevel = "Warning"
	SeverityLevelSerious  SeverityLevel = "Serious"
	SeverityLevelCritical SeverityLevel = "Critical"
)

var validSeverityLevels = map[SeverityLevel]bool{
	SeverityLevelInfo: true, SeverityLevelWarning: true,
	SeverityLevelSerious: true, SeverityLevelCritical: true,
}

// IsValid returns true if the severity level is a known level.
func (s SeverityLevel) IsValid() bool { return validSeverityLevels[s] }

// MessageType represents EGIP message types.
type MessageType string

const (
	MessageTypeHello            MessageType = "Hello"
	MessageTypeMessage          MessageType = "Message"
	MessageTypeReceipt          MessageType = "Receipt"
	MessageTypeProof            MessageType = "Proof"
	MessageTypeTreaty           MessageType = "Treaty"
	MessageTypeAuthorityRequest MessageType = "AuthorityRequest"
	MessageTypeDiscover         MessageType = "Discover"
)

var validMessageTypes = map[MessageType]bool{
	MessageTypeHello: true, MessageTypeMessage: true, MessageTypeReceipt: true,
	MessageTypeProof: true, MessageTypeTreaty: true, MessageTypeAuthorityRequest: true,
	MessageTypeDiscover: true,
}

// IsValid returns true if the message type is a known type.
func (t MessageType) IsValid() bool { return validMessageTypes[t] }

// TreatyStatus represents the state of a bilateral treaty.
type TreatyStatus string

const (
	TreatyStatusProposed   TreatyStatus = "Proposed"
	TreatyStatusActive     TreatyStatus = "Active"
	TreatyStatusSuspended  TreatyStatus = "Suspended"
	TreatyStatusTerminated TreatyStatus = "Terminated"
)

var validTreatyStatuses = map[TreatyStatus]bool{
	TreatyStatusProposed: true, TreatyStatusActive: true,
	TreatyStatusSuspended: true, TreatyStatusTerminated: true,
}

// IsValid returns true if the treaty status is a known status.
func (s TreatyStatus) IsValid() bool { return validTreatyStatuses[s] }

// IntegrityViolationType represents the type of integrity violation.
type IntegrityViolationType string

const (
	IntegrityViolationChainBreak       IntegrityViolationType = "ChainBreak"
	IntegrityViolationHashMismatch     IntegrityViolationType = "HashMismatch"
	IntegrityViolationMissingCause     IntegrityViolationType = "MissingCause"
	IntegrityViolationSignatureInvalid IntegrityViolationType = "SignatureInvalid"
	IntegrityViolationOrphanEvent      IntegrityViolationType = "OrphanEvent"
)

var validIntegrityViolationTypes = map[IntegrityViolationType]bool{
	IntegrityViolationChainBreak: true, IntegrityViolationHashMismatch: true,
	IntegrityViolationMissingCause: true, IntegrityViolationSignatureInvalid: true,
	IntegrityViolationOrphanEvent: true,
}

// IsValid returns true if the integrity violation type is a known type.
func (t IntegrityViolationType) IsValid() bool { return validIntegrityViolationTypes[t] }

// InvariantName represents the 10 system invariants.
type InvariantName string

const (
	InvariantCausality   InvariantName = "Causality"
	InvariantIntegrity   InvariantName = "Integrity"
	InvariantObservable  InvariantName = "Observable"
	InvariantSelfEvolve  InvariantName = "SelfEvolve"
	InvariantDignity     InvariantName = "Dignity"
	InvariantTransparent InvariantName = "Transparent"
	InvariantConsent     InvariantName = "Consent"
	InvariantAuthority   InvariantName = "Authority"
	InvariantVerify      InvariantName = "Verify"
	InvariantRecord      InvariantName = "Record"
)

var validInvariantNames = map[InvariantName]bool{
	InvariantCausality: true, InvariantIntegrity: true, InvariantObservable: true,
	InvariantSelfEvolve: true, InvariantDignity: true, InvariantTransparent: true,
	InvariantConsent: true, InvariantAuthority: true, InvariantVerify: true,
	InvariantRecord: true,
}

// IsValid returns true if the invariant name is a known invariant.
func (n InvariantName) IsValid() bool { return validInvariantNames[n] }

// CGERRelationship represents a cross-graph event relationship.
type CGERRelationship string

const (
	CGERRelationshipCausedBy   CGERRelationship = "CausedBy"
	CGERRelationshipReferences CGERRelationship = "References"
	CGERRelationshipRespondsTo CGERRelationship = "RespondsTo"
)

var validCGERRelationships = map[CGERRelationship]bool{
	CGERRelationshipCausedBy: true, CGERRelationshipReferences: true,
	CGERRelationshipRespondsTo: true,
}

// IsValid returns true if the CGER relationship is a known relationship.
func (r CGERRelationship) IsValid() bool { return validCGERRelationships[r] }

// ReceiptStatus represents EGIP receipt statuses.
type ReceiptStatus string

const (
	ReceiptStatusDelivered ReceiptStatus = "Delivered"
	ReceiptStatusProcessed ReceiptStatus = "Processed"
	ReceiptStatusRejected  ReceiptStatus = "Rejected"
)

var validReceiptStatuses = map[ReceiptStatus]bool{
	ReceiptStatusDelivered: true, ReceiptStatusProcessed: true, ReceiptStatusRejected: true,
}

// IsValid returns true if the receipt status is a known status.
func (s ReceiptStatus) IsValid() bool { return validReceiptStatuses[s] }

// ProofType represents EGIP proof types.
type ProofType string

const (
	ProofTypeChainSegment   ProofType = "ChainSegment"
	ProofTypeEventExistence ProofType = "EventExistence"
	ProofTypeChainSummary   ProofType = "ChainSummary"
)

var validProofTypes = map[ProofType]bool{
	ProofTypeChainSegment: true, ProofTypeEventExistence: true, ProofTypeChainSummary: true,
}

// IsValid returns true if the proof type is a known type.
func (t ProofType) IsValid() bool { return validProofTypes[t] }

// TreatyAction represents EGIP treaty actions.
type TreatyAction string

const (
	TreatyActionPropose   TreatyAction = "Propose"
	TreatyActionAccept    TreatyAction = "Accept"
	TreatyActionModify    TreatyAction = "Modify"
	TreatyActionSuspend   TreatyAction = "Suspend"
	TreatyActionTerminate TreatyAction = "Terminate"
)

var validTreatyActions = map[TreatyAction]bool{
	TreatyActionPropose: true, TreatyActionAccept: true, TreatyActionModify: true,
	TreatyActionSuspend: true, TreatyActionTerminate: true,
}

// IsValid returns true if the treaty action is a known action.
func (a TreatyAction) IsValid() bool { return validTreatyActions[a] }

// ConditionOperator represents decision tree condition operators.
type ConditionOperator string

const (
	ConditionOperatorEquals      ConditionOperator = "Equals"
	ConditionOperatorGreaterThan ConditionOperator = "GreaterThan"
	ConditionOperatorLessThan    ConditionOperator = "LessThan"
	ConditionOperatorInRange     ConditionOperator = "InRange"
	ConditionOperatorMatches     ConditionOperator = "Matches"
	ConditionOperatorExists      ConditionOperator = "Exists"
	ConditionOperatorSemantic    ConditionOperator = "Semantic"
)

var validConditionOperators = map[ConditionOperator]bool{
	ConditionOperatorEquals: true, ConditionOperatorGreaterThan: true,
	ConditionOperatorLessThan: true, ConditionOperatorInRange: true,
	ConditionOperatorMatches: true, ConditionOperatorExists: true,
	ConditionOperatorSemantic: true,
}

// IsValid returns true if the condition operator is a known operator.
func (o ConditionOperator) IsValid() bool { return validConditionOperators[o] }
