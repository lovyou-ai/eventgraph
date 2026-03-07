package decision

import (
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// EvolutionConfig controls when and how decision tree evolution occurs.
type EvolutionConfig struct {
	// MinSamples is the minimum response history size before pattern detection runs.
	MinSamples int
	// PatternThreshold is the minimum fraction of identical outcomes to consider a pattern (e.g. 0.8 = 80%).
	PatternThreshold float64
	// MinConfidence is the minimum average confidence of the dominant outcome to extract a branch.
	MinConfidence float64
}

// DefaultEvolutionConfig returns sensible defaults for tree evolution.
func DefaultEvolutionConfig() EvolutionConfig {
	return EvolutionConfig{
		MinSamples:       10,
		PatternThreshold: 0.8,
		MinConfidence:    0.7,
	}
}

// PatternResult describes a detected pattern in a leaf's response history.
type PatternResult struct {
	Detected       bool
	DominantOutput event.DecisionOutcome
	Frequency      float64
	AvgConfidence  float64
	SampleCount    int
}

// EvolutionResult describes what happened when evolution was attempted.
type EvolutionResult struct {
	Evolved       bool
	Pattern       PatternResult
	CostReduction float64 // estimated fraction of LLM calls saved
	NewVersion    int
}

// DetectPattern analyzes a leaf's response history for a dominant outcome.
func DetectPattern(stats LeafStats, config EvolutionConfig) PatternResult {
	if len(stats.ResponseHistory) < config.MinSamples {
		return PatternResult{SampleCount: len(stats.ResponseHistory)}
	}

	counts := make(map[event.DecisionOutcome]int)
	confidenceSum := make(map[event.DecisionOutcome]float64)

	for _, r := range stats.ResponseHistory {
		counts[r.Output]++
		confidenceSum[r.Output] += r.Confidence.Value()
	}

	total := len(stats.ResponseHistory)
	var dominant event.DecisionOutcome
	var maxCount int

	for outcome, count := range counts {
		if count > maxCount {
			maxCount = count
			dominant = outcome
		}
	}

	freq := float64(maxCount) / float64(total)
	avgConf := confidenceSum[dominant] / float64(maxCount)

	detected := freq >= config.PatternThreshold && avgConf >= config.MinConfidence

	return PatternResult{
		Detected:       detected,
		DominantOutput: dominant,
		Frequency:      freq,
		AvgConfidence:  avgConf,
		SampleCount:    total,
	}
}

// ExtractBranch converts a detected pattern into a mechanical leaf node,
// replacing the LLM leaf with a deterministic one.
func ExtractBranch(pattern PatternResult) *LeafNode {
	confidence := types.MustScore(clamp(pattern.AvgConfidence, 0.0, 1.0))
	return NewLeaf(pattern.DominantOutput, confidence)
}

// Evolve analyzes the tree for LLM leaves with detectable patterns and
// replaces them with mechanical branches. Returns the evolution result.
// Evolves at most one leaf per call — callers should loop until !result.Evolved
// to evolve all eligible leaves.
// The tree is mutated in place — the caller should persist after evolution.
func Evolve(tree *DecisionTree, config EvolutionConfig) EvolutionResult {
	tree.mu.Lock()
	defer tree.mu.Unlock()

	if tree.Root == nil {
		return EvolutionResult{}
	}

	evolved := evolveNode(&tree.Root, config)
	if evolved.Evolved {
		tree.Version++
		evolved.NewVersion = tree.Version
		if tree.Stats.LLMHits > 0 {
			evolved.CostReduction = evolved.Pattern.Frequency
		}
	}
	return evolved
}

func evolveNode(node *DecisionNode, config EvolutionConfig) EvolutionResult {
	switch n := (*node).(type) {
	case *InternalNode:
		// Try evolving branches first
		for i := range n.Branches {
			result := evolveNode(&n.Branches[i].Child, config)
			if result.Evolved {
				return result
			}
		}
		// Try evolving default
		if n.Default != nil {
			return evolveNode(&n.Default, config)
		}
		return EvolutionResult{}

	case *LeafNode:
		if !n.NeedsLLM {
			return EvolutionResult{}
		}
		// Copy stats under lock to avoid racing with evaluateLeaf
		n.mu.Lock()
		statsCopy := n.Stats
		statsCopy.ResponseHistory = make([]ResponseRecord, len(n.Stats.ResponseHistory))
		copy(statsCopy.ResponseHistory, n.Stats.ResponseHistory)
		n.mu.Unlock()

		pattern := DetectPattern(statsCopy, config)
		if !pattern.Detected {
			return EvolutionResult{}
		}
		*node = ExtractBranch(pattern)
		return EvolutionResult{
			Evolved: true,
			Pattern: pattern,
		}

	default:
		return EvolutionResult{}
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
