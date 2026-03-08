"""EGIP — EventGraph Inter-system Protocol.

Ported from the Go reference implementation. Provides:
- System identity (IIdentity protocol, HMAC-based test implementation)
- Signed envelope transport with canonical form
- HELLO handshake and message dispatch
- Treaty state machine with bilateral governance
- Peer trust management with decay
- Replay protection (EnvelopeDedup)
- Proof generation and verification
- Pluggable transport (ITransport protocol)
"""

from __future__ import annotations

import hashlib
import hmac
import json
import threading
import time
import uuid
from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Callable, Protocol, runtime_checkable

from .errors import EventGraphError, InvalidTransitionError
from .types import (
    ActorID,
    DomainScope,
    EnvelopeID,
    EventID,
    EventType,
    Hash,
    Option,
    PublicKey,
    Score,
    Signature,
    SystemURI,
    TreatyID,
    Weight,
)

# ── EGIP Constants ─────────────────────────────────────────────────────────

CURRENT_PROTOCOL_VERSION = 1
MAX_ENVELOPE_AGE_SECONDS = 25 * 3600  # 25 hours
DEDUP_PRUNE_INTERVAL = 1000


class MessageType(str, Enum):
    """EGIP message types."""
    HELLO = "Hello"
    MESSAGE = "Message"
    RECEIPT = "Receipt"
    PROOF = "Proof"
    TREATY = "Treaty"
    AUTHORITY_REQUEST = "AuthorityRequest"
    DISCOVER = "Discover"


class TreatyStatus(str, Enum):
    """Treaty lifecycle states."""
    PROPOSED = "Proposed"
    ACTIVE = "Active"
    SUSPENDED = "Suspended"
    TERMINATED = "Terminated"


class TreatyAction(str, Enum):
    """Treaty actions that can be applied."""
    PROPOSE = "Propose"
    ACCEPT = "Accept"
    MODIFY = "Modify"
    SUSPEND = "Suspend"
    TERMINATE = "Terminate"


class ReceiptStatus(str, Enum):
    """EGIP receipt statuses."""
    DELIVERED = "Delivered"
    PROCESSED = "Processed"
    REJECTED = "Rejected"


class ProofType(str, Enum):
    """EGIP proof types."""
    CHAIN_SEGMENT = "ChainSegment"
    EVENT_EXISTENCE = "EventExistence"
    CHAIN_SUMMARY = "ChainSummary"


class CGERRelationship(str, Enum):
    """Cross-graph event relationship types."""
    CAUSED_BY = "CausedBy"
    REFERENCES = "References"
    RESPONDS_TO = "RespondsTo"


class AuthorityLevel(str, Enum):
    """Authority levels for action approval."""
    REQUIRED = "Required"
    RECOMMENDED = "Recommended"
    NOTIFICATION = "Notification"


# ── EGIP Errors ────────────────────────────────────────────────────────────


class EGIPError(EventGraphError):
    """Base error for all EGIP protocol errors."""


class SystemNotFoundError(EGIPError):
    """The target system could not be reached."""

    def __init__(self, uri: SystemURI) -> None:
        self.uri = uri
        super().__init__(f"system not found: {uri.value}")


class EnvelopeSignatureInvalidError(EGIPError):
    """An envelope's signature failed verification."""

    def __init__(self, envelope_id: EnvelopeID) -> None:
        self.envelope_id = envelope_id
        super().__init__(f"envelope signature invalid: {envelope_id.value}")


class TreatyViolationError(EGIPError):
    """A treaty term was violated."""

    def __init__(self, treaty_id: TreatyID, term: str) -> None:
        self.treaty_id = treaty_id
        self.term = term
        super().__init__(f"treaty {treaty_id.value} violated: {term}")


class TrustInsufficientError(EGIPError):
    """A system's trust score is too low."""

    def __init__(self, system: SystemURI, score: Score, required: Score) -> None:
        self.system = system
        self.score = score
        self.required = required
        super().__init__(
            f"trust insufficient for {system.value}: have {score.value}, need {required.value}"
        )


class TransportFailureError(EGIPError):
    """A transport-level failure (retryable)."""

    def __init__(self, to: SystemURI, reason: str) -> None:
        self.to = to
        self.reason = reason
        super().__init__(f"transport failure to {to.value}: {reason}")


class DuplicateEnvelopeError(EGIPError):
    """A replay -- an envelope with this ID was already processed."""

    def __init__(self, envelope_id: EnvelopeID) -> None:
        self.envelope_id = envelope_id
        super().__init__(f"duplicate envelope: {envelope_id.value}")


class TreatyNotFoundError(EGIPError):
    """The referenced treaty does not exist."""

    def __init__(self, treaty_id: TreatyID) -> None:
        self.treaty_id = treaty_id
        super().__init__(f"treaty not found: {treaty_id.value}")


class VersionIncompatibleError(EGIPError):
    """No common protocol version exists."""

    def __init__(self, local: list[int], remote: list[int]) -> None:
        self.local = local
        self.remote = remote
        super().__init__(
            f"no compatible protocol version: local {local}, remote {remote}"
        )


