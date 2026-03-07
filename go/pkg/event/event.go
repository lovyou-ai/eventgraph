package event

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Event is the fundamental unit. Every significant action is an event.
// Immutable after construction.
type Event struct {
	version        int
	id             types.EventID
	eventType      types.EventType
	timestamp      types.Timestamp
	source         types.ActorID
	content        EventContent
	causes         []types.EventID
	conversationID types.ConversationID
	hash           types.Hash
	prevHash       types.Hash
	signature      types.Signature
}

// NewEvent creates a new immutable Event.
// For normal events — use EventFactory instead of calling this directly in production.
// This constructor exists for tests and low-level use.
// NewEvent creates a normal event. Causes must not be empty — use NonEmpty to enforce at the call site.
// EventFactory enforces NonEmpty; this constructor takes a slice for flexibility.
func NewEvent(
	version int,
	id types.EventID,
	eventType types.EventType,
	timestamp types.Timestamp,
	source types.ActorID,
	content EventContent,
	causes []types.EventID,
	conversationID types.ConversationID,
	hash types.Hash,
	prevHash types.Hash,
	signature types.Signature,
) Event {
	c := make([]types.EventID, len(causes))
	copy(c, causes)
	return Event{
		version:        version,
		id:             id,
		eventType:      eventType,
		timestamp:      timestamp,
		source:         source,
		content:        content,
		causes:         c,
		conversationID: conversationID,
		hash:           hash,
		prevHash:       prevHash,
		signature:      signature,
	}
}

// NewBootstrapEvent creates the genesis event with no causes and zero PrevHash.
// Only valid for bootstrap — normal events must use NewEvent with NonEmpty causes.
func NewBootstrapEvent(
	version int,
	id types.EventID,
	eventType types.EventType,
	timestamp types.Timestamp,
	source types.ActorID,
	content BootstrapContent,
	conversationID types.ConversationID,
	hash types.Hash,
	signature types.Signature,
) Event {
	return Event{
		version:        version,
		id:             id,
		eventType:      eventType,
		timestamp:      timestamp,
		source:         source,
		content:        content,
		causes:         nil, // bootstrap has no causes
		conversationID: conversationID,
		hash:           hash,
		prevHash:       types.ZeroHash(),
		signature:      signature,
	}
}

func (e Event) Version() int                              { return e.version }
func (e Event) ID() types.EventID                         { return e.id }
func (e Event) Type() types.EventType                     { return e.eventType }
func (e Event) Timestamp() types.Timestamp                 { return e.timestamp }
func (e Event) Source() types.ActorID                     { return e.source }
func (e Event) Content() EventContent                     { return e.content }
func (e Event) Causes() []types.EventID {
	c := make([]types.EventID, len(e.causes))
	copy(c, e.causes)
	return c
}
func (e Event) ConversationID() types.ConversationID      { return e.conversationID }
func (e Event) Hash() types.Hash                          { return e.hash }
func (e Event) PrevHash() types.Hash                      { return e.prevHash }
func (e Event) Signature() types.Signature                { return e.signature }

// IsBootstrap returns true if this is the genesis event (no causes, zero prev hash).
func (e Event) IsBootstrap() bool {
	return len(e.causes) == 0 && e.prevHash == types.ZeroHash()
}

// CanonicalForm produces the canonical string representation of this event.
// Format: version|prev_hash|id|type|source|conversation_id|timestamp_nanos|content_json
func CanonicalForm(e Event) string {
	prevHash := ""
	if !e.prevHash.IsZero() {
		prevHash = e.prevHash.Value()
	}

	contentJSON := canonicalContentJSON(e.content)

	return fmt.Sprintf("%d|%s|%s|%s|%s|%s|%d|%s",
		e.version,
		prevHash,
		e.id.Value(),
		e.eventType.Value(),
		e.source.Value(),
		e.conversationID.Value(),
		e.timestamp.UnixNano(),
		contentJSON,
	)
}

// ComputeHash computes SHA-256 of the canonical form.
func ComputeHash(canonical string) (types.Hash, error) {
	h := sha256.Sum256([]byte(canonical))
	hex := fmt.Sprintf("%x", h)
	return types.NewHash(hex)
}

// canonicalContentJSON serialises content to JSON with sorted keys, no whitespace,
// null/None fields omitted.
// Panics on marshal failure — content types are validated at construction,
// so marshal failure is an unrecoverable invariant violation.
func canonicalContentJSON(content EventContent) string {
	if content == nil {
		return "{}"
	}

	b, err := json.Marshal(content)
	if err != nil {
		panic(fmt.Sprintf("canonicalContentJSON: marshal failed for %T: %v", content, err))
	}

	// Re-parse to sort keys
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		panic(fmt.Sprintf("canonicalContentJSON: re-parse failed for %T: %v", content, err))
	}

	return sortedJSON(raw)
}

// sortedJSON produces JSON with lexicographically sorted keys, no whitespace,
// null values omitted.
func sortedJSON(m map[string]any) string {
	if len(m) == 0 {
		return "{}"
	}

	keys := make([]string, 0, len(m))
	for k, v := range m {
		if v == nil {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteByte('{')
	first := true
	for _, k := range keys {
		v := m[k]
		if !first {
			sb.WriteByte(',')
		}
		first = false
		kb, _ := json.Marshal(k)
		sb.Write(kb)
		sb.WriteByte(':')
		writeCanonicalValue(&sb, v)
	}
	sb.WriteByte('}')
	return sb.String()
}

func writeCanonicalValue(sb *strings.Builder, v any) {
	switch val := v.(type) {
	case map[string]any:
		sb.WriteString(sortedJSON(val))
	case []any:
		sb.WriteByte('[')
		for i, elem := range val {
			if i > 0 {
				sb.WriteByte(',')
			}
			writeCanonicalValue(sb, elem)
		}
		sb.WriteByte(']')
	case float64:
		// JSON numbers: no trailing zeros, no unnecessary decimal point
		s := formatCanonicalNumber(val)
		sb.WriteString(s)
	default:
		b, _ := json.Marshal(val)
		sb.Write(b)
	}
}

// formatCanonicalNumber formats a float64 per canonical form rules:
// no trailing zeros, no leading zeros (except 0.x), no + prefix.
func formatCanonicalNumber(f float64) string {
	if f == float64(int64(f)) && f >= -1e15 && f <= 1e15 {
		return fmt.Sprintf("%d", int64(f))
	}
	s := fmt.Sprintf("%g", f)
	return s
}
