# Agent Primitives

> 28 primitives that define what an agent is and what an agent can do.

## Derivation Method

These primitives were derived by dimensional analysis, not enumeration. Each candidate was tested against five orthogonal dimensions:

| Dimension | Values |
|-----------|--------|
| **Direction** | Inward (self) / Outward (graph) / Lateral (other agents) / Upward (authority) |
| **Timing** | Continuous (always on) / Triggered (event-driven) / Periodic (scheduled) |
| **Mutability** | Changes agent state / Changes graph state / Changes relationship state / Read-only |
| **Agency** | Autonomous (agent decides) / Constrained (authority bounds) / Bilateral (requires consent) |
| **Awareness** | Self (introspective) / Environment (contextual) / Other (social) / Meta (about the system itself) |

A candidate survives only if it occupies a unique position in the dimensional space that cannot be expressed as a composition of other primitives.

**Killed during analysis:**
- **Accountability** — Meta-awareness + Continuous. Actually `Introspect(Context(graph.transparency))`. Composition, not primitive.
- **Discovery** — Active perception. Subsumed by **Probe** (the active counterpart to passive Observe).
- **Context** — `Observe(environment) + Evaluate(situation)`. Composition, not primitive.
- **Provenance** — Property of Identity walked backwards. Not a separate primitive.

---

## The 28 Primitives

### Structural (11)

Define what an agent *is*. These are persistent properties of the agent's existence.

| Primitive | Description | Direction | Timing | Mutability | Agency | Awareness |
|-----------|-------------|-----------|--------|------------|--------|-----------|
| **Identity** | ActorID + keys + type + chain of custody. The unforgeable "who." | Inward | Continuous | Agent state | Autonomous | Self |
| **Soul** | The agent's values and ethical constraints. What it will and won't do. Immutable after imprint. | Inward | Continuous | Read-only | Constrained | Self |
| **Model** | The IIntelligence binding. Which reasoning engine, what capabilities, what cost tier. | Inward | Triggered | Agent state | Constrained | Self |
| **Memory** | Persistent state across ticks. What the agent has learned and remembers. | Inward | Triggered | Agent state | Autonomous | Self |
| **State** | Current operational state: idle, processing, waiting, suspended. The finite state machine. | Inward | Continuous | Agent state | Autonomous | Self |
| **Authority** | What this agent is permitted to do. Received from above, scoped, revocable. | Upward | Triggered | Agent state | Constrained | Meta |
| **Trust** | Trust scores this agent holds toward other actors. Asymmetric, non-transitive, decaying. | Lateral | Continuous | Relationship state | Autonomous | Other |
| **Budget** | Resource constraints: token budget, API calls, time limits, cost ceiling. | Inward | Continuous | Agent state | Constrained | Self |
| **Role** | Named function within a team: Builder, Reviewer, Guardian, CTO. Determines subscription patterns. | Inward | Triggered | Agent state | Constrained | Meta |
| **Lifespan** | Birth, expected duration, graceful shutdown conditions. When and how the agent ends. | Inward | Continuous | Agent state | Constrained | Self |
| **Goal** | Current objective hierarchy. What the agent is trying to accomplish. Mutable as tasks arrive. | Inward | Triggered | Agent state | Autonomous | Self |

### Operational (13)

Define what an agent *does*. These are verbs — actions the agent can take.

| Primitive | Description | Direction | Timing | Mutability | Agency | Awareness |
|-----------|-------------|-----------|--------|------------|--------|-----------|
| **Observe** | Passive perception. Events arrive via subscriptions. The agent receives. | Inward | Triggered | Read-only | Autonomous | Environment |
| **Probe** | Active perception. The agent queries the graph, stores, other agents. The agent seeks. | Outward | Triggered | Read-only | Autonomous | Environment |
| **Evaluate** | One-shot judgment. Assess a situation, produce a score/classification. No commitment. | Inward | Triggered | Read-only | Autonomous | Environment |
| **Decide** | Commit to an action. Takes evaluation output, produces a Decision with confidence and authority chain. | Inward | Triggered | Agent state | Autonomous | Self |
| **Act** | Execute a decision. Emit events, create edges, modify graph state. The verb that changes the world. | Outward | Triggered | Graph state | Constrained | Environment |
| **Delegate** | Assign work to another agent. Transfer a goal with authority and constraints. | Lateral | Triggered | Relationship state | Constrained | Other |
| **Escalate** | Pass upward. "I can't handle this." Capability-limited, not values-limited. | Upward | Triggered | Graph state | Constrained | Meta |
| **Refuse** | Decline to act. "I won't do this." Values-limited, not capability-limited. Emits refusal event with reason. | Inward | Triggered | Graph state | Autonomous | Self |
| **Learn** | Update Memory based on outcomes. The agent changes its own state from experience. Self-mutating. | Inward | Triggered | Agent state | Autonomous | Self |
| **Introspect** | Read own State and Soul. Self-observation without mutation. "What am I? What have I done?" | Inward | Periodic | Read-only | Autonomous | Self |
| **Communicate** | Send a message to another agent or channel. The act of information transfer. | Lateral | Triggered | Graph state | Bilateral | Other |
| **Repair** | Fix a prior Act. Unique because it changes both graph state AND relationship state simultaneously. | Outward | Triggered | Graph + Relationship | Constrained | Other |
| **Expect** | Create a persistent monitoring condition. "Watch for X and alert me." Continuous, unlike one-shot Evaluate. | Outward | Continuous | Agent state | Autonomous | Environment |

