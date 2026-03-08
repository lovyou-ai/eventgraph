package egip

import (
	"fmt"

	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// EGIPError is the marker interface for all EGIP protocol errors.
type EGIPError interface {
	error
	egipError()
}

// SystemNotFoundError indicates the target system could not be reached.
type SystemNotFoundError struct{ URI types.SystemURI }

func (e *SystemNotFoundError) Error() string {
	return fmt.Sprintf("system not found: %s", e.URI.Value())
}
func (e *SystemNotFoundError) egipError() {}

// EnvelopeSignatureInvalidError indicates an envelope's signature failed verification.
type EnvelopeSignatureInvalidError struct{ EnvelopeID types.EnvelopeID }

func (e *EnvelopeSignatureInvalidError) Error() string {
	return fmt.Sprintf("envelope signature invalid: %s", e.EnvelopeID.Value())
}
func (e *EnvelopeSignatureInvalidError) egipError() {}

// TreatyViolationError indicates a treaty term was violated.
type TreatyViolationError struct {
	TreatyID types.TreatyID
	Term     string
}

func (e *TreatyViolationError) Error() string {
	return fmt.Sprintf("treaty %s violated: %s", e.TreatyID.Value(), e.Term)
}
func (e *TreatyViolationError) egipError() {}

// TrustInsufficientError indicates a system's trust score is too low.
type TrustInsufficientError struct {
	System   types.SystemURI
	Score    types.Score
	Required types.Score
}

func (e *TrustInsufficientError) Error() string {
	return fmt.Sprintf("trust insufficient for %s: have %v, need %v",
		e.System.Value(), e.Score.Value(), e.Required.Value())
}
func (e *TrustInsufficientError) egipError() {}

// TransportFailureError indicates a transport-level failure (retryable).
type TransportFailureError struct {
	To     types.SystemURI
	Reason string
}

func (e *TransportFailureError) Error() string {
	return fmt.Sprintf("transport failure to %s: %s", e.To.Value(), e.Reason)
}
func (e *TransportFailureError) egipError() {}

// DuplicateEnvelopeError indicates a replay — an envelope with this ID was already processed.
type DuplicateEnvelopeError struct{ EnvelopeID types.EnvelopeID }

func (e *DuplicateEnvelopeError) Error() string {
	return fmt.Sprintf("duplicate envelope: %s", e.EnvelopeID.Value())
}
func (e *DuplicateEnvelopeError) egipError() {}

// TreatyNotFoundError indicates the referenced treaty does not exist.
type TreatyNotFoundError struct{ TreatyID types.TreatyID }

func (e *TreatyNotFoundError) Error() string {
	return fmt.Sprintf("treaty not found: %s", e.TreatyID.Value())
}
func (e *TreatyNotFoundError) egipError() {}

// VersionIncompatibleError indicates no common protocol version exists.
type VersionIncompatibleError struct {
	Local  []int
	Remote []int
}

func (e *VersionIncompatibleError) Error() string {
	return fmt.Sprintf("no compatible protocol version: local %v, remote %v", e.Local, e.Remote)
}
func (e *VersionIncompatibleError) egipError() {}
