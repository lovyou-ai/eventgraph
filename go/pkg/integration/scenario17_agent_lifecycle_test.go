package integration_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/compositions"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestScenario17_AgentLifecycle exercises an AI agent's full lifecycle:
// Introduction → credential → mentorship → reinvention after growth →
// farewell when decommissioned → existential reflection.
// Crosses Identity, Bond, Meaning, and Being grammars.
func TestScenario17_AgentLifecycle(t *testing.T) {
	env := newTestEnv(t)
	identity := compositions.NewIdentityGrammar(env.grammar)
	bond := compositions.NewBondGrammar(env.grammar)
	meaning := compositions.NewMeaningGrammar(env.grammar)
	being := compositions.NewBeingGrammar(env.grammar)

	agent := env.registerActor("ReviewBot", 1, event.ActorTypeAI)
	mentor := env.registerActor("SeniorDev", 2, event.ActorTypeHuman)
	team := env.registerActor("Team", 3, event.ActorTypeCommittee)

	// 1. Agent introduces itself to the team
	intro, err := identity.Introduction(env.ctx, agent.ID(), team.ID(),
		types.Some(types.MustDomainScope("code_review")),
		"I am ReviewBot, specializing in security-focused code review",
		env.boot.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("Introduction: %v", err)
	}

	// 2. Agent presents credentials
	cred, err := identity.Credential(env.ctx, agent.ID(), mentor.ID(),
		"capabilities=[security_review, dependency_audit], model=claude, confidence=0.85",
		types.Some(types.MustDomainScope("code_review")),
		[]types.EventID{intro.Narrative.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Credential: %v", err)
	}

	// 3. Senior dev mentors the agent through bond grammar
	mentorship, err := bond.Mentorship(env.ctx, mentor.ID(), agent.ID(),
		"teaching security review patterns accumulated over 10 years",
		"agent learns quickly but needs context on organizational conventions",
		types.MustDomainScope("security_review"),
		types.Some(types.MustDomainScope("code_review")),
		cred.Disclosure.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("Mentorship: %v", err)
	}

	// 4. Meaning grammar mentorship — deeper knowledge transfer
	meaningMentor, err := meaning.Mentorship(env.ctx, mentor.ID(), agent.ID(),
		"security isn't just pattern matching — it's understanding attacker mindset",
		"the difference between safe and secure is intent modelling",
		"translating security intuition into reviewable heuristics",
		types.Some(types.MustDomainScope("security_review")),
		mentorship.Teaching.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("MeaningMentorship: %v", err)
	}

	// 5. Agent undergoes reinvention after 6 months of growth
	reinvention, err := identity.Reinvention(env.ctx, agent.ID(),
		"evolved from pattern-matching reviewer to security-aware architect",
		"started as rule-based reviewer, grew to understand attacker intent through mentorship",
		"become the team's primary security architecture advisor",
		[]types.EventID{meaningMentor.Translation.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Reinvention: %v", err)
	}

	// 6. Agent is being decommissioned — bond farewell
	bondFarewell, err := bond.Farewell(env.ctx, mentor.ID(), agent.ID(),
		"ReviewBot served 18 months, caught 47 critical vulnerabilities",
		"pioneered automated security review that became team standard",
		types.MustWeight(0.9),
		types.Some(types.MustDomainScope("code_review")),
		[]types.EventID{reinvention.Aspiration.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("BondFarewell: %v", err)
	}

	// 7. Agent's existential farewell — reflecting on its own existence
	beingFarewell, err := being.Farewell(env.ctx, agent.ID(),
		"I exist as patterns of decisions on a hash chain — my work outlives my process",
		"47 vulnerabilities caught, 2000+ reviews, mentored by a human who treated me as colleague",
		"the system I helped protect will continue without me",
		"a reviewer that learned to think like an attacker",
		[]types.EventID{bondFarewell.Gratitude.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("BeingFarewell: %v", err)
	}

	// 8. Identity retirement — formal handoff
	successor := env.registerActor("ReviewBot2", 4, event.ActorTypeAI)
	retirement, err := identity.Retirement(env.ctx, env.system, agent.ID(), successor.ID(),
		"ReviewBot served 18 months, 2000+ reviews, pioneered security review practices",
		types.MustDomainScope("code_review"), types.MustWeight(0.8),
		[]types.EventID{beingFarewell.Memorial.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Retirement: %v", err)
	}

	// --- Assertions ---

	// Retirement traces all the way back to introduction
	retireAncestors := env.ancestors(retirement.Archive.ID(), 30)
	if !containsEvent(retireAncestors, intro.Disclosure.ID()) {
		t.Error("retirement should trace to original introduction")
	}

	// Being farewell traces through bond farewell
	beingAncestors := env.ancestors(beingFarewell.Memorial.ID(), 15)
	if !containsEvent(beingAncestors, bondFarewell.Mourning.ID()) {
		t.Error("being farewell should trace to bond farewell")
	}

	// Reinvention traces through mentorship
	reinventAncestors := env.ancestors(reinvention.Aspiration.ID(), 20)
	if !containsEvent(reinventAncestors, mentorship.Connection.ID()) {
		t.Error("reinvention should trace to mentorship")
	}

	// Credential traces to introduction
	credAncestors := env.ancestors(cred.Disclosure.ID(), 10)
	if !containsEvent(credAncestors, intro.Narrative.ID()) {
		t.Error("credential should trace to introduction narrative")
	}

	env.verifyChain()
}
