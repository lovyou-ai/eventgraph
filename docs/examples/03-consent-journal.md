# Scenario 3: Consent-Based Shared Journal

Two people maintain a shared journal where every entry requires bilateral consent. The system enforces structural consent — not "I agreed to terms of service once" but "I agree to share this specific thing now." Demonstrates the Relationship Graph pattern: vulnerability, reciprocity, betrayal, repair, forgiveness.

**Product graph:** Relationship Graph (Layer 9)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `alice` | Human | 0.5 | Partner A |
| `bob` | Human | 0.5 | Partner B |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped
- Both actors registered
- No prior relationship

## Event Sequence

### Step 1: Alice invites Bob to share a journal
**Grammar:** Invite (Endorse + Subscribe)

```
Event {
    Type:    "edge.created"
    Source:  alice
    Content: EdgeCreatedContent {
        From:      alice
        To:        bob
        EdgeType:  Subscription
        Weight:    Weight(0.5)
        Direction: Centripetal
    }
    Causes:  [bootstrap_event]
}

Event {
    Type:    "edge.created"
    Source:  alice
    Content: EdgeCreatedContent {
        From:      alice
        To:        bob
        EdgeType:  Endorsement
        Weight:    Weight(0.3)
        Direction: Centripetal
    }
    Causes:  [subscription_event]
}
```

### Step 2: Bob accepts — mutual subscription
**Grammar:** Subscribe (reciprocal)

```
Event {
    Type:    "edge.created"
    Source:  bob
    Content: EdgeCreatedContent {
        From:      bob
        To:        alice
        EdgeType:  Subscription
        Weight:    Weight(0.5)
        Direction: Centripetal
    }
    Causes:  [step_1_subscription]
}
```

### Step 3: They open a private channel
**Grammar:** Channel (private, bidirectional)

```
Event {
    Type:    "edge.created"
    Source:  system
    Content: EdgeCreatedContent {
        From:      alice
        To:        bob
        EdgeType:  Channel
        Weight:    Weight(0.5)
        Direction: Centripetal      // bidirectional channel
    }
    Causes:  [step_1_subscription, step_2_subscription]
}
```

### Step 4: Alice writes a journal entry
**Grammar:** Emit

```
Event {
    Type:    "journal.entry"
    Source:  alice
    Content: JournalEntryContent {
        Text:       "Today I realized I've been avoiding difficult conversations..."
        Visibility: Private         // only visible within the channel
    }
    Causes:  [step_3_channel]
}
```

This event is NOT shared yet. It exists on Alice's subgraph, within the channel.

### Step 5: Alice requests consent to share with Bob
**Grammar:** Consent (bilateral, atomic)

```
Event {
    Type:    "authority.requested"
    Source:  alice
    Content: AuthorityRequestContent {
        Action:        "share_journal_entry"
        Actor:         alice
        Level:         Required        // must have Bob's explicit approval
        Justification: "Share journal entry with Bob"
        Causes:        [step_4_event]
    }
    Causes:  [step_4_event]
}
```

