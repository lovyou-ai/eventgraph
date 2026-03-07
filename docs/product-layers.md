# Product Layers

Each of the 13 layers above Foundation maps to a **product graph** — a domain-specific application of the event graph infrastructure. Product graphs are lenses on the same substrate: same events, same hash chains, same trust model, different intelligence.

Layer 0 + the social grammar can EXPRESS all of these use cases. The higher layers add domain-specific INTELLIGENCE — primitives that understand patterns within that domain. A community governance platform works at Layer 0 (events, trust, authority). Layer 10 (Community) adds primitives that understand belonging gradients, welcome/exile processes, and community health patterns. The infrastructure records; the intelligence reasons.

## The 13 Product Graphs

### Layer 1: Work Graph (Agency)

**What it adds:** Observer becomes participant. Action, intention, commitment, completion.

**Product:** Task management where AI agents and humans are on the same graph. Work is recorded as events — not planned as tickets. A "company-in-a-box" for solo founders: Claude + event graph = accountable AI workforce.

**Key event flows:**
- Task decomposition: Emit (task) → Derive (subtasks) → Delegate (to agent/human) → Extend (progress) → Emit (completion)
- Agent accountability: Every agent decision is an event with causes, confidence, and authority chain
- Handoff: Delegate + Channel between human and AI agent, with authority scoping what the agent can do autonomously

**Intelligence primitives would add:**
- Workload balancing across agents
- Deadline risk detection from historical patterns
- Automatic task decomposition based on prior similar work
- Model-tier routing (simple tasks → small model, complex → large model)

**Use cases served:** AI Agent Audit Trail, Company-in-a-box, AI Agent Framework

---

### Layer 2: Market Graph (Exchange)

**What it adds:** Individual becomes dyad. Negotiation, exchange, value, fairness.

**Product:** Trust-based marketplace eliminating platform tolls. Portable reputation means a freelancer's 500-task history at 98% approval follows them everywhere. Escrow as event patterns. Smart contracts as readable agreements on hash chains.

**Key event flows:**
- Listing: Emit (offer) → Subscribe (interested parties) → Channel (negotiation) → Consent (agreement) → Emit (delivery) → Acknowledge (receipt) → Endorse (reputation)
- Escrow: Consent (terms) → Delegate (funds to escrow actor) → Emit (delivery) → Consent (release) → Transfer
- Dispute: Challenge (dispute flag) → authority.requested → authority.resolved → trust.updated

**Intelligence primitives would add:**
- Fair price detection from market patterns
- Fraud pattern recognition
- Reputation portability scoring
- Exchange reciprocity analysis

**Use cases served:** Freelancer Reputation, AI Agent Marketplace, Supply Chain Transparency

---

### Layer 3: Social Graph (Society)

**What it adds:** Dyad becomes group. Norms, roles, inclusion, exclusion.

**Product:** User-owned social platform where communities set their own norms. Users control what queries are visible. Feed is a lens on events, not an algorithm's selection. Communities are subgraphs with governance.

**Key event flows:**
- Content creation: Emit → Respond/Derive (conversation trees) → Acknowledge/Endorse (engagement)
- Community norms: Emit (norm proposal) → Respond (discussion) → Consent (adoption) → Annotate (norm applied to content)
- Moderation: violation.detected → authority.requested → authority.resolved → Retract or actor.suspended
- Discovery: Nascent modifier on Emit → Propagate by established members → trust accumulation

**Intelligence primitives would add:**
- Norm violation detection (semantic, not just keyword)
- Community health metrics
- Bridging capital measurement (who connects separate clusters)
- Echo chamber detection

**Use cases served:** Consent-Based Journal, Creator Provenance, Family Decision Log

---

### Layer 4: Justice Graph (Legal)

**What it adds:** Informal becomes formal. Evidence, adjudication, precedent, enforcement.

**Product:** Dispute resolution platform where evidence already exists because interactions were on the graph. Tiered adjudication: automatic (clear-cut) → AI arbitration (pattern matching) → human judgment (complex) → courts (last resort). Makes $500 disputes economically solvable.

**Key event flows:**
- Dispute initiation: Challenge (dispute flag on event) → Emit (claim with evidence chain) → authority.requested (adjudication)
- Evidence assembly: Traverse (causal ancestors of disputed event) → SubgraphExtract → Annotate (relevance)
- Arbitration: IDecisionMaker evaluates evidence chain → Decision with confidence → authority.resolved
- Precedent: Derive (ruling references prior rulings) → future disputes Traverse precedent chain

