# Layer 4: Legal

## Derivation

### The Gap

Layer 3 models groups with norms, roles, and sanctions. But norms are informal — they emerge from behaviour and consensus. When someone asks "what are the rules?", Layer 3 can point to conventions and taboos, but cannot produce a codified rule, a jurisdictional boundary, a formal precedent, or a right of appeal. The concept of "this rule applies here but not there" doesn't exist.

**Test:** Can you express "This rule was enacted by the governing body, applies within this jurisdiction, and overrides the conflicting norm from the neighbouring community" in Layer 3? You can have Norms, Conventions, and Sanctions. But "enacted by authority" (codification), "applies here but not there" (jurisdiction), and "overrides" (precedent and interpretation) have no Layer 3 representation. Informal governance cannot express formal law.

### The Transition

**Informal → Formal**

Emergent norms become codified rules. The fundamental new capacity: explicit, authoritative, jurisdictionally scoped rules with formal processes for interpretation, adjudication, and reform.

### Base Operations

What can a formal legal system DO that informal norms cannot?

1. **Codify** — write rules with explicit conditions and consequences
2. **Adjudicate** — resolve disputes through formal process
3. **Enforce** — take action when rules are broken
4. **Reform** — change rules based on experience and precedent

### Semantic Dimensions

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| **Function** | Prescriptive (what the rule says) / Procedural (how disputes are handled) / Corrective (what happens after) | What phase of the legal lifecycle? |
| **Authority** | Constitutive (creates the framework) / Operative (works within it) | Does this create the system or operate within it? |
| **Temporality** | Prospective (forward-looking) / Retrospective (backward-looking) | Does this look forward to prevent or backward to assess? |
| **Scope** | Universal (applies to all) / Particular (applies to specific case) | General rule or specific application? |

### Decomposition

**Group 0 — Codification** (prescriptive, constitutive)

| Primitive | Function | Authority | Temporality | Scope | What it does |
|-----------|----------|-----------|-------------|-------|--------------|
| **Rule** | Prescriptive | Constitutive | Prospective | Universal | Formal, codified rule with conditions and consequences |
| **Jurisdiction** | Prescriptive | Constitutive | Prospective | Particular | Which rules apply where |
| **Precedent** | Prescriptive | Operative | Retrospective | Universal | Past decisions that inform future ones |
| **Interpretation** | Prescriptive | Operative | Retrospective | Particular | Applying rules to specific situations |

Codification lifecycle: rules are enacted (Rule) → scoped (Jurisdiction) → past decisions accumulate (Precedent) → rules are applied to cases (Interpretation). This is how informal norms become formal law.

**Group 1 — Process** (procedural)

| Primitive | Function | Authority | Temporality | Scope | What it does |
|-----------|----------|-----------|-------------|-------|--------------|
| **Adjudication** | Procedural | Operative | Retrospective | Particular | Formal dispute resolution |
| **Appeal** | Procedural | Operative | Retrospective | Particular | Challenging a ruling |
| **DueProcess** | Procedural | Constitutive | Prospective | Universal | Ensuring procedural fairness |
| **Rights** | Procedural | Constitutive | Prospective | Universal | Fundamental protections that override other rules |

Process lifecycle: disputes are adjudicated (Adjudication) → rulings can be challenged (Appeal) → all proceedings must be fair (DueProcess) → fundamental protections constrain the whole system (Rights). Rights violations trigger immediate authority escalation.

**Group 2 — Compliance** (corrective)

| Primitive | Function | Authority | Temporality | Scope | What it does |
|-----------|----------|-----------|-------------|-------|--------------|
| **Audit** | Corrective | Operative | Retrospective | Universal | Systematic review against rules |
| **Enforcement** | Corrective | Operative | Retrospective | Particular | Taking action when rules are broken |
| **Amnesty** | Corrective | Constitutive | Retrospective | Particular | Formal forgiveness that supersedes enforcement |
| **Reform** | Corrective | Constitutive | Prospective | Universal | Changing rules based on experience |

Compliance lifecycle: actions are reviewed (Audit) → violations are enforced (Enforcement) → or forgiven (Amnesty) → and the system learns (Reform). Reform closes the loop — experience feeds back into better rules.

### Gap Analysis

| Behavior | Maps to | Notes |
|----------|---------|-------|
| "Code of conduct, Section 3.2" | Rule | |
| "EU regulations apply to EU users" | Jurisdiction | |
| "In the last similar case, we decided X" | Precedent | |
| "This rule means Y in this context" | Interpretation | |
| "The moderator hears the dispute" | Adjudication | |
| "I appeal to the review board" | Appeal | |
| "The accused was not given a chance to respond" | DueProcess (violated) | |
| "Every member has the right to be heard" | Rights | |
| "Annual compliance review" | Audit | |
| "Your account is suspended for 30 days" | Enforcement | |
| "All prior violations forgiven under new policy" | Amnesty | |
| "We're updating the rules based on last year's issues" | Reform | |
| Constitutional amendment | Reform + Rights + Rule | Cross-group composition |
| Plea bargain | Adjudication + Enforcement + Amnesty | Composition |
| Judicial review | Appeal + Interpretation + Rights | Composition |

**No gaps found.**

### Completeness Argument

1. **Dimensional coverage:** All meaningful combinations of {function, authority, temporality, scope} are covered. The three groups naturally correspond to the three phases: writing rules (Codification), applying rules (Process), and enforcing rules (Compliance).

2. **Legal theory coverage:** The distinction between constitutive rules (that create the framework) and operative rules (that work within it) maps to Hart's primary/secondary rules. Due process and rights map to constitutional constraints on the legal system itself.

3. **Layer boundary:** None of these require concepts from Layer 5 (Technology). Legal formalises governance but doesn't build — creating tools, artefacts, and processes is Layer 5's gap.

---

## Primitive Specifications

Full specifications in `docs/primitives.md` (Layer 4 section).

## Product Graph

Layer 4 maps to the **Governance Graph** — formalised rule-making with transparent adjudication. See `docs/product-layers.md`.

## Reference

- `docs/derivation-method.md` — The derivation method
- `docs/primitives.md` — Full specifications
- `docs/product-layers.md` — Governance Graph
- `docs/tests/primitives/04-community-governance.md` — Related integration test scenario
