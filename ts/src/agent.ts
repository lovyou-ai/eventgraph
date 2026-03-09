/**
 * Agent primitives — 28 primitives at Layer 1 (Agency) plus 8 named compositions.
 * Ported from the Go reference implementation.
 */
import { Event } from "./event.js";
import { Primitive, Mutation, Snapshot, Registry } from "./primitive.js";
import { PrimitiveId, Layer, Cadence, EventType, SubscriptionPattern } from "./types.js";

// ── OperationalState FSM ─────────────────────────────────────────────────

export const OperationalState = {
  Idle: "Idle",
  Processing: "Processing",
  Waiting: "Waiting",
  Escalating: "Escalating",
  Refusing: "Refusing",
  Suspended: "Suspended",
  Retiring: "Retiring",
  Retired: "Retired",
} as const;

export type OperationalState = (typeof OperationalState)[keyof typeof OperationalState];

const OPERATIONAL_TRANSITIONS: Record<OperationalState, Set<OperationalState>> = {
  [OperationalState.Idle]: new Set([OperationalState.Processing, OperationalState.Suspended, OperationalState.Retiring]),
  [OperationalState.Processing]: new Set([OperationalState.Idle, OperationalState.Waiting, OperationalState.Escalating, OperationalState.Refusing, OperationalState.Retiring]),
  [OperationalState.Waiting]: new Set([OperationalState.Processing, OperationalState.Idle, OperationalState.Retiring]),
  [OperationalState.Escalating]: new Set([OperationalState.Waiting, OperationalState.Idle]),
  [OperationalState.Refusing]: new Set([OperationalState.Idle]),
  [OperationalState.Suspended]: new Set([OperationalState.Idle, OperationalState.Retiring]),
  [OperationalState.Retiring]: new Set([OperationalState.Retired]),
  [OperationalState.Retired]: new Set(),
};

export function isValidOperationalTransition(from: OperationalState, to: OperationalState): boolean {
  return OPERATIONAL_TRANSITIONS[from]?.has(to) ?? false;
}

export function transitionTo(current: OperationalState, target: OperationalState): OperationalState {
  if (!isValidOperationalTransition(current, target)) {
    throw new Error(`Invalid transition: ${current} -> ${target}`);
  }
  return target;
}

export function isTerminal(state: OperationalState): boolean {
  return state === OperationalState.Retired;
}

export function canAct(state: OperationalState): boolean {
  return state === OperationalState.Processing;
}

// ── Agent Event Types ────────────────────────────────────────────────────

// Structural events
export const AgentEventTypeIdentityCreated = new EventType("agent.identity.created");
export const AgentEventTypeIdentityRotated = new EventType("agent.identity.rotated");
export const AgentEventTypeSoulImprinted = new EventType("agent.soul.imprinted");
export const AgentEventTypeModelBound = new EventType("agent.model.bound");
export const AgentEventTypeModelChanged = new EventType("agent.model.changed");
export const AgentEventTypeMemoryUpdated = new EventType("agent.memory.updated");
export const AgentEventTypeStateChanged = new EventType("agent.state.changed");
export const AgentEventTypeAuthorityGranted = new EventType("agent.authority.granted");
export const AgentEventTypeAuthorityRevoked = new EventType("agent.authority.revoked");
export const AgentEventTypeTrustAssessed = new EventType("agent.trust.assessed");
export const AgentEventTypeBudgetAllocated = new EventType("agent.budget.allocated");
export const AgentEventTypeBudgetExhausted = new EventType("agent.budget.exhausted");
export const AgentEventTypeRoleAssigned = new EventType("agent.role.assigned");
export const AgentEventTypeLifespanStarted = new EventType("agent.lifespan.started");
export const AgentEventTypeLifespanExtended = new EventType("agent.lifespan.extended");
export const AgentEventTypeLifespanEnded = new EventType("agent.lifespan.ended");
export const AgentEventTypeGoalSet = new EventType("agent.goal.set");
export const AgentEventTypeGoalCompleted = new EventType("agent.goal.completed");
export const AgentEventTypeGoalAbandoned = new EventType("agent.goal.abandoned");

