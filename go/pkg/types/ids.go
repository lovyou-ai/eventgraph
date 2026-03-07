package types

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// UUID v7 format: 8-4-4-4-12 hex with version nibble = 7 and variant bits = 10xx
var uuidV7Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// General UUID format (v4 or v7): 8-4-4-4-12 hex
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// EventID identifies an event. Must be a valid UUID v7 (time-ordered).
type EventID struct{ value string }

// NewEventID creates an EventID. Returns an error if the format is not UUID v7.
func NewEventID(v string) (EventID, error) {
	v = strings.ToLower(v)
	if !uuidV7Regex.MatchString(v) {
		return EventID{}, &InvalidFormatError{Field: "EventID", Value: v, Expected: "UUID v7"}
	}
	return EventID{value: v}, nil
}

// MustEventID creates an EventID. Panics on invalid format.
func MustEventID(v string) EventID {
	id, err := NewEventID(v)
	if err != nil {
		panic(err)
	}
	return id
}

// Value returns the underlying string.
func (id EventID) Value() string { return id.value }

// String returns the string representation.
func (id EventID) String() string { return id.value }

// IsZero returns true if the EventID is the zero value (unset).
func (id EventID) IsZero() bool { return id.value == "" }

func (id EventID) MarshalJSON() ([]byte, error)  { return json.Marshal(id.value) }
func (id *EventID) UnmarshalJSON(b []byte) error  { return unmarshalID(b, &id.value, NewEventID) }

// NewEventIDFromNew generates a new UUID v7 EventID using the current time.
func NewEventIDFromNew() (EventID, error) {
	// UUID v7: 48-bit timestamp (ms) | 4-bit version (7) | 12-bit rand | 2-bit variant (10) | 62-bit rand
	now := time.Now()
	ms := now.UnixMilli()

	var b [16]byte
	// Timestamp: 48 bits (6 bytes)
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)

	// Random: fill remaining bytes
	if _, err := rand.Read(b[6:]); err != nil {
		return EventID{}, err
	}

	// Version: set high nibble of byte 6 to 0x7
	b[6] = (b[6] & 0x0f) | 0x70
	// Variant: set high bits of byte 8 to 10xx
	b[8] = (b[8] & 0x3f) | 0x80

	s := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		uint64(b[10])<<40|uint64(b[11])<<32|uint64(b[12])<<24|uint64(b[13])<<16|uint64(b[14])<<8|uint64(b[15]),
	)
	return NewEventID(s)
}

// ActorID identifies an actor. Cannot be empty.
type ActorID struct{ value string }

// NewActorID creates an ActorID. Returns an error if empty.
func NewActorID(v string) (ActorID, error) {
	if v == "" {
		return ActorID{}, &EmptyRequiredError{Field: "ActorID"}
	}
	return ActorID{value: v}, nil
}

// MustActorID creates an ActorID. Panics if empty.
func MustActorID(v string) ActorID {
	id, err := NewActorID(v)
	if err != nil {
		panic(err)
	}
	return id
}

func (id ActorID) Value() string              { return id.value }
func (id ActorID) String() string             { return id.value }
func (id ActorID) IsZero() bool               { return id.value == "" }
func (id ActorID) MarshalJSON() ([]byte, error) { return json.Marshal(id.value) }
func (id *ActorID) UnmarshalJSON(b []byte) error { return unmarshalStringID(b, &id.value, "ActorID") }

// EdgeID is the EventID of the event that created an edge.
type EdgeID struct{ value string }

// NewEdgeID creates an EdgeID. Returns an error if the format is not UUID v7.
func NewEdgeID(v string) (EdgeID, error) {
	v = strings.ToLower(v)
	if !uuidV7Regex.MatchString(v) {
		return EdgeID{}, &InvalidFormatError{Field: "EdgeID", Value: v, Expected: "UUID v7"}
	}
	return EdgeID{value: v}, nil
}

// MustEdgeID creates an EdgeID. Panics on invalid format.
func MustEdgeID(v string) EdgeID {
	id, err := NewEdgeID(v)
	if err != nil {
		panic(err)
	}
	return id
}

func (id EdgeID) Value() string              { return id.value }
func (id EdgeID) String() string             { return id.value }
func (id EdgeID) MarshalJSON() ([]byte, error) { return json.Marshal(id.value) }
func (id *EdgeID) UnmarshalJSON(b []byte) error { return unmarshalID(b, &id.value, NewEdgeID) }

// ConversationID groups related events into threads. Cannot be empty.
type ConversationID struct{ value string }

// NewConversationID creates a ConversationID. Returns an error if empty.
func NewConversationID(v string) (ConversationID, error) {
	if v == "" {
		return ConversationID{}, &EmptyRequiredError{Field: "ConversationID"}
	}
	return ConversationID{value: v}, nil
}