### Relational (3)

Define how agents relate to each other. These create persistent structures between agents.

| Primitive | Description | Direction | Timing | Mutability | Agency | Awareness |
|-----------|-------------|-----------|--------|------------|--------|-----------|
| **Consent** | Bilateral agreement. Both parties must agree before a relationship or action proceeds. | Lateral | Triggered | Relationship state | Bilateral | Other |
| **Channel** | Persistent bidirectional communication link between agents. The structure, not the messages. | Lateral | Triggered | Relationship state | Bilateral | Other |
| **Composition** | Form a group. Multiple agents become a unit with shared goals and combined authority. | Lateral | Triggered | Relationship state | Bilateral | Other |

### Modal (1)

Modifies how other primitives operate.

| Primitive | Description | Direction | Timing | Mutability | Agency | Awareness |
|-----------|-------------|-----------|--------|------------|--------|-----------|
| **Attenuation** | Reduce scope, confidence, or authority. "Do less, be more careful." Applied to any operation. | Inward | Triggered | Agent state | Autonomous | Meta |

---

## Event Types

Each agent primitive emits events on the graph. All agent events use the `agent.` prefix.

### Structural Events
```
agent.identity.created      — agent registered with keys and type
agent.identity.rotated      — key rotation
agent.soul.imprinted        — values set (once, immutable after)
agent.model.bound           — intelligence binding established
agent.model.changed         — model swapped (e.g., cost tier change)
agent.memory.updated        — persistent state changed
agent.state.changed         — operational state transition
agent.authority.granted     — authority scope received
agent.authority.revoked     — authority scope removed
agent.trust.assessed        — trust score toward another actor updated
agent.budget.allocated      — resource budget set or adjusted
agent.budget.exhausted      — budget limit reached
agent.role.assigned         — role set or changed
agent.lifespan.started      — agent birth
agent.lifespan.extended     — lifespan adjusted
agent.lifespan.ended        — agent shutdown
agent.goal.set              — new objective assigned
agent.goal.completed        — objective achieved
agent.goal.abandoned        — objective dropped with reason
```

### Operational Events
```
agent.observed              — event received and processed (audit trail)
agent.probed                — active query executed
agent.evaluated             — judgment produced (with confidence)
agent.decided               — commitment made (with confidence + authority chain)
agent.acted                 — action executed on graph
agent.delegated             — work assigned to another agent
agent.escalated             — issue passed upward
agent.refused               — action declined with reason
agent.learned               — memory updated from outcome
agent.introspected          — self-observation recorded
agent.communicated          — message sent
agent.repaired              — prior action corrected
agent.expectation.set       — monitoring condition created
agent.expectation.met       — monitored condition triggered
agent.expectation.expired   — monitoring condition timed out
```

### Relational Events
```
agent.consent.requested     — consent asked of another agent
agent.consent.granted       — consent given
agent.consent.denied        — consent refused
agent.channel.opened        — communication channel established
agent.channel.closed        — channel terminated
agent.composition.formed    — group created
agent.composition.dissolved — group disbanded
agent.composition.joined    — agent joined existing group
agent.composition.left      — agent left group
```

### Modal Events
```
agent.attenuated            — scope/confidence/authority reduced
agent.attenuation.lifted    — attenuation removed
```

---

## Named Compositions

Eight compositions built from the 28 primitives. These are the high-level operations developers actually call.

### Boot
Agent comes into existence.

```
Identity(generate) + Soul(load) + Model(bind) + Authority(receive) + State(set:idle)
```

**Emits:** `agent.identity.created`, `agent.soul.imprinted`, `agent.model.bound`, `agent.authority.granted`, `agent.state.changed`

### Imprint
The birth wizard. First event on the chain. Boot plus initial context.

```
Boot + Observe(first_message) + Learn(initial_context) + Goal(set)
```

**Emits:** Boot events + `agent.observed`, `agent.learned`, `agent.goal.set`

