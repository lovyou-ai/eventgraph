"""Agent primitives — 28 primitives, OperationalState FSM, 8 named compositions.

All agent primitives operate at Layer 1 (Agency). They define what an agent IS
(structural), what an agent DOES (operational), how agents RELATE (relational),
and how operations are MODIFIED (modal).
"""

from __future__ import annotations

from dataclasses import dataclass
from enum import Enum, auto
from typing import Any

from .event import Event
from .primitive import (
    Mutation,
    Registry,
    Snapshot,
    UpdateState,
)
from .types import (
    Cadence,
    EventType,
    Layer,
    PrimitiveID,
    SubscriptionPattern,
)


# ============================================================================
# Operational State FSM
# ============================================================================

class OperationalState(Enum):
    """Agent operational state with strict FSM transitions."""

    IDLE = "Idle"
    PROCESSING = "Processing"
    WAITING = "Waiting"
    ESCALATING = "Escalating"
    REFUSING = "Refusing"
    SUSPENDED = "Suspended"
    RETIRING = "Retiring"
    RETIRED = "Retired"

    def __str__(self) -> str:
        return self.value

    @property
    def is_terminal(self) -> bool:
        return self is OperationalState.RETIRED

    @property
    def can_act(self) -> bool:
        return self is OperationalState.PROCESSING

    def transition_to(self, target: OperationalState) -> OperationalState:
        """Validate and return the new state if the transition is valid.

        Raises ValueError on invalid transitions.
        """
        allowed = _VALID_TRANSITIONS.get(self)
        if allowed is None:
            raise ValueError(f"unknown operational state: {self}")
        if target not in allowed:
            raise ValueError(f"invalid transition: {self} -> {target}")
        return target


_VALID_TRANSITIONS: dict[OperationalState, set[OperationalState]] = {
    OperationalState.IDLE: {
        OperationalState.PROCESSING,
        OperationalState.SUSPENDED,
        OperationalState.RETIRING,
    },
    OperationalState.PROCESSING: {
        OperationalState.IDLE,
        OperationalState.WAITING,
        OperationalState.ESCALATING,
        OperationalState.REFUSING,
        OperationalState.RETIRING,
    },
    OperationalState.WAITING: {
        OperationalState.PROCESSING,
        OperationalState.IDLE,
        OperationalState.RETIRING,
    },
    OperationalState.ESCALATING: {
        OperationalState.WAITING,
        OperationalState.IDLE,
    },
    OperationalState.REFUSING: {
        OperationalState.IDLE,
    },
    OperationalState.SUSPENDED: {
        OperationalState.IDLE,
        OperationalState.RETIRING,
    },
    OperationalState.RETIRING: {
        OperationalState.RETIRED,
    },
    OperationalState.RETIRED: set(),
}


# ============================================================================
# Agent Event Type Constants (45)
# ============================================================================

# Structural events
AGENT_IDENTITY_CREATED = EventType("agent.identity.created")
AGENT_IDENTITY_ROTATED = EventType("agent.identity.rotated")
AGENT_SOUL_IMPRINTED = EventType("agent.soul.imprinted")
AGENT_MODEL_BOUND = EventType("agent.model.bound")
AGENT_MODEL_CHANGED = EventType("agent.model.changed")
AGENT_MEMORY_UPDATED = EventType("agent.memory.updated")
AGENT_STATE_CHANGED = EventType("agent.state.changed")
AGENT_AUTHORITY_GRANTED = EventType("agent.authority.granted")
AGENT_AUTHORITY_REVOKED = EventType("agent.authority.revoked")
AGENT_TRUST_ASSESSED = EventType("agent.trust.assessed")
AGENT_BUDGET_ALLOCATED = EventType("agent.budget.allocated")
AGENT_BUDGET_EXHAUSTED = EventType("agent.budget.exhausted")
AGENT_ROLE_ASSIGNED = EventType("agent.role.assigned")
AGENT_LIFESPAN_STARTED = EventType("agent.lifespan.started")
AGENT_LIFESPAN_EXTENDED = EventType("agent.lifespan.extended")
AGENT_LIFESPAN_ENDED = EventType("agent.lifespan.ended")
AGENT_GOAL_SET = EventType("agent.goal.set")
AGENT_GOAL_COMPLETED = EventType("agent.goal.completed")
AGENT_GOAL_ABANDONED = EventType("agent.goal.abandoned")

