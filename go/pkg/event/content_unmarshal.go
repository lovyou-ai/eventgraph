package event

import (
	"encoding/json"
	"fmt"
)

// contentUnmarshaler deserializes JSON bytes into a typed EventContent.
type contentUnmarshaler func([]byte) (EventContent, error)

// contentUnmarshalers maps event type name → unmarshaler. Populated once at init.
var contentUnmarshalers map[string]contentUnmarshaler

func init() {
	contentUnmarshalers = map[string]contentUnmarshaler{
		// System / base
		"system.bootstrapped":      unmarshal[BootstrapContent],
		"trust.updated":            unmarshal[TrustUpdatedContent],
		"trust.score":              unmarshal[TrustScoreContent],
		"trust.decayed":            unmarshal[TrustDecayedContent],
		"authority.requested":      unmarshal[AuthorityRequestContent],
		"authority.resolved":       unmarshal[AuthorityResolvedContent],
		"authority.delegated":      unmarshal[AuthorityDelegatedContent],
		"authority.revoked":        unmarshal[AuthorityRevokedContent],
		"authority.timeout":        unmarshal[AuthorityTimeoutContent],
		"actor.registered":         unmarshal[ActorRegisteredContent],
		"actor.suspended":          unmarshal[ActorSuspendedContent],
		"actor.memorial":           unmarshal[ActorMemorialContent],
		"edge.created":             unmarshal[EdgeCreatedContent],
		"edge.superseded":          unmarshal[EdgeSupersededContent],
		"violation.detected":       unmarshal[ViolationDetectedContent],
		"chain.verified":           unmarshal[ChainVerifiedContent],
		"chain.broken":             unmarshal[ChainBrokenContent],
		"clock.tick":               unmarshal[ClockTickContent],
		"health.report":            unmarshal[HealthReportContent],
		"decision.branch.proposed": unmarshal[BranchProposedContent],
		"decision.branch.inserted": unmarshal[BranchInsertedContent],
		"decision.cost.report":     unmarshal[CostReportContent],
		"grammar.emit":             unmarshal[GrammarEmitContent],
		"grammar.respond":          unmarshal[GrammarRespondContent],
		"grammar.derive":           unmarshal[GrammarDeriveContent],
		"grammar.extend":           unmarshal[GrammarExtendContent],
		"grammar.retract":          unmarshal[GrammarRetractContent],
		"grammar.annotate":         unmarshal[GrammarAnnotateContent],
		"grammar.merge":            unmarshal[GrammarMergeContent],
		"grammar.consent":          unmarshal[GrammarConsentContent],
		"egip.hello.sent":          unmarshal[EGIPHelloSentContent],
		"egip.hello.received":      unmarshal[EGIPHelloReceivedContent],
		"egip.message.sent":        unmarshal[EGIPMessageSentContent],
		"egip.message.received":    unmarshal[EGIPMessageReceivedContent],
		"egip.receipt.sent":        unmarshal[EGIPReceiptSentContent],
		"egip.receipt.received":    unmarshal[EGIPReceiptReceivedContent],
		"egip.proof.requested":     unmarshal[EGIPProofRequestedContent],
		"egip.proof.received":      unmarshal[EGIPProofReceivedContent],
		"egip.treaty.proposed":     unmarshal[EGIPTreatyProposedContent],
		"egip.treaty.active":       unmarshal[EGIPTreatyActiveContent],
		"egip.trust.updated":       unmarshal[EGIPTrustUpdatedContent],

		// Agent structural
		"agent.identity.created":  unmarshal[AgentIdentityCreatedContent],
		"agent.identity.rotated":  unmarshal[AgentIdentityRotatedContent],
		"agent.soul.imprinted":    unmarshal[AgentSoulImprintedContent],
		"agent.model.bound":       unmarshal[AgentModelBoundContent],
		"agent.model.changed":     unmarshal[AgentModelChangedContent],
		"agent.memory.updated":    unmarshal[AgentMemoryUpdatedContent],
		"agent.state.changed":     unmarshal[AgentStateChangedContent],
		"agent.authority.granted": unmarshal[AgentAuthorityGrantedContent],
		"agent.authority.revoked": unmarshal[AgentAuthorityRevokedContent],
		"agent.trust.assessed":    unmarshal[AgentTrustAssessedContent],
		"agent.budget.allocated":  unmarshal[AgentBudgetAllocatedContent],
		"agent.budget.exhausted":  unmarshal[AgentBudgetExhaustedContent],
		"agent.role.assigned":     unmarshal[AgentRoleAssignedContent],
		"agent.lifespan.started":  unmarshal[AgentLifespanStartedContent],
		"agent.lifespan.extended": unmarshal[AgentLifespanExtendedContent],
		"agent.lifespan.ended":    unmarshal[AgentLifespanEndedContent],
		"agent.goal.set":          unmarshal[AgentGoalSetContent],
		"agent.goal.completed":    unmarshal[AgentGoalCompletedContent],
		"agent.goal.abandoned":    unmarshal[AgentGoalAbandonedContent],

		// Agent operational
		"agent.observed":            unmarshal[AgentObservedContent],
		"agent.probed":              unmarshal[AgentProbedContent],
		"agent.evaluated":           unmarshal[AgentEvaluatedContent],
		"agent.decided":             unmarshal[AgentDecidedContent],
		"agent.acted":               unmarshal[AgentActedContent],
		"agent.delegated":           unmarshal[AgentDelegatedContent],
		"agent.escalated":           unmarshal[AgentEscalatedContent],
		"agent.refused":             unmarshal[AgentRefusedContent],
		"agent.learned":             unmarshal[AgentLearnedContent],
		"agent.introspected":        unmarshal[AgentIntrospectedContent],
		"agent.communicated":        unmarshal[AgentCommunicatedContent],
		"agent.repaired":            unmarshal[AgentRepairedContent],
		"agent.expectation.set":     unmarshal[AgentExpectationSetContent],
		"agent.expectation.met":     unmarshal[AgentExpectationMetContent],
		"agent.expectation.expired": unmarshal[AgentExpectationExpiredContent],

		// Agent relational
		"agent.consent.requested":     unmarshal[AgentConsentRequestedContent],
		"agent.consent.granted":       unmarshal[AgentConsentGrantedContent],
		"agent.consent.denied":        unmarshal[AgentConsentDeniedContent],
		"agent.channel.opened":        unmarshal[AgentChannelOpenedContent],
		"agent.channel.closed":        unmarshal[AgentChannelClosedContent],
		"agent.composition.formed":    unmarshal[AgentCompositionFormedContent],
		"agent.composition.dissolved": unmarshal[AgentCompositionDissolvedContent],
		"agent.composition.joined":    unmarshal[AgentCompositionJoinedContent],
		"agent.composition.left":      unmarshal[AgentCompositionLeftContent],

		// Agent modal
		"agent.attenuated":         unmarshal[AgentAttenuatedContent],
		"agent.attenuation.lifted": unmarshal[AgentAttenuationLiftedContent],

		// CodeGraph data
		"codegraph.entity.defined":     unmarshal[CodeGraphEntityDefinedContent],
		"codegraph.entity.modified":    unmarshal[CodeGraphEntityModifiedContent],
		"codegraph.entity.related":     unmarshal[CodeGraphEntityRelatedContent],
		"codegraph.state.transitioned": unmarshal[CodeGraphStateTransitionedContent],

		// CodeGraph logic
		"codegraph.logic.transform.applied":  unmarshal[CodeGraphTransformAppliedContent],
		"codegraph.logic.condition.evaluated": unmarshal[CodeGraphConditionEvaluatedContent],
		"codegraph.logic.sequence.executed":   unmarshal[CodeGraphSequenceExecutedContent],
		"codegraph.logic.loop.iterated":       unmarshal[CodeGraphLoopIteratedContent],
		"codegraph.logic.trigger.fired":       unmarshal[CodeGraphTriggerFiredContent],
		"codegraph.logic.constraint.violated": unmarshal[CodeGraphConstraintViolatedContent],

		// CodeGraph IO
		"codegraph.io.query.executed":      unmarshal[CodeGraphQueryExecutedContent],
		"codegraph.io.command.executed":     unmarshal[CodeGraphCommandExecutedContent],
		"codegraph.io.command.rejected":     unmarshal[CodeGraphCommandRejectedContent],
		"codegraph.io.subscribe.registered": unmarshal[CodeGraphSubscribeRegisteredContent],
		"codegraph.io.authorize.granted":    unmarshal[CodeGraphAuthorizeGrantedContent],
		"codegraph.io.authorize.denied":     unmarshal[CodeGraphAuthorizeDeniedContent],
		"codegraph.io.search.executed":      unmarshal[CodeGraphSearchExecutedContent],
		"codegraph.io.interop.sent":         unmarshal[CodeGraphInteropSentContent],
		"codegraph.io.interop.received":     unmarshal[CodeGraphInteropReceivedContent],

		// CodeGraph UI
		"codegraph.ui.view.rendered":         unmarshal[CodeGraphViewRenderedContent],
		"codegraph.ui.action.invoked":        unmarshal[CodeGraphActionInvokedContent],
		"codegraph.ui.navigation.triggered":  unmarshal[CodeGraphNavigationTriggeredContent],
		"codegraph.ui.feedback.emitted":      unmarshal[CodeGraphFeedbackEmittedContent],
		"codegraph.ui.alert.dispatched":      unmarshal[CodeGraphAlertDispatchedContent],
		"codegraph.ui.drag.completed":        unmarshal[CodeGraphDragCompletedContent],
		"codegraph.ui.selection.changed":     unmarshal[CodeGraphSelectionChangedContent],
		"codegraph.ui.confirmation.resolved": unmarshal[CodeGraphConfirmationResolvedContent],

		// CodeGraph aesthetic
		"codegraph.aesthetic.skin.applied": unmarshal[CodeGraphSkinAppliedContent],

		// CodeGraph temporal
		"codegraph.temporal.undo.executed":   unmarshal[CodeGraphUndoExecutedContent],
		"codegraph.temporal.retry.attempted": unmarshal[CodeGraphRetryAttemptedContent],

		// CodeGraph resilience
		"codegraph.resilience.offline.entered": unmarshal[CodeGraphOfflineEnteredContent],
		"codegraph.resilience.offline.synced":  unmarshal[CodeGraphOfflineSyncedContent],

		// CodeGraph structural
		"codegraph.structural.scope.defined": unmarshal[CodeGraphScopeDefinedContent],

		// CodeGraph social
		"codegraph.social.presence.updated":   unmarshal[CodeGraphPresenceUpdatedContent],
		"codegraph.social.salience.triggered": unmarshal[CodeGraphSalienceTriggeredContent],
	}
}

// unmarshal is a generic helper that deserializes JSON into a value type T
// and returns it as EventContent.
func unmarshal[T EventContent](data []byte) (EventContent, error) {
	var c T
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

// UnmarshalContent deserializes JSON into the correct EventContent type
// based on the event type name. Returns an error for unknown event types.
func UnmarshalContent(eventType string, data []byte) (EventContent, error) {
	fn, ok := contentUnmarshalers[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
	return fn(data)
}

// IsKnownEventType returns true if the event type has a registered content unmarshaler.
func IsKnownEventType(eventType string) bool {
	_, ok := contentUnmarshalers[eventType]
	return ok
}

// RegisterContentUnmarshaler registers a custom content unmarshaler for an event type.
// Used by downstream packages (e.g., hive) to register their own event types
// without modifying eventgraph's init block. Call during startup before
// concurrent access.
func RegisterContentUnmarshaler(eventType string, fn func([]byte) (EventContent, error)) {
	contentUnmarshalers[eventType] = fn
}

// Unmarshal is a generic helper exposed for downstream packages that need to
// register their own content unmarshalers using the same pattern as eventgraph's
// internal types.
func Unmarshal[T EventContent](data []byte) (EventContent, error) {
	return unmarshal[T](data)
}
