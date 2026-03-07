package layer0_test

import (
	"testing"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/primitive"
	"github.com/lovyou-ai/eventgraph/go/pkg/primitive/layer0"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

var (
	systemActor = types.MustActorID("actor_00000000000000000000000000000001")
	actor2      = types.MustActorID("actor_00000000000000000000000000000002")
	convID      = types.MustConversationID("conv_00000000000000000000000000000001")
)

type testSigner struct{}

func (testSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data)
	return types.MustSignature(sig), nil
}

type headFromStore struct{ s store.Store }

func (h headFromStore) Head() (types.Option[event.Event], error) { return h.s.Head() }

func bootstrapStore(t *testing.T) (store.Store, event.Event) {
	t.Helper()
	s := store.NewInMemoryStore()
	registry := event.DefaultRegistry()
	factory := event.NewBootstrapFactory(registry)
	ev, err := factory.Init(systemActor, testSigner{})
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if _, err := s.Append(ev); err != nil {
		t.Fatalf("append bootstrap: %v", err)
	}
	return s, ev
}

func chainEvent(t *testing.T, s store.Store, causes []types.EventID) event.Event {
	t.Helper()
	registry := event.DefaultRegistry()
	factory := event.NewEventFactory(registry)
	ev, err := factory.Create(
		event.EventTypeTrustUpdated, systemActor,
		event.TrustUpdatedContent{
			Actor: actor2, Previous: types.MustScore(0.5),
			Current: types.MustScore(0.6), Domain: types.MustDomainScope("test"),
			Cause: causes[0],
		},
		causes, convID, headFromStore{s}, testSigner{},
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	if _, err := s.Append(ev); err != nil {
		t.Fatalf("append: %v", err)
	}
	return ev
}

// --- Group 0: Core ---

func TestEventPrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	p := layer0.NewEventPrimitive(systemActor, s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Event" {
			t.Errorf("ID = %q, want Event", p.ID().Value())
		}
		if p.Layer().Value() != 0 {
			t.Errorf("Layer = %d, want 0", p.Layer().Value())
		}
		if p.Lifecycle() != types.LifecycleActive {
			t.Error("expected Active lifecycle")
		}
		if p.Cadence().Value() != 1 {
			t.Errorf("Cadence = %d, want 1", p.Cadence().Value())
		}
		if len(p.Subscriptions()) != 1 || p.Subscriptions()[0].Value() != "*" {
			t.Error("expected * subscription")
		}
	})

	t.Run("ValidEvents", func(t *testing.T) {
		ev := chainEvent(t, s, []types.EventID{bootstrap.ID()})
		h := primitive.NewHarness()
		mutations, err := h.Process(p, []event.Event{ev})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		if len(mutations) < 2 {
			t.Fatalf("expected at least 2 mutations (lastEventID, eventCount), got %d", len(mutations))
		}
		changes := h.StateChanges()
		if changes["lastEventID"] != ev.ID().Value() {
			t.Errorf("lastEventID = %v, want %v", changes["lastEventID"], ev.ID().Value())
		}
	})

	t.Run("BootstrapEvent", func(t *testing.T) {
		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{bootstrap})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["lastEventID"] != bootstrap.ID().Value() {
			t.Errorf("lastEventID = %v, want %v", changes["lastEventID"], bootstrap.ID().Value())
		}
	})
}

func TestEventStorePrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	p := layer0.NewEventStorePrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "EventStore" {
			t.Errorf("ID = %q, want EventStore", p.ID().Value())
		}
		if p.Layer().Value() != 0 {
			t.Errorf("Layer = %d, want 0", p.Layer().Value())
		}
	})

	t.Run("TracksState", func(t *testing.T) {
		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{bootstrap})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["eventCount"] != 1 {
			t.Errorf("eventCount = %v, want 1", changes["eventCount"])
		}
		if changes["lastHash"] != bootstrap.Hash().Value() {
			t.Errorf("lastHash = %v, want %v", changes["lastHash"], bootstrap.Hash().Value())
		}
	})
}

func TestClockPrimitive(t *testing.T) {
	p := layer0.NewClockPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Clock" {
			t.Errorf("ID = %q, want Clock", p.ID().Value())
		}
	})

	t.Run("UpdatesTick", func(t *testing.T) {
		h := primitive.NewHarness().WithTick(types.MustTick(42))
		_, err := h.Process(p, nil)
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["currentTick"] != 42 {
			t.Errorf("currentTick = %v, want 42", changes["currentTick"])
		}
		if _, ok := changes["lastTickTime"]; !ok {
			t.Error("expected lastTickTime")
		}
	})
}

func TestHashPrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	p := layer0.NewHashPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Hash" {
			t.Errorf("ID = %q, want Hash", p.ID().Value())
		}
	})

	t.Run("ValidHash", func(t *testing.T) {
		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{bootstrap})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["chainHead"] != bootstrap.Hash().Value() {
			t.Errorf("chainHead = %v, want %v", changes["chainHead"], bootstrap.Hash().Value())
		}
	})
}

