package event

import "github.com/lovyou-ai/eventgraph/go/pkg/types"

// Agent event type constants. All use the "agent." prefix.
var (
	// Structural events
	EventTypeAgentIdentityCreated  = types.MustEventType("agent.identity.created")
	EventTypeAgentIdentityRotated  = types.MustEventType("agent.identity.rotated")
	EventTypeAgentSoulImprinted    = types.MustEventType("agent.soul.imprinted")
	EventTypeAgentModelBound       = types.MustEventType("agent.model.bound")
	EventTypeAgentModelChanged     = types.MustEventType("agent.model.changed")
	EventTypeAgentMemoryUpdated    = types.MustEventType("agent.memory.updated")
	EventTypeAgentStateChanged     = types.MustEventType("agent.state.changed")
	EventTypeAgentAuthorityGranted = types.MustEventType("agent.authority.granted")
	EventTypeAgentAuthorityRevoked = types.MustEventType("agent.authority.revoked")
	EventTypeAgentTrustAssessed    = types.MustEventType("agent.trust.assessed")
	EventTypeAgentBudgetAllocated  = types.MustEventType("agent.budget.allocated")
	EventTypeAgentBudgetExhausted  = types.MustEventType("agent.budget.exhausted")
	EventTypeAgentRoleAssigned     = types.MustEventType("agent.role.assigned")
	EventTypeAgentLifespanStarted  = types.MustEventType("agent.lifespan.started")
	EventTypeAgentLifespanExtended = types.MustEventType("agent.lifespan.extended")
	EventTypeAgentLifespanEnded    = types.MustEventType("agent.lifespan.ended")
	EventTypeAgentGoalSet          = types.MustEventType("agent.goal.set")
	EventTypeAgentGoalCompleted    = types.MustEventType("agent.goal.completed")
	EventTypeAgentGoalAbandoned    = types.MustEventType("agent.goal.abandoned")

	// Operational events
	EventTypeAgentObserved         = types.MustEventType("agent.observed")
	EventTypeAgentProbed           = types.MustEventType("agent.probed")
	EventTypeAgentEvaluated        = types.MustEventType("agent.evaluated")
	EventTypeAgentDecided          = types.MustEventType("agent.decided")
	EventTypeAgentActed            = types.MustEventType("agent.acted")
	EventTypeAgentDelegated        = types.MustEventType("agent.delegated")
	EventTypeAgentEscalated        = types.MustEventType("agent.escalated")
	EventTypeAgentRefused          = types.MustEventType("agent.refused")
	EventTypeAgentLearned          = types.MustEventType("agent.learned")
	EventTypeAgentIntrospected     = types.MustEventType("agent.introspected")
	EventTypeAgentCommunicated     = types.MustEventType("agent.communicated")
	EventTypeAgentRepaired         = types.MustEventType("agent.repaired")
	EventTypeAgentExpectationSet   = types.MustEventType("agent.expectation.set")
	EventTypeAgentExpectationMet   = types.MustEventType("agent.expectation.met")
	EventTypeAgentExpectationExpired = types.MustEventType("agent.expectation.expired")

	// Relational events
	EventTypeAgentConsentRequested      = types.MustEventType("agent.consent.requested")
	EventTypeAgentConsentGranted        = types.MustEventType("agent.consent.granted")
	EventTypeAgentConsentDenied         = types.MustEventType("agent.consent.denied")
	EventTypeAgentChannelOpened         = types.MustEventType("agent.channel.opened")
	EventTypeAgentChannelClosed         = types.MustEventType("agent.channel.closed")
	EventTypeAgentCompositionFormed     = types.MustEventType("agent.composition.formed")
	EventTypeAgentCompositionDissolved  = types.MustEventType("agent.composition.dissolved")
	EventTypeAgentCompositionJoined     = types.MustEventType("agent.composition.joined")
	EventTypeAgentCompositionLeft       = types.MustEventType("agent.composition.left")

	// Modal events
	EventTypeAgentAttenuated        = types.MustEventType("agent.attenuated")
	EventTypeAgentAttenuationLifted = types.MustEventType("agent.attenuation.lifted")
)

// AllAgentEventTypes returns all registered agent event types.
func AllAgentEventTypes() []types.EventType {
	return []types.EventType{
		// Structural
		EventTypeAgentIdentityCreated, EventTypeAgentIdentityRotated,
		EventTypeAgentSoulImprinted,
		EventTypeAgentModelBound, EventTypeAgentModelChanged,
		EventTypeAgentMemoryUpdated,
		EventTypeAgentStateChanged,
		EventTypeAgentAuthorityGranted, EventTypeAgentAuthorityRevoked,
		EventTypeAgentTrustAssessed,
		EventTypeAgentBudgetAllocated, EventTypeAgentBudgetExhausted,
		EventTypeAgentRoleAssigned,
		EventTypeAgentLifespanStarted, EventTypeAgentLifespanExtended, EventTypeAgentLifespanEnded,
		EventTypeAgentGoalSet, EventTypeAgentGoalCompleted, EventTypeAgentGoalAbandoned,
		// Operational
		EventTypeAgentObserved, EventTypeAgentProbed,
		EventTypeAgentEvaluated, EventTypeAgentDecided,
		EventTypeAgentActed, EventTypeAgentDelegated,
		EventTypeAgentEscalated, EventTypeAgentRefused,
		EventTypeAgentLearned, EventTypeAgentIntrospected,
		EventTypeAgentCommunicated, EventTypeAgentRepaired,
		EventTypeAgentExpectationSet, EventTypeAgentExpectationMet, EventTypeAgentExpectationExpired,
		// Relational
		EventTypeAgentConsentRequested, EventTypeAgentConsentGranted, EventTypeAgentConsentDenied,
		EventTypeAgentChannelOpened, EventTypeAgentChannelClosed,
		EventTypeAgentCompositionFormed, EventTypeAgentCompositionDissolved,
		EventTypeAgentCompositionJoined, EventTypeAgentCompositionLeft,
		// Modal
		EventTypeAgentAttenuated, EventTypeAgentAttenuationLifted,
	}
}
