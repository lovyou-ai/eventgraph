package decision

import (
	"sync"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// DecisionNode is a node in a decision tree — either InternalNode or LeafNode.
type DecisionNode interface {
	isDecisionNode()
}

// InternalNode branches on a condition.
type InternalNode struct {
	Condition event.Condition
	Branches  []Branch // at least one
	Default   DecisionNode
}

func (*InternalNode) isDecisionNode() {}

// Branch maps a match value to a child node.
type Branch struct {
	Match event.MatchValue
	Child DecisionNode
}

// LeafNode is a terminal node — either deterministic or needs intelligence.
type LeafNode struct {
	Outcome    types.Option[event.DecisionOutcome]
	NeedsLLM   bool
	Confidence types.Score
	mu         sync.Mutex
	Stats      LeafStats
}

func (*LeafNode) isDecisionNode() {}

// LeafStats tracks leaf usage for evolution.
type LeafStats struct {
	HitCount        int
	LLMCallCount    int
	ResponseHistory []ResponseRecord
	PatternScore    types.Score
}

// ResponseRecord records a single LLM response for pattern detection.
type ResponseRecord struct {
	Output     event.DecisionOutcome
	Confidence types.Score
}

// TreeStats tracks overall tree usage.
type TreeStats struct {
	TotalHits      int
	MechanicalHits int
	LLMHits        int
	TotalTokens    int
}

// DecisionTree is the root structure for primitive decision making.
type DecisionTree struct {
	Root    DecisionNode
	Version int
	mu      sync.RWMutex
	statsMu sync.Mutex // protects Stats independently from tree structure
	Stats   TreeStats
}

// NewDecisionTree creates a new DecisionTree with the given root.
func NewDecisionTree(root DecisionNode) *DecisionTree {
	return &DecisionTree{
		Root:    root,
		Version: 1,
	}
}

// NewLeaf creates a deterministic leaf node.
func NewLeaf(outcome event.DecisionOutcome, confidence types.Score) *LeafNode {
	return &LeafNode{
		Outcome:    types.Some(outcome),
		NeedsLLM:   false,
		Confidence: confidence,
	}
}

// NewLLMLeaf creates a leaf node that requires intelligence.
func NewLLMLeaf(confidence types.Score) *LeafNode {
	return &LeafNode{
		Outcome:  types.None[event.DecisionOutcome](),
		NeedsLLM: true,
		Confidence: confidence,
	}
}
