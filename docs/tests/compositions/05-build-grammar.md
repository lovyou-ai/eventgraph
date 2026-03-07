# Composition Test: Build Grammar (Layer 5)

Tests for the Build Grammar operations as SDK-level APIs.

## Setup

```
graph: initialized EventGraph
actors: [dev_alice (Human), ci_agent (AI), reviewer_bob (Human)]
grammar: BuildGrammar
```

## Operation Tests

### Build

**Input:** `grammar.Build({ name: "auth-service", type: "module", actor: dev_alice })`
**Assertions:**
- Create primitive activated
- Emits `artefact.created` event with version=1
- Provenance links to actor and context

### Version

**Input:** `grammar.Version({ artefact: artefact_id, changes: "add OAuth2 support" })`
**Assertions:**
- Emits `artefact.version` event caused by previous version
- Version number incremented
- Derive chain from v1 → v2

### Ship / Sunset

**Input:** `grammar.Ship({ artefact: artefact_id })` then later `grammar.Sunset({ artefact: artefact_id, replacement: v2_id })`
**Assertions:**
- Ship registers artefact as Tool (usable by other actors)
- Sunset marks deprecation with replacement link
- Breaking modifier warns dependents

### Define / Automate

**Input:** `grammar.Define({ name: "CI pipeline", steps: ["lint", "test", "build", "deploy"] })`
**Assertions:**
- Workflow primitive activated
- Steps are ordered and named
- `grammar.Automate({ workflow: workflow_id, step: 0 })` converts lint to mechanical

### Test / Review / Measure

**Input:** CI pipeline execution.
**Assertions:**
- Test emits results with coverage Score
- Review records peer assessment with approval
- Measure produces quality Score with criteria breakdown
- CI modifier triggers these automatically on change

### Feedback / Iterate / Innovate

**Input:** User feedback → iteration cycle.
**Assertions:**
- Feedback captures sentiment and specific input
- Iterate produces new version derived from previous + feedback
- Innovate detected when novelty Score is high

## Named Function Tests

### Pipeline

**Input:** `grammar.Pipeline({ artefact: artefact_id, steps: ["test", "measure", "ship"] })`
**Assertions:**
- Steps execute in order
- Each step's output is input to the next
- Failure at any step halts the pipeline
- All events causally linked

### Post-Mortem

**Input:** `grammar.PostMortem({ incident: incident_id })`
**Assertions:**
- Gathers Feedback from all involved actors
- Measures quality/process metrics
- Defines improvement actions
- All linked to the incident

## Error Cases

| Case | Expected |
|------|----------|
| Ship without passing tests | `Err(ValidationError.TestsNotPassing)` |
| Sunset without replacement (Breaking) | Warning event (not error — deprecation without replacement is valid but flagged) |
| Automate step that doesn't exist | `Err(ValidationError.StepNotFound)` |

## Reference

- `docs/compositions/05-build.md` — Build Grammar specification
- `docs/layers/05-technology.md` — Layer 5 derivation
