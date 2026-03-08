//! Layer 6 (Information) composition operations.
//!
//! 12 operations + 6 named functions for verified, provenanced knowledge.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, EventId};

/// KnowledgeGrammar provides Layer 6 (Information) composition operations.
pub struct KnowledgeGrammar<'a>(Grammar<'a>);

impl<'a> KnowledgeGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    pub fn claim(&mut self, source: ActorId, claim: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("claim: {claim}"), conv_id, causes, signer)
    }

    pub fn categorize(&mut self, source: ActorId, target: EventId, taxonomy: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "classification", taxonomy, conv_id, signer)
    }

    pub fn abstract_merge(&mut self, source: ActorId, generalization: &str, instances: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        if instances.len() < 2 {
            return Err(crate::errors::EventGraphError::GrammarViolation { detail: "abstract: requires at least two instances".to_string() });
        }
        self.0.merge(source, &format!("abstract: {generalization}"), instances, conv_id, signer)
    }

    pub fn encode(&mut self, source: ActorId, encoding: &str, original: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("encode: {encoding}"), original, conv_id, signer)
    }

    pub fn infer(&mut self, source: ActorId, conclusion: &str, premise: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("infer: {conclusion}"), premise, conv_id, signer)
    }

    pub fn remember(&mut self, source: ActorId, knowledge: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("remember: {knowledge}"), conv_id, causes, signer)
    }

    pub fn challenge(&mut self, source: ActorId, counter_evidence: &str, claim_id: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        let (_, flag) = self.0.challenge(source, &format!("challenge: {counter_evidence}"), claim_id, conv_id, signer)?;
        Ok(flag)
    }

    pub fn detect_bias(&mut self, source: ActorId, target: EventId, bias: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "bias", bias, conv_id, signer)
    }

    pub fn correct(&mut self, source: ActorId, correction: &str, original: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("correct: {correction}"), original, conv_id, signer)
    }

    pub fn trace(&mut self, source: ActorId, target: EventId, provenance: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "provenance", provenance, conv_id, signer)
    }

    pub fn recall(&mut self, source: ActorId, query: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("recall: {query}"), conv_id, causes, signer)
    }

    pub fn learn(&mut self, source: ActorId, learning: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("learn: {learning}"), conv_id, causes, signer)
    }

    // --- Named Functions ---

    pub fn retract(&mut self, source: ActorId, claim_id: EventId, reason: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.retract(source, claim_id, reason, conv_id, signer)
    }

    pub fn fact_check(&mut self, checker: ActorId, claim_id: EventId, provenance: &str, bias_analysis: &str, verdict: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<FactCheckResult> {
        let trace_ev = self.trace(checker.clone(), claim_id.clone(), provenance, conv_id.clone(), signer)?;
        let bias = self.detect_bias(checker.clone(), claim_id, bias_analysis, conv_id.clone(), signer)?;
        let verdict_ev = self.0.merge(checker, &format!("fact-check: {verdict}"), vec![trace_ev.id.clone(), bias.id.clone()], conv_id, signer)?;
        Ok(FactCheckResult { provenance: trace_ev, bias_check: bias, verdict: verdict_ev })
    }

    pub fn verify(&mut self, source: ActorId, claim_str: &str, provenance: &str, corroboration: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<VerifyResult> {
        let claim_ev = self.claim(source.clone(), claim_str, causes, conv_id.clone(), signer)?;
        let trace_ev = self.trace(source.clone(), claim_ev.id.clone(), provenance, conv_id.clone(), signer)?;
        let corroborate = self.claim(source, &format!("corroborate: {corroboration}"), vec![trace_ev.id.clone()], conv_id, signer)?;
        Ok(VerifyResult { claim: claim_ev, provenance: trace_ev, corroboration: corroborate })
    }

    pub fn survey(&mut self, source: ActorId, queries: &[&str], generalization: &str, synthesis: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<SurveyResult> {
        if queries.len() < 2 {
            return Err(crate::errors::EventGraphError::GrammarViolation { detail: "survey: requires at least two queries".to_string() });
        }
        let mut recalls = Vec::new();
        let mut recall_ids = Vec::new();
        for query in queries {
            let r = self.recall(source.clone(), query, causes.clone(), conv_id.clone(), signer)?;
            recall_ids.push(r.id.clone());
            recalls.push(r);
        }
        let abstract_ev = self.abstract_merge(source.clone(), generalization, recall_ids, conv_id.clone(), signer)?;
        let synthesis_claim = self.claim(source, &format!("synthesis: {synthesis}"), vec![abstract_ev.id.clone()], conv_id, signer)?;
        Ok(SurveyResult { recalls, abstraction: abstract_ev, synthesis: synthesis_claim })
    }

    pub fn knowledge_base(&mut self, source: ActorId, claims: &[&str], taxonomies: &[&str], memory_label: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<KnowledgeBaseResult> {
        if claims.len() != taxonomies.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation { detail: "knowledge-base: claims and taxonomies must have equal length".to_string() });
        }
        let mut claim_events = Vec::new();
        let mut categories = Vec::new();
        let mut claim_ids = Vec::new();
        for (i, c) in claims.iter().enumerate() {
            let claim_ev = self.claim(source.clone(), c, causes.clone(), conv_id.clone(), signer)?;
            let cat = self.categorize(source.clone(), claim_ev.id.clone(), taxonomies[i], conv_id.clone(), signer)?;
            claim_ids.push(cat.id.clone());
            claim_events.push(claim_ev);
            categories.push(cat);
        }
        let memory = self.remember(source, memory_label, claim_ids, conv_id, signer)?;
        Ok(KnowledgeBaseResult { claims: claim_events, categories, memory })
    }

    pub fn transfer(&mut self, source: ActorId, query: &str, encoding: &str, learning: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<TransferResult> {
        let recall_ev = self.recall(source.clone(), query, causes, conv_id.clone(), signer)?;
        let encode_ev = self.encode(source.clone(), encoding, recall_ev.id.clone(), conv_id.clone(), signer)?;
        let learn_ev = self.learn(source, learning, vec![encode_ev.id.clone()], conv_id, signer)?;
        Ok(TransferResult { recall: recall_ev, encode: encode_ev, learn: learn_ev })
    }
}

pub struct FactCheckResult { pub provenance: Event, pub bias_check: Event, pub verdict: Event }
pub struct VerifyResult { pub claim: Event, pub provenance: Event, pub corroboration: Event }
pub struct SurveyResult { pub recalls: Vec<Event>, pub abstraction: Event, pub synthesis: Event }
pub struct KnowledgeBaseResult { pub claims: Vec<Event>, pub categories: Vec<Event>, pub memory: Event }
pub struct TransferResult { pub recall: Event, pub encode: Event, pub learn: Event }
