package actor

import (
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// IActor represents an identity in the system.
type IActor interface {
	ID() types.ActorID
	PublicKey() types.PublicKey
	DisplayName() string
	Type() event.ActorType
	Metadata() map[string]any
	CreatedAt() types.Timestamp
	Status() types.ActorStatus
}

// Actor is an immutable implementation of IActor.
type Actor struct {
	id          types.ActorID
	publicKey   types.PublicKey
	displayName string
	actorType   event.ActorType
	metadata    map[string]any
	createdAt   types.Timestamp
	status      types.ActorStatus
}

// deepCopyMetadataValue recursively copies mutable values (maps, slices).
func deepCopyMetadataValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		cp := make(map[string]any, len(val))
		for k, elem := range val {
			cp[k] = deepCopyMetadataValue(elem)
		}
		return cp
	case []any:
		cp := make([]any, len(val))
		for i, elem := range val {
			cp[i] = deepCopyMetadataValue(elem)
		}
		return cp
	case []string:
		cp := make([]string, len(val))
		copy(cp, val)
		return cp
	default:
		return v
	}
}

func deepCopyMetadata(src map[string]any) map[string]any {
	if src == nil {
		return make(map[string]any)
	}
	cp := make(map[string]any, len(src))
	for k, v := range src {
		cp[k] = deepCopyMetadataValue(v)
	}
	return cp
}

// NewActor creates a new Actor.
func NewActor(
	id types.ActorID,
	publicKey types.PublicKey,
	displayName string,
	actorType event.ActorType,
	metadata map[string]any,
	createdAt types.Timestamp,
	status types.ActorStatus,
) Actor {
	md := deepCopyMetadata(metadata)
	return Actor{
		id:          id,
		publicKey:   publicKey,
		displayName: displayName,
		actorType:   actorType,
		metadata:    md,
		createdAt:   createdAt,
		status:      status,
	}
}

func (a Actor) ID() types.ActorID      { return a.id }
func (a Actor) PublicKey() types.PublicKey { return a.publicKey }
func (a Actor) DisplayName() string     { return a.displayName }
func (a Actor) Type() event.ActorType   { return a.actorType }
func (a Actor) Metadata() map[string]any {
	return deepCopyMetadata(a.metadata)
}
func (a Actor) CreatedAt() types.Timestamp    { return a.createdAt }
func (a Actor) Status() types.ActorStatus { return a.status }

// withStatus returns a copy of the actor with a new status.
func (a Actor) withStatus(status types.ActorStatus) Actor {
	md := deepCopyMetadata(a.metadata)
	return Actor{
		id:          a.id,
		publicKey:   a.publicKey,
		displayName: a.displayName,
		actorType:   a.actorType,
		metadata:    md,
		createdAt:   a.createdAt,
		status:      status,
	}
}

// withUpdates returns a copy of the actor with updates applied.
func (a Actor) withUpdates(u ActorUpdate) Actor {
	md := deepCopyMetadata(a.metadata)
	result := Actor{
		id:          a.id,
		publicKey:   a.publicKey,
		displayName: a.displayName,
		actorType:   a.actorType,
		metadata:    md,
		createdAt:   a.createdAt,
		status:      a.status,
	}
	if u.DisplayName.IsSome() {
		result.displayName = u.DisplayName.Unwrap()
	}
	if u.Metadata.IsSome() {
		for k, v := range u.Metadata.Unwrap() {
			result.metadata[k] = deepCopyMetadataValue(v)
		}
	}
	return result
}

// ActorUpdate describes updates to apply to an actor.
type ActorUpdate struct {
	DisplayName types.Option[string]
	Metadata    types.Option[map[string]any]
}

// ActorFilter describes criteria for listing actors.
type ActorFilter struct {
	Status types.Option[types.ActorStatus]
	Type   types.Option[event.ActorType]
	Limit  int
	After  types.Option[types.Cursor]
}

// IActorStore is the actor persistence interface.
type IActorStore interface {
	Register(publicKey types.PublicKey, displayName string, actorType event.ActorType) (IActor, error)
	Get(id types.ActorID) (IActor, error)
	GetByPublicKey(publicKey types.PublicKey) (IActor, error)
	Update(id types.ActorID, updates ActorUpdate) (IActor, error)
	List(filter ActorFilter) (types.Page[IActor], error)
	Suspend(id types.ActorID, reason types.EventID) (IActor, error)
	Reactivate(id types.ActorID, reason types.EventID) (IActor, error)
	Memorial(id types.ActorID, reason types.EventID) (IActor, error)
}
