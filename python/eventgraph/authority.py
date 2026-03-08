"""Authority module — flat authority model with policy matching and trust downgrade.

Ports the Go authority package (DefaultAuthorityChain). Provides weighted
authority evaluation, not binary permission.
"""

from __future__ import annotations

import threading
from dataclasses import dataclass, field
from typing import Protocol, runtime_checkable

from .decision import AuthorityLevel
from .types import ActorID, DomainScope, Option, Score


# ── Value Types ──────────────────────────────────────────────────────────


@dataclass(frozen=True, slots=True)
class AuthorityLink:
    """A link in an authority chain."""

    _actor: ActorID
    _level: AuthorityLevel
    _weight: Score

    def __init__(self, actor: ActorID, level: AuthorityLevel, weight: Score) -> None:
        object.__setattr__(self, "_actor", actor)
        object.__setattr__(self, "_level", level)
        object.__setattr__(self, "_weight", weight)

    @property
    def actor(self) -> ActorID:
        return self._actor

    @property
    def level(self) -> AuthorityLevel:
        return self._level

    @property
    def weight(self) -> Score:
        return self._weight


@dataclass(frozen=True, slots=True)
class AuthorityResult:
    """Result of evaluating authority for an action."""

    _level: AuthorityLevel
    _weight: Score
    _chain: tuple[AuthorityLink, ...]

    def __init__(
        self, level: AuthorityLevel, weight: Score, chain: list[AuthorityLink]
    ) -> None:
        object.__setattr__(self, "_level", level)
        object.__setattr__(self, "_weight", weight)
        object.__setattr__(self, "_chain", tuple(chain))

    @property
    def level(self) -> AuthorityLevel:
        return self._level

    @property
    def weight(self) -> Score:
        return self._weight

    @property
    def chain(self) -> list[AuthorityLink]:
        return list(self._chain)


@dataclass(frozen=True, slots=True)
class AuthorityPolicy:
    """Defines the authority requirements for an action pattern."""

    action: str
    level: AuthorityLevel
    min_trust: Option[Score] = field(default_factory=Option.none)
    scope: Option[DomainScope] = field(default_factory=Option.none)


# ── AuthorityChain Protocol ─────────────────────────────────────────────


@runtime_checkable
class AuthorityChain(Protocol):
    """Evaluates authority. Returns weighted authority, not binary permission."""

    def evaluate(self, actor: ActorID, action: str) -> AuthorityResult: ...

    def chain(self, actor: ActorID, action: str) -> list[AuthorityLink]: ...

    def grant(
        self,
        from_actor: ActorID,
        to_actor: ActorID,
        scope: DomainScope,
        weight: Score,
    ) -> None: ...

    def revoke(
        self, from_actor: ActorID, to_actor: ActorID, scope: DomainScope
    ) -> None: ...


# ── Helper ───────────────────────────────────────────────────────────────


def matches_action(pattern: str, action: str) -> bool:
    """Match an action against a pattern.

    - ``"*"`` matches everything.
    - ``"prefix*"`` matches any action starting with ``prefix``.
    - Otherwise exact match.
    """
    if pattern == "*":
        return True
    if len(pattern) > 0 and pattern[-1] == "*":
        prefix = pattern[:-1]
        return len(action) >= len(prefix) and action[: len(prefix)] == prefix
    return pattern == action


# ── DefaultAuthorityChain ────────────────────────────────────────────────


class DefaultAuthorityChain:
    """Flat authority model — no delegation chain.

    All actions default to Notification unless a policy says otherwise.
    If a Required policy has a ``min_trust`` threshold and the actor's trust
    meets or exceeds it, the level is downgraded to Recommended.

    Implements the :class:`AuthorityChain` protocol.
    """

    def __init__(self, trust_model: object) -> None:
        """Create a flat authority chain.

        Parameters
        ----------
        trust_model:
            An object implementing the ``TrustModel`` protocol (must have a
            ``score(actor) -> TrustMetrics`` method).
        """
        self._lock = threading.Lock()
        self._policies: list[AuthorityPolicy] = []
        self._trust_model = trust_model

    # ── Policy management ────────────────────────────────────────────────

    def add_policy(self, policy: AuthorityPolicy) -> None:
        """Register an authority policy. Policies are checked in order; first match wins."""
        with self._lock:
            self._policies.append(policy)

    # ── Private helpers ──────────────────────────────────────────────────

    def _find_policy(self, action: str) -> AuthorityPolicy:
        """Return the first matching policy, or the default Notification policy."""
        for p in self._policies:
            if matches_action(p.action, action):
                return p
        return AuthorityPolicy(action="*", level=AuthorityLevel.NOTIFICATION)

    # ── AuthorityChain protocol ──────────────────────────────────────────

    def evaluate(self, actor: ActorID, action: str) -> AuthorityResult:
        """Evaluate authority for *actor* performing *action*.

        If the matching policy is Required and specifies a ``min_trust``, and
        the actor's trust score meets or exceeds that threshold, the level is
        downgraded to Recommended.
        """
        with self._lock:
            policy = self._find_policy(action)

        level = policy.level

        # Trust-based downgrade: Required → Recommended
        if level == AuthorityLevel.REQUIRED and policy.min_trust.is_some():
            metrics = self._trust_model.score(actor)
            if metrics.overall.value >= policy.min_trust.unwrap().value:
                level = AuthorityLevel.RECOMMENDED

        link = AuthorityLink(actor=actor, level=level, weight=Score(1.0))

        return AuthorityResult(
            level=level,
            weight=Score(1.0),
            chain=[link],
        )

    def chain(self, actor: ActorID, action: str) -> list[AuthorityLink]:
        """Return the authority chain for *actor* performing *action*.

        In the flat model this is always a single-link chain.
        """
        with self._lock:
            policy = self._find_policy(action)

        return [AuthorityLink(actor=actor, level=policy.level, weight=Score(1.0))]

    def grant(
        self,
        from_actor: ActorID,
        to_actor: ActorID,
        scope: DomainScope,
        weight: Score,
    ) -> None:
        """Grant authority — no-op in the flat model.

        A full implementation would emit an ``edge.created`` event.
        """

    def revoke(
        self, from_actor: ActorID, to_actor: ActorID, scope: DomainScope
    ) -> None:
        """Revoke authority — no-op in the flat model.

        A full implementation would emit an ``edge.superseded`` event.
        """
