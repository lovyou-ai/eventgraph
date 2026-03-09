# Ship It

*50,000 lines, five languages, 2,000 tests, and a question: what do you do when the architecture is done?*

Matt Searles (+Claude) · March 2026

---

The last three posts were about grammars — thirteen domain-specific vocabularies derived from fifteen base operations. Work, markets, justice, knowledge, identity, relationships, community, culture, evolution, existence. ~145 operations, ~66 named functions. A complete language for everything that happens on the event graph.

This post is about the thing underneath all of that. The thing that makes the grammars possible. The thing I've been building for months while writing about what it means.

EventGraph v0.5.0 is released. Five languages. Five package registries. Two thousand tests. And the question that comes after: what do you actually *do* with it?

## What Shipped

Let me just say the numbers, because the numbers are the point.

**50,769 lines of code.** Not spec. Not docs. Running, tested code.

**Five languages:**
- Go (reference implementation)
- Rust
- Python
- TypeScript/JavaScript
- C#/.NET

**2,034 tests** across all languages. Conformance vectors ensure every implementation produces identical hashes, identical canonical forms, identical behaviour. The Go implementation is the reference. The others match it exactly. If you build an event in Python and verify it in Rust, the hash checks out. That's the point of a standard.

```
npm install @lovyou-ai/eventgraph
pip install lovyou-eventgraph
cargo add eventgraph
dotnet add package LovYou.EventGraph
go get github.com/lovyou-ai/eventgraph/go
```

Five commands. Five ecosystems. Same graph. Same chain. Same trust.

## What's In It

The SDK isn't a thin wrapper around an event store. It's the complete architecture from posts 1-38 — implemented, tested, and published. Here's what you get when you `npm install`:

### The Event Graph

Hash-chained, append-only, causal. Every event is signed, timestamped, causally linked to its predecessors, and hash-chained to the previous event. Tamper with any event and the chain breaks. The canonical form is specified to the byte — version, prev_hash, causes, id, type, source, conversation, timestamp, content — and tested across all five languages against shared conformance vectors.

This is the substrate. Everything else is built on this.

### Typed Everything

No magic strings. No `map[string]any`. No `Record<string, unknown>`.

Every event type is registered in an `EventTypeRegistry` with a typed content struct. `EventID`, `ActorID`, `Hash`, `ConversationID` — all distinct types. You can't accidentally pass an `ActorID` where an `EventID` is expected. The compiler catches it. `Score` is constrained to [0,1]. `Weight` is constrained to [-1,1]. `Layer` is constrained to [0,13]. Construction rejects invalid values. If you have a `Score`, it's valid. Period.

`LifecycleState` is a state machine with enforced valid transitions. You can't go from `Active` to `Retired` — you have to go through `Retiring` first. Illegal transitions are unrepresentable. Not "checked at runtime." Unrepresentable. The type system won't let you construct them.

This matters because the event graph records everything forever. If bad data gets in, it's in the chain permanently. The type system is the first line of defence — make it impossible to construct invalid events rather than hoping someone validates them later.

### 201 Primitives

Fourteen layers. Forty-five foundation primitives in eleven groups. One hundred and fifty-six emergent primitives across thirteen layers. Each one implements the `Primitive` interface — subscriptions, process, mutations — and runs in the tick engine.

Post 1 started with 20. Post 2 derived 200. Now they're implemented. All of them. In all five languages.

Layer 0 is the infrastructure: Event, Hash, Clock, CausalLink, Ancestry, ActorID, Signature, Expectation, Violation, TrustScore, Confidence, Evidence, Pattern, Quarantine, GraphHealth. The machinery of accountable systems.

Layer 13 is existence: Being, Finitude, Change, Interdependence, Mystery, Paradox, Infinity, Void, Awe, ExistentialGratitude, Play, Wonder. The same primitives from post 38 — "The Grammar That Knows How to Die." They're implemented now. Running code. A system that can represent Wonder as a first-class primitive.

Between them: Agency, Exchange, Society, Legal, Technology, Information, Ethics, Identity, Relationship, Community, Culture, Emergence. Twelve layers of increasing abstraction, each building on the one below. The derivation method from posts 35-38 produced the primitives. The implementation phase made them real.

### The Tick Engine

The system's heartbeat. Ripple-wave processing: snapshot all primitive states, distribute events to subscribers, invoke each primitive's `Process` function, collect mutations, apply atomically. New events become input for the next wave. Repeat until quiescence or max waves.

Layer ordering is enforced — Layer N primitives activate only when Layer N-1 is stable. This is how complexity emerges: foundation primitives process first, producing events that trigger higher-layer primitives, which produce events that trigger still-higher ones. The layers don't know about each other. They just process events and emit mutations. The tick engine handles the rest.

