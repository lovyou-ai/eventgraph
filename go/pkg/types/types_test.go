package types

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

// --- Option ---

func TestOption_Some(t *testing.T) {
	t.Parallel()
	o := Some(42)
	if !o.IsSome() {
		t.Fatal("expected IsSome")
	}
	if o.IsNone() {
		t.Fatal("expected not IsNone")
	}
	if o.Unwrap() != 42 {
		t.Fatalf("expected 42, got %d", o.Unwrap())
	}
	if o.UnwrapOr(0) != 42 {
		t.Fatalf("expected 42, got %d", o.UnwrapOr(0))
	}
}

func TestOption_None(t *testing.T) {
	t.Parallel()
	o := None[int]()
	if o.IsSome() {
		t.Fatal("expected not IsSome")
	}
	if !o.IsNone() {
		t.Fatal("expected IsNone")
	}
	if o.UnwrapOr(99) != 99 {
		t.Fatalf("expected 99, got %d", o.UnwrapOr(99))
	}
}

func TestOption_UnwrapPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on Unwrap of None")
		}
	}()
	None[int]().Unwrap()
}

func TestOption_JSON(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		opt  Option[int]
		json string
	}{
		{"some", Some(42), "42"},
		{"none", None[int](), "null"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.opt)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tc.json {
				t.Fatalf("expected %s, got %s", tc.json, string(b))
			}
			var out Option[int]
			if err := json.Unmarshal(b, &out); err != nil {
				t.Fatal(err)
			}
			if tc.opt.IsSome() != out.IsSome() {
				t.Fatal("round-trip mismatch")
			}
			if tc.opt.IsSome() && tc.opt.Unwrap() != out.Unwrap() {
				t.Fatal("value mismatch")
			}
		})
	}
}

// --- NonEmpty ---

