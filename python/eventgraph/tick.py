"""Tick engine — ripple-wave processor."""

from __future__ import annotations

import threading
import time
from dataclasses import dataclass, field
from typing import Any, Callable

from .event import Event, NoopSigner, create_event
from .primitive import (
    LIFECYCLE_ACTIVE,
    LIFECYCLE_EMITTING,
    LIFECYCLE_PROCESSING,
    AddEvent,
    Mutation,
    Primitive,
    Registry,
    Snapshot,
    UpdateActivation,
    UpdateLifecycle,
    UpdateState,
)
from .store import InMemoryStore
from .types import (
    ActorID,
    ConversationID,
    EventType,
    Hash,
    Layer,
    NonEmpty,
    PrimitiveID,
    SubscriptionPattern,
)


@dataclass(frozen=True)
class TickConfig:
    max_waves_per_tick: int = 10


@dataclass(frozen=True)
class TickResult:
    tick: int
    waves: int
    mutations: int
    quiesced: bool
    duration_ms: float
    errors: list[str] = field(default_factory=list)


class TickEngine:
    """Ripple-wave tick processor.

    Each tick:
    1. Snapshot all primitive states
    2. Distribute pending events to subscribing primitives
    3. Invoke each primitive's process() (subject to cadence + lifecycle + layer constraint)
    4. Collect mutations
    5. Eagerly apply AddEvent mutations — new events become input for next wave
    6. Defer non-AddEvent mutations to end of tick
    7. Repeat until quiescence or max waves
    """

    def __init__(
        self,
        registry: Registry,
        store: InMemoryStore,
        config: TickConfig | None = None,
        publisher: Callable[[Event], None] | None = None,
    ) -> None:
        self._lock = threading.Lock()
        self._registry = registry
        self._store = store
        self._config = config or TickConfig()
        self._publisher = publisher
        self._current_tick = 0
        self._signer = NoopSigner()

    def tick(self, pending_events: list[Event] | None = None) -> TickResult:
        """Run a single tick. Returns the result."""
        with self._lock:
            start = time.monotonic()
            self._current_tick += 1
            tick_num = self._current_tick

            wave_events = list(pending_events or [])
            total_mutations = 0
            deferred_mutations: list[Mutation] = []
            errors: list[str] = []
            quiesced = False
            invoked_this_tick: set[str] = set()
            waves_run = 0

            # Build initial snapshot
            snapshot = Snapshot(
                tick=tick_num,
                primitives=self._registry.all_states(),
                pending_events=list(wave_events),
                recent_events=self._store.recent(100),
            )

            for wave in range(self._config.max_waves_per_tick):
                wave_mutations, wave_errors = self._run_wave(
                    tick_num, wave, wave_events, snapshot, invoked_this_tick,
                )
                errors.extend(wave_errors)
                waves_run = wave + 1

                if not wave_mutations:
                    quiesced = True
                    break

                # Eagerly apply AddEvent mutations; defer the rest
                new_events, deferred, apply_errors = self._apply_eager_mutations(
                    wave_mutations
                )
                errors.extend(apply_errors)

                if not apply_errors:
                    deferred_mutations.extend(deferred)

                total_mutations += len(new_events)

                if not new_events:
                    if not apply_errors:
                        quiesced = True
                    break

                wave_events = new_events

                # Refresh snapshot between waves
                snapshot = Snapshot(
                    tick=tick_num,
                    primitives=self._registry.all_states(),
                    pending_events=list(wave_events),
                    recent_events=self._store.recent(100),
                )

            # Apply deferred (non-AddEvent) mutations at end of tick
            deferred_errors = self._apply_deferred_mutations(deferred_mutations)
            errors.extend(deferred_errors)
            total_mutations += len(deferred_mutations) - len(deferred_errors)

            elapsed = (time.monotonic() - start) * 1000
            return TickResult(
                tick=tick_num,
                waves=waves_run,
                mutations=total_mutations,
                quiesced=quiesced,
                duration_ms=elapsed,
                errors=errors,
            )

    def _run_wave(
        self,
        tick_num: int,
        wave: int,
        events: list[Event],
        snapshot: Snapshot,
        invoked_this_tick: set[str],
    ) -> tuple[list[Mutation], list[str]]:
        """Run a single wave. Returns (mutations, errors)."""
        eligible = self._eligible_primitives(tick_num, snapshot, invoked_this_tick)

        # Group by layer
        by_layer: dict[int, list[Primitive]] = {}
        for prim in eligible:
            layer_val = prim.layer().value
            by_layer.setdefault(layer_val, []).append(prim)

        layers = sorted(by_layer.keys())

        all_mutations: list[Mutation] = []
        wave_errors: list[str] = []

        for layer in layers:
            prims = by_layer[layer]

            for prim in prims:
                pid = prim.id()
                subs = prim.subscriptions()

                matched = _filter_events(events, subs)

                # On subsequent waves, only invoke primitives with matching events
                if not matched and pid.value in invoked_this_tick:
                    continue

                # Transition to Processing
                try:
                    self._registry.set_lifecycle(pid, LIFECYCLE_PROCESSING)
                except ValueError:
                    continue

                process_err = None
                try:
                    mutations = prim.process(tick_num, matched, snapshot)
                    all_mutations.extend(mutations)
                except Exception as e:
                    process_err = str(e)
                    wave_errors.append(f"{pid.value}: {e}")

                # Lifecycle transitions
                try:
                    if process_err is None:
                        if all_mutations:
                            self._registry.set_lifecycle(pid, LIFECYCLE_EMITTING)
                            self._registry.set_lifecycle(pid, LIFECYCLE_ACTIVE)
                        else:
                            self._registry.set_lifecycle(pid, LIFECYCLE_ACTIVE)
                        invoked_this_tick.add(pid.value)
                        self._registry.set_last_tick(pid, tick_num)
                    else:
                        # Restore to Active on error
                        self._registry.set_lifecycle(pid, LIFECYCLE_ACTIVE)
                except ValueError as e:
                    wave_errors.append(f"{pid.value} lifecycle: {e}")

        return all_mutations, wave_errors

    def _eligible_primitives(
        self,
        tick_num: int,
        snapshot: Snapshot,
        invoked_this_tick: set[str],
    ) -> list[Primitive]:
        """Return primitives eligible for this wave."""
        eligible = []

        for prim in self._registry.all():
            pid = prim.id()

            # Must be Active
            if self._registry.lifecycle(pid) != LIFECYCLE_ACTIVE:
                continue

            # Cadence gating — only on first invocation per tick
            if pid.value not in invoked_this_tick:
                last = self._registry.last_tick(pid)
                if tick_num - last < prim.cadence().value:
                    continue

            # Layer constraint
            if not _layer_stable(prim.layer(), snapshot):
                continue

            eligible.append(prim)

        return eligible

    def _apply_eager_mutations(
        self, mutations: list[Mutation]
    ) -> tuple[list[Event], list[Mutation], list[str]]:
        """Eagerly persist AddEvent mutations between waves.
        Non-AddEvent mutations are returned for deferred application."""
        new_events: list[Event] = []
        deferred: list[Mutation] = []
        errors: list[str] = []

        for m in mutations:
            if isinstance(m, AddEvent):
                try:
                    head = self._store.head()
                    prev_hash = head.unwrap().hash if head.is_some() else Hash.zero()
                    event = create_event(
                        event_type=m.type,
                        source=m.source,
                        content=m.content,
                        causes=m.causes,
                        conversation_id=m.conversation_id,
                        prev_hash=prev_hash,
                        signer=self._signer,
                    )
                    self._store.append(event)
                    if self._publisher:
                        self._publisher(event)
                    new_events.append(event)
                except Exception as e:
                    errors.append(f"AddEvent: {e}")
            else:
                deferred.append(m)

        return new_events, deferred, errors

    def _apply_deferred_mutations(self, mutations: list[Mutation]) -> list[str]:
        """Apply deferred (non-AddEvent) mutations at end of tick."""
        errors: list[str] = []
        for m in mutations:
            try:
                if isinstance(m, UpdateState):
                    self._registry.update_state(m.primitive_id, m.key, m.value)
                elif isinstance(m, UpdateActivation):
                    self._registry.set_activation(m.primitive_id, m.level)
                elif isinstance(m, UpdateLifecycle):
                    self._registry.set_lifecycle(m.primitive_id, m.state)
                elif isinstance(m, AddEvent):
                    errors.append(
                        "invariant violation: AddEvent in deferred batch"
                    )
            except Exception as e:
                errors.append(f"deferred mutation: {e}")
        return errors


def _layer_stable(layer: Layer, snapshot: Snapshot) -> bool:
    """Returns True if all Layer N-1 primitives are Active and have been
    invoked at least once. Vacuously true when no Layer N-1 primitives exist."""
    if layer.value == 0:
        return True

    target_layer = layer.value - 1
    for ps in snapshot.primitives.values():
        if ps.layer.value == target_layer:
            if ps.lifecycle != LIFECYCLE_ACTIVE:
                return False
            if ps.last_tick == 0:
                return False  # never invoked
    return True


def _filter_events(
    events: list[Event], patterns: list[SubscriptionPattern]
) -> list[Event]:
    """Return events matching any of the subscription patterns."""
    result = []
    for ev in events:
        for pat in patterns:
            if pat.matches(ev.type):
                result.append(ev)
                break
    return result