func TestSelfPrimitive(t *testing.T) {
	reg := primitive.NewRegistry()
	p := layer0.NewSelfPrimitive(systemActor, reg)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Self" {
			t.Errorf("ID = %q, want Self", p.ID().Value())
		}
	})

	t.Run("TracksIdentity", func(t *testing.T) {
		h := primitive.NewHarness()
		_, err := h.Process(p, nil)
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["systemActorID"] != systemActor.Value() {
			t.Errorf("systemActorID = %v, want %v", changes["systemActorID"], systemActor.Value())
		}
	})
}

// --- Group 1: Causality ---

func TestCausalLinkPrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	p := layer0.NewCausalLinkPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "CausalLink" {
			t.Errorf("ID = %q, want CausalLink", p.ID().Value())
		}
	})

	t.Run("ValidCauses", func(t *testing.T) {
		ev := chainEvent(t, s, []types.EventID{bootstrap.ID()})
		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{ev})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["validLinks"] != 1 {
			t.Errorf("validLinks = %v, want 1", changes["validLinks"])
		}
		if changes["invalidLinks"] != 0 {
			t.Errorf("invalidLinks = %v, want 0", changes["invalidLinks"])
		}
	})

	t.Run("BootstrapSkipped", func(t *testing.T) {
		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{bootstrap})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["validLinks"] != 0 {
			t.Errorf("validLinks = %v, want 0 (bootstrap skipped)", changes["validLinks"])
		}
	})
}

func TestAncestryPrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev1 := chainEvent(t, s, []types.EventID{bootstrap.ID()})
	p := layer0.NewAncestryPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Ancestry" {
			t.Errorf("ID = %q, want Ancestry", p.ID().Value())
		}
	})

	t.Run("FindsAncestors", func(t *testing.T) {
		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{ev1})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		depth, ok := changes["lastQueryDepth"]
		if !ok {
			t.Fatal("expected lastQueryDepth")
		}
		if depth.(int) < 1 {
			t.Errorf("lastQueryDepth = %v, want >= 1", depth)
		}
	})
}

func TestDescendancyPrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	chainEvent(t, s, []types.EventID{bootstrap.ID()})
	p := layer0.NewDescendancyPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Descendancy" {
			t.Errorf("ID = %q, want Descendancy", p.ID().Value())
		}
	})

	t.Run("FindsDescendants", func(t *testing.T) {
		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{bootstrap})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		depth, ok := changes["lastQueryDepth"]
		if !ok {
			t.Fatal("expected lastQueryDepth")
		}
		if depth.(int) < 1 {
			t.Errorf("lastQueryDepth = %v, want >= 1", depth)
		}
	})
}

func TestFirstCausePrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev1 := chainEvent(t, s, []types.EventID{bootstrap.ID()})
	ev2 := chainEvent(t, s, []types.EventID{ev1.ID()})
	p := layer0.NewFirstCausePrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "FirstCause" {
			t.Errorf("ID = %q, want FirstCause", p.ID().Value())
		}
	})

	t.Run("FindsRoot", func(t *testing.T) {
		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{ev2})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		root, ok := changes["lastFirstCause"]
		if !ok {
			t.Fatal("expected lastFirstCause")
		}
		if root != bootstrap.ID().Value() {
			t.Errorf("lastFirstCause = %v, want bootstrap %v", root, bootstrap.ID().Value())
		}
	})
}

// --- Group 2: Identity ---

func TestActorIDPrimitive(t *testing.T) {
	p := layer0.NewActorIDPrimitive(systemActor)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "ActorID" {
			t.Errorf("ID = %q, want ActorID", p.ID().Value())
		}
		subs := p.Subscriptions()
		if len(subs) != 1 || subs[0].Value() != "actor.*" {
			t.Error("expected actor.* subscription")
		}
	})

	t.Run("CountsRegistrations", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		regEv, err := factory.Create(
			event.EventTypeActorRegistered, systemActor,
			event.ActorRegisteredContent{
				ActorID:   actor2,
				PublicKey: types.MustPublicKey(make([]byte, 32)),
				Type:      event.ActorTypeHuman,
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		s.Append(regEv)

		h := primitive.NewHarness()
		_, err = h.Process(p, []event.Event{regEv})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["registeredThisTick"] != 1 {
			t.Errorf("registeredThisTick = %v, want 1", changes["registeredThisTick"])
		}
	})
}

func TestActorRegistryPrimitive(t *testing.T) {
	p := layer0.NewActorRegistryPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "ActorRegistry" {
			t.Errorf("ID = %q, want ActorRegistry", p.ID().Value())
		}
	})

	t.Run("TracksLifecycleEvents", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)

		regEv, _ := factory.Create(
			event.EventTypeActorRegistered, systemActor,
			event.ActorRegisteredContent{
				ActorID:   actor2,
				PublicKey: types.MustPublicKey(make([]byte, 32)),
				Type:      event.ActorTypeHuman,
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(regEv)

		suspEv, _ := factory.Create(
			event.EventTypeActorSuspended, systemActor,
			event.ActorSuspendedContent{ActorID: actor2, Reason: bootstrap.ID()},
			[]types.EventID{regEv.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(suspEv)

		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{regEv, suspEv})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["activeCount"] != 1 {
			t.Errorf("activeCount = %v, want 1", changes["activeCount"])
		}
		if changes["suspendedCount"] != 1 {
			t.Errorf("suspendedCount = %v, want 1", changes["suspendedCount"])
		}
	})
}

