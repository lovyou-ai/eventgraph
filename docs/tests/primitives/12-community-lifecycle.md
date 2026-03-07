# Scenario 12: Community Lifecycle

An open source project community onboards a newcomer, develops traditions, stewards shared resources, celebrates a milestone, and manages a generational succession — all on the graph.

**Product graph:** Community Graph (Layer 10)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `founder_alice` | Human | 1.0 | Project founder, original steward |
| `newcomer_bob` | Human | 0.1 (initial) | New contributor |
| `maintainer_carol` | Human | 0.9 | Senior maintainer |
| `ci_bot` | AI | 0.7 | CI/CD automation agent |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped with `system` actor
- Community subgraph "eventgraph-contributors" exists with 3 members (alice, carol, ci_bot)
- Community norms: "all PRs require review", "weekly retrospective on Fridays"
- Shared resource: "test infrastructure" stewarded by carol

## Event Sequence

### Step 1: Newcomer invited — sponsor endorsement
**Grammar:** Endorse + Subscribe (Invite named function)

```
Event {
    Type:    "edge.created"
    Source:  maintainer_carol
    Content: EdgeCreatedContent {
        EdgeType:  Endorsement
        Target:    newcomer_bob
        Weight:    Weight(0.3)
        Scope:     DomainScope("eventgraph-contributors")
    }
    Causes:  [bob_introduction_event]
}
```

```
Event {
    Type:    "edge.created"
    Source:  newcomer_bob
    Content: EdgeCreatedContent {
        EdgeType:  Subscription
        Target:    community_subgraph_event
    }
    Causes:  [step_1a_event]
}
```

**Primitives activated:** Event, EventStore, Hash, CausalLink, Signature, TrustScore, Edge

### Step 2: Newcomer settles — belonging starts low
**Grammar:** Emit

```
Event {
    Type:    "home.identified"
    Source:  system
    Content: HomeIdentifiedContent {
        Actor:     newcomer_bob
        Community: community_subgraph_event
        Belonging: Score(0.15)
    }
    Causes:  [step_1b_event]
}
```

### Step 3: Newcomer makes first contribution
**Grammar:** Emit

```
Event {
    Type:    "contribution.recorded"
    Source:  newcomer_bob
    Content: ContributionRecordedContent {
        Actor:     newcomer_bob
        Community: community_subgraph_event
        Type:      "documentation_fix"
        Value:     Score(0.4)
    }
    Causes:  [pr_merged_event]
}
```

### Step 4: Community acknowledges — belonging increases
**Grammar:** Acknowledge

```
Event {
    Type:    "home.identified"
    Source:  system
    Content: HomeIdentifiedContent {
        Actor:     newcomer_bob
        Community: community_subgraph_event
        Belonging: Score(0.35)
    }
    Causes:  [step_3_event, acknowledge_events]
}
```

### Step 5: Newcomer participates in Friday retrospective tradition
**Grammar:** Emit

```
Event {
    Type:    "tradition.identified"
    Source:  system
    Content: TraditionIdentifiedContent {
        Community: community_subgraph_event
        Practice:  "friday_retrospective"
        Age:       Duration(7776000000000000)  // ~90 days in nanoseconds
        Adherence: Score(0.85)
    }
    Causes:  [retro_event, step_3_event]
}
```

Bob participates, adherence stays high.

### Step 6: Over months, Bob becomes established — trust accumulates
**Grammar:** (automatic, multiple trust.updated events)

After 6 months and 30 contributions:

```
Event {
    Type:    "trust.updated"
    Source:  system
    Content: TrustUpdatedContent {
        Actor:    newcomer_bob
        Previous: Score(0.1)
        Current:  Score(0.65)
        Domain:   "eventgraph-contributors"
    }
    Causes:  [latest_contribution_event]
}
```

```
Event {
    Type:    "home.identified"
    Source:  system
    Content: HomeIdentifiedContent {
        Actor:     newcomer_bob
        Community: community_subgraph_event
        Belonging: Score(0.78)
    }
    Causes:  [step_6a_event]
}
```

### Step 7: Sustainability assessment — bus factor risk
**Grammar:** Emit

