package intelligence

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"strings"

	"github.com/lovyou-ai/eventgraph/go/pkg/agent"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// AgentRuntime ties together identity, intelligence, memory (event graph), and
// compositions into a running agent. Each operation emits events on the graph.
type AgentRuntime struct {
	id       types.ActorID
	provider Provider
	store    store.Store
	factory  *event.EventFactory
	signer   event.Signer
	convID   types.ConversationID

	// bootstrapID is the genesis event, used as default cause.
	bootstrapID types.EventID
}

// RuntimeConfig configures a new AgentRuntime.
type RuntimeConfig struct {
	// AgentID is the agent's identity. Required.
	AgentID types.ActorID

	// Provider is the LLM backend. Required.
	Provider Provider

	// Store is the event graph (memory). If nil, an in-memory store is created.
	Store store.Store

	// ConversationID groups this agent's events. If zero, one is generated.
	ConversationID types.ConversationID
}

// NewRuntime creates a new AgentRuntime and bootstraps it (genesis event + boot composition).
func NewRuntime(ctx context.Context, cfg RuntimeConfig) (*AgentRuntime, error) {
	if cfg.Provider == nil {
		return nil, fmt.Errorf("provider is required")
	}

	s := cfg.Store
	if s == nil {
		s = store.NewInMemoryStore()
	}

	convID := cfg.ConversationID
	if convID == (types.ConversationID{}) {
		var err error
		convID, err = types.NewConversationID("conv_agent_000000000000000000000001")
		if err != nil {
			return nil, err
		}
	}

	// Generate a signing key for this agent.
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}
	signer := &ed25519Signer{key: privKey}

	registry := event.DefaultRegistry()
	factory := event.NewEventFactory(registry)

	// Bootstrap the event graph — only if the store is empty.
	// When agents share a store, only the first agent creates the genesis event.
	// Subsequent agents join the existing graph and chain off its head.
	var bootstrapID types.EventID
	head, err := s.Head()
	if err != nil {
		return nil, fmt.Errorf("check store head: %w", err)
	}
	if head.IsSome() {
		// Store already has events — use the current head as our starting point.
		bootstrapID = head.Unwrap().ID()
	} else {
		// Empty store — create the genesis event.
		bsFactory := event.NewBootstrapFactory(registry)
		bootstrap, err := bsFactory.Init(cfg.AgentID, signer)
		if err != nil {
			return nil, fmt.Errorf("bootstrap: %w", err)
		}
		if _, err := s.Append(bootstrap); err != nil {
			return nil, fmt.Errorf("append bootstrap: %w", err)
		}
		bootstrapID = bootstrap.ID()
	}

	return &AgentRuntime{
		id:          cfg.AgentID,
		provider:    cfg.Provider,
		store:       s,
		factory:     factory,
		signer:      signer,
		convID:      convID,
		bootstrapID: bootstrapID,
	}, nil
}

// ID returns the agent's identity.
func (r *AgentRuntime) ID() types.ActorID { return r.id }

// Store returns the agent's event graph.
func (r *AgentRuntime) Store() store.Store { return r.store }

// Provider returns the agent's intelligence provider.
func (r *AgentRuntime) Provider() Provider { return r.provider }

// ════════════════════════════════════════════════════════════════════════
// Event emission
// ════════════════════════════════════════════════════════════════════════

// Emit creates and appends an event to the graph, using the most recent event as cause.
// Use this to record any EventContent (including Code Graph events) on the agent's graph.
func (r *AgentRuntime) Emit(content event.EventContent) (event.Event, error) {
	cause, err := r.lastEventID()
	if err != nil {
		return event.Event{}, err
	}
	ev, err := r.factory.Create(
		types.MustEventType(content.EventTypeName()),
		r.id,
		content,
		[]types.EventID{cause},
		r.convID,
		r.store,
		r.signer,
	)
	if err != nil {
		return event.Event{}, fmt.Errorf("create event %s: %w", content.EventTypeName(), err)
	}
	stored, err := r.store.Append(ev)
	if err != nil {
		return event.Event{}, fmt.Errorf("append event %s: %w", content.EventTypeName(), err)
	}
	return stored, nil
}

// lastEventID returns the most recent event ID (chain head), or bootstrap if empty.
func (r *AgentRuntime) lastEventID() (types.EventID, error) {
	head, err := r.store.Head()
	if err != nil {
		return types.EventID{}, err
	}
	if head.IsSome() {
		return head.Unwrap().ID(), nil
	}
	return r.bootstrapID, nil
}

// ════════════════════════════════════════════════════════════════════════
// Memory: query the event graph for context
// ════════════════════════════════════════════════════════════════════════

// Memory returns recent events from this agent's graph as context for reasoning.
func (r *AgentRuntime) Memory(limit int) ([]event.Event, error) {
	page, err := r.store.BySource(r.id, limit, types.None[types.Cursor]())
	if err != nil {
		return nil, err
	}
	return page.Items(), nil
}

