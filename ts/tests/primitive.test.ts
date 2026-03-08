import { describe, it, expect } from "vitest";
import { Registry, Lifecycle, isValidTransition, type Primitive, type Mutation, type Snapshot } from "../src/primitive.js";
import { InvalidTransitionError } from "../src/errors.js";
import {
  Activation, Cadence, EventType, Layer, PrimitiveId, SubscriptionPattern,
} from "../src/types.js";
import type { Event } from "../src/event.js";

function fakePrimitive(name: string, layer = 0): Primitive {
  return {
    id: () => new PrimitiveId(name),
    layer: () => new Layer(layer),
    cadence: () => new Cadence(1),
    subscriptions: () => [new SubscriptionPattern("*")],
    process: (_tick: number, _events: Event[], _snap: Snapshot): Mutation[] => [],
  };
}

describe("isValidTransition", () => {
  it("dormant -> activating is valid", () => {
    expect(isValidTransition(Lifecycle.Dormant, Lifecycle.Activating)).toBe(true);
  });

  it("activating -> active is valid", () => {
    expect(isValidTransition(Lifecycle.Activating, Lifecycle.Active)).toBe(true);
  });

  it("active -> processing is valid", () => {
    expect(isValidTransition(Lifecycle.Active, Lifecycle.Processing)).toBe(true);
  });

  it("processing -> emitting is valid", () => {
    expect(isValidTransition(Lifecycle.Processing, Lifecycle.Emitting)).toBe(true);
  });

  it("emitting -> active is valid", () => {
    expect(isValidTransition(Lifecycle.Emitting, Lifecycle.Active)).toBe(true);
  });

  it("processing -> active is valid", () => {
    expect(isValidTransition(Lifecycle.Processing, Lifecycle.Active)).toBe(true);
  });

  it("active -> suspending is valid", () => {
    expect(isValidTransition(Lifecycle.Active, Lifecycle.Suspending)).toBe(true);
  });

  it("suspending -> suspended is valid", () => {
    expect(isValidTransition(Lifecycle.Suspending, Lifecycle.Suspended)).toBe(true);
  });

  it("suspended -> activating is valid", () => {
    expect(isValidTransition(Lifecycle.Suspended, Lifecycle.Activating)).toBe(true);
  });

  it("active -> memorial is valid", () => {
    expect(isValidTransition(Lifecycle.Active, Lifecycle.Memorial)).toBe(true);
  });

  it("dormant -> active is invalid", () => {
    expect(isValidTransition(Lifecycle.Dormant, Lifecycle.Active)).toBe(false);
  });

  it("memorial -> anything is invalid", () => {
    expect(isValidTransition(Lifecycle.Memorial, Lifecycle.Dormant)).toBe(false);
    expect(isValidTransition(Lifecycle.Memorial, Lifecycle.Activating)).toBe(false);
    expect(isValidTransition(Lifecycle.Memorial, Lifecycle.Active)).toBe(false);
    expect(isValidTransition(Lifecycle.Memorial, Lifecycle.Processing)).toBe(false);
    expect(isValidTransition(Lifecycle.Memorial, Lifecycle.Memorial)).toBe(false);
  });

  it("dormant -> processing is invalid", () => {
    expect(isValidTransition(Lifecycle.Dormant, Lifecycle.Processing)).toBe(false);
  });

  it("emitting -> processing is invalid", () => {
    expect(isValidTransition(Lifecycle.Emitting, Lifecycle.Processing)).toBe(false);
  });

  it("unknown state returns false", () => {
    expect(isValidTransition("unknown", Lifecycle.Active)).toBe(false);
  });
});

