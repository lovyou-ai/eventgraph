//! Layer 1 (Agency) composition operations.
//!
//! 12 operations + 6 named functions for task management where AI agents
//! and humans operate on the same graph.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, DomainScope, EventId, Weight};

/// WorkGrammar provides Layer 1 (Agency) composition operations.
pub struct WorkGrammar<'a>(Grammar<'a>);

impl<'a> WorkGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    // --- Operations (12) ---

    /// Intend declares a goal or desired outcome. (Goal + Emit)
    pub fn intend(
        &mut self,
        source: ActorId,
        goal: &str,
        causes: Vec<EventId>,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.emit(source, &format!("intend: {goal}"), conv_id, causes, signer)
    }

    /// Decompose breaks a goal into actionable steps. (Plan + Derive)
    pub fn decompose(
        &mut self,
        source: ActorId,
        subtask: &str,
        goal: EventId,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.derive(source, &format!("decompose: {subtask}"), goal, conv_id, signer)
    }

    /// Assign gives work to a specific actor. (Delegation + Delegate)
    pub fn assign(
        &mut self,
        source: ActorId,
        assignee: ActorId,
        scope: &DomainScope,
        weight: Weight,
        cause: EventId,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.delegate(source, assignee, scope, weight, cause, conv_id, signer)
    }

    /// Claim takes on unassigned work. (Initiative + Emit)
    pub fn claim(
        &mut self,
        source: ActorId,
        work: &str,
        causes: Vec<EventId>,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.emit(source, &format!("claim: {work}"), conv_id, causes, signer)
    }

    /// Prioritize ranks work by importance. (Focus + Annotate)
    pub fn prioritize(
        &mut self,
        source: ActorId,
        target: EventId,
        priority: &str,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.annotate(source, target, "priority", priority, conv_id, signer)
    }

    /// Block flags work that cannot proceed. (Salience + Annotate)
    pub fn block(
        &mut self,
        source: ActorId,
        target: EventId,
        blocker: &str,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.annotate(source, target, "blocked", blocker, conv_id, signer)
    }

    /// Unblock removes an impediment to work. (Salience + Emit)
    pub fn unblock(
        &mut self,
        source: ActorId,
        resolution: &str,
        causes: Vec<EventId>,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.emit(source, &format!("unblock: {resolution}"), conv_id, causes, signer)
    }

    /// Progress reports incremental advancement. (Commitment + Extend)
    pub fn progress(
        &mut self,
        source: ActorId,
        update: &str,
        previous: EventId,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.extend(source, &format!("progress: {update}"), previous, conv_id, signer)
    }

    /// Complete marks work as done with evidence. (Commitment + Emit)
    pub fn complete(
        &mut self,
        source: ActorId,
        summary: &str,
        causes: Vec<EventId>,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.emit(source, &format!("complete: {summary}"), conv_id, causes, signer)
    }

    /// Handoff transfers work between actors. (Delegation + Consent)
    pub fn handoff(
        &mut self,
        from: ActorId,
        to: ActorId,
        description: &str,
        scope: &DomainScope,
        cause: EventId,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.consent(from, to, &format!("handoff: {description}"), scope, cause, conv_id, signer)
    }

    /// Scope defines what an actor may do autonomously. (Permission + Delegate)
    pub fn scope(
        &mut self,
        source: ActorId,
        target: ActorId,
        scope: &DomainScope,
        weight: Weight,
        cause: EventId,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.delegate(source, target, scope, weight, cause, conv_id, signer)
    }

    /// Review evaluates completed work. (Accountability + Respond)
    pub fn review(
        &mut self,
        source: ActorId,
        assessment: &str,
        target: EventId,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.0.respond(source, &format!("review: {assessment}"), target, conv_id, signer)
    }

    // --- Named Functions (6) ---

    /// StandupResult holds the events produced by a Standup.
    /// Standup gathers status from all participants and sets priority:
    /// Progress (batch) + Prioritize.
    pub fn standup(
        &mut self,
        participants: &[ActorId],
        updates: &[&str],
        lead: ActorId,
        priority: &str,
        causes: &[EventId],
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<StandupResult> {
        if participants.len() != updates.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation {
                detail: "standup: participants and updates must have equal length".to_string(),
            });
        }

        let mut result_updates = Vec::new();
        let mut last_id: Option<EventId> = None;
        for (i, actor) in participants.iter().enumerate() {
            let prev = if i == 0 {
                causes.first().cloned().ok_or_else(|| {
                    crate::errors::EventGraphError::GrammarViolation {
                        detail: "standup: causes must not be empty".to_string(),
                    }
                })?
            } else {
                last_id.clone().unwrap()
            };
            let progress = self.progress(actor.clone(), updates[i], prev, conv_id.clone(), signer)?;
            last_id = Some(progress.id.clone());
            result_updates.push(progress);
        }

        let prio = self.prioritize(lead, last_id.unwrap(), priority, conv_id, signer)?;

        Ok(StandupResult {
            updates: result_updates,
            priority: prio,
        })
    }

    /// Retrospective reviews work and identifies improvements: Review (batch) + Intend.
    pub fn retrospective(
        &mut self,
        reviewers: &[ActorId],
        assessments: &[&str],
        lead: ActorId,
        improvement: &str,
        target: EventId,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<RetrospectiveResult> {
        if reviewers.len() != assessments.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation {
                detail: "retrospective: reviewers and assessments must have equal length".to_string(),
            });
        }

        let mut reviews = Vec::new();
        let mut review_ids = Vec::new();
        for (i, reviewer) in reviewers.iter().enumerate() {
            let rev = self.review(reviewer.clone(), assessments[i], target.clone(), conv_id.clone(), signer)?;
            review_ids.push(rev.id.clone());
            reviews.push(rev);
        }

        let improvement_ev = self.intend(lead, improvement, review_ids, conv_id, signer)?;

        Ok(RetrospectiveResult {
            reviews,
            improvement: improvement_ev,
        })
    }

    /// Triage prioritises and assigns a batch of items: Prioritize + Assign + Scope (batch).
    pub fn triage(
        &mut self,
        lead: ActorId,
        items: &[EventId],
        priorities: &[&str],
        assignees: &[ActorId],
        scopes: &[DomainScope],
        weights: &[Weight],
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<TriageResult> {
        let n = items.len();
        if priorities.len() != n || assignees.len() != n || scopes.len() != n || weights.len() != n {
            return Err(crate::errors::EventGraphError::GrammarViolation {
                detail: "triage: all slices must have equal length".to_string(),
            });
        }

        let mut result = TriageResult {
            priorities: Vec::new(),
            assignments: Vec::new(),
            scopes: Vec::new(),
        };

        for i in 0..n {
            let prio = self.prioritize(lead.clone(), items[i].clone(), priorities[i], conv_id.clone(), signer)?;
            let assign = self.assign(lead.clone(), assignees[i].clone(), &scopes[i], weights[i], prio.id.clone(), conv_id.clone(), signer)?;
            let scope_ev = self.scope(lead.clone(), assignees[i].clone(), &scopes[i], weights[i], assign.id.clone(), conv_id.clone(), signer)?;
            result.priorities.push(prio);
            result.assignments.push(assign);
            result.scopes.push(scope_ev);
        }

        Ok(result)
    }

    /// Sprint plans a work cycle: Intend + Decompose + Assign (batch).
    pub fn sprint(
        &mut self,
        source: ActorId,
        goal: &str,
        subtasks: &[&str],
        assignees: &[ActorId],
        scopes: &[DomainScope],
        causes: Vec<EventId>,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<SprintResult> {
        if subtasks.len() != assignees.len() || subtasks.len() != scopes.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation {
                detail: "sprint: subtasks, assignees, and scopes must have equal length".to_string(),
            });
        }

        let intent = self.intend(source.clone(), goal, causes, conv_id.clone(), signer)?;

        let mut result_subtasks = Vec::new();
        let mut assignments = Vec::new();
        for (i, st) in subtasks.iter().enumerate() {
            let task = self.decompose(source.clone(), st, intent.id.clone(), conv_id.clone(), signer)?;
            let assign = self.assign(
                source.clone(),
                assignees[i].clone(),
                &scopes[i],
                Weight::new(0.5).unwrap(),
                task.id.clone(),
                conv_id.clone(),
                signer,
            )?;
            result_subtasks.push(task);
            assignments.push(assign);
        }

        Ok(SprintResult {
            intent,
            subtasks: result_subtasks,
            assignments,
        })
    }

    /// Escalate moves stuck work up: Block + Handoff (to higher authority).
    pub fn escalate(
        &mut self,
        source: ActorId,
        blocker: &str,
        task: EventId,
        authority: ActorId,
        scope: &DomainScope,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<EscalateResult> {
        let block_ev = self.block(source.clone(), task, blocker, conv_id.clone(), signer)?;
        let handoff_ev = self.handoff(source, authority, blocker, scope, block_ev.id.clone(), conv_id, signer)?;

        Ok(EscalateResult {
            block_event: block_ev,
            handoff_event: handoff_ev,
        })
    }

    /// DelegateAndVerify is a full delegation cycle: Assign + Scope.
    pub fn delegate_and_verify(
        &mut self,
        source: ActorId,
        assignee: ActorId,
        scope: &DomainScope,
        weight: Weight,
        cause: EventId,
        conv_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<DelegateAndVerifyResult> {
        let assign = self.assign(source.clone(), assignee.clone(), scope, weight, cause, conv_id.clone(), signer)?;
        let scope_ev = self.scope(source, assignee, scope, weight, assign.id.clone(), conv_id, signer)?;

        Ok(DelegateAndVerifyResult {
            assign_event: assign,
            scope_event: scope_ev,
        })
    }
}

pub struct StandupResult {
    pub updates: Vec<Event>,
    pub priority: Event,
}

pub struct RetrospectiveResult {
    pub reviews: Vec<Event>,
    pub improvement: Event,
}

pub struct TriageResult {
    pub priorities: Vec<Event>,
    pub assignments: Vec<Event>,
    pub scopes: Vec<Event>,
}

pub struct SprintResult {
    pub intent: Event,
    pub subtasks: Vec<Event>,
    pub assignments: Vec<Event>,
}

pub struct EscalateResult {
    pub block_event: Event,
    pub handoff_event: Event,
}

pub struct DelegateAndVerifyResult {
    pub assign_event: Event,
    pub scope_event: Event,
}
