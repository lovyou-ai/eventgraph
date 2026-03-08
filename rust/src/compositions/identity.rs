//! Layer 8 (Identity) composition operations.
//!
//! 10 operations + 5 named functions for self-sovereign identity.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, DomainScope, EventId, Weight};

/// IdentityGrammar provides Layer 8 (Identity) composition operations.
pub struct IdentityGrammar<'a>(Grammar<'a>);

impl<'a> IdentityGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    pub fn introspect(&mut self, source: ActorId, self_model: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("introspect: {self_model}"), conv_id, causes, signer)
    }

    pub fn narrate(&mut self, source: ActorId, narrative: &str, basis: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("narrate: {narrative}"), basis, conv_id, signer)
    }

    pub fn align(&mut self, source: ActorId, target: EventId, alignment: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "alignment", alignment, conv_id, signer)
    }

    pub fn bound(&mut self, source: ActorId, boundary: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("bound: {boundary}"), conv_id, causes, signer)
    }

    pub fn aspire(&mut self, source: ActorId, aspiration: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("aspire: {aspiration}"), conv_id, causes, signer)
    }

    pub fn transform(&mut self, source: ActorId, transformation: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("transform: {transformation}"), conv_id, causes, signer)
    }

    pub fn disclose(&mut self, source: ActorId, target: ActorId, scope: Option<&DomainScope>, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.channel(source, target, scope, cause, conv_id, signer)
    }

    pub fn recognize(&mut self, source: ActorId, recognition: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("recognize: {recognition}"), conv_id, causes, signer)
    }

    pub fn distinguish(&mut self, source: ActorId, target: EventId, uniqueness: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "uniqueness", uniqueness, conv_id, signer)
    }

    pub fn memorialize(&mut self, source: ActorId, memorial: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("memorialize: {memorial}"), conv_id, causes, signer)
    }

    // --- Named Functions ---

    pub fn identity_audit(&mut self, source: ActorId, self_model: &str, alignment: &str, narrative: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<IdentityAuditResult> {
        let intro = self.introspect(source.clone(), self_model, causes, conv_id.clone(), signer)?;
        let align_ev = self.align(source.clone(), intro.id.clone(), alignment, conv_id.clone(), signer)?;
        let narr = self.narrate(source, narrative, align_ev.id.clone(), conv_id, signer)?;
        Ok(IdentityAuditResult { self_model: intro, alignment: align_ev, narrative: narr })
    }

    pub fn retirement(&mut self, system: ActorId, departing: &ActorId, successor: ActorId, memorial: &str, scope: &DomainScope, weight: Weight, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<RetirementResult> {
        let mem = self.memorialize(system.clone(), &format!("retirement of {}: {memorial}", departing.value()), causes, conv_id.clone(), signer)?;
        let transfer_ev = self.0.delegate(system.clone(), successor, scope, weight, mem.id.clone(), conv_id.clone(), signer)?;
        let archive = self.0.emit(system, &format!("archive: contributions of {}", departing.value()), conv_id, vec![transfer_ev.id.clone()], signer)?;
        Ok(RetirementResult { memorial: mem, transfer: transfer_ev, archive })
    }

    pub fn credential(&mut self, source: ActorId, verifier: ActorId, self_model: &str, scope: Option<&DomainScope>, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<CredentialResult> {
        let intro = self.introspect(source.clone(), self_model, causes, conv_id.clone(), signer)?;
        let disclose_ev = self.disclose(source, verifier, scope, intro.id.clone(), conv_id, signer)?;
        Ok(CredentialResult { introspection: intro, disclosure: disclose_ev })
    }

    pub fn reinvention(&mut self, source: ActorId, transformation: &str, narrative: &str, aspiration: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<ReinventionResult> {
        let transform_ev = self.transform(source.clone(), transformation, causes, conv_id.clone(), signer)?;
        let narr = self.narrate(source.clone(), narrative, transform_ev.id.clone(), conv_id.clone(), signer)?;
        let aspire_ev = self.aspire(source, aspiration, vec![narr.id.clone()], conv_id, signer)?;
        Ok(ReinventionResult { transformation: transform_ev, narrative: narr, aspiration: aspire_ev })
    }

    pub fn introduction(&mut self, source: ActorId, target: ActorId, scope: Option<&DomainScope>, narrative: &str, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<IntroductionResult> {
        let disclose_ev = self.disclose(source.clone(), target, scope, cause, conv_id.clone(), signer)?;
        let narr = self.narrate(source, narrative, disclose_ev.id.clone(), conv_id, signer)?;
        Ok(IntroductionResult { disclosure: disclose_ev, narrative: narr })
    }
}

pub struct IdentityAuditResult { pub self_model: Event, pub alignment: Event, pub narrative: Event }
pub struct RetirementResult { pub memorial: Event, pub transfer: Event, pub archive: Event }
pub struct CredentialResult { pub introspection: Event, pub disclosure: Event }
pub struct ReinventionResult { pub transformation: Event, pub narrative: Event, pub aspiration: Event }
pub struct IntroductionResult { pub disclosure: Event, pub narrative: Event }
