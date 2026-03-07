# Scenario 5: Supply Chain Transparency

A product moves from raw material supplier through manufacturer to retailer. Each entity runs its own sovereign event graph. They communicate via EGIP — signed envelopes, cross-graph event references, and bilateral treaties. A consumer can trace the product's complete provenance across system boundaries.

**Product graph:** Work Graph (Layer 1)

---

## Actors and Systems

### System A: Farm (supplier)
| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `farmer_emma` | Human | 0.8 | Farm operator |
| `farm_system` | System | 1.0 | Farm's event graph |

### System B: Factory (manufacturer)
| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `factory_mgr` | Human | 0.7 | Factory manager |
| `qa_agent` | AI | 0.6 | Quality assurance AI |
| `factory_system` | System | 1.0 | Factory's event graph |

### System C: Retailer
| Actor | Type | Trust | Role |
|-------|------|-------|------|
| `retailer_frank` | Human | 0.75 | Retailer |
| `retail_system` | System | 1.0 | Retailer's event graph |

## Setup

- Each system has its own bootstrapped event graph with its own hash chain
- System A and System B have established a treaty (EGIP)
- System B and System C have established a treaty (EGIP)
- Inter-system trust: A↔B at Score(0.6), B↔C at Score(0.5)

## Event Sequence

### Step 1: Treaty between Farm and Factory
**EGIP:** Treaty lifecycle

```
// System A sends treaty proposal
Envelope {
    From:    "farm.example.com"
    To:      "factory.example.com"
    Type:    Treaty
    Payload: TreatyPayload {
        TreatyID:  treaty_AB
        Action:    Propose
        Terms: [
            TreatyTerm {
                Scope:     "produce_supply"
                Policy:    "Farm provides organic produce with harvest records. Factory provides processing records."
                Symmetric: false
            }
        ]
    }
}

// System B accepts
Envelope {
    From:    "factory.example.com"
    To:      "farm.example.com"
    Type:    Treaty
    Payload: TreatyPayload {
        TreatyID:  treaty_AB
        Action:    Accept
    }
}
```

Both systems record `egip.treaty.proposed` and `egip.treaty.active` events on their local graphs.

