using EventGraph.Agent;

namespace EventGraph.Tests;

// ── Helpers ──────────────────────────────────────────────────────────────

public static class AgentTestHelpers
{
    private static readonly ActorId TestActor = new("test-agent");
    private static readonly ConversationId TestConv = new("conv_test");
    private static readonly ISigner Signer = new NoopSigner();

    /// <summary>Create a test event with the given event type.</summary>
    public static Event MakeEvent(string eventType, Event? prev = null)
    {
        var prevHash = prev?.Hash ?? Hash.Zero();
        return EventFactory.CreateEvent(
            new EventType(eventType), TestActor,
            new Dictionary<string, object?>(), new List<EventId> { EventFactory.NewEventId() },
            TestConv, prevHash, Signer);
    }

    public static Snapshot EmptySnapshot(int tick = 1) =>
        new(tick, new Dictionary<string, PrimitiveState>(), new List<Event>(), new List<Event>());
}

// ── OperationalState FSM Tests ──────────────────────────────────────────

public class OperationalStateTests
{
    [Theory]
    [InlineData(OperationalState.Idle, OperationalState.Processing)]
    [InlineData(OperationalState.Idle, OperationalState.Suspended)]
    [InlineData(OperationalState.Idle, OperationalState.Retiring)]
    [InlineData(OperationalState.Processing, OperationalState.Idle)]
    [InlineData(OperationalState.Processing, OperationalState.Waiting)]
    [InlineData(OperationalState.Processing, OperationalState.Escalating)]
    [InlineData(OperationalState.Processing, OperationalState.Refusing)]
    [InlineData(OperationalState.Processing, OperationalState.Retiring)]
    [InlineData(OperationalState.Waiting, OperationalState.Processing)]
    [InlineData(OperationalState.Waiting, OperationalState.Idle)]
    [InlineData(OperationalState.Waiting, OperationalState.Retiring)]
    [InlineData(OperationalState.Escalating, OperationalState.Waiting)]
    [InlineData(OperationalState.Escalating, OperationalState.Idle)]
    [InlineData(OperationalState.Refusing, OperationalState.Idle)]
    [InlineData(OperationalState.Suspended, OperationalState.Idle)]
    [InlineData(OperationalState.Suspended, OperationalState.Retiring)]
    [InlineData(OperationalState.Retiring, OperationalState.Retired)]
    public void ValidTransitions(OperationalState from, OperationalState to)
    {
        Assert.True(OperationalStateMachine.IsValidTransition(from, to));
        var result = OperationalStateMachine.TransitionTo(from, to);
        Assert.Equal(to, result);
    }

    [Theory]
    [InlineData(OperationalState.Idle, OperationalState.Waiting)]
    [InlineData(OperationalState.Idle, OperationalState.Escalating)]
    [InlineData(OperationalState.Idle, OperationalState.Refusing)]
    [InlineData(OperationalState.Idle, OperationalState.Retired)]
    [InlineData(OperationalState.Processing, OperationalState.Suspended)]
    [InlineData(OperationalState.Waiting, OperationalState.Refusing)]
    [InlineData(OperationalState.Escalating, OperationalState.Retired)]
    [InlineData(OperationalState.Refusing, OperationalState.Processing)]
    [InlineData(OperationalState.Suspended, OperationalState.Processing)]
    [InlineData(OperationalState.Retiring, OperationalState.Idle)]
    [InlineData(OperationalState.Retired, OperationalState.Idle)]
    [InlineData(OperationalState.Retired, OperationalState.Processing)]
    public void InvalidTransitions(OperationalState from, OperationalState to)
    {
        Assert.False(OperationalStateMachine.IsValidTransition(from, to));
        Assert.Throws<InvalidTransitionException>(() => OperationalStateMachine.TransitionTo(from, to));
    }

    [Fact]
    public void RetiredIsTerminal()
    {
        Assert.True(OperationalStateMachine.IsTerminal(OperationalState.Retired));
        Assert.False(OperationalStateMachine.IsTerminal(OperationalState.Idle));
        Assert.False(OperationalStateMachine.IsTerminal(OperationalState.Processing));
        Assert.False(OperationalStateMachine.IsTerminal(OperationalState.Suspended));
    }

    [Fact]
    public void CanActOnlyWhenProcessing()
    {
        Assert.True(OperationalStateMachine.CanAct(OperationalState.Processing));
        Assert.False(OperationalStateMachine.CanAct(OperationalState.Idle));
        Assert.False(OperationalStateMachine.CanAct(OperationalState.Waiting));
        Assert.False(OperationalStateMachine.CanAct(OperationalState.Retired));
    }