# ── Identity ───────────────────────────────────────────────────────────────


@runtime_checkable
class IIdentity(Protocol):
    """A system's cryptographic identity."""

    def system_uri(self) -> SystemURI: ...
    def public_key(self) -> PublicKey: ...
    def sign(self, data: bytes) -> Signature: ...
    def verify(self, public_key: PublicKey, data: bytes, signature: Signature) -> bool: ...


class SystemIdentity:
    """HMAC-SHA256-based identity for environments without Ed25519 libraries.

    Uses a 32-byte secret key and HMAC-SHA256 truncated to 64 bytes for
    signatures. This is NOT cryptographically equivalent to Ed25519 --
    it is a test/development implementation. For production, use an
    Ed25519-backed IIdentity implementation with the `cryptography` or
    `nacl` library.

    A module-level registry maps public keys to private keys so that
    cross-system signature verification works (any SystemIdentity can
    verify signatures produced by any other SystemIdentity in the same
    process). This is acceptable because the HMAC scheme is already
    test/dev only.
    """

    # Registry: public_key bytes -> private_key bytes
    _key_registry: dict[bytes, bytes] = {}

    def __init__(self, uri: SystemURI, public_key: PublicKey, private_key: bytes) -> None:
        self._uri = uri
        self._public_key = public_key
        self._private_key = private_key
        self._created_at = time.time()
        # Register so any identity can verify our signatures
        SystemIdentity._key_registry[public_key.bytes_] = private_key

    @staticmethod
    def generate(uri: SystemURI) -> SystemIdentity:
        """Generate a new identity with a random keypair."""
        private_key = uuid.uuid4().bytes + uuid.uuid4().bytes  # 32 bytes
        # Derive a "public key" from the private key via SHA-256
        pub_bytes = hashlib.sha256(private_key).digest()
        public_key = PublicKey(pub_bytes)
        return SystemIdentity(uri, public_key, private_key)

    def system_uri(self) -> SystemURI:
        return self._uri

    def public_key(self) -> PublicKey:
        return self._public_key

    @property
    def created_at(self) -> float:
        return self._created_at

    def sign(self, data: bytes) -> Signature:
        """Sign data using HMAC-SHA256, padded to 64 bytes."""
        mac = hmac.new(self._private_key, data, hashlib.sha256).digest()
        # Pad to 64 bytes (Ed25519 signature size) by repeating the HMAC
        sig_bytes = mac + mac  # 32 + 32 = 64
        return Signature(sig_bytes)

    @staticmethod
    def _sign_with_key(private_key: bytes, data: bytes) -> Signature:
        """Compute an HMAC-SHA256 signature using the given private key."""
        mac = hmac.new(private_key, data, hashlib.sha256).digest()
        return Signature(mac + mac)

    def verify(self, public_key: PublicKey, data: bytes, signature: Signature) -> bool:
        """Verify a signature.

        Looks up the private key associated with the given public key in
        the process-wide registry, then recomputes the HMAC to check the
        signature. Falls back to checking against our own key if the
        public key is not in the registry (self-verification).
        """
        try:
            priv = SystemIdentity._key_registry.get(public_key.bytes_)
            if priv is not None:
                expected = SystemIdentity._sign_with_key(priv, data)
            else:
                # Fallback: assume it's our own key
                expected = self.sign(data)
            return hmac.compare_digest(expected.bytes_, signature.bytes_)
        except Exception:
            return False


# ── Payload Types ──────────────────────────────────────────────────────────


@dataclass(frozen=True)
class CGER:
    """Cross-graph event reference with verification tracking."""
    local_event_id: str
    remote_system: str
    remote_event_id: str
    remote_hash: str
    relationship: CGERRelationship
    verified: bool = False


@dataclass(frozen=True)
class HelloPayload:
    """Payload for HELLO messages."""
    system_uri: str
    public_key: bytes
    protocol_versions: list[int]
    capabilities: list[str]
    chain_length: int


@dataclass(frozen=True)
class MessagePayloadContent:
    """Payload for MESSAGE messages."""
    content: dict[str, Any]
    content_type: str
    conversation_id: Option[str] = field(default_factory=Option.none)
    cgers: list[CGER] = field(default_factory=list)


@dataclass(frozen=True)
class ReceiptPayload:
    """Payload for RECEIPT messages."""
    envelope_id: str
    status: ReceiptStatus
    local_event_id: Option[str] = field(default_factory=Option.none)
    reason: Option[str] = field(default_factory=Option.none)
    signature: bytes = b"\x00" * 64


@dataclass(frozen=True)
class DiscoverQuery:
    """What capabilities to search for."""
    capabilities: list[str]
    min_trust: Option[float] = field(default_factory=Option.none)


@dataclass(frozen=True)
class DiscoverResult:
    """A single discovery result."""
    system_uri: str
    public_key: bytes
    capabilities: list[str]
    trust_score: float


