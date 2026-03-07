package decision_test

import (
	"context"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/decision"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

func makeHistory(outcome event.DecisionOutcome, confidence float64, count int) []decision.ResponseRecord {
	records := make([]decision.ResponseRecord, count)
	for i := range records {
		records[i] = decision.ResponseRecord{
			Output:     outcome,
			Confidence: types.MustScore(confidence),
		}
	}
	return records
}

func TestDetectPatternInsufficientSamples(t *testing.T) {
	stats := decision.LeafStats{
		ResponseHistory: makeHistory(event.DecisionOutcomePermit, 0.9, 5),
	}
	config := decision.DefaultEvolutionConfig() // MinSamples=10

	result := decision.DetectPattern(stats, config)
	if result.Detected {
		t.Error("should not detect pattern with insufficient samples")
	}
	if result.SampleCount != 5 {
		t.Errorf("SampleCount = %d, want 5", result.SampleCount)
	}
}

func TestDetectPatternClearDominant(t *testing.T) {
	stats := decision.LeafStats{
		ResponseHistory: makeHistory(event.DecisionOutcomePermit, 0.9, 10),
	}
	config := decision.DefaultEvolutionConfig()

	result := decision.DetectPattern(stats, config)
	if !result.Detected {
		t.Fatal("should detect clear pattern (100% same outcome)")
	}
	if result.DominantOutput != event.DecisionOutcomePermit {
		t.Errorf("DominantOutput = %v, want Permit", result.DominantOutput)
	}
	if result.Frequency != 1.0 {
		t.Errorf("Frequency = %v, want 1.0", result.Frequency)
	}
	if result.AvgConfidence < 0.89 || result.AvgConfidence > 0.91 {
		t.Errorf("AvgConfidence = %v, want ~0.9", result.AvgConfidence)
	}
}

func TestDetectPatternMixedBelowThreshold(t *testing.T) {
	history := make([]decision.ResponseRecord, 10)
	for i := 0; i < 6; i++ {
		history[i] = decision.ResponseRecord{Output: event.DecisionOutcomePermit, Confidence: types.MustScore(0.9)}
	}
	for i := 6; i < 10; i++ {
		history[i] = decision.ResponseRecord{Output: event.DecisionOutcomeDeny, Confidence: types.MustScore(0.9)}
	}

	stats := decision.LeafStats{ResponseHistory: history}
	config := decision.DefaultEvolutionConfig() // threshold 0.8

	result := decision.DetectPattern(stats, config)
	if result.Detected {
		t.Error("should not detect pattern at 60% (below 80% threshold)")
	}
	if result.Frequency != 0.6 {
		t.Errorf("Frequency = %v, want 0.6", result.Frequency)
	}
}

func TestDetectPatternLowConfidence(t *testing.T) {
	stats := decision.LeafStats{
		ResponseHistory: makeHistory(event.DecisionOutcomePermit, 0.5, 10),
	}
	config := decision.DefaultEvolutionConfig() // MinConfidence=0.7

	result := decision.DetectPattern(stats, config)
	if result.Detected {
		t.Error("should not detect pattern with low average confidence (0.5 < 0.7)")
	}
}

func TestDetectPatternAboveThreshold(t *testing.T) {
	history := make([]decision.ResponseRecord, 10)
	for i := 0; i < 9; i++ {
		history[i] = decision.ResponseRecord{Output: event.DecisionOutcomeDeny, Confidence: types.MustScore(0.85)}
	}
	history[9] = decision.ResponseRecord{Output: event.DecisionOutcomePermit, Confidence: types.MustScore(0.8)}

	stats := decision.LeafStats{ResponseHistory: history}
	config := decision.DefaultEvolutionConfig()

	result := decision.DetectPattern(stats, config)
	if !result.Detected {
		t.Fatal("should detect pattern at 90% (above 80% threshold)")
	}
	if result.DominantOutput != event.DecisionOutcomeDeny {
		t.Errorf("DominantOutput = %v, want Deny", result.DominantOutput)
	}
}

func TestExtractBranch(t *testing.T) {
	pattern := decision.PatternResult{
		Detected:       true,
		DominantOutput: event.DecisionOutcomePermit,
		AvgConfidence:  0.92,
	}

	leaf := decision.ExtractBranch(pattern)
	if leaf.NeedsLLM {
		t.Error("extracted branch should not need LLM")
	}
	if !leaf.Outcome.IsSome() {
		t.Fatal("outcome should be set")
	}
	if leaf.Outcome.Unwrap() != event.DecisionOutcomePermit {
		t.Errorf("Outcome = %v, want Permit", leaf.Outcome.Unwrap())
	}
	if leaf.Confidence.Value() != 0.92 {
		t.Errorf("Confidence = %v, want 0.92", leaf.Confidence.Value())
	}
}

func TestEvolveSimpleLLMLeaf(t *testing.T) {
	leaf := decision.NewLLMLeaf(types.MustScore(0.5))
	leaf.Stats.ResponseHistory = makeHistory(event.DecisionOutcomePermit, 0.9, 12)

	tree := decision.NewDecisionTree(leaf)
	config := decision.DefaultEvolutionConfig()

	result := decision.Evolve(tree, config)
	if !result.Evolved {
		t.Fatal("tree should have evolved")
	}
	if result.NewVersion != 2 {
		t.Errorf("NewVersion = %d, want 2", result.NewVersion)
	}
	if result.Pattern.DominantOutput != event.DecisionOutcomePermit {
		t.Errorf("DominantOutput = %v, want Permit", result.Pattern.DominantOutput)
	}

	// Tree should now evaluate mechanically
	treeResult, err := decision.Evaluate(context.Background(), tree, testInput("test"), types.None[decision.IIntelligence]())
	if err != nil {
		t.Fatalf("Evaluate after evolution: %v", err)
	}
	if treeResult.Outcome != event.DecisionOutcomePermit {
		t.Errorf("post-evolution Outcome = %v, want Permit", treeResult.Outcome)
	}
	if treeResult.UsedLLM {
		t.Error("post-evolution should not use LLM")
	}
}

func TestEvolveNoEvolutionNeeded(t *testing.T) {
	// Mechanical leaf — already deterministic
	leaf := decision.NewLeaf(event.DecisionOutcomePermit, types.MustScore(0.9))
	tree := decision.NewDecisionTree(leaf)

	result := decision.Evolve(tree, decision.DefaultEvolutionConfig())
	if result.Evolved {
		t.Error("mechanical leaf should not evolve")
	}
	if tree.Version != 1 {
		t.Errorf("Version = %d, want 1 (unchanged)", tree.Version)
	}
}

func TestEvolveInsufficientHistory(t *testing.T) {
	leaf := decision.NewLLMLeaf(types.MustScore(0.5))
	leaf.Stats.ResponseHistory = makeHistory(event.DecisionOutcomePermit, 0.9, 3)

	tree := decision.NewDecisionTree(leaf)
	result := decision.Evolve(tree, decision.DefaultEvolutionConfig())
	if result.Evolved {
		t.Error("should not evolve with insufficient history")
	}
}

func TestEvolveNestedTree(t *testing.T) {
	llmLeaf := decision.NewLLMLeaf(types.MustScore(0.5))
	llmLeaf.Stats.ResponseHistory = makeHistory(event.DecisionOutcomeDeny, 0.85, 15)

	tree := decision.NewDecisionTree(&decision.InternalNode{
		Condition: event.Condition{
			Field:    types.MustFieldPath("action"),
			Operator: event.ConditionOperatorEquals,
		},
		Branches: []decision.Branch{
			{
				Match: event.MatchValue{String: types.Some("deploy")},
				Child: llmLeaf,
			},
		},
		Default: decision.NewLeaf(event.DecisionOutcomePermit, types.MustScore(0.9)),
	})

	result := decision.Evolve(tree, decision.DefaultEvolutionConfig())
	if !result.Evolved {
		t.Fatal("nested LLM leaf should have evolved")
	}
	if result.Pattern.DominantOutput != event.DecisionOutcomeDeny {
		t.Errorf("DominantOutput = %v, want Deny", result.Pattern.DominantOutput)
	}

	// Verify the evolved node works mechanically
	treeResult, err := decision.Evaluate(context.Background(), tree, testInput("deploy"), types.None[decision.IIntelligence]())
	if err != nil {
		t.Fatalf("Evaluate after evolution: %v", err)
	}
	if treeResult.Outcome != event.DecisionOutcomeDeny {
		t.Errorf("post-evolution Outcome = %v, want Deny", treeResult.Outcome)
	}
}

func TestEvolveDefaultBranch(t *testing.T) {
	llmLeaf := decision.NewLLMLeaf(types.MustScore(0.5))
	llmLeaf.Stats.ResponseHistory = makeHistory(event.DecisionOutcomeEscalate, 0.8, 10)

	tree := decision.NewDecisionTree(&decision.InternalNode{
		Condition: event.Condition{
			Field:    types.MustFieldPath("action"),
			Operator: event.ConditionOperatorEquals,
		},
		Branches: []decision.Branch{
			{
				Match: event.MatchValue{String: types.Some("deploy")},
				Child: decision.NewLeaf(event.DecisionOutcomePermit, types.MustScore(0.9)),
			},
		},
		Default: llmLeaf,
	})

	result := decision.Evolve(tree, decision.DefaultEvolutionConfig())
	if !result.Evolved {
		t.Fatal("default LLM leaf should have evolved")
	}
	if result.Pattern.DominantOutput != event.DecisionOutcomeEscalate {
		t.Errorf("DominantOutput = %v, want Escalate", result.Pattern.DominantOutput)
	}
}

func TestEvolveNilRoot(t *testing.T) {
	tree := &decision.DecisionTree{Root: nil, Version: 1}
	result := decision.Evolve(tree, decision.DefaultEvolutionConfig())
	if result.Evolved {
		t.Error("nil root should not evolve")
	}
}

func TestEvolveCostReduction(t *testing.T) {
	leaf := decision.NewLLMLeaf(types.MustScore(0.5))
	leaf.Stats.ResponseHistory = makeHistory(event.DecisionOutcomePermit, 0.9, 10)

	tree := decision.NewDecisionTree(leaf)
	tree.Stats.LLMHits = 10 // simulate prior LLM usage

	result := decision.Evolve(tree, decision.DefaultEvolutionConfig())
	if !result.Evolved {
		t.Fatal("should have evolved")
	}
	if result.CostReduction != 1.0 {
		t.Errorf("CostReduction = %v, want 1.0 (100%% same outcome)", result.CostReduction)
	}
}

func TestDefaultEvolutionConfig(t *testing.T) {
	config := decision.DefaultEvolutionConfig()
	if config.MinSamples != 10 {
		t.Errorf("MinSamples = %d, want 10", config.MinSamples)
	}
	if config.PatternThreshold != 0.8 {
		t.Errorf("PatternThreshold = %v, want 0.8", config.PatternThreshold)
	}
	if config.MinConfidence != 0.7 {
		t.Errorf("MinConfidence = %v, want 0.7", config.MinConfidence)
	}
}

func TestDetectPatternEmptyHistory(t *testing.T) {
	stats := decision.LeafStats{}
	config := decision.DefaultEvolutionConfig()

	result := decision.DetectPattern(stats, config)
	if result.Detected {
		t.Error("empty history should not detect pattern")
	}
	if result.SampleCount != 0 {
		t.Errorf("SampleCount = %d, want 0", result.SampleCount)
	}
}
