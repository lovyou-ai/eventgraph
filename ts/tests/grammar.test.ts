import { describe, it, expect, beforeEach } from "vitest";
import { Grammar } from "../src/grammar.js";
import { InMemoryStore } from "../src/store.js";
import { createBootstrap, NoopSigner } from "../src/event.js";
import { ActorId, ConversationId } from "../src/types.js";

describe("Grammar", () => {
  let store: InMemoryStore;
  let grammar: Grammar;
  const signer = new NoopSigner();
  const alice = new ActorId("alice");
  const bob = new ActorId("bob");
  const convId = new ConversationId("conv_test");

  beforeEach(() => {
    store = new InMemoryStore();
    grammar = new Grammar(store);
    // Bootstrap so there is a chain head
    const boot = createBootstrap(alice, signer);
    store.append(boot);
  });

  // ── emit ──────────────────────────────────────────────────────────

  describe("emit", () => {
    it("creates a grammar.emitted event with the given causes", () => {
      const head = store.head().unwrap();
      const ev = grammar.emit(alice, "hello world", convId, [head.id], signer);
      expect(ev.type.value).toBe("grammar.emitted");
      expect(ev.content.Body).toBe("hello world");
      expect(ev.source.value).toBe("alice");
      expect(store.count()).toBe(2);
    });

    it("throws if causes is empty", () => {
      expect(() => grammar.emit(alice, "hello", convId, [], signer)).toThrow(
        "emit requires at least one cause",
      );
    });

    it("hash-chains to the previous head", () => {
      const headBefore = store.head().unwrap();
      const ev = grammar.emit(alice, "body", convId, [headBefore.id], signer);
      expect(ev.prevHash.value).toBe(headBefore.hash.value);
    });
  });

  // ── respond ───────────────────────────────────────────────────────

  describe("respond", () => {
    it("creates a grammar.responded event with parent in content and causes", () => {
      const head = store.head().unwrap();
      const ev = grammar.respond(alice, "reply", head.id, convId, signer);
      expect(ev.type.value).toBe("grammar.responded");
      expect(ev.content.Body).toBe("reply");
      expect(ev.content.Parent).toBe(head.id.value);
      const causeValues = [...ev.causes].map((c) => c.value);
      expect(causeValues).toContain(head.id.value);
    });
  });

  // ── derive ────────────────────────────────────────────────────────

  describe("derive", () => {
    it("creates a grammar.derived event with Source in content", () => {
      const head = store.head().unwrap();
      const ev = grammar.derive(alice, "derived content", head.id, convId, signer);
      expect(ev.type.value).toBe("grammar.derived");
      expect(ev.content.Body).toBe("derived content");
      expect(ev.content.Source).toBe(head.id.value);
    });
  });

  // ── extend ────────────────────────────────────────────────────────

  describe("extend", () => {
    it("creates a grammar.extended event with Previous in content", () => {
      const head = store.head().unwrap();
      const ev = grammar.extend(alice, "continued", head.id, convId, signer);
      expect(ev.type.value).toBe("grammar.extended");
      expect(ev.content.Body).toBe("continued");
      expect(ev.content.Previous).toBe(head.id.value);
    });
  });

  // ── retract ───────────────────────────────────────────────────────

  describe("retract", () => {
    it("creates a grammar.retracted event when source matches target author", () => {
      const head = store.head().unwrap();
      const emitted = grammar.emit(alice, "to retract", convId, [head.id], signer);
      const ev = grammar.retract(alice, emitted.id, "changed my mind", convId, signer);
      expect(ev.type.value).toBe("grammar.retracted");
      expect(ev.content.Target).toBe(emitted.id.value);
      expect(ev.content.Reason).toBe("changed my mind");
    });

    it("throws if source is not the author of the target event", () => {
      const head = store.head().unwrap();
      const emitted = grammar.emit(alice, "alice's event", convId, [head.id], signer);
      expect(() =>
        grammar.retract(bob, emitted.id, "nope", convId, signer),
      ).toThrow("retract: actor bob cannot retract event");
    });
  });

  // ── annotate ──────────────────────────────────────────────────────

  describe("annotate", () => {
    it("creates a grammar.annotated event with key/value metadata", () => {
      const head = store.head().unwrap();
      const ev = grammar.annotate(alice, head.id, "priority", "high", convId, signer);
      expect(ev.type.value).toBe("grammar.annotated");
      expect(ev.content.Target).toBe(head.id.value);
      expect(ev.content.Key).toBe("priority");
      expect(ev.content.Value).toBe("high");
    });
  });

  // ── merge ─────────────────────────────────────────────────────────

  describe("merge", () => {
    it("creates a grammar.merged event joining multiple sources", () => {
      const head = store.head().unwrap();
      const ev1 = grammar.emit(alice, "branch A", convId, [head.id], signer);
      const ev2 = grammar.emit(alice, "branch B", convId, [ev1.id], signer);
      const merged = grammar.merge(alice, "merged result", [ev1.id, ev2.id], convId, signer);
      expect(merged.type.value).toBe("grammar.merged");
      expect(merged.content.Body).toBe("merged result");
      expect(merged.content.Sources).toEqual([ev1.id.value, ev2.id.value]);
      const causeValues = [...merged.causes].map((c) => c.value);
      expect(causeValues).toContain(ev1.id.value);
      expect(causeValues).toContain(ev2.id.value);
    });

    it("throws if fewer than two sources", () => {
      const head = store.head().unwrap();
      expect(() =>
        grammar.merge(alice, "only one", [head.id], convId, signer),
      ).toThrow("merge requires at least two sources");
    });

    it("throws if zero sources", () => {
      expect(() =>
        grammar.merge(alice, "none", [], convId, signer),
      ).toThrow("merge requires at least two sources");
    });
  });

  // ── chain integrity ───────────────────────────────────────────────

  describe("chain integrity", () => {
    it("maintains valid hash chain across multiple grammar operations", () => {
      const head = store.head().unwrap();
      const ev1 = grammar.emit(alice, "first", convId, [head.id], signer);
      const ev2 = grammar.respond(alice, "second", ev1.id, convId, signer);
      grammar.annotate(alice, ev2.id, "tag", "important", convId, signer);

      const verification = store.verifyChain();
      expect(verification.valid).toBe(true);
      expect(verification.length).toBe(4); // bootstrap + 3 grammar ops
    });
  });
});
