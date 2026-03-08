package integration_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/compositions"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestScenario18_WhistleblowAndRecall exercises accountability across layers:
// Knowledge fact-check reveals systematic bias → alignment whistleblow →
// justice class action → recall of responsible authority → community renewal.
// Crosses Knowledge, Alignment, Justice, and Belonging grammars.
func TestScenario18_WhistleblowAndRecall(t *testing.T) {
	env := newTestEnv(t)
	knowledge := compositions.NewKnowledgeGrammar(env.grammar)
	alignment := compositions.NewAlignmentGrammar(env.grammar)
	justice := compositions.NewJusticeGrammar(env.grammar)
	belonging := compositions.NewBelongingGrammar(env.grammar)

	auditor := env.registerActor("Auditor", 1, event.ActorTypeAI)
	official := env.registerActor("DataOfficer", 2, event.ActorTypeHuman)
	affected1 := env.registerActor("Affected1", 3, event.ActorTypeHuman)
	affected2 := env.registerActor("Affected2", 4, event.ActorTypeHuman)
	community := env.registerActor("Community", 5, event.ActorTypeCommittee)

	// 1. Fact-check reveals problematic claims in official's reports
	factCheck, err := knowledge.FactCheck(env.ctx, auditor.ID(),
		env.boot.ID(),
		"source: internal metrics dashboard, last updated 3 months ago",
		"systematic bias: reports exclude negative outcomes for preferred vendors",
		"claims are selectively accurate — omission bias confirmed",
		env.convID, signer)
	if err != nil {
		t.Fatalf("FactCheck: %v", err)
	}

	// 2. Guardrail triggered — this violates transparency constraints
	guardrail, err := alignment.Guardrail(env.ctx, auditor.ID(),
		factCheck.Verdict.ID(),
		"transparency: all material outcomes must be reported",
		"reporting accuracy vs organizational reputation",
		"escalating to external oversight — internal resolution insufficient",
		env.convID, signer)
	if err != nil {
		t.Fatalf("Guardrail: %v", err)
	}

	// 3. Whistleblow — escalate to external authority
	whistle, err := alignment.Whistleblow(env.ctx, auditor.ID(),
		"systematic omission of negative vendor outcomes in official reports",
		"3 months of reports exclude 40% of negative outcomes, affecting procurement decisions",
		"external audit required — internal reporting chain compromised",
		[]types.EventID{guardrail.Escalation.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Whistleblow: %v", err)
	}

	// 4. Class action — multiple affected parties file
	classAction, err := justice.ClassAction(env.ctx,
		[]types.ActorID{affected1.ID(), affected2.ID()},
		official.ID(), auditor.ID(),
		[]string{
			"procurement decisions based on incomplete data cost us $50k",
			"vendor selection biased — our proposals evaluated against cherry-picked benchmarks",
		},
		"fact-check proves systematic omission", "omission bias affected all procurement",
		"reports were optimized for speed, not completeness", "no intent to deceive",
		"official failed duty of care — incomplete reporting caused material harm",
		whistle.Escalation.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("ClassAction: %v", err)
	}

	// 5. Recall — community removes the official from authority
	recall, err := justice.Recall(env.ctx, auditor.ID(), community.ID(), official.ID(),
		"systematic omission in 3 months of reports, confirmed by fact-check and class action",
		"data officer violated transparency obligations",
		types.MustDomainScope("data_governance"),
		classAction.Trial.Ruling.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}

	// 6. Community renewal after the crisis
	renewal, err := belonging.Renewal(env.ctx, community.ID(),
		"trust damaged but recoverable — new reporting standards needed",
		"mandatory dual-review of all vendor reports before publication",
		"the community that learned transparency cannot be optional",
		[]types.EventID{recall.Revocation.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Renewal: %v", err)
	}

	// --- Assertions ---

	// Renewal traces all the way back to fact-check
	renewalAncestors := env.ancestors(renewal.Story.ID(), 30)
	if !containsEvent(renewalAncestors, factCheck.Verdict.ID()) {
		t.Error("renewal should trace to original fact-check")
	}

	// Recall traces through class action to whistleblow
	recallAncestors := env.ancestors(recall.Revocation.ID(), 25)
	if !containsEvent(recallAncestors, whistle.Harm.ID()) {
		t.Error("recall should trace to whistleblow harm detection")
	}

	// Class action traces through whistleblow to guardrail
	classAncestors := env.ancestors(classAction.Trial.Ruling.ID(), 25)
	if !containsEvent(classAncestors, guardrail.Constraint.ID()) {
		t.Error("class action should trace to guardrail constraint")
	}

	env.verifyChain()
}
