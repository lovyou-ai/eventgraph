# Architecture Test 7: Authority Customization

Verify that the authority system accepts custom policies, escalation rules, and approval workflows.

## Purpose

Different deployments need different authority rules. A solo founder's system auto-approves most things. An enterprise system requires multi-level approval. This test verifies the authority system is configurable, not hardcoded.

## Setup

```
policies: [
    DefaultPolicy:    { deploy: Required, comment: Notification }
    EnterprisePolicy: { deploy: Required+2approvers, comment: Recommended }
    SoloPolicy:       { deploy: Recommended, comment: Notification }
]
```

## Test Cases

### TC-7.1: Custom Policy Registration

**Input:** Register a custom authority policy with non-default rules.
**Assertions:**
- Policy is active for its scope
- authority.requested events respect the custom policy
- Default policy still applies to unscoped actions

### TC-7.2: Policy Scoping

**Input:** Register policy A for scope "deploy.*", policy B for scope "data.*".
**Assertions:**
- Deploy actions use policy A
- Data actions use policy B
- Unmatched actions use default policy

### TC-7.3: Multi-Approver

**Input:** Register policy requiring 2 approvers for "deploy.production".
**Assertions:**
- First approval doesn't resolve the request
- Second approval resolves it
- Both approvals recorded as events with causes

### TC-7.4: Timeout Behavior

**Input:** Recommended-level request with 15-minute timeout.
**Assertions:**
- Request auto-approves after 15 minutes
- Auto-approval event includes reason "timeout"
- Required-level requests do NOT auto-approve (no timeout)

### TC-7.5: Escalation Chain

**Input:** Request not resolved within escalation window.
**Assertions:**
- System escalates to next authority level
- Escalation recorded as event
- Original request remains pending
- Escalated request links causally to original

### TC-7.6: Custom Escalation Rules

**Input:** Register custom escalation: "if unresolved after 1hr, notify admin".
**Assertions:**
- Custom escalation fires after 1 hour
- Admin notification event emitted
- Configurable per-policy

### TC-7.7: Policy Override at Request Time

**Input:** Request authority with explicit level override (e.g., force Required on normally Notification action).
**Assertions:**
- Override is respected (higher level, not lower)
- Cannot downgrade from Required to Notification
- Override recorded in the request event

## Reference

- `docs/authority.md` — Authority system specification
- `docs/interfaces.md` — IAuthorityChain specification