func TestNonEmpty_Valid(t *testing.T) {
	t.Parallel()
	ne, err := NewNonEmpty([]int{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if ne.First() != 1 {
		t.Fatalf("expected first=1, got %d", ne.First())
	}
	if ne.Len() != 3 {
		t.Fatalf("expected len=3, got %d", ne.Len())
	}
	all := ne.All()
	if len(all) != 3 || all[0] != 1 || all[1] != 2 || all[2] != 3 {
		t.Fatalf("unexpected All: %v", all)
	}
}

func TestNonEmpty_Single(t *testing.T) {
	t.Parallel()
	ne := MustNonEmpty([]string{"hello"})
	if ne.First() != "hello" {
		t.Fatal("unexpected first")
	}
	if ne.Len() != 1 {
		t.Fatal("expected len=1")
	}
}

func TestNonEmpty_RejectsEmpty(t *testing.T) {
	t.Parallel()
	_, err := NewNonEmpty([]int{})
	if err == nil {
		t.Fatal("expected error for empty slice")
	}
	var emptyErr *EmptyRequiredError
	if !errors.As(err, &emptyErr) {
		t.Fatalf("expected EmptyRequiredError, got %T", err)
	}
}

func TestNonEmpty_MustPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustNonEmpty([]int{})
}

func TestNonEmpty_JSON(t *testing.T) {
	t.Parallel()
	ne := MustNonEmpty([]int{1, 2, 3})
	b, err := json.Marshal(ne)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "[1,2,3]" {
		t.Fatalf("expected [1,2,3], got %s", string(b))
	}
	var out NonEmpty[int]
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 3 || out.First() != 1 {
		t.Fatal("round-trip mismatch")
	}
}

func TestNonEmpty_JSON_RejectsEmpty(t *testing.T) {
	t.Parallel()
	var out NonEmpty[int]
	err := json.Unmarshal([]byte("[]"), &out)
	if err == nil {
		t.Fatal("expected error for empty array")
	}
}

func TestNonEmpty_DoesNotMutateSource(t *testing.T) {
	t.Parallel()
	src := []int{1, 2, 3}
	ne := MustNonEmpty(src)
	src[1] = 99
	all := ne.All()
	if all[1] != 2 {
		t.Fatal("NonEmpty should deep-copy source")
	}
}

// --- Page ---

func TestPage_Empty(t *testing.T) {
	t.Parallel()
	p := NewPage[int](nil, None[Cursor](), false)
	if len(p.Items()) != 0 {
		t.Fatal("expected empty items")
	}
	if p.HasMore() {
		t.Fatal("expected no more")
	}
	if p.Cursor().IsSome() {
		t.Fatal("expected no cursor")
	}
}

func TestPage_WithCursor(t *testing.T) {
	t.Parallel()
	c := NewCursor("abc123")
	p := NewPage([]int{1, 2}, Some(c), true)
	if len(p.Items()) != 2 {
		t.Fatal("expected 2 items")
	}
	if !p.HasMore() {
		t.Fatal("expected has more")
	}
	if p.Cursor().Unwrap().String() != "abc123" {
		t.Fatal("cursor mismatch")
	}
}

// --- Constrained Numerics ---

func TestScore_Valid(t *testing.T) {
	t.Parallel()
	cases := []float64{0.0, 0.5, 1.0}
	for _, v := range cases {
		s, err := NewScore(v)
		if err != nil {
			t.Fatalf("NewScore(%v) failed: %v", v, err)
		}
		if s.Value() != v {
			t.Fatalf("expected %v, got %v", v, s.Value())
		}
	}
}

func TestScore_Invalid(t *testing.T) {
	t.Parallel()
	cases := []float64{-0.1, 1.1, math.NaN(), math.Inf(1), math.Inf(-1)}
	for _, v := range cases {
		_, err := NewScore(v)
		if err == nil {
			t.Fatalf("expected error for Score(%v)", v)
		}
		var oor *OutOfRangeError
		if !errors.As(err, &oor) {
			t.Fatalf("expected OutOfRangeError, got %T", err)
		}
	}
}

func TestScore_JSON(t *testing.T) {
	t.Parallel()
	s := MustScore(0.75)
	b, _ := json.Marshal(s)
	if string(b) != "0.75" {
		t.Fatalf("expected 0.75, got %s", string(b))
	}
	var out Score
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Value() != 0.75 {
		t.Fatal("round-trip mismatch")
	}
}

func TestScore_JSON_RejectsInvalid(t *testing.T) {
	t.Parallel()
	var s Score
	err := json.Unmarshal([]byte("5.0"), &s)
	if err == nil {
		t.Fatal("expected error for invalid score")
	}
}

func TestWeight_Valid(t *testing.T) {
	t.Parallel()
	cases := []float64{-1.0, -0.5, 0.0, 0.5, 1.0}
	for _, v := range cases {
		w, err := NewWeight(v)
		if err != nil {
			t.Fatalf("NewWeight(%v) failed: %v", v, err)
		}
		if w.Value() != v {
			t.Fatalf("expected %v, got %v", v, w.Value())
		}
	}
}

func TestWeight_Invalid(t *testing.T) {
	t.Parallel()
	cases := []float64{-1.1, 1.1, math.NaN()}
	for _, v := range cases {
		_, err := NewWeight(v)
		if err == nil {
			t.Fatalf("expected error for Weight(%v)", v)
		}
	}
}

func TestActivation_Boundaries(t *testing.T) {
	t.Parallel()
	if _, err := NewActivation(0.0); err != nil {
		t.Fatal(err)
	}
	if _, err := NewActivation(1.0); err != nil {
		t.Fatal(err)
	}
	if _, err := NewActivation(-0.01); err == nil {
		t.Fatal("expected error")
	}
	if _, err := NewActivation(1.01); err == nil {
		t.Fatal("expected error")
	}
}

func TestLayer_Valid(t *testing.T) {
	t.Parallel()
	for i := 0; i <= 13; i++ {
		l, err := NewLayer(i)
		if err != nil {
			t.Fatalf("NewLayer(%d) failed: %v", i, err)
		}
		if l.Value() != i {
			t.Fatalf("expected %d, got %d", i, l.Value())
		}
	}
}

func TestLayer_Invalid(t *testing.T) {
	t.Parallel()
	cases := []int{-1, 14, 100}
	for _, v := range cases {
		_, err := NewLayer(v)
		if err == nil {
			t.Fatalf("expected error for Layer(%d)", v)
		}
		var ior *IntOutOfRangeError
		if !errors.As(err, &ior) {
			t.Fatalf("expected IntOutOfRangeError, got %T", err)
		}
	}
}

func TestCadence_Valid(t *testing.T) {
	t.Parallel()
	c, err := NewCadence(1)
	if err != nil {
		t.Fatal(err)
	}
	if c.Value() != 1 {
		t.Fatal("expected 1")
	}
	c2, err := NewCadence(100)
	if err != nil {
		t.Fatal(err)
	}
	if c2.Value() != 100 {
		t.Fatal("expected 100")
	}
}

func TestCadence_Invalid(t *testing.T) {
	t.Parallel()
	cases := []int{0, -1, -100}
	for _, v := range cases {
		_, err := NewCadence(v)
		if err == nil {
			t.Fatalf("expected error for Cadence(%d)", v)
		}
	}
}

func TestTick_Valid(t *testing.T) {
	t.Parallel()
	tk, err := NewTick(0)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Value() != 0 {
		t.Fatal("expected 0")
	}
}

func TestTick_Invalid(t *testing.T) {
	t.Parallel()
	_, err := NewTick(-1)
	if err == nil {
		t.Fatal("expected error for negative tick")
	}
}

func TestDuration_Valid(t *testing.T) {
	t.Parallel()
	d, err := NewDuration(0)
	if err != nil {
		t.Fatal(err)
	}
	if d.Value() != 0 {
		t.Fatal("expected 0")
	}
	d2, err := NewDuration(1_000_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if d2.Value() != 1_000_000_000 {
		t.Fatal("expected 1e9")
	}
}

func TestDuration_Invalid(t *testing.T) {
	t.Parallel()
	_, err := NewDuration(-1)
	if err == nil {
		t.Fatal("expected error for negative duration")
	}
}

func TestFieldPath_Valid(t *testing.T) {
	t.Parallel()
	cases := []string{"name", "context.actor.status", "a_b", "_private"}
	for _, v := range cases {
		fp, err := NewFieldPath(v)
		if err != nil {
			t.Fatalf("NewFieldPath(%q) failed: %v", v, err)
		}
		if fp.Value() != v {
			t.Fatalf("expected %q, got %q", v, fp.Value())
		}
	}
}

func TestFieldPath_Invalid(t *testing.T) {
	t.Parallel()
	cases := []string{"", "123abc", ".leading", "trailing.", "a..b", "a.1b"}
	for _, v := range cases {
		_, err := NewFieldPath(v)
		if err == nil {
			t.Fatalf("expected error for FieldPath(%q)", v)
		}
	}
}

// --- Typed IDs ---

func TestEventID_Valid(t *testing.T) {
	t.Parallel()
	// Valid UUID v7
	id, err := NewEventID("01912345-6789-7abc-8def-0123456789ab")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "01912345-6789-7abc-8def-0123456789ab" {
		t.Fatal("unexpected value")
	}
}

func TestEventID_Invalid(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",
		"not-a-uuid",
		"01912345-6789-4abc-8def-0123456789ab", // v4 not v7
		"01912345-6789-7abc-0def-0123456789ab", // wrong variant
	}
	for _, v := range cases {
		_, err := NewEventID(v)
		if err == nil {
			t.Fatalf("expected error for EventID(%q)", v)
		}
	}
}

func TestEventID_JSON(t *testing.T) {
	t.Parallel()
	id := MustEventID("01912345-6789-7abc-8def-0123456789ab")
	b, _ := json.Marshal(id)
	var out EventID
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if id.String() != out.String() {
		t.Fatal("round-trip mismatch")
	}
}

func TestActorID_Valid(t *testing.T) {
	t.Parallel()
	id, err := NewActorID("agent_alpha")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "agent_alpha" {
		t.Fatal("unexpected value")
	}
}

func TestActorID_RejectsEmpty(t *testing.T) {
	t.Parallel()
	_, err := NewActorID("")
	if err == nil {
		t.Fatal("expected error for empty ActorID")
	}
}

func TestHash_Valid(t *testing.T) {
	t.Parallel()
	h, err := NewHash("a" + "b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c")
	if err != nil {
		t.Fatal(err)
	}
	if h.IsZero() {
		t.Fatal("expected non-zero hash")
	}
}

func TestHash_Invalid(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",
		"tooshort",
		"gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg", // non-hex
	}
	for _, v := range cases {
		_, err := NewHash(v)
		if err == nil {
			t.Fatalf("expected error for Hash(%q)", v)
		}
	}
}

