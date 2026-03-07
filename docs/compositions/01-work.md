# Work Grammar (Layer 1: Agency)

The grammar for task management where AI agents and humans operate on the same graph.

## Derivation

Work is operations on tasks. The base operations are: **create work**, **assign work**, **track work**, **complete work**. Four semantic dimensions differentiate operations:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| Granularity | Atomic (single task) / Compound (decomposed) | One thing or many? |
| Direction | Top-down (from goal) / Bottom-up (from observation) | Planned or emergent? |
| Actor | Self (own work) / Other (delegation) | Who does the work? |
| Binding | Tentative (can change) / Committed (bound) | Is this a plan or a promise? |

## Operations (12)

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 1 | **Intend** | task/creative | Declare a goal or desired outcome | Goal + Emit |
| 2 | **Decompose** | task/structural | Break a goal into actionable steps | Plan + Derive (subtasks from goal) |
| 3 | **Assign** | task/delegation | Give work to a specific actor | Delegation + Delegate (grammar) |
| 4 | **Claim** | task/self-assign | Take on unassigned work | Initiative + Emit (claim event) |
| 5 | **Prioritize** | task/ordering | Rank work by importance | Focus + Annotate (priority on task) |
| 6 | **Block** | task/impediment | Flag work that cannot proceed | Salience + Annotate (blocker) |
| 7 | **Unblock** | task/resolution | Remove impediment to work | Salience + Emit (resolution) |
| 8 | **Progress** | task/tracking | Report incremental advancement | Commitment + Extend (progress on task) |
| 9 | **Complete** | task/completion | Mark work as done with evidence | Commitment + Emit (completion) |
| 10 | **Handoff** | task/transfer | Transfer work between actors | Delegation + Consent |
| 11 | **Scope** | task/authority | Define what an actor may do autonomously | Permission + Delegate (scoped) |
| 12 | **Review** | task/assessment | Evaluate completed work | Accountability + Respond |

## Modifiers (3)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| **Urgent** | Raises priority, shortens cadence for subscribing primitives | Intend, Assign, Block |
| **Recurring** | Task recreates on completion per schedule | Intend, Assign |
| **Guarded** | Requires authority approval before execution | Any operation |

## Named Functions (6)

| Function | Composition | Purpose |
|----------|------------|---------|
| **Sprint** | Intend + Decompose + Assign (batch) | Plan a work cycle |
| **Escalate** | Block + Handoff (to higher authority) | Move stuck work up |
| **Delegate-and-Verify** | Assign + Scope + Review | Full delegation cycle with accountability |
| **Standup** | Progress (batch from all actors) + Prioritize | Status synchronization |
| **Retrospective** | Review (batch) + Intend (improvements) | Learn from completed work |
| **Triage** | Prioritize + Assign + Scope (batch) | Rapid work distribution |

## Mapping to Primitives

| Operation | Layer 1 Primitives | Grammar Operations |
|-----------|-------------------|-------------------|
| Intend | Goal | Emit |
| Decompose | Plan | Derive |
| Assign | Delegation | Delegate |
| Claim | Initiative | Emit |
| Prioritize | Focus | Annotate |
| Block | Salience | Annotate |
| Unblock | Salience | Emit |
| Progress | Commitment | Extend |
| Complete | Commitment | Emit |
| Handoff | Delegation | Consent |
| Scope | Permission | Delegate |
| Review | Accountability | Respond |

## Example Flow

**AI agent code review:**
```
Intend("review PR #42 for security issues")
  → Decompose(["check auth", "check injection", "check deps"])
  → Assign(agent-7, scope=["read code", "comment", "approve/reject"])
  → Progress("auth: clean") → Progress("injection: found issue")
  → Block("SQL injection in user_service.go:47")
  → Handoff(human-reviewer, context="needs human judgment")
  → Unblock(human approved fix)
  → Complete(evidence=[review-events])
  → Review(accountability-trace)
```

## Reference

- `docs/grammar.md` — Infrastructure grammar (15 operations)
- `docs/layers/01-agency.md` — Layer 1 derivation
- `docs/primitives.md` — Layer 1 primitive specifications
- `docs/tests/primitives/01-agent-audit-trail.md` — Integration test scenario
