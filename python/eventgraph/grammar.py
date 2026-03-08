"""Social Grammar — the 15 social grammar operations + 4 named functions.

Ports the full grammar from the Go reference implementation.
Each operation gets the current chain head, creates a properly hash-chained
event, appends it to the store, and returns the event.

Operations 1-7: Vertex operations (Emit, Respond, Derive, Extend, Retract, Annotate, Merge)
Operations 8-15: Edge operations (Acknowledge, Propagate, Endorse, Subscribe, Channel, Delegate, Consent, Sever)
Named functions: Challenge, Recommend, Invite, Forgive
"""

from __future__ import annotations

from typing import Sequence

from .event import Event, Signer, create_event
from .store import Store
from .types import (
    ActorID,
    ConversationID,
    DomainScope,
    EdgeID,
    EventID,
    EventType,
    Hash,
    Option,
    Weight,
)


class Grammar:
    """Wraps a Store and provides high-level social grammar operations.

    Each method creates a properly hash-chained event with the right type,
    content, and causal links, then appends it to the store.
    """

    def __init__(self, store: Store) -> None:
        self._store = store

    def _prev_hash(self) -> Hash:
        """Get the prev_hash for the next event in the chain."""
        head = self._store.head()
        return head.unwrap().hash if head.is_some() else Hash.zero()

    def emit(
        self,
        source: ActorID,
        body: str,
        conversation_id: ConversationID,
        causes: list[EventID],
        signer: Signer,
    ) -> "Event":
        """Emit a new statement into the graph.

        Causes must be non-empty (enforced by create_event / NonEmpty).
        """
        if not causes:
            raise ValueError("emit requires at least one cause")
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("grammar.emitted"),
            source=source,
            content={"Body": body},
            causes=causes,
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def respond(
        self,
        source: ActorID,
        body: str,
        parent: EventID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> "Event":
        """Respond to an existing event."""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("grammar.responded"),
            source=source,
            content={"Body": body, "Parent": parent.value},
            causes=[parent],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def derive(
        self,
        source: ActorID,
        body: str,
        source_event: EventID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> "Event":
        """Derive a new insight from an existing event."""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("grammar.derived"),
            source=source,
            content={"Body": body, "Source": source_event.value},
            causes=[source_event],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def extend(
        self,
        source: ActorID,
        body: str,
        previous: EventID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> "Event":
        """Extend an existing event with additional content."""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("grammar.extended"),
            source=source,
            content={"Body": body, "Previous": previous.value},
            causes=[previous],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def retract(
        self,
        source: ActorID,
        target: EventID,
        reason: str,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> "Event":
        """Retract a previously emitted event.

        The source must be the original author of the target event.
        Raises ValueError if the source did not author the target.
        """
        target_event = self._store.get(target)
        if target_event.source.value != source.value:
            raise ValueError(
                f"actor {source.value} cannot retract event authored by {target_event.source.value}"
            )
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("grammar.retracted"),
            source=source,
            content={"Target": target.value, "Reason": reason},
            causes=[target],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def annotate(
        self,
        source: ActorID,
        target: EventID,
        key: str,
        value: str,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> "Event":
        """Annotate an existing event with a key-value pair."""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("grammar.annotated"),
            source=source,
            content={"Target": target.value, "Key": key, "Value": value},
            causes=[target],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def merge(
        self,
        source: ActorID,
        body: str,
        sources: Sequence[EventID],
        conversation_id: ConversationID,
        signer: Signer,
    ) -> "Event":
        """Merge multiple events into a synthesis.

        Requires at least 2 source events.
        """
        if len(sources) < 2:
            raise ValueError("merge requires at least 2 source events")
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("grammar.merged"),
            source=source,
            content={"Body": body, "Sources": [s.value for s in sources]},
            causes=list(sources),
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    # --- Edge operations ---

    def _edge_content(
        self,
        from_actor: ActorID,
        to_actor: ActorID,
        edge_type: str,
        weight: Weight,
        direction: str,
        scope: Option[DomainScope],
    ) -> dict:
        """Build an edge.created content dict."""
        content: dict = {
            "Direction": direction,
            "EdgeType": edge_type,
            "From": from_actor.value,
            "To": to_actor.value,
            "Weight": weight.value,
        }
        if scope.is_some():
            content["Scope"] = scope.unwrap().value
        return content

    def acknowledge(
        self,
        source: ActorID,
        target_event: EventID,
        target_actor: ActorID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> Event:
        """Create a content-free centripetal edge toward a vertex. (Operation 7)"""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("edge.created"),
            source=source,
            content=self._edge_content(
                source, target_actor, "acknowledgement",
                Weight(0), "centripetal", Option.none(),
            ),
            causes=[target_event],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def propagate(
        self,
        source: ActorID,
        target_event: EventID,
        target_actor: ActorID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> Event:
        """Redistribute a vertex into the actor's subgraph. (Operation 8)"""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("edge.created"),
            source=source,
            content=self._edge_content(
                source, target_actor, "reference",
                Weight(0), "centrifugal", Option.none(),
            ),
            causes=[target_event],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def endorse(
        self,
        source: ActorID,
        target_event: EventID,
        target_actor: ActorID,
        weight: Weight,
        scope: Option[DomainScope],
        conversation_id: ConversationID,
        signer: Signer,
    ) -> Event:
        """Create a reputation-staked edge toward a vertex. (Operation 9)"""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("edge.created"),
            source=source,
            content=self._edge_content(
                source, target_actor, "endorsement",
                weight, "centripetal", scope,
            ),
            causes=[target_event],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def subscribe(
        self,
        source: ActorID,
        target_actor: ActorID,
        scope: Option[DomainScope],
        cause: EventID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> Event:
        """Create a persistent, future-oriented edge to an actor. (Operation 10)"""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("edge.created"),
            source=source,
            content=self._edge_content(
                source, target_actor, "subscription",
                Weight(0), "centripetal", scope,
            ),
            causes=[cause],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def channel(
        self,
        source: ActorID,
        target_actor: ActorID,
        scope: Option[DomainScope],
        cause: EventID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> Event:
        """Create a private, bidirectional, content-bearing edge. (Operation 11)"""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("edge.created"),
            source=source,
            content=self._edge_content(
                source, target_actor, "channel",
                Weight(0), "centripetal", scope,
            ),
            causes=[cause],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def delegate(
        self,
        source: ActorID,
        target_actor: ActorID,
        scope: DomainScope,
        weight: Weight,
        cause: EventID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> Event:
        """Grant authority for another actor to operate as you. (Operation 12)"""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("edge.created"),
            source=source,
            content=self._edge_content(
                source, target_actor, "delegation",
                weight, "centrifugal", Option.some(scope),
            ),
            causes=[cause],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def consent(
        self,
        party_a: ActorID,
        party_b: ActorID,
        agreement: str,
        scope: DomainScope,
        cause: EventID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> Event:
        """Record a consent proposal signed by party_a. (Operation 13)"""
        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("grammar.consent"),
            source=party_a,
            content={
                "Agreement": agreement,
                "PartyA": party_a.value,
                "PartyB": party_b.value,
                "Scope": scope.value,
            },
            causes=[cause],
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    def sever(
        self,
        source: ActorID,
        previous_edge_id: EdgeID,
        cause: EventID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> Event:
        """Remove a subscription, channel, or delegation via edge supersession. (Operation 14)

        Only a party to the edge (From or To) can sever it.
        Only subscription, channel, and delegation edges are severable.
        """
        # Convert EdgeID to EventID for store lookup
        edge_event_id = EventID(previous_edge_id.value)

        # Verify the edge exists
        edge_event = self._store.get(edge_event_id)

        # Verify it is an edge.created event
        if edge_event.type.value != "edge.created":
            raise ValueError(
                f"sever: event {previous_edge_id.value} is not an edge.created event"
            )

        # Verify edge type is severable
        edge_type = edge_event.content.get("EdgeType", "")
        if edge_type not in ("subscription", "channel", "delegation"):
            raise ValueError(
                f"sever: edge type {edge_type} is not severable "
                "(only subscription, channel, delegation)"
            )

        # Verify actor is a party to the edge
        edge_from = edge_event.content.get("From", "")
        edge_to = edge_event.content.get("To", "")
        if source.value != edge_from and source.value != edge_to:
            raise ValueError(
                f"sever: actor {source.value} is not a party to edge "
                f"{previous_edge_id.value} (from={edge_from}, to={edge_to})"
            )

        # Build causes: include both the edge event and the trigger cause
        causes = [edge_event_id]
        if cause.value != edge_event_id.value:
            causes.append(cause)

        prev_hash = self._prev_hash()
        return self._store.append(create_event(
            event_type=EventType("edge.superseded"),
            source=source,
            content={
                "NewEdge": None,
                "PreviousEdge": previous_edge_id.value,
                "Reason": cause.value,
            },
            causes=causes,
            conversation_id=conversation_id,
            prev_hash=prev_hash,
            signer=signer,
        ))

    # --- Named functions (compositions of operations) ---

    def challenge(
        self,
        source: ActorID,
        body: str,
        target: EventID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> tuple[Event, Event]:
        """Respond + dispute flag: formal dispute that follows content.

        Returns (response, dispute_flag).
        """
        response = self.respond(source, body, target, conversation_id, signer)
        dispute_flag = self.annotate(
            source, response.id, "dispute", "challenged",
            conversation_id, signer,
        )
        return response, dispute_flag

    def recommend(
        self,
        source: ActorID,
        target: EventID,
        target_actor: ActorID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> tuple[Event, Event]:
        """Propagate + Channel: directed sharing to a specific person.

        Returns (propagate_event, channel_event).
        """
        propagate_ev = self.propagate(
            source, target, target_actor, conversation_id, signer,
        )
        channel_ev = self.channel(
            source, target_actor, Option.none(),
            propagate_ev.id, conversation_id, signer,
        )
        return propagate_ev, channel_ev

    def invite(
        self,
        source: ActorID,
        target: ActorID,
        weight: Weight,
        scope: Option[DomainScope],
        cause: EventID,
        conversation_id: ConversationID,
        signer: Signer,
    ) -> tuple[Event, Event]:
        """Endorse + Subscribe: trust-staked introduction of a new actor.

        Returns (endorse_event, subscribe_event).
        """
        endorse_ev = self.endorse(
            source, cause, target, weight, scope,
            conversation_id, signer,
        )
        subscribe_ev = self.subscribe(
            source, target, scope, endorse_ev.id,
            conversation_id, signer,
        )
        return endorse_ev, subscribe_ev

    def forgive(
        self,
        source: ActorID,
        sever_event: EventID,
        target: ActorID,
        scope: Option[DomainScope],
        conversation_id: ConversationID,
        signer: Signer,
    ) -> Event:
        """Subscribe after Sever: reconciliation with history intact."""
        return self.subscribe(
            source, target, scope, sever_event,
            conversation_id, signer,
        )
