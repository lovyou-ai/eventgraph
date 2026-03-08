import { describe, it, expect } from "vitest";
import { createBootstrap, NoopSigner, Event } from "../src/event.js";
import { Registry, Lifecycle, type Mutation, type Snapshot, type Primitive } from "../src/primitive.js";
import { InMemoryStore } from "../src/store.js";
import { TickEngine } from "../src/tick.js";
import {
  Activation, ActorId, Cadence, ConversationId, EventType, Layer, PrimitiveId, SubscriptionPattern,
} from "../src/types.js";

class CountingPrimitive implements Primitive {
  received = 0;
  private _id: PrimitiveId;
  constructor(name: string, private _layer = 0) { this._id = new PrimitiveId(name); }
  id() { return this._id; }
  layer() { return new Layer(this._layer); }
  cadence() { return new Cadence(1); }
  subscriptions() { return [new SubscriptionPattern("*")]; }
  process(_tick: number, events: Event[], _snap: Snapshot): Mutation[] {
    this.received += events.length;
    return [{ kind: "updateState", primitiveId: this._id, key: "count", value: this.received }];
  }
}

class EmittingPrimitive implements Primitive {
  emissions = 0;
  private _id: PrimitiveId;
  constructor(name: string, private max: number) { this._id = new PrimitiveId(name); }
  id() { return this._id; }
  layer() { return new Layer(0); }
  cadence() { return new Cadence(1); }
  subscriptions() { return [new SubscriptionPattern("*")]; }
  process(_tick: number, events: Event[]): Mutation[] {
    if (events.length === 0 || this.emissions >= this.max) return [];
    this.emissions++;
    return [{ kind: "addEvent", type: new EventType("test.emitted"), source: new ActorId("emitter"), content: { w: this.emissions }, causes: [events[0].id], conversationId: new ConversationId("conv_t") }];
  }
}

function setup(prims: Primitive[] = [], config?: { maxWavesPerTick: number }) {
  const reg = new Registry();
  const store = new InMemoryStore();
  const b = createBootstrap(new ActorId("system"), new NoopSigner());
  store.append(b);
  for (const p of prims) { reg.register(p); reg.activate(p.id()); }
  const engine = new TickEngine(reg, store, config);
  return { reg, store, engine, boot: b };
}

