package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestKnowledgeGrammar exercises the Knowledge Grammar (Layer 6: Information).
// Operations: Claim, Categorize, Abstract, Infer, Remember/Recall, Challenge,
// Detect-Bias, Correct, Trace, Learn.
// Named functions: Verify, Fact-Check, Retract.
func TestKnowledgeGrammar(t *testing.T) {
	t.Run("ClaimAndCategorize", func(t *testing.T) {
		env := newTestEnv(t)
		analyst := env.actor("Analyst", 1, event.ActorTypeAI)

		claim, _ := env.grammar.Emit(env.ctx, analyst.ID(),
			"fact: Go 1.24 supports generic type aliases, confidence 0.95",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		category, _ := env.grammar.Annotate(env.ctx, analyst.ID(),
			claim.ID(), "classification", "programming_languages/go/features",
			env.convID, signer)

		ancestors := env.ancestors(category.ID(), 5)
		if !containsEvent(ancestors, claim.ID()) {
			t.Error("category should reference claim")
		}
		env.verifyChain()
	})

	t.Run("AbstractAndInfer", func(t *testing.T) {
		env := newTestEnv(t)
		analyst := env.actor("Analyst", 1, event.ActorTypeAI)

		fact1, _ := env.grammar.Emit(env.ctx, analyst.ID(),
			"fact: Service A handles 10k RPS on Go",
			env.convID, []types.EventID{env.boot.ID()}, signer)
		fact2, _ := env.grammar.Emit(env.ctx, analyst.ID(),
			"fact: Service B handles 12k RPS on Go",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Abstract: generalization from instances
		abstraction, _ := env.grammar.Merge(env.ctx, analyst.ID(),
			"abstraction: Go services typically handle 10k+ RPS",
			[]types.EventID{fact1.ID(), fact2.ID()}, env.convID, signer)

		// Infer: draw conclusion from abstraction
		inference, _ := env.grammar.Derive(env.ctx, analyst.ID(),
			"inference: new Go service C should handle 10k+ RPS, confidence 0.7",
			abstraction.ID(), env.convID, signer)

		ancestors := env.ancestors(inference.ID(), 10)
		if !containsEvent(ancestors, fact1.ID()) {
			t.Error("inference should trace to fact1")
		}
		if !containsEvent(ancestors, fact2.ID()) {
			t.Error("inference should trace to fact2")
		}
		env.verifyChain()
	})

	t.Run("ChallengeAndCorrect", func(t *testing.T) {
		env := newTestEnv(t)
		analyst := env.actor("Analyst", 1, event.ActorTypeAI)
		reviewer := env.actor("Reviewer", 2, event.ActorTypeAI)

		claim, _ := env.grammar.Emit(env.ctx, analyst.ID(),
			"fact: Python is faster than Go for web servers",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		challenge, _ := env.grammar.Respond(env.ctx, reviewer.ID(),
			"challenge: benchmark shows Go 3x faster than Python for HTTP serving",
			claim.ID(), env.convID, signer)

		correction, _ := env.grammar.Derive(env.ctx, analyst.ID(),
			"correction: Go is significantly faster than Python for web servers",
			challenge.ID(), env.convID, signer)

		// Original claim still exists (append-only)
		original, err := env.store.Get(claim.ID())
		if err != nil {
			t.Fatalf("original claim should still exist: %v", err)
		}
		_ = original

		ancestors := env.ancestors(correction.ID(), 10)
		if !containsEvent(ancestors, claim.ID()) {
			t.Error("correction should trace to original claim")
		}
		env.verifyChain()
	})

	t.Run("DetectBias", func(t *testing.T) {
		env := newTestEnv(t)
		analyst := env.actor("Analyst", 1, event.ActorTypeAI)
		reviewer := env.actor("Reviewer", 2, event.ActorTypeAI)

		claim, _ := env.grammar.Emit(env.ctx, analyst.ID(),
			"fact: framework X is the best for microservices",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		bias, _ := env.grammar.Annotate(env.ctx, reviewer.ID(),
			claim.ID(), "bias",
			"vendor bias: all cited sources are from framework X's company",
			env.convID, signer)

		_ = bias
		env.verifyChain()
	})

	t.Run("Learn", func(t *testing.T) {
		env := newTestEnv(t)
		analyst := env.actor("Analyst", 1, event.ActorTypeAI)

		mistake, _ := env.grammar.Emit(env.ctx, analyst.ID(),
			"error: predicted Service X handles 10k RPS, actual was 6k",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		learning, _ := env.grammar.Extend(env.ctx, analyst.ID(),
			"learning: always verify benchmarks include production conditions (DB load, concurrent users)",
			mistake.ID(), env.convID, signer)

		ancestors := env.ancestors(learning.ID(), 5)
		if !containsEvent(ancestors, mistake.ID()) {
			t.Error("learning should trace to mistake")
		}
		env.verifyChain()
	})

	t.Run("FactCheck", func(t *testing.T) {
		env := newTestEnv(t)
		analyst := env.actor("Analyst", 1, event.ActorTypeAI)
		checker := env.actor("FactChecker", 2, event.ActorTypeAI)

		claim, _ := env.grammar.Emit(env.ctx, analyst.ID(),
			"fact: event sourcing always improves performance",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Trace provenance
		trace, _ := env.grammar.Annotate(env.ctx, checker.ID(),
			claim.ID(), "provenance", "source: blog post, no benchmarks cited",
			env.convID, signer)

		// Check for bias
		biasCheck, _ := env.grammar.Annotate(env.ctx, checker.ID(),
			claim.ID(), "bias_check", "absolute claim without qualification, no counter-evidence considered",
			env.convID, signer)

		// Verdict
		verdict, _ := env.grammar.Derive(env.ctx, checker.ID(),
			"fact-check: MISLEADING — event sourcing improves auditability but can decrease read performance without projections",
			claim.ID(), env.convID, signer)

		_ = trace
		_ = biasCheck
		ancestors := env.ancestors(verdict.ID(), 5)
		if !containsEvent(ancestors, claim.ID()) {
			t.Error("verdict should trace to claim")
		}
		env.verifyChain()
	})

	t.Run("SelfRetract", func(t *testing.T) {
		env := newTestEnv(t)
		analyst := env.actor("Analyst", 1, event.ActorTypeAI)

		claim, _ := env.grammar.Emit(env.ctx, analyst.ID(),
			"fact: library X has no known vulnerabilities",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Author retracts own claim
		retraction, err := env.grammar.Retract(env.ctx, analyst.ID(),
			claim.ID(), "retracted: CVE-2026-1234 discovered after publication",
			env.convID, signer)
		if err != nil {
			t.Fatalf("Retract: %v", err)
		}

		// Original preserved (provenance maintained)
		original, _ := env.store.Get(claim.ID())
		if original.ID() != claim.ID() {
			t.Error("original claim should still exist")
		}

		ancestors := env.ancestors(retraction.ID(), 5)
		if !containsEvent(ancestors, claim.ID()) {
			t.Error("retraction should trace to claim")
		}
		env.verifyChain()
	})
}