@dataclass(frozen=True)
class DiscoverPayload:
    """Payload for DISCOVER messages."""
    query: DiscoverQuery
    results: list[DiscoverResult] = field(default_factory=list)


@dataclass(frozen=True)
class TreatyTerm:
    """A single term of a bilateral treaty."""
    scope: str
    policy: str
    symmetric: bool = False


@dataclass(frozen=True)
class TreatyPayload:
    """Payload for TREATY messages."""
    treaty_id: str
    action: TreatyAction
    terms: list[TreatyTerm] = field(default_factory=list)
    reason: Option[str] = field(default_factory=Option.none)


@dataclass(frozen=True)
class AuthorityRequestPayload:
    """Payload for AUTHORITY_REQUEST messages."""
    action: str
    actor: str
    level: AuthorityLevel
    justification: str
    treaty_id: Option[str] = field(default_factory=Option.none)


@dataclass(frozen=True)
class ChainSegmentProof:
    """A contiguous portion of the hash chain."""
    events: list[dict[str, Any]]
    start_hash: str
    end_hash: str


@dataclass(frozen=True)
class EventExistenceProof:
    """Proves a specific event exists in the chain."""
    event: dict[str, Any]
    prev_hash: str
    next_hash: Option[str] = field(default_factory=Option.none)
    position: int = 0
    chain_length: int = 0


@dataclass(frozen=True)
class ChainSummaryProof:
    """A high-level integrity attestation."""
    length: int
    head_hash: str
    genesis_hash: str
    timestamp: float = 0.0


@dataclass(frozen=True)
class ProofPayload:
    """Payload for PROOF messages."""
    proof_type: ProofType
    data: ChainSegmentProof | EventExistenceProof | ChainSummaryProof


# ── Envelope ───────────────────────────────────────────────────────────────

# Union type for all payload types
PayloadType = (
    HelloPayload
    | MessagePayloadContent
    | ReceiptPayload
    | ProofPayload
    | TreatyPayload
    | AuthorityRequestPayload
    | DiscoverPayload
)


def _payload_to_dict(payload: PayloadType) -> dict[str, Any]:
    """Convert a payload dataclass to a JSON-serializable dict."""
    if isinstance(payload, HelloPayload):
        return {
            "system_uri": payload.system_uri,
            "public_key": payload.public_key.hex() if isinstance(payload.public_key, bytes) else str(payload.public_key),
            "protocol_versions": list(payload.protocol_versions),
            "capabilities": list(payload.capabilities),
            "chain_length": payload.chain_length,
        }
    elif isinstance(payload, MessagePayloadContent):
        d: dict[str, Any] = {
            "content": payload.content,
            "content_type": payload.content_type,
        }
        if payload.conversation_id.is_some():
            d["conversation_id"] = payload.conversation_id.unwrap()
        if payload.cgers:
            d["cgers"] = [
                {
                    "local_event_id": c.local_event_id,
                    "remote_system": c.remote_system,
                    "remote_event_id": c.remote_event_id,
                    "remote_hash": c.remote_hash,
                    "relationship": c.relationship.value,
                    "verified": c.verified,
                }
                for c in payload.cgers
            ]
        return d
    elif isinstance(payload, ReceiptPayload):
        d = {
            "envelope_id": payload.envelope_id,
            "status": payload.status.value,
        }
        if payload.local_event_id.is_some():
            d["local_event_id"] = payload.local_event_id.unwrap()
        if payload.reason.is_some():
            d["reason"] = payload.reason.unwrap()
        return d
    elif isinstance(payload, ProofPayload):
        proof_data: dict[str, Any]
        if isinstance(payload.data, ChainSegmentProof):
            proof_data = {
                "events": payload.data.events,
                "start_hash": payload.data.start_hash,
                "end_hash": payload.data.end_hash,
            }
        elif isinstance(payload.data, EventExistenceProof):
            proof_data = {
                "event": payload.data.event,
                "prev_hash": payload.data.prev_hash,
                "position": payload.data.position,
                "chain_length": payload.data.chain_length,
            }
            if payload.data.next_hash.is_some():
                proof_data["next_hash"] = payload.data.next_hash.unwrap()
        elif isinstance(payload.data, ChainSummaryProof):
            proof_data = {
                "length": payload.data.length,
                "head_hash": payload.data.head_hash,
                "genesis_hash": payload.data.genesis_hash,
                "timestamp": payload.data.timestamp,
            }
        else:
            proof_data = {}
        return {
            "proof_type": payload.proof_type.value,
            "data": proof_data,
        }
    elif isinstance(payload, TreatyPayload):
        d = {
            "treaty_id": payload.treaty_id,
            "action": payload.action.value,
        }
        if payload.terms:
            d["terms"] = [
                {"scope": t.scope, "policy": t.policy, "symmetric": t.symmetric}
                for t in payload.terms
            ]
        if payload.reason.is_some():
            d["reason"] = payload.reason.unwrap()
        return d
    elif isinstance(payload, AuthorityRequestPayload):
        d = {
            "action": payload.action,
            "actor": payload.actor,
            "level": payload.level.value,
            "justification": payload.justification,
        }
        if payload.treaty_id.is_some():
            d["treaty_id"] = payload.treaty_id.unwrap()
        return d
    elif isinstance(payload, DiscoverPayload):
        d: dict[str, Any] = {
            "query": {
                "capabilities": list(payload.query.capabilities),
            },
        }
        if payload.query.min_trust.is_some():
            d["query"]["min_trust"] = payload.query.min_trust.unwrap()
        if payload.results:
            d["results"] = [
                {
                    "system_uri": r.system_uri,
                    "public_key": r.public_key.hex() if isinstance(r.public_key, bytes) else str(r.public_key),
                    "capabilities": list(r.capabilities),
                    "trust_score": r.trust_score,
                }
                for r in payload.results
            ]
        return d
    else:
        raise ValueError(f"Unknown payload type: {type(payload)}")


