# Scenario 4: Community Governance

A cooperative makes a collective decision about resource allocation. The process — proposal, debate, amendment, vote, outcome — is entirely on the graph. Any member can traverse from "this policy affects me" to "who proposed it" to "what the vote was" to "what evidence was presented."

**Product graph:** Governance Graph (Layer 11)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `member_alice` | Human | 0.8 | Proposer |
| `member_bob` | Human | 0.7 | Debater, amender |
| `member_carol` | Human | 0.6 | Voter |
| `member_dave` | Human | 0.9 | Community elder (high trust) |
| `governance_bot` | AI | 0.5 | Tallies votes, enforces rules |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped
- All members registered and Subscribed to each other (community subgraph)
- `member_dave` has Delegation from other members for `DomainScope("community_policy")` with `Weight(0.8)` — elder authority
- `governance_bot` has Delegation from `member_dave` scoped to `DomainScope("vote_tally")` with `Weight(0.6)`
- Community norm exists: proposals require 3-day discussion period before vote

## Event Sequence

### Step 1: Alice proposes a budget allocation
**Grammar:** Emit

```
Event {
    Type:    "governance.proposal"
    Source:  member_alice
    Content: GovernanceProposalContent {
        Title:       "Allocate $2000 to community garden"
        Description: "Monthly budget for seeds, tools, and water"
        Amount:      2000
        Category:    "community_resources"
        VoteDeadline: "2024-12-15"
    }
    Causes:  [bootstrap_event]
}
```

### Step 2: Bob responds with a concern
**Grammar:** Respond

```
Event {
    Type:    "governance.discussion"
    Source:  member_bob
    Content: GovernanceDiscussionContent {
        Proposal: step_1_event
        Position: "concern"
        Text:     "We should cap it at $1500 and review after 3 months"
    }
    Causes:  [step_1_event]
}
```

### Step 3: Carol responds supporting Alice
**Grammar:** Respond

```
Event {
    Type:    "governance.discussion"
    Source:  member_carol
    Content: GovernanceDiscussionContent {
        Proposal: step_1_event
        Position: "support"
        Text:     "The garden feeds 12 families. $2000 is reasonable."
    }
    Causes:  [step_1_event]
}
```

### Step 4: Bob proposes an amendment
**Grammar:** Annotate (attaches modification to the proposal)

```
Event {
    Type:    "governance.amendment"
    Source:  member_bob
    Content: GovernanceAmendmentContent {
        Proposal:   step_1_event
        Change:     "Reduce to $1500 with 3-month review clause"
        Rationale:  "Compromise: fund the garden but with accountability"
    }
    Causes:  [step_1_event, step_2_event]
}
```

### Step 5: Dave endorses the amendment (elder weight)
**Grammar:** Endorse

```
Event {
    Type:    "edge.created"
    Source:  member_dave
    Content: EdgeCreatedContent {
        From:      member_dave
        To:        member_bob          // endorsing Bob's amendment
        EdgeType:  Endorsement
        Weight:    Weight(0.7)
        Direction: Centripetal
        Scope:     Some("community_policy")
    }
    Causes:  [step_4_event]
}
```

Dave's endorsement carries weight because of his high trust (0.9) and elder delegation.

### Step 6: Discussion period expires — vote opens
**Grammar:** Emit (system event)

```
Event {
    Type:    "governance.vote_open"
    Source:  governance_bot
    Content: GovernanceVoteOpenContent {
        Proposal:    step_1_event
        Amendments:  [step_4_event]
        VoteOptions: ["original", "amended", "reject"]
        Deadline:    "2024-12-15"
    }
    Causes:  [step_1_event, step_4_event]
}
```

**Authority:** `governance_bot` acts under delegation from `member_dave` in scope `vote_tally`. Notification level — logged, no approval needed.

### Step 7-9: Members vote
**Grammar:** Consent (each vote is a bilateral commitment)

