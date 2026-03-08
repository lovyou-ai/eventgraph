//! Layer 12 (Emergence) composition operations.
//!
//! 10 operations + 4 named functions for system self-awareness and evolution.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, EventId};

/// EvolutionGrammar provides Layer 12 (Emergence) composition operations.
pub struct EvolutionGrammar<'a>(Grammar<'a>);

impl<'a> EvolutionGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    pub fn detect_pattern(&mut self, source: ActorId, pattern: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("pattern: {pattern}"), conv_id, causes, signer)
    }

    pub fn model(&mut self, source: ActorId, model: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("model: {model}"), conv_id, causes, signer)
    }

    pub fn trace_loop(&mut self, source: ActorId, loop_desc: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("loop: {loop_desc}"), conv_id, causes, signer)
    }

    pub fn watch_threshold(&mut self, source: ActorId, target: EventId, threshold: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "threshold", threshold, conv_id, signer)
    }

    pub fn adapt(&mut self, source: ActorId, proposal: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("adapt: {proposal}"), conv_id, causes, signer)
    }

    pub fn select(&mut self, source: ActorId, result: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("select: {result}"), conv_id, causes, signer)
    }

    pub fn simplify(&mut self, source: ActorId, simplification: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("simplify: {simplification}"), conv_id, causes, signer)
    }

    pub fn check_integrity(&mut self, source: ActorId, assessment: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("integrity: {assessment}"), conv_id, causes, signer)
    }

    pub fn assess_resilience(&mut self, source: ActorId, assessment: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("resilience: {assessment}"), conv_id, causes, signer)
    }

    pub fn align_purpose(&mut self, source: ActorId, alignment: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("purpose: {alignment}"), conv_id, causes, signer)
    }

    // --- Named Functions ---

    pub fn self_evolve(&mut self, source: ActorId, pattern: &str, adaptation: &str, selection: &str, simplification: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<SelfEvolveResult> {
        let pat = self.detect_pattern(source.clone(), pattern, causes, conv_id.clone(), signer)?;
        let adapt_ev = self.adapt(source.clone(), adaptation, vec![pat.id.clone()], conv_id.clone(), signer)?;
        let sel = self.select(source.clone(), selection, vec![adapt_ev.id.clone()], conv_id.clone(), signer)?;
        let simp = self.simplify(source, simplification, vec![sel.id.clone()], conv_id, signer)?;
        Ok(SelfEvolveResult { pattern: pat, adaptation: adapt_ev, selection: sel, simplification: simp })
    }

    pub fn health_check(&mut self, source: ActorId, integrity: &str, resilience: &str, model_str: &str, purpose: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<HealthCheckResult> {
        let integ = self.check_integrity(source.clone(), integrity, causes, conv_id.clone(), signer)?;
        let resil = self.assess_resilience(source.clone(), resilience, vec![integ.id.clone()], conv_id.clone(), signer)?;
        let mod_ev = self.model(source.clone(), model_str, vec![resil.id.clone()], conv_id.clone(), signer)?;
        let purp = self.align_purpose(source, purpose, vec![mod_ev.id.clone()], conv_id, signer)?;
        Ok(HealthCheckResult { integrity: integ, resilience: resil, model: mod_ev, purpose: purp })
    }

    pub fn prune(&mut self, source: ActorId, unused_pattern: &str, simplification: &str, verification: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<PruneResult> {
        let pattern = self.detect_pattern(source.clone(), &format!("unused: {unused_pattern}"), causes, conv_id.clone(), signer)?;
        let simplify_ev = self.simplify(source.clone(), simplification, vec![pattern.id.clone()], conv_id.clone(), signer)?;
        let verify = self.select(source, verification, vec![simplify_ev.id.clone()], conv_id, signer)?;
        Ok(PruneResult { pattern, simplification: simplify_ev, verification: verify })
    }

    pub fn phase_transition(&mut self, source: ActorId, target: EventId, threshold: &str, model_str: &str, adaptation: &str, selection: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<PhaseTransitionResult> {
        let thresh = self.watch_threshold(source.clone(), target, threshold, conv_id.clone(), signer)?;
        let mod_ev = self.model(source.clone(), model_str, vec![thresh.id.clone()], conv_id.clone(), signer)?;
        let adapt_ev = self.adapt(source.clone(), adaptation, vec![mod_ev.id.clone()], conv_id.clone(), signer)?;
        let sel = self.select(source, selection, vec![adapt_ev.id.clone()], conv_id, signer)?;
        Ok(PhaseTransitionResult { threshold: thresh, model: mod_ev, adaptation: adapt_ev, selection: sel })
    }
}

pub struct SelfEvolveResult { pub pattern: Event, pub adaptation: Event, pub selection: Event, pub simplification: Event }
pub struct HealthCheckResult { pub integrity: Event, pub resilience: Event, pub model: Event, pub purpose: Event }
pub struct PruneResult { pub pattern: Event, pub simplification: Event, pub verification: Event }
pub struct PhaseTransitionResult { pub threshold: Event, pub model: Event, pub adaptation: Event, pub selection: Event }
