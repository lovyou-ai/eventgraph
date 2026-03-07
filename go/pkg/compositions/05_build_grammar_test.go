package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestBuildGrammar exercises the Build Grammar (Layer 5: Technology).
// Operations: Build, Version, Ship, Sunset, Define, Automate, Test, Review,
// Measure, Feedback, Iterate, Innovate.
// Named functions: Pipeline, Post-Mortem.
func TestBuildGrammar(t *testing.T) {
	t.Run("BuildAndVersion", func(t *testing.T) {
		env := newTestEnv(t)
		dev := env.actor("Developer", 1, event.ActorTypeHuman)

		v1, _ := env.grammar.Emit(env.ctx, dev.ID(),
			"artefact created: eventgraph-cli v1.0.0",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		v2, _ := env.grammar.Derive(env.ctx, dev.ID(),
			"artefact version: eventgraph-cli v1.1.0 — added JSON output",
			v1.ID(), env.convID, signer)

		v3, _ := env.grammar.Derive(env.ctx, dev.ID(),
			"artefact version: eventgraph-cli v2.0.0 — breaking: new config format",
			v2.ID(), env.convID, signer)

		// Version chain: v3 → v2 → v1
		ancestors := env.ancestors(v3.ID(), 10)
		if !containsEvent(ancestors, v1.ID()) {
			t.Error("v3 should trace to v1")
		}
		if !containsEvent(ancestors, v2.ID()) {
			t.Error("v3 should trace to v2")
		}
		env.verifyChain()
	})

	t.Run("ShipAndSunset", func(t *testing.T) {
		env := newTestEnv(t)
		dev := env.actor("Developer", 1, event.ActorTypeHuman)

		artefact, _ := env.grammar.Emit(env.ctx, dev.ID(),
			"artefact: auth-lib v1.0",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		shipped, _ := env.grammar.Annotate(env.ctx, dev.ID(),
			artefact.ID(), "status", "shipped: available in package registry",
			env.convID, signer)

		replacement, _ := env.grammar.Derive(env.ctx, dev.ID(),
			"artefact: auth-lib-v2 v2.0 — replacement for auth-lib",
			artefact.ID(), env.convID, signer)

		sunset, _ := env.grammar.Annotate(env.ctx, dev.ID(),
			artefact.ID(), "deprecated",
			"sunset: replaced by auth-lib-v2, removal date 2026-09-01",
			env.convID, signer)

		_ = shipped
		_ = sunset
		_ = replacement
		env.verifyChain()
	})

	t.Run("TestAndReview", func(t *testing.T) {
		env := newTestEnv(t)
		dev := env.actor("Developer", 1, event.ActorTypeHuman)
		reviewer := env.actor("Reviewer", 2, event.ActorTypeHuman)

		code, _ := env.grammar.Emit(env.ctx, dev.ID(),
			"code: auth module implementation",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		testResult, _ := env.grammar.Derive(env.ctx, dev.ID(),
			"test results: 45/45 passing, coverage 91%, no regressions",
			code.ID(), env.convID, signer)

		review, _ := env.grammar.Respond(env.ctx, reviewer.ID(),
			"review: code quality good, tests comprehensive, approved",
			testResult.ID(), env.convID, signer)

		ancestors := env.ancestors(review.ID(), 10)
		if !containsEvent(ancestors, code.ID()) {
			t.Error("review should trace to code")
		}
		env.verifyChain()
	})

	t.Run("FeedbackAndIterate", func(t *testing.T) {
		env := newTestEnv(t)
		dev := env.actor("Developer", 1, event.ActorTypeHuman)
		user := env.actor("User", 3, event.ActorTypeHuman)

		v1, _ := env.grammar.Emit(env.ctx, dev.ID(),
			"shipped: CLI tool v1.0",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		feedback, _ := env.grammar.Respond(env.ctx, user.ID(),
			"feedback: output is hard to read, needs colour coding and table format",
			v1.ID(), env.convID, signer)

		v2, _ := env.grammar.Derive(env.ctx, dev.ID(),
			"iterated: CLI tool v1.1 — added colour output and table format",
			feedback.ID(), env.convID, signer)

		ancestors := env.ancestors(v2.ID(), 10)
		if !containsEvent(ancestors, feedback.ID()) {
			t.Error("iteration should trace to feedback")
		}
		if !containsEvent(ancestors, v1.ID()) {
			t.Error("iteration should trace to v1")
		}
		env.verifyChain()
	})

	t.Run("Pipeline", func(t *testing.T) {
		env := newTestEnv(t)
		ci := env.actor("CI", 1, event.ActorTypeAI)

		commit, _ := env.grammar.Emit(env.ctx, ci.ID(),
			"commit: abc123 pushed to main",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		build, _ := env.grammar.Derive(env.ctx, ci.ID(),
			"build: compiled successfully, 0 warnings",
			commit.ID(), env.convID, signer)
		test, _ := env.grammar.Derive(env.ctx, ci.ID(),
			"test: 234/234 passing, coverage 88%",
			build.ID(), env.convID, signer)
		lint, _ := env.grammar.Derive(env.ctx, ci.ID(),
			"lint: 0 issues found",
			test.ID(), env.convID, signer)
		deploy, _ := env.grammar.Derive(env.ctx, ci.ID(),
			"deploy: staging deployment successful",
			lint.ID(), env.convID, signer)

		// Pipeline chain: deploy → lint → test → build → commit
		ancestors := env.ancestors(deploy.ID(), 10)
		if !containsEvent(ancestors, commit.ID()) {
			t.Error("deploy should trace to commit")
		}
		env.verifyChain()
	})

	t.Run("PostMortem", func(t *testing.T) {
		env := newTestEnv(t)
		lead := env.actor("Lead", 1, event.ActorTypeHuman)
		eng1 := env.actor("Eng1", 2, event.ActorTypeHuman)
		eng2 := env.actor("Eng2", 3, event.ActorTypeHuman)

		incident, _ := env.grammar.Emit(env.ctx, env.system,
			"incident: 45-minute production outage due to database connection pool exhaustion",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		fb1, _ := env.grammar.Respond(env.ctx, eng1.ID(),
			"feedback: connection pool was set to default 10, needs 50+ for our load",
			incident.ID(), env.convID, signer)
		fb2, _ := env.grammar.Respond(env.ctx, eng2.ID(),
			"feedback: monitoring didn't alert until connections were fully exhausted",
			incident.ID(), env.convID, signer)

		analysis, _ := env.grammar.Merge(env.ctx, lead.ID(),
			"post-mortem: root cause was under-provisioned connection pool + late alerting",
			[]types.EventID{fb1.ID(), fb2.ID()}, env.convID, signer)

		actions, _ := env.grammar.Derive(env.ctx, lead.ID(),
			"improvement actions: 1) increase pool to 100 2) add connection utilisation alert at 80%",
			analysis.ID(), env.convID, signer)

		ancestors := env.ancestors(actions.ID(), 10)
		if !containsEvent(ancestors, incident.ID()) {
			t.Error("actions should trace to incident")
		}
		env.verifyChain()
	})
}
