# Architecture Test 10: EGIP Extensibility

Verify that the inter-system protocol accepts custom message types, treaty terms, and trust federation configurations.

## Purpose

EGIP connects sovereign systems. Different deployments need different treaty terms, trust policies, and message types. This test verifies EGIP is a protocol framework, not a fixed protocol.

## Setup

```
system_a: EventGraph instance with Ed25519 keypair
system_b: EventGraph instance with Ed25519 keypair
treaty: bilateral governance agreement
```

## Test Cases

### TC-10.1: Custom Message Type

**Input:** Register a custom EGIP message type "CUSTOM_SYNC" with schema.
**Assertions:**
- Message can be sent between systems
- Receiving system validates against schema
- Unknown message types are rejected with `Err(EGIPError.UnknownMessageType)`

### TC-10.2: Cross-Graph Event Reference (CGER)

**Input:** System A references an event from System B.
**Assertions:**
- CGER includes: source system ID, event hash, event type, timestamp
- System B can verify the reference via PROOF message
- CGER doesn't require access to System B's full graph

### TC-10.3: Treaty Negotiation

**Input:** System A proposes a treaty with custom terms.
**Assertions:**
- Treaty proposal includes natural-language terms + machine-readable policy
- System B can accept, reject, or counter-propose
- Accepted treaty creates mutual governance
- Treaty is recorded as events on both graphs

### TC-10.4: Custom Treaty Terms

**Input:** Treaty with custom term: "data retention: 90 days".
**Assertions:**
- Custom term is stored in treaty
- Term is available to both systems
- Violation of custom term can be detected and flagged

### TC-10.5: Trust Federation

**Input:** System A trusts System B at 0.7. System B trusts System C at 0.8.
**Assertions:**
- Trust is non-transitive by default: System A has NO implicit trust of System C
- If explicitly configured, transitive trust can be enabled with decay
- Trust scores are asymmetric (A→B ≠ B→A)

### TC-10.6: Signed Envelope Verification

**Input:** System A sends a signed message to System B.
**Assertions:**
- Envelope includes Ed25519 signature
- System B verifies signature against System A's public key
- Tampered envelope is rejected
- Replay detection works (duplicate envelope ID rejected)

### TC-10.7: Discovery Protocol

**Input:** System A sends DISCOVER to find other systems.
**Assertions:**
- Systems respond with capability advertisement
- Discovery doesn't require prior trust
- Discovery doesn't grant any trust (trust starts at 0.0)

### TC-10.8: Custom Trust Policy

**Input:** Configure System A with custom trust policy: "start at 0.1 instead of 0.0".
**Assertions:**
- New connections start at 0.1
- Policy is per-system configurable
- Policy is recorded as system configuration event

## Reference

- `docs/protocol.md` — EGIP specification
- `docs/trust.md` — Trust model
- `docs/tests/primitives/05-supply-chain.md` — Cross-system scenario
