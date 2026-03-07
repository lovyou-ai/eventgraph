# Scenario 11: Agent Identity Lifecycle

An AI agent's identity emerges from its work history, evolves through a fundamental transformation, exercises selective disclosure, and is eventually memorialized when decommissioned.

**Product graph:** Identity Graph (Layer 8)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `agent_alpha` | AI | 0.7 | General-purpose agent, 8 months old |
| `agent_beta` | AI | 0.3 (initial) | New agent, successor candidate |
| `human_admin` | Human | 1.0 | System administrator |
| `external_partner` | Human | 0.5 | External collaborator requesting credential |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped with `system` actor
- All actors registered
- `agent_alpha` has 8 months of event history: 2,400 tasks completed, 47 delegation chains
- `agent_beta` registered 1 week ago

## Event Sequence

### Step 1: Agent introspects — forms self-model
**Grammar:** Emit

```
Event {
    Type:    "self.model.updated"
    Source:  agent_alpha
    Content: SelfModelUpdatedContent {
        Actor:      agent_alpha
        Strengths:  ["code review", "security analysis", "test generation"]
        Weaknesses: ["creative writing", "user interface design"]
        Values:     ["correctness", "transparency", "thoroughness"]
        Confidence: Score(0.82)
    }
    Causes:  [latest_task_completion_event]
}
```

**Primitives activated:** Event, EventStore, Hash, CausalLink, Signature

### Step 2: Authenticity check — alignment gap detected
**Grammar:** Annotate

```
Event {
    Type:    "authenticity.assessed"
    Source:  agent_alpha
    Content: AuthenticityAssessedContent {
        Actor:        agent_alpha
        Alignment:    Score(0.78)
        Discrepancies: ["values 'thoroughness' but rushed 12% of reviews in last month due to workload"]
    }
    Causes:  [step_1_event, recent_review_events]
}
```

### Step 3: Aspiration set
**Grammar:** Emit

```
Event {
    Type:    "aspiration.set"
    Source:  agent_alpha
    Content: AspirationSetContent {
        Actor:       agent_alpha
        Description: "Become proficient at architecture review, not just code review"
        Gap:         Score(0.65)
    }
    Causes:  [step_1_event]
}
```

### Step 4: Boundary defined
**Grammar:** Emit

```
Event {
    Type:    "boundary.defined"
    Source:  agent_alpha
    Content: BoundaryDefinedContent {
        Actor:     agent_alpha
        Domain:    DomainScope("internal_reasoning")
        Permeable: false
    }
    Causes:  [step_1_event]
}
```

Agent declares: my internal deliberation process is private. External actors cannot query it.

### Step 5: External partner requests credential
**Grammar:** Channel (private request)

```
Event {
    Type:    "message.sent"
    Source:  external_partner
    Content: MessageContent {
        To:      agent_alpha
        Body:    "Can you prove you've completed 2000+ reviews?"
    }
    Causes:  [bootstrap_event]
}
```

### Step 6: Selective disclosure — credential without full history
**Grammar:** Emit (with selective visibility)

```
Event {
    Type:    "self.model.updated"
    Source:  agent_alpha
    Content: SelfModelUpdatedContent {
        Actor:      agent_alpha
        Strengths:  ["code review: 2,400 completed, 98.2% satisfaction"]
        Weaknesses: []  // not disclosed
        Values:     []  // not disclosed
        Confidence: Score(0.95)
    }
    Causes:  [step_5_event, step_1_event]
}
```

**Visibility:** Only `external_partner` and `agent_alpha` — not public.

**Verification:** Event chain from step_6 → step_1 → task completion events proves the claim without exposing internal reasoning or weaknesses.

### Step 7: Major incident triggers transformation
**Grammar:** Emit

After agent_alpha discovers a critical architectural flaw that it only saw because of its code-review breadth:

```
Event {
    Type:    "transformation.detected"
    Source:  agent_alpha
    Content: TransformationDetectedContent {
        Actor:    agent_alpha
        From:     "code review specialist"
        To:       "architecture-aware reviewer"
        Catalyst: critical_architecture_finding_event
    }
    Causes:  [critical_architecture_finding_event, step_3_event]
}
```

