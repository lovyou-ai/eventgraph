package compositions_test

import (
	"context"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/grammar"
	"github.com/lovyou-ai/eventgraph/go/pkg/graph"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

type testSigner struct{}

func (testSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data[:min(64, len(data))])
	return types.MustSignature(sig), nil
}

var signer = testSigner{}

func testPublicKey(b byte) types.PublicKey {
	key := make([]byte, 32)
	key[0] = b
	return types.MustPublicKey(key)
}

type testEnv struct {
	t       *testing.T
	ctx     context.Context
	graph   *graph.Graph
	grammar *grammar.Grammar
	store   store.Store
	actors  actor.IActorStore
	boot    event.Event
	convID  types.ConversationID
	system  types.ActorID
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	g := graph.New(s, as)
	if err := g.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	systemActor := types.MustActorID("actor_system0000000000000000000001")
	boot, err := g.Bootstrap(systemActor, signer)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	env := &testEnv{
		t:       t,
		ctx:     context.Background(),
		graph:   g,
		grammar: grammar.New(g),
		store:   s,
		actors:  as,
		boot:    boot,
		convID:  types.MustConversationID("conv_test000000000000000000000001"),
		system:  systemActor,
	}
	t.Cleanup(func() { g.Close() })
	return env
}

func (e *testEnv) actor(name string, pkByte byte, actorType event.ActorType) actor.IActor {
	e.t.Helper()
	a, err := e.actors.Register(testPublicKey(pkByte), name, actorType)
	if err != nil {
		e.t.Fatalf("Register %s: %v", name, err)
	}
	return a
}

func (e *testEnv) verifyChain() {
	e.t.Helper()
	result, err := e.store.VerifyChain()
	if err != nil {
		e.t.Fatalf("VerifyChain: %v", err)
	}
	if !result.Valid {
		e.t.Fatalf("chain integrity broken at length %d", result.Length)
	}
}

func (e *testEnv) ancestors(id types.EventID, depth int) []event.Event {
	e.t.Helper()
	q, err := e.graph.Query()
	if err != nil {
		e.t.Fatalf("Query: %v", err)
	}
	anc, err := q.Ancestors(id, depth)
	if err != nil {
		e.t.Fatalf("Ancestors: %v", err)
	}
	return anc
}

func (e *testEnv) descendants(id types.EventID, depth int) []event.Event {
	e.t.Helper()
	q, err := e.graph.Query()
	if err != nil {
		e.t.Fatalf("Query: %v", err)
	}
	desc, err := q.Descendants(id, depth)
	if err != nil {
		e.t.Fatalf("Descendants: %v", err)
	}
	return desc
}

func containsEvent(events []event.Event, id types.EventID) bool {
	for _, ev := range events {
		if ev.ID() == id {
			return true
		}
	}
	return false
}
