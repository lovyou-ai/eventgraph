"""Immutable Event with canonical form and SHA-256 hash chain."""

from __future__ import annotations

import hashlib
import json
import os
import struct
import time
from dataclasses import dataclass
from typing import Any, Protocol

from .types import (
    ActorID,
    ConversationID,
    EventID,
    EventType,
    Hash,
    NonEmpty,
    Option,
    Signature,
)


class Signer(Protocol):
    """Anything that can sign bytes."""

    def sign(self, data: bytes) -> Signature: ...


class NoopSigner:
    """Produces zero-filled signatures."""

    def sign(self, data: bytes) -> Signature:
        return Signature(b"\x00" * 64)


# ── Canonical form ────────────────────────────────────────────────────────

class _CanonicalEncoder(json.JSONEncoder):
    """Matches Go's encoding/json: integer floats without .0, omit None values."""

    def default(self, o: Any) -> Any:
        return super().default(o)

    def encode(self, o: Any) -> str:
        if isinstance(o, dict):
            # Omit None values, sort keys
            filtered = {k: v for k, v in sorted(o.items()) if v is not None}
            return "{" + ",".join(
                json.dumps(k) + ":" + self.encode(v) for k, v in filtered.items()
            ) + "}"
        if isinstance(o, float):
            if o == int(o) and not (o != o):  # not NaN
                return str(int(o))
            return repr(o)
        if isinstance(o, list):
            return "[" + ",".join(self.encode(v) for v in o) + "]"
        return json.dumps(o, ensure_ascii=False)


def canonical_content_json(content: dict[str, Any]) -> str:
    """Produce canonical JSON: sorted keys, no whitespace, integer floats without .0, omit None."""
    return _CanonicalEncoder().encode(content)


def canonical_form(
    version: int,
    prev_hash: str,
    causes: list[str],
    event_id: str,
    event_type: str,
    source: str,
    conversation_id: str,
    timestamp_nanos: int,
    content_json: str,
) -> str:
    """Build the canonical string for hashing/signing.

    Format: version|prev_hash|causes|id|type|source|conversation_id|timestamp_nanos|content_json
    Causes are sorted lexicographically, comma-separated. Empty for bootstrap.
    """
    sorted_causes = ",".join(sorted(causes))
    return (
        f"{version}|{prev_hash}|{sorted_causes}|{event_id}|{event_type}"
        f"|{source}|{conversation_id}|{timestamp_nanos}|{content_json}"
    )


def compute_hash(canonical: str) -> Hash:
    """SHA-256 of the canonical form."""
    digest = hashlib.sha256(canonical.encode("utf-8")).hexdigest()
    return Hash(digest)


# ── Event ─────────────────────────────────────────────────────────────────

@dataclass(frozen=True, slots=True)
class Event:
    """Immutable event — validated at construction, guaranteed valid for lifetime."""

    _version: int
    _id: EventID
    _type: EventType
    _timestamp_nanos: int
    _source: ActorID
    _content: dict[str, Any]
    _causes: NonEmpty[EventID]
    _conversation_id: ConversationID
    _hash: Hash
    _prev_hash: Hash
    _signature: Signature

    @property
    def version(self) -> int:
        return self._version

    @property
    def id(self) -> EventID:
        return self._id

    @property
    def type(self) -> EventType:
        return self._type

    @property
    def timestamp_nanos(self) -> int:
        return self._timestamp_nanos

    @property
    def source(self) -> ActorID:
        return self._source

    @property
    def content(self) -> dict[str, Any]:
        return dict(self._content)  # defensive copy

    @property
    def causes(self) -> NonEmpty[EventID]:
        return self._causes

    @property
    def conversation_id(self) -> ConversationID:
        return self._conversation_id

    @property
    def hash(self) -> Hash:
        return self._hash

    @property
    def prev_hash(self) -> Hash:
        return self._prev_hash

    @property
    def signature(self) -> Signature:
        return self._signature


