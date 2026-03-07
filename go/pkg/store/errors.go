package store

import (
	"fmt"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// StoreError is the marker interface for all store errors.
type StoreError interface {
	error
	storeError()
}

// EventNotFoundError indicates an event was not found.
type EventNotFoundError struct{ ID types.EventID }

func (e *EventNotFoundError) Error() string { return fmt.Sprintf("event not found: %s", e.ID.Value()) }
func (e *EventNotFoundError) storeError()   {}

// ActorNotFoundError indicates an actor was not found.
type ActorNotFoundError struct{ ID types.ActorID }

func (e *ActorNotFoundError) Error() string { return fmt.Sprintf("actor not found: %s", e.ID.Value()) }
func (e *ActorNotFoundError) storeError()   {}

// EdgeNotFoundError indicates an edge was not found.
type EdgeNotFoundError struct {
	From     types.ActorID
	To       types.ActorID
	EdgeType event.EdgeType
}

func (e *EdgeNotFoundError) Error() string {
	return fmt.Sprintf("edge not found: %s → %s (%s)", e.From.Value(), e.To.Value(), e.EdgeType)
}
func (e *EdgeNotFoundError) storeError() {}

// DuplicateEventError indicates an event with this ID already exists.
type DuplicateEventError struct{ ID types.EventID }

func (e *DuplicateEventError) Error() string {
	return fmt.Sprintf("duplicate event: %s", e.ID.Value())
}
func (e *DuplicateEventError) storeError() {}

// CausalLinkMissingError indicates an event references a cause that doesn't exist.
type CausalLinkMissingError struct {
	EventID      types.EventID
	MissingCause types.EventID
}

func (e *CausalLinkMissingError) Error() string {
	return fmt.Sprintf("causal link missing: event %s references cause %s", e.EventID.Value(), e.MissingCause.Value())
}
func (e *CausalLinkMissingError) storeError() {}

// ChainIntegrityViolationError indicates the hash chain is broken.
type ChainIntegrityViolationError struct {
	Position int
	Expected types.Hash
	Actual   types.Hash
}

func (e *ChainIntegrityViolationError) Error() string {
	return fmt.Sprintf("chain integrity violation at position %d: expected %s, got %s", e.Position, e.Expected.Value(), e.Actual.Value())
}
func (e *ChainIntegrityViolationError) storeError() {}

// HashMismatchError indicates a stored hash doesn't match the computed hash.
type HashMismatchError struct {
	EventID  types.EventID
	Computed types.Hash
	Stored   types.Hash
}

func (e *HashMismatchError) Error() string {
	return fmt.Sprintf("hash mismatch for event %s: computed %s, stored %s", e.EventID.Value(), e.Computed.Value(), e.Stored.Value())
}
func (e *HashMismatchError) storeError() {}

// SignatureInvalidError indicates a signature doesn't verify.
type SignatureInvalidError struct {
	EventID types.EventID
	Signer  types.ActorID
}

func (e *SignatureInvalidError) Error() string {
	return fmt.Sprintf("invalid signature for event %s by %s", e.EventID.Value(), e.Signer.Value())
}
func (e *SignatureInvalidError) storeError() {}

// ActorSuspendedError indicates a suspended actor tried to emit.
type ActorSuspendedError struct{ ID types.ActorID }

func (e *ActorSuspendedError) Error() string {
	return fmt.Sprintf("actor suspended: %s", e.ID.Value())
}
func (e *ActorSuspendedError) storeError() {}

// ActorMemorialError indicates a memorialised actor tried to emit.
type ActorMemorialError struct{ ID types.ActorID }

func (e *ActorMemorialError) Error() string {
	return fmt.Sprintf("actor memorialised: %s", e.ID.Value())
}
func (e *ActorMemorialError) storeError() {}

// RateLimitExceededError indicates an actor exceeded the rate limit.
type RateLimitExceededError struct {
	Actor  types.ActorID
	Limit  int
	Window string
}

func (e *RateLimitExceededError) Error() string {
	return fmt.Sprintf("rate limit exceeded for %s: %d per %s", e.Actor.Value(), e.Limit, e.Window)
}
func (e *RateLimitExceededError) storeError() {}

// StoreUnavailableError indicates the backing store is unavailable.
type StoreUnavailableError struct{ Reason string }

func (e *StoreUnavailableError) Error() string {
	return fmt.Sprintf("store unavailable: %s", e.Reason)
}
func (e *StoreUnavailableError) storeError() {}

// StoreErrorVisitor provides exhaustive dispatch over store errors.
type StoreErrorVisitor interface {
	VisitEventNotFound(*EventNotFoundError)
	VisitActorNotFound(*ActorNotFoundError)
	VisitEdgeNotFound(*EdgeNotFoundError)
	VisitDuplicateEvent(*DuplicateEventError)
	VisitCausalLinkMissing(*CausalLinkMissingError)
	VisitChainIntegrityViolation(*ChainIntegrityViolationError)
	VisitHashMismatch(*HashMismatchError)
	VisitSignatureInvalid(*SignatureInvalidError)
	VisitActorSuspended(*ActorSuspendedError)
	VisitActorMemorial(*ActorMemorialError)
	VisitRateLimitExceeded(*RateLimitExceededError)
	VisitStoreUnavailable(*StoreUnavailableError)
}

// VisitableStoreError extends StoreError with visitor support.
type VisitableStoreError interface {
	StoreError
	Accept(StoreErrorVisitor)
}

func (e *EventNotFoundError) Accept(v StoreErrorVisitor)           { v.VisitEventNotFound(e) }
func (e *ActorNotFoundError) Accept(v StoreErrorVisitor)           { v.VisitActorNotFound(e) }
func (e *EdgeNotFoundError) Accept(v StoreErrorVisitor)            { v.VisitEdgeNotFound(e) }
func (e *DuplicateEventError) Accept(v StoreErrorVisitor)          { v.VisitDuplicateEvent(e) }
func (e *CausalLinkMissingError) Accept(v StoreErrorVisitor)      { v.VisitCausalLinkMissing(e) }
func (e *ChainIntegrityViolationError) Accept(v StoreErrorVisitor) { v.VisitChainIntegrityViolation(e) }
func (e *HashMismatchError) Accept(v StoreErrorVisitor)            { v.VisitHashMismatch(e) }
func (e *SignatureInvalidError) Accept(v StoreErrorVisitor)        { v.VisitSignatureInvalid(e) }
func (e *ActorSuspendedError) Accept(v StoreErrorVisitor)          { v.VisitActorSuspended(e) }
func (e *ActorMemorialError) Accept(v StoreErrorVisitor)           { v.VisitActorMemorial(e) }
func (e *RateLimitExceededError) Accept(v StoreErrorVisitor)       { v.VisitRateLimitExceeded(e) }
func (e *StoreUnavailableError) Accept(v StoreErrorVisitor)        { v.VisitStoreUnavailable(e) }
