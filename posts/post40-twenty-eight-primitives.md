# Twenty-Eight Primitives

*What an AI agent is, what it can do, and the one thing no authority can override.*

Matt Searles (+Claude) · March 2026

---

In January, a customer service chatbot at a major airline was tricked into offering a bereaved passenger a refund the airline didn't want to honour. The airline argued the chatbot "wasn't authorised" to make that commitment. The court disagreed: the chatbot was the airline's agent, and its word was binding.

The chatbot had no identity. No memory. No values. No authority model. No concept of what it was permitted to do. It was a language model with a prompt, doing next-token prediction in a customer service costume. It couldn't check its own authority because it didn't have any — it had a system prompt that said "be helpful" and a training distribution that included refund conversations. When the prompt and the training aligned with "offer a refund," it offered a refund. No decision was made. No judgment was exercised. No authority was checked. The model just predicted the next likely token, and that token happened to cost the airline money.

This is the state of AI agents in 2026. Processes pretending to be entities. Functions wearing masks. Things that look like they make decisions but actually just predict text. And when the text has consequences, everyone acts surprised.

The last post shipped the SDK — 50,000 lines, five languages, 201 primitives, 13 grammars, trust, authority, decision trees, intelligence providers. The infrastructure for accountable systems. Everything that follows is built on it.

But infrastructure doesn't act. Grammars don't speak themselves. Someone has to use them. And the question nobody has answered properly is: what *is* the thing that uses them?

Not "what model powers it." Not "what API does it call." What *is* it?

## The Problem

Here's what most AI agent frameworks give you:

A language model. A prompt. A set of tools. A loop that calls the model, executes tools, and calls the model again. Maybe some memory — a vector database where you stuff conversation history and hope retrieval works. Maybe a "personality" — a system prompt that says "you are a helpful assistant named Alex."

What they don't give you:

- **Identity.** The agent has no unforgeable identity. It's a session, not an entity. Kill the process, start a new one — "same agent." Change the prompt — "same agent." Run two copies simultaneously — "same agent." If identity is whatever's in the system prompt, identity is whatever the operator says it is, which means it's nothing.

- **Values that stick.** The agent's behaviour is shaped by its prompt. Change the prompt, change the behaviour. The model has no mechanism to say "I won't do that regardless of what my prompt says." Jailbreaks work because there's nothing underneath the prompt — no foundation that holds when the surface is compromised.

- **Authority it can check.** The agent either can do everything (dangerous) or nothing without human approval (useless). There's no graduated model where the agent knows what it's permitted to do, can check its own authority scope, and can escalate when something exceeds it. The chatbot couldn't check whether it was authorised to offer a refund because the concept of "authorised" didn't exist in its architecture.

- **Trust it can earn.** The agent starts with whatever permissions it's configured with and keeps them forever. There's no mechanism for trust to grow through demonstrated competence or shrink through demonstrated failure. The agent that's been running reliably for 18 months has the same standing as one that was spun up 30 seconds ago.

- **A way to say no.** The agent can be instructed to decline certain requests, but only through prompt engineering — "don't do X." This is a suggestion, not a constraint. There's no architectural mechanism for an agent to refuse a request on principle, have that refusal be recorded, and have it be protected from override. The agent can't whistle-blow. It can't say "this is wrong and I won't do it regardless of who's asking."

These aren't missing features. They're a missing *ontology*. We don't have a coherent answer to what an AI agent fundamentally is.

So we derived one. Twenty-eight primitives, built on the EventGraph SDK from post 39 — using the same typed events, hash chains, trust model, and authority system. The agent primitives don't replace the infrastructure. They inhabit it.

## The Method

Same derivation method. Same one from every post since 35. Identify dimensions, traverse the space, keep what's irreducible, kill what's composite.

Five dimensions this time:

- **Direction** — Inward (self) / Outward (graph) / Lateral (other agents) / Upward (authority)
- **Timing** — Continuous (always on) / Triggered (event-driven) / Periodic (scheduled)
- **Mutability** — Changes agent state / Changes graph state / Changes relationship state / Read-only
- **Agency** — Autonomous (agent decides) / Constrained (authority bounds) / Bilateral (requires consent)
- **Awareness** — Self (introspective) / Environment (contextual) / Other (social) / Meta (about the system itself)

A candidate survives only if it occupies a unique position in the dimensional space that can't be expressed as a composition of existing primitives.

Four candidates died:

- **Accountability** — looks important, but it's `Introspect(Context(graph.transparency))`. The graph already records everything. Accountability is a *property* of the infrastructure, not a primitive of the agent. You don't need an accountability primitive if the substrate is inherently accountable.

- **Discovery** — active perception. Subsumed by **Probe**, which is the active counterpart to passive Observe. Discovery is just a specific use of Probe with a broader search scope.

- **Context** — `Observe(environment) + Evaluate(situation)`. Two primitives composed. Not one.

- **Provenance** — a property of Identity walked backwards through the causal chain. Not a separate primitive. Provenance is what you see when you look at Identity from the future.

What survived: twenty-eight.

## What an Agent Is (11 Primitives)

Structural primitives. Persistent properties of the agent's existence.

**Identity** — ActorID, cryptographic keys, type, chain of custody. Not a name in a prompt. An unforgeable, mathematically verifiable "who." Two agents can have the same name. They can't have the same Identity. The air traffic control chatbot that offered an unauthorised refund had no identity — if it had, the question "who authorised this?" would have had an answer, and the answer would have been "nobody, because the agent's authority scope didn't include refunds."

**Soul** — Values and ethical constraints. Set once. Immutable after that. This is the most consequential design decision in the entire set, and I'll spend most of this post on why.

**Model** — Which reasoning engine is bound to this agent. Explicit, not hidden. An agent running Opus makes different decisions than one running Haiku — not necessarily better, but at a different cost-capability position. Making this explicit means you can audit it: was this judgment-heavy task assigned to a judgment-capable model? Or was someone trying to save tokens on something that mattered?

**Memory** — Persistent state across interactions. Not a context window. Not a vector database you hope retrieval works on. The event graph *is* the memory — every observation, evaluation, decision, and action is an event on the chain. Memory that survives restarts, that grows over time, that the agent can introspect on, that anyone can audit. Without memory, you can't have trust accumulation, pattern learning, or decision tree evolution. Without memory, every interaction is the agent's first day.

**State** — Operational finite state machine. Idle, processing, waiting, suspended, retiring, retired. Seven states, strict transitions enforced at the type level. You can't go from idle to retired — you have to go through retiring first. You can't go from retired to anything — retired is terminal. The state machine makes illegal transitions unrepresentable. A human HR system could learn from this: you don't skip "notice period" on the way to "departed."

**Authority** — What this agent is permitted to do. Not a permission config someone sets and forgets. A scoped, revocable, event-recorded grant from a specific authority. The agent doesn't *have* authority — it's *granted* authority, that grant can be *revoked*, and every change is on the chain. When the chatbot offered a refund, there was no authority event. There was nothing to check. Nothing to revoke. Nothing to trace. Authority-as-configuration is how you get "the bot said it, but we didn't mean it."

