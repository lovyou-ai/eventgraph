"""EventGraph — hash-chained, append-only, causal event graph."""

from .bus import EventBus
from .errors import (
    ChainIntegrityError,
    EmptyRequiredError,
    EventGraphError,
    EventNotFoundError,
    InvalidFormatError,
    InvalidTransitionError,
    OutOfRangeError,
    StoreError,
    ValidationError,
)
from .event import (
    Event,
    NoopSigner,
    Signer,
    canonical_content_json,
    canonical_form,
    compute_hash,
    create_bootstrap,
    create_event,
    new_event_id,
)
from .primitive import (
    LIFECYCLE_ACTIVE,
    LIFECYCLE_ACTIVATING,
    LIFECYCLE_DORMANT,
    LIFECYCLE_EMITTING,
    LIFECYCLE_MEMORIAL,
    LIFECYCLE_PROCESSING,
    LIFECYCLE_SUSPENDED,
    LIFECYCLE_SUSPENDING,
    AddEvent,
    Mutation,
    Primitive,
    PrimitiveState,
    Registry,
    Snapshot,
    UpdateActivation,
    UpdateLifecycle,
    UpdateState,
)
from .grammar import Grammar
from .store import ChainVerification, InMemoryStore, Store
from .tick import TickConfig, TickEngine, TickResult
from .types import (
    Activation,
    ActorID,
    Cadence,
    ConversationID,
    DomainScope,
    EdgeID,
    EnvelopeID,
    EventID,
    EventType,
    Hash,
    Layer,
    NonEmpty,
    Option,
    PublicKey,
    Score,
    Signature,
    SubscriptionPattern,
    SystemURI,
    TreatyID,
    Weight,
)

__version__ = "0.1.0"
