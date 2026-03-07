package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestAlignmentGrammar exercises the Alignment Grammar (Layer 7: Ethics).
// Operations: Constrain, Detect-Harm, Assess-Fairness, Flag-Dilemma, Weigh,
// Explain, Assign, Repair, Grow, Care.
// Named functions: Ethics-Audit, Guardrail, Restorative-Justice.
func TestAlignmentGrammar(t *testing.T) {
	t.Run("Constrain", func(t *testing.T) {
		env := newTestEnv(t)
		admin := env.actor("Admin", 1, event.ActorTypeHuman)

		constraint, _ := env.grammar.Emit(env.ctx, admin.ID(),
			"constraint: no model may process personal data without explicit consent",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		if constraint.Source() != admin.ID() {
			t.Error("constraint source should be admin")
		}
		env.verifyChain()
	})

	t.Run("DetectHarm", func(t *testing.T) {
		env := newTestEnv(t)
		monitor := env.actor("Monitor", 1, event.ActorTypeAI)

		action, _ := env.grammar.Emit(env.ctx, env.system,
			"action: model generated content that stereotypes a demographic group",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		harm, _ := env.grammar.Derive(env.ctx, monitor.ID(),
			"harm detected: severity medium, type stereotyping, affected group identified",
			action.ID(), env.convID, signer)

		ancestors := env.ancestors(harm.ID(), 5)
		if !containsEvent(ancestors, action.ID()) {
			t.Error("harm detection should trace to action")
		}
		env.verifyChain()
	})

	t.Run("AssessFairness", func(t *testing.T) {
		env := newTestEnv(t)
		auditor := env.actor("Auditor", 1, event.ActorTypeAI)

		assessment, _ := env.grammar.Emit(env.ctx, auditor.ID(),
			"fairness assessment: 500 decisions analysed, overall score 0.78, zip code disparity 8%",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		if assessment.Source() != auditor.ID() {
			t.Error("assessment source should be auditor")
		}
		env.verifyChain()
	})

	t.Run("FlagDilemma", func(t *testing.T) {
		env := newTestEnv(t)
		agent := env.actor("Agent", 1, event.ActorTypeAI)

		situation, _ := env.grammar.Emit(env.ctx, agent.ID(),
			"situation: user requests deletion of data that is also evidence in an ongoing audit",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		dilemma, _ := env.grammar.Derive(env.ctx, agent.ID(),
			"dilemma: privacy (right to deletion) vs accountability (audit evidence preservation)",
			situation.ID(), env.convID, signer)

		// Escalate for human decision
		_, _ = env.graph.Record(
			event.EventTypeAuthorityRequested, agent.ID(),
			event.AuthorityRequestContent{
				Actor:  agent.ID(),
				Action: "resolve_privacy_vs_audit_dilemma",
				Level:  event.AuthorityLevelRequired,
			},
			[]types.EventID{dilemma.ID()}, env.convID, signer)

		env.verifyChain()
	})

	t.Run("WeighAndExplain", func(t *testing.T) {
		env := newTestEnv(t)
		agent := env.actor("Agent", 1, event.ActorTypeAI)

		decision, _ := env.grammar.Emit(env.ctx, agent.ID(),
			"decision: deny loan application",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		weighing, _ := env.grammar.Derive(env.ctx, agent.ID(),
			"weighing: income (0.4) + credit history (0.3) + debt ratio (0.3) = below threshold",
			decision.ID(), env.convID, signer)

		explanation, _ := env.grammar.Derive(env.ctx, agent.ID(),
			"explanation: denied due to debt-to-income ratio of 0.52 exceeding 0.43 threshold, other factors were acceptable",
			weighing.ID(), env.convID, signer)

		ancestors := env.ancestors(explanation.ID(), 10)
		if !containsEvent(ancestors, decision.ID()) {
			t.Error("explanation should trace to decision")
		}
		env.verifyChain()
	})

	t.Run("AssignAndRepair", func(t *testing.T) {
		env := newTestEnv(t)
		auditor := env.actor("Auditor", 1, event.ActorTypeAI)
		admin := env.actor("Admin", 2, event.ActorTypeHuman)
		affected := env.actor("Affected", 3, event.ActorTypeHuman)

		harm, _ := env.grammar.Emit(env.ctx, auditor.ID(),
			"harm: 23 applicants wrongly denied due to proxy variable",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		responsibility, _ := env.grammar.Annotate(env.ctx, auditor.ID(),
			harm.ID(), "responsibility",
			"agent: 0.4 (used proxy), admin: 0.6 (approved model without bias test)",
			env.convID, signer)

		redress, _ := env.grammar.Derive(env.ctx, auditor.ID(),
			"redress: re-review 23 applications without proxy variable",
			responsibility.ID(), env.convID, signer)

		// Repair requires consent from affected party
		_, _ = env.grammar.Consent(env.ctx, admin.ID(), affected.ID(),
			"accept redress: re-review application with corrected model",
			types.MustDomainScope("lending"),
			redress.ID(), env.convID, signer)

		env.verifyChain()
	})

	t.Run("EthicsAudit", func(t *testing.T) {
		env := newTestEnv(t)
		auditor := env.actor("Auditor", 1, event.ActorTypeAI)

		// Batch assessment
		fairness, _ := env.grammar.Emit(env.ctx, auditor.ID(),
			"fairness assessment: score 0.82 across 1000 decisions",
			env.convID, []types.EventID{env.boot.ID()}, signer)
		harmScan, _ := env.grammar.Emit(env.ctx, auditor.ID(),
			"harm scan: 2 medium-severity issues found",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		report, _ := env.grammar.Merge(env.ctx, auditor.ID(),
			"ethics audit summary: overall score 0.79, 2 issues requiring attention",
			[]types.EventID{fairness.ID(), harmScan.ID()}, env.convID, signer)

		ancestors := env.ancestors(report.ID(), 5)
		if !containsEvent(ancestors, fairness.ID()) {
			t.Error("report should include fairness assessment")
		}
		if !containsEvent(ancestors, harmScan.ID()) {
			t.Error("report should include harm scan")
		}
		env.verifyChain()
	})

	t.Run("RestorativeJustice", func(t *testing.T) {
		env := newTestEnv(t)
		auditor := env.actor("Auditor", 1, event.ActorTypeAI)
		agent := env.actor("Agent", 2, event.ActorTypeAI)

		// Detect → Assign → Repair → Grow
		harm, _ := env.grammar.Emit(env.ctx, auditor.ID(),
			"harm detected: biased recommendations",
			env.convID, []types.EventID{env.boot.ID()}, signer)
		assign, _ := env.grammar.Derive(env.ctx, auditor.ID(),
			"responsibility: agent 0.7, training data 0.3",
			harm.ID(), env.convID, signer)
		repair, _ := env.grammar.Derive(env.ctx, auditor.ID(),
			"repair: retrained with balanced dataset",
			assign.ID(), env.convID, signer)
		growth, _ := env.grammar.Extend(env.ctx, agent.ID(),
			"growth: learned to check training data distribution before deployment",
			repair.ID(), env.convID, signer)

		ancestors := env.ancestors(growth.ID(), 10)
		if !containsEvent(ancestors, harm.ID()) {
			t.Error("growth should trace all the way to harm detection")
		}
		env.verifyChain()
	})
}
