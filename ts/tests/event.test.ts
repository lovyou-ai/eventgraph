import { describe, it, expect } from "vitest";
import {
  canonicalContentJson, canonicalForm, computeHash,
  createBootstrap, createEvent, newEventId, NoopSigner,
} from "../src/event.js";
import { ActorId, ConversationId, EventType, Hash } from "../src/types.js";

describe("canonicalContentJson", () => {
  it("sorts keys", () => expect(canonicalContentJson({ b: 1, a: 2 })).toBe('{"a":2,"b":1}'));
  it("no whitespace", () => expect(canonicalContentJson({ key: "val" })).not.toContain(" "));
});

describe("canonicalForm", () => {
  it("pipe-separated with sorted causes", () => {
    const canon = canonicalForm(1, "0".repeat(64), ["c2", "c1"], "eid", "trust.updated", "alice", "conv_1", 123, '{"k":"v"}');
    const parts = canon.split("|");
    expect(parts[2]).toBe("c1,c2");
    expect(parts[0]).toBe("1");
  });
  it("empty causes", () => {
    const canon = canonicalForm(1, "", [], "eid", "system.bootstrapped", "s", "c", 0, "{}");
    expect(canon.split("|")[2]).toBe("");
  });
});

describe("computeHash", () => {
  it("deterministic", () => expect(computeHash("hello").value).toBe(computeHash("hello").value));
  it("different input", () => expect(computeHash("hello").value).not.toBe(computeHash("world").value));
  it("64 hex chars", () => expect(computeHash("test").value).toHaveLength(64));
});

describe("newEventId", () => {
  it("generates valid UUID v7", () => {
    const eid = newEventId();
    expect(eid.value).toHaveLength(36);
    expect(eid.value[14]).toBe("7");
  });
});

describe("createBootstrap", () => {
  it("valid bootstrap", () => {
    const ev = createBootstrap(new ActorId("alice"), new NoopSigner());
    expect(ev.version).toBe(1);
    expect(ev.type.value).toBe("system.bootstrapped");
    expect(ev.prevHash.isZero).toBe(true);
    expect(ev.causes.length).toBe(1);
    expect(ev.causes.get(0).value).toBe(ev.id.value);
  });
});

describe("createEvent", () => {
  it("valid event", () => {
    const boot = createBootstrap(new ActorId("alice"), new NoopSigner());
    const ev = createEvent(
      new EventType("trust.updated"), new ActorId("alice"),
      { score: 0.8 }, [boot.id],
      new ConversationId("conv_1"), boot.hash, new NoopSigner(),
    );
    expect(ev.type.value).toBe("trust.updated");
    expect(ev.prevHash.value).toBe(boot.hash.value);
  });
  it("content is defensive copy", () => {
    const boot = createBootstrap(new ActorId("alice"), new NoopSigner());
    const c1 = boot.content;
    const c2 = boot.content;
    expect(c1).not.toBe(c2);
  });
});

describe("canonical formatting", () => {
  it("integer floats format without decimal", () => {
    expect(canonicalContentJson({ count: 1.0, rate: 0.5 })).toBe('{"count":1,"rate":0.5}');
  });
  it("null values are omitted", () => {
    expect(canonicalContentJson({ Actor: "actor_1", Scope: null, Reason: undefined })).toBe('{"Actor":"actor_1"}');
  });
});

describe("conformance", () => {
  it("bootstrap canonical and hash matches Go reference", () => {
    const content = {
      ActorID: "actor_00000000000000000000000000000001",
      ChainGenesis: "0".repeat(64),
      Timestamp: "2023-11-14T22:13:20Z",
    };
    const contentJson = canonicalContentJson(content);
    const canon = canonicalForm(
      1, "", [], "019462a0-0000-7000-8000-000000000001",
      "system.bootstrapped", "actor_00000000000000000000000000000001",
      "conv_00000000000000000000000000000001", 1700000000000000000,
      contentJson,
    );
    expect(canon).toContain("1|||019462a0");
    expect(computeHash(canon).value).toBe("f7cae7ae11c1232a932c64f2302432c0e304dffce80f3935e688980dfbafeb75");
  });

  it("trust updated hash matches Go reference", () => {
    const content = { Actor: "actor_00000000000000000000000000000002", Cause: "019462a0-0000-7000-8000-000000000001", Current: 0.85, Domain: "code_review", Previous: 0.8 };
    const contentJson = canonicalContentJson(content);
    const canon = canonicalForm(
      1, "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
      ["019462a0-0000-7000-8000-000000000001"],
      "019462a0-0000-7000-8000-000000000002", "trust.updated",
      "actor_00000000000000000000000000000001", "conv_00000000000000000000000000000001",
      1700000001000000000, contentJson,
    );
    expect(computeHash(canon).value).toBe("b2fbcd2684868f0b0d07d2f5136b52f14b8e749da7b4b7bae2a22f67147152b7");
  });
});
