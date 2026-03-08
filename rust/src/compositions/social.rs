//! Layer 3 (Society) composition operations.
//!
//! 5 society-specific extensions + 4 named functions for user-owned social platforms.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, DomainScope, EdgeId, EventId, Weight};

/// SocialGrammar provides Layer 3 (Society) composition operations.
pub struct SocialGrammar<'a>(Grammar<'a>);

impl<'a> SocialGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    /// Norm establishes a shared behavioural expectation.
    pub fn norm(&mut self, proposer: ActorId, supporter: ActorId, norm: &str, scope: &DomainScope, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.consent(proposer, supporter, &format!("norm: {norm}"), scope, cause, conv_id, signer)
    }

    /// Moderate enforces community norms on content.
    pub fn moderate(&mut self, moderator: ActorId, target: EventId, action: &str, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.annotate(moderator, target, "moderation", action, conv_id, signer)
    }

    /// Elect assigns a community role through collective decision.
    pub fn elect(&mut self, community: ActorId, elected: ActorId, role: &str, scope: &DomainScope, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.consent(community, elected, &format!("elect: {role}"), scope, cause, conv_id, signer)
    }

    /// Welcome is structured onboarding of a new member. Returns (endorse, subscribe).
    pub fn welcome(&mut self, sponsor: ActorId, newcomer: ActorId, weight: Weight, scope: Option<&DomainScope>, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<(Event, Event)> {
        self.0.invite(sponsor, newcomer, weight, scope, cause, conv_id, signer)
    }

    /// Exile is structured removal of a member: Emit + Sever + Annotate.
    pub fn exile(&mut self, moderator: ActorId, edge: EdgeId, reason: &str, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<ExileResult> {
        let exclusion = self.0.emit(moderator.clone(), &format!("exile: {reason}"), conv_id.clone(), vec![cause], signer)?;
        let edge_event_id = EventId::new(edge.value())?;
        let sever_ev = self.0.sever(moderator.clone(), edge_event_id, exclusion.id.clone(), conv_id.clone(), signer)?;
        let sanction = self.0.annotate(moderator, sever_ev.id.clone(), "sanction", reason, conv_id, signer)?;
        Ok(ExileResult { exclusion, sever: sever_ev, sanction })
    }

    // --- Named Functions ---

    /// Poll runs a quick community sentiment check: Emit (proposed) + Consent (batch).
    pub fn poll(&mut self, proposer: ActorId, question: &str, voters: &[ActorId], scope: &DomainScope, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<PollResult> {
        let proposal = self.0.emit(proposer.clone(), &format!("poll: {question}"), conv_id.clone(), vec![cause], signer)?;
        let mut votes = Vec::new();
        for voter in voters {
            let vote = self.0.consent(voter.clone(), proposer.clone(), &format!("vote: {question}"), scope, proposal.id.clone(), conv_id.clone(), signer)?;
            votes.push(vote);
        }
        Ok(PollResult { proposal, votes })
    }

    /// Federation creates cooperation between communities.
    pub fn federation(&mut self, community_a: ActorId, community_b: ActorId, terms: &str, scope: &DomainScope, weight: Weight, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<FederationResult> {
        let agreement = self.0.consent(community_a.clone(), community_b.clone(), &format!("federation: {terms}"), scope, cause, conv_id.clone(), signer)?;
        let delegation = self.0.delegate(community_a, community_b, scope, weight, agreement.id.clone(), conv_id, signer)?;
        Ok(FederationResult { agreement, delegation })
    }

    /// Schism splits a community over a conflicting norm.
    pub fn schism(&mut self, faction: ActorId, moderator: ActorId, conflicting_norm: &str, _scope: &DomainScope, edge: EdgeId, reason: &str, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<SchismResult> {
        let norm_ev = self.0.emit(faction.clone(), &format!("conflicting-norm: {conflicting_norm}"), conv_id.clone(), vec![cause], signer)?;
        let exile_result = self.exile(moderator, edge, reason, norm_ev.id.clone(), conv_id.clone(), signer)?;
        let community = self.0.emit(faction, &format!("new-community: split over {conflicting_norm}"), conv_id, vec![exile_result.sanction.id.clone()], signer)?;
        Ok(SchismResult { conflicting_norm: norm_ev, exile: exile_result, new_community: community })
    }
}

pub struct ExileResult { pub exclusion: Event, pub sever: Event, pub sanction: Event }
pub struct PollResult { pub proposal: Event, pub votes: Vec<Event> }
pub struct FederationResult { pub agreement: Event, pub delegation: Event }
pub struct SchismResult { pub conflicting_norm: Event, pub exile: ExileResult, pub new_community: Event }
