# Architecture Test 3: Decision Maker Pluggability

Verify that `IDecisionMaker` implementations can be swapped and that the decision tree routing works correctly.

## Purpose

The decision tree engine routes through deterministic branches first, falling through to IIntelligence only when the tree can't handle it. This test verifies that the routing works, that decision makers can be swapped, and that the mechanical-to-intelligent evolution path functions.

## Setup

```
decision_tree: DecisionTree with 3 branches:
    1. Mechanical: if lines_changed < 100 → auto-approve
    2. Mechanical: if has_security_flag → require_human
    3. Semantic: evaluate code quality → IIntelligence

makers: [
    MechanicalDecisionMaker   // always uses tree, never falls through
    HybridDecisionMaker       // tree first, falls through to IIntelligence
    HumanDecisionMaker        // always requests authority
]
```

## Test Cases

### TC-3.1: Mechanical Branch Hit

**Input:** Decision input matching mechanical branch (lines_changed=50).
**Assertions:**
- Decision resolved without IIntelligence call
- Decision event includes `path: ["branch_1"]` showing which tree branches were taken
- Confidence is 1.0 (mechanical decisions are certain)
- No authority.requested event

### TC-3.2: Semantic Fallthrough

**Input:** Decision input that passes mechanical branches but needs intelligence (lines_changed=200, no security flag).
**Assertions:**
- Tree evaluates branches 1 and 2 (no match), falls to branch 3
- IIntelligence is called exactly once
- Decision event includes confidence from IIntelligence
- Decision path shows the fallthrough

### TC-3.3: Authority Escalation

**Input:** Decision input matching security flag branch.
**Assertions:**
- Decision requires authority (authority.requested event emitted)
- Decision is pending until authority.resolved
- authority.resolved → decision.completed

### TC-3.4: Swap Decision Maker

**Input:** Same input processed by each maker.
**Assertions:**
- MechanicalDecisionMaker: resolves via tree only
- HybridDecisionMaker: resolves via tree then intelligence
- HumanDecisionMaker: always emits authority.requested
- All produce valid Decision events with correct causes

### TC-3.5: Decision Tree Evolution

**Input:** Process 100 similar decisions. 95 auto-approved by human after authority request.
**Assertions:**
- System detects the pattern (via Automation primitive at L5 or similar)
- Decision tree can be updated to add a new mechanical branch
- Subsequent decisions matching the pattern resolve mechanically
- The evolution is recorded as events (auditable)

### TC-3.6: Decision Tree Override

**Input:** Replace a decision tree branch at runtime.
**Assertions:**
- New branch takes effect on next decision
- Old branch no longer evaluates
- Change is recorded as an event
- Existing in-flight decisions complete with the old tree

### TC-3.7: Custom Decision Maker Registration

**Input:** Register a custom IDecisionMaker for a specific scope.
**Assertions:**
- Decisions within that scope route to the custom maker
- Decisions outside that scope route to the default maker
- Scope boundaries are enforced

### TC-3.8: Decision Audit Trail

**Input:** Process a decision, then traverse the audit trail.
**Assertions:**
- Decision event has causes linking to the input events
- Decision path (which branches were evaluated) is in the event content
- Confidence, authority chain, and trust weights are all recorded
- Full chain is traversable from outcome back to original trigger

## Error Cases

| Case | Input | Expected |
|------|-------|----------|
| No matching branch | Input matches no tree branches and no fallthrough | `Err(DecisionError.NoPath)` |
| Circular tree | Branch A refers to branch B refers to A | Detected at tree construction, `Err(ValidationError)` |
| Conflicting branches | Two branches both match | First match wins (deterministic ordering) |

## Reference

- `docs/interfaces.md` — IDecisionMaker interface specification
- `docs/decision-trees.md` — Decision tree mechanics and evolution