// Operational events
export const AgentEventTypeObserved = new EventType("agent.observed");
export const AgentEventTypeProbed = new EventType("agent.probed");
export const AgentEventTypeEvaluated = new EventType("agent.evaluated");
export const AgentEventTypeDecided = new EventType("agent.decided");
export const AgentEventTypeActed = new EventType("agent.acted");
export const AgentEventTypeDelegated = new EventType("agent.delegated");
export const AgentEventTypeEscalated = new EventType("agent.escalated");
export const AgentEventTypeRefused = new EventType("agent.refused");
export const AgentEventTypeLearned = new EventType("agent.learned");
export const AgentEventTypeIntrospected = new EventType("agent.introspected");
export const AgentEventTypeCommunicated = new EventType("agent.communicated");
export const AgentEventTypeRepaired = new EventType("agent.repaired");
export const AgentEventTypeExpectationSet = new EventType("agent.expectation.set");
export const AgentEventTypeExpectationMet = new EventType("agent.expectation.met");
export const AgentEventTypeExpectationExpired = new EventType("agent.expectation.expired");

// Relational events
export const AgentEventTypeConsentRequested = new EventType("agent.consent.requested");
export const AgentEventTypeConsentGranted = new EventType("agent.consent.granted");
export const AgentEventTypeConsentDenied = new EventType("agent.consent.denied");
export const AgentEventTypeChannelOpened = new EventType("agent.channel.opened");
export const AgentEventTypeChannelClosed = new EventType("agent.channel.closed");
export const AgentEventTypeCompositionFormed = new EventType("agent.composition.formed");
export const AgentEventTypeCompositionDissolved = new EventType("agent.composition.dissolved");
export const AgentEventTypeCompositionJoined = new EventType("agent.composition.joined");
export const AgentEventTypeCompositionLeft = new EventType("agent.composition.left");

// Modal events
export const AgentEventTypeAttenuated = new EventType("agent.attenuated");
export const AgentEventTypeAttenuationLifted = new EventType("agent.attenuation.lifted");

export function allAgentEventTypes(): EventType[] {
  return [
    // Structural
    AgentEventTypeIdentityCreated, AgentEventTypeIdentityRotated,
    AgentEventTypeSoulImprinted,
    AgentEventTypeModelBound, AgentEventTypeModelChanged,
    AgentEventTypeMemoryUpdated,
    AgentEventTypeStateChanged,
    AgentEventTypeAuthorityGranted, AgentEventTypeAuthorityRevoked,
    AgentEventTypeTrustAssessed,
    AgentEventTypeBudgetAllocated, AgentEventTypeBudgetExhausted,
    AgentEventTypeRoleAssigned,
    AgentEventTypeLifespanStarted, AgentEventTypeLifespanExtended, AgentEventTypeLifespanEnded,
    AgentEventTypeGoalSet, AgentEventTypeGoalCompleted, AgentEventTypeGoalAbandoned,
    // Operational
    AgentEventTypeObserved, AgentEventTypeProbed,
    AgentEventTypeEvaluated, AgentEventTypeDecided,
    AgentEventTypeActed, AgentEventTypeDelegated,
    AgentEventTypeEscalated, AgentEventTypeRefused,
    AgentEventTypeLearned, AgentEventTypeIntrospected,
    AgentEventTypeCommunicated, AgentEventTypeRepaired,
    AgentEventTypeExpectationSet, AgentEventTypeExpectationMet, AgentEventTypeExpectationExpired,
    // Relational
    AgentEventTypeConsentRequested, AgentEventTypeConsentGranted, AgentEventTypeConsentDenied,
    AgentEventTypeChannelOpened, AgentEventTypeChannelClosed,
    AgentEventTypeCompositionFormed, AgentEventTypeCompositionDissolved,
    AgentEventTypeCompositionJoined, AgentEventTypeCompositionLeft,
    // Modal
    AgentEventTypeAttenuated, AgentEventTypeAttenuationLifted,
  ];
}

// ── Shared constants ─────────────────────────────────────────────────────

const AGENT_LAYER = new Layer(1);
const CADENCE_1 = new Cadence(1);

// ── Helper to get event type string from a fake/real event ───────────────

function eventTypeValue(ev: Event): string {
  return ev.type.value;
}

