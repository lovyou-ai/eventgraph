namespace EventGraph;

public sealed record TickConfig(int MaxWavesPerTick = 10);

public sealed record TickResult(int Tick, int Waves, int Mutations, bool Quiesced, double DurationMs, List<string> Errors);

public sealed class TickEngine
{
    private readonly Lock _lock = new();
    private readonly PrimitiveRegistry _registry;
    private readonly InMemoryStore _store;
    private readonly TickConfig _config;
    private readonly Action<Event>? _publisher;
    private readonly ISigner _signer = new NoopSigner();
    private int _currentTick;

    public TickEngine(PrimitiveRegistry registry, InMemoryStore store, TickConfig? config = null, Action<Event>? publisher = null)
    {
        _registry = registry;
        _store = store;
        _config = config ?? new TickConfig();
        _publisher = publisher;
    }

    public TickResult Tick(List<Event>? pendingEvents = null)
    {
        lock (_lock)
        {
            var start = DateTime.UtcNow;
            _currentTick++;
            var tickNum = _currentTick;

            var waveEvents = new List<Event>(pendingEvents ?? []);
            var totalMutations = 0;
            var errors = new List<string>();
            var quiesced = false;
            var invokedThisTick = new HashSet<string>();
            var wavesRun = 0;
            var deferredMutations = new List<Mutation>();

            // Build initial snapshot
            var snapshot = new Snapshot(
                tickNum,
                _registry.AllStates(),
                new List<Event>(waveEvents),
                _store.Recent(100));

            for (var wave = 0; wave < _config.MaxWavesPerTick; wave++)
            {
                var (waveMutations, waveErrors) = RunWave(
                    tickNum, wave, waveEvents, snapshot, invokedThisTick);
                errors.AddRange(waveErrors);
                wavesRun = wave + 1;

                if (waveMutations.Count == 0)
                {
                    quiesced = true;
                    break;
                }

                // Eagerly apply AddEvent mutations; defer the rest
                var (newEvents, deferred, applyErrors) = ApplyEagerMutations(waveMutations);
                errors.AddRange(applyErrors);

                if (applyErrors.Count == 0)
                    deferredMutations.AddRange(deferred);

                totalMutations += newEvents.Count;

                if (newEvents.Count == 0)
                {
                    if (applyErrors.Count == 0)
                        quiesced = true;
                    break;
                }

                waveEvents = newEvents;

                // Refresh snapshot between waves
                snapshot = new Snapshot(
                    tickNum,
                    _registry.AllStates(),
                    new List<Event>(waveEvents),
                    _store.Recent(100));
            }

            // Apply deferred (non-AddEvent) mutations at end of tick
            var deferredErrors = ApplyDeferredMutations(deferredMutations);
            errors.AddRange(deferredErrors);
            totalMutations += deferredMutations.Count - deferredErrors.Count;

            var elapsed = (DateTime.UtcNow - start).TotalMilliseconds;
            return new TickResult(tickNum, wavesRun, totalMutations, quiesced, elapsed, errors);
        }
    }

    private (List<Mutation> Mutations, List<string> Errors) RunWave(
        int tickNum, int wave, List<Event> events, Snapshot snapshot, HashSet<string> invokedThisTick)
    {
        var eligible = EligiblePrimitives(tickNum, snapshot, invokedThisTick);

        // Group by layer
        var byLayer = new SortedDictionary<int, List<IPrimitive>>();
        foreach (var prim in eligible)
        {
            var l = prim.Layer.Value;
            if (!byLayer.ContainsKey(l)) byLayer[l] = new();
            byLayer[l].Add(prim);
        }

        var allMutations = new List<Mutation>();
        var waveErrors = new List<string>();

        foreach (var (layer, prims) in byLayer)
        {
            foreach (var prim in prims)
            {
                var pid = prim.Id;
                var matched = FilterEvents(events, prim.Subscriptions);

                // On subsequent waves, only invoke primitives with matching events
                if (matched.Count == 0 && invokedThisTick.Contains(pid.Value))
                    continue;

                try { _registry.SetLifecycle(pid, Lifecycle.Processing); }
                catch { continue; }

                string? processErr = null;
                try
                {
                    var mutations = prim.Process(tickNum, matched, snapshot);
                    allMutations.AddRange(mutations);
                }
                catch (Exception ex)
                {
                    processErr = ex.Message;
                    waveErrors.Add($"{pid.Value}: {ex.Message}");
                }

                // Lifecycle transitions
                try
                {
                    if (processErr == null)
                    {
                        if (allMutations.Count > 0)
                        {
                            _registry.SetLifecycle(pid, Lifecycle.Emitting);
                            _registry.SetLifecycle(pid, Lifecycle.Active);
                        }
                        else
                        {
                            _registry.SetLifecycle(pid, Lifecycle.Active);
                        }
                        invokedThisTick.Add(pid.Value);
                        _registry.SetLastTick(pid, tickNum);
                    }
                    else
                    {
                        // Restore to Active on error
                        _registry.SetLifecycle(pid, Lifecycle.Active);
                    }
                }
                catch (Exception ex) { waveErrors.Add($"{pid.Value} lifecycle: {ex.Message}"); }
            }
        }

        return (allMutations, waveErrors);
    }