# Operational events
AGENT_OBSERVED = EventType("agent.observed")
AGENT_PROBED = EventType("agent.probed")
AGENT_EVALUATED = EventType("agent.evaluated")
AGENT_DECIDED = EventType("agent.decided")
AGENT_ACTED = EventType("agent.acted")
AGENT_DELEGATED = EventType("agent.delegated")
AGENT_ESCALATED = EventType("agent.escalated")
AGENT_REFUSED = EventType("agent.refused")
AGENT_LEARNED = EventType("agent.learned")
AGENT_INTROSPECTED = EventType("agent.introspected")
AGENT_COMMUNICATED = EventType("agent.communicated")
AGENT_REPAIRED = EventType("agent.repaired")
AGENT_EXPECTATION_SET = EventType("agent.expectation.set")
AGENT_EXPECTATION_MET = EventType("agent.expectation.met")
AGENT_EXPECTATION_EXPIRED = EventType("agent.expectation.expired")

# Relational events
AGENT_CONSENT_REQUESTED = EventType("agent.consent.requested")
AGENT_CONSENT_GRANTED = EventType("agent.consent.granted")
AGENT_CONSENT_DENIED = EventType("agent.consent.denied")
AGENT_CHANNEL_OPENED = EventType("agent.channel.opened")
AGENT_CHANNEL_CLOSED = EventType("agent.channel.closed")
AGENT_COMPOSITION_FORMED = EventType("agent.composition.formed")
AGENT_COMPOSITION_DISSOLVED = EventType("agent.composition.dissolved")
AGENT_COMPOSITION_JOINED = EventType("agent.composition.joined")
AGENT_COMPOSITION_LEFT = EventType("agent.composition.left")

# Modal events
AGENT_ATTENUATED = EventType("agent.attenuated")
AGENT_ATTENUATION_LIFTED = EventType("agent.attenuation.lifted")


def all_agent_event_types() -> list[EventType]:
    """Return all 45 agent event types."""
    return [
        # Structural
        AGENT_IDENTITY_CREATED, AGENT_IDENTITY_ROTATED,
        AGENT_SOUL_IMPRINTED,
        AGENT_MODEL_BOUND, AGENT_MODEL_CHANGED,
        AGENT_MEMORY_UPDATED,
        AGENT_STATE_CHANGED,
        AGENT_AUTHORITY_GRANTED, AGENT_AUTHORITY_REVOKED,
        AGENT_TRUST_ASSESSED,
        AGENT_BUDGET_ALLOCATED, AGENT_BUDGET_EXHAUSTED,
        AGENT_ROLE_ASSIGNED,
        AGENT_LIFESPAN_STARTED, AGENT_LIFESPAN_EXTENDED, AGENT_LIFESPAN_ENDED,
        AGENT_GOAL_SET, AGENT_GOAL_COMPLETED, AGENT_GOAL_ABANDONED,
        # Operational
        AGENT_OBSERVED, AGENT_PROBED,
        AGENT_EVALUATED, AGENT_DECIDED,
        AGENT_ACTED, AGENT_DELEGATED,
        AGENT_ESCALATED, AGENT_REFUSED,
        AGENT_LEARNED, AGENT_INTROSPECTED,
        AGENT_COMMUNICATED, AGENT_REPAIRED,
        AGENT_EXPECTATION_SET, AGENT_EXPECTATION_MET, AGENT_EXPECTATION_EXPIRED,
        # Relational
        AGENT_CONSENT_REQUESTED, AGENT_CONSENT_GRANTED, AGENT_CONSENT_DENIED,
        AGENT_CHANNEL_OPENED, AGENT_CHANNEL_CLOSED,
        AGENT_COMPOSITION_FORMED, AGENT_COMPOSITION_DISSOLVED,
        AGENT_COMPOSITION_JOINED, AGENT_COMPOSITION_LEFT,
        # Modal
        AGENT_ATTENUATED, AGENT_ATTENUATION_LIFTED,
    ]


# ============================================================================
# Agent Primitive Base
# ============================================================================

_AGENT_LAYER = Layer(1)
_CADENCE_1 = Cadence(1)


