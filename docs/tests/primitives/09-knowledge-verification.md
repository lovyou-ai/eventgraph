# Scenario 9: Knowledge Verification

A system makes claims about performance metrics, one is challenged with counter-evidence, bias is detected in the original data, and the knowledge base corrects itself with full provenance.

**Product graph:** Knowledge Graph (Layer 6)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `analyst_agent` | AI | 0.6 | Makes performance claims from data |
| `reviewer_bob` | Human | 1.0 | Domain expert, reviews claims |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped with `system` actor
- `analyst_agent` and `reviewer_bob` registered
- `analyst_agent` has delegation from `reviewer_bob` scoped to `DomainScope("performance_analysis")` with `Weight(0.6)`

## Event Sequence

### Step 1: Agent makes a performance claim
**Grammar:** Emit

```
Event {
    Type:    "fact.established"
    Source:  analyst_agent
    Content: FactEstablishedContent {
        Claim:      "Service X handles 10,000 RPS with p99 < 50ms"
        Confidence: Score(0.85)
        Evidence:   [benchmark_event_1, benchmark_event_2]
        Domain:     DomainScope("performance")
    }
    Causes:  [benchmark_event_1, benchmark_event_2]
}
```

**Primitives activated:** Event, EventStore, Hash, CausalLink, Signature

### Step 2: Agent categorizes the claim
**Grammar:** Annotate

```
Event {
    Type:    "classification.assigned"
    Source:  analyst_agent
    Content: ClassificationAssignedContent {
        Event:      step_1_event
        Category:   "performance/throughput"
        Confidence: Score(0.95)
    }
    Causes:  [step_1_event]
}
```

### Step 3: Agent infers a generalization
**Grammar:** Derive

```
Event {
    Type:    "inference.drawn"
    Source:  analyst_agent
    Content: InferenceDrawnContent {
        Premises:   [step_1_event, prior_perf_fact_1, prior_perf_fact_2]
        Conclusion: "All services on framework Y handle 10k+ RPS"
        Confidence: Score(0.70)
        Method:     "inductive"
    }
    Causes:  [step_1_event, prior_perf_fact_1, prior_perf_fact_2]
}
```

### Step 4: Reviewer challenges the claim with counter-evidence
**Grammar:** Challenge

```
Event {
    Type:    "narrative.challenged"
    Source:  reviewer_bob
    Content: NarrativeChallengedContent {
        NarrativeID: step_1_event
        Challenger:  reviewer_bob
        Counter:     "Replication under production-like load shows only 6,200 RPS with p99 = 120ms"
        Evidence:    [replication_benchmark_event]
    }
    Causes:  [step_1_event, replication_benchmark_event]
}
```

### Step 5: Bias detected in original benchmark
**Grammar:** Annotate

```
Event {
    Type:    "bias.detected"
    Source:  reviewer_bob
    Content: BiasDetectedContent {
        Type:     "sampling_bias"
        Evidence: [benchmark_event_1, replication_benchmark_event]
        Severity: SeverityLevel("medium")
        Affected: DomainScope("performance")
        Description: "Original benchmark used synthetic traffic with no database contention"
    }
    Causes:  [step_4_event, benchmark_event_1]
}
```

### Step 6: Knowledge corrected
**Grammar:** Derive (from original claim + counter-evidence)

```
Event {
    Type:    "correction.applied"
    Source:  analyst_agent
    Content: CorrectionAppliedContent {
        Target:     step_1_event
        Correction: "Service X handles 6,000-7,000 RPS under production load with p99 < 150ms"
        Evidence:   replication_benchmark_event
    }
    Causes:  [step_1_event, step_4_event, step_5_event]
}
```

### Step 7: Correction propagated to dependent inference
**Grammar:** Derive

```
Event {
    Type:    "correction.propagated"
    Source:  system
    Content: CorrectionPropagatedContent {
        CorrectionID: step_6_event
        Affected:     [step_3_event]
        Description:  "Inference about framework Y throughput invalidated — premise corrected"
    }
    Causes:  [step_6_event, step_3_event]
}
```

### Step 8: Agent learns from the experience
**Grammar:** Emit

```
Event {
    Type:    "learning.occurred"
    Source:  analyst_agent
    Content: LearningOccurredContent {
        Domain:  DomainScope("performance_analysis")
        Before:  "Trusted synthetic benchmarks at face value"
        After:   "Always validate with production-representative load"
        Trigger: step_5_event
    }
    Causes:  [step_5_event, step_6_event]
}
```

### Step 9: Provenance trace
**Grammar:** Traverse

```
Event {
    Type:    "provenance.traced"
    Source:  system
    Content: ProvenanceTracedContent {
        Claim:          "Service X handles 6,000-7,000 RPS under production load"
        Chain:          [analyst_agent, reviewer_bob]
        OriginalSource: analyst_agent
        Confidence:     Score(0.80)
    }
    Causes:  [step_6_event]
}
```

### Step 10: Trust updated
**Grammar:** (automatic, triggered by correction)

```
Event {
    Type:    "trust.updated"
    Source:  system
    Content: TrustUpdatedContent {
        Actor:    analyst_agent
        Previous: Score(0.6)
        Current:  Score(0.52)
        Domain:   "performance_analysis"
        Cause:    step_6_event
    }
    Causes:  [step_6_event]
}
```

Trust decreased because the agent's original claim needed correction — but not as sharply as a violation, because the agent participated in the correction process.

## Edges Created

| Step | Edge | Type | Weight | Direction | Scope |
|------|------|------|--------|-----------|-------|
| Setup | bob → agent | Delegation | 0.6 | Centrifugal | performance_analysis |
| 2 | step_1 → performance/throughput | Classification | 0.95 | Centripetal | — |
| 4 | step_4 → step_1 | Challenge | — | Centripetal | — |
| 10 | system → agent | Trust | 0.52 | Centripetal | performance_analysis |

## Assertions

1. **Original claim preserved:** `Store.Get(step_1_event)` still exists — corrections don't delete, they supersede
2. **Correction chain:** `Store.Ancestors(step_6_event)` traces through challenge → original claim → original evidence
3. **Propagation:** All inferences depending on step_1 are flagged via step_7
4. **Provenance traversable:** From corrected claim (step_6) back to original source and evidence
5. **Bias recorded:** `Store.Query(type="bias.detected")` returns step_5 with the original evidence
6. **Learning recorded:** Agent's learning event captures the before/after behavioral change
7. **Trust decreased but not crashed:** Agent's trust reflects the error severity and correction participation
8. **Chain integrity:** `Store.VerifyChain()` passes — all hashes valid, all causes exist

## What Higher Layers Add

- **Knowledge Graph (L6):** Primitives that detect contradictions across knowledge domains automatically. Would flag the inconsistency before a human reviewer needed to find it.
- **Ethics Graph (L7):** Primitives that assess whether biased knowledge led to harmful decisions. Would trace from the correction back to any decisions made based on the wrong claim.
