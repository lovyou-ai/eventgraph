namespace EventGraph.Agent;

// All 28 agent primitives operate at Layer 1 (Agency), Cadence 1.

// ════════════════════════════════════════════════════════════════════════
// STRUCTURAL PRIMITIVES (11) — Define what an agent IS
// ════════════════════════════════════════════════════════════════════════

/// <summary>ActorID + keys + type + chain of custody. The unforgeable "who."</summary>
public sealed class IdentityPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Identity");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.identity.*"),
        new SubscriptionPattern("actor.registered"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int created = 0, rotated = 0;
        foreach (var ev in events)
        {
            var t = ev.Type.Value;
            if (t == "agent.identity.created" || t == "actor.registered") created++;
            else if (t == "agent.identity.rotated") rotated++;
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "identitiesCreated", created),
            new UpdateStateMutation(Id, "keysRotated", rotated),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>The agent's values and ethical constraints. Immutable after imprint.</summary>
public sealed class SoulPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Soul");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.soul.*"),
        new SubscriptionPattern("agent.refused"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int imprints = 0, refusals = 0;
        foreach (var ev in events)
        {
            var t = ev.Type.Value;
            if (t == "agent.soul.imprinted") imprints++;
            else if (t == "agent.refused") refusals++;
        }
        var mutations = new List<Mutation> { new UpdateStateMutation(Id, "lastTick", tick) };
        if (imprints > 0) mutations.Add(new UpdateStateMutation(Id, "imprinted", true));
        if (refusals > 0) mutations.Add(new UpdateStateMutation(Id, "soulRefusals", refusals));
        return mutations;
    }
}

/// <summary>The IIntelligence binding. Which reasoning engine, what capabilities, what cost tier.</summary>
public sealed class ModelPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Model");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.model.*") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int bindings = 0, changes = 0;
        foreach (var ev in events)
        {
            switch (ev.Type.Value)
            {
                case "agent.model.bound": bindings++; break;
                case "agent.model.changed": changes++; break;
            }
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "bindings", bindings),
            new UpdateStateMutation(Id, "modelChanges", changes),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Persistent state across ticks. What the agent has learned and remembers.</summary>
public sealed class MemoryPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Memory");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.memory.*"),
        new SubscriptionPattern("agent.learned"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int updates = 0;
        foreach (var ev in events)
        {
            var t = ev.Type.Value;
            if (t == "agent.memory.updated" || t == "agent.learned") updates++;
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "memoryUpdates", updates),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Current operational state: idle, processing, waiting, suspended. The FSM.</summary>
public sealed class StatePrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.State");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.state.*") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int transitions = 0;
        string? lastState = null;
        foreach (var ev in events)
        {
            if (ev.Type.Value == "agent.state.changed")
            {
                transitions++;
                lastState = "changed";
            }
        }
        var mutations = new List<Mutation>
        {
            new UpdateStateMutation(Id, "transitions", transitions),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
        if (lastState is not null)
            mutations.Add(new UpdateStateMutation(Id, "lastTransition", lastState));
        return mutations;
    }
}

/// <summary>What this agent is permitted to do. Received from above, scoped, revocable.</summary>
public sealed class AuthorityPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Authority");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.authority.*"),
        new SubscriptionPattern("authority.*"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int granted = 0, revoked = 0;
        foreach (var ev in events)
        {
            var t = ev.Type.Value;
            if (t == "agent.authority.granted") granted++;
            else if (t == "agent.authority.revoked") revoked++;
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "authorityGrants", granted),
            new UpdateStateMutation(Id, "authorityRevocations", revoked),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Trust scores this agent holds toward other actors. Asymmetric, non-transitive, decaying.</summary>
public sealed class TrustPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Trust");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.trust.*"),
        new SubscriptionPattern("trust.*"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int assessments = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.trust.assessed") assessments++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "trustAssessments", assessments),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Resource constraints: token budget, API calls, time limits, cost ceiling.</summary>
public sealed class BudgetPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Budget");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.budget.*") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int allocated = 0, exhausted = 0;
        foreach (var ev in events)
        {
            switch (ev.Type.Value)
            {
                case "agent.budget.allocated": allocated++; break;
                case "agent.budget.exhausted": exhausted++; break;
            }
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "allocations", allocated),
            new UpdateStateMutation(Id, "exhaustions", exhausted),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Named function within a team: Builder, Reviewer, Guardian, CTO.</summary>
public sealed class RolePrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Role");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.role.*") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int assignments = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.role.assigned") assignments++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "roleAssignments", assignments),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Birth, expected duration, graceful shutdown conditions.</summary>
public sealed class LifespanPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Lifespan");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.lifespan.*") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int started = 0, ended = 0;
        foreach (var ev in events)
        {
            switch (ev.Type.Value)
            {
                case "agent.lifespan.started": started++; break;
                case "agent.lifespan.ended": ended++; break;
            }
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "agentsStarted", started),
            new UpdateStateMutation(Id, "agentsEnded", ended),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Current objective hierarchy. What the agent is trying to accomplish.</summary>