// ════════════════════════════════════════════════════════════════════════
// STRUCTURAL PRIMITIVES (11) — Define what an agent IS
// ════════════════════════════════════════════════════════════════════════

export class IdentityPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Identity"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.identity.*"),
      new SubscriptionPattern("actor.registered"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let created = 0;
    let rotated = 0;
    for (const ev of events) {
      const t = eventTypeValue(ev);
      if (t === "agent.identity.created" || t === "actor.registered") created++;
      else if (t === "agent.identity.rotated") rotated++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "identitiesCreated", value: created },
      { kind: "updateState", primitiveId: this.id(), key: "keysRotated", value: rotated },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class SoulPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Soul"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.soul.*"),
      new SubscriptionPattern("agent.refused"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let imprints = 0;
    let refusals = 0;
    for (const ev of events) {
      const t = eventTypeValue(ev);
      if (t === "agent.soul.imprinted") imprints++;
      else if (t === "agent.refused") refusals++;
    }
    const mutations: Mutation[] = [
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
    if (imprints > 0) {
      mutations.push({ kind: "updateState", primitiveId: this.id(), key: "imprinted", value: true });
    }
    if (refusals > 0) {
      mutations.push({ kind: "updateState", primitiveId: this.id(), key: "soulRefusals", value: refusals });
    }
    return mutations;
  }
}

export class ModelPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Model"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.model.*")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let bindings = 0;
    let changes = 0;
    for (const ev of events) {
      switch (eventTypeValue(ev)) {
        case "agent.model.bound": bindings++; break;
        case "agent.model.changed": changes++; break;
      }
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "bindings", value: bindings },
      { kind: "updateState", primitiveId: this.id(), key: "modelChanges", value: changes },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class MemoryPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Memory"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.memory.*"),
      new SubscriptionPattern("agent.learned"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let updates = 0;
    for (const ev of events) {
      const t = eventTypeValue(ev);
      if (t === "agent.memory.updated" || t === "agent.learned") updates++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "memoryUpdates", value: updates },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class StatePrimitive implements Primitive {
  id() { return new PrimitiveId("agent.State"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.state.*")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let transitions = 0;
    let lastState = "";
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.state.changed") {
        transitions++;
        lastState = "changed";
      }
    }
    const mutations: Mutation[] = [
      { kind: "updateState", primitiveId: this.id(), key: "transitions", value: transitions },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
    if (lastState !== "") {
      mutations.push({ kind: "updateState", primitiveId: this.id(), key: "lastTransition", value: lastState });
    }
    return mutations;
  }
}

export class AuthorityPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Authority"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.authority.*"),
      new SubscriptionPattern("authority.*"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let granted = 0;
    let revoked = 0;
    for (const ev of events) {
      const t = eventTypeValue(ev);
      if (t === "agent.authority.granted") granted++;
      else if (t === "agent.authority.revoked") revoked++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "authorityGrants", value: granted },
      { kind: "updateState", primitiveId: this.id(), key: "authorityRevocations", value: revoked },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class TrustPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Trust"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.trust.*"),
      new SubscriptionPattern("trust.*"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let assessments = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.trust.assessed") assessments++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "trustAssessments", value: assessments },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class BudgetPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Budget"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.budget.*")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let allocated = 0;
    let exhausted = 0;
    for (const ev of events) {
      switch (eventTypeValue(ev)) {
        case "agent.budget.allocated": allocated++; break;
        case "agent.budget.exhausted": exhausted++; break;
      }
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "allocations", value: allocated },
      { kind: "updateState", primitiveId: this.id(), key: "exhaustions", value: exhausted },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class RolePrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Role"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.role.*")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let assignments = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.role.assigned") assignments++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "roleAssignments", value: assignments },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class LifespanPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Lifespan"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.lifespan.*")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let started = 0;
    let ended = 0;
    for (const ev of events) {
      switch (eventTypeValue(ev)) {
        case "agent.lifespan.started": started++; break;
        case "agent.lifespan.ended": ended++; break;
      }
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "agentsStarted", value: started },
      { kind: "updateState", primitiveId: this.id(), key: "agentsEnded", value: ended },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class GoalPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Goal"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.goal.*")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let set = 0;
    let completed = 0;
    let abandoned = 0;
    for (const ev of events) {
      switch (eventTypeValue(ev)) {
        case "agent.goal.set": set++; break;
        case "agent.goal.completed": completed++; break;
        case "agent.goal.abandoned": abandoned++; break;
      }
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "goalsSet", value: set },
      { kind: "updateState", primitiveId: this.id(), key: "goalsCompleted", value: completed },
      { kind: "updateState", primitiveId: this.id(), key: "goalsAbandoned", value: abandoned },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

