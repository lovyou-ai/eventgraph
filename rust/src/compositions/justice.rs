//! Layer 4 (Legal) composition operations.
//!
//! 12 operations + 6 named functions for transparent dispute resolution.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, DomainScope, EventId, Weight};

/// JusticeGrammar provides Layer 4 (Legal) composition operations.
pub struct JusticeGrammar<'a>(Grammar<'a>);

impl<'a> JusticeGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    pub fn legislate(&mut self, source: ActorId, rule: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("legislate: {rule}"), conv_id, causes, signer)
    }

    pub fn amend(&mut self, source: ActorId, amendment: &str, rule: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("amend: {amendment}"), rule, conv_id, signer)
    }

    pub fn repeal(&mut self, source: ActorId, rule: EventId, reason: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.retract(source, rule, reason, conv_id, signer)
    }

    pub fn file(&mut self, source: ActorId, complaint: &str, target: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        let (_, flag) = self.0.challenge(source, &format!("file: {complaint}"), target, conv_id, signer)?;
        Ok(flag)
    }

    pub fn submit(&mut self, source: ActorId, target: EventId, evidence: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "evidence", evidence, conv_id, signer)
    }

    pub fn argue(&mut self, source: ActorId, argument: &str, target: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.respond(source, &format!("argue: {argument}"), target, conv_id, signer)
    }

    pub fn judge(&mut self, source: ActorId, ruling: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("judge: {ruling}"), conv_id, causes, signer)
    }

    pub fn appeal(&mut self, source: ActorId, grounds: &str, ruling: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        let (_, flag) = self.0.challenge(source, &format!("appeal: {grounds}"), ruling, conv_id, signer)?;
        Ok(flag)
    }

    pub fn enforce(&mut self, source: ActorId, executor: ActorId, scope: &DomainScope, weight: Weight, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.delegate(source, executor, scope, weight, cause, conv_id, signer)
    }

    pub fn audit(&mut self, source: ActorId, target: EventId, findings: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "audit", findings, conv_id, signer)
    }

    pub fn pardon(&mut self, authority: ActorId, pardoned: ActorId, terms: &str, scope: &DomainScope, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.consent(authority, pardoned, &format!("pardon: {terms}"), scope, cause, conv_id, signer)
    }

    pub fn reform(&mut self, source: ActorId, proposal: &str, precedent: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("reform: {proposal}"), precedent, conv_id, signer)
    }

    // --- Named Functions ---

    pub fn trial(
        &mut self, plaintiff: ActorId, defendant: ActorId, judge_actor: ActorId,
        complaint: &str, plaintiff_evidence: &str, defendant_evidence: &str,
        plaintiff_argument: &str, defendant_argument: &str, ruling: &str,
        target: EventId, conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<TrialResult> {
        let filing = self.file(plaintiff.clone(), complaint, target, conv_id.clone(), signer)?;
        let sub1 = self.submit(plaintiff.clone(), filing.id.clone(), plaintiff_evidence, conv_id.clone(), signer)?;
        let sub2 = self.submit(defendant.clone(), filing.id.clone(), defendant_evidence, conv_id.clone(), signer)?;
        let arg1 = self.argue(plaintiff, plaintiff_argument, sub1.id.clone(), conv_id.clone(), signer)?;
        let arg2 = self.argue(defendant, defendant_argument, sub2.id.clone(), conv_id.clone(), signer)?;
        let verdict = self.judge(judge_actor, ruling, vec![arg1.id.clone(), arg2.id.clone()], conv_id, signer)?;
        Ok(TrialResult { filing, submissions: vec![sub1, sub2], arguments: vec![arg1, arg2], ruling: verdict })
    }

    pub fn constitutional_amendment(
        &mut self, proposer: ActorId, proposal: &str, legislation: &str, rights_assessment: &str,
        precedent: EventId, conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<ConstitutionalAmendmentResult> {
        let reform_ev = self.reform(proposer.clone(), proposal, precedent, conv_id.clone(), signer)?;
        let legislate_ev = self.legislate(proposer.clone(), legislation, vec![reform_ev.id.clone()], conv_id.clone(), signer)?;
        let rights = self.audit(proposer, legislate_ev.id.clone(), rights_assessment, conv_id, signer)?;
        Ok(ConstitutionalAmendmentResult { reform: reform_ev, legislation: legislate_ev, rights_check: rights })
    }

    pub fn injunction(
        &mut self, petitioner: ActorId, judge_actor: ActorId, executor: ActorId,
        complaint: &str, ruling: &str, scope: &DomainScope, weight: Weight,
        target: EventId, conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<InjunctionResult> {
        let filing = self.file(petitioner, complaint, target, conv_id.clone(), signer)?;
        let verdict = self.judge(judge_actor.clone(), &format!("emergency: {ruling}"), vec![filing.id.clone()], conv_id.clone(), signer)?;
        let enforce_ev = self.enforce(judge_actor, executor, scope, weight, verdict.id.clone(), conv_id, signer)?;
        Ok(InjunctionResult { filing, ruling: verdict, enforcement: enforce_ev })
    }

    pub fn plea(
        &mut self, defendant: ActorId, prosecutor: ActorId, executor: ActorId,
        complaint: &str, deal: &str, scope: &DomainScope, weight: Weight,
        target: EventId, conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<PleaResult> {
        let filing = self.file(prosecutor.clone(), complaint, target, conv_id.clone(), signer)?;
        let acceptance = self.pardon(prosecutor.clone(), defendant, deal, scope, filing.id.clone(), conv_id.clone(), signer)?;
        let enforce_ev = self.enforce(prosecutor, executor, scope, weight, acceptance.id.clone(), conv_id, signer)?;
        Ok(PleaResult { filing, acceptance, enforcement: enforce_ev })
    }

    pub fn class_action(
        &mut self, plaintiffs: &[ActorId], defendant: ActorId, judge_actor: ActorId,
        complaints: &[&str], evidence: &str, argument: &str,
        defense_evidence: &str, defense_argument: &str, ruling: &str,
        target: EventId, conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<ClassActionResult> {
        if plaintiffs.len() != complaints.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation { detail: "class-action: plaintiffs and complaints must have equal length".to_string() });
        }
        let mut filings = Vec::new();
        let mut filing_ids = Vec::new();
        for (i, plaintiff) in plaintiffs.iter().enumerate() {
            let f = self.file(plaintiff.clone(), complaints[i], target.clone(), conv_id.clone(), signer)?;
            filing_ids.push(f.id.clone());
            filings.push(f);
        }
        let merged = self.0.merge(plaintiffs[0].clone(), "class-action: merged complaints", filing_ids, conv_id.clone(), signer)?;
        let trial_result = self.trial(plaintiffs[0].clone(), defendant, judge_actor, "class-action", evidence, defense_evidence, argument, defense_argument, ruling, merged.id.clone(), conv_id, signer)?;
        Ok(ClassActionResult { filings, merged, trial: trial_result })
    }

    pub fn recall(
        &mut self, auditor: ActorId, community: ActorId, _official: ActorId,
        findings: &str, complaint: &str, scope: &DomainScope,
        target: EventId, conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<RecallResult> {
        let audit_ev = self.audit(auditor.clone(), target, findings, conv_id.clone(), signer)?;
        let filing = self.file(auditor, complaint, audit_ev.id.clone(), conv_id.clone(), signer)?;
        let consent_ev = self.0.consent(community.clone(), _official, &format!("recall: {complaint}"), scope, filing.id.clone(), conv_id.clone(), signer)?;
        let revocation = self.0.emit(community, &format!("role-revoked: {complaint}"), conv_id, vec![consent_ev.id.clone()], signer)?;
        Ok(RecallResult { audit: audit_ev, filing, consent: consent_ev, revocation })
    }
}

pub struct TrialResult { pub filing: Event, pub submissions: Vec<Event>, pub arguments: Vec<Event>, pub ruling: Event }
pub struct ConstitutionalAmendmentResult { pub reform: Event, pub legislation: Event, pub rights_check: Event }
pub struct InjunctionResult { pub filing: Event, pub ruling: Event, pub enforcement: Event }
pub struct PleaResult { pub filing: Event, pub acceptance: Event, pub enforcement: Event }
pub struct ClassActionResult { pub filings: Vec<Event>, pub merged: Event, pub trial: TrialResult }
pub struct RecallResult { pub audit: Event, pub filing: Event, pub consent: Event, pub revocation: Event }