describe("Registry", () => {
  it("register a primitive", () => {
    const reg = new Registry();
    const p = fakePrimitive("test.prim");
    reg.register(p);
    expect(reg.count).toBe(1);
    expect(reg.get(new PrimitiveId("test.prim"))).toBe(p);
  });

  it("reject duplicate registration", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("dup"));
    expect(() => reg.register(fakePrimitive("dup"))).toThrow("already registered");
  });

  it("get returns undefined for unregistered", () => {
    const reg = new Registry();
    expect(reg.get(new PrimitiveId("nope"))).toBeUndefined();
  });

  it("all primitives ordered by layer", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("high", 5));
    reg.register(fakePrimitive("low", 0));
    reg.register(fakePrimitive("mid", 2));
    const all = reg.all();
    expect(all.map((p) => p.id().value)).toEqual(["low", "mid", "high"]);
  });

  it("same layer ordered alphabetically", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("zebra", 0));
    reg.register(fakePrimitive("alpha", 0));
    reg.register(fakePrimitive("middle", 0));
    const all = reg.all();
    expect(all.map((p) => p.id().value)).toEqual(["alpha", "middle", "zebra"]);
  });

  it("getLifecycle returns dormant for unregistered", () => {
    const reg = new Registry();
    expect(reg.getLifecycle(new PrimitiveId("nope"))).toBe(Lifecycle.Dormant);
  });

  it("getLifecycle returns dormant for newly registered", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("p"));
    expect(reg.getLifecycle(new PrimitiveId("p"))).toBe(Lifecycle.Dormant);
  });

  it("setLifecycle valid transition", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("p"));
    reg.setLifecycle(new PrimitiveId("p"), Lifecycle.Activating);
    expect(reg.getLifecycle(new PrimitiveId("p"))).toBe(Lifecycle.Activating);
  });

  it("setLifecycle invalid transition throws InvalidTransitionError", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("p"));
    expect(() => reg.setLifecycle(new PrimitiveId("p"), Lifecycle.Active)).toThrow(InvalidTransitionError);
  });

  it("setLifecycle on unregistered throws", () => {
    const reg = new Registry();
    expect(() => reg.setLifecycle(new PrimitiveId("nope"), Lifecycle.Activating)).toThrow("not found");
  });

  it("activate shortcut transitions dormant to active", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("p"));
    reg.activate(new PrimitiveId("p"));
    expect(reg.getLifecycle(new PrimitiveId("p"))).toBe(Lifecycle.Active);
  });

  it("activate from non-dormant invalid state throws", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("p"));
    reg.activate(new PrimitiveId("p"));
    // now active; activating from active is invalid (no dormant->activating path)
    expect(() => reg.activate(new PrimitiveId("p"))).toThrow(InvalidTransitionError);
  });

  it("activate on unregistered throws", () => {
    const reg = new Registry();
    expect(() => reg.activate(new PrimitiveId("nope"))).toThrow("not found");
  });

  it("updateState sets key-value pair", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("p"));
    reg.updateState(new PrimitiveId("p"), "count", 42);
    const states = reg.allStates();
    expect(states.get("p")!.state["count"]).toBe(42);
  });

  it("updateState defensive copy", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("p"));
    const obj = { nested: true };
    reg.updateState(new PrimitiveId("p"), "data", obj);
    obj.nested = false; // mutate original
    const states = reg.allStates();
    expect(states.get("p")!.state["data"]).toEqual({ nested: true });
  });

  it("updateState on unregistered throws", () => {
    const reg = new Registry();
    expect(() => reg.updateState(new PrimitiveId("nope"), "k", "v")).toThrow("not found");
  });

  it("setActivation level", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("p"));
    reg.setActivation(new PrimitiveId("p"), new Activation(0.75));
    const states = reg.allStates();
    expect(states.get("p")!.activation.value).toBe(0.75);
  });

  it("setActivation on unregistered throws", () => {
    const reg = new Registry();
    expect(() => reg.setActivation(new PrimitiveId("nope"), new Activation(0.5))).toThrow("not found");
  });

  it("allStates returns state snapshot for each primitive", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("a", 1));
    reg.register(fakePrimitive("b", 0));
    const states = reg.allStates();
    expect(states.size).toBe(2);
    expect(states.get("a")!.layer.value).toBe(1);
    expect(states.get("b")!.layer.value).toBe(0);
  });

  it("lastTick default and set", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("p"));
    expect(reg.getLastTick(new PrimitiveId("p"))).toBe(0);
    reg.setLastTick(new PrimitiveId("p"), 5);
    expect(reg.getLastTick(new PrimitiveId("p"))).toBe(5);
  });

  it("getLastTick returns 0 for unregistered", () => {
    const reg = new Registry();
    expect(reg.getLastTick(new PrimitiveId("nope"))).toBe(0);
  });

  it("full lifecycle round-trip", () => {
    const reg = new Registry();
    reg.register(fakePrimitive("p"));
    const pid = new PrimitiveId("p");
    // dormant -> activating -> active -> processing -> emitting -> active -> memorial
    reg.setLifecycle(pid, Lifecycle.Activating);
    reg.setLifecycle(pid, Lifecycle.Active);
    reg.setLifecycle(pid, Lifecycle.Processing);
    reg.setLifecycle(pid, Lifecycle.Emitting);
    reg.setLifecycle(pid, Lifecycle.Active);
    reg.setLifecycle(pid, Lifecycle.Memorial);
    expect(reg.getLifecycle(pid)).toBe(Lifecycle.Memorial);
    // memorial is terminal
    expect(() => reg.setLifecycle(pid, Lifecycle.Dormant)).toThrow(InvalidTransitionError);
  });
});
