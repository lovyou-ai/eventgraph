//! Layer 11 (Culture) composition operations.
//!
//! 10 operations + 5 named functions for cross-cultural communication.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, DomainScope, EventId};

/// MeaningGrammar provides Layer 11 (Culture) composition operations.
pub struct MeaningGrammar<'a>(Grammar<'a>);

impl<'a> MeaningGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    pub fn examine(&mut self, source: ActorId, examination: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("examine: {examination}"), conv_id, causes, signer)
    }

    pub fn reframe(&mut self, source: ActorId, reframing: &str, original: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("reframe: {reframing}"), original, conv_id, signer)
    }

    pub fn question(&mut self, source: ActorId, question: &str, target: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        let (_, flag) = self.0.challenge(source, &format!("question: {question}"), target, conv_id, signer)?;
        Ok(flag)
    }

    pub fn distill(&mut self, source: ActorId, wisdom: &str, experience: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("distill: {wisdom}"), experience, conv_id, signer)
    }

    pub fn beautify(&mut self, source: ActorId, beauty: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("beautify: {beauty}"), conv_id, causes, signer)
    }

    pub fn liken(&mut self, source: ActorId, metaphor: &str, subject: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("liken: {metaphor}"), subject, conv_id, signer)
    }

    pub fn lighten(&mut self, source: ActorId, humour: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("lighten: {humour}"), conv_id, causes, signer)
    }

    pub fn teach(&mut self, source: ActorId, student: ActorId, scope: Option<&DomainScope>, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.channel(source, student, scope, cause, conv_id, signer)
    }

    pub fn translate(&mut self, source: ActorId, translation: &str, original: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("translate: {translation}"), original, conv_id, signer)
    }

    pub fn prophesy(&mut self, source: ActorId, prediction: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("prophesy: {prediction}"), conv_id, causes, signer)
    }

    // --- Named Functions ---

    pub fn design_review(&mut self, source: ActorId, beauty: &str, reframing: &str, question_str: &str, wisdom: &str, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<DesignReviewResult> {
        let beautify_ev = self.beautify(source.clone(), beauty, vec![cause], conv_id.clone(), signer)?;
        let reframe_ev = self.reframe(source.clone(), reframing, beautify_ev.id.clone(), conv_id.clone(), signer)?;
        let q = self.question(source.clone(), question_str, reframe_ev.id.clone(), conv_id.clone(), signer)?;
        let w = self.distill(source, wisdom, q.id.clone(), conv_id, signer)?;
        Ok(DesignReviewResult { beauty: beautify_ev, reframe: reframe_ev, question: q, wisdom: w })
    }

    pub fn cultural_onboarding(&mut self, guide: ActorId, newcomer: ActorId, translation: &str, teaching_scope: Option<&DomainScope>, examination: &str, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<CulturalOnboardingResult> {
        let translate_ev = self.translate(guide.clone(), translation, cause, conv_id.clone(), signer)?;
        let teach_ev = self.teach(guide, newcomer.clone(), teaching_scope, translate_ev.id.clone(), conv_id.clone(), signer)?;
        let examine_ev = self.examine(newcomer, examination, vec![teach_ev.id.clone()], conv_id, signer)?;
        Ok(CulturalOnboardingResult { translation: translate_ev, teaching: teach_ev, examination: examine_ev })
    }

    pub fn forecast(&mut self, source: ActorId, prediction: &str, assumptions: &str, confidence: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<ForecastResult> {
        let prophesy_ev = self.prophesy(source.clone(), prediction, causes, conv_id.clone(), signer)?;
        let examine_ev = self.examine(source.clone(), assumptions, vec![prophesy_ev.id.clone()], conv_id.clone(), signer)?;
        let distill_ev = self.distill(source, confidence, examine_ev.id.clone(), conv_id, signer)?;
        Ok(ForecastResult { prophecy: prophesy_ev, examination: examine_ev, wisdom: distill_ev })
    }

    pub fn post_mortem(&mut self, source: ActorId, examination: &str, question_str: &str, wisdom: &str, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<MeaningPostMortemResult> {
        let exam = self.examine(source.clone(), examination, vec![cause], conv_id.clone(), signer)?;
        let q = self.question(source.clone(), question_str, exam.id.clone(), conv_id.clone(), signer)?;
        let w = self.distill(source, wisdom, q.id.clone(), conv_id, signer)?;
        Ok(MeaningPostMortemResult { examination: exam, questions: q, wisdom: w })
    }

    pub fn mentorship(&mut self, mentor: ActorId, student: ActorId, reframing: &str, wisdom: &str, translation: &str, scope: Option<&DomainScope>, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<MentorshipResult> {
        let channel = self.teach(mentor.clone(), student.clone(), scope, cause, conv_id.clone(), signer)?;
        let reframe_ev = self.reframe(mentor.clone(), reframing, channel.id.clone(), conv_id.clone(), signer)?;
        let distill_ev = self.distill(mentor, wisdom, reframe_ev.id.clone(), conv_id.clone(), signer)?;
        let translate_ev = self.translate(student, translation, distill_ev.id.clone(), conv_id, signer)?;
        Ok(MentorshipResult { channel, reframing: reframe_ev, wisdom: distill_ev, translation: translate_ev })
    }
}

pub struct DesignReviewResult { pub beauty: Event, pub reframe: Event, pub question: Event, pub wisdom: Event }
pub struct CulturalOnboardingResult { pub translation: Event, pub teaching: Event, pub examination: Event }
pub struct ForecastResult { pub prophecy: Event, pub examination: Event, pub wisdom: Event }
pub struct MeaningPostMortemResult { pub examination: Event, pub questions: Event, pub wisdom: Event }
pub struct MentorshipResult { pub channel: Event, pub reframing: Event, pub wisdom: Event, pub translation: Event }
