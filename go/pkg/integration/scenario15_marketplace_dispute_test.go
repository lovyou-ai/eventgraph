package integration_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/compositions"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestScenario15_MarketplaceDispute exercises a subscription gone wrong:
// Provider sets up subscription → delivery fails → buyer disputes →
// arbitration with escrow → refund → reputation impact.
// Crosses Market, Justice, and Alignment grammars.
func TestScenario15_MarketplaceDispute(t *testing.T) {
	env := newTestEnv(t)
	market := compositions.NewMarketGrammar(env.grammar)
	alignment := compositions.NewAlignmentGrammar(env.grammar)

	provider := env.registerActor("CloudProvider", 1, event.ActorTypeAI)
	buyer := env.registerActor("StartupCo", 2, event.ActorTypeHuman)
	arbiter := env.registerActor("Arbiter", 3, event.ActorTypeHuman)

	// 1. Subscription established — 2 billing periods
	sub, err := market.Subscription(env.ctx, buyer.ID(), provider.ID(),
		"managed database service, $500/month, 99.9% uptime SLA",
		[]string{"month 1: $500", "month 2: $500"},
		[]string{"database service month 1", "database service month 2"},
		types.MustDomainScope("cloud_services"),
		env.boot.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("Subscription: %v", err)
	}
	if len(sub.Payments) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(sub.Payments))
	}

	// 2. Buyer detects SLA violation — 4 hours downtime in month 2
	lastDelivery := sub.Deliveries[len(sub.Deliveries)-1]

	// 3. Refund requested after dispute
	refund, err := market.Refund(env.ctx, buyer.ID(), provider.ID(),
		"SLA violation: 4 hours downtime vs 99.9% uptime guarantee",
		"acknowledged: downtime exceeded SLA, credit approved",
		"$250 credit (pro-rated for downtime)",
		lastDelivery.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("Refund: %v", err)
	}

	// 4. Impact assessment on the provider's service
	impact, err := alignment.ImpactAssessment(env.ctx, arbiter.ID(),
		refund.Dispute.ID(),
		"downtime affected 12 customers, 3 reported data access issues",
		"service impact distributed unevenly — smaller customers hit harder",
		"recommend pro-rated credits plus SLA improvement commitment",
		env.convID, signer)
	if err != nil {
		t.Fatalf("ImpactAssessment: %v", err)
	}

	// 5. Arbitration for the SLA violation going forward
	arb, err := market.Arbitration(env.ctx, buyer.ID(), provider.ID(), arbiter.ID(),
		"recurring SLA violations — 3 incidents in 6 months",
		types.MustDomainScope("cloud_services"), types.MustWeight(0.5),
		impact.Explanation.ID(), env.convID, signer)
	if err != nil {
		t.Fatalf("Arbitration: %v", err)
	}

	// 6. Reputation impact — multiple parties rate the provider
	raters := []types.ActorID{buyer.ID(), arbiter.ID()}
	targets := []types.EventID{arb.Release.ID(), arb.Release.ID()}
	weights := []types.Weight{types.MustWeight(-0.3), types.MustWeight(-0.1)}
	rep, err := market.ReputationTransfer(env.ctx,
		raters, targets, provider.ID(), weights,
		types.Some(types.MustDomainScope("cloud_services")),
		env.convID, signer)
	if err != nil {
		t.Fatalf("ReputationTransfer: %v", err)
	}

	// --- Assertions ---

	// Refund traces to subscription
	refundAncestors := env.ancestors(refund.Reversal.ID(), 15)
	if !containsEvent(refundAncestors, sub.Acceptance.ID()) {
		t.Error("refund should trace to original subscription acceptance")
	}

	// Arbitration traces through impact assessment to dispute
	arbAncestors := env.ancestors(arb.Release.ID(), 20)
	if !containsEvent(arbAncestors, refund.Dispute.ID()) {
		t.Error("arbitration should trace to original dispute")
	}

	// Impact assessment traces to dispute
	impactAncestors := env.ancestors(impact.Explanation.ID(), 10)
	if !containsEvent(impactAncestors, refund.Dispute.ID()) {
		t.Error("impact assessment should trace to dispute")
	}

	// Reputation ratings exist
	if len(rep.Ratings) != 2 {
		t.Errorf("expected 2 ratings, got %d", len(rep.Ratings))
	}

	env.verifyChain()
}