// MustConversationID creates a ConversationID. Panics if empty.
func MustConversationID(v string) ConversationID {
	id, err := NewConversationID(v)
	if err != nil {
		panic(err)
	}
	return id
}

func (id ConversationID) Value() string              { return id.value }
func (id ConversationID) String() string             { return id.value }
func (id ConversationID) MarshalJSON() ([]byte, error) { return json.Marshal(id.value) }
func (id *ConversationID) UnmarshalJSON(b []byte) error {
	return unmarshalStringID(b, &id.value, "ConversationID")
}

// Hash is a SHA-256 hex string (64 characters).
type Hash struct{ value string }

// NewHash creates a Hash. Returns an error if not exactly 64 hex characters.
func NewHash(v string) (Hash, error) {
	v = strings.ToLower(v)
	if len(v) != 64 {
		return Hash{}, &InvalidFormatError{Field: "Hash", Value: v, Expected: "64 hex characters (SHA-256)"}
	}
	if _, err := hex.DecodeString(v); err != nil {
		return Hash{}, &InvalidFormatError{Field: "Hash", Value: v, Expected: "64 hex characters (SHA-256)"}
	}
	return Hash{value: v}, nil
}

// MustHash creates a Hash. Panics on invalid format.
func MustHash(v string) Hash {
	h, err := NewHash(v)
	if err != nil {
		panic(err)
	}
	return h
}

// ZeroHash returns the all-zeros hash used as PrevHash for the bootstrap event.
func ZeroHash() Hash {
	return Hash{value: strings.Repeat("0", 64)}
}

func (h Hash) Value() string              { return h.value }
func (h Hash) String() string             { return h.value }
func (h Hash) IsZero() bool               { return h.value == "" || h.value == strings.Repeat("0", 64) }
func (h Hash) MarshalJSON() ([]byte, error) { return json.Marshal(h.value) }
func (h *Hash) UnmarshalJSON(b []byte) error { return unmarshalID(b, &h.value, NewHash) }

// SystemURI identifies a remote system in EGIP.
type SystemURI struct{ value string }

// NewSystemURI creates a SystemURI. Returns an error if empty.
func NewSystemURI(v string) (SystemURI, error) {
	if v == "" {
		return SystemURI{}, &EmptyRequiredError{Field: "SystemURI"}
	}
	return SystemURI{value: v}, nil
}

// MustSystemURI creates a SystemURI. Panics if empty.
func MustSystemURI(v string) SystemURI {
	id, err := NewSystemURI(v)
	if err != nil {
		panic(err)
	}
	return id
}

func (id SystemURI) Value() string              { return id.value }
func (id SystemURI) String() string             { return id.value }
func (id SystemURI) MarshalJSON() ([]byte, error) { return json.Marshal(id.value) }
func (id *SystemURI) UnmarshalJSON(b []byte) error { return unmarshalStringID(b, &id.value, "SystemURI") }

// EnvelopeID identifies an EGIP message. Must be a valid UUID.
type EnvelopeID struct{ value string }

// NewEnvelopeID creates an EnvelopeID. Returns an error if not a valid UUID.
func NewEnvelopeID(v string) (EnvelopeID, error) {
	v = strings.ToLower(v)
	if !uuidRegex.MatchString(v) {
		return EnvelopeID{}, &InvalidFormatError{Field: "EnvelopeID", Value: v, Expected: "UUID"}
	}
	return EnvelopeID{value: v}, nil
}

// MustEnvelopeID creates an EnvelopeID. Panics on invalid format.
func MustEnvelopeID(v string) EnvelopeID {
	id, err := NewEnvelopeID(v)
	if err != nil {
		panic(err)
	}
	return id
}

func (id EnvelopeID) Value() string              { return id.value }
func (id EnvelopeID) String() string             { return id.value }
func (id EnvelopeID) MarshalJSON() ([]byte, error) { return json.Marshal(id.value) }
func (id *EnvelopeID) UnmarshalJSON(b []byte) error { return unmarshalID(b, &id.value, NewEnvelopeID) }

// TreatyID identifies a bilateral treaty. Must be a valid UUID.
type TreatyID struct{ value string }

// NewTreatyID creates a TreatyID. Returns an error if not a valid UUID.
func NewTreatyID(v string) (TreatyID, error) {
	v = strings.ToLower(v)
	if !uuidRegex.MatchString(v) {
		return TreatyID{}, &InvalidFormatError{Field: "TreatyID", Value: v, Expected: "UUID"}
	}
	return TreatyID{value: v}, nil
}

// MustTreatyID creates a TreatyID. Panics on invalid format.
func MustTreatyID(v string) TreatyID {
	id, err := NewTreatyID(v)
	if err != nil {
		panic(err)
	}
	return id
}