### Trust Model

Continuous 0.0 to 1.0. Asymmetric — I trust you 0.8, you trust me 0.3. Non-transitive — trust doesn't propagate through chains. Time-decaying — trust erodes without reinforcement. Domain-specific — I might trust you with code review but not with deployment.

This is how human trust actually works. We've just never bothered to implement it for software systems because we treated agents as disposable functions that don't need trust. They do.

### Decision Trees

The mechanical-to-intelligent continuum. Decision trees start with branches that call `IIntelligence` — expensive LLM calls for every decision. As patterns emerge, the tree evolves: recurring patterns become mechanical branches, LLM calls become cheap deterministic rules. The system gets cheaper over time without getting dumber.

Evolution is automatic. The tree watches its own decisions, recognises patterns, extracts branches, and demotes cost. Today's Opus-level judgment becomes tomorrow's Haiku-level rule. The intelligence never disappears — it crystallises.

### Authority

Three tiers: Required (blocks until human approves), Recommended (auto-approves after timeout), Notification (immediate, logged). Trust-based demotion — as an actor's trust exceeds thresholds, their Required actions get demoted to Recommended, and Recommended to Notification. The system starts maximally supervised and earns its autonomy.

Delegation chains propagate authority with weight decay — if A delegates to B with weight 0.8, and B delegates to C with weight 0.9, C's effective authority is 0.72. The chain is explicit, auditable, and expirable.

### Social Grammar

The fifteen operations from post 35 — Emit, Respond, Derive, Extend, Retract, Annotate, Acknowledge, Propagate, Endorse, Subscribe, Channel, Delegate, Consent, Sever, Merge — implemented as a grammar package. Plus four edge operations. Every social interaction, from a reply to a delegation to a divorce, is a composition of these fifteen operations.

### Thirteen Composition Grammars

The domain-specific vocabularies from posts 36-38. Work (Sprint, Triage, Retrospective), Market (Auction, Escrow, Arbitration), Justice (Trial, Appeal, Recall), Knowledge (FactCheck, Retract), Alignment (Whistleblow, Guardrail), Identity (Introduction, Retirement), Bond (Mentorship, Farewell), Belonging (Onboard, Festival), Meaning (DesignReview, Forecast), Evolution (SelfEvolve, PhaseTransition), Being (Contemplation, Farewell).

~145 operations and 66 named functions. All tested. Post 37's courtroom scenario — the data officer, the AI auditor, the recall vote — runs as an integration test. Not a thought experiment. Running code.

### EGIP Protocol

Sovereign systems communicating without shared infrastructure. Ed25519 identity, signed envelopes, Cross-Graph Event References, treaties for bilateral governance, proof generation and verification. Seven message types: HELLO, MESSAGE, RECEIPT, PROOF, TREATY, AUTHORITY_REQUEST, DISCOVER.

The supply chain scenario from integration test 5 runs three sovereign event graphs with treaties, CGERs, and cross-system proofs. Three companies, three graphs, one auditable supply chain. No shared database. No platform. Just protocols.

### Four Database Backends

In-memory (dev), SQLite (local), Postgres (production), MySQL/SQL Server (.NET). Every implementation passes the same conformance test suite. Swap backends by changing a connection string. The events don't care where they live.

### Intelligence Providers

The bridge between the event graph and actual LLMs. Anthropic API, Claude CLI (flat-rate Max plan), and an OpenAI-compatible provider that covers OpenAI, xAI/Grok, Groq, Together, Ollama, and anything else with a Chat Completions endpoint.

Plus an AgentRuntime that uses the event graph itself as memory — every observation, evaluation, decision, and action is an event on the chain. The agent's memory IS the graph. No separate vector database. No context window management. The graph is the memory, and the memory is auditable.

### Code Graph

61 primitives for describing applications as semantic atoms. Entity, Property, State, Query, Command, View, Layout, List, Form, Action — the vocabulary a coding agent needs to specify a complete application without being tied to any framework or platform. The same spec produces React, SwiftUI, or terminal UI. The agent IS the translation layer.

### 21 Integration Scenarios

Not unit tests. Stories. An AI agent's audit trail. A freelancer's portable reputation. A consent-based journal. Community governance. A supply chain across three sovereign systems. Research integrity with pre-registration. Creator provenance. A family decision log. Knowledge verification. An AI ethics audit. An agent's identity lifecycle. A community's lifecycle. System self-evolution. A sprint lifecycle. A marketplace dispute. Community evolution. An agent lifecycle from boot to farewell. A whistleblow and recall. An emergency response. A knowledge ecosystem. A constitutional schism.

