package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestWorkGrammar exercises the Work Grammar (Layer 1: Agency).
// Operations: Intend, Decompose, Assign, Claim, Prioritize, Block/Unblock,
// Progress, Complete, Handoff, Scope, Review.
// Named functions: Sprint, Delegate-and-Verify, Escalate.
func TestWorkGrammar(t *testing.T) {
	t.Run("Intend", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)

		goal, err := env.grammar.Emit(env.ctx, alice.ID(),
			"goal: increase test coverage to 90% across all packages",
			env.convID, []types.EventID{env.boot.ID()}, signer)
		if err != nil {
			t.Fatalf("Emit goal: %v", err)
		}
		if goal.Type().Value() != "grammar.emit" {
			t.Errorf("type = %s, want grammar.emit", goal.Type().Value())
		}
		if goal.Source() != alice.ID() {
			t.Error("goal source should be Alice")
		}
	})

	t.Run("Decompose", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)

		goal, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"goal: ship v2.0", env.convID, []types.EventID{env.boot.ID()}, signer)

		plan, _ := env.grammar.Derive(env.ctx, alice.ID(),
			"plan: 3 steps required", goal.ID(), env.convID, signer)

		step1, _ := env.grammar.Extend(env.ctx, alice.ID(),
			"step 1: implement auth module", plan.ID(), env.convID, signer)
		step2, _ := env.grammar.Extend(env.ctx, alice.ID(),
			"step 2: add API endpoints", step1.ID(), env.convID, signer)
		step3, _ := env.grammar.Extend(env.ctx, alice.ID(),
			"step 3: write integration tests", step2.ID(), env.convID, signer)

		// Steps chain back to goal
		ancestors := env.ancestors(step3.ID(), 10)
		if !containsEvent(ancestors, goal.ID()) {
			t.Error("step3 should trace back to goal")
		}
		env.verifyChain()
	})

	t.Run("Assign", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		task, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"task: implement auth module", env.convID, []types.EventID{env.boot.ID()}, signer)

		delegation, err := env.grammar.Delegate(env.ctx, alice.ID(), bob.ID(),
			types.MustDomainScope("auth_module"), types.MustWeight(0.8),
			task.ID(), env.convID, signer)
		if err != nil {
			t.Fatalf("Delegate: %v", err)
		}

		content := delegation.Content().(event.EdgeCreatedContent)
		if content.EdgeType != event.EdgeTypeDelegation {
			t.Errorf("edge type = %s, want delegation", content.EdgeType)
		}
		if content.To != bob.ID() {
			t.Error("delegation target should be Bob")
		}
		env.verifyChain()
	})

	t.Run("Claim", func(t *testing.T) {
		env := newTestEnv(t)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		// Unassigned task on the board
		task, _ := env.grammar.Emit(env.ctx, env.system,
			"task: fix login bug (unassigned)", env.convID, []types.EventID{env.boot.ID()}, signer)

		// Bob claims it (initiative)
		claim, err := env.grammar.Derive(env.ctx, bob.ID(),
			"claimed: taking ownership of login bug fix",
			task.ID(), env.convID, signer)
		if err != nil {
			t.Fatalf("Derive claim: %v", err)
		}

		ancestors := env.ancestors(claim.ID(), 5)
		if !containsEvent(ancestors, task.ID()) {
			t.Error("claim should trace to task")
		}
	})

	t.Run("BlockAndUnblock", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)

		task, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"task: deploy service", env.convID, []types.EventID{env.boot.ID()}, signer)

		block, _ := env.grammar.Annotate(env.ctx, alice.ID(),
			task.ID(), "blocked", "waiting for DNS propagation",
			env.convID, signer)

		unblock, _ := env.grammar.Annotate(env.ctx, alice.ID(),
			task.ID(), "unblocked", "DNS propagated, ready to proceed",
			env.convID, signer)

		// Both annotations reference the task
		blockAnc := env.ancestors(block.ID(), 5)
		if !containsEvent(blockAnc, task.ID()) {
			t.Error("block should trace to task")
		}
		unblockAnc := env.ancestors(unblock.ID(), 5)
		if !containsEvent(unblockAnc, task.ID()) {
			t.Error("unblock should trace to task")
		}
		env.verifyChain()
	})

	t.Run("ProgressAndComplete", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)

		task, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"task: write tests", env.convID, []types.EventID{env.boot.ID()}, signer)

		p1, _ := env.grammar.Extend(env.ctx, alice.ID(),
			"progress: 5/15 tests written (33%)", task.ID(), env.convID, signer)
		p2, _ := env.grammar.Extend(env.ctx, alice.ID(),
			"progress: 15/15 tests written (100%)", p1.ID(), env.convID, signer)

		complete, _ := env.grammar.Derive(env.ctx, alice.ID(),
			"complete: all 15 tests passing, coverage 92%", p2.ID(), env.convID, signer)

		ancestors := env.ancestors(complete.ID(), 10)
		if !containsEvent(ancestors, task.ID()) {
			t.Error("completion should trace to original task")
		}
		env.verifyChain()
	})

	t.Run("Handoff", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		carol := env.actor("Carol", 3, event.ActorTypeHuman)

		task, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"task: maintain auth module", env.convID, []types.EventID{env.boot.ID()}, signer)

		handoff, err := env.grammar.Consent(env.ctx, alice.ID(), carol.ID(),
			"handoff: Carol takes ownership of auth module maintenance",
			types.MustDomainScope("auth_module"),
			task.ID(), env.convID, signer)
		if err != nil {
			t.Fatalf("Consent handoff: %v", err)
		}

		content := handoff.Content().(event.GrammarConsentContent)
		if content.Agreement == "" {
			t.Error("handoff should have agreement text")
		}
		env.verifyChain()
	})

	t.Run("Review", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		work, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"completed: auth module implementation", env.convID, []types.EventID{env.boot.ID()}, signer)

		review, err := env.grammar.Respond(env.ctx, bob.ID(),
			"review: code quality good, tests comprehensive, approved",
			work.ID(), env.convID, signer)
		if err != nil {
			t.Fatalf("Respond review: %v", err)
		}

		ancestors := env.ancestors(review.ID(), 5)
		if !containsEvent(ancestors, work.ID()) {
			t.Error("review should trace to work")
		}
		env.verifyChain()
	})

	t.Run("Sprint", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		// Sprint = Goal + Decompose + Assign + Progress + Complete + Review
		goal, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"sprint goal: ship auth feature", env.convID, []types.EventID{env.boot.ID()}, signer)
		task1, _ := env.grammar.Derive(env.ctx, alice.ID(),
			"task: backend auth", goal.ID(), env.convID, signer)
		task2, _ := env.grammar.Derive(env.ctx, alice.ID(),
			"task: frontend auth", goal.ID(), env.convID, signer)

		_, _ = env.grammar.Delegate(env.ctx, alice.ID(), bob.ID(),
			types.MustDomainScope("backend"), types.MustWeight(0.8),
			task1.ID(), env.convID, signer)

		done1, _ := env.grammar.Derive(env.ctx, bob.ID(),
			"complete: backend auth done", task1.ID(), env.convID, signer)
		done2, _ := env.grammar.Derive(env.ctx, alice.ID(),
			"complete: frontend auth done", task2.ID(), env.convID, signer)

		sprintDone, _ := env.grammar.Merge(env.ctx, alice.ID(),
			"sprint complete: auth feature shipped",
			[]types.EventID{done1.ID(), done2.ID()}, env.convID, signer)

		ancestors := env.ancestors(sprintDone.ID(), 10)
		if !containsEvent(ancestors, goal.ID()) {
			t.Error("sprint completion should trace to goal")
		}
		env.verifyChain()
	})

	t.Run("DelegateAndVerify", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		agent := env.actor("Agent", 4, event.ActorTypeAI)

		task, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"task: generate test data", env.convID, []types.EventID{env.boot.ID()}, signer)

		_, _ = env.grammar.Delegate(env.ctx, alice.ID(), agent.ID(),
			types.MustDomainScope("test_data"), types.MustWeight(0.6),
			task.ID(), env.convID, signer)

		work, _ := env.grammar.Derive(env.ctx, agent.ID(),
			"complete: generated 500 test records", task.ID(), env.convID, signer)

		// Alice verifies
		verification, _ := env.grammar.Respond(env.ctx, alice.ID(),
			"verified: spot-checked 50 records, all valid",
			work.ID(), env.convID, signer)

		ancestors := env.ancestors(verification.ID(), 10)
		if !containsEvent(ancestors, task.ID()) {
			t.Error("verification should trace to original task")
		}
		env.verifyChain()
	})

	t.Run("Escalate", func(t *testing.T) {
		env := newTestEnv(t)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)
		manager := env.actor("Manager", 5, event.ActorTypeHuman)

		task, _ := env.grammar.Emit(env.ctx, bob.ID(),
			"task: production database migration", env.convID, []types.EventID{env.boot.ID()}, signer)

		// Bob escalates — needs higher authority
		escalation, _ := env.graph.Record(
			event.EventTypeAuthorityRequested, bob.ID(),
			event.AuthorityRequestContent{
				Actor:  bob.ID(),
				Action: "production_database_migration",
				Level:  event.AuthorityLevelRequired,
			},
			[]types.EventID{task.ID()}, env.convID, signer)

		// Manager approves
		_, err := env.graph.Record(
			event.EventTypeAuthorityResolved, manager.ID(),
			event.AuthorityResolvedContent{
				RequestID: escalation.ID(),
				Approved:  true,
				Resolver:  manager.ID(),
				Reason:    types.None[string](),
			},
			[]types.EventID{escalation.ID()}, env.convID, signer)
		if err != nil {
			t.Fatalf("authority resolved: %v", err)
		}
		env.verifyChain()
	})
}