func TestSignaturePrimitive(t *testing.T) {
	p := layer0.NewSignaturePrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Signature" {
			t.Errorf("ID = %q, want Signature", p.ID().Value())
		}
	})

	t.Run("CountsSigned", func(t *testing.T) {
		_, bootstrap := bootstrapStore(t)
		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{bootstrap})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["signedCount"] != 1 {
			t.Errorf("signedCount = %v, want 1", changes["signedCount"])
		}
	})
}

func TestVerifyPrimitive(t *testing.T) {
	p := layer0.NewVerifyPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Verify" {
			t.Errorf("ID = %q, want Verify", p.ID().Value())
		}
	})

	t.Run("VerifiesSignatures", func(t *testing.T) {
		_, bootstrap := bootstrapStore(t)
		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{bootstrap})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["verifiedCount"] != 1 {
			t.Errorf("verifiedCount = %v, want 1", changes["verifiedCount"])
		}
		if changes["failedCount"] != 0 {
			t.Errorf("failedCount = %v, want 0", changes["failedCount"])
		}
	})
}

// --- Group 3: Expectations ---

func TestExpectationPrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	p := layer0.NewExpectationPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Expectation" {
			t.Errorf("ID = %q, want Expectation", p.ID().Value())
		}
	})

	t.Run("TracksPending", func(t *testing.T) {
		// authority.requested events count as pending expectations
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeAuthorityRequested, systemActor,
			event.AuthorityRequestContent{
				Action: "test", Actor: actor2,
				Level: event.AuthorityLevelRequired, Justification: "test",
				Causes: types.MustNonEmpty([]types.EventID{bootstrap.ID()}),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{ev})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		changes := h.StateChanges()
		if changes["pendingExpectations"] != 1 {
			t.Errorf("pendingExpectations = %v, want 1", changes["pendingExpectations"])
		}
	})

	t.Run("NoPending", func(t *testing.T) {
		ev := chainEvent(t, s, []types.EventID{bootstrap.ID()})
		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["pendingExpectations"] != 0 {
			t.Error("expected 0 pending for non-authority event")
		}
	})
}

func TestTimeoutPrimitive(t *testing.T) {
	p := layer0.NewTimeoutPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Timeout" {
			t.Errorf("ID = %q, want Timeout", p.ID().Value())
		}
		subs := p.Subscriptions()
		if len(subs) != 1 || subs[0].Value() != "authority.*" {
			t.Error("expected authority.* subscription")
		}
	})

	t.Run("CountsTimeouts", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeAuthorityTimeout, systemActor,
			event.AuthorityTimeoutContent{
				RequestID: bootstrap.ID(),
				Level:     event.AuthorityLevelRecommended,
				Duration:  types.MustDuration(1_000_000_000),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		_, err := h.Process(p, []event.Event{ev})
		if err != nil {
			t.Fatalf("Process: %v", err)
		}
		if h.StateChanges()["timeoutCount"] != 1 {
			t.Errorf("timeoutCount = %v, want 1", h.StateChanges()["timeoutCount"])
		}
	})
}

func TestViolationPrimitive(t *testing.T) {
	p := layer0.NewViolationPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Violation" {
			t.Errorf("ID = %q, want Violation", p.ID().Value())
		}
	})

	t.Run("CountsViolations", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeViolationDetected, systemActor,
			event.ViolationDetectedContent{
				Expectation: bootstrap.ID(), Actor: actor2,
				Severity: event.SeverityLevelWarning, Description: "test",
				Evidence: types.MustNonEmpty([]types.EventID{bootstrap.ID()}),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["violationCount"] != 1 {
			t.Errorf("violationCount = %v, want 1", h.StateChanges()["violationCount"])
		}
	})
}

func TestSeverityPrimitive(t *testing.T) {
	p := layer0.NewSeverityPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Severity" {
			t.Errorf("ID = %q, want Severity", p.ID().Value())
		}
	})

	t.Run("ExtractsSeverity", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeViolationDetected, systemActor,
			event.ViolationDetectedContent{
				Expectation: bootstrap.ID(), Actor: actor2,
				Severity: event.SeverityLevelCritical, Description: "critical",
				Evidence: types.MustNonEmpty([]types.EventID{bootstrap.ID()}),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["lastSeverity"] != string(event.SeverityLevelCritical) {
			t.Errorf("lastSeverity = %v, want Critical", h.StateChanges()["lastSeverity"])
		}
	})
}

// --- Group 4: Trust ---

