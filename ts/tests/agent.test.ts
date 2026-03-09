import { describe, it, expect } from "vitest";
import {
  OperationalState,
  isValidOperationalTransition,
  transitionTo,
  isTerminal,
  canAct,
  allAgentPrimitives,
  registerAllAgentPrimitives,
  isAgentPrimitive,
  allAgentEventTypes,
  allAgentCompositions,
  bootComposition,
  imprintComposition,
  taskComposition,
  superviseComposition,
  collaborateComposition,
  crisisComposition,
  retireComposition,
  whistleblowComposition,
} from "../src/agent.js";
import { Registry } from "../src/primitive.js";
import type { Snapshot } from "../src/primitive.js";
import type { Event } from "../src/event.js";
import { EventType, PrimitiveId } from "../src/types.js";

// ── Helpers ──────────────────────────────────────────────────────────────

const fakeSnapshot: Snapshot = {
  tick: 1,
  primitives: new Map(),
  pendingEvents: [],
  recentEvents: [],
};

function fakeEvent(typeStr: string): Event {
  return { type: new EventType(typeStr) } as unknown as Event;
}

// ── OperationalState FSM ─────────────────────────────────────────────────

describe("OperationalState FSM", () => {
  it("has 8 states", () => {
    const states = Object.values(OperationalState);
    expect(states.length).toBe(8);
  });

  describe("valid transitions", () => {
    const validCases: [string, string][] = [
      ["Idle", "Processing"],
      ["Idle", "Suspended"],
      ["Idle", "Retiring"],
      ["Processing", "Idle"],
      ["Processing", "Waiting"],
      ["Processing", "Escalating"],
      ["Processing", "Refusing"],
      ["Processing", "Retiring"],
      ["Waiting", "Processing"],
      ["Waiting", "Idle"],
      ["Waiting", "Retiring"],
      ["Escalating", "Waiting"],
      ["Escalating", "Idle"],
      ["Refusing", "Idle"],
      ["Suspended", "Idle"],
      ["Suspended", "Retiring"],
      ["Retiring", "Retired"],
    ];

    for (const [from, to] of validCases) {
      it(`${from} -> ${to} is valid`, () => {
        expect(isValidOperationalTransition(from as OperationalState, to as OperationalState)).toBe(true);
        const result = transitionTo(from as OperationalState, to as OperationalState);
        expect(result).toBe(to);
      });
    }
  });

  describe("invalid transitions", () => {
    const invalidCases: [string, string][] = [
      ["Idle", "Waiting"],
      ["Idle", "Escalating"],
      ["Idle", "Refusing"],
      ["Idle", "Retired"],
      ["Processing", "Suspended"],
      ["Waiting", "Escalating"],
      ["Escalating", "Retired"],
      ["Refusing", "Processing"],
      ["Suspended", "Processing"],
      ["Retiring", "Idle"],
      ["Retired", "Idle"],
      ["Retired", "Processing"],
      ["Retired", "Retired"],
    ];

    for (const [from, to] of invalidCases) {
      it(`${from} -> ${to} is invalid`, () => {
        expect(isValidOperationalTransition(from as OperationalState, to as OperationalState)).toBe(false);
        expect(() => transitionTo(from as OperationalState, to as OperationalState)).toThrow("Invalid transition");
      });
    }
  });

  it("Retired is terminal", () => {
    expect(isTerminal(OperationalState.Retired)).toBe(true);
    for (const s of Object.values(OperationalState)) {
      if (s !== OperationalState.Retired) {
        expect(isTerminal(s as OperationalState)).toBe(false);
      }
    }
  });

  it("only Processing canAct", () => {
    expect(canAct(OperationalState.Processing)).toBe(true);
    for (const s of Object.values(OperationalState)) {
      if (s !== OperationalState.Processing) {
        expect(canAct(s as OperationalState)).toBe(false);
      }
    }
  });
});

// ── Agent Primitives ─────────────────────────────────────────────────────

