namespace EventGraph.Tests;

public class EmittingPrimitive : IPrimitive
{
    public PrimitiveId Id { get; }
    public Layer Layer { get; } = new(0);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("*") };
    public Cadence Cadence { get; } = new(1);
    public int Emissions;
    private readonly int _maxEmissions;

    public EmittingPrimitive(string name, int maxEmissions = 1)
    {
        Id = new PrimitiveId(name);
        _maxEmissions = maxEmissions;
    }

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        if (events.Count == 0 || Emissions >= _maxEmissions) return new();
        Emissions++;
        return new()
        {
            new AddEventMutation(new EventType("test.emitted"), new ActorId("emitter"),
                new Dictionary<string, object?> { ["wave"] = Emissions },
                new List<EventId> { events[0].Id },
                new ConversationId("conv_tick"))
        };
    }
}

public class TickEngineTests
{
    private static (PrimitiveRegistry, InMemoryStore, TickEngine, Event) Setup(
        IPrimitive[]? prims = null, TickConfig? config = null)
    {
        var reg = new PrimitiveRegistry();
        var store = new InMemoryStore();
        var boot = EventFactory.CreateBootstrap(new ActorId("system"), new NoopSigner());
        store.Append(boot);
        foreach (var p in prims ?? [])
        {
            reg.Register(p);
            reg.Activate(p.Id);
        }
        var engine = new TickEngine(reg, store, config);
        return (reg, store, engine, boot);
    }

    [Fact]
    public void BasicTick()
    {
        var counter = new StubPrimitive("counter");
        var (_, _, engine, boot) = Setup(new IPrimitive[] { counter });
        var result = engine.Tick(new() { boot });
        Assert.Equal(1, result.Tick);
        Assert.True(result.Mutations >= 1);
        Assert.Equal(1, counter.ReceivedCount);
    }

    [Fact]
    public void Quiescence()
    {
        var counter = new StubPrimitive("counter");
        var (_, _, engine, boot) = Setup(new IPrimitive[] { counter });
        var result = engine.Tick(new() { boot });
        Assert.True(result.Quiesced);
    }

    [Fact]
    public void RippleWaves()
    {
        var emitter = new EmittingPrimitive("emitter", 3);
        var counter = new StubPrimitive("counter");
        var (_, _, engine, boot) = Setup(new IPrimitive[] { emitter, counter });
        var result = engine.Tick(new() { boot });
        Assert.True(result.Waves > 1);
        Assert.True(counter.ReceivedCount > 1);
    }

    [Fact]
    public void MaxWavesLimit()
    {
        var infinite = new InfiniteEmitter();
        var config = new TickConfig(MaxWavesPerTick: 3);
        var (_, _, engine, boot) = Setup(new IPrimitive[] { infinite }, config);
        var result = engine.Tick(new() { boot });
        Assert.Equal(3, result.Waves);
        Assert.False(result.Quiesced);
    }

    [Fact]
    public void InactivePrimitivesSkipped()
    {
        var counter = new StubPrimitive("dormant");
        var reg = new PrimitiveRegistry();
        var store = new InMemoryStore();
        var boot = EventFactory.CreateBootstrap(new ActorId("system"), new NoopSigner());
        store.Append(boot);
        reg.Register(counter); // don't activate
        var engine = new TickEngine(reg, store);
        engine.Tick(new() { boot });
        Assert.Equal(0, counter.ReceivedCount);
    }

    [Fact]
    public void TickCounterIncrements()
    {
        var (_, _, engine, boot) = Setup();
        Assert.Equal(1, engine.Tick(new() { boot }).Tick);
        Assert.Equal(2, engine.Tick().Tick);
        Assert.Equal(3, engine.Tick().Tick);
    }

    [Fact]
    public void LayerOrdering()
    {
        var order = new List<string>();
        var high = new OrderTracker("high", 5, order);
        var low = new OrderTracker("low", 0, order);
        var mid = new OrderTracker("mid", 2, order);
        var (_, _, engine, boot) = Setup(new IPrimitive[] { high, low, mid });
        engine.Tick(new() { boot });
        Assert.Equal(new[] { "low", "mid", "high" }, order.ToArray());
    }

    // --- Layer constraint tests ---

