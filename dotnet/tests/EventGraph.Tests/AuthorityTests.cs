namespace EventGraph.Tests;

public class AuthorityTests
{
    private static DefaultAuthorityChain NewChain(ITrustModel? model = null)
    {
        return new DefaultAuthorityChain(model ?? new DefaultTrustModel());
    }

    private static ActorId TestActorId(string name = "alice") => new($"actor_{name}");

    // ── 1. Default level is Notification ────────────────────────────────

    [Fact]
    public void DefaultLevelIsNotification()
    {
        var chain = NewChain();
        var result = chain.Evaluate(TestActorId(), "some.random.action");

        Assert.Equal(AuthorityLevel.Notification, result.Level);
        Assert.Single(result.Chain);
    }

    // ── 2. Policy exact match ───────────────────────────────────────────

    [Fact]
    public void PolicyExactMatch()
    {
        var chain = NewChain();
        chain.AddPolicy(new AuthorityPolicy("actor.suspend", AuthorityLevel.Required));

        var result = chain.Evaluate(TestActorId(), "actor.suspend");

        Assert.Equal(AuthorityLevel.Required, result.Level);
    }

    // ── 3. Policy wildcard match ────────────────────────────────────────

    [Fact]
    public void PolicyWildcardMatch()
    {
        var chain = NewChain();
        chain.AddPolicy(new AuthorityPolicy("trust.*", AuthorityLevel.Recommended));

        var result = chain.Evaluate(TestActorId(), "trust.update");

        Assert.Equal(AuthorityLevel.Recommended, result.Level);
    }

    // ── 4. Policy global wildcard ───────────────────────────────────────

    [Fact]
    public void PolicyGlobalWildcard()
    {
        var chain = NewChain();
        chain.AddPolicy(new AuthorityPolicy("*", AuthorityLevel.Required));

        var result = chain.Evaluate(TestActorId(), "anything.at.all");

        Assert.Equal(AuthorityLevel.Required, result.Level);
    }

    // ── 5. First match wins ─────────────────────────────────────────────

    [Fact]
    public void PolicyFirstMatchWins()
    {
        var chain = NewChain();
        chain.AddPolicy(new AuthorityPolicy("deploy", AuthorityLevel.Required));
        chain.AddPolicy(new AuthorityPolicy("deploy", AuthorityLevel.Notification));

        var result = chain.Evaluate(TestActorId(), "deploy");

        Assert.Equal(AuthorityLevel.Required, result.Level);
    }

    // ── 6. No match falls back to Notification ──────────────────────────

    [Fact]
    public void PolicyNoMatchFallsBackToNotification()
    {
        var chain = NewChain();
        chain.AddPolicy(new AuthorityPolicy("deploy", AuthorityLevel.Required));

        var result = chain.Evaluate(TestActorId(), "review");

        Assert.Equal(AuthorityLevel.Notification, result.Level);
    }

    // ── 7. Trust downgrade Required to Recommended ──────────────────────

    [Fact]
    public void TrustDoesNotDowngradeWhenTrustTooLow()
    {
        // Default trust model starts at 0.0 — below any MinTrust threshold
        var chain = NewChain();
        chain.AddPolicy(new AuthorityPolicy("deploy", AuthorityLevel.Required, MinTrust: new Score(0.001)));

        var result = chain.Evaluate(TestActorId(), "deploy");

        // Trust is 0.0 which is below 0.001, so stays Required
        Assert.Equal(AuthorityLevel.Required, result.Level);
    }

    // ── 8. Chain returns single link ────────────────────────────────────

    [Fact]
    public void ChainReturnsSingleLink()
    {
        var chain = NewChain();
        var actorId = TestActorId();

        var links = chain.Chain(actorId, "any.action");

        Assert.Single(links);
        Assert.Equal(actorId, links[0].Actor);
    }

    // ── 9. Result weight is 1.0 ─────────────────────────────────────────

    [Fact]
    public void ResultWeightIsOne()
    {
        var chain = NewChain();
        var result = chain.Evaluate(TestActorId(), "test");

        Assert.Equal(1.0, result.Weight.Value);
    }

    // ── 10. Grant and Revoke are no-op ──────────────────────────────────

    [Fact]
    public void GrantAndRevokeAreNoOp()
    {
        var chain = NewChain();
        var from = TestActorId("alice");
        var to = TestActorId("bob");
        var scope = new DomainScope("code_review");

        // Should not throw
        chain.Grant(from, to, scope, new Score(0.8));
        chain.Revoke(from, to, scope);
    }

    // ── MatchesAction helper ────────────────────────────────────────────

    [Theory]
    [InlineData("*", "anything", true)]
    [InlineData("deploy", "deploy", true)]
    [InlineData("deploy", "review", false)]
    [InlineData("trust.*", "trust.update", true)]
    [InlineData("trust.*", "trust.", true)]
    [InlineData("trust.*", "other.action", false)]
    [InlineData("a.b.*", "a.b.c", true)]
    [InlineData("a.b.*", "a.c", false)]
    public void MatchesActionHelper(string pattern, string action, bool expected)
    {
        Assert.Equal(expected, DefaultAuthorityChain.MatchesAction(pattern, action));
    }
}
