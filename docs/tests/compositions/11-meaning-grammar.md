# Composition Test: Meaning Grammar (Layer 11)

Tests for the Meaning Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph with community and culture history
actors: [sage_alice (Human), newcomer_bob (Human), system_agent (AI)]
grammar: MeaningGrammar
```

## Operation Tests

### Examine

**Input:** `grammar.Examine({ actor: system_agent })`
**Assertions:**
- SelfAwareness primitive identifies blind spots and assumptions
- Report includes limitations
- Honest about what the system doesn't know

### Reframe

**Input:** `grammar.Reframe({ situation: decision_id, from: "performance optimization", to: "user experience" })`
**Assertions:**
- Perspective primitive activated
- Derive event creates reframed view
- Original perspective preserved (not replaced)
- Insight captured

### Question

**Input:** `grammar.Question({ target: tradition_id, challenge: "why do we always do it this way?" })`
**Assertions:**
- Critique primitive activated
- Challenge operation on the target
- Alternative may be proposed (optional)

### Distill

**Input:** `grammar.Distill({ experience: [event_1, event_2, event_3] })`
**Assertions:**
- Wisdom primitive extracts insight
- Derive from experience events
- Confidence Score reflects depth of evidence
- Insight is concise, not just a summary

### Beautify / Liken / Lighten

**Input:** Expression operations.
**Assertions:**
- Beautify recognizes elegance with criteria
- Liken creates metaphor with explanatory power Score
- Lighten detects incongruity and humor
- All produce events (culture is recorded)

### Teach / Translate / Prophesy

**Input:** Transmission operations.
**Assertions:**
- Teach opens a Channel with structured knowledge transfer
- Translate adapts meaning across cultural boundaries with fidelity Score
- Prophesy extrapolates trends with stated confidence and basis

## Named Function Tests

### Post-Mortem

**Input:** `grammar.PostMortem({ incident: incident_id })`
**Assertions:**
- Examine + Question + Distill
- Blind spots identified
- Assumptions questioned
- Actionable wisdom extracted

### Cultural-Onboarding

**Input:** `grammar.CulturalOnboarding({ newcomer: bob, community: community_id })`
**Assertions:**
- Translate (Cross-Cultural) + Teach + Examine (newcomer's perspective)
- Implicit norms made explicit
- Newcomer's outside perspective valued (not just assimilated)

## Error Cases

| Case | Expected |
|------|----------|
| Distill with no experience events | `Err(ValidationError.InsufficientBasis)` |
| Translate to same culture (no boundary) | No-op (returns original) |
| Prophesy with zero confidence | Valid but flagged as speculative |

## Reference

- `docs/compositions/11-meaning.md` — Meaning Grammar specification
- `docs/layers/11-culture.md` — Layer 11 derivation