@dataclass
class Envelope:
    """Signed message container for all EGIP communication."""
    protocol_version: int
    id: EnvelopeID
    from_uri: SystemURI
    to_uri: SystemURI
    type: MessageType
    payload: PayloadType
    timestamp: float  # Unix timestamp (seconds)
    signature: Signature = field(default_factory=lambda: Signature(b"\x00" * 64))
    in_reply_to: Option[EnvelopeID] = field(default_factory=Option.none)

    def canonical_form(self) -> str:
        """Return the canonical string representation for signing."""
        payload_dict = _payload_to_dict(self.payload)
        # Round-trip through JSON for deterministic key ordering
        canonical_json = json.dumps(payload_dict, sort_keys=True, separators=(",", ":"))

        msg_type = self.type.value.lower()
        nanos = str(int(self.timestamp * 1_000_000_000))

        in_reply_to = ""
        if self.in_reply_to.is_some():
            in_reply_to = self.in_reply_to.unwrap().value

        return (
            f"{self.protocol_version}|{self.id.value}|{self.from_uri.value}"
            f"|{self.to_uri.value}|{msg_type}|{nanos}|{in_reply_to}"
            f"|{canonical_json}"
        )


def sign_envelope(env: Envelope, identity: IIdentity) -> Envelope:
    """Sign an envelope and return a new envelope with the signature set."""
    canonical = env.canonical_form()
    sig = identity.sign(canonical.encode("utf-8"))
    return Envelope(
        protocol_version=env.protocol_version,
        id=env.id,
        from_uri=env.from_uri,
        to_uri=env.to_uri,
        type=env.type,
        payload=env.payload,
        timestamp=env.timestamp,
        signature=sig,
        in_reply_to=env.in_reply_to,
    )


def verify_envelope(env: Envelope, identity: IIdentity, public_key: PublicKey) -> bool:
    """Verify an envelope's signature against a public key."""
    canonical = env.canonical_form()
    return identity.verify(public_key, canonical.encode("utf-8"), env.signature)


# ── Version Negotiation ───────────────────────────────────────────────────


def negotiate_version(local: list[int], remote: list[int]) -> Option[int]:
    """Find the highest protocol version both systems support.

    Returns Option.none() if no common version exists.
    """
    common = set(local) & set(remote)
    if not common:
        return Option.none()
    return Option.some(max(common))


# ── Treaty ─────────────────────────────────────────────────────────────────

# Valid treaty status transitions
_VALID_TREATY_TRANSITIONS: dict[TreatyStatus, list[TreatyStatus]] = {
    TreatyStatus.PROPOSED: [TreatyStatus.ACTIVE, TreatyStatus.TERMINATED],
    TreatyStatus.ACTIVE: [TreatyStatus.SUSPENDED, TreatyStatus.TERMINATED],
    TreatyStatus.SUSPENDED: [TreatyStatus.ACTIVE, TreatyStatus.TERMINATED],
    TreatyStatus.TERMINATED: [],  # terminal state
}


class Treaty:
    """Bilateral governance agreement between two systems."""

    def __init__(
        self,
        id: TreatyID,
        system_a: SystemURI,
        system_b: SystemURI,
        terms: list[TreatyTerm],
        status: TreatyStatus = TreatyStatus.PROPOSED,
    ) -> None:
        self.id = id
        self.system_a = system_a
        self.system_b = system_b
        self.status = status
        self.terms = list(terms)
        now = time.time()
        self.created_at = now
        self.updated_at = now

    def transition(self, to: TreatyStatus) -> None:
        """Attempt to move the treaty to a new status.

        Raises InvalidTransitionError if the transition is invalid.
        """
        allowed = _VALID_TREATY_TRANSITIONS.get(self.status, [])
        if to in allowed:
            self.status = to
            self.updated_at = time.time()
            return
        raise InvalidTransitionError("Treaty", self.status.value, to.value)

    def apply_action(self, action: TreatyAction) -> None:
        """Apply a treaty action. Raises on invalid transitions."""
        if action == TreatyAction.ACCEPT:
            self.transition(TreatyStatus.ACTIVE)
        elif action == TreatyAction.SUSPEND:
            self.transition(TreatyStatus.SUSPENDED)
        elif action == TreatyAction.TERMINATE:
            self.transition(TreatyStatus.TERMINATED)
        elif action == TreatyAction.MODIFY:
            if self.status != TreatyStatus.ACTIVE:
                raise InvalidTransitionError(
                    "Treaty", self.status.value, f"Modify (requires Active)"
                )
            self.updated_at = time.time()
        elif action == TreatyAction.PROPOSE:
            raise ValueError("cannot apply Propose to existing treaty")
        else:
            raise ValueError(f"unknown treaty action: {action}")

    def copy(self) -> Treaty:
        """Return a shallow copy."""
        t = Treaty(self.id, self.system_a, self.system_b, list(self.terms), self.status)
        t.created_at = self.created_at
        t.updated_at = self.updated_at
        return t