// ════════════════════════════════════════════════════════════════════════
// OPERATIONAL PRIMITIVES (13) — Define what an agent DOES
// ════════════════════════════════════════════════════════════════════════

export class ObservePrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Observe"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.observed"),
      new SubscriptionPattern("agent.*"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let observed = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.observed") observed++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "eventsObserved", value: observed },
      { kind: "updateState", primitiveId: this.id(), key: "totalEventsReceived", value: events.length },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class ProbePrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Probe"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.probed")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let probes = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.probed") probes++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "probesExecuted", value: probes },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class EvaluatePrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Evaluate"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.evaluated")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let evaluations = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.evaluated") evaluations++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "evaluations", value: evaluations },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class DecidePrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Decide"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.decided"),
      new SubscriptionPattern("agent.evaluated"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let decisions = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.decided") decisions++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "decisions", value: decisions },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class ActPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Act"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.acted"),
      new SubscriptionPattern("agent.decided"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let actions = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.acted") actions++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "actionsExecuted", value: actions },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class DelegatePrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Delegate"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.delegated")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let delegations = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.delegated") delegations++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "delegations", value: delegations },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class EscalatePrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Escalate"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.escalated")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let escalations = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.escalated") escalations++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "escalations", value: escalations },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class RefusePrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Refuse"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.refused")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let refusals = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.refused") refusals++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "refusals", value: refusals },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class LearnPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Learn"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.learned"),
      new SubscriptionPattern("agent.goal.completed"),
      new SubscriptionPattern("agent.goal.abandoned"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let lessons = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.learned") lessons++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "lessonsLearned", value: lessons },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class IntrospectPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Introspect"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.introspected")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let introspections = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.introspected") introspections++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "introspections", value: introspections },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class CommunicatePrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Communicate"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.communicated"),
      new SubscriptionPattern("agent.channel.*"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let messages = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.communicated") messages++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "messagesSent", value: messages },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class RepairPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Repair"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.repaired")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let repairs = 0;
    for (const ev of events) {
      if (eventTypeValue(ev) === "agent.repaired") repairs++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "repairs", value: repairs },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class ExpectPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Expect"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.expectation.*")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let set = 0;
    let met = 0;
    let expired = 0;
    for (const ev of events) {
      switch (eventTypeValue(ev)) {
        case "agent.expectation.set": set++; break;
        case "agent.expectation.met": met++; break;
        case "agent.expectation.expired": expired++; break;
      }
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "expectationsSet", value: set },
      { kind: "updateState", primitiveId: this.id(), key: "expectationsMet", value: met },
      { kind: "updateState", primitiveId: this.id(), key: "expectationsExpired", value: expired },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

// ════════════════════════════════════════════════════════════════════════
// RELATIONAL PRIMITIVES (3) — Define how agents relate
// ════════════════════════════════════════════════════════════════════════

export class ConsentPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Consent"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.consent.*")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let requested = 0;
    let granted = 0;
    let denied = 0;
    for (const ev of events) {
      switch (eventTypeValue(ev)) {
        case "agent.consent.requested": requested++; break;
        case "agent.consent.granted": granted++; break;
        case "agent.consent.denied": denied++; break;
      }
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "consentRequested", value: requested },
      { kind: "updateState", primitiveId: this.id(), key: "consentGranted", value: granted },
      { kind: "updateState", primitiveId: this.id(), key: "consentDenied", value: denied },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class ChannelPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Channel"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.channel.*")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let opened = 0;
    let closed = 0;
    for (const ev of events) {
      switch (eventTypeValue(ev)) {
        case "agent.channel.opened": opened++; break;
        case "agent.channel.closed": closed++; break;
      }
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "channelsOpened", value: opened },
      { kind: "updateState", primitiveId: this.id(), key: "channelsClosed", value: closed },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