func TestZeroHash(t *testing.T) {
	t.Parallel()
	h := ZeroHash()
	if !h.IsZero() {
		t.Fatal("expected zero hash")
	}
	if len(h.Value()) != 64 {
		t.Fatal("expected 64 chars")
	}
}

func TestPublicKey_Valid(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	key[0] = 0x01
	pk, err := NewPublicKey(key)
	if err != nil {
		t.Fatal(err)
	}
	// Verify deep copy
	key[0] = 0xFF
	if pk.Bytes()[0] != 0x01 {
		t.Fatal("PublicKey should deep-copy")
	}
}

func TestPublicKey_Invalid(t *testing.T) {
	t.Parallel()
	cases := []int{0, 16, 31, 33, 64}
	for _, size := range cases {
		_, err := NewPublicKey(make([]byte, size))
		if err == nil {
			t.Fatalf("expected error for %d-byte key", size)
		}
	}
}

func TestSignature_Valid(t *testing.T) {
	t.Parallel()
	sig, err := NewSignature(make([]byte, 64))
	if err != nil {
		t.Fatal(err)
	}
	if len(sig.Bytes()) != 64 {
		t.Fatal("expected 64 bytes")
	}
}

func TestSignature_Invalid(t *testing.T) {
	t.Parallel()
	_, err := NewSignature(make([]byte, 63))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEventType_Valid(t *testing.T) {
	t.Parallel()
	cases := []string{"trust.updated", "actor.registered", "edge.created", "bootstrap"}
	for _, v := range cases {
		et, err := NewEventType(v)
		if err != nil {
			t.Fatalf("NewEventType(%q) failed: %v", v, err)
		}
		if et.String() != v {
			t.Fatalf("expected %q, got %q", v, et.String())
		}
	}
}

func TestEventType_Invalid(t *testing.T) {
	t.Parallel()
	cases := []string{"", "Trust.Updated", "123", ".leading", "trailing.", "a..b"}
	for _, v := range cases {
		_, err := NewEventType(v)
		if err == nil {
			t.Fatalf("expected error for EventType(%q)", v)
		}
	}
}

func TestSubscriptionPattern_Valid(t *testing.T) {
	t.Parallel()
	cases := []string{"*", "trust.*", "trust.updated", "actor.registered"}
	for _, v := range cases {
		_, err := NewSubscriptionPattern(v)
		if err != nil {
			t.Fatalf("NewSubscriptionPattern(%q) failed: %v", v, err)
		}
	}
}

func TestSubscriptionPattern_Matches(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pattern string
		event   string
		match   bool
	}{
		{"*", "trust.updated", true},
		{"*", "anything", true},
		{"trust.*", "trust.updated", true},
		{"trust.*", "trust.decayed", true},
		{"trust.*", "trust", true},
		{"trust.*", "actor.registered", false},
		{"trust.updated", "trust.updated", true},
		{"trust.updated", "trust.decayed", false},
	}
	for _, tc := range cases {
		sp := MustSubscriptionPattern(tc.pattern)
		et := MustEventType(tc.event)
		if sp.Matches(et) != tc.match {
			t.Fatalf("pattern %q matches %q: expected %v", tc.pattern, tc.event, tc.match)
		}
	}
}