describe("TickEngine", () => {
  it("basic tick", () => {
    const c = new CountingPrimitive("counter");
    const { engine, boot } = setup([c]);
    const r = engine.tick([boot]);
    expect(r.tick).toBe(1);
    expect(c.received).toBe(1);
  });

  it("quiescence", () => {
    const { engine, boot } = setup([new CountingPrimitive("c")]);
    expect(engine.tick([boot]).quiesced).toBe(true);
  });

  it("ripple waves", () => {
    const e = new EmittingPrimitive("e", 3);
    const c = new CountingPrimitive("c");
    const { engine, boot } = setup([e, c]);
    const r = engine.tick([boot]);
    expect(r.waves).toBeGreaterThan(1);
    expect(c.received).toBeGreaterThan(1);
  });

  it("max waves limit", () => {
    const inf: Primitive = {
      id: () => new PrimitiveId("inf"),
      layer: () => new Layer(0),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("*")],
      process: (_t, events) => events.length ? [{ kind: "addEvent", type: new EventType("test.loop"), source: new ActorId("inf"), content: {}, causes: [events[0].id], conversationId: new ConversationId("c") }] : [],
    };
    const { engine, boot } = setup([inf], { maxWavesPerTick: 3 });
    const r = engine.tick([boot]);
    expect(r.waves).toBe(3);
    expect(r.quiesced).toBe(false);
  });

  it("inactive skipped", () => {
    const c = new CountingPrimitive("c");
    const reg = new Registry();
    const store = new InMemoryStore();
    const b = createBootstrap(new ActorId("sys"), new NoopSigner());
    store.append(b);
    reg.register(c); // don't activate
    const engine = new TickEngine(reg, store);
    engine.tick([b]);
    expect(c.received).toBe(0);
  });

  it("tick counter increments", () => {
    const { engine, boot } = setup();
    expect(engine.tick([boot]).tick).toBe(1);
    expect(engine.tick().tick).toBe(2);
  });

  it("layer ordering", () => {
    const order: string[] = [];
    const mkTracker = (name: string, layer: number): Primitive => ({
      id: () => new PrimitiveId(name),
      layer: () => new Layer(layer),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("*")],
      process: () => { order.push(name); return []; },
    });
    const { engine, boot } = setup([mkTracker("high", 5), mkTracker("low", 0), mkTracker("mid", 2)]);
    engine.tick([boot]);
    expect(order).toEqual(["low", "mid", "high"]);
  });

  // --- Layer constraint tests ---

  it("layer constraint blocks when lower layer not invoked", () => {
    const order: string[] = [];
    const mkTracker = (name: string, layer: number): Primitive => ({
      id: () => new PrimitiveId(name),
      layer: () => new Layer(layer),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("*")],
      process: () => { order.push(name); return []; },
    });

    const { engine, boot } = setup([mkTracker("l0", 0), mkTracker("l1", 1)]);

    // Tick 1: Layer 0 runs, Layer 1 blocked
    engine.tick([boot]);
    expect(order).toEqual(["l0"]);

    // Tick 2: Layer 0 stable, Layer 1 now eligible
    order.length = 0;
    engine.tick([boot]);
    expect(order).toContain("l0");
    expect(order).toContain("l1");
    expect(order.indexOf("l0")).toBeLessThan(order.indexOf("l1"));
  });

  it("layer constraint vacuously true for sparse layers", () => {
    let invoked = 0;
    const l1Only: Primitive = {
      id: () => new PrimitiveId("l1_only"),
      layer: () => new Layer(1),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("*")],
      process: () => { invoked++; return []; },
    };

    const { engine, boot } = setup([l1Only]);
    engine.tick([boot]);
    expect(invoked).toBe(1);
  });

  it("layer constraint blocked by dormant lower layer", () => {
    let l1Invoked = 0;

    const l0Dormant: Primitive = {
      id: () => new PrimitiveId("dormant_l0"),
      layer: () => new Layer(0),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("*")],
      process: () => [],
    };

    const l1Active: Primitive = {
      id: () => new PrimitiveId("active_l1"),
      layer: () => new Layer(1),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("*")],
      process: () => { l1Invoked++; return []; },
    };

    const reg = new Registry();
    const store = new InMemoryStore();
    const b = createBootstrap(new ActorId("sys"), new NoopSigner());
    store.append(b);

    reg.register(l0Dormant); // don't activate — stays Dormant
    reg.register(l1Active);
    reg.activate(l1Active.id());

    const engine = new TickEngine(reg, store);
    engine.tick([b]);

    expect(l1Invoked).toBe(0);
  });

  // --- Deferred mutations tests ---

  it("deferred mutations not visible between waves", () => {
    const stateValuesSeen: (unknown | undefined)[] = [];
    let callCount = 0;

    const stateful: Primitive = {
      id: () => new PrimitiveId("stateful"),
      layer: () => new Layer(0),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("*")],
      process: (_tick, events, snap) => {
        callCount++;
        const ps = snap.primitives.get("stateful");
        stateValuesSeen.push(ps?.state?.["count"]);

        if (events.length === 0) return [];

        const result: Mutation[] = [
          { kind: "updateState", primitiveId: new PrimitiveId("stateful"), key: "count", value: callCount },
        ];

        if (callCount === 1) {
          result.push({
            kind: "addEvent",
            type: new EventType("test.ripple"),
            source: new ActorId("stateful"),
            content: {},
            causes: [events[0].id],
            conversationId: new ConversationId("conv_t"),
          });
        }

        return result;
      },
    };

    const { engine, boot } = setup([stateful]);
    const r = engine.tick([boot]);

    expect(r.waves).toBeGreaterThanOrEqual(2);
    // On wave 0, state should be undefined
    expect(stateValuesSeen[0]).toBeUndefined();
    // On wave 1, UpdateState from wave 0 was deferred, so still undefined
    if (stateValuesSeen.length > 1) {
      expect(stateValuesSeen[1]).toBeUndefined();
    }
  });

  it("mixed mutations: AddEvent + UpdateState", () => {
    let emitted = false;
    const pid = new PrimitiveId("mixed");

    const mixed: Primitive = {
      id: () => pid,
      layer: () => new Layer(0),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("*")],
      process: (_tick, events) => {
        if (events.length === 0) return [];

        const result: Mutation[] = [
          { kind: "updateState", primitiveId: pid, key: "processed", value: true },
          { kind: "updateActivation", primitiveId: pid, level: new Activation(0.9) },
        ];

        if (!emitted) {
          emitted = true;
          result.push({
            kind: "addEvent",
            type: new EventType("test.mixed"),
            source: new ActorId("mixed"),
            content: {},
            causes: [events[0].id],
            conversationId: new ConversationId("conv_t"),
          });
        }

        return result;
      },
    };

    const { engine, boot } = setup([mixed]);
    const r = engine.tick([boot]);

    // AddEvent (eager) + UpdateState + UpdateActivation (deferred) = 3+
    expect(r.mutations).toBeGreaterThanOrEqual(3);
  });

  // --- UpdateLifecycle mutation ---

  it("UpdateLifecycle mutation", () => {
    let invoked = 0;
    const pid = new PrimitiveId("updater");

    const updater: Primitive = {
      id: () => pid,
      layer: () => new Layer(0),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("*")],
      process: () => {
        invoked++;
        return [{ kind: "updateLifecycle", primitiveId: pid, state: Lifecycle.Suspending }];
      },
    };

    const { engine, boot } = setup([updater]);

    const r1 = engine.tick([boot]);
    expect(r1.mutations).toBeGreaterThanOrEqual(1);

    // Tick 2: primitive is now Suspending, should NOT be eligible
    const r2 = engine.tick([boot]);
    expect(r2.mutations).toBe(0);
    expect(invoked).toBe(1);
  });

  // --- Subscription filtering ---

  it("subscription filtering: no match", () => {
    const receivedCounts: number[] = [];

    const watcher: Primitive = {
      id: () => new PrimitiveId("watcher"),
      layer: () => new Layer(0),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("trust.*")],
      process: (_t, events) => {
        receivedCounts.push(events.length);
        return [];
      },
    };

    const { engine, boot } = setup([watcher]);
    engine.tick([boot]);
    expect(receivedCounts[0]).toBe(0);
  });

  it("no subscriptions gets no events", () => {
    const receivedCounts: number[] = [];

    const nosubs: Primitive = {
      id: () => new PrimitiveId("nosubs"),
      layer: () => new Layer(0),
      cadence: () => new Cadence(1),
      subscriptions: () => [],
      process: (_t, events) => {
        receivedCounts.push(events.length);
        return [];
      },
    };

    const { engine, boot } = setup([nosubs]);
    engine.tick([boot]);
    expect(receivedCounts[0]).toBe(0);
  });

  // --- Wave limit ---

  it("wave limit with custom config", () => {
    const inf: Primitive = {
      id: () => new PrimitiveId("always"),
      layer: () => new Layer(0),
      cadence: () => new Cadence(1),
      subscriptions: () => [new SubscriptionPattern("*")],
      process: (_t, events) =>
        events.length
          ? [{ kind: "addEvent", type: new EventType("test.always"), source: new ActorId("a"), content: {}, causes: [events[0].id], conversationId: new ConversationId("c") }]
          : [],
    };

    const { engine, boot } = setup([inf], { maxWavesPerTick: 5 });
    const r = engine.tick([boot]);
    expect(r.waves).toBe(5);
    expect(r.quiesced).toBe(false);
  });
});
