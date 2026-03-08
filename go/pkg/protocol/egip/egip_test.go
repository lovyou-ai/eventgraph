package egip

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// --- Test helpers ---

type testSigner struct{}

func (s testSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data)
	return types.MustSignature(sig), nil
}

type headFromStore struct{ s store.Store }

func (h headFromStore) Head() (types.Option[event.Event], error) { return h.s.Head() }

func makeBootstrapEvent(t *testing.T) event.Event {
	t.Helper()
	registry := event.DefaultRegistry()
	factory := event.NewBootstrapFactory(registry)
	ev, err := factory.Init(
		types.MustActorID("actor_00000000000000000000000000000001"),
		testSigner{},
	)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}
	return ev
}

func makeChainedEvent(t *testing.T, s store.Store, causes []types.EventID) event.Event {
	t.Helper()
	registry := event.DefaultRegistry()
	factory := event.NewEventFactory(registry)
	ev, err := factory.Create(
		types.MustEventType("trust.updated"),
		types.MustActorID("actor_00000000000000000000000000000001"),
		event.TrustUpdatedContent{
			Actor:    types.MustActorID("actor_00000000000000000000000000000002"),
			Previous: types.MustScore(0.5),
			Current:  types.MustScore(0.6),
			Domain:   types.MustDomainScope("test"),
			Cause:    causes[0],
		},
		causes,
		types.MustConversationID("conv_00000000000000000000000000000001"),
		headFromStore{s},
		testSigner{},
	)
	if err != nil {
		t.Fatalf("create event failed: %v", err)
	}
	return ev
}

func setupStoreWithEvents(t *testing.T, count int) (store.Store, []event.Event) {
	t.Helper()
	s := store.NewInMemoryStore()
	bootstrap := makeBootstrapEvent(t)
	if _, err := s.Append(bootstrap); err != nil {
		t.Fatalf("append bootstrap: %v", err)
	}
	events := []event.Event{bootstrap}

	for i := 1; i < count; i++ {
		ev := makeChainedEvent(t, s, []types.EventID{events[i-1].ID()})
		if _, err := s.Append(ev); err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
		events = append(events, ev)
	}
	return s, events
}

// --- Identity tests ---

func TestGenerateIdentity(t *testing.T) {
	uri := types.MustSystemURI("eg://test-system")
	id, err := GenerateIdentity(uri)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	if id.SystemURI() != uri {
		t.Errorf("SystemURI = %v, want %v", id.SystemURI(), uri)
	}
	if len(id.PublicKey().Bytes()) != 32 {
		t.Errorf("PublicKey length = %d, want 32", len(id.PublicKey().Bytes()))
	}
}

func TestSignAndVerify(t *testing.T) {
	uri := types.MustSystemURI("eg://test-system")
	id, err := GenerateIdentity(uri)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	data := []byte("hello eventgraph")
	sig, err := id.Sign(data)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	valid, err := id.Verify(id.PublicKey(), data, sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Error("expected valid signature")
	}
}

func TestVerifyWrongData(t *testing.T) {
	uri := types.MustSystemURI("eg://test-system")
	id, err := GenerateIdentity(uri)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	sig, err := id.Sign([]byte("original"))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	valid, err := id.Verify(id.PublicKey(), []byte("tampered"), sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if valid {
		t.Error("expected invalid signature for tampered data")
	}
}

func TestVerifyWrongKey(t *testing.T) {
	id1, err := GenerateIdentity(types.MustSystemURI("eg://system-a"))
	if err != nil {
		t.Fatalf("GenerateIdentity 1: %v", err)
	}
	id2, err := GenerateIdentity(types.MustSystemURI("eg://system-b"))
	if err != nil {
		t.Fatalf("GenerateIdentity 2: %v", err)
	}

	data := []byte("test data")
	sig, err := id1.Sign(data)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	valid, err := id1.Verify(id2.PublicKey(), data, sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if valid {
		t.Error("expected invalid signature for wrong key")
	}
}

func TestNewIdentityFromKey(t *testing.T) {
	uri := types.MustSystemURI("eg://test-system")
	original, err := GenerateIdentity(uri)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	restored, err := NewIdentityFromKey(uri, original.privateKey)
	if err != nil {
		t.Fatalf("NewIdentityFromKey: %v", err)
	}

	if restored.PublicKey().String() != original.PublicKey().String() {
		t.Error("restored public key should match original")
	}

	data := []byte("roundtrip test")
	sig, err := original.Sign(data)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	valid, err := restored.Verify(restored.PublicKey(), data, sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Error("restored identity should verify original's signature")
	}
}

// --- Envelope tests ---

func TestEnvelopeCanonicalForm(t *testing.T) {
	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000001"),
		From:            types.MustSystemURI("eg://system-a"),
		To:              types.MustSystemURI("eg://system-b"),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        types.MustSystemURI("eg://system-a"),
			ProtocolVersions: []int{CurrentProtocolVersion},
			Capabilities:     []string{"events"},
			ChainLength:      10,
		},
		Timestamp: time.Unix(0, 1700000000000000000),
	}

	canonical, err := env.CanonicalForm()
	if err != nil {
		t.Fatalf("CanonicalForm: %v", err)
	}
	if canonical == "" {
		t.Error("canonical form should not be empty")
	}

	// Should be deterministic.
	canonical2, err := env.CanonicalForm()
	if err != nil {
		t.Fatalf("CanonicalForm second call: %v", err)
	}
	if canonical != canonical2 {
		t.Error("canonical form should be deterministic")
	}
}

func TestSignAndVerifyEnvelope(t *testing.T) {
	id, err := GenerateIdentity(types.MustSystemURI("eg://system-a"))
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000001"),
		From:            types.MustSystemURI("eg://system-a"),
		To:              types.MustSystemURI("eg://system-b"),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        id.SystemURI(),
			PublicKey:        id.PublicKey(),
			ProtocolVersions: []int{CurrentProtocolVersion},
			Capabilities:     []string{"events"},
			ChainLength:      5,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, err := SignEnvelope(env, id)
	if err != nil {
		t.Fatalf("SignEnvelope: %v", err)
	}

	valid, err := VerifyEnvelope(signed, id, id.PublicKey())
	if err != nil {
		t.Fatalf("VerifyEnvelope: %v", err)
	}
	if !valid {
		t.Error("expected valid envelope signature")
	}
}

