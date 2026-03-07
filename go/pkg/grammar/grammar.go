// Package grammar implements the 15 social grammar operations as compositions
// of the event graph primitives. This is a product layer — it builds on top of
// the graph infrastructure, not inside it.
//
// NOTE: All methods accept context.Context for future cancellation/deadline support,
// but context is not yet propagated to graph.Record. This will be addressed when
// graph.Record gains context support.
package grammar

import (
	"context"
	"fmt"

	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/graph"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Grammar provides the 15 social grammar operations.
// Each operation creates one or more events on the graph.
type Grammar struct {
	graph *graph.Graph
}

// New creates a Grammar bound to the given graph.
func New(g *graph.Graph) *Grammar {
	return &Grammar{graph: g}
}

// --- Vertex operations ---

// Emit creates independent content. (Operation 1)
func (g *Grammar) Emit(
	ctx context.Context,
	source types.ActorID,
	body string,
	conversationID types.ConversationID,
	causes []types.EventID,
	signer event.Signer,
) (event.Event, error) {
	if len(causes) == 0 {
		return event.Event{}, fmt.Errorf("emit requires at least one cause")
	}
	return g.graph.Record(
		event.EventTypeGrammarEmit, source,
		event.GrammarEmitContent{Body: body},
		causes, conversationID, signer,
	)
}

// Respond creates causally dependent, subordinate content. (Operation 2)
func (g *Grammar) Respond(
	ctx context.Context,
	source types.ActorID,
	body string,
	parent types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeGrammarRespond, source,
		event.GrammarRespondContent{Body: body, Parent: parent},
		[]types.EventID{parent}, conversationID, signer,
	)
}

// Derive creates causally dependent but independent content. (Operation 3)
func (g *Grammar) Derive(
	ctx context.Context,
	source types.ActorID,
	body string,
	sourceEvent types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeGrammarDerive, source,
		event.GrammarDeriveContent{Body: body, Source: sourceEvent},
		[]types.EventID{sourceEvent}, conversationID, signer,
	)
}

// Extend creates sequential content from the same author. (Operation 4)
func (g *Grammar) Extend(
	ctx context.Context,
	source types.ActorID,
	body string,
	previous types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeGrammarExtend, source,
		event.GrammarExtendContent{Body: body, Previous: previous},
		[]types.EventID{previous}, conversationID, signer,
	)
}

// Retract tombstones own content. Provenance survives. (Operation 5)
// Only the original author can retract their own content.
func (g *Grammar) Retract(
	ctx context.Context,
	source types.ActorID,
	target types.EventID,
	reason string,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	// Verify the source authored the target event
	targetEv, err := g.graph.Store().Get(target)
	if err != nil {
		return event.Event{}, fmt.Errorf("retract: target event not found: %w", err)
	}
	if targetEv.Source() != source {
		return event.Event{}, fmt.Errorf("retract: actor %s cannot retract event %s authored by %s", source.Value(), target.Value(), targetEv.Source().Value())
	}
	return g.graph.Record(
		event.EventTypeGrammarRetract, source,
		event.GrammarRetractContent{Target: target, Reason: reason},
		[]types.EventID{target}, conversationID, signer,
	)
}

// Annotate attaches metadata to existing content. (Operation 6)
func (g *Grammar) Annotate(
	ctx context.Context,
	source types.ActorID,
	target types.EventID,
	key, value string,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeGrammarAnnotate, source,
		event.GrammarAnnotateContent{Target: target, Key: key, Value: value},
		[]types.EventID{target}, conversationID, signer,
	)
}

// --- Edge operations ---

// Acknowledge creates a content-free centripetal edge toward a vertex. (Operation 7)
func (g *Grammar) Acknowledge(
	ctx context.Context,
	source types.ActorID,
	target types.EventID,
	targetActor types.ActorID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeEdgeCreated, source,
		event.EdgeCreatedContent{
			From:      source,
			To:        targetActor,
			EdgeType:  event.EdgeTypeAcknowledgement,
			Weight:    types.MustWeight(0),
			Direction: event.EdgeDirectionCentripetal,
			Scope:     types.None[types.DomainScope](),
		},
		[]types.EventID{target}, conversationID, signer,
	)
}

