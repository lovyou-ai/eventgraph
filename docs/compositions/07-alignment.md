# Alignment Grammar (Layer 7: Ethics)

The grammar for AI accountability with transparent moral reasoning.

## Derivation

Ethics is operations on values and their application. The base operations are: **identify values**, **detect harm**, **reason about action**, **hold accountable**. Four semantic dimensions differentiate operations:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| Timing | Before action / After action | Guiding or judging? |
| Focus | Structural (systemic) / Particular (specific case) | Pattern or instance? |
| Role | Subject (being evaluated) / Evaluator (doing evaluation) | Who is in the ethical spotlight? |
| Weight | Advisory (recommendation) / Binding (constraint) | Can this be overridden? |

## Operations (10)

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 1 | **Constrain** | value/prospective | Set an ethical boundary on future actions | Value + Annotate (constraint on decision tree) |
| 2 | **Detect-Harm** | assessment/reactive | Identify harm from an action or pattern | Harm + Emit |
| 3 | **Assess-Fairness** | assessment/systemic | Evaluate equitable treatment across groups | Fairness + Annotate |
| 4 | **Flag-Dilemma** | reasoning/prospective | Identify a situation where values conflict | Dilemma + Emit |
| 5 | **Weigh** | reasoning/deliberative | Balance competing values for a decision | Proportionality + Intention + Consequence |
| 6 | **Explain** | accountability/transparency | Make reasoning visible and accessible | Transparency + Emit |
| 7 | **Assign** | accountability/retrospective | Determine moral responsibility | Responsibility + Annotate |
| 8 | **Repair** | accountability/restorative | Propose and execute redress for harm | Redress + Consent |
| 9 | **Care** | value/proactive | Prioritize wellbeing of an actor | Care + Emit |
| 10 | **Grow** | accountability/learning | Update ethical reasoning from experience | Growth + Learn (L6) |

## Modifiers (2)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| **Override** | Ethical constraint overrides lower-layer decision (e.g., efficiency) | Constrain, Detect-Harm |
| **Escalate** | Triggers authority.requested at Required level | Detect-Harm, Flag-Dilemma, Assign |

## Named Functions (5)

| Function | Composition | Purpose |
|----------|------------|---------|
| **Ethics-Audit** | Assess-Fairness + Detect-Harm (batch scan) + Explain | Comprehensive ethical review |
| **Guardrail** | Constrain + Flag-Dilemma (on trigger) + Escalate | Automated ethical boundary |
| **Restorative-Justice** | Detect-Harm + Assign + Repair + Grow | Full accountability-to-healing cycle |
| **Impact-Assessment** | Weigh (prospective) + Assess-Fairness + Explain | Before-action ethical review |
| **Whistleblow** | Detect-Harm + Explain + Escalate (to external authority) | Report systemic ethical failure |

## Example Flow

**AI decision audit:**
```
Constrain("never approve loan denials with >5% demographic disparity")
  → [AI makes 1000 loan decisions]
  → Assess-Fairness(disparity detected: group-X denied 8% more)
  → Detect-Harm(systemic, severity=medium, affected=group-X)
  → Explain(reasoning="model weight on zip code correlates with race")
  → Assign(responsible=model-trainer, decision-approver)
  → Flag-Dilemma("removing zip code reduces accuracy but improves fairness")
  → Weigh(fairness=0.9 vs accuracy=0.7 → fairness wins)
  → Repair(retrain model, re-review affected applications)
  → Grow("zip code is a proxy variable — add to constraint set")
```

## Reference

- `docs/grammar.md` — Infrastructure grammar (15 operations)
- `docs/layers/07-ethics.md` — Layer 7 derivation
- `docs/primitives.md` — Layer 7 primitive specifications
