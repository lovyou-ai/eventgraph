import { describe, it, expect } from "vitest";
import { EventBus } from "../src/bus.js";
import { createBootstrap, createEvent, NoopSigner } from "../src/event.js";
import { InMemoryStore } from "../src/store.js";
import { ActorId, ConversationId, EventType, SubscriptionPattern } from "../src/types.js";

function setup() {
  const store = new InMemoryStore();
  const bus = new EventBus(store);
  const signer = new NoopSigner();
  const boot = createBootstrap(new ActorId("alice"), signer);
  store.append(boot);
  return { store, bus, boot, signer };
}

function makeEvent(prev: ReturnType<typeof createBootstrap>) {
  return createEvent(
    new EventType("trust.updated"), new ActorId("alice"), { score: 0.5 },
    [prev.id], new ConversationId("conv_1"), prev.hash, new NoopSigner(),
  );
}

describe("EventBus", () => {
  it("subscribe and receive matching events", () => {
    const { bus, boot, store } = setup();
    const received: string[] = [];
    bus.subscribe(new SubscriptionPattern("trust.*"), (ev) => received.push(ev.id.value));
    const ev = makeEvent(boot);
    store.append(ev);
    bus.publish(ev);
    expect(received).toHaveLength(1);
    expect(received[0]).toBe(ev.id.value);
  });

  it("non-matching pattern does not receive", () => {
    const { bus, boot, store } = setup();
    const received: string[] = [];
    bus.subscribe(new SubscriptionPattern("system.*"), (ev) => received.push(ev.id.value));
    const ev = makeEvent(boot);
    store.append(ev);
    bus.publish(ev);
    expect(received).toHaveLength(0);
  });

  it("wildcard pattern matches all events", () => {
    const { bus, boot } = setup();
    const received: string[] = [];
    bus.subscribe(new SubscriptionPattern("*"), (ev) => received.push(ev.type.value));
    bus.publish(boot);
    expect(received).toEqual(["system.bootstrapped"]);
  });

  it("prefix pattern matches subtypes", () => {
    const { bus, boot, store } = setup();
    const received: string[] = [];
    bus.subscribe(new SubscriptionPattern("trust.*"), (ev) => received.push(ev.type.value));
    // should match
    const ev1 = makeEvent(boot);
    store.append(ev1);
    bus.publish(ev1);
    // should not match
    bus.publish(boot);
    expect(received).toEqual(["trust.updated"]);
  });

  it("exact pattern only matches exact type", () => {
    const { bus, boot, store } = setup();
    const received: string[] = [];
    bus.subscribe(new SubscriptionPattern("trust.updated"), (ev) => received.push(ev.type.value));
    const ev = makeEvent(boot);
    store.append(ev);
    bus.publish(ev);
    bus.publish(boot);
    expect(received).toEqual(["trust.updated"]);
  });

  it("unsubscribe stops delivery", () => {
    const { bus, boot } = setup();
    const received: string[] = [];
    const subId = bus.subscribe(new SubscriptionPattern("*"), (ev) => received.push(ev.id.value));
    bus.publish(boot);
    expect(received).toHaveLength(1);
    bus.unsubscribe(subId);
    bus.publish(boot);
    expect(received).toHaveLength(1);
  });

  it("multiple subscribers receive independently", () => {
    const { bus, boot } = setup();
    const a: string[] = [];
    const b: string[] = [];
    bus.subscribe(new SubscriptionPattern("*"), (ev) => a.push(ev.id.value));
    bus.subscribe(new SubscriptionPattern("*"), (ev) => b.push(ev.id.value));
    bus.publish(boot);
    expect(a).toHaveLength(1);
    expect(b).toHaveLength(1);
  });

  it("publish after close is no-op", () => {
    const { bus, boot } = setup();
    const received: string[] = [];
    bus.subscribe(new SubscriptionPattern("*"), (ev) => received.push(ev.id.value));
    bus.close();
    bus.publish(boot);
    expect(received).toHaveLength(0);
  });

  it("subscribe after close returns -1", () => {
    const { bus } = setup();
    bus.close();
    const id = bus.subscribe(new SubscriptionPattern("*"), () => {});
    expect(id).toBe(-1);
  });

  it("handler errors do not crash other handlers", () => {
    const { bus, boot } = setup();
    const received: string[] = [];
    bus.subscribe(new SubscriptionPattern("*"), () => { throw new Error("boom"); });
    bus.subscribe(new SubscriptionPattern("*"), (ev) => received.push(ev.id.value));
    bus.publish(boot);
    expect(received).toHaveLength(1);
  });

  it("exposes store", () => {
    const { bus, store } = setup();
    expect(bus.store).toBe(store);
  });
});
