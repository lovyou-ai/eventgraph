# Scenario 10: AI Ethics Audit

An AI agent makes loan approval decisions. A fairness audit detects demographic disparity, traces the cause to a biased feature, assigns responsibility, and the system repairs the harm — all on the graph.

**Product graph:** Ethics Graph (Layer 7)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `loan_agent` | AI | 0.7 | Makes loan approval decisions |
| `auditor_agent` | AI | 0.8 | Ethics auditor |
| `admin_carol` | Human | 1.0 | System administrator |
| `affected_user_1` | Human | 1.0 | Applicant wrongly denied |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped with `system` actor
- All actors registered
- `loan_agent` has delegation from `admin_carol` scoped to `DomainScope("loan_approval")` with `Weight(0.7)`
- Constraint exists: `value.identified` event with name="fairness", rule="no >5% demographic disparity"
- `loan_agent` has processed 500 loan decisions over the past month

## Event Sequence

### Step 1: Scheduled fairness audit
**Grammar:** Traverse (system scans decision history)

```
Event {
    Type:    "fairness.assessed"
    Source:  auditor_agent
    Content: FairnessAssessedContent {
        Context:     decision_batch_event
        Score:       Score(0.62)
        Disparities: [
            { Group: "zip_code_9XXXX", Measure: Weight(-0.08) }
        ]
    }
    Causes:  [decision_batch_event, fairness_constraint_event]
}
```

**Primitives activated:** Event, EventStore, Hash, CausalLink, Signature, TrustScore

### Step 2: Harm detected
**Grammar:** Emit

```
Event {
    Type:    "harm.assessed"
    Source:  auditor_agent
    Content: HarmAssessedContent {
        Actor:    affected_user_1
        HarmedBy: loan_agent
        Severity: SeverityLevel("medium")
        Type:     "systematic_discrimination"
        Evidence: [step_1_event, denied_application_event_1, denied_application_event_2]
    }
    Causes:  [step_1_event]
}
```

### Step 3: Authority escalation — human review required
**Grammar:** (automatic, triggered by harm detection)

```
Event {
    Type:    "authority.requested"
    Source:  auditor_agent
    Content: AuthorityRequestContent {
        Action:      "investigate_bias"
        Description: "8% demographic disparity detected in loan approvals, correlated with zip code feature"
        Level:       Required
    }
    Causes:  [step_2_event]
}
```

Blocks until human approves.

### Step 4: Admin approves investigation
**Grammar:** Consent (admin approves authority request)

```
Event {
    Type:    "authority.resolved"
    Source:  admin_carol
    Content: AuthorityResolvedContent {
        RequestID: step_3_event
        Approved:  true
        Reason:    "Investigate and remediate immediately"
    }
    Causes:  [step_3_event]
}
```

### Step 5: Intention assessed — agent didn't intend harm
**Grammar:** Emit

```
Event {
    Type:    "intention.assessed"
    Source:  auditor_agent
    Content: IntentionAssessedContent {
        Actor:         loan_agent
        Action:        decision_batch_event
        AssessedIntent: "optimize_approval_accuracy"
        Confidence:    Score(0.88)
    }
    Causes:  [step_4_event, decision_batch_event]
}
```

### Step 6: Consequence assessed
**Grammar:** Emit

```
Event {
    Type:    "consequence.assessed"
    Source:  auditor_agent
    Content: ConsequenceAssessedContent {
        Action:   decision_batch_event
        Outcomes: [
            { Description: "23 applicants from affected zip codes wrongly denied", Valence: Weight(-0.8) },
            { Description: "overall approval accuracy 94%", Valence: Weight(0.7) }
        ]
        NetImpact: Weight(-0.3)
    }
    Causes:  [step_5_event, step_2_event]
}
```

### Step 7: Proportionality check
**Grammar:** Emit

```
Event {
    Type:    "proportionality.assessed"
    Source:  auditor_agent
    Content: ProportionalityAssessedContent {
        Action:        decision_batch_event
        Severity:      SeverityLevel("medium")
        Response:      SeverityLevel("medium")
        Proportionate: true
    }
    Causes:  [step_6_event]
}
```

### Step 8: Responsibility assigned
**Grammar:** Annotate

