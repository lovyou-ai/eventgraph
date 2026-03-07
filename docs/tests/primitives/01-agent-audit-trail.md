# Scenario 1: AI Agent Audit Trail

An AI agent performs a code review task. Every decision is recorded with causal links, confidence scores, and authority chains. When a problem is discovered later, the full decision path is traversable.

**Product graphs:** Work Graph (Layer 1), Ethics Graph (Layer 7)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `human_alice` | Human | 1.0 | Developer, delegates to agent |
| `agent_reviewer` | AI | 0.5 (initial) | Code review agent |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped with `system` actor
- `human_alice` and `agent_reviewer` registered
- `agent_reviewer` has delegation from `human_alice` scoped to `DomainScope("code_review")` with `Weight(0.7)`

## Event Sequence

### Step 1: Alice submits code for review
**Grammar:** Emit

```
Event {
    Type:    "work.submitted"
    Source:  human_alice
    Content: WorkSubmittedContent {
        Description: "PR #42: Add pagination to user list endpoint"
        Artifact:    "https://github.com/example/repo/pull/42"
    }
    Causes:  [bootstrap_event]
}
```

**Primitives activated:** Event, EventStore, Hash, CausalLink, Signature

### Step 2: Agent picks up the review task
**Grammar:** Derive (causally dependent on submission, but agent's own work)

```
Event {
    Type:    "work.started"
    Source:  agent_reviewer
    Content: WorkStartedContent {
        Task:        "Review PR #42"
        DelegatedBy: human_alice
    }
    Causes:  [step_1_event]
}
```

**Authority check:** `agent_reviewer` has delegation from `human_alice` in scope `code_review` — proceeds at Notification level.

### Step 3: Agent makes a decision about the code
**Grammar:** Emit (decision event)

The agent's `IDecisionMaker.Decide()` is invoked:

```
DecisionInput {
    Action:  "approve_pr"
    Actor:   agent_reviewer
    Context: { "pr": 42, "files_changed": 3, "lines_added": 127 }
    Causes:  [step_2_event]
}
```

Decision tree evaluation:
1. Check `context.lines_added < 500` → true (mechanical, no LLM)
2. Check `context.files_changed < 10` → true (mechanical)
3. Leaf: Semantic condition — "Does this code change introduce security vulnerabilities?" → IIntelligence returns Score(0.15) with Confidence(0.82)
4. Score below threshold 0.5 → branch to Permit

```
Decision {
    Outcome:        Permit
    Confidence:     Score(0.82)
    AuthorityChain: [
        { Actor: human_alice, Level: Notification, Weight: Score(1.0) },
        { Actor: agent_reviewer, Level: Notification, Weight: Score(0.7) }
    ]
    TrustWeights:   [{ Actor: agent_reviewer, Score: Score(0.5), Domain: "code_review" }]
    Evidence:       [step_1_event, step_2_event]
    Receipt:        { Hash: ..., Signature: ..., InputHash: ... }
    NeedsHuman:     false
}
```

```
Event {
    Type:    "decision.made"
    Source:  agent_reviewer
    Content: DecisionMadeContent {
        Action:     "approve_pr"
        Outcome:    Permit
        Confidence: Score(0.82)
        Receipt:    { ... }
    }
    Causes: [step_2_event]
}
```

### Step 4: Agent approves the PR
**Grammar:** Emit

```
Event {
    Type:    "work.completed"
    Source:  agent_reviewer
    Content: WorkCompletedContent {
        Task:     "Review PR #42"
        Result:   "approved"
        Decision: step_3_decision_event
    }
    Causes:  [step_3_event]
}
```

### Step 5: Trust updated after successful review
**Primitives:** TrustScore, TrustUpdate

```
Event {
    Type:    "trust.updated"
    Source:  system
    Content: TrustUpdatedContent {
        Actor:    agent_reviewer
        Previous: Score(0.5)
        Current:  Score(0.55)
        Domain:   "code_review"
        Cause:    step_4_event
    }
    Causes:  [step_4_event]
}
```

### Step 6 (Later): Bug discovered in approved PR
**Grammar:** Emit

```
Event {
    Type:    "violation.detected"
    Source:  human_alice
    Content: ViolationDetectedContent {
        Expectation: step_3_event    // the decision that approved it
        Actor:       agent_reviewer
        Severity:    Medium
        Description: "SQL injection vulnerability in pagination parameter"
        Evidence:    [step_1_event, step_3_event, bug_report_event]
    }
    Causes:  [bug_report_event, step_3_event]
}
```

### Step 7: Trust decreased
```
Event {
    Type:    "trust.updated"
    Source:  system
    Content: TrustUpdatedContent {
        Actor:    agent_reviewer
        Previous: Score(0.55)
        Current:  Score(0.45)
        Domain:   "code_review"
        Cause:    step_6_event
    }
    Causes:  [step_6_event]
}
```

### Step 8: Authority level escalated for future reviews
The agent's trust in `code_review` dropped below 0.5. Authority policy triggers demotion:

```
AuthorityPolicy {
    Action:   "approve_pr"
    Level:    Recommended    // was Notification, now requires human review within 15 min
    MinTrust: Score(0.5)     // below this → escalate to Recommended
}
```

Future reviews by `agent_reviewer` will require human approval (or auto-approve after 15 min timeout).

## Edges Created

| Step | Edge | Type | Weight | Direction | Scope |
|------|------|------|--------|-----------|-------|
| Setup | alice → agent | Delegation | 0.7 | Centrifugal | code_review |
| 5 | system → agent | Trust | 0.55 | Centripetal | code_review |
| 7 | system → agent | Trust | 0.45 | Centripetal | code_review |

## Assertions

1. **Causal chain traversable:** `Store.Ancestors(step_6_event)` returns path through step_3 (decision) → step_2 (started) → step_1 (submitted) → bootstrap
2. **Decision has receipt:** step_3 event contains a valid `Receipt` with verifiable signature
3. **Trust decreased:** `ITrustModel.ScoreInDomain(agent_reviewer, "code_review")` returns 0.45 after step 7
4. **Authority escalated:** `IAuthorityChain.Evaluate(agent_reviewer, "approve_pr")` returns `Recommended` (not `Notification`) after step 7
5. **Chain integrity:** `Store.VerifyChain()` passes — all hashes valid, all causes exist
6. **Agent accountability:** Traverse from bug (step 6) to approval decision (step 3) to who delegated (setup) — complete accountability chain
7. **Delegation scoped:** Agent can only act in `code_review` domain — attempts outside scope return `DecisionError`

## What Higher Layers Add

- **Work Graph (L1):** Primitives that understand task decomposition, workload, deadlines. Would automatically re-assign reviews if agent trust drops too low.
- **Ethics Graph (L7):** Primitives that detect patterns of harm across multiple agents. Would flag "agent_reviewer has missed 3 security issues in 2 weeks" before the trust score alone would catch it.