// EventsByType returns events of a specific type from the graph.
func (r *AgentRuntime) EventsByType(eventType string, limit int) ([]event.Event, error) {
	et := types.MustEventType(eventType)
	page, err := r.store.ByType(et, limit, types.None[types.Cursor]())
	if err != nil {
		return nil, err
	}
	return page.Items(), nil
}

// ════════════════════════════════════════════════════════════════════════
// Compositions
// ════════════════════════════════════════════════════════════════════════

// Boot executes the Boot composition: Identity + Soul + Model + Authority + State.
// Returns the events emitted.
func (r *AgentRuntime) Boot(agentType string, modelID string, costTier string, values []string, scope types.DomainScope, grantor types.ActorID) ([]event.Event, error) {
	contents := agent.BootEvents(r.id, agentType, modelID, costTier, values, scope, grantor)
	var events []event.Event
	for _, c := range contents {
		ev, err := r.Emit(c)
		if err != nil {
			return events, err
		}
		events = append(events, ev)
	}
	return events, nil
}

// Observe records that the agent observed something. Returns the event.
func (r *AgentRuntime) Observe(ctx context.Context, eventCount int) (event.Event, error) {
	return r.Emit(event.AgentObservedContent{
		AgentID:    r.id,
		EventCount: eventCount,
	})
}

// Evaluate uses the LLM to evaluate a subject and records the result.
func (r *AgentRuntime) Evaluate(ctx context.Context, subject string, prompt string) (event.Event, string, error) {
	memory, _ := r.Memory(10)
	resp, err := r.provider.Reason(ctx, prompt, memory)
	if err != nil {
		return event.Event{}, "", fmt.Errorf("evaluate reasoning: %w", err)
	}

	confidence := resp.Confidence()
	ev, err := r.Emit(event.AgentEvaluatedContent{
		AgentID:    r.id,
		Subject:    subject,
		Confidence: confidence,
		Result:     resp.Content(),
	})
	return ev, resp.Content(), err
}

// Decide uses the LLM to make a decision and records it.
func (r *AgentRuntime) Decide(ctx context.Context, action string, prompt string) (event.Event, string, error) {
	memory, _ := r.Memory(10)
	resp, err := r.provider.Reason(ctx, prompt, memory)
	if err != nil {
		return event.Event{}, "", fmt.Errorf("decide reasoning: %w", err)
	}

	ev, err := r.Emit(event.AgentDecidedContent{
		AgentID:    r.id,
		Action:     action,
		Confidence: resp.Confidence(),
	})
	return ev, resp.Content(), err
}

// Act records an action taken. Returns the event.
func (r *AgentRuntime) Act(ctx context.Context, action string, target string) (event.Event, error) {
	return r.Emit(event.AgentActedContent{
		AgentID: r.id,
		Action:  action,
		Target:  target,
	})
}

// Learn records a lesson learned. Returns the event.
func (r *AgentRuntime) Learn(ctx context.Context, lesson string, source string) (event.Event, error) {
	return r.Emit(event.AgentLearnedContent{
		AgentID: r.id,
		Lesson:  lesson,
		Source:   source,
	})
}

// Refuse records a refusal. Returns the event.
func (r *AgentRuntime) Refuse(ctx context.Context, action string, reason string) (event.Event, error) {
	return r.Emit(event.AgentRefusedContent{
		AgentID: r.id,
		Action:  action,
		Reason:  reason,
	})
}

// Escalate records an escalation. Returns the event.
func (r *AgentRuntime) Escalate(ctx context.Context, authority types.ActorID, reason string) (event.Event, error) {
	return r.Emit(event.AgentEscalatedContent{
		AgentID:   r.id,
		Authority: authority,
		Reason:    reason,
	})
}

// Introspect uses the LLM for self-observation and records it.
func (r *AgentRuntime) Introspect(ctx context.Context, prompt string) (event.Event, string, error) {
	memory, _ := r.Memory(20)
	resp, err := r.provider.Reason(ctx, prompt, memory)
	if err != nil {
		return event.Event{}, "", fmt.Errorf("introspect reasoning: %w", err)
	}

	ev, err := r.Emit(event.AgentIntrospectedContent{
		AgentID:     r.id,
		Observation: resp.Content(),
	})
	return ev, resp.Content(), err
}

// ════════════════════════════════════════════════════════════════════════
// High-level: Task composition (Observe → Evaluate → Decide → Act → Learn)
// ════════════════════════════════════════════════════════════════════════

// TaskResult holds the output of a full Task composition loop.
type TaskResult struct {
	Observation string
	Evaluation  string
	Decision    string
	Action      string
	Lesson      string
	Events      []event.Event
}