func TestVerifyEnvelopeTampered(t *testing.T) {
	id, err := GenerateIdentity(types.MustSystemURI("eg://system-a"))
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000001"),
		From:            types.MustSystemURI("eg://system-a"),
		To:              types.MustSystemURI("eg://system-b"),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        id.SystemURI(),
			ProtocolVersions: []int{CurrentProtocolVersion},
			Capabilities:     []string{"events"},
			ChainLength:      5,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, err := SignEnvelope(env, id)
	if err != nil {
		t.Fatalf("SignEnvelope: %v", err)
	}

	// Tamper with the envelope.
	signed.ProtocolVersion = 99

	valid, err := VerifyEnvelope(signed, id, id.PublicKey())
	if err != nil {
		t.Fatalf("VerifyEnvelope: %v", err)
	}
	if valid {
		t.Error("expected invalid signature for tampered envelope")
	}
}

// --- Version negotiation tests ---

func TestNegotiateVersion(t *testing.T) {
	tests := []struct {
		name     string
		local    []int
		remote   []int
		wantNone bool
		want     int
	}{
		{"common highest", []int{1, 2, 3}, []int{2, 3, 4}, false, 3},
		{"single match", []int{1}, []int{1}, false, 1},
		{"no overlap", []int{1, 2}, []int{3, 4}, true, 0},
		{"empty local", []int{}, []int{1}, true, 0},
		{"empty remote", []int{1}, []int{}, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NegotiateVersion(tt.local, tt.remote)
			if tt.wantNone {
				if result.IsSome() {
					t.Errorf("expected None, got Some(%d)", result.Unwrap())
				}
			} else {
				if !result.IsSome() {
					t.Error("expected Some, got None")
				} else if result.Unwrap() != tt.want {
					t.Errorf("got %d, want %d", result.Unwrap(), tt.want)
				}
			}
		})
	}
}

// --- Treaty tests ---

func TestTreatyTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    event.TreatyStatus
		to      event.TreatyStatus
		wantErr bool
	}{
		{"proposed to active", event.TreatyStatusProposed, event.TreatyStatusActive, false},
		{"proposed to terminated", event.TreatyStatusProposed, event.TreatyStatusTerminated, false},
		{"active to suspended", event.TreatyStatusActive, event.TreatyStatusSuspended, false},
		{"active to terminated", event.TreatyStatusActive, event.TreatyStatusTerminated, false},
		{"suspended to active", event.TreatyStatusSuspended, event.TreatyStatusActive, false},
		{"suspended to terminated", event.TreatyStatusSuspended, event.TreatyStatusTerminated, false},
		{"terminated is terminal", event.TreatyStatusTerminated, event.TreatyStatusActive, true},
		{"proposed to suspended invalid", event.TreatyStatusProposed, event.TreatyStatusSuspended, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			treaty := &Treaty{
				Status: tt.from,
			}
			err := treaty.Transition(tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transition(%s→%s) error = %v, wantErr %v", tt.from, tt.to, err, tt.wantErr)
			}
			if err == nil && treaty.Status != tt.to {
				t.Errorf("status = %s, want %s", treaty.Status, tt.to)
			}
		})
	}
}

func TestTreatyApplyAction(t *testing.T) {
	tests := []struct {
		name       string
		initial    event.TreatyStatus
		action     event.TreatyAction
		wantStatus event.TreatyStatus
		wantErr    bool
	}{
		{"accept proposed", event.TreatyStatusProposed, event.TreatyActionAccept, event.TreatyStatusActive, false},
		{"suspend active", event.TreatyStatusActive, event.TreatyActionSuspend, event.TreatyStatusSuspended, false},
		{"terminate active", event.TreatyStatusActive, event.TreatyActionTerminate, event.TreatyStatusTerminated, false},
		{"modify active", event.TreatyStatusActive, event.TreatyActionModify, event.TreatyStatusActive, false},
		{"modify proposed fails", event.TreatyStatusProposed, event.TreatyActionModify, event.TreatyStatusProposed, true},
		{"propose on existing fails", event.TreatyStatusActive, event.TreatyActionPropose, event.TreatyStatusActive, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			treaty := &Treaty{
				Status: tt.initial,
			}
			err := treaty.ApplyAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyAction(%s) error = %v, wantErr %v", tt.action, err, tt.wantErr)
			}
			if err == nil && treaty.Status != tt.wantStatus {
				t.Errorf("status = %s, want %s", treaty.Status, tt.wantStatus)
			}
		})
	}
}

func TestNewTreaty(t *testing.T) {
	id := types.MustTreatyID("00000000-0000-0000-0000-000000000002")
	a := types.MustSystemURI("eg://system-a")
	b := types.MustSystemURI("eg://system-b")
	terms := []TreatyTerm{{
		Scope:     types.MustDomainScope("events"),
		Policy:    "share-all",
		Symmetric: true,
	}}

	treaty := NewTreaty(id, a, b, terms)

	if treaty.ID != id {
		t.Errorf("ID = %v, want %v", treaty.ID, id)
	}
	if treaty.Status != event.TreatyStatusProposed {
		t.Errorf("Status = %v, want Proposed", treaty.Status)
	}
	if len(treaty.Terms) != 1 {
		t.Errorf("Terms count = %d, want 1", len(treaty.Terms))
	}
}

// --- PeerStore tests ---

func TestPeerStoreRegisterAndGet(t *testing.T) {
	ps := NewPeerStore()
	uri := types.MustSystemURI("eg://remote-system")
	pubKey, _ := types.NewPublicKey(make([]byte, 32))

	record := ps.Register(uri, pubKey, []string{"events"}, 1)
	if record.SystemURI != uri {
		t.Errorf("SystemURI = %v, want %v", record.SystemURI, uri)
	}
	if record.Trust.Value() != 0.0 {
		t.Errorf("initial trust = %v, want 0.0", record.Trust.Value())
	}

	got, ok := ps.Get(uri)
	if !ok {
		t.Fatal("expected peer to be found")
	}
	if got.SystemURI != uri {
		t.Errorf("Get SystemURI = %v, want %v", got.SystemURI, uri)
	}

	// Verify returned copy is independent of internal state.
	got.NegotiatedVersion = 99
	got2, _ := ps.Get(uri)
	if got2.NegotiatedVersion == 99 {
		t.Error("Get should return a copy, not a reference to internal state")
	}
}

func TestPeerStoreGetMissing(t *testing.T) {
	ps := NewPeerStore()
	_, ok := ps.Get(types.MustSystemURI("eg://nonexistent"))
	if ok {
		t.Error("expected peer not found")
	}
}

func TestPeerStoreUpdateTrust(t *testing.T) {
	ps := NewPeerStore()
	uri := types.MustSystemURI("eg://remote")
	pubKey, _ := types.NewPublicKey(make([]byte, 32))
	ps.Register(uri, pubKey, nil, 1)

	score, ok := ps.UpdateTrust(uri, TrustImpactValidProof)
	if !ok {
		t.Fatal("expected peer found")
	}
	if score.Value() != TrustImpactValidProof {
		t.Errorf("trust = %v, want %v", score.Value(), TrustImpactValidProof)
	}
}

