# Layer 3: Society

## Derivation

### The Gap

Layer 2 models dyadic exchange — two actors communicating, negotiating, and forming agreements. But when a third actor joins, new phenomena appear that dyads can't express. What's acceptable isn't just what two people agree on — it's what the GROUP considers normal. Roles, norms, reputation within a community, inclusion, and exclusion don't exist at the dyad level.

**Test:** Can you express "The community considers this behavior unacceptable" in Layer 2? You can have individual Breaches and bilateral Contracts, but the concept of a norm — a shared expectation held by a group, not derived from any specific agreement — has no Layer 2 representation. Norms emerge from groups, not pairs.

### The Transition

**Dyad → Group**

Two actors in relation become N actors forming a collective. The fundamental new capacity: state that belongs to a group rather than to individuals or pairs.

### Base Operations

What can a group DO that a dyad cannot?

1. **Norm** — establish shared expectations not derived from bilateral agreement
2. **Role** — assign social positions with specific expectations
3. **Include/Exclude** — control group membership
4. **Reputation** — establish standing within a community

### Semantic Dimensions

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| **Scope of effect** | Individual (about one member) / Collective (about the group) | Does this affect one person or everyone? |
| **Formality** | Informal (implicit) / Formal (explicit, recorded) | Is this a tacit understanding or a stated rule? |
| **Direction** | Centripetal (toward group) / Centrifugal (toward individual) | Does this bring people together or push them apart? |
| **Temporality** | Emergent (from patterns) / Declared (from authority) | Did this arise naturally or was it established by decree? |

### Decomposition

**Group 0 — Norms** (collective expectations)

| Primitive | Effect | Formality | Direction | Temporality | What it does |
|-----------|--------|-----------|-----------|-------------|--------------|
| **Norm** | Collective | Formal | Centripetal | Declared | Shared behavioral expectation |
| **Convention** | Collective | Informal | Centripetal | Emergent | Pattern that becomes expected |
| **Taboo** | Collective | Informal | Centrifugal | Emergent | Strong negative norm — "never do this" |
| **Sanction** | Individual | Formal | Centrifugal | Declared | Consequence for norm violation |

Norm lifecycle: patterns form (Convention) → become expectations (Norm) → strong negatives crystallize (Taboo) → violations have consequences (Sanction). This is how societies regulate behavior.

**Group 1 — Roles** (social positions)

| Primitive | Effect | Formality | Direction | Temporality | What it does |
|-----------|--------|-----------|-----------|-------------|--------------|
| **Role** | Individual | Formal | Centripetal | Declared | Named position with expectations |
| **Status** | Individual | Informal | Centripetal | Emergent | Standing within the group |
| **Influence** | Individual | Informal | Centripetal | Emergent | Capacity to affect group decisions |
| **Succession** | Individual | Formal | Centripetal | Declared | Transfer of role from one actor to another |

Role lifecycle: roles exist (Role) → actors have standing (Status) → standing creates influence (Influence) → roles change hands (Succession).

**Group 2 — Membership** (group boundary)

| Primitive | Effect | Formality | Direction | Temporality | What it does |
|-----------|--------|-----------|-----------|-------------|--------------|
| **Inclusion** | Individual | Formal | Centripetal | Declared | Welcoming new members |
| **Exclusion** | Individual | Formal | Centrifugal | Declared | Removing members |
| **Reputation** | Individual | Informal | Centripetal | Emergent | Community standing from track record |
| **Solidarity** | Collective | Informal | Centripetal | Emergent | Group cohesion and mutual support |

Membership lifecycle: join (Inclusion) → build standing (Reputation) → group coheres (Solidarity) → or member is removed (Exclusion). Solidarity is the collective counterpart to individual Reputation.

### Gap Analysis

| Behavior | Maps to | Notes |
|----------|---------|-------|
| "We don't do that here" | Taboo | |
| "Everyone reviews code before merging" | Norm or Convention | Depends on whether it was declared |
| "Alice is a moderator" | Role | |
| "Bob has high standing in the community" | Status | |
| "Carol's opinion carries weight" | Influence | |
| "Bob takes over as team lead" | Succession | |
| "Welcome to the community" | Inclusion | |
| "You're banned" | Exclusion | |
| "Alice has completed 50 projects with high ratings" | Reputation | |
| "This group supports its members" | Solidarity | |
| "Violating the code of conduct results in suspension" | Sanction | |
| Community votes on a proposal | Norm + L1.Permission + L0.Authority | Cross-layer composition |
| Formation of a subgroup | Inclusion (subgroup) | Nested groups |
| Online community moderation | Norm + Sanction + Exclusion | Composition |

**No gaps found.**

### Completeness Argument

1. **Dimensional coverage:** All meaningful combinations covered. The key distinction is formal/informal crossed with individual/collective — this generates norms (collective formal), conventions (collective informal), roles (individual formal), and status (individual informal).

2. **Sociological coverage:** The four pillars of social structure from sociology — norms, roles, membership, and stratification — are all represented. Stratification emerges from Status + Influence + Reputation without needing a separate primitive.

3. **Layer boundary:** None of these require concepts from Layer 4 (Legal). Society has informal norms and sanctions, but not formal adjudication, precedent, or jurisdiction. Those are Layer 4's gap.

---

## Primitive Specifications

Full specifications in `docs/primitives.md` (Layer 3 section).

## Product Graph

Layer 3 maps to the **Social Graph** — user-owned social platform where communities set norms. See `docs/product-layers.md`.

## Reference

- `docs/derivation-method.md` — The derivation method
- `docs/primitives.md` — Full specifications
- `docs/product-layers.md` — Social Graph
- `docs/tests/primitives/04-community-governance.md` — Related integration test scenario
