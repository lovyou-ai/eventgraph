# Meaning Grammar (Layer 11: Culture)

The grammar for cross-cultural communication with reflection, expression, and transmission.

## Derivation

Culture is operations on meaning itself. The base operations are: **reflect**, **express**, **transmit**, **anticipate**. Four semantic dimensions differentiate operations:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| Level | Direct (about things) / Meta (about how we see things) | First-order or reflexive? |
| Medium | Literal (plain) / Figurative (artistic, symbolic) | Said directly or through art? |
| Direction | Inward (self-examination) / Outward (sharing with others) | Looking in or projecting out? |
| Purpose | Understanding (knowing) / Creating (making) | Comprehending or generating? |

## Operations (10)

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 1 | **Examine** | reflection/self | Identify blind spots and assumptions | SelfAwareness + Emit |
| 2 | **Reframe** | reflection/perspective | See a situation from a different viewpoint | Perspective + Derive (reframed) |
| 3 | **Question** | reflection/critique | Challenge what's taken for granted | Critique + Challenge |
| 4 | **Distill** | reflection/wisdom | Extract what truly matters from experience | Wisdom + Derive (from experience chain) |
| 5 | **Beautify** | expression/aesthetic | Recognize or create beauty and elegance | Aesthetic + Annotate or Emit |
| 6 | **Liken** | expression/metaphor | Explain one thing in terms of another | Metaphor + Derive |
| 7 | **Lighten** | expression/humour | Find incongruity and playfulness | Humour + Emit |
| 8 | **Teach** | transmission/share | Deliberately transfer knowledge | Teaching + Channel |
| 9 | **Translate** | transmission/bridge | Make meaning accessible across boundaries | Translation + Derive (adapted) |
| 10 | **Prophesy** | transmission/anticipate | Extrapolate where current patterns lead | Prophecy + Emit |

Note: **Silence** (the absence primitive) has no operation — it's detected, not performed. It manifests as the meaningful gap between other operations.

## Modifiers (2)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| **Cross-Cultural** | Explicitly bridges between different communities | Translate, Teach, Reframe |
| **Archival** | Preserved for future generations | Distill, Teach, Beautify |

## Named Functions (5)

| Function | Composition | Purpose |
|----------|------------|---------|
| **Post-Mortem** | Examine + Question + Distill | Learn from failure through reflection |
| **Design-Review** | Beautify + Reframe + Question + Distill | Evaluate elegance and fitness of design |
| **Mentorship** | Teach + Reframe + Distill + Translate (to student's context) | Deep knowledge transfer |
| **Cultural-Onboarding** | Translate (Cross-Cultural) + Teach + Examine (newcomer's perspective) | Help newcomers understand implicit norms |
| **Forecast** | Prophesy + Examine (assumptions) + Distill (confidence) | Grounded prediction with stated assumptions |

## Example Flow

**System reflecting on its own culture:**
```
Examine(blind-spots=["we assume speed matters most"],
        assumptions=["more features = better"],
        limitations=["we can't evaluate our own aesthetic"])
  → Question("why do we assume more features = better?")
  → Reframe(from="feature count", to="user outcomes achieved")
  → Liken("the codebase is a garden — it needs pruning, not just planting")
  → Distill("simplicity wins: the best feature is often the one you don't build")
  → Teach(student=new-team, topic="our philosophy of simplicity")
  → Translate(Cross-Cultural: for team coming from enterprise culture,
              adaptation="simplicity doesn't mean fewer tests")
  → Prophesy("if we keep adding features at this rate,
              maintenance will exceed development in 6 months",
              confidence=0.7, basis=[velocity-trend, bug-rate-trend])
  → [meaningful Silence: no one argues with the prophecy]
```

## Reference

- `docs/grammar.md` — Infrastructure grammar (15 operations)
- `docs/layers/11-culture.md` — Layer 11 derivation
- `docs/primitives.md` — Layer 11 primitive specifications
