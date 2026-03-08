"""Tests for the tick engine — ripple-wave processing."""

from eventgraph.event import NoopSigner, create_bootstrap
from eventgraph.primitive import (
    LIFECYCLE_ACTIVE,
    LIFECYCLE_DORMANT,
    LIFECYCLE_SUSPENDING,
    AddEvent,
    Mutation,
    Registry,
    Snapshot,
    UpdateActivation,
    UpdateLifecycle,
    UpdateState,
)
from eventgraph.store import InMemoryStore
from eventgraph.tick import TickConfig, TickEngine, TickResult
from eventgraph.types import (
    Activation,
    ActorID,
    Cadence,
    ConversationID,
    EventType,
    Layer,
    PrimitiveID,
    SubscriptionPattern,
)


class CountingPrimitive:
    """Counts events it receives."""

    def __init__(self, name: str, layer: int = 0) -> None:
        self._id = PrimitiveID(name)
        self._layer = Layer(layer)
        self.received_count = 0

    def id(self) -> PrimitiveID:
        return self._id

    def layer(self) -> Layer:
        return self._layer

    def process(self, tick, events, snapshot) -> list[Mutation]:
        self.received_count += len(events)
        return [
            UpdateState(primitive_id=self._id, key="count", value=self.received_count)
        ]

    def subscriptions(self) -> list[SubscriptionPattern]:
        return [SubscriptionPattern("*")]

    def cadence(self) -> Cadence:
        return Cadence(1)


class EmittingPrimitive:
    """Emits a new event on each process call (causes ripple waves)."""

    def __init__(self, name: str, max_emissions: int = 1) -> None:
        self._id = PrimitiveID(name)
        self._layer = Layer(0)
        self.emissions = 0
        self.max_emissions = max_emissions

    def id(self) -> PrimitiveID:
        return self._id

    def layer(self) -> Layer:
        return self._layer

    def process(self, tick, events, snapshot) -> list[Mutation]:
        if not events or self.emissions >= self.max_emissions:
            return []
        self.emissions += 1
        return [
            AddEvent(
                type=EventType("test.emitted"),
                source=ActorID("emitter"),
                content={"wave": self.emissions},
                causes=[events[0].id],
                conversation_id=ConversationID("conv_tick"),
            )
        ]

    def subscriptions(self) -> list[SubscriptionPattern]:
        return [SubscriptionPattern("*")]

    def cadence(self) -> Cadence:
        return Cadence(1)


def _setup(prims=None, config=None):
    """Create a registry, store, and engine with bootstrap event."""
    reg = Registry()
    store = InMemoryStore()
    boot = create_bootstrap(source=ActorID("system"), signer=NoopSigner())
    store.append(boot)

    for p in (prims or []):
        reg.register(p)
        reg.activate(p.id())

    engine = TickEngine(reg, store, config)
    return reg, store, engine, boot


