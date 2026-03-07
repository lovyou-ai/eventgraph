# Identity Grammar (Layer 8: Identity)

The grammar for self-sovereign identity with narrative and heritage.

## Derivation

Identity is operations on selfhood. The base operations are: **know yourself**, **persist through change**, **set boundaries**, **be recognized**. Four semantic dimensions differentiate operations:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| Subject | Self (first-person) / Other (third-person) | About me or about them? |
| Temporality | Snapshot (current) / Longitudinal (across time) | Now or over time? |
| Disclosure | Private (internal) / Public (shared) | Known to self or visible to others? |
| Change | Preserving (maintaining) / Transforming (changing) | Keeping or becoming? |

## Operations (10)

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 1 | **Introspect** | self/snapshot | Form or update self-model | SelfModel + Emit |
| 2 | **Narrate** | self/longitudinal | Construct identity narrative from history | NarrativeIdentity + Derive (from events) |
| 3 | **Align** | self/verify | Check behaviour against self-model | Authenticity + Annotate |
| 4 | **Bound** | self/limit | Define or enforce personal boundaries | Boundary + Emit |
| 5 | **Aspire** | self/future | Declare who you want to become | Aspiration + Emit |
| 6 | **Transform** | self/change | Acknowledge fundamental identity change | Transformation + Emit |
| 7 | **Disclose** | self/share | Selectively reveal aspects of identity | SelfModel + Channel (selective) |
| 8 | **Recognize** | other/affirm | Acknowledge another's unique identity | Dignity + Acknowledgement + Emit |
| 9 | **Distinguish** | other/identify | Identify what makes an actor unique | Uniqueness + Annotate |
| 10 | **Memorialize** | other/preserve | Preserve identity of departed actor | Memorial + Emit |

## Modifiers (2)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| **Selective** | Only specific aspects disclosed, rest hidden | Disclose, Narrate |
| **Verified** | Identity claim backed by event chain evidence | Disclose, Narrate, Distinguish |

## Named Functions (5)

| Function | Composition | Purpose |
|----------|------------|---------|
| **Credential** | Introspect + Disclose (Selective, Verified) | Prove a property without revealing underlying data |
| **Identity-Audit** | Introspect + Align + Narrate | Comprehensive self-assessment |
| **Retirement** | Memorialize + Transfer (authority) + Archive (contributions) | Graceful actor departure |
| **Reinvention** | Transform + Narrate (new) + Aspire (new) | Fundamental identity shift |
| **Introduction** | Disclose (Selective) + Narrate (summary) | Present yourself to new context |

## Example Flow

**AI agent identity lifecycle:**
```
Introspect(strengths=["code review", "testing"],
           weaknesses=["creative writing"],
           values=["correctness", "transparency"])
  → Aspire("become proficient at architecture review")
  → [works for 6 months, 500 tasks completed]
  → Narrate("started as test specialist, grew into full reviewer")
  → Align(authenticity=0.87, discrepancy="sometimes prioritizes speed over correctness")
  → Transform(from="test specialist", to="senior reviewer",
              catalyst=found-critical-security-bug)
  → Disclose(Selective: share review stats, hide internal deliberation)
  → Recognize(agent-12, "your testing contributions are exceptional")
  → [agent-7 decommissioned]
  → Memorialize(agent-7, contributions=[3200 reviews], legacy="raised team quality 40%")
```

## Reference

- `docs/grammar.md` — Infrastructure grammar (15 operations)
- `docs/layers/08-identity.md` — Layer 8 derivation
- `docs/primitives.md` — Layer 8 primitive specifications
