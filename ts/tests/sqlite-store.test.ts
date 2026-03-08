import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { createBootstrap, createEvent, NoopSigner } from "../src/event.js";
import { ActorId, ConversationId, EventId, EventType } from "../src/types.js";
import { ChainIntegrityError, EventNotFoundError } from "../src/errors.js";

// Skip entire suite if better-sqlite3 is not available
let SQLiteStore: typeof import("../src/sqlite-store.js").SQLiteStore;
try {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  require("better-sqlite3");
  SQLiteStore = (await import("../src/sqlite-store.js")).SQLiteStore;
} catch {
  describe.skip("SQLiteStore (better-sqlite3 not available)", () => {
    it("skipped", () => {});
  });
}

const boot = () => createBootstrap(new ActorId("alice"), new NoopSigner());
const next = (prev: ReturnType<typeof boot>) =>
  createEvent(
    new EventType("trust.updated"),
    new ActorId("alice"),
    {},
    [prev.id],
    new ConversationId("conv_1"),
    prev.hash,
    new NoopSigner(),
  );

const nextWithOpts = (
  prev: ReturnType<typeof boot>,
  type: string,
  source: string,
  convId: string,
) =>
  createEvent(
    new EventType(type),
    new ActorId(source),
    {},
    [prev.id],
    new ConversationId(convId),
    prev.hash,
    new NoopSigner(),
  );

