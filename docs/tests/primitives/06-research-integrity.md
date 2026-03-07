# Scenario 6: Research Integrity

A researcher pre-registers a hypothesis, conducts experiments, and publishes results. The hash chain proves the hypothesis existed before the data. Failed analysis attempts are visible — not just the successful one. Peer review happens on the graph.

**Product graph:** Research Graph (Layer 5)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `researcher_grace` | Human | 0.6 | Principal investigator |
| `analysis_agent` | AI | 0.4 | Statistical analysis AI |
| `reviewer_henry` | Human | 0.8 | Peer reviewer |
| `reviewer_iris` | Human | 0.75 | Peer reviewer |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped
- All actors registered
- `analysis_agent` has Delegation from `researcher_grace` in scope `statistical_analysis` with `Weight(0.5)`

## Event Sequence

### Step 1: Grace pre-registers her hypothesis
**Grammar:** Emit

```
Event {
    Type:    "research.hypothesis"
    Source:  researcher_grace
    Content: ResearchHypothesisContent {
        Title:       "Caffeine intake correlates with code review accuracy"
        Hypothesis:  "Developers who consume 200-400mg caffeine daily will have 15% higher code review accuracy than those who consume none"
        Domain:      "software_engineering"
        Variables:   ["caffeine_mg_daily", "review_accuracy_pct"]
        Methodology: "Randomized controlled trial, 60 participants, 4 weeks"
    }
    Causes:  [bootstrap_event]
}
```

**Key property:** This event is hash-chained with a timestamp. The hash chain proves this hypothesis was registered BEFORE any data was collected. No p-hacking.

### Step 2: Grace documents her methodology
**Grammar:** Extend (sequential, same author)

```
Event {
    Type:    "research.methodology"
    Source:  researcher_grace
    Content: ResearchMethodologyContent {
        Hypothesis:     step_1_event
        SampleSize:     60
        Duration:       "4 weeks"
        Randomization:  "Block randomization, 3 groups: 0mg, 200mg, 400mg"
        Measurement:    "Automated code review accuracy scoring via test suite"
        StatisticalTest: "One-way ANOVA with post-hoc Tukey HSD"
        AlphaLevel:     0.05
        PowerAnalysis:  "80% power to detect 15% difference at n=20/group"
    }
    Causes:  [step_1_event]
}
```

### Step 3: Data collection (events recorded as they happen)
**Grammar:** Emit (multiple data collection events)

```
Event {
    Type:    "research.data_collected"
    Source:  researcher_grace
    Content: ResearchDataCollectedContent {
        Methodology: step_2_event
        Week:        1
        Participants: 58       // 2 dropped out
        DataHash:    Hash("...")  // hash of the raw data file — not the data itself
    }
    Causes:  [step_2_event]
}
// ... similar events for weeks 2, 3, 4
```

### Step 4: First analysis attempt — fails
**Grammar:** Derive (analysis derived from data)

```
Event {
    Type:    "research.analysis"
    Source:  analysis_agent
    Content: ResearchAnalysisContent {
        Data:         [week_1_event, week_2_event, week_3_event, week_4_event]
        Method:       "One-way ANOVA"
        Result:       "F(2,55) = 1.23, p = 0.301"
        Significant:  false
        Notes:        "No significant effect found with original grouping"
    }
    Causes:  [week_4_event]
}
```

**This event stays on the graph.** The researcher cannot delete it. The failed analysis is part of the record.

### Step 5: Grace adjusts analysis approach
**Grammar:** Derive (new analysis, causally linked to first attempt)

```
Event {
    Type:    "research.analysis"
    Source:  analysis_agent
    Content: ResearchAnalysisContent {
        Data:         [week_1_event, week_2_event, week_3_event, week_4_event]
        Method:       "One-way ANOVA with outlier removal (pre-registered criteria)"
        Result:       "F(2,52) = 4.87, p = 0.011"
        Significant:  true
        Notes:        "Significant after removing 3 outliers per pre-registered criteria"
        PreviousAttempt: step_4_event   // explicitly linked to prior analysis
    }
    Causes:  [step_4_event]    // caused by the failed attempt — not independent
}
```

The causal link from step 4 to step 5 makes it clear this was a second attempt. A reviewer can see BOTH analyses.

### Step 6: Grace writes up results
**Grammar:** Derive

```
Event {
    Type:    "research.manuscript"
    Source:  researcher_grace
    Content: ResearchManuscriptContent {
        Title:       "Caffeine and Code Review Accuracy: A Randomized Trial"
        Hypothesis:  step_1_event
        Methodology: step_2_event
        Analysis:    step_5_event
        Conclusion:  "Moderate caffeine intake (200-400mg) associated with 12% improvement in code review accuracy (p=0.011)"
        DataAvailability: "Raw data hash: ..."
    }
    Causes:  [step_5_event]
}
```

