# Bond Grammar (Layer 9: Relationship)

The grammar for deep relational bonds with repair and intimacy.

## Derivation

Relationships are operations on the space between two actors. The base operations are: **connect**, **deepen**, **break**, **repair**. Four semantic dimensions differentiate operations:

| Dimension | Values | What it distinguishes |
|-----------|--------|-----------------------|
| State | Forming / Stable / Ruptured / Resolved | Where in the relationship lifecycle? |
| Depth | Surface (functional) / Deep (transformative) | Does this touch who the participants are? |
| Symmetry | Mutual (both equally) / Unilateral (one-sided) | Both parties or just one? |
| Valence | Positive (strengthening) / Negative (weakening) | Building or eroding? |

## Operations (10)

| # | Operation | Type | Definition | Primitives |
|---|-----------|------|-----------|------------|
| 1 | **Connect** | bond/form | Initiate a relational bond | Attachment + Subscribe (mutual) |
| 2 | **Balance** | bond/maintain | Assess and adjust reciprocity | Reciprocity + Annotate |
| 3 | **Deepen** | bond/trust | Extend relational trust beyond transactional | Trust (Relational) + Consent |
| 4 | **Open** | intimacy/share | Share vulnerability with another | Vulnerability + Channel (Transient) |
| 5 | **Attune** | intimacy/perceive | Develop accurate understanding of another | Understanding + Emit |
| 6 | **Feel-With** | intimacy/respond | Express empathy for another's state | Empathy + Respond |
| 7 | **Break** | rupture/detect | Acknowledge a relational rupture | Rupture + Emit |
| 8 | **Apologize** | repair/initiate | Acknowledge harm and take responsibility | Apology + Emit |
| 9 | **Reconcile** | repair/rebuild | Rebuild relationship after rupture | Reconciliation + Consent |
| 10 | **Mourn** | loss/process | Process the permanent end of a relationship | Loss + Emit |

## Modifiers (2)

| Modifier | Effect | Applies to |
|----------|--------|-----------|
| **Witnessed** | A third party observes and can attest | Apologize, Reconcile, Deepen |
| **Private** | Visible only to the two participants | Open, Feel-With, Break |

## Named Functions (5)

| Function | Composition | Purpose |
|----------|------------|---------|
| **Betrayal-Repair** | Break + Apologize + Reconcile + Deepen | Full rupture-to-growth cycle |
| **Check-In** | Balance + Attune + Feel-With | Regular relationship health assessment |
| **Forgive** | Break (acknowledged) + Subscribe (after Sever) | Re-establish connection with history intact |
| **Mentorship** | Connect + Deepen + Attune + Teaching (L11) | Deep developmental relationship |
| **Farewell** | Mourn + Memorialize (L8) + Gratitude (L2) | Honoring a relationship that's ending |

## Example Flow

**Relationship rupture and repair:**
```
Connect(alice, bob, context="project collaborators")
  → Balance(reciprocity=0.82, "balanced contribution")
  → Deepen(relational-trust: alice trusts bob with code review autonomy)
  → Open(alice shares: "I'm struggling with imposter syndrome")
  → Attune(bob: accuracy=0.78, "alice needs affirmation, not advice")
  → [bob publicly criticizes alice's code without private discussion first]
  → Break(cause="public criticism violated trust",
          severity=high, repairable=true)
  → Apologize(bob→alice, "I should have talked to you privately first")
  → Reconcile(progress=0.6 → 0.8 → 0.95,
              new-basis="always discuss privately before public feedback")
  → Deepen(relational-trust: higher than before rupture)
```

## Reference

- `docs/grammar.md` — Infrastructure grammar (15 operations)
- `docs/layers/09-relationship.md` — Layer 9 derivation
- `docs/primitives.md` — Layer 9 primitive specifications
- `docs/tests/primitives/03-consent-journal.md` — Related integration test scenario
