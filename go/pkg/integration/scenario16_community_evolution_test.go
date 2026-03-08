package integration_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/compositions"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestScenario16_CommunityEvolution exercises a community's lifecycle:
// Onboard newcomers → establish commons governance → celebrate festival →
// detect growing pains → phase transition → renewal.
// Crosses Belonging, Social, and Evolution grammars.
func TestScenario16_CommunityEvolution(t *testing.T) {
	env := newTestEnv(t)
	belonging := compositions.NewBelongingGrammar(env.grammar)
	social := compositions.NewSocialGrammar(env.grammar)
	evolution := compositions.NewEvolutionGrammar(env.grammar)

	founder := env.registerActor("Founder", 1, event.ActorTypeHuman)
	steward := env.registerActor("Steward", 2, event.ActorTypeHuman)
	newcomer := env.registerActor("Newcomer", 3, event.ActorTypeHuman)
	community := env.registerActor("Community", 4, event.ActorTypeCommittee)

	// 1. Onboard a newcomer
	onboard, err := belonging.Onboard(env.ctx, founder.ID(), newcomer.ID(), community.ID(),
		types.Some(types.MustDomainScope("general")),
		"opened registration for newcomer",
		"attended welcome ceremony",
		"first documentation contribution",
		env.boot.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("Onboard: %v", err)
	}

	// 2. Establish commons governance — shared resources managed
	commons, err := belonging.CommonsGovernance(env.ctx, founder.ID(), steward.ID(),
		types.MustDomainScope("shared_resources"), types.MustWeight(0.7),
		"resources sustainable at current usage levels",
		"shared resources require 2/3 vote for allocation changes",
		"initial audit: 3 resource pools, all within capacity",
		onboard.Contribution.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("CommonsGovernance: %v", err)
	}

	// 3. Community holds a festival to celebrate growth
	festival, err := belonging.Festival(env.ctx, founder.ID(),
		"community reached 50 members milestone",
		"annual review ceremony",
		"from 3 founders to 50 members in 8 months",
		"open-source toolkit for new communities",
		[]types.EventID{commons.Audit.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Festival: %v", err)
	}

	// 4. Establish a community norm through polling
	poll, err := social.Poll(env.ctx, founder.ID(),
		"should we adopt weekly async standups?",
		[]types.ActorID{steward.ID(), newcomer.ID()},
		types.MustDomainScope("process"),
		festival.Gift.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}

	// 5. Evolution detects growing pains — phase transition needed
	transition, err := evolution.PhaseTransition(env.ctx, env.system,
		poll.Proposal.ID(),
		"community size crossed 50 — informal coordination breaking down",
		"current flat structure creates 1225 communication pairs",
		"introduce working groups with elected leads",
		"working groups reduce coordination pairs by 80%",
		env.convID, signer)
	if err != nil {
		t.Fatalf("PhaseTransition: %v", err)
	}

	// 6. Community renewal based on evolution insights
	renewal, err := belonging.Renewal(env.ctx, founder.ID(),
		"structure evolved: flat → working groups, coordination improved",
		"weekly working group sync replaces ad-hoc coordination",
		"chapter 2: the community that learned to scale",
		[]types.EventID{transition.Selection.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Renewal: %v", err)
	}

	// --- Assertions ---

	// Renewal traces all the way back to onboarding
	renewalAncestors := env.ancestors(renewal.Story.ID(), 30)
	if !containsEvent(renewalAncestors, onboard.Contribution.ID()) {
		t.Error("renewal should trace to original onboarding")
	}

	// Phase transition traces to community poll
	transitionAncestors := env.ancestors(transition.Selection.ID(), 15)
	if !containsEvent(transitionAncestors, poll.Proposal.ID()) {
		t.Error("phase transition should trace to poll proposal")
	}

	// Festival traces to commons governance
	festivalAncestors := env.ancestors(festival.Gift.ID(), 15)
	if !containsEvent(festivalAncestors, commons.Audit.ID()) {
		t.Error("festival should trace to commons audit")
	}

	// Commons governance traces to onboarding
	commonsAncestors := env.ancestors(commons.Audit.ID(), 15)
	if !containsEvent(commonsAncestors, onboard.Contribution.ID()) {
		t.Error("commons governance should trace to onboarding contribution")
	}

	env.verifyChain()
}
