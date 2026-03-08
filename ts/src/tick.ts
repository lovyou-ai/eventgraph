import { Event, createEvent, NoopSigner } from "./event.js";
import { Lifecycle, type Mutation, type Primitive, type PrimitiveState, Registry, type Snapshot } from "./primitive.js";
import { InMemoryStore } from "./store.js";
import { Hash, PrimitiveId, type SubscriptionPattern, type Layer } from "./types.js";

export interface TickConfig {
  maxWavesPerTick: number;
}

export interface TickResult {
  tick: number;
  waves: number;
  mutations: number;
  quiesced: boolean;
  durationMs: number;
  errors: string[];
}

export class TickEngine {
  private readonly registry: Registry;
  private readonly store: InMemoryStore;
  private readonly config: TickConfig;
  private readonly publisher?: (ev: Event) => void;
  private readonly signer = new NoopSigner();
  private currentTick = 0;

  constructor(
    registry: Registry,
    store: InMemoryStore,
    config?: Partial<TickConfig>,
    publisher?: (ev: Event) => void,
  ) {
    this.registry = registry;
    this.store = store;
    this.config = { maxWavesPerTick: 10, ...config };
    this.publisher = publisher;
  }

  tick(pendingEvents?: Event[]): TickResult {
    const start = performance.now();
    this.currentTick++;
    const tickNum = this.currentTick;

    let waveEvents = [...(pendingEvents ?? [])];
    let totalMutations = 0;
    const errors: string[] = [];
    let quiesced = false;
    const invokedThisTick = new Set<string>();
    let wavesRun = 0;
    const deferredMutations: Mutation[] = [];

    // Build initial snapshot
    let snapshot: Snapshot = {
      tick: tickNum,
      primitives: this.registry.allStates(),
      pendingEvents: [...waveEvents],
      recentEvents: this.store.recent(100),
    };

    for (let wave = 0; wave < this.config.maxWavesPerTick; wave++) {
      const [waveMutations, waveErrors] = this.runWave(
        tickNum, wave, waveEvents, snapshot, invokedThisTick,
      );
      errors.push(...waveErrors);
      wavesRun = wave + 1;

      if (waveMutations.length === 0) {
        quiesced = true;
        break;
      }

      // Eagerly apply AddEvent mutations; defer the rest
      const [newEvents, deferred, applyErrors] = this.applyEagerMutations(waveMutations);
      errors.push(...applyErrors);

      if (applyErrors.length === 0) {
        deferredMutations.push(...deferred);
      }

      totalMutations += newEvents.length;

      if (newEvents.length === 0) {
        if (applyErrors.length === 0) {
          quiesced = true;
        }
        break;
      }

      waveEvents = newEvents;

      // Refresh snapshot between waves
      snapshot = {
        tick: tickNum,
        primitives: this.registry.allStates(),
        pendingEvents: [...waveEvents],
        recentEvents: this.store.recent(100),
      };
    }

    // Apply deferred (non-AddEvent) mutations at end of tick
    const deferredErrors = this.applyDeferredMutations(deferredMutations);
    errors.push(...deferredErrors);
    totalMutations += deferredMutations.length - deferredErrors.length;

    return {
      tick: tickNum,
      waves: wavesRun,
      mutations: totalMutations,
      quiesced,
      durationMs: performance.now() - start,
      errors,
    };
  }

