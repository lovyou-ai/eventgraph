namespace EventGraph;

/// <summary>
/// Social grammar providing the 15 social grammar operations and 4 named functions
/// that create properly hash-chained events on an IStore. Ports from the Go
/// reference implementation.
/// </summary>
public sealed class Grammar
{
    private readonly IStore _store;

    public Grammar(IStore store) => _store = store;

    /// <summary>Exposes the underlying store for edge validation (e.g. Sever).</summary>
    internal IStore Store => _store;

    private Hash PrevHash()
    {
        var head = _store.Head();
        return head.IsSome ? head.Unwrap().Hash : Hash.Zero();
    }

    // --- Vertex operations (7) ---

    /// <summary>Creates independent content. Requires at least one cause. (Operation 1)</summary>
    public Event Emit(ActorId source, string body, ConversationId conversationId, List<EventId> causes, ISigner signer)
    {
        if (causes.Count == 0)
            throw new ArgumentException("Emit requires at least one cause");

        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.emitted"), source,
            new Dictionary<string, object?> { ["Body"] = body },
            causes, conversationId, PrevHash(), signer));
    }

    /// <summary>Creates causally dependent, subordinate content. (Operation 2)</summary>
    public Event Respond(ActorId source, string body, EventId parent, ConversationId conversationId, ISigner signer)
    {
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.responded"), source,
            new Dictionary<string, object?> { ["Body"] = body, ["Parent"] = parent.Value },
            new List<EventId> { parent }, conversationId, PrevHash(), signer));
    }

    /// <summary>Creates causally dependent but independent content. (Operation 3)</summary>
    public Event Derive(ActorId source, string body, EventId sourceEvent, ConversationId conversationId, ISigner signer)
    {
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.derived"), source,
            new Dictionary<string, object?> { ["Body"] = body, ["Source"] = sourceEvent.Value },
            new List<EventId> { sourceEvent }, conversationId, PrevHash(), signer));
    }

    /// <summary>Creates sequential content from the same author. (Operation 4)</summary>
    public Event Extend(ActorId source, string body, EventId previous, ConversationId conversationId, ISigner signer)
    {
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.extended"), source,
            new Dictionary<string, object?> { ["Body"] = body, ["Previous"] = previous.Value },
            new List<EventId> { previous }, conversationId, PrevHash(), signer));
    }

    /// <summary>Tombstones own content. Only the original author can retract. (Operation 5)</summary>
    public Event Retract(ActorId source, EventId target, string reason, ConversationId conversationId, ISigner signer)
    {
        var targetEvent = _store.Get(target);
        if (targetEvent.Source != source)
            throw new InvalidOperationException(
                $"Actor {source.Value} cannot retract event {target.Value} authored by {targetEvent.Source.Value}");

        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.retracted"), source,
            new Dictionary<string, object?> { ["Target"] = target.Value, ["Reason"] = reason },
            new List<EventId> { target }, conversationId, PrevHash(), signer));
    }

    /// <summary>Attaches metadata to existing content. (Operation 6)</summary>
    public Event Annotate(ActorId source, EventId target, string key, string value, ConversationId conversationId, ISigner signer)
    {
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.annotated"), source,
            new Dictionary<string, object?> { ["Target"] = target.Value, ["Key"] = key, ["Value"] = value },
            new List<EventId> { target }, conversationId, PrevHash(), signer));
    }

    /// <summary>Joins two or more independent subtrees. Requires at least two sources. (Operation 15)</summary>
    public Event Merge(ActorId source, string body, List<EventId> sources, ConversationId conversationId, ISigner signer)
    {
        if (sources.Count < 2)
            throw new ArgumentException("Merge requires at least two sources");

        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.merged"), source,
            new Dictionary<string, object?> { ["Body"] = body, ["Sources"] = sources.Select(s => (object?)s.Value).ToList() },
            sources, conversationId, PrevHash(), signer));
    }

    // --- Edge operations (8) ---

    private Event RecordEdge(
        ActorId source, ActorId from, ActorId to,
        string edgeType, double weightVal, string direction,
        Option<DomainScope> scope, List<EventId> causes,
        ConversationId conversationId, ISigner signer)
    {
        var scopeVal = scope.IsSome ? (object?)scope.Unwrap().Value : null;
        return _store.Append(EventFactory.CreateEvent(
            new EventType("edge.created"), source,
            new Dictionary<string, object?>
            {
                ["From"] = from.Value,
                ["To"] = to.Value,
                ["EdgeType"] = edgeType,
                ["Weight"] = weightVal,
                ["Direction"] = direction,
                ["Scope"] = scopeVal
            },
            causes, conversationId, PrevHash(), signer));
    }

    /// <summary>Creates a content-free centripetal edge toward a vertex. (Operation 7)</summary>
    public Event Acknowledge(ActorId source, EventId targetEvent, ActorId targetActor,
        ConversationId conversationId, ISigner signer)
    {
        return RecordEdge(source, source, targetActor,
            "acknowledgement", 0.0, "centripetal",
            Option<DomainScope>.None(),
            new List<EventId> { targetEvent },
            conversationId, signer);
    }

    /// <summary>Redistributes a vertex into the actor's subgraph. (Operation 8)</summary>
    public Event Propagate(ActorId source, EventId targetEvent, ActorId targetActor,
        ConversationId conversationId, ISigner signer)
    {
        return RecordEdge(source, source, targetActor,
            "reference", 0.0, "centrifugal",
            Option<DomainScope>.None(),
            new List<EventId> { targetEvent },
            conversationId, signer);
    }

    /// <summary>Creates a reputation-staked edge toward a vertex. (Operation 9)</summary>
    public Event Endorse(ActorId source, EventId targetEvent, ActorId targetActor,
        Weight weight, Option<DomainScope> scope,
        ConversationId conversationId, ISigner signer)
    {
        return RecordEdge(source, source, targetActor,
            "endorsement", weight.Value, "centripetal",
            scope,
            new List<EventId> { targetEvent },
            conversationId, signer);
    }

    /// <summary>Creates a persistent, future-oriented edge to an actor. (Operation 10)</summary>
    public Event Subscribe(ActorId source, ActorId targetActor,
        Option<DomainScope> scope, EventId cause,
        ConversationId conversationId, ISigner signer)
    {
        return RecordEdge(source, source, targetActor,
            "subscription", 0.0, "centripetal",
            scope,
            new List<EventId> { cause },
            conversationId, signer);
    }

    /// <summary>Creates a private, bidirectional, content-bearing edge. (Operation 11)</summary>
    public Event Channel(ActorId source, ActorId targetActor,
        Option<DomainScope> scope, EventId cause,
        ConversationId conversationId, ISigner signer)
    {
        return RecordEdge(source, source, targetActor,
            "channel", 0.0, "centripetal",
            scope,
            new List<EventId> { cause },
            conversationId, signer);
    }

    /// <summary>Grants authority for another actor to operate as you. (Operation 12)</summary>
    public Event Delegate(ActorId source, ActorId targetActor,
        DomainScope scope, Weight weight, EventId cause,
        ConversationId conversationId, ISigner signer)
    {
        return RecordEdge(source, source, targetActor,
            "delegation", weight.Value, "centrifugal",
            Option<DomainScope>.Some(scope),
            new List<EventId> { cause },
            conversationId, signer);
    }

    /// <summary>Records a consent proposal signed by partyA. (Operation 13)</summary>
    public Event Consent(ActorId partyA, ActorId partyB, string agreement,
        DomainScope scope, EventId cause,
        ConversationId conversationId, ISigner signer)
    {
        return _store.Append(EventFactory.CreateEvent(
            new EventType("grammar.consented"), partyA,
            new Dictionary<string, object?>
            {
                ["PartyA"] = partyA.Value,
                ["PartyB"] = partyB.Value,
                ["Agreement"] = agreement,
                ["Scope"] = scope.Value
            },
            new List<EventId> { cause }, conversationId, PrevHash(), signer));
    }

    /// <summary>Removes a subscription, channel, or delegation via edge supersession. (Operation 14)</summary>
    public Event Sever(ActorId source, EdgeId previousEdge, EventId cause,
        ConversationId conversationId, ISigner signer)
    {
        if (cause == default)
            throw new ArgumentException("Sever: cause must not be zero");

        // Convert EdgeId to EventId for lookup (edge events are events)
        var edgeEventId = new EventId(previousEdge.Value);
        var edgeEv = _store.Get(edgeEventId);

        // Verify it's an edge.created event
        if (edgeEv.Type != new EventType("edge.created"))
            throw new InvalidOperationException(
                $"Sever: event {previousEdge.Value} is not an edge.created event");

        // Only subscriptions, channels, and delegations are severable
        var edgeType = edgeEv.Content["EdgeType"]?.ToString();
        if (edgeType != "subscription" && edgeType != "channel" && edgeType != "delegation")
            throw new InvalidOperationException(
                $"Sever: edge type {edgeType} is not severable (only subscription, channel, delegation)");

        // Verify the actor is a party to the edge
        var fromVal = edgeEv.Content["From"]?.ToString();
        var toVal = edgeEv.Content["To"]?.ToString();
        if (fromVal != source.Value && toVal != source.Value)
            throw new InvalidOperationException(
                $"Sever: actor {source.Value} is not a party to edge {previousEdge.Value} (from={fromVal}, to={toVal})");

        // Include both the edge event and the trigger cause in the causal set
        var causes = new List<EventId> { edgeEventId };
        if (cause != edgeEventId)
            causes.Add(cause);

        return _store.Append(EventFactory.CreateEvent(
            new EventType("edge.superseded"), source,
            new Dictionary<string, object?>
            {
                ["PreviousEdge"] = previousEdge.Value,
                ["Reason"] = cause.Value
            },
            causes, conversationId, PrevHash(), signer));
    }

    // --- Named functions (4) ---

    /// <summary>Respond + dispute flag: formal dispute that follows content.</summary>
    public (Event Response, Event DisputeFlag) Challenge(ActorId source, string body,
        EventId target, ConversationId conversationId, ISigner signer)
    {
        var response = Respond(source, body, target, conversationId, signer);
        var disputeFlag = Annotate(source, response.Id, "dispute", "challenged", conversationId, signer);
        return (response, disputeFlag);
    }

    /// <summary>Propagate + Channel: directed sharing to a specific person.</summary>
    public (Event PropagateEv, Event ChannelEv) Recommend(ActorId source,
        EventId target, ActorId targetActor,
        ConversationId conversationId, ISigner signer)
    {
        var propagateEv = Propagate(source, target, targetActor, conversationId, signer);
        var channelEv = Channel(source, targetActor, Option<DomainScope>.None(), propagateEv.Id, conversationId, signer);
        return (propagateEv, channelEv);
    }

    /// <summary>Endorse + Subscribe: trust-staked introduction of a new actor.</summary>
    public (Event EndorseEv, Event SubscribeEv) Invite(ActorId source,
        ActorId target, Weight weight, Option<DomainScope> scope,
        EventId cause, ConversationId conversationId, ISigner signer)
    {
        var endorseEv = Endorse(source, cause, target, weight, scope, conversationId, signer);
        var subscribeEv = Subscribe(source, target, scope, endorseEv.Id, conversationId, signer);
        return (endorseEv, subscribeEv);
    }

    /// <summary>Subscribe after Sever: reconciliation with history intact.</summary>
    public Event Forgive(ActorId source, EventId severEvent, ActorId target,
        Option<DomainScope> scope, ConversationId conversationId, ISigner signer)
    {
        return Subscribe(source, target, scope, severEvent, conversationId, signer);
    }
}