func TestTrustScorePrimitive(t *testing.T) {
	p := layer0.NewTrustScorePrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "TrustScore" {
			t.Errorf("ID = %q, want TrustScore", p.ID().Value())
		}
		subs := p.Subscriptions()
		if len(subs) != 1 || subs[0].Value() != "trust.*" {
			t.Error("expected trust.* subscription")
		}
	})

	t.Run("CountsScoreSnapshots", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeTrustScore, systemActor,
			event.TrustScoreContent{
				Actor: actor2,
				Metrics: event.NewTrustMetrics(
					actor2, types.MustScore(0.8), nil, types.MustScore(0.9),
					types.MustWeight(0.0), nil, types.NewTimestamp(time.Now()), types.MustScore(0.05),
				),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["scoreSnapshots"] != 1 {
			t.Errorf("scoreSnapshots = %v, want 1", h.StateChanges()["scoreSnapshots"])
		}
	})
}

func TestTrustUpdatePrimitive(t *testing.T) {
	p := layer0.NewTrustUpdatePrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "TrustUpdate" {
			t.Errorf("ID = %q, want TrustUpdate", p.ID().Value())
		}
	})

	t.Run("TracksUpdatesAndDecays", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)

		ev1, _ := factory.Create(
			event.EventTypeTrustUpdated, systemActor,
			event.TrustUpdatedContent{
				Actor: actor2, Previous: types.MustScore(0.5),
				Current: types.MustScore(0.7), Domain: types.MustDomainScope("test"),
				Cause: bootstrap.ID(),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev1)

		ev2, _ := factory.Create(
			event.EventTypeTrustDecayed, systemActor,
			event.TrustDecayedContent{
				Actor: actor2, Previous: types.MustScore(0.7),
				Current: types.MustScore(0.65), Elapsed: types.MustDuration(1_000_000_000),
				Rate: types.MustScore(0.05),
			},
			[]types.EventID{ev1.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev2)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev1, ev2})
		changes := h.StateChanges()
		if changes["trustUpdates"] != 1 {
			t.Errorf("trustUpdates = %v, want 1", changes["trustUpdates"])
		}
		if changes["trustDecays"] != 1 {
			t.Errorf("trustDecays = %v, want 1", changes["trustDecays"])
		}
	})
}

func TestCorroborationPrimitive(t *testing.T) {
	p := layer0.NewCorroborationPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Corroboration" {
			t.Errorf("ID = %q, want Corroboration", p.ID().Value())
		}
	})

	t.Run("TracksUniqueSources", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)

		ev1, _ := factory.Create(
			event.EventTypeTrustUpdated, systemActor,
			event.TrustUpdatedContent{
				Actor: actor2, Previous: types.MustScore(0.5),
				Current: types.MustScore(0.7), Domain: types.MustDomainScope("test"),
				Cause: bootstrap.ID(),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev1)

		ev2, _ := factory.Create(
			event.EventTypeTrustUpdated, actor2,
			event.TrustUpdatedContent{
				Actor: systemActor, Previous: types.MustScore(0.3),
				Current: types.MustScore(0.5), Domain: types.MustDomainScope("test"),
				Cause: ev1.ID(),
			},
			[]types.EventID{ev1.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev2)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev1, ev2})
		if h.StateChanges()["uniqueSources"] != 2 {
			t.Errorf("uniqueSources = %v, want 2", h.StateChanges()["uniqueSources"])
		}
	})
}

func TestContradictionPrimitive(t *testing.T) {
	p := layer0.NewContradictionPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Contradiction" {
			t.Errorf("ID = %q, want Contradiction", p.ID().Value())
		}
	})

	t.Run("DetectsContradiction", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)

		// Trust increase
		ev1, _ := factory.Create(
			event.EventTypeTrustUpdated, systemActor,
			event.TrustUpdatedContent{
				Actor: actor2, Previous: types.MustScore(0.5),
				Current: types.MustScore(0.7), Domain: types.MustDomainScope("test"),
				Cause: bootstrap.ID(),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev1)

		// Trust decrease
		ev2, _ := factory.Create(
			event.EventTypeTrustUpdated, actor2,
			event.TrustUpdatedContent{
				Actor: systemActor, Previous: types.MustScore(0.8),
				Current: types.MustScore(0.3), Domain: types.MustDomainScope("test"),
				Cause: ev1.ID(),
			},
			[]types.EventID{ev1.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev2)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev1, ev2})
		changes := h.StateChanges()
		if changes["contradictions"] != 1 {
			t.Errorf("contradictions = %v, want 1", changes["contradictions"])
		}
		if changes["increases"] != 1 {
			t.Errorf("increases = %v, want 1", changes["increases"])
		}
		if changes["decreases"] != 1 {
			t.Errorf("decreases = %v, want 1", changes["decreases"])
		}
	})

	t.Run("NoContradiction", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)

		ev1, _ := factory.Create(
			event.EventTypeTrustUpdated, systemActor,
			event.TrustUpdatedContent{
				Actor: actor2, Previous: types.MustScore(0.5),
				Current: types.MustScore(0.7), Domain: types.MustDomainScope("test"),
				Cause: bootstrap.ID(),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev1)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev1})
		if h.StateChanges()["contradictions"] != 0 {
			t.Error("expected no contradictions with only increases")
		}
	})
}

// --- Group 5: Confidence ---