  private runWave(
    tickNum: number,
    wave: number,
    events: Event[],
    snapshot: Snapshot,
    invokedThisTick: Set<string>,
  ): [Mutation[], string[]] {
    const eligible = this.eligiblePrimitives(tickNum, snapshot, invokedThisTick);

    // Group by layer
    const byLayer = new Map<number, Primitive[]>();
    for (const prim of eligible) {
      const l = prim.layer().value;
      if (!byLayer.has(l)) byLayer.set(l, []);
      byLayer.get(l)!.push(prim);
    }

    const layers = [...byLayer.keys()].sort((a, b) => a - b);

    const allMutations: Mutation[] = [];
    const waveErrors: string[] = [];

    for (const layer of layers) {
      const prims = byLayer.get(layer)!;

      for (const prim of prims) {
        const pid = prim.id();
        const matched = filterEvents(events, prim.subscriptions());

        // On subsequent waves, only invoke primitives with matching events
        if (matched.length === 0 && invokedThisTick.has(pid.value)) {
          continue;
        }

        try { this.registry.setLifecycle(pid, Lifecycle.Processing); }
        catch { continue; }

        let processErr: string | null = null;
        try {
          const mutations = prim.process(tickNum, matched, snapshot);
          allMutations.push(...mutations);
        } catch (e) {
          processErr = `${e}`;
          waveErrors.push(`${pid.value}: ${e}`);
        }

        // Lifecycle transitions
        try {
          if (processErr === null) {
            if (allMutations.length > 0) {
              this.registry.setLifecycle(pid, Lifecycle.Emitting);
              this.registry.setLifecycle(pid, Lifecycle.Active);
            } else {
              this.registry.setLifecycle(pid, Lifecycle.Active);
            }
            invokedThisTick.add(pid.value);
            this.registry.setLastTick(pid, tickNum);
          } else {
            // Restore to Active on error
            this.registry.setLifecycle(pid, Lifecycle.Active);
          }
        } catch (e) { waveErrors.push(`${pid.value} lifecycle: ${e}`); }
      }
    }

    return [allMutations, waveErrors];
  }

  private eligiblePrimitives(
    tickNum: number,
    snapshot: Snapshot,
    invokedThisTick: Set<string>,
  ): Primitive[] {
    const eligible: Primitive[] = [];

    for (const prim of this.registry.all()) {
      const pid = prim.id();

      // Must be Active
      if (this.registry.getLifecycle(pid) !== Lifecycle.Active) continue;

      // Cadence gating — only on first invocation per tick
      if (!invokedThisTick.has(pid.value)) {
        const last = this.registry.getLastTick(pid);
        if (tickNum - last < prim.cadence().value) continue;
      }

      // Layer constraint
      if (!layerStable(prim.layer(), snapshot)) continue;

      eligible.push(prim);
    }

    return eligible;
  }

  /**
   * Eagerly persist AddEvent mutations between waves.
   * Non-AddEvent mutations are returned for deferred application.
   */
  private applyEagerMutations(mutations: Mutation[]): [Event[], Mutation[], string[]] {
    const newEvents: Event[] = [];
    const deferred: Mutation[] = [];
    const errors: string[] = [];

    for (const m of mutations) {
      if (m.kind === "addEvent") {
        try {
          const head = this.store.head();
          const prevHash = head.isSome ? head.unwrap().hash : Hash.zero();
          const ev = createEvent(m.type, m.source, m.content, m.causes, m.conversationId, prevHash, this.signer);
          this.store.append(ev);
          this.publisher?.(ev);
          newEvents.push(ev);
        } catch (e) {
          errors.push(`AddEvent: ${e}`);
        }
      } else {
        deferred.push(m);
      }
    }

    return [newEvents, deferred, errors];
  }

  /**
   * Apply deferred (non-AddEvent) mutations at end of tick.
   */
  private applyDeferredMutations(mutations: Mutation[]): string[] {
    const errors: string[] = [];
    for (const m of mutations) {
      try {
        switch (m.kind) {
          case "addEvent":
            errors.push("invariant violation: AddEvent in deferred batch");
            break;
          case "updateState":
            this.registry.updateState(m.primitiveId, m.key, m.value);
            break;
          case "updateActivation":
            this.registry.setActivation(m.primitiveId, m.level);
            break;
          case "updateLifecycle":
            this.registry.setLifecycle(m.primitiveId, m.state);
            break;
        }
      } catch (e) {
        errors.push(`deferred mutation: ${e}`);
      }
    }
    return errors;
  }
}

/**
 * Returns true if all Layer N-1 primitives are Active and have been invoked
 * at least once. Vacuously true when no Layer N-1 primitives are registered.
 */
function layerStable(layer: Layer, snapshot: Snapshot): boolean {
  if (layer.value === 0) return true;

  const targetLayer = layer.value - 1;
  for (const ps of snapshot.primitives.values()) {
    if (ps.layer.value === targetLayer) {
      if (ps.lifecycle !== Lifecycle.Active) return false;
      if (ps.lastTick === 0) return false; // never invoked
    }
  }
  return true;
}

function filterEvents(events: Event[], patterns: SubscriptionPattern[]): Event[] {
  return events.filter((ev) => patterns.some((p) => p.matches(ev.type)));
}
