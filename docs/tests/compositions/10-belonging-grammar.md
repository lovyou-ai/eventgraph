# Composition Test: Belonging Grammar (Layer 10)

Tests for the Belonging Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph with community
actors: [alice (Human, established member), newcomer_bob (Human), steward_carol (Human)]
community: "eventgraph-contributors"
grammar: BelongingGrammar
```

## Operation Tests

### Settle

**Input:** `grammar.Settle({ actor: newcomer_bob, community: community_id })`
**Assertions:**
- Home primitive activated
- Belonging Score starts low, increases with participation
- Subscribe to community established

### Contribute

**Input:** `grammar.Contribute({ actor: newcomer_bob, type: "bug fix", value: 0.6 })`
**Assertions:**
- Contribution event emitted
- Community contribution ledger updated
- Belonging Score increases for contributor

### Include

**Input:** `grammar.Include({ community: community_id, action: "improve onboarding docs", beneficiary: "newcomers" })`
**Assertions:**
- Inclusion primitive activated
- Barriers assessment updated
- Inclusion Score for community increases

### Practice

**Input:** `grammar.Practice({ community: community_id, tradition: "friday-review" })`
**Assertions:**
- Tradition primitive notes observance
- Adherence Score updated
- Collective modifier: all participants recorded

### Steward / Sustain

**Input:** `grammar.Steward({ actor: carol, resource: "test infrastructure", community: community_id })`
**Assertions:**
- Commons primitive identifies resource with steward
- `grammar.Sustain({ community: community_id })` assesses long-term viability
- Sustainability risks identified

### Pass-On

**Input:** `grammar.PassOn({ from: carol, to: alice, scope: "test infrastructure" })`
**Assertions:**
- Succession requires Consent from both parties
- Authority and responsibility transfer
- Succession event links old and new steward

### Celebrate / Tell / Gift

**Input:** Community milestone celebration.
**Assertions:**
- Celebrate marks achievement with significance Score
- Ceremony event records participants
- Tell adds chapter to community story
- Gift records unconditional giving (no obligation created)

## Named Function Tests

### Onboard

**Input:** `grammar.Onboard({ newcomer: bob, community: community_id })`
**Assertions:**
- Include + Settle + Practice (introductory) + Contribute (first)
- Full welcome experience
- Newcomer's belonging Score tracks through each step

### Festival

**Input:** `grammar.Festival({ community: community_id, occasion: "v2.0 launch" })`
**Assertions:**
- Celebrate (Collective) + Practice + Tell + Gift
- All community members can participate
- Story chapter records the event

### Succession

**Input:** `grammar.Succession({ community: community_id, role: "maintainer", from: carol, to: alice })`
**Assertions:**
- Sustain + Pass-On + Celebrate (Witnessed) + Tell
- Full generational transfer with community record

## Error Cases

| Case | Expected |
|------|----------|
| Pass-On without Steward relationship | `Err(ValidationError.NotSteward)` |
| Practice non-existent tradition | Creates new tradition (Founding modifier) |
| Settle in private community without invitation | `Err(AuthorityError.InvitationRequired)` |

## Reference

- `docs/compositions/10-belonging.md` — Belonging Grammar specification
- `docs/layers/10-community.md` — Layer 10 derivation