describe("agent primitives", () => {
  const all = allAgentPrimitives();

  it("returns exactly 28 primitives", () => {
    expect(all.length).toBe(28);
  });

  it("all IDs are unique", () => {
    const ids = new Set(all.map((p) => p.id().value));
    expect(ids.size).toBe(28);
  });

  it("all primitives are at Layer 1", () => {
    for (const p of all) {
      expect(p.layer().value).toBe(1);
    }
  });

  it("all primitives have cadence 1", () => {
    for (const p of all) {
      expect(p.cadence().value).toBe(1);
    }
  });

  it("all primitives have non-empty subscriptions", () => {
    for (const p of all) {
      const subs = p.subscriptions();
      expect(subs.length).toBeGreaterThan(0);
      for (const s of subs) {
        expect(s.value).toBeTruthy();
      }
    }
  });

  it("all primitive IDs start with agent.", () => {
    for (const p of all) {
      expect(p.id().value.startsWith("agent.")).toBe(true);
    }
  });

  it("isAgentPrimitive recognises agent primitives", () => {
    for (const p of all) {
      expect(isAgentPrimitive(p.id())).toBe(true);
    }
    expect(isAgentPrimitive(new PrimitiveId("Event"))).toBe(false);
    expect(isAgentPrimitive(new PrimitiveId("Clock"))).toBe(false);
  });

  it("structural primitives count is 11", () => {
    const structural = [
      "agent.Identity", "agent.Soul", "agent.Model", "agent.Memory",
      "agent.State", "agent.Authority", "agent.Trust", "agent.Budget",
      "agent.Role", "agent.Lifespan", "agent.Goal",
    ];
    const ids = all.map((p) => p.id().value);
    const found = structural.filter((s) => ids.includes(s));
    expect(found.length).toBe(11);
  });

  it("operational primitives count is 13", () => {
    const operational = [
      "agent.Observe", "agent.Probe", "agent.Evaluate", "agent.Decide",
      "agent.Act", "agent.Delegate", "agent.Escalate", "agent.Refuse",
      "agent.Learn", "agent.Introspect", "agent.Communicate", "agent.Repair",
      "agent.Expect",
    ];
    const ids = all.map((p) => p.id().value);
    const found = operational.filter((s) => ids.includes(s));
    expect(found.length).toBe(13);
  });

  it("relational primitives count is 3", () => {
    const relational = ["agent.Consent", "agent.Channel", "agent.Composition"];
    const ids = all.map((p) => p.id().value);
    const found = relational.filter((s) => ids.includes(s));
    expect(found.length).toBe(3);
  });

  it("modal primitives count is 1", () => {
    const modal = ["agent.Attenuation"];
    const ids = all.map((p) => p.id().value);
    const found = modal.filter((s) => ids.includes(s));
    expect(found.length).toBe(1);
  });

  it("all 28 primitives process with empty events", () => {
    for (const p of all) {
      const mutations = p.process(1, [], fakeSnapshot);
      expect(mutations.length).toBeGreaterThan(0);
      // Every primitive should emit a lastTick mutation
      const lastTickMutation = mutations.find(
        (m) => m.kind === "updateState" && m.key === "lastTick",
      );
      expect(lastTickMutation).toBeDefined();
      if (lastTickMutation && lastTickMutation.kind === "updateState") {
        expect(lastTickMutation.value).toBe(1);
      }
    }
  });

  it("process records tick correctly at tick 42", () => {
    for (const p of all) {
      const mutations = p.process(42, [], fakeSnapshot);
      const lastTickMutation = mutations.find(
        (m) => m.kind === "updateState" && m.key === "lastTick",
      );
      expect(lastTickMutation).toBeDefined();
      if (lastTickMutation && lastTickMutation.kind === "updateState") {
        expect(lastTickMutation.value).toBe(42);
      }
    }
  });
});

// ── Individual primitive processing tests ────────────────────────────────