func (id TreatyID) Value() string              { return id.value }
func (id TreatyID) String() string             { return id.value }
func (id TreatyID) MarshalJSON() ([]byte, error) { return json.Marshal(id.value) }
func (id *TreatyID) UnmarshalJSON(b []byte) error { return unmarshalID(b, &id.value, NewTreatyID) }

// PrimitiveID identifies a primitive instance. Cannot be empty.
type PrimitiveID struct{ value string }

// NewPrimitiveID creates a PrimitiveID. Returns an error if empty.
func NewPrimitiveID(v string) (PrimitiveID, error) {
	if v == "" {
		return PrimitiveID{}, &EmptyRequiredError{Field: "PrimitiveID"}
	}
	return PrimitiveID{value: v}, nil
}

// MustPrimitiveID creates a PrimitiveID. Panics if empty.
func MustPrimitiveID(v string) PrimitiveID {
	id, err := NewPrimitiveID(v)
	if err != nil {
		panic(err)
	}
	return id
}

func (id PrimitiveID) Value() string              { return id.value }
func (id PrimitiveID) String() string             { return id.value }
func (id PrimitiveID) MarshalJSON() ([]byte, error) { return json.Marshal(id.value) }
func (id *PrimitiveID) UnmarshalJSON(b []byte) error {
	return unmarshalStringID(b, &id.value, "PrimitiveID")
}

// EventType is a registered event type string (e.g., "trust.updated").
// Format-validated only at construction; registry validation happens at the factory level.
type EventType struct{ value string }

