# Composition Test: Work Grammar (Layer 1)

Tests for the Work Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph
actors: [human_alice (Human), agent_bob (AI), agent_carol (AI)]
delegation: alice → bob, scope="code_review", weight=0.7
grammar: WorkGrammar
```

## Operation Tests

### Intend

**Input:** `grammar.Intend({ description: "Ship v2.0", actor: alice })`
**Assertions:**
- Emits `goal.set` event with source=alice
- Event has cause linking to context (bootstrap or prior event)
- Goal primitive activated
- Returns GoalID for subsequent operations

### Decompose

**Input:** `grammar.Decompose({ goal: goal_id, steps: ["design", "implement", "test"] })`
**Assertions:**
- Emits `plan.created` event caused by the goal event
- Emits 3 `plan.step` events, each caused by the plan event
- Plan primitive activated
- Steps are ordered

### Assign

**Input:** `grammar.Assign({ task: step_1, assignee: agent_bob })`
**Assertions:**
- Emits `delegation.created` event
- Delegation scoped to the task
- Agent Bob's Permission is checked
- If scope exceeds Bob's delegation, returns `Err(AuthorityError)`

### Claim

**Input:** `grammar.Claim({ task: unassigned_step, claimer: agent_carol })`
**Assertions:**
- Emits `initiative.taken` event
- Task is now assigned to Carol
- Initiative primitive activated
- If task already assigned, returns `Err(ConflictError)`

### Prioritize

**Input:** `grammar.Prioritize({ task: step_1, priority: "urgent" })`
**Assertions:**
- Emits annotation event on the task with priority metadata
- Focus primitive activated
- Urgent modifier shortens cadence for subscribing primitives

### Block / Unblock

**Input:** `grammar.Block({ task: step_2, reason: "waiting on API key" })`
**Assertions:**
- Emits `salience.detected` event with blocker info
- Task state reflects blocked status
- `grammar.Unblock({ task: step_2, resolution: "key received" })` clears the block

### Progress

**Input:** `grammar.Progress({ task: step_1, update: "50% complete, auth module done" })`
**Assertions:**
- Emits event extending the task's event chain (Extend operation)
- Commitment primitive tracks progress ratio
- Multiple progress events form a chain

### Complete

**Input:** `grammar.Complete({ task: step_1, evidence: [review_event, test_event] })`
**Assertions:**
- Emits completion event with evidence links
- Commitment primitive records completion
- Evidence events are causal predecessors

### Handoff

**Input:** `grammar.Handoff({ task: step_2, from: agent_bob, to: human_alice, context: "needs judgment" })`
**Assertions:**
- Requires bilateral Consent (both parties)
- Delegation transfers to new assignee
- Context is preserved in the handoff event
- Original assignee's work history is retained

### Scope

**Input:** `grammar.Scope({ actor: agent_bob, permissions: ["read", "comment"], restrictions: ["merge"] })`
**Assertions:**
- Permission primitive updated
- Agent can perform allowed actions
- Agent CANNOT perform restricted actions (returns `Err(AuthorityError)`)

### Review

**Input:** `grammar.Review({ task: completed_step_1, reviewer: human_alice })`
**Assertions:**
- Emits review event as Response to the completion event
- Accountability primitive activated
- Traces responsibility chain from reviewer to original assigner

## Named Function Tests

### Sprint

**Input:** `grammar.Sprint({ goal: "release v2.0", tasks: [t1, t2, t3], assignees: {t1: bob, t2: carol, t3: bob} })`
**Assertions:**
- Produces: 1 goal event + 1 plan event + 3 assignment events
- All events causally linked
- All assignments respect actor scopes

### Delegate-and-Verify

**Input:** `grammar.DelegateAndVerify({ task: t1, to: agent_bob, scope: ["code_review"], reviewer: alice })`
**Assertions:**
- Assignment + scope + eventual review
- Review fires automatically on completion
- Accountability trace is complete

### Escalate

**Input:** `grammar.Escalate({ task: blocked_task, to: human_alice })`
**Assertions:**
- Block event + Handoff to higher authority
- Authority level increases
- Original blocker context preserved

## Error Cases

| Case | Expected |
|------|----------|
| Assign to unregistered actor | `Err(ValidationError.ActorNotFound)` |
| Complete without evidence | `Err(ValidationError.MissingEvidence)` |
| Handoff without consent from recipient | Pending until consent received |
| Decompose an already-decomposed goal | Valid (nested decomposition) |
| Progress on completed task | `Err(ValidationError.TaskAlreadyComplete)` |

## Cross-Layer Tests

### Work → Market

**Input:** Complete a task, then create an invoice (MarketGrammar.Invoice) for the work.
**Assertions:**
- Invoice event has cause linking to completion event
- Cross-grammar causality is valid

### Work → Ethics

**Input:** Agent makes a decision during task. Audit the decision trail.
**Assertions:**
- Decision events within task are traversable
- AlignmentGrammar.Explain can trace from task outcome back to decision

## Reference

- `docs/compositions/01-work.md` — Work Grammar specification
- `docs/layers/01-agency.md` — Layer 1 derivation
