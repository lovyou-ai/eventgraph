package event

import "github.com/lovyou-ai/eventgraph/go/pkg/types"

// agentContent is embedded in all agent content types to satisfy the
// EventContent interface's Accept method. Agent content types use their
// own visitor (AgentEventContentVisitor) rather than the base visitor.
type agentContent struct{}

func (agentContent) Accept(EventContentVisitor) {}

// --- Agent structural content ---

// AgentIdentityCreatedContent records agent registration.
type AgentIdentityCreatedContent struct {
	agentContent
	AgentID   types.ActorID  `json:"AgentID"`
	PublicKey types.PublicKey `json:"PublicKey"`
	AgentType string         `json:"AgentType"`
}

func (c AgentIdentityCreatedContent) EventTypeName() string { return "agent.identity.created" }

// AgentIdentityRotatedContent records key rotation.
type AgentIdentityRotatedContent struct {
	agentContent
	AgentID    types.ActorID  `json:"AgentID"`
	NewKey     types.PublicKey `json:"NewKey"`
	PreviousKey types.PublicKey `json:"PreviousKey"`
}

func (c AgentIdentityRotatedContent) EventTypeName() string { return "agent.identity.rotated" }

// AgentSoulImprintedContent records values set (once, immutable after).
type AgentSoulImprintedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Values  []string      `json:"Values"`
}

func (c AgentSoulImprintedContent) EventTypeName() string { return "agent.soul.imprinted" }

// AgentModelBoundContent records intelligence binding.
type AgentModelBoundContent struct {
	agentContent
	AgentID  types.ActorID `json:"AgentID"`
	ModelID  string        `json:"ModelID"`
	CostTier string        `json:"CostTier"`
}

func (c AgentModelBoundContent) EventTypeName() string { return "agent.model.bound" }

// AgentModelChangedContent records model swap.
type AgentModelChangedContent struct {
	agentContent
	AgentID       types.ActorID `json:"AgentID"`
	PreviousModel string        `json:"PreviousModel"`
	NewModel      string        `json:"NewModel"`
	Reason        string        `json:"Reason"`
}

func (c AgentModelChangedContent) EventTypeName() string { return "agent.model.changed" }

// AgentMemoryUpdatedContent records persistent state change.
type AgentMemoryUpdatedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Key     string        `json:"Key"`
	Action  string        `json:"Action"` // "set", "delete", "archive"
}

func (c AgentMemoryUpdatedContent) EventTypeName() string { return "agent.memory.updated" }

// AgentStateChangedContent records operational state transition.
type AgentStateChangedContent struct {
	agentContent
	AgentID  types.ActorID `json:"AgentID"`
	Previous string        `json:"Previous"`
	Current  string        `json:"Current"`
}

func (c AgentStateChangedContent) EventTypeName() string { return "agent.state.changed" }

// AgentAuthorityGrantedContent records authority scope received.
type AgentAuthorityGrantedContent struct {
	agentContent
	AgentID types.ActorID     `json:"AgentID"`
	Scope   types.DomainScope `json:"Scope"`
	Grantor types.ActorID     `json:"Grantor"`
}

func (c AgentAuthorityGrantedContent) EventTypeName() string { return "agent.authority.granted" }

// AgentAuthorityRevokedContent records authority scope removed.
type AgentAuthorityRevokedContent struct {
	agentContent
	AgentID types.ActorID     `json:"AgentID"`
	Scope   types.DomainScope `json:"Scope"`
	Revoker types.ActorID     `json:"Revoker"`
	Reason  string            `json:"Reason"`
}

func (c AgentAuthorityRevokedContent) EventTypeName() string { return "agent.authority.revoked" }

// AgentTrustAssessedContent records trust score update toward another actor.
type AgentTrustAssessedContent struct {
	agentContent
	AgentID  types.ActorID `json:"AgentID"`
	Target   types.ActorID `json:"Target"`
	Previous types.Score   `json:"Previous"`
	Current  types.Score   `json:"Current"`
}