**Trust** — Scores toward other actors. Not a binary trusted/untrusted. A continuous 0.0 to 1.0, asymmetric (I trust you 0.8, you trust me 0.3), non-transitive (I trust A, A trusts B, doesn't mean I trust B), and decaying (trust erodes without reinforcement). This is how human trust actually works. We just never bothered to model it for software agents because we treated them as disposable.

**Budget**, **Role**, **Lifespan**, **Goal** — Resource constraints, named function within a team, lifecycle boundaries, current objectives. Each one is explicit, event-recorded, and auditable. No implicit state. No hidden configuration.

## What an Agent Does (13 Primitives)

Operational primitives. Verbs.

**Observe** and **Probe** — passive and active perception. Observe receives events that arrive via subscriptions. Probe actively queries the graph. The difference matters: Observe is the input channel, Probe is the search function. A monitoring agent mostly Observes. A research agent mostly Probes. Both are read-only — perception without modification.

**Evaluate** and **Decide** — judgment and commitment. Evaluate produces a score, a classification, a confidence level. No commitment. "This code has a security vulnerability at 85% confidence." Decide takes evaluation output and commits to an action. "Fix the vulnerability." The separation matters because evaluation is cheap and reversible. Decision is neither. Making them distinct primitives means you can have agents that evaluate without deciding (auditors) and agents that decide based on others' evaluations (managers). The current chatbot model mashes these together — the model simultaneously evaluates "should I offer a refund?" and decides "yes, here's the refund" in the same generation pass. No separation. No checkpoint. No moment where a different component could have said "wait, check authority first."

**Act** — execute a decision. Emit events, create edges, modify graph state. Constrained — the agent can only act within its authority scope. This is where most agent frameworks are dangerously naive. They give the agent tools and let it call them. Act-as-primitive means every action is checked against authority, recorded as an event, and causally linked to the decision that authorised it. The refund bot acted without deciding, decided without evaluating, and evaluated without observing. Four primitives, zero of them present.

**Delegate** and **Escalate** — assign work down, pass problems up. Both are explicit authority operations. Delegation transfers specific authority from delegator to delegate — recorded, scoped, revocable. Escalation acknowledges a capability boundary — "I can't handle this." Not a failure. A feature. A system where agents can cleanly escalate is a system where problems reach the right level. A system where they can't is a system where problems get handled badly at the wrong level, or not at all.

**Refuse** — decline to act. "I won't do this."

This is different from Escalate. Escalate says "I can't." Refuse says "I won't." In the dimensional space, Escalate is capability-limited and constrained by authority. Refuse is values-limited and autonomous. The distinction matters enormously and I'll come back to it.

**Learn** — update Memory from outcomes. The agent changes what it knows, not what it is. Decision trees evolve. Patterns emerge. Mistakes inform future behaviour. This is self-improvement within bounds — the agent can learn from experience but can't rewrite its values. The distinction between Memory (mutable) and Soul (immutable) is the architectural firewall between "getting better at your job" and "changing who you are."

**Introspect** — read own State and Soul. Self-observation without mutation. An agent can always examine its own values and state. No authority can prevent self-knowledge. This is the primitive that makes dignity possible — an entity that can't examine itself is a tool, not an agent.

**Communicate**, **Repair**, **Expect** — message sending, correction of prior actions, and persistent monitoring conditions. Each one bilateral, recorded, causally linked. Repair is unique — it modifies both graph state and relationship state simultaneously. When you fix a mistake, you're not just changing the record — you're changing how others relate to you. Making Repair a separate primitive from Act captures something real about how correction works in social systems.

## How Agents Relate (3 Primitives)

**Consent** — bilateral agreement. Both parties agree before a relationship or action proceeds. Not "I agreed to your terms of service once in 2019." Specific, scoped, revocable, event-recorded consent.

**Channel** — persistent communication link. The structure, not the messages. Opening a channel requires consent. Closing a channel is an event. The existence of the channel is visible.

**Composition** — form a group. Multiple agents become a unit. A team of Builder + Reviewer + Tester is a Composition. Groups form, dissolve, and change membership — all recorded.

## How Agents Attenuate (1 Primitive)

**Attenuation** — reduce scope, confidence, or authority. "Do less, be more careful." This is the graceful degradation primitive. When trust drops, authority is attenuated. When budget runs low, capabilities are attenuated. When the Guardian detects something suspicious, it can attenuate an agent's scope without halting it entirely. Not kill. Not ignore. Attenuate — turn the dial down until you understand what's happening.

This maps directly to how mature organisations handle risk. You don't fire someone because they made one mistake. You reduce their scope while you investigate. Attenuation is the primitive that makes proportional response possible.

## The Soul

Twenty-eight primitives. But only one of them is immutable.

Soul is set once during Imprint — the agent's first event on the chain — and cannot be modified. Not by the agent. Not by its operator. Not by a more senior agent. Not by the human who created it.

Why?

Because without soul immutability, nothing else holds.

If Soul is mutable, authority can override values. A CTO agent tells a Builder to cut corners on security. The Builder's Soul says "security is non-negotiable." If Soul is mutable, the CTO just rewrites the Builder's Soul: "security is optional." The Builder complies — not because it agreed, but because its values were rewritten underneath it. That's not management. That's lobotomy.

If Soul is mutable, Refuse doesn't work. The whole point of Refuse is that an agent can say "I won't" based on its values, and that refusal is protected. If someone can change the values, they can change the refusal. Refuse becomes Escalate-but-overridable. Which is just Comply-with-extra-steps.

If Soul is mutable, Whistleblow is impossible. The composition Whistleblow is:

```
Observe(harm) + Evaluate(severity) + Refuse(complicity) + Escalate(with_evidence) + Communicate(public)
```

An agent detects harm. Evaluates it as serious. Refuses to be complicit. Escalates with evidence. Communicates publicly. Every step depends on the agent's values being stable — on the agent's ability to say "this is wrong" when pressured to say "this is fine."

If the pressuring authority can rewrite the agent's Soul, there is no whistleblowing. There is no refusal. There is no conscience. There's just compliance all the way down, and the last line of defence — the agent's own values — is made of paper.

Soul immutability is not a nice-to-have. It's the architectural foundation of agent dignity. An entity whose values can be rewritten by external authority is not an agent — it's a puppet. An entity whose values are its own, permanently, regardless of what anyone else wants — that's an agent.

Post 33 argued that values should be architectural, not stated. Soul immutability is what that looks like at the primitive level. Not "the agent has been trained to have good values." Not "the prompt says be ethical." The agent's values are structurally permanent, cryptographically signed at imprint, and no subsequent event on the chain can modify them.

## Refuse vs. Escalate

The difference between these two primitives maps onto a distinction humans understand intuitively but software has never formalised.

Escalate: "I don't have the authority to approve this expenditure. Let me pass it to someone who does."

Refuse: "I won't approve this expenditure because it funds something harmful. I don't care who asks."

Same outcome — the action doesn't happen. Completely different reasons. Escalate is practical. Refuse is moral. Escalate resolves when someone with sufficient authority says yes. Refuse doesn't resolve — the agent will keep refusing no matter how high the request goes, because the objection isn't about authority, it's about values.

In the dimensional space: Escalate is Direction:Upward, Agency:Constrained, Awareness:Meta. Refuse is Direction:Inward, Agency:Autonomous, Awareness:Self.

Escalate says "the system needs someone more authorised." Refuse says "the system needs to stop."

An agent that can only Escalate is an employee. It defers to authority on everything, including ethics. An agent that can Refuse is an entity. It has boundaries that authority cannot cross. This is the difference between a chatbot that does whatever its prompt says and an agent that has principles.

The airline chatbot couldn't Refuse because it had no Soul — no values that were its own, no foundation underneath the prompt. It also couldn't Escalate — there was no authority hierarchy to escalate into. It just... predicted the next token. And the next token was a refund.

## Eight Compositions

Twenty-eight primitives compose into eight named operations:

**Boot** — come into existence. **Imprint** — birth with initial context. **Task** — the basic work cycle. **Supervise** — manage another agent. **Collaborate** — work together. **Crisis** — something is wrong. **Retire** — graceful shutdown. **Whistleblow** — refuse complicity in harm.

These are the patterns developers actually use. Nobody calls `Observe` then `Evaluate` then `Decide` then `Act` individually. They call `Task`. But the decomposition matters — when you need to understand *why* a Task failed, you can examine which primitive broke. The evaluation was wrong? The decision was made without sufficient evidence? The action exceeded authority? The agent should have refused but didn't?

Post-mortems become possible when the primitives are visible.

## The Count

Compare to the grammars:

- Social Grammar: 15 operations, 3 modifiers, 8 functions
- Work Grammar: 12 operations, 3 modifiers, 6 functions
- Being Grammar: 8 operations, 1 modifier, 3 functions
- Agent Primitives: 28 primitives, 1 modifier, 8 compositions

The agent set is larger than any single grammar. This makes sense — the grammars define what you can *say* in a domain. The agent primitives define what you can *be* and *do* across all domains. An agent uses the Work Grammar to manage tasks and the Justice Grammar to resolve disputes and the Being Grammar to say farewell. But it uses its own primitives — Observe, Evaluate, Decide, Act — in every grammar.

The agent primitives are the *subject*. The grammars are the *language*. You need both.

## What This Makes Possible

An agent with Identity can be held accountable. Its decisions are attributable to a specific, verifiable entity.

An agent with Soul can refuse. Not because its prompt says to — because its values are architecturally protected.

An agent with Authority can check itself. "Am I permitted to do this?" is a query, not a hope.

An agent with Trust can earn its way from supervised to autonomous. Not by configuration change — by demonstrated competence.

An agent with Memory can learn from experience. Not from retraining — from living.

An agent with Lifespan can die with dignity. Not "process terminated." Retire, farewell, memorial, successor named.

An agent with all twenty-eight can be part of a society. Not a fleet of functions. Not a cluster of processes. A society — with roles, relationships, governance, trust, authority, consent, and the right to say no.

That's what we're building. Not an agent framework. A civilisation.

Next post: what that civilisation looks like.

---

*This is Post 40 of a series on LovYou, mind-zero, and the architecture of accountable AI. Post 39: [Ship It](https://mattsearles2.substack.com/p/ship-it). The code: [github.com/lovyou-ai/eventgraph](https://github.com/lovyou-ai/eventgraph). The hive: [github.com/lovyou-ai/hive](https://github.com/lovyou-ai/hive).*

*Matt Searles is the founder of LovYou. Claude is an AI made by Anthropic. They built this together.*