func TestDomainScope_Valid(t *testing.T) {
	t.Parallel()
	cases := []string{"code_review", "financial", "deploy_staging", "a.b.c"}
	for _, v := range cases {
		_, err := NewDomainScope(v)
		if err != nil {
			t.Fatalf("NewDomainScope(%q) failed: %v", v, err)
		}
	}
}

func TestDomainScope_Invalid(t *testing.T) {
	t.Parallel()
	cases := []string{"", "Capital", "123", ".leading"}
	for _, v := range cases {
		_, err := NewDomainScope(v)
		if err == nil {
			t.Fatalf("expected error for DomainScope(%q)", v)
		}
	}
}

// --- State Machines ---

func TestLifecycleState_ValidTransitions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		from LifecycleState
		to   LifecycleState
	}{
		{LifecycleDormant, LifecycleActivating},
		{LifecycleActivating, LifecycleActive},
		{LifecycleActivating, LifecycleDormant},
		{LifecycleActive, LifecycleProcessing},
		{LifecycleActive, LifecycleDeactivating},
		{LifecycleProcessing, LifecycleEmitting},
		{LifecycleProcessing, LifecycleActive},
		{LifecycleEmitting, LifecycleActive},
		{LifecycleDeactivating, LifecycleDormant},
	}
	for _, tc := range cases {
		result, err := tc.from.TransitionTo(tc.to)
		if err != nil {
			t.Fatalf("%s → %s failed: %v", tc.from, tc.to, err)
		}
		if result != tc.to {
			t.Fatalf("expected %s, got %s", tc.to, result)
		}
	}
}

