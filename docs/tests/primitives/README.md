# Primitive Integration Tests

Infrastructure-level integration test scenarios. Each exercises the full Layer 0 stack (store, bus, primitives, tick engine, trust, authority, decisions) through a concrete product use case.

These tests use only Layer 0 primitives and the 15 social grammar operations. They work with just the infrastructure — no product layer intelligence required. Each scenario notes which product graph (Layer 1-13) would eventually add domain-specific intelligence on top.

## Scenarios

| # | Scenario | Product Graph | Grammar Operations |
|---|----------|--------------|-------------------|
| 1 | [AI Agent Audit Trail](01-agent-audit-trail.md) | Work (L1) / Ethics (L7) | Emit, Respond, Derive, Delegate |
| 2 | [Freelancer Reputation](02-freelancer-reputation.md) | Market (L2) | Emit, Endorse, Subscribe, Acknowledge, Channel, Consent |
| 3 | [Consent-Based Journal](03-consent-journal.md) | Relationship (L9) | Emit, Channel, Consent, Respond, Sever, Forgive |
| 4 | [Community Governance](04-community-governance.md) | Governance (L11) | Emit, Respond, Consent, Delegate, Annotate |
| 5 | [Supply Chain Transparency](05-supply-chain.md) | Work (L1) | Emit, Derive, Endorse + EGIP |
| 6 | [Research Integrity](06-research-integrity.md) | Research (L5) | Emit, Extend, Derive, Respond, Endorse, Challenge |
| 7 | [Creator Provenance](07-creator-provenance.md) | Culture (L12) | Emit, Derive, Annotate, Endorse |
| 8 | [Family Decision Log](08-family-decision-log.md) | Social (L3) | Emit, Respond, Delegate, Consent |
| 9 | [Knowledge Verification](09-knowledge-verification.md) | Knowledge (L6) | Emit, Derive, Annotate, Challenge, Traverse |
| 10 | [AI Ethics Audit](10-ai-ethics-audit.md) | Ethics (L7) | Emit, Annotate, Consent, Traverse |
| 11 | [Agent Identity Lifecycle](11-agent-identity-lifecycle.md) | Identity (L8) | Emit, Annotate, Derive, Channel |
| 12 | [Community Lifecycle](12-community-lifecycle.md) | Community (L10) | Emit, Endorse, Subscribe, Acknowledge, Consent |
| 13 | [System Self-Evolution](13-system-self-evolution.md) | Emergence (L12) | Emit, Derive, Annotate, Consent |

## How to Read These

Each scenario specifies:

- **Actors** — who's involved, with types (Human, AI, System)
- **Setup** — what exists on the graph before the scenario begins
- **Event sequence** — numbered steps, each with the grammar operation, event type, content, and causes
- **Edges created** — trust, endorsement, subscription, delegation edges and their weights
- **Trust and authority flows** — how trust scores change, what authority levels apply
- **Assertions** — what an integration test should verify
- **What higher layers add** — what intelligence the product graph's Layer N primitives would contribute

## Using as Tests

Each scenario is an acceptance test specification. To implement:

1. Create actors via `IActorStore.Register()`
2. Bootstrap the graph via `BootstrapFactory.Init()`
3. Execute each step via `IGraph.Record()` or `EdgeFactory.Create()`
4. After each step, verify assertions against `Store` queries and `ITrustModel` scores
5. At the end, verify the full chain via `Store.VerifyChain()`

The `PrimitiveTestHarness` can be used to verify that the right primitives fire on each event and produce the expected mutations.
