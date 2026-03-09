// Agent primitives: 28 primitives across 4 categories (structural, operational,
// relational, modal), 8 named compositions, and the OperationalState FSM.
// All agent primitives operate at Layer 1 (Agency).

use std::fmt;

use serde_json::Value;

use crate::event::Event;
use crate::primitive::{Mutation, Primitive, Registry, Snapshot};
use crate::types::*;

// ── Agent Event Type Constants ──────────────────────────────────────────

// Structural events
pub const AGENT_IDENTITY_CREATED: &str = "agent.identity.created";
pub const AGENT_IDENTITY_ROTATED: &str = "agent.identity.rotated";
pub const AGENT_SOUL_IMPRINTED: &str = "agent.soul.imprinted";
pub const AGENT_MODEL_BOUND: &str = "agent.model.bound";
pub const AGENT_MODEL_CHANGED: &str = "agent.model.changed";
pub const AGENT_MEMORY_UPDATED: &str = "agent.memory.updated";
pub const AGENT_STATE_CHANGED: &str = "agent.state.changed";
pub const AGENT_AUTHORITY_GRANTED: &str = "agent.authority.granted";
pub const AGENT_AUTHORITY_REVOKED: &str = "agent.authority.revoked";
pub const AGENT_TRUST_ASSESSED: &str = "agent.trust.assessed";
pub const AGENT_BUDGET_ALLOCATED: &str = "agent.budget.allocated";
pub const AGENT_BUDGET_EXHAUSTED: &str = "agent.budget.exhausted";
pub const AGENT_ROLE_ASSIGNED: &str = "agent.role.assigned";
pub const AGENT_LIFESPAN_STARTED: &str = "agent.lifespan.started";
pub const AGENT_LIFESPAN_EXTENDED: &str = "agent.lifespan.extended";
pub const AGENT_LIFESPAN_ENDED: &str = "agent.lifespan.ended";
pub const AGENT_GOAL_SET: &str = "agent.goal.set";
pub const AGENT_GOAL_COMPLETED: &str = "agent.goal.completed";
pub const AGENT_GOAL_ABANDONED: &str = "agent.goal.abandoned";

// Operational events
pub const AGENT_OBSERVED: &str = "agent.observed";
pub const AGENT_PROBED: &str = "agent.probed";
pub const AGENT_EVALUATED: &str = "agent.evaluated";
pub const AGENT_DECIDED: &str = "agent.decided";
pub const AGENT_ACTED: &str = "agent.acted";
pub const AGENT_DELEGATED: &str = "agent.delegated";
pub const AGENT_ESCALATED: &str = "agent.escalated";
pub const AGENT_REFUSED: &str = "agent.refused";
pub const AGENT_LEARNED: &str = "agent.learned";
pub const AGENT_INTROSPECTED: &str = "agent.introspected";
pub const AGENT_COMMUNICATED: &str = "agent.communicated";
pub const AGENT_REPAIRED: &str = "agent.repaired";
pub const AGENT_EXPECTATION_SET: &str = "agent.expectation.set";
pub const AGENT_EXPECTATION_MET: &str = "agent.expectation.met";
pub const AGENT_EXPECTATION_EXPIRED: &str = "agent.expectation.expired";

// Relational events
pub const AGENT_CONSENT_REQUESTED: &str = "agent.consent.requested";
pub const AGENT_CONSENT_GRANTED: &str = "agent.consent.granted";
pub const AGENT_CONSENT_DENIED: &str = "agent.consent.denied";
pub const AGENT_CHANNEL_OPENED: &str = "agent.channel.opened";
pub const AGENT_CHANNEL_CLOSED: &str = "agent.channel.closed";
pub const AGENT_COMPOSITION_FORMED: &str = "agent.composition.formed";
pub const AGENT_COMPOSITION_DISSOLVED: &str = "agent.composition.dissolved";
pub const AGENT_COMPOSITION_JOINED: &str = "agent.composition.joined";
pub const AGENT_COMPOSITION_LEFT: &str = "agent.composition.left";

// Modal events
pub const AGENT_ATTENUATED: &str = "agent.attenuated";
pub const AGENT_ATTENUATION_LIFTED: &str = "agent.attenuation.lifted";