// Propagate redistributes a vertex into the actor's subgraph. (Operation 8)
func (g *Grammar) Propagate(
	ctx context.Context,
	source types.ActorID,
	target types.EventID,
	targetActor types.ActorID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeEdgeCreated, source,
		event.EdgeCreatedContent{
			From:      source,
			To:        targetActor,
			EdgeType:  event.EdgeTypeReference,
			Weight:    types.MustWeight(0),
			Direction: event.EdgeDirectionCentrifugal,
			Scope:     types.None[types.DomainScope](),
		},
		[]types.EventID{target}, conversationID, signer,
	)
}

// Endorse creates a reputation-staked edge toward a vertex. (Operation 9)
func (g *Grammar) Endorse(
	ctx context.Context,
	source types.ActorID,
	target types.EventID,
	targetActor types.ActorID,
	weight types.Weight,
	scope types.Option[types.DomainScope],
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeEdgeCreated, source,
		event.EdgeCreatedContent{
			From:      source,
			To:        targetActor,
			EdgeType:  event.EdgeTypeEndorsement,
			Weight:    weight,
			Direction: event.EdgeDirectionCentripetal,
			Scope:     scope,
		},
		[]types.EventID{target}, conversationID, signer,
	)
}

// Subscribe creates a persistent, future-oriented edge to an actor. (Operation 10)
func (g *Grammar) Subscribe(
	ctx context.Context,
	source types.ActorID,
	target types.ActorID,
	scope types.Option[types.DomainScope],
	cause types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeEdgeCreated, source,
		event.EdgeCreatedContent{
			From:      source,
			To:        target,
			EdgeType:  event.EdgeTypeSubscription,
			Weight:    types.MustWeight(0),
			Direction: event.EdgeDirectionCentripetal,
			Scope:     scope,
		},
		[]types.EventID{cause}, conversationID, signer,
	)
}

// Channel creates a private, bidirectional, content-bearing edge. (Operation 11)
func (g *Grammar) Channel(
	ctx context.Context,
	source types.ActorID,
	target types.ActorID,
	scope types.Option[types.DomainScope],
	cause types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeEdgeCreated, source,
		event.EdgeCreatedContent{
			From:      source,
			To:        target,
			EdgeType:  event.EdgeTypeChannel,
			Weight:    types.MustWeight(0),
			Direction: event.EdgeDirectionCentripetal,
			Scope:     scope,
		},
		[]types.EventID{cause}, conversationID, signer,
	)
}

// Delegate grants authority for another actor to operate as you. (Operation 12)
func (g *Grammar) Delegate(
	ctx context.Context,
	source types.ActorID,
	target types.ActorID,
	scope types.DomainScope,
	weight types.Weight,
	cause types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeEdgeCreated, source,
		event.EdgeCreatedContent{
			From:      source,
			To:        target,
			EdgeType:  event.EdgeTypeDelegation,
			Weight:    weight,
			Direction: event.EdgeDirectionCentrifugal,
			Scope:     types.Some(scope),
		},
		[]types.EventID{cause}, conversationID, signer,
	)
}

// Consent records a consent proposal signed by partyA. (Operation 13)
// LIMITATION: This is currently single-signed (partyA only). A full dual-consent
// protocol requires a two-phase flow (propose → accept) with both parties signing
// separate causally-linked events. This will be addressed in a future RFC.
func (g *Grammar) Consent(
	ctx context.Context,
	partyA types.ActorID,
	partyB types.ActorID,
	agreement string,
	scope types.DomainScope,
	cause types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.graph.Record(
		event.EventTypeGrammarConsent, partyA,
		event.GrammarConsentContent{
			Parties:   [2]types.ActorID{partyA, partyB},
			Agreement: agreement,
			Scope:     scope,
		},
		[]types.EventID{cause}, conversationID, signer,
	)
}

