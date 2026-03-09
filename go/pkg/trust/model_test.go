package trust_test

import (
	"context"
	"testing"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/trust"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

func testPublicKey(b byte) types.PublicKey {
	key := make([]byte, 32)
	key[0] = b
	return types.MustPublicKey(key)
}

func testActor(t *testing.T, name string, b byte) actor.IActor {
	t.Helper()
	s := actor.NewInMemoryActorStore()
	a, err := s.Register(testPublicKey(b), name, event.ActorTypeHuman)
	if err != nil {
		t.Fatalf("register actor: %v", err)
	}
	return a
}

func testTrustEvent(actorID types.ActorID, prev, curr float64) event.Event {
	content := event.TrustUpdatedContent{
		Actor:    actorID,
		Previous: types.MustScore(prev),
		Current:  types.MustScore(curr),
		Domain:   types.MustDomainScope("general"),
		Cause:    types.MustEventID("019462a0-0000-7000-8000-000000000001"),
	}
	sig := make([]byte, 64)
	// Each call generates a unique EventID so deduplication doesn't suppress distinct evidence.
	evID, _ := types.NewEventIDFromNew()
	return event.NewEvent(
		1,
		evID,
		event.EventTypeTrustUpdated,
		types.Now(),
		actorID,
		content,
		[]types.EventID{types.MustEventID("019462a0-0000-7000-8000-000000000001")},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(),
		types.ZeroHash(),
		types.MustSignature(sig),
	)
}

func TestInitialTrustIsZero(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	metrics, err := model.Score(context.Background(), a)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if metrics.Overall().Value() != 0.0 {
		t.Errorf("initial trust = %v, want 0.0", metrics.Overall().Value())
	}
}

func TestUpdateIncreaseTrust(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	ev := testTrustEvent(a.ID(), 0.0, 0.05)
	metrics, err := model.Update(context.Background(), a, ev)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if metrics.Overall().Value() <= 0.0 {
		t.Errorf("trust should have increased, got %v", metrics.Overall().Value())
	}
}

func TestUpdateDecreaseTrust(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	// First increase
	ev1 := testTrustEvent(a.ID(), 0.0, 0.08)
	model.Update(context.Background(), a, ev1)

	// Then decrease
	ev2 := testTrustEvent(a.ID(), 0.08, 0.0)
	metrics, err := model.Update(context.Background(), a, ev2)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	// Should be lower after negative update
	if metrics.Overall().Value() >= 0.08 {
		t.Errorf("trust should have decreased, got %v", metrics.Overall().Value())
	}
}

func TestMaxAdjustmentClamp(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	// Try to increase by 0.5 — should be clamped to MaxAdjustment (0.1)
	ev := testTrustEvent(a.ID(), 0.0, 0.5)
	metrics, _ := model.Update(context.Background(), a, ev)
	if metrics.Overall().Value() > 0.1 {
		t.Errorf("trust should be clamped to 0.1, got %v", metrics.Overall().Value())
	}
}

func TestTrustNeverExceedsBounds(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	// Many positive updates
	for i := 0; i < 20; i++ {
		ev := testTrustEvent(a.ID(), 0.0, 0.1)
		model.Update(context.Background(), a, ev)
	}

	metrics, _ := model.Score(context.Background(), a)
	if metrics.Overall().Value() > 1.0 {
		t.Errorf("trust exceeded 1.0: %v", metrics.Overall().Value())
	}
	if metrics.Overall().Value() < 0.0 {
		t.Errorf("trust below 0.0: %v", metrics.Overall().Value())
	}
}

func TestDecay(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	// Build up some trust
	for i := 0; i < 5; i++ {
		ev := testTrustEvent(a.ID(), 0.0, 0.1)
		model.Update(context.Background(), a, ev)
	}

	before, _ := model.Score(context.Background(), a)

	// Decay 10 days
	model.Decay(context.Background(), a, 10*24*time.Hour)

	after, _ := model.Score(context.Background(), a)
	if after.Overall().Value() >= before.Overall().Value() {
		t.Errorf("trust should decrease after decay: before=%v, after=%v",
			before.Overall().Value(), after.Overall().Value())
	}
}

func TestDecayNeverBelowZero(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	// Decay 1000 days on a fresh actor (trust = 0)
	metrics, _ := model.Decay(context.Background(), a, 1000*24*time.Hour)
	if metrics.Overall().Value() < 0.0 {
		t.Errorf("trust should not go below 0: %v", metrics.Overall().Value())
	}
}

func TestBetweenNoRelationship(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	from := testActor(t, "Alice", 1)
	to := testActor(t, "Bob", 2)

	metrics, err := model.Between(context.Background(), from, to)
	if err != nil {
		t.Fatalf("Between: %v", err)
	}
	if metrics.Overall().Value() != 0.0 {
		t.Errorf("initial between trust = %v, want 0.0", metrics.Overall().Value())
	}
	if metrics.Confidence().Value() != 0.0 {
		t.Errorf("confidence should be 0 with no evidence, got %v", metrics.Confidence().Value())
	}
}

func TestScoreInDomain(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	metrics, err := model.ScoreInDomain(context.Background(), a, types.MustDomainScope("code_review"))
	if err != nil {
		t.Fatalf("ScoreInDomain: %v", err)
	}
	if metrics.Overall().Value() != 0.0 {
		t.Errorf("initial domain trust = %v, want 0.0", metrics.Overall().Value())
	}
}

func TestConfidenceGrows(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	// No evidence -> confidence near 0
	m1, _ := model.Score(context.Background(), a)
	if m1.Confidence().Value() > 0.1 {
		t.Errorf("initial confidence too high: %v", m1.Confidence().Value())
	}

	// Add evidence
	for i := 0; i < 10; i++ {
		ev := testTrustEvent(a.ID(), 0.0, 0.05)
		model.Update(context.Background(), a, ev)
	}

	m2, _ := model.Score(context.Background(), a)
	if m2.Confidence().Value() <= m1.Confidence().Value() {
		t.Errorf("confidence should grow with evidence: before=%v, after=%v",
			m1.Confidence().Value(), m2.Confidence().Value())
	}
}

func TestTrendPositive(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	ev := testTrustEvent(a.ID(), 0.0, 0.05)
	metrics, _ := model.Update(context.Background(), a, ev)
	if metrics.Trend().Value() <= 0 {
		t.Errorf("trend should be positive after positive update: %v", metrics.Trend().Value())
	}
}

func TestTrendNegative(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	// First increase to have something to decrease from
	ev1 := testTrustEvent(a.ID(), 0.0, 0.05)
	model.Update(context.Background(), a, ev1)

	// Then decrease
	ev2 := testTrustEvent(a.ID(), 0.1, 0.0)
	metrics, _ := model.Update(context.Background(), a, ev2)
	if metrics.Trend().Value() >= 0.1 {
		t.Errorf("trend should decrease with negative update: %v", metrics.Trend().Value())
	}
}

func TestCustomConfig(t *testing.T) {
	config := trust.DefaultConfig{
		InitialTrust:  types.MustScore(0.5),
		DecayRate:     types.MustScore(0.05),
		MaxAdjustment: types.MustWeight(0.2),
	}
	model := trust.NewDefaultTrustModelWithConfig(config)
	a := testActor(t, "Alice", 1)

	metrics, _ := model.Score(context.Background(), a)
	if metrics.Overall().Value() != 0.5 {
		t.Errorf("initial trust = %v, want 0.5", metrics.Overall().Value())
	}
}

func TestNonTrustEventGivesSmallPositive(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	// An event that is not TrustUpdatedContent
	sig := make([]byte, 64)
	ev := event.NewBootstrapEvent(
		1,
		types.MustEventID("019462a0-0000-7000-8000-000000000003"),
		event.EventTypeSystemBootstrapped,
		types.Now(),
		a.ID(),
		event.BootstrapContent{
			ActorID:      a.ID(),
			ChainGenesis: types.ZeroHash(),
			Timestamp:    types.Now(),
		},
		types.MustConversationID("conv_test000000000000000000000001"),
		types.ZeroHash(),
		types.MustSignature(sig),
	)

	metrics, err := model.Update(context.Background(), a, ev)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if metrics.Overall().Value() <= 0.0 {
		t.Errorf("non-trust event should give small positive, got %v", metrics.Overall().Value())
	}
}

func TestEvidenceHistoryTrimmed(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	// Add 150 updates
	for i := 0; i < 150; i++ {
		ev := testTrustEvent(a.ID(), 0.0, 0.01)
		model.Update(context.Background(), a, ev)
	}

	metrics, _ := model.Score(context.Background(), a)
	if len(metrics.Evidence()) > 100 {
		t.Errorf("evidence should be capped at 100, got %d", len(metrics.Evidence()))
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)
	b := testActor(t, "Bob", 2)

	// Build up some trust state.
	ev1 := testTrustEvent(a.ID(), 0.0, 0.5)
	model.Update(context.Background(), a, ev1)

	ev2 := testTrustEvent(b.ID(), 0.0, 0.3)
	model.Update(context.Background(), b, ev2)

	ev3 := testTrustEvent(b.ID(), 0.0, 0.8)
	model.UpdateBetween(context.Background(), a, b, ev3)

	// Export.
	data, err := model.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}

	// Import into a fresh model.
	model2 := trust.NewDefaultTrustModel()
	if err := model2.ImportJSON(data); err != nil {
		t.Fatalf("ImportJSON: %v", err)
	}

	// Verify Alice's score survived.
	metrics1, _ := model.Score(context.Background(), a)
	metrics2, _ := model2.Score(context.Background(), a)
	if metrics1.Overall().Value() != metrics2.Overall().Value() {
		t.Errorf("Alice score mismatch: original=%v restored=%v",
			metrics1.Overall().Value(), metrics2.Overall().Value())
	}

	// Verify Bob's score survived.
	metricsB1, _ := model.Score(context.Background(), b)
	metricsB2, _ := model2.Score(context.Background(), b)
	if metricsB1.Overall().Value() != metricsB2.Overall().Value() {
		t.Errorf("Bob score mismatch: original=%v restored=%v",
			metricsB1.Overall().Value(), metricsB2.Overall().Value())
	}

	// Verify directed trust (A→B) survived.
	between1, _ := model.Between(context.Background(), a, b)
	between2, _ := model2.Between(context.Background(), a, b)
	if between1.Overall().Value() != between2.Overall().Value() {
		t.Errorf("A→B trust mismatch: original=%v restored=%v",
			between1.Overall().Value(), between2.Overall().Value())
	}
}

func TestExportImportPreservesDomainScores(t *testing.T) {
	model := trust.NewDefaultTrustModel()
	a := testActor(t, "Alice", 1)

	ev := testTrustEvent(a.ID(), 0.0, 0.6)
	model.Update(context.Background(), a, ev)

	data, _ := model.ExportJSON()
	model2 := trust.NewDefaultTrustModel()
	model2.ImportJSON(data)

	metrics, _ := model2.ScoreInDomain(context.Background(), a, types.MustDomainScope("general"))
	if metrics.Overall().Value() <= 0.0 {
		t.Error("domain score should survive export/import")
	}
}
