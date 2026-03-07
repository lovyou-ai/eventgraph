package compositions_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// TestMarketGrammar exercises the Market Grammar (Layer 2: Exchange).
// Operations: List, Bid, Inquire, Negotiate, Accept, Decline,
// Invoice/Pay/Deliver/Confirm, Rate, Dispute, Escrow/Release.
func TestMarketGrammar(t *testing.T) {
	t.Run("ListAndBid", func(t *testing.T) {
		env := newTestEnv(t)
		carol := env.actor("Carol", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		listing, _ := env.grammar.Emit(env.ctx, carol.ID(),
			"listing: web scraping service, $500/mo, unlimited requests",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		bid, _ := env.grammar.Respond(env.ctx, bob.ID(),
			"bid: $400/mo for 6-month commitment",
			listing.ID(), env.convID, signer)

		ancestors := env.ancestors(bid.ID(), 5)
		if !containsEvent(ancestors, listing.ID()) {
			t.Error("bid should trace to listing")
		}
		env.verifyChain()
	})

	t.Run("Inquire", func(t *testing.T) {
		env := newTestEnv(t)
		carol := env.actor("Carol", 1, event.ActorTypeHuman)
		dave := env.actor("Dave", 3, event.ActorTypeHuman)

		listing, _ := env.grammar.Emit(env.ctx, carol.ID(),
			"listing: GraphQL API development, $3000",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		inquiry, _ := env.grammar.Respond(env.ctx, dave.ID(),
			"inquiry: does this include schema design or just resolver implementation?",
			listing.ID(), env.convID, signer)

		clarification, _ := env.grammar.Respond(env.ctx, carol.ID(),
			"clarification: includes full schema design + resolvers + documentation",
			inquiry.ID(), env.convID, signer)

		ancestors := env.ancestors(clarification.ID(), 10)
		if !containsEvent(ancestors, listing.ID()) {
			t.Error("clarification should trace to listing")
		}
		env.verifyChain()
	})

	t.Run("NegotiateAndAccept", func(t *testing.T) {
		env := newTestEnv(t)
		carol := env.actor("Carol", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		listing, _ := env.grammar.Emit(env.ctx, carol.ID(),
			"listing: data pipeline, $5000",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Open negotiation channel
		channel, _ := env.grammar.Channel(env.ctx, carol.ID(), bob.ID(),
			types.Some(types.MustDomainScope("data_pipeline")),
			listing.ID(), env.convID, signer)

		// Negotiate
		offer, _ := env.grammar.Respond(env.ctx, bob.ID(),
			"counter: $4200 with 3-week delivery",
			channel.ID(), env.convID, signer)
		counterOffer, _ := env.grammar.Respond(env.ctx, carol.ID(),
			"counter: $4500, 3 weeks is fine",
			offer.ID(), env.convID, signer)

		// Accept via bilateral consent
		agreement, _ := env.grammar.Consent(env.ctx, carol.ID(), bob.ID(),
			"agreement: data pipeline, $4500, 3-week delivery",
			types.MustDomainScope("data_pipeline"),
			counterOffer.ID(), env.convID, signer)

		ancestors := env.ancestors(agreement.ID(), 10)
		if !containsEvent(ancestors, listing.ID()) {
			t.Error("agreement should trace to listing")
		}
		env.verifyChain()
	})

	t.Run("Decline", func(t *testing.T) {
		env := newTestEnv(t)
		carol := env.actor("Carol", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		listing, _ := env.grammar.Emit(env.ctx, carol.ID(),
			"listing: logo design, $200",
			env.convID, []types.EventID{env.boot.ID()}, signer)
		bid, _ := env.grammar.Respond(env.ctx, bob.ID(),
			"bid: $50", listing.ID(), env.convID, signer)

		decline, _ := env.grammar.Respond(env.ctx, carol.ID(),
			"declined: price too low for quality expected",
			bid.ID(), env.convID, signer)

		ancestors := env.ancestors(decline.ID(), 5)
		if !containsEvent(ancestors, bid.ID()) {
			t.Error("decline should trace to bid")
		}
		env.verifyChain()
	})

	t.Run("DeliverAndConfirm", func(t *testing.T) {
		env := newTestEnv(t)
		carol := env.actor("Carol", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		contract, _ := env.grammar.Consent(env.ctx, carol.ID(), bob.ID(),
			"contract: REST API, $2800",
			types.MustDomainScope("software"),
			env.boot.ID(), env.convID, signer)

		delivery, _ := env.grammar.Derive(env.ctx, bob.ID(),
			"delivered: REST API complete, 47 endpoints",
			contract.ID(), env.convID, signer)

		confirmation, _ := env.grammar.Acknowledge(env.ctx, carol.ID(),
			delivery.ID(), bob.ID(), env.convID, signer)

		ancestors := env.ancestors(confirmation.ID(), 10)
		if !containsEvent(ancestors, contract.ID()) {
			t.Error("confirmation should trace to contract")
		}
		env.verifyChain()
	})

	t.Run("Rate", func(t *testing.T) {
		env := newTestEnv(t)
		carol := env.actor("Carol", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		delivery, _ := env.grammar.Emit(env.ctx, bob.ID(),
			"delivered: project complete",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		// Rate = Endorse with weight (reputation staked)
		endorsement, _ := env.grammar.Endorse(env.ctx, carol.ID(),
			delivery.ID(), bob.ID(), types.MustWeight(0.9),
			types.Some(types.MustDomainScope("software")),
			env.convID, signer)

		content := endorsement.Content().(event.EdgeCreatedContent)
		if content.Weight.Value() != 0.9 {
			t.Errorf("rating weight = %v, want 0.9", content.Weight.Value())
		}
		env.verifyChain()
	})

	t.Run("Dispute", func(t *testing.T) {
		env := newTestEnv(t)
		carol := env.actor("Carol", 1, event.ActorTypeHuman)
		bob := env.actor("Bob", 2, event.ActorTypeHuman)

		contract, _ := env.grammar.Consent(env.ctx, carol.ID(), bob.ID(),
			"contract: mobile app, $5000",
			types.MustDomainScope("software"),
			env.boot.ID(), env.convID, signer)

		delivery, _ := env.grammar.Derive(env.ctx, bob.ID(),
			"delivered: mobile app", contract.ID(), env.convID, signer)

		dispute, _ := env.grammar.Respond(env.ctx, carol.ID(),
			"dispute: app crashes on Android 12, does not meet acceptance criteria",
			delivery.ID(), env.convID, signer)

		// Violation recorded
		_, err := env.graph.Record(
			event.EventTypeViolationDetected, env.system,
			event.ViolationDetectedContent{
				Expectation: contract.ID(),
				Actor:       bob.ID(),
				Severity:    event.SeverityLevelSerious,
				Description: "delivered app crashes on target platform",
				Evidence:    types.MustNonEmpty([]types.EventID{dispute.ID()}),
			},
			[]types.EventID{dispute.ID()}, env.convID, signer)
		if err != nil {
			t.Fatalf("violation: %v", err)
		}
		env.verifyChain()
	})

	t.Run("Auction", func(t *testing.T) {
		env := newTestEnv(t)
		seller := env.actor("Seller", 1, event.ActorTypeHuman)
		bidder1 := env.actor("Bidder1", 2, event.ActorTypeHuman)
		bidder2 := env.actor("Bidder2", 3, event.ActorTypeHuman)

		listing, _ := env.grammar.Emit(env.ctx, seller.ID(),
			"auction: vintage laptop, starting $100",
			env.convID, []types.EventID{env.boot.ID()}, signer)

		bid1, _ := env.grammar.Respond(env.ctx, bidder1.ID(),
			"bid: $150", listing.ID(), env.convID, signer)
		bid2, _ := env.grammar.Respond(env.ctx, bidder2.ID(),
			"bid: $200", listing.ID(), env.convID, signer)

		// Seller accepts highest bid
		_, _ = env.grammar.Consent(env.ctx, seller.ID(), bidder2.ID(),
			"sold: vintage laptop at $200",
			types.MustDomainScope("auction"),
			bid2.ID(), env.convID, signer)

		_ = bid1 // lower bid stays on graph
		env.verifyChain()
	})
}
