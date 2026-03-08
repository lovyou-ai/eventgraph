package egip

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// CurrentProtocolVersion is the EGIP protocol version this implementation supports.
const CurrentProtocolVersion = 1

// MaxEnvelopeAge is the maximum age of an incoming envelope before it is
// rejected as stale. This bounds the replay window even after dedup entries
// are pruned.
const MaxEnvelopeAge = 25 * time.Hour

// Handler orchestrates EGIP protocol interactions: HELLO handshake,
// message dispatch, replay deduplication, and trust updates.
type Handler struct {
	identity  IIdentity
	transport ITransport
	peers     *PeerStore
	treaties  *TreatyStore
	dedup     *EnvelopeDedup

	// LocalProtocolVersions lists the protocol versions this system supports.
	LocalProtocolVersions []int

	// Capabilities lists what this system can do.
	Capabilities []string

	// ChainLength returns the current chain length (for HELLO).
	ChainLength func() (int, error)

	// OnMessage is called when a verified MESSAGE envelope arrives.
	// The handler verifies the signature and deduplicates before calling this.
	OnMessage func(from types.SystemURI, payload *MessagePayloadContent) error

	// OnAuthorityRequest is called when a verified AUTHORITY_REQUEST arrives.
	OnAuthorityRequest func(from types.SystemURI, payload *AuthorityRequestPayload) error

	// OnDiscover is called when a verified DISCOVER query arrives.
	// Returns results to send back.
	OnDiscover func(from types.SystemURI, query DiscoverQuery) ([]DiscoverResult, error)
}

// NewHandler creates a new protocol handler.
func NewHandler(identity IIdentity, transport ITransport, peers *PeerStore, treaties *TreatyStore) *Handler {
	return &Handler{
		identity:              identity,
		transport:             transport,
		peers:                 peers,
		treaties:              treaties,
		dedup:                 NewEnvelopeDedup(),
		LocalProtocolVersions: []int{CurrentProtocolVersion},
		Capabilities:          []string{"treaty", "proof"},
	}
}

// Hello performs the HELLO handshake with a remote system.
// Sends a HELLO and expects a HELLO response. The peer is not registered
// until the response HELLO arrives via HandleIncoming — this avoids storing
// a partially-initialised record with a zero PublicKey.
func (h *Handler) Hello(ctx context.Context, to types.SystemURI) error {
	chainLen := 0
	if h.ChainLength != nil {
		var err error
		chainLen, err = h.ChainLength()
		if err != nil {
			return fmt.Errorf("chain length: %w", err)
		}
	}

	uuid, err := generateUUID4()
	if err != nil {
		return fmt.Errorf("generate envelope ID: %w", err)
	}
	envID, err := types.NewEnvelopeID(uuid)
	if err != nil {
		return fmt.Errorf("create envelope ID: %w", err)
	}

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              envID,
		From:            h.identity.SystemURI(),
		To:              to,
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        h.identity.SystemURI(),
			PublicKey:        h.identity.PublicKey(),
			ProtocolVersions: h.LocalProtocolVersions,
			Capabilities:     h.Capabilities,
			ChainLength:      chainLen,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, err := SignEnvelope(env, h.identity)
	if err != nil {
		return fmt.Errorf("sign hello: %w", err)
	}

	receipt, err := h.transport.Send(ctx, to, signed)
	if err != nil {
		h.peers.UpdateTrust(to, TrustImpactNoHelloResponse)
		return &TransportFailureError{To: to, Reason: err.Error()}
	}

	if receipt != nil && receipt.Status == event.ReceiptStatusRejected {
		reason := ""
		if receipt.Reason.IsSome() {
			reason = receipt.Reason.Unwrap()
		}
		return fmt.Errorf("hello rejected: %s", reason)
	}

	return nil
}

