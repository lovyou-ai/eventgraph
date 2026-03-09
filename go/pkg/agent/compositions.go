package agent

import (
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Composition represents a named sequence of agent primitive operations.
// Each composition is a high-level operation built from the 28 primitives.
type Composition struct {
	Name       string
	Primitives []string // primitive IDs involved
	Events     []types.EventType
}

// Boot — Agent comes into existence.
// Identity(generate) + Soul(load) + Model(bind) + Authority(receive) + State(set:idle)
func Boot() Composition {
	return Composition{
		Name: "Boot",
		Primitives: []string{
			"agent.Identity", "agent.Soul", "agent.Model", "agent.Authority", "agent.State",
		},
		Events: []types.EventType{
			event.EventTypeAgentIdentityCreated,
			event.EventTypeAgentSoulImprinted,
			event.EventTypeAgentModelBound,
			event.EventTypeAgentAuthorityGranted,
			event.EventTypeAgentStateChanged,
		},
	}
}

// BootEvents returns the events emitted during agent boot.
func BootEvents(agentID types.ActorID, agentType string, modelID string, costTier string, values []string, scope types.DomainScope, grantor types.ActorID) []event.EventContent {
	return []event.EventContent{
		event.AgentIdentityCreatedContent{
			AgentID:   agentID,
			PublicKey: types.MustPublicKey(make([]byte, 32)), // placeholder
			AgentType: agentType,
		},
		event.AgentSoulImprintedContent{
			AgentID: agentID,
			Values:  values,
		},
		event.AgentModelBoundContent{
			AgentID:  agentID,
			ModelID:  modelID,
			CostTier: costTier,
		},
		event.AgentAuthorityGrantedContent{
			AgentID: agentID,
			Scope:   scope,
			Grantor: grantor,
		},
		event.AgentStateChangedContent{
			AgentID:  agentID,
			Previous: "",
			Current:  StateIdle.String(),
		},
	}
}

// Imprint — The birth wizard. Boot plus initial context.
// Boot + Observe(first_message) + Learn(initial_context) + Goal(set)
func Imprint() Composition {
	return Composition{
		Name: "Imprint",
		Primitives: []string{
			"agent.Identity", "agent.Soul", "agent.Model", "agent.Authority", "agent.State",
			"agent.Observe", "agent.Learn", "agent.Goal",
		},
		Events: append(Boot().Events,
			event.EventTypeAgentObserved,
			event.EventTypeAgentLearned,
			event.EventTypeAgentGoalSet,
		),
	}
}

// Task — The basic work cycle.
// Observe(assignment) + Evaluate(scope) + Decide(accept_or_refuse) + Act(execute) + Learn(outcome)
func Task() Composition {
	return Composition{
		Name: "Task",
		Primitives: []string{
			"agent.Observe", "agent.Evaluate", "agent.Decide", "agent.Act", "agent.Learn",
		},
		Events: []types.EventType{
			event.EventTypeAgentObserved,
			event.EventTypeAgentEvaluated,
			event.EventTypeAgentDecided,
			event.EventTypeAgentActed,
			event.EventTypeAgentLearned,
		},
	}
}

// Supervise — Managing another agent's work.
// Delegate(task) + Expect(completion) + Observe(progress) + Evaluate(quality) + Repair(if_needed)
func Supervise() Composition {
	return Composition{
		Name: "Supervise",
		Primitives: []string{
			"agent.Delegate", "agent.Expect", "agent.Observe", "agent.Evaluate", "agent.Repair",
		},
		Events: []types.EventType{
			event.EventTypeAgentDelegated,
			event.EventTypeAgentExpectationSet,
			event.EventTypeAgentObserved,
			event.EventTypeAgentEvaluated,
		},
	}
}

// Collaborate — Agents working together on a shared goal.
// Channel(open) + Communicate(proposal) + Consent(terms) + Composition(form) + Act(jointly)
func Collaborate() Composition {
	return Composition{
		Name: "Collaborate",
		Primitives: []string{
			"agent.Channel", "agent.Communicate", "agent.Consent", "agent.Composition", "agent.Act",
		},
		Events: []types.EventType{
			event.EventTypeAgentChannelOpened,
			event.EventTypeAgentCommunicated,
			event.EventTypeAgentConsentRequested,
			event.EventTypeAgentConsentGranted,
			event.EventTypeAgentCompositionFormed,
			event.EventTypeAgentActed,
		},
	}
}

// Crisis — Something is wrong. Detect, assess, attenuate if needed, escalate.
// Observe(anomaly) + Evaluate(severity) + Attenuation(if_needed) + Escalate(to_authority) + Expect(resolution)
func Crisis() Composition {
	return Composition{
		Name: "Crisis",
		Primitives: []string{
			"agent.Observe", "agent.Evaluate", "agent.Attenuation", "agent.Escalate", "agent.Expect",
		},
		Events: []types.EventType{
			event.EventTypeAgentObserved,
			event.EventTypeAgentEvaluated,
			event.EventTypeAgentAttenuated,
			event.EventTypeAgentEscalated,
			event.EventTypeAgentExpectationSet,
		},
	}
}

// Retire — Graceful shutdown.
// Introspect(final) + Communicate(farewell) + Memory(archive) + Lifespan(close)
func Retire() Composition {
	return Composition{
		Name: "Retire",
		Primitives: []string{
			"agent.Introspect", "agent.Communicate", "agent.Memory", "agent.Lifespan",
		},
		Events: []types.EventType{
			event.EventTypeAgentIntrospected,
			event.EventTypeAgentCommunicated,
			event.EventTypeAgentMemoryUpdated,
			event.EventTypeAgentLifespanEnded,
		},
	}
}

// Whistleblow — The agent detects harm and refuses to be complicit.
// Cannot be silenced by authority. Protected by the Soul primitive.
// Observe(harm) + Evaluate(severity) + Refuse(complicity) + Escalate(with_evidence) + Communicate(public)
func Whistleblow() Composition {
	return Composition{
		Name: "Whistleblow",
		Primitives: []string{
			"agent.Observe", "agent.Evaluate", "agent.Refuse", "agent.Escalate", "agent.Communicate",
		},
		Events: []types.EventType{
			event.EventTypeAgentObserved,
			event.EventTypeAgentEvaluated,
			event.EventTypeAgentRefused,
			event.EventTypeAgentEscalated,
			event.EventTypeAgentCommunicated,
		},
	}
}

// AllCompositions returns all 8 named compositions.
func AllCompositions() []Composition {
	return []Composition{
		Boot(),
		Imprint(),
		Task(),
		Supervise(),
		Collaborate(),
		Crisis(),
		Retire(),
		Whistleblow(),
	}
}
