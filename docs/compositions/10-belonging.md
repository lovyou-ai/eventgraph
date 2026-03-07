# Belonging Grammar (Layer 10: Community)

The grammar for communities with shared resources, traditions, and belonging.

## Derivation

Community is operations on collective life. The base operations are: **belong**, **steward**, **celebrate**, **give**. Four semantic dimensions differentiate operations:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| Subject | Individual (one member) / Collective (the community) | Personal or communal? |
| Temporality | Present (now) / Legacy (generational) | This moment or across time? |
| Flow | Inward (receiving) / Outward (contributing) | Taking or giving? |
| Register | Practical (material) / Ceremonial (symbolic) | Functional or meaningful? |

## Operations (10)

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 1 | **Settle** | belonging/arrive | Develop a sense of home in a community | Home + Subscribe (community) |
| 2 | **Contribute** | belonging/give | Add value to the community | Contribution + Emit |
| 3 | **Include** | belonging/welcome | Remove barriers to participation | Inclusion + Emit |
| 4 | **Practice** | belonging/tradition | Participate in a community tradition | Tradition + Emit (observance) |
| 5 | **Steward** | stewardship/care | Take responsibility for shared resources | Commons + Delegate (to steward) |
| 6 | **Sustain** | stewardship/assess | Evaluate long-term viability | Sustainability + Emit |
| 7 | **Pass-On** | stewardship/transfer | Transfer stewardship to next generation | Succession + Consent |
| 8 | **Celebrate** | ceremony/mark | Formally recognize an achievement | Milestone + Ceremony + Emit |
| 9 | **Tell** | ceremony/narrate | Add a chapter to the community's story | Story + Emit |
| 10 | **Gift** | generosity/give | Give without expectation of return | Gift + Emit |

## Modifiers (2)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| **Collective** | Operation requires community-wide participation | Practice, Celebrate, Sustain |
| **Founding** | Marks this as originating a new tradition/resource | Practice, Steward, Tell |

## Named Functions (5)

| Function | Composition | Purpose |
|----------|------------|---------|
| **Onboard** | Include + Settle + Practice (introductory) + Contribute (first) | Full newcomer welcome |
| **Festival** | Celebrate (Collective) + Practice + Tell + Gift | Community-wide celebration event |
| **Succession** | Sustain + Pass-On + Celebrate (Witnessed) + Tell | Full generational transfer |
| **Commons-Governance** | Steward + Sustain + Legislate (L4) + Audit (L4) | Manage shared resources with rules |
| **Renewal** | Sustain (crisis detected) + Practice (evolved) + Tell (new chapter) | Community regeneration |

## Example Flow

**Open source community lifecycle:**
```
Settle(alice, community="eventgraph-contributors", belonging=0.3)
  → Include(improve docs for newcomers, beneficiary=all-newcomers)
  → Contribute(alice, type="bug fix", value=0.6)
  → Contribute(alice, type="feature", value=0.8)
  → Settle(belonging=0.7, alice feels at home)
  → Practice("Friday code review" tradition, adherence=0.9)
  → Steward(alice, resource="test infrastructure")
  → Sustain(score=0.85, risk="bus factor on auth module")
  → Pass-On(alice→bob, scope="test infrastructure stewardship")
  → Celebrate(milestone="v2.0 shipped", contributors=[alice, bob, ...])
  → Tell("the v2 migration: how we rewrote auth in 3 months")
  → Gift(alice→community, "wrote comprehensive testing guide, no strings")
```

## Reference

- `docs/grammar.md` — Infrastructure grammar (15 operations)
- `docs/layers/10-community.md` — Layer 10 derivation
- `docs/primitives.md` — Layer 10 primitive specifications
