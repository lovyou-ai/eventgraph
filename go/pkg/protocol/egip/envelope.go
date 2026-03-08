package egip

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Envelope is the signed message container for all EGIP communication.
type Envelope struct {
	ProtocolVersion int
	ID              types.EnvelopeID
	From            types.SystemURI
	To              types.SystemURI
	Type            event.MessageType
	Payload         MessagePayload
	Timestamp       time.Time
	Signature       types.Signature
	InReplyTo       types.Option[types.EnvelopeID]
}

// MessagePayload is the interface for all EGIP message payloads.
type MessagePayload interface {
	messagePayload()
}

// CanonicalForm returns the canonical string representation for signing.
func (e *Envelope) CanonicalForm() (string, error) {
	payloadJSON, err := json.Marshal(e.Payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	// Round-trip to normalize nested key order (sorted) and whitespace.
	// Uses any because payloads may contain nested maps from json.RawMessage
	// fields whose key order is not deterministic from the source.
	var raw any
	if err := json.Unmarshal(payloadJSON, &raw); err != nil {
		return "", fmt.Errorf("unmarshal payload for canonical form: %w", err)
	}
	canonical, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("re-marshal canonical payload: %w", err)
	}

	msgType := strings.ToLower(string(e.Type))
	nanos := strconv.FormatInt(e.Timestamp.UnixNano(), 10)

	inReplyTo := ""
	if e.InReplyTo.IsSome() {
		inReplyTo = e.InReplyTo.Unwrap().Value()
	}

	return fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%s",
		e.ProtocolVersion,
		e.ID.Value(),
		e.From.Value(),
		e.To.Value(),
		msgType,
		nanos,
		inReplyTo,
		string(canonical),
	), nil
}

// SignEnvelope signs the envelope using the given identity and returns a new envelope with the signature set.
func SignEnvelope(env *Envelope, identity IIdentity) (*Envelope, error) {
	canonical, err := env.CanonicalForm()
	if err != nil {
		return nil, fmt.Errorf("canonical form: %w", err)
	}

	sig, err := identity.Sign([]byte(canonical))
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}

	signed := *env
	signed.Signature = sig
	return &signed, nil
}

// VerifyEnvelope verifies the envelope's signature against the given public key.
func VerifyEnvelope(env *Envelope, identity IIdentity, publicKey types.PublicKey) (bool, error) {
	canonical, err := env.CanonicalForm()
	if err != nil {
		return false, fmt.Errorf("canonical form: %w", err)
	}

	return identity.Verify(publicKey, []byte(canonical), env.Signature)
}

// --- Payload Types ---

// HelloPayload is the payload for HELLO messages.
type HelloPayload struct {
	SystemURI        types.SystemURI  `json:"system_uri"`
	PublicKey        types.PublicKey   `json:"public_key"`
	ProtocolVersions []int            `json:"protocol_versions"`
	Capabilities     []string         `json:"capabilities"`
	ChainLength      int              `json:"chain_length"`
}

func (p HelloPayload) messagePayload() {}

// MessagePayloadContent is the payload for MESSAGE messages.
type MessagePayloadContent struct {
	Content        event.EventContent             `json:"-"`
	ContentJSON    json.RawMessage                `json:"content"`
	ContentType    types.EventType                `json:"content_type"`
	ConversationID types.Option[types.ConversationID] `json:"conversation_id"`
	CGERs          []CGER                         `json:"cgers,omitempty"`
}

func (p MessagePayloadContent) messagePayload() {}

// ReceiptPayload is the payload for RECEIPT messages.
type ReceiptPayload struct {
	EnvelopeID   types.EnvelopeID              `json:"envelope_id"`
	Status       event.ReceiptStatus           `json:"status"`
	LocalEventID types.Option[types.EventID]   `json:"local_event_id"`
	Reason       types.Option[string]          `json:"reason"`
	Signature    types.Signature               `json:"signature"`
}

func (p ReceiptPayload) messagePayload() {}

// ProofPayload is the payload for PROOF messages.
type ProofPayload struct {
	ProofType event.ProofType `json:"proof_type"`
	Data      ProofData       `json:"data"`
}

