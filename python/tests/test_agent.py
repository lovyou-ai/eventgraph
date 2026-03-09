"""Tests for the 28 agent primitives, OperationalState FSM, and 8 compositions."""

from __future__ import annotations

import pytest

from eventgraph.agent import (
    # FSM
    OperationalState,
    # Primitives
    ALL_AGENT_PRIMITIVE_CLASSES,
    _AgentBase,
    all_primitives,
    register_all,
    is_agent_primitive,
    # Compositions
    AgentComposition,
    all_compositions,
    boot,
    imprint,
    task,
    supervise,
    collaborate,
    crisis,
    retire,
    whistleblow,
    # Event types
    all_agent_event_types,
    AGENT_IDENTITY_CREATED,
)
from eventgraph.event import Event, NoopSigner, create_bootstrap, create_event
from eventgraph.primitive import (
    Registry,
    Snapshot,
    UpdateState,
)
from eventgraph.types import (
    ActorID,
    Cadence,
    ConversationID,
    EventID,
    EventType,
    Hash,
    Layer,
    NonEmpty,
    PrimitiveID,
    Signature,
    SubscriptionPattern,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _empty_snapshot(tick: int = 1) -> Snapshot:
    return Snapshot(tick=tick, primitives={}, pending_events=[], recent_events=[])


def _make_event(event_type: str) -> Event:
    """Create a minimal Event with the given type for testing process()."""
    signer = NoopSigner()
    bootstrap = create_bootstrap(source=ActorID("system"), signer=signer)
    return create_event(
        event_type=EventType(event_type),
        source=ActorID("test-agent"),
        content={"test": True},
        causes=[bootstrap.id],
        conversation_id=ConversationID("conv_test"),
        prev_hash=bootstrap.hash,
        signer=signer,
    )


# ===========================================================================
# 1. OperationalState FSM
# ===========================================================================

class TestOperationalStateFSM:
    """Test valid and invalid state transitions."""

    def test_all_states_exist(self) -> None:
        states = list(OperationalState)
        assert len(states) == 8

    def test_string_values(self) -> None:
        assert str(OperationalState.IDLE) == "Idle"
        assert str(OperationalState.PROCESSING) == "Processing"
        assert str(OperationalState.WAITING) == "Waiting"
        assert str(OperationalState.ESCALATING) == "Escalating"
        assert str(OperationalState.REFUSING) == "Refusing"
        assert str(OperationalState.SUSPENDED) == "Suspended"
        assert str(OperationalState.RETIRING) == "Retiring"
        assert str(OperationalState.RETIRED) == "Retired"

    def test_is_terminal(self) -> None:
        assert OperationalState.RETIRED.is_terminal is True
        for s in OperationalState:
            if s is not OperationalState.RETIRED:
                assert s.is_terminal is False

    def test_can_act(self) -> None:
        assert OperationalState.PROCESSING.can_act is True
        for s in OperationalState:
            if s is not OperationalState.PROCESSING:
                assert s.can_act is False

    # Valid transitions
    def test_idle_to_processing(self) -> None:
        result = OperationalState.IDLE.transition_to(OperationalState.PROCESSING)
        assert result is OperationalState.PROCESSING

    def test_idle_to_suspended(self) -> None:
        result = OperationalState.IDLE.transition_to(OperationalState.SUSPENDED)
        assert result is OperationalState.SUSPENDED

    def test_idle_to_retiring(self) -> None:
        result = OperationalState.IDLE.transition_to(OperationalState.RETIRING)
        assert result is OperationalState.RETIRING

    def test_processing_to_idle(self) -> None:
        result = OperationalState.PROCESSING.transition_to(OperationalState.IDLE)
        assert result is OperationalState.IDLE

    def test_processing_to_waiting(self) -> None:
        result = OperationalState.PROCESSING.transition_to(OperationalState.WAITING)
        assert result is OperationalState.WAITING

    def test_processing_to_escalating(self) -> None:
        result = OperationalState.PROCESSING.transition_to(OperationalState.ESCALATING)
        assert result is OperationalState.ESCALATING

    def test_processing_to_refusing(self) -> None:
        result = OperationalState.PROCESSING.transition_to(OperationalState.REFUSING)
        assert result is OperationalState.REFUSING

    def test_processing_to_retiring(self) -> None:
        result = OperationalState.PROCESSING.transition_to(OperationalState.RETIRING)
        assert result is OperationalState.RETIRING

    def test_waiting_to_processing(self) -> None:
        result = OperationalState.WAITING.transition_to(OperationalState.PROCESSING)
        assert result is OperationalState.PROCESSING

    def test_waiting_to_idle(self) -> None:
        result = OperationalState.WAITING.transition_to(OperationalState.IDLE)
        assert result is OperationalState.IDLE

    def test_waiting_to_retiring(self) -> None:
        result = OperationalState.WAITING.transition_to(OperationalState.RETIRING)
        assert result is OperationalState.RETIRING

    def test_escalating_to_waiting(self) -> None:
        result = OperationalState.ESCALATING.transition_to(OperationalState.WAITING)
        assert result is OperationalState.WAITING

    def test_escalating_to_idle(self) -> None:
        result = OperationalState.ESCALATING.transition_to(OperationalState.IDLE)
        assert result is OperationalState.IDLE

    def test_refusing_to_idle(self) -> None:
        result = OperationalState.REFUSING.transition_to(OperationalState.IDLE)
        assert result is OperationalState.IDLE

    def test_suspended_to_idle(self) -> None:
        result = OperationalState.SUSPENDED.transition_to(OperationalState.IDLE)
        assert result is OperationalState.IDLE

    def test_suspended_to_retiring(self) -> None:
        result = OperationalState.SUSPENDED.transition_to(OperationalState.RETIRING)
        assert result is OperationalState.RETIRING

    def test_retiring_to_retired(self) -> None:
        result = OperationalState.RETIRING.transition_to(OperationalState.RETIRED)
        assert result is OperationalState.RETIRED

    # Invalid transitions
    def test_idle_to_waiting_invalid(self) -> None:
        with pytest.raises(ValueError, match="invalid transition"):
            OperationalState.IDLE.transition_to(OperationalState.WAITING)

    def test_idle_to_escalating_invalid(self) -> None:
        with pytest.raises(ValueError, match="invalid transition"):
            OperationalState.IDLE.transition_to(OperationalState.ESCALATING)

    def test_idle_to_retired_invalid(self) -> None:
        with pytest.raises(ValueError, match="invalid transition"):
            OperationalState.IDLE.transition_to(OperationalState.RETIRED)

    def test_retired_is_terminal(self) -> None:
        with pytest.raises(ValueError, match="invalid transition"):
            OperationalState.RETIRED.transition_to(OperationalState.IDLE)

    def test_retired_to_processing_invalid(self) -> None:
        with pytest.raises(ValueError, match="invalid transition"):
            OperationalState.RETIRED.transition_to(OperationalState.PROCESSING)

    def test_refusing_to_processing_invalid(self) -> None:
        with pytest.raises(ValueError, match="invalid transition"):
            OperationalState.REFUSING.transition_to(OperationalState.PROCESSING)

    def test_retiring_to_idle_invalid(self) -> None:
        with pytest.raises(ValueError, match="invalid transition"):
            OperationalState.RETIRING.transition_to(OperationalState.IDLE)

    def test_full_lifecycle(self) -> None:
        """Walk Idle -> Processing -> Waiting -> Processing -> Idle -> Retiring -> Retired."""
        s = OperationalState.IDLE
        s = s.transition_to(OperationalState.PROCESSING)
        s = s.transition_to(OperationalState.WAITING)
        s = s.transition_to(OperationalState.PROCESSING)
        s = s.transition_to(OperationalState.IDLE)
        s = s.transition_to(OperationalState.RETIRING)
        s = s.transition_to(OperationalState.RETIRED)
        assert s.is_terminal


# ===========================================================================
# 2. All 28 Agent Primitives
# ===========================================================================

class TestAllAgentPrimitives:
    def test_count(self) -> None:
        assert len(ALL_AGENT_PRIMITIVE_CLASSES) == 28

    def test_all_primitives_returns_28(self) -> None:
        prims = all_primitives()
        assert len(prims) == 28

    def test_all_unique_ids(self) -> None:
        prims = all_primitives()
        ids = [p.id().value for p in prims]
        assert len(ids) == len(set(ids)), f"duplicate IDs found: {ids}"

    def test_all_at_layer_1(self) -> None:
        for p in all_primitives():
            assert p.layer().value == 1, f"{p.id().value} not at layer 1"

    def test_all_cadence_1(self) -> None:
        for p in all_primitives():
            assert p.cadence().value == 1

    def test_all_have_subscriptions(self) -> None:
        for p in all_primitives():
            subs = p.subscriptions()
            assert len(subs) > 0, f"{p.id().value} has no subscriptions"
            for s in subs:
                assert isinstance(s, SubscriptionPattern)

    def test_all_ids_start_with_agent(self) -> None:
        for p in all_primitives():
            assert p.id().value.startswith("agent."), f"{p.id().value} missing agent. prefix"

    def test_is_agent_primitive(self) -> None:
        for p in all_primitives():
            assert is_agent_primitive(p.id()) is True
        assert is_agent_primitive(PrimitiveID("notanagent")) is False

    def test_isinstance_agent_base(self) -> None:
        for p in all_primitives():
            assert isinstance(p, _AgentBase)


class TestAgentPrimitiveProcess:
    """Test that each primitive can process events and returns mutations."""

    def test_all_process_empty(self) -> None:
        snap = _empty_snapshot()
        for p in all_primitives():
            mutations = p.process(tick=1, events=[], snapshot=snap)
            assert isinstance(mutations, list)
            assert len(mutations) > 0, f"{p.id().value} returned no mutations"
            # All should include a lastTick mutation
            last_ticks = [
                m for m in mutations
                if isinstance(m, UpdateState) and m.key == "lastTick"
            ]
            assert len(last_ticks) == 1, f"{p.id().value} missing lastTick mutation"
            assert last_ticks[0].value == 1

    def test_identity_counts_events(self) -> None:
        from eventgraph.agent import IdentityPrimitive
        p = IdentityPrimitive()
        ev = _make_event("agent.identity.created")
        mutations = p.process(tick=5, events=[ev], snapshot=_empty_snapshot(5))
        state = {m.key: m.value for m in mutations if isinstance(m, UpdateState)}
        assert state["identitiesCreated"] == 1
        assert state["keysRotated"] == 0
        assert state["lastTick"] == 5

    def test_soul_imprint(self) -> None:
        from eventgraph.agent import SoulPrimitive
        p = SoulPrimitive()
        ev = _make_event("agent.soul.imprinted")
        mutations = p.process(tick=3, events=[ev], snapshot=_empty_snapshot(3))
        state = {m.key: m.value for m in mutations if isinstance(m, UpdateState)}
        assert state["imprinted"] is True
        assert state["lastTick"] == 3

    def test_goal_counts(self) -> None:
        from eventgraph.agent import GoalPrimitive
        p = GoalPrimitive()
        evs = [_make_event("agent.goal.set"), _make_event("agent.goal.completed")]
        mutations = p.process(tick=2, events=evs, snapshot=_empty_snapshot(2))
        state = {m.key: m.value for m in mutations if isinstance(m, UpdateState)}
        assert state["goalsSet"] == 1
        assert state["goalsCompleted"] == 1
        assert state["goalsAbandoned"] == 0

    def test_observe_counts_total(self) -> None:
        from eventgraph.agent import ObservePrimitive
        p = ObservePrimitive()
        evs = [_make_event("agent.observed"), _make_event("agent.acted")]
        mutations = p.process(tick=4, events=evs, snapshot=_empty_snapshot(4))
        state = {m.key: m.value for m in mutations if isinstance(m, UpdateState)}
        assert state["eventsObserved"] == 1
        assert state["totalEventsReceived"] == 2

    def test_expect_counts_all_types(self) -> None:
        from eventgraph.agent import ExpectPrimitive
        p = ExpectPrimitive()
        evs = [
            _make_event("agent.expectation.set"),
            _make_event("agent.expectation.met"),
            _make_event("agent.expectation.expired"),
        ]
        mutations = p.process(tick=7, events=evs, snapshot=_empty_snapshot(7))
        state = {m.key: m.value for m in mutations if isinstance(m, UpdateState)}
        assert state["expectationsSet"] == 1
        assert state["expectationsMet"] == 1
        assert state["expectationsExpired"] == 1

    def test_consent_counts(self) -> None:
        from eventgraph.agent import ConsentPrimitive
        p = ConsentPrimitive()
        evs = [
            _make_event("agent.consent.requested"),
            _make_event("agent.consent.granted"),
            _make_event("agent.consent.denied"),
        ]
        mutations = p.process(tick=6, events=evs, snapshot=_empty_snapshot(6))
        state = {m.key: m.value for m in mutations if isinstance(m, UpdateState)}
        assert state["consentRequested"] == 1
        assert state["consentGranted"] == 1
        assert state["consentDenied"] == 1

    def test_composition_counts(self) -> None:
        from eventgraph.agent import CompositionPrimitive
        p = CompositionPrimitive()
        evs = [
            _make_event("agent.composition.formed"),
            _make_event("agent.composition.joined"),
            _make_event("agent.composition.left"),
        ]
        mutations = p.process(tick=8, events=evs, snapshot=_empty_snapshot(8))
        state = {m.key: m.value for m in mutations if isinstance(m, UpdateState)}
        assert state["groupsFormed"] == 1
        assert state["membersJoined"] == 1
        assert state["membersLeft"] == 1
        assert state["groupsDissolved"] == 0

    def test_attenuation_counts(self) -> None:
        from eventgraph.agent import AttenuationPrimitive
        p = AttenuationPrimitive()
        evs = [
            _make_event("agent.attenuated"),
            _make_event("agent.attenuation.lifted"),
            _make_event("agent.budget.exhausted"),
        ]
        mutations = p.process(tick=9, events=evs, snapshot=_empty_snapshot(9))
        state = {m.key: m.value for m in mutations if isinstance(m, UpdateState)}
        assert state["attenuations"] == 1
        assert state["lifts"] == 1
        assert state["budgetTriggered"] == 1


# ===========================================================================
# 3. Registration
# ===========================================================================

class TestRegistration:
    def test_register_all(self) -> None:
        reg = Registry()
        register_all(reg)
        assert reg.count() == 28

    def test_register_all_unique(self) -> None:
        reg = Registry()
        register_all(reg)
        prims = reg.all()
        ids = [p.id().value for p in prims]
        assert len(ids) == len(set(ids))

    def test_all_active_after_register(self) -> None:
        from eventgraph.primitive import LIFECYCLE_ACTIVE
        reg = Registry()
        register_all(reg)
        for p in reg.all():
            assert reg.lifecycle(p.id()) == LIFECYCLE_ACTIVE


# ===========================================================================
# 4. Agent Event Types
# ===========================================================================

class TestAgentEventTypes:
    def test_count(self) -> None:
        assert len(all_agent_event_types()) == 45

    def test_all_start_with_agent(self) -> None:
        for et in all_agent_event_types():
            assert et.value.startswith("agent.")

    def test_all_unique(self) -> None:
        values = [et.value for et in all_agent_event_types()]
        assert len(values) == len(set(values))

    def test_specific_types(self) -> None:
        assert AGENT_IDENTITY_CREATED.value == "agent.identity.created"


# ===========================================================================
# 5. Compositions
# ===========================================================================

class TestCompositions:
    def test_all_compositions_count(self) -> None:
        comps = all_compositions()
        assert len(comps) == 8

    def test_all_unique_names(self) -> None:
        comps = all_compositions()
        names = [c.name for c in comps]
        assert len(names) == len(set(names))

    def test_boot(self) -> None:
        c = boot()
        assert c.name == "Boot"
        assert len(c.primitives) == 5
        assert "agent.Identity" in c.primitives
        assert "agent.Soul" in c.primitives
        assert "agent.Model" in c.primitives
        assert "agent.Authority" in c.primitives
        assert "agent.State" in c.primitives
        assert len(c.events) == 5

    def test_imprint(self) -> None:
        c = imprint()
        assert c.name == "Imprint"
        assert len(c.primitives) == 8
        # Imprint includes Boot primitives plus Observe, Learn, Goal
        assert "agent.Observe" in c.primitives
        assert "agent.Learn" in c.primitives
        assert "agent.Goal" in c.primitives
        assert len(c.events) == 8  # Boot (5) + 3

    def test_task(self) -> None:
        c = task()
        assert c.name == "Task"
        assert len(c.primitives) == 5
        assert len(c.events) == 5

    def test_supervise(self) -> None:
        c = supervise()
        assert c.name == "Supervise"
        assert len(c.primitives) == 5
        assert "agent.Delegate" in c.primitives
        assert "agent.Expect" in c.primitives
        assert len(c.events) == 4

    def test_collaborate(self) -> None:
        c = collaborate()
        assert c.name == "Collaborate"
        assert len(c.primitives) == 5
        assert "agent.Channel" in c.primitives
        assert "agent.Composition" in c.primitives
        assert len(c.events) == 6

    def test_crisis(self) -> None:
        c = crisis()
        assert c.name == "Crisis"
        assert len(c.primitives) == 5
        assert "agent.Attenuation" in c.primitives
        assert "agent.Escalate" in c.primitives
        assert len(c.events) == 5

    def test_retire(self) -> None:
        c = retire()
        assert c.name == "Retire"
        assert len(c.primitives) == 4
        assert "agent.Introspect" in c.primitives
        assert "agent.Lifespan" in c.primitives
        assert len(c.events) == 4

    def test_whistleblow(self) -> None:
        c = whistleblow()
        assert c.name == "Whistleblow"
        assert len(c.primitives) == 5
        assert "agent.Refuse" in c.primitives
        assert "agent.Escalate" in c.primitives
        assert "agent.Communicate" in c.primitives
        assert len(c.events) == 5

    def test_all_composition_primitives_are_valid(self) -> None:
        """Every primitive ID referenced in compositions exists in the 28."""
        valid_ids = {p.id().value for p in all_primitives()}
        for c in all_compositions():
            for pid in c.primitives:
                assert pid in valid_ids, f"composition {c.name!r} references unknown primitive {pid!r}"

    def test_composition_is_frozen(self) -> None:
        c = boot()
        with pytest.raises(AttributeError):
            c.name = "Modified"  # type: ignore[misc]