func (c AgentTrustAssessedContent) EventTypeName() string { return "agent.trust.assessed" }

// AgentBudgetAllocatedContent records resource budget set or adjusted.
type AgentBudgetAllocatedContent struct {
	agentContent
	AgentID    types.ActorID `json:"AgentID"`
	TokenLimit int           `json:"TokenLimit"`
	CostLimit  float64       `json:"CostLimit"`
	TimeLimit  int64         `json:"TimeLimit"` // nanoseconds
}

func (c AgentBudgetAllocatedContent) EventTypeName() string { return "agent.budget.allocated" }

// AgentBudgetExhaustedContent records budget limit reached.
type AgentBudgetExhaustedContent struct {
	agentContent
	AgentID  types.ActorID `json:"AgentID"`
	Resource string        `json:"Resource"` // "tokens", "cost", "time"
}

func (c AgentBudgetExhaustedContent) EventTypeName() string { return "agent.budget.exhausted" }

// AgentRoleAssignedContent records role set or changed.
type AgentRoleAssignedContent struct {
	agentContent
	AgentID      types.ActorID `json:"AgentID"`
	Role         string        `json:"Role"`
	PreviousRole string        `json:"PreviousRole,omitempty"`
}

func (c AgentRoleAssignedContent) EventTypeName() string { return "agent.role.assigned" }

// AgentLifespanStartedContent records agent birth.
type AgentLifespanStartedContent struct {
	agentContent
	AgentID types.ActorID   `json:"AgentID"`
	Started types.Timestamp `json:"Started"`
}

func (c AgentLifespanStartedContent) EventTypeName() string { return "agent.lifespan.started" }

// AgentLifespanExtendedContent records lifespan adjustment.
type AgentLifespanExtendedContent struct {
	agentContent
	AgentID  types.ActorID   `json:"AgentID"`
	NewEnd   types.Timestamp `json:"NewEnd"`
	Reason   string          `json:"Reason"`
}

func (c AgentLifespanExtendedContent) EventTypeName() string { return "agent.lifespan.extended" }

// AgentLifespanEndedContent records agent shutdown.
type AgentLifespanEndedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Reason  string        `json:"Reason"`
}

func (c AgentLifespanEndedContent) EventTypeName() string { return "agent.lifespan.ended" }

// AgentGoalSetContent records new objective assigned.
type AgentGoalSetContent struct {
	agentContent
	AgentID     types.ActorID `json:"AgentID"`
	Goal        string        `json:"Goal"`
	Priority    int           `json:"Priority"`
	ParentGoal  string        `json:"ParentGoal,omitempty"`
}

func (c AgentGoalSetContent) EventTypeName() string { return "agent.goal.set" }

// AgentGoalCompletedContent records objective achieved.
type AgentGoalCompletedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Goal    string        `json:"Goal"`
}

func (c AgentGoalCompletedContent) EventTypeName() string { return "agent.goal.completed" }

// AgentGoalAbandonedContent records objective dropped with reason.
type AgentGoalAbandonedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Goal    string        `json:"Goal"`
	Reason  string        `json:"Reason"`
}

func (c AgentGoalAbandonedContent) EventTypeName() string { return "agent.goal.abandoned" }

// --- Agent operational content ---

// AgentObservedContent records event received and processed.
type AgentObservedContent struct {
	agentContent
	AgentID    types.ActorID `json:"AgentID"`
	EventCount int           `json:"EventCount"`
}

func (c AgentObservedContent) EventTypeName() string { return "agent.observed" }

// AgentProbedContent records active query executed.
type AgentProbedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Query   string        `json:"Query"`
	Results int           `json:"Results"`
}

func (c AgentProbedContent) EventTypeName() string { return "agent.probed" }

