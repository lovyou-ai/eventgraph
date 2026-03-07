# Architecture Test 11: Composition Safety

Verify that primitives and grammar operations compose without violating system invariants.

## Purpose

The SDK provides composition grammars (Work, Market, Justice, etc.) that combine multiple primitives into single operations. This test verifies that compositions maintain all invariants — hash chain integrity, causal links, typed content, authority checks — regardless of how they're combined.

## Setup

```
graph: initialized EventGraph with full primitive stack
actors: [human_a, agent_b]
grammars: [WorkGrammar, MarketGrammar]
```

## Test Cases

### TC-11.1: Composition Preserves Causality

**Input:** Execute a WorkGrammar.Sprint (Intend + Decompose + Assign batch).
**Assertions:**
- Every event in the Sprint has valid causes
- Goal event causes Plan events
- Plan events cause Delegation events
- No event is orphaned (all linked to the Sprint's root)

### TC-11.2: Composition Preserves Hash Chain

**Input:** Execute any grammar operation.
**Assertions:**
- All events emitted by the composition are hash-chained
- `Store.VerifyChain()` passes after the composition
- PrevHash links are correct for every event

### TC-11.3: Cross-Grammar Composition

**Input:** WorkGrammar.Assign → MarketGrammar.Invoice (assign work, then invoice for it).
**Assertions:**
- Invoice event has causal link to Assignment event
- Cross-grammar link is valid
- Both grammars' invariants are maintained

### TC-11.4: Composition Atomicity

**Input:** Execute a multi-step composition where step 3 of 5 fails validation.
**Assertions:**
- Steps 1-2 are NOT committed (all-or-nothing)
- Store state is unchanged
- Error identifies which step failed and why

### TC-11.5: Composition Authority

**Input:** Execute a composition where one step requires authority.
**Assertions:**
- Composition pauses at the authority-requiring step
- Prior steps are held (not committed until authority resolves)
- On approval, remaining steps execute
- On rejection, entire composition is rolled back

### TC-11.6: Parallel Composition

**Input:** Two actors execute compositions simultaneously on overlapping state.
**Assertions:**
- No data corruption
- Both compositions produce valid events
- Hash chain remains linear (serialized at commit time)
- Trust scores are consistent

### TC-11.7: Composition with Custom Primitives

**Input:** Create a custom grammar operation that combines built-in and custom primitives.
**Assertions:**
- Custom primitive integrates cleanly
- Events from custom primitive pass validation
- Causal links between built-in and custom events are valid

### TC-11.8: Named Function Decomposition

**Input:** Execute a named function (e.g., MarketGrammar.Auction).
**Assertions:**
- Auction decomposes into its constituent operations (List, Bid×N, Accept)
- Each constituent operation is independently valid
- The full function produces a coherent event subgraph
- Traversing the subgraph tells the story of the auction

### TC-11.9: Invariant Preservation Under Stress

**Input:** Execute 1000 random grammar operations from random actors.
**Assertions:**
- After all operations: `Store.VerifyChain()` passes
- Every event has valid causes
- Every event has valid signatures
- No orphaned events
- Trust scores are all within [0.0, 1.0]

## Reference

- `docs/compositions/` — Grammar specifications
- `docs/interfaces.md` — Core interfaces and invariants
