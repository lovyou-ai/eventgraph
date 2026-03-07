# Composition Test: Knowledge Grammar (Layer 6)

Tests for the Knowledge Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph
actors: [researcher_alice (Human), analyst_agent (AI), peer_bob (Human)]
grammar: KnowledgeGrammar
```

## Operation Tests

### Claim

**Input:** `grammar.Claim({ assertion: "treatment X reduces latency 30%", evidence: [exp_42], confidence: 0.85, actor: alice })`
**Assertions:**
- Fact primitive activated
- Emits `fact.established` event with evidence links and confidence
- Evidence events are causal predecessors
- Confidence is a valid Score

### Categorize

**Input:** `grammar.Categorize({ claim: claim_id, category: "performance/optimization" })`
**Assertions:**
- Classification primitive activated
- Taxonomy updated if category is new
- Claim queryable by category

### Abstract / Encode

**Input:** `grammar.Abstract({ instances: [claim_1, claim_2, claim_3], generalization: "caching improves all IO-bound operations" })`
**Assertions:**
- Abstraction primitive activated
- Generalization event links to all instances
- Generality Score computed
- `grammar.Encode({ source: abstraction_id, format: "JSON" })` transforms representation

### Infer

**Input:** `grammar.Infer({ premises: [fact_1, fact_2], conclusion: "X works because of caching", method: "deductive" })`
**Assertions:**
- Inference primitive activated
- Conclusion event has causes linking to all premises
- Method is recorded
- Confidence derived from premise confidences

### Remember / Recall

**Input:** `grammar.Remember({ key: "treatment-X", content: claim_id, importance: 0.8 })`
**Assertions:**
- Memory primitive stores the mapping
- `grammar.Recall({ key: "treatment-X" })` retrieves it
- Recall returns relevance Score
- Memory has lastAccessed timestamp

### Challenge

**Input:** `grammar.Challenge({ claim: claim_id, counter_evidence: [exp_58], explanation: "only 12% improvement" })`
**Assertions:**
- Narrative primitive constructs the counter-narrative
- Challenge event as Response to original claim
- Both claim and challenge coexist on graph (not deletion)
- Original author notified

### Detect-Bias / Correct / Trace

**Input:** Bias detection → correction → provenance trace.
**Assertions:**
- Detect-Bias identifies systematic distortion with evidence
- Correct creates a new claim linked to the error and evidence
- Correction propagates to dependent inferences
- Trace follows chain to original source

### Learn

**Input:** `grammar.Learn({ domain: "performance", insight: "always test with representative workloads", trigger: correction_id })`
**Assertions:**
- Learning primitive activated
- Behavioral change recorded (before/after)
- Trigger event is causal predecessor

## Named Function Tests

### Verify

**Input:** `grammar.Verify({ claim: claim_id })`
**Assertions:**
- Traces provenance to original source
- Looks for corroborating claims
- Returns verification status with confidence

### Fact-Check

**Input:** `grammar.FactCheck({ claim: claim_id })`
**Assertions:**
- Trace + Detect-Bias + Challenge or Verify
- Full report with provenance chain, bias check, and verdict

### Retract

**Input:** `grammar.Retract({ claim: claim_id, reason: "methodology was flawed" })`
**Assertions:**
- Self-challenge + correction
- Dependent inferences flagged for re-evaluation
- Original claim NOT deleted (provenance preserved)
- Retraction event links to reason

## Error Cases

| Case | Expected |
|------|----------|
| Claim without evidence | `Err(ValidationError.NoEvidence)` |
| Infer from retracted premise | Warning event (inference flagged, not blocked) |
| Recall non-existent key | Returns None (not error) |
| Circular inference (A→B→A) | Detected, `Err(ValidationError.CircularInference)` |

## Reference

- `docs/compositions/06-knowledge.md` — Knowledge Grammar specification
- `docs/layers/06-information.md` — Layer 6 derivation
- `docs/tests/primitives/06-research-integrity.md` — Integration test scenario