func TestPeerStoreUpdateTrustClampMax(t *testing.T) {
	ps := NewPeerStore()
	uri := types.MustSystemURI("eg://remote")
	pubKey, _ := types.NewPublicKey(make([]byte, 32))
	ps.Register(uri, pubKey, nil, 1)

	// Large positive delta should be clamped to MaxAdjustment.
	score, ok := ps.UpdateTrust(uri, 1.0)
	if !ok {
		t.Fatal("expected peer found")
	}
	maxAdj := InterSystemMaxAdjustment.Value()
	if score.Value() != maxAdj {
		t.Errorf("trust = %v, want %v (clamped)", score.Value(), maxAdj)
	}
}

func TestPeerStoreUpdateTrustClampNegative(t *testing.T) {
	ps := NewPeerStore()
	uri := types.MustSystemURI("eg://remote")
	pubKey, _ := types.NewPublicKey(make([]byte, 32))
	ps.Register(uri, pubKey, nil, 1)

	// Already at 0, negative delta should stay at 0.
	score, ok := ps.UpdateTrust(uri, -0.5)
	if !ok {
		t.Fatal("expected peer found")
	}
	if score.Value() != 0.0 {
		t.Errorf("trust = %v, want 0.0", score.Value())
	}
}

func TestPeerStoreUpdateTrustMissing(t *testing.T) {
	ps := NewPeerStore()
	_, ok := ps.UpdateTrust(types.MustSystemURI("eg://nonexistent"), 0.1)
	if ok {
		t.Error("expected not found")
	}
}

func TestPeerStoreReRegister(t *testing.T) {
	ps := NewPeerStore()
	uri := types.MustSystemURI("eg://remote")
	pubKey1, _ := types.NewPublicKey(make([]byte, 32))
	pubKey2Bytes := make([]byte, 32)
	pubKey2Bytes[0] = 1
	pubKey2, _ := types.NewPublicKey(pubKey2Bytes)

	ps.Register(uri, pubKey1, []string{"events"}, 1)
	ps.UpdateTrust(uri, 0.03)

	// Re-register should NOT update key (prevents key-substitution attack),
	// but should update capabilities and version, and preserve trust.
	record := ps.Register(uri, pubKey2, []string{"events", "proofs"}, 2)
	if record.PublicKey.String() != pubKey1.String() {
		t.Error("re-register should preserve original public key")
	}
	if record.NegotiatedVersion != 2 {
		t.Errorf("version = %d, want 2", record.NegotiatedVersion)
	}
	if record.Trust.Value() < 0.01 {
		t.Error("re-register should preserve trust")
	}
}

func TestPeerStoreAll(t *testing.T) {
	ps := NewPeerStore()
	pubKey, _ := types.NewPublicKey(make([]byte, 32))
	ps.Register(types.MustSystemURI("eg://a"), pubKey, nil, 1)
	ps.Register(types.MustSystemURI("eg://b"), pubKey, nil, 1)

	all := ps.All()
	if len(all) != 2 {
		t.Errorf("All() count = %d, want 2", len(all))
	}

	// Verify returned copies are independent.
	if len(all) > 0 {
		all[0].NegotiatedVersion = 99
		got, _ := ps.Get(all[0].SystemURI)
		if got.NegotiatedVersion == 99 {
			t.Error("All should return copies, not references to internal state")
		}
	}
}

func TestPeerStoreDecayAll(t *testing.T) {
	ps := NewPeerStore()
	uri := types.MustSystemURI("eg://remote")
	pubKey, _ := types.NewPublicKey(make([]byte, 32))
	ps.Register(uri, pubKey, nil, 1)

	// Build up trust to 0.05 (max single adjustment).
	ps.UpdateTrust(uri, 0.05)

	// Manually set LastDecayedAt to 2 days ago to test decay.
	ps.mu.Lock()
	ps.peers[uri.Value()].LastDecayedAt = time.Now().Add(-48 * time.Hour)
	ps.mu.Unlock()

	ps.DecayAll()

	got, ok := ps.Get(uri)
	if !ok {
		t.Fatal("expected peer to be found")
	}

	// Decay should be 0.02 * 2 = 0.04, so trust should be ~0.01.
	expectedApprox := 0.05 - (InterSystemDecayRate.Value() * 2.0)
	if got.Trust.Value() < expectedApprox-0.005 || got.Trust.Value() > expectedApprox+0.005 {
		t.Errorf("trust after decay = %v, want ~%v", got.Trust.Value(), expectedApprox)
	}

	// Calling DecayAll again immediately should not compound (LastDecayedAt was updated).
	trustBefore := got.Trust.Value()
	ps.DecayAll()
	got2, _ := ps.Get(uri)
	if got2.Trust.Value() < trustBefore-0.001 {
		t.Errorf("DecayAll should not compound: before=%v, after=%v", trustBefore, got2.Trust.Value())
	}
}

func TestPeerStoreDecayAllClampsToZero(t *testing.T) {
	ps := NewPeerStore()
	uri := types.MustSystemURI("eg://remote")
	pubKey, _ := types.NewPublicKey(make([]byte, 32))
	ps.Register(uri, pubKey, nil, 1)

	// Trust starts at 0.0, set LastDecayedAt to long ago.
	ps.mu.Lock()
	ps.peers[uri.Value()].LastDecayedAt = time.Now().Add(-365 * 24 * time.Hour)
	ps.mu.Unlock()

	ps.DecayAll()

	got, _ := ps.Get(uri)
	if got.Trust.Value() != 0.0 {
		t.Errorf("trust should clamp to 0.0, got %v", got.Trust.Value())
	}
}

// --- Proof verification tests ---

func TestVerifyChainSegmentValid(t *testing.T) {
	s, events := setupStoreWithEvents(t, 3)
	_ = s

	proof := &ChainSegmentProof{
		Events:    events,
		StartHash: events[0].PrevHash(),
		EndHash:   events[len(events)-1].Hash(),
	}

	if !VerifyChainSegment(proof) {
		t.Error("expected valid chain segment")
	}
}

func TestVerifyChainSegmentEmpty(t *testing.T) {
	proof := &ChainSegmentProof{
		Events: []event.Event{},
	}
	if VerifyChainSegment(proof) {
		t.Error("expected invalid for empty events")
	}
}

func TestVerifyChainSegmentBadStartHash(t *testing.T) {
	_, events := setupStoreWithEvents(t, 2)

	proof := &ChainSegmentProof{
		Events:    events,
		StartHash: events[1].Hash(), // wrong start hash
		EndHash:   events[len(events)-1].Hash(),
	}

	if VerifyChainSegment(proof) {
		t.Error("expected invalid for wrong start hash")
	}
}