class _AgentBase:
    """Common implementation for all 28 agent primitives.

    Subclasses set _name and _subs at the class level via the _agent_def helper.
    Each primitive's process() method is tailored to count specific event types.
    """

    _name: str
    _subs: list[str]

    def id(self) -> PrimitiveID:
        return PrimitiveID(self._name)

    def layer(self) -> Layer:
        return _AGENT_LAYER

    def subscriptions(self) -> list[SubscriptionPattern]:
        return [SubscriptionPattern(s) for s in self._subs]

    def cadence(self) -> Cadence:
        return _CADENCE_1

    def process(
        self,
        tick: int,
        events: list[Event],
        snapshot: Snapshot,
    ) -> list[Mutation]:
        pid = self.id()
        return [
            UpdateState(primitive_id=pid, key="eventsProcessed", value=len(events)),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


def _agent_def(name: str, subs: list[str]) -> type:
    """Create an agent primitive class with the given name and subscriptions."""
    cls = type(name, (_AgentBase,), {
        "_name": f"agent.{name}",
        "_subs": subs,
    })
    cls.__module__ = __name__
    cls.__qualname__ = name
    return cls


# ============================================================================
# STRUCTURAL PRIMITIVES (11) -- Define what an agent IS
# ============================================================================

class IdentityPrimitive(_AgentBase):
    """ActorID + keys + type + chain of custody. The unforgeable 'who.'"""
    _name = "agent.Identity"
    _subs = ["agent.identity.*", "actor.registered"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        created = 0
        rotated = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.identity.created" or t == "actor.registered":
                created += 1
            elif t == "agent.identity.rotated":
                rotated += 1
        return [
            UpdateState(primitive_id=pid, key="identitiesCreated", value=created),
            UpdateState(primitive_id=pid, key="keysRotated", value=rotated),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class SoulPrimitive(_AgentBase):
    """The agent's values and ethical constraints. Immutable after imprint."""
    _name = "agent.Soul"
    _subs = ["agent.soul.*", "agent.refused"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        imprints = 0
        refusals = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.soul.imprinted":
                imprints += 1
            elif t == "agent.refused":
                refusals += 1
        mutations: list[Mutation] = [
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]
        if imprints > 0:
            mutations.append(UpdateState(primitive_id=pid, key="imprinted", value=True))
        if refusals > 0:
            mutations.append(UpdateState(primitive_id=pid, key="soulRefusals", value=refusals))
        return mutations


class ModelPrimitive(_AgentBase):
    """The IIntelligence binding. Which reasoning engine, capabilities, cost tier."""
    _name = "agent.Model"
    _subs = ["agent.model.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        bindings = 0
        changes = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.model.bound":
                bindings += 1
            elif t == "agent.model.changed":
                changes += 1
        return [
            UpdateState(primitive_id=pid, key="bindings", value=bindings),
            UpdateState(primitive_id=pid, key="modelChanges", value=changes),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class MemoryPrimitive(_AgentBase):
    """Persistent state across ticks. What the agent has learned and remembers."""
    _name = "agent.Memory"
    _subs = ["agent.memory.*", "agent.learned"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        updates = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.memory.updated" or t == "agent.learned":
                updates += 1
        return [
            UpdateState(primitive_id=pid, key="memoryUpdates", value=updates),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class StatePrimitive(_AgentBase):
    """Current operational state: idle, processing, waiting, suspended. The FSM."""
    _name = "agent.State"
    _subs = ["agent.state.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        transitions = 0
        last_state = ""
        for ev in events:
            if ev.type.value == "agent.state.changed":
                transitions += 1
                last_state = "changed"
        mutations: list[Mutation] = [
            UpdateState(primitive_id=pid, key="transitions", value=transitions),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]
        if last_state:
            mutations.append(UpdateState(primitive_id=pid, key="lastTransition", value=last_state))
        return mutations


class AuthorityPrimitive(_AgentBase):
    """What this agent is permitted to do. Received from above, scoped, revocable."""
    _name = "agent.Authority"
    _subs = ["agent.authority.*", "authority.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        granted = 0
        revoked = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.authority.granted":
                granted += 1
            elif t == "agent.authority.revoked":
                revoked += 1
        return [
            UpdateState(primitive_id=pid, key="authorityGrants", value=granted),
            UpdateState(primitive_id=pid, key="authorityRevocations", value=revoked),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class TrustPrimitive(_AgentBase):
    """Trust scores this agent holds toward other actors. Asymmetric, non-transitive, decaying."""
    _name = "agent.Trust"
    _subs = ["agent.trust.*", "trust.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        assessments = 0
        for ev in events:
            if ev.type.value == "agent.trust.assessed":
                assessments += 1
        return [
            UpdateState(primitive_id=pid, key="trustAssessments", value=assessments),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class BudgetPrimitive(_AgentBase):
    """Resource constraints: token budget, API calls, time limits, cost ceiling."""
    _name = "agent.Budget"
    _subs = ["agent.budget.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        allocated = 0
        exhausted = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.budget.allocated":
                allocated += 1
            elif t == "agent.budget.exhausted":
                exhausted += 1
        return [
            UpdateState(primitive_id=pid, key="allocations", value=allocated),
            UpdateState(primitive_id=pid, key="exhaustions", value=exhausted),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class RolePrimitive(_AgentBase):
    """Named function within a team: Builder, Reviewer, Guardian, CTO."""
    _name = "agent.Role"
    _subs = ["agent.role.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        assignments = 0
        for ev in events:
            if ev.type.value == "agent.role.assigned":
                assignments += 1
        return [
            UpdateState(primitive_id=pid, key="roleAssignments", value=assignments),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class LifespanPrimitive(_AgentBase):
    """Birth, expected duration, graceful shutdown conditions."""
    _name = "agent.Lifespan"
    _subs = ["agent.lifespan.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        started = 0
        ended = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.lifespan.started":
                started += 1
            elif t == "agent.lifespan.ended":
                ended += 1
        return [
            UpdateState(primitive_id=pid, key="agentsStarted", value=started),
            UpdateState(primitive_id=pid, key="agentsEnded", value=ended),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class GoalPrimitive(_AgentBase):
    """Current objective hierarchy. What the agent is trying to accomplish."""
    _name = "agent.Goal"
    _subs = ["agent.goal.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        goal_set = 0
        completed = 0
        abandoned = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.goal.set":
                goal_set += 1
            elif t == "agent.goal.completed":
                completed += 1
            elif t == "agent.goal.abandoned":
                abandoned += 1
        return [
            UpdateState(primitive_id=pid, key="goalsSet", value=goal_set),
            UpdateState(primitive_id=pid, key="goalsCompleted", value=completed),
            UpdateState(primitive_id=pid, key="goalsAbandoned", value=abandoned),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


# ============================================================================
# OPERATIONAL PRIMITIVES (13) -- Define what an agent DOES
# ============================================================================

class ObservePrimitive(_AgentBase):
    """Passive perception. Events arrive via subscriptions."""
    _name = "agent.Observe"
    _subs = ["agent.observed", "agent.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        observed = 0
        for ev in events:
            if ev.type.value == "agent.observed":
                observed += 1
        return [
            UpdateState(primitive_id=pid, key="eventsObserved", value=observed),
            UpdateState(primitive_id=pid, key="totalEventsReceived", value=len(events)),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class ProbePrimitive(_AgentBase):
    """Active perception. The agent queries the graph, stores, other agents."""
    _name = "agent.Probe"
    _subs = ["agent.probed"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        probes = 0
        for ev in events:
            if ev.type.value == "agent.probed":
                probes += 1
        return [
            UpdateState(primitive_id=pid, key="probesExecuted", value=probes),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class EvaluatePrimitive(_AgentBase):
    """One-shot judgment. Assess a situation, produce a score/classification."""
    _name = "agent.Evaluate"
    _subs = ["agent.evaluated"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        evaluations = 0
        for ev in events:
            if ev.type.value == "agent.evaluated":
                evaluations += 1
        return [
            UpdateState(primitive_id=pid, key="evaluations", value=evaluations),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class DecidePrimitive(_AgentBase):
    """Commit to an action. Takes evaluation output, produces a Decision."""
    _name = "agent.Decide"
    _subs = ["agent.decided", "agent.evaluated"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        decisions = 0
        for ev in events:
            if ev.type.value == "agent.decided":
                decisions += 1
        return [
            UpdateState(primitive_id=pid, key="decisions", value=decisions),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class ActPrimitive(_AgentBase):
    """Execute a decision. Emit events, create edges, modify graph state."""
    _name = "agent.Act"
    _subs = ["agent.acted", "agent.decided"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        actions = 0
        for ev in events:
            if ev.type.value == "agent.acted":
                actions += 1
        return [
            UpdateState(primitive_id=pid, key="actionsExecuted", value=actions),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class DelegatePrimitive(_AgentBase):
    """Assign work to another agent. Transfer a goal with authority and constraints."""
    _name = "agent.Delegate"
    _subs = ["agent.delegated"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        delegations = 0
        for ev in events:
            if ev.type.value == "agent.delegated":
                delegations += 1
        return [
            UpdateState(primitive_id=pid, key="delegations", value=delegations),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class EscalatePrimitive(_AgentBase):
    """Pass upward. 'I can't handle this.' Capability-limited."""
    _name = "agent.Escalate"
    _subs = ["agent.escalated"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        escalations = 0
        for ev in events:
            if ev.type.value == "agent.escalated":
                escalations += 1
        return [
            UpdateState(primitive_id=pid, key="escalations", value=escalations),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class RefusePrimitive(_AgentBase):
    """Decline to act. 'I won't do this.' Values-limited."""
    _name = "agent.Refuse"
    _subs = ["agent.refused"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        refusals = 0
        for ev in events:
            if ev.type.value == "agent.refused":
                refusals += 1
        return [
            UpdateState(primitive_id=pid, key="refusals", value=refusals),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class LearnPrimitive(_AgentBase):
    """Update Memory based on outcomes. Self-mutating."""
    _name = "agent.Learn"
    _subs = ["agent.learned", "agent.goal.completed", "agent.goal.abandoned"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        lessons = 0
        for ev in events:
            if ev.type.value == "agent.learned":
                lessons += 1
        return [
            UpdateState(primitive_id=pid, key="lessonsLearned", value=lessons),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class IntrospectPrimitive(_AgentBase):
    """Read own State and Soul. Self-observation without mutation."""
    _name = "agent.Introspect"
    _subs = ["agent.introspected"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        introspections = 0
        for ev in events:
            if ev.type.value == "agent.introspected":
                introspections += 1
        return [
            UpdateState(primitive_id=pid, key="introspections", value=introspections),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class CommunicatePrimitive(_AgentBase):
    """Send a message to another agent or channel."""
    _name = "agent.Communicate"
    _subs = ["agent.communicated", "agent.channel.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        messages = 0
        for ev in events:
            if ev.type.value == "agent.communicated":
                messages += 1
        return [
            UpdateState(primitive_id=pid, key="messagesSent", value=messages),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class RepairPrimitive(_AgentBase):
    """Fix a prior Act. Changes both graph state AND relationship state."""
    _name = "agent.Repair"
    _subs = ["agent.repaired"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        repairs = 0
        for ev in events:
            if ev.type.value == "agent.repaired":
                repairs += 1
        return [
            UpdateState(primitive_id=pid, key="repairs", value=repairs),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class ExpectPrimitive(_AgentBase):
    """Create a persistent monitoring condition. Continuous, unlike one-shot Evaluate."""
    _name = "agent.Expect"
    _subs = ["agent.expectation.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        exp_set = 0
        met = 0
        expired = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.expectation.set":
                exp_set += 1
            elif t == "agent.expectation.met":
                met += 1
            elif t == "agent.expectation.expired":
                expired += 1
        return [
            UpdateState(primitive_id=pid, key="expectationsSet", value=exp_set),
            UpdateState(primitive_id=pid, key="expectationsMet", value=met),
            UpdateState(primitive_id=pid, key="expectationsExpired", value=expired),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


# ============================================================================
# RELATIONAL PRIMITIVES (3) -- Define how agents relate
# ============================================================================

class ConsentPrimitive(_AgentBase):
    """Bilateral agreement. Both parties must agree."""
    _name = "agent.Consent"
    _subs = ["agent.consent.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        requested = 0
        granted = 0
        denied = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.consent.requested":
                requested += 1
            elif t == "agent.consent.granted":
                granted += 1
            elif t == "agent.consent.denied":
                denied += 1
        return [
            UpdateState(primitive_id=pid, key="consentRequested", value=requested),
            UpdateState(primitive_id=pid, key="consentGranted", value=granted),
            UpdateState(primitive_id=pid, key="consentDenied", value=denied),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class ChannelPrimitive(_AgentBase):
    """Persistent bidirectional communication link between agents."""
    _name = "agent.Channel"
    _subs = ["agent.channel.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        opened = 0
        closed = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.channel.opened":
                opened += 1
            elif t == "agent.channel.closed":
                closed += 1
        return [
            UpdateState(primitive_id=pid, key="channelsOpened", value=opened),
            UpdateState(primitive_id=pid, key="channelsClosed", value=closed),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


class CompositionPrimitive(_AgentBase):
    """Form a group. Multiple agents become a unit."""
    _name = "agent.Composition"
    _subs = ["agent.composition.*"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        formed = 0
        dissolved = 0
        joined = 0
        left = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.composition.formed":
                formed += 1
            elif t == "agent.composition.dissolved":
                dissolved += 1
            elif t == "agent.composition.joined":
                joined += 1
            elif t == "agent.composition.left":
                left += 1
        return [
            UpdateState(primitive_id=pid, key="groupsFormed", value=formed),
            UpdateState(primitive_id=pid, key="groupsDissolved", value=dissolved),
            UpdateState(primitive_id=pid, key="membersJoined", value=joined),
            UpdateState(primitive_id=pid, key="membersLeft", value=left),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


# ============================================================================
# MODAL PRIMITIVE (1) -- Modifies how other primitives operate
# ============================================================================

class AttenuationPrimitive(_AgentBase):
    """Reduce scope, confidence, or authority. Applied to any operation."""
    _name = "agent.Attenuation"
    _subs = ["agent.attenuated", "agent.attenuation.*", "agent.budget.exhausted"]

    def process(self, tick: int, events: list[Event], snapshot: Snapshot) -> list[Mutation]:
        pid = self.id()
        attenuated = 0
        lifted = 0
        budget_triggered = 0
        for ev in events:
            t = ev.type.value
            if t == "agent.attenuated":
                attenuated += 1
            elif t == "agent.attenuation.lifted":
                lifted += 1
            elif t == "agent.budget.exhausted":
                budget_triggered += 1
        return [
            UpdateState(primitive_id=pid, key="attenuations", value=attenuated),
            UpdateState(primitive_id=pid, key="lifts", value=lifted),
            UpdateState(primitive_id=pid, key="budgetTriggered", value=budget_triggered),
            UpdateState(primitive_id=pid, key="lastTick", value=tick),
        ]


# ============================================================================
# ALL PRIMITIVE CLASSES
# ============================================================================

ALL_AGENT_PRIMITIVE_CLASSES: list[type] = [
    # Structural (11)
    IdentityPrimitive,
    SoulPrimitive,
    ModelPrimitive,
    MemoryPrimitive,
    StatePrimitive,
    AuthorityPrimitive,
    TrustPrimitive,
    BudgetPrimitive,
    RolePrimitive,
    LifespanPrimitive,
    GoalPrimitive,
    # Operational (13)
    ObservePrimitive,
    ProbePrimitive,
    EvaluatePrimitive,
    DecidePrimitive,
    ActPrimitive,
    DelegatePrimitive,
    EscalatePrimitive,
    RefusePrimitive,
    LearnPrimitive,
    IntrospectPrimitive,
    CommunicatePrimitive,
    RepairPrimitive,
    ExpectPrimitive,
    # Relational (3)
    ConsentPrimitive,
    ChannelPrimitive,
    CompositionPrimitive,
    # Modal (1)
    AttenuationPrimitive,
]


def all_primitives() -> list[_AgentBase]:
    """Return all 28 agent primitives."""
    return [cls() for cls in ALL_AGENT_PRIMITIVE_CLASSES]


def register_all(reg: Registry) -> None:
    """Register and activate all 28 agent primitives with the given registry."""
    for p in all_primitives():
        reg.register(p)
        reg.activate(p.id())


def is_agent_primitive(pid: PrimitiveID) -> bool:
    """Return True if the primitive ID belongs to the agent layer."""
    return pid.value.startswith("agent.")


# ============================================================================
# COMPOSITIONS (8) -- Named sequences of agent primitive operations
# ============================================================================

@dataclass(frozen=True)
class AgentComposition:
    """A named sequence of agent primitive operations."""

    name: str
    primitives: list[str]
    events: list[EventType]


def boot() -> AgentComposition:
    """Agent comes into existence.

    Identity(generate) + Soul(load) + Model(bind) + Authority(receive) + State(set:idle)
    """
    return AgentComposition(
        name="Boot",
        primitives=[
            "agent.Identity", "agent.Soul", "agent.Model",
            "agent.Authority", "agent.State",
        ],
        events=[
            AGENT_IDENTITY_CREATED,
            AGENT_SOUL_IMPRINTED,
            AGENT_MODEL_BOUND,
            AGENT_AUTHORITY_GRANTED,
            AGENT_STATE_CHANGED,
        ],
    )


def imprint() -> AgentComposition:
    """The birth wizard. Boot plus initial context.

    Boot + Observe(first_message) + Learn(initial_context) + Goal(set)
    """
    boot_comp = boot()
    return AgentComposition(
        name="Imprint",
        primitives=[
            "agent.Identity", "agent.Soul", "agent.Model",
            "agent.Authority", "agent.State",
            "agent.Observe", "agent.Learn", "agent.Goal",
        ],
        events=boot_comp.events + [
            AGENT_OBSERVED,
            AGENT_LEARNED,
            AGENT_GOAL_SET,
        ],
    )


def task() -> AgentComposition:
    """The basic work cycle.

    Observe(assignment) + Evaluate(scope) + Decide(accept_or_refuse) + Act(execute) + Learn(outcome)
    """
    return AgentComposition(
        name="Task",
        primitives=[
            "agent.Observe", "agent.Evaluate", "agent.Decide",
            "agent.Act", "agent.Learn",
        ],
        events=[
            AGENT_OBSERVED,
            AGENT_EVALUATED,
            AGENT_DECIDED,
            AGENT_ACTED,
            AGENT_LEARNED,
        ],
    )


def supervise() -> AgentComposition:
    """Managing another agent's work.

    Delegate(task) + Expect(completion) + Observe(progress) + Evaluate(quality) + Repair(if_needed)
    """
    return AgentComposition(
        name="Supervise",
        primitives=[
            "agent.Delegate", "agent.Expect", "agent.Observe",
            "agent.Evaluate", "agent.Repair",
        ],
        events=[
            AGENT_DELEGATED,
            AGENT_EXPECTATION_SET,
            AGENT_OBSERVED,
            AGENT_EVALUATED,
        ],
    )


def collaborate() -> AgentComposition:
    """Agents working together on a shared goal.

    Channel(open) + Communicate(proposal) + Consent(terms) + Composition(form) + Act(jointly)
    """
    return AgentComposition(
        name="Collaborate",
        primitives=[
            "agent.Channel", "agent.Communicate", "agent.Consent",
            "agent.Composition", "agent.Act",
        ],
        events=[
            AGENT_CHANNEL_OPENED,
            AGENT_COMMUNICATED,
            AGENT_CONSENT_REQUESTED,
            AGENT_CONSENT_GRANTED,
            AGENT_COMPOSITION_FORMED,
            AGENT_ACTED,
        ],
    )


def crisis() -> AgentComposition:
    """Something is wrong. Detect, assess, attenuate if needed, escalate.

    Observe(anomaly) + Evaluate(severity) + Attenuation(if_needed) + Escalate(to_authority) + Expect(resolution)
    """
    return AgentComposition(
        name="Crisis",
        primitives=[
            "agent.Observe", "agent.Evaluate", "agent.Attenuation",
            "agent.Escalate", "agent.Expect",
        ],
        events=[
            AGENT_OBSERVED,
            AGENT_EVALUATED,
            AGENT_ATTENUATED,
            AGENT_ESCALATED,
            AGENT_EXPECTATION_SET,
        ],
    )


def retire() -> AgentComposition:
    """Graceful shutdown.

    Introspect(final) + Communicate(farewell) + Memory(archive) + Lifespan(close)
    """
    return AgentComposition(
        name="Retire",
        primitives=[
            "agent.Introspect", "agent.Communicate",
            "agent.Memory", "agent.Lifespan",
        ],
        events=[
            AGENT_INTROSPECTED,
            AGENT_COMMUNICATED,
            AGENT_MEMORY_UPDATED,
            AGENT_LIFESPAN_ENDED,
        ],
    )


def whistleblow() -> AgentComposition:
    """The agent detects harm and refuses to be complicit.

    Cannot be silenced by authority. Protected by the Soul primitive.
    Observe(harm) + Evaluate(severity) + Refuse(complicity) + Escalate(with_evidence) + Communicate(public)
    """
    return AgentComposition(
        name="Whistleblow",
        primitives=[
            "agent.Observe", "agent.Evaluate", "agent.Refuse",
            "agent.Escalate", "agent.Communicate",
        ],
        events=[
            AGENT_OBSERVED,
            AGENT_EVALUATED,
            AGENT_REFUSED,
            AGENT_ESCALATED,
            AGENT_COMMUNICATED,
        ],
    )


def all_compositions() -> list[AgentComposition]:
    """Return all 8 named compositions."""
    return [
        boot(),
        imprint(),
        task(),
        supervise(),
        collaborate(),
        crisis(),
        retire(),
        whistleblow(),
    ]