def new_treaty(
    id: TreatyID,
    system_a: SystemURI,
    system_b: SystemURI,
    terms: list[TreatyTerm],
) -> Treaty:
    """Create a new treaty in Proposed status."""
    return Treaty(id, system_a, system_b, terms)


# ── Trust ──────────────────────────────────────────────────────────────────

INTER_SYSTEM_DECAY_RATE = 0.02  # per day
INTER_SYSTEM_MAX_ADJUSTMENT = 0.05

# Trust impact constants
TRUST_IMPACT_VALID_PROOF = 0.02
TRUST_IMPACT_RECEIPT_ON_TIME = 0.01
TRUST_IMPACT_TREATY_HONOURED = 0.03
TRUST_IMPACT_TREATY_VIOLATED = -0.15
TRUST_IMPACT_INVALID_PROOF = -0.10
TRUST_IMPACT_SIGNATURE_INVALID = -0.20
TRUST_IMPACT_NO_HELLO_RESPONSE = -0.05


@dataclass
class PeerRecord:
    """Tracks the state of a known remote system."""
    system_uri: SystemURI
    public_key: PublicKey
    trust: Score
    capabilities: list[str]
    negotiated_version: int
    last_seen: float
    first_seen: float
    last_decayed_at: float


class PeerStore:
    """Manages known peer systems and their trust scores. Thread-safe."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._peers: dict[str, PeerRecord] = {}

    def register(
        self,
        uri: SystemURI,
        public_key: PublicKey,
        capabilities: list[str],
        negotiated_version: int,
    ) -> PeerRecord:
        """Add or update a peer from a HELLO exchange."""
        with self._lock:
            key = uri.value
            now = time.time()

            if key in self._peers:
                existing = self._peers[key]
                # Do NOT overwrite PublicKey on re-registration (TOFU)
                existing.capabilities = capabilities
                existing.negotiated_version = negotiated_version
                existing.last_seen = now
                return existing

            record = PeerRecord(
                system_uri=uri,
                public_key=public_key,
                trust=Score(0.0),
                capabilities=capabilities,
                negotiated_version=negotiated_version,
                last_seen=now,
                first_seen=now,
                last_decayed_at=now,
            )
            self._peers[key] = record
            return record

    def get(self, uri: SystemURI) -> tuple[PeerRecord, bool]:
        """Return a copy of the peer record. Returns (record, found)."""
        with self._lock:
            record = self._peers.get(uri.value)
            if record is None:
                return PeerRecord(
                    system_uri=uri,
                    public_key=PublicKey(b"\x00" * 32),
                    trust=Score(0.0),
                    capabilities=[],
                    negotiated_version=0,
                    last_seen=0.0,
                    first_seen=0.0,
                    last_decayed_at=0.0,
                ), False
            # Return a copy
            return PeerRecord(
                system_uri=record.system_uri,
                public_key=record.public_key,
                trust=record.trust,
                capabilities=list(record.capabilities),
                negotiated_version=record.negotiated_version,
                last_seen=record.last_seen,
                first_seen=record.first_seen,
                last_decayed_at=record.last_decayed_at,
            ), True

    def update_trust(self, uri: SystemURI, delta: float) -> tuple[Score, bool]:
        """Adjust a peer's trust score. Positive deltas capped at max adjustment."""
        with self._lock:
            record = self._peers.get(uri.value)
            if record is None:
                return Score(0.0), False

            # Positive trust accumulates gradually; negative hits immediately
            if delta > 0:
                delta = min(delta, INTER_SYSTEM_MAX_ADJUSTMENT)

            new_val = record.trust.value + delta
            new_val = max(0.0, min(1.0, new_val))

            record.trust = Score(new_val)
            record.last_seen = time.time()
            return record.trust, True

    def decay_all(self) -> None:
        """Apply time-based trust decay to all peers."""
        with self._lock:
            now = time.time()
            for record in self._peers.values():
                days_since = (now - record.last_decayed_at) / 86400.0
                if days_since <= 0:
                    continue
                decay = INTER_SYSTEM_DECAY_RATE * days_since
                new_val = max(0.0, record.trust.value - decay)
                record.trust = Score(new_val)
                record.last_decayed_at = now

    def all(self) -> list[PeerRecord]:
        """Return copies of all peer records."""
        with self._lock:
            return [
                PeerRecord(
                    system_uri=r.system_uri,
                    public_key=r.public_key,
                    trust=r.trust,
                    capabilities=list(r.capabilities),
                    negotiated_version=r.negotiated_version,
                    last_seen=r.last_seen,
                    first_seen=r.first_seen,
                    last_decayed_at=r.last_decayed_at,
                )
                for r in self._peers.values()
            ]


