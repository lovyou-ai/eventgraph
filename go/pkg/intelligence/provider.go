package intelligence

import (
	"fmt"

	"github.com/lovyou-ai/eventgraph/go/pkg/decision"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Provider extends IIntelligence with metadata about the backing LLM.
type Provider interface {
	decision.IIntelligence

	// Name returns the provider identifier (e.g., "anthropic", "openai", "ollama").
	Name() string

	// Model returns the model identifier (e.g., "claude-sonnet-4-6", "gpt-4o").
	Model() string
}

// Config holds the configuration for creating a Provider.
type Config struct {
	// Provider name: "anthropic", "openai-compatible"
	Provider string

	// Model identifier (provider-specific).
	Model string

	// APIKey for authentication. If empty, the provider may fall back to
	// environment variables (e.g., ANTHROPIC_API_KEY).
	APIKey string

	// BaseURL overrides the default API endpoint.
	// Useful for proxies, Ollama (http://localhost:11434/v1), Azure, etc.
	BaseURL string

	// MaxTokens caps the response length. Defaults to 1024 if zero.
	MaxTokens int

	// Temperature controls randomness. Zero means provider default.
	Temperature float64

	// MaxBudgetUSD caps the cost per Claude CLI call in dollars.
	// Only used by the claude-cli provider. Defaults to $1.00 if zero.
	MaxBudgetUSD float64

	// SystemPrompt is prepended to every Reason call.
	SystemPrompt string

	// MCPConfigPath is an optional path to an MCP server config JSON file.
	// If set, the claude-cli provider passes --mcp-config to give the agent
	// access to MCP tools during reasoning. Ignored by other providers.
	MCPConfigPath string
}

// New creates a Provider from the given Config.
func New(cfg Config) (Provider, error) {
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 1024
	}

	switch cfg.Provider {
	case "anthropic":
		return newAnthropicProvider(cfg)
	case "claude-cli":
		return newClaudeCliProvider(cfg)
	case "openai-compatible", "openai", "xai", "groq", "together", "ollama":
		return newOpenAICompatibleProvider(cfg)
	default:
		return nil, fmt.Errorf("unknown provider: %q (supported: anthropic, claude-cli, openai-compatible, openai, xai, groq, together, ollama)", cfg.Provider)
	}
}

// eventsToMessages converts event history into a simple text context
// suitable for passing as conversation history to an LLM.
func eventsToMessages(events []event.Event) string {
	if len(events) == 0 {
		return ""
	}
	var b []byte
	b = append(b, "Event history:\n"...)
	for i, ev := range events {
		if i >= 20 {
			b = append(b, fmt.Sprintf("... and %d more events\n", len(events)-20)...)
			break
		}
		b = append(b, fmt.Sprintf("- [%s] %s by %s\n", ev.Type().Value(), ev.ID().Value(), ev.Source().Value())...)
	}
	return string(b)
}

// parseConfidence extracts a confidence score from token usage.
// Higher token usage with a stop reason of "end_turn" suggests higher confidence.
// This is a heuristic — real confidence requires model introspection.
func parseConfidence(tokensUsed int) types.Score {
	// Default to 0.7 — we can't truly measure confidence from outside the model.
	// Future: use log-probs or model self-assessment.
	s, _ := types.NewScore(0.7)
	return s
}
