package intelligence

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	claudecode "github.com/severity1/claude-agent-sdk-go"

	"github.com/lovyou-ai/eventgraph/go/pkg/decision"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
)

// errStaleSession is returned by consumeIterator when the CLI reports that a
// resumed session no longer exists. Callers retry without --resume.
var errStaleSession = errors.New("stale session")

// claudeSDKProvider implements Provider by using the community Go wrapper
// for the Claude Agent SDK. Unlike the raw claude-cli provider, this gives:
//   - Typed NDJSON streaming (AssistantMessage, ResultMessage, etc.)
//   - Session management via WithResume for warm context across calls
//   - Structured message parsing (thinking blocks, tool use, etc.)
//   - Better error typing (CLINotFoundError, ProcessError, etc.)
type claudeSDKProvider struct {
	model         string
	maxBudget     float64
	systemPrompt  string
	cliPath       string // path to claude binary, default "" (auto-detect)
	mcpConfigPath string // optional MCP server config file
	sessionID     string // if set, uses WithResume for warm sessions
	mu            sync.Mutex
}

func newClaudeSDKProvider(cfg Config) (*claudeSDKProvider, error) {
	model := cfg.Model
	if model == "" {
		model = "sonnet"
	}

	cliPath := ""
	if cfg.BaseURL != "" {
		cliPath = cfg.BaseURL
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

// SessionID returns the current session ID.
func (p *claudeSDKProvider) SessionID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sessionID
}

func (p *claudeSDKProvider) Reason(ctx context.Context, prompt string, history []event.Event) (decision.Response, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, defaultReasonTimeout)
	defer cancel()

	var fullPrompt strings.Builder
	historyText := eventsToMessages(history)
	if historyText != "" {
		fullPrompt.WriteString(historyText)
		fullPrompt.WriteString("\n---\n\n")
	}
	fullPrompt.WriteString(prompt)
	text := fullPrompt.String()

	result, usage, err := p.query(ctx, text, p.baseOptions())
	if err != nil && p.shouldRetryCold(err) {
		log.Printf("[claude-sdk] stale session for reason, retrying cold")
		p.clearSession()
		result, usage, err = p.query(ctx, text, p.baseOptions())
	}
	if err != nil {
		return decision.Response{}, fmt.Errorf("claude-sdk reason: %w", err)
	}
	if result == "" {
		return decision.Response{}, fmt.Errorf("claude-sdk returned empty result")
	}

	return decision.NewResponse(result, defaultConfidence(), usage), nil
}

func (p *claudeSDKProvider) Operate(ctx context.Context, task decision.OperateTask) (decision.OperateResult, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, defaultOperateTimeout)
	defer cancel()

	if task.WorkDir == "" {
		return decision.OperateResult{}, fmt.Errorf("Operate requires WorkDir")
	}
	if task.Instruction == "" {
		return decision.OperateResult{}, fmt.Errorf("Operate requires Instruction")
	}

	buildOpts := func() []claudecode.Option {
		opts := p.baseOptions()
		opts = append(opts,
			claudecode.WithCwd(task.WorkDir),
			claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
		)
		if len(task.AllowedTools) > 0 {
			opts = append(opts, claudecode.WithAllowedTools(task.AllowedTools...))
		} else {
			opts = append(opts, claudecode.WithAllowedTools("Read", "Edit", "Write", "Bash", "Glob", "Grep"))
		}
		return opts
	}

	result, usage, err := p.query(ctx, task.Instruction, buildOpts())
	if err != nil && p.shouldRetryCold(err) {
		log.Printf("[claude-sdk] stale session for operate, retrying cold")
		p.clearSession()
		result, usage, err = p.query(ctx, task.Instruction, buildOpts())
	}
	if err != nil {
		return decision.OperateResult{}, fmt.Errorf("claude-sdk operate: %w", err)
	}

	return decision.OperateResult{Summary: result, Usage: usage}, nil
}

// query runs a single Query call and consumes the iterator.
func (p *claudeSDKProvider) query(ctx context.Context, prompt string, opts []claudecode.Option) (string, decision.TokenUsage, error) {
	iter, err := claudecode.Query(ctx, prompt, opts...)
	if err != nil {
		return "", decision.TokenUsage{}, err
	}
	defer iter.Close()
	return p.consumeIterator(ctx, iter)
}

