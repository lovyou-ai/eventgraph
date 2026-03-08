namespace EventGraph;

// ── AuthorityLink ──────────────────────────────────────────────────────

/// <summary>A single link in an authority chain — actor, level, and weight.</summary>
public sealed record AuthorityLink(ActorId Actor, AuthorityLevel Level, Score Weight);

// ── AuthorityResult ────────────────────────────────────────────────────

/// <summary>Result of evaluating authority for an action.</summary>
public sealed record AuthorityResult(
    AuthorityLevel Level,
    Score Weight,
    IReadOnlyList<AuthorityLink> Chain);

// ── AuthorityPolicy ────────────────────────────────────────────────────

/// <summary>Defines the authority requirements for an action pattern.</summary>
public sealed record AuthorityPolicy(
    string Action,
    AuthorityLevel Level,
    Score? MinTrust = null,
    DomainScope? Scope = null);

// ── IAuthorityChain ────────────────────────────────────────────────────

/// <summary>Evaluates authority. Returns weighted authority, not binary permission.</summary>
public interface IAuthorityChain
{
    /// <summary>Evaluate the authority level for an actor performing an action.</summary>
    AuthorityResult Evaluate(ActorId actor, string action);

    /// <summary>Return the authority chain for an actor performing an action.</summary>
    IReadOnlyList<AuthorityLink> Chain(ActorId actor, string action);

    /// <summary>Grant authority from one actor to another in a domain scope.</summary>
    void Grant(ActorId from, ActorId to, DomainScope scope, Score weight);

    /// <summary>Revoke authority from one actor to another in a domain scope.</summary>
    void Revoke(ActorId from, ActorId to, DomainScope scope);
}

// ── DefaultAuthorityChain ──────────────────────────────────────────────

/// <summary>
/// Flat authority model — no delegation chain.
/// All actions default to Notification unless a policy says otherwise.
/// Thread-safe with lock.
/// </summary>
public sealed class DefaultAuthorityChain : IAuthorityChain
{
    private readonly object _lock = new();
    private readonly List<AuthorityPolicy> _policies = new();
    private readonly ITrustModel _trustModel;

    public DefaultAuthorityChain(ITrustModel trustModel)
    {
        _trustModel = trustModel;
    }

    /// <summary>Register an authority policy. Policies are checked in order; first match wins.</summary>
    public void AddPolicy(AuthorityPolicy policy)
    {
        lock (_lock)
        {
            _policies.Add(policy);
        }
    }

    public AuthorityResult Evaluate(ActorId actor, string action)
    {
        AuthorityPolicy policy;
        lock (_lock)
        {
            policy = FindPolicy(action);
        }

        var level = policy.Level;

        // If actor has high enough trust, downgrade Required -> Recommended
        if (level == AuthorityLevel.Required && policy.MinTrust is not null)
        {
            // Build a minimal Actor to query trust — ITrustModel.Score takes Actor
            var metrics = _trustModel.Score(BuildMinimalActor(actor));
            if (metrics.Overall.Value >= policy.MinTrust.Value.Value)
            {
                level = AuthorityLevel.Recommended;
            }
        }

        var link = new AuthorityLink(actor, level, new Score(1.0));

        return new AuthorityResult(level, new Score(1.0), new[] { link });
    }

    public IReadOnlyList<AuthorityLink> Chain(ActorId actor, string action)
    {
        AuthorityPolicy policy;
        lock (_lock)
        {
            policy = FindPolicy(action);
        }

        return new[] { new AuthorityLink(actor, policy.Level, new Score(1.0)) };
    }

    /// <summary>Grant is a no-op in the flat model — would emit an edge.created event in full impl.</summary>
    public void Grant(ActorId from, ActorId to, DomainScope scope, Score weight)
    {
        // No-op stub — flat model has no delegation
    }

    /// <summary>Revoke is a no-op in the flat model — would emit a supersede event in full impl.</summary>
    public void Revoke(ActorId from, ActorId to, DomainScope scope)
    {
        // No-op stub — flat model has no delegation
    }

    // ── Private helpers ────────────────────────────────────────────────

    /// <summary>Find the first matching policy. Must be called under lock.</summary>
    private AuthorityPolicy FindPolicy(string action)
    {
        foreach (var p in _policies)
        {
            if (MatchesAction(p.Action, action))
                return p;
        }

        // Default: Notification level
        return new AuthorityPolicy("*", AuthorityLevel.Notification);
    }

    /// <summary>Build a minimal Actor for trust model queries.</summary>
    private static Actor BuildMinimalActor(ActorId id)
    {
        return new Actor(
            id,
            new PublicKey(new byte[32]),
            "authority-query",
            ActorType.System,
            null,
            0,
            ActorStatus.Active);
    }

    /// <summary>Match an action against a policy pattern. Supports exact, prefix wildcard, and global wildcard.</summary>
    public static bool MatchesAction(string pattern, string action)
    {
        if (pattern == "*")
            return true;

        if (pattern.Length > 0 && pattern[^1] == '*')
        {
            var prefix = pattern[..^1];
            return action.Length >= prefix.Length && action[..prefix.Length] == prefix;
        }

        return pattern == action;
    }
}