    private List<IPrimitive> EligiblePrimitives(
        int tickNum, Snapshot snapshot, HashSet<string> invokedThisTick)
    {
        var eligible = new List<IPrimitive>();

        foreach (var prim in _registry.All())
        {
            var pid = prim.Id;

            // Must be Active
            if (_registry.GetLifecycle(pid) != Lifecycle.Active) continue;

            // Cadence gating — only on first invocation per tick
            if (!invokedThisTick.Contains(pid.Value))
            {
                var last = _registry.GetLastTick(pid);
                if (tickNum - last < prim.Cadence.Value) continue;
            }

            // Layer constraint
            if (!LayerStable(prim.Layer, snapshot)) continue;

            eligible.Add(prim);
        }

        return eligible;
    }

    /// <summary>
    /// Eagerly persist AddEvent mutations between waves.
    /// Non-AddEvent mutations are returned for deferred application.
    /// </summary>
    private (List<Event> NewEvents, List<Mutation> Deferred, List<string> Errors) ApplyEagerMutations(List<Mutation> mutations)
    {
        var newEvents = new List<Event>();
        var deferred = new List<Mutation>();
        var errors = new List<string>();

        foreach (var m in mutations)
        {
            if (m is AddEventMutation ae)
            {
                try
                {
                    var head = _store.Head();
                    var prevHash = head.IsSome ? head.Unwrap().Hash : Hash.Zero();
                    var ev = EventFactory.CreateEvent(ae.Type, ae.Source, ae.Content, ae.Causes, ae.ConversationId, prevHash, _signer);
                    _store.Append(ev);
                    _publisher?.Invoke(ev);
                    newEvents.Add(ev);
                }
                catch (Exception ex)
                {
                    errors.Add($"AddEvent: {ex.Message}");
                }
            }
            else
            {
                deferred.Add(m);
            }
        }

        return (newEvents, deferred, errors);
    }

    /// <summary>
    /// Apply deferred (non-AddEvent) mutations at end of tick.
    /// </summary>
    private List<string> ApplyDeferredMutations(List<Mutation> mutations)
    {
        var errors = new List<string>();
        foreach (var m in mutations)
        {
            try
            {
                switch (m)
                {
                    case AddEventMutation:
                        errors.Add("invariant violation: AddEvent in deferred batch");
                        break;
                    case UpdateStateMutation us:
                        _registry.UpdateState(us.PrimitiveId, us.Key, us.Value);
                        break;
                    case UpdateActivationMutation ua:
                        _registry.SetActivation(ua.PrimitiveId, ua.Level);
                        break;
                    case UpdateLifecycleMutation ul:
                        _registry.SetLifecycle(ul.PrimitiveId, ul.State);
                        break;
                }
            }
            catch (Exception ex)
            {
                errors.Add($"deferred mutation: {ex.Message}");
            }
        }
        return errors;
    }

    /// <summary>
    /// Returns true if all Layer N-1 primitives are Active and have been invoked
    /// at least once. Vacuously true when no Layer N-1 primitives are registered.
    /// </summary>
    private static bool LayerStable(Layer layer, Snapshot snapshot)
    {
        if (layer.Value == 0) return true;

        var targetLayer = layer.Value - 1;
        foreach (var ps in snapshot.Primitives.Values)
        {
            if (ps.Layer.Value == targetLayer)
            {
                if (ps.LifecycleState != Lifecycle.Active) return false;
                if (ps.LastTick == 0) return false; // never invoked
            }
        }
        return true;
    }

    private static List<Event> FilterEvents(List<Event> events, List<SubscriptionPattern> patterns)
    {
        var result = new List<Event>();
        foreach (var ev in events)
        {
            foreach (var pat in patterns)
            {
                if (pat.Matches(ev.Type))
                {
                    result.Add(ev);
                    break;
                }
            }
        }
        return result;
    }
}
