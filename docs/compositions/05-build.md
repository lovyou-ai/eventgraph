# Build Grammar (Layer 5: Technology)

The grammar for development, CI/CD, and artefact lifecycle with provenance.

## Derivation

Building is operations on artefacts and processes. The base operations are: **create artefact**, **define process**, **verify**, **improve**. Four semantic dimensions differentiate operations:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| Object | Artefact (thing) / Process (method) | What you make vs. how you make it |
| Lifecycle | Create / Operate / Retire | Beginning, middle, or end? |
| Feedback | None / Positive (improve) / Negative (fix) | What kind of response? |
| Automation | Manual / Automated | Human or machine? |

## Operations (12)

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 1 | **Build** | artefact/create | Create a new artefact with provenance | Create + Emit |
| 2 | **Version** | artefact/iterate | Release a new version of an artefact | Create (version) + Derive (from previous) |
| 3 | **Ship** | artefact/release | Deploy an artefact for use | Tool (registered) + Emit |
| 4 | **Sunset** | artefact/retire | Deprecate an artefact with migration path | Deprecation + Annotate (replacement) |
| 5 | **Define** | process/create | Establish a repeatable workflow | Workflow + Emit |
| 6 | **Automate** | process/optimize | Convert a manual step to mechanical | Automation + Derive (from workflow) |
| 7 | **Test** | verify/artefact | Run verification against artefact | Testing + Emit (results) |
| 8 | **Review** | verify/peer | Peer assessment of artefact or decision | Review + Respond |
| 9 | **Measure** | verify/quality | Assess quality against criteria | Quality + Annotate (scores) |
| 10 | **Feedback** | improve/input | Provide structured input on outcomes | Feedback + Respond |
| 11 | **Iterate** | improve/cycle | Improve through repeated refinement | Iteration + Derive (from previous + feedback) |
| 12 | **Innovate** | improve/breakthrough | Create something genuinely new | Innovation + Emit |

## Modifiers (3)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| **CI** | Automatically triggered on change | Test, Measure, Review |
| **Breaking** | Flags that dependents may need changes | Version, Ship, Sunset |
| **Experimental** | Marks as unstable, not for production reliance | Build, Ship, Innovate |

## Named Functions (5)

| Function | Composition | Purpose |
|----------|------------|---------|
| **Pipeline** | Define + Test + Measure + Ship (chained) | Full CI/CD flow |
| **Spike** | Build (Experimental) + Test + Feedback + decide | Exploratory prototype |
| **Migration** | Sunset + Version + Ship (replacement) + Test | Replace deprecated with new |
| **Post-Mortem** | Feedback (batch) + Measure + Define (improvements) | Learn from failure |
| **Tech-Debt** | Measure + Annotate (debt marker) + Iterate (scheduled) | Track and pay down debt |

## Mapping to Primitives

| Operation | Layer 5 Primitives | Grammar Operations |
|-----------|-------------------|-------------------|
| Build | Create | Emit |
| Version | Create | Derive |
| Ship | Tool | Emit |
| Sunset | Deprecation | Annotate |
| Define | Workflow | Emit |
| Automate | Automation | Derive |
| Test | Testing | Emit |
| Review | Review | Respond |
| Measure | Quality | Annotate |
| Feedback | Feedback | Respond |
| Iterate | Iteration | Derive |
| Innovate | Innovation | Emit |

## Example Flow

**Feature development:**
```
Build("auth-service v1", provenance=[spec-doc, design-review])
  → Define("auth-service CI pipeline: lint → test → build → deploy")
  → Test(unit=passing, integration=passing, coverage=87%)
  → Review(reviewer=alice, approved=true, comments="clean impl")
  → Measure(quality=0.91, criteria={security: 0.95, perf: 0.88})
  → Ship(deployed to staging)
  → Feedback("login latency too high on mobile")
  → Iterate(v1.1: "add connection pooling", delta=+0.12 perf)
  → Test(CI: all passing) → Ship(production)
  → Version(v2: "add OAuth2 support", Breaking)
  → Sunset(v1, replacement=v2, deadline="2026-06-01")
```

## Reference

- `docs/grammar.md` — Infrastructure grammar (15 operations)
- `docs/layers/05-technology.md` — Layer 5 derivation
- `docs/primitives.md` — Layer 5 primitive specifications
- `docs/tests/primitives/06-research-integrity.md` — Related integration test scenario
