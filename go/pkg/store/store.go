package store

import (
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Store is the event and edge persistence interface.
// Implementations must be safe for concurrent access.
// Append serialises hash chain writes internally.
type Store interface {
	// Append persists a pre-built Event from EventFactory.
	// Validates PrevHash matches chain head, recomputes and verifies Hash.
	// Idempotent: if the same ID already exists, returns the stored event without error.
	Append(ev event.Event) (event.Event, error)

	// Get retrieves an event by ID.
	Get(id types.EventID) (event.Event, error)

	// Head returns the most recent event, or None if the store is empty.
	Head() (types.Option[event.Event], error)

	// Recent returns the most recent events.
	Recent(limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error)

	// ByType returns events of a specific type.
	ByType(eventType types.EventType, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error)

	// BySource returns events from a specific actor.
	BySource(source types.ActorID, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error)

	// ByConversation returns events in a conversation thread.
	ByConversation(id types.ConversationID, limit int, after types.Option[types.Cursor]) (types.Page[event.Event], error)

	// Since returns events after a specific event ID.
	Since(afterID types.EventID, limit int) (types.Page[event.Event], error)

	// Ancestors returns causal ancestors up to maxDepth.
	Ancestors(id types.EventID, maxDepth int) ([]event.Event, error)

	// Descendants returns causal descendants up to maxDepth.
	Descendants(id types.EventID, maxDepth int) ([]event.Event, error)

	// EdgesFrom returns edges originating from an actor of a specific type.
	EdgesFrom(id types.ActorID, edgeType event.EdgeType) ([]event.Edge, error)

	// EdgesTo returns edges pointing to an actor of a specific type.
	EdgesTo(id types.ActorID, edgeType event.EdgeType) ([]event.Edge, error)

	// EdgeBetween returns the edge between two actors of a specific type, if any.
	EdgeBetween(from types.ActorID, to types.ActorID, edgeType event.EdgeType) (types.Option[event.Edge], error)

	// Count returns the total number of events.
	Count() (int, error)

	// VerifyChain verifies the hash chain integrity.
	VerifyChain() (event.ChainVerifiedContent, error)

	// Close releases resources.
	Close() error
}

// IIdentity provides signing and verification for events.
type IIdentity interface {
	// SystemURI returns this system's identity URI.
	SystemURI() types.SystemURI

	// PublicKey returns this system's public key.
	PublicKey() types.PublicKey

	// Sign signs the given data, returning a signature.
	Sign(data []byte) (types.Signature, error)

	// Verify checks a signature against a public key and data.
	Verify(publicKey types.PublicKey, data []byte, signature types.Signature) (bool, error)
}
