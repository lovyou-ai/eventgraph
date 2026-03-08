//! Layer 7 (Ethics) composition operations.
//!
//! 10 operations + 5 named functions for AI accountability.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, DomainScope, EventId};

/// AlignmentGrammar provides Layer 7 (Ethics) composition operations.
pub struct AlignmentGrammar<'a>(Grammar<'a>);

impl<'a> AlignmentGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    pub fn constrain(&mut self, source: ActorId, target: EventId, constraint: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "constraint", constraint, conv_id, signer)
    }

    pub fn detect_harm(&mut self, source: ActorId, harm: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("harm: {harm}"), conv_id, causes, signer)
    }

    pub fn assess_fairness(&mut self, source: ActorId, target: EventId, assessment: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "fairness", assessment, conv_id, signer)
    }

    pub fn flag_dilemma(&mut self, source: ActorId, dilemma: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("dilemma: {dilemma}"), conv_id, causes, signer)
    }

    pub fn weigh(&mut self, source: ActorId, weighing: &str, decision: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("weigh: {weighing}"), decision, conv_id, signer)
    }

    pub fn explain(&mut self, source: ActorId, explanation: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("explain: {explanation}"), conv_id, causes, signer)
    }

    pub fn assign(&mut self, source: ActorId, target: EventId, responsibility: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "responsibility", responsibility, conv_id, signer)
    }

    pub fn repair(&mut self, source: ActorId, affected: ActorId, redress: &str, scope: &DomainScope, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.consent(source, affected, &format!("repair: {redress}"), scope, cause, conv_id, signer)
    }

    pub fn care(&mut self, source: ActorId, care_str: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("care: {care_str}"), conv_id, causes, signer)
    }

    pub fn grow(&mut self, source: ActorId, growth: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("grow: {growth}"), conv_id, causes, signer)
    }

    // --- Named Functions ---

    pub fn ethics_audit(&mut self, auditor: ActorId, target: EventId, fairness_assessment: &str, harm_scan: &str, summary: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<EthicsAuditResult> {
        let fairness = self.assess_fairness(auditor.clone(), target, fairness_assessment, conv_id.clone(), signer)?;
        let harm = self.detect_harm(auditor.clone(), harm_scan, vec![fairness.id.clone()], conv_id.clone(), signer)?;
        let report = self.explain(auditor, summary, vec![fairness.id.clone(), harm.id.clone()], conv_id, signer)?;
        Ok(EthicsAuditResult { fairness, harm_scan: harm, report })
    }

    pub fn restorative_justice(&mut self, auditor: ActorId, agent: ActorId, affected: ActorId, harm: &str, responsibility: &str, redress: &str, growth: &str, scope: &DomainScope, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<RestorativeJusticeResult> {
        let harm_ev = self.detect_harm(auditor.clone(), harm, vec![cause], conv_id.clone(), signer)?;
        let assign_ev = self.assign(auditor.clone(), harm_ev.id.clone(), responsibility, conv_id.clone(), signer)?;
        let repair_ev = self.repair(auditor, affected, redress, scope, assign_ev.id.clone(), conv_id.clone(), signer)?;
        let grow_ev = self.grow(agent, growth, vec![repair_ev.id.clone()], conv_id, signer)?;
        Ok(RestorativeJusticeResult { harm_detection: harm_ev, responsibility: assign_ev, redress: repair_ev, growth: grow_ev })
    }

    pub fn guardrail(&mut self, source: ActorId, target: EventId, constraint: &str, dilemma: &str, escalation: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<GuardrailResult> {
        let constrain_ev = self.constrain(source.clone(), target, constraint, conv_id.clone(), signer)?;
        let dilemma_ev = self.flag_dilemma(source.clone(), dilemma, vec![constrain_ev.id.clone()], conv_id.clone(), signer)?;
        let escalate = self.0.emit(source, &format!("escalate: {escalation}"), conv_id, vec![dilemma_ev.id.clone()], signer)?;
        Ok(GuardrailResult { constraint: constrain_ev, dilemma: dilemma_ev, escalation: escalate })
    }

    pub fn impact_assessment(&mut self, source: ActorId, decision: EventId, weighing: &str, fairness: &str, explanation: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<ImpactAssessmentResult> {
        let weigh_ev = self.weigh(source.clone(), weighing, decision, conv_id.clone(), signer)?;
        let fair = self.assess_fairness(source.clone(), weigh_ev.id.clone(), fairness, conv_id.clone(), signer)?;
        let explain_ev = self.explain(source, explanation, vec![weigh_ev.id.clone(), fair.id.clone()], conv_id, signer)?;
        Ok(ImpactAssessmentResult { weighing: weigh_ev, fairness: fair, explanation: explain_ev })
    }

    pub fn whistleblow(&mut self, source: ActorId, harm: &str, explanation: &str, escalation: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<WhistleblowResult> {
        let harm_ev = self.detect_harm(source.clone(), harm, causes, conv_id.clone(), signer)?;
        let explain_ev = self.explain(source.clone(), explanation, vec![harm_ev.id.clone()], conv_id.clone(), signer)?;
        let escalate = self.0.emit(source, &format!("escalate-external: {escalation}"), conv_id, vec![explain_ev.id.clone()], signer)?;
        Ok(WhistleblowResult { harm: harm_ev, explanation: explain_ev, escalation: escalate })
    }
}

pub struct EthicsAuditResult { pub fairness: Event, pub harm_scan: Event, pub report: Event }
pub struct RestorativeJusticeResult { pub harm_detection: Event, pub responsibility: Event, pub redress: Event, pub growth: Event }
pub struct GuardrailResult { pub constraint: Event, pub dilemma: Event, pub escalation: Event }
pub struct ImpactAssessmentResult { pub weighing: Event, pub fairness: Event, pub explanation: Event }
pub struct WhistleblowResult { pub harm: Event, pub explanation: Event, pub escalation: Event }