func TestVerifyChainSegmentBadEndHash(t *testing.T) {
	_, events := setupStoreWithEvents(t, 2)

	proof := &ChainSegmentProof{
		Events:    events,
		StartHash: events[0].PrevHash(),
		EndHash:   events[0].Hash(), // wrong end hash (should be last event's hash)
	}

	if VerifyChainSegment(proof) {
		t.Error("expected invalid for wrong end hash")
	}
}

func TestVerifyEventExistenceValid(t *testing.T) {
	_, events := setupStoreWithEvents(t, 3)
	evt := events[1]

	proof := &EventExistenceProof{
		Event:       evt,
		PrevHash:    evt.PrevHash(),
		NextHash:    types.Some(events[2].Hash()),
		Position:    1,
		ChainLength: 3,
	}

	if !VerifyEventExistence(proof) {
		t.Error("expected valid event existence proof")
	}
}

func TestVerifyEventExistencePositionOutOfRange(t *testing.T) {
	_, events := setupStoreWithEvents(t, 2)
	evt := events[0]

	proof := &EventExistenceProof{
		Event:       evt,
		PrevHash:    evt.PrevHash(),
		Position:    5, // out of range
		ChainLength: 2,
	}

	if VerifyEventExistence(proof) {
		t.Error("expected invalid for out-of-range position")
	}
}

// --- ValidateProof dispatch tests ---

func TestValidateProofChainSummary(t *testing.T) {
	payload := &ProofPayload{
		ProofType: event.ProofTypeChainSummary,
		Data: ChainSummaryProof{
			Length:    10,
			Timestamp: time.Now(),
		},
	}
	valid, err := ValidateProof(payload)
	if err != nil {
		t.Fatalf("ValidateProof: %v", err)
	}
	if !valid {
		t.Error("expected valid chain summary")
	}
}

func TestValidateProofChainSummaryEmpty(t *testing.T) {
	payload := &ProofPayload{
		ProofType: event.ProofTypeChainSummary,
		Data: ChainSummaryProof{
			Length: 0,
		},
	}
	valid, err := ValidateProof(payload)
	if err != nil {
		t.Fatalf("ValidateProof: %v", err)
	}
	if valid {
		t.Error("expected invalid for zero-length chain summary")
	}
}

func TestValidateProofUnknownType(t *testing.T) {
	payload := &ProofPayload{
		ProofType: event.ProofType("Unknown"),
		Data:      nil,
	}
	_, err := ValidateProof(payload)
	if err == nil {
		t.Error("expected error for unknown proof type")
	}
}

// --- ProofTypeFromData tests ---

func TestProofTypeFromData(t *testing.T) {
	tests := []struct {
		name    string
		data    ProofData
		want    event.ProofType
		wantErr bool
	}{
		{"chain segment", ChainSegmentProof{}, event.ProofTypeChainSegment, false},
		{"chain segment ptr", &ChainSegmentProof{}, event.ProofTypeChainSegment, false},
		{"event existence", EventExistenceProof{}, event.ProofTypeEventExistence, false},
		{"event existence ptr", &EventExistenceProof{}, event.ProofTypeEventExistence, false},
		{"chain summary", ChainSummaryProof{}, event.ProofTypeChainSummary, false},
		{"chain summary ptr", &ChainSummaryProof{}, event.ProofTypeChainSummary, false},
		{"unknown type", nil, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProofTypeFromData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProofTypeFromData error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ProofTypeFromData = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- ProofGenerator tests ---

func TestProofGeneratorChainSummary(t *testing.T) {
	s, events := setupStoreWithEvents(t, 3)
	pg := NewProofGenerator(s)

	proof, err := pg.GenerateChainSummary()
	if err != nil {
		t.Fatalf("GenerateChainSummary: %v", err)
	}

	if proof.Length != 3 {
		t.Errorf("Length = %d, want 3", proof.Length)
	}
	if proof.HeadHash != events[2].Hash() {
		t.Error("HeadHash should match last event's hash")
	}
}

func TestProofGeneratorEventExistence(t *testing.T) {
	s, events := setupStoreWithEvents(t, 3)
	pg := NewProofGenerator(s)

	proof, err := pg.GenerateEventExistence(events[1].ID())
	if err != nil {
		t.Fatalf("GenerateEventExistence: %v", err)
	}

	if proof.Event.ID() != events[1].ID() {
		t.Error("proof event should match requested event")
	}
	if proof.ChainLength != 3 {
		t.Errorf("ChainLength = %d, want 3", proof.ChainLength)
	}
}

// --- Error type tests ---

func TestErrorTypes(t *testing.T) {
	uri := types.MustSystemURI("eg://test")
	envID := types.MustEnvelopeID("00000000-0000-0000-0000-000000000001")
	treatyID := types.MustTreatyID("00000000-0000-0000-0000-000000000002")

	errors := []EGIPError{
		&SystemNotFoundError{URI: uri},
		&EnvelopeSignatureInvalidError{EnvelopeID: envID},
		&TreatyViolationError{TreatyID: treatyID, Term: "share-all"},
		&TrustInsufficientError{System: uri, Score: types.MustScore(0.1), Required: types.MustScore(0.5)},
		&TransportFailureError{To: uri, Reason: "timeout"},
		&DuplicateEnvelopeError{EnvelopeID: envID},
		&VersionIncompatibleError{Local: []int{1}, Remote: []int{2}},
	}

	for _, e := range errors {
		if e.Error() == "" {
			t.Errorf("%T.Error() should not be empty", e)
		}
		// Verify it satisfies EGIPError (compile-time via slice type, but also runtime).
		var _ EGIPError = e
	}
}

// --- Concurrent access test ---

func TestPeerStoreConcurrent(t *testing.T) {
	ps := NewPeerStore()
	pubKey, _ := types.NewPublicKey(make([]byte, 32))
	uri := types.MustSystemURI("eg://concurrent")
	ps.Register(uri, pubKey, []string{"events"}, 1)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			ps.UpdateTrust(uri, 0.01)
		}()
		go func() {
			defer wg.Done()
			ps.Get(uri)
		}()
		go func() {
			defer wg.Done()
			ps.All()
		}()
	}
	wg.Wait()
	// If we get here without a race detector panic, the test passes.
}

// --- Verify error branches ---

func TestIdentityCreatedAt(t *testing.T) {
	before := time.Now()
	id, err := GenerateIdentity(types.MustSystemURI("eg://test"))
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}
	after := time.Now()
	if id.CreatedAt().Before(before) || id.CreatedAt().After(after) {
		t.Error("CreatedAt should be between before and after GenerateIdentity call")
	}
}