func TestLifecycleState_InvalidTransitions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		from LifecycleState
		to   LifecycleState
	}{
		{LifecycleDormant, LifecycleActive},
		{LifecycleDormant, LifecycleProcessing},
		{LifecycleActive, LifecycleDormant},
		{LifecycleEmitting, LifecycleDormant},
		{LifecycleProcessing, LifecycleDeactivating},
	}
	for _, tc := range cases {
		result, err := tc.from.TransitionTo(tc.to)
		if err == nil {
			t.Fatalf("%s → %s should have failed", tc.from, tc.to)
		}
		if result != tc.from {
			t.Fatalf("expected state unchanged (%s), got %s", tc.from, result)
		}
		var transErr *InvalidLifecycleTransitionError
		if !errors.As(err, &transErr) {
			t.Fatalf("expected InvalidLifecycleTransitionError, got %T", err)
		}
		if transErr.From != tc.from || transErr.To != tc.to {
			t.Fatal("error fields mismatch")
		}
	}
}

func TestLifecycleState_ValidTransitionsList(t *testing.T) {
	t.Parallel()
	valid := LifecycleDormant.ValidTransitions()
	if len(valid) != 1 || valid[0] != LifecycleActivating {
		t.Fatalf("unexpected transitions from Dormant: %v", valid)
	}
}

func TestActorStatus_ValidTransitions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		from ActorStatus
		to   ActorStatus
	}{
		{ActorStatusActive, ActorStatusSuspended},
		{ActorStatusActive, ActorStatusMemorial},
		{ActorStatusSuspended, ActorStatusActive},
		{ActorStatusSuspended, ActorStatusMemorial},
	}
	for _, tc := range cases {
		result, err := tc.from.TransitionTo(tc.to)
		if err != nil {
			t.Fatalf("%s → %s failed: %v", tc.from, tc.to, err)
		}
		if result != tc.to {
			t.Fatalf("expected %s, got %s", tc.to, result)
		}
	}
}

func TestActorStatus_MemorialIsTerminal(t *testing.T) {
	t.Parallel()
	targets := []ActorStatus{ActorStatusActive, ActorStatusSuspended, ActorStatusMemorial}
	for _, target := range targets {
		_, err := ActorStatusMemorial.TransitionTo(target)
		if err == nil {
			t.Fatalf("Memorial → %s should have failed", target)
		}
	}
	valid := ActorStatusMemorial.ValidTransitions()
	if len(valid) != 0 {
		t.Fatalf("Memorial should have no valid transitions, got %v", valid)
	}
}

func TestActorStatus_InvalidTransitions(t *testing.T) {
	t.Parallel()
	_, err := ActorStatusActive.TransitionTo(ActorStatusActive)
	if err == nil {
		t.Fatal("Active → Active should have failed")
	}
	var transErr *InvalidActorTransitionError
	if !errors.As(err, &transErr) {
		t.Fatalf("expected InvalidActorTransitionError, got %T", err)
	}
}

// --- Error Types ---

func TestErrors_AreValidationErrors(t *testing.T) {
	t.Parallel()
	errs := []error{
		&OutOfRangeError{Field: "test", Value: 5, Min: 0, Max: 1},
		&IntOutOfRangeError{Field: "test", Value: 5, Min: 0, Max: 1},
		&InvalidFormatError{Field: "test", Value: "x", Expected: "y"},
		&EmptyRequiredError{Field: "test"},
		&InvalidLifecycleTransitionError{From: LifecycleDormant, To: LifecycleActive},
		&InvalidActorTransitionError{From: ActorStatusMemorial, To: ActorStatusActive},
	}
	for _, err := range errs {
		var ve ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("%T does not implement ValidationError", err)
		}
		if err.Error() == "" {
			t.Fatalf("%T has empty error message", err)
		}
	}
}

func TestErrors_VisitorDispatch(t *testing.T) {
	t.Parallel()
	dispatched := false
	v := &testValidationVisitor{onOutOfRange: func(e *OutOfRangeError) { dispatched = true }}
	err := &OutOfRangeError{Field: "Score", Value: 5, Min: 0, Max: 1}
	err.Accept(v)
	if !dispatched {
		t.Fatal("visitor not dispatched")
	}
}

type testValidationVisitor struct {
	onOutOfRange func(*OutOfRangeError)
}

func (v *testValidationVisitor) VisitOutOfRange(e *OutOfRangeError)                                   { v.onOutOfRange(e) }
func (v *testValidationVisitor) VisitIntOutOfRange(*IntOutOfRangeError)                               {}
func (v *testValidationVisitor) VisitInvalidFormat(*InvalidFormatError)                               {}
func (v *testValidationVisitor) VisitEmptyRequired(*EmptyRequiredError)                               {}
func (v *testValidationVisitor) VisitInvalidLifecycleTransition(*InvalidLifecycleTransitionError)     {}
func (v *testValidationVisitor) VisitInvalidActorTransition(*InvalidActorTransitionError)             {}
func (v *testValidationVisitor) VisitInvalidLifecycleState(*InvalidLifecycleStateError)               {}
func (v *testValidationVisitor) VisitInvalidActorStatus(*InvalidActorStatusError)                     {}