// AgentEvaluatedContent records judgment produced.
type AgentEvaluatedContent struct {
	agentContent
	AgentID    types.ActorID `json:"AgentID"`
	Subject    string        `json:"Subject"`
	Confidence types.Score   `json:"Confidence"`
	Result     string        `json:"Result"`
}

func (c AgentEvaluatedContent) EventTypeName() string { return "agent.evaluated" }

// AgentDecidedContent records commitment made.
type AgentDecidedContent struct {
	agentContent
	AgentID    types.ActorID `json:"AgentID"`
	Action     string        `json:"Action"`
	Confidence types.Score   `json:"Confidence"`
}

func (c AgentDecidedContent) EventTypeName() string { return "agent.decided" }

// AgentActedContent records action executed on graph.
type AgentActedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Action  string        `json:"Action"`
	Target  string        `json:"Target"`
}

func (c AgentActedContent) EventTypeName() string { return "agent.acted" }

// AgentDelegatedContent records work assigned to another agent.
type AgentDelegatedContent struct {
	agentContent
	AgentID  types.ActorID `json:"AgentID"`
	Delegate types.ActorID `json:"Delegate"`
	Task     string        `json:"Task"`
}

func (c AgentDelegatedContent) EventTypeName() string { return "agent.delegated" }

// AgentEscalatedContent records issue passed upward.
type AgentEscalatedContent struct {
	agentContent
	AgentID   types.ActorID `json:"AgentID"`
	Authority types.ActorID `json:"Authority"`
	Reason    string        `json:"Reason"`
}

func (c AgentEscalatedContent) EventTypeName() string { return "agent.escalated" }

// AgentRefusedContent records action declined with reason.
type AgentRefusedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Action  string        `json:"Action"`
	Reason  string        `json:"Reason"`
}

func (c AgentRefusedContent) EventTypeName() string { return "agent.refused" }

// AgentLearnedContent records memory updated from outcome.
type AgentLearnedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Lesson  string        `json:"Lesson"`
	Source  string        `json:"Source"`
}

func (c AgentLearnedContent) EventTypeName() string { return "agent.learned" }

// AgentIntrospectedContent records self-observation.
type AgentIntrospectedContent struct {
	agentContent
	AgentID     types.ActorID `json:"AgentID"`
	Observation string        `json:"Observation"`
}

func (c AgentIntrospectedContent) EventTypeName() string { return "agent.introspected" }

// AgentCommunicatedContent records message sent.
type AgentCommunicatedContent struct {
	agentContent
	AgentID   types.ActorID `json:"AgentID"`
	Recipient types.ActorID `json:"Recipient"`
	Channel   string        `json:"Channel"`
}

func (c AgentCommunicatedContent) EventTypeName() string { return "agent.communicated" }

// AgentRepairedContent records prior action corrected.
type AgentRepairedContent struct {
	agentContent
	AgentID       types.ActorID `json:"AgentID"`
	OriginalEvent types.EventID `json:"OriginalEvent"`
	Correction    string        `json:"Correction"`
}

func (c AgentRepairedContent) EventTypeName() string { return "agent.repaired" }

// AgentExpectationSetContent records monitoring condition created.
type AgentExpectationSetContent struct {
	agentContent
	AgentID   types.ActorID `json:"AgentID"`
	Condition string        `json:"Condition"`
}

func (c AgentExpectationSetContent) EventTypeName() string { return "agent.expectation.set" }

// AgentExpectationMetContent records monitored condition triggered.
type AgentExpectationMetContent struct {
	agentContent
	AgentID   types.ActorID `json:"AgentID"`
	Condition string        `json:"Condition"`
}

func (c AgentExpectationMetContent) EventTypeName() string { return "agent.expectation.met" }

// AgentExpectationExpiredContent records monitoring condition timed out.
type AgentExpectationExpiredContent struct {
	agentContent
	AgentID   types.ActorID `json:"AgentID"`
	Condition string        `json:"Condition"`
}

