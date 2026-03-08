namespace EventGraph.Tests;

public class CanonicalFormTests
{
    [Fact]
    public void SortedKeysNoWhitespace()
    {
        var content = new Dictionary<string, object?> { ["b"] = 1, ["a"] = 2 };
        var json = CanonicalForm.CanonicalContentJson(content);
        Assert.DoesNotContain(" ", json);
        Assert.StartsWith("{\"a\"", json);
    }

    [Fact]
    public void PipeSeparatedFormat()
    {
        var canon = CanonicalForm.Build(
            1, new string('0', 64),
            new[] { "c2", "c1" },
            "eid", "trust.updated", "actor_alice", "conv_1",
            123456789, "{\"key\":\"val\"}");

        var parts = canon.Split('|');
        Assert.Equal("1", parts[0]);
        Assert.Equal(new string('0', 64), parts[1]);
        Assert.Equal("c1,c2", parts[2]); // sorted
        Assert.Equal("eid", parts[3]);
        Assert.Equal("trust.updated", parts[4]);
    }

    [Fact]
    public void EmptyCauses()
    {
        var canon = CanonicalForm.Build(1, "", Array.Empty<string>(), "eid", "system.bootstrapped", "s", "c", 0, "{}");
        var parts = canon.Split('|');
        Assert.Equal("", parts[2]);
    }
}

public class ComputeHashTests
{
    [Fact]
    public void Deterministic()
    {
        var h1 = CanonicalForm.ComputeHash("hello");
        var h2 = CanonicalForm.ComputeHash("hello");
        Assert.Equal(h1, h2);
    }

    [Fact]
    public void DifferentInput()
    {
        var h1 = CanonicalForm.ComputeHash("hello");
        var h2 = CanonicalForm.ComputeHash("world");
        Assert.NotEqual(h1, h2);
    }

    [Fact]
    public void Returns64Hex()
    {
        var h = CanonicalForm.ComputeHash("test");
        Assert.Equal(64, h.Value.Length);
    }
}

public class EventFactoryTests
{
    [Fact]
    public void NewEventIdGeneratesValidV7()
    {
        var eid = EventFactory.NewEventId();
        Assert.Equal(36, eid.Value.Length);
        Assert.Equal('7', eid.Value[14]);
    }

    [Fact]
    public void CreateBootstrapValid()
    {
        var signer = new NoopSigner();
        var source = new ActorId("actor_alice");
        var ev = EventFactory.CreateBootstrap(source, signer);

        Assert.Equal(1, ev.Version);
        Assert.Equal("system.bootstrapped", ev.Type.Value);
        Assert.Equal("actor_alice", ev.Source.Value);
        Assert.True(ev.PrevHash.IsZero);
        Assert.Equal(1, ev.Causes.Count);
        Assert.Equal(ev.Id, ev.Causes[0]); // self-referencing
    }

    [Fact]
    public void CreateEventValid()
    {
        var signer = new NoopSigner();
        var boot = EventFactory.CreateBootstrap(new ActorId("alice"), signer);

        var ev = EventFactory.CreateEvent(
            new EventType("trust.updated"),
            new ActorId("alice"),
            new Dictionary<string, object?> { ["score"] = 0.8 },
            new List<EventId> { boot.Id },
            new ConversationId("conv_1"),
            boot.Hash, signer);

        Assert.Equal("trust.updated", ev.Type.Value);
        Assert.Equal(boot.Hash, ev.PrevHash);
        Assert.Equal(1, ev.Causes.Count);
    }

    [Fact]
    public void ContentIsDefensiveCopy()
    {
        var signer = new NoopSigner();
        var ev = EventFactory.CreateBootstrap(new ActorId("alice"), signer);
        var content = ev.Content;
        Assert.False(ReferenceEquals(content, ev.Content));
    }
}

public class NumberFormattingTests
{
    [Fact]
    public void IntegerLikeFloatFormatsWithoutDecimal()
    {
        var content = new Dictionary<string, object?> { ["val"] = 1.0 };
        var json = CanonicalForm.CanonicalContentJson(content);
        // Go outputs 1, not 1.0 — we must match
        Assert.Equal("{\"val\":1}", json);
    }