func TestConfidencePrimitive(t *testing.T) {
	p := layer0.NewConfidencePrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Confidence" {
			t.Errorf("ID = %q, want Confidence", p.ID().Value())
		}
		subs := p.Subscriptions()
		if len(subs) != 1 || subs[0].Value() != "decision.*" {
			t.Error("expected decision.* subscription")
		}
	})

	t.Run("CountsDecisions", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeDecisionCostReport, systemActor,
			event.CostReportContent{
				PrimitiveID: types.MustPrimitiveID("Self"), TreeVersion: 1,
				TotalLeaves: 5, LLMLeaves: 1,
				MechanicalRate: types.MustScore(0.8), TotalTokens: 100,
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["decisionsThisTick"] != 1 {
			t.Errorf("decisionsThisTick = %v, want 1", h.StateChanges()["decisionsThisTick"])
		}
	})
}

func TestEvidencePrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	p := layer0.NewEvidencePrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Evidence" {
			t.Errorf("ID = %q, want Evidence", p.ID().Value())
		}
	})

	t.Run("CountsCausalLinks", func(t *testing.T) {
		ev := chainEvent(t, s, []types.EventID{bootstrap.ID()})
		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["evidenceLinks"] != 1 {
			t.Errorf("evidenceLinks = %v, want 1", h.StateChanges()["evidenceLinks"])
		}
	})
}

func TestRevisionPrimitive(t *testing.T) {
	p := layer0.NewRevisionPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Revision" {
			t.Errorf("ID = %q, want Revision", p.ID().Value())
		}
		subs := p.Subscriptions()
		if len(subs) != 1 || subs[0].Value() != "grammar.*" {
			t.Error("expected grammar.* subscription")
		}
	})

	t.Run("CountsRetractions", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeGrammarRetract, systemActor,
			event.GrammarRetractContent{Target: bootstrap.ID(), Reason: "correction"},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["retractions"] != 1 {
			t.Errorf("retractions = %v, want 1", h.StateChanges()["retractions"])
		}
	})
}

func TestUncertaintyPrimitive(t *testing.T) {
	p := layer0.NewUncertaintyPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Uncertainty" {
			t.Errorf("ID = %q, want Uncertainty", p.ID().Value())
		}
	})

	t.Run("CountsEscalations", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeAuthorityRequested, systemActor,
			event.AuthorityRequestContent{
				Action: "risky", Actor: actor2,
				Level: event.AuthorityLevelRequired, Justification: "uncertain",
				Causes: types.MustNonEmpty([]types.EventID{bootstrap.ID()}),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["escalations"] != 1 {
			t.Errorf("escalations = %v, want 1", h.StateChanges()["escalations"])
		}
	})
}

// --- Group 6: Instrumentation ---

func TestInstrumentationSpecPrimitive(t *testing.T) {
	p := layer0.NewInstrumentationSpecPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "InstrumentationSpec" {
			t.Errorf("ID = %q, want InstrumentationSpec", p.ID().Value())
		}
	})

	t.Run("TracksPrimitives", func(t *testing.T) {
		h := primitive.NewHarness()
		h.Process(p, nil)
		// Harness auto-registers the primitive, so at least 1 in snapshot
		tracked := h.StateChanges()["primitivesTracked"]
		if tracked.(int) < 1 {
			t.Errorf("primitivesTracked = %v, want >= 1", tracked)
		}
	})
}

func TestCoverageCheckPrimitive(t *testing.T) {
	p := layer0.NewCoverageCheckPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "CoverageCheck" {
			t.Errorf("ID = %q, want CoverageCheck", p.ID().Value())
		}
		if p.Cadence().Value() != 5 {
			t.Errorf("Cadence = %d, want 5", p.Cadence().Value())
		}
	})

	t.Run("CountsActivePrimitives", func(t *testing.T) {
		h := primitive.NewHarness()
		h.Process(p, nil)
		active := h.StateChanges()["activePrimitives"]
		if active.(int) < 1 {
			t.Errorf("activePrimitives = %v, want >= 1", active)
		}
	})
}

func TestGapPrimitive(t *testing.T) {
	p := layer0.NewGapPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Gap" {
			t.Errorf("ID = %q, want Gap", p.ID().Value())
		}
	})

	t.Run("DetectsGap", func(t *testing.T) {
		h := primitive.NewHarness()
		h.Process(p, nil) // no events = gap
		changes := h.StateChanges()
		if changes["gapDetected"] != 1 {
			t.Errorf("gapDetected = %v, want 1", changes["gapDetected"])
		}
		if changes["eventsInTick"] != 0 {
			t.Errorf("eventsInTick = %v, want 0", changes["eventsInTick"])
		}
	})

	t.Run("NoGap", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		ev := chainEvent(t, s, []types.EventID{bootstrap.ID()})
		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["gapDetected"] != 0 {
			t.Error("expected no gap with events")
		}
	})
}

