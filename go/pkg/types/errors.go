package types

import "fmt"

// ValidationError is the marker interface for all construction validation errors.
type ValidationError interface {
	error
	validationError()
}

// OutOfRangeError indicates a numeric value was outside the allowed range.
type OutOfRangeError struct {
	Field string
	Value float64
	Min   float64
	Max   float64
}

func (e *OutOfRangeError) Error() string {
	return fmt.Sprintf("%s: value %v out of range [%v, %v]", e.Field, e.Value, e.Min, e.Max)
}
func (e *OutOfRangeError) validationError() {}

// IntOutOfRangeError indicates an integer value was outside the allowed range.
type IntOutOfRangeError struct {
	Field string
	Value int
	Min   int
	Max   int
}

func (e *IntOutOfRangeError) Error() string {
	return fmt.Sprintf("%s: value %d out of range [%d, %d]", e.Field, e.Value, e.Min, e.Max)
}
func (e *IntOutOfRangeError) validationError() {}

// InvalidFormatError indicates a string value did not match the expected format.
type InvalidFormatError struct {
	Field    string
	Value    string
	Expected string
}

func (e *InvalidFormatError) Error() string {
	return fmt.Sprintf("%s: invalid format %q, expected %s", e.Field, e.Value, e.Expected)
}
func (e *InvalidFormatError) validationError() {}

// EmptyRequiredError indicates a required field was empty.
type EmptyRequiredError struct {
	Field string
}

func (e *EmptyRequiredError) Error() string {
	return fmt.Sprintf("%s: required but empty", e.Field)
}
func (e *EmptyRequiredError) validationError() {}

// InvalidLifecycleTransitionError indicates an invalid lifecycle state transition.
type InvalidLifecycleTransitionError struct {
	From         LifecycleState
	To           LifecycleState
	ValidTargets []LifecycleState
}

func (e *InvalidLifecycleTransitionError) Error() string {
	return fmt.Sprintf("invalid lifecycle transition: %s → %s (valid: %v)", e.From, e.To, e.ValidTargets)
}
func (e *InvalidLifecycleTransitionError) validationError() {}

// InvalidActorTransitionError indicates an invalid actor status transition.
type InvalidActorTransitionError struct {
	From         ActorStatus
	To           ActorStatus
	ValidTargets []ActorStatus
}

func (e *InvalidActorTransitionError) Error() string {
	return fmt.Sprintf("invalid actor transition: %s → %s (valid: %v)", e.From, e.To, e.ValidTargets)
}
func (e *InvalidActorTransitionError) validationError() {}

// InvalidLifecycleStateError indicates a string is not a recognized lifecycle state.
type InvalidLifecycleStateError struct {
	Value string
}

func (e *InvalidLifecycleStateError) Error() string {
	return fmt.Sprintf("invalid lifecycle state: %q", e.Value)
}
func (e *InvalidLifecycleStateError) validationError() {}

// InvalidActorStatusError indicates a string is not a recognized actor status.
type InvalidActorStatusError struct {
	Value string
}

func (e *InvalidActorStatusError) Error() string {
	return fmt.Sprintf("invalid actor status: %q", e.Value)
}
func (e *InvalidActorStatusError) validationError() {}

// ValidationErrorVisitor provides exhaustive dispatch over validation errors.
type ValidationErrorVisitor interface {
	VisitOutOfRange(*OutOfRangeError)
	VisitIntOutOfRange(*IntOutOfRangeError)
	VisitInvalidFormat(*InvalidFormatError)
	VisitEmptyRequired(*EmptyRequiredError)
	VisitInvalidLifecycleTransition(*InvalidLifecycleTransitionError)
	VisitInvalidActorTransition(*InvalidActorTransitionError)
	VisitInvalidLifecycleState(*InvalidLifecycleStateError)
	VisitInvalidActorStatus(*InvalidActorStatusError)
}

// VisitableValidationError extends ValidationError with visitor support.
type VisitableValidationError interface {
	ValidationError
	Accept(ValidationErrorVisitor)
}

func (e *OutOfRangeError) Accept(v ValidationErrorVisitor)                  { v.VisitOutOfRange(e) }
func (e *IntOutOfRangeError) Accept(v ValidationErrorVisitor)               { v.VisitIntOutOfRange(e) }
func (e *InvalidFormatError) Accept(v ValidationErrorVisitor)               { v.VisitInvalidFormat(e) }
func (e *EmptyRequiredError) Accept(v ValidationErrorVisitor)               { v.VisitEmptyRequired(e) }
func (e *InvalidLifecycleTransitionError) Accept(v ValidationErrorVisitor)  { v.VisitInvalidLifecycleTransition(e) }
func (e *InvalidActorTransitionError) Accept(v ValidationErrorVisitor)      { v.VisitInvalidActorTransition(e) }
func (e *InvalidLifecycleStateError) Accept(v ValidationErrorVisitor)       { v.VisitInvalidLifecycleState(e) }
func (e *InvalidActorStatusError) Accept(v ValidationErrorVisitor)          { v.VisitInvalidActorStatus(e) }