    [Fact]
    public void FractionFormatsCorrectly()
    {
        var content = new Dictionary<string, object?> { ["val"] = 0.5 };
        var json = CanonicalForm.CanonicalContentJson(content);
        Assert.Equal("{\"val\":0.5}", json);
    }

    [Fact]
    public void NullOmittedFromCanonicalJson()
    {
        var content = new Dictionary<string, object?>
        {
            ["Actor"] = "actor_1",
            ["Scope"] = null,
        };
        var json = CanonicalForm.CanonicalContentJson(content);
        Assert.Equal("{\"Actor\":\"actor_1\"}", json);
    }
}

/// <summary>Cross-language conformance tests matching Go reference hashes.</summary>
public class ConformanceTests
{
    [Fact]
    public void BootstrapEventHash()
    {
        var content = new Dictionary<string, object?>
        {
            ["ActorID"] = "actor_00000000000000000000000000000001",
            ["ChainGenesis"] = "0000000000000000000000000000000000000000000000000000000000000000",
            ["Timestamp"] = "2023-11-14T22:13:20Z",
        };
        var contentJson = CanonicalForm.CanonicalContentJson(content);
        var canon = CanonicalForm.Build(
            1, "", Array.Empty<string>(),
            "019462a0-0000-7000-8000-000000000001",
            "system.bootstrapped",
            "actor_00000000000000000000000000000001",
            "conv_00000000000000000000000000000001",
            1700000000000000000, contentJson);

        // Must start with "1|||" (empty prev_hash AND empty causes)
        Assert.StartsWith("1|||", canon);
        var hash = CanonicalForm.ComputeHash(canon);
        Assert.Equal("f7cae7ae11c1232a932c64f2302432c0e304dffce80f3935e688980dfbafeb75", hash.Value);
    }

    [Fact]
    public void TrustUpdatedEventHash()
    {
        var content = new Dictionary<string, object?>
        {
            ["Actor"] = "actor_00000000000000000000000000000002",
            ["Cause"] = "019462a0-0000-7000-8000-000000000001",
            ["Current"] = 0.85,
            ["Domain"] = "code_review",
            ["Previous"] = 0.8,
        };
        var contentJson = CanonicalForm.CanonicalContentJson(content);
        var canon = CanonicalForm.Build(
            1, "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
            new[] { "019462a0-0000-7000-8000-000000000001" },
            "019462a0-0000-7000-8000-000000000002",
            "trust.updated",
            "actor_00000000000000000000000000000001",
            "conv_00000000000000000000000000000001",
            1700000001000000000, contentJson);

        var hash = CanonicalForm.ComputeHash(canon);
        Assert.Equal("b2fbcd2684868f0b0d07d2f5136b52f14b8e749da7b4b7bae2a22f67147152b7", hash.Value);
    }

    [Fact]
    public void EdgeCreatedKeyOrderingHash()
    {
        var content = new Dictionary<string, object?>
        {
            ["Weight"] = 0.5,
            ["From"] = "actor_00000000000000000000000000000001",
            ["To"] = "actor_00000000000000000000000000000002",
            ["EdgeType"] = "Trust",
            ["Direction"] = "Centripetal",
        };
        var contentJson = CanonicalForm.CanonicalContentJson(content);
        var canon = CanonicalForm.Build(
            1, "b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3",
            new[] { "019462a0-0000-7000-8000-000000000001" },
            "019462a0-0000-7000-8000-000000000003",
            "edge.created",
            "actor_00000000000000000000000000000001",
            "conv_00000000000000000000000000000001",
            1700000002000000000, contentJson);

        var hash = CanonicalForm.ComputeHash(canon);
        Assert.Equal("4e5c6710ca9325676663b4a66d2e82114fcd8fb49dbe5705795051e0b0be374c", hash.Value);
    }
}
