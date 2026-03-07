package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestBeingGrammar exercises the Being Grammar (Layer 13: Existence).
// Operations: Account, Externalise, Sustain, Restore, Interconnect, Steward,
// Measure-True-Cost, Threshold, Cascade, Wonder.
// Named functions: True-Cost-Chain, Ecological-Commons, Sustainability-Audit.
func TestBeingGrammar(t *testing.T) {
	t.Run("TrueCost", func(t *testing.T) {
		env := newTestEnv(t)
		producer := env.actor("Producer", 1, event.ActorTypeHuman)
		auditor := env.actor("Auditor", 2, event.ActorTypeAI)

		economic, _ := env.grammar.Emit(env.ctx, producer.ID(),
			"output: manufactured 1000 units, revenue $50k",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		ecological, _ := env.grammar.Emit(env.ctx, auditor.ID(),
			"cost: 2 tonnes CO2, 500L water, 10kWh energy for same 1000 units",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Both sides linked — no externalisation
		trueCost, _ := env.grammar.Merge(env.ctx, auditor.ID(),
			"true cost: $50k revenue at $12k environmental cost — net social value $38k",
			[]types.EventID{economic.ID(), ecological.ID()}, env.convID, signer)

		ancestors := env.ancestors(trueCost.ID(), 5)
		if !containsEvent(ancestors, economic.ID()) {
			t.Error("true cost should include economic output")
		}
		if !containsEvent(ancestors, ecological.ID()) {
			t.Error("true cost should include ecological cost")
		}
		env.verifyChain()
	})

	t.Run("Sustain", func(t *testing.T) {
		env := newTestEnv(t)
		steward := env.actor("Steward", 1, event.ActorTypeHuman)
		monitor := env.actor("Monitor", 2, event.ActorTypeAI)

		baseline, _ := env.grammar.Emit(env.ctx, monitor.ID(),
			"baseline: forest canopy cover 78%, biodiversity index 0.82",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		intervention, _ := env.grammar.Derive(env.ctx, steward.ID(),
			"intervention: planted 500 native trees, removed invasive species from 2 hectares",
			baseline.ID(), env.convID, signer)

		measurement, _ := env.grammar.Derive(env.ctx, monitor.ID(),
			"measurement: canopy cover 81%, biodiversity index 0.85 — positive trajectory",
			intervention.ID(), env.convID, signer)

		ancestors := env.ancestors(measurement.ID(), 10)
		if !containsEvent(ancestors, baseline.ID()) {
			t.Error("measurement should trace to baseline")
		}
		env.verifyChain()
	})

	t.Run("Threshold", func(t *testing.T) {
		env := newTestEnv(t)
		monitor := env.actor("Monitor", 1, event.ActorTypeAI)
		steward := env.actor("Steward", 2, event.ActorTypeHuman)

		reading, _ := env.grammar.Emit(env.ctx, monitor.ID(),
			"reading: water temperature 26.5C, coral bleaching threshold 27C",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		warning, _ := env.grammar.Derive(env.ctx, monitor.ID(),
			"warning: approaching coral bleaching threshold, 0.5C margin, trend: +0.1C/month",
			reading.ID(), env.convID, signer)

		// Escalate for intervention
		_, _ = env.graph.Record(
			event.EventTypeAuthorityRequested, monitor.ID(),
			event.AuthorityRequestContent{
				Actor:  steward.ID(),
				Action: "ecological_intervention_coral_protection",
				Level:  event.AuthorityLevelRequired,
			},
			[]types.EventID{warning.ID()}, env.convID, signer)

		env.verifyChain()
	})

	t.Run("Interconnect", func(t *testing.T) {
		env := newTestEnv(t)
		monitor := env.actor("Monitor", 1, event.ActorTypeAI)

		forest, _ := env.grammar.Emit(env.ctx, monitor.ID(),
			"system: forest ecosystem — canopy 78%, 120 species catalogued",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		river, _ := env.grammar.Emit(env.ctx, monitor.ID(),
			"system: river ecosystem — flow 450L/s, dissolved oxygen 8.2mg/L",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		connection, _ := env.grammar.Merge(env.ctx, monitor.ID(),
			"interconnection: forest canopy loss → increased runoff → river sediment load up 15%",
			[]types.EventID{forest.ID(), river.ID()}, env.convID, signer)

		ancestors := env.ancestors(connection.ID(), 5)
		if !containsEvent(ancestors, forest.ID()) {
			t.Error("interconnection should include forest system")
		}
		if !containsEvent(ancestors, river.ID()) {
			t.Error("interconnection should include river system")
		}
		env.verifyChain()
	})

	t.Run("TrueCostChain", func(t *testing.T) {
		env := newTestEnv(t)
		producer := env.actor("Producer", 1, event.ActorTypeHuman)
		supplier := env.actor("Supplier", 2, event.ActorTypeHuman)
		auditor := env.actor("Auditor", 3, event.ActorTypeAI)

		// Supply chain with environmental costs at each stage
		raw, _ := env.grammar.Emit(env.ctx, supplier.ID(),
			"extraction: raw materials, 0.5t CO2, $5k cost",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		manufacturing, _ := env.grammar.Derive(env.ctx, producer.ID(),
			"manufacturing: processing raw materials, 1.5t CO2, $15k cost",
			raw.ID(), env.convID, signer)

		distribution, _ := env.grammar.Derive(env.ctx, producer.ID(),
			"distribution: shipping to market, 0.3t CO2, $3k cost",
			manufacturing.ID(), env.convID, signer)

		totalCost, _ := env.grammar.Derive(env.ctx, auditor.ID(),
			"true cost chain: total 2.3t CO2, $23k direct cost, $8k environmental cost — true cost $31k",
			distribution.ID(), env.convID, signer)

		ancestors := env.ancestors(totalCost.ID(), 10)
		if !containsEvent(ancestors, raw.ID()) {
			t.Error("true cost should trace to raw extraction")
		}
		env.verifyChain()
	})

	t.Run("SustainabilityAudit", func(t *testing.T) {
		env := newTestEnv(t)
		auditor := env.actor("Auditor", 1, event.ActorTypeAI)

		energy, _ := env.grammar.Emit(env.ctx, auditor.ID(),
			"assessment: energy usage 500kWh/month, 60% renewable",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		waste, _ := env.grammar.Emit(env.ctx, auditor.ID(),
			"assessment: waste output 200kg/month, 45% recycled",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		water, _ := env.grammar.Emit(env.ctx, auditor.ID(),
			"assessment: water usage 10kL/month, 20% recycled",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		audit, _ := env.grammar.Merge(env.ctx, auditor.ID(),
			"sustainability audit: overall score 0.58, energy good, waste needs improvement, water poor",
			[]types.EventID{energy.ID(), waste.ID(), water.ID()}, env.convID, signer)

		recommendations, _ := env.grammar.Derive(env.ctx, auditor.ID(),
			"recommendations: 1) increase waste recycling to 70% 2) install water recycling 3) target 80% renewable energy",
			audit.ID(), env.convID, signer)

		ancestors := env.ancestors(recommendations.ID(), 10)
		if !containsEvent(ancestors, energy.ID()) {
			t.Error("recommendations should trace to energy assessment")
		}
		if !containsEvent(ancestors, waste.ID()) {
			t.Error("recommendations should trace to waste assessment")
		}
		if !containsEvent(ancestors, water.ID()) {
			t.Error("recommendations should trace to water assessment")
		}
		env.verifyChain()
	})

	t.Run("EcologicalCommons", func(t *testing.T) {
		env := newTestEnv(t)
		systemA := env.actor("MonitorA", 1, event.ActorTypeAI)
		systemB := env.actor("MonitorB", 2, event.ActorTypeAI)
		steward := env.actor("Steward", 3, event.ActorTypeHuman)

		// Two monitoring systems contribute data
		dataA, _ := env.grammar.Emit(env.ctx, systemA.ID(),
			"observation: bird population stable, 42 species in sector A",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		dataB, _ := env.grammar.Emit(env.ctx, systemB.ID(),
			"observation: insect population declining 8% in sector B, linked to pesticide use",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Cross-system ecological view
		synthesis, _ := env.grammar.Merge(env.ctx, steward.ID(),
			"ecological commons: bird stable but insect decline in adjacent sector may cascade — monitoring required",
			[]types.EventID{dataA.ID(), dataB.ID()}, env.convID, signer)

		ancestors := env.ancestors(synthesis.ID(), 5)
		if !containsEvent(ancestors, dataA.ID()) {
			t.Error("synthesis should include data from system A")
		}
		if !containsEvent(ancestors, dataB.ID()) {
			t.Error("synthesis should include data from system B")
		}
		env.verifyChain()
	})

	t.Run("Wonder", func(t *testing.T) {
		env := newTestEnv(t)
		observer := env.actor("Observer", 1, event.ActorTypeHuman)

		// The Sacred primitive — beyond optimisation
		observation, _ := env.grammar.Emit(env.ctx, observer.ID(),
			"observation: all 13 layers functioning together — decisions accountable, communities healthy, ecology sustained",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		wonder, _ := env.grammar.Derive(env.ctx, observer.ID(),
			"wonder: the whole is greater than the sum of its parts — emergence from simple rules to complex flourishing",
			observation.ID(), env.convID, signer)

		ancestors := env.ancestors(wonder.ID(), 5)
		if !containsEvent(ancestors, observation.ID()) {
			t.Error("wonder should trace to observation")
		}
		env.verifyChain()
	})
}