func TestBlindPrimitive(t *testing.T) {
	p := layer0.NewBlindPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Blind" {
			t.Errorf("ID = %q, want Blind", p.ID().Value())
		}
		if p.Cadence().Value() != 5 {
			t.Errorf("Cadence = %d, want 5", p.Cadence().Value())
		}
	})

	t.Run("CountsDormant", func(t *testing.T) {
		h := primitive.NewHarness()
		h.Process(p, nil)
		// All auto-registered primitives are Active, so dormant should be 0
		if h.StateChanges()["dormantPrimitives"] != 0 {
			t.Errorf("dormantPrimitives = %v, want 0", h.StateChanges()["dormantPrimitives"])
		}
	})
}

// --- Group 7: Query ---

func TestPathQueryPrimitive(t *testing.T) {
	s, _ := bootstrapStore(t)
	p := layer0.NewPathQueryPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "PathQuery" {
			t.Errorf("ID = %q, want PathQuery", p.ID().Value())
		}
		subs := p.Subscriptions()
		if len(subs) != 1 || subs[0].Value() != "query.*" {
			t.Error("expected query.* subscription")
		}
	})

	t.Run("CountsQueries", func(t *testing.T) {
		_, bootstrap := bootstrapStore(t)
		h := primitive.NewHarness()
		h.Process(p, []event.Event{bootstrap})
		if h.StateChanges()["queriesProcessed"] != 1 {
			t.Errorf("queriesProcessed = %v, want 1", h.StateChanges()["queriesProcessed"])
		}
	})
}

func TestSubgraphExtractPrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	p := layer0.NewSubgraphExtractPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "SubgraphExtract" {
			t.Errorf("ID = %q, want SubgraphExtract", p.ID().Value())
		}
	})

	t.Run("MeasuresSubgraph", func(t *testing.T) {
		ev := chainEvent(t, s, []types.EventID{bootstrap.ID()})
		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		size := h.StateChanges()["lastSubgraphSize"]
		if size.(int) < 1 {
			t.Errorf("lastSubgraphSize = %v, want >= 1", size)
		}
	})
}

func TestAnnotatePrimitive(t *testing.T) {
	p := layer0.NewAnnotatePrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Annotate" {
			t.Errorf("ID = %q, want Annotate", p.ID().Value())
		}
	})

	t.Run("CountsAnnotations", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeGrammarAnnotate, systemActor,
			event.GrammarAnnotateContent{Target: bootstrap.ID(), Key: "tag", Value: "important"},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["annotations"] != 1 {
			t.Errorf("annotations = %v, want 1", h.StateChanges()["annotations"])
		}
	})
}

func TestTimelinePrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	p := layer0.NewTimelinePrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Timeline" {
			t.Errorf("ID = %q, want Timeline", p.ID().Value())
		}
	})

	t.Run("CountsTotalEvents", func(t *testing.T) {
		chainEvent(t, s, []types.EventID{bootstrap.ID()})
		h := primitive.NewHarness()
		h.Process(p, nil)
		count := h.StateChanges()["totalEvents"]
		if count.(int) < 2 {
			t.Errorf("totalEvents = %v, want >= 2", count)
		}
	})
}

// --- Group 8: Integrity ---

func TestHashChainPrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	p := layer0.NewHashChainPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "HashChain" {
			t.Errorf("ID = %q, want HashChain", p.ID().Value())
		}
	})

	t.Run("TracksChain", func(t *testing.T) {
		h := primitive.NewHarness()
		h.Process(p, []event.Event{bootstrap})
		changes := h.StateChanges()
		if changes["chainHead"] != bootstrap.Hash().Value() {
			t.Errorf("chainHead = %v, want %v", changes["chainHead"], bootstrap.Hash().Value())
		}
		if changes["chainLength"].(int) < 1 {
			t.Errorf("chainLength = %v, want >= 1", changes["chainLength"])
		}
	})
}

func TestChainVerifyPrimitive(t *testing.T) {
	s, _ := bootstrapStore(t)
	p := layer0.NewChainVerifyPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "ChainVerify" {
			t.Errorf("ID = %q, want ChainVerify", p.ID().Value())
		}
		if p.Cadence().Value() != 10 {
			t.Errorf("Cadence = %d, want 10", p.Cadence().Value())
		}
	})

	t.Run("VerifiesChain", func(t *testing.T) {
		h := primitive.NewHarness()
		h.Process(p, nil)
		changes := h.StateChanges()
		if changes["chainValid"] != true {
			t.Errorf("chainValid = %v, want true", changes["chainValid"])
		}
	})
}

func TestWitnessPrimitive(t *testing.T) {
	p := layer0.NewWitnessPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Witness" {
			t.Errorf("ID = %q, want Witness", p.ID().Value())
		}
	})

	t.Run("CountsWitnessed", func(t *testing.T) {
		_, bootstrap := bootstrapStore(t)
		h := primitive.NewHarness()
		h.Process(p, []event.Event{bootstrap})
		if h.StateChanges()["witnessed"] != 1 {
			t.Errorf("witnessed = %v, want 1", h.StateChanges()["witnessed"])
		}
	})
}

func TestIntegrityViolationPrimitive(t *testing.T) {
	p := layer0.NewIntegrityViolationPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "IntegrityViolation" {
			t.Errorf("ID = %q, want IntegrityViolation", p.ID().Value())
		}
	})

	t.Run("NoBreaks", func(t *testing.T) {
		_, bootstrap := bootstrapStore(t)
		h := primitive.NewHarness()
		h.Process(p, []event.Event{bootstrap})
		if h.StateChanges()["chainBreaks"] != 0 {
			t.Errorf("chainBreaks = %v, want 0", h.StateChanges()["chainBreaks"])
		}
	})
}

