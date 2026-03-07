package grammar_test

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

func newTestGrammar(t *testing.T) (*grammar.Grammar, *graph.Graph, types.ActorID, event.Event) {
	t.Helper()
	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	g := graph.New(s, as)

	actorID := types.MustActorID("actor_test00000000000000000000001")
	bootstrap, err := g.Bootstrap(actorID, testSigner{})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	return grammar.New(g), g, actorID, bootstrap
}

var convID = types.MustConversationID("conv_test000000000000000000000001")
var signer = testSigner{}
var ctx = context.Background()

// --- Vertex operations ---

func TestEmit(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	ev, err := gr.Emit(ctx, actorID, "Hello world", convID, []types.EventID{bootstrap.ID()}, signer)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if ev.Type() != event.EventTypeGrammarEmit {
		t.Errorf("Type = %v, want grammar.emit", ev.Type().Value())
	}
	content, ok := ev.Content().(event.GrammarEmitContent)
	if !ok {
		t.Fatalf("Content type = %T, want GrammarEmitContent", ev.Content())
	}
	if content.Body != "Hello world" {
		t.Errorf("Body = %q, want %q", content.Body, "Hello world")
	}
}

func TestEmitRequiresCause(t *testing.T) {
	gr, _, actorID, _ := newTestGrammar(t)

	_, err := gr.Emit(ctx, actorID, "body", convID, nil, signer)
	if err == nil {
		t.Fatal("expected error for emit without causes")
	}
}

func TestRespond(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	emitEv, _ := gr.Emit(ctx, actorID, "original", convID, []types.EventID{bootstrap.ID()}, signer)

	ev, err := gr.Respond(ctx, actorID, "reply", emitEv.ID(), convID, signer)
	if err != nil {
		t.Fatalf("Respond: %v", err)
	}
	if ev.Type() != event.EventTypeGrammarRespond {
		t.Errorf("Type = %v, want grammar.respond", ev.Type().Value())
	}
	content := ev.Content().(event.GrammarRespondContent)
	if content.Parent != emitEv.ID() {
		t.Errorf("Parent = %v, want %v", content.Parent, emitEv.ID())
	}
	// Verify causal link
	if len(ev.Causes()) != 1 || ev.Causes()[0] != emitEv.ID() {
		t.Errorf("Causes should be [%v], got %v", emitEv.ID(), ev.Causes())
	}
}

func TestDerive(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	emitEv, _ := gr.Emit(ctx, actorID, "source material", convID, []types.EventID{bootstrap.ID()}, signer)

	ev, err := gr.Derive(ctx, actorID, "derived work", emitEv.ID(), convID, signer)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if ev.Type() != event.EventTypeGrammarDerive {
		t.Errorf("Type = %v, want grammar.derive", ev.Type().Value())
	}
	content := ev.Content().(event.GrammarDeriveContent)
	if content.Source != emitEv.ID() {
		t.Errorf("Source = %v, want %v", content.Source, emitEv.ID())
	}
}

func TestExtend(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	part1, _ := gr.Emit(ctx, actorID, "part 1", convID, []types.EventID{bootstrap.ID()}, signer)

	ev, err := gr.Extend(ctx, actorID, "part 2", part1.ID(), convID, signer)
	if err != nil {
		t.Fatalf("Extend: %v", err)
	}
	if ev.Type() != event.EventTypeGrammarExtend {
		t.Errorf("Type = %v, want grammar.extend", ev.Type().Value())
	}
	content := ev.Content().(event.GrammarExtendContent)
	if content.Previous != part1.ID() {
		t.Errorf("Previous = %v, want %v", content.Previous, part1.ID())
	}
}

func TestRetract(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	emitEv, _ := gr.Emit(ctx, actorID, "regrettable", convID, []types.EventID{bootstrap.ID()}, signer)

	ev, err := gr.Retract(ctx, actorID, emitEv.ID(), "changed my mind", convID, signer)
	if err != nil {
		t.Fatalf("Retract: %v", err)
	}
	if ev.Type() != event.EventTypeGrammarRetract {
		t.Errorf("Type = %v, want grammar.retract", ev.Type().Value())
	}
	content := ev.Content().(event.GrammarRetractContent)
	if content.Target != emitEv.ID() {
		t.Errorf("Target = %v, want %v", content.Target, emitEv.ID())
	}
	if content.Reason != "changed my mind" {
		t.Errorf("Reason = %q, want %q", content.Reason, "changed my mind")
	}
}

