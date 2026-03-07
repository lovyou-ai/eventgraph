package types

import "encoding/json"

// NonEmpty is a slice guaranteed to have at least one element.
// Construction rejects empty input.
type NonEmpty[T any] struct {
	head T
	tail []T
}

// NewNonEmpty creates a NonEmpty from a slice. Returns an error if the slice is empty.
func NewNonEmpty[T any](items []T) (NonEmpty[T], error) {
	if len(items) == 0 {
		return NonEmpty[T]{}, &EmptyRequiredError{Field: "NonEmpty"}
	}
	tail := make([]T, len(items)-1)
	copy(tail, items[1:])
	return NonEmpty[T]{head: items[0], tail: tail}, nil
}

// MustNonEmpty creates a NonEmpty from a slice. Panics if the slice is empty.
func MustNonEmpty[T any](items []T) NonEmpty[T] {
	ne, err := NewNonEmpty(items)
	if err != nil {
		panic(err)
	}
	return ne
}

// First returns the first element. Always succeeds.
func (ne NonEmpty[T]) First() T { return ne.head }

// All returns all elements as a new slice.
func (ne NonEmpty[T]) All() []T {
	result := make([]T, 1+len(ne.tail))
	result[0] = ne.head
	copy(result[1:], ne.tail)
	return result
}

// Len returns the number of elements.
func (ne NonEmpty[T]) Len() int { return 1 + len(ne.tail) }

// MarshalJSON marshals as a JSON array.
func (ne NonEmpty[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(ne.All())
}

// UnmarshalJSON unmarshals from a JSON array. Returns an error if the array is empty.
func (ne *NonEmpty[T]) UnmarshalJSON(data []byte) error {
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	result, err := NewNonEmpty(items)
	if err != nil {
		return err
	}
	*ne = result
	return nil
}
