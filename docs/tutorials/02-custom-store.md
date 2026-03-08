# Tutorial: Implement a Custom Store

The `Store` interface is how EventGraph persists events. This tutorial implements a file-based store that writes events to a JSON-lines file.

## The Store Interface

```go
type Store interface {
    Append(event Event) (Event, error)
    Get(id types.EventID) (Event, error)
    Head() types.Option[Event]
    Count() int
    VerifyChain() ChainVerification
    Close() error
}
```

Six methods. That's the entire contract. The conformance test suite verifies any implementation.

## Step 1: Define the Store

```go
package jsonlstore

import (
    "encoding/json"
    "os"
    "sync"

    "github.com/lovyou-ai/eventgraph/go/pkg/event"
    "github.com/lovyou-ai/eventgraph/go/pkg/store"
    "github.com/lovyou-ai/eventgraph/go/pkg/types"
)

type JSONLStore struct {
    mu     sync.Mutex
    file   *os.File
    events []event.Event
    index  map[string]int // event_id -> position
}

func New(path string) (*JSONLStore, error) {
    f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
        return nil, err
    }
    s := &JSONLStore{
        file:  f,
        index: make(map[string]int),
    }
    // Load existing events from file
    if err := s.loadExisting(); err != nil {
        f.Close()
        return nil, err
    }
    return s, nil
}
```

## Step 2: Implement Append

The critical method. Must maintain hash chain integrity:

```go
func (s *JSONLStore) Append(ev event.Event) (event.Event, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Verify chain link
    if len(s.events) > 0 {
        last := s.events[len(s.events)-1]
        if ev.PrevHash() != last.Hash() {
            return event.Event{}, store.ChainIntegrityError{
                Position: len(s.events),
                Detail:   "prev_hash does not match head hash",
            }
        }
    }

    // Serialize and write
    data, err := json.Marshal(eventToJSON(ev))
    if err != nil {
        return event.Event{}, err
    }
    data = append(data, '\n')
    if _, err := s.file.Write(data); err != nil {
        return event.Event{}, err
    }

    // Update in-memory index
    s.events = append(s.events, ev)
    s.index[ev.ID().Value()] = len(s.events) - 1

    return ev, nil
}
```

**Key invariant:** The chain link check is mandatory. If `prev_hash` doesn't match the head event's hash, reject the append. This is what makes the event graph tamper-evident.

## Step 3: Implement Get, Head, Count

```go
func (s *JSONLStore) Get(id types.EventID) (event.Event, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    pos, ok := s.index[id.Value()]
    if !ok {
        return event.Event{}, store.EventNotFoundError{EventID: id.Value()}
    }
    return s.events[pos], nil
}

func (s *JSONLStore) Head() types.Option[event.Event] {
    s.mu.Lock()
    defer s.mu.Unlock()
    if len(s.events) == 0 {
        return types.None[event.Event]()
    }
    return types.Some(s.events[len(s.events)-1])
}

func (s *JSONLStore) Count() int {
    s.mu.Lock()
    defer s.mu.Unlock()
    return len(s.events)
}
```

## Step 4: Implement VerifyChain

Walk the chain and verify every link:

```go
func (s *JSONLStore) VerifyChain() store.ChainVerification {
    s.mu.Lock()
    defer s.mu.Unlock()
    for i := 1; i < len(s.events); i++ {
        if s.events[i-1].Hash() != s.events[i].PrevHash() {
            return store.ChainVerification{Valid: false, Length: i}
        }
    }
    return store.ChainVerification{Valid: true, Length: len(s.events)}
}

func (s *JSONLStore) Close() error {
    return s.file.Close()
}
```

## Step 5: Run the Conformance Suite

Every store implementation must pass the shared conformance tests:

```go
package jsonlstore_test

import (
    "os"
    "testing"

    "github.com/lovyou-ai/eventgraph/go/pkg/store/storetest"
)

func TestConformance(t *testing.T) {
    storetest.Run(t, func() store.Store {
        f, _ := os.CreateTemp("", "eventgraph-*.jsonl")
        s, _ := jsonlstore.New(f.Name())
        t.Cleanup(func() {
            s.Close()
            os.Remove(f.Name())
        })
        return s
    })
}
```

The conformance suite tests:
- Append and retrieve events
- Hash chain integrity enforcement
- Concurrent access safety
- Chain verification
- Error handling (not found, broken chain)

If your store passes the conformance suite, it works with any EventGraph component.

## What's Next

- See `go/pkg/store/pgstore/` for the PostgreSQL implementation
- See `go/pkg/store/storetest/suite.go` for the full conformance test list
- Read `docs/interfaces.md` for the complete Store interface specification
