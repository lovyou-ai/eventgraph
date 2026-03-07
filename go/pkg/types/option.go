// Package types provides foundational value objects, constrained numerics,
// typed IDs, and state machines for the event graph.
package types

import (
	"encoding/json"
)

// Option represents an explicitly optional value.
// Zero value is None. Use Some() or None() constructors.
type Option[T any] struct {
	value T
	valid bool
}

// Some creates an Option containing a value.
func Some[T any](v T) Option[T] { return Option[T]{value: v, valid: true} }

// None creates an empty Option.
func None[T any]() Option[T] { return Option[T]{} }

// IsSome returns true if the Option contains a value.
func (o Option[T]) IsSome() bool { return o.valid }

// IsNone returns true if the Option is empty.
func (o Option[T]) IsNone() bool { return !o.valid }

// Unwrap returns the contained value. Panics if None.
func (o Option[T]) Unwrap() T {
	if !o.valid {
		panic("unwrap on None")
	}
	return o.value
}

// UnwrapOr returns the contained value or the provided default.
func (o Option[T]) UnwrapOr(def T) T {
	if o.valid {
		return o.value
	}
	return def
}

// MarshalJSON marshals Some(v) as v, None as null.
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.valid {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

// UnmarshalJSON unmarshals a value into Some, or null into None.
func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = None[T]()
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = Some(v)
	return nil
}
