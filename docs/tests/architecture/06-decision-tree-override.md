# Architecture Test 6: Decision Tree Override

Verify that decision trees can be replaced, extended, and evolved at runtime.

## Purpose

Decision trees are the mechanical-to-intelligent continuum. They must be modifiable at runtime — branches added, conditions changed, thresholds tuned — without restarting the system. This is core to SELF-EVOLVE.

## Setup

```
tree: DecisionTree {
    branches: [
        { condition: Threshold("risk", 0.3), action: "auto_approve" }
        { condition: Threshold("risk", 0.7), action: "request_review" }
        { condition: Semantic("evaluate risk"), action: "escalate" }
    ]
}
```

## Test Cases

### TC-6.1: Add Branch at Runtime

**Input:** Add a new mechanical branch: `if category == "docs" → auto_approve`.
**Assertions:**
- New branch evaluates on subsequent decisions
- Addition is recorded as an event (`decision_tree.updated`)
- Event includes before/after tree state

### TC-6.2: Remove Branch at Runtime

**Input:** Remove the Semantic branch.
**Assertions:**
- Subsequent decisions only evaluate mechanical branches
- If no branch matches, decision returns `Err(DecisionError.NoPath)`
- Removal recorded as event

### TC-6.3: Modify Threshold

**Input:** Change risk threshold from 0.3 to 0.5 for auto_approve.
**Assertions:**
- Decisions with risk 0.4 now go to request_review (were auto_approve)
- Modification recorded as event with old/new values

### TC-6.4: Replace Entire Tree

**Input:** Replace the tree with a completely new one.
**Assertions:**
- Old tree no longer evaluates
- New tree takes effect on next decision
- Replacement recorded as event
- In-flight decisions under old tree complete normally

### TC-6.5: Branch Ordering

**Input:** Add two branches that could both match the same input.
**Assertions:**
- First matching branch wins (deterministic)
- Ordering is explicit, not implicit
- Re-ordering is possible and recorded

### TC-6.6: Semantic to Mechanical Migration

**Input:** Track that a Semantic branch resolves the same way 50 times consecutively.
**Assertions:**
- System can propose replacing the Semantic branch with a Mechanical one
- Proposal is recorded as an event
- If approved, the branch is replaced
- Cost savings are measurable (no IIntelligence call)

### TC-6.7: Rollback

**Input:** Update tree, observe bad outcomes, rollback to previous version.
**Assertions:**
- Previous tree version is retrievable from event history
- Rollback restores the old tree
- Rollback is itself recorded as an event
- Decisions resume with old tree behavior

## Reference

- `docs/decision-trees.md` — Decision tree mechanics
- `docs/interfaces.md` — IDecisionMaker specification