# ── UUID v7 generation ────────────────────────────────────────────────────

def new_event_id() -> EventID:
    """Generate a new UUID v7 EventID using the current time."""
    ms = int(time.time() * 1000)
    b = bytearray(16)
    # Timestamp: 48 bits (6 bytes)
    struct.pack_into(">Q", b, 0, ms)
    # The first 2 bytes are the high bytes of the 8-byte pack; shift down
    b[0:6] = struct.pack(">Q", ms)[2:8]
    # Random: fill remaining bytes
    rand_bytes = os.urandom(10)
    b[6:16] = rand_bytes
    # Version: set high nibble of byte 6 to 0x7
    b[6] = (b[6] & 0x0F) | 0x70
    # Variant: set high bits of byte 8 to 10xx
    b[8] = (b[8] & 0x3F) | 0x80

    s = (
        f"{b[0]:02x}{b[1]:02x}{b[2]:02x}{b[3]:02x}-"
        f"{b[4]:02x}{b[5]:02x}-"
        f"{b[6]:02x}{b[7]:02x}-"
        f"{b[8]:02x}{b[9]:02x}-"
        f"{b[10]:02x}{b[11]:02x}{b[12]:02x}{b[13]:02x}{b[14]:02x}{b[15]:02x}"
    )
    return EventID(s)


# ── EventFactory ──────────────────────────────────────────────────────────

def create_event(
    event_type: EventType,
    source: ActorID,
    content: dict[str, Any],
    causes: list[EventID],
    conversation_id: ConversationID,
    prev_hash: Hash,
    signer: Signer,
    version: int = 1,
) -> Event:
    """Create, hash, and sign an event. The only way to construct an Event."""
    event_id = new_event_id()
    timestamp_nanos = int(time.time() * 1_000_000_000)
    content_json = canonical_content_json(content)

    canon = canonical_form(
        version=version,
        prev_hash=prev_hash.value,
        causes=[c.value for c in causes],
        event_id=event_id.value,
        event_type=event_type.value,
        source=source.value,
        conversation_id=conversation_id.value,
        timestamp_nanos=timestamp_nanos,
        content_json=content_json,
    )

    event_hash = compute_hash(canon)
    sig = signer.sign(canon.encode("utf-8"))

    return Event(
        _version=version,
        _id=event_id,
        _type=event_type,
        _timestamp_nanos=timestamp_nanos,
        _source=source,
        _content=content,
        _causes=NonEmpty.of(causes),
        _conversation_id=conversation_id,
        _hash=event_hash,
        _prev_hash=prev_hash,
        _signature=sig,
    )


def create_bootstrap(
    source: ActorID,
    signer: Signer,
    version: int = 1,
) -> Event:
    """Create the genesis/bootstrap event — no causes, empty prev_hash."""
    event_id = new_event_id()
    timestamp_nanos = int(time.time() * 1_000_000_000)
    conversation_id = ConversationID(f"conv_{source.value}")

    content = {
        "ActorID": source.value,
        "ChainGenesis": Hash.zero().value,
        "Timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    }
    content_json = canonical_content_json(content)

    canon = canonical_form(
        version=version,
        prev_hash="",
        causes=[],
        event_id=event_id.value,
        event_type="system.bootstrapped",
        source=source.value,
        conversation_id=conversation_id.value,
        timestamp_nanos=timestamp_nanos,
        content_json=content_json,
    )

    event_hash = compute_hash(canon)
    sig = signer.sign(canon.encode("utf-8"))

    return Event(
        _version=version,
        _id=event_id,
        _type=EventType("system.bootstrapped"),
        _timestamp_nanos=timestamp_nanos,
        _source=source,
        _content=content,
        _causes=NonEmpty.of([event_id]),  # bootstrap self-references
        _conversation_id=conversation_id,
        _hash=event_hash,
        _prev_hash=Hash.zero(),
        _signature=sig,
    )
