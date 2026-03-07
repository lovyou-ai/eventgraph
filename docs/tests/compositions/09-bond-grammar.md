# Composition Test: Bond Grammar (Layer 9)

Tests for the Bond Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph
actors: [alice (Human), bob (Human)]
trust: alice→bob = 0.5 (moderate)
grammar: BondGrammar
```

## Operation Tests

### Connect

**Input:** `grammar.Connect({ actor_a: alice, actor_b: bob, context: "project collaborators" })`
**Assertions:**
- Attachment primitive activated
- Mutual Subscribe created
- Initial attachment strength and quality assessed
- Connection event links to context

### Balance

**Input:** `grammar.Balance({ between: [alice, bob] })`
**Assertions:**
- Reciprocity primitive evaluates give/take ratio
- Balance Weight computed (-1.0 to 1.0, 0 = balanced)
- Direction identified (who gives more)

### Deepen

**Input:** `grammar.Deepen({ between: [alice, bob] })` after extended interaction history.
**Assertions:**
- Relational Trust assessed (deeper than L0 transactional trust)
- Vulnerability Score computed
- Requires sufficient history (not instant)
- Consent from both parties

### Open

**Input:** `grammar.Open({ actor: alice, with: bob, domain: "career concerns", depth: 0.7 })`
**Assertions:**
- Vulnerability primitive activated
- Channel is Transient (ephemeral sharing)
- Private modifier (only alice and bob see it)
- Depth Score reflects the significance of the disclosure

### Attune / Feel-With

**Input:** `grammar.Attune({ observer: bob, subject: alice })` → `grammar.FeelWith({ actor: bob, toward: alice, context: harm_event })`
**Assertions:**
- Attune builds understanding model with accuracy Score
- Feel-With expresses empathy in response to specific context
- Both require existing relationship (Connect first)

### Break

**Input:** `grammar.Break({ between: [alice, bob], cause: "public criticism", severity: "high" })`
**Assertions:**
- Rupture primitive activated
- Trust drops sharply
- Repairable flag assessed
- Cause event linked

### Apologize / Reconcile

**Input:** `grammar.Apologize({ from: bob, to: alice, harm: criticism_event })` → `grammar.Reconcile({ between: [alice, bob] })`
**Assertions:**
- Apology acknowledges specific harm
- Reconciliation is gradual (progress Score 0→1)
- Trust rebuilds slowly
- New basis established

### Mourn

**Input:** `grammar.Mourn({ actor: alice, lost: bob, relationship: "collaborator" })` (after bob's departure)
**Assertions:**
- Loss primitive processes the permanent end
- Impact severity assessed
- Linked to the actor's memorial event

## Named Function Tests

### Betrayal-Repair

**Input:** Full cycle: Break → Apologize → Reconcile → Deepen.
**Assertions:**
- Relationship stronger after repair (Growth primitive)
- Full event chain traversable
- Trust trajectory: high → crash → gradual rebuild → higher

### Check-In

**Input:** `grammar.CheckIn({ between: [alice, bob] })`
**Assertions:**
- Balance + Attune + Feel-With
- Regular health assessment of the relationship
- No action required if healthy

### Forgive

**Input:** `grammar.Forgive({ between: [alice, bob] })` after Sever.
**Assertions:**
- Subscribe re-established after Sever
- History intact (not erased)
- Trust starts rebuilding from low but non-zero base

## Error Cases

| Case | Expected |
|------|----------|
| Open without existing Connect | `Err(ValidationError.NoRelationship)` |
| Apologize for someone else's harm | `Err(ValidationError.NotResponsible)` |
| Reconcile without Apology | Valid but harder (progress slower) |
| Break an already-ruptured relationship | Idempotent (no error, no duplicate) |

## Reference

- `docs/compositions/09-bond.md` — Bond Grammar specification
- `docs/layers/09-relationship.md` — Layer 9 derivation
- `docs/tests/primitives/03-consent-journal.md` — Integration test scenario
