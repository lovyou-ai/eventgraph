package intelligence

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lovyou-ai/eventgraph/go/pkg/decision"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
)

// errStaleSession is returned when the CLI reports that a resumed session
// no longer exists. Callers retry without --resume.
var errStaleSession = errors.New("stale session")

// claudeSDKProvider implements Provider using Claude CLI with stream-json
// output and our own NDJSON parser. Unlike the Go SDK wrapper, we parse
// inline (like the Python SDK) — unknown message types are skipped, not
// errors. No errChan/msgChan race. Session management via --resume.
type claudeSDKProvider struct {
	model         string
	maxBudget     float64
	systemPrompt  string
	cliPath       string
	mcpConfigPath string
	sessionID     string
	mu            sync.Mutex
}

func newClaudeSDKProvider(cfg Config) (*claudeSDKProvider, error) {
	model := cfg.Model
	if model == "" {
		model = "sonnet"
	}

	cliPath := "claude"
	if cfg.BaseURL != "" {
		cliPath = cfg.BaseURL
	}
	if _, err := exec.LookPath(cliPath); err != nil {
		return nil, fmt.Errorf("claude CLI not found in PATH: %w", err)
	}

	maxBudget := 1.0
	if cfg.MaxBudgetUSD > 0 {
		maxBudget = cfg.MaxBudgetUSD
	}

	mcpConfig := cfg.MCPConfigPath
	if mcpConfig != "" {
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

	return &claudeSDKProvider{
		model:         model,
		maxBudget:     maxBudget,
		systemPrompt:  cfg.SystemPrompt,
		cliPath:       cliPath,
		mcpConfigPath: mcpConfig,
		sessionID:     cfg.SessionID,
	}, nil
}

func (p *claudeSDKProvider) Name() string  { return "claude-sdk" }
func (p *claudeSDKProvider) Model() string { return p.model }

func (p *claudeSDKProvider) SessionID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sessionID
}

func (p *claudeSDKProvider) Reason(ctx context.Context, prompt string, history []event.Event) (decision.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultReasonTimeout)
	defer cancel()

	var fullPrompt strings.Builder
	historyText := eventsToMessages(history)
	if historyText != "" {
		fullPrompt.WriteString(historyText)
		fullPrompt.WriteString("\n---\n\n")
	}
	fullPrompt.WriteString(prompt)

	result, err := p.run(ctx, fullPrompt.String(), p.baseArgs(), "")
	if err != nil && p.shouldRetryCold(err) {
		log.Printf("[claude-sdk] stale session for reason, retrying cold")
		p.clearSession()
		result, err = p.run(ctx, fullPrompt.String(), p.baseArgs(), "")
	}
	if err != nil {
		return decision.Response{}, fmt.Errorf("claude-sdk reason: %w", err)
	}

	return decision.NewResponse(result.Result, defaultConfidence(), result.usage()), nil
}

func (p *claudeSDKProvider) Operate(ctx context.Context, task decision.OperateTask) (decision.OperateResult, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultOperateTimeout)
	defer cancel()

	if task.WorkDir == "" {
		return decision.OperateResult{}, fmt.Errorf("Operate requires WorkDir")
	}
	if task.Instruction == "" {
		return decision.OperateResult{}, fmt.Errorf("Operate requires Instruction")
	}

	buildArgs := func() []string {
		a := p.baseArgs()
		a = append(a, "--permission-mode", "bypassPermissions")
		if len(task.AllowedTools) > 0 {
			a = append(a, "--allowed-tools", strings.Join(task.AllowedTools, ","))
		} else {
			a = append(a, "--allowed-tools", "Read,Edit,Write,Bash,Glob,Grep")
		}
		return a
	}

	result, err := p.run(ctx, task.Instruction, buildArgs(), task.WorkDir)
	if err != nil && p.shouldRetryCold(err) {
		log.Printf("[claude-sdk] stale session for operate, retrying cold")
		p.clearSession()
		result, err = p.run(ctx, task.Instruction, buildArgs(), task.WorkDir)
	}
	if err != nil {
		return decision.OperateResult{}, fmt.Errorf("claude-sdk operate: %w", err)
	}

	return decision.OperateResult{Summary: result.Result, Usage: result.usage()}, nil
}

// streamResult holds the accumulated output from parsing NDJSON.
type streamResult struct {
	Result    string
	SessionID string
	IsError   bool
	Subtype   string
	NumTurns  int
	DurationMs int
	Cost      float64
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
}

func (r *streamResult) usage() decision.TokenUsage {
	return decision.TokenUsage{
		InputTokens:      r.InputTokens,
		OutputTokens:     r.OutputTokens,
		CacheReadTokens:  r.CacheReadTokens,
		CacheWriteTokens: r.CacheWriteTokens,
		CostUSD:          r.Cost,
	}
}