public sealed class GoalPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Goal");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.goal.*") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int set = 0, completed = 0, abandoned = 0;
        foreach (var ev in events)
        {
            switch (ev.Type.Value)
            {
                case "agent.goal.set": set++; break;
                case "agent.goal.completed": completed++; break;
                case "agent.goal.abandoned": abandoned++; break;
            }
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "goalsSet", set),
            new UpdateStateMutation(Id, "goalsCompleted", completed),
            new UpdateStateMutation(Id, "goalsAbandoned", abandoned),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

// ════════════════════════════════════════════════════════════════════════
// OPERATIONAL PRIMITIVES (13) — Define what an agent DOES
// ════════════════════════════════════════════════════════════════════════

/// <summary>Passive perception. Events arrive via subscriptions.</summary>
public sealed class ObservePrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Observe");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.observed"),
        new SubscriptionPattern("agent.*"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int observed = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.observed") observed++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "eventsObserved", observed),
            new UpdateStateMutation(Id, "totalEventsReceived", events.Count),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Active perception. The agent queries the graph, stores, other agents.</summary>
public sealed class ProbePrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Probe");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.probed") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int probes = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.probed") probes++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "probesExecuted", probes),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>One-shot judgment. Assess a situation, produce a score/classification.</summary>
public sealed class EvaluatePrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Evaluate");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.evaluated") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int evaluations = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.evaluated") evaluations++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "evaluations", evaluations),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Commit to an action. Takes evaluation output, produces a Decision.</summary>
public sealed class DecidePrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Decide");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.decided"),
        new SubscriptionPattern("agent.evaluated"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int decisions = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.decided") decisions++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "decisions", decisions),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Execute a decision. Emit events, create edges, modify graph state.</summary>
public sealed class ActPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Act");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.acted"),
        new SubscriptionPattern("agent.decided"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int actions = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.acted") actions++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "actionsExecuted", actions),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Assign work to another agent. Transfer a goal with authority and constraints.</summary>
public sealed class DelegatePrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Delegate");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.delegated") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int delegations = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.delegated") delegations++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "delegations", delegations),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Pass upward. "I can't handle this." Capability-limited.</summary>
public sealed class EscalatePrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Escalate");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.escalated") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int escalations = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.escalated") escalations++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "escalations", escalations),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Decline to act. "I won't do this." Values-limited. Emits refusal event with reason.</summary>
public sealed class RefusePrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Refuse");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.refused") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int refusals = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.refused") refusals++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "refusals", refusals),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Update Memory based on outcomes. Self-mutating.</summary>
public sealed class LearnPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Learn");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.learned"),
        new SubscriptionPattern("agent.goal.completed"),
        new SubscriptionPattern("agent.goal.abandoned"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int lessons = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.learned") lessons++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "lessonsLearned", lessons),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Read own State and Soul. Self-observation without mutation.</summary>
public sealed class IntrospectPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Introspect");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.introspected") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int introspections = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.introspected") introspections++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "introspections", introspections),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Send a message to another agent or channel.</summary>
public sealed class CommunicatePrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Communicate");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.communicated"),
        new SubscriptionPattern("agent.channel.*"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int messages = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.communicated") messages++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "messagesSent", messages),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Fix a prior Act. Changes both graph state AND relationship state.</summary>
public sealed class RepairPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Repair");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.repaired") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int repairs = 0;
        foreach (var ev in events)
            if (ev.Type.Value == "agent.repaired") repairs++;
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "repairs", repairs),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Create a persistent monitoring condition. "Watch for X and alert me."</summary>
public sealed class ExpectPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Expect");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.expectation.*") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int set = 0, met = 0, expired = 0;
        foreach (var ev in events)
        {
            switch (ev.Type.Value)
            {
                case "agent.expectation.set": set++; break;
                case "agent.expectation.met": met++; break;
                case "agent.expectation.expired": expired++; break;
            }
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "expectationsSet", set),
            new UpdateStateMutation(Id, "expectationsMet", met),
            new UpdateStateMutation(Id, "expectationsExpired", expired),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

// ════════════════════════════════════════════════════════════════════════
// RELATIONAL PRIMITIVES (3) — Define how agents relate
// ════════════════════════════════════════════════════════════════════════

/// <summary>Bilateral agreement. Both parties must agree.</summary>
public sealed class ConsentPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Consent");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.consent.*") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int requested = 0, granted = 0, denied = 0;
        foreach (var ev in events)
        {
            switch (ev.Type.Value)
            {
                case "agent.consent.requested": requested++; break;
                case "agent.consent.granted": granted++; break;
                case "agent.consent.denied": denied++; break;
            }
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "consentRequested", requested),
            new UpdateStateMutation(Id, "consentGranted", granted),
            new UpdateStateMutation(Id, "consentDenied", denied),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Persistent bidirectional communication link between agents.</summary>
