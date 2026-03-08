namespace EventGraph.Tests;

public class GrammarTests
{
    private static readonly ActorId Alice = new("alice");
    private static readonly ActorId Bob = new("bob");
    private static readonly ConversationId ConvId = new("conv_1");
    private static readonly ISigner Signer = new NoopSigner();

    private static (Grammar grammar, IStore store, Event bootstrap) Setup()
    {
        var store = new InMemoryStore();
        var boot = EventFactory.CreateBootstrap(Alice, Signer);
        store.Append(boot);
        var grammar = new Grammar(store);
        return (grammar, store, boot);
    }

    // --- Emit ---

    [Fact]
    public void Emit_CreatesEventWithCorrectType()
    {
        var (grammar, _, boot) = Setup();
        var ev = grammar.Emit(Alice, "Hello world", ConvId, new List<EventId> { boot.Id }, Signer);
        Assert.Equal("grammar.emitted", ev.Type.Value);
    }

    [Fact]
    public void Emit_ContentContainsBody()
    {
        var (grammar, _, boot) = Setup();
        var ev = grammar.Emit(Alice, "Hello world", ConvId, new List<EventId> { boot.Id }, Signer);
        Assert.Equal("Hello world", ev.Content["Body"]);
    }

    [Fact]
    public void Emit_CausesMatchProvided()
    {
        var (grammar, _, boot) = Setup();
        var ev = grammar.Emit(Alice, "Hello", ConvId, new List<EventId> { boot.Id }, Signer);
        Assert.Single(ev.Causes);
        Assert.Equal(boot.Id, ev.Causes[0]);
    }

    [Fact]
    public void Emit_ThrowsOnEmptyCauses()
    {
        var (grammar, _, _) = Setup();
        Assert.Throws<ArgumentException>(() =>
            grammar.Emit(Alice, "body", ConvId, new List<EventId>(), Signer));
    }

    [Fact]
    public void Emit_HashChainIsValid()
    {
        var (grammar, store, boot) = Setup();
        var ev = grammar.Emit(Alice, "Hello", ConvId, new List<EventId> { boot.Id }, Signer);
        Assert.Equal(boot.Hash, ev.PrevHash);
        var v = store.VerifyChain();
        Assert.True(v.Valid);
    }

    // --- Respond ---