# ── Treaty Store ───────────────────────────────────────────────────────────


class TreatyStore:
    """Manages bilateral treaties. Thread-safe."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._treaties: dict[str, Treaty] = {}

    def put(self, treaty: Treaty) -> None:
        """Store or update a treaty."""
        with self._lock:
            self._treaties[treaty.id.value] = treaty

    def get(self, id: TreatyID) -> tuple[Treaty, bool]:
        """Return a copy of a treaty by ID. Returns (treaty, found)."""
        with self._lock:
            t = self._treaties.get(id.value)
            if t is None:
                return None, False  # type: ignore[return-value]
            return t.copy(), True

    def apply(self, id: TreatyID, fn: Callable[[Treaty], None]) -> None:
        """Perform a read-modify-write on a treaty under a single lock.

        Raises TreatyNotFoundError if the treaty does not exist.
        """
        with self._lock:
            t = self._treaties.get(id.value)
            if t is None:
                raise TreatyNotFoundError(id)
            fn(t)

    def by_system(self, uri: SystemURI) -> list[Treaty]:
        """Return all treaties involving a given system URI."""
        with self._lock:
            result = []
            for t in self._treaties.values():
                if t.system_a.value == uri.value or t.system_b.value == uri.value:
                    result.append(t.copy())
            return result

    def active(self) -> list[Treaty]:
        """Return all active treaties."""
        with self._lock:
            return [t.copy() for t in self._treaties.values() if t.status == TreatyStatus.ACTIVE]


# ── Envelope Dedup ─────────────────────────────────────────────────────────


class EnvelopeDedup:
    """Replay protection by tracking seen envelope IDs with TTL-based pruning."""

    def __init__(self, ttl: float | None = None) -> None:
        self._lock = threading.Lock()
        self._seen: dict[str, float] = {}
        self._ttl = ttl if ttl is not None else (MAX_ENVELOPE_AGE_SECONDS + 3600)
        self._check_count = 0

    def check(self, id: EnvelopeID) -> bool:
        """Return True if the envelope ID has not been seen before.

        Records the ID and returns False on subsequent calls.
        """
        with self._lock:
            key = id.value
            if key in self._seen:
                return False

            self._seen[key] = time.time()
            self._check_count += 1

            if self._check_count % DEDUP_PRUNE_INTERVAL == 0:
                self._prune_locked()

            return True

    def prune(self) -> int:
        """Remove expired entries older than TTL. Returns count removed."""
        with self._lock:
            return self._prune_locked()

    def _prune_locked(self) -> int:
        cutoff = time.time() - self._ttl
        to_remove = [k for k, ts in self._seen.items() if ts < cutoff]
        for k in to_remove:
            del self._seen[k]
        return len(to_remove)

    def size(self) -> int:
        """Return the number of tracked envelope IDs."""
        with self._lock:
            return len(self._seen)


# ── Proof Verification ────────────────────────────────────────────────────


def verify_chain_segment(proof: ChainSegmentProof) -> bool:
    """Verify that a chain segment is internally consistent."""
    if not proof.events:
        return False

    # Check first event's prev_hash matches start_hash
    first_event = proof.events[0]
    if first_event.get("prev_hash") != proof.start_hash:
        return False

    # Check internal hash chain continuity
    for i in range(1, len(proof.events)):
        if proof.events[i].get("prev_hash") != proof.events[i - 1].get("hash"):
            return False

    # Check last event's hash matches end_hash
    if proof.events[-1].get("hash") != proof.end_hash:
        return False

    return True


def verify_event_existence(proof: EventExistenceProof) -> bool:
    """Verify basic properties of an event existence proof."""
    # Event's prev_hash should match the proof's prev_hash
    if proof.event.get("prev_hash") != proof.prev_hash:
        return False

    # Position and chain length should be consistent
    if proof.position < 0 or proof.position >= proof.chain_length:
        return False

    # Event should have a non-empty hash
    if not proof.event.get("hash"):
        return False

    return True


def validate_proof(payload: ProofPayload) -> tuple[bool, str | None]:
    """Dispatch to the appropriate proof verifier.

    Returns (valid, error_message).
    """
    if isinstance(payload.data, ChainSegmentProof):
        return verify_chain_segment(payload.data), None
    elif isinstance(payload.data, EventExistenceProof):
        return verify_event_existence(payload.data), None
    elif isinstance(payload.data, ChainSummaryProof):
        return payload.data.length > 0, None
    else:
        return False, f"unknown proof type: {type(payload.data)}"


# ── Proof Generator ───────────────────────────────────────────────────────


class ProofGenerator:
    """Creates integrity proofs from event data.

    This is a simplified version that works with dict-based events.
    Production implementations would use the Store interface.
    """

    def __init__(self, events: list[dict[str, Any]]) -> None:
        self._events = events

    def generate_chain_summary(self) -> ChainSummaryProof:
        """Produce a high-level chain integrity attestation."""
        if not self._events:
            raise ValueError("empty chain")

        head = self._events[-1]
        return ChainSummaryProof(
            length=len(self._events),
            head_hash=head.get("hash", ""),
            genesis_hash=self._events[0].get("prev_hash", ""),
            timestamp=time.time(),
        )

    def generate_event_existence(self, event_id: str) -> EventExistenceProof:
        """Prove that a specific event exists in the chain."""
        for i, evt in enumerate(self._events):
            if evt.get("id") == event_id:
                next_hash: Option[str] = Option.none()
                if i + 1 < len(self._events):
                    next_hash = Option.some(self._events[i + 1].get("hash", ""))
                return EventExistenceProof(
                    event=evt,
                    prev_hash=evt.get("prev_hash", ""),
                    next_hash=next_hash,
                    position=i,
                    chain_length=len(self._events),
                )
        raise ValueError(f"event not found: {event_id}")


# ── Transport ──────────────────────────────────────────────────────────────


@runtime_checkable
class ITransport(Protocol):
    """Pluggable transport layer for EGIP communication."""

    def send(self, to: SystemURI, envelope: Envelope) -> ReceiptPayload | None: ...


# ── Handler ────────────────────────────────────────────────────────────────


def _generate_uuid4() -> str:
    """Generate a random UUID v4 string."""
    return str(uuid.uuid4())


class Handler:
    """Orchestrates EGIP protocol interactions.

    Handles HELLO handshake, message dispatch, replay deduplication,
    and trust updates.
    """

    def __init__(
        self,
        identity: IIdentity,
        transport: ITransport,
        peers: PeerStore,
        treaties: TreatyStore,
    ) -> None:
        self.identity = identity
        self.transport = transport
        self.peers = peers
        self.treaties = treaties
        self.dedup = EnvelopeDedup()
        self.local_protocol_versions = [CURRENT_PROTOCOL_VERSION]
        self.capabilities = ["treaty", "proof"]
        self.chain_length: Callable[[], int] | None = None
        self.on_message: Callable[[SystemURI, MessagePayloadContent], None] | None = None
        self.on_authority_request: Callable[[SystemURI, AuthorityRequestPayload], None] | None = None
        self.on_discover: Callable[[SystemURI, DiscoverQuery], list[DiscoverResult]] | None = None

    def hello(self, to: SystemURI) -> None:
        """Perform the HELLO handshake with a remote system."""
        chain_len = 0
        if self.chain_length is not None:
            chain_len = self.chain_length()

        env_id = EnvelopeID(_generate_uuid4())

        env = Envelope(
            protocol_version=CURRENT_PROTOCOL_VERSION,
            id=env_id,
            from_uri=self.identity.system_uri(),
            to_uri=to,
            type=MessageType.HELLO,
            payload=HelloPayload(
                system_uri=self.identity.system_uri().value,
                public_key=self.identity.public_key().bytes_,
                protocol_versions=list(self.local_protocol_versions),
                capabilities=list(self.capabilities),
                chain_length=chain_len,
            ),
            timestamp=time.time(),
            in_reply_to=Option.none(),
        )

        signed = sign_envelope(env, self.identity)

        receipt = self.transport.send(to, signed)

        if receipt is not None and receipt.status == ReceiptStatus.REJECTED:
            reason = ""
            if receipt.reason.is_some():
                reason = receipt.reason.unwrap()
            raise EGIPError(f"hello rejected: {reason}")

    def handle_incoming(self, env: Envelope) -> None:
        """Process an incoming envelope.

        Checks timestamp freshness, deduplicates, verifies signature,
        dispatches to the appropriate handler, and updates trust.
        """
        # Timestamp freshness
        age = time.time() - env.timestamp
        if age > MAX_ENVELOPE_AGE_SECONDS or age < -300:
            raise EGIPError(f"envelope timestamp out of range: age {age:.0f}s")

        # Replay deduplication
        if not self.dedup.check(env.id):
            raise DuplicateEnvelopeError(env.id)

        # Look up sender's public key
        peer, known = self.peers.get(env.from_uri)

        # For HELLO, use public key from payload (TOFU model)
        if env.type == MessageType.HELLO:
            if not isinstance(env.payload, HelloPayload):
                raise EGIPError(f"invalid hello payload type: {type(env.payload)}")
            pub_key = PublicKey(env.payload.public_key)
        else:
            if not known:
                raise SystemNotFoundError(env.from_uri)
            pub_key = peer.public_key

        # Verify signature
        valid = verify_envelope(env, self.identity, pub_key)
        if not valid:
            self.peers.update_trust(env.from_uri, TRUST_IMPACT_SIGNATURE_INVALID)
            raise EnvelopeSignatureInvalidError(env.id)

        # Dispatch by message type
        if env.type == MessageType.HELLO:
            self._handle_hello(env)
        elif env.type == MessageType.MESSAGE:
            self._handle_message(env)
        elif env.type == MessageType.RECEIPT:
            self._handle_receipt(env)
        elif env.type == MessageType.PROOF:
            self._handle_proof(env)
        elif env.type == MessageType.TREATY:
            self._handle_treaty(env)
        elif env.type == MessageType.AUTHORITY_REQUEST:
            self._handle_authority_request(env)
        elif env.type == MessageType.DISCOVER:
            self._handle_discover(env)
        else:
            raise EGIPError(f"unknown message type: {env.type}")

    def _handle_hello(self, env: Envelope) -> None:
        hello = env.payload
        if not isinstance(hello, HelloPayload):
            raise EGIPError(f"invalid hello payload type: {type(hello)}")

        version = negotiate_version(self.local_protocol_versions, hello.protocol_versions)
        if not version.is_some():
            raise VersionIncompatibleError(
                self.local_protocol_versions, hello.protocol_versions
            )

        self.peers.register(
            SystemURI(hello.system_uri),
            PublicKey(hello.public_key),
            hello.capabilities,
            version.unwrap(),
        )

    def _handle_message(self, env: Envelope) -> None:
        if not isinstance(env.payload, MessagePayloadContent):
            raise EGIPError(f"invalid message payload type: {type(env.payload)}")

        self.peers.update_trust(env.from_uri, TRUST_IMPACT_RECEIPT_ON_TIME)

        if self.on_message is not None:
            self.on_message(env.from_uri, env.payload)

    def _handle_receipt(self, env: Envelope) -> None:
        if not isinstance(env.payload, ReceiptPayload):
            raise EGIPError(f"invalid receipt payload type: {type(env.payload)}")

        receipt = env.payload
        if receipt.status in (ReceiptStatus.PROCESSED, ReceiptStatus.DELIVERED):
            self.peers.update_trust(env.from_uri, TRUST_IMPACT_RECEIPT_ON_TIME)

    def _handle_proof(self, env: Envelope) -> None:
        if not isinstance(env.payload, ProofPayload):
            raise EGIPError(f"invalid proof payload type: {type(env.payload)}")

        valid, _ = validate_proof(env.payload)
        if valid:
            self.peers.update_trust(env.from_uri, TRUST_IMPACT_VALID_PROOF)
        else:
            self.peers.update_trust(env.from_uri, TRUST_IMPACT_INVALID_PROOF)

    def _handle_treaty(self, env: Envelope) -> None:
        if not isinstance(env.payload, TreatyPayload):
            raise EGIPError(f"invalid treaty payload type: {type(env.payload)}")

        payload = env.payload
        tid = TreatyID(payload.treaty_id)

        if payload.action == TreatyAction.PROPOSE:
            treaty = new_treaty(tid, env.from_uri, env.to_uri, payload.terms)
            self.treaties.put(treaty)
        elif payload.action == TreatyAction.ACCEPT:
            def accept(t: Treaty) -> None:
                t.apply_action(TreatyAction.ACCEPT)
                self.peers.update_trust(env.from_uri, TRUST_IMPACT_TREATY_HONOURED)
            self.treaties.apply(tid, accept)
        elif payload.action == TreatyAction.SUSPEND:
            self.treaties.apply(tid, lambda t: t.apply_action(TreatyAction.SUSPEND))
        elif payload.action == TreatyAction.TERMINATE:
            self.treaties.apply(tid, lambda t: t.apply_action(TreatyAction.TERMINATE))
        elif payload.action == TreatyAction.MODIFY:
            def modify(t: Treaty) -> None:
                t.apply_action(TreatyAction.MODIFY)
                t.terms = list(payload.terms)
            self.treaties.apply(tid, modify)

    def _handle_authority_request(self, env: Envelope) -> None:
        if not isinstance(env.payload, AuthorityRequestPayload):
            raise EGIPError(f"invalid authority request payload type: {type(env.payload)}")

        if self.on_authority_request is not None:
            self.on_authority_request(env.from_uri, env.payload)

    def _handle_discover(self, env: Envelope) -> None:
        if not isinstance(env.payload, DiscoverPayload):
            raise EGIPError(f"invalid discover payload type: {type(env.payload)}")

        if self.on_discover is None:
            return

        results = self.on_discover(env.from_uri, env.payload.query)

        resp_id = EnvelopeID(_generate_uuid4())
        resp = Envelope(
            protocol_version=CURRENT_PROTOCOL_VERSION,
            id=resp_id,
            from_uri=self.identity.system_uri(),
            to_uri=env.from_uri,
            type=MessageType.DISCOVER,
            payload=DiscoverPayload(
                query=env.payload.query,
                results=results,
            ),
            timestamp=time.time(),
            in_reply_to=Option.some(env.id),
        )

        signed = sign_envelope(resp, self.identity)
        self.transport.send(env.from_uri, signed)
