# Knowledge Grammar (Layer 6: Information)

The grammar for verified, provenanced knowledge with bias detection.

## Derivation

Knowledge is operations on claims and their evidence. The base operations are: **represent**, **verify**, **challenge**, **correct**. Four semantic dimensions differentiate operations:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| Epistemic status | Claimed / Evidenced / Verified / Retracted | How confident are we? |
| Scope | Particular (one claim) / General (pattern or principle) | Specific or abstract? |
| Direction | Constructive (adding knowledge) / Deconstructive (challenging it) | Building up or tearing down? |
| Source | Primary (direct observation) / Derived (inferred from other claims) | First-hand or reasoned? |

## Operations (12)

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 1 | **Claim** | knowledge/assert | Make a knowledge claim with evidence | Fact + Emit |
| 2 | **Categorize** | knowledge/organize | Assign a claim to a taxonomy | Classification + Annotate |
| 3 | **Abstract** | knowledge/generalize | Generalize from specific instances | Abstraction + Derive (from instances) |
| 4 | **Encode** | knowledge/transform | Transform between representations | Encoding + Derive |
| 5 | **Infer** | knowledge/derive | Draw a new conclusion from premises | Inference + Derive (from facts) |
| 6 | **Remember** | knowledge/store | Store knowledge for long-term retrieval | Memory + Emit |
| 7 | **Recall** | knowledge/retrieve | Retrieve stored knowledge by relevance | Memory (recalled) + Traverse |
| 8 | **Challenge** | truth/question | Present counter-evidence to a claim | Narrative + Challenge |
| 9 | **Detect-Bias** | truth/critique | Identify systematic distortion | Bias + Annotate |
| 10 | **Correct** | truth/fix | Fix an error and propagate correction | Correction + Derive (from evidence) |
| 11 | **Trace** | truth/provenance | Track a claim to its original source | Provenance + Traverse |
| 12 | **Learn** | knowledge/update | Update behaviour based on new knowledge | Learning + Emit |

## Modifiers (2)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| **Confidence** | Attaches explicit confidence score to operation | Claim, Infer, Challenge |
| **Perishable** | Knowledge auto-flags for re-verification after TTL | Claim, Remember |

## Named Functions (6)

| Function | Composition | Purpose |
|----------|------------|---------|
| **Verify** | Claim + Trace + Claim (corroborating) | Establish a claim as fact |
| **Retract** | Challenge (self) + Correct + propagate to dependents | Withdraw a claim with chain repair |
| **Survey** | Recall (batch) + Abstract + Claim (synthesis) | Survey existing knowledge on a topic |
| **Fact-Check** | Trace + Detect-Bias + Challenge or Verify | Full provenance and bias check |
| **Knowledge-Base** | Claim + Categorize + Remember (batch) | Build structured knowledge store |
| **Transfer** | Recall + Encode (new format) + Learn (new context) | Move knowledge across domains |

## Mapping to Primitives

| Operation | Layer 6 Primitives | Grammar Operations |
|-----------|-------------------|-------------------|
| Claim | Fact | Emit |
| Categorize | Classification | Annotate |
| Abstract | Abstraction | Derive |
| Encode | Encoding | Derive |
| Infer | Inference | Derive |
| Remember | Memory | Emit |
| Recall | Memory | Traverse |
| Challenge | Narrative | Challenge |
| Detect-Bias | Bias | Annotate |
| Correct | Correction | Derive |
| Trace | Provenance | Traverse |
| Learn | Learning | Emit |

## Example Flow

**Research knowledge lifecycle:**
```
Claim("treatment X reduces latency by 30%",
      evidence=[experiment-42, data-set-7], confidence=0.85)
  → Categorize(taxonomy="performance/optimization")
  → Remember(importance=0.8, key="treatment-X-latency")
  → Infer("treatment X likely works because of caching effect",
          premises=[claim-1, known-cache-behavior])
  → Challenge("replication found only 12% improvement",
              evidence=[experiment-58], confidence=0.90)
  → Detect-Bias("original experiment used atypical workload")
  → Correct(original-claim, "15-20% under typical workload",
            evidence=[experiment-42, experiment-58])
  → Learn("always test with representative workloads")
```

## Reference

- `docs/grammar.md` — Infrastructure grammar (15 operations)
- `docs/layers/06-information.md` — Layer 6 derivation
- `docs/primitives.md` — Layer 6 primitive specifications
- `docs/tests/primitives/06-research-integrity.md` — Integration test scenario
