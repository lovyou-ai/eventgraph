package decision

import (
	"context"
	"fmt"
	"strings"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// maxResponseHistory caps the ResponseHistory slice on each leaf to prevent unbounded growth.
const maxResponseHistory = 200

// TreeResult is the output of tree evaluation — a simpler structure than Decision.
// The caller constructs a full Decision from this plus authority chain, receipt, etc.
type TreeResult struct {
	Outcome    event.DecisionOutcome
	Confidence types.Score
	Path       []event.PathStep
	UsedLLM    bool
}

// EvaluateInput is the input to tree evaluation — simpler than DecisionInput.
type EvaluateInput struct {
	Action  string
	Actor   types.ActorID
	Context map[string]any
	History []event.Event
}

// Evaluate walks the decision tree with the given input and optional intelligence.
// Returns a TreeResult with the outcome, confidence, and the path taken.
// The tree read lock is held during traversal but released before any LLM I/O
// to prevent blocking Evolve and other concurrent Evaluate callers.
func Evaluate(ctx context.Context, tree *DecisionTree, input EvaluateInput, intelligence types.Option[IIntelligence]) (TreeResult, error) {
	tree.mu.RLock()

	var path []event.PathStep
	node := tree.Root

	for {
		switch n := node.(type) {
		case *InternalNode:
			if n.Condition.Operator == event.ConditionOperatorSemantic {
				// Snapshot node data while lock is held, then release before LLM I/O.
				// evaluateSemantic may call intel.Reason which is unbounded I/O.
				// Concurrent Evolve can replace Branches[i].Child and Default via
				// evolveNode, so we must not read from n after releasing the lock.
				snap := semanticSnapshot{
					condition: n.Condition,
					branches:  make([]Branch, len(n.Branches)),
					dflt:      n.Default,
				}
				copy(snap.branches, n.Branches)
				tree.mu.RUnlock()
				next, step, err := evaluateSemanticFromSnapshot(ctx, snap, input, intelligence)
				if err != nil {
					return TreeResult{}, err
				}
				path = append(path, step)
				node = next
				// Re-acquire lock for next iteration's tree traversal
				tree.mu.RLock()
			} else {
				next, step, err := evaluateMechanical(n, input)
				if err != nil {
					tree.mu.RUnlock()
					return TreeResult{}, err
				}
				path = append(path, step)
				node = next
			}

		case *LeafNode:
			// Release tree lock before evaluateLeaf, which may perform
			// unbounded LLM I/O. Leaf-level mutations are protected by
			// leaf.mu and tree.statsMu independently.
			tree.mu.RUnlock()
			return evaluateLeaf(ctx, n, input, path, tree, intelligence)

		default:
			tree.mu.RUnlock()
			return TreeResult{}, fmt.Errorf("unknown decision node type: %T", node)
		}
	}
}

func evaluateMechanical(n *InternalNode, input EvaluateInput) (DecisionNode, event.PathStep, error) {
	value := extractField(input, n.Condition.Field)

	for _, branch := range n.Branches {
		matched, err := testCondition(value, n.Condition.Operator, branch.Match)
		if err != nil {
			return nil, event.PathStep{}, err
		}
		if matched {
			step := event.PathStep{Condition: n.Condition, Branch: branch.Match}
			return branch.Child, step, nil
		}
	}

	// No branch matched — take default
	if n.Default == nil {
		return nil, event.PathStep{}, fmt.Errorf("no branch matched and no default node set for condition on field %q", n.Condition.Field.Value())
	}
	step := event.PathStep{
		Condition: n.Condition,
		Branch:    event.MatchValue{String: types.Some("default")},
	}
	return n.Default, step, nil
}

// semanticSnapshot holds a copy of InternalNode fields needed by semantic
// evaluation. Captured while the tree read lock is held so that evaluateSemantic
// does not race with concurrent Evolve calls that replace child nodes.
type semanticSnapshot struct {
	condition event.Condition
	branches  []Branch
	dflt      DecisionNode
}

func evaluateSemanticFromSnapshot(ctx context.Context, snap semanticSnapshot, input EvaluateInput, intelligence types.Option[IIntelligence]) (DecisionNode, event.PathStep, error) {
	defaultStep := event.PathStep{
		Condition: snap.condition,
		Branch:    event.MatchValue{String: types.Some("default")},
	}

	returnDefault := func() (DecisionNode, event.PathStep, error) {
		if snap.dflt == nil {
			return nil, event.PathStep{}, fmt.Errorf("no branch matched and no default node set for semantic condition on field %q", snap.condition.Field.Value())
		}
		return snap.dflt, defaultStep, nil
	}

	if !intelligence.IsSome() {
		return returnDefault()
	}

	intel := intelligence.Unwrap()
	prompt := ""
	if snap.condition.Prompt.IsSome() {
		prompt = snap.condition.Prompt.Unwrap()
	}

	resp, err := intel.Reason(ctx, prompt, input.History)
	if err != nil {
		// Intelligence failed — fall through to default
		return returnDefault()
	}

	// Route to branch[0] if the LLM response meets the threshold (or no threshold is set).
	// Current limitation: semantic nodes are binary (confident → branch[0], else → default).
	// Future: match resp.Content() against branch match values for multi-way semantic routing.
	if len(snap.branches) > 0 {
		if !snap.condition.Threshold.IsSome() || resp.Confidence().Value() >= snap.condition.Threshold.Unwrap().Value() {
			branch := snap.branches[0]
			step := event.PathStep{Condition: snap.condition, Branch: branch.Match}
			return branch.Child, step, nil
		}
	}

	return returnDefault()
}

func evaluateLeaf(ctx context.Context, leaf *LeafNode, input EvaluateInput, path []event.PathStep, tree *DecisionTree, intelligence types.Option[IIntelligence]) (TreeResult, error) {
	// Capture immutable leaf fields under the lock to establish a happens-before
	// edge per the Go memory model. These fields are set at construction and never
	// modified, but concurrent leaf.mu writes (Stats) require synchronization.
	leaf.mu.Lock()
	leaf.Stats.HitCount++
	needsLLM := leaf.NeedsLLM
	outcome := leaf.Outcome
	confidence := leaf.Confidence
	leaf.mu.Unlock()

	if !needsLLM {
		tree.statsMu.Lock()
		tree.Stats.TotalHits++
		tree.Stats.MechanicalHits++
		tree.statsMu.Unlock()

		if !outcome.IsSome() {
			return TreeResult{}, fmt.Errorf("mechanical leaf has no outcome (NeedsLLM=false but Outcome is None)")
		}
		return TreeResult{
			Outcome:    outcome.Unwrap(),
			Confidence: confidence,
			Path:       path,
			UsedLLM:    false,
		}, nil
	}

	// Needs LLM
	tree.statsMu.Lock()
	tree.Stats.TotalHits++
	tree.Stats.LLMHits++
	tree.statsMu.Unlock()

	if !intelligence.IsSome() {
		return TreeResult{}, &IntelligenceUnavailableError{}
	}

	leaf.mu.Lock()
	leaf.Stats.LLMCallCount++
	leaf.mu.Unlock()

	intel := intelligence.Unwrap()
	prompt := formatPrompt(input, path)
	resp, err := intel.Reason(ctx, prompt, input.History)
	if err != nil {
		return TreeResult{}, err
	}

	llmOutcome := parseOutcome(resp.Content())

	tree.statsMu.Lock()
	tree.Stats.TotalTokens += resp.TokensUsed()
	tree.statsMu.Unlock()

	leaf.mu.Lock()
	leaf.Stats.ResponseHistory = append(leaf.Stats.ResponseHistory, ResponseRecord{
		Output:     llmOutcome,
		Confidence: resp.Confidence(),
	})
	if len(leaf.Stats.ResponseHistory) > maxResponseHistory {
		tail := make([]ResponseRecord, maxResponseHistory)
		copy(tail, leaf.Stats.ResponseHistory[len(leaf.Stats.ResponseHistory)-maxResponseHistory:])
		leaf.Stats.ResponseHistory = tail
	}
	leaf.mu.Unlock()

	return TreeResult{
		Outcome:    llmOutcome,
		Confidence: resp.Confidence(),
		Path:       path,
		UsedLLM:    true,
	}, nil
}

// extractField gets a value from the EvaluateInput context by dot-path.
func extractField(input EvaluateInput, field types.FieldPath) any {
	path := field.Value()

	switch {
	case path == "action":
		return input.Action
	case path == "actor":
		return input.Actor.Value()
	case strings.HasPrefix(path, "context."):
		key := strings.TrimPrefix(path, "context.")
		if input.Context != nil {
			return input.Context[key]
		}
		return nil
	default:
		if input.Context != nil {
			return input.Context[path]
		}
		return nil
	}
}

// testCondition evaluates a condition operator against a value and match.
func testCondition(value any, op event.ConditionOperator, match event.MatchValue) (bool, error) {
	switch op {
	case event.ConditionOperatorEquals:
		return equalsMatch(value, match), nil
	case event.ConditionOperatorGreaterThan:
		return numericCompare(value, match, func(a, b float64) bool { return a > b }), nil
	case event.ConditionOperatorLessThan:
		return numericCompare(value, match, func(a, b float64) bool { return a < b }), nil
	case event.ConditionOperatorExists:
		exists := value != nil
		// If Match.Boolean is set, respect it: false means "does not exist"
		if match.Boolean.IsSome() {
			return exists == match.Boolean.Unwrap(), nil
		}
		return exists, nil
	case event.ConditionOperatorMatches:
		return patternMatch(value, match), nil
	case event.ConditionOperatorSemantic:
		// Semantic operators are handled by evaluateSemantic before reaching here.
		return false, fmt.Errorf("ConditionOperatorSemantic must not reach testCondition; use evaluateSemantic")
	default:
		return false, fmt.Errorf("unhandled ConditionOperator: %s", op)
	}
}

func equalsMatch(value any, match event.MatchValue) bool {
	if match.String.IsSome() {
		s, ok := value.(string)
		return ok && s == match.String.Unwrap()
	}
	if match.Number.IsSome() {
		return toFloat64(value) == match.Number.Unwrap()
	}
	if match.Boolean.IsSome() {
		b, ok := value.(bool)
		return ok && b == match.Boolean.Unwrap()
	}
	if match.EventType.IsSome() {
		s, ok := value.(string)
		return ok && s == match.EventType.Unwrap().Value()
	}
	return false
}

func numericCompare(value any, match event.MatchValue, cmp func(float64, float64) bool) bool {
	if !match.Number.IsSome() {
		return false
	}
	return cmp(toFloat64(value), match.Number.Unwrap())
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func patternMatch(value any, match event.MatchValue) bool {
	if !match.String.IsSome() {
		return false
	}
	s, ok := value.(string)
	if !ok {
		return false
	}
	pattern := match.String.Unwrap()
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, strings.TrimSuffix(pattern, "*"))
	}
	return s == pattern
}

// parseOutcome extracts a decision outcome from LLM response text.
// Priority is fail-safe: deny > escalate > permit > defer.
// If multiple keywords appear in the response, the most restrictive wins.
func parseOutcome(content string) event.DecisionOutcome {
	lower := strings.ToLower(strings.TrimSpace(content))
	switch {
	case strings.Contains(lower, "deny"):
		return event.DecisionOutcomeDeny
	case strings.Contains(lower, "escalate"):
		return event.DecisionOutcomeEscalate
	case strings.Contains(lower, "permit"):
		return event.DecisionOutcomePermit
	default:
		return event.DecisionOutcomeDefer
	}
}

func formatPrompt(input EvaluateInput, path []event.PathStep) string {
	var b strings.Builder
	b.WriteString("Action: ")
	b.WriteString(input.Action)
	b.WriteString("\nActor: ")
	b.WriteString(input.Actor.Value())
	if len(path) > 0 {
		b.WriteString("\nPath taken: ")
		for i, step := range path {
			if i > 0 {
				b.WriteString(" -> ")
			}
			b.WriteString(step.Condition.Field.Value())
		}
	}
	return b.String()
}