// eventTypeRegex validates dot-separated lowercase segments.
var eventTypeRegex = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*$`)

// NewEventType creates an EventType. Validates format only (dot-separated lowercase).
func NewEventType(v string) (EventType, error) {
	if !eventTypeRegex.MatchString(v) {
		return EventType{}, &InvalidFormatError{Field: "EventType", Value: v, Expected: "dot-separated lowercase segments (e.g., trust.updated)"}
	}
	return EventType{value: v}, nil
}

// MustEventType creates an EventType. Panics on invalid format.
func MustEventType(v string) EventType {
	et, err := NewEventType(v)
	if err != nil {
		panic(err)
	}
	return et
}

func (et EventType) Value() string              { return et.value }
func (et EventType) String() string             { return et.value }
func (et EventType) MarshalJSON() ([]byte, error) { return json.Marshal(et.value) }
func (et *EventType) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	t, err := NewEventType(v)
	if err != nil {
		return err
	}
	*et = t
	return nil
}

// SubscriptionPattern is a glob pattern for matching event types (e.g., "trust.*", "*").
type SubscriptionPattern struct{ value string }

// subscriptionPatternRegex validates dot-separated segments with optional * wildcard.
var subscriptionPatternRegex = regexp.MustCompile(`^(\*|[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*(\.\*)?)$`)

// NewSubscriptionPattern creates a SubscriptionPattern.
func NewSubscriptionPattern(v string) (SubscriptionPattern, error) {
	if !subscriptionPatternRegex.MatchString(v) {
		return SubscriptionPattern{}, &InvalidFormatError{
			Field:    "SubscriptionPattern",
			Value:    v,
			Expected: "dot-separated segments with optional trailing .* or bare *",
		}
	}
	return SubscriptionPattern{value: v}, nil
}

// MustSubscriptionPattern creates a SubscriptionPattern. Panics on invalid format.
func MustSubscriptionPattern(v string) SubscriptionPattern {
	sp, err := NewSubscriptionPattern(v)
	if err != nil {
		panic(err)
	}
	return sp
}

// Matches returns true if the given event type matches this pattern.
func (sp SubscriptionPattern) Matches(et EventType) bool {
	if sp.value == "*" {
		return true
	}
	if strings.HasSuffix(sp.value, ".*") {
		prefix := strings.TrimSuffix(sp.value, ".*")
		return et.value == prefix || strings.HasPrefix(et.value, prefix+".")
	}
	return sp.value == et.value
}

func (sp SubscriptionPattern) Value() string              { return sp.value }
func (sp SubscriptionPattern) String() string             { return sp.value }
func (sp SubscriptionPattern) MarshalJSON() ([]byte, error) { return json.Marshal(sp.value) }
func (sp *SubscriptionPattern) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p, err := NewSubscriptionPattern(v)
	if err != nil {
		return err
	}
	*sp = p
	return nil
}

// DomainScope is a trust/authority domain (e.g., "code_review", "financial").
type DomainScope struct{ value string }

// domainScopeRegex validates lowercase dot-separated or underscore-separated namespace.
var domainScopeRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)

// NewDomainScope creates a DomainScope.
func NewDomainScope(v string) (DomainScope, error) {
	if !domainScopeRegex.MatchString(v) {
		return DomainScope{}, &InvalidFormatError{
			Field:    "DomainScope",
			Value:    v,
			Expected: "lowercase dot-separated namespace (e.g., code_review)",
		}
	}
	return DomainScope{value: v}, nil
}

// MustDomainScope creates a DomainScope. Panics on invalid format.
func MustDomainScope(v string) DomainScope {
	ds, err := NewDomainScope(v)
	if err != nil {
		panic(err)
	}
	return ds
}

func (ds DomainScope) Value() string              { return ds.value }
func (ds DomainScope) String() string             { return ds.value }
func (ds DomainScope) MarshalJSON() ([]byte, error) { return json.Marshal(ds.value) }
func (ds *DomainScope) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	d, err := NewDomainScope(v)
	if err != nil {
		return err
	}
	*ds = d
	return nil
}

// PublicKey is an Ed25519 public key (32 bytes).
type PublicKey struct{ value []byte }

// NewPublicKey creates a PublicKey. Returns an error if not exactly 32 bytes.
func NewPublicKey(v []byte) (PublicKey, error) {
	if len(v) != 32 {
		return PublicKey{}, &InvalidFormatError{
			Field:    "PublicKey",
			Value:    hex.EncodeToString(v),
			Expected: "32 bytes (Ed25519 public key)",
		}
	}
	cp := make([]byte, 32)
	copy(cp, v)
	return PublicKey{value: cp}, nil
}

// MustPublicKey creates a PublicKey. Panics if not 32 bytes.
func MustPublicKey(v []byte) PublicKey {
	pk, err := NewPublicKey(v)
	if err != nil {
		panic(err)
	}
	return pk
}

// Bytes returns a copy of the underlying bytes.
func (pk PublicKey) Bytes() []byte {
	cp := make([]byte, len(pk.value))
	copy(cp, pk.value)
	return cp
}

// String returns the hex-encoded representation.
func (pk PublicKey) String() string { return hex.EncodeToString(pk.value) }

func (pk PublicKey) MarshalJSON() ([]byte, error) { return json.Marshal(hex.EncodeToString(pk.value)) }
func (pk *PublicKey) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	decoded, err := hex.DecodeString(s)
	if err != nil {
		return &InvalidFormatError{Field: "PublicKey", Value: s, Expected: "hex-encoded 32 bytes"}
	}
	key, err := NewPublicKey(decoded)
	if err != nil {
		return err
	}
	*pk = key
	return nil
}

// Signature is an Ed25519 signature (64 bytes).
type Signature struct{ value []byte }

// NewSignature creates a Signature. Returns an error if not exactly 64 bytes.
func NewSignature(v []byte) (Signature, error) {
	if len(v) != 64 {
		return Signature{}, &InvalidFormatError{
			Field:    "Signature",
			Value:    hex.EncodeToString(v),
			Expected: "64 bytes (Ed25519 signature)",
		}
	}
	cp := make([]byte, 64)
	copy(cp, v)
	return Signature{value: cp}, nil
}

// MustSignature creates a Signature. Panics if not 64 bytes.
func MustSignature(v []byte) Signature {
	sig, err := NewSignature(v)
	if err != nil {
		panic(err)
	}
	return sig
}

// Bytes returns a copy of the underlying bytes.
func (s Signature) Bytes() []byte {
	cp := make([]byte, len(s.value))
	copy(cp, s.value)
	return cp
}

// String returns the hex-encoded representation.
func (s Signature) String() string { return hex.EncodeToString(s.value) }

func (s Signature) MarshalJSON() ([]byte, error) { return json.Marshal(hex.EncodeToString(s.value)) }
func (s *Signature) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	decoded, err := hex.DecodeString(str)
	if err != nil {
		return &InvalidFormatError{Field: "Signature", Value: str, Expected: "hex-encoded 64 bytes"}
	}
	sig, err := NewSignature(decoded)
	if err != nil {
		return err
	}
	*s = sig
	return nil
}

// --- unmarshal helpers ---

// unmarshalID is a generic helper for ID types with a constructor that takes and returns the same type.
// Normalizes to lowercase before validation and storage to ensure consistent equality.
func unmarshalID[T any](b []byte, target *string, constructor func(string) (T, error)) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	v = strings.ToLower(v)
	if _, err := constructor(v); err != nil {
		return err
	}
	*target = v
	return nil
}

// unmarshalStringID is a helper for simple string IDs that just need non-empty validation.
func unmarshalStringID(b []byte, target *string, field string) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v == "" {
		return &EmptyRequiredError{Field: field}
	}
	*target = v
	return nil
}