// HandleIncoming processes an incoming envelope: checks timestamp freshness,
// deduplicates, verifies signature, dispatches to the appropriate handler,
// and updates trust.
func (h *Handler) HandleIncoming(ctx context.Context, env *Envelope) error {
	// Timestamp freshness — reject stale envelopes even if dedup has pruned them.
	age := time.Since(env.Timestamp)
	if age > MaxEnvelopeAge || age < -5*time.Minute {
		return fmt.Errorf("envelope timestamp out of range: age %v", age)
	}

	// Replay deduplication.
	if !h.dedup.Check(env.ID) {
		return &DuplicateEnvelopeError{EnvelopeID: env.ID}
	}

	// Look up the sender's public key.
	peer, known := h.peers.Get(env.From)

	// For HELLO messages, use the public key from the payload (TOFU model).
	// Note: this is Trust-On-First-Use. The first HELLO from a URI establishes
	// the key binding. Callers who need authenticated introduction should
	// pre-load keys into PeerStore before communication begins.
	var pubKey types.PublicKey
	if env.Type == event.MessageTypeHello {
		hello, ok := env.Payload.(HelloPayload)
		if !ok {
			return fmt.Errorf("invalid hello payload type: %T", env.Payload)
		}
		pubKey = hello.PublicKey
	} else {
		if !known {
			return &SystemNotFoundError{URI: env.From}
		}
		pubKey = peer.PublicKey
	}

	// Verify signature.
	valid, err := VerifyEnvelope(env, h.identity, pubKey)
	if err != nil {
		return fmt.Errorf("verify envelope: %w", err)
	}
	if !valid {
		h.peers.UpdateTrust(env.From, TrustImpactSignatureInvalid)
		return &EnvelopeSignatureInvalidError{EnvelopeID: env.ID}
	}

	// Dispatch by message type.
	switch env.Type {
	case event.MessageTypeHello:
		return h.handleHello(env)
	case event.MessageTypeMessage:
		return h.handleMessage(env)
	case event.MessageTypeReceipt:
		return h.handleReceipt(env)
	case event.MessageTypeProof:
		return h.handleProof(env)
	case event.MessageTypeTreaty:
		return h.handleTreaty(env)
	case event.MessageTypeAuthorityRequest:
		return h.handleAuthorityRequest(env)
	case event.MessageTypeDiscover:
		return h.handleDiscover(ctx, env)
	default:
		return fmt.Errorf("unknown message type: %s", env.Type)
	}
}

func (h *Handler) handleHello(env *Envelope) error {
	hello, ok := env.Payload.(HelloPayload)
	if !ok {
		return fmt.Errorf("invalid hello payload type: %T", env.Payload)
	}

	// Negotiate protocol version.
	version := NegotiateVersion(h.LocalProtocolVersions, hello.ProtocolVersions)
	if !version.IsSome() {
		return &VersionIncompatibleError{
			Local:  h.LocalProtocolVersions,
			Remote: hello.ProtocolVersions,
		}
	}

	// Register or update the peer.
	h.peers.Register(hello.SystemURI, hello.PublicKey, hello.Capabilities, version.Unwrap())
	return nil
}

func (h *Handler) handleMessage(env *Envelope) error {
	msg, ok := env.Payload.(MessagePayloadContent)
	if !ok {
		return fmt.Errorf("invalid message payload type: %T", env.Payload)
	}

	h.peers.UpdateTrust(env.From, TrustImpactReceiptOnTime)

	if h.OnMessage != nil {
		return h.OnMessage(env.From, &msg)
	}
	return nil
}

func (h *Handler) handleReceipt(env *Envelope) error {
	receipt, ok := env.Payload.(ReceiptPayload)
	if !ok {
		return fmt.Errorf("invalid receipt payload type: %T", env.Payload)
	}

	if receipt.Status == event.ReceiptStatusProcessed || receipt.Status == event.ReceiptStatusDelivered {
		h.peers.UpdateTrust(env.From, TrustImpactReceiptOnTime)
	}
	return nil
}