/// Returns all 45 agent event type strings.
pub fn all_agent_event_types() -> Vec<&'static str> {
    vec![
        // Structural
        AGENT_IDENTITY_CREATED, AGENT_IDENTITY_ROTATED,
        AGENT_SOUL_IMPRINTED,
        AGENT_MODEL_BOUND, AGENT_MODEL_CHANGED,
        AGENT_MEMORY_UPDATED,
        AGENT_STATE_CHANGED,
        AGENT_AUTHORITY_GRANTED, AGENT_AUTHORITY_REVOKED,
        AGENT_TRUST_ASSESSED,
        AGENT_BUDGET_ALLOCATED, AGENT_BUDGET_EXHAUSTED,
        AGENT_ROLE_ASSIGNED,
        AGENT_LIFESPAN_STARTED, AGENT_LIFESPAN_EXTENDED, AGENT_LIFESPAN_ENDED,
        AGENT_GOAL_SET, AGENT_GOAL_COMPLETED, AGENT_GOAL_ABANDONED,
        // Operational
        AGENT_OBSERVED, AGENT_PROBED,
        AGENT_EVALUATED, AGENT_DECIDED,
        AGENT_ACTED, AGENT_DELEGATED,
        AGENT_ESCALATED, AGENT_REFUSED,
        AGENT_LEARNED, AGENT_INTROSPECTED,
        AGENT_COMMUNICATED, AGENT_REPAIRED,
        AGENT_EXPECTATION_SET, AGENT_EXPECTATION_MET, AGENT_EXPECTATION_EXPIRED,
        // Relational
        AGENT_CONSENT_REQUESTED, AGENT_CONSENT_GRANTED, AGENT_CONSENT_DENIED,
        AGENT_CHANNEL_OPENED, AGENT_CHANNEL_CLOSED,
        AGENT_COMPOSITION_FORMED, AGENT_COMPOSITION_DISSOLVED,
        AGENT_COMPOSITION_JOINED, AGENT_COMPOSITION_LEFT,
        // Modal
        AGENT_ATTENUATED, AGENT_ATTENUATION_LIFTED,
    ]
}

// ── OperationalState FSM ────────────────────────────────────────────────

/// Represents the agent's current operational state.
/// Follows a strict FSM with enforced valid transitions.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum OperationalState {
    Idle,
    Processing,
    Waiting,
    Escalating,
    Refusing,
    Suspended,
    Retiring,
    Retired,
}

impl OperationalState {
    /// Returns the valid states this state can transition to.
    fn valid_targets(self) -> &'static [OperationalState] {
        use OperationalState::*;
        match self {
            Idle => &[Processing, Suspended, Retiring],
            Processing => &[Idle, Waiting, Escalating, Refusing, Retiring],
            Waiting => &[Processing, Idle, Retiring],
            Escalating => &[Waiting, Idle],
            Refusing => &[Idle],
            Suspended => &[Idle, Retiring],
            Retiring => &[Retired],
            Retired => &[],
        }
    }

    /// Validates and returns the new state if the transition is valid.
    pub fn transition_to(self, target: OperationalState) -> Result<OperationalState, String> {
        if self.valid_targets().contains(&target) {
            Ok(target)
        } else {
            Err(format!("invalid transition: {} -> {}", self, target))
        }
    }

    /// Returns true if this is a terminal state.
    pub fn is_terminal(self) -> bool {
        self == OperationalState::Retired
    }

    /// Returns true if the agent can perform actions in this state.
    pub fn can_act(self) -> bool {
        self == OperationalState::Processing
    }

    pub fn as_str(self) -> &'static str {
        match self {
            Self::Idle => "Idle",
            Self::Processing => "Processing",
            Self::Waiting => "Waiting",
            Self::Escalating => "Escalating",
            Self::Refusing => "Refusing",
            Self::Suspended => "Suspended",
            Self::Retiring => "Retiring",
            Self::Retired => "Retired",
        }
    }
}

impl fmt::Display for OperationalState {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.as_str())
    }
}

// ── Helper: count events matching a type ────────────────────────────────

fn count_matching(events: &[Event], event_type: &str) -> usize {
    events.iter().filter(|e| e.event_type.value() == event_type).count()
}

fn update_state(id: &str, key: &str, value: Value) -> Mutation {
    Mutation::UpdateState {
        primitive_id: PrimitiveId::new(id).unwrap(),
        key: key.to_string(),
        value,
    }
}