// --- Envelope Dedup tests ---

func TestEnvelopeDedupCheck(t *testing.T) {
	d := NewEnvelopeDedup()
	id := types.MustEnvelopeID("00000000-0000-0000-0000-000000000001")

	if !d.Check(id) {
		t.Error("first check should return true")
	}
	if d.Check(id) {
		t.Error("second check should return false (duplicate)")
	}
}

func TestEnvelopeDedupSize(t *testing.T) {
	d := NewEnvelopeDedup()
	if d.Size() != 0 {
		t.Errorf("empty dedup Size = %d, want 0", d.Size())
	}

	d.Check(types.MustEnvelopeID("00000000-0000-0000-0000-000000000001"))
	d.Check(types.MustEnvelopeID("00000000-0000-0000-0000-000000000002"))
	if d.Size() != 2 {
		t.Errorf("Size = %d, want 2", d.Size())
	}
}

func TestEnvelopeDedupPrune(t *testing.T) {
	d := NewEnvelopeDedupWithTTL(1 * time.Millisecond)
	d.Check(types.MustEnvelopeID("00000000-0000-0000-0000-000000000001"))

	time.Sleep(5 * time.Millisecond)
	removed := d.Prune()
	if removed != 1 {
		t.Errorf("Prune removed %d, want 1", removed)
	}
	if d.Size() != 0 {
		t.Errorf("Size after prune = %d, want 0", d.Size())
	}
}

func TestEnvelopeDedupConcurrent(t *testing.T) {
	d := NewEnvelopeDedup()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := types.MustEnvelopeID(fmt.Sprintf("00000000-0000-0000-0000-%012d", n))
			d.Check(id)
		}(i)
	}
	wg.Wait()
	if d.Size() != 20 {
		t.Errorf("Size = %d, want 20", d.Size())
	}
}

// --- Treaty Store tests ---

func TestTreatyStorePutAndGet(t *testing.T) {
	ts := NewTreatyStore()
	id := types.MustTreatyID("00000000-0000-0000-0000-000000000010")
	a := types.MustSystemURI("eg://system-a")
	b := types.MustSystemURI("eg://system-b")

	treaty := NewTreaty(id, a, b, []TreatyTerm{
		{Scope: types.MustDomainScope("events"), Policy: "share", Symmetric: true},
	})
	ts.Put(treaty)

	got, ok := ts.Get(id)
	if !ok {
		t.Fatal("expected treaty to be found")
	}
	if got.ID != id {
		t.Errorf("ID = %v, want %v", got.ID, id)
	}
	if got.Status != event.TreatyStatusProposed {
		t.Errorf("Status = %v, want Proposed", got.Status)
	}

	// Verify returned copy is independent.
	got.Status = event.TreatyStatusTerminated
	got2, _ := ts.Get(id)
	if got2.Status == event.TreatyStatusTerminated {
		t.Error("Get should return a copy, not a reference to internal state")
	}
}

func TestTreatyStoreGetMissing(t *testing.T) {
	ts := NewTreatyStore()
	_, ok := ts.Get(types.MustTreatyID("00000000-0000-0000-0000-000000000099"))
	if ok {
		t.Error("expected treaty not found")
	}
}

func TestTreatyStoreBySystem(t *testing.T) {
	ts := NewTreatyStore()
	a := types.MustSystemURI("eg://system-a")
	b := types.MustSystemURI("eg://system-b")
	c := types.MustSystemURI("eg://system-c")

	ts.Put(NewTreaty(types.MustTreatyID("00000000-0000-0000-0000-000000000011"), a, b, nil))
	ts.Put(NewTreaty(types.MustTreatyID("00000000-0000-0000-0000-000000000012"), b, c, nil))
	ts.Put(NewTreaty(types.MustTreatyID("00000000-0000-0000-0000-000000000013"), a, c, nil))

	byA := ts.BySystem(a)
	if len(byA) != 2 {
		t.Errorf("BySystem(a) count = %d, want 2", len(byA))
	}

	byB := ts.BySystem(b)
	if len(byB) != 2 {
		t.Errorf("BySystem(b) count = %d, want 2", len(byB))
	}

	byC := ts.BySystem(c)
	if len(byC) != 2 {
		t.Errorf("BySystem(c) count = %d, want 2", len(byC))
	}
}

func TestTreatyStoreActive(t *testing.T) {
	ts := NewTreatyStore()
	a := types.MustSystemURI("eg://system-a")
	b := types.MustSystemURI("eg://system-b")

	t1 := NewTreaty(types.MustTreatyID("00000000-0000-0000-0000-000000000020"), a, b, nil)
	t1.Transition(event.TreatyStatusActive)
	ts.Put(t1)

	t2 := NewTreaty(types.MustTreatyID("00000000-0000-0000-0000-000000000021"), a, b, nil)
	ts.Put(t2) // stays Proposed

	active := ts.Active()
	if len(active) != 1 {
		t.Errorf("Active() count = %d, want 1", len(active))
	}
}

// --- Mock transport for handler tests ---

type mockTransport struct {
	sent     []*Envelope
	incoming chan IncomingEnvelope
	onSend   func(to types.SystemURI, env *Envelope) (*ReceiptPayload, error)
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		incoming: make(chan IncomingEnvelope, 10),
	}
}

func (m *mockTransport) Send(_ context.Context, to types.SystemURI, env *Envelope) (*ReceiptPayload, error) {
	m.sent = append(m.sent, env)
	if m.onSend != nil {
		return m.onSend(to, env)
	}
	return &ReceiptPayload{
		EnvelopeID: env.ID,
		Status:     event.ReceiptStatusDelivered,
	}, nil
}

func (m *mockTransport) Listen(_ context.Context) <-chan IncomingEnvelope {
	return m.incoming
}

// --- Handler tests ---

func makeTestHandler(t *testing.T) (*Handler, *SystemIdentity, *mockTransport) {
	t.Helper()
	id, err := GenerateIdentity(types.MustSystemURI("eg://local"))
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}
	transport := newMockTransport()
	peers := NewPeerStore()
	treaties := NewTreatyStore()

	h := NewHandler(id, transport, peers, treaties)
	return h, id, transport
}

func TestHandlerHello(t *testing.T) {
	h, _, transport := makeTestHandler(t)
	ctx := context.Background()

	remote := types.MustSystemURI("eg://remote")
	err := h.Hello(ctx, remote)
	if err != nil {
		t.Fatalf("Hello: %v", err)
	}

	if len(transport.sent) != 1 {
		t.Fatalf("expected 1 sent envelope, got %d", len(transport.sent))
	}
	if transport.sent[0].Type != event.MessageTypeHello {
		t.Errorf("sent type = %s, want Hello", transport.sent[0].Type)
	}
}

