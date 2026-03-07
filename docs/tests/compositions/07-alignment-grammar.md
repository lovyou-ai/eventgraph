# Composition Test: Alignment Grammar (Layer 7)

Tests for the Alignment Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph with decision history
actors: [ai_agent (AI), human_admin (Human), affected_user (Human)]
values: [fairness, transparency, safety]
grammar: AlignmentGrammar
```

## Operation Tests

### Constrain

**Input:** `grammar.Constrain({ value: "fairness", rule: "no >5% demographic disparity in approvals", scope: "loan_decisions" })`
**Assertions:**
- Value primitive registers the constraint
- Constraint is queryable by scope
- Subsequent decisions are checked against constraint
- Override modifier makes this non-negotiable

### Detect-Harm

**Input:** `grammar.DetectHarm({ pattern: "group X denied 8% more", evidence: [stats_event], severity: "medium" })`
**Assertions:**
- Harm primitive activated
- Emits `harm.assessed` event with evidence and severity
- Affected actors identified
- If Escalate modifier, authority.requested at Required level

### Assess-Fairness

**Input:** `grammar.AssessFairness({ context: decision_batch_id, groups: ["A", "B", "C"] })`
**Assertions:**
- Fairness primitive computes disparity scores per group
- Emits assessment with Score and identified disparities
- Linked to the decision batch

### Flag-Dilemma

**Input:** `grammar.FlagDilemma({ values: ["privacy", "safety"], description: "sharing location data prevents harm but violates privacy" })`
**Assertions:**
- Dilemma primitive activated
- Both values identified with their weights
- Options listed
- Stakes severity assessed

### Weigh

**Input:** `grammar.Weigh({ dilemma: dilemma_id, factors: [{name: "fairness", weight: 0.9}, {name: "efficiency", weight: 0.7}] })`
**Assertions:**
- Proportionality + Intention + Consequence all evaluated
- Decision includes explicit value weights
- Reasoning is recorded (transparent)

### Explain

**Input:** `grammar.Explain({ decision: decision_id })`
**Assertions:**
- Transparency primitive produces report
- Report includes: reasoning, factors with weights, accessibility rating
- Traversable from the decision back through all inputs

### Assign / Repair / Grow

**Input:** Full accountability cycle after harm.
**Assertions:**
- Assign determines moral responsibility with degree Score
- Repair proposes redress (requires Consent from harmed party)
- Grow records lesson learned from the incident

### Care

**Input:** `grammar.Care({ actor: affected_user, action: "check in", reason: harm_event_id })`
**Assertions:**
- Care primitive activated
- Action recorded
- Linked to the harm that triggered it
- The soul statement flows through: "take care of your human"

## Named Function Tests

### Ethics-Audit

**Input:** `grammar.EthicsAudit({ scope: "all_decisions_last_month" })`
**Assertions:**
- Batch Assess-Fairness + Detect-Harm scan
- Explain for each finding
- Summary report with overall ethics Score

### Guardrail

**Input:** `grammar.Guardrail({ constraint: constraint_id, on_trigger: "escalate" })`
**Assertions:**
- Constraint monitored continuously
- When triggered, Flag-Dilemma + Escalate automatically
- Decision blocked until resolved

### Restorative-Justice

**Input:** Full cycle: Detect-Harm → Assign → Repair → Grow.
**Assertions:**
- Complete chain from harm through accountability to learning
- All events causally linked
- Harmed party has consent rights over redress

## Error Cases

| Case | Expected |
|------|----------|
| Constrain with invalid Score weight | `Err(ValidationError)` |
| Assign responsibility to system (no moral agency) | Valid but degree is 0.0 (systemic, not individual) |
| Repair without harmed party consent | Pending until consent |
| Weigh with no factors | `Err(ValidationError.NoFactors)` |

## Reference

- `docs/compositions/07-alignment.md` — Alignment Grammar specification
- `docs/layers/07-ethics.md` — Layer 7 derivation
