package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestJusticeGrammar exercises the Justice Grammar (Layer 4: Legal).
// Operations: Legislate, Amend, File, Submit/Argue, Judge, Appeal, Enforce, Audit/Reform.
// Named functions: Trial, Injunction.
func TestJusticeGrammar(t *testing.T) {
	t.Run("Legislate", func(t *testing.T) {
		env := newTestEnv(t)
		council := env.actor("Council", 1, event.ActorTypeHuman)

		rule, _ := env.grammar.Emit(env.ctx, council.ID(),
			"rule enacted: all deployments require passing CI before merge",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		if rule.Source() != council.ID() {
			t.Error("rule source should be council")
		}
		env.verifyChain()
	})

	t.Run("AmendAndRepeal", func(t *testing.T) {
		env := newTestEnv(t)
		council := env.actor("Council", 1, event.ActorTypeHuman)

		rule, _ := env.grammar.Emit(env.ctx, council.ID(),
			"rule: max 2 deployments per day",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		amendment, _ := env.grammar.Derive(env.ctx, council.ID(),
			"amendment: max 5 deployments per day (updated based on team growth)",
			rule.ID(), env.convID, signer)

		repeal, _ := env.grammar.Retract(env.ctx, council.ID(),
			rule.ID(), "repealed: deployment limit no longer needed with CI/CD",
			env.convID, signer)

		// Amendment traces to original rule
		ancestors := env.ancestors(amendment.ID(), 5)
		if !containsEvent(ancestors, rule.ID()) {
			t.Error("amendment should trace to original rule")
		}

		// Repeal traces to rule
		repealAncestors := env.ancestors(repeal.ID(), 5)
		if !containsEvent(repealAncestors, rule.ID()) {
			t.Error("repeal should trace to original rule")
		}
		env.verifyChain()
	})

	t.Run("FileAndJudge", func(t *testing.T) {
		env := newTestEnv(t)
		plaintiff := env.actor("Plaintiff", 1, event.ActorTypeHuman)
		respondent := env.actor("Respondent", 2, event.ActorTypeHuman)
		judge := env.actor("Judge", 3, event.ActorTypeHuman)

		// File dispute
		filing, _ := env.grammar.Emit(env.ctx, plaintiff.ID(),
			"dispute filed: respondent deployed without CI, causing production outage",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Submit evidence
		evidence, _ := env.grammar.Respond(env.ctx, plaintiff.ID(),
			"evidence: deployment log shows no CI run, incident report #42",
			filing.ID(), env.convID, signer)

		// Respondent argues
		defense, _ := env.grammar.Respond(env.ctx, respondent.ID(),
			"defense: emergency hotfix, CI was down, filed waiver request",
			filing.ID(), env.convID, signer)

		// Judge rules
		ruling, _ := env.grammar.Merge(env.ctx, judge.ID(),
			"ruling: violation confirmed but mitigated — emergency exception valid, respondent must document procedure",
			[]types.EventID{evidence.ID(), defense.ID()}, env.convID, signer)

		ancestors := env.ancestors(ruling.ID(), 10)
		if !containsEvent(ancestors, filing.ID()) {
			t.Error("ruling should trace to filing")
		}
		env.verifyChain()
	})

	t.Run("Appeal", func(t *testing.T) {
		env := newTestEnv(t)
		respondent := env.actor("Respondent", 2, event.ActorTypeHuman)
		appellate := env.actor("AppellateJudge", 4, event.ActorTypeHuman)

		ruling, _ := env.grammar.Emit(env.ctx, env.system,
			"ruling: respondent must pay $500 fine",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		appeal, _ := env.grammar.Respond(env.ctx, respondent.ID(),
			"appeal: ruling was disproportionate given emergency circumstances",
			ruling.ID(), env.convID, signer)

		appellateRuling, _ := env.grammar.Derive(env.ctx, appellate.ID(),
			"appellate ruling: fine reduced to $100, original ruling partially overturned",
			appeal.ID(), env.convID, signer)

		ancestors := env.ancestors(appellateRuling.ID(), 10)
		if !containsEvent(ancestors, ruling.ID()) {
			t.Error("appellate ruling should trace to original ruling")
		}
		env.verifyChain()
	})

	t.Run("EnforceAndPardon", func(t *testing.T) {
		env := newTestEnv(t)
		enforcer := env.actor("Enforcer", 1, event.ActorTypeHuman)
		offender := env.actor("Offender", 2, event.ActorTypeHuman)

		ruling, _ := env.grammar.Emit(env.ctx, env.system,
			"ruling: offender suspended for 7 days",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		enforcement, _ := env.grammar.Derive(env.ctx, enforcer.ID(),
			"enforcement: suspension applied",
			ruling.ID(), env.convID, signer)

		// Pardon
		pardon, _ := env.grammar.Derive(env.ctx, enforcer.ID(),
			"pardon: suspension lifted early for good behaviour",
			enforcement.ID(), env.convID, signer)

		ancestors := env.ancestors(pardon.ID(), 10)
		if !containsEvent(ancestors, ruling.ID()) {
			t.Error("pardon should trace to original ruling")
		}
		_ = offender
		env.verifyChain()
	})

	t.Run("AuditAndReform", func(t *testing.T) {
		env := newTestEnv(t)
		auditor := env.actor("Auditor", 1, event.ActorTypeAI)

		rule, _ := env.grammar.Emit(env.ctx, env.system,
			"rule: all PRs require 2 approvals",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		audit, _ := env.grammar.Derive(env.ctx, auditor.ID(),
			"audit: reviewed 500 PRs, 23 merged with only 1 approval, compliance rate 95.4%",
			rule.ID(), env.convID, signer)

		reform, _ := env.grammar.Derive(env.ctx, auditor.ID(),
			"reform proposal: automate approval check as CI gate to prevent bypass",
			audit.ID(), env.convID, signer)

		ancestors := env.ancestors(reform.ID(), 10)
		if !containsEvent(ancestors, rule.ID()) {
			t.Error("reform should trace to original rule")
		}
		env.verifyChain()
	})

	t.Run("Trial", func(t *testing.T) {
		env := newTestEnv(t)
		plaintiff := env.actor("Plaintiff", 1, event.ActorTypeHuman)
		defendant := env.actor("Defendant", 2, event.ActorTypeHuman)
		judge := env.actor("Judge", 3, event.ActorTypeHuman)

		// Trial = File → Submit (both) → Argue (both) → Judge
		filing, _ := env.grammar.Emit(env.ctx, plaintiff.ID(),
			"case filed: data breach due to negligent security practices",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		pEvidence, _ := env.grammar.Respond(env.ctx, plaintiff.ID(),
			"plaintiff evidence: unpatched server, no encryption at rest",
			filing.ID(), env.convID, signer)
		dEvidence, _ := env.grammar.Respond(env.ctx, defendant.ID(),
			"defendant evidence: patch schedule existed, incident was zero-day",
			filing.ID(), env.convID, signer)

		pArgument, _ := env.grammar.Extend(env.ctx, plaintiff.ID(),
			"plaintiff argument: zero-day doesn't excuse missing encryption",
			pEvidence.ID(), env.convID, signer)
		dArgument, _ := env.grammar.Extend(env.ctx, defendant.ID(),
			"defendant argument: encryption was in deployment pipeline",
			dEvidence.ID(), env.convID, signer)

		verdict, _ := env.grammar.Merge(env.ctx, judge.ID(),
			"verdict: defendant liable — encryption should have been deployed earlier",
			[]types.EventID{pArgument.ID(), dArgument.ID()}, env.convID, signer)

		ancestors := env.ancestors(verdict.ID(), 15)
		if !containsEvent(ancestors, filing.ID()) {
			t.Error("verdict should trace to original filing")
		}
		if !containsEvent(ancestors, pEvidence.ID()) {
			t.Error("verdict should trace to plaintiff evidence")
		}
		if !containsEvent(ancestors, dEvidence.ID()) {
			t.Error("verdict should trace to defendant evidence")
		}
		env.verifyChain()
	})
}