func TestHandlerHelloTransportFailure(t *testing.T) {
	h, _, transport := makeTestHandler(t)
	transport.onSend = func(_ types.SystemURI, _ *Envelope) (*ReceiptPayload, error) {
		return nil, fmt.Errorf("connection refused")
	}

	ctx := context.Background()
	err := h.Hello(ctx, types.MustSystemURI("eg://unreachable"))
	if err == nil {
		t.Fatal("expected error for transport failure")
	}
	if _, ok := err.(*TransportFailureError); !ok {
		t.Errorf("expected TransportFailureError, got %T", err)
	}
}

func TestHandlerHandleIncomingHello(t *testing.T) {
	h, localID, _ := makeTestHandler(t)

	remoteID, err := GenerateIdentity(types.MustSystemURI("eg://remote"))
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000100"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        remoteID.SystemURI(),
			PublicKey:        remoteID.PublicKey(),
			ProtocolVersions: []int{CurrentProtocolVersion},
			Capabilities:     []string{"events", "treaty"},
			ChainLength:      42,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, err := SignEnvelope(env, remoteID)
	if err != nil {
		t.Fatalf("SignEnvelope: %v", err)
	}

	ctx := context.Background()
	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	// Peer should be registered.
	peer, ok := h.peers.Get(remoteID.SystemURI())
	if !ok {
		t.Fatal("expected remote peer to be registered")
	}
	if peer.NegotiatedVersion != 1 {
		t.Errorf("NegotiatedVersion = %d, want 1", peer.NegotiatedVersion)
	}
	if len(peer.Capabilities) != 2 {
		t.Errorf("Capabilities count = %d, want 2", len(peer.Capabilities))
	}
}

func TestHandlerHandleIncomingReplayRejected(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000200"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        remoteID.SystemURI(),
			PublicKey:        remoteID.PublicKey(),
			ProtocolVersions: []int{CurrentProtocolVersion},
			Capabilities:     []string{"events"},
			ChainLength:      1,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	// First should succeed.
	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("first HandleIncoming: %v", err)
	}

	// Replay should be rejected.
	err := h.HandleIncoming(ctx, signed)
	if err == nil {
		t.Fatal("expected DuplicateEnvelopeError for replay")
	}
	if _, ok := err.(*DuplicateEnvelopeError); !ok {
		t.Errorf("expected DuplicateEnvelopeError, got %T: %v", err, err)
	}
}

func TestHandlerHandleIncomingInvalidSignature(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	otherID, _ := GenerateIdentity(types.MustSystemURI("eg://other"))

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000300"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        remoteID.SystemURI(),
			PublicKey:        remoteID.PublicKey(),
			ProtocolVersions: []int{CurrentProtocolVersion},
			Capabilities:     nil,
			ChainLength:      0,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	// Sign with wrong key.
	signed, _ := SignEnvelope(env, otherID)

	ctx := context.Background()
	err := h.HandleIncoming(ctx, signed)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
	if _, ok := err.(*EnvelopeSignatureInvalidError); !ok {
		t.Errorf("expected EnvelopeSignatureInvalidError, got %T: %v", err, err)
	}
}

func TestHandlerHandleIncomingVersionIncompatible(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	h.LocalProtocolVersions = []int{2, 3} // only support v2 and v3

	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000400"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        remoteID.SystemURI(),
			PublicKey:        remoteID.PublicKey(),
			ProtocolVersions: []int{CurrentProtocolVersion}, // only v1
			Capabilities:     nil,
			ChainLength:      0,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	err := h.HandleIncoming(ctx, signed)
	if err == nil {
		t.Fatal("expected version incompatible error")
	}
	if _, ok := err.(*VersionIncompatibleError); !ok {
		t.Errorf("expected VersionIncompatibleError, got %T: %v", err, err)
	}
}

func TestHandlerHandleIncomingMessage(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))

	// Register the peer first (HELLO would do this normally).
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), []string{"events"}, 1)

	var receivedFrom types.SystemURI
	var receivedPayload *MessagePayloadContent
	h.OnMessage = func(from types.SystemURI, payload *MessagePayloadContent) error {
		receivedFrom = from
		receivedPayload = payload
		return nil
	}

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000500"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeMessage,
		Payload: MessagePayloadContent{
			ContentType:    types.MustEventType("trust.updated"),
			ConversationID: types.None[types.ConversationID](),
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	if receivedFrom != remoteID.SystemURI() {
		t.Errorf("OnMessage from = %v, want %v", receivedFrom, remoteID.SystemURI())
	}
	if receivedPayload == nil {
		t.Error("OnMessage payload should not be nil")
	}
}

