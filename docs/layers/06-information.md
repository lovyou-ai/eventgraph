# Layer 6: Information

## Derivation

### The Gap

Layer 5 builds artefacts and processes, but everything is concrete — specific events, specific actions, specific tools. There's no concept of symbolic representation, abstraction, or meaning independent of the physical events that carry it. A system can build a database but cannot reason about what "database" means as a category, cannot generalise from examples to principles, and cannot track whether its knowledge is accurate.

**Test:** Can you express "The pattern we've seen in three projects generalises to a principle about distributed systems, and our earlier belief about consistency was wrong" in Layer 5? You can create artefacts, test them, and iterate. But "generalises to a principle" (abstraction), "belief" (knowledge claim), and "was wrong" (correction of knowledge) have no Layer 5 representation. Building is not knowing.

### The Transition

**Physical → Symbolic**

Concrete artefacts become abstract knowledge. The fundamental new capacity: creating symbols that stand for things, forming abstractions from instances, establishing facts, and reasoning about truth.

### Base Operations

What can a knower DO that a builder cannot?

1. **Represent** — create symbols that stand for things
2. **Generalise** — abstract from specific instances to general principles
3. **Verify** — establish whether claims are true
4. **Correct** — fix errors in what's known

### Semantic Dimensions

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| **Level** | Concrete (specific instances) / Abstract (general principles) | Particular or general? |
| **Validity** | Unverified (claimed) / Verified (confirmed) / Retracted (corrected) | What's the epistemic status? |
| **Direction** | Constructive (building knowledge) / Critical (challenging knowledge) | Adding to or questioning what's known? |
| **Persistence** | Transient (working memory) / Durable (long-term knowledge) | How long does this knowledge persist? |

### Decomposition

**Group 0 — Representation** (how meaning is encoded)

| Primitive | Level | Validity | Direction | Persistence | What it does |
|-----------|-------|----------|-----------|-------------|--------------|
| **Symbol** | Concrete | Unverified | Constructive | Durable | Creating and interpreting symbolic representations |
| **Abstraction** | Abstract | Unverified | Constructive | Durable | Generalising from specifics |
| **Classification** | Both | Unverified | Constructive | Durable | Organising information into categories |
| **Encoding** | Concrete | Unverified | Constructive | Transient | Transforming information between representations |

Representation lifecycle: symbols are created (Symbol) → patterns are generalised (Abstraction) → knowledge is organised (Classification) → and transformed between formats (Encoding). This is how raw events become structured knowledge.

**Group 1 — Knowledge** (what is known)

| Primitive | Level | Validity | Direction | Persistence | What it does |
|-----------|-------|----------|-----------|-------------|--------------|
| **Fact** | Concrete | Verified | Constructive | Durable | Establishing verified claims |
| **Inference** | Abstract | Unverified | Constructive | Durable | Deriving new knowledge from existing facts |
| **Memory** | Both | Verified | Constructive | Durable | Long-term knowledge retention and retrieval |
| **Learning** | Abstract | Verified | Constructive | Durable | Updating behaviour based on experience |

Knowledge lifecycle: claims are verified (Fact) → new knowledge is derived (Inference) → important knowledge is retained (Memory) → and behaviour changes (Learning). Learning is the bridge from knowing to doing differently.

**Group 2 — Truth** (what is real)

| Primitive | Level | Validity | Direction | Persistence | What it does |
|-----------|-------|----------|-----------|-------------|--------------|
| **Narrative** | Abstract | Unverified | Constructive | Durable | Constructing coherent stories from events |
| **Bias** | Abstract | Verified | Critical | Durable | Detecting systematic distortions |
| **Correction** | Concrete | Retracted | Critical | Durable | Fixing errors in the knowledge base |
| **Provenance** | Concrete | Verified | Critical | Durable | Tracking origin and chain of custody |

Truth lifecycle: stories are constructed (Narrative) → systematic distortions are detected (Bias) → errors are fixed (Correction) → and sources are tracked (Provenance). The critical direction primitives (Bias, Correction, Provenance) are what prevent knowledge from becoming dogma.

### Gap Analysis

| Behavior | Maps to | Notes |
|----------|---------|-------|
| "Let X represent the user's trust level" | Symbol | |
| "All three cases follow the same pattern" | Abstraction | |
| "This is a type-3 security issue" | Classification | |
| "Convert the report to JSON format" | Encoding | |
| "This claim is supported by evidence A, B, C" | Fact | |
| "From facts X and Y, we can conclude Z" | Inference | |
| "Store this for later — it's important" | Memory | |
| "After the outage, we changed our monitoring" | Learning | |
| "Here's what happened and why" | Narrative | |
| "The training data over-represents group A" | Bias | |
| "Our earlier assessment was wrong, here's the fix" | Correction | |
| "This claim originated from source S" | Provenance | |
| Knowledge graph construction | Symbol + Classification + Fact | Composition |
| Scientific method | Fact + Inference + Correction + Provenance | Composition |
| Disinformation detection | Narrative + Bias + Provenance | Composition |

**No gaps found.**

### Completeness Argument

1. **Dimensional coverage:** The {level, validity, direction, persistence} space is covered. The constructive/critical distinction ensures the system can both build knowledge and challenge it — without the critical dimension, knowledge ossifies.

2. **Epistemological coverage:** The three pillars of epistemology — representation (how do we encode knowledge?), justification (how do we know it's true?), and error correction (what do we do when we're wrong?) — map to the three groups.

3. **Layer boundary:** None of these require concepts from Layer 7 (Ethics). Information models what *is* — facts, knowledge, truth. Whether those facts are *good* or *just* is Layer 7's gap. A system can know that an action occurred and violated a rule without reasoning about whether the rule itself is just.

---

## Primitive Specifications

Full specifications in `docs/primitives.md` (Layer 6 section).

## Product Graph

Layer 6 maps to the **Knowledge Graph** — verified, provenanced knowledge with bias detection. See `docs/product-layers.md`.

## Reference

- `docs/derivation-method.md` — The derivation method
- `docs/primitives.md` — Full specifications
- `docs/product-layers.md` — Knowledge Graph
- `docs/tests/primitives/06-research-integrity.md` — Related integration test scenario
