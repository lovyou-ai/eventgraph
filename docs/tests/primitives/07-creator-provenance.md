# Scenario 7: Creator Provenance

An artist creates a work, documenting their creative process — inspiration, drafts, tools, influences. The chain of creation IS the provenance. AI-generated content is structurally distinguishable: it has a single Emit with no creative history. Human work has a rich Derive chain.

**Product graph:** Culture Graph (Layer 12)

---

## Actors

| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `artist_kai` | Human | 0.6 | Visual artist |
| `mentor_luna` | Human | 0.9 | Established artist, mentor |
| `ai_generator` | AI | 0.3 | AI image generation tool |
| `system` | System | 1.0 | The graph itself |

## Setup

- Graph bootstrapped
- All actors registered
- `artist_kai` Subscribes to `mentor_luna` (learning relationship)
- `mentor_luna` has high trust from years of verified creative output

## Event Sequence

### Step 1: Kai encounters an inspiring work
**Grammar:** Annotate (metadata on existing content)

```
Event {
    Type:    "creative.inspiration"
    Source:  artist_kai
    Content: CreativeInspirationContent {
        Description: "Saw Luna's watercolour series 'Morning Light' at the gallery"
        Reference:   luna_artwork_event     // link to Luna's published work
        Medium:      "watercolour"
        Elements:    ["light diffusion", "colour layering", "negative space"]
    }
    Causes:  [luna_artwork_event]
}
```

### Step 2: Kai studies the technique
**Grammar:** Derive (learning derived from inspiration)

```
Event {
    Type:    "creative.study"
    Source:  artist_kai
    Content: CreativeStudyContent {
        Inspiration: step_1_event
        Technique:   "Wet-on-wet watercolour layering"
        Notes:       "Practiced colour gradients for 3 hours. Key insight: timing of second layer application."
        Duration:    Duration(10800000000000)   // 3 hours
    }
    Causes:  [step_1_event]
}
```

### Step 3: Kai creates a first draft
**Grammar:** Derive (creative output derived from study)

```
Event {
    Type:    "creative.draft"
    Source:  artist_kai
    Content: CreativeDraftContent {
        Title:       "Evening Harbour (draft 1)"
        Medium:      "watercolour"
        Influences:  [step_1_event, step_2_event]
        ArtifactHash: Hash("...")     // hash of the digital image — provenance without hosting
        Notes:       "First attempt. Colours too muddy in the sky area."
    }
    Causes:  [step_2_event]
}
```

### Step 4: Kai asks Luna for feedback
**Grammar:** Channel + Respond

```
// Kai shares draft via channel
Event {
    Type:    "creative.feedback_request"
    Source:  artist_kai
    Content: CreativeFeedbackRequestContent {
        Draft:   step_3_event
        Question: "The sky colours are muddying. How do you handle wet-on-wet timing?"
    }
    Causes:  [step_3_event]
}

// Luna responds with guidance
Event {
    Type:    "creative.feedback"
    Source:  mentor_luna
    Content: CreativeFeedbackContent {
        Draft:    step_3_event
        Feedback: "Wait until the first layer is damp, not wet. The sheen should be gone but the paper still cool to touch."
        Technique: "Touch test for wet-on-wet timing"
    }
    Causes:  [step_4_request_event]
}
```

### Step 5: Kai creates a second draft
**Grammar:** Derive (revision, caused by feedback)

```
Event {
    Type:    "creative.draft"
    Source:  artist_kai
    Content: CreativeDraftContent {
        Title:        "Evening Harbour (draft 2)"
        Medium:       "watercolour"
        Influences:   [step_1_event, step_2_event, step_4_feedback_event]
        ArtifactHash: Hash("...")
        Notes:        "Applied Luna's touch test. Sky much cleaner. Harbour reflections working."
        PreviousDraft: step_3_event
    }
    Causes:  [step_3_event, step_4_feedback_event]
}
```

