package bus

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// closeTimeout is the maximum time Close() waits for subscriber goroutines to drain.
const closeTimeout = 30 * time.Second

// SubscriptionID identifies an active subscription.
type SubscriptionID uint64

// IBus wraps a Store with pub/sub fan-out.
type IBus interface {
	Store() store.Store
	Subscribe(pattern types.SubscriptionPattern, handler func(event.Event)) SubscriptionID
	Unsubscribe(id SubscriptionID)
	Publish(ev event.Event)
	Close() error
}

type subscription struct {
	id        SubscriptionID
	pattern   types.SubscriptionPattern
	handler   func(event.Event)
	buffer    chan event.Event
	closeOnce sync.Once
	lastPanic atomic.Value // stores the most recent recovered panic value, if any
	done      chan struct{} // closed when the delivery goroutine exits
}

// EventBus implements IBus with non-blocking delivery.
// Slow subscribers get dropped events, not blocked writers.
type EventBus struct {
	store       store.Store
	mu          sync.RWMutex
	subs        map[SubscriptionID]*subscription
	nextID      atomic.Uint64
	bufferSize  int
	closed      atomic.Bool
}

// NewEventBus creates a new EventBus wrapping the given Store.
func NewEventBus(s store.Store, bufferSize int) *EventBus {
	if bufferSize <= 0 {
		bufferSize = 256
	}
	b := &EventBus{
		store:      s,
		subs:       make(map[SubscriptionID]*subscription),
		bufferSize: bufferSize,
	}
	return b
}

func (b *EventBus) Store() store.Store { return b.store }

// Subscribe registers a handler for events matching the pattern.
// The handler is called asynchronously from a per-subscriber goroutine.
// Returns 0 if the bus is closed.
func (b *EventBus) Subscribe(pattern types.SubscriptionPattern, handler func(event.Event)) SubscriptionID {
	if b.closed.Load() {
		return 0
	}

	id := SubscriptionID(b.nextID.Add(1))
	ch := make(chan event.Event, b.bufferSize)

	sub := &subscription{
		id:      id,
		pattern: pattern,
		handler: handler,
		buffer:  ch,
		done:    make(chan struct{}),
	}

	// Register before starting goroutine so Publish can deliver immediately.
	b.mu.Lock()
	if b.closed.Load() {
		b.mu.Unlock()
		return 0
	}
	b.subs[id] = sub
	b.mu.Unlock()

	// Start delivery goroutine with panic recovery.
	// Panics are caught to prevent a misbehaving subscriber from killing delivery.
	// The panic value is stored on the subscription for diagnostic visibility.
	go func() {
		defer close(sub.done)
		for ev := range ch {
			func() {
				defer func() {
					if r := recover(); r != nil {
						sub.lastPanic.Store(r)
					}
				}()
				handler(ev)
			}()
		}
	}()

	return id
}

// LastPanic returns the most recent panic value for a subscription, or nil.
func (b *EventBus) LastPanic(id SubscriptionID) any {
	b.mu.RLock()
	sub, ok := b.subs[id]
	b.mu.RUnlock()
	if !ok {
		return nil
	}
	return sub.lastPanic.Load()
}

// Unsubscribe removes a subscription.
func (b *EventBus) Unsubscribe(id SubscriptionID) {
	b.mu.Lock()
	sub, ok := b.subs[id]
	if ok {
		delete(b.subs, id)
	}
	b.mu.Unlock()

	if ok {
		sub.closeOnce.Do(func() { close(sub.buffer) })
	}
}

// Publish delivers an event to all matching subscribers.
// Non-blocking: if a subscriber's buffer is full, the event is dropped.
func (b *EventBus) Publish(ev event.Event) {
	if b.closed.Load() {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sub := range b.subs {
		if sub.pattern.Matches(ev.Type()) {
			select {
			case sub.buffer <- ev:
				// delivered
			default:
				// buffer full — drop (overflow)
			}
		}
	}
}

// Close stops all subscriber goroutines.
func (b *EventBus) Close() error {
	if b.closed.Swap(true) {
		return nil // already closed
	}

	b.mu.Lock()
	subs := make(map[SubscriptionID]*subscription, len(b.subs))
	for k, v := range b.subs {
		subs[k] = v
	}
	b.subs = make(map[SubscriptionID]*subscription)
	b.mu.Unlock()

	for _, sub := range subs {
		sub.closeOnce.Do(func() { close(sub.buffer) })
	}
	// Wait for all subscriber goroutines to drain and exit, with a shared
	// deadline to prevent deadlock if a handler blocks indefinitely.
	// The deadline is global — remaining time is recalculated for each subscriber
	// so that one slow subscriber doesn't consume the entire budget.
	deadline := time.Now().Add(closeTimeout)
	for _, sub := range subs {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("eventbus: close timed out after %v waiting for subscriber goroutines", closeTimeout)
		}
		timer := time.NewTimer(remaining)
		select {
		case <-sub.done:
			timer.Stop()
		case <-timer.C:
			return fmt.Errorf("eventbus: close timed out after %v waiting for subscriber goroutines", closeTimeout)
		}
	}
	return nil
}
