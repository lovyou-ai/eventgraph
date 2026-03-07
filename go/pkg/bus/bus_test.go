package bus_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/bus"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// testSigner implements event.Signer for tests.
type testSigner struct{}

func (testSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data[:min(64, len(data))])
	return types.MustSignature(sig), nil
}

func makeTestEvent(t *testing.T, s store.Store, eventType types.EventType) event.Event {
	t.Helper()
	signer := testSigner{}
	actorID := types.MustActorID("actor_test00000000000000000000001")
	registry := event.DefaultRegistry()
	factory := event.NewEventFactory(registry)

	// Ensure bootstrap exists
	count, _ := s.Count()
	if count == 0 {
		bf := event.NewBootstrapFactory(registry)
		bootstrap, err := bf.Init(actorID, signer)
		if err != nil {
			t.Fatalf("bootstrap: %v", err)
		}
		if _, err := s.Append(bootstrap); err != nil {
			t.Fatalf("append bootstrap: %v", err)
		}
	}

	head, _ := s.Head()
	causeID := head.Unwrap().ID()
	convID := types.MustConversationID("conv_test000000000000000000000001")

	causeEventID := types.MustEventID("019462a0-0000-7000-8000-000000000001")

	var content event.EventContent
	switch eventType {
	case event.EventTypeTrustUpdated:
		content = event.TrustUpdatedContent{
			Actor:    actorID,
			Previous: types.MustScore(0.0),
			Current:  types.MustScore(0.5),
			Domain:   types.MustDomainScope("general"),
			Cause:    causeEventID,
		}
	case event.EventTypeEdgeCreated:
		content = event.EdgeCreatedContent{
			From:      actorID,
			To:        types.MustActorID("actor_test00000000000000000000002"),
			EdgeType:  event.EdgeTypeTrust,
			Weight:    types.MustWeight(0.5),
			Direction: event.EdgeDirectionCentripetal,
			Scope:     types.Some(types.MustDomainScope("general")),
		}
	default:
		content = event.TrustUpdatedContent{
			Actor:    actorID,
			Previous: types.MustScore(0.0),
			Current:  types.MustScore(0.5),
			Domain:   types.MustDomainScope("general"),
			Cause:    causeEventID,
		}
	}

	ev, err := factory.Create(
		eventType,
		actorID,
		content,
		[]types.EventID{causeID},
		convID,
		s,
		signer,
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	stored, err := s.Append(ev)
	if err != nil {
		t.Fatalf("append event: %v", err)
	}
	return stored
}

func TestNewEventBus(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)
	defer b.Close()

	if b.Store() != s {
		t.Error("Store() should return the wrapped store")
	}
}

func TestNewEventBusDefaultBuffer(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 0) // should default to 256
	defer b.Close()

	if b.Store() != s {
		t.Error("Store() should return the wrapped store")
	}
}

func TestSubscribeAndPublish(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)
	defer b.Close()

	var received atomic.Int32
	done := make(chan struct{})

	pattern := types.MustSubscriptionPattern("*")
	b.Subscribe(pattern, func(ev event.Event) {
		received.Add(1)
		done <- struct{}{}
	})

	ev := makeTestEvent(t, s, event.EventTypeTrustUpdated)
	b.Publish(ev)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delivery")
	}

	if received.Load() != 1 {
		t.Errorf("expected 1 delivery, got %d", received.Load())
	}
}

func TestPatternMatching(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)
	defer b.Close()

	var trustCount, edgeCount atomic.Int32
	trustDone := make(chan struct{}, 10)
	edgeDone := make(chan struct{}, 10)

	// Subscribe only to trust.*
	b.Subscribe(types.MustSubscriptionPattern("trust.*"), func(ev event.Event) {
		trustCount.Add(1)
		trustDone <- struct{}{}
	})

	// Subscribe only to edge.*
	b.Subscribe(types.MustSubscriptionPattern("edge.*"), func(ev event.Event) {
		edgeCount.Add(1)
		edgeDone <- struct{}{}
	})

	// Publish a trust event
	ev1 := makeTestEvent(t, s, event.EventTypeTrustUpdated)
	b.Publish(ev1)

	select {
	case <-trustDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for trust delivery")
	}

	// Publish an edge event
	ev2 := makeTestEvent(t, s, event.EventTypeEdgeCreated)
	b.Publish(ev2)

	select {
	case <-edgeDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for edge delivery")
	}

	// Give async delivery a moment
	time.Sleep(50 * time.Millisecond)

	if trustCount.Load() != 1 {
		t.Errorf("trust subscriber: expected 1, got %d", trustCount.Load())
	}
	if edgeCount.Load() != 1 {
		t.Errorf("edge subscriber: expected 1, got %d", edgeCount.Load())
	}
}

func TestFanOut(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)
	defer b.Close()

	const numSubs = 5
	var received atomic.Int32
	var wg sync.WaitGroup
	wg.Add(numSubs)

	pattern := types.MustSubscriptionPattern("*")
	for i := 0; i < numSubs; i++ {
		b.Subscribe(pattern, func(ev event.Event) {
			received.Add(1)
			wg.Done()
		})
	}

	ev := makeTestEvent(t, s, event.EventTypeTrustUpdated)
	b.Publish(ev)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fan-out delivery")
	}

	if received.Load() != numSubs {
		t.Errorf("expected %d deliveries, got %d", numSubs, received.Load())
	}
}