func (h *Handler) handleProof(env *Envelope) error {
	proof, ok := env.Payload.(ProofPayload)
	if !ok {
		return fmt.Errorf("invalid proof payload type: %T", env.Payload)
	}

	valid, err := ValidateProof(&proof)
	if err != nil {
		return fmt.Errorf("validate proof: %w", err)
	}

	if valid {
		h.peers.UpdateTrust(env.From, TrustImpactValidProof)
	} else {
		h.peers.UpdateTrust(env.From, TrustImpactInvalidProof)
	}
	return nil
}

func (h *Handler) handleTreaty(env *Envelope) error {
	payload, ok := env.Payload.(TreatyPayload)
	if !ok {
		return fmt.Errorf("invalid treaty payload type: %T", env.Payload)
	}

	switch payload.Action {
	case event.TreatyActionPropose:
		treaty := NewTreaty(payload.TreatyID, env.From, env.To, payload.Terms)
		h.treaties.Put(treaty)
		return nil

	case event.TreatyActionAccept:
		return h.treaties.Apply(payload.TreatyID, func(treaty *Treaty) error {
			if err := treaty.ApplyAction(event.TreatyActionAccept); err != nil {
				return err
			}
			h.peers.UpdateTrust(env.From, TrustImpactTreatyHonoured)
			return nil
		})

	case event.TreatyActionSuspend:
		return h.treaties.Apply(payload.TreatyID, func(treaty *Treaty) error {
			return treaty.ApplyAction(event.TreatyActionSuspend)
		})

	case event.TreatyActionTerminate:
		return h.treaties.Apply(payload.TreatyID, func(treaty *Treaty) error {
			return treaty.ApplyAction(event.TreatyActionTerminate)
		})

	case event.TreatyActionModify:
		return h.treaties.Apply(payload.TreatyID, func(treaty *Treaty) error {
			if err := treaty.ApplyAction(event.TreatyActionModify); err != nil {
				return err
			}
			treaty.Terms = payload.Terms
			return nil
		})

	default:
		return fmt.Errorf("unknown treaty action: %s", payload.Action)
	}
}

func (h *Handler) handleAuthorityRequest(env *Envelope) error {
	payload, ok := env.Payload.(AuthorityRequestPayload)
	if !ok {
		return fmt.Errorf("invalid authority request payload type: %T", env.Payload)
	}

	if h.OnAuthorityRequest != nil {
		return h.OnAuthorityRequest(env.From, &payload)
	}
	return nil
}

func (h *Handler) handleDiscover(ctx context.Context, env *Envelope) error {
	payload, ok := env.Payload.(DiscoverPayload)
	if !ok {
		return fmt.Errorf("invalid discover payload type: %T", env.Payload)
	}

	if h.OnDiscover == nil {
		return nil
	}

	results, err := h.OnDiscover(env.From, payload.Query)
	if err != nil {
		return fmt.Errorf("discover handler: %w", err)
	}

	// Send discovery results back to the querying system.
	uuid, err := generateUUID4()
	if err != nil {
		return fmt.Errorf("generate response envelope ID: %w", err)
	}
	respID, err := types.NewEnvelopeID(uuid)
	if err != nil {
		return fmt.Errorf("create response envelope ID: %w", err)
	}

	resp := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              respID,
		From:            h.identity.SystemURI(),
		To:              env.From,
		Type:            event.MessageTypeDiscover,
		Payload: DiscoverPayload{
			Query:   payload.Query,
			Results: results,
		},
		Timestamp: time.Now(),
		InReplyTo: types.Some(env.ID),
	}

	signed, err := SignEnvelope(resp, h.identity)
	if err != nil {
		return fmt.Errorf("sign discover response: %w", err)
	}

	_, err = h.transport.Send(ctx, env.From, signed)
	if err != nil {
		return fmt.Errorf("send discover response: %w", err)
	}

	return nil
}

// --- Treaty Store ---

// TreatyStore manages bilateral treaties.
type TreatyStore struct {
	mu       sync.RWMutex
	treaties map[string]*Treaty // keyed by TreatyID.Value()
}

