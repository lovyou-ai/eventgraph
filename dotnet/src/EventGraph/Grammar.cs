namespace EventGraph;

/// <summary>
/// Social grammar providing high-level vertex operations that create properly
/// hash-chained events on an IStore. Ports the vertex operations from the Go
/// reference implementation.
/// </summary>
public sealed class Grammar
{
    private readonly IStore _store;

    public Grammar(IStore store) => _store = store;

    /// <summary>Creates independent content. Requires at least one cause.</summary>
    public Event Emit(ActorId source, string body, ConversationId conversationId, List<EventId> causes, ISigner signer)
    {
        if (causes.Count == 0)
            throw new ArgumentException("Emit requires at least one cause");

        var head = _store.Head();
        var prevHash = head.IsSome ? head.Unwrap().Hash : Hash.Zero();
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.emitted"), source,
            new Dictionary<string, object?> { ["Body"] = body },
            causes, conversationId, prevHash, signer));
    }

    /// <summary>Creates causally dependent, subordinate content.</summary>
    public Event Respond(ActorId source, string body, EventId parent, ConversationId conversationId, ISigner signer)
    {
        var head = _store.Head();
        var prevHash = head.IsSome ? head.Unwrap().Hash : Hash.Zero();
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.responded"), source,
            new Dictionary<string, object?> { ["Body"] = body, ["Parent"] = parent.Value },
            new List<EventId> { parent }, conversationId, prevHash, signer));
    }

    /// <summary>Creates causally dependent but independent content.</summary>
    public Event Derive(ActorId source, string body, EventId sourceEvent, ConversationId conversationId, ISigner signer)
    {
        var head = _store.Head();
        var prevHash = head.IsSome ? head.Unwrap().Hash : Hash.Zero();
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.derived"), source,
            new Dictionary<string, object?> { ["Body"] = body, ["Source"] = sourceEvent.Value },
            new List<EventId> { sourceEvent }, conversationId, prevHash, signer));
    }

    /// <summary>Creates sequential content from the same author.</summary>
    public Event Extend(ActorId source, string body, EventId previous, ConversationId conversationId, ISigner signer)
    {
        var head = _store.Head();
        var prevHash = head.IsSome ? head.Unwrap().Hash : Hash.Zero();
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.extended"), source,
            new Dictionary<string, object?> { ["Body"] = body, ["Previous"] = previous.Value },
            new List<EventId> { previous }, conversationId, prevHash, signer));
    }

    /// <summary>Tombstones own content. Only the original author can retract.</summary>
    public Event Retract(ActorId source, EventId target, string reason, ConversationId conversationId, ISigner signer)
    {
        var targetEvent = _store.Get(target);
        if (targetEvent.Source != source)
            throw new InvalidOperationException(
                $"Actor {source.Value} cannot retract event {target.Value} authored by {targetEvent.Source.Value}");

        var head = _store.Head();
        var prevHash = head.IsSome ? head.Unwrap().Hash : Hash.Zero();
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.retracted"), source,
            new Dictionary<string, object?> { ["Target"] = target.Value, ["Reason"] = reason },
            new List<EventId> { target }, conversationId, prevHash, signer));
    }

    /// <summary>Attaches metadata to existing content.</summary>
    public Event Annotate(ActorId source, EventId target, string key, string value, ConversationId conversationId, ISigner signer)
    {
        var head = _store.Head();
        var prevHash = head.IsSome ? head.Unwrap().Hash : Hash.Zero();
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.annotated"), source,
            new Dictionary<string, object?> { ["Target"] = target.Value, ["Key"] = key, ["Value"] = value },
            new List<EventId> { target }, conversationId, prevHash, signer));
    }

    /// <summary>Joins two or more independent subtrees. Requires at least two sources.</summary>
    public Event Merge(ActorId source, string body, List<EventId> sources, ConversationId conversationId, ISigner signer)
    {
        if (sources.Count < 2)
            throw new ArgumentException("Merge requires at least two sources");

        var head = _store.Head();
        var prevHash = head.IsSome ? head.Unwrap().Hash : Hash.Zero();
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.merged"), source,
            new Dictionary<string, object?> { ["Body"] = body, ["Sources"] = sources.Select(s => (object?)s.Value).ToList() },
            sources, conversationId, prevHash, signer));
    }
}
