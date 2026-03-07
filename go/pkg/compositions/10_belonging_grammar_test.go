package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestBelongingGrammar exercises the Belonging Grammar (Layer 10: Community).
// Operations: Welcome, Exile, Norm, Remember, Nest, Grow, Shrink, Bridge, Heal, Celebrate.
// Named functions: Onboarding, Fractal-Nest, Community-Health.
func TestBelongingGrammar(t *testing.T) {
	t.Run("Welcome", func(t *testing.T) {
		env := newTestEnv(t)
		elder := env.actor("Elder", 1, event.ActorTypeHuman)
		newcomer := env.actor("Newcomer", 2, event.ActorTypeHuman)

		// Existing member endorses newcomer
		invite, _, _ := env.grammar.Invite(env.ctx, elder.ID(), newcomer.ID(),
			types.MustWeight(0.5),
			types.Some(types.MustDomainScope("community_alpha")),
			env.boot.ID(), env.convID, signer)

		// Newcomer subscribes
		sub, _ := env.grammar.Subscribe(env.ctx, newcomer.ID(), elder.ID(),
			types.Some(types.MustDomainScope("community_alpha")),
			invite.ID(), env.convID, signer)

		// Community acknowledges
		ack, _ := env.grammar.Acknowledge(env.ctx, elder.ID(),
			sub.ID(), newcomer.ID(), env.convID, signer)

		ancestors := env.ancestors(ack.ID(), 10)
		if !containsEvent(ancestors, invite.ID()) {
			t.Error("acknowledgement should trace to invite")
		}
		env.verifyChain()
	})

	t.Run("Exile", func(t *testing.T) {
		env := newTestEnv(t)
		moderator := env.actor("Moderator", 1, event.ActorTypeHuman)
		member := env.actor("Member", 2, event.ActorTypeHuman)

		sub, _ := env.grammar.Subscribe(env.ctx, member.ID(), moderator.ID(),
			types.Some(types.MustDomainScope("community_alpha")),
			env.boot.ID(), env.convID, signer)

		// Violation detected
		violation, _ := env.graph.Record(
			event.EventTypeViolationDetected, moderator.ID(),
			event.ViolationDetectedContent{
				Expectation: env.boot.ID(),
				Actor:       member.ID(),
				Severity:    event.SeverityLevelSerious,
				Description: "repeated harassment after warnings",
				Evidence:    types.MustNonEmpty([]types.EventID{sub.ID()}),
			},
			[]types.EventID{sub.ID()}, env.convID, signer)

		// Community decision to exile
		edgeID, _ := types.NewEdgeID(sub.ID().Value())
		sever, _ := env.grammar.Sever(env.ctx, moderator.ID(),
			edgeID, violation.ID(), env.convID, signer)

		ancestors := env.ancestors(sever.ID(), 10)
		if !containsEvent(ancestors, violation.ID()) {
			t.Error("exile should trace to violation")
		}
		env.verifyChain()
	})

	t.Run("NormAdoption", func(t *testing.T) {
		env := newTestEnv(t)
		proposer := env.actor("Proposer", 1, event.ActorTypeHuman)
		member := env.actor("Member", 2, event.ActorTypeHuman)

		norm, _ := env.grammar.Emit(env.ctx, proposer.ID(),
			"norm proposal: all code contributions require tests",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		discussion, _ := env.grammar.Respond(env.ctx, member.ID(),
			"support: agree, tests catch regressions early",
			norm.ID(), env.convID, signer)

		adoption, _ := env.grammar.Consent(env.ctx, proposer.ID(), member.ID(),
			"adopt norm: code contributions require tests",
			types.MustDomainScope("community_alpha"),
			discussion.ID(), env.convID, signer)

		ancestors := env.ancestors(adoption.ID(), 10)
		if !containsEvent(ancestors, norm.ID()) {
			t.Error("adoption should trace to norm proposal")
		}
		env.verifyChain()
	})

	t.Run("CollectiveMemory", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		milestone, _ := env.grammar.Emit(env.ctx, alice.ID(),
			"milestone: community reached 1000 contributions",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		reflection, _ := env.grammar.Respond(env.ctx, bob.ID(),
			"reflection: started with 3 people, now 50 active contributors",
			milestone.ID(), env.convID, signer)

		memory, _ := env.grammar.Merge(env.ctx, env.system,
			"collective memory: community grew from 3 to 50 members, reaching 1000 contributions in 6 months",
			[]types.EventID{milestone.ID(), reflection.ID()}, env.convID, signer)

		ancestors := env.ancestors(memory.ID(), 5)
		if !containsEvent(ancestors, milestone.ID()) {
			t.Error("memory should include milestone")
		}
		if !containsEvent(ancestors, reflection.ID()) {
			t.Error("memory should include reflection")
		}
		env.verifyChain()
	})

	t.Run("FractalNest", func(t *testing.T) {
		env := newTestEnv(t)
		admin := env.actor("Admin", 1, event.ActorTypeHuman)
		lead := env.actor("Lead", 2, event.ActorTypeHuman)

		// Parent community
		parent, _ := env.grammar.Emit(env.ctx, admin.ID(),
			"community: engineering_org",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Child community nested within
		child, _ := env.grammar.Derive(env.ctx, admin.ID(),
			"community: backend_team (nested in engineering_org)",
			parent.ID(), env.convID, signer)

		// Delegate authority downward
		delegation, _ := env.grammar.Delegate(env.ctx, admin.ID(), lead.ID(),
			types.MustDomainScope("backend_team"), types.MustWeight(0.7),
			child.ID(), env.convID, signer)

		ancestors := env.ancestors(delegation.ID(), 10)
		if !containsEvent(ancestors, parent.ID()) {
			t.Error("delegation should trace to parent community")
		}
		env.verifyChain()
	})

	t.Run("CommunityHealth", func(t *testing.T) {
		env := newTestEnv(t)
		monitor := env.actor("Monitor", 1, event.ActorTypeAI)

		// Gather health signals
		activity, _ := env.grammar.Emit(env.ctx, monitor.ID(),
			"signal: 45 events/day, 12 active members, 3 new joins this week",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		sentiment, _ := env.grammar.Emit(env.ctx, monitor.ID(),
			"signal: 82% positive interactions, 2 unresolved disputes",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		health, _ := env.grammar.Merge(env.ctx, monitor.ID(),
			"community health: score 0.78, growth healthy, 2 disputes need attention",
			[]types.EventID{activity.ID(), sentiment.ID()}, env.convID, signer)

		ancestors := env.ancestors(health.ID(), 5)
		if !containsEvent(ancestors, activity.ID()) {
			t.Error("health should include activity signal")
		}
		if !containsEvent(ancestors, sentiment.ID()) {
			t.Error("health should include sentiment signal")
		}
		env.verifyChain()
	})

	t.Run("Bridge", func(t *testing.T) {
		env := newTestEnv(t)
		bridge := env.actor("Bridge", 1, event.ActorTypeHuman)

		communityA, _ := env.grammar.Emit(env.ctx, bridge.ID(),
			"community: open_source_project",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		communityB, _ := env.grammar.Emit(env.ctx, bridge.ID(),
			"community: academic_research_group",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Bridge connects two communities
		connection, _ := env.grammar.Merge(env.ctx, bridge.ID(),
			"bridge: connecting open_source_project with academic_research_group for mutual benefit",
			[]types.EventID{communityA.ID(), communityB.ID()}, env.convID, signer)

		ancestors := env.ancestors(connection.ID(), 5)
		if !containsEvent(ancestors, communityA.ID()) {
			t.Error("bridge should connect to community A")
		}
		if !containsEvent(ancestors, communityB.ID()) {
			t.Error("bridge should connect to community B")
		}
		env.verifyChain()
	})
}
