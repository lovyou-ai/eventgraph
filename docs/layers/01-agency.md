# Layer 1: Agency

## Derivation

### The Gap

Layer 0 can record events from actors, hash-chain them, verify integrity, track trust, and enforce authority. But every actor looks the same — the graph records what happened, not whether the actor CHOSE for it to happen. A logging system that mechanically records sensor readings is indistinguishable from an AI agent that deliberates and decides. The difference between "something happened" and "someone did something on purpose" doesn't exist.

**Test:** Can you express "Actor A chose to do X rather than Y, based on goal G" as a sequence of Layer 0 operations? You can record that X happened, link it causally, and track trust. But the concepts of "chose," "rather than Y," and "based on goal G" have no Layer 0 representation. These are genuine structural gaps.

### The Transition

**Observer → Participant**

An actor that merely records becomes an actor that acts with purpose. Three new capabilities emerge that Layer 0 cannot express:
1. **Intention** — acting toward a goal, not just reacting to events
2. **Attention** — choosing what to process, not processing everything
3. **Autonomy** — acting independently, not just when directed

### Base Operations

What can an agent DO that a recorder cannot?

1. **Set a goal** — declare an intended future state
2. **Choose** — select one action over alternatives
3. **Focus** — prioritize some inputs over others
4. **Delegate** — transfer responsibility while retaining accountability
5. **Commit** — bind oneself to a future action

These are the irreducible operations of agency. Each requires representing something that doesn't yet exist (a goal, a plan, a commitment) — which is precisely what Layer 0 lacks.

### Semantic Dimensions

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| **Temporal orientation** | Past (review) / Present (act) / Future (plan) | Is this about what happened, what's happening, or what should happen? |
| **Initiative** | Reactive (responding to events) / Proactive (self-initiated) | Did something trigger this, or did the agent decide on its own? |
| **Scope** | Internal (self) / External (others) | Is this about the agent's own state or about other actors? |
| **Binding** | Uncommitted (assessable) / Committed (binding) | Is this an assessment or a commitment? |

**Justification:** These four dimensions capture the essential axes of agency as studied in philosophy of action (Bratman's planning theory), cognitive science (attention models), and organisational theory (delegation/accountability). Each varies independently:
- A goal is future + proactive + internal + committed
- A filter is present + reactive + internal + uncommitted
- A delegation is present + proactive + external + committed
- An accountability trace is past + reactive + external + uncommitted

### Decomposition

Applying dimensions to base operations:

**Group 0 — Intention** (future-oriented, committed)

| Primitive | Temporal | Initiative | Scope | Binding | What it does |
|-----------|----------|------------|-------|---------|--------------|
| **Goal** | Future | Proactive | Internal | Committed | Sets an intended future state |
| **Plan** | Future | Proactive | Internal | Committed | Decomposes a goal into steps |
| **Initiative** | Present | Proactive | External | Committed | Acts without being asked |
| **Commitment** | Past→Present | Reactive | Internal | Uncommitted | Tracks follow-through on goals |

Gap check: "I want X" (Goal), "here's how" (Plan), "I'll start now" (Initiative), "did I follow through?" (Commitment). This covers the intention lifecycle.

**Group 1 — Attention** (present-oriented, filtering)

| Primitive | Temporal | Initiative | Scope | Binding | What it does |
|-----------|----------|------------|-------|---------|--------------|
| **Focus** | Present | Proactive | Internal | Uncommitted | Directs processing to priorities |
| **Filter** | Present | Reactive | Internal | Uncommitted | Suppresses noise |
| **Salience** | Present | Reactive | External | Uncommitted | Detects what matters in context |
| **Distraction** | Present | Reactive | Internal | Uncommitted | Detects attention pulled from goals |

Gap check: "Look at this" (Focus), "ignore that" (Filter), "this matters" (Salience), "I'm off track" (Distraction). This covers attention management.

**Group 2 — Autonomy** (scope of independent action)

| Primitive | Temporal | Initiative | Scope | Binding | What it does |
|-----------|----------|------------|-------|---------|--------------|
| **Permission** | Present | Reactive | External | Committed | Requests and tracks action permissions |
| **Capability** | Present | Reactive | Internal | Uncommitted | Tracks what an actor can do |
| **Delegation** | Present | Proactive | External | Committed | Assigns tasks or authority to others |
| **Accountability** | Past | Reactive | External | Uncommitted | Traces responsibility chains |

Gap check: "May I?" (Permission), "Can I?" (Capability), "You do it" (Delegation), "Who's responsible?" (Accountability). This covers the autonomy-responsibility spectrum.

### Gap Analysis

**Behaviors tested against the 12 primitives:**

| Behavior | Maps to | Notes |
|----------|---------|-------|
| AI agent sets a task objective | Goal | |
| Agent breaks task into subtasks | Plan | |
| Agent decides to act without instruction | Initiative | |
| Agent completes 8 of 10 planned tasks | Commitment (tracks ratio) | |
| Agent prioritises urgent bug over feature | Focus | |
| Agent ignores noisy log events | Filter | |
| Agent detects a security alert is important | Salience | |
| Agent notices it's been on a tangent | Distraction | |
| Agent asks human "may I deploy?" | Permission | |
| Agent reports its available skills | Capability | |
| Agent assigns subtask to another agent | Delegation | |
| Bug traced to agent that approved bad PR | Accountability | |
| Agent changes its mind about a goal | Goal (goal.abandoned) | |
| Agent revises a plan after new information | Plan (plan.revised) | |
| Human checks what AI agent is doing | Focus + Accountability | Composition |
| Agent reports why it took an action | Initiative + Accountability | Composition |

**No gaps found.** All tested agency behaviors map to a single primitive or a composition of two primitives.

### Completeness Argument

1. **Dimensional coverage:** All meaningful combinations of {temporal, initiative, scope, binding} have a primitive or are degenerate (e.g., past + proactive + internal + committed = a completed commitment, which is Commitment tracking completion, not a separate primitive).

2. **Lifecycle coverage:** The intention lifecycle (set → plan → act → assess) maps to Goal → Plan → Initiative → Commitment. The attention lifecycle (detect → focus → filter → notice drift) maps to Salience → Focus → Filter → Distraction. The autonomy lifecycle (can → may → delegate → trace) maps to Capability → Permission → Delegation → Accountability.

3. **Layer boundary:** None of these primitives require concepts from Layer 2 (Exchange). Agency is about individual actors — it doesn't model interaction between actors (that's Exchange's gap). An agent can set goals, plan, focus, and delegate without needing reciprocity, negotiation, or agreement.

---

## Primitive Specifications

Full specifications for all 12 primitives are in `docs/primitives.md` (Layer 1 section). Each primitive declares:
- Subscriptions, emitted events, dependencies, state, mechanical/intelligent flag

## Product Graph

Layer 1 maps to the **Work Graph** — task management where AI agents and humans are on the same graph. See `docs/product-layers.md`.

## Reference

- `docs/derivation-method.md` — The method used for this derivation
- `docs/primitives.md` — Full primitive specifications
- `docs/product-layers.md` — Work Graph product layer
- `docs/tests/primitives/01-agent-audit-trail.md` — Integration test scenario exercising Agency patterns