    [Fact]
    public void LayerConstraintBlocksUninvokedLowerLayer()
    {
        var order = new List<string>();
        var l0 = new OrderTracker("layer0", 0, order);
        var l1 = new OrderTracker("layer1", 1, order);
        var (_, _, engine, boot) = Setup(new IPrimitive[] { l0, l1 });

        // Tick 1: Layer 0 runs, Layer 1 blocked
        engine.Tick(new() { boot });
        Assert.Equal(new[] { "layer0" }, order.ToArray());

        // Tick 2: Layer 0 stable, Layer 1 now eligible
        order.Clear();
        engine.Tick(new() { boot });
        Assert.Contains("layer0", order);
        Assert.Contains("layer1", order);
        Assert.True(order.IndexOf("layer0") < order.IndexOf("layer1"));
    }

    [Fact]
    public void LayerConstraintVacuouslyTrueForSparseLayers()
    {
        var invoked = 0;
        var l1Only = new DelegatePrimitive("l1_only", 1, (_, _, _) =>
        {
            invoked++;
            return new();
        });

        var (_, _, engine, boot) = Setup(new IPrimitive[] { l1Only });
        engine.Tick(new() { boot });
        Assert.Equal(1, invoked);
    }

    [Fact]
    public void LayerConstraintBlockedByDormantLowerLayer()
    {
        var l1Invoked = 0;
        var l0 = new DelegatePrimitive("dormant_l0", 0, (_, _, _) => new());
        var l1 = new DelegatePrimitive("active_l1", 1, (_, _, _) =>
        {
            l1Invoked++;
            return new();
        });

        var reg = new PrimitiveRegistry();
        var store = new InMemoryStore();
        var boot = EventFactory.CreateBootstrap(new ActorId("system"), new NoopSigner());
        store.Append(boot);

        reg.Register(l0); // Don't activate — stays Dormant
        reg.Register(l1);
        reg.Activate(l1.Id);

        var engine = new TickEngine(reg, store);
        engine.Tick(new() { boot });

        Assert.Equal(0, l1Invoked);
    }

    // --- Deferred mutations tests ---

    [Fact]
    public void DeferredMutationsNotVisibleBetweenWaves()
    {
        var stateValuesSeen = new List<object?>();
        var callCount = 0;
        var pid = new PrimitiveId("stateful");

        var stateful = new DelegatePrimitive("stateful", 0, (tick, events, snapshot) =>
        {
            callCount++;
            var ps = snapshot.Primitives.GetValueOrDefault("stateful");
            stateValuesSeen.Add(ps?.State.GetValueOrDefault("count"));

            if (events.Count == 0) return new();

            var result = new List<Mutation>
            {
                new UpdateStateMutation(pid, "count", callCount),
            };

            if (callCount == 1)
            {
                result.Add(new AddEventMutation(
                    new EventType("test.ripple"), new ActorId("stateful"),
                    new(), new List<EventId> { events[0].Id },
                    new ConversationId("conv_t")));
            }

            return result;
        });

        var (_, _, engine, boot) = Setup(new IPrimitive[] { stateful });
        var r = engine.Tick(new() { boot });

        Assert.True(r.Waves >= 2);
        // On wave 0, state should be null (not yet set)
        Assert.Null(stateValuesSeen[0]);
        // On wave 1, UpdateState from wave 0 was deferred, so still null
        if (stateValuesSeen.Count > 1)
            Assert.Null(stateValuesSeen[1]);
    }

    [Fact]
    public void MixedMutationsAddEventAndUpdateState()
    {
        var emitted = false;
        var pid = new PrimitiveId("mixed");

        var mixed = new DelegatePrimitive("mixed", 0, (tick, events, snapshot) =>
        {
            if (events.Count == 0) return new();

            var result = new List<Mutation>
            {
                new UpdateStateMutation(pid, "processed", true),
                new UpdateActivationMutation(pid, new Activation(0.9)),
            };

            if (!emitted)
            {
                emitted = true;
                result.Add(new AddEventMutation(
                    new EventType("test.mixed"), new ActorId("mixed"),
                    new(), new List<EventId> { events[0].Id },
                    new ConversationId("conv_t")));
            }

            return result;
        });

        var (_, _, engine, boot) = Setup(new IPrimitive[] { mixed });
        var r = engine.Tick(new() { boot });

        // AddEvent (eager) + UpdateState + UpdateActivation (deferred)
        Assert.True(r.Mutations >= 3);
    }