export class CompositionPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Composition"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [new SubscriptionPattern("agent.composition.*")];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let formed = 0;
    let dissolved = 0;
    let joined = 0;
    let left = 0;
    for (const ev of events) {
      switch (eventTypeValue(ev)) {
        case "agent.composition.formed": formed++; break;
        case "agent.composition.dissolved": dissolved++; break;
        case "agent.composition.joined": joined++; break;
        case "agent.composition.left": left++; break;
      }
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "groupsFormed", value: formed },
      { kind: "updateState", primitiveId: this.id(), key: "groupsDissolved", value: dissolved },
      { kind: "updateState", primitiveId: this.id(), key: "membersJoined", value: joined },
      { kind: "updateState", primitiveId: this.id(), key: "membersLeft", value: left },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

// ════════════════════════════════════════════════════════════════════════
// MODAL PRIMITIVE (1) — Modifies how other primitives operate
// ════════════════════════════════════════════════════════════════════════

export class AttenuationPrimitive implements Primitive {
  id() { return new PrimitiveId("agent.Attenuation"); }
  layer() { return AGENT_LAYER; }
  cadence() { return CADENCE_1; }
  subscriptions() {
    return [
      new SubscriptionPattern("agent.attenuated"),
      new SubscriptionPattern("agent.attenuation.*"),
      new SubscriptionPattern("agent.budget.exhausted"),
    ];
  }
  process(tick: number, events: Event[], _snapshot: Snapshot): Mutation[] {
    let attenuated = 0;
    let lifted = 0;
    let budgetTriggered = 0;
    for (const ev of events) {
      const t = eventTypeValue(ev);
      if (t === "agent.attenuated") attenuated++;
      else if (t === "agent.attenuation.lifted") lifted++;
      else if (t === "agent.budget.exhausted") budgetTriggered++;
    }
    return [
      { kind: "updateState", primitiveId: this.id(), key: "attenuations", value: attenuated },
      { kind: "updateState", primitiveId: this.id(), key: "lifts", value: lifted },
      { kind: "updateState", primitiveId: this.id(), key: "budgetTriggered", value: budgetTriggered },
      { kind: "updateState", primitiveId: this.id(), key: "lastTick", value: tick },
    ];
  }
}

// ════════════════════════════════════════════════════════════════════════
// REGISTRATION
// ════════════════════════════════════════════════════════════════════════

/** Returns all 28 agent primitives. */
export function allAgentPrimitives(): Primitive[] {
  return [
    // Structural (11)
    new IdentityPrimitive(),
    new SoulPrimitive(),
    new ModelPrimitive(),
    new MemoryPrimitive(),
    new StatePrimitive(),
    new AuthorityPrimitive(),
    new TrustPrimitive(),
    new BudgetPrimitive(),
    new RolePrimitive(),
    new LifespanPrimitive(),
    new GoalPrimitive(),
    // Operational (13)
    new ObservePrimitive(),
    new ProbePrimitive(),
    new EvaluatePrimitive(),
    new DecidePrimitive(),
    new ActPrimitive(),
    new DelegatePrimitive(),
    new EscalatePrimitive(),
    new RefusePrimitive(),
    new LearnPrimitive(),
    new IntrospectPrimitive(),
    new CommunicatePrimitive(),
    new RepairPrimitive(),
    new ExpectPrimitive(),
    // Relational (3)
    new ConsentPrimitive(),
    new ChannelPrimitive(),
    new CompositionPrimitive(),
    // Modal (1)
    new AttenuationPrimitive(),
  ];
}

/** Registers all 28 agent primitives with the given registry and activates them. */
export function registerAllAgentPrimitives(registry: Registry): void {
  for (const p of allAgentPrimitives()) {
    registry.register(p);
    registry.activate(p.id());
  }
}

/** Returns true if the primitive ID belongs to the agent layer. */
export function isAgentPrimitive(id: PrimitiveId): boolean {
  return id.value.startsWith("agent.");
}

// ════════════════════════════════════════════════════════════════════════
// COMPOSITIONS (8)
// ════════════════════════════════════════════════════════════════════════

export interface AgentComposition {
  name: string;
  primitives: string[];
  events: EventType[];
}