```
// Alice votes for original
Event {
    Type:    "governance.vote"
    Source:  member_alice
    Content: GovernanceVoteContent {
        Proposal: step_1_event
        Choice:   "original"
    }
    Causes:  [step_6_event]
}

// Bob votes for amended
Event {
    Type:    "governance.vote"
    Source:  member_bob
    Content: GovernanceVoteContent {
        Proposal: step_1_event
        Choice:   "amended"
    }
    Causes:  [step_6_event]
}

// Carol votes for amended
Event {
    Type:    "governance.vote"
    Source:  member_carol
    Content: GovernanceVoteContent {
        Proposal: step_1_event
        Choice:   "amended"
    }
    Causes:  [step_6_event]
}
```

### Step 10: Dave votes — elder endorsement visible
```
Event {
    Type:    "governance.vote"
    Source:  member_dave
    Content: GovernanceVoteContent {
        Proposal: step_1_event
        Choice:   "amended"
    }
    Causes:  [step_6_event]
}
```

### Step 11: Bot tallies and records outcome
**Grammar:** Derive (outcome derived from votes)

```
Event {
    Type:    "governance.outcome"
    Source:  governance_bot
    Content: GovernanceOutcomeContent {
        Proposal:       step_1_event
        Result:         "amended"
        VoteCount:      { "original": 1, "amended": 3, "reject": 0 }
        Amendment:      step_4_event
        EffectiveDate:  "2024-12-16"
        ReviewDate:     "2025-03-16"
    }
    Causes:  [step_7_event, step_8_event, step_9_event, step_10_event]
}
```

All four votes are in the causes — the outcome is causally linked to every vote.

### Step 12: Budget allocation enacted
**Grammar:** Derive (spending derived from outcome)

```
Event {
    Type:    "governance.enacted"
    Source:  system
    Content: GovernanceEnactedContent {
        Outcome:    step_11_event
        Policy:     "Allocate $1500/month to community garden, review in 3 months"
        Amount:     1500
    }
    Causes:  [step_11_event]
}
```

### Step 13: Trust updated — governance participation rewarded

```
Event {
    Type:    "trust.updated"
    Source:  system
    Content: TrustUpdatedContent {
        Actor:    member_bob
        Previous: Score(0.7)
        Current:  Score(0.73)
        Domain:   "community_policy"
        Cause:    step_11_event
    }
    Causes:  [step_11_event]
}
```

Bob's amendment was adopted — trust increase in governance domain.

## Edges Created

| Step | Edge | Type | Weight | Scope |
|------|------|------|--------|-------|
| Setup | dave → alice,bob,carol | Delegation | 0.8 | community_policy |
| Setup | dave → governance_bot | Delegation | 0.6 | vote_tally |
| 5 | dave → bob | Endorsement | 0.7 | community_policy |

## Assertions

1. **Full transparency:** Any member can Traverse from step_12 (enacted policy) through step_11 (outcome) to step_7-10 (every vote) to step_1 (original proposal)
2. **Amendment chain:** step_4 (amendment) causally linked to step_2 (concern) — the amendment addresses the concern
3. **Vote integrity:** Every vote is signed by the voter and hash-chained — no vote manipulation
4. **Elder influence visible:** Dave's endorsement (step 5) and vote (step 10) are on the graph — influence is transparent, not hidden
5. **Bot authority scoped:** `governance_bot` can only tally votes and open voting — cannot vote, propose, or amend
6. **Discussion period enforced:** Vote cannot open before discussion period expires (3 days after proposal)
7. **Outcome causally complete:** step_11 has ALL four votes in its causes — no votes excluded
8. **Trust reflects participation:** Members who participate constructively (especially adopted amendments) gain trust
9. **Chain integrity:** Full hash chain verification passes

## What Higher Layers Add

- **Governance Graph (L11):** Primitives that detect power concentration (one person winning every vote), corruption patterns (voting always aligns with endorsement payments), and policy impact tracking (did the garden allocation actually help?).
- **Community Graph (L10):** Primitives that understand community health — is participation declining? Are certain members being systematically excluded from decisions?
- **Justice Graph (L4):** If a member disputes the outcome, the dispute resolution system has the complete evidence chain already on the graph.
