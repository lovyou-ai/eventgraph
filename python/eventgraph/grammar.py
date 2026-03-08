"""Social Grammar — high-level vertex operations on the event graph.

Ports the vertex operations from the Go reference implementation.
Each operation gets the current chain head, creates a properly hash-chained
event, appends it to the store, and returns the event.
"""

from __future__ import annotations

from typing import Sequence

from .event import Signer, create_event
from .store import Store
from .types import ActorID, ConversationID, EventID, EventType, Hash


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
