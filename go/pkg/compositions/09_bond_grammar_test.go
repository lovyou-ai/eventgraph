package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestBondGrammar exercises the Bond Grammar (Layer 9: Relationship).
// Operations: Connect, Balance, Deepen, Open, Attune, Feel-With, Break,
// Apologize, Reconcile, Mourn.
// Named functions: Betrayal-Repair, Check-In, Forgive.
func TestBondGrammar(t *testing.T) {
	t.Run("Connect", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		// Mutual subscribe
		sub1, _ := env.grammar.Subscribe(env.ctx, alice.ID(), bob.ID(),
			types.Some(types.MustDomainScope("collaboration")),
			env.boot.ID(), env.convID, signer)
		_, _ = env.grammar.Subscribe(env.ctx, bob.ID(), alice.ID(),
			types.Some(types.MustDomainScope("collaboration")),
			sub1.ID(), env.convID, signer)

		env.verifyChain()
	})

	t.Run("Balance", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		// Alice helps Bob
		help1, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"help: reviewed Bob's PR",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Bob helps Alice back
		help2, _ := env.grammar.Emit(env.ctx, bob.ID(),
			"help: debugged Alice's test failures",
			env.convID, []types.EventID{help1.ID()}, signer)

		// Balance check
		balance, _ := env.grammar.Annotate(env.ctx, env.system,
			help2.ID(), "reciprocity",
			"give/take ratio: 0.0 (balanced), Alice gave 1, Bob gave 1",
			env.convID, signer)

		_ = balance
		env.verifyChain()
	})

	t.Run("DeepenAndOpen", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		// Open private channel (vulnerability)
		channel, _ := env.grammar.Channel(env.ctx, alice.ID(), bob.ID(),
			types.Some(types.MustDomainScope("personal")),
			env.boot.ID(), env.convID, signer)

		// Vulnerable sharing
		share, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"vulnerable: struggling with imposter syndrome about the architecture role",
			env.convID, []types.EventID{channel.ID()}, signer)

		// Empathetic response
		response, _ := env.grammar.Respond(env.ctx, bob.ID(),
			"empathy: I felt the same when I first led a project — it gets easier",
			share.ID(), env.convID, signer)

		ancestors := env.ancestors(response.ID(), 10)
		if !containsEvent(ancestors, channel.ID()) {
			t.Error("response should trace to channel opening")
		}
		env.verifyChain()
	})

	t.Run("BreakAndApologize", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		// Establish connection
		sub, _ := env.grammar.Subscribe(env.ctx, alice.ID(), bob.ID(),
			types.None[types.DomainScope](),
			env.boot.ID(), env.convID, signer)

		// Rupture
		rupture, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"rupture: Bob took credit for shared work in the team meeting",
			env.convID, []types.EventID{sub.ID()}, signer)

		// Apology
		apology, _ := env.grammar.Respond(env.ctx, bob.ID(),
			"apology: I should have credited you — it was our joint work and I was wrong to present it as mine",
			rupture.ID(), env.convID, signer)

		ancestors := env.ancestors(apology.ID(), 5)
		if !containsEvent(ancestors, rupture.ID()) {
			t.Error("apology should trace to rupture")
		}
		env.verifyChain()
	})

	t.Run("Reconcile", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		rupture, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"rupture: trust broken",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		apology, _ := env.grammar.Respond(env.ctx, bob.ID(),
			"apology: acknowledging harm caused",
			rupture.ID(), env.convID, signer)

		reconcile, _ := env.grammar.Derive(env.ctx, env.system,
			"reconciliation: progress 0.3, both parties engaging, trust rebuilding slowly",
			apology.ID(), env.convID, signer)

		ancestors := env.ancestors(reconcile.ID(), 10)
		if !containsEvent(ancestors, rupture.ID()) {
			t.Error("reconciliation should trace to rupture")
		}
		env.verifyChain()
	})

	t.Run("BetrayalRepair", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		// Break → Apologize → Reconcile → Deepen
		channel, _ := env.grammar.Channel(env.ctx, alice.ID(), bob.ID(),
			types.None[types.DomainScope](),
			env.boot.ID(), env.convID, signer)

		betrayal, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"betrayal: Bob shared private conversation externally",
			env.convID, []types.EventID{channel.ID()}, signer)

		apology, _ := env.grammar.Respond(env.ctx, bob.ID(),
			"apology: I violated your trust by sharing our private conversation",
			betrayal.ID(), env.convID, signer)

		reconcile, _ := env.grammar.Derive(env.ctx, env.system,
			"reconciliation: progress 0.5, new boundaries established",
			apology.ID(), env.convID, signer)

		deepen, _ := env.grammar.Extend(env.ctx, env.system,
			"deepened: relationship stronger after repair — trust rebuilt on new basis",
			reconcile.ID(), env.convID, signer)

		ancestors := env.ancestors(deepen.ID(), 10)
		if !containsEvent(ancestors, betrayal.ID()) {
			t.Error("deepened relationship should trace to original betrayal")
		}
		env.verifyChain()
	})

	t.Run("Forgive", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		sub, _ := env.grammar.Subscribe(env.ctx, alice.ID(), bob.ID(),
			types.None[types.DomainScope](),
			env.boot.ID(), env.convID, signer)

		violation, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"violation: trust broken",
			env.convID, []types.EventID{sub.ID()}, signer)

		edgeID, _ := types.NewEdgeID(sub.ID().Value())
		severEv, _ := env.grammar.Sever(env.ctx, alice.ID(),
			edgeID, violation.ID(), env.convID, signer)

		// Forgive = Subscribe after Sever
		forgiveEv, err := env.grammar.Forgive(env.ctx, alice.ID(),
			severEv.ID(), bob.ID(),
			types.None[types.DomainScope](),
			env.convID, signer)
		if err != nil {
			t.Fatalf("Forgive: %v", err)
		}

		ancestors := env.ancestors(forgiveEv.ID(), 10)
		if !containsEvent(ancestors, severEv.ID()) {
			t.Error("forgiveness should trace to sever")
		}
		env.verifyChain()
	})
}
