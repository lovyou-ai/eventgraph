//! Layer 2 (Exchange) composition operations.
//!
//! 14 operations + 7 named functions for trust-based marketplaces.

use crate::errors::Result;
use crate::event::{Event, Signer};
use crate::grammar::Grammar;
use crate::store::InMemoryStore;
use crate::types::{ActorId, ConversationId, DomainScope, EventId, Weight};

/// MarketGrammar provides Layer 2 (Exchange) composition operations.
pub struct MarketGrammar<'a>(Grammar<'a>);

impl<'a> MarketGrammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self(Grammar::new(store))
    }

    // --- Operations (14) ---

    /// List publishes an offering to the market.
    pub fn list(&mut self, source: ActorId, offering: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("list: {offering}"), conv_id, causes, signer)
    }

    /// Bid makes a counter-offer on a listing.
    pub fn bid(&mut self, source: ActorId, offer: &str, listing: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.respond(source, &format!("bid: {offer}"), listing, conv_id, signer)
    }

    /// Inquire asks for clarification about an offering.
    pub fn inquire(&mut self, source: ActorId, question: &str, listing: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.respond(source, &format!("inquire: {question}"), listing, conv_id, signer)
    }

    /// Negotiate opens a channel for refining terms.
    pub fn negotiate(&mut self, source: ActorId, counterparty: ActorId, scope: Option<&DomainScope>, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.channel(source, counterparty, scope, cause, conv_id, signer)
    }

    /// Accept accepts terms, creating mutual obligation.
    pub fn accept(&mut self, buyer: ActorId, seller: ActorId, terms: &str, scope: &DomainScope, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.consent(buyer, seller, &format!("accept: {terms}"), scope, cause, conv_id, signer)
    }

    /// Decline rejects an offer.
    pub fn decline(&mut self, source: ActorId, reason: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("decline: {reason}"), conv_id, causes, signer)
    }

    /// Invoice formalizes a payment obligation.
    pub fn invoice(&mut self, source: ActorId, description: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("invoice: {description}"), conv_id, causes, signer)
    }

    /// Pay satisfies a financial obligation.
    pub fn pay(&mut self, source: ActorId, description: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("pay: {description}"), conv_id, causes, signer)
    }

    /// Deliver satisfies a service/goods obligation.
    pub fn deliver(&mut self, source: ActorId, description: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("deliver: {description}"), conv_id, causes, signer)
    }

    /// Confirm acknowledges receipt and satisfaction.
    pub fn confirm(&mut self, source: ActorId, confirmation: &str, causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.emit(source, &format!("confirm: {confirmation}"), conv_id, causes, signer)
    }

    /// Rate provides structured feedback on an exchange.
    pub fn rate(&mut self, source: ActorId, target: EventId, target_actor: ActorId, weight: Weight, scope: Option<&DomainScope>, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.endorse(source, target, target_actor, weight, scope, conv_id, signer)
    }

    /// Dispute flags a failed obligation.
    pub fn dispute(&mut self, source: ActorId, complaint: &str, target: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        let (_, flag) = self.0.challenge(source, &format!("dispute: {complaint}"), target, conv_id, signer)?;
        Ok(flag)
    }

    /// Escrow holds value pending conditions.
    pub fn escrow(&mut self, source: ActorId, escrow_actor: ActorId, scope: &DomainScope, weight: Weight, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.delegate(source, escrow_actor, scope, weight, cause, conv_id, signer)
    }

    /// Release releases escrowed value on condition.
    pub fn release(&mut self, party_a: ActorId, party_b: ActorId, terms: &str, scope: &DomainScope, cause: EventId, conv_id: ConversationId, signer: &dyn Signer) -> Result<Event> {
        self.0.consent(party_a, party_b, &format!("release: {terms}"), scope, cause, conv_id, signer)
    }

    // --- Named Functions (7) ---

    /// Auction runs competitive bidding: List + Bid (multiple) + Accept (highest).
    pub fn auction(
        &mut self, seller: ActorId, offering: &str,
        bidders: &[ActorId], bids: &[&str], winner_idx: usize,
        scope: &DomainScope, causes: Vec<EventId>,
        conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<AuctionResult> {
        if bidders.len() != bids.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation { detail: "auction: bidders and bids must have equal length".to_string() });
        }
        if winner_idx >= bidders.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation { detail: "auction: winner_idx out of range".to_string() });
        }
        let listing = self.list(seller.clone(), offering, causes, conv_id.clone(), signer)?;
        let mut bid_events = Vec::new();
        for (i, bidder) in bidders.iter().enumerate() {
            let b = self.bid(bidder.clone(), bids[i], listing.id.clone(), conv_id.clone(), signer)?;
            bid_events.push(b);
        }
        let acceptance = self.accept(bidders[winner_idx].clone(), seller, &format!("auction won: {}", bids[winner_idx]), scope, bid_events[winner_idx].id.clone(), conv_id, signer)?;
        Ok(AuctionResult { listing, bids: bid_events, acceptance })
    }

    /// Milestone is staged delivery and payment: Accept + Deliver (partial) + Pay (partial).
    pub fn milestone(
        &mut self, buyer: ActorId, seller: ActorId,
        terms: &str, milestones: &[&str], payments: &[&str],
        scope: &DomainScope, cause: EventId,
        conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<MilestoneResult> {
        if milestones.len() != payments.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation { detail: "milestone: milestones and payments must have equal length".to_string() });
        }
        let acceptance = self.accept(buyer.clone(), seller.clone(), terms, scope, cause, conv_id.clone(), signer)?;
        let mut deliveries = Vec::new();
        let mut payment_events = Vec::new();
        let mut prev = acceptance.id.clone();
        for i in 0..milestones.len() {
            let d = self.deliver(seller.clone(), milestones[i], vec![prev], conv_id.clone(), signer)?;
            let p = self.pay(buyer.clone(), payments[i], vec![d.id.clone()], conv_id.clone(), signer)?;
            prev = p.id.clone();
            deliveries.push(d);
            payment_events.push(p);
        }
        Ok(MilestoneResult { acceptance, deliveries, payments: payment_events })
    }

    /// Barter exchanges goods for goods: List + Bid (goods) + Accept.
    pub fn barter(
        &mut self, party_a: ActorId, party_b: ActorId,
        offer_a: &str, offer_b: &str, scope: &DomainScope,
        causes: Vec<EventId>, conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<BarterResult> {
        let listing = self.list(party_a.clone(), offer_a, causes, conv_id.clone(), signer)?;
        let counter = self.bid(party_b.clone(), offer_b, listing.id.clone(), conv_id.clone(), signer)?;
        let acceptance = self.accept(party_a, party_b, &format!("barter: {offer_a} for {offer_b}"), scope, counter.id.clone(), conv_id, signer)?;
        Ok(BarterResult { listing, counter_offer: counter, acceptance })
    }

    /// Subscription creates recurring delivery and payment.
    pub fn subscription(
        &mut self, subscriber: ActorId, provider: ActorId,
        terms: &str, periods: &[&str], deliveries: &[&str],
        scope: &DomainScope, cause: EventId,
        conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<SubscriptionResult> {
        if periods.len() != deliveries.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation { detail: "subscription: periods and deliveries must have equal length".to_string() });
        }
        let acceptance = self.accept(subscriber.clone(), provider.clone(), terms, scope, cause, conv_id.clone(), signer)?;
        let mut payment_events = Vec::new();
        let mut delivery_events = Vec::new();
        let mut prev = acceptance.id.clone();
        for i in 0..periods.len() {
            let p = self.pay(subscriber.clone(), periods[i], vec![prev], conv_id.clone(), signer)?;
            let d = self.deliver(provider.clone(), deliveries[i], vec![p.id.clone()], conv_id.clone(), signer)?;
            prev = d.id.clone();
            payment_events.push(p);
            delivery_events.push(d);
        }
        Ok(SubscriptionResult { acceptance, payments: payment_events, deliveries: delivery_events })
    }

    /// Refund processes a return: Dispute + resolution + Pay (reversed).
    pub fn refund(
        &mut self, buyer: ActorId, seller: ActorId,
        complaint: &str, resolution: &str, refund_amount: &str,
        target: EventId, conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<RefundResult> {
        let dispute_ev = self.dispute(buyer.clone(), complaint, target, conv_id.clone(), signer)?;
        let resolution_ev = self.0.emit(seller.clone(), &format!("resolution: {resolution}"), conv_id.clone(), vec![dispute_ev.id.clone()], signer)?;
        let reversal = self.pay(seller, &format!("refund: {refund_amount}"), vec![resolution_ev.id.clone()], conv_id, signer)?;
        Ok(RefundResult { dispute: dispute_ev, resolution: resolution_ev, reversal })
    }

    /// ReputationTransfer collects ratings from multiple parties.
    pub fn reputation_transfer(
        &mut self, raters: &[ActorId], targets: &[EventId], target_actor: ActorId,
        weights: &[Weight], scope: Option<&DomainScope>,
        conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<ReputationTransferResult> {
        if raters.len() != targets.len() || raters.len() != weights.len() {
            return Err(crate::errors::EventGraphError::GrammarViolation { detail: "reputation-transfer: raters, targets, and weights must have equal length".to_string() });
        }
        let mut ratings = Vec::new();
        for (i, rater) in raters.iter().enumerate() {
            let r = self.rate(rater.clone(), targets[i].clone(), target_actor.clone(), weights[i], scope, conv_id.clone(), signer)?;
            ratings.push(r);
        }
        Ok(ReputationTransferResult { ratings })
    }

    /// Arbitration resolves a dispute with escrow: Dispute + Escrow + Release.
    pub fn arbitration(
        &mut self, plaintiff: ActorId, defendant: ActorId, arbiter: ActorId,
        complaint: &str, scope: &DomainScope, weight: Weight,
        target: EventId, conv_id: ConversationId, signer: &dyn Signer,
    ) -> Result<ArbitrationResult> {
        let dispute_ev = self.dispute(plaintiff.clone(), complaint, target, conv_id.clone(), signer)?;
        let escrow_ev = self.escrow(defendant, arbiter.clone(), scope, weight, dispute_ev.id.clone(), conv_id.clone(), signer)?;
        let release_ev = self.release(arbiter, plaintiff, "arbitration resolved", scope, escrow_ev.id.clone(), conv_id, signer)?;
        Ok(ArbitrationResult { dispute: dispute_ev, escrow: escrow_ev, release: release_ev })
    }
}

pub struct AuctionResult { pub listing: Event, pub bids: Vec<Event>, pub acceptance: Event }
pub struct MilestoneResult { pub acceptance: Event, pub deliveries: Vec<Event>, pub payments: Vec<Event> }
pub struct BarterResult { pub listing: Event, pub counter_offer: Event, pub acceptance: Event }
pub struct SubscriptionResult { pub acceptance: Event, pub payments: Vec<Event>, pub deliveries: Vec<Event> }
pub struct RefundResult { pub dispute: Event, pub resolution: Event, pub reversal: Event }
pub struct ReputationTransferResult { pub ratings: Vec<Event> }
pub struct ArbitrationResult { pub dispute: Event, pub escrow: Event, pub release: Event }
