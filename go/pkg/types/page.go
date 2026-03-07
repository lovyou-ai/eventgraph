package types

// Cursor is an opaque pagination token.
type Cursor struct {
	value string
}

// NewCursor creates a Cursor from a string value.
func NewCursor(v string) Cursor { return Cursor{value: v} }

// MustCursor creates a Cursor. Convenience alias for NewCursor.
func MustCursor(v string) Cursor { return Cursor{value: v} }

// Value returns the cursor's underlying string.
func (c Cursor) Value() string { return c.value }

// String returns the cursor's string representation.
func (c Cursor) String() string { return c.value }

// Page is a paginated result set with cursor-based navigation.
type Page[T any] struct {
	items   []T
	cursor  Option[Cursor]
	hasMore bool
}

// NewPage creates a Page with the given items, cursor, and hasMore flag.
func NewPage[T any](items []T, cursor Option[Cursor], hasMore bool) Page[T] {
	cp := make([]T, len(items))
	copy(cp, items)
	return Page[T]{items: cp, cursor: cursor, hasMore: hasMore}
}

// Items returns a copy of the page's items.
func (p Page[T]) Items() []T {
	cp := make([]T, len(p.items))
	copy(cp, p.items)
	return cp
}

// Cursor returns the pagination cursor for fetching the next page.
func (p Page[T]) Cursor() Option[Cursor] { return p.cursor }

// HasMore returns true if more pages are available.
func (p Page[T]) HasMore() bool { return p.hasMore }