    [Fact]
    public void Respond_CreatesEventWithCorrectType()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "original", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Respond(Alice, "reply", emitEv.Id, ConvId, Signer);
        Assert.Equal("grammar.responded", ev.Type.Value);
    }

    [Fact]
    public void Respond_ContentContainsBodyAndParent()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "original", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Respond(Alice, "reply", emitEv.Id, ConvId, Signer);
        Assert.Equal("reply", ev.Content["Body"]);
        Assert.Equal(emitEv.Id.Value, ev.Content["Parent"]);
    }

    [Fact]
    public void Respond_CauseIsParent()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "original", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Respond(Alice, "reply", emitEv.Id, ConvId, Signer);
        Assert.Single(ev.Causes);
        Assert.Equal(emitEv.Id, ev.Causes[0]);
    }

    // --- Derive ---

    [Fact]
    public void Derive_CreatesEventWithCorrectType()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "source material", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Derive(Alice, "derived work", emitEv.Id, ConvId, Signer);
        Assert.Equal("grammar.derived", ev.Type.Value);
    }

    [Fact]
    public void Derive_ContentContainsBodyAndSource()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "source material", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Derive(Alice, "derived work", emitEv.Id, ConvId, Signer);
        Assert.Equal("derived work", ev.Content["Body"]);
        Assert.Equal(emitEv.Id.Value, ev.Content["Source"]);
    }

    [Fact]
    public void Derive_CauseIsSourceEvent()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "source material", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Derive(Alice, "derived", emitEv.Id, ConvId, Signer);
        Assert.Single(ev.Causes);
        Assert.Equal(emitEv.Id, ev.Causes[0]);
    }

    // --- Extend ---

    [Fact]
    public void Extend_CreatesEventWithCorrectType()
    {
        var (grammar, _, boot) = Setup();
        var part1 = grammar.Emit(Alice, "part 1", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Extend(Alice, "part 2", part1.Id, ConvId, Signer);
        Assert.Equal("grammar.extended", ev.Type.Value);
    }

    [Fact]
    public void Extend_ContentContainsBodyAndPrevious()
    {
        var (grammar, _, boot) = Setup();
        var part1 = grammar.Emit(Alice, "part 1", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Extend(Alice, "part 2", part1.Id, ConvId, Signer);
        Assert.Equal("part 2", ev.Content["Body"]);
        Assert.Equal(part1.Id.Value, ev.Content["Previous"]);
    }

    [Fact]
    public void Extend_CauseIsPreviousEvent()
    {
        var (grammar, _, boot) = Setup();
        var part1 = grammar.Emit(Alice, "part 1", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Extend(Alice, "part 2", part1.Id, ConvId, Signer);
        Assert.Single(ev.Causes);
        Assert.Equal(part1.Id, ev.Causes[0]);
    }

    // --- Retract ---

    [Fact]
    public void Retract_CreatesEventWithCorrectType()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "regrettable", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Retract(Alice, emitEv.Id, "changed my mind", ConvId, Signer);
        Assert.Equal("grammar.retracted", ev.Type.Value);
    }

    [Fact]
    public void Retract_ContentContainsTargetAndReason()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "regrettable", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Retract(Alice, emitEv.Id, "changed my mind", ConvId, Signer);
        Assert.Equal(emitEv.Id.Value, ev.Content["Target"]);
        Assert.Equal("changed my mind", ev.Content["Reason"]);
    }

    [Fact]
    public void Retract_CauseIsTarget()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "regrettable", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Retract(Alice, emitEv.Id, "changed my mind", ConvId, Signer);
        Assert.Single(ev.Causes);
        Assert.Equal(emitEv.Id, ev.Causes[0]);
    }

    [Fact]
    public void Retract_ThrowsWhenNotAuthor()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "content", ConvId, new List<EventId> { boot.Id }, Signer);
        Assert.Throws<InvalidOperationException>(() =>
            grammar.Retract(Bob, emitEv.Id, "not mine", ConvId, Signer));
    }

    // --- Annotate ---

    [Fact]
    public void Annotate_CreatesEventWithCorrectType()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "content", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Annotate(Alice, emitEv.Id, "mood", "happy", ConvId, Signer);
        Assert.Equal("grammar.annotated", ev.Type.Value);
    }

    [Fact]
    public void Annotate_ContentContainsTargetKeyValue()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "content", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Annotate(Alice, emitEv.Id, "mood", "happy", ConvId, Signer);
        Assert.Equal(emitEv.Id.Value, ev.Content["Target"]);
        Assert.Equal("mood", ev.Content["Key"]);
        Assert.Equal("happy", ev.Content["Value"]);
    }

    [Fact]
    public void Annotate_CauseIsTarget()
    {
        var (grammar, _, boot) = Setup();
        var emitEv = grammar.Emit(Alice, "content", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Annotate(Alice, emitEv.Id, "mood", "happy", ConvId, Signer);
        Assert.Single(ev.Causes);
        Assert.Equal(emitEv.Id, ev.Causes[0]);
    }

    // --- Merge ---

    [Fact]
    public void Merge_CreatesEventWithCorrectType()
    {
        var (grammar, _, boot) = Setup();
        var ev1 = grammar.Emit(Alice, "branch A", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev2 = grammar.Emit(Alice, "branch B", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Merge(Alice, "merged", new List<EventId> { ev1.Id, ev2.Id }, ConvId, Signer);
        Assert.Equal("grammar.merged", ev.Type.Value);
    }

    [Fact]
    public void Merge_ContentContainsBodyAndSources()
    {
        var (grammar, _, boot) = Setup();
        var ev1 = grammar.Emit(Alice, "branch A", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev2 = grammar.Emit(Alice, "branch B", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Merge(Alice, "merged", new List<EventId> { ev1.Id, ev2.Id }, ConvId, Signer);
        Assert.Equal("merged", ev.Content["Body"]);
        var sources = (IList<object?>)ev.Content["Sources"]!;
        Assert.Equal(2, sources.Count);
        Assert.Equal(ev1.Id.Value, sources[0]);
        Assert.Equal(ev2.Id.Value, sources[1]);
    }

    [Fact]
    public void Merge_CausesAreAllSources()
    {
        var (grammar, _, boot) = Setup();
        var ev1 = grammar.Emit(Alice, "branch A", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev2 = grammar.Emit(Alice, "branch B", ConvId, new List<EventId> { boot.Id }, Signer);
        var ev = grammar.Merge(Alice, "merged", new List<EventId> { ev1.Id, ev2.Id }, ConvId, Signer);
        Assert.Equal(2, ev.Causes.Count);
    }

    [Fact]
    public void Merge_ThrowsWithFewerThanTwoSources()
    {
        var (grammar, _, boot) = Setup();
        var ev1 = grammar.Emit(Alice, "only one", ConvId, new List<EventId> { boot.Id }, Signer);
        Assert.Throws<ArgumentException>(() =>
            grammar.Merge(Alice, "bad merge", new List<EventId> { ev1.Id }, ConvId, Signer));
    }

    // --- Chain integrity across all operations ---

    [Fact]
    public void AllOperations_MaintainChainIntegrity()
    {
        var (grammar, store, boot) = Setup();

        var emitEv = grammar.Emit(Alice, "emit", ConvId, new List<EventId> { boot.Id }, Signer);
        grammar.Respond(Alice, "respond", emitEv.Id, ConvId, Signer);
        grammar.Derive(Alice, "derive", emitEv.Id, ConvId, Signer);
        grammar.Extend(Alice, "extend", emitEv.Id, ConvId, Signer);
        grammar.Retract(Alice, emitEv.Id, "reason", ConvId, Signer);
        grammar.Annotate(Alice, emitEv.Id, "k", "v", ConvId, Signer);

        var emit2 = grammar.Emit(Alice, "emit2", ConvId, new List<EventId> { boot.Id }, Signer);
        grammar.Merge(Alice, "merged", new List<EventId> { emitEv.Id, emit2.Id }, ConvId, Signer);

        // 1 bootstrap + 8 grammar ops = 9 events
        Assert.Equal(9, store.Count());

        var v = store.VerifyChain();
        Assert.True(v.Valid);
        Assert.Equal(9, v.Length);
    }
}
