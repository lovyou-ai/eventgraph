# Scenario 8: Family Decision Log

A family makes a significant financial decision together. The process — proposal, consultation, information gathering, decision — is recorded. Years later, when a similar decision arises, the family can query "how did we decide this last time?" and see the complete reasoning.

**Product graph:** Social Graph (Layer 3)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `parent_maria` | Human | 0.85 | Parent, proposer |
| `parent_james` | Human | 0.85 | Parent, co-decision maker |
| `teen_sophie` | Human | 0.5 | Teenager, consulted |
| `advisor_agent` | AI | 0.4 | Financial research assistant |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped, family members registered
- `parent_maria` and `parent_james` have mutual Delegation for `DomainScope("family_finance")` with `Weight(0.9)` — equal authority
- `teen_sophie` has Subscribe to family decisions (can see and comment, not decide)
- `advisor_agent` has Delegation from `parent_james` scoped to `DomainScope("financial_research")` with `Weight(0.3)` — can research but not decide

## Event Sequence

### Step 1: Maria proposes buying a house
**Grammar:** Emit

```
Event {
    Type:    "family.proposal"
    Source:  parent_maria
    Content: FamilyProposalContent {
        Title:       "Buy a house in the Eastside neighbourhood"
        Category:    "family_finance"
        Description: "3-bedroom house, $450K, 20% down, 30-year mortgage"
        Impact:      "Major financial commitment. Affects everyone."
    }
    Causes:  [bootstrap_event]
}
```

### Step 2: James asks the AI to research
**Grammar:** Delegate (task to agent)

```
Event {
    Type:    "work.delegated"
    Source:  parent_james
    Content: WorkDelegatedContent {
        Task:      "Research housing market in Eastside. Compare with renting."
        DelegatedTo: advisor_agent
        Scope:     "financial_research"
        Deadline:  "2024-12-01"
    }
    Causes:  [step_1_event]
}
```

### Step 3: Agent researches and reports
**Grammar:** Derive

```
Event {
    Type:    "research.report"
    Source:  advisor_agent
    Content: ResearchReportContent {
        Task:       step_2_event
        Findings:   "Eastside median price $440K (+8% YoY). Rent equivalent: $2,200/mo. Mortgage at current rates: $2,400/mo. Break-even: 5 years. School rating: 8/10."
        Confidence: Score(0.75)
        Sources:    ["zillow_data_hash", "school_rating_hash", "mortgage_calc"]
    }
    Causes:  [step_2_event]
}
```

**Authority:** Agent acts under Delegation from James in `financial_research` scope. Notification level — no approval needed for research.

### Step 4: Sophie shares her perspective
**Grammar:** Respond

```
Event {
    Type:    "family.discussion"
    Source:  teen_sophie
    Content: FamilyDiscussionContent {
        Proposal: step_1_event
        Position: "support_conditional"
        Text:     "I'd like to stay in Eastside — my friends and school are here. But I'd want my own room."
    }
    Causes:  [step_1_event]
}
```

### Step 5: James responds with concerns
**Grammar:** Respond

```
Event {
    Type:    "family.discussion"
    Source:  parent_james
    Content: FamilyDiscussionContent {
        Proposal: step_1_event
        Position: "concern"
        Text:     "The mortgage is $200/mo more than rent. We'd need to cut the vacation budget. Also, agent's report shows break-even at 5 years — are we sure we'll stay that long?"
    }
    Causes:  [step_1_event, step_3_event]   // James's concern is informed by the AI report
}
```

### Step 6: Maria addresses concerns
**Grammar:** Respond

```
Event {
    Type:    "family.discussion"
    Source:  parent_maria
    Content: FamilyDiscussionContent {
        Proposal: step_1_event
        Position: "rebuttal"
        Text:     "We've been here 3 years already, likely another 10 for Sophie's school. That's well past break-even. And equity builds wealth vs rent."
    }
    Causes:  [step_5_event]
}
```

### Step 7: Maria and James make the decision
**Grammar:** Consent (bilateral, between the two decision-makers)