// RunTask executes the full Task composition with LLM reasoning at each step.
func (r *AgentRuntime) RunTask(ctx context.Context, taskDescription string) (*TaskResult, error) {
	result := &TaskResult{}

	// 1. Observe
	ev, err := r.Observe(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("observe: %w", err)
	}
	result.Events = append(result.Events, ev)

	// 2. Evaluate
	evalPrompt := fmt.Sprintf("You are evaluating a task.\nTask: %s\n\nEvaluate the scope, required authority, and complexity. Respond concisely in 2-3 sentences.", taskDescription)
	ev, evaluation, err := r.Evaluate(ctx, taskDescription, evalPrompt)
	if err != nil {
		return nil, fmt.Errorf("evaluate: %w", err)
	}
	result.Evaluation = evaluation
	result.Events = append(result.Events, ev)

	// 3. Decide
	decidePrompt := fmt.Sprintf("Based on this evaluation: %s\n\nDecide: should you PROCEED with this task or ESCALATE to a human?\nRespond with exactly: PROCEED or ESCALATE", evaluation)
	ev, decision, err := r.Decide(ctx, taskDescription, decidePrompt)
	if err != nil {
		return nil, fmt.Errorf("decide: %w", err)
	}
	result.Decision = strings.TrimSpace(decision)
	result.Events = append(result.Events, ev)

	// 4. Act
	actionDesc := "execute: " + taskDescription
	if strings.HasPrefix(strings.ToUpper(result.Decision), "ESCALATE") {
		actionDesc = "escalate: " + taskDescription
	}
	ev, err = r.Act(ctx, actionDesc, taskDescription)
	if err != nil {
		return nil, fmt.Errorf("act: %w", err)
	}
	result.Action = actionDesc
	result.Events = append(result.Events, ev)

	// 5. Learn
	learnPrompt := fmt.Sprintf("Task completed: %s\nEvaluation: %s\nDecision: %s\n\nWhat lesson should be recorded? One sentence starting with 'When'.", taskDescription, evaluation, result.Decision)
	memory, _ := r.Memory(5)
	resp, err := r.provider.Reason(ctx, learnPrompt, memory)
	if err != nil {
		return nil, fmt.Errorf("learn reasoning: %w", err)
	}
	ev, err = r.Learn(ctx, resp.Content(), "task_completion")
	if err != nil {
		return nil, fmt.Errorf("learn: %w", err)
	}
	result.Lesson = resp.Content()
	result.Events = append(result.Events, ev)

	return result, nil
}

// ════════════════════════════════════════════════════════════════════════
// Research: read web resources and extract information
// ════════════════════════════════════════════════════════════════════════

// Research asks the agent to read a URL (or other resource) and extract
// structured information. The LLM backend handles fetching — Claude CLI
// has built-in web tools, other providers accept URLs in prompts.
// Records an observation (what was read) and an evaluation (what was extracted).
func (r *AgentRuntime) Research(ctx context.Context, url string, extractionPrompt string) (observation event.Event, evaluation string, err error) {
	// 1. Record that we're reading this resource
	obs, err := r.Observe(ctx, 1)
	if err != nil {
		return event.Event{}, "", fmt.Errorf("observe: %w", err)
	}

	// 2. Ask the LLM to read and extract
	fullPrompt := fmt.Sprintf("Read the following URL and %s\n\nURL: %s", extractionPrompt, url)
	memory, _ := r.Memory(5)
	resp, err := r.provider.Reason(ctx, fullPrompt, memory)
	if err != nil {
		return event.Event{}, "", fmt.Errorf("research reasoning: %w", err)
	}

	// 3. Record what was extracted
	_, err = r.Emit(event.AgentEvaluatedContent{
		AgentID:    r.id,
		Subject:    "research:" + url,
		Confidence: resp.Confidence(),
		Result:     resp.Content(),
	})
	if err != nil {
		return event.Event{}, "", fmt.Errorf("record evaluation: %w", err)
	}

	return obs, resp.Content(), nil
}

// ════════════════════════════════════════════════════════════════════════
// Coding: use Claude CLI with tools for code tasks
// ════════════════════════════════════════════════════════════════════════

// CodeReview asks the agent to review code and records the evaluation.
func (r *AgentRuntime) CodeReview(ctx context.Context, code string, language string) (event.Event, string, error) {
	prompt := fmt.Sprintf("Review this %s code for bugs, security issues, and quality problems.\nBe specific. List issues found, or say \"No issues found\" if clean.\n\n%s", language, code)

	return r.Evaluate(ctx, "code_review:"+language, prompt)
}

// CodeWrite asks the agent to write code and records the action.
func (r *AgentRuntime) CodeWrite(ctx context.Context, spec string, language string) (string, error) {
	prompt := fmt.Sprintf(`Write %s code for the following specification.
Return ONLY the code, no explanation, no markdown fences.

Spec: %s`, language, spec)

	memory, _ := r.Memory(5)
	resp, err := r.provider.Reason(ctx, prompt, memory)
	if err != nil {
		return "", fmt.Errorf("code write reasoning: %w", err)
	}

	_, err = r.Act(ctx, "write_code:"+language, spec)
	if err != nil {
		return "", fmt.Errorf("record code write: %w", err)
	}

	return resp.Content(), nil
}

// ════════════════════════════════════════════════════════════════════════
// Signer implementation
// ════════════════════════════════════════════════════════════════════════

type ed25519Signer struct {
	key ed25519.PrivateKey
}

func (s *ed25519Signer) Sign(data []byte) (types.Signature, error) {
	sig := ed25519.Sign(s.key, data)
	return types.NewSignature(sig)
}