```
Event {
    Type:    "sustainability.assessed"
    Source:  ci_bot
    Content: SustainabilityAssessedContent {
        Community: community_subgraph_event
        Score:     Score(0.72)
        Risks:     ["bus factor: carol is sole steward of test infrastructure", "no documented succession plan"]
        Horizon:   Duration(15552000000000000)  // ~180 days
    }
    Causes:  [quarterly_assessment_event]
}
```

### Step 8: Succession planned — carol passes stewardship to bob
**Grammar:** Consent (bilateral)

```
Event {
    Type:    "succession.planned"
    Source:  maintainer_carol
    Content: SuccessionPlannedContent {
        From:  maintainer_carol
        To:    newcomer_bob
        Scope: DomainScope("test_infrastructure")
    }
    Causes:  [step_7_event]
}
```

```
Event {
    Type:    "succession.completed"
    Source:  system
    Content: SuccessionCompletedContent {
        PlanID: step_8a_event
    }
    Causes:  [step_8a_event, bob_consent_event]
}
```

### Step 9: Community celebrates v2.0 milestone
**Grammar:** Emit

```
Event {
    Type:    "milestone.reached"
    Source:  founder_alice
    Content: MilestoneReachedContent {
        Community:    community_subgraph_event
        Description:  "EventGraph v2.0 released"
        Significance: Score(0.95)
        Contributors: [founder_alice, maintainer_carol, newcomer_bob, ci_bot]
    }
    Causes:  [v2_release_event]
}
```

```
Event {
    Type:    "ceremony.held"
    Source:  founder_alice
    Content: CeremonyHeldContent {
        Community:    community_subgraph_event
        Type:         "release_celebration"
        Participants: [founder_alice, maintainer_carol, newcomer_bob, ci_bot]
        Significance: Score(0.90)
    }
    Causes:  [step_9a_event]
}
```

### Step 10: Community story chapter added
**Grammar:** Emit

```
Event {
    Type:    "community.story"
    Source:  founder_alice
    Content: CommunityStoryContent {
        Community: community_subgraph_event
        Chapter:   "The v2.0 journey: how Bob went from newcomer to steward in 6 months"
        Events:    [step_1a_event, step_3_event, step_8b_event, step_9a_event]
        Teller:    founder_alice
    }
    Causes:  [step_9b_event]
}
```

### Step 11: Gift — Alice shares tool with no strings attached
**Grammar:** Emit

```
Event {
    Type:    "gift.given"
    Source:  founder_alice
    Content: GiftGivenContent {
        From:          founder_alice
        To:            newcomer_bob
        Description:   "Custom test harness I built — it's yours to maintain now"
        Unconditional: true
    }
    Causes:  [step_8b_event]
}
```

No obligation created. Unlike Exchange (Layer 2), this is purely generative.

## Edges Created

| Step | Edge | Type | Weight | Direction | Scope |
|------|------|------|--------|-----------|-------|
| 1 | carol → bob | Endorsement | 0.3 | Centripetal | community |
| 1 | bob → community | Subscription | — | Centripetal | — |
| 6 | system → bob | Trust | 0.65 | Centripetal | community |
| 8 | carol → bob | Delegation | — | Centrifugal | test_infrastructure |

## Assertions

1. **Belonging gradient:** Bob's belonging Score increases over time: 0.15 → 0.35 → 0.78
2. **Trust accumulation:** Bob's trust rises from 0.1 to 0.65 through consistent contributions
3. **Succession bilateral:** Both carol and bob consented to the stewardship transfer
4. **Sustainability improved:** After succession, bus factor risk is reduced (queryable)
5. **Tradition maintained:** Friday retrospective adherence Score stays high through the scenario
6. **Gift creates no obligation:** No `obligation.created` event follows the gift
7. **Story references real events:** All events cited in step 10 exist on the graph
8. **Milestone includes all contributors:** Everyone who contributed is listed
9. **Chain integrity:** `Store.VerifyChain()` passes

## What Higher Layers Add

- **Community Graph (L10):** Primitives that predict belonging trajectory — would flag newcomers likely to leave early and suggest interventions.
- **Culture Graph (L11):** Primitives that detect cultural drift — would notice if the Friday retrospective tradition is evolving in meaning over time.