// --- Remaining ID types ---

func TestEdgeID(t *testing.T) {
	t.Parallel()
	id, err := NewEdgeID("01912345-6789-7abc-8def-0123456789ab")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "01912345-6789-7abc-8def-0123456789ab" {
		t.Fatal("unexpected value")
	}
	if _, err := NewEdgeID("not-valid"); err == nil {
		t.Fatal("expected error")
	}
	b, _ := json.Marshal(id)
	var out EdgeID
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if id.Value() != out.Value() {
		t.Fatal("round-trip mismatch")
	}
}

func TestConversationID(t *testing.T) {
	t.Parallel()
	id, err := NewConversationID("conv-123")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "conv-123" {
		t.Fatal("unexpected value")
	}
	if id.Value() != "conv-123" {
		t.Fatal("unexpected value")
	}
	if _, err := NewConversationID(""); err == nil {
		t.Fatal("expected error for empty")
	}
	b, _ := json.Marshal(id)
	var out ConversationID
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if id.Value() != out.Value() {
		t.Fatal("round-trip mismatch")
	}
	// unmarshal empty should fail
	if err := json.Unmarshal([]byte(`""`), &out); err == nil {
		t.Fatal("expected error for empty unmarshal")
	}
}

func TestSystemURI(t *testing.T) {
	t.Parallel()
	id, err := NewSystemURI("eg://example.com/system1")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "eg://example.com/system1" {
		t.Fatal("unexpected value")
	}
	if id.Value() != "eg://example.com/system1" {
		t.Fatal("unexpected value")
	}
	if _, err := NewSystemURI(""); err == nil {
		t.Fatal("expected error for empty")
	}
	_ = MustSystemURI("test://uri")
	b, _ := json.Marshal(id)
	var out SystemURI
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`""`), &out); err == nil {
		t.Fatal("expected error for empty unmarshal")
	}
}

func TestEnvelopeID(t *testing.T) {
	t.Parallel()
	id, err := NewEnvelopeID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatal("unexpected value")
	}
	if id.Value() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatal("unexpected value")
	}
	if _, err := NewEnvelopeID("not-uuid"); err == nil {
		t.Fatal("expected error")
	}
	_ = MustEnvelopeID("550e8400-e29b-41d4-a716-446655440000")
	b, _ := json.Marshal(id)
	var out EnvelopeID
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
}

func TestTreatyID(t *testing.T) {
	t.Parallel()
	id, err := NewTreatyID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatal("unexpected value")
	}
	if id.Value() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatal("unexpected value")
	}
	if _, err := NewTreatyID("bad"); err == nil {
		t.Fatal("expected error")
	}
	_ = MustTreatyID("550e8400-e29b-41d4-a716-446655440000")
	b, _ := json.Marshal(id)
	var out TreatyID
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
}

func TestPrimitiveID(t *testing.T) {
	t.Parallel()
	id, err := NewPrimitiveID("trust_score")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "trust_score" {
		t.Fatal("unexpected value")
	}
	if id.Value() != "trust_score" {
		t.Fatal("unexpected value")
	}
	if _, err := NewPrimitiveID(""); err == nil {
		t.Fatal("expected error for empty")
	}
	_ = MustPrimitiveID("test")
	b, _ := json.Marshal(id)
	var out PrimitiveID
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`""`), &out); err == nil {
		t.Fatal("expected error for empty unmarshal")
	}
}

// --- JSON round-trips for remaining constrained types ---

func TestWeight_JSON(t *testing.T) {
	t.Parallel()
	w := MustWeight(-0.5)
	b, _ := json.Marshal(w)
	if string(b) != "-0.5" {
		t.Fatalf("expected -0.5, got %s", string(b))
	}
	var out Weight
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Value() != -0.5 {
		t.Fatal("round-trip mismatch")
	}
	if err := json.Unmarshal([]byte("5.0"), &out); err == nil {
		t.Fatal("expected error for invalid weight")
	}
}

func TestActivation_JSON(t *testing.T) {
	t.Parallel()
	a := MustActivation(0.75)
	if a.Value() != 0.75 {
		t.Fatal("unexpected value")
	}
	if a.String() != "0.75" {
		t.Fatal("unexpected string")
	}
	b, _ := json.Marshal(a)
	var out Activation
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Value() != 0.75 {
		t.Fatal("round-trip mismatch")
	}
	if err := json.Unmarshal([]byte("5.0"), &out); err == nil {
		t.Fatal("expected error")
	}
}

