# Tutorial: Connect Two Event Graphs

EventGraph systems are sovereign — each has its own identity, event chain, and trust model. EGIP (EventGraph Inter-system Protocol) lets them communicate without shared infrastructure.

## What You'll Build

Two independent event graphs that:
1. Establish identity with Ed25519 keypairs
2. Exchange HELLO messages
3. Negotiate a treaty for bilateral governance
4. Send cross-graph event references (CGERs)
5. Build trust through successful interactions

## Prerequisites

- Completed the minimal example (`examples/minimal/`)
- Understanding of event graphs (bootstrap, events, hash chain)

## Step 1: Create Two Sovereign Systems

Each system has its own identity, store, and event chain:

```go
package main

import (
    "fmt"

    "github.com/lovyou-ai/eventgraph/go/pkg/event"
    "github.com/lovyou-ai/eventgraph/go/pkg/graph"
    "github.com/lovyou-ai/eventgraph/go/pkg/protocol/egip"
    "github.com/lovyou-ai/eventgraph/go/pkg/types"
)

func main() {
    // System A: a research lab
    identityA := egip.GenerateIdentity(types.MustSystemURI("lab.example.org"))

    // System B: a publisher
    identityB := egip.GenerateIdentity(types.MustSystemURI("publisher.example.org"))

    fmt.Printf("Lab:       %s\n", identityA.URI().Value())
    fmt.Printf("Publisher: %s\n", identityB.URI().Value())
}
```

Each identity is an Ed25519 keypair. No central registry. No shared authority. Systems prove who they are by signing messages.

## Step 2: Exchange HELLO Messages

The HELLO handshake establishes a connection:

```go
    // Create protocol handlers
    handlerA := egip.NewHandler(identityA, transportA)
    handlerB := egip.NewHandler(identityB, transportB)

    // A says hello to B
    hello := &egip.Envelope{
        ProtocolVersion: egip.CurrentProtocolVersion,
        ID:              newEnvelopeID(),
        From:            identityA.URI(),
        To:              identityB.URI(),
        MessageType:     event.MessageTypeHello,
        TimestampNanos:  time.Now().UnixNano(),
        Payload: egip.HelloPayload{
            SystemURI:        identityA.URI(),
            PublicKey:         identityA.PublicKey(),
            Capabilities:     []string{"events", "proofs"},
            ProtocolVersions: []int{egip.CurrentProtocolVersion},
        },
        Signature: /* signed by A */,
    }

    // B receives and auto-responds with its own HELLO
    handlerB.HandleIncoming(hello)
```

After HELLO:
- Both systems know each other's public key
- Both can verify each other's signatures
- Trust starts at 0.0 — it must be earned

## Step 3: Negotiate a Treaty

Treaties define bilateral governance rules:

```go
    treaty := &egip.Treaty{
        ID:          newTreatyID(),
        Proposer:    identityA.URI(),
        Acceptor:    identityB.URI(),
        Terms: egip.TreatyTerms{
            AllowedEventTypes: []types.EventType{
                types.MustEventType("research.published"),
                types.MustEventType("review.completed"),
            },
            RequireReceipts: true,
            MaxCGERsPerDay:  100,
        },
        State: egip.TreatyProposed,
    }

    // A proposes, B accepts
    handlerA.ProposeTreaty(treaty)
    handlerB.AcceptTreaty(treaty.ID)
```

Treaty states follow a state machine: `Proposed → Active → Suspended → Terminated`. Both parties must agree to activate.

## Step 4: Send Cross-Graph Event References

CGERs let events in one graph reference events in another:

```go
    // Lab publishes a research finding
    labEvent := createEvent("research.published", map[string]any{
        "title":  "Novel Trust Metrics",
        "doi":    "10.1234/trust.2026",
    })

    // Send a MESSAGE with CGER to the publisher
    message := &egip.Envelope{
        MessageType: event.MessageTypeMessage,
        Payload: egip.MessagePayload{
            TreatyID: treaty.ID,
            CGERs: []egip.CGER{{
                SourceSystem:  identityA.URI(),
                SourceEventID: labEvent.ID(),
                SourceHash:    labEvent.Hash(),
                EventType:     labEvent.Type(),
                Summary:       "New trust metrics research published",
            }},
        },
    }

    handlerB.HandleIncoming(message)
```

CGERs contain:
- **SourceSystem** — who created the original event
- **SourceEventID** — the event's ID in the source graph
- **SourceHash** — for verification without access to the source chain
- **EventType** and **Summary** — what happened, in brief

The receiving system can request a full integrity proof:

```go
    // Publisher requests proof of the research event
    proof := &egip.Envelope{
        MessageType: event.MessageTypeProof,
        Payload: egip.ProofRequestPayload{
            RequestedEventID: labEvent.ID(),
            ProofType:        egip.ProofTypeEventExistence,
        },
    }
```

## Step 5: Build Trust

Trust accumulates through successful interactions:

```go
    // After receiving valid messages and proofs, trust grows
    trust := handlerB.PeerStore().Trust(identityA.URI())
    fmt.Printf("Trust in lab: %.2f\n", trust.Value())
    // Output: Trust in lab: 0.05  (after first valid interaction)
```

Trust properties:
- **Asymmetric** — A's trust in B ≠ B's trust in A
- **Non-transitive** — A trusts B, B trusts C ≠ A trusts C
- **Decaying** — trust diminishes without ongoing interaction
- **Contextual** — trust can be domain-specific
- **Bounded** — positive adjustments capped at 0.05, negative uncapped (security violations hit immediately)

## Running the Full Example

See `examples/multi-system/main.go` for a complete runnable example with two sovereign systems exchanging HELLO, negotiating a treaty, sending CGERs, requesting proofs, and accumulating trust.

```bash
cd examples/multi-system
go run .
```

## What's Next

- Read `docs/protocol.md` for the full EGIP specification
- See `go/pkg/protocol/egip/` for the reference implementation
- See `go/pkg/integration/scenario05_supply_chain_test.go` for a three-system supply chain example with treaties, CGERs, and proofs