### Step 2: Farm records harvest
**Grammar:** Emit (on System A's graph)

```
// On System A's graph
Event {
    Type:    "produce.harvested"
    Source:  farmer_emma
    Content: ProduceHarvestedContent {
        Product:   "Organic Tomatoes"
        Quantity:  500     // kg
        Location:  "Farm A, Plot 7"
        Method:    "organic, no pesticides"
        HarvestDate: "2024-11-15"
    }
    Causes:  [bootstrap_event_A]
}
```

### Step 3: Farm sends harvest record to Factory via EGIP
**EGIP:** Message with CGER

```
Envelope {
    From:    "farm.example.com"
    To:      "factory.example.com"
    Type:    Message
    Payload: MessagePayload {
        Content:     step_2_content
        ContentType: "produce.harvested"
        CGERs: [
            CGER {
                System:  "farm.example.com"
                EventID: step_2_event_id
                Hash:    step_2_hash
            }
        ]
    }
}
```

System B records `egip.message.received` locally and sends a Receipt.

### Step 4: Factory receives and processes
**Grammar:** Derive (on System B's graph, caused by CGER from System A)

```
// On System B's graph
Event {
    Type:    "produce.received"
    Source:  factory_mgr
    Content: ProduceReceivedContent {
        Product:    "Organic Tomatoes"
        Quantity:   500
        Supplier:   "farm.example.com"
        SourceCGER: CGER { System: "farm.example.com", EventID: step_2_event_id, Hash: step_2_hash }
    }
    Causes:  [egip_message_received_event]
}
```

The CGER links this event to the farm's harvest event across graph boundaries.

### Step 5: QA agent inspects the produce
**Grammar:** Derive

```
// On System B's graph
Event {
    Type:    "qa.inspection"
    Source:  qa_agent
    Content: QAInspectionContent {
        Product:    "Organic Tomatoes"
        Result:     "passed"
        Tests:      ["visual", "pesticide_residue", "freshness"]
        Confidence: Score(0.92)
    }
    Causes:  [step_4_event]
}
```

### Step 6: Factory processes into product
**Grammar:** Derive

```
// On System B's graph
Event {
    Type:    "product.manufactured"
    Source:  factory_mgr
    Content: ProductManufacturedContent {
        Product:      "Organic Tomato Sauce"
        BatchID:      "BATCH-2024-1115-001"
        Ingredients:  [step_4_event]        // traced back to specific received produce
        QAReport:     step_5_event
        ProcessDate:  "2024-11-16"
    }
    Causes:  [step_4_event, step_5_event]
}
```

### Step 7: Factory endorses farm's produce quality
**Grammar:** Endorse (cross-system reputation)

```
// EGIP message back to farm
Envelope {
    From:    "factory.example.com"
    To:      "farm.example.com"
    Type:    Message
    Payload: MessagePayload {
        Content: EndorsementContent {
            Endorser: "factory.example.com"
            Subject:  step_2_event_id      // the harvest event
            Quality:  Score(0.9)
            Domain:   "produce_quality"
        }
        ContentType: "endorsement"
    }
}
```

Inter-system trust A→B increases from 0.6 to 0.65.

### Step 8: Factory sends product record to Retailer via EGIP
**EGIP:** Message with CGERs (chained across two systems)

```
Envelope {
    From:    "factory.example.com"
    To:      "retail.example.com"
    Type:    Message
    Payload: MessagePayload {
        Content:     step_6_content
        ContentType: "product.manufactured"
        CGERs: [
            // Reference to factory's manufacturing event
            CGER { System: "factory.example.com", EventID: step_6_event_id, Hash: step_6_hash },
            // Reference to farm's original harvest (transitive provenance)
            CGER { System: "farm.example.com", EventID: step_2_event_id, Hash: step_2_hash }
        ]
    }
}
```

### Step 9: Retailer receives and lists product
**Grammar:** Derive (on System C's graph)

```
// On System C's graph
Event {
    Type:    "product.listed"
    Source:  retailer_frank
    Content: ProductListedContent {
        Product:    "Organic Tomato Sauce"
        BatchID:    "BATCH-2024-1115-001"
        Price:      8.99
        Provenance: [
            CGER { System: "farm.example.com", EventID: step_2_event_id },
            CGER { System: "factory.example.com", EventID: step_6_event_id }
        ]
    }
    Causes:  [egip_message_received_event]
}
```

### Step 10: Consumer traces provenance
**Grammar:** Traverse (read-only, cross-system)

A consumer scans the product and queries provenance:

```
// Query: Where did this product come from?
// Start: product.listed event on retail system

Step 1: Retail graph → product.listed → CGER to factory.example.com
Step 2: EGIP proof request to factory.example.com

Envelope {
    From:    "retail.example.com"
    To:      "factory.example.com"
    Type:    Proof
    Payload: ProofPayload {
        ProofType: EventExistence
        Data: EventExistenceProof {
            Event:    step_6_event     // the manufacturing event
            Position: 42
        }
    }
}

Step 3: Factory graph → product.manufactured → CGER to farm.example.com
Step 4: EGIP proof request to farm.example.com (via factory, or direct if treaty exists)

Result: Complete chain visible:
  Farm (organic tomatoes, Plot 7, no pesticides)
  → Factory (received, QA passed, processed)
  → Retailer (listed at $8.99)
```

## Cross-System Trust Flow

```
Inter-system trust:
  farm ↔ factory: 0.60 → 0.65 (after quality endorsement)
  factory ↔ retail: 0.50 → 0.53 (after successful delivery)

Trust is non-transitive: retail trusts factory, factory trusts farm,
but retail does NOT automatically trust farm. Each relationship is independent.
```

## Assertions

1. **Cross-system provenance:** Consumer can trace from retail listing through factory manufacturing to farm harvest — across three sovereign systems
2. **CGER integrity:** Each CGER contains the source system, event ID, and hash — verifiable without trusting the intermediary
3. **Non-transitive trust:** Retail's trust in farm is independent of factory's trust in farm
4. **Treaty governance:** Farm and factory interactions are governed by treaty terms — only produce_supply events are shared
5. **QA auditability:** The QA inspection (step 5) is on the factory graph — if the product fails later, the inspection is auditable
6. **Signed envelopes:** Every EGIP message is signed by the sending system — no forgery
7. **Hash chain per system:** Each system maintains its own hash chain independently — no shared infrastructure
8. **Endorsement staking:** Factory's endorsement of farm quality is reputation-staked — if the produce later causes problems, the endorsement is evidence
9. **Proof verifiable:** Consumer can request existence proofs for any event in the chain without needing full access to the system's graph

## What Higher Layers Add

- **Work Graph (L1):** Primitives that understand production workflows, detect bottlenecks, and predict supply issues from historical patterns.
- **Market Graph (L2):** Fair pricing derived from transparent cost chains — the true cost of organic tomato sauce is visible.
- **Ethics Graph (L7):** Detects labour violations or environmental harm in the supply chain by analysing patterns across EGIP messages (e.g., harvest events at 3am suggest labour issues).
- **Existence Graph (L13):** Links economic output (tomato sauce sold) to ecological cost (water used, land impacted) on the same graph.
