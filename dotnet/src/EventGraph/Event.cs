using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

namespace EventGraph;

/// <summary>Anything that can sign bytes.</summary>
public interface ISigner
{
    Signature Sign(byte[] data);
}

/// <summary>Produces zero-filled signatures.</summary>
public sealed class NoopSigner : ISigner
{
    public Signature Sign(byte[] data) => new(new byte[64]);
}

/// <summary>Immutable event — validated at construction, guaranteed valid for lifetime.</summary>
public sealed class Event
{
    public int Version { get; }
    public EventId Id { get; }
    public EventType Type { get; }
    public long TimestampNanos { get; }
    public ActorId Source { get; }
    private readonly Dictionary<string, object?> _content;
    public IReadOnlyDictionary<string, object?> Content => new Dictionary<string, object?>(_content);
    public NonEmpty<EventId> Causes { get; }
    public ConversationId ConversationId { get; }
    public Hash Hash { get; }
    public Hash PrevHash { get; }
    public Signature Signature { get; }

    internal Event(
        int version, EventId id, EventType type, long timestampNanos,
        ActorId source, Dictionary<string, object?> content,
        NonEmpty<EventId> causes, ConversationId conversationId,
        Hash hash, Hash prevHash, Signature signature)
    {
        Version = version;
        Id = id;
        Type = type;
        TimestampNanos = timestampNanos;
        Source = source;
        _content = content;
        Causes = causes;
        ConversationId = conversationId;
        Hash = hash;
        PrevHash = prevHash;
        Signature = signature;
    }
}

public static class CanonicalForm
{
    /// <summary>Produce canonical JSON: sorted keys, no whitespace, omit null values.</summary>
    public static string CanonicalContentJson(Dictionary<string, object?> content)
    {
        var filtered = new SortedDictionary<string, object?>();
        foreach (var kv in content)
        {
            if (kv.Value is not null)
                filtered[kv.Key] = kv.Value;
        }
        return JsonSerializer.Serialize(filtered, new JsonSerializerOptions
        {
            WriteIndented = false,
            DefaultIgnoreCondition = System.Text.Json.Serialization.JsonIgnoreCondition.WhenWritingNull,
        });
    }

    /// <summary>Build the canonical string for hashing/signing.</summary>
    public static string Build(
        int version, string prevHash, IReadOnlyList<string> causes,
        string eventId, string eventType, string source,
        string conversationId, long timestampNanos, string contentJson)
    {
        var sortedCauses = causes.OrderBy(c => c, StringComparer.Ordinal).ToList();
        var causesStr = string.Join(",", sortedCauses);
        return $"{version}|{prevHash}|{causesStr}|{eventId}|{eventType}|{source}|{conversationId}|{timestampNanos}|{contentJson}";
    }

    /// <summary>SHA-256 of the canonical form.</summary>
    public static Hash ComputeHash(string canonical)
    {
        var bytes = SHA256.HashData(Encoding.UTF8.GetBytes(canonical));
        return new Hash(Convert.ToHexString(bytes).ToLowerInvariant());
    }
}

public static class EventFactory
{
    /// <summary>Generate a new UUID v7 EventId using the current time.</summary>
    public static EventId NewEventId()
    {
        var ms = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
        Span<byte> b = stackalloc byte[16];

        // Timestamp: 48 bits
        b[0] = (byte)(ms >> 40);
        b[1] = (byte)(ms >> 32);
        b[2] = (byte)(ms >> 24);
        b[3] = (byte)(ms >> 16);
        b[4] = (byte)(ms >> 8);
        b[5] = (byte)ms;

        // Random fill
        RandomNumberGenerator.Fill(b[6..]);

        // Version 7
        b[6] = (byte)((b[6] & 0x0F) | 0x70);
        // Variant 10xx
        b[8] = (byte)((b[8] & 0x3F) | 0x80);

        var s = $"{b[0]:x2}{b[1]:x2}{b[2]:x2}{b[3]:x2}-{b[4]:x2}{b[5]:x2}-{b[6]:x2}{b[7]:x2}-{b[8]:x2}{b[9]:x2}-{b[10]:x2}{b[11]:x2}{b[12]:x2}{b[13]:x2}{b[14]:x2}{b[15]:x2}";
        return new EventId(s);
    }

    /// <summary>Create, hash, and sign an event.</summary>
    public static Event CreateEvent(
        EventType eventType, ActorId source,
        Dictionary<string, object?> content, List<EventId> causes,
        ConversationId conversationId, Hash prevHash,
        ISigner signer, int version = 1)
    {
        var eventId = NewEventId();
        var timestampNanos = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() * 1_000_000;
        var contentJson = CanonicalForm.CanonicalContentJson(content);

        var canon = CanonicalForm.Build(
            version, prevHash.Value,
            causes.Select(c => c.Value).ToList(),
            eventId.Value, eventType.Value,
            source.Value, conversationId.Value,
            timestampNanos, contentJson);

        var hash = CanonicalForm.ComputeHash(canon);
        var sig = signer.Sign(Encoding.UTF8.GetBytes(canon));

        return new Event(version, eventId, eventType, timestampNanos,
            source, content, NonEmpty<EventId>.Of(causes),
            conversationId, hash, prevHash, sig);
    }

    /// <summary>Create the genesis/bootstrap event.</summary>
    public static Event CreateBootstrap(ActorId source, ISigner signer, int version = 1)
    {
        var eventId = NewEventId();
        var timestampNanos = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() * 1_000_000;
        var conversationId = new ConversationId($"conv_{source.Value}");

        var content = new Dictionary<string, object?>
        {
            ["ActorID"] = source.Value,
            ["ChainGenesis"] = Hash.Zero().Value,
            ["Timestamp"] = DateTimeOffset.UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ"),
        };
        var contentJson = CanonicalForm.CanonicalContentJson(content);

        var canon = CanonicalForm.Build(
            version, "", Array.Empty<string>(),
            eventId.Value, "system.bootstrapped",
            source.Value, conversationId.Value,
            timestampNanos, contentJson);

        var hash = CanonicalForm.ComputeHash(canon);
        var sig = signer.Sign(Encoding.UTF8.GetBytes(canon));

        return new Event(version, eventId,
            new EventType("system.bootstrapped"), timestampNanos,
            source, content,
            NonEmpty<EventId>.Of(new List<EventId> { eventId }),
            conversationId, hash, Hash.Zero(), sig);
    }
}
