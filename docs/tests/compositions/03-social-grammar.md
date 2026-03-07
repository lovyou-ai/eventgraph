# Composition Test: Social Grammar (Layer 3)

Tests for the Social Grammar operations and Layer 3 extensions as SDK-level APIs.

## Setup

```
graph: initialized EventGraph
actors: [alice (Human), bob (Human), carol (Human), mod_bot (AI)]
community: "dev-community" subgraph with norms
grammar: SocialGrammar (15 base + 5 extensions)
```

## Base Operation Tests

The 15 base operations are tested in `docs/conformance/`. This file tests the Layer 3 extensions and cross-layer behavior.

### Norm

**Input:** `grammar.Norm({ description: "All PRs require review", community: community_id })`
**Assertions:**
- Requires community Consent (not just one actor's declaration)
- Emits `norm.established` event
- Norm primitive activated
- Subsequent violations are detectable

### Moderate

**Input:** `grammar.Moderate({ content: post_id, norm: norm_id, action: "retract" })`
**Assertions:**
- Sanction primitive activated
- Content retracted (Retract operation)
- Moderation event links to violated norm
- Actor (mod_bot) must have moderator Role

### Elect

**Input:** `grammar.Elect({ role: "moderator", candidate: carol, community: community_id })`
**Assertions:**
- Requires community Consent (vote)
- Role primitive assigns role to carol
- Election event records vote results
- Role has associated permissions

### Welcome

**Input:** `grammar.Welcome({ newcomer: new_actor, community: community_id, sponsor: alice })`
**Assertions:**
- Inclusion primitive activated
- Sponsor's endorsement creates initial trust
- Newcomer gets Subscribe to community
- Onboarding tradition triggered (if defined)

### Exile

**Input:** `grammar.Exile({ actor: bad_actor, community: community_id, reason: "repeated norm violations" })`
**Assertions:**
- Requires authority (community vote or moderator decision)
- Exclusion primitive activated
- Sever removes subscriptions
- Sanction event records reason
- Actor can no longer emit to community subgraph

## Named Function Tests

### Poll

**Input:** `grammar.Poll({ question: "Adopt TypeScript?", community: community_id })`
**Assertions:**
- Norm proposed + batch Consent
- Each community member can vote once
- Result tallied after deadline
- Result event summarizes votes

### Schism

**Input:** Community splits over irreconcilable norm conflict.
**Assertions:**
- Two new community subgraphs created
- Members self-select via Subscribe
- History preserved in both (shared past, divergent future)
- Each community can set independent norms

### Federation

**Input:** Two communities agree to cooperate.
**Assertions:**
- Bilateral Consent between community representatives
- Cross-community Delegate (scoped)
- Federated queries work across both communities
- Each community retains sovereignty over its norms

## Error Cases

| Case | Expected |
|------|----------|
| Moderate without moderator Role | `Err(AuthorityError)` |
| Exile without community authority | `Err(AuthorityError)` |
| Welcome to full community (if capacity limit) | `Err(ValidationError.CapacityExceeded)` |
| Norm that contradicts a Right (L4) | `Err(ValidationError.RightsViolation)` |

## Reference

- `docs/grammar.md` — The 15 base operations
- `docs/compositions/03-social.md` — Layer 3 extensions
- `docs/layers/03-society.md` — Layer 3 derivation
- `docs/tests/primitives/04-community-governance.md` — Integration test scenario
