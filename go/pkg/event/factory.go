package event

import (
	"encoding/hex"
	"fmt"

	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// CurrentEventVersion is the schema version for newly created events.
const CurrentEventVersion = 1

// Signer provides signing capability for event creation.
// Defined here to avoid circular imports (store imports event).
type Signer interface {
	Sign(data []byte) (types.Signature, error)
}

// HeadProvider provides the current chain head for event creation.
// This is a subset of Store — avoids circular dependency.
type HeadProvider interface {
	Head() (types.Option[Event], error)
}

// EventFactory creates fully-validated, immutable events.
type EventFactory struct {
	Registry *EventTypeRegistry
}

// NewEventFactory creates a new EventFactory with the given registry.
func NewEventFactory(registry *EventTypeRegistry) *EventFactory {
	return &EventFactory{Registry: registry}
}

// Create creates a new event. Validates all inputs, computes hash, signs.
func (f *EventFactory) Create(
	eventType types.EventType,
	source types.ActorID,
	content EventContent,
	causes []types.EventID,
	conversationID types.ConversationID,
	head HeadProvider,
	signer Signer,
) (Event, error) {
	if len(causes) == 0 {
		return Event{}, fmt.Errorf("event must have at least one cause (use BootstrapFactory for genesis)")
	}

	if err := f.Registry.Validate(eventType, content); err != nil {
		return Event{}, err
	}

	headOpt, err := head.Head()
	if err != nil {
		return Event{}, err
	}
	prevHash := types.ZeroHash()
	if headOpt.IsSome() {
		prevHash = headOpt.Unwrap().Hash()
	}

	id, err := types.NewEventIDFromNew()
	if err != nil {
		return Event{}, err
	}

	// Defensive copy of causes to prevent caller mutation
	causesCopy := make([]types.EventID, len(causes))
	copy(causesCopy, causes)

	timestamp := types.Now()

	// Build a temporary event to compute canonical form, then construct
	// the final immutable event via NewEvent with all fields.
	tmp := Event{
		version:        CurrentEventVersion,
		id:             id,
		eventType:      eventType,
		timestamp:      timestamp,
		source:         source,
		content:        content,
		causes:         causesCopy,
		conversationID: conversationID,
		prevHash:       prevHash,
	}

	canonical := CanonicalForm(tmp)
	hash, err := ComputeHash(canonical)
	if err != nil {
		return Event{}, err
	}

	// Sign the hash bytes — binds signature to the exact hash value.
	hashBytes, err := hex.DecodeString(hash.Value())
	if err != nil {
		return Event{}, fmt.Errorf("decode hash for signing: %w", err)
	}
	sig, err := signer.Sign(hashBytes)
	if err != nil {
		return Event{}, err
	}

	return NewEvent(CurrentEventVersion, id, eventType, timestamp, source, content, causesCopy,
		conversationID, hash, prevHash, sig), nil
}

// BootstrapFactory creates the genesis event.
type BootstrapFactory struct {
	Registry *EventTypeRegistry
}

// NewBootstrapFactory creates a new BootstrapFactory.
func NewBootstrapFactory(registry *EventTypeRegistry) *BootstrapFactory {
	return &BootstrapFactory{Registry: registry}
}

// Init creates the genesis event — no causes, zero PrevHash.
func (f *BootstrapFactory) Init(
	systemActor types.ActorID,
	signer Signer,
) (Event, error) {
	now := types.Now()
	id, err := types.NewEventIDFromNew()
	if err != nil {
		return Event{}, err
	}

	convID, err := types.NewConversationID("conv_bootstrap_00000000000000000001")
	if err != nil {
		return Event{}, err
	}

	content := BootstrapContent{
		ActorID:      systemActor,
		ChainGenesis: types.ZeroHash(),
		Timestamp:    now,
	}

	// Build a temporary event to compute canonical form, then construct
	// the final immutable event via NewBootstrapEvent with all fields.
	tmp := Event{
		version:        CurrentEventVersion,
		id:             id,
		eventType:      EventTypeSystemBootstrapped,
		timestamp:      now,
		source:         systemActor,
		content:        content,
		causes:         nil,
		conversationID: convID,
		prevHash:       types.ZeroHash(),
	}

	canonical := CanonicalForm(tmp)
	hash, err := ComputeHash(canonical)
	if err != nil {
		return Event{}, err
	}

	// Sign the hash bytes — binds signature to the exact hash value.
	hashBytes, err := hex.DecodeString(hash.Value())
	if err != nil {
		return Event{}, fmt.Errorf("decode hash for signing: %w", err)
	}
	sig, err := signer.Sign(hashBytes)
	if err != nil {
		return Event{}, err
	}

	return NewBootstrapEvent(CurrentEventVersion, id, EventTypeSystemBootstrapped, now, systemActor,
		content, convID, hash, sig), nil
}
