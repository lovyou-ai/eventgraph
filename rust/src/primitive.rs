use std::collections::HashMap;
use serde_json::Value;

use crate::errors::EventGraphError;
use crate::event::Event;
use crate::types::*;

// ── Mutations ──────────────────────────────────────────────────────────

#[derive(Debug, Clone)]
pub enum Mutation {
    AddEvent {
        event_type: EventType,
        source: ActorId,
        content: std::collections::BTreeMap<String, Value>,
        causes: Vec<EventId>,
        conversation_id: ConversationId,
    },
    UpdateState {
        primitive_id: PrimitiveId,
        key: String,
        value: Value,
    },
    UpdateActivation {
        primitive_id: PrimitiveId,
        level: Activation,
    },
    UpdateLifecycle {
        primitive_id: PrimitiveId,
        state: LifecycleState,
    },
}

// ── Snapshot ────────────────────────────────────────────────────────────

#[derive(Debug, Clone)]
pub struct PrimitiveState {
    pub id: PrimitiveId,
    pub layer: Layer,
    pub lifecycle: LifecycleState,
    pub activation: Activation,
    pub cadence: Cadence,
    pub state: HashMap<String, Value>,
    pub last_tick: u64,
}

#[derive(Debug, Clone)]
pub struct Snapshot {
    pub tick: u64,
    pub primitives: HashMap<String, PrimitiveState>,
    pub pending_events: Vec<Event>,
    pub recent_events: Vec<Event>,
}

// ── Primitive trait ────────────────────────────────────────────────────

pub trait Primitive {
    fn id(&self) -> PrimitiveId;
    fn layer(&self) -> Layer;
    fn process(&self, tick: u64, events: &[Event], snapshot: &Snapshot) -> Vec<Mutation>;
    fn subscriptions(&self) -> Vec<SubscriptionPattern>;
    fn cadence(&self) -> Cadence;
}

// ── Registry ───────────────────────────────────────────────────────────

struct MutableState {
    activation: Activation,
    lifecycle: LifecycleState,
    state: HashMap<String, Value>,
    last_tick: u64,
}

pub struct Registry {
    primitives: HashMap<String, Box<dyn Primitive>>,
    states: HashMap<String, MutableState>,
    ordered: Vec<String>,
}

impl Registry {
    pub fn new() -> Self {
        Self {
            primitives: HashMap::new(),
            states: HashMap::new(),
            ordered: Vec::new(),
        }
    }

    pub fn register(&mut self, p: Box<dyn Primitive>) -> crate::errors::Result<()> {
        let key = p.id().value().to_string();
        if self.primitives.contains_key(&key) {
            return Err(EventGraphError::InvalidFormat {
                type_name: "Registry",
                value: key,
                expected: "unique primitive ID",
            });
        }
        self.states.insert(key.clone(), MutableState {
            activation: Activation::new(0.0).unwrap(),
            lifecycle: LifecycleState::Dormant,
            state: HashMap::new(),
            last_tick: 0,
        });
        self.primitives.insert(key, p);
        self.rebuild_order();
        Ok(())
    }

    pub fn get(&self, id: &PrimitiveId) -> Option<&dyn Primitive> {
        self.primitives.get(id.value()).map(|b| b.as_ref())
    }

    pub fn all(&self) -> Vec<&dyn Primitive> {
        self.ordered.iter()
            .filter_map(|k| self.primitives.get(k).map(|b| b.as_ref()))
            .collect()
    }

    pub fn count(&self) -> usize { self.primitives.len() }

    pub fn all_states(&self) -> HashMap<String, PrimitiveState> {
        let mut result = HashMap::new();
        for (key, p) in &self.primitives {
            let ms = self.states.get(key).unwrap();
            result.insert(key.clone(), PrimitiveState {
                id: p.id(),
                layer: p.layer(),
                lifecycle: ms.lifecycle,
                activation: ms.activation,
                cadence: p.cadence(),
                state: ms.state.clone(),
                last_tick: ms.last_tick,
            });
        }
        result
    }

    pub fn get_lifecycle(&self, id: &PrimitiveId) -> LifecycleState {
        self.states.get(id.value())
            .map(|s| s.lifecycle)
            .unwrap_or(LifecycleState::Dormant)
    }

    pub fn set_lifecycle(&mut self, id: &PrimitiveId, state: LifecycleState) -> crate::errors::Result<()> {
        let ms = self.states.get_mut(id.value())
            .ok_or_else(|| EventGraphError::EventNotFound { event_id: id.value().to_string() })?;
        if !ms.lifecycle.can_transition_to(state) {
            return Err(EventGraphError::InvalidTransition {
                from: ms.lifecycle.to_string(),
                to: state.to_string(),
            });
        }
        ms.lifecycle = state;
        Ok(())
    }

    pub fn activate(&mut self, id: &PrimitiveId) -> crate::errors::Result<()> {
        let ms = self.states.get_mut(id.value())
            .ok_or_else(|| EventGraphError::EventNotFound { event_id: id.value().to_string() })?;
        if !ms.lifecycle.can_transition_to(LifecycleState::Activating) {
            return Err(EventGraphError::InvalidTransition {
                from: ms.lifecycle.to_string(),
                to: "activating".to_string(),
            });
        }
        ms.lifecycle = LifecycleState::Active;
        Ok(())
    }

    pub fn set_activation(&mut self, id: &PrimitiveId, level: Activation) -> crate::errors::Result<()> {
        let ms = self.states.get_mut(id.value())
            .ok_or_else(|| EventGraphError::EventNotFound { event_id: id.value().to_string() })?;
        ms.activation = level;
        Ok(())
    }

    pub fn update_state(&mut self, id: &PrimitiveId, key: &str, value: Value) -> crate::errors::Result<()> {
        let ms = self.states.get_mut(id.value())
            .ok_or_else(|| EventGraphError::EventNotFound { event_id: id.value().to_string() })?;
        ms.state.insert(key.to_string(), value);
        Ok(())
    }

    pub fn get_last_tick(&self, id: &PrimitiveId) -> u64 {
        self.states.get(id.value()).map(|s| s.last_tick).unwrap_or(0)
    }

    pub fn set_last_tick(&mut self, id: &PrimitiveId, tick: u64) {
        if let Some(ms) = self.states.get_mut(id.value()) {
            ms.last_tick = tick;
        }
    }

    fn rebuild_order(&mut self) {
        let mut keys: Vec<String> = self.primitives.keys().cloned().collect();
        keys.sort_by(|a, b| {
            let la = self.primitives[a].layer().value();
            let lb = self.primitives[b].layer().value();
            la.cmp(&lb).then_with(|| a.cmp(b))
        });
        self.ordered = keys;
    }
}

impl Default for Registry {
    fn default() -> Self { Self::new() }
}
