import { describe, it, expect } from "vitest";
import {
  DefaultAuthorityChain,
  matchesAction,
  AuthorityLevel,
} from "../src/authority.js";
import { DefaultTrustModel } from "../src/trust.js";
import { Actor, ActorType, ActorStatus, InMemoryActorStore } from "../src/actor.js";
import {
  ActorId,
  ConversationId,
  DomainScope,
  EventId,
  EventType,
  Hash,
  NonEmpty,
  PublicKey,
  Score,
  Signature,
  Weight,
} from "../src/types.js";
import { Event, newEventId } from "../src/event.js";

// ── Helpers ──────────────────────────────────────────────────────────────

function testPublicKey(b: number): PublicKey {
  const key = new Uint8Array(32);
  key[0] = b;
  return new PublicKey(key);
}

function testActor(name: string, b: number): Actor {
  const store = new InMemoryActorStore();
  return store.register(testPublicKey(b), name, ActorType.Human);
}

function testTrustEvent(actorId: ActorId, prev: number, curr: number): Event {
  const content: Record<string, unknown> = {
    actor: actorId.value,
    previous: prev,
    current: curr,
    domain: "general",
  };
  const sig = new Signature(new Uint8Array(64));
  return new Event(
    1,
    newEventId(),
    new EventType("trust.updated"),
    Date.now() * 1_000_000,
    actorId,
    content,
    NonEmpty.of([newEventId()]),
    new ConversationId("conv_test"),
    Hash.zero(),
    Hash.zero(),
    sig,
  );
}

// ── Tests ────────────────────────────────────────────────────────────────

describe("DefaultAuthorityChain", () => {
  it("defaultNotification — unmatched action defaults to Notification", () => {
    const model = new DefaultTrustModel();
    const chain = new DefaultAuthorityChain(model);
    const actor = testActor("Alice", 1);

    const result = chain.evaluate(actor, "some.random.action");
    expect(result.level).toBe(AuthorityLevel.Notification);
  });

  it("policyRequired — exact match returns Required", () => {
    const model = new DefaultTrustModel();
    const chain = new DefaultAuthorityChain(model);
    chain.addPolicy({ action: "actor.suspend", level: AuthorityLevel.Required });

    const actor = testActor("Alice", 1);
    const result = chain.evaluate(actor, "actor.suspend");
    expect(result.level).toBe(AuthorityLevel.Required);
  });

  it("policyRecommended — exact match returns Recommended", () => {
    const model = new DefaultTrustModel();
    const chain = new DefaultAuthorityChain(model);
    chain.addPolicy({ action: "review.code", level: AuthorityLevel.Recommended });

    const actor = testActor("Alice", 1);
    const result = chain.evaluate(actor, "review.code");
    expect(result.level).toBe(AuthorityLevel.Recommended);
  });

  it("wildcardPolicy — prefix wildcard matches", () => {
    const model = new DefaultTrustModel();
    const chain = new DefaultAuthorityChain(model);
    chain.addPolicy({ action: "trust.*", level: AuthorityLevel.Recommended });

    const actor = testActor("Alice", 1);
    const result = chain.evaluate(actor, "trust.update");
    expect(result.level).toBe(AuthorityLevel.Recommended);
  });

  it("catchAllPolicy — * matches everything", () => {
    const model = new DefaultTrustModel();
    const chain = new DefaultAuthorityChain(model);
    chain.addPolicy({ action: "*", level: AuthorityLevel.Required });

    const actor = testActor("Alice", 1);
    const result = chain.evaluate(actor, "anything.at.all");
    expect(result.level).toBe(AuthorityLevel.Required);
  });

  it("firstMatchWins — first matching policy takes precedence", () => {
    const model = new DefaultTrustModel();
    const chain = new DefaultAuthorityChain(model);
    chain.addPolicy({ action: "deploy", level: AuthorityLevel.Required });
    chain.addPolicy({ action: "deploy", level: AuthorityLevel.Notification });

    const actor = testActor("Alice", 1);
    const result = chain.evaluate(actor, "deploy");
    expect(result.level).toBe(AuthorityLevel.Required);
  });

  it("trustDowngrade — Required downgrades to Recommended when trust is sufficient", () => {
    const model = new DefaultTrustModel();
    const chain = new DefaultAuthorityChain(model);
    chain.addPolicy({
      action: "deploy",
      level: AuthorityLevel.Required,
      minTrust: new Score(0.05),
    });

    const actor = testActor("Alice", 1);

    // Build trust above threshold by updating multiple times
    for (let i = 0; i < 10; i++) {
      const ev = testTrustEvent(actor.id, 0.0, 0.1);
      model.update(actor, ev);
    }

    const result = chain.evaluate(actor, "deploy");
    expect(result.level).toBe(AuthorityLevel.Recommended);
  });

  it("trustNoDowngrade — Required stays when trust is insufficient", () => {
    const model = new DefaultTrustModel();
    const chain = new DefaultAuthorityChain(model);
    chain.addPolicy({
      action: "deploy",
      level: AuthorityLevel.Required,
      minTrust: new Score(0.99),
    });

    const actor = testActor("Alice", 1);
    // Initial trust is 0.0, well below 0.99
    const result = chain.evaluate(actor, "deploy");
    expect(result.level).toBe(AuthorityLevel.Required);
  });

  it("chainReturnsSingleLink — flat model returns one-element chain", () => {
    const model = new DefaultTrustModel();
    const chain = new DefaultAuthorityChain(model);

    const actor = testActor("Alice", 1);
    const links = chain.chain(actor, "any.action");
    expect(links).toHaveLength(1);
    expect(links[0].actor.value).toBe(actor.id.value);
  });

  it("evaluateWeightIs1 — flat model always returns weight 1.0", () => {
    const model = new DefaultTrustModel();
    const chain = new DefaultAuthorityChain(model);

    const actor = testActor("Alice", 1);
    const result = chain.evaluate(actor, "test");
    expect(result.weight.value).toBe(1.0);
  });
});

describe("matchesAction", () => {
  it("exact match", () => {
    expect(matchesAction("deploy", "deploy")).toBe(true);
    expect(matchesAction("deploy", "review")).toBe(false);
  });

  it("prefix wildcard", () => {
    expect(matchesAction("trust.*", "trust.update")).toBe(true);
    expect(matchesAction("trust.*", "trust.")).toBe(true);
    expect(matchesAction("trust.*", "other.action")).toBe(false);
  });

  it("catch-all wildcard", () => {
    expect(matchesAction("*", "anything")).toBe(true);
    expect(matchesAction("*", "")).toBe(true);
  });
});