    [Fact]
    public void SelfTransitionIsInvalid()
    {
        foreach (OperationalState state in Enum.GetValues<OperationalState>())
            Assert.False(OperationalStateMachine.IsValidTransition(state, state));
    }
}

// ── Agent Event Types Tests ─────────────────────────────────────────────

public class AgentEventTypeTests
{
    [Fact]
    public void AllReturns45EventTypes()
    {
        var all = AgentEventTypes.All();
        Assert.Equal(45, all.Count);
    }

    [Fact]
    public void AllEventTypesStartWithAgent()
    {
        foreach (var et in AgentEventTypes.All())
            Assert.StartsWith("agent.", et.Value);
    }

    [Fact]
    public void AllEventTypesAreUnique()
    {
        var all = AgentEventTypes.All();
        var unique = new HashSet<string>(all.Select(e => e.Value));
        Assert.Equal(all.Count, unique.Count);
    }

    [Fact]
    public void SpecificEventTypesExist()
    {
        Assert.Equal("agent.identity.created", AgentEventTypes.IdentityCreated.Value);
        Assert.Equal("agent.soul.imprinted", AgentEventTypes.SoulImprinted.Value);
        Assert.Equal("agent.attenuated", AgentEventTypes.Attenuated.Value);
        Assert.Equal("agent.composition.formed", AgentEventTypes.CompositionFormed.Value);
    }
}

// ── Agent Primitives Tests ──────────────────────────────────────────────

public class AgentPrimitivesTests
{
    [Fact]
    public void AllPrimitivesReturns28()
    {
        var all = AgentPrimitiveFactory.AllPrimitives();
        Assert.Equal(28, all.Count);
    }

    [Fact]
    public void AllPrimitivesHaveUniqueIds()
    {
        var all = AgentPrimitiveFactory.AllPrimitives();
        var ids = new HashSet<string>(all.Select(p => p.Id.Value));
        Assert.Equal(28, ids.Count);
    }

    [Fact]
    public void AllPrimitivesAreLayer1()
    {
        foreach (var p in AgentPrimitiveFactory.AllPrimitives())
            Assert.Equal(1, p.Layer.Value);
    }

    [Fact]
    public void AllPrimitivesHaveCadence1()
    {
        foreach (var p in AgentPrimitiveFactory.AllPrimitives())
            Assert.Equal(1, p.Cadence.Value);
    }

    [Fact]
    public void AllPrimitivesHaveSubscriptions()
    {
        foreach (var p in AgentPrimitiveFactory.AllPrimitives())
            Assert.NotEmpty(p.Subscriptions);
    }

    [Fact]
    public void AllPrimitivesStartWithAgentPrefix()
    {
        foreach (var p in AgentPrimitiveFactory.AllPrimitives())
            Assert.True(AgentPrimitiveFactory.IsAgentPrimitive(p.Id));
    }

    [Fact]
    public void RegisterAllRegisters28Primitives()
    {
        var registry = new PrimitiveRegistry();
        AgentPrimitiveFactory.RegisterAll(registry);
        Assert.Equal(28, registry.Count);
    }

    [Fact]
    public void RegisterAllActivatesPrimitives()
    {
        var registry = new PrimitiveRegistry();
        AgentPrimitiveFactory.RegisterAll(registry);
        foreach (var p in AgentPrimitiveFactory.AllPrimitives())
            Assert.Equal(Lifecycle.Active, registry.GetLifecycle(p.Id));
    }

    [Fact]
    public void IsAgentPrimitiveReturnsFalseForNonAgent()
    {
        Assert.False(AgentPrimitiveFactory.IsAgentPrimitive(new PrimitiveId("system.Event")));
        Assert.False(AgentPrimitiveFactory.IsAgentPrimitive(new PrimitiveId("Hash")));
    }

    // ── Individual primitive processing tests ────────────────────────────

