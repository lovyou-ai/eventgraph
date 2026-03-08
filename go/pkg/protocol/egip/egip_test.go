package egip

import (
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
		ProtocolVersion: 1,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000001"),
		From:            types.MustSystemURI("eg://system-a"),
		To:              types.MustSystemURI("eg://system-b"),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        types.MustSystemURI("eg://system-a"),
			ProtocolVersions: []int{1},
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
		ProtocolVersion: 1,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000001"),
		From:            types.MustSystemURI("eg://system-a"),
		To:              types.MustSystemURI("eg://system-b"),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        id.SystemURI(),
			PublicKey:        id.PublicKey(),
			ProtocolVersions: []int{1},
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
		ProtocolVersion: 1,
		ID:              types.MustEnvelopeID("00000000-0000-0000-0000-000000000001"),
		From:            types.MustSystemURI("eg://system-a"),
		To:              types.MustSystemURI("eg://system-b"),
		Type:            event.MessageTypeHello,
		Payload: HelloPayload{
			SystemURI:        id.SystemURI(),
			ProtocolVersions: []int{1},
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

	// Re-register should update key but preserve trust.
	record := ps.Register(uri, pubKey2, []string{"events", "proofs"}, 2)
	if record.PublicKey.String() != pubKey2.String() {
		t.Error("re-register should update public key")
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

	// Manually set LastSeen to 2 days ago to test decay.
	ps.mu.Lock()
	ps.peers[uri.Value()].LastSeen = time.Now().Add(-48 * time.Hour)
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

	// Calling DecayAll again immediately should not compound (LastSeen was updated).
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

	// Trust starts at 0.0, set LastSeen to long ago.
	ps.mu.Lock()
	ps.peers[uri.Value()].LastSeen = time.Now().Add(-365 * 24 * time.Hour)
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
