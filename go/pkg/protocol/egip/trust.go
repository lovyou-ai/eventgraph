package egip

import (
	"sync"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Inter-system trust parameters (more conservative than intra-system).
var (
	InterSystemDecayRate      = types.MustScore(0.02) // per day
	InterSystemMaxAdjustment  = types.MustWeight(0.05)
)

// Trust impact constants for inter-system actions.
const (
	TrustImpactValidProof       = 0.02
	TrustImpactReceiptOnTime    = 0.01
	TrustImpactTreatyHonoured   = 0.03
	TrustImpactTreatyViolated   = -0.15
	TrustImpactInvalidProof     = -0.10
	TrustImpactSignatureInvalid = -0.20
	TrustImpactNoHelloResponse  = -0.05
)

// PeerRecord tracks the state of a known remote system.
type PeerRecord struct {
	SystemURI    types.SystemURI
	PublicKey    types.PublicKey
	Trust        types.Score
	Capabilities []string
	NegotiatedVersion int
	LastSeen     time.Time
	FirstSeen    time.Time
}

// PeerStore manages known peer systems and their trust scores.
type PeerStore struct {
	mu    sync.RWMutex
	peers map[string]*PeerRecord // keyed by SystemURI.Value()
}

// NewPeerStore creates a new peer store.
func NewPeerStore() *PeerStore {
	return &PeerStore{
		peers: make(map[string]*PeerRecord),
	}
}

// Register adds or updates a peer from a HELLO exchange.
func (ps *PeerStore) Register(uri types.SystemURI, publicKey types.PublicKey, capabilities []string, negotiatedVersion int) *PeerRecord {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	key := uri.Value()
	now := time.Now()

	if existing, ok := ps.peers[key]; ok {
		existing.PublicKey = publicKey
		existing.Capabilities = capabilities
		existing.NegotiatedVersion = negotiatedVersion
		existing.LastSeen = now
		return existing
	}

	record := &PeerRecord{
		SystemURI:         uri,
		PublicKey:         publicKey,
		Trust:            types.MustScore(0.0),
		Capabilities:     capabilities,
		NegotiatedVersion: negotiatedVersion,
		LastSeen:          now,
		FirstSeen:         now,
	}
	ps.peers[key] = record
	return record
}

// Get returns a copy of the peer record by URI.
func (ps *PeerStore) Get(uri types.SystemURI) (PeerRecord, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	record, ok := ps.peers[uri.Value()]
	if !ok {
		return PeerRecord{}, false
	}
	return *record, true
}

// UpdateTrust adjusts a peer's trust score by the given delta, clamped to [0,1]
// and to the max single adjustment.
func (ps *PeerStore) UpdateTrust(uri types.SystemURI, delta float64) (types.Score, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	record, ok := ps.peers[uri.Value()]
	if !ok {
		return types.MustScore(0.0), false
	}

	// Clamp delta to max adjustment.
	maxAdj := InterSystemMaxAdjustment.Value()
	if delta > maxAdj {
		delta = maxAdj
	}
	if delta < -maxAdj {
		delta = -maxAdj
	}

	newVal := record.Trust.Value() + delta
	if newVal < 0.0 {
		newVal = 0.0
	}
	if newVal > 1.0 {
		newVal = 1.0
	}

	record.Trust = types.MustScore(newVal)
	record.LastSeen = time.Now()
	return record.Trust, true
}

// DecayAll applies time-based trust decay to all peers.
func (ps *PeerStore) DecayAll() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	decayPerDay := InterSystemDecayRate.Value()

	for _, record := range ps.peers {
		daysSince := now.Sub(record.LastSeen).Hours() / 24.0
		if daysSince <= 0 {
			continue
		}
		decay := decayPerDay * daysSince
		newVal := record.Trust.Value() - decay
		if newVal < 0.0 {
			newVal = 0.0
		}
		record.Trust = types.MustScore(newVal)
		record.LastSeen = now
	}
}

// All returns copies of all peer records.
func (ps *PeerStore) All() []PeerRecord {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := make([]PeerRecord, 0, len(ps.peers))
	for _, r := range ps.peers {
		result = append(result, *r)
	}
	return result
}