func TestAnnotate(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	emitEv, _ := gr.Emit(ctx, actorID, "content", convID, []types.EventID{bootstrap.ID()}, signer)

	ev, err := gr.Annotate(ctx, actorID, emitEv.ID(), "mood", "happy", convID, signer)
	if err != nil {
		t.Fatalf("Annotate: %v", err)
	}
	if ev.Type() != event.EventTypeGrammarAnnotate {
		t.Errorf("Type = %v, want grammar.annotate", ev.Type().Value())
	}
	content := ev.Content().(event.GrammarAnnotateContent)
	if content.Key != "mood" || content.Value != "happy" {
		t.Errorf("Key/Value = %q/%q, want mood/happy", content.Key, content.Value)
	}
}

func TestMerge(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	ev1, _ := gr.Emit(ctx, actorID, "branch A", convID, []types.EventID{bootstrap.ID()}, signer)
	ev2, _ := gr.Emit(ctx, actorID, "branch B", convID, []types.EventID{bootstrap.ID()}, signer)

	ev, err := gr.Merge(ctx, actorID, "merged", []types.EventID{ev1.ID(), ev2.ID()}, convID, signer)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if ev.Type() != event.EventTypeGrammarMerge {
		t.Errorf("Type = %v, want grammar.merge", ev.Type().Value())
	}
	if len(ev.Causes()) != 2 {
		t.Errorf("Causes count = %d, want 2", len(ev.Causes()))
	}
}

func TestMergeRequiresTwoSources(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	_, err := gr.Merge(ctx, actorID, "bad merge", []types.EventID{bootstrap.ID()}, convID, signer)
	if err == nil {
		t.Fatal("expected error for merge with single source")
	}
}

// --- Edge operations ---

func TestAcknowledge(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	emitEv, _ := gr.Emit(ctx, actorID, "content", convID, []types.EventID{bootstrap.ID()}, signer)
	targetActor := types.MustActorID("actor_test00000000000000000000002")

	ev, err := gr.Acknowledge(ctx, actorID, emitEv.ID(), targetActor, convID, signer)
	if err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	if ev.Type() != event.EventTypeEdgeCreated {
		t.Errorf("Type = %v, want edge.created", ev.Type().Value())
	}
	content := ev.Content().(event.EdgeCreatedContent)
	if content.EdgeType != event.EdgeTypeEndorsement {
		t.Errorf("EdgeType = %v, want Endorsement", content.EdgeType)
	}
	if content.Weight.Value() != 0 {
		t.Errorf("Weight = %v, want 0 (content-free)", content.Weight.Value())
	}
	if content.Direction != event.EdgeDirectionCentripetal {
		t.Errorf("Direction = %v, want Centripetal", content.Direction)
	}
}

func TestPropagate(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	emitEv, _ := gr.Emit(ctx, actorID, "content", convID, []types.EventID{bootstrap.ID()}, signer)
	targetActor := types.MustActorID("actor_test00000000000000000000002")

	ev, err := gr.Propagate(ctx, actorID, emitEv.ID(), targetActor, convID, signer)
	if err != nil {
		t.Fatalf("Propagate: %v", err)
	}
	content := ev.Content().(event.EdgeCreatedContent)
	if content.EdgeType != event.EdgeTypeReference {
		t.Errorf("EdgeType = %v, want Reference", content.EdgeType)
	}
	if content.Direction != event.EdgeDirectionCentrifugal {
		t.Errorf("Direction = %v, want Centrifugal", content.Direction)
	}
}

func TestEndorse(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	emitEv, _ := gr.Emit(ctx, actorID, "content", convID, []types.EventID{bootstrap.ID()}, signer)
	targetActor := types.MustActorID("actor_test00000000000000000000002")

	ev, err := gr.Endorse(ctx, actorID, emitEv.ID(), targetActor,
		types.MustWeight(0.8), types.Some(types.MustDomainScope("code")), convID, signer)
	if err != nil {
		t.Fatalf("Endorse: %v", err)
	}
	content := ev.Content().(event.EdgeCreatedContent)
	if content.EdgeType != event.EdgeTypeEndorsement {
		t.Errorf("EdgeType = %v, want Endorsement", content.EdgeType)
	}
	if content.Weight.Value() != 0.8 {
		t.Errorf("Weight = %v, want 0.8", content.Weight.Value())
	}
}