describe("individual primitive processing", () => {
  it("IdentityPrimitive counts created and rotated events", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.Identity")!;
    const events = [
      fakeEvent("agent.identity.created"),
      fakeEvent("agent.identity.created"),
      fakeEvent("agent.identity.rotated"),
      fakeEvent("actor.registered"),
    ];
    const mutations = p.process(1, events, fakeSnapshot);
    const created = mutations.find((m) => m.kind === "updateState" && m.key === "identitiesCreated");
    const rotated = mutations.find((m) => m.kind === "updateState" && m.key === "keysRotated");
    expect(created && created.kind === "updateState" ? created.value : undefined).toBe(3);
    expect(rotated && rotated.kind === "updateState" ? rotated.value : undefined).toBe(1);
  });

  it("SoulPrimitive tracks imprints and refusals", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.Soul")!;
    const events = [fakeEvent("agent.soul.imprinted"), fakeEvent("agent.refused")];
    const mutations = p.process(1, events, fakeSnapshot);
    const imprinted = mutations.find((m) => m.kind === "updateState" && m.key === "imprinted");
    const refusals = mutations.find((m) => m.kind === "updateState" && m.key === "soulRefusals");
    expect(imprinted && imprinted.kind === "updateState" ? imprinted.value : undefined).toBe(true);
    expect(refusals && refusals.kind === "updateState" ? refusals.value : undefined).toBe(1);
  });

  it("SoulPrimitive without imprint does not emit imprinted mutation", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.Soul")!;
    const mutations = p.process(1, [], fakeSnapshot);
    const imprinted = mutations.find((m) => m.kind === "updateState" && m.key === "imprinted");
    expect(imprinted).toBeUndefined();
  });

  it("GoalPrimitive counts set/completed/abandoned", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.Goal")!;
    const events = [
      fakeEvent("agent.goal.set"),
      fakeEvent("agent.goal.set"),
      fakeEvent("agent.goal.completed"),
      fakeEvent("agent.goal.abandoned"),
    ];
    const mutations = p.process(1, events, fakeSnapshot);
    const set = mutations.find((m) => m.kind === "updateState" && m.key === "goalsSet");
    const completed = mutations.find((m) => m.kind === "updateState" && m.key === "goalsCompleted");
    const abandoned = mutations.find((m) => m.kind === "updateState" && m.key === "goalsAbandoned");
    expect(set && set.kind === "updateState" ? set.value : undefined).toBe(2);
    expect(completed && completed.kind === "updateState" ? completed.value : undefined).toBe(1);
    expect(abandoned && abandoned.kind === "updateState" ? abandoned.value : undefined).toBe(1);
  });

  it("ObservePrimitive counts observed and total events", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.Observe")!;
    const events = [
      fakeEvent("agent.observed"),
      fakeEvent("agent.observed"),
      fakeEvent("agent.acted"),
    ];
    const mutations = p.process(1, events, fakeSnapshot);
    const observed = mutations.find((m) => m.kind === "updateState" && m.key === "eventsObserved");
    const total = mutations.find((m) => m.kind === "updateState" && m.key === "totalEventsReceived");
    expect(observed && observed.kind === "updateState" ? observed.value : undefined).toBe(2);
    expect(total && total.kind === "updateState" ? total.value : undefined).toBe(3);
  });

  it("AttenuationPrimitive counts attenuated, lifted, and budget triggered", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.Attenuation")!;
    const events = [
      fakeEvent("agent.attenuated"),
      fakeEvent("agent.attenuation.lifted"),
      fakeEvent("agent.budget.exhausted"),
      fakeEvent("agent.budget.exhausted"),
    ];
    const mutations = p.process(1, events, fakeSnapshot);
    const att = mutations.find((m) => m.kind === "updateState" && m.key === "attenuations");
    const lifts = mutations.find((m) => m.kind === "updateState" && m.key === "lifts");
    const budget = mutations.find((m) => m.kind === "updateState" && m.key === "budgetTriggered");
    expect(att && att.kind === "updateState" ? att.value : undefined).toBe(1);
    expect(lifts && lifts.kind === "updateState" ? lifts.value : undefined).toBe(1);
    expect(budget && budget.kind === "updateState" ? budget.value : undefined).toBe(2);
  });

  it("ConsentPrimitive counts requested/granted/denied", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.Consent")!;
    const events = [
      fakeEvent("agent.consent.requested"),
      fakeEvent("agent.consent.granted"),
      fakeEvent("agent.consent.denied"),
    ];
    const mutations = p.process(1, events, fakeSnapshot);
    const req = mutations.find((m) => m.kind === "updateState" && m.key === "consentRequested");
    const gra = mutations.find((m) => m.kind === "updateState" && m.key === "consentGranted");
    const den = mutations.find((m) => m.kind === "updateState" && m.key === "consentDenied");
    expect(req && req.kind === "updateState" ? req.value : undefined).toBe(1);
    expect(gra && gra.kind === "updateState" ? gra.value : undefined).toBe(1);
    expect(den && den.kind === "updateState" ? den.value : undefined).toBe(1);
  });

  it("CompositionPrimitive counts formed/dissolved/joined/left", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.Composition")!;
    const events = [
      fakeEvent("agent.composition.formed"),
      fakeEvent("agent.composition.dissolved"),
      fakeEvent("agent.composition.joined"),
      fakeEvent("agent.composition.left"),
    ];
    const mutations = p.process(1, events, fakeSnapshot);
    const formed = mutations.find((m) => m.kind === "updateState" && m.key === "groupsFormed");
    const dissolved = mutations.find((m) => m.kind === "updateState" && m.key === "groupsDissolved");
    const joined = mutations.find((m) => m.kind === "updateState" && m.key === "membersJoined");
    const left = mutations.find((m) => m.kind === "updateState" && m.key === "membersLeft");
    expect(formed && formed.kind === "updateState" ? formed.value : undefined).toBe(1);
    expect(dissolved && dissolved.kind === "updateState" ? dissolved.value : undefined).toBe(1);
    expect(joined && joined.kind === "updateState" ? joined.value : undefined).toBe(1);
    expect(left && left.kind === "updateState" ? left.value : undefined).toBe(1);
  });

  it("ExpectPrimitive counts set/met/expired", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.Expect")!;
    const events = [
      fakeEvent("agent.expectation.set"),
      fakeEvent("agent.expectation.met"),
      fakeEvent("agent.expectation.expired"),
    ];
    const mutations = p.process(1, events, fakeSnapshot);
    const set = mutations.find((m) => m.kind === "updateState" && m.key === "expectationsSet");
    const met = mutations.find((m) => m.kind === "updateState" && m.key === "expectationsMet");
    const expired = mutations.find((m) => m.kind === "updateState" && m.key === "expectationsExpired");
    expect(set && set.kind === "updateState" ? set.value : undefined).toBe(1);
    expect(met && met.kind === "updateState" ? met.value : undefined).toBe(1);
    expect(expired && expired.kind === "updateState" ? expired.value : undefined).toBe(1);
  });

  it("StatePrimitive tracks transitions", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.State")!;
    const events = [fakeEvent("agent.state.changed"), fakeEvent("agent.state.changed")];
    const mutations = p.process(1, events, fakeSnapshot);
    const transitions = mutations.find((m) => m.kind === "updateState" && m.key === "transitions");
    const lastTransition = mutations.find((m) => m.kind === "updateState" && m.key === "lastTransition");
    expect(transitions && transitions.kind === "updateState" ? transitions.value : undefined).toBe(2);
    expect(lastTransition && lastTransition.kind === "updateState" ? lastTransition.value : undefined).toBe("changed");
  });

  it("StatePrimitive omits lastTransition when no state changes", () => {
    const p = allAgentPrimitives().find((x) => x.id().value === "agent.State")!;
    const mutations = p.process(1, [], fakeSnapshot);
    const lastTransition = mutations.find((m) => m.kind === "updateState" && m.key === "lastTransition");
    expect(lastTransition).toBeUndefined();
  });
});