### Step 6: Kai publishes the final work
**Grammar:** Emit

```
Event {
    Type:    "creative.published"
    Source:  artist_kai
    Content: CreativePublishedContent {
        Title:        "Evening Harbour"
        Medium:       "watercolour"
        ArtifactHash: Hash("...")
        Provenance:   [step_1_event, step_2_event, step_3_event, step_5_event]
        Influences:   [luna_artwork_event]
        Tools:        ["Winsor & Newton watercolours", "Arches 300gsm cold press"]
    }
    Causes:  [step_5_event]
}
```

### Step 7: Luna endorses the work
**Grammar:** Endorse

```
Event {
    Type:    "edge.created"
    Source:  mentor_luna
    Content: EdgeCreatedContent {
        From:      mentor_luna
        To:        artist_kai
        EdgeType:  Endorsement
        Weight:    Weight(0.6)
        Direction: Centripetal
        Scope:     Some("visual_art")
    }
    Causes:  [step_6_event]
}
```

Luna's endorsement is reputation-staked — her trust (0.9) backs this endorsement.

### Step 8: Trust updated

```
Event {
    Type:    "trust.updated"
    Source:  system
    Content: TrustUpdatedContent {
        Actor:    artist_kai
        Previous: Score(0.6)
        Current:  Score(0.68)
        Domain:   "visual_art"
        Cause:    step_7_event
    }
    Causes:  [step_7_event]
}
```

---

### Contrast: AI-generated content

Compare Kai's creative chain with AI-generated content:

```
// AI-generated image — single event, no creative history
Event {
    Type:    "creative.published"
    Source:  ai_generator
    Content: CreativePublishedContent {
        Title:        "Harbour at Sunset"
        Medium:       "digital"
        ArtifactHash: Hash("...")
        Provenance:   []              // empty — no creative process
        Influences:   []              // no declared influences
        Tools:        ["stable_diffusion_v3"]
    }
    Causes:  [prompt_event]           // only cause is the prompt
}
```

**Structural difference:**
- Kai's work: 8 events, rich Derive chain (inspiration → study → draft → feedback → revision → publication), mentor endorsement, 3-hour study session
- AI work: 1-2 events, no Derive chain, no study, no feedback, no iterative refinement

The graph doesn't JUDGE AI content. It makes the provenance VISIBLE. Consumers decide what they value.

## Assertions

1. **Creative chain traversable:** Traverse from step_6 (published) back through drafts, feedback, study, to inspiration — complete creative provenance
2. **Influence declared:** step_1 causally links to Luna's work — influence is explicit, not hidden
3. **Iteration visible:** Two drafts exist (step_3, step_5). The revision is caused by feedback (step_4) — the improvement process is documented
4. **Mentor contribution recorded:** Luna's feedback is on the graph. Her contribution to the work is visible without ownership claims.
5. **AI content structurally different:** AI-generated content has no Derive chain, no study events, no iterative drafts — structurally distinguishable without content analysis
6. **Endorsement staked:** Luna's endorsement carries her reputation weight. If Kai plagiarises later, Luna's endorsement history is part of the record.
7. **Tradition visible:** The chain from Luna's work through Kai's study to Kai's publication shows tradition transmission — technique flowing from teacher to student
8. **Artifact integrity:** ArtifactHash allows verification that the digital image hasn't been modified since publication

## What Higher Layers Add

- **Culture Graph (L12):** Primitives that understand tradition transmission (teacher → student chains), creative authenticity patterns, and meaning preservation. Would detect "this technique has been transmitted through 5 generations of artists" or flag "this influence chain was fabricated."
- **Knowledge Graph (L6):** Links creative works to art historical context — movements, periods, stylistic evolution across artists.
- **Identity Graph (L8):** Kai's artistic identity emerges from his body of work. Selective disclosure: prove "I have 20 endorsed watercolour works" without revealing the works themselves.
