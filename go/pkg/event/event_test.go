package event

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// --- Constants tests ---

func TestEdgeTypeIsValid(t *testing.T) {
	valid := []EdgeType{
		EdgeTypeTrust, EdgeTypeAuthority, EdgeTypeSubscription,
		EdgeTypeEndorsement, EdgeTypeDelegation, EdgeTypeCausation,
		EdgeTypeReference, EdgeTypeChannel, EdgeTypeAnnotation,
	}
	for _, et := range valid {
		if !et.IsValid() {
			t.Errorf("expected %q to be valid", et)
		}
	}
	if EdgeType("bogus").IsValid() {
		t.Error("expected bogus EdgeType to be invalid")
	}
}

func TestAuthorityLevelIsValid(t *testing.T) {
	valid := []AuthorityLevel{AuthorityLevelRequired, AuthorityLevelRecommended, AuthorityLevelNotification}
	for _, al := range valid {
		if !al.IsValid() {
			t.Errorf("expected %q to be valid", al)
		}
	}
	if AuthorityLevel("bogus").IsValid() {
		t.Error("expected bogus AuthorityLevel to be invalid")
	}
}

func TestDecisionOutcomeIsValid(t *testing.T) {
	valid := []DecisionOutcome{DecisionOutcomePermit, DecisionOutcomeDeny, DecisionOutcomeDefer, DecisionOutcomeEscalate}
	for _, do := range valid {
		if !do.IsValid() {
			t.Errorf("expected %q to be valid", do)
		}
	}
	if DecisionOutcome("bogus").IsValid() {
		t.Error("expected bogus DecisionOutcome to be invalid")
	}
}

func TestActorTypeIsValid(t *testing.T) {
	valid := []ActorType{ActorTypeHuman, ActorTypeAI, ActorTypeSystem, ActorTypeCommittee, ActorTypeRulesEngine}
	for _, at := range valid {
		if !at.IsValid() {
			t.Errorf("expected %q to be valid", at)
		}
	}
	if ActorType("bogus").IsValid() {
		t.Error("expected bogus ActorType to be invalid")
	}
}

func TestEdgeDirectionIsValid(t *testing.T) {
	valid := []EdgeDirection{EdgeDirectionCentripetal, EdgeDirectionCentrifugal}
	for _, ed := range valid {
		if !ed.IsValid() {
			t.Errorf("expected %q to be valid", ed)
		}
	}
	if EdgeDirection("bogus").IsValid() {
		t.Error("expected bogus EdgeDirection to be invalid")
	}
}

