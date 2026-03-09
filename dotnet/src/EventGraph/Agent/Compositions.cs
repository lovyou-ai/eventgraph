namespace EventGraph.Agent;

/// <summary>A named sequence of agent primitive operations.</summary>
public sealed record AgentComposition(string Name, List<string> Primitives, List<EventType> Events);

/// <summary>8 named compositions built from the 28 agent primitives.</summary>
public static class AgentCompositions
{
    /// <summary>Agent comes into existence.
    /// Identity(generate) + Soul(load) + Model(bind) + Authority(receive) + State(set:idle)</summary>
    public static AgentComposition Boot() => new(
        "Boot",
        new() { "agent.Identity", "agent.Soul", "agent.Model", "agent.Authority", "agent.State" },
        new()
        {
            AgentEventTypes.IdentityCreated,
            AgentEventTypes.SoulImprinted,
            AgentEventTypes.ModelBound,
            AgentEventTypes.AuthorityGranted,
            AgentEventTypes.StateChanged,
        }
    );

    /// <summary>The birth wizard. Boot plus initial context.
    /// Boot + Observe(first_message) + Learn(initial_context) + Goal(set)</summary>
    public static AgentComposition Imprint()
    {
        var bootComp = Boot();
        var events = new List<EventType>(bootComp.Events)
        {
            AgentEventTypes.Observed,
            AgentEventTypes.Learned,
            AgentEventTypes.GoalSet,
        };
        return new(
            "Imprint",
            new()
            {
                "agent.Identity", "agent.Soul", "agent.Model", "agent.Authority", "agent.State",
                "agent.Observe", "agent.Learn", "agent.Goal",
            },
            events
        );
    }

    /// <summary>The basic work cycle.
    /// Observe(assignment) + Evaluate(scope) + Decide(accept_or_refuse) + Act(execute) + Learn(outcome)</summary>
    public static AgentComposition Task() => new(
        "Task",
        new() { "agent.Observe", "agent.Evaluate", "agent.Decide", "agent.Act", "agent.Learn" },
        new()
        {
            AgentEventTypes.Observed,
            AgentEventTypes.Evaluated,
            AgentEventTypes.Decided,
            AgentEventTypes.Acted,
            AgentEventTypes.Learned,
        }
    );

    /// <summary>Managing another agent's work.
    /// Delegate(task) + Expect(completion) + Observe(progress) + Evaluate(quality) + Repair(if_needed)</summary>
    public static AgentComposition Supervise() => new(
        "Supervise",
        new() { "agent.Delegate", "agent.Expect", "agent.Observe", "agent.Evaluate", "agent.Repair" },
        new()
        {
            AgentEventTypes.Delegated,
            AgentEventTypes.ExpectationSet,
            AgentEventTypes.Observed,
            AgentEventTypes.Evaluated,
        }
    );

    /// <summary>Agents working together on a shared goal.
    /// Channel(open) + Communicate(proposal) + Consent(terms) + Composition(form) + Act(jointly)</summary>
    public static AgentComposition Collaborate() => new(
        "Collaborate",
        new() { "agent.Channel", "agent.Communicate", "agent.Consent", "agent.Composition", "agent.Act" },
        new()
        {
            AgentEventTypes.ChannelOpened,
            AgentEventTypes.Communicated,
            AgentEventTypes.ConsentRequested,
            AgentEventTypes.ConsentGranted,
            AgentEventTypes.CompositionFormed,
            AgentEventTypes.Acted,
        }
    );

    /// <summary>Something is wrong. Detect, assess, attenuate if needed, escalate.
    /// Observe(anomaly) + Evaluate(severity) + Attenuation(if_needed) + Escalate(to_authority) + Expect(resolution)</summary>
    public static AgentComposition Crisis() => new(
        "Crisis",
        new() { "agent.Observe", "agent.Evaluate", "agent.Attenuation", "agent.Escalate", "agent.Expect" },
        new()
        {
            AgentEventTypes.Observed,
            AgentEventTypes.Evaluated,
            AgentEventTypes.Attenuated,
            AgentEventTypes.Escalated,
            AgentEventTypes.ExpectationSet,
        }
    );

    /// <summary>Graceful shutdown.
    /// Introspect(final) + Communicate(farewell) + Memory(archive) + Lifespan(close)</summary>
    public static AgentComposition Retire() => new(
        "Retire",
        new() { "agent.Introspect", "agent.Communicate", "agent.Memory", "agent.Lifespan" },
        new()
        {
            AgentEventTypes.Introspected,
            AgentEventTypes.Communicated,
            AgentEventTypes.MemoryUpdated,
            AgentEventTypes.LifespanEnded,
        }
    );

    /// <summary>The agent detects harm and refuses to be complicit.
    /// Cannot be silenced by authority. Protected by the Soul primitive.
    /// Observe(harm) + Evaluate(severity) + Refuse(complicity) + Escalate(with_evidence) + Communicate(public)</summary>
    public static AgentComposition Whistleblow() => new(
        "Whistleblow",
        new() { "agent.Observe", "agent.Evaluate", "agent.Refuse", "agent.Escalate", "agent.Communicate" },
        new()
        {
            AgentEventTypes.Observed,
            AgentEventTypes.Evaluated,
            AgentEventTypes.Refused,
            AgentEventTypes.Escalated,
            AgentEventTypes.Communicated,
        }
    );

    /// <summary>Returns all 8 named compositions.</summary>
    public static List<AgentComposition> All() => new()
    {
        Boot(),
        Imprint(),
        Task(),
        Supervise(),
        Collaborate(),
        Crisis(),
        Retire(),
        Whistleblow(),
    };
}
