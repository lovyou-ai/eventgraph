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
func Evaluate(ctx context.Context, tree *DecisionTree, input EvaluateInput, intelligence types.Option[IIntelligence]) (TreeResult, error) {
	tree.mu.RLock()
	defer tree.mu.RUnlock()

	var path []event.PathStep
	node := tree.Root

	for {
		switch n := node.(type) {
		case *InternalNode:
			next, step, err := evaluateInternal(ctx, n, input, intelligence)
			if err != nil {
				return TreeResult{}, err
			}
			path = append(path, step)
			node = next

		case *LeafNode:
			return evaluateLeaf(ctx, n, input, path, tree, intelligence)

		default:
			return TreeResult{}, fmt.Errorf("unknown decision node type: %T", node)
		}
	}
}

func evaluateInternal(ctx context.Context, n *InternalNode, input EvaluateInput, intelligence types.Option[IIntelligence]) (DecisionNode, event.PathStep, error) {
	if n.Condition.Operator == event.ConditionOperatorSemantic {
		return evaluateSemantic(ctx, n, input, intelligence)
	}
	return evaluateMechanical(n, input)
}

func evaluateMechanical(n *InternalNode, input EvaluateInput) (DecisionNode, event.PathStep, error) {
	value := extractField(input, n.Condition.Field)

	for _, branch := range n.Branches {
		if testCondition(value, n.Condition.Operator, branch.Match) {
			step := event.PathStep{Condition: n.Condition, Branch: branch.Match}
			return branch.Child, step, nil
		}
	}

	// No branch matched — take default
	step := event.PathStep{
		Condition: n.Condition,
		Branch:    event.MatchValue{String: types.Some("default")},
	}
	return n.Default, step, nil
}

func evaluateSemantic(ctx context.Context, n *InternalNode, input EvaluateInput, intelligence types.Option[IIntelligence]) (DecisionNode, event.PathStep, error) {
	if !intelligence.IsSome() {
		step := event.PathStep{
			Condition: n.Condition,
			Branch:    event.MatchValue{String: types.Some("default")},
		}
		return n.Default, step, nil
	}

	intel := intelligence.Unwrap()
	prompt := ""
	if n.Condition.Prompt.IsSome() {
		prompt = n.Condition.Prompt.Unwrap()
	}

	resp, err := intel.Reason(ctx, prompt, nil)
	if err != nil {
		// Intelligence failed — fall through to default
		step := event.PathStep{
			Condition: n.Condition,
			Branch:    event.MatchValue{String: types.Some("default")},
		}
		return n.Default, step, nil
	}

	// Check if response confidence meets threshold.
	// Current limitation: semantic nodes are binary (confident → branch[0], else → default).
	// Future: match resp.Content() against branch match values for multi-way semantic routing.
	if n.Condition.Threshold.IsSome() {
		threshold := n.Condition.Threshold.Unwrap()
		if resp.Confidence().Value() >= threshold.Value() && len(n.Branches) > 0 {
			branch := n.Branches[0]
			step := event.PathStep{Condition: n.Condition, Branch: branch.Match}
			return branch.Child, step, nil
		}
	}

	step := event.PathStep{
		Condition: n.Condition,
		Branch:    event.MatchValue{String: types.Some("default")},
	}
	return n.Default, step, nil
}

func evaluateLeaf(ctx context.Context, leaf *LeafNode, input EvaluateInput, path []event.PathStep, tree *DecisionTree, intelligence types.Option[IIntelligence]) (TreeResult, error) {
	leaf.mu.Lock()
	leaf.Stats.HitCount++
	leaf.mu.Unlock()

	tree.statsMu.Lock()
	tree.Stats.TotalHits++
	tree.statsMu.Unlock()

	if !leaf.NeedsLLM {
		tree.statsMu.Lock()
		tree.Stats.MechanicalHits++
		tree.statsMu.Unlock()

		if !leaf.Outcome.IsSome() {
			return TreeResult{}, fmt.Errorf("mechanical leaf has no outcome (NeedsLLM=false but Outcome is None)")
		}
		outcome := leaf.Outcome.Unwrap()
		return TreeResult{
			Outcome:    outcome,
			Confidence: leaf.Confidence,
			Path:       path,
			UsedLLM:    false,
		}, nil
	}

	// Needs LLM
	if !intelligence.IsSome() {
		return TreeResult{}, &InsufficientAuthorityError{
			Actor:    input.Actor,
			Action:   input.Action,
			Required: event.AuthorityLevelRequired,
		}
	}

	tree.statsMu.Lock()
	tree.Stats.LLMHits++
	tree.statsMu.Unlock()

	leaf.mu.Lock()
	leaf.Stats.LLMCallCount++
	leaf.mu.Unlock()

	intel := intelligence.Unwrap()
	prompt := formatPrompt(input, path)
	resp, err := intel.Reason(ctx, prompt, input.History)
	if err != nil {
		return TreeResult{}, err
	}

	tree.statsMu.Lock()
	tree.Stats.TotalTokens += resp.TokensUsed()
	tree.statsMu.Unlock()

	outcome := parseOutcome(resp.Content())

	leaf.mu.Lock()
	leaf.Stats.ResponseHistory = append(leaf.Stats.ResponseHistory, ResponseRecord{
		Output:     outcome,
		Confidence: resp.Confidence(),
	})
	if len(leaf.Stats.ResponseHistory) > maxResponseHistory {
		leaf.Stats.ResponseHistory = leaf.Stats.ResponseHistory[len(leaf.Stats.ResponseHistory)-maxResponseHistory:]
	}
	leaf.mu.Unlock()

	return TreeResult{
		Outcome:    outcome,
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
func testCondition(value any, op event.ConditionOperator, match event.MatchValue) bool {
	switch op {
	case event.ConditionOperatorEquals:
		return equalsMatch(value, match)
	case event.ConditionOperatorGreaterThan:
		return numericCompare(value, match, func(a, b float64) bool { return a > b })
	case event.ConditionOperatorLessThan:
		return numericCompare(value, match, func(a, b float64) bool { return a < b })
	case event.ConditionOperatorExists:
		exists := value != nil
		// If Match.Boolean is set, respect it: false means "does not exist"
		if match.Boolean.IsSome() {
			return exists == match.Boolean.Unwrap()
		}
		return exists
	case event.ConditionOperatorMatches:
		return patternMatch(value, match)
	case event.ConditionOperatorSemantic:
		// Semantic operators are handled by evaluateSemantic before reaching here.
		return false
	}
	// No default — exhaustive matching per CLAUDE.md.
	// New ConditionOperator variants must be handled explicitly above.
	panic(fmt.Sprintf("unhandled ConditionOperator: %s", op))
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

func parseOutcome(content string) event.DecisionOutcome {
	lower := strings.ToLower(strings.TrimSpace(content))
	switch {
	case strings.Contains(lower, "permit"):
		return event.DecisionOutcomePermit
	case strings.Contains(lower, "deny"):
		return event.DecisionOutcomeDeny
	case strings.Contains(lower, "escalate"):
		return event.DecisionOutcomeEscalate
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