func TestUnsubscribe(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)
	defer b.Close()

	var received atomic.Int32
	pattern := types.MustSubscriptionPattern("*")
	id := b.Subscribe(pattern, func(ev event.Event) {
		received.Add(1)
	})

	b.Unsubscribe(id)

	ev := makeTestEvent(t, s, event.EventTypeTrustUpdated)
	b.Publish(ev)

	// Give async delivery a chance (it shouldn't happen)
	time.Sleep(100 * time.Millisecond)

	if received.Load() != 0 {
		t.Errorf("expected 0 deliveries after unsubscribe, got %d", received.Load())
	}
}

func TestUnsubscribeNonexistent(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)
	defer b.Close()

	// Should not panic
	b.Unsubscribe(bus.SubscriptionID(9999))
}

func TestSlowSubscriberOverflow(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 2) // tiny buffer
	defer b.Close()

	blocker := make(chan struct{})
	var received atomic.Int32

	pattern := types.MustSubscriptionPattern("*")
	b.Subscribe(pattern, func(ev event.Event) {
		received.Add(1)
		<-blocker // block until released
	})

	// Publish more events than the buffer can hold
	for i := 0; i < 5; i++ {
		ev := makeTestEvent(t, s, event.EventTypeTrustUpdated)
		b.Publish(ev)
	}

	// Release the handler
	time.Sleep(50 * time.Millisecond)
	close(blocker)

	// Wait for delivery
	time.Sleep(100 * time.Millisecond)

	// Should have received at most buffer size + 1 (the one being processed + buffer)
	got := received.Load()
	if got > 3 {
		t.Errorf("slow subscriber should have dropped events, got %d", got)
	}
	if got == 0 {
		t.Error("should have received at least some events")
	}
}

func TestPublishAfterClose(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)

	var received atomic.Int32
	pattern := types.MustSubscriptionPattern("*")
	b.Subscribe(pattern, func(ev event.Event) {
		received.Add(1)
	})

	b.Close()

	ev := makeTestEvent(t, s, event.EventTypeTrustUpdated)
	b.Publish(ev) // should be a no-op

	time.Sleep(50 * time.Millisecond)

	if received.Load() != 0 {
		t.Errorf("expected 0 deliveries after close, got %d", received.Load())
	}
}

func TestCloseIdempotent(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)

	err1 := b.Close()
	err2 := b.Close()

	if err1 != nil {
		t.Errorf("first close: %v", err1)
	}
	if err2 != nil {
		t.Errorf("second close: %v", err2)
	}
}

func TestConcurrentSubscribePublish(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 64)
	defer b.Close()

	// Create events sequentially (store requires serial appends for hash chain)
	var events []event.Event
	for i := 0; i < 10; i++ {
		events = append(events, makeTestEvent(t, s, event.EventTypeTrustUpdated))
	}

	var wg sync.WaitGroup
	pattern := types.MustSubscriptionPattern("*")

	// Concurrently subscribe
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Subscribe(pattern, func(ev event.Event) {})
		}()
	}

	// Concurrently publish pre-created events
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(ev event.Event) {
			defer wg.Done()
			b.Publish(ev)
		}(events[i])
	}

	wg.Wait()
}

func TestLastPanic(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)
	defer b.Close()

	delivered := make(chan struct{})
	pattern := types.MustSubscriptionPattern("*")
	id := b.Subscribe(pattern, func(ev event.Event) {
		defer func() { delivered <- struct{}{} }()
		panic("test panic")
	})

	ev := makeTestEvent(t, s, event.EventTypeTrustUpdated)
	b.Publish(ev)

	// Wait for the panicking handler to run
	select {
	case <-delivered:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panicking handler")
	}

	// Give recovery a moment to store the panic value
	time.Sleep(50 * time.Millisecond)

	p := b.LastPanic(id)
	if p == nil {
		t.Fatal("expected non-nil LastPanic after handler panic")
	}
	if p != "test panic" {
		t.Errorf("LastPanic = %v, want %q", p, "test panic")
	}

	// Verify the goroutine survives — publish another event
	b.Publish(ev)
	select {
	case <-delivered:
	case <-time.After(time.Second):
		t.Fatal("subscriber goroutine did not survive after panic")
	}
}

func TestLastPanicNonexistent(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)
	defer b.Close()

	p := b.LastPanic(bus.SubscriptionID(9999))
	if p != nil {
		t.Errorf("expected nil for nonexistent subscription, got %v", p)
	}
}

func TestSubscriptionIDsAreUnique(t *testing.T) {
	s := store.NewInMemoryStore()
	b := bus.NewEventBus(s, 16)
	defer b.Close()

	pattern := types.MustSubscriptionPattern("*")
	ids := make(map[bus.SubscriptionID]bool)

	for i := 0; i < 100; i++ {
		id := b.Subscribe(pattern, func(ev event.Event) {})
		if ids[id] {
			t.Fatalf("duplicate subscription ID: %d", id)
		}
		ids[id] = true
	}
}
