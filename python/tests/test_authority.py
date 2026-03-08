"""Tests for the authority module."""

from __future__ import annotations

from eventgraph.authority import (
    AuthorityLink,
    AuthorityPolicy,
    AuthorityResult,
    DefaultAuthorityChain,
    matches_action,
)
from eventgraph.decision import AuthorityLevel
from eventgraph.trust import DefaultTrustModel, TrustConfig
from eventgraph.types import ActorID, DomainScope, Option, Score

from eventgraph.event import (
    Event,
    NoopSigner,
    create_event,
    new_event_id,
)
from eventgraph.types import ConversationID, EventType, Hash


# ── Helpers ──────────────────────────────────────────────────────────────


def _make_trust_event(actor_id: ActorID, current: float) -> Event:
    """Create a trust.updated evidence event."""
    return create_event(
        event_type=EventType("trust.updated"),
        source=actor_id,
        content={"current": current, "domain": "general"},
        causes=[new_event_id()],
        conversation_id=ConversationID("conv_test"),
        prev_hash=Hash.zero(),
        signer=NoopSigner(),
    )


# ── DefaultAuthorityChain tests ──────────────────────────────────────────


def test_default_notification() -> None:
    """Unmatched action returns Notification level."""
    model = DefaultTrustModel()
    chain = DefaultAuthorityChain(trust_model=model)
    actor = ActorID("actor_alice")

    result = chain.evaluate(actor, "some.random.action")

    assert result.level == AuthorityLevel.NOTIFICATION
    assert len(result.chain) == 1


def test_policy_required() -> None:
    """Matched action with Required policy returns Required level."""
    model = DefaultTrustModel()
    chain = DefaultAuthorityChain(trust_model=model)
    chain.add_policy(AuthorityPolicy(action="actor.suspend", level=AuthorityLevel.REQUIRED))
    actor = ActorID("actor_alice")

    result = chain.evaluate(actor, "actor.suspend")

    assert result.level == AuthorityLevel.REQUIRED


def test_policy_recommended() -> None:
    """Matched action with Recommended policy returns Recommended level."""
    model = DefaultTrustModel()
    chain = DefaultAuthorityChain(trust_model=model)
    chain.add_policy(
        AuthorityPolicy(action="review.code", level=AuthorityLevel.RECOMMENDED)
    )
    actor = ActorID("actor_alice")

    result = chain.evaluate(actor, "review.code")

    assert result.level == AuthorityLevel.RECOMMENDED


def test_wildcard_policy() -> None:
    """Prefix wildcard 'trust.*' matches 'trust.update'."""
    model = DefaultTrustModel()
    chain = DefaultAuthorityChain(trust_model=model)
    chain.add_policy(
        AuthorityPolicy(action="trust.*", level=AuthorityLevel.RECOMMENDED)
    )
    actor = ActorID("actor_alice")

    result = chain.evaluate(actor, "trust.update")

    assert result.level == AuthorityLevel.RECOMMENDED


def test_catch_all_policy() -> None:
    """'*' matches everything."""
    model = DefaultTrustModel()
    chain = DefaultAuthorityChain(trust_model=model)
    chain.add_policy(AuthorityPolicy(action="*", level=AuthorityLevel.REQUIRED))
    actor = ActorID("actor_alice")

    result = chain.evaluate(actor, "anything.at.all")

    assert result.level == AuthorityLevel.REQUIRED


def test_first_match_wins() -> None:
    """Multiple policies — first match returned."""
    model = DefaultTrustModel()
    chain = DefaultAuthorityChain(trust_model=model)
    chain.add_policy(AuthorityPolicy(action="deploy", level=AuthorityLevel.REQUIRED))
    chain.add_policy(
        AuthorityPolicy(action="deploy", level=AuthorityLevel.NOTIFICATION)
    )
    actor = ActorID("actor_alice")

    result = chain.evaluate(actor, "deploy")

    assert result.level == AuthorityLevel.REQUIRED


def test_trust_downgrade() -> None:
    """Required with min_trust — high trust downgrades to Recommended."""
    model = DefaultTrustModel(config=TrustConfig(initial_trust=Score(0.0)))
    chain = DefaultAuthorityChain(trust_model=model)

    actor = ActorID("actor_alice")

    # Build trust above threshold via evidence events
    for i in range(5):
        ev = _make_trust_event(actor, 0.8)
        model.update(actor, ev)

    chain.add_policy(
        AuthorityPolicy(
            action="deploy",
            level=AuthorityLevel.REQUIRED,
            min_trust=Option.some(Score(0.01)),
        )
    )

    result = chain.evaluate(actor, "deploy")

    assert result.level == AuthorityLevel.RECOMMENDED


def test_trust_no_downgrade() -> None:
    """Required with min_trust — low trust stays Required."""
    model = DefaultTrustModel(config=TrustConfig(initial_trust=Score(0.0)))
    chain = DefaultAuthorityChain(trust_model=model)

    chain.add_policy(
        AuthorityPolicy(
            action="deploy",
            level=AuthorityLevel.REQUIRED,
            min_trust=Option.some(Score(0.5)),
        )
    )
    actor = ActorID("actor_alice")

    # Trust is 0.0 (initial), below 0.5 threshold
    result = chain.evaluate(actor, "deploy")

    assert result.level == AuthorityLevel.REQUIRED


def test_chain_returns_single_link() -> None:
    """chain() returns a single-link chain in the flat model."""
    model = DefaultTrustModel()
    chain = DefaultAuthorityChain(trust_model=model)
    actor = ActorID("actor_alice")

    links = chain.chain(actor, "any.action")

    assert len(links) == 1
    assert links[0].actor == actor


def test_evaluate_weight_is_1() -> None:
    """Evaluate always returns weight 1.0 in the flat model."""
    model = DefaultTrustModel()
    chain = DefaultAuthorityChain(trust_model=model)
    actor = ActorID("actor_alice")

    result = chain.evaluate(actor, "test")

    assert result.weight.value == 1.0


# ── matches_action helper tests ─────────────────────────────────────────


def test_matches_action_exact() -> None:
    assert matches_action("deploy", "deploy") is True
    assert matches_action("deploy", "review") is False


def test_matches_action_star() -> None:
    assert matches_action("*", "anything") is True


def test_matches_action_prefix() -> None:
    assert matches_action("trust.*", "trust.update") is True
    assert matches_action("trust.*", "trust.") is True
    assert matches_action("trust.*", "trus") is False
