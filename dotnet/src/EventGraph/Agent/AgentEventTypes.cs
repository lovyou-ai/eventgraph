namespace EventGraph.Agent;

/// <summary>All 45 agent event type constants. All use the "agent." prefix.</summary>
public static class AgentEventTypes
{
    // ── Structural events ────────────────────────────────────────────────
    public static readonly EventType IdentityCreated = new("agent.identity.created");
    public static readonly EventType IdentityRotated = new("agent.identity.rotated");
    public static readonly EventType SoulImprinted = new("agent.soul.imprinted");
    public static readonly EventType ModelBound = new("agent.model.bound");
    public static readonly EventType ModelChanged = new("agent.model.changed");
    public static readonly EventType MemoryUpdated = new("agent.memory.updated");
    public static readonly EventType StateChanged = new("agent.state.changed");
    public static readonly EventType AuthorityGranted = new("agent.authority.granted");
    public static readonly EventType AuthorityRevoked = new("agent.authority.revoked");
    public static readonly EventType TrustAssessed = new("agent.trust.assessed");
    public static readonly EventType BudgetAllocated = new("agent.budget.allocated");
    public static readonly EventType BudgetExhausted = new("agent.budget.exhausted");
    public static readonly EventType RoleAssigned = new("agent.role.assigned");
    public static readonly EventType LifespanStarted = new("agent.lifespan.started");
    public static readonly EventType LifespanExtended = new("agent.lifespan.extended");
    public static readonly EventType LifespanEnded = new("agent.lifespan.ended");
    public static readonly EventType GoalSet = new("agent.goal.set");
    public static readonly EventType GoalCompleted = new("agent.goal.completed");
    public static readonly EventType GoalAbandoned = new("agent.goal.abandoned");

    // ── Operational events ───────────────────────────────────────────────
    public static readonly EventType Observed = new("agent.observed");
    public static readonly EventType Probed = new("agent.probed");
    public static readonly EventType Evaluated = new("agent.evaluated");
    public static readonly EventType Decided = new("agent.decided");
    public static readonly EventType Acted = new("agent.acted");
    public static readonly EventType Delegated = new("agent.delegated");
    public static readonly EventType Escalated = new("agent.escalated");
    public static readonly EventType Refused = new("agent.refused");
    public static readonly EventType Learned = new("agent.learned");
    public static readonly EventType Introspected = new("agent.introspected");
    public static readonly EventType Communicated = new("agent.communicated");
    public static readonly EventType Repaired = new("agent.repaired");
    public static readonly EventType ExpectationSet = new("agent.expectation.set");
    public static readonly EventType ExpectationMet = new("agent.expectation.met");
    public static readonly EventType ExpectationExpired = new("agent.expectation.expired");

    // ── Relational events ────────────────────────────────────────────────
    public static readonly EventType ConsentRequested = new("agent.consent.requested");
    public static readonly EventType ConsentGranted = new("agent.consent.granted");
    public static readonly EventType ConsentDenied = new("agent.consent.denied");
    public static readonly EventType ChannelOpened = new("agent.channel.opened");
    public static readonly EventType ChannelClosed = new("agent.channel.closed");
    public static readonly EventType CompositionFormed = new("agent.composition.formed");
    public static readonly EventType CompositionDissolved = new("agent.composition.dissolved");
    public static readonly EventType CompositionJoined = new("agent.composition.joined");
    public static readonly EventType CompositionLeft = new("agent.composition.left");

    // ── Modal events ─────────────────────────────────────────────────────
    public static readonly EventType Attenuated = new("agent.attenuated");
    public static readonly EventType AttenuationLifted = new("agent.attenuation.lifted");

    /// <summary>Returns all 45 registered agent event types.</summary>
    public static List<EventType> All() => new()
    {
        // Structural
        IdentityCreated, IdentityRotated,
        SoulImprinted,
        ModelBound, ModelChanged,
        MemoryUpdated,
        StateChanged,
        AuthorityGranted, AuthorityRevoked,
        TrustAssessed,
        BudgetAllocated, BudgetExhausted,
        RoleAssigned,
        LifespanStarted, LifespanExtended, LifespanEnded,
        GoalSet, GoalCompleted, GoalAbandoned,
        // Operational
        Observed, Probed,
        Evaluated, Decided,
        Acted, Delegated,
        Escalated, Refused,
        Learned, Introspected,
        Communicated, Repaired,
        ExpectationSet, ExpectationMet, ExpectationExpired,
        // Relational
        ConsentRequested, ConsentGranted, ConsentDenied,
        ChannelOpened, ChannelClosed,
        CompositionFormed, CompositionDissolved,
        CompositionJoined, CompositionLeft,
        // Modal
        Attenuated, AttenuationLifted,
    };
}
