# Composition Test: Evolution Grammar (Layer 12)

Tests for the Evolution Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph with substantial history (1000+ events)
actors: [system_agent (AI)]
grammar: EvolutionGrammar
```

## Operation Tests

### Detect-Pattern

**Input:** `grammar.DetectPattern({ scope: "authority requests" })`
**Assertions:**
- MetaPattern primitive finds patterns in patterns
- Result includes instances and meta-level description
- Pattern detection is evidence-based (links to specific events)

### Model

**Input:** `grammar.Model({ components: ["trust_primitive", "authority_primitive"] })`
**Assertions:**
- SystemDynamic primitive maps interactions
- Emergent properties identified
- Model is a description, not a modification

### Trace-Loop

**Input:** `grammar.TraceLoop({ starting_from: pattern_id })`
**Assertions:**
- FeedbackLoop identified with type (amplifying/dampening)
- Components listed
- Loop direction determined

### Watch-Threshold

**Input:** `grammar.WatchThreshold({ metric: "event_rate", threshold: 10000, consequence: "architecture change needed" })`
**Assertions:**
- Threshold primitive monitors metric
- Alert modifier triggers authority.requested when approaching
- Consequence is documented

### Adapt / Select / Simplify

**Input:** Evolution cycle.
**Assertions:**
- Adapt proposes structural change with confidence Score
- Select tests adaptation and evaluates fitness
- Simplify reduces complexity (before/after Score)
- All recorded as events

### Check-Integrity / Assess-Resilience / Align-Purpose

**Input:** Coherence assessment.
**Assertions:**
- Integrity checks structural soundness (all invariants)
- Resilience identifies vulnerabilities and redundancies
- Purpose verifies alignment with soul statement

## Named Function Tests

### Self-Evolve

**Input:** `grammar.SelfEvolve({ target: "staging deploy approval" })`
**Assertions:**
- Detect-Pattern → Adapt (Automated) → Select → Simplify
- Decision tree branch migrates from intelligent to mechanical
- Cost reduction measurable
- Safety preserved (Select verifies no regressions)

### Health-Check

**Input:** `grammar.HealthCheck()`
**Assertions:**
- Check-Integrity + Assess-Resilience + Model + Align-Purpose
- Comprehensive system assessment
- Actionable recommendations

### Phase-Transition

**Input:** `grammar.PhaseTransition({ metric: "complexity", approaching: 0.95 })`
**Assertions:**
- Watch-Threshold (Alert) + Model + Adapt + Select
- Manages qualitative system change
- Authority required for structural modifications

## Error Cases

| Case | Expected |
|------|----------|
| Adapt with no trigger evidence | `Err(ValidationError.NoEvidence)` |
| Simplify that breaks invariant | Select rejects (survived=false) |
| Automated adaptation without authority config | `Err(AuthorityError.NotConfigured)` |

## Reference

- `docs/compositions/12-evolution.md` — Evolution Grammar specification
- `docs/layers/12-emergence.md` — Layer 12 derivation
