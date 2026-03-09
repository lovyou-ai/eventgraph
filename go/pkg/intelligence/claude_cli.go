package intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lovyou-ai/eventgraph/go/pkg/decision"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
)

// claudeCliResult is the JSON output from `claude -p --output-format json`.
type claudeCliResult struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	StopReason   string  `json:"stop_reason"`
}

// claudeCliProvider implements Provider by shelling out to the `claude` CLI.
// This uses whatever authentication Claude Code already has (OAuth, API key, etc.)
// without requiring separate credentials.
type claudeCliProvider struct {
	model         string
	maxBudget     float64
	systemPrompt  string
	claudePath    string // path to claude binary, default "claude"
	mcpConfigPath string // optional MCP server config file
}

func newClaudeCliProvider(cfg Config) (*claudeCliProvider, error) {
	model := cfg.Model
	if model == "" {
		model = "sonnet"
	}

	claudePath := "claude"
	if cfg.BaseURL != "" {
		// BaseURL repurposed as path to claude binary for testing.
		claudePath = cfg.BaseURL
	}

	// Verify claude is available.
	if _, err := exec.LookPath(claudePath); err != nil {
		return nil, fmt.Errorf("claude CLI not found in PATH: %w", err)
	}

	maxBudget := 1.0 // default $1 per call
	if cfg.MaxBudgetUSD > 0 {
		maxBudget = cfg.MaxBudgetUSD
	}

	return &claudeCliProvider{
		model:         model,
		maxBudget:     maxBudget,
		systemPrompt:  cfg.SystemPrompt,
		claudePath:    claudePath,
		mcpConfigPath: cfg.MCPConfigPath,
	}, nil
}

func (p *claudeCliProvider) Name() string  { return "claude-cli" }
func (p *claudeCliProvider) Model() string { return p.model }

func (p *claudeCliProvider) Reason(ctx context.Context, prompt string, history []event.Event) (decision.Response, error) {
	// Build the full prompt with history context.
	var fullPrompt strings.Builder
	historyText := eventsToMessages(history)
	if historyText != "" {
		fullPrompt.WriteString(historyText)
		fullPrompt.WriteString("\n---\n\n")
	}
	fullPrompt.WriteString(prompt)

	// Build command args.
	args := []string{
		"-p",
		"--output-format", "json",
		"--model", p.model,
		"--max-budget-usd", fmt.Sprintf("%.2f", p.maxBudget),
		"--no-session-persistence",
	}
	if p.systemPrompt != "" {
		args = append(args, "--system-prompt", p.systemPrompt)
	}
	if p.mcpConfigPath != "" {
		args = append(args, "--mcp-config", p.mcpConfigPath)
	}

	cmd := exec.CommandContext(ctx, p.claudePath, args...)
	cmd.Stdin = strings.NewReader(fullPrompt.String())

	// Unset CLAUDECODE to allow nested invocation (e.g., when run from within Claude Code).
	cmd.Env = removeEnv(cmd.Environ(), "CLAUDECODE")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if we got JSON output despite non-zero exit.
		if stdout.Len() > 0 {
			var result claudeCliResult
			if jsonErr := json.Unmarshal(stdout.Bytes(), &result); jsonErr == nil && result.Result != "" {
				// Budget exceeded but still got a result.
				return p.resultToResponse(result)
			}
		}
		return decision.Response{}, fmt.Errorf("claude CLI error: %w\nstderr: %s", err, stderr.String())
	}

	var result claudeCliResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return decision.Response{}, fmt.Errorf("failed to parse claude CLI JSON output: %w\nraw: %s", err, stdout.String())
	}

	if result.IsError {
		return decision.Response{}, fmt.Errorf("claude CLI returned error: %s (subtype: %s)", result.Result, result.Subtype)
	}

	return p.resultToResponse(result)
}

func (p *claudeCliProvider) resultToResponse(result claudeCliResult) (decision.Response, error) {
	tokensUsed := result.Usage.InputTokens + result.Usage.OutputTokens
	confidence := parseConfidence(tokensUsed)

	return decision.NewResponse(result.Result, confidence, tokensUsed), nil
}

// removeEnv returns a copy of env with the named variable removed.
func removeEnv(env []string, key string) []string {
	if env == nil {
		return nil
	}
	prefix := key + "="
	out := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return out
}

// Compile-time check.
var _ Provider = (*claudeCliProvider)(nil)

// NewClaudeCliConfig creates a Config for the Claude CLI provider.
// This is a convenience function for the most common case.
func NewClaudeCliConfig(model string) Config {
	if model == "" {
		model = "sonnet"
	}
	return Config{
		Provider: "claude-cli",
		Model:    model,
	}
}

