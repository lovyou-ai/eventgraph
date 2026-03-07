# Composition Test: Market Grammar (Layer 2)

Tests for the Market Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph
actors: [seller_alice (Human), buyer_bob (Human), escrow_system (System)]
trust: alice→bob = 0.3 (low, new relationship)
grammar: MarketGrammar
```

## Operation Tests

### List

**Input:** `grammar.List({ description: "Code review, $100/hr", actor: alice })`
**Assertions:**
- Emits `offer.created` event
- Offer primitive activated
- Listing is discoverable via query

### Bid

**Input:** `grammar.Bid({ listing: listing_id, counter: "$90/hr", actor: bob })`
**Assertions:**
- Emits `offer.counter` event caused by the listing event
- Offer primitive tracks the counter
- Listing author (alice) is notified

### Inquire

**Input:** `grammar.Inquire({ listing: listing_id, question: "What's the codebase size?", actor: bob })`
**Assertions:**
- Emits clarification event as Response to listing
- Clarification primitive activated
- Alice can Respond to the inquiry

### Negotiate

**Input:** `grammar.Negotiate({ listing: listing_id, parties: [alice, bob] })`
**Assertions:**
- Opens a Channel between parties
- Negotiation primitive tracks state
- Channel is private (only alice and bob)

### Accept

**Input:** `grammar.Accept({ offer: bid_id, actor: alice })`
**Assertions:**
- Requires bilateral Consent (alice accepts bob's bid)
- Emits `acceptance.confirmed` event
- Creates Obligation events for both parties
- Trust between parties starts accumulating

### Decline

**Input:** `grammar.Decline({ offer: bid_id, actor: alice })`
**Assertions:**
- Emits `acceptance.rejected` event
- Bid is closed
- No obligations created

### Invoice / Pay / Deliver / Confirm

**Input:** Full exchange cycle after Accept.
**Assertions:**
- Invoice creates formal obligation event
- Deliver marks service obligation as fulfilled
- Pay marks financial obligation as fulfilled
- Confirm acknowledges receipt
- Each step has causal link to previous
- Obligation primitive tracks fulfilled/pending

### Rate

**Input:** `grammar.Rate({ exchange: exchange_id, score: 5, comment: "excellent", actor: bob })`
**Assertions:**
- Emits Gratitude event + Endorsement edge
- Reputation updated for alice
- Trust score between bob→alice increases
- Rating is causally linked to the exchange

### Dispute

**Input:** `grammar.Dispute({ obligation: obligation_id, reason: "work not delivered", actor: bob })`
**Assertions:**
- Emits `breach.detected` event
- Challenge flag on the obligation
- Trust between parties decreases
- Triggers authority request if configured

### Escrow / Release

**Input:** `grammar.Escrow({ amount: "$900", parties: [alice, bob] })` then `grammar.Release({ escrow: escrow_id })`
**Assertions:**
- Escrow creates held obligation delegated to escrow_system
- Release requires bilateral Consent or authority ruling
- Released funds create obligation fulfillment event

## Named Function Tests

### Auction

**Input:** `grammar.Auction({ listing: listing_id, bids: [bid_1, bid_2, bid_3], duration: "24h" })`
**Assertions:**
- Multiple bids accepted
- Highest bid wins (Accept called for winner)
- Other bids are Declined
- Timed modifier enforces deadline

### Milestone

**Input:** `grammar.Milestone({ contract: contract_id, stages: [{deliver: "auth", pay: "$300"}, {deliver: "api", pay: "$600"}] })`
**Assertions:**
- Staged Deliver + Pay pairs
- Each stage requires Confirm before next unlocks
- Partial completion is valid (stops at stage 2)

### Reputation-Transfer

**Input:** `grammar.ReputationTransfer({ actor: alice, from_market: "freelance", to_market: "consulting" })`
**Assertions:**
- Endorsement chain is portable
- Receiving market sees alice's rating history
- Trust scores are context-weighted (not 1:1 transfer)

## Error Cases

| Case | Expected |
|------|----------|
| Bid on expired listing (Timed modifier) | `Err(ValidationError.Expired)` |
| Accept own listing | `Err(ValidationError.SelfDeal)` |
| Pay without corresponding obligation | `Err(ValidationError.NoObligation)` |
| Rate without completing exchange | `Err(ValidationError.ExchangeIncomplete)` |
| Double-release escrow | `Err(ValidationError.AlreadyReleased)` |

## Reference

- `docs/compositions/02-market.md` — Market Grammar specification
- `docs/layers/02-exchange.md` — Layer 2 derivation
- `docs/tests/primitives/02-freelancer-reputation.md` — Integration test scenario