fn int_val(n: usize) -> Value {
    Value::Number(serde_json::Number::from(n))
}

fn tick_val(tick: u64) -> Value {
    Value::Number(serde_json::Number::from(tick))
}

// ══════════════════════════════════════════════════════════════════════════
// STRUCTURAL PRIMITIVES (11) — Define what an agent IS
// ══════════════════════════════════════════════════════════════════════════

struct AgentPrimitiveDef {
    name: &'static str,
    subs: &'static [&'static str],
    process_fn: fn(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation>,
}

/// Identity — ActorID + keys + type + chain of custody.
fn process_identity(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let created = events.iter().filter(|e| {
        let t = e.event_type.value();
        t == AGENT_IDENTITY_CREATED || t == "actor.registered"
    }).count();
    let rotated = count_matching(events, AGENT_IDENTITY_ROTATED);
    vec![
        update_state(id, "identitiesCreated", int_val(created)),
        update_state(id, "keysRotated", int_val(rotated)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Soul — The agent's values and ethical constraints.
fn process_soul(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let imprints = count_matching(events, AGENT_SOUL_IMPRINTED);
    let refusals = count_matching(events, AGENT_REFUSED);
    let mut mutations = vec![update_state(id, "lastTick", tick_val(tick))];
    if imprints > 0 {
        mutations.push(update_state(id, "imprinted", Value::Bool(true)));
    }
    if refusals > 0 {
        mutations.push(update_state(id, "soulRefusals", int_val(refusals)));
    }
    mutations
}

/// Model — The IIntelligence binding.
fn process_model(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let bindings = count_matching(events, AGENT_MODEL_BOUND);
    let changes = count_matching(events, AGENT_MODEL_CHANGED);
    vec![
        update_state(id, "bindings", int_val(bindings)),
        update_state(id, "modelChanges", int_val(changes)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Memory — Persistent state across ticks.
fn process_memory(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let updates = events.iter().filter(|e| {
        let t = e.event_type.value();
        t == AGENT_MEMORY_UPDATED || t == AGENT_LEARNED
    }).count();
    vec![
        update_state(id, "memoryUpdates", int_val(updates)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// State — Current operational state FSM.
fn process_state(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let transitions = count_matching(events, AGENT_STATE_CHANGED);
    let mut mutations = vec![
        update_state(id, "transitions", int_val(transitions)),
        update_state(id, "lastTick", tick_val(tick)),
    ];
    if transitions > 0 {
        mutations.push(update_state(id, "lastTransition", Value::String("changed".to_string())));
    }
    mutations
}

/// Authority — What this agent is permitted to do.
fn process_authority(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let granted = count_matching(events, AGENT_AUTHORITY_GRANTED);
    let revoked = count_matching(events, AGENT_AUTHORITY_REVOKED);
    vec![
        update_state(id, "authorityGrants", int_val(granted)),
        update_state(id, "authorityRevocations", int_val(revoked)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Trust — Trust scores toward other actors.
fn process_trust(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let assessments = count_matching(events, AGENT_TRUST_ASSESSED);
    vec![
        update_state(id, "trustAssessments", int_val(assessments)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Budget — Resource constraints.
fn process_budget(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let allocated = count_matching(events, AGENT_BUDGET_ALLOCATED);
    let exhausted = count_matching(events, AGENT_BUDGET_EXHAUSTED);
    vec![
        update_state(id, "allocations", int_val(allocated)),
        update_state(id, "exhaustions", int_val(exhausted)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Role — Named function within a team.
fn process_role(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let assignments = count_matching(events, AGENT_ROLE_ASSIGNED);
    vec![
        update_state(id, "roleAssignments", int_val(assignments)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Lifespan — Birth, expected duration, graceful shutdown conditions.
fn process_lifespan(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let started = count_matching(events, AGENT_LIFESPAN_STARTED);
    let ended = count_matching(events, AGENT_LIFESPAN_ENDED);
    vec![
        update_state(id, "agentsStarted", int_val(started)),
        update_state(id, "agentsEnded", int_val(ended)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Goal — Current objective hierarchy.
fn process_goal(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let set = count_matching(events, AGENT_GOAL_SET);
    let completed = count_matching(events, AGENT_GOAL_COMPLETED);
    let abandoned = count_matching(events, AGENT_GOAL_ABANDONED);
    vec![
        update_state(id, "goalsSet", int_val(set)),
        update_state(id, "goalsCompleted", int_val(completed)),
        update_state(id, "goalsAbandoned", int_val(abandoned)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

// ══════════════════════════════════════════════════════════════════════════
// OPERATIONAL PRIMITIVES (13) — Define what an agent DOES
// ══════════════════════════════════════════════════════════════════════════

/// Observe — Passive perception.
fn process_observe(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let observed = count_matching(events, AGENT_OBSERVED);
    vec![
        update_state(id, "eventsObserved", int_val(observed)),
        update_state(id, "totalEventsReceived", int_val(events.len())),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Probe — Active perception.
fn process_probe(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let probes = count_matching(events, AGENT_PROBED);
    vec![
        update_state(id, "probesExecuted", int_val(probes)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Evaluate — One-shot judgment.
fn process_evaluate(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let evaluations = count_matching(events, AGENT_EVALUATED);
    vec![
        update_state(id, "evaluations", int_val(evaluations)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Decide — Commit to an action.
fn process_decide(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let decisions = count_matching(events, AGENT_DECIDED);
    vec![
        update_state(id, "decisions", int_val(decisions)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Act — Execute a decision.
fn process_act(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let actions = count_matching(events, AGENT_ACTED);
    vec![
        update_state(id, "actionsExecuted", int_val(actions)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Delegate — Assign work to another agent.
fn process_delegate(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let delegations = count_matching(events, AGENT_DELEGATED);
    vec![
        update_state(id, "delegations", int_val(delegations)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Escalate — Pass upward.
fn process_escalate(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let escalations = count_matching(events, AGENT_ESCALATED);
    vec![
        update_state(id, "escalations", int_val(escalations)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Refuse — Decline to act.
fn process_refuse(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let refusals = count_matching(events, AGENT_REFUSED);
    vec![
        update_state(id, "refusals", int_val(refusals)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Learn — Update Memory based on outcomes.
fn process_learn(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let lessons = count_matching(events, AGENT_LEARNED);
    vec![
        update_state(id, "lessonsLearned", int_val(lessons)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Introspect — Read own State and Soul.
fn process_introspect(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let introspections = count_matching(events, AGENT_INTROSPECTED);
    vec![
        update_state(id, "introspections", int_val(introspections)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Communicate — Send a message to another agent or channel.
fn process_communicate(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let messages = count_matching(events, AGENT_COMMUNICATED);
    vec![
        update_state(id, "messagesSent", int_val(messages)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Repair — Fix a prior Act.
fn process_repair(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let repairs = count_matching(events, AGENT_REPAIRED);
    vec![
        update_state(id, "repairs", int_val(repairs)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Expect — Create a persistent monitoring condition.
fn process_expect(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let set = count_matching(events, AGENT_EXPECTATION_SET);
    let met = count_matching(events, AGENT_EXPECTATION_MET);
    let expired = count_matching(events, AGENT_EXPECTATION_EXPIRED);
    vec![
        update_state(id, "expectationsSet", int_val(set)),
        update_state(id, "expectationsMet", int_val(met)),
        update_state(id, "expectationsExpired", int_val(expired)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

// ══════════════════════════════════════════════════════════════════════════
// RELATIONAL PRIMITIVES (3) — Define how agents relate
// ══════════════════════════════════════════════════════════════════════════

/// Consent — Bilateral agreement.
fn process_consent(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let requested = count_matching(events, AGENT_CONSENT_REQUESTED);
    let granted = count_matching(events, AGENT_CONSENT_GRANTED);
    let denied = count_matching(events, AGENT_CONSENT_DENIED);
    vec![
        update_state(id, "consentRequested", int_val(requested)),
        update_state(id, "consentGranted", int_val(granted)),
        update_state(id, "consentDenied", int_val(denied)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Channel — Persistent bidirectional communication link.
fn process_channel(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let opened = count_matching(events, AGENT_CHANNEL_OPENED);
    let closed = count_matching(events, AGENT_CHANNEL_CLOSED);
    vec![
        update_state(id, "channelsOpened", int_val(opened)),
        update_state(id, "channelsClosed", int_val(closed)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

/// Composition — Form a group.
fn process_composition(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let formed = count_matching(events, AGENT_COMPOSITION_FORMED);
    let dissolved = count_matching(events, AGENT_COMPOSITION_DISSOLVED);
    let joined = count_matching(events, AGENT_COMPOSITION_JOINED);
    let left = count_matching(events, AGENT_COMPOSITION_LEFT);
    vec![
        update_state(id, "groupsFormed", int_val(formed)),
        update_state(id, "groupsDissolved", int_val(dissolved)),
        update_state(id, "membersJoined", int_val(joined)),
        update_state(id, "membersLeft", int_val(left)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

// ══════════════════════════════════════════════════════════════════════════
// MODAL PRIMITIVE (1) — Modifies how other primitives operate
// ══════════════════════════════════════════════════════════════════════════

/// Attenuation — Reduce scope, confidence, or authority.
fn process_attenuation(id: &str, tick: u64, events: &[Event]) -> Vec<Mutation> {
    let attenuated = count_matching(events, AGENT_ATTENUATED);
    let lifted = count_matching(events, AGENT_ATTENUATION_LIFTED);
    let budget_triggered = count_matching(events, AGENT_BUDGET_EXHAUSTED);
    vec![
        update_state(id, "attenuations", int_val(attenuated)),
        update_state(id, "lifts", int_val(lifted)),
        update_state(id, "budgetTriggered", int_val(budget_triggered)),
        update_state(id, "lastTick", tick_val(tick)),
    ]
}

// ── Agent Primitive wrapper ─────────────────────────────────────────────

struct AgentPrimitive {
    def: &'static AgentPrimitiveDef,
}

impl Primitive for AgentPrimitive {
    fn id(&self) -> PrimitiveId {
        PrimitiveId::new(self.def.name).unwrap()
    }

    fn layer(&self) -> Layer {
        Layer::new(1).unwrap()
    }

    fn subscriptions(&self) -> Vec<SubscriptionPattern> {
        self.def.subs.iter()
            .map(|s| SubscriptionPattern::new(*s).unwrap())
            .collect()
    }

    fn cadence(&self) -> Cadence {
        Cadence::new(1).unwrap()
    }

    fn process(&self, tick: u64, events: &[Event], _snapshot: &Snapshot) -> Vec<Mutation> {
        (self.def.process_fn)(self.def.name, tick, events)
    }
}

// ── Static definitions for all 28 primitives ────────────────────────────

static PRIMITIVE_DEFS: &[AgentPrimitiveDef] = &[
    // Structural (11)
    AgentPrimitiveDef { name: "agent.Identity",    subs: &["agent.identity.*", "actor.registered"], process_fn: process_identity },
    AgentPrimitiveDef { name: "agent.Soul",        subs: &["agent.soul.*", "agent.refused"],        process_fn: process_soul },
    AgentPrimitiveDef { name: "agent.Model",       subs: &["agent.model.*"],                        process_fn: process_model },
    AgentPrimitiveDef { name: "agent.Memory",      subs: &["agent.memory.*", "agent.learned"],      process_fn: process_memory },
    AgentPrimitiveDef { name: "agent.State",       subs: &["agent.state.*"],                        process_fn: process_state },
    AgentPrimitiveDef { name: "agent.Authority",   subs: &["agent.authority.*", "authority.*"],      process_fn: process_authority },
    AgentPrimitiveDef { name: "agent.Trust",       subs: &["agent.trust.*", "trust.*"],              process_fn: process_trust },
    AgentPrimitiveDef { name: "agent.Budget",      subs: &["agent.budget.*"],                        process_fn: process_budget },
    AgentPrimitiveDef { name: "agent.Role",        subs: &["agent.role.*"],                          process_fn: process_role },
    AgentPrimitiveDef { name: "agent.Lifespan",    subs: &["agent.lifespan.*"],                      process_fn: process_lifespan },
    AgentPrimitiveDef { name: "agent.Goal",        subs: &["agent.goal.*"],                          process_fn: process_goal },
    // Operational (13)
    AgentPrimitiveDef { name: "agent.Observe",     subs: &["agent.observed", "agent.*"],             process_fn: process_observe },
    AgentPrimitiveDef { name: "agent.Probe",       subs: &["agent.probed"],                          process_fn: process_probe },
    AgentPrimitiveDef { name: "agent.Evaluate",    subs: &["agent.evaluated"],                       process_fn: process_evaluate },
    AgentPrimitiveDef { name: "agent.Decide",      subs: &["agent.decided", "agent.evaluated"],      process_fn: process_decide },
    AgentPrimitiveDef { name: "agent.Act",         subs: &["agent.acted", "agent.decided"],          process_fn: process_act },
    AgentPrimitiveDef { name: "agent.Delegate",    subs: &["agent.delegated"],                       process_fn: process_delegate },
    AgentPrimitiveDef { name: "agent.Escalate",    subs: &["agent.escalated"],                       process_fn: process_escalate },
    AgentPrimitiveDef { name: "agent.Refuse",      subs: &["agent.refused"],                         process_fn: process_refuse },
    AgentPrimitiveDef { name: "agent.Learn",       subs: &["agent.learned", "agent.goal.completed", "agent.goal.abandoned"], process_fn: process_learn },
    AgentPrimitiveDef { name: "agent.Introspect",  subs: &["agent.introspected"],                    process_fn: process_introspect },
    AgentPrimitiveDef { name: "agent.Communicate", subs: &["agent.communicated", "agent.channel.*"], process_fn: process_communicate },
    AgentPrimitiveDef { name: "agent.Repair",      subs: &["agent.repaired"],                        process_fn: process_repair },
    AgentPrimitiveDef { name: "agent.Expect",      subs: &["agent.expectation.*"],                   process_fn: process_expect },
    // Relational (3)
    AgentPrimitiveDef { name: "agent.Consent",     subs: &["agent.consent.*"],                       process_fn: process_consent },
    AgentPrimitiveDef { name: "agent.Channel",     subs: &["agent.channel.*"],                       process_fn: process_channel },
    AgentPrimitiveDef { name: "agent.Composition", subs: &["agent.composition.*"],                   process_fn: process_composition },
    // Modal (1)
    AgentPrimitiveDef { name: "agent.Attenuation", subs: &["agent.attenuated", "agent.attenuation.*", "agent.budget.exhausted"], process_fn: process_attenuation },
];

// ── Public API ──────────────────────────────────────────────────────────

/// Returns all 28 agent primitives as boxed trait objects.
pub fn all_primitives() -> Vec<Box<dyn Primitive>> {
    PRIMITIVE_DEFS.iter()
        .map(|def| Box::new(AgentPrimitive { def }) as Box<dyn Primitive>)
        .collect()
}

/// Registers all 28 agent primitives with the given registry and activates them.
pub fn register_all(registry: &mut Registry) -> crate::errors::Result<()> {
    for p in all_primitives() {
        let id = p.id();
        registry.register(p)?;
        registry.activate(&id)?;
    }
    Ok(())
}

/// Returns true if the primitive ID belongs to the agent layer.
pub fn is_agent_primitive(id: &PrimitiveId) -> bool {
    id.value().starts_with("agent.")
}

// ══════════════════════════════════════════════════════════════════════════
// COMPOSITIONS (8) — Named sequences of agent primitives
// ══════════════════════════════════════════════════════════════════════════

/// Represents a named sequence of agent primitive operations.
#[derive(Debug, Clone)]
pub struct Composition {
    pub name: &'static str,
    pub primitives: Vec<&'static str>,
    pub events: Vec<&'static str>,
}

/// Boot — Agent comes into existence.
/// Identity(generate) + Soul(load) + Model(bind) + Authority(receive) + State(set:idle)
pub fn boot() -> Composition {
    Composition {
        name: "Boot",
        primitives: vec![
            "agent.Identity", "agent.Soul", "agent.Model", "agent.Authority", "agent.State",
        ],
        events: vec![
            AGENT_IDENTITY_CREATED, AGENT_SOUL_IMPRINTED, AGENT_MODEL_BOUND,
            AGENT_AUTHORITY_GRANTED, AGENT_STATE_CHANGED,
        ],
    }
}

/// Imprint — The birth wizard. Boot plus initial context.
/// Boot + Observe(first_message) + Learn(initial_context) + Goal(set)
pub fn imprint() -> Composition {
    let mut events = boot().events;
    events.extend_from_slice(&[AGENT_OBSERVED, AGENT_LEARNED, AGENT_GOAL_SET]);
    Composition {
        name: "Imprint",
        primitives: vec![
            "agent.Identity", "agent.Soul", "agent.Model", "agent.Authority", "agent.State",
            "agent.Observe", "agent.Learn", "agent.Goal",
        ],
        events,
    }
}

/// Task — The basic work cycle.
/// Observe(assignment) + Evaluate(scope) + Decide(accept_or_refuse) + Act(execute) + Learn(outcome)
pub fn task() -> Composition {
    Composition {
        name: "Task",
        primitives: vec![
            "agent.Observe", "agent.Evaluate", "agent.Decide", "agent.Act", "agent.Learn",
        ],
        events: vec![
            AGENT_OBSERVED, AGENT_EVALUATED, AGENT_DECIDED, AGENT_ACTED, AGENT_LEARNED,
        ],
    }
}

/// Supervise — Managing another agent's work.
/// Delegate(task) + Expect(completion) + Observe(progress) + Evaluate(quality) + Repair(if_needed)
pub fn supervise() -> Composition {
    Composition {
        name: "Supervise",
        primitives: vec![
            "agent.Delegate", "agent.Expect", "agent.Observe", "agent.Evaluate", "agent.Repair",
        ],
        events: vec![
            AGENT_DELEGATED, AGENT_EXPECTATION_SET, AGENT_OBSERVED, AGENT_EVALUATED,
        ],
    }
}

/// Collaborate — Agents working together on a shared goal.
/// Channel(open) + Communicate(proposal) + Consent(terms) + Composition(form) + Act(jointly)
pub fn collaborate() -> Composition {
    Composition {
        name: "Collaborate",
        primitives: vec![
            "agent.Channel", "agent.Communicate", "agent.Consent", "agent.Composition", "agent.Act",
        ],
        events: vec![
            AGENT_CHANNEL_OPENED, AGENT_COMMUNICATED,
            AGENT_CONSENT_REQUESTED, AGENT_CONSENT_GRANTED,
            AGENT_COMPOSITION_FORMED, AGENT_ACTED,
        ],
    }
}

/// Crisis — Something is wrong. Detect, assess, attenuate if needed, escalate.
/// Observe(anomaly) + Evaluate(severity) + Attenuation(if_needed) + Escalate(to_authority) + Expect(resolution)
pub fn crisis() -> Composition {
    Composition {
        name: "Crisis",
        primitives: vec![
            "agent.Observe", "agent.Evaluate", "agent.Attenuation", "agent.Escalate", "agent.Expect",
        ],
        events: vec![
            AGENT_OBSERVED, AGENT_EVALUATED, AGENT_ATTENUATED,
            AGENT_ESCALATED, AGENT_EXPECTATION_SET,
        ],
    }
}

/// Retire — Graceful shutdown.
/// Introspect(final) + Communicate(farewell) + Memory(archive) + Lifespan(close)
pub fn retire() -> Composition {
    Composition {
        name: "Retire",
        primitives: vec![
            "agent.Introspect", "agent.Communicate", "agent.Memory", "agent.Lifespan",
        ],
        events: vec![
            AGENT_INTROSPECTED, AGENT_COMMUNICATED, AGENT_MEMORY_UPDATED, AGENT_LIFESPAN_ENDED,
        ],
    }
}

/// Whistleblow — The agent detects harm and refuses to be complicit.
/// Cannot be silenced by authority. Protected by the Soul primitive.
/// Observe(harm) + Evaluate(severity) + Refuse(complicity) + Escalate(with_evidence) + Communicate(public)
pub fn whistleblow() -> Composition {
    Composition {
        name: "Whistleblow",
        primitives: vec![
            "agent.Observe", "agent.Evaluate", "agent.Refuse", "agent.Escalate", "agent.Communicate",
        ],
        events: vec![
            AGENT_OBSERVED, AGENT_EVALUATED, AGENT_REFUSED,
            AGENT_ESCALATED, AGENT_COMMUNICATED,
        ],
    }
}

/// Returns all 8 named compositions.
pub fn all_compositions() -> Vec<Composition> {
    vec![
        boot(), imprint(), task(), supervise(),
        collaborate(), crisis(), retire(), whistleblow(),
    ]
}
