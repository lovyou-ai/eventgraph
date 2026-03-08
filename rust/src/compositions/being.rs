//! Layer 13 (Existence) composition operations.
//!
//! 8 operations + 3 named functions for the system's relationship with its own existence.
//! This is the sparsest grammar -- existence does not compose into complex workflows.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, EventId};

/// BeingGrammar provides Layer 13 (Existence) composition operations.
pub struct BeingGrammar<'a>(Grammar<'a>);

impl<'a> BeingGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    pub fn exist(&mut self, source: ActorId, observation: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("exist: {observation}"), conv_id, causes, signer)
    }

    pub fn accept(&mut self, source: ActorId, limitation: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("accept: {limitation}"), conv_id, causes, signer)
    }

    pub fn observe_change(&mut self, source: ActorId, observation: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("change: {observation}"), conv_id, causes, signer)
    }

    pub fn map_web(&mut self, source: ActorId, mapping: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("web: {mapping}"), conv_id, causes, signer)
    }

    pub fn face_mystery(&mut self, source: ActorId, mystery: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("mystery: {mystery}"), conv_id, causes, signer)
    }

    pub fn hold_paradox(&mut self, source: ActorId, paradox: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("paradox: {paradox}"), conv_id, causes, signer)
    }

    pub fn marvel(&mut self, source: ActorId, awe: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("marvel: {awe}"), conv_id, causes, signer)
    }

    pub fn ask_why(&mut self, source: ActorId, question: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("wonder: {question}"), conv_id, causes, signer)
    }

    // --- Named Functions ---

    pub fn farewell(&mut self, source: ActorId, limitation: &str, interconnection: &str, awe: &str, memorial: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<BeingFarewellResult> {
        let accept_ev = self.accept(source.clone(), limitation, causes, conv_id.clone(), signer)?;
        let web = self.map_web(source.clone(), interconnection, vec![accept_ev.id.clone()], conv_id.clone(), signer)?;
        let marvel_ev = self.marvel(source.clone(), awe, vec![web.id.clone()], conv_id.clone(), signer)?;
        let mem = self.0.emit(source, &format!("memorialize: {memorial}"), conv_id, vec![marvel_ev.id.clone()], signer)?;
        Ok(BeingFarewellResult { acceptance: accept_ev, web, awe: marvel_ev, memorial: mem })
    }

    pub fn contemplation(&mut self, source: ActorId, change: &str, mystery: &str, awe: &str, question: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<ContemplationResult> {
        let change_ev = self.observe_change(source.clone(), change, causes, conv_id.clone(), signer)?;
        let mystery_ev = self.face_mystery(source.clone(), mystery, vec![change_ev.id.clone()], conv_id.clone(), signer)?;
        let awe_ev = self.marvel(source.clone(), awe, vec![mystery_ev.id.clone()], conv_id.clone(), signer)?;
        let wonder_ev = self.ask_why(source, question, vec![awe_ev.id.clone()], conv_id, signer)?;
        Ok(ContemplationResult { change: change_ev, mystery: mystery_ev, awe: awe_ev, wonder: wonder_ev })
    }

    pub fn existential_audit(&mut self, source: ActorId, existence: &str, limitation: &str, interconnection: &str, purpose: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<ExistentialAuditResult> {
        let exist_ev = self.exist(source.clone(), existence, causes, conv_id.clone(), signer)?;
        let accept_ev = self.accept(source.clone(), limitation, vec![exist_ev.id.clone()], conv_id.clone(), signer)?;
        let web = self.map_web(source.clone(), interconnection, vec![accept_ev.id.clone()], conv_id.clone(), signer)?;
        let purp = self.0.emit(source, &format!("purpose: {purpose}"), conv_id, vec![web.id.clone()], signer)?;
        Ok(ExistentialAuditResult { existence: exist_ev, acceptance: accept_ev, web, purpose: purp })
    }
}

pub struct BeingFarewellResult { pub acceptance: Event, pub web: Event, pub awe: Event, pub memorial: Event }
pub struct ContemplationResult { pub change: Event, pub mystery: Event, pub awe: Event, pub wonder: Event }
pub struct ExistentialAuditResult { pub existence: Event, pub acceptance: Event, pub web: Event, pub purpose: Event }
