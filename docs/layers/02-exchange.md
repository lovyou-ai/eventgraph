# Layer 2: Exchange

## Derivation

### The Gap

Layer 1 gives actors goals, plans, attention, and autonomy. An actor can decide, delegate, and track commitment. But all of this is unilateral — one actor acting on the world. When two actors interact, Layer 1 has delegation ("you do this for me") but no concept of reciprocity ("I'll do this IF you do that"). Offer, acceptance, obligation, and fairness don't exist.

**Test:** Can you express "Alice offers Bob $100 for a code review, Bob accepts, Alice is now obligated to pay" in Layer 1? Alice can set a Goal, delegate to Bob, and track Commitment. But the concepts of "offer" (conditional on acceptance), "acceptance" (binding bilateral agreement), and "obligation" (created by the agreement, not by delegation) have no Layer 1 representation. Delegation is unilateral; exchange is bilateral.

### The Transition

**Individual → Dyad**

A single actor becomes two actors in relation. The fundamental new capacity: operations that require two participants and create mutual state.

### Base Operations

What can a dyad DO that an individual cannot?

1. **Communicate** — send structured information to a specific recipient
2. **Propose** — make a conditional offer
3. **Agree** — create mutual binding state
4. **Owe** — create asymmetric obligation from agreement

### Semantic Dimensions

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| **Symmetry** | Symmetric (both parties equal) / Asymmetric (one gives, one receives) | Are both parties doing the same thing? |
| **Binding force** | Informational (no obligation) / Conditional (if accepted) / Binding (obligation created) | What happens after this operation? |
| **Completeness** | Partial (in progress) / Complete (resolved) | Is the exchange still open? |
| **Valence** | Positive (value added) / Neutral (information) / Negative (cost, debt) | Does this create value, information, or obligation? |

### Decomposition

**Group 0 — Communication** (informational, symmetric)

| Primitive | Symmetry | Binding | Completeness | Valence | What it does |
|-----------|----------|---------|-------------|---------|--------------|
| **Message** | Asymmetric | Informational | Partial | Neutral | Send structured information |
| **Acknowledgement** | Asymmetric | Informational | Complete | Neutral | Confirm receipt and understanding |
| **Clarification** | Symmetric | Informational | Partial | Neutral | Resolve ambiguity |
| **Context** | Symmetric | Informational | Partial | Neutral | Maintain shared conversational state |

Communication is the substrate of exchange. You can't negotiate if you can't talk. These four primitives cover: send → confirm → clarify → accumulate understanding.

**Group 1 — Reciprocity** (conditional → binding)

| Primitive | Symmetry | Binding | Completeness | Valence | What it does |
|-----------|----------|---------|-------------|---------|--------------|
| **Offer** | Asymmetric | Conditional | Partial | Positive | Propose something to another |
| **Acceptance** | Asymmetric | Binding | Complete | Positive | Accept or reject a proposal |
| **Obligation** | Asymmetric | Binding | Partial | Negative | Track what actors owe each other |
| **Gratitude** | Asymmetric | Informational | Complete | Positive | Recognise fulfilment beyond expectation |

Reciprocity lifecycle: propose → accept/reject → owe → fulfil → gratitude. Gratitude closes the loop by recognising extraordinary fulfilment, strengthening the relationship edge.

**Group 2 — Agreement** (bilateral, binding)

| Primitive | Symmetry | Binding | Completeness | Valence | What it does |
|-----------|----------|---------|-------------|---------|--------------|
| **Negotiation** | Symmetric | Conditional | Partial | Neutral | Iterative proposal refinement |
| **Contract** | Symmetric | Binding | Complete | Positive | Formalised mutual commitment |
| **Breach** | Asymmetric | Binding | Partial | Negative | Detect when obligations are violated |
| **Resolution** | Symmetric | Binding | Complete | Neutral | Resolve disputes about agreements |

Agreement lifecycle: negotiate → contract → (breach? → resolution) or fulfilment. These primitives handle the formal side of exchange.

### Gap Analysis

| Behavior | Maps to | Notes |
|----------|---------|-------|
| Alice messages Bob | Message | |
| Bob confirms he received Alice's message | Acknowledgement | |
| Bob asks "what did you mean by X?" | Clarification | |
| Alice offers to sell her car for $5000 | Offer | |
| Bob says "I'll take it" | Acceptance | |
| Alice owes Bob the car; Bob owes Alice $5000 | Obligation (two instances) | |
| Bob says "thanks, car runs great" | Gratitude | |
| Alice and Bob haggle on price | Negotiation | |
| They sign a purchase agreement | Contract | |
| Bob doesn't pay | Breach | |
| They go through dispute resolution | Resolution | |
| Shared understanding of conversation | Context | |
| Bartering goods | Offer + Acceptance + Obligation | Composition |
| Escrow | Obligation + Contract + Delegation (from L1) | Cross-layer composition |
| Reputation from exchange | trust.updated (from L0) via Gratitude | L0 primitives |

**No gaps found.** Escrow and complex financial instruments are compositions, not missing primitives.

### Completeness Argument

1. **Dimensional coverage:** The {symmetry, binding, completeness, valence} space is covered. Degenerate combinations (e.g., symmetric + binding + complete + negative = mutual breach, which is just two Breach events) don't require new primitives.

2. **Lifecycle coverage:** Communication: send → confirm → clarify → context. Reciprocity: offer → accept → obligate → gratitude. Agreement: negotiate → contract → breach/resolution. All three lifecycles are complete.

3. **Layer boundary:** None of these primitives require concepts from Layer 3 (Society). Exchange is about dyads — two actors. Group norms, roles, and collective decision-making are Layer 3's gap.

---

## Primitive Specifications

Full specifications in `docs/primitives.md` (Layer 2 section).

## Product Graph

Layer 2 maps to the **Market Graph** — trust-based marketplace eliminating platform tolls. See `docs/product-layers.md`.

## Reference

- `docs/derivation-method.md` — The derivation method
- `docs/primitives.md` — Full specifications
- `docs/product-layers.md` — Market Graph
- `docs/tests/primitives/02-freelancer-reputation.md` — Integration test scenario