func TestHandlerHandleIncomingReceipt(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000600"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeReceipt,
		Payload: ReceiptPayload{
			EnvelopeID: types.MustEnvelopeID("00000000-0000-0000-0000-000000000001"),
			Status:     event.ReceiptStatusProcessed,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	// Trust should increase for receipt.
	peer, _ := h.peers.Get(remoteID.SystemURI())
	if peer.Trust.Value() <= 0 {
		t.Error("trust should increase on receipt")
	}
}

func TestHandlerHandleIncomingProof(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000700"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeProof,
		Payload: ProofPayload{
			ProofType: event.ProofTypeChainSummary,
			Data: ChainSummaryProof{
				Length:    50,
				Timestamp: time.Now(),
			},
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	// Trust should increase for valid proof.
	peer, _ := h.peers.Get(remoteID.SystemURI())
	if peer.Trust.Value() <= 0 {
		t.Error("trust should increase for valid proof")
	}
}

func TestHandlerHandleIncomingTreatyPropose(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	treatyID := types.MustTreatyID("00000000-0000-0000-0000-000000000030")
	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000800"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeTreaty,
		Payload: TreatyPayload{
			TreatyID: treatyID,
			Action:   event.TreatyActionPropose,
			Terms: []TreatyTerm{
				{Scope: types.MustDomainScope("events"), Policy: "share-all", Symmetric: true},
			},
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	// Treaty should be stored.
	treaty, ok := h.treaties.Get(treatyID)
	if !ok {
		t.Fatal("expected treaty to be stored")
	}
	if treaty.Status != event.TreatyStatusProposed {
		t.Errorf("treaty status = %v, want Proposed", treaty.Status)
	}
}

func TestHandlerHandleIncomingTreatyAccept(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	// Pre-create a proposed treaty.
	treatyID := types.MustTreatyID("00000000-0000-0000-0000-000000000031")
	treaty := NewTreaty(treatyID, localID.SystemURI(), remoteID.SystemURI(), nil)
	h.treaties.Put(treaty)

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000801"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeTreaty,
		Payload: TreatyPayload{
			TreatyID: treatyID,
			Action:   event.TreatyActionAccept,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	got, _ := h.treaties.Get(treatyID)
	if got.Status != event.TreatyStatusActive {
		t.Errorf("treaty status = %v, want Active", got.Status)
	}

	// Trust should increase on accept.
	peer, _ := h.peers.Get(remoteID.SystemURI())
	if peer.Trust.Value() <= 0 {
		t.Error("trust should increase on treaty accept")
	}
}

func TestHandlerHandleIncomingUnknownSender(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://unknown"))

	// Don't register the peer — MESSAGE from unknown sender should fail.
	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000900"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeMessage,
		Payload: MessagePayloadContent{
			ContentType: types.MustEventType("trust.updated"),
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	err := h.HandleIncoming(ctx, signed)
	if err == nil {
		t.Fatal("expected error for unknown sender")
	}
	if _, ok := err.(*SystemNotFoundError); !ok {
		t.Errorf("expected SystemNotFoundError, got %T: %v", err, err)
	}
}

func TestGenerateUUID4Format(t *testing.T) {
	uuid, err := generateUUID4()
	if err != nil {
		t.Fatalf("generateUUID4: %v", err)
	}
	if len(uuid) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid))
	}
	// Check dashes at correct positions.
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		t.Errorf("UUID format invalid: %s", uuid)
	}
}

func TestHandlerHelloWithChainLength(t *testing.T) {
	h, _, _ := makeTestHandler(t)
	h.ChainLength = func() (int, error) { return 42, nil }

	ctx := context.Background()
	err := h.Hello(ctx, types.MustSystemURI("eg://remote"))
	if err != nil {
		t.Fatalf("Hello: %v", err)
	}
}

func TestHandlerHelloRejected(t *testing.T) {
	h, _, transport := makeTestHandler(t)
	transport.onSend = func(_ types.SystemURI, _ *Envelope) (*ReceiptPayload, error) {
		return &ReceiptPayload{
			Status: event.ReceiptStatusRejected,
			Reason: types.Some("go away"),
		}, nil
	}

	ctx := context.Background()
	err := h.Hello(ctx, types.MustSystemURI("eg://hostile"))
	if err == nil {
		t.Fatal("expected error for rejected hello")
	}
}

func TestHandlerHandleIncomingTreatySuspend(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	treatyID := types.MustTreatyID("00000000-0000-0000-0000-000000000032")
	treaty := NewTreaty(treatyID, localID.SystemURI(), remoteID.SystemURI(), nil)
	treaty.Transition(event.TreatyStatusActive)
	h.treaties.Put(treaty)

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000810"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeTreaty,
		Payload: TreatyPayload{
			TreatyID: treatyID,
			Action:   event.TreatyActionSuspend,
			Reason:   types.Some("maintenance"),
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	got, _ := h.treaties.Get(treatyID)
	if got.Status != event.TreatyStatusSuspended {
		t.Errorf("treaty status = %v, want Suspended", got.Status)
	}
}

func TestHandlerHandleIncomingTreatyTerminate(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	treatyID := types.MustTreatyID("00000000-0000-0000-0000-000000000033")
	treaty := NewTreaty(treatyID, localID.SystemURI(), remoteID.SystemURI(), nil)
	treaty.Transition(event.TreatyStatusActive)
	h.treaties.Put(treaty)

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000811"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeTreaty,
		Payload: TreatyPayload{
			TreatyID: treatyID,
			Action:   event.TreatyActionTerminate,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	got, _ := h.treaties.Get(treatyID)
	if got.Status != event.TreatyStatusTerminated {
		t.Errorf("treaty status = %v, want Terminated", got.Status)
	}
}

func TestHandlerHandleIncomingTreatyModify(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	treatyID := types.MustTreatyID("00000000-0000-0000-0000-000000000034")
	treaty := NewTreaty(treatyID, localID.SystemURI(), remoteID.SystemURI(), []TreatyTerm{
		{Scope: types.MustDomainScope("events"), Policy: "share-all", Symmetric: true},
	})
	treaty.Transition(event.TreatyStatusActive)
	h.treaties.Put(treaty)

	newTerms := []TreatyTerm{
		{Scope: types.MustDomainScope("events"), Policy: "share-limited", Symmetric: false},
		{Scope: types.MustDomainScope("trust"), Policy: "read-only", Symmetric: true},
	}

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000812"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeTreaty,
		Payload: TreatyPayload{
			TreatyID: treatyID,
			Action:   event.TreatyActionModify,
			Terms:    newTerms,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	got, _ := h.treaties.Get(treatyID)
	if len(got.Terms) != 2 {
		t.Errorf("Terms count = %d, want 2", len(got.Terms))
	}
}

func TestHandlerHandleIncomingAuthorityRequest(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	var receivedPayload *AuthorityRequestPayload
	h.OnAuthorityRequest = func(from types.SystemURI, payload *AuthorityRequestPayload) error {
		receivedPayload = payload
		return nil
	}

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000820"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeAuthorityRequest,
		Payload: AuthorityRequestPayload{
			Action:        types.MustDomainScope("deploy"),
			Actor:         types.MustActorID("actor_00000000000000000000000000000001"),
			Level:         event.AuthorityLevelRequired,
			Justification: "production deploy",
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	if receivedPayload == nil {
		t.Fatal("OnAuthorityRequest should have been called")
	}
	if receivedPayload.Action.Value() != "deploy" {
		t.Errorf("action = %s, want deploy", receivedPayload.Action.Value())
	}
}

func TestHandlerHandleIncomingDiscover(t *testing.T) {
	h, localID, transport := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	var receivedQuery DiscoverQuery
	h.OnDiscover = func(from types.SystemURI, query DiscoverQuery) ([]DiscoverResult, error) {
		receivedQuery = query
		return []DiscoverResult{
			{SystemURI: types.MustSystemURI("eg://found"), TrustScore: types.MustScore(0.5)},
		}, nil
	}

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000830"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeDiscover,
		Payload: DiscoverPayload{
			Query: DiscoverQuery{
				Capabilities: []string{"proof", "treaty"},
				MinTrust:     types.Some(types.MustScore(0.3)),
			},
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	if len(receivedQuery.Capabilities) != 2 {
		t.Errorf("query capabilities count = %d, want 2", len(receivedQuery.Capabilities))
	}

	// Verify a response envelope was sent back.
	if len(transport.sent) != 1 {
		t.Fatalf("expected 1 sent response, got %d", len(transport.sent))
	}
	resp := transport.sent[0]
	if resp.Type != event.MessageTypeDiscover {
		t.Errorf("response type = %s, want Discover", resp.Type)
	}
	if !resp.InReplyTo.IsSome() || resp.InReplyTo.Unwrap() != env.ID {
		t.Error("response InReplyTo should reference the query envelope")
	}
}

func TestValidateProofChainSegment(t *testing.T) {
	_, events := setupStoreWithEvents(t, 3)

	payload := &ProofPayload{
		ProofType: event.ProofTypeChainSegment,
		Data: &ChainSegmentProof{
			Events:    events,
			StartHash: events[0].PrevHash(),
			EndHash:   events[len(events)-1].Hash(),
		},
	}
	valid, err := ValidateProof(payload)
	if err != nil {
		t.Fatalf("ValidateProof: %v", err)
	}
	if !valid {
		t.Error("expected valid chain segment proof")
	}
}

func TestValidateProofEventExistence(t *testing.T) {
	_, events := setupStoreWithEvents(t, 3)
	evt := events[1]

	payload := &ProofPayload{
		ProofType: event.ProofTypeEventExistence,
		Data: &EventExistenceProof{
			Event:       evt,
			PrevHash:    evt.PrevHash(),
			NextHash:    types.Some(events[2].Hash()),
			Position:    1,
			ChainLength: 3,
		},
	}
	valid, err := ValidateProof(payload)
	if err != nil {
		t.Fatalf("ValidateProof: %v", err)
	}
	if !valid {
		t.Error("expected valid event existence proof")
	}
}

func TestValidateProofInvalidProof(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	// Send an invalid proof (zero-length chain summary).
	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000840"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeProof,
		Payload: ProofPayload{
			ProofType: event.ProofTypeChainSummary,
			Data:      ChainSummaryProof{Length: 0},
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	if err := h.HandleIncoming(ctx, signed); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	// Trust should decrease for invalid proof.
	peer, _ := h.peers.Get(remoteID.SystemURI())
	if peer.Trust.Value() != 0.0 {
		t.Errorf("trust should stay at 0 (can't go negative), got %v", peer.Trust.Value())
	}
}

func TestHandlerHandleIncomingStaleTimestamp(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000860"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        remoteID.SystemURI(),
			PublicKey:        remoteID.PublicKey(),
			ProtocolVersions: []int{CurrentProtocolVersion},
		},
		Timestamp: time.Now().Add(-48 * time.Hour), // 2 days old
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	err := h.HandleIncoming(ctx, signed)
	if err == nil {
		t.Fatal("expected error for stale timestamp")
	}
}

func TestHandlerHandleIncomingFutureTimestamp(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000861"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        remoteID.SystemURI(),
			PublicKey:        remoteID.PublicKey(),
			ProtocolVersions: []int{CurrentProtocolVersion},
		},
		Timestamp: time.Now().Add(10 * time.Minute), // 10 min in future
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	err := h.HandleIncoming(ctx, signed)
	if err == nil {
		t.Fatal("expected error for future timestamp")
	}
}

func TestTreatyStoreApply(t *testing.T) {
	ts := NewTreatyStore()
	id := types.MustTreatyID("00000000-0000-0000-0000-000000000040")
	a := types.MustSystemURI("eg://system-a")
	b := types.MustSystemURI("eg://system-b")

	treaty := NewTreaty(id, a, b, []TreatyTerm{
		{Scope: types.MustDomainScope("events"), Policy: "share", Symmetric: true},
	})
	ts.Put(treaty)

	err := ts.Apply(id, func(t *Treaty) error {
		return t.Transition(event.TreatyStatusActive)
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, _ := ts.Get(id)
	if got.Status != event.TreatyStatusActive {
		t.Errorf("status = %v, want Active", got.Status)
	}
}

func TestTreatyStoreApplyNotFound(t *testing.T) {
	ts := NewTreatyStore()
	err := ts.Apply(types.MustTreatyID("00000000-0000-0000-0000-000000000099"), func(t *Treaty) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for treaty not found")
	}
}

func TestHandlerHelloChainLengthError(t *testing.T) {
	h, _, _ := makeTestHandler(t)
	h.ChainLength = func() (int, error) { return 0, fmt.Errorf("store unavailable") }

	ctx := context.Background()
	err := h.Hello(ctx, types.MustSystemURI("eg://remote"))
	if err == nil {
		t.Fatal("expected error for chain length failure")
	}
}

func TestPeerStoreDecayUsesLastDecayedAt(t *testing.T) {
	ps := NewPeerStore()
	uri := types.MustSystemURI("eg://remote")
	pubKey, _ := types.NewPublicKey(make([]byte, 32))
	ps.Register(uri, pubKey, nil, 1)
	ps.UpdateTrust(uri, 0.05)

	// Set LastDecayedAt to 2 days ago, but LastSeen to now.
	ps.mu.Lock()
	ps.peers[uri.Value()].LastDecayedAt = time.Now().Add(-48 * time.Hour)
	ps.mu.Unlock()

	ps.DecayAll()

	got, _ := ps.Get(uri)
	// Decay should be 0.02 * 2 = 0.04, so trust should be ~0.01.
	expected := 0.05 - (InterSystemDecayRate.Value() * 2.0)
	if got.Trust.Value() < expected-0.005 || got.Trust.Value() > expected+0.005 {
		t.Errorf("trust after decay = %v, want ~%v", got.Trust.Value(), expected)
	}

	// LastSeen should NOT have been modified by DecayAll.
	ps.mu.RLock()
	lastSeen := ps.peers[uri.Value()].LastSeen
	ps.mu.RUnlock()
	if time.Since(lastSeen) > 1*time.Second {
		t.Error("DecayAll should not modify LastSeen")
	}
}

func TestHandlerHandleIncomingTreatyNotFound(t *testing.T) {
	h, localID, _ := makeTestHandler(t)
	remoteID, _ := GenerateIdentity(types.MustSystemURI("eg://remote"))
	h.peers.Register(remoteID.SystemURI(), remoteID.PublicKey(), nil, 1)

	env := &Envelope{
		ProtocolVersion: CurrentProtocolVersion,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000850"),
		From:            remoteID.SystemURI(),
		To:              localID.SystemURI(),
		Type:            event.MessageTypeTreaty,
		Payload: TreatyPayload{
			TreatyID: types.MustTreatyID("00000000-0000-0000-0000-000000000099"),
			Action:   event.TreatyActionAccept,
		},
		Timestamp: time.Now(),
		InReplyTo: types.None[types.EnvelopeID](),
	}

	signed, _ := SignEnvelope(env, remoteID)
	ctx := context.Background()

	err := h.HandleIncoming(ctx, signed)
	if err == nil {
		t.Fatal("expected error for treaty not found")
	}
}