**Intelligence primitives would add:**
- Precedent matching (similar disputes → similar resolutions)
- Evidence relevance scoring
- Bias detection in adjudication patterns
- Jurisdictional routing

**Use cases served:** Dispute Resolution, Evidence-as-a-Service, Transparent Hiring

---

### Layer 5: Research Graph (Technology)

**What it adds:** Governing becomes building. Hypothesis, method, reproducibility, knowledge.

**Product:** Research integrity infrastructure. Pre-registration as structural property: hypothesis is hash-chained BEFORE experiment. Analysis history visible — not just the successful final run. Peer review as graph operations. mind-zero is the first project.

**Key event flows:**
- Pre-registration: Emit (hypothesis, hash-chained with timestamp proof) → Extend (methodology) → Emit (data collection protocol)
- Experiment: Emit (raw data) → Derive (analysis, causes = data + methodology) → Derive (results, causes = analysis)
- Failed attempts visible: Every analysis run is an event, not just the one that worked
- Peer review: Respond (review) → Endorse or Challenge → Annotate (revisions) → Consent (publication)
- Replication: Derive (replication study, causes = original) → Compare results

**Intelligence primitives would add:**
- Statistical validity checking
- Methodology gap detection
- Cross-study pattern synthesis
- Citation integrity verification

**Use cases served:** Research Integrity, Environmental Monitoring, mind-zero

---

### Layer 6: Knowledge Graph (Information)

**What it adds:** Physical becomes symbolic. Claims, evidence, provenance, contradiction.

**Product:** Claims as events with evidence chains. Challenges coexist with assertions — you don't delete the wrong answer, you record the correction with causal links to the evidence. Source reputation derived from track record. AI content structurally distinguishable by absent creative chains.

**Key event flows:**
- Knowledge claim: Emit (claim) → Annotate (evidence links) → Endorse (expert support) → trust.updated on claim author
- Challenge: Challenge (counter-evidence) → Respond (rebuttal or concession) → Merge (synthesis)
- Provenance: Derive chain shows where knowledge came from → Traverse to original source
- AI detection: Human creative work has rich Derive chains (inspiration → drafts → revision). AI output has a single Emit.

**Intelligence primitives would add:**
- Contradiction detection across knowledge domains
- Source reliability scoring
- Information decay tracking (outdated claims)
- Semantic similarity for duplicate detection

**Use cases served:** Personal Knowledge Graph, Creator Provenance

---

### Layer 7: Ethics Graph (Ethics)

**What it adds:** Is becomes ought. Values, constraints, harm, accountability.

**Product:** AI accountability infrastructure. Every AI decision is visible in real-time: what was decided, what values constrained it, what authority approved it, what confidence level applied. Harm detection across layers via pattern recognition.

**Key event flows:**
- Decision audit: IDecisionMaker.Decide() → Decision event with confidence, authority chain, trust weights → Receipt (cryptographic proof)
- Harm detection: Pattern primitive detects harm signal → violation.detected → authority.requested (escalation)
- Value constraint: Decision tree encodes ethical constraints → Semantic conditions evaluate edge cases → evolution tracks which constraints are triggered
- Accountability chain: Traverse from harm event → through causal ancestors → to authorising decision → to approving human

**Intelligence primitives would add:**
- Cross-domain harm pattern detection
- Ethical dilemma classification
- Accountability gap identification
- Value drift detection over time

**Use cases served:** Enterprise AI Accountability, AI Agent Audit Trail, Financial Market Accountability

---

### Layer 8: Identity Graph (Identity)

**What it adds:** Doing becomes being. Self-sovereignty, selective disclosure, digital death.

**Product:** Identity that emerges from verifiable action history, not self-reported claims. Selective disclosure: share proof of credential without revealing the credential. AI agent identity through authority chains — an agent's identity IS its decision history. Digital death protocols preserve the graph while marking the actor as Memorial.

**Key event flows:**
- Identity accumulation: Actor's events form identity over time → Traverse shows who they are by what they've done
- Selective disclosure: Emit (proof of property) with zero-knowledge link → Verifier checks proof without seeing underlying events
- AI identity: Agent registered with ActorType.AI → all decisions recorded → identity = decision pattern
- Digital death: actor.memorial → Memorial modifier → graph preserved, actor can no longer emit

**Intelligence primitives would add:**
- Identity coherence verification
- Impersonation detection
- Credential validity tracking
- Identity evolution patterns

