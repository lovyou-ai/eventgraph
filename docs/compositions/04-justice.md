# Justice Grammar (Layer 4: Legal)

The grammar for transparent dispute resolution and governance formalisation.

## Derivation

Justice is operations on rules and disputes. The base operations are: **make rules**, **bring disputes**, **judge**, **enforce**. Four semantic dimensions differentiate operations:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| Actor | Authority (rule-maker/judge) / Party (disputant/subject) | Who is acting? |
| Phase | Legislative (making rules) / Judicial (applying rules) / Executive (enforcing rules) | Which branch of governance? |
| Direction | Forward (prospective rule) / Backward (retrospective judgment) | About the future or the past? |
| Formality | Procedural (process) / Substantive (content) | About how or about what? |

## Operations (12)

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 1 | **Legislate** | rule/create | Enact a formal rule | Rule + Emit |
| 2 | **Amend** | rule/modify | Change an existing rule | Rule (amended) + Derive |
| 3 | **Repeal** | rule/remove | Revoke an existing rule | Rule (repealed) + Retract |
| 4 | **File** | dispute/initiate | Bring a formal complaint | DueProcess + Challenge |
| 5 | **Submit** | evidence/present | Present evidence for a case | Precedent + Annotate (evidence links) |
| 6 | **Argue** | dispute/advocate | Make a legal argument | Interpretation + Respond |
| 7 | **Judge** | dispute/resolve | Render a formal ruling | Adjudication + Emit (ruling) |
| 8 | **Appeal** | dispute/challenge | Challenge a ruling to higher authority | Appeal + Challenge |
| 9 | **Enforce** | compliance/act | Execute consequences of a ruling | Enforcement + Delegate (to executor) |
| 10 | **Audit** | compliance/review | Systematic review against rules | Audit + Traverse |
| 11 | **Pardon** | compliance/override | Formally forgive a violation | Amnesty + Consent (authority) |
| 12 | **Reform** | rule/evolve | Propose systemic rule change based on experience | Reform + Derive (from precedent chain) |

## Modifiers (2)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| **Precedential** | Ruling becomes citable in future cases | Judge, Appeal |
| **Emergency** | Bypasses normal process timeline | Enforce, Legislate |

## Named Functions (6)

| Function | Composition | Purpose |
|----------|------------|---------|
| **Trial** | File + Submit (both sides) + Argue (both sides) + Judge | Full adjudication process |
| **Constitutional-Amendment** | Reform + Legislate (supermajority Consent) + Rights check | Fundamental rule change |
| **Injunction** | File + Judge (Emergency) + Enforce (temporary) | Urgent temporary measure |
| **Plea** | File + Accept (reduced penalty) + Enforce | Expedited resolution |
| **Class-Action** | File (multiple parties, Merge) + Trial | Multiple parties, one proceeding |
| **Recall** | Audit + File (against authority) + Consent (community) + role.revoked | Remove authority holder |

## Mapping to Primitives

| Operation | Layer 4 Primitives | Grammar Operations |
|-----------|-------------------|-------------------|
| Legislate | Rule | Emit |
| Amend | Rule | Derive |
| Repeal | Rule | Retract |
| File | DueProcess | Challenge |
| Submit | Precedent | Annotate |
| Argue | Interpretation | Respond |
| Judge | Adjudication | Emit |
| Appeal | Appeal | Challenge |
| Enforce | Enforcement | Delegate |
| Audit | Audit | Traverse |
| Pardon | Amnesty | Consent |
| Reform | Reform | Derive |

## Example Flow

**Community dispute resolution:**
```
File("user-X posted spam in #general, violating Rule 3.2")
  → Submit(evidence=[event-123, event-456, rule-3.2])
  → Submit(defense=[event-789, "I thought it was on-topic"])
  → Argue("Rule 3.2 applies because...", precedent=[case-42])
  → Argue("Case-42 is distinguishable because...")
  → Judge(ruling="violation confirmed, 7-day suspension",
          precedent=true, reasoning="...")
  → Enforce(suspension applied)
  -- or --
  → Appeal(grounds="due process violated, no warning given")
  → Judge(appeal: "original ruling modified, warning issued instead")
```

## Reference

- `docs/grammar.md` — Infrastructure grammar (15 operations)
- `docs/layers/04-legal.md` — Layer 4 derivation
- `docs/primitives.md` — Layer 4 primitive specifications
- `docs/tests/primitives/04-community-governance.md` — Related integration test scenario
