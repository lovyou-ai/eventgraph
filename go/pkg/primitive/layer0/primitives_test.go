package layer0_test

import (
	"testing"

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

// --- Registration ---

func TestAllPrimitivesRegister(t *testing.T) {
	s := store.NewInMemoryStore()
	reg := primitive.NewRegistry()

	prims := []primitive.Primitive{
		layer0.NewEventPrimitive(systemActor, s),
		layer0.NewEventStorePrimitive(s),
		layer0.NewClockPrimitive(),
		layer0.NewHashPrimitive(s),
		layer0.NewSelfPrimitive(systemActor, reg),
		layer0.NewCausalLinkPrimitive(s),
		layer0.NewAncestryPrimitive(s),
		layer0.NewDescendancyPrimitive(s),
		layer0.NewFirstCausePrimitive(s),
		layer0.NewActorIDPrimitive(systemActor),
		layer0.NewActorRegistryPrimitive(),
		layer0.NewSignaturePrimitive(),
		layer0.NewVerifyPrimitive(),
	}

	for _, p := range prims {
		if err := reg.Register(p); err != nil {
			t.Errorf("Register %q: %v", p.ID().Value(), err)
		}
		if p.Layer().Value() != 0 {
			t.Errorf("%q: Layer = %d, want 0", p.ID().Value(), p.Layer().Value())
		}
		if p.Cadence().Value() != 1 {
			t.Errorf("%q: Cadence = %d, want 1", p.ID().Value(), p.Cadence().Value())
		}
		if p.Lifecycle() != types.LifecycleActive {
			t.Errorf("%q: Lifecycle = %v, want Active", p.ID().Value(), p.Lifecycle())
		}
		if len(p.Subscriptions()) == 0 {
			t.Errorf("%q: no subscriptions", p.ID().Value())
		}
	}

	if reg.Count() != 13 {
		t.Errorf("registered %d primitives, want 13", reg.Count())
	}
}
