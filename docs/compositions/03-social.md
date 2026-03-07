# Social Grammar (Layer 3: Society)

The grammar for user-owned social platforms where communities set their own norms.

**This is the original grammar derived in Post 35.** The full specification lives in `docs/grammar.md`. This file documents the Layer 3 primitive mappings and Society-specific extensions.

## Operations (15)

The 15 social grammar operations are defined in `docs/grammar.md`:

| # | Operation | Layer 3 Primitives Used |
|---|-----------|------------------------|
| 1 | Emit | — (Layer 0 only) |
| 2 | Respond | — (Layer 0 only) |
| 3 | Derive | — (Layer 0 only) |
| 4 | Extend | — (Layer 0 only) |
| 5 | Retract | Sanction (when retraction is norm-enforced) |
| 6 | Annotate | Convention (when annotation reflects community practice) |
| 7 | Acknowledge | Status (acknowledgement from high-status member carries weight) |
| 8 | Propagate | Influence (propagation by influential members has wider reach) |
| 9 | Endorse | Reputation (endorsement builds community standing) |
| 10 | Subscribe | Inclusion (subscription to community = joining) |
| 11 | Channel | — (Layer 0 only) |
| 12 | Delegate | Role (delegation within community context assigns role) |
| 13 | Consent | Norm (collective consent establishes norms) |
| 14 | Sever | Exclusion (severing from community = exile) |
| 15 | Merge | Solidarity (merging community threads = cohesion) |

## Society-Specific Extensions (5)

Operations that emerge when the social grammar meets Layer 3 primitives:

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 16 | **Norm** | governance/establish | Establish a shared behavioural expectation | Norm + Consent (community) |
| 17 | **Moderate** | governance/enforce | Enforce community norms on content | Sanction + Retract or Annotate |
| 18 | **Elect** | governance/role | Assign a community role through collective decision | Role + Consent (community) |
| 19 | **Welcome** | membership/inbound | Structured onboarding of new member | Inclusion + Invite (Endorse + Subscribe) |
| 20 | **Exile** | membership/outbound | Structured removal of member | Exclusion + Sever + Sanction |

## Modifiers (3)

From `docs/grammar.md`: Transient, Nascent, Conditional.

## Named Functions (8)

From `docs/grammar.md`: Recommend, Challenge, Curate, Collaborate, Forgive, Invite, Memorial, Transfer.

Plus Society-specific functions:

| Function | Composition | Purpose |
|----------|------------|---------|
| **Poll** | Norm (proposed) + Consent (batch) | Quick community sentiment check |
| **Schism** | Norm (conflicting) + Exile (faction) + new community subgraph | Community splits over irreconcilable norms |
| **Federation** | Consent (bilateral between communities) + Delegate (cross-community) | Communities cooperate while maintaining autonomy |

## Reference

- `docs/grammar.md` — The canonical social grammar specification
- `docs/layers/03-society.md` — Layer 3 derivation
- `docs/primitives.md` — Layer 3 primitive specifications
- `docs/tests/primitives/04-community-governance.md` — Integration test scenario
