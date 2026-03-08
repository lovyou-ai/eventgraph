//! Layer 10 (Community) composition operations.
//!
//! 10 operations + 5 named functions for communities with shared resources.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, DomainScope, EventId, Weight};

/// BelongingGrammar provides Layer 10 (Community) composition operations.
pub struct BelongingGrammar<'a>(Grammar<'a>);

impl<'a> BelongingGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    pub fn settle(&mut self, source: ActorId, community: ActorId, scope: Option<&DomainScope>, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.subscribe(source, community, scope, cause, conv_id, signer)
    }

    pub fn contribute(&mut self, source: ActorId, contribution: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("contribute: {contribution}"), conv_id, causes, signer)
    }

    pub fn include(&mut self, source: ActorId, action: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("include: {action}"), conv_id, causes, signer)
    }

    pub fn practice(&mut self, source: ActorId, tradition: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("practice: {tradition}"), conv_id, causes, signer)
    }

    pub fn steward(&mut self, source: ActorId, steward_actor: ActorId, scope: &DomainScope, weight: Weight, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.delegate(source, steward_actor, scope, weight, cause, conv_id, signer)
    }

    pub fn sustain(&mut self, source: ActorId, assessment: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("sustain: {assessment}"), conv_id, causes, signer)
    }

    pub fn pass_on(&mut self, from: ActorId, to: ActorId, scope: &DomainScope, description: &str, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.consent(from, to, &format!("pass-on: {description}"), scope, cause, conv_id, signer)
    }

    pub fn celebrate(&mut self, source: ActorId, celebration: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("celebrate: {celebration}"), conv_id, causes, signer)
    }

    pub fn tell(&mut self, source: ActorId, story: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("tell: {story}"), conv_id, causes, signer)
    }

    pub fn gift(&mut self, source: ActorId, gift_str: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("gift: {gift_str}"), conv_id, causes, signer)
    }

    // --- Named Functions ---

    pub fn festival(&mut self, source: ActorId, celebration: &str, tradition: &str, story: &str, gift_str: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<FestivalResult> {
        let celebrate_ev = self.celebrate(source.clone(), celebration, causes, conv_id.clone(), signer)?;
        let practice_ev = self.practice(source.clone(), tradition, vec![celebrate_ev.id.clone()], conv_id.clone(), signer)?;
        let tell_ev = self.tell(source.clone(), story, vec![practice_ev.id.clone()], conv_id.clone(), signer)?;
        let gift_ev = self.gift(source, gift_str, vec![tell_ev.id.clone()], conv_id, signer)?;
        Ok(FestivalResult { celebration: celebrate_ev, practice: practice_ev, story: tell_ev, gift: gift_ev })
    }

    pub fn commons_governance(&mut self, source: ActorId, steward_actor: ActorId, scope: &DomainScope, weight: Weight, assessment: &str, rule: &str, findings: &str, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<CommonsGovernanceResult> {
        let stewardship = self.steward(source.clone(), steward_actor.clone(), scope, weight, cause, conv_id.clone(), signer)?;
        let sustain_ev = self.sustain(steward_actor.clone(), assessment, vec![stewardship.id.clone()], conv_id.clone(), signer)?;
        let legislate = self.0.emit(source, &format!("legislate: {rule}"), conv_id.clone(), vec![sustain_ev.id.clone()], signer)?;
        let audit = self.0.annotate(steward_actor, legislate.id.clone(), "audit", findings, conv_id, signer)?;
        Ok(CommonsGovernanceResult { stewardship, assessment: sustain_ev, legislation: legislate, audit })
    }

    pub fn renewal(&mut self, source: ActorId, assessment: &str, evolved_practice: &str, new_story: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<RenewalResult> {
        let sustain_ev = self.sustain(source.clone(), assessment, causes, conv_id.clone(), signer)?;
        let practice_ev = self.practice(source.clone(), evolved_practice, vec![sustain_ev.id.clone()], conv_id.clone(), signer)?;
        let story = self.tell(source, new_story, vec![practice_ev.id.clone()], conv_id, signer)?;
        Ok(RenewalResult { assessment: sustain_ev, practice: practice_ev, story })
    }

    pub fn onboard(&mut self, sponsor: ActorId, newcomer: ActorId, community: ActorId, scope: Option<&DomainScope>, inclusion_action: &str, tradition: &str, first_contribution: &str, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<OnboardResult> {
        let inclusion = self.include(sponsor, inclusion_action, vec![cause], conv_id.clone(), signer)?;
        let settle_ev = self.settle(newcomer.clone(), community, scope, inclusion.id.clone(), conv_id.clone(), signer)?;
        let practice_ev = self.practice(newcomer.clone(), tradition, vec![settle_ev.id.clone()], conv_id.clone(), signer)?;
        let contrib = self.contribute(newcomer, first_contribution, vec![practice_ev.id.clone()], conv_id, signer)?;
        Ok(OnboardResult { inclusion, settlement: settle_ev, first_practice: practice_ev, contribution: contrib })
    }

    pub fn succession(&mut self, outgoing: ActorId, incoming: ActorId, assessment: &str, scope: &DomainScope, celebration: &str, story: &str, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<SuccessionResult> {
        let sustain_ev = self.sustain(outgoing.clone(), assessment, vec![cause], conv_id.clone(), signer)?;
        let transfer = self.pass_on(outgoing.clone(), incoming, scope, "stewardship transfer", sustain_ev.id.clone(), conv_id.clone(), signer)?;
        let celebrate_ev = self.celebrate(outgoing.clone(), celebration, vec![transfer.id.clone()], conv_id.clone(), signer)?;
        let tell_ev = self.tell(outgoing, story, vec![celebrate_ev.id.clone()], conv_id, signer)?;
        Ok(SuccessionResult { assessment: sustain_ev, transfer, celebration: celebrate_ev, story: tell_ev })
    }
}

pub struct FestivalResult { pub celebration: Event, pub practice: Event, pub story: Event, pub gift: Event }
pub struct CommonsGovernanceResult { pub stewardship: Event, pub assessment: Event, pub legislation: Event, pub audit: Event }
pub struct RenewalResult { pub assessment: Event, pub practice: Event, pub story: Event }
pub struct OnboardResult { pub inclusion: Event, pub settlement: Event, pub first_practice: Event, pub contribution: Event }
pub struct SuccessionResult { pub assessment: Event, pub transfer: Event, pub celebration: Event, pub story: Event }