// ── Registry ─────────────────────────────────────────────────────────────

describe("registerAllAgentPrimitives", () => {
  it("registers 28 primitives in the registry", () => {
    const reg = new Registry();
    registerAllAgentPrimitives(reg);
    expect(reg.count).toBe(28);
  });

  it("all registered primitives are active", () => {
    const reg = new Registry();
    registerAllAgentPrimitives(reg);
    for (const p of allAgentPrimitives()) {
      expect(reg.getLifecycle(p.id())).toBe("active");
    }
  });

  it("double registration throws", () => {
    const reg = new Registry();
    registerAllAgentPrimitives(reg);
    expect(() => registerAllAgentPrimitives(reg)).toThrow("already registered");
  });
});

// ── Agent Event Types ────────────────────────────────────────────────────

describe("agent event types", () => {
  it("returns 45 event types", () => {
    const types = allAgentEventTypes();
    expect(types.length).toBe(45);
  });

  it("all event types are unique", () => {
    const types = allAgentEventTypes();
    const values = new Set(types.map((t) => t.value));
    expect(values.size).toBe(45);
  });

  it("all event types start with agent.", () => {
    for (const t of allAgentEventTypes()) {
      expect(t.value.startsWith("agent.")).toBe(true);
    }
  });
});

// ── Compositions ─────────────────────────────────────────────────────────