func TestSubscribe(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	targetActor := types.MustActorID("actor_test00000000000000000000002")

	ev, err := gr.Subscribe(ctx, actorID, targetActor,
		types.Some(types.MustDomainScope("updates")), bootstrap.ID(), convID, signer)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	content := ev.Content().(event.EdgeCreatedContent)
	if content.EdgeType != event.EdgeTypeSubscription {
		t.Errorf("EdgeType = %v, want Subscription", content.EdgeType)
	}
}

func TestChannel(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	targetActor := types.MustActorID("actor_test00000000000000000000002")

	ev, err := gr.Channel(ctx, actorID, targetActor, types.None[types.DomainScope](), bootstrap.ID(), convID, signer)
	if err != nil {
		t.Fatalf("Channel: %v", err)
	}
	content := ev.Content().(event.EdgeCreatedContent)
	if content.EdgeType != event.EdgeTypeChannel {
		t.Errorf("EdgeType = %v, want Channel", content.EdgeType)
	}
}

func TestDelegate(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	targetActor := types.MustActorID("actor_test00000000000000000000002")

	ev, err := gr.Delegate(ctx, actorID, targetActor,
		types.MustDomainScope("admin"), types.MustWeight(0.9), bootstrap.ID(), convID, signer)
	if err != nil {
		t.Fatalf("Delegate: %v", err)
	}
	content := ev.Content().(event.EdgeCreatedContent)
	if content.EdgeType != event.EdgeTypeDelegation {
		t.Errorf("EdgeType = %v, want Delegation", content.EdgeType)
	}
	if content.Direction != event.EdgeDirectionCentrifugal {
		t.Errorf("Direction = %v, want Centrifugal", content.Direction)
	}
}

func TestConsent(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	partyB := types.MustActorID("actor_test00000000000000000000002")

	ev, err := gr.Consent(ctx, actorID, partyB, "we agree to collaborate",
		types.MustDomainScope("project_x"), bootstrap.ID(), convID, signer)
	if err != nil {
		t.Fatalf("Consent: %v", err)
	}
	if ev.Type() != event.EventTypeGrammarConsent {
		t.Errorf("Type = %v, want grammar.consent", ev.Type().Value())
	}
	content := ev.Content().(event.GrammarConsentContent)
	if content.Parties[0] != actorID || content.Parties[1] != partyB {
		t.Errorf("Parties mismatch")
	}
}

func TestSever(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	targetActor := types.MustActorID("actor_test00000000000000000000002")
	subEv, _ := gr.Subscribe(ctx, actorID, targetActor, types.None[types.DomainScope](), bootstrap.ID(), convID, signer)

	// Sever needs an EdgeID — use a generated one
	edgeID := types.MustEdgeID(subEv.ID().Value())

	ev, err := gr.Sever(ctx, actorID, edgeID, subEv.ID(), convID, signer)
	if err != nil {
		t.Fatalf("Sever: %v", err)
	}
	if ev.Type() != event.EventTypeEdgeSuperseded {
		t.Errorf("Type = %v, want edge.superseded", ev.Type().Value())
	}
}

// --- Named functions ---

func TestRecommend(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	emitEv, _ := gr.Emit(ctx, actorID, "check this out", convID, []types.EventID{bootstrap.ID()}, signer)
	targetActor := types.MustActorID("actor_test00000000000000000000002")

	propEv, chanEv, err := gr.Recommend(ctx, actorID, emitEv.ID(), targetActor, convID, signer)
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}

	// Propagate event
	propContent := propEv.Content().(event.EdgeCreatedContent)
	if propContent.EdgeType != event.EdgeTypeReference {
		t.Errorf("Propagate EdgeType = %v, want Reference", propContent.EdgeType)
	}

	// Channel event (caused by propagate)
	chanContent := chanEv.Content().(event.EdgeCreatedContent)
	if chanContent.EdgeType != event.EdgeTypeChannel {
		t.Errorf("Channel EdgeType = %v, want Channel", chanContent.EdgeType)
	}
	if len(chanEv.Causes()) != 1 || chanEv.Causes()[0] != propEv.ID() {
		t.Errorf("Channel should be caused by propagate event")
	}
}

