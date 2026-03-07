# Composition Test: Justice Grammar (Layer 4)

Tests for the Justice Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph with Layer 3 community
actors: [alice (Human), bob (Human), judge_agent (AI), admin (Human)]
rules: [rule_1: "no spam", rule_2: "code of conduct"]
grammar: JusticeGrammar
```

## Operation Tests

### Legislate

**Input:** `grammar.Legislate({ text: "Members must disclose AI usage", scope: community_id, authority: admin })`
**Assertions:**
- Rule primitive activated
- Emits `rule.enacted` event
- Authority check: admin must have legislative authority
- Rule is queryable within jurisdiction

### Amend / Repeal

**Input:** `grammar.Amend({ rule: rule_1, amendment: "spam includes off-topic AI content" })`
**Assertions:**
- Emits `rule.amended` event caused by original rule
- Amendment linked to the rule it modifies
- `grammar.Repeal({ rule: rule_1 })` tombstones the rule

### File

**Input:** `grammar.File({ complaint: "bob posted spam", evidence: [post_id], plaintiff: alice })`
**Assertions:**
- DueProcess primitive activated
- Emits dispute event with evidence links
- Respondent (bob) notified
- Case created with adjudicator assignment

### Submit / Argue

**Input:** Both parties submit evidence and arguments.
**Assertions:**
- Evidence events link to the case
- Arguments reference rules and precedents
- Both parties get equal opportunity (DueProcess)

### Judge

**Input:** `grammar.Judge({ case: case_id, ruling: "violation confirmed", severity: "warning" })`
**Assertions:**
- Adjudication primitive renders ruling
- Ruling event includes reasoning and cited precedents
- Precedent is set (if Precedential modifier)
- DueProcess verified before ruling accepted

### Appeal

**Input:** `grammar.Appeal({ ruling: ruling_id, grounds: "no warning was given first", appellant: bob })`
**Assertions:**
- Appeal primitive activated
- New case created at higher authority level
- Original ruling suspended pending appeal
- Appeal event links to original ruling

### Enforce / Pardon

**Input:** `grammar.Enforce({ ruling: ruling_id, action: "7-day suspension" })`
**Assertions:**
- Enforcement action applied
- Actor's status updated
- `grammar.Pardon({ actor: bob, scope: community_id, authority: admin })` overrides

### Audit / Reform

**Input:** `grammar.Audit({ scope: community_id })` → `grammar.Reform({ rule: rule_2, proposal: "add appeals process" })`
**Assertions:**
- Audit reviews all actions against all rules
- Findings recorded as events
- Reform proposal linked to audit findings

## Named Function Tests

### Trial

**Input:** Full trial: File → Submit (both sides) → Argue (both sides) → Judge.
**Assertions:**
- All events form a coherent causal chain
- DueProcess satisfied at each step
- Ruling references all submitted evidence

### Injunction

**Input:** `grammar.Injunction({ target: bad_actor, action: "temporary ban", reason: "ongoing harm" })`
**Assertions:**
- Emergency modifier bypasses normal timeline
- Temporary enforcement applied immediately
- Full trial must follow

## Error Cases

| Case | Expected |
|------|----------|
| Judge without adjudicator authority | `Err(AuthorityError)` |
| Enforce without prior ruling | `Err(ValidationError.NoRuling)` |
| Appeal after appeal deadline | `Err(ValidationError.Expired)` |
| Pardon without high authority | `Err(AuthorityError.InsufficientLevel)` |

## Reference

- `docs/compositions/04-justice.md` — Justice Grammar specification
- `docs/layers/04-legal.md` — Layer 4 derivation
