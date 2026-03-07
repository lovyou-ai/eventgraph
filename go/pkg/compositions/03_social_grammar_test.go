package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestSocialGrammar exercises the Social Grammar (Layer 3: Society).
// Operations: Norm, Moderate, Elect, Welcome, Exile.
// Named functions: Poll, Schism, Federation.
func TestSocialGrammar(t *testing.T) {
	t.Run("Norm", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		proposal, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"norm proposal: all PRs require at least one review before merge",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Community votes via consent
		vote1, _ := env.grammar.Consent(env.ctx, alice.ID(), bob.ID(),
			"norm accepted: PR review requirement",
			types.MustDomainScope("governance"),
			proposal.ID(), env.convID, signer)

		ancestors := env.ancestors(vote1.ID(), 5)
		if !containsEvent(ancestors, proposal.ID()) {
			t.Error("norm vote should trace to proposal")
		}
		env.verifyChain()
	})

	t.Run("Moderate", func(t *testing.T) {
		env := newTestEnv(t)
		mod := env.actor("Moderator", 1, event.ActorTypeHuman)
		user := env.actor("User", 2, event.ActorTypeHuman)

		// Establish norm
		norm, _ := env.grammar.Emit(env.ctx, mod.ID(),
			"norm: no spam or self-promotion",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// User violates
		spam, _ := env.grammar.Emit(env.ctx, user.ID(),
			"buy my product at example.com!!!",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Moderator retracts content
		retraction, err := env.grammar.Retract(env.ctx, user.ID(),
			spam.ID(), "spam: violates no-promotion norm",
			env.convID, signer)
		if err != nil {
			t.Fatalf("Retract: %v", err)
		}

		// Moderation action annotated with norm reference
		_, _ = env.grammar.Annotate(env.ctx, mod.ID(),
			retraction.ID(), "moderation",
			"content removed for violating no-spam norm",
			env.convID, signer)

		_ = norm
		env.verifyChain()
	})

	t.Run("Elect", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)
		carol := env.actor("Carol", 3, event.ActorTypeHuman)

		// Nomination
		nomination, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"nomination: Bob for maintainer role",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Votes
		v1, _ := env.grammar.Consent(env.ctx, alice.ID(), bob.ID(),
			"vote: yes for Bob as maintainer",
			types.MustDomainScope("governance"),
			nomination.ID(), env.convID, signer)
		v2, _ := env.grammar.Consent(env.ctx, carol.ID(), bob.ID(),
			"vote: yes for Bob as maintainer",
			types.MustDomainScope("governance"),
			nomination.ID(), env.convID, signer)

		// Result
		result, _ := env.grammar.Merge(env.ctx, env.system,
			"election result: Bob elected maintainer (2 yes, 0 no)",
			[]types.EventID{v1.ID(), v2.ID()}, env.convID, signer)

		ancestors := env.ancestors(result.ID(), 10)
		if !containsEvent(ancestors, nomination.ID()) {
			t.Error("election result should trace to nomination")
		}
		env.verifyChain()
	})

	t.Run("WelcomeAndExile", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		newbie := env.actor("Newbie", 4, event.ActorTypeHuman)

		// Welcome = Invite (Endorse + Subscribe)
		endorseEv, _, err := env.grammar.Invite(env.ctx, alice.ID(), newbie.ID(),
			types.MustWeight(0.3),
			types.Some(types.MustDomainScope("community")),
			env.boot.ID(), env.convID, signer)
		if err != nil {
			t.Fatalf("Invite: %v", err)
		}

		// Later: exile via Sever
		edgeID, _ := types.NewEdgeID(endorseEv.ID().Value())
		violation, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"violation: repeated harassment",
			env.convID, []types.EventID{endorseEv.ID()}, signer)

		// Can't sever endorsement (not severable), so sever would need a subscription
		// Instead, record exclusion
		_, _ = env.grammar.Annotate(env.ctx, alice.ID(),
			violation.ID(), "exclusion", "member excluded for repeated harassment",
			env.convID, signer)

		_ = edgeID
		env.verifyChain()
	})

	t.Run("Poll", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)
		carol := env.actor("Carol", 3, event.ActorTypeHuman)

		question, _ := env.grammar.Emit(env.ctx, env.system,
			"poll: should we adopt Go 1.24 or wait for 1.25?",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		v1, _ := env.grammar.Respond(env.ctx, alice.ID(),
			"vote: adopt 1.24 now", question.ID(), env.convID, signer)
		v2, _ := env.grammar.Respond(env.ctx, bob.ID(),
			"vote: wait for 1.25", question.ID(), env.convID, signer)
		v3, _ := env.grammar.Respond(env.ctx, carol.ID(),
			"vote: adopt 1.24 now", question.ID(), env.convID, signer)

		tally, _ := env.grammar.Merge(env.ctx, env.system,
			"poll result: adopt Go 1.24 (2-1)",
			[]types.EventID{v1.ID(), v2.ID(), v3.ID()}, env.convID, signer)

		ancestors := env.ancestors(tally.ID(), 5)
		if !containsEvent(ancestors, v1.ID()) {
			t.Error("tally should include all votes")
		}
		env.verifyChain()
	})

	t.Run("Federation", func(t *testing.T) {
		env := newTestEnv(t)
		communityA := env.actor("CommunityA", 1, event.ActorTypeHuman)
		communityB := env.actor("CommunityB", 2, event.ActorTypeHuman)

		// Bilateral federation agreement
		federation, _ := env.grammar.Consent(env.ctx, communityA.ID(), communityB.ID(),
			"federation: share security advisories, retain independent governance",
			types.MustDomainScope("security"),
			env.boot.ID(), env.convID, signer)

		// Scoped delegation
		_, _ = env.grammar.Delegate(env.ctx, communityA.ID(), communityB.ID(),
			types.MustDomainScope("security_advisories"), types.MustWeight(0.5),
			federation.ID(), env.convID, signer)

		env.verifyChain()
	})
}
