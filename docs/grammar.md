# Social Grammar

The semantic graph grammar for social interaction. Derived in Post 35 of the LovYou series.

This is a **product layer** specification — these operations are compositions of Layer 0 primitives, designed for building social interfaces on the event graph.

## Derivation

Social interactions are operations on a graph. Graph theory gives us vertices and edges but is content-agnostic (doesn't model what an edge means) and time-agnostic (doesn't model whether a vertex is permanent). Six semantic dimensions extend graph theory into the social domain:

| Dimension | Values |
|-----------|--------|
| Causality | Independent / Dependent (responsive, divergent, sequential) |
| Content | Content-bearing / Structural-only |
| Temporality | Persistent / Transient |
| Visibility | Public / Private |
| Direction | Centripetal (toward content) / Centrifugal (into actor's subgraph) |
| Authorship | Same actor / Different actor / Mutual |

## Operations (15)

| # | Operation | Type | Definition |
|---|-----------|------|-----------|
| 1 | Emit | vertex/creative | Create independent content |
| 2 | Respond | vertex/creative | Create causally dependent, subordinate content |
| 3 | Derive | vertex/creative | Create causally dependent, independent content |
| 4 | Extend | vertex/creative | Create sequential content, same author |
| 5 | Retract | vertex/destructive | Tombstone own content (provenance survives) |
| 6 | Annotate | vertex/parasitic | Attach metadata to existing content |
| 7 | Acknowledge | edge/constructive | Content-free edge toward vertex (centripetal) |
| 8 | Propagate | edge/constructive | Redistribute vertex into actor's subgraph (centrifugal) |
| 9 | Endorse | edge/constructive | Reputation-staked edge toward vertex |
| 10 | Subscribe | edge/constructive | Persistent, future-oriented edge to actor |
| 11 | Channel | edge/constructive | Private, bidirectional, content-bearing edge |
| 12 | Delegate | edge/meta | Grant authority for another actor to operate as you |
| 13 | Consent | bilateral | Mutual, atomic, dual-signed event |
| 14 | Sever | edge/destructive | Remove subscription, channel, or delegation |
| 15 | Merge | vertex/convergent | Join two independent subtrees |

Plus: **Traverse** (read-only navigation/distance measurement)

## Modifiers (3)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| Transient | Vertex self-tombstones after TTL | Any vertex operation |
| Nascent | Flags for discovery (actor has low centrality) | Any vertex operation |
| Conditional | Executes when graph condition is met | Any operation |

## Named Functions (8)

Compositions of the 15 operations:

| Function | Composition | Purpose |
|----------|------------|---------|
| Recommend | Propagate + Channel | Directed sharing to specific person |
| Challenge | Respond + dispute flag | Formal dispute that follows content |
| Curate | Emit + reference edges | Organise existing content into collections |
| Collaborate | Consent + Emit | Co-authorship |
| Forgive | Subscribe after Sever | Reconciliation with history intact |
| Invite | Endorse + Subscribe | Trust-staked introduction of new actor |
| Memorial | Actor permanence modifier | Preserve graph when actor can no longer operate |
| Transfer | Delegate + authority reassignment | Change ownership of content or subgraph |

## Mapping to Layer 0

Each social grammar operation maps to Layer 0 primitives:

| Operation | Layer 0 Primitives |
|---|---|
| **Emit** | EventStore (append), Signature (sign), Hash (chain), CausalLink (causes) |
| **Respond** | EventStore + CausalLink (cause = parent event), Ancestry (thread) |
| **Derive** | EventStore + CausalLink (cause = source event, but independent content) |
| **Extend** | EventStore + CausalLink (cause = own previous, same source) |
| **Retract** | EventStore (tombstone event), CausalLink (cause = retracted event) |
| **Annotate** | EventStore + Edge (EdgeType.Annotation to target), Annotate primitive |
| **Acknowledge** | Edge (EdgeType.Endorsement, weight=0, centripetal), no content |
| **Propagate** | Edge (EdgeType.Reference, centrifugal), SubgraphExtract |
| **Endorse** | Edge (EdgeType.Endorsement, weighted) + TrustScore (stake) + Evidence |
| **Subscribe** | Edge (EdgeType.Subscription, persistent) |
| **Channel** | Edge (EdgeType.Channel, bidirectional) + visibility constraint |
| **Delegate** | Edge (EdgeType.Delegation, scoped) + Authority (grant) |
| **Consent** | Authority (bilateral) + dual Signature + Verify |
| **Sever** | Edge supersession (edge.superseded event) |
| **Merge** | EventStore + CausalLink (multiple causes from independent subtrees) |
| **Traverse** | PathQuery + SubgraphExtract + Timeline (read-only) |

## Interface Lenses

The grammar is interface-agnostic. Different UIs render the same operations differently:

- **Garden** (social): Emissions as plants, Acknowledgments as sunlight, Derivations as branches
- **Politics Page** (governance): Emissions as proposals, Responses as debate, Consents as votes
- **Market** (exchange): Emissions as listings, Channels as negotiations, Consents as transactions

Same grammar. Same graph. Different lens.

## Reference

Full derivation: [The Missing Social Grammar](https://mattsearles2.substack.com) (Post 35)
- `docs/compositions/` — Per-layer composition grammars applying this same method to every product graph