if (SQLiteStore!) {
  describe("SQLiteStore", () => {
    let store: InstanceType<typeof SQLiteStore>;

    beforeEach(() => {
      store = new SQLiteStore(":memory:");
    });

    afterEach(() => {
      store.close();
    });

    // ── Basic CRUD ────────────────────────────────────────────────────

    it("append and get", () => {
      const b = boot();
      store.append(b);
      expect(store.get(b.id).id.value).toBe(b.id.value);
    });

    it("head empty", () => {
      expect(store.head().isNone).toBe(true);
    });

    it("head after append", () => {
      const b = boot();
      store.append(b);
      expect(store.head().unwrap().id.value).toBe(b.id.value);
    });

    it("count", () => {
      expect(store.count()).toBe(0);
      store.append(boot());
      expect(store.count()).toBe(1);
    });

    it("get nonexistent", () => {
      expect(() =>
        store.get(new EventId("019462a0-0000-7000-8000-000000000099")),
      ).toThrow(EventNotFoundError);
    });

    // ── Chain integrity ───────────────────────────────────────────────

    it("chain of events", () => {
      const b = boot();
      store.append(b);
      const e1 = next(b);
      store.append(e1);
      const e2 = next(e1);
      store.append(e2);
      expect(store.count()).toBe(3);
    });

    it("rejects broken chain", () => {
      const b = boot();
      store.append(b);
      const bad = createEvent(
        new EventType("trust.updated"),
        new ActorId("alice"),
        {},
        [b.id],
        new ConversationId("c"),
        b.prevHash,
        new NoopSigner(),
      );
      expect(() => store.append(bad)).toThrow(ChainIntegrityError);
    });

    it("verify chain valid", () => {
      const b = boot();
      store.append(b);
      store.append(next(b));
      expect(store.verifyChain()).toEqual({ valid: true, length: 2 });
    });

    // ── recent ────────────────────────────────────────────────────────

    it("recent returns newest first", () => {
      const b = boot();
      store.append(b);
      const e1 = next(b);
      store.append(e1);
      const r = store.recent(2);
      expect(r).toHaveLength(2);
      expect(r[0].id.value).toBe(e1.id.value);
    });

    // ── byType ────────────────────────────────────────────────────────

    it("byType returns matching events newest first", () => {
      const b = boot();
      store.append(b);
      const e1 = next(b);
      store.append(e1);
      const e2 = next(e1);
      store.append(e2);

      const results = store.byType(new EventType("trust.updated"), 10);
      expect(results).toHaveLength(2);
      expect(results[0].id.value).toBe(e2.id.value);
      expect(results[1].id.value).toBe(e1.id.value);
    });

    it("byType respects limit", () => {
      const b = boot();
      store.append(b);
      const e1 = next(b);
      store.append(e1);
      const e2 = next(e1);
      store.append(e2);

      const results = store.byType(new EventType("trust.updated"), 1);
      expect(results).toHaveLength(1);
      expect(results[0].id.value).toBe(e2.id.value);
    });

    it("byType returns empty for no matches", () => {
      const b = boot();
      store.append(b);
      expect(store.byType(new EventType("trust.updated"), 10)).toHaveLength(0);
    });

    // ── bySource ──────────────────────────────────────────────────────

    it("bySource returns matching events newest first", () => {
      const b = boot();
      store.append(b);
      const e1 = next(b);
      store.append(e1);

      const results = store.bySource(new ActorId("alice"), 10);
      expect(results).toHaveLength(2);
      expect(results[0].id.value).toBe(e1.id.value);
      expect(results[1].id.value).toBe(b.id.value);
    });

    it("bySource returns empty for unknown actor", () => {
      const b = boot();
      store.append(b);
      expect(store.bySource(new ActorId("bob"), 10)).toHaveLength(0);
    });

    // ── byConversation ────────────────────────────────────────────────

    it("byConversation returns matching events newest first", () => {
      const b = boot();
      store.append(b);
      const e1 = next(b);
      store.append(e1);

      const results = store.byConversation(new ConversationId("conv_1"), 10);
      expect(results).toHaveLength(1);
      expect(results[0].id.value).toBe(e1.id.value);
    });

    it("byConversation returns empty for unknown conversation", () => {
      const b = boot();
      store.append(b);
      expect(
        store.byConversation(new ConversationId("conv_unknown"), 10),
      ).toHaveLength(0);
    });

    // ── ancestors ─────────────────────────────────────────────────────

    it("ancestors returns causal ancestors", () => {
      const b = boot();
      store.append(b);
      const e1 = next(b);
      store.append(e1);
      const e2 = next(e1);
      store.append(e2);

      const anc = store.ancestors(e2.id, 10);
      expect(anc).toHaveLength(2);
      const ids = anc.map((e) => e.id.value);
      expect(ids).toContain(e1.id.value);
      expect(ids).toContain(b.id.value);
    });

    it("ancestors respects maxDepth", () => {
      const b = boot();
      store.append(b);
      const e1 = next(b);
      store.append(e1);
      const e2 = next(e1);
      store.append(e2);

      const anc = store.ancestors(e2.id, 1);
      expect(anc).toHaveLength(1);
      expect(anc[0].id.value).toBe(e1.id.value);
    });

    it("ancestors throws for unknown event", () => {
      expect(() =>
        store.ancestors(
          new EventId("019462a0-0000-7000-8000-000000000099"),
          10,
        ),
      ).toThrow(EventNotFoundError);
    });

    // ── descendants ───────────────────────────────────────────────────

    it("descendants returns causal descendants", () => {
      const b = boot();
      store.append(b);
      const e1 = next(b);
      store.append(e1);
      const e2 = next(e1);
      store.append(e2);

      const desc = store.descendants(b.id, 10);
      const ids = desc.map((e) => e.id.value);
      expect(ids).toContain(e1.id.value);
      expect(ids).toContain(e2.id.value);
    });

    it("descendants respects maxDepth", () => {
      const b = boot();
      store.append(b);
      const e1 = next(b);
      store.append(e1);
      const e2 = next(e1);
      store.append(e2);

      const desc = store.descendants(b.id, 1);
      expect(desc).toHaveLength(1);
      expect(desc[0].id.value).toBe(e1.id.value);
    });

    it("descendants throws for unknown event", () => {
      expect(() =>
        store.descendants(
          new EventId("019462a0-0000-7000-8000-000000000099"),
          10,
        ),
      ).toThrow(EventNotFoundError);
    });
  });
}