describe("agent compositions", () => {
  it("returns 8 compositions", () => {
    const comps = allAgentCompositions();
    expect(comps.length).toBe(8);
  });

  it("all compositions have unique names", () => {
    const names = new Set(allAgentCompositions().map((c) => c.name));
    expect(names.size).toBe(8);
  });

  it("all compositions have non-empty primitives and events", () => {
    for (const c of allAgentCompositions()) {
      expect(c.primitives.length).toBeGreaterThan(0);
      expect(c.events.length).toBeGreaterThan(0);
    }
  });

  it("Boot has 5 primitives and 5 events", () => {
    const c = bootComposition();
    expect(c.name).toBe("Boot");
    expect(c.primitives.length).toBe(5);
    expect(c.events.length).toBe(5);
  });

  it("Imprint has 8 primitives and 8 events", () => {
    const c = imprintComposition();
    expect(c.name).toBe("Imprint");
    expect(c.primitives.length).toBe(8);
    expect(c.events.length).toBe(8);
  });

  it("Task has 5 primitives and 5 events", () => {
    const c = taskComposition();
    expect(c.name).toBe("Task");
    expect(c.primitives.length).toBe(5);
    expect(c.events.length).toBe(5);
  });

  it("Supervise has 5 primitives and 4 events", () => {
    const c = superviseComposition();
    expect(c.name).toBe("Supervise");
    expect(c.primitives.length).toBe(5);
    expect(c.events.length).toBe(4);
  });

  it("Collaborate has 5 primitives and 6 events", () => {
    const c = collaborateComposition();
    expect(c.name).toBe("Collaborate");
    expect(c.primitives.length).toBe(5);
    expect(c.events.length).toBe(6);
  });

  it("Crisis has 5 primitives and 5 events", () => {
    const c = crisisComposition();
    expect(c.name).toBe("Crisis");
    expect(c.primitives.length).toBe(5);
    expect(c.events.length).toBe(5);
  });

  it("Retire has 4 primitives and 4 events", () => {
    const c = retireComposition();
    expect(c.name).toBe("Retire");
    expect(c.primitives.length).toBe(4);
    expect(c.events.length).toBe(4);
  });

  it("Whistleblow has 5 primitives and 5 events", () => {
    const c = whistleblowComposition();
    expect(c.name).toBe("Whistleblow");
    expect(c.primitives.length).toBe(5);
    expect(c.events.length).toBe(5);
  });

  it("all composition primitive IDs reference valid agent primitives", () => {
    const validIds = new Set(allAgentPrimitives().map((p) => p.id().value));
    for (const c of allAgentCompositions()) {
      for (const pid of c.primitives) {
        expect(validIds.has(pid)).toBe(true);
      }
    }
  });

  it("all composition event types are valid agent event types", () => {
    const validTypes = new Set(allAgentEventTypes().map((t) => t.value));
    for (const c of allAgentCompositions()) {
      for (const et of c.events) {
        expect(validTypes.has(et.value)).toBe(true);
      }
    }
  });
});