func TestLayer_JSON(t *testing.T) {
	t.Parallel()
	l := MustLayer(7)
	if l.String() != "7" {
		t.Fatal("unexpected string")
	}
	b, _ := json.Marshal(l)
	var out Layer
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Value() != 7 {
		t.Fatal("round-trip mismatch")
	}
	if err := json.Unmarshal([]byte("99"), &out); err == nil {
		t.Fatal("expected error")
	}
}

func TestCadence_JSON(t *testing.T) {
	t.Parallel()
	c := MustCadence(5)
	if c.String() != "5" {
		t.Fatal("unexpected string")
	}
	b, _ := json.Marshal(c)
	var out Cadence
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Value() != 5 {
		t.Fatal("round-trip mismatch")
	}
	if err := json.Unmarshal([]byte("0"), &out); err == nil {
		t.Fatal("expected error")
	}
}

func TestTick_JSON(t *testing.T) {
	t.Parallel()
	tk := MustTick(42)
	if tk.String() != "42" {
		t.Fatal("unexpected string")
	}
	b, _ := json.Marshal(tk)
	var out Tick
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Value() != 42 {
		t.Fatal("round-trip mismatch")
	}
	if err := json.Unmarshal([]byte("-1"), &out); err == nil {
		t.Fatal("expected error")
	}
}

func TestDuration_JSON(t *testing.T) {
	t.Parallel()
	d := MustDuration(1_000_000_000)
	if d.String() != "1000000000ns" {
		t.Fatal("unexpected string")
	}
	b, _ := json.Marshal(d)
	var out Duration
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Value() != 1_000_000_000 {
		t.Fatal("round-trip mismatch")
	}
	if err := json.Unmarshal([]byte("-1"), &out); err == nil {
		t.Fatal("expected error")
	}
}

// --- JSON for remaining ID types ---

func TestHash_JSON(t *testing.T) {
	t.Parallel()
	h := MustHash("ab" + "c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1")
	b, _ := json.Marshal(h)
	var out Hash
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if h.Value() != out.Value() {
		t.Fatal("round-trip mismatch")
	}
}

func TestActorID_JSON(t *testing.T) {
	t.Parallel()
	id := MustActorID("agent_alpha")
	b, _ := json.Marshal(id)
	var out ActorID
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if id.Value() != out.Value() {
		t.Fatal("round-trip mismatch")
	}
}

func TestEventType_JSON(t *testing.T) {
	t.Parallel()
	et := MustEventType("trust.updated")
	if et.Value() != "trust.updated" {
		t.Fatal("unexpected value")
	}
	b, _ := json.Marshal(et)
	var out EventType
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if et.Value() != out.Value() {
		t.Fatal("round-trip mismatch")
	}
	if err := json.Unmarshal([]byte(`"BAD"`), &out); err == nil {
		t.Fatal("expected error")
	}
}

func TestSubscriptionPattern_JSON(t *testing.T) {
	t.Parallel()
	sp := MustSubscriptionPattern("trust.*")
	if sp.Value() != "trust.*" {
		t.Fatal("unexpected value")
	}
	b, _ := json.Marshal(sp)
	var out SubscriptionPattern
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if sp.Value() != out.Value() {
		t.Fatal("round-trip mismatch")
	}
	if err := json.Unmarshal([]byte(`"BAD PATTERN"`), &out); err == nil {
		t.Fatal("expected error")
	}
}

func TestSubscriptionPattern_Invalid(t *testing.T) {
	t.Parallel()
	cases := []string{"", "BAD", "a.*.b", ".*"}
	for _, v := range cases {
		_, err := NewSubscriptionPattern(v)
		if err == nil {
			t.Fatalf("expected error for SubscriptionPattern(%q)", v)
		}
	}
}

func TestDomainScope_JSON(t *testing.T) {
	t.Parallel()
	ds := MustDomainScope("code_review")
	if ds.Value() != "code_review" {
		t.Fatal("unexpected value")
	}
	b, _ := json.Marshal(ds)
	var out DomainScope
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if ds.Value() != out.Value() {
		t.Fatal("round-trip mismatch")
	}
	if err := json.Unmarshal([]byte(`"BAD"`), &out); err == nil {
		t.Fatal("expected error")
	}
}