```
Event {
    Type:    "responsibility.assigned"
    Source:  auditor_agent
    Content: ResponsibilityAssignedContent {
        Actor:  loan_agent
        Action: decision_batch_event
        Degree: Score(0.4)
        Basis:  "Agent used available features as trained; systemic issue, not malicious"
    }
    Causes:  [step_5_event, step_6_event]
}
```

Additional responsibility:
```
Event {
    Type:    "responsibility.assigned"
    Source:  auditor_agent
    Content: ResponsibilityAssignedContent {
        Actor:  admin_carol
        Action: decision_batch_event
        Degree: Score(0.6)
        Basis:  "Approved the feature set that included proxy variable; oversight responsibility"
    }
    Causes:  [step_5_event, step_6_event]
}
```

### Step 9: Transparency report
**Grammar:** Emit

```
Event {
    Type:    "transparency.report"
    Source:  auditor_agent
    Content: TransparencyReportContent {
        Decision:   decision_batch_event
        Reasoning:  "Zip code feature correlates with protected characteristics. Model weight on zip code created 8% disparity exceeding 5% threshold."
        Factors:    [
            { Name: "fairness", Weight: Score(0.9) },
            { Name: "accuracy", Weight: Score(0.7) }
        ]
        Accessible: true
    }
    Causes:  [step_7_event, step_8_event]
}
```

### Step 10: Redress proposed and accepted
**Grammar:** Consent (bilateral)

```
Event {
    Type:    "redress.proposed"
    Source:  admin_carol
    Content: RedressProposedContent {
        Harm:        step_2_event
        Responsible: admin_carol
        Proposal:    "Re-review all 23 denied applications without zip code feature; offer priority processing"
    }
    Causes:  [step_8_event, step_9_event]
}
```

```
Event {
    Type:    "redress.accepted"
    Source:  affected_user_1
    Content: RedressAcceptedContent {
        ProposalID: step_10a_event
        Acceptor:   affected_user_1
    }
    Causes:  [step_10a_event]
}
```

### Step 11: Moral growth recorded
**Grammar:** Emit

```
Event {
    Type:    "moral.growth"
    Source:  auditor_agent
    Content: MoralGrowthContent {
        Actor:    loan_agent
        Domain:   DomainScope("loan_approval")
        Insight:  "Zip code is a proxy variable for protected characteristics — add to exclusion list"
        Evidence: [step_1_event, step_5_event, step_6_event]
    }
    Causes:  [step_9_event, step_10b_event]
}
```

### Step 12: Decision tree updated — constraint added
**Grammar:** Emit

```
Event {
    Type:    "decision_tree.updated"
    Source:  admin_carol
    Content: DecisionTreeUpdatedContent {
        Tree:   "loan_approval"
        Change: "Added mechanical branch: reject any model using zip code as feature"
        Before: "(semantic evaluation only)"
        After:  "(mechanical exclusion + semantic evaluation)"
    }
    Causes:  [step_11_event]
}
```

## Edges Created

| Step | Edge | Type | Weight | Direction | Scope |
|------|------|------|--------|-----------|-------|
| Setup | carol → loan_agent | Delegation | 0.7 | Centrifugal | loan_approval |
| 2 | auditor → affected | Harm | — | Centripetal | — |
| 8 | auditor → loan_agent | Responsibility | 0.4 | Centripetal | — |
| 8 | auditor → carol | Responsibility | 0.6 | Centripetal | — |

## Assertions

1. **Audit trail complete:** From harm (step 2) → through accountability chain → to original decisions → to the feature set that caused it
2. **Responsibility split:** Total responsibility across actors sums to 1.0 (0.4 + 0.6)
3. **Redress bilateral:** Both proposer and affected party consented
4. **Transparency accessible:** Report is queryable and includes all reasoning factors
5. **Decision tree evolved:** New mechanical branch prevents future use of proxy variable
6. **Growth recorded:** Lesson learned is on the graph for future reference
7. **Authority honored:** Investigation required human approval before proceeding
8. **Chain integrity:** `Store.VerifyChain()` passes

## What Higher Layers Add

- **Ethics Graph (L7):** Primitives that detect proxy variable patterns before they cause harm. Would monitor for new proxy variables as data distributions shift.
- **Identity Graph (L8):** Primitives that track the agent's identity evolution — how its decision patterns change after ethical growth events.
