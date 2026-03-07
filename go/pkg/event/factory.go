package event

import (
	"fmt"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

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

	ev := Event{
		version:        1,
		id:             id,
		eventType:      eventType,
		timestamp:      time.Now().UTC(),
		source:         source,
		content:        content,
		causes:         causes,
		conversationID: conversationID,
		prevHash:       prevHash,
	}

	canonical := CanonicalForm(ev)
	hash, err := ComputeHash(canonical)
	if err != nil {
		return Event{}, err
	}
	ev.hash = hash

	sig, err := signer.Sign([]byte(canonical))
	if err != nil {
		return Event{}, err
	}
	ev.signature = sig

	return ev, nil
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
	now := time.Now().UTC()
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

	ev := Event{
		version:        1,
		id:             id,
		eventType:      types.MustEventType("system.bootstrapped"),
		timestamp:      now,
		source:         systemActor,
		content:        content,
		causes:         nil,
		conversationID: convID,
		prevHash:       types.ZeroHash(),
	}

	canonical := CanonicalForm(ev)
	hash, err := ComputeHash(canonical)
	if err != nil {
		return Event{}, err
	}
	ev.hash = hash

	sig, err := signer.Sign([]byte(canonical))
	if err != nil {
		return Event{}, err
	}
	ev.signature = sig

	return ev, nil
}