### Step 7: Henry reviews — requests the full analysis chain
**Grammar:** Traverse + Respond

```
// Henry traverses the causal chain:
Store.Ancestors(step_6_event, maxDepth: 10)
→ [step_5 (analysis), step_4 (failed analysis), week_1-4 (data), step_2 (methodology), step_1 (hypothesis)]

// Henry sees both analyses and responds:
Event {
    Type:    "research.review"
    Source:  reviewer_henry
    Content: ResearchReviewContent {
        Manuscript:  step_6_event
        Decision:    "revise"
        Comments:    "The effect size (12%) is below the pre-registered target (15%). The outlier removal is justified but should be discussed more thoroughly. Both analyses must be reported in the paper."
        Score:       Score(0.6)
    }
    Causes:  [step_6_event]
}
```

### Step 8: Iris reviews — endorses
**Grammar:** Respond + Endorse

```
Event {
    Type:    "research.review"
    Source:  reviewer_iris
    Content: ResearchReviewContent {
        Manuscript:  step_6_event
        Decision:    "accept_with_minor"
        Comments:    "Sound methodology, appropriate pre-registration. Agree both analyses should be reported."
        Score:       Score(0.75)
    }
    Causes:  [step_6_event]
}

Event {
    Type:    "edge.created"
    Source:  reviewer_iris
    Content: EdgeCreatedContent {
        From:      reviewer_iris
        To:        researcher_grace
        EdgeType:  Endorsement
        Weight:    Weight(0.4)
        Direction: Centripetal
        Scope:     Some("software_engineering")
    }
    Causes:  [step_8_review_event]
}
```

### Step 9: Grace revises and resubmits
**Grammar:** Derive (revision caused by reviews)

```
Event {
    Type:    "research.revision"
    Source:  researcher_grace
    Content: ResearchRevisionContent {
        Original:    step_6_event
        Reviews:     [step_7_event, step_8_event]
        Changes:     "Added failed analysis to results section. Expanded outlier discussion. Noted effect size below pre-registered target."
    }
    Causes:  [step_7_event, step_8_event]   // revision caused by BOTH reviews
}
```

### Step 10: Publication — the complete chain is the publication
**Grammar:** Emit

```
Event {
    Type:    "research.published"
    Source:  system
    Content: ResearchPublishedContent {
        Manuscript:   step_9_event
        Hypothesis:   step_1_event
        AllAnalyses:  [step_4_event, step_5_event]
        Reviews:      [step_7_event, step_8_event]
        DOI:          "10.1234/example.2024.001"
    }
    Causes:  [step_9_event]
}
```

### Step 11: Trust updated

```
Event {
    Type:    "trust.updated"
    Source:  system
    Content: TrustUpdatedContent {
        Actor:    researcher_grace
        Previous: Score(0.6)
        Current:  Score(0.67)
        Domain:   "software_engineering"
        Cause:    step_10_event
    }
    Causes:  [step_10_event]
}
```

## Assertions

1. **Pre-registration provable:** step_1 (hypothesis) is hash-chained before step_3 (data collection) — timestamp order is cryptographically verifiable
2. **Failed analyses visible:** step_4 (failed ANOVA) is on the graph and causally linked to step_5 (successful analysis) — no hiding failed attempts
3. **All analyses linked:** Reviewer can traverse from manuscript to ALL prior analyses, not just the successful one
4. **Methodology before data:** step_2 (methodology) is hash-chained before step_3 (data) — analysis plan was declared first
5. **Review chain complete:** step_9 (revision) has both reviews in its causes — all feedback addressed
6. **Outlier criteria pre-registered:** The outlier removal in step_5 references pre-registered criteria in step_2 — not post-hoc
7. **Replication ready:** A future researcher can Derive from step_10 and link their replication study to the original
8. **Trust reflects peer validation:** Grace's trust increases after peer-reviewed publication
9. **Chain integrity:** All events hash-chained, all causes valid, full chain verifiable

## What Higher Layers Add

- **Research Graph (L5):** Primitives that detect statistical issues (underpowered studies, suspicious p-values), track replication patterns across studies, and flag methodology gaps before peer review.
- **Knowledge Graph (L6):** Links this finding to related research — "3 other studies found similar caffeine effects" or "this contradicts Smith et al. 2023."
- **Ethics Graph (L7):** Detects patterns of data manipulation across a researcher's publication history.
