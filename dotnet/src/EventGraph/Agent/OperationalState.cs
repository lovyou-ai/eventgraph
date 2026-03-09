namespace EventGraph.Agent;

/// <summary>
/// Agent operational state FSM. Strict transitions enforced.
/// Idle -> {Processing, Suspended, Retiring}
/// Processing -> {Idle, Waiting, Escalating, Refusing, Retiring}
/// Waiting -> {Processing, Idle, Retiring}
/// Escalating -> {Waiting, Idle}
/// Refusing -> {Idle}
/// Suspended -> {Idle, Retiring}
/// Retiring -> {Retired}
/// Retired -> {} (terminal)
/// </summary>
public enum OperationalState
{
    Idle,
    Processing,
    Waiting,
    Escalating,
    Refusing,
    Suspended,
    Retiring,
    Retired,
}

public static class OperationalStateMachine
{
    private static readonly Dictionary<OperationalState, HashSet<OperationalState>> Transitions = new()
    {
        [OperationalState.Idle] = new() { OperationalState.Processing, OperationalState.Suspended, OperationalState.Retiring },
        [OperationalState.Processing] = new() { OperationalState.Idle, OperationalState.Waiting, OperationalState.Escalating, OperationalState.Refusing, OperationalState.Retiring },
        [OperationalState.Waiting] = new() { OperationalState.Processing, OperationalState.Idle, OperationalState.Retiring },
        [OperationalState.Escalating] = new() { OperationalState.Waiting, OperationalState.Idle },
        [OperationalState.Refusing] = new() { OperationalState.Idle },
        [OperationalState.Suspended] = new() { OperationalState.Idle, OperationalState.Retiring },
        [OperationalState.Retiring] = new() { OperationalState.Retired },
        [OperationalState.Retired] = new(),
    };

    /// <summary>Returns true if the transition from <paramref name="from"/> to <paramref name="to"/> is valid.</summary>
    public static bool IsValidTransition(OperationalState from, OperationalState to) =>
        Transitions.TryGetValue(from, out var targets) && targets.Contains(to);

    /// <summary>Validates and returns the target state if the transition is valid; throws otherwise.</summary>
    public static OperationalState TransitionTo(OperationalState from, OperationalState to)
    {
        if (!IsValidTransition(from, to))
            throw new InvalidTransitionException(from.ToString(), to.ToString());
        return to;
    }

    /// <summary>Returns true if this is a terminal state (no valid outgoing transitions).</summary>
    public static bool IsTerminal(OperationalState state) => state == OperationalState.Retired;

    /// <summary>Returns true if the agent can perform actions in this state.</summary>
    public static bool CanAct(OperationalState state) => state == OperationalState.Processing;
}