    [Fact]
    public void IdentityPrimitive_ProcessesEvents()
    {
        var p = new IdentityPrimitive();
        var events = new List<Event>
        {
            AgentTestHelpers.MakeEvent("agent.identity.created"),
            AgentTestHelpers.MakeEvent("agent.identity.rotated"),
            AgentTestHelpers.MakeEvent("actor.registered"),
        };
        var mutations = p.Process(5, events, AgentTestHelpers.EmptySnapshot(5));
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "identitiesCreated" && (int)u.Value! == 2);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "keysRotated" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "lastTick" && (int)u.Value! == 5);
    }

    [Fact]
    public void SoulPrimitive_TracksImprintsAndRefusals()
    {
        var p = new SoulPrimitive();
        var events = new List<Event>
        {
            AgentTestHelpers.MakeEvent("agent.soul.imprinted"),
            AgentTestHelpers.MakeEvent("agent.refused"),
        };
        var mutations = p.Process(3, events, AgentTestHelpers.EmptySnapshot(3));
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "imprinted" && (bool)u.Value! == true);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "soulRefusals" && (int)u.Value! == 1);
    }

    [Fact]
    public void GoalPrimitive_TracksAllGoalEvents()
    {
        var p = new GoalPrimitive();
        var events = new List<Event>
        {
            AgentTestHelpers.MakeEvent("agent.goal.set"),
            AgentTestHelpers.MakeEvent("agent.goal.set"),
            AgentTestHelpers.MakeEvent("agent.goal.completed"),
            AgentTestHelpers.MakeEvent("agent.goal.abandoned"),
        };
        var mutations = p.Process(1, events, AgentTestHelpers.EmptySnapshot());
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "goalsSet" && (int)u.Value! == 2);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "goalsCompleted" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "goalsAbandoned" && (int)u.Value! == 1);
    }

    [Fact]
    public void ObservePrimitive_TracksTotalAndObserved()
    {
        var p = new ObservePrimitive();
        var events = new List<Event>
        {
            AgentTestHelpers.MakeEvent("agent.observed"),
            AgentTestHelpers.MakeEvent("agent.acted"),
        };
        var mutations = p.Process(1, events, AgentTestHelpers.EmptySnapshot());
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "eventsObserved" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "totalEventsReceived" && (int)u.Value! == 2);
    }

    [Fact]
    public void AttenuationPrimitive_TracksAllCategories()
    {
        var p = new AttenuationPrimitive();
        var events = new List<Event>
        {
            AgentTestHelpers.MakeEvent("agent.attenuated"),
            AgentTestHelpers.MakeEvent("agent.attenuation.lifted"),
            AgentTestHelpers.MakeEvent("agent.budget.exhausted"),
        };
        var mutations = p.Process(1, events, AgentTestHelpers.EmptySnapshot());
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "attenuations" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "lifts" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "budgetTriggered" && (int)u.Value! == 1);
    }

    [Fact]
    public void CompositionPrimitive_TracksGroupOperations()
    {
        var p = new CompositionPrimitive();
        var events = new List<Event>
        {
            AgentTestHelpers.MakeEvent("agent.composition.formed"),
            AgentTestHelpers.MakeEvent("agent.composition.joined"),
            AgentTestHelpers.MakeEvent("agent.composition.left"),
            AgentTestHelpers.MakeEvent("agent.composition.dissolved"),
        };
        var mutations = p.Process(1, events, AgentTestHelpers.EmptySnapshot());
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "groupsFormed" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "membersJoined" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "membersLeft" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "groupsDissolved" && (int)u.Value! == 1);
    }

    [Fact]
    public void ExpectPrimitive_TracksAllExpectations()
    {
        var p = new ExpectPrimitive();
        var events = new List<Event>
        {
            AgentTestHelpers.MakeEvent("agent.expectation.set"),
            AgentTestHelpers.MakeEvent("agent.expectation.met"),
            AgentTestHelpers.MakeEvent("agent.expectation.expired"),
        };
        var mutations = p.Process(1, events, AgentTestHelpers.EmptySnapshot());
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "expectationsSet" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "expectationsMet" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "expectationsExpired" && (int)u.Value! == 1);
    }

    [Fact]
    public void ConsentPrimitive_TracksAllConsentEvents()
    {
        var p = new ConsentPrimitive();
        var events = new List<Event>
        {
            AgentTestHelpers.MakeEvent("agent.consent.requested"),
            AgentTestHelpers.MakeEvent("agent.consent.granted"),
            AgentTestHelpers.MakeEvent("agent.consent.denied"),
        };
        var mutations = p.Process(1, events, AgentTestHelpers.EmptySnapshot());
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "consentRequested" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "consentGranted" && (int)u.Value! == 1);
        Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "consentDenied" && (int)u.Value! == 1);
    }

    [Fact]
    public void AllPrimitives_ProcessEmptyEvents()
    {
        var emptyEvents = new List<Event>();
        var snap = AgentTestHelpers.EmptySnapshot();
        foreach (var p in AgentPrimitiveFactory.AllPrimitives())
        {
            var mutations = p.Process(1, emptyEvents, snap);
            Assert.NotNull(mutations);
            Assert.NotEmpty(mutations);
            Assert.Contains(mutations, m => m is UpdateStateMutation u && u.Key == "lastTick" && (int)u.Value! == 1);
        }
    }
}