// --- Group 9: Deception ---

func TestPatternPrimitive(t *testing.T) {
	p := layer0.NewPatternPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Pattern" {
			t.Errorf("ID = %q, want Pattern", p.ID().Value())
		}
	})

	t.Run("TracksTypes", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		ev := chainEvent(t, s, []types.EventID{bootstrap.ID()})
		h := primitive.NewHarness()
		h.Process(p, []event.Event{bootstrap, ev})
		changes := h.StateChanges()
		if changes["totalEvents"] != 2 {
			t.Errorf("totalEvents = %v, want 2", changes["totalEvents"])
		}
		if changes["uniqueTypes"].(int) < 1 {
			t.Errorf("uniqueTypes = %v, want >= 1", changes["uniqueTypes"])
		}
	})
}

func TestDeceptionIndicatorPrimitive(t *testing.T) {
	p := layer0.NewDeceptionIndicatorPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "DeceptionIndicator" {
			t.Errorf("ID = %q, want DeceptionIndicator", p.ID().Value())
		}
		subs := p.Subscriptions()
		if len(subs) != 2 {
			t.Errorf("expected 2 subscriptions (trust.*, violation.*), got %d", len(subs))
		}
	})

	t.Run("CountsIndicators", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeTrustDecayed, systemActor,
			event.TrustDecayedContent{
				Actor: actor2, Previous: types.MustScore(0.7),
				Current: types.MustScore(0.5), Elapsed: types.MustDuration(1_000_000_000),
				Rate: types.MustScore(0.1),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["indicators"] != 1 {
			t.Errorf("indicators = %v, want 1", h.StateChanges()["indicators"])
		}
	})
}

func TestSuspicionPrimitive(t *testing.T) {
	p := layer0.NewSuspicionPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Suspicion" {
			t.Errorf("ID = %q, want Suspicion", p.ID().Value())
		}
	})

	t.Run("TracksSuspects", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeTrustDecayed, systemActor,
			event.TrustDecayedContent{
				Actor: actor2, Previous: types.MustScore(0.6),
				Current: types.MustScore(0.3), Elapsed: types.MustDuration(1_000_000_000),
				Rate: types.MustScore(0.15),
			},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["suspectedActors"] != 1 {
			t.Errorf("suspectedActors = %v, want 1", h.StateChanges()["suspectedActors"])
		}
	})
}

func TestQuarantinePrimitive(t *testing.T) {
	p := layer0.NewQuarantinePrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Quarantine" {
			t.Errorf("ID = %q, want Quarantine", p.ID().Value())
		}
		subs := p.Subscriptions()
		if len(subs) != 1 || subs[0].Value() != "actor.*" {
			t.Error("expected actor.* subscription")
		}
	})

	t.Run("CountsQuarantined", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		registry := event.DefaultRegistry()
		factory := event.NewEventFactory(registry)
		ev, _ := factory.Create(
			event.EventTypeActorSuspended, systemActor,
			event.ActorSuspendedContent{ActorID: actor2, Reason: bootstrap.ID()},
			[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
		)
		s.Append(ev)

		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["quarantinedThisTick"] != 1 {
			t.Errorf("quarantinedThisTick = %v, want 1", h.StateChanges()["quarantinedThisTick"])
		}
	})
}

// --- Group 10: Health ---

func TestGraphHealthPrimitive(t *testing.T) {
	s, _ := bootstrapStore(t)
	p := layer0.NewGraphHealthPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "GraphHealth" {
			t.Errorf("ID = %q, want GraphHealth", p.ID().Value())
		}
	})

	t.Run("TracksHealth", func(t *testing.T) {
		h := primitive.NewHarness()
		h.Process(p, nil)
		changes := h.StateChanges()
		if changes["eventCount"].(int) < 1 {
			t.Errorf("eventCount = %v, want >= 1", changes["eventCount"])
		}
	})
}

func TestInvariantPrimitive(t *testing.T) {
	p := layer0.NewInvariantPrimitive()

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Invariant" {
			t.Errorf("ID = %q, want Invariant", p.ID().Value())
		}
	})

	t.Run("NoViolationsOnValidEvents", func(t *testing.T) {
		s, bootstrap := bootstrapStore(t)
		ev := chainEvent(t, s, []types.EventID{bootstrap.ID()})
		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["invariantViolations"] != 0 {
			t.Error("expected 0 violations for valid events")
		}
	})
}

func TestInvariantCheckPrimitive(t *testing.T) {
	s, _ := bootstrapStore(t)
	p := layer0.NewInvariantCheckPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "InvariantCheck" {
			t.Errorf("ID = %q, want InvariantCheck", p.ID().Value())
		}
		if p.Cadence().Value() != 10 {
			t.Errorf("Cadence = %d, want 10", p.Cadence().Value())
		}
	})

	t.Run("ChecksChain", func(t *testing.T) {
		h := primitive.NewHarness()
		h.Process(p, nil)
		changes := h.StateChanges()
		if changes["chainIntact"] != true {
			t.Errorf("chainIntact = %v, want true", changes["chainIntact"])
		}
	})
}