### Step 6: Bob consents
**Grammar:** Consent (Bob's half)

```
Event {
    Type:    "authority.resolved"
    Source:  bob
    Content: AuthorityResolvedContent {
        RequestID:    step_5_event
        Approved:     true
        Resolver:     bob
        Reason:       Some("I appreciate you sharing this")
    }
    Causes:  [step_5_event]
}
```

Now the journal entry is visible to Bob within the channel.

### Step 7: Bob responds with his own entry
**Grammar:** Respond (causally dependent, subordinate)

```
Event {
    Type:    "journal.entry"
    Source:  bob
    Content: JournalEntryContent {
        Text:       "I've noticed that too. I think we both do it..."
        Visibility: Private
    }
    Causes:  [step_4_event, step_6_event]   // responds to Alice's entry, after consent
}
```

Bob's response doesn't need separate consent — by responding within the channel, both parties see it. The channel IS the consent boundary.

### Step 8: Trust accumulates from reciprocal sharing

```
Event {
    Type:    "trust.updated"
    Source:  system
    Content: TrustUpdatedContent {
        Actor:    bob
        Previous: Score(0.5)
        Current:  Score(0.55)
        Domain:   "personal"
        Cause:    step_7_event
    }
    Causes:  [step_7_event]
}
```

Reciprocal: Alice's trust in Bob goes up because he responded. Bob's trust in Alice goes up because she shared first (vulnerability).

### Step 9: Time passes — trust decays without interaction

```
Event {
    Type:    "trust.decayed"
    Source:  system
    Content: TrustDecayedContent {
        Actor:    bob
        Previous: Score(0.55)
        Current:  Score(0.52)
        Elapsed:  Duration(2592000000000000)   // 30 days in nanoseconds
        Rate:     Score(0.01)
    }
    Causes:  [step_8_event]
}
```

### Step 10: Betrayal — Bob shares Alice's private entry externally
**Grammar:** Propagate (but violates channel privacy)

```
Event {
    Type:    "violation.detected"
    Source:  system
    Content: ViolationDetectedContent {
        Expectation: step_3_channel        // the channel established privacy
        Actor:       bob
        Severity:    High
        Description: "Private channel content shared outside channel boundary"
        Evidence:    [step_4_event, step_3_channel, external_share_event]
    }
    Causes:  [external_share_event]
}
```

### Step 11: Trust drops sharply

```
Event {
    Type:    "trust.updated"
    Source:  system
    Content: TrustUpdatedContent {
        Actor:    bob
        Previous: Score(0.52)
        Current:  Score(0.2)
        Domain:   "personal"
        Cause:    step_10_event
    }
    Causes:  [step_10_event]
}
```

### Step 12: Alice severs the channel
**Grammar:** Sever

```
Event {
    Type:    "edge.superseded"
    Source:  alice
    Content: EdgeSupersededContent {
        PreviousEdge: step_3_channel_edge
        NewEdge:      None              // severed, not replaced
        Reason:       step_10_event
    }
    Causes:  [step_10_event]
}
```

The channel is closed. No new entries can be shared. History remains — append-only.

### Step 13 (Much later): Forgiveness — Alice re-subscribes
**Grammar:** Forgive (Subscribe after Sever)

```
Event {
    Type:    "edge.created"
    Source:  alice
    Content: EdgeCreatedContent {
        From:      alice
        To:        bob
        EdgeType:  Subscription
        Weight:    Weight(0.2)          // lower weight than before — trust rebuilt slowly
        Direction: Centripetal
    }
    Causes:  [step_12_event]            // causally linked to the severance — history visible
}
```

The causal link from step 12 (Sever) to step 13 (resubscribe) IS the Forgive operation. The graph records that this relationship broke and was repaired. The history of betrayal and repair is intact.

## Trust Flow

```
Alice ↔ Bob trust in "personal":
  0.50 → 0.55 (reciprocal sharing)
  0.55 → 0.52 (decay from inactivity)
  0.52 → 0.20 (betrayal — privacy violation)
  0.20 → ... (slow rebuild after Forgive)
```

## Assertions

1. **Structural consent:** Step 4 entry is only visible to Bob after step 6 consent — not before
2. **Channel privacy:** Events within the channel are not queryable by actors outside it
3. **Bilateral consent:** The consent event requires both parties' signatures
4. **Betrayal detectable:** The system detects when channel content is shared outside the channel boundary
5. **Trust reflects relationship:** Trust drops sharply on betrayal, accumulates slowly on reciprocity
6. **History survives:** After Sever, the journal entries still exist. After Forgive, the betrayal history is still traversable.
7. **Forgive ≠ Forget:** The causal chain from betrayal through severance to forgiveness is complete and visible
8. **Decay without interaction:** Trust decays over time without new events

## What Higher Layers Add

- **Relationship Graph (L9):** Primitives that understand vulnerability (sharing personal content = vulnerable state), attunement (detecting reciprocity patterns), and repair cycles. Would surface "this relationship has a pattern of share-betray-repair-share" or flag escalating control patterns.
- **Ethics Graph (L7):** Would detect if the betrayal pattern suggests domestic abuse (one-sided control, isolation, escalating severity).