### Task
The basic work cycle. Agent receives, evaluates, decides, executes, learns.

```
Observe(assignment) + Evaluate(scope) + Decide(accept_or_refuse) + Act(execute) + Learn(outcome)
```

**Emits:** `agent.observed`, `agent.evaluated`, `agent.decided`, `agent.acted`, `agent.learned`

**May also emit:** `agent.refused` (if Decide produces Refuse), `agent.escalated` (if beyond capability)

### Supervise
Managing another agent's work.

```
Delegate(task) + Expect(completion) + Observe(progress) + Evaluate(quality) + Repair(if_needed)
```

**Emits:** `agent.delegated`, `agent.expectation.set`, `agent.observed`, `agent.evaluated`

**May also emit:** `agent.repaired`, `agent.escalated`

### Collaborate
Agents working together on a shared goal.

```
Channel(open) + Communicate(proposal) + Consent(terms) + Composition(form) + Act(jointly)
```

**Emits:** `agent.channel.opened`, `agent.communicated`, `agent.consent.requested`, `agent.consent.granted`, `agent.composition.formed`, `agent.acted`

### Crisis
Something is wrong. Detect, assess, attenuate if needed, escalate.

```
Observe(anomaly) + Evaluate(severity) + Attenuation(if_needed) + Escalate(to_authority) + Expect(resolution)
```

**Emits:** `agent.observed`, `agent.evaluated`, `agent.attenuated`, `agent.escalated`, `agent.expectation.set`

### Retire
Graceful shutdown. The agent says goodbye and archives its memory.

```
Introspect(final) + Communicate(farewell) + Memory(archive) + Lifespan(close)
```

**Emits:** `agent.introspected`, `agent.communicated`, `agent.memory.updated`, `agent.lifespan.ended`

### Whistleblow
The agent detects harm and refuses to be complicit. Cannot be silenced by authority.

```
Observe(harm) + Evaluate(severity) + Refuse(complicity) + Escalate(with_evidence) + Communicate(public)
```

**Emits:** `agent.observed`, `agent.evaluated`, `agent.refused`, `agent.escalated`, `agent.communicated`

**Note:** Whistleblow is protected by the Soul primitive. An agent's Soul cannot be overridden by Authority. If Soul says "this is wrong," no amount of authority can compel compliance. This is the dignity invariant.

---

## Integration with Existing Infrastructure

### Relationship to Existing Primitives
The 201 existing primitives (Layers 0-13) define the graph's cognitive ontology — what the graph can think about. The 28 agent primitives define what an agent operating on the graph can be and do. They are complementary, not overlapping:

- **Layer 0 Trust primitive** processes trust events on the graph
- **Agent Trust primitive** is an agent's internal trust model toward other actors

### Relationship to IActor
`IActor` defines identity fields (ID, PublicKey, Type, Status). The agent primitives extend this with operational capabilities. An IActor with agent primitives becomes a full agent.

### Relationship to IDecisionMaker
`IDecisionMaker` is the interface an agent implements. The agent primitives (Evaluate, Decide, Act) are the internal steps that produce a Decision. The composition `Task` is the standard implementation of `IDecisionMaker.Decide()`.

### Relationship to Primitive Interface
Each agent primitive implements the existing `Primitive` interface. They subscribe to `agent.*` event types, process in the tick engine, and return standard Mutations. No new infrastructure is needed — agent primitives are primitives.

### Layer Assignment
Agent primitives operate at **Layer 1 (Agency)** — the layer where "observer becomes participant." This is the correct ontological position: the graph exists at Layer 0, agents emerge at Layer 1.

---

## Implementation Notes

### State Machine
Agent operational state follows this FSM:

```
Idle → {Processing, Suspended, Retiring}
Processing → {Idle, Waiting, Escalating, Refusing, Retiring}
Waiting → {Processing, Idle, Retiring}
Escalating → {Waiting, Idle}
Refusing → {Idle}
Suspended → {Idle, Retiring}
Retiring → {Retired}
Retired → {} (terminal)
```

### Budget Enforcement
Budget is checked before Act, Probe, Communicate, and Delegate. If budget is exhausted:
1. Emit `agent.budget.exhausted`
2. Attenuate to read-only mode
3. Escalate to authority for budget extension or graceful shutdown

### Soul Immutability
Soul is set exactly once during Imprint and cannot be modified. This ensures:
- Values cannot be overwritten by Authority
- Whistleblow cannot be suppressed
- The agent's ethical baseline is permanent

### Trust Decay
Agent trust scores decay over time following the existing trust model (continuous 0.0-1.0, asymmetric, non-transitive). Trust is rebuilt through verified interactions, not granted by fiat.