/** Boot — Agent comes into existence. */
export function bootComposition(): AgentComposition {
  return {
    name: "Boot",
    primitives: ["agent.Identity", "agent.Soul", "agent.Model", "agent.Authority", "agent.State"],
    events: [
      AgentEventTypeIdentityCreated,
      AgentEventTypeSoulImprinted,
      AgentEventTypeModelBound,
      AgentEventTypeAuthorityGranted,
      AgentEventTypeStateChanged,
    ],
  };
}

/** Imprint — The birth wizard. Boot plus initial context. */
export function imprintComposition(): AgentComposition {
  return {
    name: "Imprint",
    primitives: [
      "agent.Identity", "agent.Soul", "agent.Model", "agent.Authority", "agent.State",
      "agent.Observe", "agent.Learn", "agent.Goal",
    ],
    events: [
      ...bootComposition().events,
      AgentEventTypeObserved,
      AgentEventTypeLearned,
      AgentEventTypeGoalSet,
    ],
  };
}

/** Task — The basic work cycle. */
export function taskComposition(): AgentComposition {
  return {
    name: "Task",
    primitives: ["agent.Observe", "agent.Evaluate", "agent.Decide", "agent.Act", "agent.Learn"],
    events: [
      AgentEventTypeObserved,
      AgentEventTypeEvaluated,
      AgentEventTypeDecided,
      AgentEventTypeActed,
      AgentEventTypeLearned,
    ],
  };
}

/** Supervise — Managing another agent's work. */
export function superviseComposition(): AgentComposition {
  return {
    name: "Supervise",
    primitives: ["agent.Delegate", "agent.Expect", "agent.Observe", "agent.Evaluate", "agent.Repair"],
    events: [
      AgentEventTypeDelegated,
      AgentEventTypeExpectationSet,
      AgentEventTypeObserved,
      AgentEventTypeEvaluated,
    ],
  };
}

/** Collaborate — Agents working together on a shared goal. */
export function collaborateComposition(): AgentComposition {
  return {
    name: "Collaborate",
    primitives: ["agent.Channel", "agent.Communicate", "agent.Consent", "agent.Composition", "agent.Act"],
    events: [
      AgentEventTypeChannelOpened,
      AgentEventTypeCommunicated,
      AgentEventTypeConsentRequested,
      AgentEventTypeConsentGranted,
      AgentEventTypeCompositionFormed,
      AgentEventTypeActed,
    ],
  };
}

/** Crisis — Something is wrong. Detect, assess, attenuate if needed, escalate. */
export function crisisComposition(): AgentComposition {
  return {
    name: "Crisis",
    primitives: ["agent.Observe", "agent.Evaluate", "agent.Attenuation", "agent.Escalate", "agent.Expect"],
    events: [
      AgentEventTypeObserved,
      AgentEventTypeEvaluated,
      AgentEventTypeAttenuated,
      AgentEventTypeEscalated,
      AgentEventTypeExpectationSet,
    ],
  };
}

/** Retire — Graceful shutdown. */
export function retireComposition(): AgentComposition {
  return {
    name: "Retire",
    primitives: ["agent.Introspect", "agent.Communicate", "agent.Memory", "agent.Lifespan"],
    events: [
      AgentEventTypeIntrospected,
      AgentEventTypeCommunicated,
      AgentEventTypeMemoryUpdated,
      AgentEventTypeLifespanEnded,
    ],
  };
}

/** Whistleblow — The agent detects harm and refuses to be complicit. */
export function whistleblowComposition(): AgentComposition {
  return {
    name: "Whistleblow",
    primitives: ["agent.Observe", "agent.Evaluate", "agent.Refuse", "agent.Escalate", "agent.Communicate"],
    events: [
      AgentEventTypeObserved,
      AgentEventTypeEvaluated,
      AgentEventTypeRefused,
      AgentEventTypeEscalated,
      AgentEventTypeCommunicated,
    ],
  };
}

/** Returns all 8 named compositions. */
export function allAgentCompositions(): AgentComposition[] {
  return [
    bootComposition(),
    imprintComposition(),
    taskComposition(),
    superviseComposition(),
    collaborateComposition(),
    crisisComposition(),
    retireComposition(),
    whistleblowComposition(),
  ];
}
