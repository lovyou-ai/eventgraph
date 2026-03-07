package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestEvolutionGrammar exercises the Evolution Grammar (Layer 12: Culture).
// Operations: Transmit, Adapt, Preserve, Create, Ritual, Curate, Interpret,
// Translate, Honour, Innovate.
// Named functions: Tradition-Chain, Creative-Provenance, Language-Archive.
func TestEvolutionGrammar(t *testing.T) {
	t.Run("TransmitAndAdapt", func(t *testing.T) {
		env := newTestEnv(t)
		teacher := env.actor("Teacher", 1, event.ActorTypeHuman)
		student := env.actor("Student", 2, event.ActorTypeHuman)

		original, _ := env.grammar.Emit(env.ctx, teacher.ID(),
			"tradition: pair programming — two developers, one keyboard, continuous review",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		adaptation, _ := env.grammar.Derive(env.ctx, student.ID(),
			"adaptation: mob programming — whole team, one screen, rotating driver every 15 min",
			original.ID(), env.convID, signer)

		ancestors := env.ancestors(adaptation.ID(), 5)
		if !containsEvent(ancestors, original.ID()) {
			t.Error("adaptation should trace to original tradition")
		}
		env.verifyChain()
	})

	t.Run("Preserve", func(t *testing.T) {
		env := newTestEnv(t)
		elder := env.actor("Elder", 1, event.ActorTypeHuman)
		archivist := env.actor("Archivist", 2, event.ActorTypeAI)

		practice, _ := env.grammar.Emit(env.ctx, elder.ID(),
			"practice: weekly team retrospective with talking stick protocol",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		context, _ := env.grammar.Annotate(env.ctx, elder.ID(),
			practice.ID(), "context",
			"origin: adapted from indigenous council practices, used since 2019",
			env.convID, signer)

		archive, _ := env.grammar.Annotate(env.ctx, archivist.ID(),
			practice.ID(), "preservation",
			"archived: full protocol documented, video examples stored, elder verified",
			env.convID, signer)

		_ = context
		_ = archive
		env.verifyChain()
	})

	t.Run("CreativeProvenance", func(t *testing.T) {
		env := newTestEnv(t)
		artist := env.actor("Artist", 1, event.ActorTypeHuman)

		// Rich derive chain shows creative process
		inspiration, _ := env.grammar.Emit(env.ctx, artist.ID(),
			"inspiration: watching starlings murmur at dusk",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		sketch, _ := env.grammar.Derive(env.ctx, artist.ID(),
			"draft: initial sketch of distributed consensus as flocking behaviour",
			inspiration.ID(), env.convID, signer)

		revision, _ := env.grammar.Derive(env.ctx, artist.ID(),
			"revision: added local-only interaction rules, removed central coordinator",
			sketch.ID(), env.convID, signer)

		final, _ := env.grammar.Derive(env.ctx, artist.ID(),
			"final: 'Murmuration Protocol' — distributed consensus inspired by starling flocks",
			revision.ID(), env.convID, signer)

		// Provenance chain: final → revision → sketch → inspiration
		ancestors := env.ancestors(final.ID(), 10)
		if !containsEvent(ancestors, inspiration.ID()) {
			t.Error("final work should trace to inspiration")
		}
		if !containsEvent(ancestors, sketch.ID()) {
			t.Error("final work should trace to sketch")
		}
		env.verifyChain()
	})

	t.Run("Ritual", func(t *testing.T) {
		env := newTestEnv(t)
		alice := env.actor("Alice", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)
		charlie := env.actor("Charlie", 3, event.ActorTypeHuman)

		// Collective synchronous event
		ritualStart, _ := env.grammar.Emit(env.ctx, env.system,
			"ritual: Friday demo day begins",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		demo1, _ := env.grammar.Respond(env.ctx, alice.ID(),
			"demo: new search feature with fuzzy matching",
			ritualStart.ID(), env.convID, signer)
		demo2, _ := env.grammar.Respond(env.ctx, bob.ID(),
			"demo: performance improvements — 3x faster query execution",
			ritualStart.ID(), env.convID, signer)
		demo3, _ := env.grammar.Respond(env.ctx, charlie.ID(),
			"demo: accessibility audit results and fixes",
			ritualStart.ID(), env.convID, signer)

		// Merge into shared experience
		shared, _ := env.grammar.Merge(env.ctx, env.system,
			"ritual complete: 3 demos shared, team aligned on progress",
			[]types.EventID{demo1.ID(), demo2.ID(), demo3.ID()}, env.convID, signer)

		ancestors := env.ancestors(shared.ID(), 5)
		if !containsEvent(ancestors, demo1.ID()) {
			t.Error("shared experience should include demo1")
		}
		if !containsEvent(ancestors, demo2.ID()) {
			t.Error("shared experience should include demo2")
		}
		if !containsEvent(ancestors, demo3.ID()) {
			t.Error("shared experience should include demo3")
		}
		env.verifyChain()
	})

	t.Run("TraditionChain", func(t *testing.T) {
		env := newTestEnv(t)
		gen1 := env.actor("Gen1", 1, event.ActorTypeHuman)
		gen2 := env.actor("Gen2", 2, event.ActorTypeHuman)
		gen3 := env.actor("Gen3", 3, event.ActorTypeHuman)

		// Three-generation transmission chain
		original, _ := env.grammar.Emit(env.ctx, gen1.ID(),
			"tradition: code review checklist — security, tests, docs",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		adapted1, _ := env.grammar.Derive(env.ctx, gen2.ID(),
			"adaptation: added performance benchmarks to checklist",
			original.ID(), env.convID, signer)

		adapted2, _ := env.grammar.Derive(env.ctx, gen3.ID(),
			"adaptation: added accessibility check, removed outdated docs requirement",
			adapted1.ID(), env.convID, signer)

		// Full lineage visible
		ancestors := env.ancestors(adapted2.ID(), 10)
		if !containsEvent(ancestors, original.ID()) {
			t.Error("third generation should trace to original")
		}
		if !containsEvent(ancestors, adapted1.ID()) {
			t.Error("third generation should trace to second generation")
		}
		env.verifyChain()
	})

	t.Run("LanguageArchive", func(t *testing.T) {
		env := newTestEnv(t)
		speaker := env.actor("Speaker", 1, event.ActorTypeHuman)
		linguist := env.actor("Linguist", 2, event.ActorTypeAI)
		elder := env.actor("Elder", 3, event.ActorTypeHuman)

		record, _ := env.grammar.Emit(env.ctx, speaker.ID(),
			"linguistic record: greeting phrase 'Yaama' — hello in Gamilaraay",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		context, _ := env.grammar.Annotate(env.ctx, linguist.ID(),
			record.ID(), "linguistic_context",
			"usage: informal greeting, all contexts, tonal emphasis on first syllable",
			env.convID, signer)

		verification, _ := env.grammar.Endorse(env.ctx, elder.ID(),
			record.ID(), speaker.ID(), types.MustWeight(1.0),
			types.None[types.DomainScope](), env.convID, signer)

		_ = context
		ancestors := env.ancestors(verification.ID(), 5)
		if !containsEvent(ancestors, record.ID()) {
			t.Error("elder verification should trace to linguistic record")
		}
		env.verifyChain()
	})

	t.Run("Interpret", func(t *testing.T) {
		env := newTestEnv(t)
		creator := env.actor("Creator", 1, event.ActorTypeHuman)
		critic := env.actor("Critic", 2, event.ActorTypeHuman)

		work, _ := env.grammar.Emit(env.ctx, creator.ID(),
			"work: open-source governance framework",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		interpretation, _ := env.grammar.Derive(env.ctx, critic.ID(),
			"interpretation: the framework embodies commons-based peer production, echoing Ostrom's principles",
			work.ID(), env.convID, signer)

		ancestors := env.ancestors(interpretation.ID(), 5)
		if !containsEvent(ancestors, work.ID()) {
			t.Error("interpretation should trace to original work")
		}
		env.verifyChain()
	})
}
