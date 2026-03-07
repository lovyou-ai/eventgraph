package bus

import (
	"sync"
	"sync/atomic"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

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
	}

	// Start delivery goroutine with panic recovery.
	// Panics are caught to prevent a misbehaving subscriber from killing delivery.
	// The panic value is stored on the subscription for diagnostic visibility.
	go func() {
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

	b.mu.Lock()
	b.subs[id] = sub
	b.mu.Unlock()

	return id
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
	return nil
}
