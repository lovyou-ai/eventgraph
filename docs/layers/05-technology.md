# Layer 5: Technology

## Derivation

### The Gap

Layer 4 formalises rules and adjudication, but it governs — it doesn't build. There's no concept of creating tools, artefacts, or systems. An actor can follow rules but cannot create something new that extends the system's capabilities.

**Test:** Can you express "Alice built a tool that automates the review process, version 2 improves on version 1, and it should be deprecated when version 3 ships" in Layer 4? You can enact rules about tools and adjudicate disputes about them. But "built" (creation), "improves on" (iteration), "automates" (process transformation), and "deprecated" (artefact lifecycle) have no Layer 4 representation. Governing is not building.

### The Transition

**Governing → Building**

An actor that can only follow rules becomes one that can create new things. The fundamental new capacity: making artefacts, defining processes, and improving both through feedback cycles.

### Base Operations

What can a builder DO that a rule-follower cannot?

1. **Create** — make a new artefact
2. **Define process** — establish repeatable workflows
3. **Test** — verify that things work
4. **Improve** — make things better through iteration

### Semantic Dimensions

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| **Object** | Artefact (thing made) / Process (way of making) | Is this about what's built or how it's built? |
| **Phase** | Genesis (creation) / Operation (use) / Retirement (end-of-life) | Where in the lifecycle? |
| **Direction** | Forward (making new) / Backward (assessing existing) | Building or evaluating? |
| **Agency** | Manual (human-driven) / Mechanical (automated) | Who does the work? |

### Decomposition

**Group 0 — Artefact** (things made)

| Primitive | Object | Phase | Direction | Agency | What it does |
|-----------|--------|-------|-----------|--------|--------------|
| **Create** | Artefact | Genesis | Forward | Manual | Making new things |
| **Tool** | Artefact | Operation | Forward | Both | Artefacts that extend capabilities |
| **Quality** | Artefact | Operation | Backward | Both | Assessing how well something was made |
| **Deprecation** | Artefact | Retirement | Backward | Both | When artefacts should no longer be used |

Artefact lifecycle: things are created (Create) → some become tools (Tool) → quality is assessed (Quality) → eventually they're retired (Deprecation). This is the lifecycle of every built thing.

**Group 1 — Process** (how things are made)

| Primitive | Object | Phase | Direction | Agency | What it does |
|-----------|--------|-------|-----------|--------|--------------|
| **Workflow** | Process | Genesis | Forward | Manual | Defining repeatable processes |
| **Automation** | Process | Operation | Forward | Mechanical | Converting manual workflows to mechanical ones |
| **Testing** | Process | Operation | Backward | Both | Verifying artefacts and processes work correctly |
| **Review** | Process | Operation | Backward | Manual | Peer assessment of artefacts and decisions |

Process lifecycle: workflows are defined (Workflow) → repetitive parts are automated (Automation) → everything is verified (Testing) → and peer-assessed (Review). Automation is key to SELF-EVOLVE — identifying patterns that can migrate from intelligent to mechanical.

**Group 2 — Improvement** (making things better)

| Primitive | Object | Phase | Direction | Agency | What it does |
|-----------|--------|-------|-----------|--------|--------------|
| **Feedback** | Both | Operation | Backward | Manual | Structured input on outcomes |
| **Iteration** | Both | Operation | Forward | Both | Improving through repeated cycles |
| **Innovation** | Artefact | Genesis | Forward | Both | Creating something genuinely new |
| **Legacy** | Artefact | Retirement | Backward | Both | What persists after deprecation |

Improvement lifecycle: feedback is gathered (Feedback) → improvements are made iteratively (Iteration) → occasionally something genuinely new emerges (Innovation) → and what was learned persists (Legacy). Legacy ensures knowledge isn't lost when artefacts are deprecated.

### Gap Analysis

| Behavior | Maps to | Notes |
|----------|---------|-------|
| "I wrote a new module" | Create | |
| "This library provides authentication" | Tool | |
| "Code quality score: 87/100" | Quality | |
| "This API is deprecated, use v2 instead" | Deprecation | |
| "The CI pipeline has three stages" | Workflow | |
| "Auto-format on save" | Automation | |
| "All tests pass" | Testing | |
| "PR approved by two reviewers" | Review | |
| "Users report the UI is confusing" | Feedback | |
| "Sprint 3 improved load time by 40%" | Iteration | |
| "This is a novel approach to caching" | Innovation | |
| "Lessons from the deprecated v1 system" | Legacy | |
| Continuous integration pipeline | Workflow + Testing + Automation | Composition |
| Technical debt tracking | Quality + Deprecation + Feedback | Composition |
| Open source contribution | Create + Review + L2.Offer | Cross-layer composition |

**No gaps found.**

### Completeness Argument

1. **Dimensional coverage:** The {object, phase, direction, agency} space is covered. The three lifecycle phases (genesis, operation, retirement) each have both forward and backward primitives.

2. **Engineering coverage:** The build-measure-learn cycle from lean methodology maps directly: Create/Workflow (build) → Testing/Quality/Feedback (measure) → Iteration/Innovation (learn). Automation captures the mechanical-to-intelligent transition that's central to SELF-EVOLVE.

3. **Layer boundary:** None of these require concepts from Layer 6 (Information). Technology builds concrete artefacts and processes — symbolic representation, abstraction, and meaning independent of physical events are Layer 6's gap.

---

## Primitive Specifications

Full specifications in `docs/primitives.md` (Layer 5 section).

## Product Graph

Layer 5 maps to the **Build Graph** — development and CI/CD with provenance. See `docs/product-layers.md`.

## Reference

- `docs/derivation-method.md` — The derivation method
- `docs/primitives.md` — Full specifications
- `docs/product-layers.md` — Build Graph
- `docs/tests/primitives/01-agent-audit-trail.md` — Related integration test scenario