**Unresolved tension:** Right to be forgotten vs append-only graph. Potential resolution: event content can be encrypted with a key that's destroyed, but the hash chain (structural proof) survives.

**Use cases served:** Cross-Border Identity, Universal Identity, Transparent Hiring

---

### Layer 9: Relationship Graph (Relationship)

**What it adds:** Self becomes self-with-other. Vulnerability, attunement, betrayal, repair, forgiveness.

**Product:** The LovYou origin. Consent as continuous architecture, not one-time checkbox. Betrayal and repair as primitives — the system understands that relationships break and can be repaired, and that the repair history matters. Privacy-first: relationship data visible only to participants.

**Key event flows:**
- Connection: Subscribe (mutual) → Channel (private communication) → trust accumulation over interactions
- Vulnerability: Channel + Transient modifier (ephemeral sharing, proves willingness to be vulnerable without permanent record)
- Betrayal/repair: violation.detected → trust.updated (sharp drop) → Sever → time passes → Forgive (Subscribe after Sever, history intact)
- Reciprocity: Pattern analysis across Channel events — is communication balanced? Is one party always initiating?
- Consent: Every shared event requires bilateral Consent. Not "I agreed once" but "I agree to this specific thing now."

**Intelligence primitives would add:**
- Reciprocity pattern detection
- Communication health scoring
- Domestic violence early warning (escalating control patterns)
- Repair cycle recognition

**Use cases served:** Relationship Health Platform, Consent-Based Journal, Dating Infrastructure

---

### Layer 10: Community Graph (Community)

**What it adds:** Relationship becomes belonging. Welcome, exile, norms, collective memory.

**Product:** Living community systems with portable memory. Belonging as gradient (not binary member/non-member). Welcome and exile as structured processes with transparency. Nested fractal communities — a neighbourhood within a city within a region.

**Key event flows:**
- Welcome: Invite (Endorse + Subscribe from existing member) → newcomer Emits → community Acknowledges → trust accumulates → belonging gradient increases
- Exile: violation.detected → authority.requested → Respond (community discussion) → Consent (community decision) → Sever → actor excluded from community subgraph
- Norms: Emit (norm) → Consent (adoption) → norms apply to community subgraph → violation.detected when breached
- Memory: Community events persist even when members leave → Traverse shows community history → new members can learn from past
- Fractal nesting: Community A is subgraph of Community B → norms cascade downward → authority delegated upward

**Intelligence primitives would add:**
- Community health metrics (growth/death spiral detection)
- Welcome effectiveness tracking
- Norm evolution patterns
- Bridging vs bonding capital measurement

**Use cases served:** Community Governance, Language Preservation, Online Community Health

---

### Layer 11: Governance Graph (Governance)

**What it adds:** Living becomes seeing. Transparency, accountability, power visibility.

**Product:** Makes visible: who decides, why, and whom it affects. Rules and enforcement on the same graph — you can Traverse from "this rule was enforced on me" to "this is who made the rule" to "this is what process was followed." Applies to democratic, corporate, platform, and AI governance equally.

**Key event flows:**
- Proposal: Emit (policy proposal) → Respond (debate) → Annotate (amendments) → Consent (vote) → Derive (enacted policy)
- Decision transparency: authority.requested → Decision with full chain → authority.resolved → all visible to governed actors
- Budget: Emit (allocation) → Derive (expenditure, causes = allocation) → Traverse shows where money went
- Lobbying: If lobbying interactions are events → Channel between lobbyist and decision-maker → Traverse shows influence on decisions
- Recall: violation.detected on governor → authority.requested (recall process) → Consent (community vote) → actor.suspended

**Intelligence primitives would add:**
- Corruption pattern detection
- Nepotism graph analysis
- Policy impact prediction
- Power concentration monitoring

**Use cases served:** Transparent Governance SaaS, City-Scale Dashboard, Open Governance Standard

---

### Layer 12: Culture Graph (Culture)

**What it adds:** Content becomes architecture. Meaning, tradition, ritual, preservation.