func (p ProofPayload) messagePayload() {}

// ProofData is the interface for proof-type-specific data.
type ProofData interface {
	proofData()
}

// ChainSegmentProof is a contiguous portion of the hash chain.
type ChainSegmentProof struct {
	Events    []event.Event `json:"events"`
	StartHash types.Hash    `json:"start_hash"`
	EndHash   types.Hash    `json:"end_hash"`
}

func (p ChainSegmentProof) proofData() {}

// EventExistenceProof proves a specific event exists in the chain.
type EventExistenceProof struct {
	Event       event.Event              `json:"event"`
	PrevHash    types.Hash               `json:"prev_hash"`
	NextHash    types.Option[types.Hash] `json:"next_hash"`
	Position    int                      `json:"position"`
	ChainLength int                      `json:"chain_length"`
}

func (p EventExistenceProof) proofData() {}

// ChainSummaryProof is a high-level integrity attestation.
type ChainSummaryProof struct {
	Length      int        `json:"length"`
	HeadHash    types.Hash `json:"head_hash"`
	GenesisHash types.Hash `json:"genesis_hash"`
	Timestamp   time.Time  `json:"timestamp"`
}

func (p ChainSummaryProof) proofData() {}

// TreatyPayload is the payload for TREATY messages.
type TreatyPayload struct {
	TreatyID types.TreatyID       `json:"treaty_id"`
	Action   event.TreatyAction   `json:"action"`
	Terms    []TreatyTerm         `json:"terms,omitempty"`
	Reason   types.Option[string] `json:"reason"`
}

func (p TreatyPayload) messagePayload() {}

// TreatyTerm defines a single term of a bilateral treaty.
type TreatyTerm struct {
	Scope     types.DomainScope `json:"scope"`
	Policy    string            `json:"policy"`
	Symmetric bool              `json:"symmetric"`
}

// AuthorityRequestPayload is the payload for AUTHORITY_REQUEST messages.
type AuthorityRequestPayload struct {
	Action        types.DomainScope            `json:"action"`
	Actor         types.ActorID                `json:"actor"`
	Level         event.AuthorityLevel         `json:"level"`
	Justification string                       `json:"justification"`
	TreatyID      types.Option[types.TreatyID] `json:"treaty_id"`
}

func (p AuthorityRequestPayload) messagePayload() {}

// DiscoverPayload is the payload for DISCOVER messages.
type DiscoverPayload struct {
	Query   DiscoverQuery    `json:"query"`
	Results []DiscoverResult `json:"results,omitempty"`
}

func (p DiscoverPayload) messagePayload() {}

// DiscoverQuery specifies what capabilities to search for.
type DiscoverQuery struct {
	Capabilities []string              `json:"capabilities"`
	MinTrust     types.Option[types.Score] `json:"min_trust"`
}

// DiscoverResult is a single discovery result.
type DiscoverResult struct {
	SystemURI    types.SystemURI `json:"system_uri"`
	PublicKey    types.PublicKey  `json:"public_key"`
	Capabilities []string        `json:"capabilities"`
	TrustScore   types.Score     `json:"trust_score"`
}

// --- CGER (enhanced) ---

// CGER represents a cross-graph event reference with verification tracking.
type CGER struct {
	LocalEventID types.EventID        `json:"local_event_id"`
	RemoteSystem types.SystemURI      `json:"remote_system"`
	RemoteEventID string              `json:"remote_event_id"`
	RemoteHash   types.Hash           `json:"remote_hash"`
	Relationship event.CGERRelationship `json:"relationship"`
	Verified     bool                 `json:"verified"`
}

// NegotiateVersion finds the highest protocol version both systems support.
// Returns None if no common version exists.
func NegotiateVersion(local, remote []int) types.Option[int] {
	best := -1
	for _, l := range local {
		for _, r := range remote {
			if l == r && l > best {
				best = l
			}
		}
	}
	if best < 0 {
		return types.None[int]()
	}
	return types.Some(best)
}