    // --- UpdateLifecycle mutation ---

    [Fact]
    public void UpdateLifecycleMutation()
    {
        var invoked = 0;
        var pid = new PrimitiveId("updater");

        var updater = new DelegatePrimitive("updater", 0, (_, _, _) =>
        {
            invoked++;
            return new List<Mutation>
            {
                new UpdateLifecycleMutation(pid, Lifecycle.Suspending)
            };
        });

        var (_, _, engine, boot) = Setup(new IPrimitive[] { updater });

        var r1 = engine.Tick(new() { boot });
        Assert.True(r1.Mutations >= 1);

        // Tick 2: primitive is now Suspending, should NOT be eligible
        var r2 = engine.Tick(new() { boot });
        Assert.Equal(0, r2.Mutations);
        Assert.Equal(1, invoked);
    }

    // --- Subscription filtering ---

    [Fact]
    public void SubscriptionFilteringNoMatch()
    {
        var receivedCounts = new List<int>();

        var watcher = new DelegatePrimitive("watcher", 0, (_, events, _) =>
        {
            receivedCounts.Add(events.Count);
            return new();
        }, new List<SubscriptionPattern> { new("trust.*") });

        var (_, _, engine, boot) = Setup(new IPrimitive[] { watcher });
        engine.Tick(new() { boot });
        Assert.Equal(0, receivedCounts[0]);
    }

    [Fact]
    public void NoSubscriptionsGetsNoEvents()
    {
        var receivedCounts = new List<int>();

        var nosubs = new DelegatePrimitive("nosubs", 0, (_, events, _) =>
        {
            receivedCounts.Add(events.Count);
            return new();
        }, new List<SubscriptionPattern>());

        var (_, _, engine, boot) = Setup(new IPrimitive[] { nosubs });
        engine.Tick(new() { boot });
        Assert.Equal(0, receivedCounts[0]);
    }

    // --- Wave limit ---

    [Fact]
    public void WaveLimitWithCustomConfig()
    {
        var always = new DelegatePrimitive("always", 0, (_, events, _) =>
        {
            if (events.Count == 0) return new();
            return new List<Mutation>
            {
                new AddEventMutation(new EventType("test.always"), new ActorId("a"),
                    new(), new List<EventId> { events[0].Id },
                    new ConversationId("c"))
            };
        });

        var config = new TickConfig(MaxWavesPerTick: 5);
        var (_, _, engine, boot) = Setup(new IPrimitive[] { always }, config);
        var r = engine.Tick(new() { boot });
        Assert.Equal(5, r.Waves);
        Assert.False(r.Quiesced);
    }

    // --- Helper classes ---

    private class InfiniteEmitter : IPrimitive
    {
        public PrimitiveId Id { get; } = new("infinite");
        public Layer Layer { get; } = new(0);
        public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("*") };
        public Cadence Cadence { get; } = new(1);

        public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
        {
            if (events.Count == 0) return new();
            return new()
            {
                new AddEventMutation(new EventType("test.loop"), new ActorId("inf"),
                    new(), new List<EventId> { events[0].Id }, new ConversationId("conv_inf"))
            };
        }
    }

    private class OrderTracker : IPrimitive
    {
        public PrimitiveId Id { get; }
        public Layer Layer { get; }
        public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("*") };
        public Cadence Cadence { get; } = new(1);
        private readonly List<string> _order;

        public OrderTracker(string name, int layer, List<string> order)
        {
            Id = new PrimitiveId(name);
            Layer = new Layer(layer);
            _order = order;
        }

        public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
        {
            _order.Add(Id.Value);
            return new();
        }
    }

    private class DelegatePrimitive : IPrimitive
    {
        public PrimitiveId Id { get; }
        public Layer Layer { get; }
        public List<SubscriptionPattern> Subscriptions { get; }
        public Cadence Cadence { get; } = new(1);
        private readonly Func<int, List<Event>, Snapshot, List<Mutation>> _process;

        public DelegatePrimitive(string name, int layer,
            Func<int, List<Event>, Snapshot, List<Mutation>> process,
            List<SubscriptionPattern>? subscriptions = null)
        {
            Id = new PrimitiveId(name);
            Layer = new Layer(layer);
            Subscriptions = subscriptions ?? new() { new SubscriptionPattern("*") };
            _process = process;
        }

        public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
            => _process(tick, events, snapshot);
    }
}