**Product:** Preserves provenance of meaning across time. When a cultural practice is recorded as events, the chain of transmission (teacher → student → student's student) is visible. Digital ritual — collective synchronous events that create shared meaning. The Sacred primitive is explicitly beyond optimisation.

**Key event flows:**
- Tradition transmission: Emit (cultural practice) → Derive (adaptation by next generation, causes = original) → chain shows evolution while preserving provenance
- Language preservation: Emit (linguistic record) → Annotate (context, usage, nuance) → Endorse (elder verification) → Subscribe (learner)
- Creative provenance: Derive chain from inspiration through drafts to final work → Traverse shows creative process → distinguishable from AI generation
- Ritual: Conditional modifier (executes at specific time) + multiple simultaneous Emits from community members → Merge (shared experience)

**Intelligence primitives would add:**
- Cultural drift detection
- Meaning preservation scoring
- Tradition continuity verification
- Creative authenticity analysis

**Use cases served:** Language Preservation, Creator Provenance, Ecological Commons

---

### Layer 13: Existence Graph (Existence)

**What it adds:** Everything becomes the fact of everything. Ecology, sustainability, existential accounting.

**Product:** Models ecosystems using event graph primitives. Economic output and ecological cost on the same graph — externalisation structurally impossible when both accounts are linked. The cascade reversed: functioning infrastructure across all thirteen layers creates conditions for human flourishing.

**Key event flows:**
- True cost: Emit (economic output) + Emit (ecological cost, causes = same operation) → Traverse shows both sides → no externalisation
- Sustainability: Annotate (environmental impact on production events) → aggregate across supply chain → visible true cost
- Ecological commons: EGIP between environmental monitoring systems → trust accumulation → cross-system ecological view

**Intelligence primitives would add:**
- Ecological tipping point detection
- True cost calculation
- Sustainability trajectory modelling
- Cross-system environmental correlation

**Use cases served:** Ecological Commons, Post-Scarcity Coordination, Environmental Monitoring

---

## Cross-Cutting Patterns

Several patterns appear across multiple product graphs:

### Portable Reputation
Trust scores accumulated in one product graph are visible (with appropriate authority) in others. A freelancer's Market Graph reputation informs their Social Graph standing. An AI agent's Work Graph history IS its Identity Graph.

### Tiered Adjudication
Justice Graph pattern that applies everywhere: automatic (clear-cut, decision tree handles it) → AI arbitration (Semantic conditions, IIntelligence) → human judgment (authority.requested, Required level) → formal process (external).

### Perverse Incentive Detection
When a system's rules create unintended incentives, the event graph makes the pattern visible. The reward event causes behaviour events that cause harm events — the causal chain shows the perverse incentive. Intelligence primitives at Layer 7+ detect these patterns.

### Lenses
The same event graph supports multiple simultaneous product layers. A single event might be visible in the Work Graph (task completed), Market Graph (payment triggered), Social Graph (reputation updated), and Governance Graph (budget spent). Different UIs render different lenses on the same substrate.

---

## Implementation Priority

Product layers build on each other. Recommended order:

1. **Work Graph** (Layer 1) — Most immediate value. Solo founders + AI agents. Lovatts deployment.
2. **Market Graph** (Layer 2) — Natural extension of Work Graph. Freelancer economy.
3. **Social Graph** (Layer 3) — Requires community features. Builds on Work + Market trust.
4. **Justice Graph** (Layer 4) — Requires dispute resolution. Builds on evidence from layers 1-3.
5. **Research Graph** (Layer 5) — mind-zero is the first project. Can develop in parallel.
6. **Knowledge Graph** (Layer 6) — Builds on Research Graph provenance patterns.
7. **Ethics Graph** (Layer 7) — Requires patterns from all lower layers to detect harm.
8. **Identity Graph** (Layer 8) — Emerges from accumulated history across all lower layers.
9. **Relationship Graph** (Layer 9) — The LovYou origin. Can develop in parallel with layers 1-3.
10. **Community Graph** (Layer 10) — Builds on Relationship + Social patterns.
11. **Governance Graph** (Layer 11) — Requires Community + Justice foundations.
12. **Culture Graph** (Layer 12) — Requires Community + Knowledge foundations.
13. **Existence Graph** (Layer 13) — Requires everything below to be functioning.

---

## Reference

- `docs/grammar.md` — The 15 social grammar operations that compose all product layer interactions
- `docs/compositions/` — Per-layer composition grammars (Work, Market, Justice, etc.)
- `docs/interfaces.md` — Core type system and interfaces
- `docs/primitives.md` — All 201 primitives across 14 layers
- `docs/layers/` — Per-layer primitive derivations
- `docs/tests/primitives/` — Infrastructure-level example scenarios
- [What You Could Build](https://mattsearles2.substack.com/p/what-you-could-build) — 34 use cases across scales
- [The Map Complete](https://mattsearles2.substack.com/p/the-map-complete) — The 13 product graphs