func TestInvite(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	targetActor := types.MustActorID("actor_test00000000000000000000002")

	endorseEv, subEv, err := gr.Invite(ctx, actorID, targetActor,
		types.MustWeight(0.7), types.Some(types.MustDomainScope("community")),
		bootstrap.ID(), convID, signer)
	if err != nil {
		t.Fatalf("Invite: %v", err)
	}

	// Endorse event
	endorseContent := endorseEv.Content().(event.EdgeCreatedContent)
	if endorseContent.EdgeType != event.EdgeTypeEndorsement {
		t.Errorf("Endorse EdgeType = %v, want Endorsement", endorseContent.EdgeType)
	}

	// Subscribe event (caused by endorse)
	subContent := subEv.Content().(event.EdgeCreatedContent)
	if subContent.EdgeType != event.EdgeTypeSubscription {
		t.Errorf("Subscribe EdgeType = %v, want Subscription", subContent.EdgeType)
	}
	if len(subEv.Causes()) != 1 || subEv.Causes()[0] != endorseEv.ID() {
		t.Errorf("Subscribe should be caused by endorse event")
	}
}

func TestForgive(t *testing.T) {
	gr, _, actorID, bootstrap := newTestGrammar(t)

	targetActor := types.MustActorID("actor_test00000000000000000000002")

	// Subscribe, then sever, then forgive
	subEv, _ := gr.Subscribe(ctx, actorID, targetActor, types.None[types.DomainScope](), bootstrap.ID(), convID, signer)
	edgeID := types.MustEdgeID(subEv.ID().Value())
	severEv, _ := gr.Sever(ctx, actorID, edgeID, subEv.ID(), convID, signer)

	forgiveEv, err := gr.Forgive(ctx, actorID, severEv.ID(), targetActor, types.None[types.DomainScope](), convID, signer)
	if err != nil {
		t.Fatalf("Forgive: %v", err)
	}

	// Forgive is a Subscribe caused by the sever
	content := forgiveEv.Content().(event.EdgeCreatedContent)
	if content.EdgeType != event.EdgeTypeSubscription {
		t.Errorf("Forgive EdgeType = %v, want Subscription", content.EdgeType)
	}
	if len(forgiveEv.Causes()) != 1 || forgiveEv.Causes()[0] != severEv.ID() {
		t.Errorf("Forgive should be caused by sever event")
	}
}

// --- Chain integrity ---

func TestAllOperationsAreOnChain(t *testing.T) {
	gr, g, actorID, bootstrap := newTestGrammar(t)

	targetActor := types.MustActorID("actor_test00000000000000000000002")

	// Execute every operation
	emitEv, _ := gr.Emit(ctx, actorID, "emit", convID, []types.EventID{bootstrap.ID()}, signer)
	gr.Respond(ctx, actorID, "respond", emitEv.ID(), convID, signer)
	gr.Derive(ctx, actorID, "derive", emitEv.ID(), convID, signer)
	gr.Extend(ctx, actorID, "extend", emitEv.ID(), convID, signer)
	gr.Retract(ctx, actorID, emitEv.ID(), "reason", convID, signer)
	gr.Annotate(ctx, actorID, emitEv.ID(), "k", "v", convID, signer)

	emit2, _ := gr.Emit(ctx, actorID, "emit2", convID, []types.EventID{bootstrap.ID()}, signer)
	gr.Merge(ctx, actorID, "merge", []types.EventID{emitEv.ID(), emit2.ID()}, convID, signer)

	gr.Acknowledge(ctx, actorID, emitEv.ID(), targetActor, convID, signer)
	gr.Propagate(ctx, actorID, emitEv.ID(), targetActor, convID, signer)
	gr.Endorse(ctx, actorID, emitEv.ID(), targetActor, types.MustWeight(0.5), types.None[types.DomainScope](), convID, signer)
	gr.Subscribe(ctx, actorID, targetActor, types.None[types.DomainScope](), bootstrap.ID(), convID, signer)
	gr.Channel(ctx, actorID, targetActor, types.None[types.DomainScope](), bootstrap.ID(), convID, signer)
	gr.Delegate(ctx, actorID, targetActor, types.MustDomainScope("admin"), types.MustWeight(0.5), bootstrap.ID(), convID, signer)
	gr.Consent(ctx, actorID, targetActor, "agree", types.MustDomainScope("scope"), bootstrap.ID(), convID, signer)

	// 1 bootstrap + 15 grammar operations = 16 events
	count, _ := g.Store().Count()
	if count != 16 {
		t.Errorf("Count = %d, want 16", count)
	}

	// Verify chain integrity
	if _, err := g.Store().VerifyChain(); err != nil {
		t.Errorf("Chain integrity violated: %v", err)
	}
}
