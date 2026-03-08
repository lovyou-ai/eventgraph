use std::collections::HashSet;
use std::time::Instant;

use crate::event::{Event, NoopSigner, create_event};
use crate::primitive::{Mutation, Registry, Snapshot};
use crate::store::{InMemoryStore, Store};
use crate::types::*;

pub struct TickConfig {
    pub max_waves_per_tick: u32,
}

impl Default for TickConfig {
    fn default() -> Self { Self { max_waves_per_tick: 10 } }
}

pub struct TickResult {
    pub tick: u64,
    pub waves: u32,
    pub mutations: usize,
    pub quiesced: bool,
    pub duration_ms: f64,
    pub errors: Vec<String>,
}

pub struct TickEngine {
    registry: Registry,
    store: InMemoryStore,
    config: TickConfig,
    signer: NoopSigner,
    current_tick: u64,
}

impl TickEngine {
    pub fn new(registry: Registry, store: InMemoryStore, config: Option<TickConfig>) -> Self {
        Self {
            registry,
            store,
            config: config.unwrap_or_default(),
            signer: NoopSigner,
            current_tick: 0,
        }
    }

    pub fn registry(&self) -> &Registry { &self.registry }
    pub fn registry_mut(&mut self) -> &mut Registry { &mut self.registry }
    pub fn store(&self) -> &InMemoryStore { &self.store }

    pub fn tick(&mut self, pending_events: Option<Vec<Event>>) -> TickResult {
        let start = Instant::now();
        self.current_tick += 1;
        let tick_num = self.current_tick;

        let mut wave_events = pending_events.unwrap_or_default();
        let mut total_mutations = 0usize;
        let mut errors: Vec<String> = Vec::new();
        let mut quiesced = false;
        let mut invoked_this_tick: HashSet<String> = HashSet::new();
        let mut waves_run = 0u32;

        for wave in 0..self.config.max_waves_per_tick {
            if wave_events.is_empty() && wave > 0 {
                quiesced = true;
                break;
            }
            waves_run = wave + 1;

            let snapshot = Snapshot {
                tick: tick_num,
                primitives: self.registry.all_states(),
                pending_events: wave_events.clone(),
                recent_events: self.store.recent(50).into_iter().cloned().collect(),
            };

            let mut new_events: Vec<Event> = Vec::new();
            let mut all_mutations: Vec<Mutation> = Vec::new();

            // Collect primitive info before processing (to avoid borrow issues)
            let prim_info: Vec<(PrimitiveId, Vec<SubscriptionPattern>, Cadence)> = self.registry
                .all()
                .iter()
                .map(|p| (p.id(), p.subscriptions(), p.cadence()))
                .collect();

            for (pid, subs, cadence) in &prim_info {
                let lifecycle = self.registry.get_lifecycle(pid);
                if lifecycle != LifecycleState::Active { continue; }

                if !invoked_this_tick.contains(pid.value()) {
                    let last = self.registry.get_last_tick(pid);
                    if tick_num - last < cadence.value() as u64 { continue; }
                }

                let matched: Vec<Event> = wave_events.iter()
                    .filter(|ev| subs.iter().any(|s| s.matches(&ev.event_type)))
                    .cloned()
                    .collect();

                if matched.is_empty() && wave > 0 { continue; }

                if self.registry.set_lifecycle(pid, LifecycleState::Processing).is_err() {
                    continue;
                }

                match self.registry.get(pid) {
                    Some(prim) => {
                        let mutations = prim.process(tick_num, &matched, &snapshot);
                        all_mutations.extend(mutations);
                    }
                    None => {
                        errors.push(format!("{}: primitive not found", pid.value()));
                    }
                }

                if let Err(e) = self.registry.set_lifecycle(pid, LifecycleState::Emitting) {
                    errors.push(format!("{} lifecycle: {e}", pid.value()));
                } else if let Err(e) = self.registry.set_lifecycle(pid, LifecycleState::Active) {
                    errors.push(format!("{} lifecycle: {e}", pid.value()));
                }

                invoked_this_tick.insert(pid.value().to_string());
            }

            for m in all_mutations {
                total_mutations += 1;
                match self.apply_mutation(m) {
                    Ok(Some(ev)) => new_events.push(ev),
                    Ok(None) => {}
                    Err(e) => errors.push(format!("mutation: {e}")),
                }
            }

            wave_events = new_events;
        }

        for pid_val in &invoked_this_tick {
            let pid = PrimitiveId::new(pid_val.clone()).unwrap();
            self.registry.set_last_tick(&pid, tick_num);
        }

        TickResult {
            tick: tick_num,
            waves: waves_run,
            mutations: total_mutations,
            quiesced,
            duration_ms: start.elapsed().as_secs_f64() * 1000.0,
            errors,
        }
    }

    fn apply_mutation(&mut self, m: Mutation) -> crate::errors::Result<Option<Event>> {
        match m {
            Mutation::AddEvent { event_type, source, content, causes, conversation_id } => {
                let prev_hash = self.store.head()
                    .map(|e| e.hash.clone())
                    .unwrap_or_else(Hash::zero);
                let ev = create_event(
                    event_type, source, content, causes,
                    conversation_id, prev_hash, &self.signer, 1,
                );
                let ev = self.store.append(ev)?;
                Ok(Some(ev))
            }
            Mutation::UpdateState { primitive_id, key, value } => {
                self.registry.update_state(&primitive_id, &key, value)?;
                Ok(None)
            }
            Mutation::UpdateActivation { primitive_id, level } => {
                self.registry.set_activation(&primitive_id, level)?;
                Ok(None)
            }
            Mutation::UpdateLifecycle { primitive_id, state } => {
                self.registry.set_lifecycle(&primitive_id, state)?;
                Ok(None)
            }
        }
    }
}
