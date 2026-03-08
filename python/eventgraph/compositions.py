"""Composition Grammars — domain-specific operations built on the base Grammar.

Ports all 13 composition grammars from the Go reference implementation.
Each grammar wraps a Grammar instance and provides domain-specific operations
that delegate to Grammar methods with prefix strings.

Layer 1:  WorkGrammar       (Agency)
Layer 2:  MarketGrammar     (Exchange)
Layer 3:  SocialGrammar     (Society)
Layer 4:  JusticeGrammar    (Legal)
Layer 5:  BuildGrammar      (Technology)
Layer 6:  KnowledgeGrammar  (Information)
Layer 7:  AlignmentGrammar  (Ethics)
Layer 8:  IdentityGrammar   (Identity)
Layer 9:  BondGrammar       (Relationship)
Layer 10: BelongingGrammar  (Community)
Layer 11: MeaningGrammar    (Culture)
Layer 12: EvolutionGrammar  (Emergence)
Layer 13: BeingGrammar      (Existence)
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Sequence

from .event import Event, Signer
from .grammar import Grammar
from .types import (
    ActorID,
    ConversationID,
    DomainScope,
    EdgeID,
    EventID,
    Option,
    Weight,
)


# =============================================================================
# Layer 1: WorkGrammar (Agency)
# =============================================================================


@dataclass(frozen=True)
class StandupResult:
    updates: list[Event]
    priority: Event


@dataclass(frozen=True)
class RetrospectiveResult:
    reviews: list[Event]
    improvement: Event


@dataclass(frozen=True)
class TriageResult:
    priorities: list[Event]
    assignments: list[Event]
    scopes: list[Event]


@dataclass(frozen=True)
class SprintResult:
    intent: Event
    subtasks: list[Event]
    assignments: list[Event]


@dataclass(frozen=True)
class EscalateResult:
    block_event: Event
    handoff_event: Event


@dataclass(frozen=True)
class DelegateAndVerifyResult:
    assign_event: Event
    scope_event: Event


class WorkGrammar:
    """Layer 1 (Agency) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (12) ---

    def intend(
        self, source: ActorID, goal: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Declare a goal or desired outcome."""
        return self._g.emit(source, "intend: " + goal, conv_id, causes, signer)

    def decompose(
        self, source: ActorID, subtask: str,
        goal: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Break a goal into actionable steps."""
        return self._g.derive(source, "decompose: " + subtask, goal, conv_id, signer)

    def assign(
        self, source: ActorID, assignee: ActorID,
        scope: DomainScope, weight: Weight,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Give work to a specific actor."""
        return self._g.delegate(source, assignee, scope, weight, cause, conv_id, signer)

    def claim(
        self, source: ActorID, work: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Take on unassigned work."""
        return self._g.emit(source, "claim: " + work, conv_id, causes, signer)

    def prioritize(
        self, source: ActorID, target: EventID,
        priority: str, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Rank work by importance."""
        return self._g.annotate(source, target, "priority", priority, conv_id, signer)

    def block(
        self, source: ActorID, target: EventID,
        blocker: str, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Flag work that cannot proceed."""
        return self._g.annotate(source, target, "blocked", blocker, conv_id, signer)

    def unblock(
        self, source: ActorID, resolution: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Remove an impediment to work."""
        return self._g.emit(source, "unblock: " + resolution, conv_id, causes, signer)

    def progress(
        self, source: ActorID, update: str,
        previous: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Report incremental advancement."""
        return self._g.extend(source, "progress: " + update, previous, conv_id, signer)

    def complete(
        self, source: ActorID, summary: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Mark work as done with evidence."""
        return self._g.emit(source, "complete: " + summary, conv_id, causes, signer)

    def handoff(
        self, from_actor: ActorID, to_actor: ActorID,
        description: str, scope: DomainScope,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Transfer work between actors."""
        return self._g.consent(from_actor, to_actor, "handoff: " + description, scope, cause, conv_id, signer)

    def scope(
        self, source: ActorID, target: ActorID,
        scope: DomainScope, weight: Weight,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Define what an actor may do autonomously."""
        return self._g.delegate(source, target, scope, weight, cause, conv_id, signer)

    def review(
        self, source: ActorID, assessment: str,
        target: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Evaluate completed work."""
        return self._g.respond(source, "review: " + assessment, target, conv_id, signer)

    # --- Named Functions (6) ---

    def standup(
        self, participants: list[ActorID], updates: list[str],
        lead: ActorID, priority: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> StandupResult:
        """Gather status and set priority: Progress (batch) + Prioritize."""
        if len(participants) != len(updates):
            raise ValueError("standup: participants and updates must have equal length")

        result_updates: list[Event] = []
        last_id: EventID | None = None
        for i, actor in enumerate(participants):
            prev: EventID
            if i == 0:
                prev = causes[0] if causes else None  # type: ignore
            else:
                prev = last_id  # type: ignore
            prog = self.progress(actor, updates[i], prev, conv_id, signer)
            result_updates.append(prog)
            last_id = prog.id

        prio = self.prioritize(lead, last_id, priority, conv_id, signer)  # type: ignore
        return StandupResult(updates=result_updates, priority=prio)

    def retrospective(
        self, reviewers: list[ActorID], assessments: list[str],
        lead: ActorID, improvement: str,
        target: EventID, conv_id: ConversationID, signer: Signer,
    ) -> RetrospectiveResult:
        """Review work and identify improvements: Review (batch) + Intend."""
        if len(reviewers) != len(assessments):
            raise ValueError("retrospective: reviewers and assessments must have equal length")

        reviews: list[Event] = []
        review_ids: list[EventID] = []
        for i, reviewer in enumerate(reviewers):
            rev = self.review(reviewer, assessments[i], target, conv_id, signer)
            reviews.append(rev)
            review_ids.append(rev.id)

        improve = self.intend(lead, improvement, review_ids, conv_id, signer)
        return RetrospectiveResult(reviews=reviews, improvement=improve)

    def triage(
        self, lead: ActorID,
        items: list[EventID], priorities: list[str],
        assignees: list[ActorID], scopes: list[DomainScope], weights: list[Weight],
        conv_id: ConversationID, signer: Signer,
    ) -> TriageResult:
        """Prioritize and assign a batch: Prioritize + Assign + Scope (batch)."""
        n = len(items)
        if len(priorities) != n or len(assignees) != n or len(scopes) != n or len(weights) != n:
            raise ValueError("triage: all lists must have equal length")

        result_priorities: list[Event] = []
        result_assignments: list[Event] = []
        result_scopes: list[Event] = []
        for i in range(n):
            prio = self.prioritize(lead, items[i], priorities[i], conv_id, signer)
            result_priorities.append(prio)

            assign_ev = self.assign(lead, assignees[i], scopes[i], weights[i], prio.id, conv_id, signer)
            result_assignments.append(assign_ev)

            scope_ev = self.scope(lead, assignees[i], scopes[i], weights[i], assign_ev.id, conv_id, signer)
            result_scopes.append(scope_ev)

        return TriageResult(priorities=result_priorities, assignments=result_assignments, scopes=result_scopes)

    def sprint(
        self, source: ActorID, goal: str,
        subtasks: list[str], assignees: list[ActorID], scopes: list[DomainScope],
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> SprintResult:
        """Plan a work cycle: Intend + Decompose + Assign (batch)."""
        if len(subtasks) != len(assignees) or len(subtasks) != len(scopes):
            raise ValueError("sprint: subtasks, assignees, and scopes must have equal length")

        intent = self.intend(source, goal, causes, conv_id, signer)
        result_subtasks: list[Event] = []
        result_assignments: list[Event] = []
        for i, st in enumerate(subtasks):
            task = self.decompose(source, st, intent.id, conv_id, signer)
            result_subtasks.append(task)
            assign_ev = self.assign(source, assignees[i], scopes[i], Weight(0.5), task.id, conv_id, signer)
            result_assignments.append(assign_ev)

        return SprintResult(intent=intent, subtasks=result_subtasks, assignments=result_assignments)

    def escalate(
        self, source: ActorID, blocker: str,
        task: EventID, authority: ActorID, scope: DomainScope,
        conv_id: ConversationID, signer: Signer,
    ) -> EscalateResult:
        """Move stuck work up: Block + Handoff."""
        block_ev = self.block(source, task, blocker, conv_id, signer)
        handoff_ev = self.handoff(source, authority, blocker, scope, block_ev.id, conv_id, signer)
        return EscalateResult(block_event=block_ev, handoff_event=handoff_ev)

    def delegate_and_verify(
        self, source: ActorID, assignee: ActorID,
        scope: DomainScope, weight: Weight,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> DelegateAndVerifyResult:
        """Full delegation cycle: Assign + Scope."""
        assign_ev = self.assign(source, assignee, scope, weight, cause, conv_id, signer)
        scope_ev = self.scope(source, assignee, scope, weight, assign_ev.id, conv_id, signer)
        return DelegateAndVerifyResult(assign_event=assign_ev, scope_event=scope_ev)


# =============================================================================
# Layer 2: MarketGrammar (Exchange)
# =============================================================================


@dataclass(frozen=True)
class AuctionResult:
    listing: Event
    bids: list[Event]
    acceptance: Event


@dataclass(frozen=True)
class MilestoneResult:
    acceptance: Event
    deliveries: list[Event]
    payments: list[Event]


@dataclass(frozen=True)
class BarterResult:
    listing: Event
    counter_offer: Event
    acceptance: Event


@dataclass(frozen=True)
class MarketSubscriptionResult:
    acceptance: Event
    payments: list[Event]
    deliveries: list[Event]


@dataclass(frozen=True)
class RefundResult:
    dispute: Event
    resolution: Event
    reversal: Event


@dataclass(frozen=True)
class ReputationTransferResult:
    ratings: list[Event]


@dataclass(frozen=True)
class ArbitrationResult:
    dispute: Event
    escrow: Event
    release: Event


class MarketGrammar:
    """Layer 2 (Exchange) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (14) ---

    def list_offering(
        self, source: ActorID, offering: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Publish an offering to the market."""
        return self._g.emit(source, "list: " + offering, conv_id, causes, signer)

    def bid(
        self, source: ActorID, offer: str,
        listing: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Make a counter-offer on a listing."""
        return self._g.respond(source, "bid: " + offer, listing, conv_id, signer)

    def inquire(
        self, source: ActorID, question: str,
        listing: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Ask for clarification about an offering."""
        return self._g.respond(source, "inquire: " + question, listing, conv_id, signer)

    def negotiate(
        self, source: ActorID, counterparty: ActorID,
        scope: Option[DomainScope],
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Open a channel for refining terms."""
        return self._g.channel(source, counterparty, scope, cause, conv_id, signer)

    def accept(
        self, buyer: ActorID, seller: ActorID,
        terms: str, scope: DomainScope,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Accept terms, creating mutual obligation."""
        return self._g.consent(buyer, seller, "accept: " + terms, scope, cause, conv_id, signer)

    def decline(
        self, source: ActorID, reason: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Reject an offer."""
        return self._g.emit(source, "decline: " + reason, conv_id, causes, signer)

    def invoice(
        self, source: ActorID, description: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Formalize a payment obligation."""
        return self._g.emit(source, "invoice: " + description, conv_id, causes, signer)

    def pay(
        self, source: ActorID, description: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Satisfy a financial obligation."""
        return self._g.emit(source, "pay: " + description, conv_id, causes, signer)

    def deliver(
        self, source: ActorID, description: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Satisfy a service/goods obligation."""
        return self._g.emit(source, "deliver: " + description, conv_id, causes, signer)

    def confirm(
        self, source: ActorID, confirmation: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Acknowledge receipt and satisfaction."""
        return self._g.emit(source, "confirm: " + confirmation, conv_id, causes, signer)

    def rate(
        self, source: ActorID,
        target: EventID, target_actor: ActorID,
        weight: Weight, scope: Option[DomainScope],
        conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Provide structured feedback on an exchange."""
        return self._g.endorse(source, target, target_actor, weight, scope, conv_id, signer)

    def dispute(
        self, source: ActorID, complaint: str,
        target: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Flag a failed obligation."""
        _, flag = self._g.challenge(source, "dispute: " + complaint, target, conv_id, signer)
        return flag

    def escrow(
        self, source: ActorID, escrow_actor: ActorID,
        scope: DomainScope, weight: Weight,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Hold value pending conditions."""
        return self._g.delegate(source, escrow_actor, scope, weight, cause, conv_id, signer)

    def release(
        self, party_a: ActorID, party_b: ActorID,
        terms: str, scope: DomainScope,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Release escrowed value on condition."""
        return self._g.consent(party_a, party_b, "release: " + terms, scope, cause, conv_id, signer)

    # --- Named Functions (7) ---

    def auction(
        self, seller: ActorID, offering: str,
        bidders: list[ActorID], bids: list[str],
        winner_idx: int, scope: DomainScope,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> AuctionResult:
        """Run competitive bidding: List + Bid (multiple) + Accept (highest)."""
        if len(bidders) != len(bids):
            raise ValueError("auction: bidders and bids must have equal length")
        if winner_idx < 0 or winner_idx >= len(bidders):
            raise ValueError("auction: winner_idx out of range")

        listing = self.list_offering(seller, offering, causes, conv_id, signer)
        result_bids: list[Event] = []
        for i, bidder in enumerate(bidders):
            b = self.bid(bidder, bids[i], listing.id, conv_id, signer)
            result_bids.append(b)

        acceptance = self.accept(
            bidders[winner_idx], seller,
            "auction won: " + bids[winner_idx], scope,
            result_bids[winner_idx].id, conv_id, signer,
        )
        return AuctionResult(listing=listing, bids=result_bids, acceptance=acceptance)

    def milestone(
        self, buyer: ActorID, seller: ActorID,
        terms: str, milestones: list[str], payments: list[str],
        scope: DomainScope,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> MilestoneResult:
        """Staged delivery and payment: Accept + Deliver (partial) + Pay (partial)."""
        if len(milestones) != len(payments):
            raise ValueError("milestone: milestones and payments must have equal length")

        acceptance = self.accept(buyer, seller, terms, scope, cause, conv_id, signer)
        result_deliveries: list[Event] = []
        result_payments: list[Event] = []
        prev = acceptance.id
        for i in range(len(milestones)):
            delivery = self.deliver(seller, milestones[i], [prev], conv_id, signer)
            result_deliveries.append(delivery)
            payment = self.pay(buyer, payments[i], [delivery.id], conv_id, signer)
            result_payments.append(payment)
            prev = payment.id

        return MilestoneResult(acceptance=acceptance, deliveries=result_deliveries, payments=result_payments)

    def barter(
        self, party_a: ActorID, party_b: ActorID,
        offer_a: str, offer_b: str, scope: DomainScope,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> BarterResult:
        """Exchange goods for goods: List + Bid (goods) + Accept."""
        listing = self.list_offering(party_a, offer_a, causes, conv_id, signer)
        counter = self.bid(party_b, offer_b, listing.id, conv_id, signer)
        acceptance = self.accept(
            party_a, party_b,
            "barter: " + offer_a + " for " + offer_b, scope,
            counter.id, conv_id, signer,
        )
        return BarterResult(listing=listing, counter_offer=counter, acceptance=acceptance)

    def subscription(
        self, subscriber: ActorID, provider: ActorID,
        terms: str, periods: list[str], deliveries: list[str],
        scope: DomainScope,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> MarketSubscriptionResult:
        """Recurring delivery and payment: Accept + Pay + Deliver (repeated)."""
        if len(periods) != len(deliveries):
            raise ValueError("subscription: periods and deliveries must have equal length")

        acceptance = self.accept(subscriber, provider, terms, scope, cause, conv_id, signer)
        result_payments: list[Event] = []
        result_deliveries: list[Event] = []
        prev = acceptance.id
        for i in range(len(periods)):
            payment = self.pay(subscriber, periods[i], [prev], conv_id, signer)
            result_payments.append(payment)
            delivery = self.deliver(provider, deliveries[i], [payment.id], conv_id, signer)
            result_deliveries.append(delivery)
            prev = delivery.id

        return MarketSubscriptionResult(acceptance=acceptance, payments=result_payments, deliveries=result_deliveries)

    def refund(
        self, buyer: ActorID, seller: ActorID,
        complaint: str, resolution: str, refund_amount: str,
        target: EventID, conv_id: ConversationID, signer: Signer,
    ) -> RefundResult:
        """Process a return: Dispute + resolution + Pay (reversed)."""
        dispute_ev = self.dispute(buyer, complaint, target, conv_id, signer)
        resolution_ev = self._g.emit(seller, "resolution: " + resolution, conv_id, [dispute_ev.id], signer)
        reversal = self.pay(seller, "refund: " + refund_amount, [resolution_ev.id], conv_id, signer)
        return RefundResult(dispute=dispute_ev, resolution=resolution_ev, reversal=reversal)

    def reputation_transfer(
        self, raters: list[ActorID], targets: list[EventID], target_actor: ActorID,
        weights: list[Weight], scope: Option[DomainScope],
        conv_id: ConversationID, signer: Signer,
    ) -> ReputationTransferResult:
        """Collect ratings from multiple parties: Rate (batch)."""
        if len(raters) != len(targets) or len(raters) != len(weights):
            raise ValueError("reputation-transfer: raters, targets, and weights must have equal length")

        ratings: list[Event] = []
        for i, rater in enumerate(raters):
            rating = self.rate(rater, targets[i], target_actor, weights[i], scope, conv_id, signer)
            ratings.append(rating)
        return ReputationTransferResult(ratings=ratings)

    def arbitration(
        self, plaintiff: ActorID, defendant: ActorID,
        arbiter: ActorID, complaint: str,
        scope: DomainScope, weight: Weight,
        target: EventID, conv_id: ConversationID, signer: Signer,
    ) -> ArbitrationResult:
        """Resolve a dispute with escrow: Dispute + Escrow + Release."""
        dispute_ev = self.dispute(plaintiff, complaint, target, conv_id, signer)
        escrow_ev = self.escrow(defendant, arbiter, scope, weight, dispute_ev.id, conv_id, signer)
        release_ev = self.release(arbiter, plaintiff, "arbitration resolved", scope, escrow_ev.id, conv_id, signer)
        return ArbitrationResult(dispute=dispute_ev, escrow=escrow_ev, release=release_ev)


# =============================================================================
# Layer 3: SocialGrammar (Society)
# =============================================================================


@dataclass(frozen=True)
class ExileResult:
    exclusion: Event
    sever: Event
    sanction: Event


@dataclass(frozen=True)
class PollResult:
    proposal: Event
    votes: list[Event]


@dataclass(frozen=True)
class FederationResult:
    agreement: Event
    delegation: Event


@dataclass(frozen=True)
class SchismResult:
    conflicting_norm: Event
    exile: ExileResult
    new_community: Event


class SocialGrammar:
    """Layer 3 (Society) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (5) ---

    def norm(
        self, proposer: ActorID, supporter: ActorID,
        norm_text: str, scope: DomainScope,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Establish a shared behavioural expectation."""
        return self._g.consent(proposer, supporter, "norm: " + norm_text, scope, cause, conv_id, signer)

    def moderate(
        self, moderator: ActorID, target: EventID,
        action: str, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Enforce community norms on content."""
        return self._g.annotate(moderator, target, "moderation", action, conv_id, signer)

    def elect(
        self, community: ActorID, elected: ActorID,
        role: str, scope: DomainScope,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> Event:
        """Assign a community role through collective decision."""
        return self._g.consent(community, elected, "elect: " + role, scope, cause, conv_id, signer)

    def welcome(
        self, sponsor: ActorID, newcomer: ActorID,
        weight: Weight, scope: Option[DomainScope],
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> tuple[Event, Event]:
        """Structured onboarding of a new member."""
        return self._g.invite(sponsor, newcomer, weight, scope, cause, conv_id, signer)

    def exile(
        self, moderator: ActorID,
        edge: EdgeID, reason: str,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> ExileResult:
        """Structured removal of a member."""
        exclusion = self._g.emit(moderator, "exile: " + reason, conv_id, [cause], signer)
        sever_ev = self._g.sever(moderator, edge, exclusion.id, conv_id, signer)
        sanction = self._g.annotate(moderator, sever_ev.id, "sanction", reason, conv_id, signer)
        return ExileResult(exclusion=exclusion, sever=sever_ev, sanction=sanction)

    # --- Named Functions (4) ---

    def poll(
        self, proposer: ActorID, question: str,
        voters: list[ActorID], scope: DomainScope,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> PollResult:
        """Quick community sentiment check: Propose + Consent (batch)."""
        proposal = self._g.emit(proposer, "poll: " + question, conv_id, [cause], signer)
        votes: list[Event] = []
        for voter in voters:
            vote = self._g.consent(voter, proposer, "vote: " + question, scope, proposal.id, conv_id, signer)
            votes.append(vote)
        return PollResult(proposal=proposal, votes=votes)

    def federation(
        self, community_a: ActorID, community_b: ActorID,
        terms: str, scope: DomainScope, weight: Weight,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> FederationResult:
        """Create cooperation between communities."""
        agreement = self._g.consent(community_a, community_b, "federation: " + terms, scope, cause, conv_id, signer)
        delegation = self._g.delegate(community_a, community_b, scope, weight, agreement.id, conv_id, signer)
        return FederationResult(agreement=agreement, delegation=delegation)

    def schism(
        self, faction: ActorID, moderator: ActorID,
        conflicting_norm: str, scope: DomainScope,
        edge: EdgeID, reason: str,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> SchismResult:
        """Split a community over a conflicting norm."""
        norm_ev = self._g.emit(faction, "conflicting-norm: " + conflicting_norm, conv_id, [cause], signer)
        exile_result = self.exile(moderator, edge, reason, norm_ev.id, conv_id, signer)
        new_community = self._g.emit(
            faction, "new-community: split over " + conflicting_norm,
            conv_id, [exile_result.sanction.id], signer,
        )
        return SchismResult(conflicting_norm=norm_ev, exile=exile_result, new_community=new_community)


# =============================================================================
# Layer 4: JusticeGrammar (Legal)
# =============================================================================


@dataclass(frozen=True)
class TrialResultJ:
    filing: Event
    submissions: list[Event]
    arguments: list[Event]
    ruling: Event


@dataclass(frozen=True)
class ConstitutionalAmendmentResult:
    reform: Event
    legislation: Event
    rights_check: Event


@dataclass(frozen=True)
class InjunctionResult:
    filing: Event
    ruling: Event
    enforcement: Event


@dataclass(frozen=True)
class PleaResult:
    filing: Event
    acceptance: Event
    enforcement: Event


@dataclass(frozen=True)
class ClassActionResult:
    filings: list[Event]
    merged: Event
    trial: TrialResultJ


@dataclass(frozen=True)
class RecallResult:
    audit: Event
    filing: Event
    consent: Event
    revocation: Event


class JusticeGrammar:
    """Layer 4 (Legal) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (12) ---

    def legislate(self, source: ActorID, rule: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "legislate: " + rule, conv_id, causes, signer)

    def amend(self, source: ActorID, amendment: str, rule: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "amend: " + amendment, rule, conv_id, signer)

    def repeal(self, source: ActorID, rule: EventID, reason: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.retract(source, rule, reason, conv_id, signer)

    def file(self, source: ActorID, complaint: str, target: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        _, flag = self._g.challenge(source, "file: " + complaint, target, conv_id, signer)
        return flag

    def submit(self, source: ActorID, target: EventID, evidence: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "evidence", evidence, conv_id, signer)

    def argue(self, source: ActorID, argument: str, target: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.respond(source, "argue: " + argument, target, conv_id, signer)

    def judge(self, source: ActorID, ruling: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "judge: " + ruling, conv_id, causes, signer)

    def appeal(self, source: ActorID, grounds: str, ruling: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        _, flag = self._g.challenge(source, "appeal: " + grounds, ruling, conv_id, signer)
        return flag

    def enforce(self, source: ActorID, executor: ActorID, scope: DomainScope, weight: Weight, cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.delegate(source, executor, scope, weight, cause, conv_id, signer)

    def audit(self, source: ActorID, target: EventID, findings: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "audit", findings, conv_id, signer)

    def pardon(self, authority: ActorID, pardoned: ActorID, terms: str, scope: DomainScope, cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.consent(authority, pardoned, "pardon: " + terms, scope, cause, conv_id, signer)

    def reform(self, source: ActorID, proposal: str, precedent: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "reform: " + proposal, precedent, conv_id, signer)

    # --- Named Functions (6) ---

    def trial(
        self, plaintiff: ActorID, defendant: ActorID,
        judge_actor: ActorID, complaint: str,
        plaintiff_evidence: str, defendant_evidence: str,
        plaintiff_argument: str, defendant_argument: str,
        ruling: str,
        target: EventID, conv_id: ConversationID, signer: Signer,
    ) -> TrialResultJ:
        filing = self.file(plaintiff, complaint, target, conv_id, signer)
        sub1 = self.submit(plaintiff, filing.id, plaintiff_evidence, conv_id, signer)
        sub2 = self.submit(defendant, filing.id, defendant_evidence, conv_id, signer)
        arg1 = self.argue(plaintiff, plaintiff_argument, sub1.id, conv_id, signer)
        arg2 = self.argue(defendant, defendant_argument, sub2.id, conv_id, signer)
        verdict = self.judge(judge_actor, ruling, [arg1.id, arg2.id], conv_id, signer)
        return TrialResultJ(filing=filing, submissions=[sub1, sub2], arguments=[arg1, arg2], ruling=verdict)

    def constitutional_amendment(
        self, proposer: ActorID,
        proposal: str, legislation: str, rights_assessment: str,
        precedent: EventID, conv_id: ConversationID, signer: Signer,
    ) -> ConstitutionalAmendmentResult:
        reform_ev = self.reform(proposer, proposal, precedent, conv_id, signer)
        legislate_ev = self.legislate(proposer, legislation, [reform_ev.id], conv_id, signer)
        rights = self.audit(proposer, legislate_ev.id, rights_assessment, conv_id, signer)
        return ConstitutionalAmendmentResult(reform=reform_ev, legislation=legislate_ev, rights_check=rights)

    def injunction(
        self, petitioner: ActorID, judge_actor: ActorID,
        executor: ActorID, complaint: str, ruling: str,
        scope: DomainScope, weight: Weight,
        target: EventID, conv_id: ConversationID, signer: Signer,
    ) -> InjunctionResult:
        filing = self.file(petitioner, complaint, target, conv_id, signer)
        verdict = self.judge(judge_actor, "emergency: " + ruling, [filing.id], conv_id, signer)
        enforce_ev = self.enforce(judge_actor, executor, scope, weight, verdict.id, conv_id, signer)
        return InjunctionResult(filing=filing, ruling=verdict, enforcement=enforce_ev)

    def plea(
        self, defendant: ActorID, prosecutor: ActorID,
        executor: ActorID, complaint: str, deal: str,
        scope: DomainScope, weight: Weight,
        target: EventID, conv_id: ConversationID, signer: Signer,
    ) -> PleaResult:
        filing = self.file(prosecutor, complaint, target, conv_id, signer)
        acceptance = self.pardon(prosecutor, defendant, deal, scope, filing.id, conv_id, signer)
        enforce_ev = self.enforce(prosecutor, executor, scope, weight, acceptance.id, conv_id, signer)
        return PleaResult(filing=filing, acceptance=acceptance, enforcement=enforce_ev)

    def class_action(
        self, plaintiffs: list[ActorID],
        defendant: ActorID, judge_actor: ActorID,
        complaints: list[str], evidence: str, argument: str,
        defense_evidence: str, defense_argument: str, ruling: str,
        target: EventID, conv_id: ConversationID, signer: Signer,
    ) -> ClassActionResult:
        if len(plaintiffs) != len(complaints):
            raise ValueError("class-action: plaintiffs and complaints must have equal length")

        filings: list[Event] = []
        filing_ids: list[EventID] = []
        for i, plaintiff in enumerate(plaintiffs):
            f = self.file(plaintiff, complaints[i], target, conv_id, signer)
            filings.append(f)
            filing_ids.append(f.id)

        merged = self._g.merge(plaintiffs[0], "class-action: merged complaints", filing_ids, conv_id, signer)
        trial_result = self.trial(
            plaintiffs[0], defendant, judge_actor,
            "class-action", evidence, defense_evidence, argument, defense_argument, ruling,
            merged.id, conv_id, signer,
        )
        return ClassActionResult(filings=filings, merged=merged, trial=trial_result)

    def recall(
        self, auditor: ActorID, community: ActorID,
        official: ActorID, findings: str, complaint: str,
        scope: DomainScope,
        target: EventID, conv_id: ConversationID, signer: Signer,
    ) -> RecallResult:
        audit_ev = self.audit(auditor, target, findings, conv_id, signer)
        filing = self.file(auditor, complaint, audit_ev.id, conv_id, signer)
        consent_ev = self._g.consent(community, official, "recall: " + complaint, scope, filing.id, conv_id, signer)
        revocation = self._g.emit(community, "role-revoked: " + complaint, conv_id, [consent_ev.id], signer)
        return RecallResult(audit=audit_ev, filing=filing, consent=consent_ev, revocation=revocation)


# =============================================================================
# Layer 5: BuildGrammar (Technology)
# =============================================================================


@dataclass(frozen=True)
class SpikeResult:
    build: Event
    test: Event
    feedback: Event
    decision: Event


@dataclass(frozen=True)
class MigrationResult:
    sunset: Event
    version: Event
    ship: Event
    test: Event


@dataclass(frozen=True)
class TechDebtResult:
    measure: Event
    debt_mark: Event
    iteration: Event


@dataclass(frozen=True)
class PipelineResult:
    definition: Event
    test_result: Event
    metrics: Event
    deployment: Event


@dataclass(frozen=True)
class PostMortemResult:
    feedback: list[Event]
    analysis: Event
    improvements: Event


class BuildGrammar:
    """Layer 5 (Technology) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (12) ---

    def build(self, source: ActorID, artefact: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "build: " + artefact, conv_id, causes, signer)

    def version(self, source: ActorID, version_str: str, previous: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "version: " + version_str, previous, conv_id, signer)

    def ship(self, source: ActorID, deployment: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "ship: " + deployment, conv_id, causes, signer)

    def sunset(self, source: ActorID, target: EventID, migration: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "deprecated", migration, conv_id, signer)

    def define(self, source: ActorID, workflow: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "define: " + workflow, conv_id, causes, signer)

    def automate(self, source: ActorID, automation: str, workflow: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "automate: " + automation, workflow, conv_id, signer)

    def test(self, source: ActorID, results: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "test: " + results, conv_id, causes, signer)

    def review(self, source: ActorID, assessment: str, target: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.respond(source, "review: " + assessment, target, conv_id, signer)

    def measure(self, source: ActorID, target: EventID, scores: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "quality", scores, conv_id, signer)

    def feedback(self, source: ActorID, feedback_str: str, target: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.respond(source, "feedback: " + feedback_str, target, conv_id, signer)

    def iterate(self, source: ActorID, improvement: str, previous: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "iterate: " + improvement, previous, conv_id, signer)

    def innovate(self, source: ActorID, innovation: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "innovate: " + innovation, conv_id, causes, signer)

    # --- Named Functions (5) ---

    def spike(
        self, source: ActorID,
        experiment: str, test_results: str, feedback_str: str, decision: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> SpikeResult:
        build_ev = self.build(source, "spike: " + experiment, causes, conv_id, signer)
        test_ev = self.test(source, test_results, [build_ev.id], conv_id, signer)
        fb = self.feedback(source, feedback_str, test_ev.id, conv_id, signer)
        dec = self._g.emit(source, "spike-decision: " + decision, conv_id, [fb.id], signer)
        return SpikeResult(build=build_ev, test=test_ev, feedback=fb, decision=dec)

    def migration(
        self, source: ActorID,
        deprecated_target: EventID, migration_path: str,
        new_version: str, deployment: str, test_results: str,
        conv_id: ConversationID, signer: Signer,
    ) -> MigrationResult:
        sunset_ev = self.sunset(source, deprecated_target, migration_path, conv_id, signer)
        version_ev = self.version(source, new_version, sunset_ev.id, conv_id, signer)
        ship_ev = self.ship(source, deployment, [version_ev.id], conv_id, signer)
        test_ev = self.test(source, test_results, [ship_ev.id], conv_id, signer)
        return MigrationResult(sunset=sunset_ev, version=version_ev, ship=ship_ev, test=test_ev)

    def tech_debt(
        self, source: ActorID,
        target: EventID, scores: str, debt_description: str, plan: str,
        conv_id: ConversationID, signer: Signer,
    ) -> TechDebtResult:
        measure_ev = self.measure(source, target, scores, conv_id, signer)
        debt = self._g.annotate(source, measure_ev.id, "tech_debt", debt_description, conv_id, signer)
        iterate_ev = self.iterate(source, plan, debt.id, conv_id, signer)
        return TechDebtResult(measure=measure_ev, debt_mark=debt, iteration=iterate_ev)

    def pipeline(
        self, source: ActorID,
        workflow: str, test_results: str, metrics: str, deployment: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> PipelineResult:
        def_ev = self.define(source, workflow, causes, conv_id, signer)
        test_ev = self.test(source, test_results, [def_ev.id], conv_id, signer)
        measure_ev = self.measure(source, test_ev.id, metrics, conv_id, signer)
        ship_ev = self.ship(source, deployment, [measure_ev.id], conv_id, signer)
        return PipelineResult(definition=def_ev, test_result=test_ev, metrics=measure_ev, deployment=ship_ev)

    def post_mortem(
        self, lead: ActorID,
        contributors: list[ActorID], feedbacks: list[str],
        analysis: str, improvements: str,
        incident: EventID, conv_id: ConversationID, signer: Signer,
    ) -> PostMortemResult:
        if len(contributors) != len(feedbacks):
            raise ValueError("post-mortem: contributors and feedbacks must have equal length")

        fb_list: list[Event] = []
        fb_ids: list[EventID] = []
        for i, contrib in enumerate(contributors):
            fb = self.feedback(contrib, feedbacks[i], incident, conv_id, signer)
            fb_list.append(fb)
            fb_ids.append(fb.id)

        analysis_ev = self.measure(lead, fb_ids[-1], "post-mortem: " + analysis, conv_id, signer)
        improve = self.define(lead, improvements, [analysis_ev.id], conv_id, signer)
        return PostMortemResult(feedback=fb_list, analysis=analysis_ev, improvements=improve)


# =============================================================================
# Layer 6: KnowledgeGrammar (Information)
# =============================================================================


@dataclass(frozen=True)
class FactCheckResult:
    provenance: Event
    bias_check: Event
    verdict: Event


@dataclass(frozen=True)
class VerifyResult:
    claim: Event
    provenance: Event
    corroboration: Event


@dataclass(frozen=True)
class SurveyResult:
    recalls: list[Event]
    abstraction: Event
    synthesis: Event


@dataclass(frozen=True)
class KnowledgeBaseResult:
    claims: list[Event]
    categories: list[Event]
    memory: Event


@dataclass(frozen=True)
class TransferResult:
    recall: Event
    encode: Event
    learn: Event


class KnowledgeGrammar:
    """Layer 6 (Information) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (12) ---

    def claim(self, source: ActorID, claim_text: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "claim: " + claim_text, conv_id, causes, signer)

    def categorize(self, source: ActorID, target: EventID, taxonomy: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "classification", taxonomy, conv_id, signer)

    def abstract(self, source: ActorID, generalization: str, instances: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        if len(instances) < 2:
            raise ValueError("abstract: requires at least two instances")
        return self._g.merge(source, "abstract: " + generalization, instances, conv_id, signer)

    def encode(self, source: ActorID, encoding: str, original: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "encode: " + encoding, original, conv_id, signer)

    def infer(self, source: ActorID, conclusion: str, premise: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "infer: " + conclusion, premise, conv_id, signer)

    def remember(self, source: ActorID, knowledge: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "remember: " + knowledge, conv_id, causes, signer)

    def challenge(self, source: ActorID, counter_evidence: str, claim_event: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        _, flag = self._g.challenge(source, "challenge: " + counter_evidence, claim_event, conv_id, signer)
        return flag

    def detect_bias(self, source: ActorID, target: EventID, bias: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "bias", bias, conv_id, signer)

    def correct(self, source: ActorID, correction: str, original: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "correct: " + correction, original, conv_id, signer)

    def trace(self, source: ActorID, target: EventID, provenance: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "provenance", provenance, conv_id, signer)

    def recall(self, source: ActorID, query: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "recall: " + query, conv_id, causes, signer)

    def learn(self, source: ActorID, learning: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "learn: " + learning, conv_id, causes, signer)

    # --- Named Functions (6) ---

    def retract(self, source: ActorID, claim_event: EventID, reason: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.retract(source, claim_event, reason, conv_id, signer)

    def fact_check(
        self, checker: ActorID,
        claim_event: EventID, provenance: str, bias_analysis: str, verdict: str,
        conv_id: ConversationID, signer: Signer,
    ) -> FactCheckResult:
        trace_ev = self.trace(checker, claim_event, provenance, conv_id, signer)
        bias = self.detect_bias(checker, claim_event, bias_analysis, conv_id, signer)
        verdict_ev = self._g.merge(checker, "fact-check: " + verdict, [trace_ev.id, bias.id], conv_id, signer)
        return FactCheckResult(provenance=trace_ev, bias_check=bias, verdict=verdict_ev)

    def verify(
        self, source: ActorID,
        claim_text: str, provenance: str, corroboration: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> VerifyResult:
        claim_ev = self.claim(source, claim_text, causes, conv_id, signer)
        trace_ev = self.trace(source, claim_ev.id, provenance, conv_id, signer)
        corroborate = self.claim(source, "corroborate: " + corroboration, [trace_ev.id], conv_id, signer)
        return VerifyResult(claim=claim_ev, provenance=trace_ev, corroboration=corroborate)

    def survey(
        self, source: ActorID,
        queries: list[str], generalization: str, synthesis: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> SurveyResult:
        if len(queries) < 2:
            raise ValueError("survey: requires at least two queries")

        recalls: list[Event] = []
        recall_ids: list[EventID] = []
        for query in queries:
            r = self.recall(source, query, causes, conv_id, signer)
            recalls.append(r)
            recall_ids.append(r.id)

        abstract_ev = self.abstract(source, generalization, recall_ids, conv_id, signer)
        synthesis_claim = self.claim(source, "synthesis: " + synthesis, [abstract_ev.id], conv_id, signer)
        return SurveyResult(recalls=recalls, abstraction=abstract_ev, synthesis=synthesis_claim)

    def knowledge_base(
        self, source: ActorID,
        claims: list[str], taxonomies: list[str], memory_label: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> KnowledgeBaseResult:
        if len(claims) != len(taxonomies):
            raise ValueError("knowledge-base: claims and taxonomies must have equal length")

        claim_list: list[Event] = []
        cat_list: list[Event] = []
        claim_ids: list[EventID] = []
        for i, c in enumerate(claims):
            claim_ev = self.claim(source, c, causes, conv_id, signer)
            claim_list.append(claim_ev)
            cat = self.categorize(source, claim_ev.id, taxonomies[i], conv_id, signer)
            cat_list.append(cat)
            claim_ids.append(cat.id)

        memory = self.remember(source, memory_label, claim_ids, conv_id, signer)
        return KnowledgeBaseResult(claims=claim_list, categories=cat_list, memory=memory)

    def transfer(
        self, source: ActorID,
        query: str, encoding: str, learning: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> TransferResult:
        recall_ev = self.recall(source, query, causes, conv_id, signer)
        encode_ev = self.encode(source, encoding, recall_ev.id, conv_id, signer)
        learn_ev = self.learn(source, learning, [encode_ev.id], conv_id, signer)
        return TransferResult(recall=recall_ev, encode=encode_ev, learn=learn_ev)


# =============================================================================
# Layer 7: AlignmentGrammar (Ethics)
# =============================================================================


@dataclass(frozen=True)
class EthicsAuditResult:
    fairness: Event
    harm_scan: Event
    report: Event


@dataclass(frozen=True)
class RestorativeJusticeResult:
    harm_detection: Event
    responsibility: Event
    redress: Event
    growth: Event


@dataclass(frozen=True)
class GuardrailResult:
    constraint: Event
    dilemma: Event
    escalation: Event


@dataclass(frozen=True)
class ImpactAssessmentResult:
    weighing: Event
    fairness: Event
    explanation: Event


@dataclass(frozen=True)
class WhistleblowResult:
    harm: Event
    explanation: Event
    escalation: Event


class AlignmentGrammar:
    """Layer 7 (Ethics) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (10) ---

    def constrain(self, source: ActorID, target: EventID, constraint: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "constraint", constraint, conv_id, signer)

    def detect_harm(self, source: ActorID, harm: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "harm: " + harm, conv_id, causes, signer)

    def assess_fairness(self, source: ActorID, target: EventID, assessment: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "fairness", assessment, conv_id, signer)

    def flag_dilemma(self, source: ActorID, dilemma: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "dilemma: " + dilemma, conv_id, causes, signer)

    def weigh(self, source: ActorID, weighing: str, decision: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "weigh: " + weighing, decision, conv_id, signer)

    def explain(self, source: ActorID, explanation: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "explain: " + explanation, conv_id, causes, signer)

    def assign(self, source: ActorID, target: EventID, responsibility: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "responsibility", responsibility, conv_id, signer)

    def repair(self, source: ActorID, affected: ActorID, redress: str, scope: DomainScope, cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.consent(source, affected, "repair: " + redress, scope, cause, conv_id, signer)

    def care(self, source: ActorID, care_text: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "care: " + care_text, conv_id, causes, signer)

    def grow(self, source: ActorID, growth: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "grow: " + growth, conv_id, causes, signer)

    # --- Named Functions (5) ---

    def ethics_audit(
        self, auditor: ActorID, target: EventID,
        fairness_assessment: str, harm_scan: str, summary: str,
        conv_id: ConversationID, signer: Signer,
    ) -> EthicsAuditResult:
        fairness = self.assess_fairness(auditor, target, fairness_assessment, conv_id, signer)
        harm = self.detect_harm(auditor, harm_scan, [fairness.id], conv_id, signer)
        report = self.explain(auditor, summary, [fairness.id, harm.id], conv_id, signer)
        return EthicsAuditResult(fairness=fairness, harm_scan=harm, report=report)

    def restorative_justice(
        self, auditor: ActorID, agent: ActorID, affected: ActorID,
        harm: str, responsibility: str, redress: str, growth: str,
        scope: DomainScope, cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> RestorativeJusticeResult:
        harm_ev = self.detect_harm(auditor, harm, [cause], conv_id, signer)
        assign_ev = self.assign(auditor, harm_ev.id, responsibility, conv_id, signer)
        repair_ev = self.repair(auditor, affected, redress, scope, assign_ev.id, conv_id, signer)
        grow_ev = self.grow(agent, growth, [repair_ev.id], conv_id, signer)
        return RestorativeJusticeResult(harm_detection=harm_ev, responsibility=assign_ev, redress=repair_ev, growth=grow_ev)

    def guardrail(
        self, source: ActorID, target: EventID,
        constraint: str, dilemma: str, escalation: str,
        conv_id: ConversationID, signer: Signer,
    ) -> GuardrailResult:
        constrain_ev = self.constrain(source, target, constraint, conv_id, signer)
        dilemma_ev = self.flag_dilemma(source, dilemma, [constrain_ev.id], conv_id, signer)
        escalate = self._g.emit(source, "escalate: " + escalation, conv_id, [dilemma_ev.id], signer)
        return GuardrailResult(constraint=constrain_ev, dilemma=dilemma_ev, escalation=escalate)

    def impact_assessment(
        self, source: ActorID, decision: EventID,
        weighing: str, fairness: str, explanation: str,
        conv_id: ConversationID, signer: Signer,
    ) -> ImpactAssessmentResult:
        weigh_ev = self.weigh(source, weighing, decision, conv_id, signer)
        fair = self.assess_fairness(source, weigh_ev.id, fairness, conv_id, signer)
        explain_ev = self.explain(source, explanation, [weigh_ev.id, fair.id], conv_id, signer)
        return ImpactAssessmentResult(weighing=weigh_ev, fairness=fair, explanation=explain_ev)

    def whistleblow(
        self, source: ActorID,
        harm: str, explanation: str, escalation: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> WhistleblowResult:
        harm_ev = self.detect_harm(source, harm, causes, conv_id, signer)
        explain_ev = self.explain(source, explanation, [harm_ev.id], conv_id, signer)
        escalate = self._g.emit(source, "escalate-external: " + escalation, conv_id, [explain_ev.id], signer)
        return WhistleblowResult(harm=harm_ev, explanation=explain_ev, escalation=escalate)


# =============================================================================
# Layer 8: IdentityGrammar (Identity)
# =============================================================================


@dataclass(frozen=True)
class IdentityAuditResult:
    self_model: Event
    alignment: Event
    narrative: Event


@dataclass(frozen=True)
class RetirementResult:
    memorial: Event
    transfer: Event
    archive: Event


@dataclass(frozen=True)
class CredentialResult:
    introspection: Event
    disclosure: Event


@dataclass(frozen=True)
class ReinventionResult:
    transformation: Event
    narrative: Event
    aspiration: Event


@dataclass(frozen=True)
class IntroductionResult:
    disclosure: Event
    narrative: Event


class IdentityGrammar:
    """Layer 8 (Identity) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (10) ---

    def introspect(self, source: ActorID, self_model: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "introspect: " + self_model, conv_id, causes, signer)

    def narrate(self, source: ActorID, narrative: str, basis: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "narrate: " + narrative, basis, conv_id, signer)

    def align(self, source: ActorID, target: EventID, alignment: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "alignment", alignment, conv_id, signer)

    def bound(self, source: ActorID, boundary: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "bound: " + boundary, conv_id, causes, signer)

    def aspire(self, source: ActorID, aspiration: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "aspire: " + aspiration, conv_id, causes, signer)

    def transform(self, source: ActorID, transformation: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "transform: " + transformation, conv_id, causes, signer)

    def disclose(self, source: ActorID, target: ActorID, scope: Option[DomainScope], cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.channel(source, target, scope, cause, conv_id, signer)

    def recognize(self, source: ActorID, recognition: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "recognize: " + recognition, conv_id, causes, signer)

    def distinguish(self, source: ActorID, target: EventID, uniqueness: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "uniqueness", uniqueness, conv_id, signer)

    def memorialize(self, source: ActorID, memorial: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "memorialize: " + memorial, conv_id, causes, signer)

    # --- Named Functions (5) ---

    def identity_audit(
        self, source: ActorID,
        self_model: str, alignment: str, narrative: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> IdentityAuditResult:
        intro = self.introspect(source, self_model, causes, conv_id, signer)
        align_ev = self.align(source, intro.id, alignment, conv_id, signer)
        narr = self.narrate(source, narrative, align_ev.id, conv_id, signer)
        return IdentityAuditResult(self_model=intro, alignment=align_ev, narrative=narr)

    def retirement(
        self, system: ActorID, departing: ActorID, successor: ActorID,
        memorial: str, scope: DomainScope, weight: Weight,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> RetirementResult:
        mem = self.memorialize(system, f"retirement of {departing.value}: {memorial}", causes, conv_id, signer)
        transfer = self._g.delegate(system, successor, scope, weight, mem.id, conv_id, signer)
        archive = self._g.emit(system, f"archive: contributions of {departing.value}", conv_id, [transfer.id], signer)
        return RetirementResult(memorial=mem, transfer=transfer, archive=archive)

    def credential(
        self, source: ActorID, verifier: ActorID,
        self_model: str, scope: Option[DomainScope],
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> CredentialResult:
        intro = self.introspect(source, self_model, causes, conv_id, signer)
        disclose_ev = self.disclose(source, verifier, scope, intro.id, conv_id, signer)
        return CredentialResult(introspection=intro, disclosure=disclose_ev)

    def reinvention(
        self, source: ActorID,
        transformation: str, narrative: str, aspiration: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> ReinventionResult:
        transform_ev = self.transform(source, transformation, causes, conv_id, signer)
        narr = self.narrate(source, narrative, transform_ev.id, conv_id, signer)
        aspire_ev = self.aspire(source, aspiration, [narr.id], conv_id, signer)
        return ReinventionResult(transformation=transform_ev, narrative=narr, aspiration=aspire_ev)

    def introduction(
        self, source: ActorID, target: ActorID,
        scope: Option[DomainScope], narrative: str,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> IntroductionResult:
        disclose_ev = self.disclose(source, target, scope, cause, conv_id, signer)
        narr = self.narrate(source, narrative, disclose_ev.id, conv_id, signer)
        return IntroductionResult(disclosure=disclose_ev, narrative=narr)


# =============================================================================
# Layer 9: BondGrammar (Relationship)
# =============================================================================


@dataclass(frozen=True)
class BetrayalRepairResult:
    rupture: Event
    apology: Event
    reconciliation: Event
    deepened: Event


@dataclass(frozen=True)
class CheckInResult:
    balance: Event
    attunement: Event
    empathy: Event


@dataclass(frozen=True)
class BondMentorshipResult:
    connection: Event
    deepening: Event
    attunement: Event
    teaching: Event


@dataclass(frozen=True)
class BondFarewellResult:
    mourning: Event
    memorial: Event
    gratitude: Event


class BondGrammar:
    """Layer 9 (Relationship) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (10) ---

    def connect(
        self, source: ActorID, target: ActorID,
        scope: Option[DomainScope],
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> tuple[Event, Event]:
        sub1 = self._g.subscribe(source, target, scope, cause, conv_id, signer)
        sub2 = self._g.subscribe(target, source, scope, sub1.id, conv_id, signer)
        return sub1, sub2

    def balance(self, source: ActorID, target: EventID, assessment: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "reciprocity", assessment, conv_id, signer)

    def deepen(self, source: ActorID, other: ActorID, basis: str, scope: DomainScope, cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.consent(source, other, "deepen: " + basis, scope, cause, conv_id, signer)

    def open(self, source: ActorID, target: ActorID, scope: Option[DomainScope], cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.channel(source, target, scope, cause, conv_id, signer)

    def attune(self, source: ActorID, understanding: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "attune: " + understanding, conv_id, causes, signer)

    def feel_with(self, source: ActorID, empathy: str, target: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.respond(source, "empathy: " + empathy, target, conv_id, signer)

    def break_(self, source: ActorID, rupture: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "rupture: " + rupture, conv_id, causes, signer)

    def apologize(self, source: ActorID, apology: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "apology: " + apology, conv_id, causes, signer)

    def reconcile(self, source: ActorID, other: ActorID, progress: str, scope: DomainScope, cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.consent(source, other, "reconcile: " + progress, scope, cause, conv_id, signer)

    def mourn(self, source: ActorID, loss: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "mourn: " + loss, conv_id, causes, signer)

    # --- Named Functions (5) ---

    def betrayal_repair(
        self, injured: ActorID, offender: ActorID,
        rupture: str, apology: str, reconciliation: str, new_basis: str,
        scope: DomainScope,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> BetrayalRepairResult:
        rupture_ev = self.break_(injured, rupture, causes, conv_id, signer)
        apology_ev = self.apologize(offender, apology, [rupture_ev.id], conv_id, signer)
        reconcile_ev = self.reconcile(injured, offender, reconciliation, scope, apology_ev.id, conv_id, signer)
        deepen_ev = self.deepen(injured, offender, new_basis, scope, reconcile_ev.id, conv_id, signer)
        return BetrayalRepairResult(rupture=rupture_ev, apology=apology_ev, reconciliation=reconcile_ev, deepened=deepen_ev)

    def check_in(
        self, source: ActorID,
        balance_target: EventID, assessment: str,
        attunement: str, empathy: str,
        conv_id: ConversationID, signer: Signer,
    ) -> CheckInResult:
        bal = self.balance(source, balance_target, assessment, conv_id, signer)
        att = self.attune(source, attunement, [bal.id], conv_id, signer)
        emp = self.feel_with(source, empathy, att.id, conv_id, signer)
        return CheckInResult(balance=bal, attunement=att, empathy=emp)

    def mentorship(
        self, mentor: ActorID, mentee: ActorID,
        basis: str, understanding: str,
        scope: DomainScope, teaching_scope: Option[DomainScope],
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> BondMentorshipResult:
        connect_ev = self._g.subscribe(mentee, mentor, teaching_scope, cause, conv_id, signer)
        deepen_ev = self.deepen(mentor, mentee, basis, scope, connect_ev.id, conv_id, signer)
        attune_ev = self.attune(mentor, understanding, [deepen_ev.id], conv_id, signer)
        teach = self._g.channel(mentor, mentee, teaching_scope, attune_ev.id, conv_id, signer)
        return BondMentorshipResult(connection=connect_ev, deepening=deepen_ev, attunement=attune_ev, teaching=teach)

    def farewell(
        self, source: ActorID, departing: ActorID,
        loss: str, memorial: str, gratitude_weight: Weight,
        scope: Option[DomainScope],
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> BondFarewellResult:
        mourn_ev = self.mourn(source, loss, causes, conv_id, signer)
        mem = self._g.emit(source, "memorialize: " + memorial, conv_id, [mourn_ev.id], signer)
        gratitude = self._g.endorse(source, mem.id, departing, gratitude_weight, scope, conv_id, signer)
        return BondFarewellResult(mourning=mourn_ev, memorial=mem, gratitude=gratitude)

    def forgive(
        self, source: ActorID,
        sever_event: EventID, target: ActorID,
        scope: Option[DomainScope],
        conv_id: ConversationID, signer: Signer,
    ) -> Event:
        return self._g.forgive(source, sever_event, target, scope, conv_id, signer)


# =============================================================================
# Layer 10: BelongingGrammar (Community)
# =============================================================================


@dataclass(frozen=True)
class FestivalResult:
    celebration: Event
    practice: Event
    story: Event
    gift: Event


@dataclass(frozen=True)
class CommonsGovernanceResult:
    stewardship: Event
    assessment: Event
    legislation: Event
    audit: Event


@dataclass(frozen=True)
class RenewalResult:
    assessment: Event
    practice: Event
    story: Event


@dataclass(frozen=True)
class OnboardResult:
    inclusion: Event
    settlement: Event
    first_practice: Event
    contribution: Event


@dataclass(frozen=True)
class SuccessionResult:
    assessment: Event
    transfer: Event
    celebration: Event
    story: Event


class BelongingGrammar:
    """Layer 10 (Community) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (10) ---

    def settle(self, source: ActorID, community: ActorID, scope: Option[DomainScope], cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.subscribe(source, community, scope, cause, conv_id, signer)

    def contribute(self, source: ActorID, contribution: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "contribute: " + contribution, conv_id, causes, signer)

    def include(self, source: ActorID, action: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "include: " + action, conv_id, causes, signer)

    def practice(self, source: ActorID, tradition: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "practice: " + tradition, conv_id, causes, signer)

    def steward(self, source: ActorID, steward_actor: ActorID, scope: DomainScope, weight: Weight, cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.delegate(source, steward_actor, scope, weight, cause, conv_id, signer)

    def sustain(self, source: ActorID, assessment: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "sustain: " + assessment, conv_id, causes, signer)

    def pass_on(self, from_actor: ActorID, to_actor: ActorID, scope: DomainScope, description: str, cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.consent(from_actor, to_actor, "pass-on: " + description, scope, cause, conv_id, signer)

    def celebrate(self, source: ActorID, celebration: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "celebrate: " + celebration, conv_id, causes, signer)

    def tell(self, source: ActorID, story: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "tell: " + story, conv_id, causes, signer)

    def gift(self, source: ActorID, gift_text: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "gift: " + gift_text, conv_id, causes, signer)

    # --- Named Functions (5) ---

    def festival(
        self, source: ActorID,
        celebration: str, tradition: str, story: str, gift_text: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> FestivalResult:
        celebrate_ev = self.celebrate(source, celebration, causes, conv_id, signer)
        practice_ev = self.practice(source, tradition, [celebrate_ev.id], conv_id, signer)
        tell_ev = self.tell(source, story, [practice_ev.id], conv_id, signer)
        gift_ev = self.gift(source, gift_text, [tell_ev.id], conv_id, signer)
        return FestivalResult(celebration=celebrate_ev, practice=practice_ev, story=tell_ev, gift=gift_ev)

    def commons_governance(
        self, source: ActorID, steward_actor: ActorID,
        scope: DomainScope, weight: Weight,
        assessment: str, rule: str, findings: str,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> CommonsGovernanceResult:
        stewardship = self.steward(source, steward_actor, scope, weight, cause, conv_id, signer)
        sustain_ev = self.sustain(steward_actor, assessment, [stewardship.id], conv_id, signer)
        legislate = self._g.emit(source, "legislate: " + rule, conv_id, [sustain_ev.id], signer)
        audit_ev = self._g.annotate(steward_actor, legislate.id, "audit", findings, conv_id, signer)
        return CommonsGovernanceResult(stewardship=stewardship, assessment=sustain_ev, legislation=legislate, audit=audit_ev)

    def renewal(
        self, source: ActorID,
        assessment: str, evolved_practice: str, new_story: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> RenewalResult:
        sustain_ev = self.sustain(source, assessment, causes, conv_id, signer)
        practice_ev = self.practice(source, evolved_practice, [sustain_ev.id], conv_id, signer)
        story = self.tell(source, new_story, [practice_ev.id], conv_id, signer)
        return RenewalResult(assessment=sustain_ev, practice=practice_ev, story=story)

    def onboard(
        self, sponsor: ActorID, newcomer: ActorID, community: ActorID,
        scope: Option[DomainScope],
        inclusion_action: str, tradition: str, first_contribution: str,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> OnboardResult:
        inclusion = self.include(sponsor, inclusion_action, [cause], conv_id, signer)
        settle_ev = self.settle(newcomer, community, scope, inclusion.id, conv_id, signer)
        practice_ev = self.practice(newcomer, tradition, [settle_ev.id], conv_id, signer)
        contrib = self.contribute(newcomer, first_contribution, [practice_ev.id], conv_id, signer)
        return OnboardResult(inclusion=inclusion, settlement=settle_ev, first_practice=practice_ev, contribution=contrib)

    def succession(
        self, outgoing: ActorID, incoming: ActorID,
        assessment: str, scope: DomainScope, celebration: str, story: str,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> SuccessionResult:
        sustain_ev = self.sustain(outgoing, assessment, [cause], conv_id, signer)
        transfer = self.pass_on(outgoing, incoming, scope, "stewardship transfer", sustain_ev.id, conv_id, signer)
        celebrate_ev = self.celebrate(outgoing, celebration, [transfer.id], conv_id, signer)
        tell_ev = self.tell(outgoing, story, [celebrate_ev.id], conv_id, signer)
        return SuccessionResult(assessment=sustain_ev, transfer=transfer, celebration=celebrate_ev, story=tell_ev)


# =============================================================================
# Layer 11: MeaningGrammar (Culture)
# =============================================================================


@dataclass(frozen=True)
class DesignReviewResult:
    beauty: Event
    reframe: Event
    question: Event
    wisdom: Event


@dataclass(frozen=True)
class CulturalOnboardingResult:
    translation: Event
    teaching: Event
    examination: Event


@dataclass(frozen=True)
class ForecastResult:
    prophecy: Event
    examination: Event
    wisdom: Event


@dataclass(frozen=True)
class MeaningPostMortemResult:
    examination: Event
    questions: Event
    wisdom: Event


@dataclass(frozen=True)
class MeaningMentorshipResult:
    channel: Event
    reframing: Event
    wisdom: Event
    translation: Event


class MeaningGrammar:
    """Layer 11 (Culture) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (10) ---

    def examine(self, source: ActorID, examination: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "examine: " + examination, conv_id, causes, signer)

    def reframe(self, source: ActorID, reframing: str, original: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "reframe: " + reframing, original, conv_id, signer)

    def question(self, source: ActorID, question_text: str, target: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        _, flag = self._g.challenge(source, "question: " + question_text, target, conv_id, signer)
        return flag

    def distill(self, source: ActorID, wisdom: str, experience: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "distill: " + wisdom, experience, conv_id, signer)

    def beautify(self, source: ActorID, beauty: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "beautify: " + beauty, conv_id, causes, signer)

    def liken(self, source: ActorID, metaphor: str, subject: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "liken: " + metaphor, subject, conv_id, signer)

    def lighten(self, source: ActorID, humour: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "lighten: " + humour, conv_id, causes, signer)

    def teach(self, source: ActorID, student: ActorID, scope: Option[DomainScope], cause: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.channel(source, student, scope, cause, conv_id, signer)

    def translate(self, source: ActorID, translation: str, original: EventID, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.derive(source, "translate: " + translation, original, conv_id, signer)

    def prophesy(self, source: ActorID, prediction: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "prophesy: " + prediction, conv_id, causes, signer)

    # --- Named Functions (5) ---

    def design_review(
        self, source: ActorID,
        beauty: str, reframing: str, question_text: str, wisdom: str,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> DesignReviewResult:
        beautify_ev = self.beautify(source, beauty, [cause], conv_id, signer)
        reframe_ev = self.reframe(source, reframing, beautify_ev.id, conv_id, signer)
        q = self.question(source, question_text, reframe_ev.id, conv_id, signer)
        w = self.distill(source, wisdom, q.id, conv_id, signer)
        return DesignReviewResult(beauty=beautify_ev, reframe=reframe_ev, question=q, wisdom=w)

    def cultural_onboarding(
        self, guide: ActorID, newcomer: ActorID,
        translation: str, teaching_scope: Option[DomainScope], examination: str,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> CulturalOnboardingResult:
        translate_ev = self.translate(guide, translation, cause, conv_id, signer)
        teach_ev = self.teach(guide, newcomer, teaching_scope, translate_ev.id, conv_id, signer)
        examine_ev = self.examine(newcomer, examination, [teach_ev.id], conv_id, signer)
        return CulturalOnboardingResult(translation=translate_ev, teaching=teach_ev, examination=examine_ev)

    def forecast(
        self, source: ActorID,
        prediction: str, assumptions: str, confidence: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> ForecastResult:
        prophesy_ev = self.prophesy(source, prediction, causes, conv_id, signer)
        examine_ev = self.examine(source, assumptions, [prophesy_ev.id], conv_id, signer)
        distill_ev = self.distill(source, confidence, examine_ev.id, conv_id, signer)
        return ForecastResult(prophecy=prophesy_ev, examination=examine_ev, wisdom=distill_ev)

    def post_mortem(
        self, source: ActorID,
        examination: str, question_text: str, wisdom: str,
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> MeaningPostMortemResult:
        exam = self.examine(source, examination, [cause], conv_id, signer)
        q = self.question(source, question_text, exam.id, conv_id, signer)
        w = self.distill(source, wisdom, q.id, conv_id, signer)
        return MeaningPostMortemResult(examination=exam, questions=q, wisdom=w)

    def mentorship(
        self, mentor: ActorID, student: ActorID,
        reframing: str, wisdom: str, translation: str,
        scope: Option[DomainScope],
        cause: EventID, conv_id: ConversationID, signer: Signer,
    ) -> MeaningMentorshipResult:
        channel_ev = self.teach(mentor, student, scope, cause, conv_id, signer)
        reframe_ev = self.reframe(mentor, reframing, channel_ev.id, conv_id, signer)
        distill_ev = self.distill(mentor, wisdom, reframe_ev.id, conv_id, signer)
        translate_ev = self.translate(student, translation, distill_ev.id, conv_id, signer)
        return MeaningMentorshipResult(channel=channel_ev, reframing=reframe_ev, wisdom=distill_ev, translation=translate_ev)


# =============================================================================
# Layer 12: EvolutionGrammar (Emergence)
# =============================================================================


@dataclass(frozen=True)
class SelfEvolveResult:
    pattern: Event
    adaptation: Event
    selection: Event
    simplification: Event


@dataclass(frozen=True)
class HealthCheckResult:
    integrity: Event
    resilience: Event
    model: Event
    purpose: Event


@dataclass(frozen=True)
class PruneResult:
    pattern: Event
    simplification: Event
    verification: Event


@dataclass(frozen=True)
class PhaseTransitionResult:
    threshold: Event
    model: Event
    adaptation: Event
    selection: Event


class EvolutionGrammar:
    """Layer 12 (Emergence) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (10) ---

    def detect_pattern(self, source: ActorID, pattern: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "pattern: " + pattern, conv_id, causes, signer)

    def model(self, source: ActorID, model_str: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "model: " + model_str, conv_id, causes, signer)

    def trace_loop(self, source: ActorID, loop: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "loop: " + loop, conv_id, causes, signer)

    def watch_threshold(self, source: ActorID, target: EventID, threshold: str, conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.annotate(source, target, "threshold", threshold, conv_id, signer)

    def adapt(self, source: ActorID, proposal: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "adapt: " + proposal, conv_id, causes, signer)

    def select(self, source: ActorID, result: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "select: " + result, conv_id, causes, signer)

    def simplify(self, source: ActorID, simplification: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "simplify: " + simplification, conv_id, causes, signer)

    def check_integrity(self, source: ActorID, assessment: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "integrity: " + assessment, conv_id, causes, signer)

    def assess_resilience(self, source: ActorID, assessment: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "resilience: " + assessment, conv_id, causes, signer)

    def align_purpose(self, source: ActorID, alignment: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "purpose: " + alignment, conv_id, causes, signer)

    # --- Named Functions (4) ---

    def self_evolve(
        self, source: ActorID,
        pattern: str, adaptation: str, selection: str, simplification: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> SelfEvolveResult:
        pat = self.detect_pattern(source, pattern, causes, conv_id, signer)
        adapt_ev = self.adapt(source, adaptation, [pat.id], conv_id, signer)
        sel = self.select(source, selection, [adapt_ev.id], conv_id, signer)
        simp = self.simplify(source, simplification, [sel.id], conv_id, signer)
        return SelfEvolveResult(pattern=pat, adaptation=adapt_ev, selection=sel, simplification=simp)

    def health_check(
        self, source: ActorID,
        integrity: str, resilience: str, model_str: str, purpose: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> HealthCheckResult:
        integ = self.check_integrity(source, integrity, causes, conv_id, signer)
        resil = self.assess_resilience(source, resilience, [integ.id], conv_id, signer)
        mod = self.model(source, model_str, [resil.id], conv_id, signer)
        purp = self.align_purpose(source, purpose, [mod.id], conv_id, signer)
        return HealthCheckResult(integrity=integ, resilience=resil, model=mod, purpose=purp)

    def prune(
        self, source: ActorID,
        unused_pattern: str, simplification: str, verification: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> PruneResult:
        pattern = self.detect_pattern(source, "unused: " + unused_pattern, causes, conv_id, signer)
        simplify_ev = self.simplify(source, simplification, [pattern.id], conv_id, signer)
        verify = self.select(source, verification, [simplify_ev.id], conv_id, signer)
        return PruneResult(pattern=pattern, simplification=simplify_ev, verification=verify)

    def phase_transition(
        self, source: ActorID, target: EventID,
        threshold: str, model_str: str, adaptation: str, selection: str,
        conv_id: ConversationID, signer: Signer,
    ) -> PhaseTransitionResult:
        thresh = self.watch_threshold(source, target, threshold, conv_id, signer)
        mod = self.model(source, model_str, [thresh.id], conv_id, signer)
        adapt_ev = self.adapt(source, adaptation, [mod.id], conv_id, signer)
        sel = self.select(source, selection, [adapt_ev.id], conv_id, signer)
        return PhaseTransitionResult(threshold=thresh, model=mod, adaptation=adapt_ev, selection=sel)


# =============================================================================
# Layer 13: BeingGrammar (Existence)
# =============================================================================


@dataclass(frozen=True)
class BeingFarewellResult:
    acceptance: Event
    web: Event
    awe: Event
    memorial: Event


@dataclass(frozen=True)
class ContemplationResult:
    change: Event
    mystery: Event
    awe: Event
    wonder: Event


@dataclass(frozen=True)
class ExistentialAuditResult:
    existence: Event
    acceptance: Event
    web: Event
    purpose: Event


class BeingGrammar:
    """Layer 13 (Existence) composition operations."""

    def __init__(self, g: Grammar) -> None:
        self._g = g

    # --- Operations (8) ---

    def exist(self, source: ActorID, observation: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "exist: " + observation, conv_id, causes, signer)

    def accept(self, source: ActorID, limitation: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "accept: " + limitation, conv_id, causes, signer)

    def observe_change(self, source: ActorID, observation: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "change: " + observation, conv_id, causes, signer)

    def map_web(self, source: ActorID, mapping: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "web: " + mapping, conv_id, causes, signer)

    def face_mystery(self, source: ActorID, mystery: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "mystery: " + mystery, conv_id, causes, signer)

    def hold_paradox(self, source: ActorID, paradox: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "paradox: " + paradox, conv_id, causes, signer)

    def marvel(self, source: ActorID, awe: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "marvel: " + awe, conv_id, causes, signer)

    def ask_why(self, source: ActorID, question: str, causes: list[EventID], conv_id: ConversationID, signer: Signer) -> Event:
        return self._g.emit(source, "wonder: " + question, conv_id, causes, signer)

    # --- Named Functions (3) ---

    def farewell(
        self, source: ActorID,
        limitation: str, interconnection: str, awe: str, memorial: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> BeingFarewellResult:
        accept_ev = self.accept(source, limitation, causes, conv_id, signer)
        web = self.map_web(source, interconnection, [accept_ev.id], conv_id, signer)
        marvel_ev = self.marvel(source, awe, [web.id], conv_id, signer)
        mem = self._g.emit(source, "memorialize: " + memorial, conv_id, [marvel_ev.id], signer)
        return BeingFarewellResult(acceptance=accept_ev, web=web, awe=marvel_ev, memorial=mem)

    def contemplation(
        self, source: ActorID,
        change: str, mystery: str, awe: str, question: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> ContemplationResult:
        change_ev = self.observe_change(source, change, causes, conv_id, signer)
        mystery_ev = self.face_mystery(source, mystery, [change_ev.id], conv_id, signer)
        awe_ev = self.marvel(source, awe, [mystery_ev.id], conv_id, signer)
        wonder_ev = self.ask_why(source, question, [awe_ev.id], conv_id, signer)
        return ContemplationResult(change=change_ev, mystery=mystery_ev, awe=awe_ev, wonder=wonder_ev)

    def existential_audit(
        self, source: ActorID,
        existence: str, limitation: str, interconnection: str, purpose: str,
        causes: list[EventID], conv_id: ConversationID, signer: Signer,
    ) -> ExistentialAuditResult:
        exist_ev = self.exist(source, existence, causes, conv_id, signer)
        accept_ev = self.accept(source, limitation, [exist_ev.id], conv_id, signer)
        web = self.map_web(source, interconnection, [accept_ev.id], conv_id, signer)
        purp = self._g.emit(source, "purpose: " + purpose, conv_id, [web.id], signer)
        return ExistentialAuditResult(existence=exist_ev, acceptance=accept_ev, web=web, purpose=purp)
