# Scenario 2: Freelancer Reputation

A freelancer completes work for a client. The work, approval, and endorsement are recorded as events. The freelancer's reputation is portable — a new client queries their verifiable history before hiring.

**Product graph:** Market Graph (Layer 2)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `freelancer_bob` | Human | 0.5 (initial) | Freelancer |
| `client_carol` | Human | 0.8 | First client |
| `client_dave` | Human | 0.7 | Second client (queries reputation) |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped
- All actors registered
- `client_carol` Subscribes to `freelancer_bob` (edge: Subscription)

## Event Sequence

### Step 1: Carol posts a job
**Grammar:** Emit

```
Event {
    Type:    "market.listing"
    Source:  client_carol
    Content: MarketListingContent {
        Description: "Build REST API for inventory management"
        Budget:      5000
        Domain:      "software_development"
    }
    Causes:  [bootstrap_event]
}
```

### Step 2: Bob expresses interest
**Grammar:** Respond

```
Event {
    Type:    "market.proposal"
    Source:  freelancer_bob
    Content: MarketProposalContent {
        Listing:  step_1_event
        Estimate: 4500
        Timeline: "2 weeks"
    }
    Causes:  [step_1_event]
}
```

### Step 3: Carol opens a private channel with Bob
**Grammar:** Channel

```
Event {
    Type:    "edge.created"
    Source:  client_carol
    Content: EdgeCreatedContent {
        From:      client_carol
        To:        freelancer_bob
        EdgeType:  Channel
        Weight:    Weight(0.5)
        Direction: Centripetal
    }
    Causes:  [step_2_event]
}
```

### Step 4: They negotiate and agree on terms
**Grammar:** Consent (mutual, atomic, dual-signed)

```
Event {
    Type:    "market.agreement"
    Source:  system                  // system records the bilateral consent
    Content: MarketAgreementContent {
        Listing:    step_1_event
        Proposal:   step_2_event
        PartyA:     client_carol
        PartyB:     freelancer_bob
        Amount:     4500
        Deadline:   "2024-12-01"
        Domain:     "software_development"
    }
    Causes:  [step_2_event, step_3_event]
    // Both signatures present — this is the Consent grammar operation
}
```

**Authority:** Consent requires bilateral — both actors sign the canonical form. Neither party can forge this event.

### Step 5: Bob delivers the work
**Grammar:** Emit

```
Event {
    Type:    "work.delivered"
    Source:  freelancer_bob
    Content: WorkDeliveredContent {
        Agreement: step_4_event
        Artifact:  "https://github.com/carol-co/inventory-api"
    }
    Causes:  [step_4_event]
}
```

### Step 6: Carol acknowledges receipt
**Grammar:** Acknowledge (content-free edge toward the delivery)

```
Event {
    Type:    "edge.created"
    Source:  client_carol
    Content: EdgeCreatedContent {
        From:      client_carol
        To:        freelancer_bob
        EdgeType:  Endorsement
        Weight:    Weight(0.0)       // Acknowledge = endorsement with zero weight
        Direction: Centripetal
    }
    Causes:  [step_5_event]
}
```

### Step 7: Carol endorses Bob's work
**Grammar:** Endorse (reputation-staked edge)

```
Event {
    Type:    "edge.created"
    Source:  client_carol
    Content: EdgeCreatedContent {
        From:      client_carol
        To:        freelancer_bob
        EdgeType:  Endorsement
        Weight:    Weight(0.8)       // strong endorsement
        Direction: Centripetal
        Scope:     Some("software_development")
    }
    Causes:  [step_5_event, step_6_event]
}
```

Carol stakes her own reputation on this endorsement. If Bob later does bad work, Carol's endorsement history is part of the record.

### Step 8: Trust updated from successful completion
```
Event {
    Type:    "trust.updated"
    Source:  system
    Content: TrustUpdatedContent {
        Actor:    freelancer_bob
        Previous: Score(0.5)
        Current:  Score(0.62)
        Domain:   "software_development"
        Cause:    step_7_event
    }
    Causes:  [step_7_event]
}
```

### Step 9: Dave queries Bob's reputation before hiring
**Grammar:** Traverse (read-only)

Dave is considering hiring Bob. He queries the graph:

```
// Query 1: Bob's trust in software development
ITrustModel.ScoreInDomain(freelancer_bob, "software_development")
→ TrustMetrics {
    Actor:      freelancer_bob
    Overall:    Score(0.62)
    ByDomain:   { "software_development": Score(0.62) }
    Confidence: Score(0.4)      // only one engagement — low confidence
    Evidence:   [step_7_event, step_8_event]
}

// Query 2: Endorsements of Bob
Store.EdgesTo(freelancer_bob, Endorsement)
→ [
    Edge { From: client_carol, Weight: 0.8, Scope: "software_development" }
]

// Query 3: Bob's work history
Store.BySource(freelancer_bob)
→ Page { Items: [step_2_event, step_5_event] }

// Query 4: Carol's trustworthiness as an endorser
ITrustModel.Score(client_carol)
→ TrustMetrics { Overall: Score(0.8) }
```

Dave can verify: Bob completed work, Carol endorsed it, Carol herself is trustworthy. This is portable — Bob didn't have to ask Carol to write a reference letter.

### Step 10: Dave hires Bob
**Grammar:** Consent

```
Event {
    Type:    "market.agreement"
    Source:  system
    Content: MarketAgreementContent {
        PartyA:  client_dave
        PartyB:  freelancer_bob
        Amount:  6000
        Domain:  "software_development"
    }
    Causes:  [step_9_traverse_results]   // Dave's decision was informed by Bob's history
}
```

## Edges Created

| Step | Edge | Type | Weight | Scope |
|------|------|------|--------|-------|
| Setup | carol → bob | Subscription | 0.5 | — |
| 3 | carol ↔ bob | Channel | 0.5 | — |
| 6 | carol → bob | Endorsement | 0.0 | — |
| 7 | carol → bob | Endorsement | 0.8 | software_development |

## Trust Flow

```
Bob's trust in software_development:
  0.50 (initial) → 0.62 (after Carol's endorsement)

Carol's trust (unchanged at 0.8, but now has endorsement history):
  If Bob later fails, Carol's endorsement record is evidence
```

## Assertions

1. **Portable reputation:** Dave can query Bob's work history and endorsements without Bob's involvement
2. **Endorsement staking:** Carol's endorsement is an edge with weight — her reputation is linked to Bob's performance
3. **Bilateral consent:** The market.agreement event requires both signatures — neither party can forge it
4. **Causal chain:** step_4 (agreement) causes step_5 (delivery) causes step_7 (endorsement) causes step_8 (trust update)
5. **Domain-specific trust:** Bob's trust in `software_development` is 0.62, but trust in other domains remains at initial
6. **Confidence reflects evidence:** With only one engagement, confidence is low (0.4). More engagements increase confidence.
7. **Chain integrity:** All events hash-chained, all causes valid

## What Higher Layers Add

- **Market Graph (L2):** Primitives that understand fair pricing, detect fraud patterns, manage escrow lifecycle. Would flag "this price is 3x market rate" or "this client has disputed 80% of deliveries."
- **Identity Graph (L8):** Bob's freelancer identity emerges from his work history. Selective disclosure: Bob proves "I have 5 completed projects rated above 0.8" without revealing client names.
