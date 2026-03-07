package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestMeaningGrammar exercises the Meaning Grammar (Layer 11: Governance).
// Operations: Propose, Debate, Amend, Vote, Enact, Budget, Audit, Recall,
// Delegate-Up, Transparency.
// Named functions: Policy-Cycle, Power-Map, Accountability-Chain.
func TestMeaningGrammar(t *testing.T) {
	t.Run("ProposeAndDebate", func(t *testing.T) {
		env := newTestEnv(t)
		councillor := env.actor("Councillor", 1, event.ActorTypeHuman)
		citizen := env.actor("Citizen", 2, event.ActorTypeHuman)

		proposal, _ := env.grammar.Emit(env.ctx, councillor.ID(),
			"proposal: allocate 20% of budget to open-source infrastructure",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		debate1, _ := env.grammar.Respond(env.ctx, citizen.ID(),
			"support: open-source reduces vendor lock-in, long-term savings",
			proposal.ID(), env.convID, signer)

		debate2, _ := env.grammar.Respond(env.ctx, councillor.ID(),
			"counter: need to balance with maintenance costs of self-hosted solutions",
			debate1.ID(), env.convID, signer)

		ancestors := env.ancestors(debate2.ID(), 10)
		if !containsEvent(ancestors, proposal.ID()) {
			t.Error("debate should trace to proposal")
		}
		env.verifyChain()
	})

	t.Run("AmendAndVote", func(t *testing.T) {
		env := newTestEnv(t)
		author := env.actor("Author", 1, event.ActorTypeHuman)
		voter := env.actor("Voter", 2, event.ActorTypeHuman)

		proposal, _ := env.grammar.Emit(env.ctx, author.ID(),
			"proposal: mandatory code review for all PRs",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		amendment, _ := env.grammar.Derive(env.ctx, voter.ID(),
			"amendment: mandatory code review for PRs >50 lines, optional for smaller",
			proposal.ID(), env.convID, signer)

		vote, _ := env.grammar.Consent(env.ctx, author.ID(), voter.ID(),
			"vote: approve amended proposal",
			types.MustDomainScope("governance"),
			amendment.ID(), env.convID, signer)

		ancestors := env.ancestors(vote.ID(), 10)
		if !containsEvent(ancestors, proposal.ID()) {
			t.Error("vote should trace to original proposal")
		}
		env.verifyChain()
	})

	t.Run("Enact", func(t *testing.T) {
		env := newTestEnv(t)
		governor := env.actor("Governor", 1, event.ActorTypeHuman)

		proposal, _ := env.grammar.Emit(env.ctx, governor.ID(),
			"proposal: weekly standup for all teams",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		vote, _ := env.grammar.Annotate(env.ctx, env.system,
			proposal.ID(), "vote_result",
			"approved: 8 for, 2 against, 1 abstain",
			env.convID, signer)

		enacted, _ := env.grammar.Derive(env.ctx, governor.ID(),
			"enacted: weekly standup policy effective immediately",
			vote.ID(), env.convID, signer)

		ancestors := env.ancestors(enacted.ID(), 10)
		if !containsEvent(ancestors, proposal.ID()) {
			t.Error("enacted policy should trace to proposal")
		}
		env.verifyChain()
	})

	t.Run("BudgetTransparency", func(t *testing.T) {
		env := newTestEnv(t)
		treasurer := env.actor("Treasurer", 1, event.ActorTypeHuman)
		auditor := env.actor("Auditor", 2, event.ActorTypeAI)

		allocation, _ := env.grammar.Emit(env.ctx, treasurer.ID(),
			"budget allocation: $50k infrastructure, $30k training, $20k tools",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		spend1, _ := env.grammar.Derive(env.ctx, treasurer.ID(),
			"expenditure: $12k cloud hosting (infrastructure budget)",
			allocation.ID(), env.convID, signer)

		spend2, _ := env.grammar.Derive(env.ctx, treasurer.ID(),
			"expenditure: $8k conference attendance (training budget)",
			allocation.ID(), env.convID, signer)

		audit, _ := env.grammar.Merge(env.ctx, auditor.ID(),
			"budget audit: $20k spent of $100k allocated, all expenditures traced to allocations",
			[]types.EventID{spend1.ID(), spend2.ID()}, env.convID, signer)

		ancestors := env.ancestors(audit.ID(), 10)
		if !containsEvent(ancestors, allocation.ID()) {
			t.Error("audit should trace to allocation")
		}
		env.verifyChain()
	})

	t.Run("Recall", func(t *testing.T) {
		env := newTestEnv(t)
		community := env.actor("Community", 1, event.ActorTypeCommittee)
		governor := env.actor("Governor", 2, event.ActorTypeHuman)

		delegation, _ := env.grammar.Delegate(env.ctx, env.system, governor.ID(),
			types.MustDomainScope("governance"), types.MustWeight(0.9),
			env.boot.ID(), env.convID, signer)

		violation, _ := env.graph.Record(
			event.EventTypeViolationDetected, community.ID(),
			event.ViolationDetectedContent{
				Expectation: env.boot.ID(),
				Actor:       governor.ID(),
				Severity:    event.SeverityLevelCritical,
				Description: "misuse of governance authority for personal benefit",
				Evidence:    types.MustNonEmpty([]types.EventID{delegation.ID()}),
			},
			[]types.EventID{delegation.ID()}, env.convID, signer)

		recall, _ := env.grammar.Derive(env.ctx, community.ID(),
			"recall: governor removed from authority, community vote 85% in favour",
			violation.ID(), env.convID, signer)

		ancestors := env.ancestors(recall.ID(), 10)
		if !containsEvent(ancestors, violation.ID()) {
			t.Error("recall should trace to violation")
		}
		env.verifyChain()
	})

	t.Run("PolicyCycle", func(t *testing.T) {
		env := newTestEnv(t)
		lead := env.actor("Lead", 1, event.ActorTypeHuman)
		member := env.actor("Member", 2, event.ActorTypeHuman)

		// Propose → Debate → Amend → Vote → Enact → Measure → Iterate
		proposal, _ := env.grammar.Emit(env.ctx, lead.ID(),
			"proposal: deploy to production only on Tuesdays and Thursdays",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		debate, _ := env.grammar.Respond(env.ctx, member.ID(),
			"concern: too restrictive, blocks urgent fixes",
			proposal.ID(), env.convID, signer)

		amended, _ := env.grammar.Derive(env.ctx, lead.ID(),
			"amendment: deploy Tue/Thu for features, any day for hotfixes",
			debate.ID(), env.convID, signer)

		enacted, _ := env.grammar.Derive(env.ctx, lead.ID(),
			"enacted: amended deploy policy",
			amended.ID(), env.convID, signer)

		measurement, _ := env.grammar.Annotate(env.ctx, env.system,
			enacted.ID(), "effectiveness",
			"30 days: 40% fewer incidents, 2 hotfix deploys needed",
			env.convID, signer)

		iteration, _ := env.grammar.Derive(env.ctx, lead.ID(),
			"iteration: add Wednesday deploy window based on measurement data",
			measurement.ID(), env.convID, signer)

		ancestors := env.ancestors(iteration.ID(), 15)
		if !containsEvent(ancestors, proposal.ID()) {
			t.Error("iteration should trace to original proposal")
		}
		env.verifyChain()
	})

	t.Run("AccountabilityChain", func(t *testing.T) {
		env := newTestEnv(t)
		governor := env.actor("Governor", 1, event.ActorTypeHuman)
		affected := env.actor("Affected", 2, event.ActorTypeHuman)
		auditor := env.actor("Auditor", 3, event.ActorTypeAI)

		decision, _ := env.grammar.Emit(env.ctx, governor.ID(),
			"decision: cut training budget by 50%",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		impact, _ := env.grammar.Respond(env.ctx, affected.ID(),
			"impact: team unable to attend critical security training",
			decision.ID(), env.convID, signer)

		chain, _ := env.grammar.Derive(env.ctx, auditor.ID(),
			"accountability: decision by Governor → training cut → security skill gap → increased risk",
			impact.ID(), env.convID, signer)

		ancestors := env.ancestors(chain.ID(), 10)
		if !containsEvent(ancestors, decision.ID()) {
			t.Error("accountability chain should trace to decision")
		}
		env.verifyChain()
	})
}