class TestTickEngine:
    def test_basic_tick(self):
        counter = CountingPrimitive("counter")
        reg, store, engine, boot = _setup([counter])

        result = engine.tick([boot])
        assert result.tick == 1
        assert result.mutations >= 1
        assert counter.received_count == 1

    def test_quiescence(self):
        counter = CountingPrimitive("counter")
        _, _, engine, boot = _setup([counter])

        result = engine.tick([boot])
        assert result.quiesced is True  # no new events emitted

    def test_ripple_waves(self):
        emitter = EmittingPrimitive("emitter", max_emissions=3)
        counter = CountingPrimitive("counter")
        reg, store, engine, boot = _setup([emitter, counter])

        result = engine.tick([boot])
        assert result.waves > 1  # ripple happened
        assert counter.received_count > 1  # received original + emitted

    def test_max_waves_limit(self):
        # Emitter that always emits (infinite ripple)
        class InfiniteEmitter:
            def __init__(self):
                self._id = PrimitiveID("infinite")

            def id(self): return self._id
            def layer(self): return Layer(0)
            def subscriptions(self): return [SubscriptionPattern("*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                if events:
                    return [AddEvent(
                        type=EventType("test.loop"),
                        source=ActorID("inf"),
                        content={},
                        causes=[events[0].id],
                        conversation_id=ConversationID("conv_inf"),
                    )]
                return []

        config = TickConfig(max_waves_per_tick=3)
        _, _, engine, boot = _setup([InfiniteEmitter()], config)

        result = engine.tick([boot])
        assert result.waves == 3
        assert result.quiesced is False

    def test_inactive_primitives_skipped(self):
        counter = CountingPrimitive("dormant_counter")
        reg = Registry()
        store = InMemoryStore()
        boot = create_bootstrap(source=ActorID("system"), signer=NoopSigner())
        store.append(boot)
        reg.register(counter)
        # Don't activate — stays dormant

        engine = TickEngine(reg, store)
        result = engine.tick([boot])
        assert counter.received_count == 0

    def test_cadence_respected(self):
        class SlowPrimitive(CountingPrimitive):
            def cadence(self): return Cadence(3)

        slow = SlowPrimitive("slow_prim")
        reg, store, engine, boot = _setup([slow])

        r1 = engine.tick([boot])  # tick 1: 1-0=1 < 3 → skip
        assert slow.received_count == 0

        engine.tick([])  # tick 2
        assert slow.received_count == 0

        engine.tick([boot])  # tick 3: 3-0=3 >= 3, should process
        assert slow.received_count == 1

    def test_tick_counter_increments(self):
        _, _, engine, boot = _setup()
        r1 = engine.tick([boot])
        r2 = engine.tick([])
        r3 = engine.tick([])
        assert r1.tick == 1
        assert r2.tick == 2
        assert r3.tick == 3

    def test_published_events(self):
        emitter = EmittingPrimitive("pub_emitter", max_emissions=1)
        published: list = []

        reg = Registry()
        store = InMemoryStore()
        boot = create_bootstrap(source=ActorID("system"), signer=NoopSigner())
        store.append(boot)
        reg.register(emitter)
        reg.activate(emitter.id())

        engine = TickEngine(reg, store, publisher=lambda ev: published.append(ev))
        engine.tick([boot])
        assert len(published) >= 1

    def test_layer_ordering(self):
        """Lower layers process before higher layers."""
        order = []

        class OrderTracker:
            def __init__(self, name, layer_val):
                self._id = PrimitiveID(name)
                self._layer = Layer(layer_val)

            def id(self): return self._id
            def layer(self): return self._layer
            def subscriptions(self): return [SubscriptionPattern("*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                order.append(self._id.value)
                return []

        p_high = OrderTracker("high", 5)
        p_low = OrderTracker("low", 0)
        p_mid = OrderTracker("mid", 2)

        reg, store, engine, boot = _setup([p_high, p_low, p_mid])
        engine.tick([boot])

        assert order == ["low", "mid", "high"]

    # --- New tests for layer constraint ---

    def test_layer_constraint_blocks_uninvoked_lower_layer(self):
        """Layer 1 should not run until Layer 0 has been invoked at least once."""
        order = []

        class OrderTracker:
            def __init__(self, name, layer_val):
                self._id = PrimitiveID(name)
                self._layer = Layer(layer_val)

            def id(self): return self._id
            def layer(self): return self._layer
            def subscriptions(self): return [SubscriptionPattern("*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                order.append(self._id.value)
                return []

        p0 = OrderTracker("layer0", 0)
        p1 = OrderTracker("layer1", 1)

        _, _, engine, boot = _setup([p0, p1])

        # Tick 1: Layer 0 runs, Layer 1 blocked (Layer 0 never invoked before)
        engine.tick([boot])
        assert order == ["layer0"]

        # Tick 2: Layer 0 stable, Layer 1 now eligible
        order.clear()
        engine.tick([boot])
        assert "layer0" in order
        assert "layer1" in order
        assert order.index("layer0") < order.index("layer1")

    def test_layer_constraint_vacuously_true_sparse_layers(self):
        """Layer 1 with no Layer 0 primitives should run (vacuously stable)."""
        invoked = []

        class L1Prim:
            def __init__(self):
                self._id = PrimitiveID("l1_only")

            def id(self): return self._id
            def layer(self): return Layer(1)
            def subscriptions(self): return [SubscriptionPattern("*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                invoked.append(True)
                return []

        _, _, engine, boot = _setup([L1Prim()])
        engine.tick([boot])
        assert len(invoked) == 1

    def test_layer_constraint_blocked_by_dormant_lower_layer(self):
        """Layer 1 should be blocked when a Layer 0 primitive is Dormant."""
        l1_invoked = []

        class L0Prim:
            def __init__(self):
                self._id = PrimitiveID("dormant_l0")

            def id(self): return self._id
            def layer(self): return Layer(0)
            def subscriptions(self): return [SubscriptionPattern("*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                return []

        class L1Prim:
            def __init__(self):
                self._id = PrimitiveID("active_l1")

            def id(self): return self._id
            def layer(self): return Layer(1)
            def subscriptions(self): return [SubscriptionPattern("*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                l1_invoked.append(True)
                return []

        reg = Registry()
        store = InMemoryStore()
        boot = create_bootstrap(source=ActorID("system"), signer=NoopSigner())
        store.append(boot)

        l0 = L0Prim()
        l1 = L1Prim()

        reg.register(l0)
        # Don't activate l0 — stays Dormant
        reg.register(l1)
        reg.activate(l1.id())

        engine = TickEngine(reg, store)
        engine.tick([boot])

        # Layer 1 should NOT run because Layer 0 has a Dormant primitive
        assert len(l1_invoked) == 0

    # --- Deferred mutations tests ---

    def test_deferred_mutations_not_visible_between_waves(self):
        """UpdateState should be deferred to end of tick, not applied between waves."""
        state_values_seen: list = []

        class StatefulPrim:
            def __init__(self):
                self._id = PrimitiveID("stateful")
                self._call_count = 0

            def id(self): return self._id
            def layer(self): return Layer(0)
            def subscriptions(self): return [SubscriptionPattern("*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                self._call_count += 1
                # Record what state the snapshot shows
                ps = snapshot.primitives.get("stateful")
                state_values_seen.append(
                    ps.state.get("count") if ps else None
                )

                if not events:
                    return []

                result = [
                    UpdateState(
                        primitive_id=self._id, key="count", value=self._call_count,
                    ),
                ]

                # Emit event on first call to trigger wave 1
                if self._call_count == 1:
                    result.append(
                        AddEvent(
                            type=EventType("test.ripple"),
                            source=ActorID("stateful"),
                            content={},
                            causes=[events[0].id],
                            conversation_id=ConversationID("conv_test"),
                        )
                    )

                return result

        _, _, engine, boot = _setup([StatefulPrim()])
        result = engine.tick([boot])

        assert result.waves >= 2
        # On wave 0, state should be empty (None)
        assert state_values_seen[0] is None
        # On wave 1, UpdateState from wave 0 was deferred, so still not visible
        if len(state_values_seen) > 1:
            assert state_values_seen[1] is None

    def test_mixed_mutations_add_event_and_update_state(self):
        """Primitive emitting both AddEvent and UpdateState in same wave."""

        class MixedPrim:
            def __init__(self):
                self._id = PrimitiveID("mixed")
                self._emitted = False

            def id(self): return self._id
            def layer(self): return Layer(0)
            def subscriptions(self): return [SubscriptionPattern("*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                if not events:
                    return []

                result = [
                    UpdateState(
                        primitive_id=self._id, key="processed", value=True,
                    ),
                    UpdateActivation(
                        primitive_id=self._id, level=Activation(0.9),
                    ),
                ]

                if not self._emitted:
                    self._emitted = True
                    result.append(
                        AddEvent(
                            type=EventType("test.mixed"),
                            source=ActorID("mixed"),
                            content={},
                            causes=[events[0].id],
                            conversation_id=ConversationID("conv_test"),
                        )
                    )

                return result

        _, _, engine, boot = _setup([MixedPrim()])
        result = engine.tick([boot])

        # AddEvent (eager) + UpdateState + UpdateActivation (deferred)
        assert result.mutations >= 3

    # --- UpdateLifecycle mutation ---

    def test_update_lifecycle_mutation(self):
        """Primitive can request lifecycle change via UpdateLifecycle mutation."""

        class LifecycleUpdater:
            def __init__(self):
                self._id = PrimitiveID("updater")
                self._invoked = 0

            def id(self): return self._id
            def layer(self): return Layer(0)
            def subscriptions(self): return [SubscriptionPattern("*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                self._invoked += 1
                return [
                    UpdateLifecycle(
                        primitive_id=self._id,
                        state=LIFECYCLE_SUSPENDING,
                    )
                ]

        prim = LifecycleUpdater()
        _, _, engine, boot = _setup([prim])

        r1 = engine.tick([boot])
        assert r1.mutations >= 1

        # Tick 2: primitive is now Suspending, should NOT be eligible
        r2 = engine.tick([boot])
        assert r2.mutations == 0
        assert prim._invoked == 1

    # --- Subscription filtering ---

    def test_subscription_filtering_no_match(self):
        """Primitive with specific subscriptions gets no unmatched events."""
        received_counts = []

        class TrustWatcher:
            def __init__(self):
                self._id = PrimitiveID("watcher")

            def id(self): return self._id
            def layer(self): return Layer(0)
            def subscriptions(self): return [SubscriptionPattern("trust.*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                received_counts.append(len(events))
                return []

        _, _, engine, boot = _setup([TrustWatcher()])

        # Bootstrap has type "system.bootstrap" — should NOT match "trust.*"
        engine.tick([boot])
        assert received_counts[0] == 0

    def test_no_subscriptions_gets_no_events(self):
        """Primitive with empty subscriptions receives no matched events."""
        received_counts = []

        class NoSubsPrim:
            def __init__(self):
                self._id = PrimitiveID("nosubs")

            def id(self): return self._id
            def layer(self): return Layer(0)
            def subscriptions(self): return []
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                received_counts.append(len(events))
                return []

        _, _, engine, boot = _setup([NoSubsPrim()])
        engine.tick([boot])
        assert received_counts[0] == 0

    # --- Wave limit ---

    def test_wave_limit_with_custom_config(self):
        """Wave limit caps at MaxWavesPerTick."""

        class AlwaysEmitter:
            def __init__(self):
                self._id = PrimitiveID("always")

            def id(self): return self._id
            def layer(self): return Layer(0)
            def subscriptions(self): return [SubscriptionPattern("*")]
            def cadence(self): return Cadence(1)

            def process(self, tick, events, snapshot):
                if events:
                    return [AddEvent(
                        type=EventType("test.always"),
                        source=ActorID("always"),
                        content={},
                        causes=[events[0].id],
                        conversation_id=ConversationID("conv_always"),
                    )]
                return []

        config = TickConfig(max_waves_per_tick=5)
        _, _, engine, boot = _setup([AlwaysEmitter()], config)

        result = engine.tick([boot])
        assert result.waves == 5
        assert result.quiesced is False
