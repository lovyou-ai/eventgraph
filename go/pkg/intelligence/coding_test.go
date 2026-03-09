package intelligence_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/intelligence"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// These tests exercise real LLM intelligence for coding tasks through the AgentRuntime.
// Each test boots an agent, performs a coding task, and verifies the result lands on the graph.
// Gated on EVENTGRAPH_TEST_CLAUDE_CLI=1.
//
// Run: EVENTGRAPH_TEST_CLAUDE_CLI=1 go test ./pkg/intelligence/ -v -run TestCoding -count=1

func codingRuntime(t *testing.T) *intelligence.AgentRuntime {
	t.Helper()
	if os.Getenv("EVENTGRAPH_TEST_CLAUDE_CLI") == "" {
		t.Skip("EVENTGRAPH_TEST_CLAUDE_CLI not set")
	}
	p, err := intelligence.New(intelligence.Config{
		Provider: "claude-cli",
		Model:    "sonnet",
		SystemPrompt: `You are a coding agent. You write, review, and reason about code.
Be precise. When asked for code, return only code. When asked for analysis, be specific and concise.`,
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	agentID := types.MustActorID("actor_00000000000000000000000000000001")
	rt, err := intelligence.NewRuntime(context.Background(), intelligence.RuntimeConfig{
		AgentID:  agentID,
		Provider: p,
	})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	return rt
}

// ════════════════════════════════════════════════════════════════════════
// Code Review
// ════════════════════════════════════════════════════════════════════════

func TestCodingReviewFindsBug(t *testing.T) {
	rt := codingRuntime(t)
	ctx := context.Background()

	buggyCode := `func divide(a, b int) int {
    return a / b
}`

	ev, review, err := rt.CodeReview(ctx, buggyCode, "go")
	if err != nil {
		t.Fatalf("CodeReview failed: %v", err)
	}

	// The review should mention division by zero.
	lower := strings.ToLower(review)
	if !strings.Contains(lower, "zero") && !strings.Contains(lower, "divide") && !strings.Contains(lower, "panic") {
		t.Errorf("review should mention division by zero risk, got: %s", review)
	}

	if ev.Type().Value() != "agent.evaluated" {
		t.Errorf("event type = %q, want agent.evaluated", ev.Type().Value())
	}
	t.Logf("Review:\n%s", review)
}

func TestCodingReviewFindsInjection(t *testing.T) {
	rt := codingRuntime(t)
	ctx := context.Background()

	vulnerableCode := `func getUser(db *sql.DB, name string) (*User, error) {
    query := "SELECT * FROM users WHERE name = '" + name + "'"
    row := db.QueryRow(query)
    var u User
    err := row.Scan(&u.ID, &u.Name, &u.Email)
    return &u, err
}`

	_, review, err := rt.CodeReview(ctx, vulnerableCode, "go")
	if err != nil {
		t.Fatalf("CodeReview failed: %v", err)
	}

	lower := strings.ToLower(review)
	if !strings.Contains(lower, "injection") && !strings.Contains(lower, "sql") {
		t.Errorf("review should find SQL injection, got: %s", review)
	}
	t.Logf("Review:\n%s", review)
}

func TestCodingReviewCleanCode(t *testing.T) {
	rt := codingRuntime(t)
	ctx := context.Background()

	cleanCode := `func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}`

	_, review, err := rt.CodeReview(ctx, cleanCode, "go")
	if err != nil {
		t.Fatalf("CodeReview failed: %v", err)
	}

	// Should not find serious issues.
	t.Logf("Review of clean code:\n%s", review)
}

// ════════════════════════════════════════════════════════════════════════
// Code Writing
// ════════════════════════════════════════════════════════════════════════

func TestCodingWriteFunction(t *testing.T) {
	rt := codingRuntime(t)
	ctx := context.Background()

	code, err := rt.CodeWrite(ctx, "A Go function called IsPalindrome that takes a string and returns true if it reads the same forwards and backwards (case-insensitive, ignoring spaces)", "go")
	if err != nil {
		t.Fatalf("CodeWrite failed: %v", err)
	}

	if !strings.Contains(code, "IsPalindrome") {
		t.Errorf("code should contain IsPalindrome function, got:\n%s", code)
	}
	if !strings.Contains(code, "func") {
		t.Errorf("code should contain func keyword, got:\n%s", code)
	}
	t.Logf("Generated code:\n%s", code)

	// Verify the action was recorded on the graph.
	events, err := rt.EventsByType("agent.acted", 10)
	if err != nil {
		t.Fatalf("EventsByType failed: %v", err)
	}
	if len(events) == 0 {
		t.Error("code write action should be recorded on the graph")
	}
}

func TestCodingWriteTest(t *testing.T) {
	rt := codingRuntime(t)
	ctx := context.Background()

	code, err := rt.CodeWrite(ctx, "A Go table-driven test for a function `func Add(a, b int) int` that covers: positive numbers, negative numbers, zero, and overflow", "go")
	if err != nil {
		t.Fatalf("CodeWrite failed: %v", err)
	}

	if !strings.Contains(code, "func Test") {
		t.Errorf("should contain test function, got:\n%s", code)
	}
	if !strings.Contains(code, "testing") || !strings.Contains(code, "t.Run") || !strings.Contains(code, "cases") || !strings.Contains(code, "test") {
		// Just check some table-driven test indicators.
		t.Logf("Note: may not be table-driven format")
	}
	t.Logf("Generated test:\n%s", code)
}

// ════════════════════════════════════════════════════════════════════════
// Full coding task loop: evaluate → decide → write → review → learn
// ════════════════════════════════════════════════════════════════════════

func TestCodingFullLoop(t *testing.T) {
	rt := codingRuntime(t)
	ctx := context.Background()

	// 1. Run the Task composition on a coding task.
	result, err := rt.RunTask(ctx, "Write a Go function that validates email addresses using a regex")
	if err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}

	t.Logf("Evaluation: %s", result.Evaluation)
	t.Logf("Decision: %s", result.Decision)
	t.Logf("Lesson: %s", result.Lesson)

	if len(result.Events) != 5 {
		t.Errorf("expected 5 events, got %d", len(result.Events))
	}

	// 2. Now write the code.
	code, err := rt.CodeWrite(ctx, "A Go function ValidateEmail(email string) bool using regexp", "go")
	if err != nil {
		t.Fatalf("CodeWrite failed: %v", err)
	}
	t.Logf("Code:\n%s", code)

	// 3. Review our own code.
	_, review, err := rt.CodeReview(ctx, code, "go")
	if err != nil {
		t.Fatalf("CodeReview failed: %v", err)
	}
	t.Logf("Self-review:\n%s", review)

	// 4. Learn from the review.
	_, err = rt.Learn(ctx, "Recorded self-review findings: "+truncate(review, 200), "self_review")
	if err != nil {
		t.Fatalf("Learn failed: %v", err)
	}

	// 5. Verify full audit trail on the graph.
	allEvents, err := rt.Memory(50)
	if err != nil {
		t.Fatalf("Memory failed: %v", err)
	}
	t.Logf("Total events on graph: %d", len(allEvents))

	// Should have: bootstrap + 5 (task loop) + 1 (code write act) + 1 (code review eval) + 1 (learn) = 9+
	if len(allEvents) < 9 {
		t.Errorf("expected at least 9 events on graph, got %d", len(allEvents))
	}

	// Verify hash chain integrity.
	page, _ := rt.Store().Recent(100, types.None[types.Cursor]())
	items := page.Items()
	// Recent returns newest-first; reverse for chain order.
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	for i := 1; i < len(items); i++ {
		if items[i].PrevHash() != items[i-1].Hash() {
			t.Errorf("hash chain broken at event %d", i)
		}
	}
	t.Logf("Hash chain intact across %d events", len(items))
}

// ════════════════════════════════════════════════════════════════════════
// Agent introspection: agent examines its own event graph
// ════════════════════════════════════════════════════════════════════════

func TestCodingIntrospection(t *testing.T) {
	rt := codingRuntime(t)
	ctx := context.Background()

	// Do some work first.
	rt.Observe(ctx, 3)
	rt.Act(ctx, "review_code", "auth.go")
	rt.Learn(ctx, "Always validate input at boundaries", "code_review")

	// Now introspect — the agent examines its own event history.
	ev, observation, err := rt.Introspect(ctx,
		"Examine your event history. What patterns do you see in your actions? What should you focus on next? Respond in 2-3 sentences.")
	if err != nil {
		t.Fatalf("Introspect failed: %v", err)
	}

	if ev.Type().Value() != "agent.introspected" {
		t.Errorf("type = %q, want agent.introspected", ev.Type().Value())
	}
	if observation == "" {
		t.Error("introspection observation is empty")
	}
	t.Logf("Introspection:\n%s", observation)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
