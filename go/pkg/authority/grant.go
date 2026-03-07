package authority

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// grantAndPersist creates an authority edge and persists it to the store
// as an edge.created event. Used by both DefaultAuthorityChain and DelegationChain.
func grantAndPersist(s store.Store, factory *event.EventFactory, signer event.Signer, from actor.IActor, to actor.IActor, scope types.DomainScope, weight types.Score) (event.Edge, error) {
	// Convert Score [0,1] → Weight [-1,1] once for both content and edge.
	// Clamp to handle floating-point precision at boundaries.
	raw := weight.Value()*2 - 1
	if raw > 1.0 {
		raw = 1.0
	} else if raw < -1.0 {
		raw = -1.0
	}
	edgeWeight := types.MustWeight(raw)

	// Build the edge content
	content := event.EdgeCreatedContent{
		From:      from.ID(),
		To:        to.ID(),
		EdgeType:  event.EdgeTypeAuthority,
		Weight:    edgeWeight,
		Direction: event.EdgeDirectionCentrifugal,
		Scope:     types.Some(scope),
	}

	// Need a cause — use head of chain
	headOpt, err := s.Head()
	if err != nil {
		return event.Edge{}, err
	}
	if !headOpt.IsSome() {
		return event.Edge{}, fmt.Errorf("cannot grant authority: store has no head event (causality invariant)")
	}
	causes := []types.EventID{headOpt.Unwrap().ID()}

	convID, err := newGrantConversationID()
	if err != nil {
		return event.Edge{}, fmt.Errorf("generate grant conversation ID: %w", err)
	}

	ev, err := factory.Create(
		event.EventTypeEdgeCreated, from.ID(), content, causes,
		convID, s, signer,
	)
	if err != nil {
		return event.Edge{}, err
	}

	stored, err := s.Append(ev)
	if err != nil {
		return event.Edge{}, err
	}

	// Build the edge from the stored event
	edgeID := types.MustEdgeID(stored.ID().Value())
	edge, err := event.NewEdge(
		edgeID, from.ID(), to.ID(),
		event.EdgeTypeAuthority,
		edgeWeight,
		event.EdgeDirectionCentrifugal,
		types.Some(scope),
		nil,
		stored.Timestamp(),
		types.None[types.Timestamp](),
	)
	if err != nil {
		return event.Edge{}, err
	}

	return edge, nil
}

// newGrantConversationID generates a unique conversation ID for a grant operation.
func newGrantConversationID() (types.ConversationID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return types.ConversationID{}, err
	}
	return types.NewConversationID("conv_grant_" + hex.EncodeToString(b[:]))
}
