package agent_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/agent"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/primitive"
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
	factory := event.NewBootstrapFactory(event.DefaultRegistry())
	ev, err := factory.Init(systemActor, testSigner{})
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if _, err := s.Append(ev); err != nil {
		t.Fatalf("append: %v", err)
	}
	return s, ev
}

func agentContentFor(eventType types.EventType) event.EventContent {
	switch eventType {
	case event.EventTypeAgentIdentityCreated:
		return event.AgentIdentityCreatedContent{AgentID: actor2, PublicKey: types.MustPublicKey(make([]byte, 32)), AgentType: "ai"}
	case event.EventTypeAgentIdentityRotated:
		return event.AgentIdentityRotatedContent{AgentID: actor2, NewKey: types.MustPublicKey(make([]byte, 32)), PreviousKey: types.MustPublicKey(make([]byte, 32))}
	case event.EventTypeAgentSoulImprinted:
		return event.AgentSoulImprintedContent{AgentID: actor2, Values: []string{"care"}}
	case event.EventTypeAgentModelBound:
		return event.AgentModelBoundContent{AgentID: actor2, ModelID: "test", CostTier: "standard"}
	case event.EventTypeAgentModelChanged:
		return event.AgentModelChangedContent{AgentID: actor2, PreviousModel: "a", NewModel: "b", Reason: "upgrade"}
	case event.EventTypeAgentMemoryUpdated:
		return event.AgentMemoryUpdatedContent{AgentID: actor2, Key: "k", Action: "set"}
	case event.EventTypeAgentStateChanged:
		return event.AgentStateChangedContent{AgentID: actor2, Previous: "Idle", Current: "Processing"}
	case event.EventTypeAgentAuthorityGranted:
		return event.AgentAuthorityGrantedContent{AgentID: actor2, Scope: types.MustDomainScope("test"), Grantor: systemActor}
	case event.EventTypeAgentAuthorityRevoked:
		return event.AgentAuthorityRevokedContent{AgentID: actor2, Scope: types.MustDomainScope("test"), Revoker: systemActor, Reason: "expired"}
	case event.EventTypeAgentTrustAssessed:
		return event.AgentTrustAssessedContent{AgentID: actor2, Target: systemActor, Previous: types.MustScore(0.5), Current: types.MustScore(0.6)}
	case event.EventTypeAgentBudgetAllocated:
		return event.AgentBudgetAllocatedContent{AgentID: actor2, TokenLimit: 1000, CostLimit: 1.0, TimeLimit: 60000000000}
	case event.EventTypeAgentBudgetExhausted:
		return event.AgentBudgetExhaustedContent{AgentID: actor2, Resource: "tokens"}
	case event.EventTypeAgentRoleAssigned:
		return event.AgentRoleAssignedContent{AgentID: actor2, Role: "Builder"}
	case event.EventTypeAgentLifespanStarted:
		return event.AgentLifespanStartedContent{AgentID: actor2, Started: types.Now()}
	case event.EventTypeAgentLifespanExtended:
		return event.AgentLifespanExtendedContent{AgentID: actor2, NewEnd: types.Now(), Reason: "more work"}
	case event.EventTypeAgentLifespanEnded:
		return event.AgentLifespanEndedContent{AgentID: actor2, Reason: "complete"}
	case event.EventTypeAgentGoalSet:
		return event.AgentGoalSetContent{AgentID: actor2, Goal: "test", Priority: 1}
	case event.EventTypeAgentGoalCompleted:
		return event.AgentGoalCompletedContent{AgentID: actor2, Goal: "test"}
	case event.EventTypeAgentGoalAbandoned:
		return event.AgentGoalAbandonedContent{AgentID: actor2, Goal: "test", Reason: "superseded"}
	case event.EventTypeAgentObserved:
		return event.AgentObservedContent{AgentID: actor2, EventCount: 1}
	case event.EventTypeAgentProbed:
		return event.AgentProbedContent{AgentID: actor2, Query: "test", Results: 1}
	case event.EventTypeAgentEvaluated:
		return event.AgentEvaluatedContent{AgentID: actor2, Subject: "test", Confidence: types.MustScore(0.9), Result: "pass"}
	case event.EventTypeAgentDecided:
		return event.AgentDecidedContent{AgentID: actor2, Action: "test", Confidence: types.MustScore(0.9)}
	case event.EventTypeAgentActed:
		return event.AgentActedContent{AgentID: actor2, Action: "test", Target: "graph"}
	case event.EventTypeAgentDelegated:
		return event.AgentDelegatedContent{AgentID: actor2, Delegate: systemActor, Task: "test"}
	case event.EventTypeAgentEscalated:
		return event.AgentEscalatedContent{AgentID: actor2, Authority: systemActor, Reason: "beyond capability"}
	case event.EventTypeAgentRefused:
		return event.AgentRefusedContent{AgentID: actor2, Action: "harm", Reason: "values violation"}
	case event.EventTypeAgentLearned:
		return event.AgentLearnedContent{AgentID: actor2, Lesson: "test", Source: "outcome"}
	case event.EventTypeAgentIntrospected:
		return event.AgentIntrospectedContent{AgentID: actor2, Observation: "self-check"}
	case event.EventTypeAgentCommunicated:
		return event.AgentCommunicatedContent{AgentID: actor2, Recipient: systemActor, Channel: "direct"}
	case event.EventTypeAgentRepaired:
		return event.AgentRepairedContent{AgentID: actor2, OriginalEvent: types.MustEventID("01912345-6789-7abc-8def-0123456789ab"), Correction: "fix"}
	case event.EventTypeAgentExpectationSet:
		return event.AgentExpectationSetContent{AgentID: actor2, Condition: "completion"}
	case event.EventTypeAgentExpectationMet:
		return event.AgentExpectationMetContent{AgentID: actor2, Condition: "completion"}
	case event.EventTypeAgentExpectationExpired:
		return event.AgentExpectationExpiredContent{AgentID: actor2, Condition: "timeout"}
	case event.EventTypeAgentConsentRequested:
		return event.AgentConsentRequestedContent{AgentID: actor2, Target: systemActor, Action: "collaborate"}
	case event.EventTypeAgentConsentGranted:
		return event.AgentConsentGrantedContent{AgentID: actor2, Requester: systemActor, Action: "collaborate"}
	case event.EventTypeAgentConsentDenied:
		return event.AgentConsentDeniedContent{AgentID: actor2, Requester: systemActor, Action: "collaborate", Reason: "busy"}
	case event.EventTypeAgentChannelOpened:
		return event.AgentChannelOpenedContent{AgentID: actor2, Peer: systemActor, Channel: "direct"}
	case event.EventTypeAgentChannelClosed:
		return event.AgentChannelClosedContent{AgentID: actor2, Peer: systemActor, Channel: "direct", Reason: "done"}
	case event.EventTypeAgentCompositionFormed:
		return event.AgentCompositionFormedContent{AgentID: actor2, Members: []types.ActorID{actor2, systemActor}, Purpose: "test"}
	case event.EventTypeAgentCompositionDissolved:
		return event.AgentCompositionDissolvedContent{AgentID: actor2, GroupID: "g1", Reason: "complete"}
	case event.EventTypeAgentCompositionJoined:
		return event.AgentCompositionJoinedContent{AgentID: actor2, GroupID: "g1"}
	case event.EventTypeAgentCompositionLeft:
		return event.AgentCompositionLeftContent{AgentID: actor2, GroupID: "g1", Reason: "done"}
	case event.EventTypeAgentAttenuated:
		return event.AgentAttenuatedContent{AgentID: actor2, Dimension: "scope", Previous: "full", Current: "limited", Reason: "budget"}
	case event.EventTypeAgentAttenuationLifted:
		return event.AgentAttenuationLiftedContent{AgentID: actor2, Dimension: "scope", Reason: "budget restored"}
	default:
		return event.AgentStateChangedContent{AgentID: actor2, Previous: "Idle", Current: "Processing"}
	}
}