// shouldRetryCold returns true when the error is a stale/unavailable session.
func (p *claudeSDKProvider) shouldRetryCold(err error) bool {
	p.mu.Lock()
	hasSess := p.sessionID != ""
	p.mu.Unlock()
	if !hasSess {
		return false
	}
	return errors.Is(err, errStaleSession) || isSessionError(err)
}

func (p *claudeSDKProvider) clearSession() {
	p.mu.Lock()
	p.sessionID = ""
	p.mu.Unlock()
}

// baseOptions builds the common option set for all SDK calls.
func (p *claudeSDKProvider) baseOptions() []claudecode.Option {
	opts := []claudecode.Option{
		claudecode.WithModel(p.model),
		claudecode.WithMaxBudgetUSD(p.maxBudget),
		claudecode.WithStderrCallback(func(line string) {
			if line != "" {
				log.Printf("[claude-sdk:stderr] %s", line)
			}
		}),
	}

	p.mu.Lock()
	sid := p.sessionID
	p.mu.Unlock()
	if sid != "" {
		opts = append(opts, claudecode.WithResume(sid))
	}

	if p.systemPrompt != "" {
		opts = append(opts, claudecode.WithSystemPrompt(p.systemPrompt))
	}
	if p.mcpConfigPath != "" {
		opts = append(opts, claudecode.WithExtraArgs(map[string]*string{
			"mcp-config": &p.mcpConfigPath,
		}))
	}
	if p.cliPath != "" {
		opts = append(opts, claudecode.WithCLIPath(p.cliPath))
	}
	opts = append(opts,
		claudecode.WithEnvVar("DATABASE_URL", ""),
		claudecode.WithEnvVar("HIVE_AGENT_ID", ""),
		claudecode.WithEnvVar("HIVE_HUMAN_ID", ""),
	)

	return opts
}

// consumeIterator reads all messages from a Query iterator, printing progress
// heartbeats, and returns the final result text, token usage, and any error.
// Returns errStaleSession when the CLI reports a missing session.
func (p *claudeSDKProvider) consumeIterator(ctx context.Context, iter claudecode.MessageIterator) (string, decision.TokenUsage, error) {
	start := time.Now()
	ticker := time.NewTicker(progressInterval)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			elapsed := time.Since(start).Round(time.Second)
			fmt.Printf("  ⏳ thinking %s...\n", elapsed)
		}
	}()

	var resultText string
	var usage decision.TokenUsage

	for {
		msg, err := iter.Next(ctx)
		if errors.Is(err, claudecode.ErrNoMoreMessages) {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "unknown message type") {
				log.Printf("[claude-sdk] skipping: %v", err)
				continue
			}
			return "", decision.TokenUsage{}, err
		}

		switch m := msg.(type) {
		case *claudecode.ResultMessage:
			if m.Result != nil {
				resultText = *m.Result
			}
			if m.IsError {
				// Stale session: CLI can't find the conversation to resume.
				if m.NumTurns == 0 && m.DurationMs == 0 && m.Subtype == "error_during_execution" {
					return "", decision.TokenUsage{}, errStaleSession
				}
				errText := "(no result text)"
				if m.Result != nil {
					errText = *m.Result
				}
				return "", decision.TokenUsage{}, fmt.Errorf("claude error (subtype=%s, turns=%d, duration=%dms): %s",
					m.Subtype, m.NumTurns, m.DurationMs, errText)
			}

			if m.Usage != nil {
				usage = parseSDKUsage(*m.Usage)
			}
			if m.TotalCostUSD != nil {
				usage.CostUSD = *m.TotalCostUSD
			}
			if m.SessionID != "" {
				p.mu.Lock()
				p.sessionID = m.SessionID
				p.mu.Unlock()
			}
		}
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Printf("  ⏳ thinking done (%s)\n", elapsed)

	return resultText, usage, nil
}

func parseSDKUsage(m map[string]any) decision.TokenUsage {
	return decision.TokenUsage{
		InputTokens:      intFromMap(m, "input_tokens"),
		OutputTokens:     intFromMap(m, "output_tokens"),
		CacheReadTokens:  intFromMap(m, "cache_read_input_tokens"),
		CacheWriteTokens: intFromMap(m, "cache_creation_input_tokens"),
	}
}

func intFromMap(m map[string]any, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
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

func isSessionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "session") ||
		strings.Contains(msg, "already in use") ||
		strings.Contains(msg, "not found") ||
		claudecode.IsProcessError(err)
}

var (
	_ Provider           = (*claudeSDKProvider)(nil)
	_ decision.IOperator = (*claudeSDKProvider)(nil)
)