func TestExpectationStatusIsValid(t *testing.T) {
	valid := []ExpectationStatus{ExpectationStatusPending, ExpectationStatusMet, ExpectationStatusViolated, ExpectationStatusExpired}
	for _, es := range valid {
		if !es.IsValid() {
			t.Errorf("expected %q to be valid", es)
		}
	}
	if ExpectationStatus("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

func TestSeverityLevelIsValid(t *testing.T) {
	valid := []SeverityLevel{SeverityLevelInfo, SeverityLevelWarning, SeverityLevelSerious, SeverityLevelCritical}
	for _, sl := range valid {
		if !sl.IsValid() {
			t.Errorf("expected %q to be valid", sl)
		}
	}
	if SeverityLevel("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

func TestMessageTypeIsValid(t *testing.T) {
	valid := []MessageType{
		MessageTypeHello, MessageTypeMessage, MessageTypeReceipt,
		MessageTypeProof, MessageTypeTreaty, MessageTypeAuthorityRequest, MessageTypeDiscover,
	}
	for _, mt := range valid {
		if !mt.IsValid() {
			t.Errorf("expected %q to be valid", mt)
		}
	}
	if MessageType("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

func TestTreatyStatusIsValid(t *testing.T) {
	valid := []TreatyStatus{TreatyStatusProposed, TreatyStatusActive, TreatyStatusSuspended, TreatyStatusTerminated}
	for _, ts := range valid {
		if !ts.IsValid() {
			t.Errorf("expected %q to be valid", ts)
		}
	}
	if TreatyStatus("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

func TestIntegrityViolationTypeIsValid(t *testing.T) {
	valid := []IntegrityViolationType{
		IntegrityViolationChainBreak, IntegrityViolationHashMismatch,
		IntegrityViolationMissingCause, IntegrityViolationSignatureInvalid,
		IntegrityViolationOrphanEvent,
	}
	for _, iv := range valid {
		if !iv.IsValid() {
			t.Errorf("expected %q to be valid", iv)
		}
	}
	if IntegrityViolationType("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

func TestInvariantNameIsValid(t *testing.T) {
	valid := []InvariantName{
		InvariantCausality, InvariantIntegrity, InvariantObservable,
		InvariantSelfEvolve, InvariantDignity, InvariantTransparent,
		InvariantConsent, InvariantAuthority, InvariantVerify, InvariantRecord,
	}
	for _, in := range valid {
		if !in.IsValid() {
			t.Errorf("expected %q to be valid", in)
		}
	}
	if InvariantName("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

func TestCGERRelationshipIsValid(t *testing.T) {
	valid := []CGERRelationship{CGERRelationshipCausedBy, CGERRelationshipReferences, CGERRelationshipRespondsTo}
	for _, cr := range valid {
		if !cr.IsValid() {
			t.Errorf("expected %q to be valid", cr)
		}
	}
	if CGERRelationship("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

func TestReceiptStatusIsValid(t *testing.T) {
	valid := []ReceiptStatus{ReceiptStatusDelivered, ReceiptStatusProcessed, ReceiptStatusRejected}
	for _, rs := range valid {
		if !rs.IsValid() {
			t.Errorf("expected %q to be valid", rs)
		}
	}
	if ReceiptStatus("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

func TestProofTypeIsValid(t *testing.T) {
	valid := []ProofType{ProofTypeChainSegment, ProofTypeEventExistence, ProofTypeChainSummary}
	for _, pt := range valid {
		if !pt.IsValid() {
			t.Errorf("expected %q to be valid", pt)
		}
	}
	if ProofType("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

func TestTreatyActionIsValid(t *testing.T) {
	valid := []TreatyAction{TreatyActionPropose, TreatyActionAccept, TreatyActionModify, TreatyActionSuspend, TreatyActionTerminate}
	for _, ta := range valid {
		if !ta.IsValid() {
			t.Errorf("expected %q to be valid", ta)
		}
	}
	if TreatyAction("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

func TestConditionOperatorIsValid(t *testing.T) {
	valid := []ConditionOperator{
		ConditionOperatorEquals, ConditionOperatorGreaterThan, ConditionOperatorLessThan,
		ConditionOperatorInRange, ConditionOperatorMatches, ConditionOperatorExists,
		ConditionOperatorSemantic,
	}
	for _, co := range valid {
		if !co.IsValid() {
			t.Errorf("expected %q to be valid", co)
		}
	}
	if ConditionOperator("bogus").IsValid() {
		t.Error("expected bogus to be invalid")
	}
}

// --- Visitor tests ---

type edgeTypeCollector struct{ visited string }

func (c *edgeTypeCollector) VisitTrust()        { c.visited = "Trust" }
func (c *edgeTypeCollector) VisitAuthority()    { c.visited = "Authority" }
func (c *edgeTypeCollector) VisitSubscription() { c.visited = "Subscription" }
func (c *edgeTypeCollector) VisitEndorsement()  { c.visited = "Endorsement" }
func (c *edgeTypeCollector) VisitDelegation()   { c.visited = "Delegation" }
func (c *edgeTypeCollector) VisitCausation()    { c.visited = "Causation" }
func (c *edgeTypeCollector) VisitReference()    { c.visited = "Reference" }
func (c *edgeTypeCollector) VisitChannel()      { c.visited = "Channel" }
func (c *edgeTypeCollector) VisitAnnotation()       { c.visited = "Annotation" }
func (c *edgeTypeCollector) VisitAcknowledgement()  { c.visited = "Acknowledgement" }

func TestEdgeTypeVisitor(t *testing.T) {
	tests := []struct {
		et       EdgeType
		expected string
	}{
		{EdgeTypeTrust, "Trust"},
		{EdgeTypeAuthority, "Authority"},
		{EdgeTypeSubscription, "Subscription"},
		{EdgeTypeEndorsement, "Endorsement"},
		{EdgeTypeDelegation, "Delegation"},
		{EdgeTypeCausation, "Causation"},
		{EdgeTypeReference, "Reference"},
		{EdgeTypeChannel, "Channel"},
		{EdgeTypeAnnotation, "Annotation"},
		{EdgeTypeAcknowledgement, "Acknowledgement"},
	}
	for _, tt := range tests {
		c := &edgeTypeCollector{}
		tt.et.Accept(c)
		if c.visited != tt.expected {
			t.Errorf("EdgeType %q: visited %q, want %q", tt.et, c.visited, tt.expected)
		}
	}
}

type authorityLevelCollector struct{ visited string }

func (c *authorityLevelCollector) VisitRequired()     { c.visited = "Required" }
func (c *authorityLevelCollector) VisitRecommended()   { c.visited = "Recommended" }
func (c *authorityLevelCollector) VisitNotification()  { c.visited = "Notification" }

func TestAuthorityLevelVisitor(t *testing.T) {
	tests := []struct {
		al       AuthorityLevel
		expected string
	}{
		{AuthorityLevelRequired, "Required"},
		{AuthorityLevelRecommended, "Recommended"},
		{AuthorityLevelNotification, "Notification"},
	}
	for _, tt := range tests {
		c := &authorityLevelCollector{}
		tt.al.Accept(c)
		if c.visited != tt.expected {
			t.Errorf("AuthorityLevel %q: visited %q, want %q", tt.al, c.visited, tt.expected)
		}
	}
}

type decisionOutcomeCollector struct{ visited string }

func (c *decisionOutcomeCollector) VisitPermit()   { c.visited = "Permit" }
func (c *decisionOutcomeCollector) VisitDeny()     { c.visited = "Deny" }
func (c *decisionOutcomeCollector) VisitDefer()    { c.visited = "Defer" }
func (c *decisionOutcomeCollector) VisitEscalate() { c.visited = "Escalate" }

func TestDecisionOutcomeVisitor(t *testing.T) {
	tests := []struct {
		do       DecisionOutcome
		expected string
	}{
		{DecisionOutcomePermit, "Permit"},
		{DecisionOutcomeDeny, "Deny"},
		{DecisionOutcomeDefer, "Defer"},
		{DecisionOutcomeEscalate, "Escalate"},
	}
	for _, tt := range tests {
		c := &decisionOutcomeCollector{}
		tt.do.Accept(c)
		if c.visited != tt.expected {
			t.Errorf("DecisionOutcome %q: visited %q, want %q", tt.do, c.visited, tt.expected)
		}
	}
}

// --- Content tests ---

func TestEventContentTypeName(t *testing.T) {
	tests := []struct {
		content  EventContent
		expected string
	}{
		{TrustUpdatedContent{}, "trust.updated"},
		{TrustScoreContent{}, "trust.score"},
		{TrustDecayedContent{}, "trust.decayed"},
		{AuthorityRequestContent{}, "authority.requested"},
		{AuthorityResolvedContent{}, "authority.resolved"},
		{AuthorityDelegatedContent{}, "authority.delegated"},
		{AuthorityRevokedContent{}, "authority.revoked"},
		{AuthorityTimeoutContent{}, "authority.timeout"},
		{ActorRegisteredContent{}, "actor.registered"},
		{ActorSuspendedContent{}, "actor.suspended"},
		{ActorMemorialContent{}, "actor.memorial"},
		{EdgeCreatedContent{}, "edge.created"},
		{EdgeSupersededContent{}, "edge.superseded"},
		{ViolationDetectedContent{}, "violation.detected"},
		{ChainVerifiedContent{}, "chain.verified"},
		{ChainBrokenContent{}, "chain.broken"},
		{BootstrapContent{}, "system.bootstrapped"},
		{ClockTickContent{}, "clock.tick"},
		{HealthReportContent{}, "health.report"},
		{BranchProposedContent{}, "decision.branch.proposed"},
		{BranchInsertedContent{}, "decision.branch.inserted"},
		{CostReportContent{}, "decision.cost.report"},
		{EGIPHelloSentContent{}, "egip.hello.sent"},
		{EGIPHelloReceivedContent{}, "egip.hello.received"},
		{EGIPMessageSentContent{}, "egip.message.sent"},
		{EGIPMessageReceivedContent{}, "egip.message.received"},
		{EGIPReceiptSentContent{}, "egip.receipt.sent"},
		{EGIPReceiptReceivedContent{}, "egip.receipt.received"},
		{EGIPProofRequestedContent{}, "egip.proof.requested"},
		{EGIPProofReceivedContent{}, "egip.proof.received"},
		{EGIPTreatyProposedContent{}, "egip.treaty.proposed"},
		{EGIPTreatyActiveContent{}, "egip.treaty.active"},
		{EGIPTrustUpdatedContent{}, "egip.trust.updated"},
	}
	for _, tt := range tests {
		if got := tt.content.EventTypeName(); got != tt.expected {
			t.Errorf("EventTypeName() = %q, want %q", got, tt.expected)
		}
	}
}

// contentVisitorCollector implements EventContentVisitor to track which method was called.
type contentVisitorCollector struct{ visited string }

func (c *contentVisitorCollector) VisitTrustUpdated(TrustUpdatedContent)             { c.visited = "trust.updated" }
func (c *contentVisitorCollector) VisitTrustScore(TrustScoreContent)                 { c.visited = "trust.score" }
func (c *contentVisitorCollector) VisitTrustDecayed(TrustDecayedContent)             { c.visited = "trust.decayed" }
func (c *contentVisitorCollector) VisitAuthorityRequested(AuthorityRequestContent)    { c.visited = "authority.requested" }
func (c *contentVisitorCollector) VisitAuthorityResolved(AuthorityResolvedContent)    { c.visited = "authority.resolved" }
func (c *contentVisitorCollector) VisitAuthorityDelegated(AuthorityDelegatedContent)  { c.visited = "authority.delegated" }
func (c *contentVisitorCollector) VisitAuthorityRevoked(AuthorityRevokedContent)      { c.visited = "authority.revoked" }
func (c *contentVisitorCollector) VisitAuthorityTimeout(AuthorityTimeoutContent)      { c.visited = "authority.timeout" }
func (c *contentVisitorCollector) VisitActorRegistered(ActorRegisteredContent)        { c.visited = "actor.registered" }
func (c *contentVisitorCollector) VisitActorSuspended(ActorSuspendedContent)          { c.visited = "actor.suspended" }
func (c *contentVisitorCollector) VisitActorMemorial(ActorMemorialContent)            { c.visited = "actor.memorial" }
func (c *contentVisitorCollector) VisitEdgeCreated(EdgeCreatedContent)                { c.visited = "edge.created" }
func (c *contentVisitorCollector) VisitEdgeSuperseded(EdgeSupersededContent)          { c.visited = "edge.superseded" }
func (c *contentVisitorCollector) VisitViolationDetected(ViolationDetectedContent)    { c.visited = "violation.detected" }
func (c *contentVisitorCollector) VisitChainVerified(ChainVerifiedContent)            { c.visited = "chain.verified" }
func (c *contentVisitorCollector) VisitChainBroken(ChainBrokenContent)               { c.visited = "chain.broken" }
func (c *contentVisitorCollector) VisitBootstrap(BootstrapContent)                    { c.visited = "system.bootstrapped" }
func (c *contentVisitorCollector) VisitClockTick(ClockTickContent)                    { c.visited = "clock.tick" }
func (c *contentVisitorCollector) VisitHealthReport(HealthReportContent)              { c.visited = "health.report" }
func (c *contentVisitorCollector) VisitBranchProposed(BranchProposedContent)          { c.visited = "decision.branch.proposed" }
func (c *contentVisitorCollector) VisitBranchInserted(BranchInsertedContent)          { c.visited = "decision.branch.inserted" }
func (c *contentVisitorCollector) VisitCostReport(CostReportContent)                 { c.visited = "decision.cost.report" }
func (c *contentVisitorCollector) VisitEGIPHelloSent(EGIPHelloSentContent)            { c.visited = "egip.hello.sent" }
func (c *contentVisitorCollector) VisitEGIPHelloReceived(EGIPHelloReceivedContent)    { c.visited = "egip.hello.received" }
func (c *contentVisitorCollector) VisitEGIPMessageSent(EGIPMessageSentContent)        { c.visited = "egip.message.sent" }
func (c *contentVisitorCollector) VisitEGIPMessageReceived(EGIPMessageReceivedContent) { c.visited = "egip.message.received" }
func (c *contentVisitorCollector) VisitEGIPReceiptSent(EGIPReceiptSentContent)        { c.visited = "egip.receipt.sent" }
func (c *contentVisitorCollector) VisitEGIPReceiptReceived(EGIPReceiptReceivedContent) { c.visited = "egip.receipt.received" }
func (c *contentVisitorCollector) VisitEGIPProofRequested(EGIPProofRequestedContent)  { c.visited = "egip.proof.requested" }
func (c *contentVisitorCollector) VisitEGIPProofReceived(EGIPProofReceivedContent)    { c.visited = "egip.proof.received" }
func (c *contentVisitorCollector) VisitEGIPTreatyProposed(EGIPTreatyProposedContent)  { c.visited = "egip.treaty.proposed" }
func (c *contentVisitorCollector) VisitEGIPTreatyActive(EGIPTreatyActiveContent)      { c.visited = "egip.treaty.active" }
func (c *contentVisitorCollector) VisitEGIPTrustUpdated(EGIPTrustUpdatedContent)      { c.visited = "egip.trust.updated" }
func (c *contentVisitorCollector) VisitGrammarEmit(GrammarEmitContent)                { c.visited = "grammar.emit" }
func (c *contentVisitorCollector) VisitGrammarRespond(GrammarRespondContent)          { c.visited = "grammar.respond" }
func (c *contentVisitorCollector) VisitGrammarDerive(GrammarDeriveContent)            { c.visited = "grammar.derive" }
func (c *contentVisitorCollector) VisitGrammarExtend(GrammarExtendContent)            { c.visited = "grammar.extend" }
func (c *contentVisitorCollector) VisitGrammarRetract(GrammarRetractContent)          { c.visited = "grammar.retract" }
func (c *contentVisitorCollector) VisitGrammarAnnotate(GrammarAnnotateContent)        { c.visited = "grammar.annotate" }
func (c *contentVisitorCollector) VisitGrammarMerge(GrammarMergeContent)              { c.visited = "grammar.merge" }
func (c *contentVisitorCollector) VisitGrammarConsent(GrammarConsentContent)          { c.visited = "grammar.consent" }

func TestEventContentVisitorDispatch(t *testing.T) {
	contents := []EventContent{
		TrustUpdatedContent{}, TrustScoreContent{}, TrustDecayedContent{},
		AuthorityRequestContent{}, AuthorityResolvedContent{}, AuthorityDelegatedContent{},
		AuthorityRevokedContent{}, AuthorityTimeoutContent{},
		ActorRegisteredContent{}, ActorSuspendedContent{}, ActorMemorialContent{},
		EdgeCreatedContent{}, EdgeSupersededContent{},
		ViolationDetectedContent{}, ChainVerifiedContent{}, ChainBrokenContent{},
		BootstrapContent{}, ClockTickContent{}, HealthReportContent{},
		BranchProposedContent{}, BranchInsertedContent{}, CostReportContent{},
		GrammarEmitContent{}, GrammarRespondContent{}, GrammarDeriveContent{},
		GrammarExtendContent{}, GrammarRetractContent{}, GrammarAnnotateContent{},
		GrammarMergeContent{}, GrammarConsentContent{},
		EGIPHelloSentContent{}, EGIPHelloReceivedContent{},
		EGIPMessageSentContent{}, EGIPMessageReceivedContent{},
		EGIPReceiptSentContent{}, EGIPReceiptReceivedContent{},
		EGIPProofRequestedContent{}, EGIPProofReceivedContent{},
		EGIPTreatyProposedContent{}, EGIPTreatyActiveContent{},
		EGIPTrustUpdatedContent{},
	}
	for _, content := range contents {
		c := &contentVisitorCollector{}
		content.Accept(c)
		if c.visited != content.EventTypeName() {
			t.Errorf("visitor dispatched to %q for content type %q", c.visited, content.EventTypeName())
		}
	}
}

// --- Edge metadata visitor tests ---

type edgeMetadataCollector struct{ visited EdgeType }

func (c *edgeMetadataCollector) VisitTrust(TrustEdgeMetadata)               { c.visited = EdgeTypeTrust }
func (c *edgeMetadataCollector) VisitAuthority(AuthorityEdgeMetadata)       { c.visited = EdgeTypeAuthority }
func (c *edgeMetadataCollector) VisitSubscription(SubscriptionEdgeMetadata) { c.visited = EdgeTypeSubscription }
func (c *edgeMetadataCollector) VisitEndorsement(EndorsementEdgeMetadata)   { c.visited = EdgeTypeEndorsement }
func (c *edgeMetadataCollector) VisitDelegation(DelegationEdgeMetadata)     { c.visited = EdgeTypeDelegation }
func (c *edgeMetadataCollector) VisitCausation(CausationEdgeMetadata)       { c.visited = EdgeTypeCausation }
func (c *edgeMetadataCollector) VisitReference(ReferenceEdgeMetadata)       { c.visited = EdgeTypeReference }
func (c *edgeMetadataCollector) VisitChannel(ChannelEdgeMetadata)           { c.visited = EdgeTypeChannel }
func (c *edgeMetadataCollector) VisitAnnotation(AnnotationEdgeMetadata)     { c.visited = EdgeTypeAnnotation }

func TestEdgeMetadataVisitorDispatch(t *testing.T) {
	metadatas := []EdgeMetadata{
		TrustEdgeMetadata{}, AuthorityEdgeMetadata{}, SubscriptionEdgeMetadata{},
		EndorsementEdgeMetadata{}, DelegationEdgeMetadata{}, CausationEdgeMetadata{},
		ReferenceEdgeMetadata{}, ChannelEdgeMetadata{}, AnnotationEdgeMetadata{},
	}
	for _, md := range metadatas {
		c := &edgeMetadataCollector{}
		md.Accept(c)
		if c.visited != md.EdgeTypeName() {
			t.Errorf("visitor dispatched to %q for metadata type %q", c.visited, md.EdgeTypeName())
		}
	}
}

// --- Edge tests ---

func TestNewEdge(t *testing.T) {
	edgeID := types.MustEdgeID("019462a0-0000-7000-8000-000000000001")
	fromID := types.MustActorID("actor_00000000000000000000000000000001")
	toID := types.MustActorID("actor_00000000000000000000000000000002")
	weight := types.MustWeight(0.5)
	scope := types.Some(types.MustDomainScope("code_review"))
	now := types.Now()

	edge, err := NewEdge(edgeID, fromID, toID, EdgeTypeTrust, weight, EdgeDirectionCentripetal,
		scope, TrustEdgeMetadata{Domain: types.MustDomainScope("code_review")}, now, types.None[types.Timestamp]())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if edge.ID() != edgeID {
		t.Errorf("ID = %v, want %v", edge.ID(), edgeID)
	}
	if edge.From() != fromID {
		t.Errorf("From = %v, want %v", edge.From(), fromID)
	}
	if edge.To() != toID {
		t.Errorf("To = %v, want %v", edge.To(), toID)
	}
	if edge.Type() != EdgeTypeTrust {
		t.Errorf("Type = %v, want %v", edge.Type(), EdgeTypeTrust)
	}
	if edge.Direction() != EdgeDirectionCentripetal {
		t.Errorf("Direction = %v, want %v", edge.Direction(), EdgeDirectionCentripetal)
	}
}

func TestNewEdgeInvalidEdgeType(t *testing.T) {
	edgeID := types.MustEdgeID("019462a0-0000-7000-8000-000000000001")
	fromID := types.MustActorID("actor_00000000000000000000000000000001")
	toID := types.MustActorID("actor_00000000000000000000000000000002")

	_, err := NewEdge(edgeID, fromID, toID, EdgeType("bogus"), types.MustWeight(0), EdgeDirectionCentripetal,
		types.None[types.DomainScope](), nil, types.Now(), types.None[types.Timestamp]())
	if err == nil {
		t.Fatal("expected error for invalid EdgeType")
	}
}

func TestNewEdgeInvalidDirection(t *testing.T) {
	edgeID := types.MustEdgeID("019462a0-0000-7000-8000-000000000001")
	fromID := types.MustActorID("actor_00000000000000000000000000000001")
	toID := types.MustActorID("actor_00000000000000000000000000000002")

	_, err := NewEdge(edgeID, fromID, toID, EdgeTypeTrust, types.MustWeight(0), EdgeDirection("bogus"),
		types.None[types.DomainScope](), nil, types.Now(), types.None[types.Timestamp]())
	if err == nil {
		t.Fatal("expected error for invalid EdgeDirection")
	}
}

// --- Event tests ---

func TestNewEvent(t *testing.T) {
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	et := EventTypeTrustUpdated
	source := types.MustActorID("actor_00000000000000000000000000000001")
	conv := types.MustConversationID("conv_00000000000000000000000000000001")
	hash := types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	prevHash := types.ZeroHash()
	sig := types.MustSignature(make([]byte, 64))
	now := types.Now()
	ev := NewEvent(1, id, et, now, source, TrustUpdatedContent{}, []types.EventID{id}, conv, hash, prevHash, sig)
	if ev.Version() != 1 {
		t.Errorf("Version = %d, want 1", ev.Version())
	}
	if ev.ID() != id {
		t.Errorf("ID = %v, want %v", ev.ID(), id)
	}
	if ev.Type() != et {
		t.Errorf("Type = %v, want %v", ev.Type(), et)
	}
	if ev.Source() != source {
		t.Errorf("Source = %v, want %v", ev.Source(), source)
	}
	if ev.ConversationID() != conv {
		t.Errorf("ConversationID = %v, want %v", ev.ConversationID(), conv)
	}
	if ev.Hash() != hash {
		t.Errorf("Hash = %v, want %v", ev.Hash(), hash)
	}
	if ev.IsBootstrap() {
		t.Error("expected non-bootstrap event")
	}
}

func TestNewBootstrapEvent(t *testing.T) {
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	et := EventTypeSystemBootstrapped
	source := types.MustActorID("actor_00000000000000000000000000000001")
	conv := types.MustConversationID("conv_00000000000000000000000000000001")
	hash := types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	sig := types.MustSignature(make([]byte, 64))
	now := types.Now()

	ev := NewBootstrapEvent(1, id, et, now, source, BootstrapContent{
		ActorID: source, ChainGenesis: types.ZeroHash(), Timestamp: now,
	}, conv, hash, sig)
	if !ev.IsBootstrap() {
		t.Error("expected bootstrap event")
	}
	if ev.PrevHash() != types.ZeroHash() {
		t.Errorf("PrevHash should be zero hash for bootstrap")
	}
	if len(ev.Causes()) != 0 {
		t.Errorf("Causes should be empty for bootstrap, got %d", len(ev.Causes()))
	}
}

// --- Canonical form tests (conformance vectors) ---

func TestCanonicalFormBootstrap(t *testing.T) {
	// From canonical-vectors.json: bootstrap_event
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	et := EventTypeSystemBootstrapped
	source := types.MustActorID("actor_00000000000000000000000000000001")
	conv := types.MustConversationID("conv_00000000000000000000000000000001")
	ts := types.NewTimestamp(time.Unix(0, 1700000000000000000))

	// For canonical form, we need JSON-serializable content matching the vector.
	// Use rawJSONContent to produce exact JSON.
	content := rawJSONContent{
		typeName: "system.bootstrapped",
		data: map[string]any{
			"actorID":      "actor_00000000000000000000000000000001",
			"chainGenesis": "0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp":    "2023-11-14T22:13:20Z",
		},
	}

	ev := NewBootstrapEvent(1, id, et, ts, source, BootstrapContent{}, conv,
		types.MustHash("0000000000000000000000000000000000000000000000000000000000000000"),
		types.MustSignature(make([]byte, 64)))
	// Override content for canonical form test
	ev.content = content

	canonical := CanonicalForm(ev)
	expected := `1|||019462a0-0000-7000-8000-000000000001|system.bootstrapped|actor_00000000000000000000000000000001|conv_00000000000000000000000000000001|1700000000000000000|{"actorID":"actor_00000000000000000000000000000001","chainGenesis":"0000000000000000000000000000000000000000000000000000000000000000","timestamp":"2023-11-14T22:13:20Z"}`

	if canonical != expected {
		t.Errorf("canonical form mismatch.\ngot:  %s\nwant: %s", canonical, expected)
	}
}

func TestCanonicalFormTrustUpdated(t *testing.T) {
	// From canonical-vectors.json: trust_updated_event
	id := types.MustEventID("019462a0-0000-7000-8000-000000000002")
	et := EventTypeTrustUpdated
	source := types.MustActorID("actor_00000000000000000000000000000001")
	conv := types.MustConversationID("conv_00000000000000000000000000000001")
	ts := types.NewTimestamp(time.Unix(0, 1700000001000000000))
	prevHash := types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	content := rawJSONContent{
		typeName: "trust.updated",
		data: map[string]any{
			"actor":    "actor_00000000000000000000000000000002",
			"cause":    "019462a0-0000-7000-8000-000000000001",
			"current":  0.85,
			"domain":   "code_review",
			"previous": 0.8,
		},
	}

	ev := NewEvent(1, id, et, ts, source, content,
		[]types.EventID{types.MustEventID("019462a0-0000-7000-8000-000000000001")}, conv,
		types.MustHash("0000000000000000000000000000000000000000000000000000000000000000"), prevHash,
		types.MustSignature(make([]byte, 64)))

	canonical := CanonicalForm(ev)
	expected := `1|a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2|019462a0-0000-7000-8000-000000000001|019462a0-0000-7000-8000-000000000002|trust.updated|actor_00000000000000000000000000000001|conv_00000000000000000000000000000001|1700000001000000000|{"actor":"actor_00000000000000000000000000000002","cause":"019462a0-0000-7000-8000-000000000001","current":0.85,"domain":"code_review","previous":0.8}`

	if canonical != expected {
		t.Errorf("canonical form mismatch.\ngot:  %s\nwant: %s", canonical, expected)
	}
}

func TestCanonicalFormKeyOrdering(t *testing.T) {
	// From canonical-vectors.json: content_json_key_ordering
	id := types.MustEventID("019462a0-0000-7000-8000-000000000003")
	et := EventTypeEdgeCreated
	source := types.MustActorID("actor_00000000000000000000000000000001")
	conv := types.MustConversationID("conv_00000000000000000000000000000001")
	ts := types.NewTimestamp(time.Unix(0, 1700000002000000000))
	prevHash := types.MustHash("b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3")
	// Keys deliberately out of order — canonical form must sort them
	content := rawJSONContent{
		typeName: "edge.created",
		data: map[string]any{
			"weight":    0.5,
			"from":      "actor_00000000000000000000000000000001",
			"to":        "actor_00000000000000000000000000000002",
			"edgeType":  "Trust",
			"direction": "Centripetal",
		},
	}

	ev := NewEvent(1, id, et, ts, source, content,
		[]types.EventID{types.MustEventID("019462a0-0000-7000-8000-000000000001")}, conv,
		types.MustHash("0000000000000000000000000000000000000000000000000000000000000000"), prevHash,
		types.MustSignature(make([]byte, 64)))

	canonical := CanonicalForm(ev)
	expected := `1|b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3|019462a0-0000-7000-8000-000000000001|019462a0-0000-7000-8000-000000000003|edge.created|actor_00000000000000000000000000000001|conv_00000000000000000000000000000001|1700000002000000000|{"direction":"Centripetal","edgeType":"Trust","from":"actor_00000000000000000000000000000001","to":"actor_00000000000000000000000000000002","weight":0.5}`

	if canonical != expected {
		t.Errorf("canonical form mismatch.\ngot:  %s\nwant: %s", canonical, expected)
	}
}

func TestCanonicalNumberFormatting(t *testing.T) {
	// From canonical-vectors.json: content_json_number_formatting
	tests := []struct {
		input    float64
		expected string
	}{
		{1.0, "1"},
		{0.5, "0.5"},
		{0.10, "0.1"},
		{100.0, "100"},
		{0.001, "0.001"},
	}
	for _, tt := range tests {
		got := formatCanonicalNumber(tt.input)
		if got != tt.expected {
			t.Errorf("formatCanonicalNumber(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCanonicalNullOmission(t *testing.T) {
	// From canonical-vectors.json: content_json_null_omission
	m := map[string]any{
		"actor":  "actor_00000000000000000000000000000001",
		"scope":  nil,
		"reason": nil,
	}
	got := sortedJSON(m)
	expected := `{"actor":"actor_00000000000000000000000000000001"}`
	if got != expected {
		t.Errorf("sortedJSON with nulls = %q, want %q", got, expected)
	}
}

// --- Hash computation tests ---

func TestComputeHash(t *testing.T) {
	canonical := "1||019462a0-0000-7000-8000-000000000001|system.bootstrapped|actor_00000000000000000000000000000001|conv_00000000000000000000000000000001|1700000000000000000|{}"
	hash, err := ComputeHash(canonical)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify against manual sha256
	h := sha256.Sum256([]byte(canonical))
	expected := fmt.Sprintf("%x", h)
	if hash.Value() != expected {
		t.Errorf("ComputeHash = %q, want %q", hash.Value(), expected)
	}
}

func TestHashChainLinking(t *testing.T) {
	// Build a mini chain: event 0 → event 1 → event 2
	// Each event's hash becomes the next event's prev_hash
	source := types.MustActorID("actor_00000000000000000000000000000001")
	conv := types.MustConversationID("conv_00000000000000000000000000000001")
	et := EventTypeSystemBootstrapped
	sig := types.MustSignature(make([]byte, 64))

	// Event 0 (bootstrap)
	ev0 := NewBootstrapEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000001"),
		et, types.NewTimestamp(time.Unix(0, 1700000000000000000)), source,
		BootstrapContent{}, conv,
		types.MustHash("0000000000000000000000000000000000000000000000000000000000000000"), sig)
	ev0.content = rawJSONContent{typeName: "system.bootstrapped", data: map[string]any{"v": 0}}
	canonical0 := CanonicalForm(ev0)
	hash0, err := ComputeHash(canonical0)
	if err != nil {
		t.Fatalf("hash0: %v", err)
	}

	// Event 1 (prev_hash = hash0)
	ev1 := NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000002"),
		EventTypeTrustUpdated,
		types.NewTimestamp(time.Unix(0, 1700000001000000000)), source,
		rawJSONContent{typeName: "trust.updated", data: map[string]any{"v": 1}},
		[]types.EventID{ev0.ID()}, conv,
		types.MustHash("0000000000000000000000000000000000000000000000000000000000000000"), hash0, sig)
	canonical1 := CanonicalForm(ev1)
	hash1, err := ComputeHash(canonical1)
	if err != nil {
		t.Fatalf("hash1: %v", err)
	}

	// Event 2 (prev_hash = hash1)
	ev2 := NewEvent(1,
		types.MustEventID("019462a0-0000-7000-8000-000000000003"),
		EventTypeTrustUpdated,
		types.NewTimestamp(time.Unix(0, 1700000002000000000)), source,
		rawJSONContent{typeName: "trust.updated", data: map[string]any{"v": 2}},
		[]types.EventID{ev1.ID()}, conv,
		types.MustHash("0000000000000000000000000000000000000000000000000000000000000000"), hash1, sig)
	canonical2 := CanonicalForm(ev2)
	hash2, err := ComputeHash(canonical2)
	if err != nil {
		t.Fatalf("hash2: %v", err)
	}

	// Verify chain integrity: each hash is unique and chains correctly
	if hash0 == hash1 || hash1 == hash2 || hash0 == hash2 {
		t.Error("all hashes should be unique")
	}
	if ev1.PrevHash() != hash0 {
		t.Errorf("ev1.PrevHash = %v, want %v", ev1.PrevHash(), hash0)
	}
	if ev2.PrevHash() != hash1 {
		t.Errorf("ev2.PrevHash = %v, want %v", ev2.PrevHash(), hash1)
	}
	_ = hash2
}

// --- Registry tests ---

func TestDefaultRegistry(t *testing.T) {
	r := DefaultRegistry()
	allTypes := r.AllTypes()
	if len(allTypes) != 41 {
		t.Errorf("expected 41 registered types, got %d", len(allTypes))
	}
	if !r.IsRegistered(EventTypeTrustUpdated) {
		t.Error("trust.updated should be registered")
	}
	if r.IsRegistered(types.MustEventType("bogus.type99")) {
		t.Error("bogus.type should not be registered")
	}
}

func TestRegistryValidate(t *testing.T) {
	r := DefaultRegistry()
	et := EventTypeTrustUpdated

	// Matching content type
	err := r.Validate(et, TrustUpdatedContent{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Mismatched content type
	err = r.Validate(et, BootstrapContent{})
	if err == nil {
		t.Error("expected error for mismatched content type")
	}

	// Unregistered type
	err = r.Validate(types.MustEventType("bogus.type99"), TrustUpdatedContent{})
	if err == nil {
		t.Error("expected error for unregistered type")
	}
}

// --- Decision tests ---

func TestNewDecision(t *testing.T) {
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	eventID := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	score := types.MustScore(0.9)
	authChain, _ := types.NewNonEmpty([]AuthorityLink{
		{Actor: actorID, Level: AuthorityLevelRequired, Weight: score},
	})
	evidence, _ := types.NewNonEmpty([]types.EventID{eventID})
	receipt := NewReceipt(
		types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"),
		types.Now(), actorID, types.MustSignature(make([]byte, 64)),
		types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"),
		eventID,
	)

	d, err := NewDecision(DecisionOutcomePermit, score, authChain, nil, evidence, receipt, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Outcome() != DecisionOutcomePermit {
		t.Errorf("Outcome = %v, want Permit", d.Outcome())
	}
	if d.NeedsHuman() {
		t.Error("NeedsHuman should be false")
	}
}

func TestNewDecisionInvalidOutcome(t *testing.T) {
	score := types.MustScore(0.5)
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	eventID := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	authChain, _ := types.NewNonEmpty([]AuthorityLink{
		{Actor: actorID, Level: AuthorityLevelRequired, Weight: score},
	})
	evidence, _ := types.NewNonEmpty([]types.EventID{eventID})

	_, err := NewDecision(DecisionOutcome("bogus"), score, authChain, nil, evidence, Receipt{}, false)
	if err == nil {
		t.Fatal("expected error for invalid outcome")
	}
}

func TestDecisionTrustWeightsDefensiveCopy(t *testing.T) {
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	eventID := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	score := types.MustScore(0.5)
	authChain, _ := types.NewNonEmpty([]AuthorityLink{
		{Actor: actorID, Level: AuthorityLevelRequired, Weight: score},
	})
	evidence, _ := types.NewNonEmpty([]types.EventID{eventID})

	original := []TrustWeight{{Actor: actorID, Score: score, Domain: types.MustDomainScope("test")}}
	d, _ := NewDecision(DecisionOutcomePermit, score, authChain, original, evidence, Receipt{}, false)

	// Mutate original — should not affect decision
	original[0].Score = types.MustScore(0.9)
	if d.TrustWeights()[0].Score == types.MustScore(0.9) {
		t.Error("TrustWeights should be defensively copied")
	}
}

// --- TrustMetrics tests ---

func TestNewTrustMetrics(t *testing.T) {
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	overall := types.MustScore(0.8)
	confidence := types.MustScore(0.9)
	trend := types.MustWeight(0.1)
	decayRate := types.MustScore(0.05)
	byDomain := map[types.DomainScope]types.Score{
		types.MustDomainScope("code_review"): types.MustScore(0.9),
	}
	evidence := []types.EventID{types.MustEventID("019462a0-0000-7000-8000-000000000001")}
	now := types.Now()

	m := NewTrustMetrics(actorID, overall, byDomain, confidence, trend, evidence, now, decayRate)
	if m.Actor() != actorID {
		t.Errorf("Actor = %v, want %v", m.Actor(), actorID)
	}
	if m.Overall() != overall {
		t.Errorf("Overall = %v, want %v", m.Overall(), overall)
	}

	// Defensive copy
	byDomain[types.MustDomainScope("code_review")] = types.MustScore(0.1)
	if m.ByDomain()[types.MustDomainScope("code_review")] == types.MustScore(0.1) {
		t.Error("ByDomain should be defensively copied")
	}
}

// --- Expectation tests ---

func TestNewExpectation(t *testing.T) {
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	trigger := types.MustEventID("019462a0-0000-7000-8000-000000000002")
	now := types.Now()

	exp, err := NewExpectation(id, trigger, "test", now, SeverityLevelWarning, ExpectationStatusPending)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.Severity() != SeverityLevelWarning {
		t.Errorf("Severity = %v, want Warning", exp.Severity())
	}
}

func TestNewExpectationInvalidSeverity(t *testing.T) {
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	_, err := NewExpectation(id, id, "test", types.Now(), SeverityLevel("bogus"), ExpectationStatusPending)
	if err == nil {
		t.Fatal("expected error for invalid severity")
	}
}

func TestNewExpectationInvalidStatus(t *testing.T) {
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	_, err := NewExpectation(id, id, "test", types.Now(), SeverityLevelInfo, ExpectationStatus("bogus"))
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

// --- ViolationRecord tests ---

func TestNewViolationRecord(t *testing.T) {
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	evidence, _ := types.NewNonEmpty([]types.EventID{id})

	vr, err := NewViolationRecord(id, id, SeverityLevelCritical, actorID, "test violation", evidence)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vr.Severity() != SeverityLevelCritical {
		t.Errorf("Severity = %v, want Critical", vr.Severity())
	}
}

func TestNewViolationRecordInvalidSeverity(t *testing.T) {
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	evidence, _ := types.NewNonEmpty([]types.EventID{id})

	_, err := NewViolationRecord(id, id, SeverityLevel("bogus"), actorID, "test", evidence)
	if err == nil {
		t.Fatal("expected error for invalid severity")
	}
}

// --- Receipt tests ---

func TestReceipt(t *testing.T) {
	hash := types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	eventID := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	sig := types.MustSignature(make([]byte, 64))
	now := types.Now()

	r := NewReceipt(hash, now, actorID, sig, hash, eventID)
	if r.Hash() != hash {
		t.Errorf("Hash = %v, want %v", r.Hash(), hash)
	}
	if r.SignedBy() != actorID {
		t.Errorf("SignedBy = %v, want %v", r.SignedBy(), actorID)
	}
	if r.ChainPos() != eventID {
		t.Errorf("ChainPos = %v, want %v", r.ChainPos(), eventID)
	}
}

// --- Getter coverage tests ---

func TestEventAllGetters(t *testing.T) {
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	et := EventTypeTrustUpdated
	source := types.MustActorID("actor_00000000000000000000000000000001")
	conv := types.MustConversationID("conv_00000000000000000000000000000001")
	hash := types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	sig := types.MustSignature(make([]byte, 64))
	now := types.Now()
	content := TrustUpdatedContent{Actor: source}

	ev := NewEvent(1, id, et, now, source, content, []types.EventID{id}, conv, hash, hash, sig)

	// Cover all getters
	_ = ev.Timestamp()
	_ = ev.Content()
	_ = ev.Signature()
}

func TestEdgeAllGetters(t *testing.T) {
	edgeID := types.MustEdgeID("019462a0-0000-7000-8000-000000000001")
	fromID := types.MustActorID("actor_00000000000000000000000000000001")
	toID := types.MustActorID("actor_00000000000000000000000000000002")
	weight := types.MustWeight(0.5)
	scope := types.Some(types.MustDomainScope("code_review"))
	now := types.Now()
	expires := types.Some(types.NewTimestamp(now.Value().Add(time.Hour)))

	edge, _ := NewEdge(edgeID, fromID, toID, EdgeTypeTrust, weight, EdgeDirectionCentripetal,
		scope, TrustEdgeMetadata{}, now, expires)

	if edge.Weight() != weight {
		t.Errorf("Weight mismatch")
	}
	if !edge.Scope().IsSome() {
		t.Error("Scope should be Some")
	}
	if edge.Metadata() == nil {
		t.Error("Metadata should not be nil")
	}
	if edge.CreatedAt() != now {
		t.Error("CreatedAt mismatch")
	}
	if !edge.ExpiresAt().IsSome() {
		t.Error("ExpiresAt should be Some")
	}
}

func TestDecisionAllGetters(t *testing.T) {
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	eventID := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	score := types.MustScore(0.9)
	authChain, _ := types.NewNonEmpty([]AuthorityLink{
		{Actor: actorID, Level: AuthorityLevelRequired, Weight: score},
	})
	evidence, _ := types.NewNonEmpty([]types.EventID{eventID})
	receipt := NewReceipt(
		types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"),
		types.Now(), actorID, types.MustSignature(make([]byte, 64)),
		types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"),
		eventID,
	)

	d, _ := NewDecision(DecisionOutcomePermit, score, authChain, nil, evidence, receipt, false)

	if d.Confidence() != score {
		t.Error("Confidence mismatch")
	}
	if d.AuthorityChain().Len() != 1 {
		t.Error("AuthorityChain should have 1 link")
	}
	if d.Evidence().Len() != 1 {
		t.Error("Evidence should have 1 item")
	}
	if d.Receipt().Hash() != receipt.Hash() {
		t.Error("Receipt mismatch")
	}
}

func TestReceiptAllGetters(t *testing.T) {
	hash := types.MustHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	eventID := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	sig := types.MustSignature(make([]byte, 64))
	now := types.Now()

	r := NewReceipt(hash, now, actorID, sig, hash, eventID)
	if r.Timestamp() != now {
		t.Error("Timestamp mismatch")
	}
	_ = r.Signature() // Signature contains []byte, not comparable
	if r.InputHash() != hash {
		t.Error("InputHash mismatch")
	}
}

func TestTrustMetricsAllGetters(t *testing.T) {
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	overall := types.MustScore(0.8)
	confidence := types.MustScore(0.9)
	trend := types.MustWeight(0.1)
	decayRate := types.MustScore(0.05)
	eventID := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	evidence := []types.EventID{eventID}
	now := types.Now()

	m := NewTrustMetrics(actorID, overall, nil, confidence, trend, evidence, now, decayRate)
	if m.Confidence() != confidence {
		t.Error("Confidence mismatch")
	}
	if m.Trend() != trend {
		t.Error("Trend mismatch")
	}
	if len(m.Evidence()) != 1 {
		t.Error("Evidence should have 1 item")
	}
	if m.LastUpdated() != now {
		t.Error("LastUpdated mismatch")
	}
	if m.DecayRate() != decayRate {
		t.Error("DecayRate mismatch")
	}
}

func TestExpectationAllGetters(t *testing.T) {
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	trigger := types.MustEventID("019462a0-0000-7000-8000-000000000002")
	now := types.Now()

	exp, _ := NewExpectation(id, trigger, "test desc", now, SeverityLevelWarning, ExpectationStatusPending)
	if exp.ID() != id {
		t.Error("ID mismatch")
	}
	if exp.Trigger() != trigger {
		t.Error("Trigger mismatch")
	}
	if exp.Description() != "test desc" {
		t.Error("Description mismatch")
	}
	if exp.Deadline() != now {
		t.Error("Deadline mismatch")
	}
	if exp.Status() != ExpectationStatusPending {
		t.Error("Status mismatch")
	}
}

func TestViolationRecordAllGetters(t *testing.T) {
	id := types.MustEventID("019462a0-0000-7000-8000-000000000001")
	actorID := types.MustActorID("actor_00000000000000000000000000000001")
	evidence, _ := types.NewNonEmpty([]types.EventID{id})

	vr, _ := NewViolationRecord(id, id, SeverityLevelCritical, actorID, "desc", evidence)
	if vr.ID() != id {
		t.Error("ID mismatch")
	}
	if vr.Expectation() != id {
		t.Error("Expectation mismatch")
	}
	if vr.Actor() != actorID {
		t.Error("Actor mismatch")
	}
	if vr.Description() != "desc" {
		t.Error("Description mismatch")
	}
	if vr.Evidence().Len() != 1 {
		t.Error("Evidence should have 1 item")
	}
}

// Test canonical JSON with arrays and nested objects
func TestCanonicalJSONArray(t *testing.T) {
	content := rawJSONContent{
		typeName: "test",
		data: map[string]any{
			"items": []any{"b", "a"},
			"nested": map[string]any{
				"z": 1.0,
				"a": 2.0,
			},
		},
	}
	b, _ := json.Marshal(content)
	var raw map[string]any
	json.Unmarshal(b, &raw)
	got := sortedJSON(raw)
	expected := `{"items":["b","a"],"nested":{"a":2,"z":1}}`
	if got != expected {
		t.Errorf("sortedJSON with arrays/nested = %q, want %q", got, expected)
	}
}

func TestCanonicalContentJSONNilContent(t *testing.T) {
	got := canonicalContentJSON(nil)
	if got != "{}" {
		t.Errorf("canonicalContentJSON(nil) = %q, want %q", got, "{}")
	}
}

// --- rawJSONContent helper for canonical form testing ---

// rawJSONContent is a test helper that produces exact JSON from a map.
type rawJSONContent struct {
	typeName string
	data     map[string]any
}

func (c rawJSONContent) EventTypeName() string         { return c.typeName }
func (c rawJSONContent) Accept(v EventContentVisitor)  {} // not used in canonical form tests
func (c rawJSONContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.data)
}