func makeAgentEvent(t *testing.T, s store.Store, eventType types.EventType, causes []types.EventID) event.Event {
	t.Helper()
	factory := event.NewEventFactory(event.DefaultRegistry())
	content := agentContentFor(eventType)
	ev, err := factory.Create(
		eventType, systemActor, content,
		causes, convID, headFromStore{s}, testSigner{},
	)
	if err != nil {
		t.Fatalf("create event %s: %v", eventType.Value(), err)
	}
	if _, err := s.Append(ev); err != nil {
		t.Fatalf("append: %v", err)
	}
	return ev
}

func TestAllPrimitivesCount(t *testing.T) {
	prims := agent.AllPrimitives()
	if len(prims) != 28 {
		t.Errorf("AllPrimitives() returned %d, want 28", len(prims))
	}
}

func TestAllPrimitivesRegister(t *testing.T) {
	reg := primitive.NewRegistry()
	for _, p := range agent.AllPrimitives() {
		if err := reg.Register(p); err != nil {
			t.Errorf("Register %q: %v", p.ID().Value(), err)
		}
	}
	if reg.Count() != 28 {
		t.Errorf("registered %d primitives, want 28", reg.Count())
	}
}

func TestAllPrimitivesLayer(t *testing.T) {
	for _, p := range agent.AllPrimitives() {
		if p.Layer().Value() != 1 {
			t.Errorf("%q: Layer = %d, want 1", p.ID().Value(), p.Layer().Value())
		}
	}
}

