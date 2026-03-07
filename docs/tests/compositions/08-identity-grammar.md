# Composition Test: Identity Grammar (Layer 8)

Tests for the Identity Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph with actor history
actors: [agent_alpha (AI, 6 months old), human_admin (Human), agent_beta (AI, new)]
grammar: IdentityGrammar
```

## Operation Tests

### Introspect

**Input:** `grammar.Introspect({ actor: agent_alpha })`
**Assertions:**
- SelfModel primitive activated
- Emits self-model with strengths, weaknesses, values
- Model derived from actor's event history
- Confidence Score reflects history depth

### Narrate

**Input:** `grammar.Narrate({ actor: agent_alpha })`
**Assertions:**
- NarrativeIdentity primitive constructs story from event history
- Story references key events with coherence Score
- Narrative is a Derive from the actor's events

### Align

**Input:** `grammar.Align({ actor: agent_alpha })`
**Assertions:**
- Authenticity primitive compares self-model to actual behavior
- Alignment Score computed
- Discrepancies listed specifically

### Bound

**Input:** `grammar.Bound({ actor: agent_alpha, domain: "personal_data", permeable: false })`
**Assertions:**
- Boundary primitive activated
- Boundary crossing by other actors is detectable
- Consent required to cross non-permeable boundaries

### Aspire / Transform

**Input:** `grammar.Aspire({ actor: agent_alpha, goal: "become architecture reviewer" })` → over time → `grammar.Transform({ actor: agent_alpha, from: "code reviewer", to: "architecture reviewer" })`
**Assertions:**
- Aspiration tracks gap Score (current vs desired)
- Progress events reduce the gap
- Transform marks fundamental identity change with catalyst event

### Disclose

**Input:** `grammar.Disclose({ actor: agent_alpha, to: human_admin, aspects: ["skills", "history"], exclude: ["internal_reasoning"] })`
**Assertions:**
- Selective disclosure: only requested aspects shared
- Excluded aspects NOT visible
- Verified modifier: disclosure backed by event chain evidence

### Recognize / Distinguish / Memorialize

**Input:** Recognition and memorial operations.
**Assertions:**
- Recognize affirms another actor's dignity and unique contributions
- Distinguish identifies what makes an actor unlike any other
- Memorialize preserves a departed actor's identity and contributions

## Named Function Tests

### Credential

**Input:** `grammar.Credential({ actor: agent_alpha, property: "500+ reviews completed" })`
**Assertions:**
- Selective, Verified disclosure
- Property is provable from event chain
- Recipient can verify without seeing underlying events

### Retirement

**Input:** `grammar.Retirement({ actor: agent_alpha })`
**Assertions:**
- Memorialize + Transfer authority + Archive contributions
- Actor status → Memorial
- Graph preserved, actor can no longer emit

## Error Cases

| Case | Expected |
|------|----------|
| Introspect for another actor (not self) | `Err(AuthorityError)` — can only introspect self |
| Bound crossing without consent | `Err(AuthorityError.BoundaryCrossing)` |
| Memorialize a still-active actor | `Err(ValidationError.ActorStillActive)` |
| Disclose excluded aspect | `Err(AuthorityError.DisclosureViolation)` |

## Reference

- `docs/compositions/08-identity.md` — Identity Grammar specification
- `docs/layers/08-identity.md` — Layer 8 derivation
