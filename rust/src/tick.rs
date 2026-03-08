use std::collections::{HashMap, HashSet};
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
        let mut deferred_mutations: Vec<Mutation> = Vec::new();
        let mut errors: Vec<String> = Vec::new();
        let mut quiesced = false;
        let mut invoked_this_tick: HashSet<String> = HashSet::new();
        let mut waves_run = 0u32;

        // Build initial snapshot
        let mut snapshot = Snapshot {
            tick: tick_num,
            primitives: self.registry.all_states(),
            pending_events: wave_events.clone(),
            recent_events: self.store.recent(100).into_iter().cloned().collect(),
        };

        for wave in 0..self.config.max_waves_per_tick {
            let (wave_mutations, wave_errors) = self.run_wave(
                tick_num, wave, &wave_events, &snapshot, &mut invoked_this_tick,
            );
            errors.extend(wave_errors);
            waves_run = wave + 1;

            if wave_mutations.is_empty() {
                quiesced = true;
                break;
            }

            // Eagerly apply AddEvent mutations; defer the rest to end of tick
            let (new_events, deferred, apply_errors) = self.apply_eager_mutations(wave_mutations);
            errors.extend(apply_errors.iter().map(|e| e.to_string()));

            if apply_errors.is_empty() {
                deferred_mutations.extend(deferred);
            }

            total_mutations += new_events.len();

            if new_events.is_empty() {
                if apply_errors.is_empty() {
                    quiesced = true;
                }
                break;
            }

            wave_events = new_events;

            // Refresh snapshot between waves
            snapshot.pending_events = wave_events.clone();
            snapshot.primitives = self.registry.all_states();
            snapshot.recent_events = self.store.recent(100).into_iter().cloned().collect();
        }

        // Apply deferred (non-AddEvent) mutations at end of tick
        let deferred_count = deferred_mutations.len();
        let mut deferred_errors = 0usize;
        for m in deferred_mutations {
            match self.apply_deferred_mutation(m) {
                Ok(()) => {}
                Err(e) => {
                    errors.push(format!("deferred mutation: {e}"));
                    deferred_errors += 1;
                }
            }
        }
        total_mutations += deferred_count - deferred_errors;

        TickResult {
            tick: tick_num,
            waves: waves_run,
            mutations: total_mutations,
            quiesced,
            duration_ms: start.elapsed().as_secs_f64() * 1000.0,
            errors,
        }
    }

    fn run_wave(
        &mut self,
        tick_num: u64,
        wave: u32,
        events: &[Event],
        snapshot: &Snapshot,
        invoked_this_tick: &mut HashSet<String>,
    ) -> (Vec<Mutation>, Vec<String>) {
        let eligible = self.eligible_primitives(tick_num, snapshot, invoked_this_tick);

        // Group by layer
        let mut by_layer: HashMap<u8, Vec<(PrimitiveId, Vec<SubscriptionPattern>, Cadence)>> = HashMap::new();
        for (pid, subs, cadence, layer) in &eligible {
            by_layer.entry(layer.value()).or_default().push((pid.clone(), subs.clone(), cadence.clone()));
        }

        let mut layers: Vec<u8> = by_layer.keys().copied().collect();
        layers.sort();

        let mut all_mutations: Vec<Mutation> = Vec::new();
        let mut wave_errors: Vec<String> = Vec::new();

        for layer in layers {
            let prims = by_layer.get(&layer).unwrap();

            for (pid, subs, _cadence) in prims {
                let matched: Vec<Event> = events.iter()
                    .filter(|ev| subs.iter().any(|s| s.matches(&ev.event_type)))
                    .cloned()
                    .collect();

                // On subsequent waves, only invoke primitives with matching events
                if matched.is_empty() && invoked_this_tick.contains(pid.value()) {
                    continue;
                }

                if self.registry.set_lifecycle(pid, LifecycleState::Processing).is_err() {
                    continue;
                }

                let mut process_err = None;
                match self.registry.get(pid) {
                    Some(prim) => {
                        let mutations = prim.process(tick_num, &matched, snapshot);
                        all_mutations.extend(mutations);
                    }
                    None => {
                        process_err = Some(format!("{}: primitive not found", pid.value()));
                    }
                }

                // Lifecycle transitions: Processing → Emitting → Active (or Processing → Active)
                if process_err.is_none() {
                    if !all_mutations.is_empty() {
                        if let Err(e) = self.registry.set_lifecycle(pid, LifecycleState::Emitting) {
                            wave_errors.push(format!("{} lifecycle: {e}", pid.value()));
                        } else if let Err(e) = self.registry.set_lifecycle(pid, LifecycleState::Active) {
                            wave_errors.push(format!("{} lifecycle: {e}", pid.value()));
                        }
                    } else {
                        if let Err(e) = self.registry.set_lifecycle(pid, LifecycleState::Active) {
                            wave_errors.push(format!("{} lifecycle: {e}", pid.value()));
                        }
                    }
                    invoked_this_tick.insert(pid.value().to_string());
                    self.registry.set_last_tick(pid, tick_num);
                } else {
                    // Restore to Active on error
                    let _ = self.registry.set_lifecycle(pid, LifecycleState::Active);
                    wave_errors.push(process_err.unwrap());
                }
            }
        }

        (all_mutations, wave_errors)
    }

    fn eligible_primitives(
        &self,
        tick_num: u64,
        snapshot: &Snapshot,
        invoked_this_tick: &HashSet<String>,
    ) -> Vec<(PrimitiveId, Vec<SubscriptionPattern>, Cadence, Layer)> {
        let mut eligible = Vec::new();

        let prim_info: Vec<(PrimitiveId, Vec<SubscriptionPattern>, Cadence, Layer)> = self.registry
            .all()
            .iter()
            .map(|p| (p.id(), p.subscriptions(), p.cadence(), p.layer()))
            .collect();

        for (pid, subs, cadence, layer) in prim_info {
            // Must be Active
            if self.registry.get_lifecycle(&pid) != LifecycleState::Active {
                continue;
            }

            // Cadence gating — only on first invocation per tick
            if !invoked_this_tick.contains(pid.value()) {
                let last = self.registry.get_last_tick(&pid);
                if tick_num - last < cadence.value() as u64 {
                    continue;
                }
            }

            // Layer constraint
            if !layer_stable(&layer, snapshot) {
                continue;
            }

            eligible.push((pid, subs, cadence, layer));
        }

        eligible
    }

    /// Eagerly persist AddEvent mutations between waves.
    /// Non-AddEvent mutations are returned for deferred application at end of tick.
    fn apply_eager_mutations(
        &mut self,
        mutations: Vec<Mutation>,
    ) -> (Vec<Event>, Vec<Mutation>, Vec<String>) {
        let mut new_events = Vec::new();
        let mut deferred = Vec::new();
        let mut errors = Vec::new();

        for m in mutations {
            match m {
                Mutation::AddEvent { event_type, source, content, causes, conversation_id } => {
                    let prev_hash = self.store.head()
                        .map(|e| e.hash.clone())
                        .unwrap_or_else(Hash::zero);
                    let ev = create_event(
                        event_type, source, content, causes,
                        conversation_id, prev_hash, &self.signer, 1,
                    );
                    match self.store.append(ev) {
                        Ok(ev) => new_events.push(ev),
                        Err(e) => errors.push(format!("AddEvent: {e}")),
                    }
                }
                other => {
                    deferred.push(other);
                }
            }
        }

        (new_events, deferred, errors)
    }

    /// Apply a deferred (non-AddEvent) mutation.
    fn apply_deferred_mutation(&mut self, m: Mutation) -> crate::errors::Result<()> {
        match m {
            Mutation::AddEvent { .. } => {
                // Should not happen — AddEvent handled eagerly
                Err(crate::errors::EventGraphError::InvalidFormat {
                    type_name: "Mutation",
                    value: "AddEvent".to_string(),
                    expected: "non-AddEvent mutation in deferred batch",
                })
            }
            Mutation::UpdateState { primitive_id, key, value } => {
                self.registry.update_state(&primitive_id, &key, value)
            }
            Mutation::UpdateActivation { primitive_id, level } => {
                self.registry.set_activation(&primitive_id, level)
            }
            Mutation::UpdateLifecycle { primitive_id, state } => {
                self.registry.set_lifecycle(&primitive_id, state)
            }
        }
    }
}

/// Returns true if all registered Layer N-1 primitives are Active and have been
/// invoked at least once. Vacuously true when no Layer N-1 primitives are registered.
fn layer_stable(layer: &Layer, snapshot: &Snapshot) -> bool {
    if layer.value() == 0 {
        return true; // Layer 0 always eligible
    }

    let target_layer = layer.value() - 1;
    for ps in snapshot.primitives.values() {
        if ps.layer.value() == target_layer {
            if ps.lifecycle != LifecycleState::Active {
                return false;
            }
            if ps.last_tick == 0 {
                return false; // never invoked
            }
        }
    }
    true
}
