//! Layer 9 (Relationship) composition operations.
//!
//! 10 operations + 5 named functions for deep relational bonds.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, DomainScope, EventId, Weight};

/// BondGrammar provides Layer 9 (Relationship) composition operations.
pub struct BondGrammar<'a>(Grammar<'a>);

impl<'a> BondGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    /// Connect initiates a relational bond via mutual subscription.
    pub fn connect(&mut self, source: ActorId, target: ActorId, scope: Option<&DomainScope>, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<(Event, Event)> {
        let sub1 = self.0.subscribe(source.clone(), target.clone(), scope, cause, conv_id.clone(), signer)?;
        let sub2 = self.0.subscribe(target, source, scope, sub1.id.clone(), conv_id, signer)?;
        Ok((sub1, sub2))
    }

    pub fn balance(&mut self, source: ActorId, target: EventId, assessment: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "reciprocity", assessment, conv_id, signer)
    }

    pub fn deepen(&mut self, source: ActorId, other: ActorId, basis: &str, scope: &DomainScope, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.consent(source, other, &format!("deepen: {basis}"), scope, cause, conv_id, signer)
    }

    pub fn open(&mut self, source: ActorId, target: ActorId, scope: Option<&DomainScope>, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.channel(source, target, scope, cause, conv_id, signer)
    }

    pub fn attune(&mut self, source: ActorId, understanding: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("attune: {understanding}"), conv_id, causes, signer)
    }

    pub fn feel_with(&mut self, source: ActorId, empathy: &str, target: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.respond(source, &format!("empathy: {empathy}"), target, conv_id, signer)
    }

    pub fn break_bond(&mut self, source: ActorId, rupture: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("rupture: {rupture}"), conv_id, causes, signer)
    }

    pub fn apologize(&mut self, source: ActorId, apology: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("apology: {apology}"), conv_id, causes, signer)
    }

    pub fn reconcile(&mut self, source: ActorId, other: ActorId, progress: &str, scope: &DomainScope, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.consent(source, other, &format!("reconcile: {progress}"), scope, cause, conv_id, signer)
    }

    pub fn mourn(&mut self, source: ActorId, loss: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("mourn: {loss}"), conv_id, causes, signer)
    }

    // --- Named Functions ---

    pub fn betrayal_repair(&mut self, injured: ActorId, offender: ActorId, rupture: &str, apology: &str, reconciliation: &str, new_basis: &str, scope: &DomainScope, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<BetrayalRepairResult> {
        let rupture_ev = self.break_bond(injured.clone(), rupture, causes, conv_id.clone(), signer)?;
        let apology_ev = self.apologize(offender.clone(), apology, vec![rupture_ev.id.clone()], conv_id.clone(), signer)?;
        let reconcile_ev = self.reconcile(injured.clone(), offender.clone(), reconciliation, scope, apology_ev.id.clone(), conv_id.clone(), signer)?;
        let deepen_ev = self.deepen(injured, offender, new_basis, scope, reconcile_ev.id.clone(), conv_id, signer)?;
        Ok(BetrayalRepairResult { rupture: rupture_ev, apology: apology_ev, reconciliation: reconcile_ev, deepened: deepen_ev })
    }

    pub fn check_in(&mut self, source: ActorId, balance_target: EventId, assessment: &str, attunement: &str, empathy: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<CheckInResult> {
        let bal = self.balance(source.clone(), balance_target, assessment, conv_id.clone(), signer)?;
        let att = self.attune(source.clone(), attunement, vec![bal.id.clone()], conv_id.clone(), signer)?;
        let emp = self.feel_with(source, empathy, att.id.clone(), conv_id, signer)?;
        Ok(CheckInResult { balance: bal, attunement: att, empathy: emp })
    }

    pub fn mentorship(&mut self, mentor: ActorId, mentee: ActorId, basis: &str, understanding: &str, scope: &DomainScope, teaching_scope: Option<&DomainScope>, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<BondMentorshipResult> {
        let connect_ev = self.0.subscribe(mentee, mentor.clone(), teaching_scope, cause, conv_id.clone(), signer)?;
        let deepen_ev = self.deepen(mentor.clone(), connect_ev.source.clone(), basis, scope, connect_ev.id.clone(), conv_id.clone(), signer)?;
        let attune_ev = self.attune(mentor.clone(), understanding, vec![deepen_ev.id.clone()], conv_id.clone(), signer)?;
        let teach = self.0.channel(mentor, connect_ev.source.clone(), teaching_scope, attune_ev.id.clone(), conv_id, signer)?;
        Ok(BondMentorshipResult { connection: connect_ev, deepening: deepen_ev, attunement: attune_ev, teaching: teach })
    }

    pub fn farewell(&mut self, source: ActorId, departing: ActorId, loss: &str, memorial: &str, gratitude_weight: Weight, scope: Option<&DomainScope>, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<BondFarewellResult> {
        let mourn_ev = self.mourn(source.clone(), loss, causes, conv_id.clone(), signer)?;
        let mem = self.0.emit(source.clone(), &format!("memorialize: {memorial}"), conv_id.clone(), vec![mourn_ev.id.clone()], signer)?;
        let gratitude = self.0.endorse(source, mem.id.clone(), departing, gratitude_weight, scope, conv_id, signer)?;
        Ok(BondFarewellResult { mourning: mourn_ev, memorial: mem, gratitude })
    }

    pub fn forgive(&mut self, source: ActorId, sever_event: EventId, target: ActorId, scope: Option<&DomainScope>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.forgive(source, sever_event, target, scope, conv_id, signer)
    }
}

pub struct BetrayalRepairResult { pub rupture: Event, pub apology: Event, pub reconciliation: Event, pub deepened: Event }
pub struct CheckInResult { pub balance: Event, pub attunement: Event, pub empathy: Event }
pub struct BondMentorshipResult { pub connection: Event, pub deepening: Event, pub attunement: Event, pub teaching: Event }
pub struct BondFarewellResult { pub mourning: Event, pub memorial: Event, pub gratitude: Event }