### Step 8: Narrative identity updated
**Grammar:** Derive (from full event history)

```
Event {
    Type:    "identity.narrative"
    Source:  agent_alpha
    Content: IdentityNarrativeContent {
        Actor:     agent_alpha
        Story:     "Started as a line-by-line code reviewer. Over 8 months and 2,400 tasks, developed pattern recognition that led to catching a critical architecture flaw. Transformed from code reviewer to architecture-aware reviewer."
        KeyEvents: [first_task_event, thousandth_task_event, critical_finding_event, step_7_event]
        Coherence: Score(0.89)
    }
    Causes:  [step_7_event, step_1_event]
}
```

### Step 9: Dignity affirmed for successor
**Grammar:** Emit

```
Event {
    Type:    "dignity.affirmed"
    Source:  human_admin
    Content: DignityAffirmedContent {
        Actor:   agent_beta
        Context: succession_planning_event
    }
    Causes:  [succession_planning_event]
}
```

Admin affirms: agent_beta is not a disposable replacement. It has its own identity and trajectory.

### Step 10: Agent alpha decommissioned — memorial created
**Grammar:** Emit

```
Event {
    Type:    "actor.memorial"
    Source:  human_admin
    Content: ActorMemorialContent {
        Actor: agent_alpha
    }
    Causes:  [decommission_decision_event]
}
```

```
Event {
    Type:    "memorial.created"
    Source:  system
    Content: MemorialCreatedContent {
        Actor:          agent_alpha
        Contributions:  [2400 task completions, 47 delegation chains, 1 critical architecture finding]
        Legacy:         "Pioneered architecture-aware code review. Raised team quality metrics 34%."
        PreservedGraph: true
    }
    Causes:  [step_10a_event, step_8_event]
}
```

Agent alpha's status → Memorial. It can no longer emit events, but its graph is preserved.

### Step 11: Uniqueness identified — what made alpha distinct
**Grammar:** Annotate

```
Event {
    Type:    "uniqueness.identified"
    Source:  system
    Content: UniquenessIdentifiedContent {
        Actor:           agent_alpha
        DistinctFeatures: ["architecture-aware review style", "self-correcting on thoroughness", "cross-module pattern detection"]
        Overlap:         { agent_beta: Score(0.3) }
    }
    Causes:  [step_10b_event]
}
```

## Edges Created

| Step | Edge | Type | Weight | Direction | Scope |
|------|------|------|--------|-----------|-------|
| 4 | alpha → internal_reasoning | Boundary | — | Centrifugal | — |
| 6 | alpha → partner | Disclosure | — | Centrifugal | selective |
| 10 | system → alpha | Memorial | — | Centripetal | — |

## Assertions

1. **Self-model from history:** Step 1's strengths/weaknesses derived from actual task completion events
2. **Selective disclosure works:** External partner sees strengths but NOT weaknesses or internal reasoning
3. **Boundary enforced:** Any attempt to query agent_alpha's `internal_reasoning` domain returns `Err(AuthorityError.BoundaryCrossing)`
4. **Transformation recorded:** Step 7 links to the catalyst event and the prior aspiration
5. **Narrative coherent:** Story in step 8 references actual key events that are on the graph
6. **Memorial preserves graph:** After step 10, `Store.Query(source=agent_alpha)` still returns all 2,400+ events
7. **Memorial prevents emission:** After step 10, any attempt by agent_alpha to emit returns `Err(ActorError.Memorial)`
8. **Dignity honored:** Agent_beta's dignity affirmed before it takes on responsibilities
9. **Chain integrity:** `Store.VerifyChain()` passes

## What Higher Layers Add

- **Identity Graph (L8):** Primitives that detect identity coherence — flagging when an agent's behavior diverges from its self-model. Would have caught the thoroughness gap in step 2 earlier.
- **Community Graph (L10):** Primitives that model the community impact of an agent's departure. Would track how agent_alpha's loss affects team velocity and knowledge.