// NewTreatyStore creates a new treaty store.
func NewTreatyStore() *TreatyStore {
	return &TreatyStore{
		treaties: make(map[string]*Treaty),
	}
}

// Put stores or updates a treaty.
func (ts *TreatyStore) Put(treaty *Treaty) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.treaties[treaty.ID.Value()] = treaty
}

// Get returns a treaty by ID. Returns a copy and whether it was found.
func (ts *TreatyStore) Get(id types.TreatyID) (*Treaty, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	t, ok := ts.treaties[id.Value()]
	if !ok {
		return nil, false
	}
	cp := *t
	cp.Terms = append([]TreatyTerm(nil), t.Terms...)
	return &cp, true
}

// Apply performs a read-modify-write on a treaty under a single write lock.
// This prevents race conditions from concurrent Get/mutate/Put sequences.
func (ts *TreatyStore) Apply(id types.TreatyID, fn func(*Treaty) error) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	t, ok := ts.treaties[id.Value()]
	if !ok {
		return &TreatyNotFoundError{TreatyID: id}
	}

	return fn(t)
}

// BySystem returns all treaties involving a given system URI.
func (ts *TreatyStore) BySystem(uri types.SystemURI) []*Treaty {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var result []*Treaty
	for _, t := range ts.treaties {
		if t.SystemA.Value() == uri.Value() || t.SystemB.Value() == uri.Value() {
			cp := *t
			cp.Terms = append([]TreatyTerm(nil), t.Terms...)
			result = append(result, &cp)
		}
	}
	return result
}

// Active returns all active treaties.
func (ts *TreatyStore) Active() []*Treaty {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var result []*Treaty
	for _, t := range ts.treaties {
		if t.Status == event.TreatyStatusActive {
			cp := *t
			cp.Terms = append([]TreatyTerm(nil), t.Terms...)
			result = append(result, &cp)
		}
	}
	return result
}

// --- Envelope Dedup ---

// dedupPruneInterval controls how often Check auto-prunes expired entries.
const dedupPruneInterval = 1000

// EnvelopeDedup provides replay protection by tracking seen envelope IDs.
// Auto-prunes expired entries every dedupPruneInterval Check calls.
type EnvelopeDedup struct {
	mu       sync.Mutex
	seen     map[string]time.Time
	ttl      time.Duration
	checkCnt atomic.Int64
}

// NewEnvelopeDedup creates a dedup tracker with a 24-hour TTL.
func NewEnvelopeDedup() *EnvelopeDedup {
	return &EnvelopeDedup{
		seen: make(map[string]time.Time),
		ttl:  MaxEnvelopeAge + time.Hour, // Must exceed MaxEnvelopeAge to prevent replay gap.
	}
}

// NewEnvelopeDedupWithTTL creates a dedup tracker with a custom TTL.
func NewEnvelopeDedupWithTTL(ttl time.Duration) *EnvelopeDedup {
	return &EnvelopeDedup{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}
}

// Check returns true if the envelope ID has not been seen before.
// Records the ID and returns false on subsequent calls.
// Periodically prunes expired entries to prevent unbounded growth.
func (d *EnvelopeDedup) Check(id types.EnvelopeID) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := id.Value()
	if _, exists := d.seen[key]; exists {
		return false
	}

	d.seen[key] = time.Now()

	// Auto-prune on schedule.
	if d.checkCnt.Add(1)%dedupPruneInterval == 0 {
		d.pruneLocked()
	}

	return true
}

// Prune removes expired entries older than TTL.
func (d *EnvelopeDedup) Prune() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pruneLocked()
}

func (d *EnvelopeDedup) pruneLocked() int {
	cutoff := time.Now().Add(-d.ttl)
	removed := 0
	for key, ts := range d.seen {
		if ts.Before(cutoff) {
			delete(d.seen, key)
			removed++
		}
	}
	return removed
}

// Size returns the number of tracked envelope IDs.
func (d *EnvelopeDedup) Size() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.seen)
}

// generateUUID4 creates a random UUID v4 string.
func generateUUID4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
