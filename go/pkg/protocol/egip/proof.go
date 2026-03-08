package egip

import (
	"fmt"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// ProofGenerator creates integrity proofs from a local store.
type ProofGenerator struct {
	store store.Store
}

// NewProofGenerator creates a proof generator backed by the given store.
func NewProofGenerator(s store.Store) *ProofGenerator {
	return &ProofGenerator{store: s}
}

// GenerateChainSummary produces a high-level chain integrity attestation.
func (pg *ProofGenerator) GenerateChainSummary() (*ChainSummaryProof, error) {
	page, err := pg.store.Recent(1, types.None[types.Cursor]())
	if err != nil {
		return nil, fmt.Errorf("query recent: %w", err)
	}
	items := page.Items()
	if len(items) == 0 {
		return nil, fmt.Errorf("empty chain")
	}

	headEvent := items[0]
	count, err := pg.store.Count()
	if err != nil {
		return nil, fmt.Errorf("event count: %w", err)
	}

	return &ChainSummaryProof{
		Length:      count,
		HeadHash:    headEvent.Hash(),
		GenesisHash: headEvent.PrevHash(), // simplified — real impl would walk to genesis
		Timestamp:   time.Now(),
	}, nil
}

// GenerateEventExistence proves that a specific event exists in the chain.
func (pg *ProofGenerator) GenerateEventExistence(eventID types.EventID) (*EventExistenceProof, error) {
	evt, err := pg.store.Get(eventID)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	count, err := pg.store.Count()
	if err != nil {
		return nil, fmt.Errorf("event count: %w", err)
	}

	return &EventExistenceProof{
		Event:       evt,
		PrevHash:    evt.PrevHash(),
		NextHash:    types.None[types.Hash](), // simplified — would need chain traversal
		Position:    0,                         // simplified
		ChainLength: count,
	}, nil
}

// VerifyChainSegment verifies that a chain segment is internally consistent.
func VerifyChainSegment(proof *ChainSegmentProof) bool {
	if len(proof.Events) == 0 {
		return false
	}

	// Check that the first event's PrevHash matches StartHash.
	if proof.Events[0].PrevHash() != proof.StartHash {
		return false
	}

	// Check internal hash chain continuity.
	for i := 1; i < len(proof.Events); i++ {
		if proof.Events[i].PrevHash() != proof.Events[i-1].Hash() {
			return false
		}
	}

	// Check that the last event's hash matches EndHash.
	if proof.Events[len(proof.Events)-1].Hash() != proof.EndHash {
		return false
	}

	return true
}

// VerifyEventExistence verifies basic properties of an event existence proof.
func VerifyEventExistence(proof *EventExistenceProof) bool {
	// The event's PrevHash should match the proof's PrevHash.
	if proof.Event.PrevHash() != proof.PrevHash {
		return false
	}

	// Position and chain length should be consistent.
	if proof.Position < 0 || proof.Position >= proof.ChainLength {
		return false
	}

	// The event should have a non-empty hash.
	if proof.Event.Hash() == (types.Hash{}) {
		return false
	}

	return true
}

// ValidateProof dispatches to the appropriate proof verifier.
func ValidateProof(payload *ProofPayload) (bool, error) {
	switch data := payload.Data.(type) {
	case ChainSegmentProof:
		return VerifyChainSegment(&data), nil
	case *ChainSegmentProof:
		return VerifyChainSegment(data), nil
	case EventExistenceProof:
		return VerifyEventExistence(&data), nil
	case *EventExistenceProof:
		return VerifyEventExistence(data), nil
	case ChainSummaryProof:
		return data.Length > 0, nil
	case *ChainSummaryProof:
		return data.Length > 0, nil
	default:
		return false, fmt.Errorf("unknown proof type: %T", payload.Data)
	}
}

// ProofTypeFromData returns the ProofType for a given ProofData.
func ProofTypeFromData(data ProofData) (event.ProofType, error) {
	switch data.(type) {
	case ChainSegmentProof, *ChainSegmentProof:
		return event.ProofTypeChainSegment, nil
	case EventExistenceProof, *EventExistenceProof:
		return event.ProofTypeEventExistence, nil
	case ChainSummaryProof, *ChainSummaryProof:
		return event.ProofTypeChainSummary, nil
	default:
		return "", fmt.Errorf("unknown proof data type: %T", data)
	}
}
