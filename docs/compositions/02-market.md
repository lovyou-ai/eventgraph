# Market Grammar (Layer 2: Exchange)

The grammar for trust-based marketplaces eliminating platform tolls.

## Derivation

Markets are operations on value exchange. The base operations are: **offer value**, **negotiate terms**, **execute exchange**, **assess outcome**. Four semantic dimensions differentiate operations:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| Phase | Pre-agreement / Agreement / Post-agreement | Before, during, or after the deal? |
| Symmetry | One-sided (offer/demand) / Bilateral (mutual) | One party or both? |
| Commitment | Revocable (can withdraw) / Binding (locked in) | Can you back out? |
| Value flow | Outward (giving) / Inward (receiving) / Neutral (information) | Who gets what? |

## Operations (14)

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 1 | **List** | offer/creative | Publish an offering to the market | Offer + Emit |
| 2 | **Bid** | offer/response | Make a counter-offer on a listing | Offer + Respond |
| 3 | **Inquire** | info/request | Ask for clarification about an offering | Clarification + Respond |
| 4 | **Negotiate** | agreement/iterative | Refine terms through back-and-forth | Negotiation + Channel |
| 5 | **Accept** | agreement/binding | Accept terms, creating mutual obligation | Acceptance + Consent |
| 6 | **Decline** | agreement/terminal | Reject an offer, closing negotiation | Acceptance (rejected) + Emit |
| 7 | **Invoice** | obligation/create | Formalize payment obligation | Obligation + Emit |
| 8 | **Pay** | obligation/fulfill | Satisfy a financial obligation | Obligation (fulfilled) + Emit |
| 9 | **Deliver** | obligation/fulfill | Satisfy a service/goods obligation | Obligation (fulfilled) + Emit |
| 10 | **Confirm** | verification/receipt | Acknowledge receipt and satisfaction | Acknowledgement + Emit |
| 11 | **Rate** | reputation/feedback | Provide structured feedback on exchange | Gratitude + Endorse |
| 12 | **Dispute** | breach/initiate | Flag a failed obligation | Breach + Challenge |
| 13 | **Escrow** | obligation/held | Hold value pending conditions | Obligation + Delegate (to escrow actor) |
| 14 | **Release** | obligation/resolve | Release escrowed value on condition | Resolution + Consent |

## Modifiers (3)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| **Timed** | Operation expires after deadline | List, Bid, Escrow |
| **Guaranteed** | Backed by staked reputation | List, Accept, Deliver |
| **Anonymous** | Identity disclosed only on Accept | List, Bid, Inquire |

## Named Functions (7)

| Function | Composition | Purpose |
|----------|------------|---------|
| **Auction** | List + Bid (multiple) + Accept (highest) | Competitive bidding |
| **Barter** | List + Bid (goods, not currency) + Accept | Goods-for-goods exchange |
| **Subscription** | Accept + Pay (recurring) + Deliver (recurring) | Ongoing service agreement |
| **Refund** | Dispute + Resolution + Pay (reversed) | Return value after dispute |
| **Milestone** | Accept + Deliver (partial) + Pay (partial, repeated) | Staged delivery and payment |
| **Reputation-Transfer** | Rate (from multiple exchanges) → portable Endorse chain | Carry reputation across markets |
| **Arbitration** | Dispute + Escrow + Release (per ruling) | Third-party dispute resolution |

## Mapping to Primitives

| Operation | Layer 2 Primitives | Grammar Operations |
|-----------|-------------------|-------------------|
| List | Offer | Emit |
| Bid | Offer | Respond |
| Inquire | Clarification | Respond |
| Negotiate | Negotiation | Channel |
| Accept | Acceptance | Consent |
| Decline | Acceptance (rejected) | Emit |
| Invoice | Obligation | Emit |
| Pay | Obligation (fulfilled) | Emit |
| Deliver | Obligation (fulfilled) | Emit |
| Confirm | Acknowledgement | Emit |
| Rate | Gratitude | Endorse |
| Dispute | Breach | Challenge |
| Escrow | Obligation | Delegate |
| Release | Resolution | Consent |

## Example Flow

**Freelancer engagement:**
```
List("code review, $100/hr, Rust expertise")
  → Inquire("what's the codebase size?")
  → Bid("$90/hr, available next week")
  → Negotiate(Channel: scope, timeline, milestones)
  → Accept(Consent: terms locked)
  → Escrow($900 for 10hr engagement)
  → Deliver("completed review of auth module")
  → Confirm("review accepted, quality excellent")
  → Release(escrow → freelancer)
  → Rate(5/5, "thorough, found critical bug")
```

## Reference

- `docs/grammar.md` — Infrastructure grammar (15 operations)
- `docs/layers/02-exchange.md` — Layer 2 derivation
- `docs/primitives.md` — Layer 2 primitive specifications
- `docs/tests/primitives/02-freelancer-reputation.md` — Integration test scenario