func (p *claudeSDKProvider) baseArgs() []string {
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--model", p.model,
		"--max-budget-usd", fmt.Sprintf("%.2f", p.maxBudget),
	}
	p.mu.Lock()
	sid := p.sessionID
	p.mu.Unlock()
	if sid != "" {
		args = append(args, "--resume", sid)
	}
	if p.systemPrompt != "" {
		args = append(args, "--system-prompt", p.systemPrompt)
	}
	if p.mcpConfigPath != "" {
		args = append(args, "--mcp-config", p.mcpConfigPath)
	}
	return args
}

// run spawns the claude CLI and parses its NDJSON output line by line.
// Unknown message types (rate_limit_event, etc.) are skipped — not errors.
// This is the Python SDK's approach: forward-compatible, no race conditions.
func (p *claudeSDKProvider) run(ctx context.Context, prompt string, args []string, workDir string) (*streamResult, error) {
	cmd := exec.CommandContext(ctx, p.cliPath, args...)
	cmd.Stdin = strings.NewReader(prompt)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = scrubEnv(cmd.Environ(), "CLAUDECODE", "DATABASE_URL", "HIVE_AGENT_ID", "HIVE_HUMAN_ID")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr // let claude's stderr flow through for debugging

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude: %w", err)
	}

	// Parse NDJSON inline — one goroutine, one channel, no races.
	var result streamResult
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB for large tool results

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue // malformed line, skip
		}

		msgType, _ := msg["type"].(string)
		switch msgType {
		case "system":
			// Capture session ID from init message.
			if sid, ok := msg["session_id"].(string); ok && sid != "" {
				result.SessionID = sid
			}

		case "result":
			// Final message — contains result text, usage, cost, session ID.
			if r, ok := msg["result"].(string); ok {
				result.Result = r
			}
			result.IsError, _ = msg["is_error"].(bool)
			result.Subtype, _ = msg["subtype"].(string)
			result.NumTurns = intFromAny(msg["num_turns"])
			result.DurationMs = intFromAny(msg["duration_ms"])
			result.Cost, _ = msg["total_cost_usd"].(float64)
			if sid, ok := msg["session_id"].(string); ok && sid != "" {
				result.SessionID = sid
			}
			if u, ok := msg["usage"].(map[string]any); ok {
				result.InputTokens = intFromAny(u["input_tokens"])
				result.OutputTokens = intFromAny(u["output_tokens"])
				result.CacheReadTokens = intFromAny(u["cache_read_input_tokens"])
				result.CacheWriteTokens = intFromAny(u["cache_creation_input_tokens"])
			}

		case "assistant":
			// Log tool use for progress visibility.
			if m, ok := msg["message"].(map[string]any); ok {
				if content, ok := m["content"].([]any); ok {
					for _, c := range content {
						block, _ := c.(map[string]any)
						if block["type"] == "tool_use" {
							name, _ := block["name"].(string)
							log.Printf("  🔧 %s", name)
						}
					}
				}
			}

		default:
			// Forward-compatible: skip unrecognized message types so newer
			// CLI versions don't crash older SDK versions. (Python SDK pattern.)
		}
	}

	// Wait for process to exit.
	if err := cmd.Wait(); err != nil {
		// If we got a result despite non-zero exit, use it.
		if result.Result != "" || result.IsError {
			// fall through to error checking below
		} else {
			return nil, fmt.Errorf("claude CLI error: %w", err)
		}
	}

	// Capture session for warm resumption.
	p.captureSession(result.SessionID)

	// Check for errors in the result.
	if result.IsError {
		if result.NumTurns == 0 && result.DurationMs == 0 {
			return nil, errStaleSession
		}
		errText := result.Result
		if errText == "" {
			errText = "(no result text)"
		}
		return nil, fmt.Errorf("claude error (subtype=%s, turns=%d): %s",
			result.Subtype, result.NumTurns, errText)
	}

	return &result, nil
}

func (p *claudeSDKProvider) captureSession(sid string) {
	if sid == "" {
		return
	}
	p.mu.Lock()
	p.sessionID = sid
	p.mu.Unlock()
}

func (p *claudeSDKProvider) shouldRetryCold(err error) bool {
	p.mu.Lock()
	has := p.sessionID != ""
	p.mu.Unlock()
	return has && errors.Is(err, errStaleSession)
}

func (p *claudeSDKProvider) clearSession() {
	p.mu.Lock()
	p.sessionID = ""
	p.mu.Unlock()
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

func scrubEnv(env []string, keys ...string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		skip := false
		for _, k := range keys {
			if strings.HasPrefix(e, k+"=") {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, e)
		}
	}
	return out
}

var (
	_ Provider           = (*claudeSDKProvider)(nil)
	_ decision.IOperator = (*claudeSDKProvider)(nil)
)