func (c AgentExpectationExpiredContent) EventTypeName() string { return "agent.expectation.expired" }

// --- Agent relational content ---

// AgentConsentRequestedContent records consent asked of another agent.
type AgentConsentRequestedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Target  types.ActorID `json:"Target"`
	Action  string        `json:"Action"`
}

func (c AgentConsentRequestedContent) EventTypeName() string { return "agent.consent.requested" }

// AgentConsentGrantedContent records consent given.
type AgentConsentGrantedContent struct {
	agentContent
	AgentID   types.ActorID `json:"AgentID"`
	Requester types.ActorID `json:"Requester"`
	Action    string        `json:"Action"`
}

func (c AgentConsentGrantedContent) EventTypeName() string { return "agent.consent.granted" }

// AgentConsentDeniedContent records consent refused.
type AgentConsentDeniedContent struct {
	agentContent
	AgentID   types.ActorID `json:"AgentID"`
	Requester types.ActorID `json:"Requester"`
	Action    string        `json:"Action"`
	Reason    string        `json:"Reason"`
}

func (c AgentConsentDeniedContent) EventTypeName() string { return "agent.consent.denied" }

// AgentChannelOpenedContent records communication channel established.
type AgentChannelOpenedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Peer    types.ActorID `json:"Peer"`
	Channel string        `json:"Channel"`
}

func (c AgentChannelOpenedContent) EventTypeName() string { return "agent.channel.opened" }

// AgentChannelClosedContent records channel terminated.
type AgentChannelClosedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	Peer    types.ActorID `json:"Peer"`
	Channel string        `json:"Channel"`
	Reason  string        `json:"Reason"`
}

func (c AgentChannelClosedContent) EventTypeName() string { return "agent.channel.closed" }

// AgentCompositionFormedContent records group created.
type AgentCompositionFormedContent struct {
	agentContent
	AgentID types.ActorID   `json:"AgentID"`
	Members []types.ActorID `json:"Members"`
	Purpose string          `json:"Purpose"`
}

func (c AgentCompositionFormedContent) EventTypeName() string { return "agent.composition.formed" }

// AgentCompositionDissolvedContent records group disbanded.
type AgentCompositionDissolvedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	GroupID string        `json:"GroupID"`
	Reason  string        `json:"Reason"`
}

func (c AgentCompositionDissolvedContent) EventTypeName() string { return "agent.composition.dissolved" }

// AgentCompositionJoinedContent records agent joined existing group.
type AgentCompositionJoinedContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	GroupID string        `json:"GroupID"`
}

func (c AgentCompositionJoinedContent) EventTypeName() string { return "agent.composition.joined" }

// AgentCompositionLeftContent records agent left group.
type AgentCompositionLeftContent struct {
	agentContent
	AgentID types.ActorID `json:"AgentID"`
	GroupID string        `json:"GroupID"`
	Reason  string        `json:"Reason"`
}

func (c AgentCompositionLeftContent) EventTypeName() string { return "agent.composition.left" }

// --- Agent modal content ---

// AgentAttenuatedContent records scope/confidence/authority reduced.
type AgentAttenuatedContent struct {
	agentContent
	AgentID   types.ActorID `json:"AgentID"`
	Dimension string        `json:"Dimension"` // "scope", "confidence", "authority"
	Previous  string        `json:"Previous"`
	Current   string        `json:"Current"`
	Reason    string        `json:"Reason"`
}

func (c AgentAttenuatedContent) EventTypeName() string { return "agent.attenuated" }

// AgentAttenuationLiftedContent records attenuation removed.
type AgentAttenuationLiftedContent struct {
	agentContent
	AgentID   types.ActorID `json:"AgentID"`
	Dimension string        `json:"Dimension"`
	Reason    string        `json:"Reason"`
}

func (c AgentAttenuationLiftedContent) EventTypeName() string { return "agent.attenuation.lifted" }