func TestAllPrimitivesLifecycle(t *testing.T) {
	for _, p := range agent.AllPrimitives() {
		if p.Lifecycle() != types.LifecycleActive {
			t.Errorf("%q: Lifecycle = %v, want Active", p.ID().Value(), p.Lifecycle())
		}
	}
}

func TestAllPrimitivesHaveSubscriptions(t *testing.T) {
	for _, p := range agent.AllPrimitives() {
		if len(p.Subscriptions()) == 0 {
			t.Errorf("%q: no subscriptions", p.ID().Value())
		}
	}
}

func TestAllPrimitivesAgentPrefix(t *testing.T) {
	for _, p := range agent.AllPrimitives() {
		if !agent.IsAgentPrimitive(p.ID()) {
			t.Errorf("%q: expected agent. prefix", p.ID().Value())
		}
	}
}

func TestRegisterAll(t *testing.T) {
	reg := primitive.NewRegistry()
	if err := agent.RegisterAll(reg); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	if reg.Count() != 28 {
		t.Errorf("registered %d, want 28", reg.Count())
	}
	// Verify all are Active (RegisterAll activates them)
	for _, p := range agent.AllPrimitives() {
		if reg.Lifecycle(p.ID()) != types.LifecycleActive {
			t.Errorf("%q: not activated", p.ID().Value())
		}
	}
}

func TestRegisterAllDuplicate(t *testing.T) {
	reg := primitive.NewRegistry()
	if err := agent.RegisterAll(reg); err != nil {
		t.Fatalf("first RegisterAll: %v", err)
	}
	if err := agent.RegisterAll(reg); err == nil {
		t.Error("second RegisterAll should fail")
	}
}

func TestUniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range agent.AllPrimitives() {
		id := p.ID().Value()
		if seen[id] {
			t.Errorf("duplicate primitive ID: %q", id)
		}
		seen[id] = true
	}
}

// --- Individual primitive Process tests ---

func TestIdentityProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentIdentityCreated, []types.EventID{bootstrap.ID()})
	p := agent.NewIdentityPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestSoulProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentSoulImprinted, []types.EventID{bootstrap.ID()})
	p := agent.NewSoulPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) < 2 {
		t.Fatalf("expected at least 2 mutations (lastTick + imprinted), got %d", len(mutations))
	}
}

func TestModelProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentModelBound, []types.EventID{bootstrap.ID()})
	p := agent.NewModelPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestMemoryProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentMemoryUpdated, []types.EventID{bootstrap.ID()})
	p := agent.NewMemoryPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestStateProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentStateChanged, []types.EventID{bootstrap.ID()})
	p := agent.NewStatePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) < 2 {
		t.Fatalf("expected at least 2 mutations, got %d", len(mutations))
	}
}

func TestAuthorityProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentAuthorityGranted, []types.EventID{bootstrap.ID()})
	p := agent.NewAuthorityPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestTrustProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentTrustAssessed, []types.EventID{bootstrap.ID()})
	p := agent.NewTrustPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestBudgetProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentBudgetAllocated, []types.EventID{bootstrap.ID()})
	p := agent.NewBudgetPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestRoleProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentRoleAssigned, []types.EventID{bootstrap.ID()})
	p := agent.NewRolePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestLifespanProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentLifespanStarted, []types.EventID{bootstrap.ID()})
	p := agent.NewLifespanPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestGoalProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentGoalSet, []types.EventID{bootstrap.ID()})
	p := agent.NewGoalPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestObserveProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentObserved, []types.EventID{bootstrap.ID()})
	p := agent.NewObservePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestProbeProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentProbed, []types.EventID{bootstrap.ID()})
	p := agent.NewProbePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestEvaluateProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentEvaluated, []types.EventID{bootstrap.ID()})
	p := agent.NewEvaluatePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestDecideProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentDecided, []types.EventID{bootstrap.ID()})
	p := agent.NewDecidePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestActProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentActed, []types.EventID{bootstrap.ID()})
	p := agent.NewActPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestDelegateProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentDelegated, []types.EventID{bootstrap.ID()})
	p := agent.NewDelegatePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestEscalateProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentEscalated, []types.EventID{bootstrap.ID()})
	p := agent.NewEscalatePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestRefuseProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentRefused, []types.EventID{bootstrap.ID()})
	p := agent.NewRefusePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestLearnProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentLearned, []types.EventID{bootstrap.ID()})
	p := agent.NewLearnPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestIntrospectProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentIntrospected, []types.EventID{bootstrap.ID()})
	p := agent.NewIntrospectPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestCommunicateProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentCommunicated, []types.EventID{bootstrap.ID()})
	p := agent.NewCommunicatePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestRepairProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentRepaired, []types.EventID{bootstrap.ID()})
	p := agent.NewRepairPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestExpectProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentExpectationSet, []types.EventID{bootstrap.ID()})
	p := agent.NewExpectPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestConsentProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentConsentRequested, []types.EventID{bootstrap.ID()})
	p := agent.NewConsentPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestChannelProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentChannelOpened, []types.EventID{bootstrap.ID()})
	p := agent.NewChannelPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestCompositionProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentCompositionFormed, []types.EventID{bootstrap.ID()})
	p := agent.NewCompositionPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestAttenuationProcess(t *testing.T) {
	s, bootstrap := bootstrapStore(t)
	ev := makeAgentEvent(t, s, event.EventTypeAgentAttenuated, []types.EventID{bootstrap.ID()})
	p := agent.NewAttenuationPrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(mutations) == 0 {
		t.Fatal("expected mutations")
	}
}

func TestProcessWithNoMatchingEvents(t *testing.T) {
	// When a primitive receives events that don't match its subscriptions,
	// it should still produce state update mutations (at minimum lastTick).
	s, bootstrap := bootstrapStore(t)
	// Use a trust event — doesn't match most agent subscriptions
	factory := event.NewEventFactory(event.DefaultRegistry())
	ev, err := factory.Create(
		event.EventTypeTrustUpdated, systemActor,
		event.TrustUpdatedContent{
			Actor: actor2, Previous: types.MustScore(0.5),
			Current: types.MustScore(0.6), Domain: types.MustDomainScope("test"),
			Cause: bootstrap.ID(),
		},
		[]types.EventID{bootstrap.ID()}, convID, headFromStore{s}, testSigner{},
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	p := agent.NewRefusePrimitive()
	mutations, err := p.Process(types.MustTick(1), []event.Event{ev}, primitive.Snapshot{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	// Should still update refusals (0) and lastTick
	if len(mutations) < 2 {
		t.Errorf("expected at least 2 mutations, got %d", len(mutations))
	}
}