```
Event {
    Type:    "family.decision"
    Source:  system
    Content: FamilyDecisionContent {
        Proposal:     step_1_event
        Decision:     "approved"
        DecisionMakers: [parent_maria, parent_james]
        Consulted:    [teen_sophie]
        InformedBy:   [step_3_event, step_4_event, step_5_event, step_6_event]
        Conditions:   "Proceed with offer up to $460K. Sophie gets her own room."
        Justification: "Break-even is 5 years, we plan to stay 10+. Sophie's school and social connections are here."
    }
    Causes:  [step_4_event, step_5_event, step_6_event]
}
```

Both parents sign this event. The Consent is bilateral between the two adults. Sophie was consulted (her opinion is in the causes) but didn't need to approve.

### Step 8: Decision recorded with authority chain

```
Decision {
    Outcome:        Permit
    Confidence:     Score(0.78)
    AuthorityChain: [
        { Actor: parent_maria, Level: Required, Weight: Score(0.9) },
        { Actor: parent_james, Level: Required, Weight: Score(0.9) }
    ]
    TrustWeights: [
        { Actor: advisor_agent, Score: Score(0.4), Domain: "financial_research" }
    ]
    Evidence:       [step_1_event, step_3_event, step_4_event, step_5_event, step_6_event]
    NeedsHuman:     false    // both humans already decided
}
```

---

### Years Later: Similar Decision Arises

### Step 9: James proposes renovating the house (3 years later)
**Grammar:** Emit

```
Event {
    Type:    "family.proposal"
    Source:  parent_james
    Content: FamilyProposalContent {
        Title:       "Kitchen renovation, $35K budget"
        Category:    "family_finance"
        Description: "Full kitchen remodel. Contractor estimates $30-40K."
        Impact:      "Significant but not life-changing. 3-month disruption."
    }
    Causes:  [bootstrap_event]
}
```

### Step 10: Family queries past decisions
**Grammar:** Traverse

```
// Query: How have we decided major financial things before?
Store.ByType("family.decision")
→ Page { Items: [step_7_event] }

// Query: What was the process for the house purchase?
Store.Ancestors(step_7_event, maxDepth: 5)
→ [step_6 (Maria rebuttal), step_5 (James concern), step_4 (Sophie input),
   step_3 (AI research), step_2 (delegated research), step_1 (proposal)]

// The family sees:
// 1. We consulted Sophie
// 2. We got AI research first
// 3. We discussed concerns openly
// 4. We required bilateral consent
// 5. We documented conditions and justification
```

This becomes the template for the renovation decision. Same process, different scale.

## Trust and Authority Flow

```
Authority structure:
  maria + james: Required (both must approve family_finance decisions)
  sophie: Consulted (opinion recorded, not required)
  advisor_agent: Notification (can research autonomously)

Trust:
  advisor_agent in financial_research: 0.4 → 0.43 (research was useful)
  sophie: unchanged (consulted, no authority action)
```

## Assertions

1. **Consultation recorded:** Sophie's opinion (step 4) is in the causes of the decision (step 7) — she was heard
2. **Decision bilateral:** Both parents signed the decision event — neither can unilaterally decide
3. **AI contribution scoped:** Agent could only research, not decide. Its delegation was scoped to `financial_research`.
4. **Reasoning preserved:** The justification in step 7 references specific evidence — not just "we decided yes"
5. **Causal chain complete:** Decision → discussions → research → proposal — full reasoning chain traversable
6. **Conditions documented:** "Sophie gets her own room" is in the decision — accountable commitment
7. **Precedent queryable:** Years later, the family can query past decisions and see the full process
8. **Agent accountability:** The AI's research (step 3) includes sources and confidence — if the market data was wrong, that's traceable
9. **Authority appropriate:** Financial decisions require Required level from both parents — the system enforces this
10. **Chain integrity:** All events hash-chained across the years

## What Higher Layers Add

- **Social Graph (L3):** Primitives that understand group dynamics — is one family member consistently overruled? Are teenagers' opinions actually influencing decisions or being recorded but ignored?
- **Governance Graph (L11):** Primitives that detect concentration of decision power. In a family, that might flag "James has vetoed 4 of Maria's last 5 proposals" as a pattern worth surfacing.
- **Relationship Graph (L9):** Primitives that understand family relationship health from decision patterns — collaborative decisions strengthen relationships; unilateral ones strain them.
