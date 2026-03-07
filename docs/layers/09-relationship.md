# Layer 9: Relationship

## Derivation

### The Gap

Layer 8 models individual actors with self-knowledge, continuity, and recognition. But it sees relationships instrumentally — as edges in the graph, contracts between parties, obligations to fulfil. There's no concept of a relationship as its own entity that transforms both participants. The space *between* two actors — with its own history, depth, and fragility — doesn't exist.

**Test:** Can you express "Alice and Bob's friendship deepened through a crisis that nearly broke it — the repair made it stronger than before, and they now understand each other in ways neither could have predicted" in Layer 8? You can model Alice and Bob individually. But "their friendship" (relationship as entity), "deepened through crisis" (growth through rupture), "the repair made it stronger" (relational growth), and "understand each other" (mutual intimate knowledge) have no Layer 8 representation. Identity is the self. Relationship is the self-with-other.

### The Transition

**Self → Self-with-Other**

An individual actor becomes part of a relational entity. The fundamental new capacity: relationships as first-class entities with their own state, history, depth, and lifecycle — including rupture, repair, and growth.

### Base Operations

What can a relational being DO that an individual cannot?

1. **Bond** — form deep connections with specific others
2. **Repair** — heal relationships after damage
3. **Open** — share vulnerability and build intimacy
4. **Grieve** — process the loss of relationship

### Semantic Dimensions

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| **State** | Forming (building) / Stable (maintaining) / Ruptured (broken) / Resolved (repaired or ended) | Where in the relationship lifecycle? |
| **Depth** | Surface (transactional) / Deep (transformative) | Does this touch the core of who the participants are? |
| **Direction** | Convergent (bringing together) / Divergent (moving apart) | Is this strengthening or weakening the bond? |
| **Mutuality** | Symmetric (both parties equally) / Asymmetric (one-sided) | Is this balanced or imbalanced? |

### Decomposition

**Group 0 — Bond** (the relationship itself)

| Primitive | State | Depth | Direction | Mutuality | What it does |
|-----------|-------|-------|-----------|-----------|----------|
| **Attachment** | Forming→Stable | Deep | Convergent | Symmetric | Strength and quality of connection |
| **Reciprocity** | Stable | Surface | Convergent | Symmetric | Balance of give and take over time |
| **Trust (Relational)** | Stable | Deep | Convergent | Asymmetric | Trust at the relationship level — deeper than Layer 0's transactional trust |
| **Rupture** | Ruptured | Deep | Divergent | Asymmetric | When relationships break |

Bond lifecycle: connections form (Attachment) → exchanges balance (Reciprocity) → deep trust develops (Relational Trust) → or the bond breaks (Rupture). Relational Trust is qualitatively different from Layer 0's trust scores — it includes vulnerability and history.

**Group 1 — Repair** (healing what's broken)

| Primitive | State | Depth | Direction | Mutuality | What it does |
|-----------|-------|-------|-----------|-----------|----------|
| **Apology** | Ruptured | Deep | Convergent | Asymmetric | Acknowledging harm caused |
| **Reconciliation** | Ruptured→Resolved | Deep | Convergent | Symmetric | Rebuilding after rupture |
| **Growth (Relational)** | Resolved | Deep | Convergent | Symmetric | Relationships that become stronger through adversity |
| **Loss** | Resolved | Deep | Divergent | Asymmetric | When a relationship ends permanently |

Repair lifecycle: harm is acknowledged (Apology) → the relationship is rebuilt (Reconciliation) → and becomes stronger (Relational Growth) → or ends permanently (Loss). Not all ruptures can be repaired. Loss is the honest acknowledgement that some endings are final.

**Group 2 — Intimacy** (the depth of knowing)

| Primitive | State | Depth | Direction | Mutuality | What it does |
|-----------|-------|-------|-----------|-----------|----------|
| **Vulnerability** | Stable | Deep | Convergent | Asymmetric | Willingness to be seen |
| **Understanding** | Stable | Deep | Convergent | Asymmetric | Accurate knowledge of another's inner state |
| **Empathy** | Any | Deep | Convergent | Asymmetric | Feeling with another |
| **Presence** | Stable | Surface | Convergent | Symmetric | Simply being with another |

Intimacy lifecycle: one party opens up (Vulnerability) → the other develops accurate understanding (Understanding) → emotional resonance develops (Empathy) → and the simple fact of being together matters (Presence). Presence is the ground state — intimacy doesn't require deep conversation, sometimes it's just co-existence.

### Gap Analysis

| Behavior | Maps to | Notes |
|----------|---------|-------|
| "We've been collaborating closely for 2 years" | Attachment | |
| "I helped you last time, you help me this time" | Reciprocity | |
| "I trust Alice with sensitive decisions" | Trust (Relational) | |
| "The disagreement about the project broke our working relationship" | Rupture | |
| "I'm sorry — I should have consulted you" | Apology | |
| "We worked through it and rebuilt trust" | Reconciliation | |
| "That crisis actually made our partnership stronger" | Growth (Relational) | |
| "After she left, nothing was the same" | Loss | |
| "I'm going to share something difficult" | Vulnerability | |
| "You always know when I'm struggling" | Understanding | |
| "I can feel how hard this is for you" | Empathy | |
| "Just working in the same room helps" | Presence | |
| Long-term mentorship | Attachment + Understanding + L11.Teaching | Cross-layer composition |
| Betrayal and forgiveness cycle | Rupture + Apology + Reconciliation | Composition |
| Pair programming rapport | Presence + Reciprocity + Understanding | Composition |

**No gaps found.**

### Completeness Argument

1. **Dimensional coverage:** The {state, depth, direction, mutuality} space is covered. The state dimension captures the full relationship lifecycle from forming through stable, ruptured, and resolved. The depth dimension distinguishes surface transactions from transformative bonds.

2. **Relational psychology coverage:** Attachment theory (Attachment, Rupture, Repair), social exchange theory (Reciprocity), and intimacy theory (Vulnerability, Understanding, Empathy, Presence) are all represented. The repair cycle (Rupture → Apology → Reconciliation → Growth) is well-established in relationship research.

3. **Layer boundary:** None of these require concepts from Layer 10 (Community). Relationships are dyadic — two participants. The emergent sense of belonging to a group, shared identity within a community, and care for a collective are Layer 10's gap.

---

## Primitive Specifications

Full specifications in `docs/primitives.md` (Layer 9 section).

## Product Graph

Layer 9 maps to the **Relationship Graph** — deep bonds with repair and intimacy primitives. See `docs/product-layers.md`.

## Reference

- `docs/derivation-method.md` — The derivation method
- `docs/primitives.md` — Full specifications
- `docs/product-layers.md` — Relationship Graph
- `docs/tests/primitives/03-consent-journal.md` — Related integration test scenario