// Sever removes a subscription, channel, or delegation via edge supersession. (Operation 14)
// Only a party to the edge (From or To) can sever it.
func (g *Grammar) Sever(
	ctx context.Context,
	source types.ActorID,
	previousEdge types.EdgeID,
	cause types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	// Verify the edge exists and the actor is a party to it
	edgeEventID, err := types.NewEventID(previousEdge.Value())
	if err != nil {
		return event.Event{}, fmt.Errorf("sever: invalid edge ID: %w", err)
	}
	edgeEv, err := g.graph.Store().Get(edgeEventID)
	if err != nil {
		return event.Event{}, fmt.Errorf("sever: edge not found: %w", err)
	}
	ec, ok := edgeEv.Content().(event.EdgeCreatedContent)
	if !ok {
		return event.Event{}, fmt.Errorf("sever: event %s is not an edge.created event", previousEdge.Value())
	}
	if ec.From != source && ec.To != source {
		return event.Event{}, fmt.Errorf("sever: actor %s is not a party to edge %s (from=%s, to=%s)", source.Value(), previousEdge.Value(), ec.From.Value(), ec.To.Value())
	}
	// Include both the edge event and the trigger cause in the causal set.
	// The edge event is a direct causal predecessor (this event supersedes it).
	causes := []types.EventID{edgeEventID}
	if cause != edgeEventID {
		causes = append(causes, cause)
	}
	return g.graph.Record(
		event.EventTypeEdgeSuperseded, source,
		event.EdgeSupersededContent{
			PreviousEdge: previousEdge,
			NewEdge:      types.None[types.EdgeID](),
			Reason:       cause,
		},
		causes, conversationID, signer,
	)
}

// Merge joins two or more independent subtrees. (Operation 15)
func (g *Grammar) Merge(
	ctx context.Context,
	source types.ActorID,
	body string,
	sources []types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	if len(sources) < 2 {
		return event.Event{}, fmt.Errorf("merge requires at least two sources")
	}
	return g.graph.Record(
		event.EventTypeGrammarMerge, source,
		event.NewGrammarMergeContent(body, sources),
		sources, conversationID, signer,
	)
}

// --- Named functions (compositions) ---

// Recommend is Propagate + Channel: directed sharing to a specific person.
func (g *Grammar) Recommend(
	ctx context.Context,
	source types.ActorID,
	target types.EventID,
	targetActor types.ActorID,
	conversationID types.ConversationID,
	signer event.Signer,
) (propagateEv event.Event, channelEv event.Event, err error) {
	propagateEv, err = g.Propagate(ctx, source, target, targetActor, conversationID, signer)
	if err != nil {
		return event.Event{}, event.Event{}, fmt.Errorf("recommend/propagate: %w", err)
	}
	channelEv, err = g.Channel(ctx, source, targetActor, types.None[types.DomainScope](), propagateEv.ID(), conversationID, signer)
	if err != nil {
		return event.Event{}, event.Event{}, fmt.Errorf("recommend/channel: %w", err)
	}
	return propagateEv, channelEv, nil
}

// Invite is Endorse + Subscribe: trust-staked introduction of a new actor.
func (g *Grammar) Invite(
	ctx context.Context,
	source types.ActorID,
	target types.ActorID,
	weight types.Weight,
	scope types.Option[types.DomainScope],
	cause types.EventID,
	conversationID types.ConversationID,
	signer event.Signer,
) (endorseEv event.Event, subscribeEv event.Event, err error) {
	endorseEv, err = g.Endorse(ctx, source, cause, target, weight, scope, conversationID, signer)
	if err != nil {
		return event.Event{}, event.Event{}, fmt.Errorf("invite/endorse: %w", err)
	}
	subscribeEv, err = g.Subscribe(ctx, source, target, scope, endorseEv.ID(), conversationID, signer)
	if err != nil {
		return event.Event{}, event.Event{}, fmt.Errorf("invite/subscribe: %w", err)
	}
	return endorseEv, subscribeEv, nil
}

// Forgive is Subscribe after Sever: reconciliation with history intact.
func (g *Grammar) Forgive(
	ctx context.Context,
	source types.ActorID,
	severEvent types.EventID,
	target types.ActorID,
	scope types.Option[types.DomainScope],
	conversationID types.ConversationID,
	signer event.Signer,
) (event.Event, error) {
	return g.Subscribe(ctx, source, target, scope, severEvent, conversationID, signer)
}
