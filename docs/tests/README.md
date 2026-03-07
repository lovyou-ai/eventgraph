# Test Specifications

Test specifications for the event graph SDK. Three test suites covering the full stack:

1. **Primitive tests** — Integration tests exercising Layer 0 primitives through real product use cases
2. **Architecture tests** — Verify the plugin contract: overridability, extensibility, isolation, and composition of core interfaces
3. **Composition tests** — Verify each layer's grammar operations as SDK-level APIs

## Primitive Tests

Integration tests that exercise the full Layer 0 stack through concrete product use cases. See [`primitives/README.md`](primitives/README.md).

| # | Scenario | Product Graph |
|---|----------|--------------|
| 1 | [AI Agent Audit Trail](primitives/01-agent-audit-trail.md) | Work (L1) / Ethics (L7) |
| 2 | [Freelancer Reputation](primitives/02-freelancer-reputation.md) | Market (L2) |
| 3 | [Consent-Based Journal](primitives/03-consent-journal.md) | Relationship (L9) |
| 4 | [Community Governance](primitives/04-community-governance.md) | Governance (L11) |
| 5 | [Supply Chain Transparency](primitives/05-supply-chain.md) | Work (L1) |
| 6 | [Research Integrity](primitives/06-research-integrity.md) | Research (L5) |
| 7 | [Creator Provenance](primitives/07-creator-provenance.md) | Culture (L12) |
| 8 | [Family Decision Log](primitives/08-family-decision-log.md) | Social (L3) |
| 9 | [Knowledge Verification](primitives/09-knowledge-verification.md) | Knowledge (L6) |
| 10 | [AI Ethics Audit](primitives/10-ai-ethics-audit.md) | Ethics (L7) |
| 11 | [Agent Identity Lifecycle](primitives/11-agent-identity-lifecycle.md) | Identity (L8) |
| 12 | [Community Lifecycle](primitives/12-community-lifecycle.md) | Community (L10) |
| 13 | [System Self-Evolution](primitives/13-system-self-evolution.md) | Emergence (L12) |

## Architecture Tests

These verify that the system's extension points actually work — that you can swap implementations, register custom types, and compose primitives without breaking invariants.

| # | Test Suite | What it verifies |
|---|-----------|------------------|
| 1 | [Store Conformance](architecture/01-store-conformance.md) | Any Store impl passes the same test suite |
| 2 | [Intelligence Pluggability](architecture/02-intelligence-pluggability.md) | Swap IIntelligence, same behavioral contract |
| 3 | [Decision Maker Pluggability](architecture/03-decision-maker-pluggability.md) | Swap IDecisionMaker, decision tree still works |
| 4 | [Primitive Registration](architecture/04-primitive-registration.md) | Register custom primitives, lifecycle honored |
| 5 | [Event Type Extensibility](architecture/05-event-type-extensibility.md) | Register custom event types, validation works |
| 6 | [Decision Tree Override](architecture/06-decision-tree-override.md) | Replace/extend decision trees at runtime |
| 7 | [Authority Customization](architecture/07-authority-customization.md) | Custom authority policies, escalation rules |
| 8 | [Bus & Subscription](architecture/08-bus-subscription.md) | Custom subscribers, filtering, ordering guarantees |
| 9 | [Tick Engine Isolation](architecture/09-tick-engine-isolation.md) | Snapshot immutability, atomic mutations, wave limits |
| 10 | [EGIP Extensibility](architecture/10-egip-extensibility.md) | Custom message types, treaty terms, trust federation |
| 11 | [Composition Safety](architecture/11-composition-safety.md) | Primitives compose without violating invariants |
| 12 | [Concurrency & Ordering](architecture/12-concurrency-ordering.md) | Concurrent access, hash chain integrity under load |

## Composition Tests

These verify each layer's grammar operations — the named compositions that the SDK provides as developer-facing APIs. Each test exercises the full operation lifecycle, including the primitives it composes and the events it produces.

| # | Grammar | Test File |
|---|---------|-----------|
| 1 | Work | [Work Grammar Tests](compositions/01-work-grammar.md) |
| 2 | Market | [Market Grammar Tests](compositions/02-market-grammar.md) |
| 3 | Social | [Social Grammar Tests](compositions/03-social-grammar.md) |
| 4 | Justice | [Justice Grammar Tests](compositions/04-justice-grammar.md) |
| 5 | Build | [Build Grammar Tests](compositions/05-build-grammar.md) |
| 6 | Knowledge | [Knowledge Grammar Tests](compositions/06-knowledge-grammar.md) |
| 7 | Ethics | [Alignment Grammar Tests](compositions/07-alignment-grammar.md) |
| 8 | Identity | [Identity Grammar Tests](compositions/08-identity-grammar.md) |
| 9 | Relationship | [Bond Grammar Tests](compositions/09-bond-grammar.md) |
| 10 | Community | [Belonging Grammar Tests](compositions/10-belonging-grammar.md) |
| 11 | Culture | [Meaning Grammar Tests](compositions/11-meaning-grammar.md) |
| 12 | Emergence | [Evolution Grammar Tests](compositions/12-evolution-grammar.md) |
| 13 | Existence | [Being Grammar Tests](compositions/13-being-grammar.md) |

## How to Read These

Each test spec includes:

- **Setup** — what needs to exist before the test runs
- **Test cases** — individual operations with inputs, expected events, and assertions
- **Named function tests** — multi-operation compositions
- **Error cases** — what happens when operations fail, inputs are invalid, or invariants are violated
- **Cross-layer tests** — operations that span multiple layers

## Implementation Guidance

Architecture tests should be **table-driven** and **implementation-agnostic**:

```go
// Every Store implementation runs the same suite
func TestStoreConformance(t *testing.T, factory func() Store) {
    for _, tc := range storeConformanceCases {
        t.Run(tc.Name, func(t *testing.T) {
            store := factory()
            // ...
        })
    }
}
```

Composition tests should verify the **event chain**, not just the final state:

```go
// Verify that Intend produces the right events with the right causes
func TestWorkGrammar_Intend(t *testing.T) {
    result := workGrammar.Intend(IntendInput{...})
    assert.EventType(t, result.Events[0], "goal.set")
    assert.HasCause(t, result.Events[0], bootstrapEvent)
    assert.PrimitiveActivated(t, result, "Goal")
}
```

## Reference

- `docs/tests/primitives/` — Infrastructure integration test scenarios (Layer 0 focus)
- `docs/compositions/` — The grammar specifications being tested
- `docs/conformance/` — Language-agnostic conformance test vectors
- `docs/coding-standards/go.md` — Go test patterns and conventions
