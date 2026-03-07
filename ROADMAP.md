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
- [x] `go/pkg/grammar/grammar.go` — 15 social grammar operations + 3 named functions
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

### Group 3 — Expectations
- [ ] Expectation primitive
- [ ] Timeout primitive
- [ ] Violation primitive
- [ ] Severity primitive

### Group 4 — Trust
- [ ] TrustScore primitive
- [ ] TrustUpdate primitive
- [ ] Corroboration primitive
- [ ] Contradiction primitive

### Group 5 — Confidence
- [ ] Confidence primitive
- [ ] Evidence primitive
- [ ] Revision primitive
- [ ] Uncertainty primitive

### Group 6 — Instrumentation
- [ ] InstrumentationSpec primitive
- [ ] CoverageCheck primitive
- [ ] Gap primitive
- [ ] Blind primitive

### Group 7 — Query
- [ ] PathQuery primitive
- [ ] SubgraphExtract primitive
- [ ] Annotate primitive
- [ ] Timeline primitive

### Group 8 — Integrity
- [ ] HashChain primitive
- [ ] ChainVerify primitive
- [ ] Witness primitive
- [ ] IntegrityViolation primitive

### Group 9 — Deception
- [ ] Pattern primitive
- [ ] DeceptionIndicator primitive
- [ ] Suspicion primitive
- [ ] Quarantine primitive

### Group 10 — Health
- [ ] GraphHealth primitive
- [ ] Invariant primitive
- [ ] InvariantCheck primitive
- [ ] Bootstrap primitive

---

## Phase 3: Communication Protocol

Inter-primitive communication within a single system.

- [ ] `go/pkg/protocol/message.go` — Four event types: MessageSent, MessageReceived, Decision, Action
- [ ] `go/pkg/protocol/listen_say.go` — Listen/Say interface for communicator primitives
- [ ] `go/pkg/protocol/router.go` — Semantic routing (Self primitive routes to relevant domain primitives)
- [ ] `go/pkg/protocol/knowledge.go` — Three-layer knowledge architecture (context, memory, structural change)
- [ ] Tests for all of the above

---

## Phase 4: EGIP (Inter-System Protocol)

Sovereign systems communicating across graph boundaries.

- [ ] `go/pkg/protocol/egip/identity.go` — Ed25519 keypair, System URI
- [ ] `go/pkg/protocol/egip/cger.go` — Cross-Graph Event Reference
- [ ] `go/pkg/protocol/egip/envelope.go` — Signed message envelope
- [ ] `go/pkg/protocol/egip/messages.go` — Seven message types
- [ ] `go/pkg/protocol/egip/treaty.go` — Treaty model, lifecycle, bilateral governance
- [ ] `go/pkg/protocol/egip/trust.go` — Trust accumulation model
- [ ] `go/pkg/protocol/egip/proof.go` — Integrity proofs (chain segment, event existence, chain summary)
- [ ] Tests for all of the above

---

## Phase 5: Layers 1-13

Each layer has 12 primitives in 3 groups of 4. Layer N depends on Layer N-1 being stable.

### Layer 1 — Agency (Observer → Participant)
- [ ] Specification (`docs/layers/01-agency.md`)
- [ ] 12 primitives implementation + tests

### Layer 2 — Exchange (Individual → Dyad)
- [ ] Specification (`docs/layers/02-exchange.md`)
- [ ] 12 primitives implementation + tests

### Layer 3 — Society (Dyad → Group)
- [ ] Specification
- [ ] 12 primitives implementation + tests

### Layer 4 — Legal (Informal → Formal)
- [ ] Specification
- [ ] 12 primitives implementation + tests

### Layer 5 — Technology (Governing → Building)
- [ ] Specification
- [ ] 12 primitives implementation + tests

### Layer 6 — Information (Physical → Symbolic)
- [ ] Specification
- [ ] 12 primitives implementation + tests

### Layer 7 — Ethics (Is → Ought)
- [ ] Specification
- [ ] 12 primitives implementation + tests

### Layer 8 — Identity (Doing → Being)
- [ ] Specification
- [ ] 12 primitives implementation + tests

### Layer 9 — Relationship (Self → Self-with-Other)
- [ ] Specification
- [ ] 12 primitives implementation + tests

### Layer 10 — Community (Relationship → Belonging)
- [ ] Specification
- [ ] 12 primitives implementation + tests

### Layer 11 — Culture (Living → Seeing)
- [ ] Specification
- [ ] 12 primitives implementation + tests

### Layer 12 — Emergence (Content → Architecture)
- [ ] Specification
- [ ] 12 primitives implementation + tests

### Layer 13 — Existence (Everything → The Fact of Everything)
- [ ] Specification
- [ ] 12 primitives implementation + tests

---

## Phase 6: Language Packages

Each language package must pass the language-agnostic conformance test suite.

### Rust
- [ ] Core event types + hash chain
- [ ] Store trait + InMemory implementation
- [ ] Bus
- [ ] Primitive trait + Registry
- [ ] Tick engine
- [ ] Conformance tests passing

### Python
- [ ] Core event types + hash chain
- [ ] Store protocol + InMemory implementation
- [ ] Bus
- [ ] Primitive protocol + Registry
- [ ] Tick engine
- [ ] Conformance tests passing

### .NET
- [ ] Core event types + hash chain
- [ ] IStore interface + InMemory implementation
- [ ] Bus
- [ ] IPrimitive interface + Registry
- [ ] Tick engine
- [ ] Conformance tests passing

---

## Phase 7: Documentation & Examples

- [x] `docs/conformance/` — Language-agnostic conformance test specification
- [x] `docs/conformance/canonical-vectors.json` — Test vectors for canonical form, hash, and signature verification
- [ ] `examples/minimal/` — Smallest possible event graph (10 lines of code)
- [ ] `examples/social/` — The 15 social grammar operations on the event graph
- [ ] `examples/multi-system/` — Two systems communicating via EGIP
- [ ] Tutorial: "Build your first primitive"
- [ ] Tutorial: "Implement a custom store"
- [ ] Tutorial: "Connect two event graphs"

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