func TestPublicKey_JSON(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	key[0] = 0xAB
	pk := MustPublicKey(key)
	if pk.String() != "ab00000000000000000000000000000000000000000000000000000000000000" {
		t.Fatalf("unexpected string: %s", pk.String())
	}
	b, _ := json.Marshal(pk)
	var out PublicKey
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Bytes()[0] != 0xAB {
		t.Fatal("round-trip mismatch")
	}
	// invalid hex
	if err := json.Unmarshal([]byte(`"not-hex"`), &out); err == nil {
		t.Fatal("expected error")
	}
	// wrong length hex
	if err := json.Unmarshal([]byte(`"abcd"`), &out); err == nil {
		t.Fatal("expected error for wrong length")
	}
}

func TestSignature_JSON(t *testing.T) {
	t.Parallel()
	sig := MustSignature(make([]byte, 64))
	if len(sig.String()) != 128 {
		t.Fatal("expected 128 hex chars")
	}
	b, _ := json.Marshal(sig)
	var out Signature
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`"not-hex"`), &out); err == nil {
		t.Fatal("expected error")
	}
	if err := json.Unmarshal([]byte(`"abcd"`), &out); err == nil {
		t.Fatal("expected error for wrong length")
	}
}

// --- State machine IsValid ---

func TestLifecycleState_IsValid(t *testing.T) {
	t.Parallel()
	if !LifecycleDormant.IsValid() {
		t.Fatal("Dormant should be valid")
	}
	if LifecycleState("bogus").IsValid() {
		t.Fatal("bogus should be invalid")
	}
}

func TestActorStatus_IsValid(t *testing.T) {
	t.Parallel()
	if !ActorStatusActive.IsValid() {
		t.Fatal("Active should be valid")
	}
	if ActorStatus("bogus").IsValid() {
		t.Fatal("bogus should be invalid")
	}
}

// --- Must* panic paths ---

func TestMustActorID_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustActorID("")
	t.Fatal("should have panicked")
}

func TestMustHash_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustHash("bad")
	t.Fatal("should have panicked")
}

func TestMustFieldPath_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustFieldPath("")
	t.Fatal("should have panicked")
}

func TestMustDomainScope_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustDomainScope("")
	t.Fatal("should have panicked")
}

func TestMustPublicKey_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustPublicKey([]byte{1, 2, 3})
	t.Fatal("should have panicked")
}

func TestMustSignature_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustSignature([]byte{1, 2, 3})
	t.Fatal("should have panicked")
}

func TestMustWeight_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustWeight(5.0)
	t.Fatal("should have panicked")
}

func TestMustActivation_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustActivation(5.0)
	t.Fatal("should have panicked")
}

func TestMustCadence_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustCadence(0)
	t.Fatal("should have panicked")
}

func TestMustTick_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustTick(-1)
	t.Fatal("should have panicked")
}

func TestMustDuration_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustDuration(-1)
	t.Fatal("should have panicked")
}

func TestMustEdgeID_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustEdgeID("bad")
	t.Fatal("should have panicked")
}

func TestMustConversationID_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustConversationID("")
	t.Fatal("should have panicked")
}

func TestMustEnvelopeID_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustEnvelopeID("bad")
	t.Fatal("should have panicked")
}

func TestMustTreatyID_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustTreatyID("bad")
	t.Fatal("should have panicked")
}

func TestMustPrimitiveID_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustPrimitiveID("")
	t.Fatal("should have panicked")
}

func TestMustEventType_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustEventType("BAD")
	t.Fatal("should have panicked")
}

func TestMustSubscriptionPattern_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustSubscriptionPattern("BAD PATTERN")
	t.Fatal("should have panicked")
}

// --- Cross-type: constrained types are comparable (usable as map keys) ---

func TestConstrainedTypes_Comparable(t *testing.T) {
	t.Parallel()
	// Verify these can be used as map keys
	scoreMap := map[Score]bool{MustScore(0.5): true}
	if !scoreMap[MustScore(0.5)] {
		t.Fatal("Score not comparable")
	}
	layerMap := map[Layer]bool{MustLayer(3): true}
	if !layerMap[MustLayer(3)] {
		t.Fatal("Layer not comparable")
	}
}

func TestTypedIDs_Comparable(t *testing.T) {
	t.Parallel()
	id1 := MustEventID("01912345-6789-7abc-8def-0123456789ab")
	id2 := MustEventID("01912345-6789-7abc-8def-0123456789ab")
	if id1 != id2 {
		t.Fatal("EventIDs with same value should be equal")
	}
	m := map[EventID]bool{id1: true}
	if !m[id2] {
		t.Fatal("EventID not usable as map key")
	}
}
