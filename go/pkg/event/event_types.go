package event

import "github.com/lovyou-ai/eventgraph/go/pkg/types"

// Standard event type constants. Use these instead of bare strings.
var (
	// Trust
	EventTypeTrustUpdated = types.MustEventType("trust.updated")
	EventTypeTrustScore   = types.MustEventType("trust.score")
	EventTypeTrustDecayed = types.MustEventType("trust.decayed")

	// Authority
	EventTypeAuthorityRequested = types.MustEventType("authority.requested")
	EventTypeAuthorityResolved  = types.MustEventType("authority.resolved")
	EventTypeAuthorityDelegated = types.MustEventType("authority.delegated")
	EventTypeAuthorityRevoked   = types.MustEventType("authority.revoked")
	EventTypeAuthorityTimeout   = types.MustEventType("authority.timeout")

	// Actor
	EventTypeActorRegistered = types.MustEventType("actor.registered")
	EventTypeActorSuspended  = types.MustEventType("actor.suspended")
	EventTypeActorMemorial   = types.MustEventType("actor.memorial")

	// Edge
	EventTypeEdgeCreated    = types.MustEventType("edge.created")
	EventTypeEdgeSuperseded = types.MustEventType("edge.superseded")

	// Chain integrity
	EventTypeViolationDetected = types.MustEventType("violation.detected")
	EventTypeChainVerified     = types.MustEventType("chain.verified")
	EventTypeChainBroken       = types.MustEventType("chain.broken")

	// System
	EventTypeSystemBootstrapped = types.MustEventType("system.bootstrapped")
	EventTypeClockTick          = types.MustEventType("clock.tick")
	EventTypeHealthReport       = types.MustEventType("health.report")

	// Decision tree evolution
	EventTypeDecisionBranchProposed = types.MustEventType("decision.branch.proposed")
	EventTypeDecisionBranchInserted = types.MustEventType("decision.branch.inserted")
	EventTypeDecisionCostReport     = types.MustEventType("decision.cost.report")

	// EGIP protocol
	EventTypeEGIPHelloSent        = types.MustEventType("egip.hello.sent")
	EventTypeEGIPHelloReceived    = types.MustEventType("egip.hello.received")
	EventTypeEGIPMessageSent      = types.MustEventType("egip.message.sent")
	EventTypeEGIPMessageReceived  = types.MustEventType("egip.message.received")
	EventTypeEGIPReceiptSent      = types.MustEventType("egip.receipt.sent")
	EventTypeEGIPReceiptReceived  = types.MustEventType("egip.receipt.received")
	EventTypeEGIPProofRequested   = types.MustEventType("egip.proof.requested")
	EventTypeEGIPProofReceived    = types.MustEventType("egip.proof.received")
	EventTypeEGIPTreatyProposed   = types.MustEventType("egip.treaty.proposed")
	EventTypeEGIPTreatyActive     = types.MustEventType("egip.treaty.active")
	EventTypeEGIPTrustUpdated     = types.MustEventType("egip.trust.updated")
)
