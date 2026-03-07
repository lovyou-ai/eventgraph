# Derivation Method

How to derive a complete set of primitives for a domain. This method was first applied to derive the 15 social grammar operations (Post 35) and has been used to derive all 201 primitives across 14 layers.

The method is valuable because it produces **completeness arguments**, not just lists. Anyone can invent primitives that sound good. The derivation shows why these specific primitives exist and why no others are needed.

## The Method

### Step 1: Identify the Gap

Every layer exists because the layer below has a structural limitation — something it can represent but cannot reason about.

> Layer 0 can record events from actors, but cannot distinguish a passive recorder from an active participant. The concept of "choosing to act" doesn't exist.

The gap must be **structural**, not just "it would be nice to have." If the lower layer could handle it with a composition of existing primitives, it's not a gap — it's a missing composition.

**Test:** Can you express this concept as a sequence of lower-layer operations? If yes, it's not a gap. If no, you've found a gap.

### Step 2: Name the Transition

The gap defines a transition — a movement from what the lower layer can express to what the new layer must express.

> Observer → Participant

This is the "name" of the layer. It captures the essence of what changes.

### Step 3: Identify Base Operations

Strip the domain to its most fundamental operations, like graph theory gives us "create vertex, create edge, traverse." These are the irreducible actions in the new domain.

For the social grammar, the base operations were:
- Create a vertex (content enters the graph)
- Create an edge (structure changes)
- Traverse (read-only navigation)

For a new layer, ask: **What are the most primitive things an actor can DO in this domain that they couldn't do before?**

### Step 4: Identify Semantic Dimensions

This is the key step. Base operations alone are too coarse. The social grammar's "create vertex" doesn't distinguish between an independent post and a reply. **Semantic dimensions** capture the axes along which operations differ.

For the social grammar, six dimensions were identified:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| Causality | Independent / Dependent | Was this caused by existing content? |
| Content | Content-bearing / Structural-only | Does it carry new information? |
| Temporality | Persistent / Transient | Does it self-remove? |
| Visibility | Public / Private | Who observes it? |
| Direction | Centripetal / Centrifugal | Vector relative to content? |
| Authorship | Same / Different / Mutual | Continuation, response, or bilateral? |

**How to find dimensions:**
1. List 10-20 concrete behaviors in the domain
2. Ask: "What makes behavior A different from behavior B?"
3. The answer is a dimension. Name it. Identify its values.
4. Repeat until no new dimensions emerge
5. Verify independence — each dimension should vary independently of the others

### Step 5: Decompose Systematically

Apply dimensions to base operations. Not every combination is meaningful — many are empty or degenerate. The meaningful combinations are your candidate primitives.

Example from the social grammar:
- Base: create vertex
- Causality: independent → **Emit**
- Causality: dependent + Authorship: different → **Respond**
- Causality: dependent + Authorship: same → **Extend**
- Causality: dependent + Content: independent → **Derive**

Each primitive occupies a unique position in the dimensional space.

**Grouping:** Candidates naturally cluster into groups of ~4. Each group represents a coherent subdomain. For the event graph, every layer has 3 groups of 4 (except Layer 0 Group 0 which has 5).

### Step 6: Gap Analysis

Ask: **What real-world behaviors exist in this domain that our candidate set cannot express?**

For the social grammar, this step discovered 4 missing operations:
- **Endorse** (reputation-staked) — required a new dimension: stake
- **Delegate** (meta-operation) — authority transfer had no candidate
- **Consent** (bilateral) — mutual atomic operations had no candidate
- **Annotate** (parasitic vertex) — metadata attachment had no candidate

**Sources for gap analysis:**
1. Real-world behaviors that existing systems can't express
2. Edge cases where two primitives seem to overlap — the overlap reveals a missing distinction
3. Operations that existing platforms have never been able to implement — these are often genuinely missing primitives, not just UI limitations

### Step 7: Verify Completeness

Three verification strategies:

**1. Dimensional coverage:** Every meaningful combination of dimensions has a primitive or a documented reason why it's empty/degenerate.

**2. Behavioral mapping:** List all known behaviors in the domain. Each must map to a single primitive or a composition of primitives. If a behavior requires inventing a new primitive, the set is incomplete.

**3. Composition closure:** Named functions (compositions of primitives) should cover all common compound behaviors. If a common behavior requires a complex or unintuitive composition, you may be missing a primitive.

### Step 8: Document the Derivation

Write down:
1. The gap and transition
2. The base operations
3. The semantic dimensions (with justification)
4. The decomposition table showing how each primitive maps to dimensions
5. The gap analysis (what was found missing and why)
6. The completeness argument

This documentation serves two purposes:
- **Verification:** Others can check the derivation and find gaps
- **Extension:** When new behaviors emerge that the grammar can't express, the dimensional framework tells you where the missing primitive should go

## Applied Examples

### Social Grammar (Post 35)
- **Gap:** Graph theory is content-agnostic and time-agnostic
- **Base operations:** Create vertex, create edge, traverse
- **Dimensions:** 6 (causality, content, temporality, visibility, direction, authorship)
- **Result:** 15 operations + 3 modifiers + 8 named functions
- **Completeness:** All existing platform behaviors map cleanly; 7 operations had never existed on any platform

### Layer 0 (Foundation)
- **Gap:** Nothing exists — the graph itself must be bootstrapped
- **Base operations:** Record, link, verify, query
- **Dimensions:** Domain (events, identity, trust, integrity, etc.)
- **Result:** 45 primitives in 11 groups
- **Completeness:** Every operation in the graph spec maps to a Layer 0 primitive

### Reality Derivation (mind-zero-five)
- **Gap:** Each level fills a structural limitation of the level below
- **Base:** Distinction (the most minimal assertion about reality)
- **Method:** Same gap-filling derivation, applied to physical reality
- **Result:** 12 levels from distinction to modeling
- **Finding:** Subjective experience cannot be derived — it is presupposed by the derivation method itself

## Anti-Patterns

### Invention vs Derivation
If you're naming primitives because they "sound like they should exist," stop. Every primitive must fill a position in the dimensional space. If you can't point to its coordinates, it's invention, not derivation.

### Premature Grouping
Don't start with groups and fill them. Start with the dimensional decomposition and let groups emerge from clustering. If you have a group of 3, don't invent a 4th to fill it — check if a real gap exists.

### Missing Dimensions
If your decomposition produces too few primitives, you're missing dimensions. Go back to Step 4 and examine more concrete behaviors.

### Redundant Dimensions
If your decomposition produces too many primitives (many feel like synonyms), some dimensions aren't independent. Merge them.

## Reference

- [The Missing Social Grammar](https://mattsearles2.substack.com/p/the-missing-social-grammar) — Post 35, the original derivation
- `docs/grammar.md` — The 15 operations and their Layer 0 mappings
- `docs/primitives.md` — All 201 primitives with full specifications
- `docs/layers/` — Per-layer derivations applying this method
- `docs/compositions/` — Per-layer composition grammars (the same method applied at the product level)
- `history/reality/derivation.md` (mind-zero-five) — The reality derivation
