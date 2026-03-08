# Roadmap

## How to Use This

Pick any unclaimed task. Create a branch. Submit a PR. See `CONTRIBUTING.md` for standards.

Tasks are ordered by dependency — work higher in the list before lower. Tasks within a section can often be parallelised.

**For automated implementers:** See `docs/implementation-order.md` for the strict dependency DAG with compilation-order tasks, acceptance criteria, and the implementer loop.

**Status key:** DONE | IN PROGRESS | NEEDED | BLOCKED (by what)

---

## Phase 1: Foundation

The event graph core — the substrate everything else builds on.

### Event Graph Core — DONE

The reference implementation exists in [mind-zero-five](https://github.com/mattxo/mind-zero-five). These tasks are primarily extraction, cleanup, and making the code package-friendly.

**See `docs/implementation-order.md` for the strict dependency-ordered implementation DAG with acceptance criteria.**

#### Tier 0-1: Foundation Types — DONE
- [x] `go/pkg/types/option.go` — `Option[T]` generic type (Some, None, Unwrap, JSON)
- [x] `go/pkg/types/nonempty.go` — `NonEmpty[T]` generic type (rejects empty)
- [x] `go/pkg/types/page.go` — `Page[T]` pagination, `Cursor`
- [x] `go/pkg/types/errors.go` — All `ValidationError` types
- [x] `go/pkg/types/ids.go` — Value objects: EventID, ActorID, Hash, ConversationID, SystemURI, PublicKey, Signature, etc.
- [x] `go/pkg/types/constrained.go` — Constrained numerics: Score [0,1], Weight [-1,1], Activation [0,1], Layer [0,13], Cadence [1,∞), Tick [0,∞)
- [x] `go/pkg/types/statemachine.go` — LifecycleState and ActorStatus state machines (enforced valid transitions)
- [x] `go/pkg/types/types_test.go` — Construction validation, rejection of invalid values, equality, state transitions, conformance vectors

#### Tier 2-3: Events and Content — DONE
- [x] `go/pkg/event/constants.go` — All enums with `IsValid()` and Visitor interfaces
- [x] `go/pkg/event/content.go` — EventContent interface, all content structs, EventTypeRegistry, EventContentVisitor
- [x] `go/pkg/event/edge.go` — Edge struct, EdgeMetadata interface, EdgeTypeRegistry, EdgeMetadataVisitor, all metadata types
- [x] `go/pkg/event/event.go` — Event struct (immutable), canonical form, hash computation
- [x] `go/pkg/event/decision.go` — Decision, DecisionInput, Receipt, TrustMetrics, AuthorityLink, TrustWeight, Expectation, ViolationRecord
- [x] `go/pkg/event/event_test.go` — Canonical form vectors, hash chain tests, content validation
- [x] `go/pkg/event/conformance_test.go` — Conformance vectors matching `docs/conformance/canonical-vectors.json`

#### Tier 4-5: Store and Actor — DONE
- [x] `go/pkg/store/errors.go` — All `StoreError` types, StoreErrorVisitor
- [x] `go/pkg/store/store.go` — Store interface definition
- [x] `go/pkg/store/memory.go` — InMemoryStore implementation (chain locking, edge indexing, concurrent-safe)
- [x] `go/pkg/store/storetest/suite.go` — Shared conformance test suite (importable by any store implementation)
- [x] `go/pkg/store/store_test.go` — Runs conformance suite + memory-specific tests
- [x] `go/pkg/store/pgstore/pgstore.go` — PostgresStore implementation (pgx/v5, advisory lock serialization, recursive CTE traversal)
- [x] `go/pkg/store/pgstore/pgstore_test.go` — Runs conformance suite (set `EVENTGRAPH_POSTGRES_URL` to enable)
- [x] `go/pkg/actor/actor.go` — IActor, Actor, IActorStore, ActorUpdate, ActorFilter
- [x] `go/pkg/actor/memory.go` — InMemoryActorStore
- [x] `go/pkg/actor/actor_test.go` — Registration, lookup, lifecycle, pagination
- [x] `go/pkg/event/factory.go` — EventFactory, BootstrapFactory

#### Tier 6-7: Bus, Decision, Trust, Authority — DONE
- [x] `go/pkg/bus/bus.go` — IBus, EventBus (non-blocking, overflow handling)
- [x] `go/pkg/bus/bus_test.go` — Concurrency, backpressure, overflow
- [x] `go/pkg/decision/tree.go` — DecisionTree, nodes, conditions, stats
- [x] `go/pkg/decision/evaluate.go` — Tree evaluation, path tracking, Semantic conditions
- [x] `go/pkg/decision/evolve.go` — Pattern recognition, branch extraction, cost demotion
- [x] `go/pkg/decision/intelligence.go` — IIntelligence, IDecisionMaker interfaces
- [x] `go/pkg/decision/decision_test.go` — Comprehensive tree evaluation tests
- [x] `go/pkg/decision/evolve_test.go` — Evolution pattern detection, branch extraction tests
- [x] `go/pkg/trust/model.go` — ITrustModel, DefaultTrustModel (decay, recovery)
- [x] `go/pkg/trust/model_test.go` — Scoring, decay, domain-specific, boundary values
- [x] `go/pkg/authority/authority.go` — IAuthorityChain, DefaultAuthorityChain, AuthorityResult, policies
- [x] `go/pkg/authority/chain.go` — DelegationChain: delegation walk, weight propagation, expiry
- [x] `go/pkg/authority/authority_test.go` — Evaluation, policies, trust-based demotion
- [x] `go/pkg/authority/chain_test.go` — Delegation chains, expiry, best weight selection

#### Tier 8-9: Primitives and Tick Engine — DONE
- [x] `go/pkg/primitive/primitive.go` — Primitive interface, Mutation types, MutationVisitor
- [x] `go/pkg/primitive/registry.go` — PrimitiveRegistry
- [x] `go/pkg/primitive/harness.go` — PrimitiveTestHarness (builder pattern, mutation capture)
- [x] `go/pkg/primitive/primitive_test.go` — Registry, lifecycle, harness tests
- [x] `go/pkg/primitive/harness_test.go` — Harness tests: process, emit, edges, activation, lifecycle, mixed
- [x] `go/pkg/tick/engine.go` — Tick engine, wave processing, quiescence, layer ordering
- [x] `go/pkg/tick/engine_test.go` — Ripple, wave limit, quiescence, layer constraint, concurrency

#### Tier 10: Top-Level API — DONE
- [x] `go/pkg/graph/graph.go` — IGraph (Evaluate, Record, Query, Bootstrap, Start, Close)
- [x] `go/pkg/graph/query.go` — IGraphQuery (Recent, ByType, BySource, ByConversation, Ancestors, Descendants, TrustScore, TrustBetween, Actor, EventCount)
- [x] `go/pkg/graph/graph_test.go` — End-to-end integration tests
- [x] `go/pkg/grammar/grammar.go` — 15 social grammar operations + 4 named functions (Invite, Forgive, Challenge, Propagate)
- [x] `go/pkg/grammar/grammar_test.go` — Grammar operation tests
- [x] `go/cmd/eg/main.go` — CLI for interacting with any store
- [x] `docs/conformance/canonical-vectors.json` — Language-agnostic conformance test vectors

### Primitive Framework — DONE

The architecture for primitives — the 201 agents that form the cognitive layers.

- [x] `go/pkg/primitive/primitive.go` — Primitive interface, Mutation types, Registry
- [x] `go/pkg/primitive/registry.go` — Primitive registry (register, get, by-layer, subscribers-for)
- [x] `go/pkg/primitive/primitive_test.go` — Registry tests
- [x] `go/pkg/primitive/harness.go` — PrimitiveTestHarness
- [x] `go/pkg/primitive/harness_test.go` — Harness tests

### Tick Engine — DONE

The ripple-wave processor — the system's heartbeat.

- [x] `go/pkg/tick/engine.go` — Tick engine, wave processing, quiescence detection
- [x] `go/pkg/tick/engine_test.go` — Ripple tests, wave limit tests, quiescence tests

### Decision Tree Engine — DONE

The mechanical-to-intelligent continuum.

- [x] `go/pkg/decision/tree.go` — Tree structure, internal nodes, leaf nodes, conditions, branches
- [x] `go/pkg/decision/evaluate.go` — Tree evaluation, path tracking
- [x] `go/pkg/decision/evolve.go` — Pattern recognition, branch extraction, cost demotion
- [x] `go/pkg/decision/intelligence.go` — IIntelligence interface, IDecisionMaker interface, DecisionInput/Decision types
- [x] `go/pkg/decision/decision_test.go` — Comprehensive evaluation tests
- [x] `go/pkg/decision/evolve_test.go` — Evolution tests
- [x] `go/pkg/trust/model.go` — ITrustModel interface, default implementation
- [x] `go/pkg/trust/model_test.go` — Trust scoring, decay, domain-specific trust tests
- [x] `go/pkg/authority/authority.go` — DefaultAuthorityChain (flat model)
- [x] `go/pkg/authority/chain.go` — DelegationChain (multi-hop delegation)
- [x] `go/pkg/authority/authority_test.go` — Authority evaluation tests
- [x] `go/pkg/authority/chain_test.go` — Delegation chain tests

---

## Phase 2: Layer 0 Primitives

The 45 foundation primitives in 11 groups. Each primitive needs: implementation, tests, documentation.

### Group 0 — Core — DONE
- [x] Event primitive — validates hash integrity and causal links
- [x] EventStore primitive — tracks chain head and event count
- [x] Clock primitive — tick counting and timestamps
- [x] Hash primitive — SHA-256 chain verification
- [x] Self primitive — system identity and primitive registry tracking

### Group 1 — Causality — DONE
- [x] CausalLink primitive — validates causal edges, tracks valid/invalid links
- [x] Ancestry primitive — traverses causal chains upward via Store
- [x] Descendancy primitive — traverses causal chains downward via Store
- [x] FirstCause primitive — walks to root cause (bootstrap event)

### Group 2 — Identity — DONE
- [x] ActorID primitive — tracks actor registrations
- [x] ActorRegistry primitive — tracks actor lifecycle (active/suspended/memorial)
- [x] Signature primitive — tracks Ed25519 signature presence
- [x] Verify primitive — verifies signature format, tracks verified/failed counts

### Group 3 — Expectations — DONE
- [x] Expectation primitive — tracks pending expectations from authority requests
- [x] Timeout primitive — monitors for expired expectations
- [x] Violation primitive — detects and records unmet expectations
- [x] Severity primitive — classifies violations by severity level

### Group 4 — Trust — DONE
- [x] TrustScore primitive — monitors trust score snapshots
- [x] TrustUpdate primitive — tracks trust changes and decay
- [x] Corroboration primitive — detects multi-source agreement
- [x] Contradiction primitive — detects conflicting trust signals

### Group 5 — Confidence — DONE
- [x] Confidence primitive — tracks decision confidence levels
- [x] Evidence primitive — tracks causal evidence chains
- [x] Revision primitive — tracks content retractions
- [x] Uncertainty primitive — monitors escalations from low confidence

### Group 6 — Instrumentation — DONE
- [x] InstrumentationSpec primitive — defines measurement scope
- [x] CoverageCheck primitive — verifies all event types have subscribers
- [x] Gap primitive — detects time periods with no events
- [x] Blind primitive — detects blind spots with no instrumentation

### Group 7 — Query — DONE
- [x] PathQuery primitive — supports causal path queries
- [x] SubgraphExtract primitive — extracts subgraphs around events
- [x] Annotate primitive — tracks annotation events
- [x] Timeline primitive — provides chronological event views

### Group 8 — Integrity — DONE
- [x] HashChain primitive — maintains and monitors hash chain
- [x] ChainVerify primitive — periodically verifies chain integrity
- [x] Witness primitive — records event witnessing for third-party verification
- [x] IntegrityViolation primitive — detects chain integrity violations

### Group 9 — Deception — DONE
- [x] Pattern primitive — detects recurring event patterns
- [x] DeceptionIndicator primitive — watches for deceptive behaviour signs
- [x] Suspicion primitive — tracks actors with declining trust
- [x] Quarantine primitive — manages actor quarantine

### Group 10 — Health — DONE
- [x] GraphHealth primitive — monitors overall graph health
- [x] Invariant primitive — defines and checks system invariants
- [x] InvariantCheck primitive — periodic invariant verification
- [x] Bootstrap primitive — monitors system bootstrap status

---

## Phase 3: Communication Protocol — DONE

Inter-primitive communication is the event graph itself. Primitives communicate by emitting events (via `AddEvent` mutations), and the tick engine routes those events to primitives whose `Subscriptions()` patterns match. This is already fully implemented:

- [x] **Message passing** — primitives emit typed events, tick engine delivers to subscribers
- [x] **Listen/Say** — `Subscriptions()` defines what a primitive listens for; `AddEvent` mutations are how it speaks
- [x] **Routing** — tick engine matches event types to subscription patterns, respects layer ordering
- [x] **Knowledge architecture** — per-tick snapshots (context), primitive state (memory), lifecycle mutations (structural change)

The subscription contracts for all 201 primitives are specified in `docs/primitives.md`.

---

## Phase 4: Layers 1-13 — DONE

156 primitives across 13 layers (12 per layer, 3 groups of 4). Each implements the Primitive interface with correct layer, subscriptions matching `docs/primitives.md`, and state-tracking Process methods. All mechanical implementations — intelligent behaviour (IIntelligence, IDecisionMaker) will be wired when those interfaces are needed.

### Layer 1 — Agency (Observer → Participant) — DONE
- [x] `go/pkg/primitive/layer1/` — Goal, Plan, Initiative, Commitment, Focus, Filter, Salience, Distraction, Permission, Capability, Delegation, Accountability

### Layer 2 — Exchange (Individual → Dyad) — DONE
- [x] `go/pkg/primitive/layer2/` — Message, Acknowledgement, Clarification, Context, Offer, Acceptance, Obligation, Gratitude, Negotiation, Consent, Contract, Dispute

### Layer 3 — Society (Dyad → Group) — DONE
- [x] `go/pkg/primitive/layer3/` — Group, Role, Reputation, Exclusion, Vote, Consensus, Dissent, Majority, Convention, Norm, Sanction, Forgiveness

### Layer 4 — Legal (Informal → Formal) — DONE
- [x] `go/pkg/primitive/layer4/` — Rule, Jurisdiction, Precedent, Interpretation, Adjudication, Appeal, DueProcess, Rights, Audit, Enforcement, Amnesty, Reform

### Layer 5 — Technology (Governing → Building) — DONE
- [x] `go/pkg/primitive/layer5/` — Create, Tool, Quality, Deprecation, Workflow, Automation, Testing, Review, Feedback, Iteration, Innovation, Legacy

### Layer 6 — Information (Physical → Symbolic) — DONE
- [x] `go/pkg/primitive/layer6/` — Symbol, Abstraction, Classification, Encoding, Fact, Inference, Memory, Learning, Narrative, Bias, Correction, Provenance

### Layer 7 — Ethics (Is → Ought) — DONE
- [x] `go/pkg/primitive/layer7/` — Value, Harm, Fairness, Care, Dilemma, Proportionality, Intention, Consequence, Responsibility, Transparency, Redress, Growth

### Layer 8 — Identity (Doing → Being) — DONE
- [x] `go/pkg/primitive/layer8/` — SelfModel, Authenticity, NarrativeIdentity, Boundary, Persistence, Transformation, Heritage, Aspiration, Dignity, Acknowledgement, Uniqueness, Memorial

### Layer 9 — Relationship (Self → Self-with-Other) — DONE
- [x] `go/pkg/primitive/layer9/` — Attachment, Reciprocity, RelationalTrust, Rupture, Apology, Reconciliation, RelationalGrowth, Loss, Vulnerability, Understanding, Empathy, Presence

### Layer 10 — Community (Relationship → Belonging) — DONE
- [x] `go/pkg/primitive/layer10/` — Home, Contribution, Inclusion, Tradition, Commons, Sustainability, Succession, Renewal, Milestone, Ceremony, Story, Gift

### Layer 11 — Culture (Living → Seeing) — DONE
- [x] `go/pkg/primitive/layer11/` — SelfAwareness, Perspective, Critique, Wisdom, Aesthetic, Metaphor, Humour, Silence, Teaching, Translation, Archive, Prophecy

### Layer 12 — Emergence (Content → Architecture) — DONE
- [x] `go/pkg/primitive/layer12/` — MetaPattern, SystemDynamic, FeedbackLoop, Threshold, Adaptation, Selection, Complexification, Simplification, SystemicIntegrity, Harmony, Resilience, Purpose

### Layer 13 — Existence (Everything → The Fact of Everything) — DONE
- [x] `go/pkg/primitive/layer13/` — Being, Finitude, Change, Interdependence, Mystery, Paradox, Infinity, Void, Awe, ExistentialGratitude, Play, Wonder

### Integration Test Scenarios — DONE

13 end-to-end scenarios exercising the full primitive stack through concrete use cases. Each scenario uses social grammar operations + direct event recording through a domain-specific story. All tests in `go/pkg/integration/`.

| # | Scenario | Product Graph | Status |
|---|----------|--------------|--------|
| 1 | [AI Agent Audit Trail](docs/tests/primitives/01-agent-audit-trail.md) | Work / Ethics | DONE |
| 2 | [Freelancer Reputation](docs/tests/primitives/02-freelancer-reputation.md) | Market | DONE |
| 3 | [Consent-Based Journal](docs/tests/primitives/03-consent-journal.md) | Relationship | DONE |
| 4 | [Community Governance](docs/tests/primitives/04-community-governance.md) | Governance | DONE |
| 5 | [Supply Chain Transparency](docs/tests/primitives/05-supply-chain.md) | Work | DONE (multi-system EGIP: 3 sovereign graphs, treaties, CGERs, proofs, trust) |
| 6 | [Research Integrity](docs/tests/primitives/06-research-integrity.md) | Research | DONE |
| 7 | [Creator Provenance](docs/tests/primitives/07-creator-provenance.md) | Culture | DONE |
| 8 | [Family Decision Log](docs/tests/primitives/08-family-decision-log.md) | Social | DONE |
| 9 | [Knowledge Verification](docs/tests/primitives/09-knowledge-verification.md) | Knowledge | DONE |
| 10 | [AI Ethics Audit](docs/tests/primitives/10-ai-ethics-audit.md) | Ethics | DONE |
| 11 | [Agent Identity Lifecycle](docs/tests/primitives/11-agent-identity-lifecycle.md) | Identity | DONE |
| 12 | [Community Lifecycle](docs/tests/primitives/12-community-lifecycle.md) | Community | DONE |
| 13 | [System Self-Evolution](docs/tests/primitives/13-system-self-evolution.md) | Emergence | DONE |

### Composition Grammars — DONE

13 per-layer composition grammar types in `go/pkg/compositions/`. Each grammar wraps `*grammar.Grammar` and provides domain-specific operations that compose the base 15+1 grammar operations (15 base + Challenge). ~145 operations + 66 named functions total, with full test suites (157 subtests). All named functions from specs fully implemented.

| # | Grammar | Layer | Operations | Named Functions | Status |
|---|---------|-------|------------|-----------------|--------|
| 1 | WorkGrammar | Agency | 12 | Sprint, Escalate, DelegateAndVerify, Standup, Retrospective, Triage | DONE |
| 2 | MarketGrammar | Exchange | 14 | Auction, Milestone, Barter, Subscription, Refund, ReputationTransfer, Arbitration | DONE |
| 3 | SocialGrammar | Society | 5 | Poll, Federation, Schism | DONE |
| 4 | JusticeGrammar | Legal | 12 | Trial, ConstitutionalAmendment, Injunction, Plea, ClassAction, Recall | DONE |
| 5 | BuildGrammar | Technology | 12 | Pipeline, PostMortem, Spike, Migration, TechDebt | DONE |
| 6 | KnowledgeGrammar | Information | 12 | Retract, FactCheck, Verify, Survey, KnowledgeBase, Transfer | DONE |
| 7 | AlignmentGrammar | Ethics | 10 | EthicsAudit, RestorativeJustice, Guardrail, ImpactAssessment, Whistleblow | DONE |
| 8 | IdentityGrammar | Identity | 10 | IdentityAudit, Retirement, Credential, Reinvention, Introduction | DONE |
| 9 | BondGrammar | Relationship | 10 | BetrayalRepair, CheckIn, Forgive, Mentorship, Farewell | DONE |
| 10 | BelongingGrammar | Community | 10 | Onboard, Succession, Festival, CommonsGovernance, Renewal | DONE |
| 11 | MeaningGrammar | Culture | 10 | PostMortem, Mentorship, DesignReview, CulturalOnboarding, Forecast | DONE |
| 12 | EvolutionGrammar | Emergence | 10 | SelfEvolve, HealthCheck, Prune, PhaseTransition | DONE |
| 13 | BeingGrammar | Existence | 8 | Contemplation, ExistentialAudit, Farewell | DONE |

---

## Phase 5: EGIP (Inter-System Protocol) — DONE

Sovereign systems communicating across graph boundaries.

- [x] `go/pkg/protocol/egip/identity.go` — Ed25519 keypair, System URI, IIdentity interface
- [x] `go/pkg/protocol/egip/envelope.go` — Signed message envelope, canonical form, all 7 payload types, CGERs
- [x] `go/pkg/protocol/egip/treaty.go` — Treaty model, lifecycle state machine, bilateral governance
- [x] `go/pkg/protocol/egip/trust.go` — PeerStore, inter-system trust accumulation, time-based decay
- [x] `go/pkg/protocol/egip/proof.go` — Integrity proofs (chain segment, event existence, chain summary), ProofGenerator, verification
- [x] `go/pkg/protocol/egip/transport.go` — ITransport interface (pluggable: HTTP, WebSocket, gRPC, etc.)
- [x] `go/pkg/protocol/egip/errors.go` — 7 typed EGIP errors with EGIPError marker interface
- [x] `go/pkg/protocol/egip/handler.go` — Protocol handler (HELLO handshake, message dispatch, replay dedup, trust updates), TreatyStore, EnvelopeDedup
- [x] `go/pkg/protocol/egip/egip_test.go` — 75 tests, 86.9% coverage

---

## Phase 6: Language Packages

Each language package must pass the language-agnostic conformance test suite.

### Rust — DONE (176 tests)
- [x] Core event types + hash chain
- [x] Store trait + InMemory implementation (with query methods: by_type, by_source, by_conversation, ancestors, descendants)
- [x] Bus
- [x] Primitive trait + Registry
- [x] Tick engine
- [x] Social grammar (7 vertex operations: emit, respond, derive, extend, retract, annotate, merge)
- [x] Actor module (ActorType, ActorStatus state machine, InMemoryActorStore)
- [x] Trust module (DefaultTrustModel, directional trust, decay, domain scores)
- [x] Decision module (decision trees, evaluate, evolve LLM→mechanical)
- [x] Conformance tests (matching Go reference hashes)

### Python — DONE (218 tests)
- [x] Core event types + hash chain
- [x] Store protocol + InMemory implementation (with query methods: by_type, by_source, by_conversation, ancestors, descendants)
- [x] Bus
- [x] Primitive protocol + Registry
- [x] Tick engine
- [x] Social grammar (7 vertex operations: emit, respond, derive, extend, retract, annotate, merge)
- [x] Actor module (ActorType, ActorStatus state machine, InMemoryActorStore)
- [x] Trust module (DefaultTrustModel, directional trust, decay, domain scores)
- [x] Decision module (decision trees, evaluate, evolve LLM→mechanical)
- [x] Conformance tests (matching Go reference hashes)

### TypeScript/npm — DONE (231 tests)
- [x] Core event types + hash chain
- [x] Store interface + InMemory implementation (with query methods: byType, bySource, byConversation, ancestors, descendants)
- [x] Bus
- [x] Primitive interface + Registry
- [x] Tick engine
- [x] Social grammar (7 vertex operations: emit, respond, derive, extend, retract, annotate, merge)
- [x] Actor module (ActorType, ActorStatus state machine, InMemoryActorStore)
- [x] Trust module (DefaultTrustModel, directional trust, decay, domain scores)
- [x] Decision module (decision trees, evaluate, evolve LLM→mechanical)
- [x] Conformance tests (matching Go reference hashes)

### .NET — DONE (191 tests)
- [x] Core event types + hash chain
- [x] IStore interface + InMemory implementation (with query methods: ByType, BySource, ByConversation, Ancestors, Descendants)
- [x] Bus
- [x] IPrimitive interface + Registry
- [x] Tick engine
- [x] Social grammar (7 vertex operations: Emit, Respond, Derive, Extend, Retract, Annotate, Merge)
- [x] Actor module (ActorType, ActorStatus state machine, InMemoryActorStore)
- [x] Trust module (DefaultTrustModel, directional trust, decay, domain scores)
- [x] Decision module (decision trees, evaluate, evolve LLM→mechanical)
- [x] Conformance tests (matching Go reference hashes)

---

## Phase 7: Documentation & Examples

- [x] `docs/conformance/` — Language-agnostic conformance test specification
- [x] `docs/conformance/canonical-vectors.json` — Test vectors for canonical form, hash, and signature verification
- [x] `examples/minimal/` — Bootstrap, emit, derive, verify chain (runnable Go program)
- [x] `examples/social/` — All 15 social grammar operations demonstrated
- [x] `examples/multi-system/` — Two sovereign systems with EGIP: HELLO, treaty, CGERs, proof, trust
- [x] Tutorial: "Build your first primitive" (`docs/tutorials/01-first-primitive.md`)
- [x] Tutorial: "Implement a custom store" (`docs/tutorials/02-custom-store.md`)
- [x] Tutorial: "Connect two event graphs" (`docs/tutorials/03-connect-two-graphs.md`)

### CI/CD & Publishing — DONE
- [x] `.github/workflows/ci.yml` — Tests all languages on push/PR (Go, Python 3.10-3.13, .NET 9, Rust, TypeScript/Node 22)
- [x] `.github/workflows/publish-npm.yml` — Publishes `@lovyou-ai/eventgraph` to npm on release
- [x] `.github/workflows/publish-pypi.yml` — Publishes `eventgraph` to PyPI on release (trusted publisher)
- [x] `.github/workflows/publish-nuget.yml` — Publishes `LovYou.EventGraph` to NuGet on release
- [x] `.github/workflows/publish-crates.yml` — Publishes `eventgraph` to crates.io on release

---

## Future

These are product layers — built *on* the event graph, not part of the infrastructure package:

- [ ] Product layer: Social Grammar (15 operations from Post 35)
- [ ] Product layer: Governance (Post 34)
- [ ] Product layer: Exchange / Market
- [ ] Product layer: Task Management (hierarchical decomposition, model-tier routing)
- [ ] WebAssembly builds for browser-based event graphs
- [ ] Mobile SDKs
- [ ] Reference UI implementations
- [ ] Hosted persistence service