// ── Agent Compositions Tests ────────────────────────────────────────────

public class AgentCompositionsTests
{
    [Fact]
    public void AllReturns8Compositions()
    {
        var all = AgentCompositions.All();
        Assert.Equal(8, all.Count);
    }

    [Fact]
    public void AllCompositionsHaveUniqueNames()
    {
        var all = AgentCompositions.All();
        var names = new HashSet<string>(all.Select(c => c.Name));
        Assert.Equal(8, names.Count);
    }

    [Fact]
    public void AllCompositionsHaveNonEmptyPrimitives()
    {
        foreach (var c in AgentCompositions.All())
        {
            Assert.NotEmpty(c.Primitives);
            Assert.NotEmpty(c.Events);
        }
    }

    [Fact]
    public void AllCompositionPrimitivesStartWithAgent()
    {
        foreach (var c in AgentCompositions.All())
            foreach (var pId in c.Primitives)
                Assert.StartsWith("agent.", pId);
    }

    [Fact]
    public void Boot_Has5Primitives5Events()
    {
        var boot = AgentCompositions.Boot();
        Assert.Equal("Boot", boot.Name);
        Assert.Equal(5, boot.Primitives.Count);
        Assert.Equal(5, boot.Events.Count);
        Assert.Contains("agent.Identity", boot.Primitives);
        Assert.Contains("agent.Soul", boot.Primitives);
        Assert.Contains("agent.Model", boot.Primitives);
        Assert.Contains("agent.Authority", boot.Primitives);
        Assert.Contains("agent.State", boot.Primitives);
    }

    [Fact]
    public void Imprint_ExtendsBoot()
    {
        var imprint = AgentCompositions.Imprint();
        Assert.Equal("Imprint", imprint.Name);
        Assert.Equal(8, imprint.Primitives.Count);
        Assert.Equal(8, imprint.Events.Count);
        // Contains all Boot primitives plus 3 more
        Assert.Contains("agent.Identity", imprint.Primitives);
        Assert.Contains("agent.Observe", imprint.Primitives);
        Assert.Contains("agent.Learn", imprint.Primitives);
        Assert.Contains("agent.Goal", imprint.Primitives);
    }

    [Fact]
    public void Task_Has5Primitives5Events()
    {
        var task = AgentCompositions.Task();
        Assert.Equal("Task", task.Name);
        Assert.Equal(5, task.Primitives.Count);
        Assert.Equal(5, task.Events.Count);
    }

    [Fact]
    public void Supervise_Has5Primitives4Events()
    {
        var sup = AgentCompositions.Supervise();
        Assert.Equal("Supervise", sup.Name);
        Assert.Equal(5, sup.Primitives.Count);
        Assert.Equal(4, sup.Events.Count);
    }

    [Fact]
    public void Collaborate_Has5Primitives6Events()
    {
        var collab = AgentCompositions.Collaborate();
        Assert.Equal("Collaborate", collab.Name);
        Assert.Equal(5, collab.Primitives.Count);
        Assert.Equal(6, collab.Events.Count);
    }

    [Fact]
    public void Crisis_Has5Primitives5Events()
    {
        var crisis = AgentCompositions.Crisis();
        Assert.Equal("Crisis", crisis.Name);
        Assert.Equal(5, crisis.Primitives.Count);
        Assert.Equal(5, crisis.Events.Count);
    }

    [Fact]
    public void Retire_Has4Primitives4Events()
    {
        var retire = AgentCompositions.Retire();
        Assert.Equal("Retire", retire.Name);
        Assert.Equal(4, retire.Primitives.Count);
        Assert.Equal(4, retire.Events.Count);
    }

    [Fact]
    public void Whistleblow_Has5Primitives5Events()
    {
        var wb = AgentCompositions.Whistleblow();
        Assert.Equal("Whistleblow", wb.Name);
        Assert.Equal(5, wb.Primitives.Count);
        Assert.Equal(5, wb.Events.Count);
    }

    [Fact]
    public void CompositionNamesAreCorrect()
    {
        var expected = new[] { "Boot", "Imprint", "Task", "Supervise", "Collaborate", "Crisis", "Retire", "Whistleblow" };
        var actual = AgentCompositions.All().Select(c => c.Name).ToArray();
        Assert.Equal(expected, actual);
    }
}
