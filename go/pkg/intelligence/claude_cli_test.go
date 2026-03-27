package intelligence_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/decision"
	"github.com/lovyou-ai/eventgraph/go/pkg/intelligence"
)

// TestMain handles the fake claude subprocess mode used by TestOperateIsErrorReturnsError.
// When GO_FAKE_CLAUDE_MODE is set, this binary acts as a fake claude CLI that emits
// a specific JSON response and exits, without running any tests.
func TestMain(m *testing.M) {
	switch os.Getenv("GO_FAKE_CLAUDE_MODE") {
	case "is_error_exit1":
		// Simulate claude outputting an error result with non-zero exit.
		// This is the scenario the IsError fix guards against.
		fmt.Print(`{"type":"result","subtype":"error_during_execution","is_error":true,"result":"task failed: permission denied","usage":{"input_tokens":0,"output_tokens":0},"total_cost_usd":0}`)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// TestOperateIsErrorReturnsError verifies that Operate returns an error when
// the JSON result has is_error:true and exit status 1.
// This test fails without the IsError check in the non-zero exit path of Operate.
func TestOperateIsErrorReturnsError(t *testing.T) {
	// Use the test binary itself as a fake claude. When invoked with
	// GO_FAKE_CLAUDE_MODE=is_error_exit1, TestMain exits early with error JSON
	// and a non-zero exit code — simulating a real claude failure.
	testBin, err := os.Executable()
	if err != nil {
		t.Fatalf("could not get test binary path: %v", err)
	}

	t.Setenv("GO_FAKE_CLAUDE_MODE", "is_error_exit1")

	p, err := intelligence.New(intelligence.Config{
		Provider: "claude-cli",
		Model:    "sonnet",
		BaseURL:  testBin, // repurposed as claude binary path in tests
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	op, ok := p.(decision.IOperator)
	if !ok {
		t.Fatal("claude-cli provider does not implement IOperator")
	}

	workDir := t.TempDir()
	_, err = op.Operate(context.Background(), decision.OperateTask{
		WorkDir:     workDir,
		Instruction: "do something",
	})
	if err == nil {
		t.Fatal("Operate should return an error when is_error:true and exit status 1, but got nil")
	}
	if !strings.Contains(err.Error(), "task failed") {
		t.Errorf("error should contain the result message, got: %v", err)
	}
}
