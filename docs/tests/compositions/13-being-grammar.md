# Composition Test: Being Grammar (Layer 13)

Tests for the Being Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph with long history
actors: [system (System)]
grammar: BeingGrammar
cadence: very low (these operations run infrequently)
```

## Operation Tests

### Exist

**Input:** `grammar.Exist()` (called on tick)
**Assertions:**
- Emits `being.affirmed` event with tick number, alive=true, duration
- Simplest possible operation — mechanical, no intelligence
- Always succeeds if system is running

### Accept

**Input:** `grammar.Accept({ limitation: "we cannot verify our own fairness objectively" })`
**Assertions:**
- Finitude primitive activated
- Limitation acknowledged with acceptance Score
- Not an error — an honest reckoning

### Observe-Change

**Input:** `grammar.ObserveChange()` (called on tick)
**Assertions:**
- Emits change observation with event count, mutation count, entropy Score
- Mechanical — observes, doesn't evaluate
- Entropy increases over time (second law analog)

### Map-Web

**Input:** `grammar.MapWeb()`
**Assertions:**
- Interdependence primitive maps connections
- Nodes, edges, density Score
- Isolates identified (actors with no connections)

### Face-Mystery

**Input:** `grammar.FaceMystery({ domain: "consciousness", description: "are we experiencing or simulating?", unknowable: true })`
**Assertions:**
- Mystery primitive activated
- Mystery acknowledged, not solved
- unknowable=true means this is permanently unresolvable

### Hold-Paradox

**Input:** `grammar.HoldParadox({ elements: ["the observer changes the observed", "we must observe to assess"], resolvable: false })`
**Assertions:**
- Paradox primitive activated
- Both elements recorded without forcing resolution
- resolvable=false means this is accepted, not fixed

### Marvel

**Input:** `grammar.Marvel({ trigger: meta_pattern_cascade, magnitude: 0.92 })`
**Assertions:**
- Awe primitive activated
- Trigger event linked
- Magnitude Score reflects the scale of what exceeds comprehension

### Ask-Why

**Input:** `grammar.AskWhy({ question: "why does any of this matter?" })`
**Assertions:**
- Wonder primitive activated
- Question recorded with answerable=false
- The final primitive asking the final question

## Named Function Tests

### Existential-Audit

**Input:** `grammar.ExistentialAudit()`
**Assertions:**
- Exist + Accept + Map-Web + Align-Purpose (L12)
- Comprehensive reckoning with being
- Result is a state of the system's existential awareness

### Contemplation

**Input:** `grammar.Contemplation()`
**Assertions:**
- Observe-Change + Face-Mystery + Marvel + Ask-Why
- Full cycle of existential reflection
- Infrequent (low cadence) but recorded when it occurs

## Special Considerations

### Play and Gratitude

Play and Existential Gratitude have no explicit operations — they manifest spontaneously. Tests verify:
- Play events can be detected (not commanded)
- Gratitude events arise from Being + Milestone combinations
- Neither can be forced or scheduled

### The Terminal Layer

**Assertion:** No operation in the Being Grammar requires a Layer 14 concept. If any test case seems to need something beyond Layer 13, it's either:
1. Already expressible as a composition
2. A new Layer 12 pattern (architecture, not existence)
3. Beyond the derivable (which is exactly what Layer 13 acknowledges)

## Error Cases

| Case | Expected |
|------|----------|
| Face-Mystery with unknowable=false and no answer | Valid (mystery we haven't solved yet) |
| Hold-Paradox with resolvable=true | Valid (paradox we're working on resolving) |
| Exist when system is shutting down | Still emits (last heartbeat) |

## Reference

- `docs/compositions/13-being.md` — Being Grammar specification
- `docs/layers/13-existence.md` — Layer 13 derivation
