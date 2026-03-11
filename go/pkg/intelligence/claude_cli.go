package intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lovyou-ai/eventgraph/go/pkg/decision"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
)

const (
	// defaultReasonTimeout is the maximum time a single Reason() call can run.
	// Kills the claude CLI subprocess if exceeded.
	defaultReasonTimeout = 5 * time.Minute

	// defaultOperateTimeout is the maximum time a single Operate() call can run.
	// Operate tasks (code generation) are heavier than Reason tasks.
	defaultOperateTimeout = 10 * time.Minute
)

// claudeCliResult is the JSON output from `claude -p --output-format json`.
type claudeCliResult struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
	Usage struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
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

	mcpConfig := cfg.MCPConfigPath
	if mcpConfig != "" {
		// Validate that the MCP config path is absolute and points to a regular file.
		if !filepath.IsAbs(mcpConfig) {
			return nil, fmt.Errorf("MCPConfigPath must be absolute: %s", mcpConfig)
		}
		info, err := os.Stat(mcpConfig)
		if err != nil {
			return nil, fmt.Errorf("MCPConfigPath: %w", err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("MCPConfigPath is not a regular file: %s", mcpConfig)
		}
	}

	return &claudeCliProvider{
		model:         model,
		maxBudget:     maxBudget,
		systemPrompt:  cfg.SystemPrompt,
		claudePath:    claudePath,
		mcpConfigPath: mcpConfig,
	}, nil
}

func (p *claudeCliProvider) Name() string  { return "claude-cli" }
func (p *claudeCliProvider) Model() string { return p.model }

func (p *claudeCliProvider) Reason(ctx context.Context, prompt string, history []event.Event) (decision.Response, error) {
	// Always apply the default timeout. context.WithTimeout takes the minimum
	// of parent deadline and child timeout, so this caps each call without
	// overriding a tighter parent deadline.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, defaultReasonTimeout)
	defer cancel()

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
	}
	args = append(args, "--no-session-persistence")
	if p.systemPrompt != "" {
		args = append(args, "--system-prompt", p.systemPrompt)
	}
	if p.mcpConfigPath != "" {
		args = append(args, "--mcp-config", p.mcpConfigPath)
	}

	cmd := exec.CommandContext(ctx, p.claudePath, args...)
	cmd.Stdin = strings.NewReader(fullPrompt.String())

	// Scrub env vars that should not leak into the Claude CLI subprocess.
	// CLAUDECODE is removed to allow nested invocation.
	// DATABASE_URL, HIVE_AGENT_ID, HIVE_HUMAN_ID contain credentials/identity
	// that the Claude CLI process doesn't need and shouldn't forward further.
	env := removeEnv(cmd.Environ(), "CLAUDECODE")
	env = removeEnv(env, "DATABASE_URL")
	env = removeEnv(env, "HIVE_AGENT_ID")
	env = removeEnv(env, "HIVE_HUMAN_ID")
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := runWithProgress(cmd, "  ⏳ thinking"); err != nil {
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
	confidence := defaultConfidence()
	usage := decision.TokenUsage{
		InputTokens:      result.Usage.InputTokens,
		OutputTokens:     result.Usage.OutputTokens,
		CacheReadTokens:  result.Usage.CacheReadInputTokens,
		CacheWriteTokens: result.Usage.CacheCreationInputTokens,
		CostUSD:          result.TotalCostUSD,
	}

	return decision.NewResponse(result.Result, confidence, usage), nil
}

func (p *claudeCliProvider) Operate(ctx context.Context, task decision.OperateTask) (decision.OperateResult, error) {
	// Always apply the default timeout. context.WithTimeout takes the minimum
	// of parent deadline and child timeout, so this caps each call without
	// overriding a tighter parent deadline.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, defaultOperateTimeout)
	defer cancel()

	if task.WorkDir == "" {
		return decision.OperateResult{}, fmt.Errorf("Operate requires WorkDir")
	}
	if task.Instruction == "" {
		return decision.OperateResult{}, fmt.Errorf("Operate requires Instruction")
	}

	args := []string{
		"-p",
		"--output-format", "json",
		"--model", p.model,
		"--max-budget-usd", fmt.Sprintf("%.2f", p.maxBudget),
		"--no-session-persistence",
		"--dangerously-skip-permissions",
	}

	if len(task.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(task.AllowedTools, ","))
	} else {
		// Default tools for agentic coding
		args = append(args, "--allowedTools", "Read,Edit,Write,Bash,Glob,Grep")
	}

	if p.systemPrompt != "" {
		args = append(args, "--system-prompt", p.systemPrompt)
	}
	if p.mcpConfigPath != "" {
		args = append(args, "--mcp-config", p.mcpConfigPath)
	}

	cmd := exec.CommandContext(ctx, p.claudePath, args...)
	cmd.Dir = task.WorkDir
	cmd.Stdin = strings.NewReader(task.Instruction)

	env := removeEnv(cmd.Environ(), "CLAUDECODE")
	env = removeEnv(env, "DATABASE_URL")
	env = removeEnv(env, "HIVE_AGENT_ID")
	env = removeEnv(env, "HIVE_HUMAN_ID")
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := runWithProgress(cmd, "  ⏳ working"); err != nil {
		// Check if we got JSON output despite non-zero exit.
		if stdout.Len() > 0 {
			var result claudeCliResult
			if jsonErr := json.Unmarshal(stdout.Bytes(), &result); jsonErr == nil && result.Result != "" {
				return p.resultToOperateResult(result), nil
			}
		}
		return decision.OperateResult{}, fmt.Errorf("claude CLI operate error: %w\nstderr: %s", err, stderr.String())
	}

	var result claudeCliResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return decision.OperateResult{}, fmt.Errorf("failed to parse claude CLI JSON output: %w\nraw: %s", err, stdout.String())
	}

	if result.IsError {
		return decision.OperateResult{}, fmt.Errorf("claude CLI operate returned error: %s (subtype: %s)", result.Result, result.Subtype)
	}

	return p.resultToOperateResult(result), nil
}

func (p *claudeCliProvider) resultToOperateResult(result claudeCliResult) decision.OperateResult {
	return decision.OperateResult{
		Summary: result.Result,
		Usage: decision.TokenUsage{
			InputTokens:      result.Usage.InputTokens,
			OutputTokens:     result.Usage.OutputTokens,
			CacheReadTokens:  result.Usage.CacheReadInputTokens,
			CacheWriteTokens: result.Usage.CacheCreationInputTokens,
			CostUSD:          result.TotalCostUSD,
		},
	}
}

// Compile-time check that claudeCliProvider implements IOperator.
var _ decision.IOperator = (*claudeCliProvider)(nil)

// progressInterval is how often runWithProgress prints an elapsed-time heartbeat.
const progressInterval = 10 * time.Second

// runWithProgress runs a command and prints periodic elapsed-time updates to
// stdout so the parent process doesn't appear frozen during long LLM calls.
func runWithProgress(cmd *exec.Cmd, prefix string) error {
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	start := time.Now()
	ticker := time.NewTicker(progressInterval)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			elapsed := time.Since(start).Round(time.Second)
			fmt.Printf("%s done (%s)\n", prefix, elapsed)
			return err
		case <-ticker.C:
			elapsed := time.Since(start).Round(time.Second)
			fmt.Printf("%s %s...\n", prefix, elapsed)
		}
	}
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

