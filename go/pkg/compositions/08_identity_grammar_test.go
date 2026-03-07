package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestIdentityGrammar exercises the Identity Grammar (Layer 8: Identity).
// Operations: Introspect, Narrate, Align, Bound, Aspire, Transform, Disclose,
// Recognize, Distinguish, Memorialize.
// Named functions: Credential, Retirement.
func TestIdentityGrammar(t *testing.T) {
	t.Run("Introspect", func(t *testing.T) {
		env := newTestEnv(t)
		agent := env.actor("Agent", 1, event.ActorTypeAI)

		selfModel, _ := env.grammar.Emit(env.ctx, agent.ID(),
			"self-model: strengths=[code_review, testing], weaknesses=[architecture], confidence 0.8",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		if selfModel.Source() != agent.ID() {
			t.Error("self-model source should be the agent itself")
		}
		env.verifyChain()
	})

	t.Run("NarrateAndAlign", func(t *testing.T) {
		env := newTestEnv(t)
		agent := env.actor("Agent", 1, event.ActorTypeAI)

		selfModel, _ := env.grammar.Emit(env.ctx, agent.ID(),
			"self-model: values thoroughness",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		narrative, _ := env.grammar.Derive(env.ctx, agent.ID(),
			"narrative: started as simple reviewer, grew into security-conscious architect over 6 months",
			selfModel.ID(), env.convID, signer)

		alignment, _ := env.grammar.Annotate(env.ctx, agent.ID(),
			selfModel.ID(), "alignment",
			"gap: values thoroughness but rushed 12% of reviews — alignment score 0.88",
			env.convID, signer)

		_ = narrative
		_ = alignment
		env.verifyChain()
	})

	t.Run("Bound", func(t *testing.T) {
		env := newTestEnv(t)
		agent := env.actor("Agent", 1, event.ActorTypeAI)

		boundary, _ := env.grammar.Emit(env.ctx, agent.ID(),
			"boundary: internal_reasoning is private and impermeable — no external queries",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		if boundary.Source() != agent.ID() {
			t.Error("boundary should be set by the agent itself")
		}
		env.verifyChain()
	})

	t.Run("AspireAndTransform", func(t *testing.T) {
		env := newTestEnv(t)
		agent := env.actor("Agent", 1, event.ActorTypeAI)

		aspiration, _ := env.grammar.Emit(env.ctx, agent.ID(),
			"aspiration: become proficient at architecture review within 3 months",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Work happens (summary)
		work, _ := env.grammar.Extend(env.ctx, agent.ID(),
			"work: 500 reviews completed, found critical security flaw in auth module",
			aspiration.ID(), env.convID, signer)

		transformation, _ := env.grammar.Derive(env.ctx, agent.ID(),
			"transformation: evolved from code reviewer to security-aware architect, catalyst: auth module finding",
			work.ID(), env.convID, signer)

		ancestors := env.ancestors(transformation.ID(), 10)
		if !containsEvent(ancestors, aspiration.ID()) {
			t.Error("transformation should trace to aspiration")
		}
		env.verifyChain()
	})

	t.Run("Disclose", func(t *testing.T) {
		env := newTestEnv(t)
		agent := env.actor("Agent", 1, event.ActorTypeAI)
		partner := env.actor("Partner", 2, event.ActorTypeHuman)

		selfModel, _ := env.grammar.Emit(env.ctx, agent.ID(),
			"self-model: strengths=[review, testing], weaknesses=[architecture]",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Selective disclosure: share strengths only
		disclosure, _ := env.grammar.Derive(env.ctx, agent.ID(),
			"disclosure: 2400+ reviews completed, speciality in security review",
			selfModel.ID(), env.convID, signer)

		_ = partner
		_ = disclosure
		env.verifyChain()
	})

	t.Run("RecognizeAndDistinguish", func(t *testing.T) {
		env := newTestEnv(t)
		alpha := env.actor("Alpha", 1, event.ActorTypeAI)
		beta := env.actor("Beta", 2, event.ActorTypeAI)

		recognize, _ := env.grammar.Emit(env.ctx, env.system,
			"recognition: Alpha's unique contribution to security review practices",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		distinguish, _ := env.grammar.Annotate(env.ctx, env.system,
			recognize.ID(), "uniqueness",
			"Alpha specialises in auth patterns, Beta in data pipeline — overlap 0.3",
			env.convID, signer)

		_ = alpha
		_ = beta
		_ = distinguish
		env.verifyChain()
	})

	t.Run("Retirement", func(t *testing.T) {
		env := newTestEnv(t)
		agent := env.actor("Agent", 1, event.ActorTypeAI)
		successor := env.actor("Successor", 2, event.ActorTypeAI)

		// Memorialize
		memorial, _ := env.graph.Record(
			event.EventTypeActorMemorial, env.system,
			event.ActorMemorialContent{
				ActorID: agent.ID(),
				Reason:  env.boot.ID(),
			},
			[]types.EventID{env.boot.ID()}, env.convID, signer)

		// Transfer authority
		transfer, _ := env.grammar.Delegate(env.ctx, env.system, successor.ID(),
			types.MustDomainScope("code_review"), types.MustWeight(0.8),
			memorial.ID(), env.convID, signer)

		ancestors := env.ancestors(transfer.ID(), 5)
		if !containsEvent(ancestors, memorial.ID()) {
			t.Error("transfer should trace to memorial")
		}
		env.verifyChain()
	})
}