func TestBootstrapPrimitive(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	p := layer0.NewBootstrapPrimitive(s)

	t.Run("Interface", func(t *testing.T) {
		if p.ID().Value() != "Bootstrap" {
			t.Errorf("ID = %q, want Bootstrap", p.ID().Value())
		}
		subs := p.Subscriptions()
		if len(subs) != 1 || subs[0].Value() != "system.*" {
			t.Error("expected system.* subscription")
		}
	})

	t.Run("DetectsBootstrap", func(t *testing.T) {
		h := primitive.NewHarness()
		h.Process(p, []event.Event{bootstrap})
		changes := h.StateChanges()
		if changes["bootstrapped"] != true {
			t.Errorf("bootstrapped = %v, want true", changes["bootstrapped"])
		}
		if changes["eventCount"].(int) < 1 {
			t.Errorf("eventCount = %v, want >= 1", changes["eventCount"])
		}
	})

	t.Run("NoBootstrapEvent", func(t *testing.T) {
		ev := chainEvent(t, s, []types.EventID{bootstrap.ID()})
		h := primitive.NewHarness()
		h.Process(p, []event.Event{ev})
		if h.StateChanges()["bootstrapped"] != false {
			t.Error("expected bootstrapped=false for non-bootstrap event")
		}
	})
}

// --- Registration ---

func TestAllPrimitivesRegister(t *testing.T) {
	s := store.NewInMemoryStore()
	reg := primitive.NewRegistry()

	prims := []primitive.Primitive{
		// Group 0: Core
		layer0.NewEventPrimitive(systemActor, s),
		layer0.NewEventStorePrimitive(s),
		layer0.NewClockPrimitive(),
		layer0.NewHashPrimitive(s),
		layer0.NewSelfPrimitive(systemActor, reg),
		// Group 1: Causality
		layer0.NewCausalLinkPrimitive(s),
		layer0.NewAncestryPrimitive(s),
		layer0.NewDescendancyPrimitive(s),
		layer0.NewFirstCausePrimitive(s),
		// Group 2: Identity
		layer0.NewActorIDPrimitive(systemActor),
		layer0.NewActorRegistryPrimitive(),
		layer0.NewSignaturePrimitive(),
		layer0.NewVerifyPrimitive(),
		// Group 3: Expectations
		layer0.NewExpectationPrimitive(s),
		layer0.NewTimeoutPrimitive(),
		layer0.NewViolationPrimitive(),
		layer0.NewSeverityPrimitive(),
		// Group 4: Trust
		layer0.NewTrustScorePrimitive(),
		layer0.NewTrustUpdatePrimitive(),
		layer0.NewCorroborationPrimitive(),
		layer0.NewContradictionPrimitive(),
		// Group 5: Confidence
		layer0.NewConfidencePrimitive(),
		layer0.NewEvidencePrimitive(s),
		layer0.NewRevisionPrimitive(),
		layer0.NewUncertaintyPrimitive(),
		// Group 6: Instrumentation
		layer0.NewInstrumentationSpecPrimitive(),
		layer0.NewCoverageCheckPrimitive(),
		layer0.NewGapPrimitive(),
		layer0.NewBlindPrimitive(),
		// Group 7: Query
		layer0.NewPathQueryPrimitive(s),
		layer0.NewSubgraphExtractPrimitive(s),
		layer0.NewAnnotatePrimitive(),
		layer0.NewTimelinePrimitive(s),
		// Group 8: Integrity
		layer0.NewHashChainPrimitive(s),
		layer0.NewChainVerifyPrimitive(s),
		layer0.NewWitnessPrimitive(),
		layer0.NewIntegrityViolationPrimitive(),
		// Group 9: Deception
		layer0.NewPatternPrimitive(),
		layer0.NewDeceptionIndicatorPrimitive(),
		layer0.NewSuspicionPrimitive(),
		layer0.NewQuarantinePrimitive(),
		// Group 10: Health
		layer0.NewGraphHealthPrimitive(s),
		layer0.NewInvariantPrimitive(),
		layer0.NewInvariantCheckPrimitive(s),
		layer0.NewBootstrapPrimitive(s),
	}

	for _, p := range prims {
		if err := reg.Register(p); err != nil {
			t.Errorf("Register %q: %v", p.ID().Value(), err)
		}
		if p.Layer().Value() != 0 {
			t.Errorf("%q: Layer = %d, want 0", p.ID().Value(), p.Layer().Value())
		}
		if p.Lifecycle() != types.LifecycleActive {
			t.Errorf("%q: Lifecycle = %v, want Active", p.ID().Value(), p.Lifecycle())
		}
		if len(p.Subscriptions()) == 0 {
			t.Errorf("%q: no subscriptions", p.ID().Value())
		}
	}

	if reg.Count() != 45 {
		t.Errorf("registered %d primitives, want 45", reg.Count())
	}
}