public sealed class ChannelPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Channel");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.channel.*") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int opened = 0, closed = 0;
        foreach (var ev in events)
        {
            switch (ev.Type.Value)
            {
                case "agent.channel.opened": opened++; break;
                case "agent.channel.closed": closed++; break;
            }
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "channelsOpened", opened),
            new UpdateStateMutation(Id, "channelsClosed", closed),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

/// <summary>Form a group. Multiple agents become a unit.</summary>
public sealed class CompositionPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Composition");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new() { new SubscriptionPattern("agent.composition.*") };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int formed = 0, dissolved = 0, joined = 0, left = 0;
        foreach (var ev in events)
        {
            switch (ev.Type.Value)
            {
                case "agent.composition.formed": formed++; break;
                case "agent.composition.dissolved": dissolved++; break;
                case "agent.composition.joined": joined++; break;
                case "agent.composition.left": left++; break;
            }
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "groupsFormed", formed),
            new UpdateStateMutation(Id, "groupsDissolved", dissolved),
            new UpdateStateMutation(Id, "membersJoined", joined),
            new UpdateStateMutation(Id, "membersLeft", left),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

// ════════════════════════════════════════════════════════════════════════
// MODAL PRIMITIVE (1) — Modifies how other primitives operate
// ════════════════════════════════════════════════════════════════════════

/// <summary>Reduce scope, confidence, or authority. "Do less, be more careful."</summary>
public sealed class AttenuationPrimitive : IPrimitive
{
    public PrimitiveId Id { get; } = new("agent.Attenuation");
    public Layer Layer { get; } = new(1);
    public List<SubscriptionPattern> Subscriptions { get; } = new()
    {
        new SubscriptionPattern("agent.attenuated"),
        new SubscriptionPattern("agent.attenuation.*"),
        new SubscriptionPattern("agent.budget.exhausted"),
    };
    public Cadence Cadence { get; } = new(1);

    public List<Mutation> Process(int tick, List<Event> events, Snapshot snapshot)
    {
        int attenuated = 0, lifted = 0, budgetTriggered = 0;
        foreach (var ev in events)
        {
            var t = ev.Type.Value;
            if (t == "agent.attenuated") attenuated++;
            else if (t == "agent.attenuation.lifted") lifted++;
            else if (t == "agent.budget.exhausted") budgetTriggered++;
        }
        return new List<Mutation>
        {
            new UpdateStateMutation(Id, "attenuations", attenuated),
            new UpdateStateMutation(Id, "lifts", lifted),
            new UpdateStateMutation(Id, "budgetTriggered", budgetTriggered),
            new UpdateStateMutation(Id, "lastTick", tick),
        };
    }
}

// ════════════════════════════════════════════════════════════════════════
// REGISTRATION
// ════════════════════════════════════════════════════════════════════════

/// <summary>Factory and registration for all 28 agent primitives.</summary>
public static class AgentPrimitiveFactory
{
    /// <summary>Returns all 28 agent primitives.</summary>
    public static List<IPrimitive> AllPrimitives() => new()
    {
        // Structural (11)
        new IdentityPrimitive(),
        new SoulPrimitive(),
        new ModelPrimitive(),
        new MemoryPrimitive(),
        new StatePrimitive(),
        new AuthorityPrimitive(),
        new TrustPrimitive(),
        new BudgetPrimitive(),
        new RolePrimitive(),
        new LifespanPrimitive(),
        new GoalPrimitive(),
        // Operational (13)
        new ObservePrimitive(),
        new ProbePrimitive(),
        new EvaluatePrimitive(),
        new DecidePrimitive(),
        new ActPrimitive(),
        new DelegatePrimitive(),
        new EscalatePrimitive(),
        new RefusePrimitive(),
        new LearnPrimitive(),
        new IntrospectPrimitive(),
        new CommunicatePrimitive(),
        new RepairPrimitive(),
        new ExpectPrimitive(),
        // Relational (3)
        new ConsentPrimitive(),
        new ChannelPrimitive(),
        new CompositionPrimitive(),
        // Modal (1)
        new AttenuationPrimitive(),
    };

    /// <summary>Registers all 28 agent primitives and activates them.</summary>
    public static void RegisterAll(PrimitiveRegistry registry)
    {
        foreach (var p in AllPrimitives())
        {
            registry.Register(p);
            registry.Activate(p.Id);
        }
    }

    /// <summary>Returns true if the primitive ID belongs to the agent layer.</summary>
    public static bool IsAgentPrimitive(PrimitiveId id) => id.Value.StartsWith("agent.");
}
