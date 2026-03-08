package integration_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/compositions"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestScenario14_SprintLifecycle exercises a complete development cycle:
// Sprint planning → daily standups → spike experiment → pipeline deployment →
// retrospective → tech debt tracking. Crosses Work, Build, and Knowledge grammars.
func TestScenario14_SprintLifecycle(t *testing.T) {
	env := newTestEnv(t)
	work := compositions.NewWorkGrammar(env.grammar)
	build := compositions.NewBuildGrammar(env.grammar)
	knowledge := compositions.NewKnowledgeGrammar(env.grammar)

	lead := env.registerActor("TechLead", 1, event.ActorTypeHuman)
	alice := env.registerActor("Alice", 2, event.ActorTypeHuman)
	bob := env.registerActor("Bob", 3, event.ActorTypeHuman)
	ci := env.registerActor("CI", 4, event.ActorTypeAI)

	// 1. Sprint planning — decompose goal into tasks and assign
	sprint, err := work.Sprint(env.ctx, lead.ID(), "Sprint 12: search feature",
		[]string{"build search index", "add fuzzy matching"},
		[]types.ActorID{alice.ID(), bob.ID()},
		[]types.DomainScope{types.MustDomainScope("search_index"), types.MustDomainScope("fuzzy_matching")},
		[]types.EventID{env.boot.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Sprint: %v", err)
	}

	// 2. Day 1 standup — both devs report progress
	standup1, err := work.Standup(env.ctx,
		[]types.ActorID{alice.ID(), bob.ID()},
		[]string{"schema designed, starting implementation", "researching fuzzy algorithms"},
		lead.ID(), "search index is critical path",
		[]types.EventID{sprint.Intent.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Standup: %v", err)
	}

	// 3. Bob runs a spike to evaluate fuzzy matching libraries
	spike, err := build.Spike(env.ctx, bob.ID(),
		"evaluate Levenshtein vs trigram for fuzzy matching",
		"trigram: 2ms avg, Levenshtein: 8ms avg, both >95% accuracy",
		"trigram is 4x faster with comparable accuracy",
		"adopt trigram approach",
		[]types.EventID{standup1.Priority.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Spike: %v", err)
	}

	// 4. Record the spike finding as verified knowledge
	verified, err := knowledge.Verify(env.ctx, bob.ID(),
		"trigram matching is 4x faster than Levenshtein with >95% accuracy",
		"benchmarked on 10k document corpus with real queries",
		"consistent with published research on approximate string matching",
		[]types.EventID{spike.Decision.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	// 5. Alice completes the search index, run CI pipeline
	pipeline, err := build.Pipeline(env.ctx, ci.ID(),
		"search index build + deploy",
		"all 47 tests pass, coverage 91%",
		"latency p99=12ms, memory=240MB",
		"deployed to staging",
		[]types.EventID{verified.Corroboration.ID()}, env.convID, signer)
	if err != nil {
		t.Fatalf("Pipeline: %v", err)
	}

	// 6. Sprint retrospective — both devs review, lead sets improvement goal
	retro, err := work.Retrospective(env.ctx,
		[]types.ActorID{alice.ID(), bob.ID()},
		[]string{
			"search index shipped on time, spike approach saved 3 days",
			"fuzzy matching integrated cleanly, trigram decision validated",
		},
		lead.ID(), "adopt spike-first approach for all algorithm decisions",
		sprint.Intent.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("Retrospective: %v", err)
	}

	// 7. Tech debt identified during retrospective
	techDebt, err := build.TechDebt(env.ctx, lead.ID(),
		pipeline.Deployment.ID(),
		"search index lacks pagination, will hit memory limits at >100k docs",
		"add cursor-based pagination to search results",
		"schedule for Sprint 13",
		env.convID, signer)
	if err != nil {
		t.Fatalf("TechDebt: %v", err)
	}

	// --- Assertions ---

	// Spike decision traces back to sprint planning
	spikeAncestors := env.ancestors(spike.Decision.ID(), 15)
	if !containsEvent(spikeAncestors, sprint.Intent.ID()) {
		t.Error("spike decision should trace to sprint intent")
	}

	// Pipeline traces through verified knowledge
	pipelineAncestors := env.ancestors(pipeline.Deployment.ID(), 20)
	if !containsEvent(pipelineAncestors, verified.Claim.ID()) {
		t.Error("pipeline should trace to verified knowledge claim")
	}

	// Retrospective improvement traces to original sprint
	retroAncestors := env.ancestors(retro.Improvement.ID(), 15)
	if !containsEvent(retroAncestors, sprint.Intent.ID()) {
		t.Error("retrospective improvement should trace to sprint intent")
	}

	// Tech debt traces to deployment
	debtAncestors := env.ancestors(techDebt.Iteration.ID(), 10)
	if !containsEvent(debtAncestors, pipeline.Deployment.ID()) {
		t.Error("tech debt should trace to deployment")
	}

	env.verifyChain()

	// Sprint(1 intent + 2 subtasks + 2 assignments) + Standup(2 progress + 1 priority) +
	// Spike(4) + Verify(3) + Pipeline(4) + Retrospective(2 reviews + 1 improvement) +
	// TechDebt(3) + bootstrap(1) = 26
	if count := env.eventCount(); count != 26 {
		t.Errorf("event count = %d, want 26", count)
	}
}
