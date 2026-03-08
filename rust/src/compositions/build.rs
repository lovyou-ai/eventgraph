//! Layer 5 (Technology) composition operations.
//!
//! 12 operations + 5 named functions for development, CI/CD, and artefact lifecycle.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, EventId};

/// BuildGrammar provides Layer 5 (Technology) composition operations.
pub struct BuildGrammar<'a>(Grammar<'a>);

impl<'a> BuildGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    pub fn build(&mut self, source: ActorId, artefact: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("build: {artefact}"), conv_id, causes, signer)
    }

    pub fn version(&mut self, source: ActorId, version: &str, previous: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("version: {version}"), previous, conv_id, signer)
    }

    pub fn ship(&mut self, source: ActorId, deployment: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("ship: {deployment}"), conv_id, causes, signer)
    }

    pub fn sunset(&mut self, source: ActorId, target: EventId, migration: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "deprecated", migration, conv_id, signer)
    }

    pub fn define(&mut self, source: ActorId, workflow: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("define: {workflow}"), conv_id, causes, signer)
    }

    pub fn automate(&mut self, source: ActorId, automation: &str, workflow: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("automate: {automation}"), workflow, conv_id, signer)
    }

    pub fn test(&mut self, source: ActorId, results: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("test: {results}"), conv_id, causes, signer)
    }

    pub fn review(&mut self, source: ActorId, assessment: &str, target: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.respond(source, &format!("review: {assessment}"), target, conv_id, signer)
    }

    pub fn measure(&mut self, source: ActorId, target: EventId, scores: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(source, target, "quality", scores, conv_id, signer)
    }

    pub fn feedback(&mut self, source: ActorId, feedback: &str, target: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.respond(source, &format!("feedback: {feedback}"), target, conv_id, signer)
    }

    pub fn iterate(&mut self, source: ActorId, improvement: &str, previous: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.derive(source, &format!("iterate: {improvement}"), previous, conv_id, signer)
    }

    pub fn innovate(&mut self, source: ActorId, innovation: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("innovate: {innovation}"), conv_id, causes, signer)
    }

    // --- Named Functions ---

    pub fn spike(&mut self, source: ActorId, experiment: &str, test_results: &str, feedback_str: &str, decision: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<SpikeResult> {
        let build_ev = self.build(source.clone(), &format!("spike: {experiment}"), causes, conv_id.clone(), signer)?;
        let test_ev = self.test(source.clone(), test_results, vec![build_ev.id.clone()], conv_id.clone(), signer)?;
        let fb = self.feedback(source.clone(), feedback_str, test_ev.id.clone(), conv_id.clone(), signer)?;
        let dec = self.0.emit(source, &format!("spike-decision: {decision}"), conv_id, vec![fb.id.clone()], signer)?;
        Ok(SpikeResult { build: build_ev, test: test_ev, feedback: fb, decision: dec })
    }

    pub fn migration(&mut self, source: ActorId, deprecated_target: EventId, migration_path: &str, new_version: &str, deployment: &str, test_results: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<MigrationResult> {
        let sunset_ev = self.sunset(source.clone(), deprecated_target, migration_path, conv_id.clone(), signer)?;
        let version_ev = self.version(source.clone(), new_version, sunset_ev.id.clone(), conv_id.clone(), signer)?;
        let ship_ev = self.ship(source.clone(), deployment, vec![version_ev.id.clone()], conv_id.clone(), signer)?;
        let test_ev = self.test(source, test_results, vec![ship_ev.id.clone()], conv_id, signer)?;
        Ok(MigrationResult { sunset: sunset_ev, version: version_ev, ship: ship_ev, test: test_ev })
    }

    pub fn tech_debt(&mut self, source: ActorId, target: EventId, scores: &str, debt_description: &str, plan: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<TechDebtResult> {
        let measure_ev = self.measure(source.clone(), target, scores, conv_id.clone(), signer)?;
        let debt = self.0.annotate(source.clone(), measure_ev.id.clone(), "tech_debt", debt_description, conv_id.clone(), signer)?;
        let iterate_ev = self.iterate(source, plan, debt.id.clone(), conv_id, signer)?;
        Ok(TechDebtResult { measure: measure_ev, debt_mark: debt, iteration: iterate_ev })
    }

    pub fn pipeline(&mut self, source: ActorId, workflow: &str, test_results: &str, metrics: &str, deployment: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<PipelineResult> {
        let def = self.define(source.clone(), workflow, causes, conv_id.clone(), signer)?;
        let test_ev = self.test(source.clone(), test_results, vec![def.id.clone()], conv_id.clone(), signer)?;
        let measure_ev = self.measure(source.clone(), test_ev.id.clone(), metrics, conv_id.clone(), signer)?;
        let ship_ev = self.ship(source, deployment, vec![measure_ev.id.clone()], conv_id, signer)?;
        Ok(PipelineResult { definition: def, test_result: test_ev, metrics: measure_ev, deployment: ship_ev })
    }

    pub fn post_mortem(&mut self, lead: ActorId, contributors: &[ActorId], feedbacks: &[&str], analysis: &str, improvements: &str, incident: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<PostMortemResult> {
        if contributors.len() != feedbacks.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation { detail: "post-mortem: contributors and feedbacks must have equal length".to_string() });
        }
        let mut fb_events = Vec::new();
        let mut fb_ids = Vec::new();
        for (i, contrib) in contributors.iter().enumerate() {
            let fb = self.feedback(contrib.clone(), feedbacks[i], incident.clone(), conv_id.clone(), signer)?;
            fb_ids.push(fb.id.clone());
            fb_events.push(fb);
        }
        let analysis_ev = self.measure(lead.clone(), fb_ids.last().unwrap().clone(), &format!("post-mortem: {analysis}"), conv_id.clone(), signer)?;
        let improve = self.define(lead, improvements, vec![analysis_ev.id.clone()], conv_id, signer)?;
        Ok(PostMortemResult { feedback: fb_events, analysis: analysis_ev, improvements: improve })
    }
}

pub struct SpikeResult { pub build: Event, pub test: Event, pub feedback: Event, pub decision: Event }
pub struct MigrationResult { pub sunset: Event, pub version: Event, pub ship: Event, pub test: Event }
pub struct TechDebtResult { pub measure: Event, pub debt_mark: Event, pub iteration: Event }
pub struct PipelineResult { pub definition: Event, pub test_result: Event, pub metrics: Event, pub deployment: Event }
pub struct PostMortemResult { pub feedback: Vec<Event>, pub analysis: Event, pub improvements: Event }
