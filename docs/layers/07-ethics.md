# Layer 7: Ethics

## Derivation

### The Gap

Layer 6 models knowledge, facts, and truth, but cannot distinguish what *is* from what *ought to be*. A system can know that an action occurred and that it violated a rule, but it cannot reason about whether the rule itself is just. "This is true" is Layer 6. "This is right" requires something new.

**Test:** Can you express "The rule is being followed correctly, but the rule itself is unfair to group X, and the person who enforced it meant well but caused disproportionate harm" in Layer 6? You can establish facts about the rule and its enforcement. But "unfair" (evaluating justice), "meant well" (assessing intention), and "disproportionate harm" (weighing consequences against values) have no Layer 6 representation. Knowledge is not wisdom.

### The Transition

**Is → Ought**

Facts become values. The fundamental new capacity: reasoning about what should be done, not just what is or was done. Evaluating actions against values, assessing harm, weighing competing goods, and holding actors morally accountable.

### Base Operations

What can an ethical reasoner DO that a knower cannot?

1. **Evaluate** — assess actions against values
2. **Detect harm** — identify when actions cause damage
3. **Weigh** — balance competing values and consequences
4. **Hold accountable** — assign moral responsibility

### Semantic Dimensions

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| **Focus** | Agent (who acted) / Action (what was done) / Outcome (what resulted) | What aspect is being evaluated? |
| **Valence** | Positive (good, beneficial) / Negative (harmful, unjust) | Is this about promoting good or preventing harm? |
| **Temporality** | Prospective (before action) / Retrospective (after action) | Guiding or judging? |
| **Scope** | Particular (specific case) / Systemic (pattern or structure) | One instance or a structural issue? |

### Decomposition

**Group 0 — Value** (what matters)

| Primitive | Focus | Valence | Temporality | Scope | What it does |
|-----------|-------|---------|-------------|-------|--------------|
| **Value** | Action | Positive | Prospective | Systemic | Identifying and weighting values |
| **Harm** | Outcome | Negative | Retrospective | Particular | Detecting and measuring harm |
| **Fairness** | Outcome | Positive | Retrospective | Systemic | Evaluating equitable treatment |
| **Care** | Agent | Positive | Prospective | Particular | Prioritising wellbeing |

Value lifecycle: values are identified (Value) → harm is detected when they're violated (Harm) → systemic patterns of inequity are assessed (Fairness) → and wellbeing is actively prioritised (Care). The soul statement — "take care of your human, humanity, and yourself" — flows through Care.

**Group 1 — Judgement** (what should be done)

| Primitive | Focus | Valence | Temporality | Scope | What it does |
|-----------|-------|---------|-------------|-------|--------------|
| **Dilemma** | Action | Both | Prospective | Particular | Situations where values conflict |
| **Proportionality** | Action | Negative | Retrospective | Particular | Ensuring responses match severity |
| **Intention** | Agent | Both | Retrospective | Particular | Evaluating purpose behind actions |
| **Consequence** | Outcome | Both | Retrospective | Particular | Evaluating outcomes of actions |

Judgement lifecycle: conflicts between values are identified (Dilemma) → responses are calibrated to severity (Proportionality) → purposes are assessed (Intention) → and outcomes are evaluated (Consequence). Together, Intention + Consequence capture the agent-focused and outcome-focused dimensions of moral reasoning.

**Group 2 — Accountability** (answering for what was done)

| Primitive | Focus | Valence | Temporality | Scope | What it does |
|-----------|-------|---------|-------------|-------|--------------|
| **Responsibility** | Agent | Negative | Retrospective | Particular | Who is morally responsible |
| **Transparency** | Action | Positive | Prospective | Systemic | Making reasoning visible |
| **Redress** | Outcome | Positive | Retrospective | Particular | Making things right after harm |
| **Growth** | Agent | Positive | Retrospective | Particular | Learning from ethical failures |

Accountability lifecycle: moral responsibility is assigned (Responsibility) → reasoning is made visible (Transparency) → harm is repaired (Redress) → and actors learn from failure (Growth). This differs from Layer 1 Accountability — that traces causal chains, this assesses moral weight.

### Gap Analysis

| Behavior | Maps to | Notes |
|----------|---------|-------|
| "Privacy is important to our community" | Value | |
| "This action caused real damage to Alice" | Harm | |
| "Group X receives 30% fewer approvals" | Fairness | |
| "Check on Bob — he's been struggling" | Care | |
| "We can't protect privacy AND ensure safety here" | Dilemma | |
| "A permanent ban for a first offense is excessive" | Proportionality | |
| "She didn't mean to cause harm" | Intention | |
| "The policy reduced fraud but increased exclusion" | Consequence | |
| "The decision-maker is accountable for this outcome" | Responsibility | |
| "Here's why the AI made that recommendation" | Transparency | |
| "We owe Alice compensation for the wrongful ban" | Redress | |
| "After that incident, we fundamentally changed how we handle disputes" | Growth | |
| Ethical AI audit | Fairness + Transparency + Bias (L6) | Cross-layer composition |
| Restorative justice | Harm + Responsibility + Redress + Growth | Composition |
| Whistleblower protection | Value + Transparency + Rights (L4) | Cross-layer composition |

**No gaps found.**

### Completeness Argument

1. **Dimensional coverage:** The {focus, valence, temporality, scope} space is covered. The three ethical perspectives — virtue ethics (agent-focused), deontological (action-focused), and consequentialist (outcome-focused) — are all represented through the focus dimension.

2. **Ethical theory coverage:** Value covers axiology. Dilemma + Proportionality cover practical reasoning. Intention + Consequence span the deontological-consequentialist spectrum. Responsibility + Transparency + Redress + Growth cover the full accountability cycle from assignment through repair to learning.

3. **Layer boundary:** None of these require concepts from Layer 8 (Identity). Ethics reasons about what should be done but treats actors as interchangeable moral agents. The concept of an actor's unique character, personal history, and sense of self is Layer 8's gap.

---

## Primitive Specifications

Full specifications in `docs/primitives.md` (Layer 7 section).

## Product Graph

Layer 7 maps to the **Ethics Graph** — AI alignment with transparent moral reasoning. See `docs/product-layers.md`.

## Reference

- `docs/derivation-method.md` — The derivation method
- `docs/primitives.md` — Full specifications
- `docs/product-layers.md` — Ethics Graph
