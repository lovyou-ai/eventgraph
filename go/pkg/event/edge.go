package event

import (
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Edge is a typed, weighted, directional relationship. Immutable after construction.
type Edge struct {
	id        types.EdgeID
	from      types.ActorID
	to        types.ActorID
	edgeType  EdgeType
	weight    types.Weight
	direction EdgeDirection
	scope     types.Option[types.DomainScope]
	metadata  EdgeMetadata
	createdAt types.Timestamp
	expiresAt types.Option[types.Timestamp]
}

// NewEdge creates a new immutable Edge. All fields are validated by their types.
func NewEdge(
	id types.EdgeID,
	from types.ActorID,
	to types.ActorID,
	edgeType EdgeType,
	weight types.Weight,
	direction EdgeDirection,
	scope types.Option[types.DomainScope],
	metadata EdgeMetadata,
	createdAt types.Timestamp,
	expiresAt types.Option[types.Timestamp],
) (Edge, error) {
	if !edgeType.IsValid() {
		return Edge{}, &types.InvalidFormatError{Field: "EdgeType", Value: string(edgeType), Expected: "valid EdgeType constant"}
	}
	if !direction.IsValid() {
		return Edge{}, &types.InvalidFormatError{Field: "EdgeDirection", Value: string(direction), Expected: "Centripetal or Centrifugal"}
	}
	return Edge{
		id:        id,
		from:      from,
		to:        to,
		edgeType:  edgeType,
		weight:    weight,
		direction: direction,
		scope:     scope,
		metadata:  metadata,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}, nil
}

func (e Edge) ID() types.EdgeID                        { return e.id }
func (e Edge) From() types.ActorID                     { return e.from }
func (e Edge) To() types.ActorID                       { return e.to }
func (e Edge) Type() EdgeType                          { return e.edgeType }
func (e Edge) Weight() types.Weight                    { return e.weight }
func (e Edge) Direction() EdgeDirection                 { return e.direction }
func (e Edge) Scope() types.Option[types.DomainScope]  { return e.scope }
func (e Edge) Metadata() EdgeMetadata                  { return e.metadata }
func (e Edge) CreatedAt() types.Timestamp                    { return e.createdAt }
func (e Edge) ExpiresAt() types.Option[types.Timestamp]      { return e.expiresAt }

// --- Edge Metadata ---

// EdgeMetadata is the interface for all typed edge metadata.
type EdgeMetadata interface {
	EdgeTypeName() EdgeType
	Accept(EdgeMetadataVisitor)
}

// EdgeMetadataVisitor provides exhaustive dispatch over edge metadata types.
type EdgeMetadataVisitor interface {
	VisitTrust(TrustEdgeMetadata)
	VisitAuthority(AuthorityEdgeMetadata)
	VisitSubscription(SubscriptionEdgeMetadata)
	VisitEndorsement(EndorsementEdgeMetadata)
	VisitDelegation(DelegationEdgeMetadata)
	VisitCausation(CausationEdgeMetadata)
	VisitReference(ReferenceEdgeMetadata)
	VisitChannel(ChannelEdgeMetadata)
	VisitAnnotation(AnnotationEdgeMetadata)
}

// TrustEdgeMetadata carries metadata for trust edges.
type TrustEdgeMetadata struct {
	Domain      types.DomainScope
	Evidence    []types.EventID
	DecayRate   types.Score
	LastUpdated types.Timestamp
}

func (m TrustEdgeMetadata) EdgeTypeName() EdgeType           { return EdgeTypeTrust }
func (m TrustEdgeMetadata) Accept(v EdgeMetadataVisitor) { v.VisitTrust(m) }

// AuthorityEdgeMetadata carries metadata for authority edges.
type AuthorityEdgeMetadata struct {
	Scope         types.DomainScope
	Delegated     bool
	DelegatedFrom types.Option[types.ActorID]
	Constraints   []string
}

func (m AuthorityEdgeMetadata) EdgeTypeName() EdgeType           { return EdgeTypeAuthority }
func (m AuthorityEdgeMetadata) Accept(v EdgeMetadataVisitor) { v.VisitAuthority(m) }

// SubscriptionEdgeMetadata carries metadata for subscription edges.
type SubscriptionEdgeMetadata struct {
	Patterns []types.SubscriptionPattern
	Muted    bool
}

func (m SubscriptionEdgeMetadata) EdgeTypeName() EdgeType           { return EdgeTypeSubscription }
func (m SubscriptionEdgeMetadata) Accept(v EdgeMetadataVisitor) { v.VisitSubscription(m) }

// EndorsementEdgeMetadata carries metadata for endorsement edges.
type EndorsementEdgeMetadata struct {
	Stake  types.Score
	Target types.EventID
	Domain types.Option[types.DomainScope]
}

func (m EndorsementEdgeMetadata) EdgeTypeName() EdgeType           { return EdgeTypeEndorsement }
func (m EndorsementEdgeMetadata) Accept(v EdgeMetadataVisitor) { v.VisitEndorsement(m) }

// DelegationEdgeMetadata carries metadata for delegation edges.
type DelegationEdgeMetadata struct {
	Scope       types.DomainScope
	Constraints []string
	RevokedBy   types.Option[types.EventID]
}

func (m DelegationEdgeMetadata) EdgeTypeName() EdgeType           { return EdgeTypeDelegation }
func (m DelegationEdgeMetadata) Accept(v EdgeMetadataVisitor) { v.VisitDelegation(m) }

// CausationEdgeMetadata carries metadata for causation edges.
type CausationEdgeMetadata struct {
	Relationship string
}

func (m CausationEdgeMetadata) EdgeTypeName() EdgeType           { return EdgeTypeCausation }
func (m CausationEdgeMetadata) Accept(v EdgeMetadataVisitor) { v.VisitCausation(m) }

// ReferenceEdgeMetadata carries metadata for reference edges (cross-graph).
type ReferenceEdgeMetadata struct {
	CGER     types.Option[CGER]
	Verified bool
}

func (m ReferenceEdgeMetadata) EdgeTypeName() EdgeType           { return EdgeTypeReference }
func (m ReferenceEdgeMetadata) Accept(v EdgeMetadataVisitor) { v.VisitReference(m) }

// ChannelEdgeMetadata carries metadata for channel edges.
type ChannelEdgeMetadata struct {
	Encrypted bool
	CreatedBy types.ActorID
}

func (m ChannelEdgeMetadata) EdgeTypeName() EdgeType           { return EdgeTypeChannel }
func (m ChannelEdgeMetadata) Accept(v EdgeMetadataVisitor) { v.VisitChannel(m) }

// AnnotationEdgeMetadata carries metadata for annotation edges.
type AnnotationEdgeMetadata struct {
	Key       string
	Value     EventContent
	Annotator types.ActorID
}

func (m AnnotationEdgeMetadata) EdgeTypeName() EdgeType           { return EdgeTypeAnnotation }
func (m AnnotationEdgeMetadata) Accept(v EdgeMetadataVisitor) { v.VisitAnnotation(m) }

// CGER represents a cross-graph event reference.
type CGER struct {
	System       types.SystemURI
	EventID      types.EventID
	Relationship CGERRelationship
}