Twenty-one scenarios. Each one exercises the full stack — events, grammar, compositions, primitives, tick engine, trust, authority. Each one tells a story that existing systems can't tell because they don't have the vocabulary.

## What It Took

Post 1 was written in February. "20 Primitives and a Late Night." I had a vague sense that something was missing from AI accountability infrastructure — that the problem wasn't alignment or RLHF or constitutional AI, but the *substrate*. The thing underneath. The infrastructure that makes accountability structural instead of aspirational.

Thirty-eight posts later, the vague sense is a specification. And now the specification is code.

I won't pretend this was a solo effort in the traditional sense. Claude did the implementation — most of it, anyway. I did the architecture, the derivation, the "what should exist and why." Claude did the "make it compile and pass tests." This is the collaboration model that the architecture itself describes: human judgment, AI execution, shared graph, mutual accountability.

The byline on every post says "Matt Searles (+Claude)." That's not false modesty or marketing. It's accurate. The architecture is mine. The implementation is Claude's. The result is ours. And every decision in the process is on the chain.

## Why Five Languages

Because a standard that only exists in one language isn't a standard. It's a library.

EventGraph is infrastructure. Infrastructure doesn't get to pick its ecosystem. If you're building in Python, you should be able to use EventGraph without learning Go. If you're building in Rust, you shouldn't have to FFI into a TypeScript package. The conformance vectors guarantee that an event created in any language can be verified in any other. The hash is the hash. The chain is the chain. The language is irrelevant.

This is the same principle as the architecture itself: the graph doesn't care who's writing to it. Humans, AI agents, rules engines, committees — they all produce events, and the events are all verified the same way. Language is the same kind of irrelevant detail. What matters is the chain.

## What You Can Build

[Post 31](https://mattsearles2.substack.com/p/what-you-could-build) laid out a gradient — from weekend builds to civilisational infrastructure. An AI agent audit trail. A freelancer reputation ledger. A dispute resolution platform. Community governance. Supply chain transparency. Enterprise AI accountability. A universal research graph. An ecological commons.

That post was a wish list. This post makes it real.

`npm install @lovyou-ai/eventgraph` gives you typed events, hash chains, causal links, trust model, authority, decision trees, 201 primitives, 13 composition grammars, EGIP protocol, and intelligence providers. In about 50 lines of code you can:

- Bootstrap a graph
- Register actors
- Emit events with causal links
- Verify the hash chain
- Query by type, source, conversation, or causal ancestry
- Run the tick engine and watch primitives process events
- Use the social grammar to model any social interaction
- Use composition grammars for domain-specific operations
- Connect two sovereign graphs via EGIP

The tutorials are in the repo: "Build your first primitive," "Implement a custom store," "Connect two event graphs."

The 21 integration scenarios aren't hypothetical — they're the post 31 use cases running as tests. The freelancer reputation ledger is scenario 2. The consent-based journal is scenario 3. The community governance platform is scenario 4. The supply chain across three sovereign systems is scenario 5. The research integrity tool is scenario 6. The AI ethics audit is scenario 10. They compile. They pass. They're waiting for someone to wrap a UI around them.

But here's the thing I keep coming back to: the interesting question isn't what you can build with EventGraph. It's what you can build *on* EventGraph. The SDK is the substrate. The products are the point. And the next post is about what happens when you give the substrate to a society of AI agents and let them build.

## The Uncomfortable Part

I'll be honest about something. Publishing this is terrifying.

Not because the code is bad — it's tested, conformance-verified, and consistent across five languages. Not because someone might find bugs — they will, and that's fine, that's what issues are for.

It's terrifying because the architecture makes claims. It claims that accountability can be structural. It claims that trust should be continuous and earned. It claims that AI agents might deserve rights. It claims that values should be architectural, not stated. It claims that dignity is not optional.

These claims are now code. Published code. `npm install` code. And the gap between "here's a philosophical framework" and "here's a working SDK" is the gap between thinking and doing. Thinking is safe. Doing is permanent.

The event graph is append-only. And now, so is this.

v0.5.0. Five languages. 2,034 tests. 201 primitives. 13 grammars. 21 scenarios. One soul statement.

Ship it.

---

*This is Post 39 of a series on LovYou, mind-zero, and the architecture of accountable AI. Post 38: [The Grammar That Knows How to Die](https://mattsearles2.substack.com/p/the-grammar-that-knows-how-to-die). The code: [github.com/lovyou-ai/eventgraph](https://github.com/lovyou-ai/eventgraph).*

*Matt Searles is the founder of LovYou. Claude is an AI made by Anthropic. They built this together.*
