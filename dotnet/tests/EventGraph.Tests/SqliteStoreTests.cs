using EventGraph.Sqlite;

namespace EventGraph.Tests;

public class SqliteStoreTests : IDisposable
{
    private readonly SqliteStore _store;

    public SqliteStoreTests()
    {
        _store = new SqliteStore("Data Source=:memory:");
    }

    public void Dispose()
    {
        _store.Dispose();
    }

    private static Event Bootstrap() => EventFactory.CreateBootstrap(new ActorId("alice"), new NoopSigner());

    private static Event NextEvent(Event prev) => EventFactory.CreateEvent(
        new EventType("trust.updated"), new ActorId("alice"),
        new Dictionary<string, object?> { ["score"] = 0.5 },
        new List<EventId> { prev.Id },
        new ConversationId("conv_1"), prev.Hash, new NoopSigner());

    private static Event NextEventWithType(Event prev, EventType type, ActorId source, ConversationId convId) =>
        EventFactory.CreateEvent(
            type, source,
            new Dictionary<string, object?> { ["score"] = 0.5 },
            new List<EventId> { prev.Id },
            convId, prev.Hash, new NoopSigner());

    // ── Basic CRUD ──────────────────────────────────────────────────────

    [Fact]
    public void AppendAndGet()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        var retrieved = _store.Get(boot.Id);
        Assert.Equal(boot.Id, retrieved.Id);
    }

    [Fact]
    public void HeadEmpty()
    {
        Assert.True(_store.Head().IsNone);
    }

    [Fact]
    public void HeadAfterAppend()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        Assert.Equal(boot.Id, _store.Head().Unwrap().Id);
    }

    [Fact]
    public void Count()
    {
        Assert.Equal(0, _store.Count());
        _store.Append(Bootstrap());
        Assert.Equal(1, _store.Count());
    }

    [Fact]
    public void GetNonexistent()
    {
        Assert.Throws<EventNotFoundException>(() =>
            _store.Get(new EventId("019462a0-0000-7000-8000-000000000099")));
    }

    // ── Chain integrity ─────────────────────────────────────────────────

    [Fact]
    public void ChainOfEvents()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        var e1 = NextEvent(boot);
        _store.Append(e1);
        var e2 = NextEvent(e1);
        _store.Append(e2);
        Assert.Equal(3, _store.Count());
        Assert.Equal(e2.Id, _store.Head().Unwrap().Id);
    }

    [Fact]
    public void RejectsBrokenChain()
    {
        var boot = Bootstrap();
        _store.Append(boot);

        var bad = EventFactory.CreateEvent(
            new EventType("trust.updated"), new ActorId("alice"),
            new Dictionary<string, object?>(), new List<EventId> { boot.Id },
            new ConversationId("conv_1"), boot.PrevHash, new NoopSigner()); // wrong prev_hash

        Assert.Throws<ChainIntegrityException>(() => _store.Append(bad));
    }

    [Fact]
    public void VerifyChainEmpty()
    {
        var v = _store.VerifyChain();
        Assert.True(v.Valid);
        Assert.Equal(0, v.Length);
    }

    [Fact]
    public void VerifyChainValid()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        _store.Append(NextEvent(boot));
        var v = _store.VerifyChain();
        Assert.True(v.Valid);
        Assert.Equal(2, v.Length);
    }

    // ── Recent ──────────────────────────────────────────────────────────

    [Fact]
    public void Recent()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        var e1 = NextEvent(boot);
        _store.Append(e1);
        var e2 = NextEvent(e1);
        _store.Append(e2);

        var recent = _store.Recent(2);
        Assert.Equal(2, recent.Count);
        Assert.Equal(e2.Id, recent[0].Id); // newest first
        Assert.Equal(e1.Id, recent[1].Id);
    }

    // ── ByType ──────────────────────────────────────────────────────────

    [Fact]
    public void ByTypeFiltersCorrectly()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        var trustType = new EventType("trust.updated");
        var otherType = new EventType("edge.created");

        var e1 = NextEventWithType(boot, trustType, new ActorId("alice"), new ConversationId("conv_1"));
        _store.Append(e1);
        var e2 = NextEventWithType(e1, otherType, new ActorId("alice"), new ConversationId("conv_1"));
        _store.Append(e2);
        var e3 = NextEventWithType(e2, trustType, new ActorId("alice"), new ConversationId("conv_1"));
        _store.Append(e3);

        var results = _store.ByType(trustType, 10);
        Assert.Equal(2, results.Count);
        Assert.Equal(e3.Id, results[0].Id); // newest first
        Assert.Equal(e1.Id, results[1].Id);
    }

    [Fact]
    public void ByTypeRespectsLimit()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        var trustType = new EventType("trust.updated");

        var e1 = NextEventWithType(boot, trustType, new ActorId("alice"), new ConversationId("conv_1"));
        _store.Append(e1);
        var e2 = NextEventWithType(e1, trustType, new ActorId("alice"), new ConversationId("conv_1"));
        _store.Append(e2);

        var results = _store.ByType(trustType, 1);
        Assert.Single(results);
        Assert.Equal(e2.Id, results[0].Id); // newest first
    }

    // ── BySource ────────────────────────────────────────────────────────

    [Fact]
    public void BySourceFiltersCorrectly()
    {
        var boot = Bootstrap(); // source: alice
        _store.Append(boot);

        var e1 = NextEventWithType(boot, new EventType("trust.updated"), new ActorId("alice"), new ConversationId("conv_1"));
        _store.Append(e1);
        var e2 = NextEventWithType(e1, new EventType("trust.updated"), new ActorId("bob"), new ConversationId("conv_1"));
        _store.Append(e2);

        var results = _store.BySource(new ActorId("bob"), 10);
        Assert.Single(results);
        Assert.Equal(e2.Id, results[0].Id);
    }

    // ── ByConversation ──────────────────────────────────────────────────

    [Fact]
    public void ByConversationFiltersCorrectly()
    {
        var boot = Bootstrap();
        _store.Append(boot);

        var conv1 = new ConversationId("conv_1");
        var conv2 = new ConversationId("conv_2");

        var e1 = NextEventWithType(boot, new EventType("trust.updated"), new ActorId("alice"), conv1);
        _store.Append(e1);
        var e2 = NextEventWithType(e1, new EventType("trust.updated"), new ActorId("alice"), conv2);
        _store.Append(e2);
        var e3 = NextEventWithType(e2, new EventType("trust.updated"), new ActorId("alice"), conv1);
        _store.Append(e3);

        var results = _store.ByConversation(conv1, 10);
        Assert.Equal(2, results.Count);
        Assert.Equal(e3.Id, results[0].Id); // newest first
        Assert.Equal(e1.Id, results[1].Id);
    }

    // ── Ancestors ───────────────────────────────────────────────────────

    [Fact]
    public void AncestorsTraversesCausalChain()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        var e1 = NextEvent(boot);
        _store.Append(e1);
        var e2 = NextEvent(e1);
        _store.Append(e2);

        var ancestors = _store.Ancestors(e2.Id, 10);
        Assert.Equal(2, ancestors.Count); // e1 and boot
        Assert.Contains(ancestors, e => e.Id == e1.Id);
        Assert.Contains(ancestors, e => e.Id == boot.Id);
    }

    [Fact]
    public void AncestorsRespectsMaxDepth()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        var e1 = NextEvent(boot);
        _store.Append(e1);
        var e2 = NextEvent(e1);
        _store.Append(e2);

        var ancestors = _store.Ancestors(e2.Id, 1);
        Assert.Single(ancestors); // only e1
        Assert.Equal(e1.Id, ancestors[0].Id);
    }

    [Fact]
    public void AncestorsDoesNotIncludeStartEvent()
    {
        var boot = Bootstrap();
        _store.Append(boot);

        var ancestors = _store.Ancestors(boot.Id, 10);
        Assert.DoesNotContain(ancestors, e => e.Id == boot.Id);
    }

    [Fact]
    public void AncestorsThrowsForUnknownEvent()
    {
        Assert.Throws<EventNotFoundException>(() =>
            _store.Ancestors(new EventId("019462a0-0000-7000-8000-000000000099"), 10));
    }

    // ── Descendants ─────────────────────────────────────────────────────

    [Fact]
    public void DescendantsTraversesCausalChain()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        var e1 = NextEvent(boot);
        _store.Append(e1);
        var e2 = NextEvent(e1);
        _store.Append(e2);

        var descendants = _store.Descendants(boot.Id, 10);
        Assert.Equal(2, descendants.Count); // e1 and e2
        Assert.Contains(descendants, e => e.Id == e1.Id);
        Assert.Contains(descendants, e => e.Id == e2.Id);
    }

    [Fact]
    public void DescendantsRespectsMaxDepth()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        var e1 = NextEvent(boot);
        _store.Append(e1);
        var e2 = NextEvent(e1);
        _store.Append(e2);

        var descendants = _store.Descendants(boot.Id, 1);
        Assert.Single(descendants); // only e1
        Assert.Equal(e1.Id, descendants[0].Id);
    }

    [Fact]
    public void DescendantsDoesNotIncludeStartEvent()
    {
        var boot = Bootstrap();
        _store.Append(boot);
        var e1 = NextEvent(boot);
        _store.Append(e1);

        var descendants = _store.Descendants(boot.Id, 10);
        Assert.DoesNotContain(descendants, e => e.Id == boot.Id);
    }

    [Fact]
    public void DescendantsThrowsForUnknownEvent()
    {
        Assert.Throws<EventNotFoundException>(() =>
            _store.Descendants(new EventId("019462a0-0000-7000-8000-000000000099"), 10));
    }
}
