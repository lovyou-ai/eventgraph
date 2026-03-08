/**
 * Composition grammars — domain-specific operations built on the base Grammar.
 * Ports the 13 Go composition packages (Layers 1-13) to TypeScript.
 */
import type { Event, Signer } from "./event.js";
import { Grammar } from "./grammar.js";
import { ActorId, ConversationId, DomainScope, EdgeId, EventId, Option, Weight } from "./types.js";

// ── Layer 1: Work Grammar (Agency) ──────────────────────────────────────

export interface StandupResult {
  updates: Event[];
  priority: Event;
}

export interface RetrospectiveResult {
  reviews: Event[];
  improvement: Event;
}

export interface TriageResult {
  priorities: Event[];
  assignments: Event[];
  scopes: Event[];
}

export interface SprintResult {
  intent: Event;
  subtasks: Event[];
  assignments: Event[];
}

export interface EscalateResult {
  blockEvent: Event;
  handoffEvent: Event;
}

export interface DelegateAndVerifyResult {
  assignEvent: Event;
  scopeEvent: Event;
}

export class WorkGrammar {
  constructor(private readonly g: Grammar) {}

  intend(source: ActorId, goal: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "intend: " + goal, convId, causes, signer);
  }

  decompose(source: ActorId, subtask: string, goal: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "decompose: " + subtask, goal, convId, signer);
  }

  assign(source: ActorId, assignee: ActorId, scope: DomainScope, weight: Weight, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.delegate(source, assignee, scope, weight, cause, convId, signer);
  }

  claim(source: ActorId, work: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "claim: " + work, convId, causes, signer);
  }

  prioritize(source: ActorId, target: EventId, priority: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "priority", priority, convId, signer);
  }

  block(source: ActorId, target: EventId, blocker: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "blocked", blocker, convId, signer);
  }

  unblock(source: ActorId, resolution: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "unblock: " + resolution, convId, causes, signer);
  }

  progress(source: ActorId, update: string, previous: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.extend(source, "progress: " + update, previous, convId, signer);
  }

  complete(source: ActorId, summary: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "complete: " + summary, convId, causes, signer);
  }

  handoff(from: ActorId, to: ActorId, description: string, scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.consent(from, to, "handoff: " + description, scope, cause, convId, signer);
  }

  scope(source: ActorId, target: ActorId, scope: DomainScope, weight: Weight, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.delegate(source, target, scope, weight, cause, convId, signer);
  }

  review(source: ActorId, assessment: string, target: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.respond(source, "review: " + assessment, target, convId, signer);
  }

  standup(
    participants: ActorId[], updates: string[], lead: ActorId, priority: string,
    causes: EventId[], convId: ConversationId, signer: Signer,
  ): StandupResult {
    if (participants.length !== updates.length) {
      throw new Error("standup: participants and updates must have equal length");
    }
    const result: Event[] = [];
    let lastId: EventId | undefined;
    for (let i = 0; i < participants.length; i++) {
      const prev = i === 0 ? (causes.length > 0 ? causes[0] : undefined) : lastId;
      if (!prev) throw new Error("standup: no cause available");
      const p = this.progress(participants[i], updates[i], prev, convId, signer);
      result.push(p);
      lastId = p.id;
    }
    const prio = this.prioritize(lead, lastId!, priority, convId, signer);
    return { updates: result, priority: prio };
  }

  retrospective(
    reviewers: ActorId[], assessments: string[], lead: ActorId, improvement: string,
    target: EventId, convId: ConversationId, signer: Signer,
  ): RetrospectiveResult {
    if (reviewers.length !== assessments.length) {
      throw new Error("retrospective: reviewers and assessments must have equal length");
    }
    const reviews: Event[] = [];
    const reviewIds: EventId[] = [];
    for (let i = 0; i < reviewers.length; i++) {
      const rev = this.review(reviewers[i], assessments[i], target, convId, signer);
      reviews.push(rev);
      reviewIds.push(rev.id);
    }
    const improve = this.intend(lead, improvement, reviewIds, convId, signer);
    return { reviews, improvement: improve };
  }

  triage(
    lead: ActorId, items: EventId[], priorities: string[],
    assignees: ActorId[], scopes: DomainScope[], weights: Weight[],
    convId: ConversationId, signer: Signer,
  ): TriageResult {
    const n = items.length;
    if (priorities.length !== n || assignees.length !== n || scopes.length !== n || weights.length !== n) {
      throw new Error("triage: all arrays must have equal length");
    }
    const result: TriageResult = { priorities: [], assignments: [], scopes: [] };
    for (let i = 0; i < n; i++) {
      const prio = this.prioritize(lead, items[i], priorities[i], convId, signer);
      result.priorities.push(prio);
      const asgn = this.assign(lead, assignees[i], scopes[i], weights[i], prio.id, convId, signer);
      result.assignments.push(asgn);
      const sc = this.scope(lead, assignees[i], scopes[i], weights[i], asgn.id, convId, signer);
      result.scopes.push(sc);
    }
    return result;
  }

  sprint(
    source: ActorId, goal: string, subtasks: string[], assignees: ActorId[],
    scopes: DomainScope[], causes: EventId[], convId: ConversationId, signer: Signer,
  ): SprintResult {
    if (subtasks.length !== assignees.length || subtasks.length !== scopes.length) {
      throw new Error("sprint: subtasks, assignees, and scopes must have equal length");
    }
    const intent = this.intend(source, goal, causes, convId, signer);
    const subs: Event[] = [];
    const asgns: Event[] = [];
    for (let i = 0; i < subtasks.length; i++) {
      const task = this.decompose(source, subtasks[i], intent.id, convId, signer);
      subs.push(task);
      const asgn = this.assign(source, assignees[i], scopes[i], new Weight(0.5), task.id, convId, signer);
      asgns.push(asgn);
    }
    return { intent, subtasks: subs, assignments: asgns };
  }

  escalate(
    source: ActorId, blocker: string, task: EventId, authority: ActorId,
    scope: DomainScope, convId: ConversationId, signer: Signer,
  ): EscalateResult {
    const blockEv = this.block(source, task, blocker, convId, signer);
    const handoffEv = this.handoff(source, authority, blocker, scope, blockEv.id, convId, signer);
    return { blockEvent: blockEv, handoffEvent: handoffEv };
  }

  delegateAndVerify(
    source: ActorId, assignee: ActorId, scope: DomainScope, weight: Weight,
    cause: EventId, convId: ConversationId, signer: Signer,
  ): DelegateAndVerifyResult {
    const assignEv = this.assign(source, assignee, scope, weight, cause, convId, signer);
    const scopeEv = this.scope(source, assignee, scope, weight, assignEv.id, convId, signer);
    return { assignEvent: assignEv, scopeEvent: scopeEv };
  }
}

// ── Layer 2: Market Grammar (Exchange) ──────────────────────────────────

export interface AuctionResult {
  listing: Event;
  bids: Event[];
  acceptance: Event;
}

export interface MilestoneResult {
  acceptance: Event;
  deliveries: Event[];
  payments: Event[];
}

export interface BarterResult {
  listing: Event;
  counterOffer: Event;
  acceptance: Event;
}

export interface MarketSubscriptionResult {
  acceptance: Event;
  payments: Event[];
  deliveries: Event[];
}

export interface RefundResult {
  dispute: Event;
  resolution: Event;
  reversal: Event;
}

export interface ReputationTransferResult {
  ratings: Event[];
}

export interface ArbitrationResult {
  dispute: Event;
  escrow: Event;
  release: Event;
}

export class MarketGrammar {
  constructor(private readonly g: Grammar) {}

  list(source: ActorId, offering: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "list: " + offering, convId, causes, signer);
  }

  bid(source: ActorId, offer: string, listing: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.respond(source, "bid: " + offer, listing, convId, signer);
  }

  inquire(source: ActorId, question: string, listing: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.respond(source, "inquire: " + question, listing, convId, signer);
  }

  negotiate(source: ActorId, counterparty: ActorId, scope: Option<DomainScope>, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.channel(source, counterparty, scope, cause, convId, signer);
  }

  accept(buyer: ActorId, seller: ActorId, terms: string, scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.consent(buyer, seller, "accept: " + terms, scope, cause, convId, signer);
  }

  decline(source: ActorId, reason: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "decline: " + reason, convId, causes, signer);
  }

  invoice(source: ActorId, description: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "invoice: " + description, convId, causes, signer);
  }

  pay(source: ActorId, description: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "pay: " + description, convId, causes, signer);
  }

  deliver(source: ActorId, description: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "deliver: " + description, convId, causes, signer);
  }

  confirm(source: ActorId, confirmation: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "confirm: " + confirmation, convId, causes, signer);
  }

  rate(source: ActorId, target: EventId, targetActor: ActorId, weight: Weight, scope: Option<DomainScope>, convId: ConversationId, signer: Signer): Event {
    return this.g.endorse(source, target, targetActor, weight, scope, convId, signer);
  }

  dispute(source: ActorId, complaint: string, target: EventId, convId: ConversationId, signer: Signer): Event {
    const { disputeFlag } = this.g.challenge(source, "dispute: " + complaint, target, convId, signer);
    return disputeFlag;
  }

  escrow(source: ActorId, escrowActor: ActorId, scope: DomainScope, weight: Weight, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.delegate(source, escrowActor, scope, weight, cause, convId, signer);
  }

  release(partyA: ActorId, partyB: ActorId, terms: string, scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.consent(partyA, partyB, "release: " + terms, scope, cause, convId, signer);
  }

  auction(
    seller: ActorId, offering: string, bidders: ActorId[], bids: string[],
    winnerIdx: number, scope: DomainScope, causes: EventId[], convId: ConversationId, signer: Signer,
  ): AuctionResult {
    if (bidders.length !== bids.length) throw new Error("auction: bidders and bids must have equal length");
    if (winnerIdx < 0 || winnerIdx >= bidders.length) throw new Error("auction: winnerIdx out of range");
    const listing = this.list(seller, offering, causes, convId, signer);
    const bidEvents: Event[] = [];
    for (let i = 0; i < bidders.length; i++) {
      bidEvents.push(this.bid(bidders[i], bids[i], listing.id, convId, signer));
    }
    const acceptance = this.accept(bidders[winnerIdx], seller, "auction won: " + bids[winnerIdx], scope, bidEvents[winnerIdx].id, convId, signer);
    return { listing, bids: bidEvents, acceptance };
  }

  milestone(
    buyer: ActorId, seller: ActorId, terms: string, milestones: string[], payments: string[],
    scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer,
  ): MilestoneResult {
    if (milestones.length !== payments.length) throw new Error("milestone: milestones and payments must have equal length");
    const acceptance = this.accept(buyer, seller, terms, scope, cause, convId, signer);
    const deliveries: Event[] = [];
    const paymentEvents: Event[] = [];
    let prev = acceptance.id;
    for (let i = 0; i < milestones.length; i++) {
      const delivery = this.deliver(seller, milestones[i], [prev], convId, signer);
      deliveries.push(delivery);
      const payment = this.pay(buyer, payments[i], [delivery.id], convId, signer);
      paymentEvents.push(payment);
      prev = payment.id;
    }
    return { acceptance, deliveries, payments: paymentEvents };
  }

  barter(
    partyA: ActorId, partyB: ActorId, offerA: string, offerB: string,
    scope: DomainScope, causes: EventId[], convId: ConversationId, signer: Signer,
  ): BarterResult {
    const listing = this.list(partyA, offerA, causes, convId, signer);
    const counterOffer = this.bid(partyB, offerB, listing.id, convId, signer);
    const acceptance = this.accept(partyA, partyB, "barter: " + offerA + " for " + offerB, scope, counterOffer.id, convId, signer);
    return { listing, counterOffer, acceptance };
  }

  subscription(
    subscriber: ActorId, provider: ActorId, terms: string, periods: string[],
    deliveries: string[], scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer,
  ): MarketSubscriptionResult {
    if (periods.length !== deliveries.length) throw new Error("subscription: periods and deliveries must have equal length");
    const acceptance = this.accept(subscriber, provider, terms, scope, cause, convId, signer);
    const paymentEvents: Event[] = [];
    const deliveryEvents: Event[] = [];
    let prev = acceptance.id;
    for (let i = 0; i < periods.length; i++) {
      const payment = this.pay(subscriber, periods[i], [prev], convId, signer);
      paymentEvents.push(payment);
      const delivery = this.deliver(provider, deliveries[i], [payment.id], convId, signer);
      deliveryEvents.push(delivery);
      prev = delivery.id;
    }
    return { acceptance, payments: paymentEvents, deliveries: deliveryEvents };
  }

  refund(
    buyer: ActorId, seller: ActorId, complaint: string, resolution: string,
    refundAmount: string, target: EventId, convId: ConversationId, signer: Signer,
  ): RefundResult {
    const disputeEv = this.dispute(buyer, complaint, target, convId, signer);
    const resolutionEv = this.g.emit(seller, "resolution: " + resolution, convId, [disputeEv.id], signer);
    const reversal = this.pay(seller, "refund: " + refundAmount, [resolutionEv.id], convId, signer);
    return { dispute: disputeEv, resolution: resolutionEv, reversal };
  }

  reputationTransfer(
    raters: ActorId[], targets: EventId[], targetActor: ActorId, weights: Weight[],
    scope: Option<DomainScope>, convId: ConversationId, signer: Signer,
  ): ReputationTransferResult {
    if (raters.length !== targets.length || raters.length !== weights.length) {
      throw new Error("reputation-transfer: raters, targets, and weights must have equal length");
    }
    const ratings: Event[] = [];
    for (let i = 0; i < raters.length; i++) {
      ratings.push(this.rate(raters[i], targets[i], targetActor, weights[i], scope, convId, signer));
    }
    return { ratings };
  }

  arbitration(
    plaintiff: ActorId, defendant: ActorId, arbiter: ActorId, complaint: string,
    scope: DomainScope, weight: Weight, target: EventId, convId: ConversationId, signer: Signer,
  ): ArbitrationResult {
    const disputeEv = this.dispute(plaintiff, complaint, target, convId, signer);
    const escrowEv = this.escrow(defendant, arbiter, scope, weight, disputeEv.id, convId, signer);
    const releaseEv = this.release(arbiter, plaintiff, "arbitration resolved", scope, escrowEv.id, convId, signer);
    return { dispute: disputeEv, escrow: escrowEv, release: releaseEv };
  }
}

// ── Layer 3: Social Grammar (Society) ───────────────────────────────────

export interface ExileResult {
  exclusion: Event;
  sever: Event;
  sanction: Event;
}

export interface PollResult {
  proposal: Event;
  votes: Event[];
}

export interface FederationResult {
  agreement: Event;
  delegation: Event;
}

export interface SchismResult {
  conflictingNorm: Event;
  exile: ExileResult;
  newCommunity: Event;
}

export class SocialGrammar {
  constructor(private readonly g: Grammar) {}

  norm(proposer: ActorId, supporter: ActorId, norm: string, scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.consent(proposer, supporter, "norm: " + norm, scope, cause, convId, signer);
  }

  moderate(moderator: ActorId, target: EventId, action: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(moderator, target, "moderation", action, convId, signer);
  }

  elect(community: ActorId, elected: ActorId, role: string, scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.consent(community, elected, "elect: " + role, scope, cause, convId, signer);
  }

  welcome(sponsor: ActorId, newcomer: ActorId, weight: Weight, scope: Option<DomainScope>, cause: EventId, convId: ConversationId, signer: Signer): { endorseEv: Event; subscribeEv: Event } {
    return this.g.invite(sponsor, newcomer, weight, scope, cause, convId, signer);
  }

  exile(moderator: ActorId, edge: EdgeId, reason: string, cause: EventId, convId: ConversationId, signer: Signer): ExileResult {
    const exclusion = this.g.emit(moderator, "exile: " + reason, convId, [cause], signer);
    const sever = this.g.sever(moderator, edge, exclusion.id, convId, signer);
    const sanction = this.g.annotate(moderator, sever.id, "sanction", reason, convId, signer);
    return { exclusion, sever, sanction };
  }

  poll(proposer: ActorId, question: string, voters: ActorId[], scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer): PollResult {
    const proposal = this.g.emit(proposer, "poll: " + question, convId, [cause], signer);
    const votes: Event[] = [];
    for (const voter of voters) {
      votes.push(this.g.consent(voter, proposer, "vote: " + question, scope, proposal.id, convId, signer));
    }
    return { proposal, votes };
  }

  federation(communityA: ActorId, communityB: ActorId, terms: string, scope: DomainScope, weight: Weight, cause: EventId, convId: ConversationId, signer: Signer): FederationResult {
    const agreement = this.g.consent(communityA, communityB, "federation: " + terms, scope, cause, convId, signer);
    const delegation = this.g.delegate(communityA, communityB, scope, weight, agreement.id, convId, signer);
    return { agreement, delegation };
  }

  schism(
    faction: ActorId, moderator: ActorId, conflictingNorm: string, scope: DomainScope,
    edge: EdgeId, reason: string, cause: EventId, convId: ConversationId, signer: Signer,
  ): SchismResult {
    const normEv = this.g.emit(faction, "conflicting-norm: " + conflictingNorm, convId, [cause], signer);
    const exileResult = this.exile(moderator, edge, reason, normEv.id, convId, signer);
    const newCommunity = this.g.emit(faction, "new-community: split over " + conflictingNorm, convId, [exileResult.sanction.id], signer);
    return { conflictingNorm: normEv, exile: exileResult, newCommunity };
  }
}

// ── Layer 4: Justice Grammar (Legal) ────────────────────────────────────

export interface TrialResult {
  filing: Event;
  submissions: Event[];
  arguments: Event[];
  ruling: Event;
}

export interface ConstitutionalAmendmentResult {
  reform: Event;
  legislation: Event;
  rightsCheck: Event;
}

export interface InjunctionResult {
  filing: Event;
  ruling: Event;
  enforcement: Event;
}

export interface PleaResult {
  filing: Event;
  acceptance: Event;
  enforcement: Event;
}

export interface ClassActionResult {
  filings: Event[];
  merged: Event;
  trial: TrialResult;
}

export interface RecallResult {
  audit: Event;
  filing: Event;
  consent: Event;
  revocation: Event;
}

export class JusticeGrammar {
  constructor(private readonly g: Grammar) {}

  legislate(source: ActorId, rule: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "legislate: " + rule, convId, causes, signer);
  }

  amend(source: ActorId, amendment: string, rule: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "amend: " + amendment, rule, convId, signer);
  }

  repeal(source: ActorId, rule: EventId, reason: string, convId: ConversationId, signer: Signer): Event {
    return this.g.retract(source, rule, reason, convId, signer);
  }

  file(source: ActorId, complaint: string, target: EventId, convId: ConversationId, signer: Signer): Event {
    const { disputeFlag } = this.g.challenge(source, "file: " + complaint, target, convId, signer);
    return disputeFlag;
  }

  submit(source: ActorId, target: EventId, evidence: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "evidence", evidence, convId, signer);
  }

  argue(source: ActorId, argument: string, target: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.respond(source, "argue: " + argument, target, convId, signer);
  }

  judge(source: ActorId, ruling: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "judge: " + ruling, convId, causes, signer);
  }

  appeal(source: ActorId, grounds: string, ruling: EventId, convId: ConversationId, signer: Signer): Event {
    const { disputeFlag } = this.g.challenge(source, "appeal: " + grounds, ruling, convId, signer);
    return disputeFlag;
  }

  enforce(source: ActorId, executor: ActorId, scope: DomainScope, weight: Weight, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.delegate(source, executor, scope, weight, cause, convId, signer);
  }

  audit(source: ActorId, target: EventId, findings: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "audit", findings, convId, signer);
  }

  pardon(authority: ActorId, pardoned: ActorId, terms: string, scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.consent(authority, pardoned, "pardon: " + terms, scope, cause, convId, signer);
  }

  reform(source: ActorId, proposal: string, precedent: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "reform: " + proposal, precedent, convId, signer);
  }

  trial(
    plaintiff: ActorId, defendant: ActorId, judgeActor: ActorId, complaint: string,
    plaintiffEvidence: string, defendantEvidence: string,
    plaintiffArgument: string, defendantArgument: string, ruling: string,
    target: EventId, convId: ConversationId, signer: Signer,
  ): TrialResult {
    const filing = this.file(plaintiff, complaint, target, convId, signer);
    const sub1 = this.submit(plaintiff, filing.id, plaintiffEvidence, convId, signer);
    const sub2 = this.submit(defendant, filing.id, defendantEvidence, convId, signer);
    const arg1 = this.argue(plaintiff, plaintiffArgument, sub1.id, convId, signer);
    const arg2 = this.argue(defendant, defendantArgument, sub2.id, convId, signer);
    const verdict = this.judge(judgeActor, ruling, [arg1.id, arg2.id], convId, signer);
    return { filing, submissions: [sub1, sub2], arguments: [arg1, arg2], ruling: verdict };
  }

  constitutionalAmendment(proposer: ActorId, proposal: string, legislation: string, rightsAssessment: string, precedent: EventId, convId: ConversationId, signer: Signer): ConstitutionalAmendmentResult {
    const reformEv = this.reform(proposer, proposal, precedent, convId, signer);
    const legislateEv = this.legislate(proposer, legislation, [reformEv.id], convId, signer);
    const rights = this.audit(proposer, legislateEv.id, rightsAssessment, convId, signer);
    return { reform: reformEv, legislation: legislateEv, rightsCheck: rights };
  }

  injunction(
    petitioner: ActorId, judgeActor: ActorId, executor: ActorId,
    complaint: string, ruling: string, scope: DomainScope, weight: Weight,
    target: EventId, convId: ConversationId, signer: Signer,
  ): InjunctionResult {
    const filing = this.file(petitioner, complaint, target, convId, signer);
    const verdict = this.judge(judgeActor, "emergency: " + ruling, [filing.id], convId, signer);
    const enforcement = this.enforce(judgeActor, executor, scope, weight, verdict.id, convId, signer);
    return { filing, ruling: verdict, enforcement };
  }

  plea(
    defendant: ActorId, prosecutor: ActorId, executor: ActorId,
    complaint: string, deal: string, scope: DomainScope, weight: Weight,
    target: EventId, convId: ConversationId, signer: Signer,
  ): PleaResult {
    const filing = this.file(prosecutor, complaint, target, convId, signer);
    const acceptance = this.pardon(prosecutor, defendant, deal, scope, filing.id, convId, signer);
    const enforcement = this.enforce(prosecutor, executor, scope, weight, acceptance.id, convId, signer);
    return { filing, acceptance, enforcement };
  }

  classAction(
    plaintiffs: ActorId[], defendant: ActorId, judgeActor: ActorId,
    complaints: string[], evidence: string, argument: string,
    defenseEvidence: string, defenseArgument: string, ruling: string,
    target: EventId, convId: ConversationId, signer: Signer,
  ): ClassActionResult {
    if (plaintiffs.length !== complaints.length) throw new Error("class-action: plaintiffs and complaints must have equal length");
    const filings: Event[] = [];
    const filingIds: EventId[] = [];
    for (let i = 0; i < plaintiffs.length; i++) {
      const filing = this.file(plaintiffs[i], complaints[i], target, convId, signer);
      filings.push(filing);
      filingIds.push(filing.id);
    }
    const merged = this.g.merge(plaintiffs[0], "class-action: merged complaints", filingIds, convId, signer);
    const trialResult = this.trial(plaintiffs[0], defendant, judgeActor, "class-action", evidence, defenseEvidence, argument, defenseArgument, ruling, merged.id, convId, signer);
    return { filings, merged, trial: trialResult };
  }

  recall(
    auditor: ActorId, community: ActorId, official: ActorId,
    findings: string, complaint: string, scope: DomainScope,
    target: EventId, convId: ConversationId, signer: Signer,
  ): RecallResult {
    const auditEv = this.audit(auditor, target, findings, convId, signer);
    const filing = this.file(auditor, complaint, auditEv.id, convId, signer);
    const consentEv = this.g.consent(community, official, "recall: " + complaint, scope, filing.id, convId, signer);
    const revocation = this.g.emit(community, "role-revoked: " + complaint, convId, [consentEv.id], signer);
    return { audit: auditEv, filing, consent: consentEv, revocation };
  }
}

// ── Layer 5: Build Grammar (Technology) ─────────────────────────────────

export interface SpikeResult {
  build: Event;
  test: Event;
  feedback: Event;
  decision: Event;
}

export interface MigrationResult {
  sunset: Event;
  version: Event;
  ship: Event;
  test: Event;
}

export interface TechDebtResult {
  measure: Event;
  debtMark: Event;
  iteration: Event;
}

export interface PipelineResult {
  definition: Event;
  testResult: Event;
  metrics: Event;
  deployment: Event;
}

export interface PostMortemResult {
  feedback: Event[];
  analysis: Event;
  improvements: Event;
}

export class BuildGrammar {
  constructor(private readonly g: Grammar) {}

  build(source: ActorId, artefact: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "build: " + artefact, convId, causes, signer);
  }

  version(source: ActorId, ver: string, previous: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "version: " + ver, previous, convId, signer);
  }

  ship(source: ActorId, deployment: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "ship: " + deployment, convId, causes, signer);
  }

  sunset(source: ActorId, target: EventId, migration: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "deprecated", migration, convId, signer);
  }

  define(source: ActorId, workflow: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "define: " + workflow, convId, causes, signer);
  }

  automate(source: ActorId, automation: string, workflow: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "automate: " + automation, workflow, convId, signer);
  }

  test(source: ActorId, results: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "test: " + results, convId, causes, signer);
  }

  review(source: ActorId, assessment: string, target: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.respond(source, "review: " + assessment, target, convId, signer);
  }

  measure(source: ActorId, target: EventId, scores: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "quality", scores, convId, signer);
  }

  feedback(source: ActorId, fb: string, target: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.respond(source, "feedback: " + fb, target, convId, signer);
  }

  iterate(source: ActorId, improvement: string, previous: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "iterate: " + improvement, previous, convId, signer);
  }

  innovate(source: ActorId, innovation: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "innovate: " + innovation, convId, causes, signer);
  }

  spike(source: ActorId, experiment: string, testResults: string, fb: string, decision: string, causes: EventId[], convId: ConversationId, signer: Signer): SpikeResult {
    const buildEv = this.build(source, "spike: " + experiment, causes, convId, signer);
    const testEv = this.test(source, testResults, [buildEv.id], convId, signer);
    const feedbackEv = this.feedback(source, fb, testEv.id, convId, signer);
    const decisionEv = this.g.emit(source, "spike-decision: " + decision, convId, [feedbackEv.id], signer);
    return { build: buildEv, test: testEv, feedback: feedbackEv, decision: decisionEv };
  }

  migration(source: ActorId, deprecatedTarget: EventId, migrationPath: string, newVersion: string, deployment: string, testResults: string, convId: ConversationId, signer: Signer): MigrationResult {
    const sunsetEv = this.sunset(source, deprecatedTarget, migrationPath, convId, signer);
    const versionEv = this.version(source, newVersion, sunsetEv.id, convId, signer);
    const shipEv = this.ship(source, deployment, [versionEv.id], convId, signer);
    const testEv = this.test(source, testResults, [shipEv.id], convId, signer);
    return { sunset: sunsetEv, version: versionEv, ship: shipEv, test: testEv };
  }

  techDebt(source: ActorId, target: EventId, scores: string, debtDescription: string, plan: string, convId: ConversationId, signer: Signer): TechDebtResult {
    const measureEv = this.measure(source, target, scores, convId, signer);
    const debtMark = this.g.annotate(source, measureEv.id, "tech_debt", debtDescription, convId, signer);
    const iteration = this.iterate(source, plan, debtMark.id, convId, signer);
    return { measure: measureEv, debtMark, iteration };
  }

  pipeline(source: ActorId, workflow: string, testResults: string, metrics: string, deployment: string, causes: EventId[], convId: ConversationId, signer: Signer): PipelineResult {
    const def = this.define(source, workflow, causes, convId, signer);
    const testEv = this.test(source, testResults, [def.id], convId, signer);
    const measureEv = this.measure(source, testEv.id, metrics, convId, signer);
    const shipEv = this.ship(source, deployment, [measureEv.id], convId, signer);
    return { definition: def, testResult: testEv, metrics: measureEv, deployment: shipEv };
  }

  postMortem(lead: ActorId, contributors: ActorId[], feedbacks: string[], analysis: string, improvements: string, incident: EventId, convId: ConversationId, signer: Signer): PostMortemResult {
    if (contributors.length !== feedbacks.length) throw new Error("post-mortem: contributors and feedbacks must have equal length");
    const fbEvents: Event[] = [];
    const fbIds: EventId[] = [];
    for (let i = 0; i < contributors.length; i++) {
      const fb = this.feedback(contributors[i], feedbacks[i], incident, convId, signer);
      fbEvents.push(fb);
      fbIds.push(fb.id);
    }
    const analysisEv = this.measure(lead, fbIds[fbIds.length - 1], "post-mortem: " + analysis, convId, signer);
    const improveEv = this.define(lead, improvements, [analysisEv.id], convId, signer);
    return { feedback: fbEvents, analysis: analysisEv, improvements: improveEv };
  }
}

// ── Layer 6: Knowledge Grammar (Information) ────────────────────────────

export interface FactCheckResult {
  provenance: Event;
  biasCheck: Event;
  verdict: Event;
}

export interface VerifyResult {
  claim: Event;
  provenance: Event;
  corroboration: Event;
}

export interface SurveyResult {
  recalls: Event[];
  abstraction: Event;
  synthesis: Event;
}

export interface KnowledgeBaseResult {
  claims: Event[];
  categories: Event[];
  memory: Event;
}

export interface TransferResult {
  recall: Event;
  encode: Event;
  learn: Event;
}

export class KnowledgeGrammar {
  constructor(private readonly g: Grammar) {}

  claim(source: ActorId, claimText: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "claim: " + claimText, convId, causes, signer);
  }

  categorize(source: ActorId, target: EventId, taxonomy: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "classification", taxonomy, convId, signer);
  }

  abstract(source: ActorId, generalization: string, instances: EventId[], convId: ConversationId, signer: Signer): Event {
    if (instances.length < 2) throw new Error("abstract: requires at least two instances");
    return this.g.merge(source, "abstract: " + generalization, instances, convId, signer);
  }

  encode(source: ActorId, encoding: string, original: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "encode: " + encoding, original, convId, signer);
  }

  infer(source: ActorId, conclusion: string, premise: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "infer: " + conclusion, premise, convId, signer);
  }

  remember(source: ActorId, knowledge: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "remember: " + knowledge, convId, causes, signer);
  }

  challenge(source: ActorId, counterEvidence: string, claimEvent: EventId, convId: ConversationId, signer: Signer): Event {
    const { disputeFlag } = this.g.challenge(source, "challenge: " + counterEvidence, claimEvent, convId, signer);
    return disputeFlag;
  }

  detectBias(source: ActorId, target: EventId, bias: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "bias", bias, convId, signer);
  }

  correct(source: ActorId, correction: string, original: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "correct: " + correction, original, convId, signer);
  }

  trace(source: ActorId, target: EventId, provenance: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "provenance", provenance, convId, signer);
  }

  recall(source: ActorId, query: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "recall: " + query, convId, causes, signer);
  }

  learn(source: ActorId, learning: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "learn: " + learning, convId, causes, signer);
  }

  retract(source: ActorId, claimEvent: EventId, reason: string, convId: ConversationId, signer: Signer): Event {
    return this.g.retract(source, claimEvent, reason, convId, signer);
  }

  factCheck(checker: ActorId, claimEvent: EventId, provenance: string, biasAnalysis: string, verdict: string, convId: ConversationId, signer: Signer): FactCheckResult {
    const traceEv = this.trace(checker, claimEvent, provenance, convId, signer);
    const bias = this.detectBias(checker, claimEvent, biasAnalysis, convId, signer);
    const verdictEv = this.g.merge(checker, "fact-check: " + verdict, [traceEv.id, bias.id], convId, signer);
    return { provenance: traceEv, biasCheck: bias, verdict: verdictEv };
  }

  verify(source: ActorId, claimText: string, provenance: string, corroboration: string, causes: EventId[], convId: ConversationId, signer: Signer): VerifyResult {
    const claimEv = this.claim(source, claimText, causes, convId, signer);
    const traceEv = this.trace(source, claimEv.id, provenance, convId, signer);
    const corroborate = this.claim(source, "corroborate: " + corroboration, [traceEv.id], convId, signer);
    return { claim: claimEv, provenance: traceEv, corroboration: corroborate };
  }

  survey(source: ActorId, queries: string[], generalization: string, synthesis: string, causes: EventId[], convId: ConversationId, signer: Signer): SurveyResult {
    if (queries.length < 2) throw new Error("survey: requires at least two queries");
    const recalls: Event[] = [];
    const recallIds: EventId[] = [];
    for (const query of queries) {
      const r = this.recall(source, query, causes, convId, signer);
      recalls.push(r);
      recallIds.push(r.id);
    }
    const abstraction = this.abstract(source, generalization, recallIds, convId, signer);
    const synthesisClaim = this.claim(source, "synthesis: " + synthesis, [abstraction.id], convId, signer);
    return { recalls, abstraction, synthesis: synthesisClaim };
  }

  knowledgeBase(source: ActorId, claims: string[], taxonomies: string[], memoryLabel: string, causes: EventId[], convId: ConversationId, signer: Signer): KnowledgeBaseResult {
    if (claims.length !== taxonomies.length) throw new Error("knowledge-base: claims and taxonomies must have equal length");
    const claimEvents: Event[] = [];
    const categories: Event[] = [];
    const catIds: EventId[] = [];
    for (let i = 0; i < claims.length; i++) {
      const c = this.claim(source, claims[i], causes, convId, signer);
      claimEvents.push(c);
      const cat = this.categorize(source, c.id, taxonomies[i], convId, signer);
      categories.push(cat);
      catIds.push(cat.id);
    }
    const memory = this.remember(source, memoryLabel, catIds, convId, signer);
    return { claims: claimEvents, categories, memory };
  }

  transfer(source: ActorId, query: string, encoding: string, learning: string, causes: EventId[], convId: ConversationId, signer: Signer): TransferResult {
    const recallEv = this.recall(source, query, causes, convId, signer);
    const encodeEv = this.encode(source, encoding, recallEv.id, convId, signer);
    const learnEv = this.learn(source, learning, [encodeEv.id], convId, signer);
    return { recall: recallEv, encode: encodeEv, learn: learnEv };
  }
}

// ── Layer 7: Alignment Grammar (Ethics) ─────────────────────────────────

export interface EthicsAuditResult {
  fairness: Event;
  harmScan: Event;
  report: Event;
}

export interface RestorativeJusticeResult {
  harmDetection: Event;
  responsibility: Event;
  redress: Event;
  growth: Event;
}

export interface GuardrailResult {
  constraint: Event;
  dilemma: Event;
  escalation: Event;
}

export interface ImpactAssessmentResult {
  weighing: Event;
  fairness: Event;
  explanation: Event;
}

export interface WhistleblowResult {
  harm: Event;
  explanation: Event;
  escalation: Event;
}

export class AlignmentGrammar {
  constructor(private readonly g: Grammar) {}

  constrain(source: ActorId, target: EventId, constraint: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "constraint", constraint, convId, signer);
  }

  detectHarm(source: ActorId, harm: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "harm: " + harm, convId, causes, signer);
  }

  assessFairness(source: ActorId, target: EventId, assessment: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "fairness", assessment, convId, signer);
  }

  flagDilemma(source: ActorId, dilemma: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "dilemma: " + dilemma, convId, causes, signer);
  }

  weigh(source: ActorId, weighing: string, decision: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "weigh: " + weighing, decision, convId, signer);
  }

  explain(source: ActorId, explanation: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "explain: " + explanation, convId, causes, signer);
  }

  assign(source: ActorId, target: EventId, responsibility: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "responsibility", responsibility, convId, signer);
  }

  repair(source: ActorId, affected: ActorId, redress: string, scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.consent(source, affected, "repair: " + redress, scope, cause, convId, signer);
  }

  care(source: ActorId, careText: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "care: " + careText, convId, causes, signer);
  }

  grow(source: ActorId, growth: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "grow: " + growth, convId, causes, signer);
  }

  ethicsAudit(auditor: ActorId, target: EventId, fairnessAssessment: string, harmScan: string, summary: string, convId: ConversationId, signer: Signer): EthicsAuditResult {
    const fairness = this.assessFairness(auditor, target, fairnessAssessment, convId, signer);
    const harm = this.detectHarm(auditor, harmScan, [fairness.id], convId, signer);
    const report = this.explain(auditor, summary, [fairness.id, harm.id], convId, signer);
    return { fairness, harmScan: harm, report };
  }

  restorativeJustice(
    auditor: ActorId, agent: ActorId, affected: ActorId,
    harm: string, responsibility: string, redress: string, growth: string,
    scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer,
  ): RestorativeJusticeResult {
    const harmEv = this.detectHarm(auditor, harm, [cause], convId, signer);
    const assignEv = this.assign(auditor, harmEv.id, responsibility, convId, signer);
    const repairEv = this.repair(auditor, affected, redress, scope, assignEv.id, convId, signer);
    const growEv = this.grow(agent, growth, [repairEv.id], convId, signer);
    return { harmDetection: harmEv, responsibility: assignEv, redress: repairEv, growth: growEv };
  }

  guardrail(source: ActorId, target: EventId, constraint: string, dilemma: string, escalation: string, convId: ConversationId, signer: Signer): GuardrailResult {
    const constraintEv = this.constrain(source, target, constraint, convId, signer);
    const dilemmaEv = this.flagDilemma(source, dilemma, [constraintEv.id], convId, signer);
    const escalateEv = this.g.emit(source, "escalate: " + escalation, convId, [dilemmaEv.id], signer);
    return { constraint: constraintEv, dilemma: dilemmaEv, escalation: escalateEv };
  }

  impactAssessment(source: ActorId, decision: EventId, weighing: string, fairness: string, explanation: string, convId: ConversationId, signer: Signer): ImpactAssessmentResult {
    const weighEv = this.weigh(source, weighing, decision, convId, signer);
    const fair = this.assessFairness(source, weighEv.id, fairness, convId, signer);
    const explain = this.explain(source, explanation, [weighEv.id, fair.id], convId, signer);
    return { weighing: weighEv, fairness: fair, explanation: explain };
  }

  whistleblow(source: ActorId, harm: string, explanation: string, escalation: string, causes: EventId[], convId: ConversationId, signer: Signer): WhistleblowResult {
    const harmEv = this.detectHarm(source, harm, causes, convId, signer);
    const explain = this.explain(source, explanation, [harmEv.id], convId, signer);
    const escalate = this.g.emit(source, "escalate-external: " + escalation, convId, [explain.id], signer);
    return { harm: harmEv, explanation: explain, escalation: escalate };
  }
}

// ── Layer 8: Identity Grammar ───────────────────────────────────────────

export interface IdentityAuditResult {
  selfModel: Event;
  alignment: Event;
  narrative: Event;
}

export interface RetirementResult {
  memorial: Event;
  transfer: Event;
  archive: Event;
}

export interface CredentialResult {
  introspection: Event;
  disclosure: Event;
}

export interface ReinventionResult {
  transformation: Event;
  narrative: Event;
  aspiration: Event;
}

export interface IntroductionResult {
  disclosure: Event;
  narrative: Event;
}

export class IdentityGrammar {
  constructor(private readonly g: Grammar) {}

  introspect(source: ActorId, selfModel: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "introspect: " + selfModel, convId, causes, signer);
  }

  narrate(source: ActorId, narrative: string, basis: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "narrate: " + narrative, basis, convId, signer);
  }

  align(source: ActorId, target: EventId, alignment: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "alignment", alignment, convId, signer);
  }

  bound(source: ActorId, boundary: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "bound: " + boundary, convId, causes, signer);
  }

  aspire(source: ActorId, aspiration: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "aspire: " + aspiration, convId, causes, signer);
  }

  transform(source: ActorId, transformation: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "transform: " + transformation, convId, causes, signer);
  }

  disclose(source: ActorId, target: ActorId, scope: Option<DomainScope>, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.channel(source, target, scope, cause, convId, signer);
  }

  recognize(source: ActorId, recognition: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "recognize: " + recognition, convId, causes, signer);
  }

  distinguish(source: ActorId, target: EventId, uniqueness: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "uniqueness", uniqueness, convId, signer);
  }

  memorialize(source: ActorId, memorial: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "memorialize: " + memorial, convId, causes, signer);
  }

  identityAudit(source: ActorId, selfModel: string, alignment: string, narrative: string, causes: EventId[], convId: ConversationId, signer: Signer): IdentityAuditResult {
    const intro = this.introspect(source, selfModel, causes, convId, signer);
    const alignEv = this.align(source, intro.id, alignment, convId, signer);
    const narr = this.narrate(source, narrative, alignEv.id, convId, signer);
    return { selfModel: intro, alignment: alignEv, narrative: narr };
  }

  retirement(
    system: ActorId, departing: ActorId, successor: ActorId, memorial: string,
    scope: DomainScope, weight: Weight, causes: EventId[], convId: ConversationId, signer: Signer,
  ): RetirementResult {
    const mem = this.memorialize(system, `retirement of ${departing.value}: ${memorial}`, causes, convId, signer);
    const transferEv = this.g.delegate(system, successor, scope, weight, mem.id, convId, signer);
    const archive = this.g.emit(system, `archive: contributions of ${departing.value}`, convId, [transferEv.id], signer);
    return { memorial: mem, transfer: transferEv, archive };
  }

  credential(source: ActorId, verifier: ActorId, selfModel: string, scope: Option<DomainScope>, causes: EventId[], convId: ConversationId, signer: Signer): CredentialResult {
    const intro = this.introspect(source, selfModel, causes, convId, signer);
    const disclosure = this.disclose(source, verifier, scope, intro.id, convId, signer);
    return { introspection: intro, disclosure };
  }

  reinvention(source: ActorId, transformation: string, narrative: string, aspiration: string, causes: EventId[], convId: ConversationId, signer: Signer): ReinventionResult {
    const transformEv = this.transform(source, transformation, causes, convId, signer);
    const narr = this.narrate(source, narrative, transformEv.id, convId, signer);
    const aspireEv = this.aspire(source, aspiration, [narr.id], convId, signer);
    return { transformation: transformEv, narrative: narr, aspiration: aspireEv };
  }

  introduction(source: ActorId, target: ActorId, scope: Option<DomainScope>, narrative: string, cause: EventId, convId: ConversationId, signer: Signer): IntroductionResult {
    const disclosure = this.disclose(source, target, scope, cause, convId, signer);
    const narr = this.narrate(source, narrative, disclosure.id, convId, signer);
    return { disclosure, narrative: narr };
  }
}

// ── Layer 9: Bond Grammar (Relationship) ────────────────────────────────

export interface BetrayalRepairResult {
  rupture: Event;
  apology: Event;
  reconciliation: Event;
  deepened: Event;
}

export interface CheckInResult {
  balance: Event;
  attunement: Event;
  empathy: Event;
}

export interface BondMentorshipResult {
  connection: Event;
  deepening: Event;
  attunement: Event;
  teaching: Event;
}

export interface BondFarewellResult {
  mourning: Event;
  memorial: Event;
  gratitude: Event;
}

export class BondGrammar {
  constructor(private readonly g: Grammar) {}

  connect(source: ActorId, target: ActorId, scope: Option<DomainScope>, cause: EventId, convId: ConversationId, signer: Signer): { sub1: Event; sub2: Event } {
    const sub1 = this.g.subscribe(source, target, scope, cause, convId, signer);
    const sub2 = this.g.subscribe(target, source, scope, sub1.id, convId, signer);
    return { sub1, sub2 };
  }

  balance(source: ActorId, target: EventId, assessment: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "reciprocity", assessment, convId, signer);
  }

  deepen(source: ActorId, other: ActorId, basis: string, scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.consent(source, other, "deepen: " + basis, scope, cause, convId, signer);
  }

  open(source: ActorId, target: ActorId, scope: Option<DomainScope>, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.channel(source, target, scope, cause, convId, signer);
  }

  attune(source: ActorId, understanding: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "attune: " + understanding, convId, causes, signer);
  }

  feelWith(source: ActorId, empathy: string, target: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.respond(source, "empathy: " + empathy, target, convId, signer);
  }

  break_(source: ActorId, rupture: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "rupture: " + rupture, convId, causes, signer);
  }

  apologize(source: ActorId, apology: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "apology: " + apology, convId, causes, signer);
  }

  reconcile(source: ActorId, other: ActorId, progress: string, scope: DomainScope, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.consent(source, other, "reconcile: " + progress, scope, cause, convId, signer);
  }

  mourn(source: ActorId, loss: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "mourn: " + loss, convId, causes, signer);
  }

  betrayalRepair(
    injured: ActorId, offender: ActorId,
    rupture: string, apology: string, reconciliation: string, newBasis: string,
    scope: DomainScope, causes: EventId[], convId: ConversationId, signer: Signer,
  ): BetrayalRepairResult {
    const ruptureEv = this.break_(injured, rupture, causes, convId, signer);
    const apologyEv = this.apologize(offender, apology, [ruptureEv.id], convId, signer);
    const reconcileEv = this.reconcile(injured, offender, reconciliation, scope, apologyEv.id, convId, signer);
    const deepened = this.deepen(injured, offender, newBasis, scope, reconcileEv.id, convId, signer);
    return { rupture: ruptureEv, apology: apologyEv, reconciliation: reconcileEv, deepened };
  }

  checkIn(source: ActorId, balanceTarget: EventId, assessment: string, attunement: string, empathy: string, convId: ConversationId, signer: Signer): CheckInResult {
    const bal = this.balance(source, balanceTarget, assessment, convId, signer);
    const att = this.attune(source, attunement, [bal.id], convId, signer);
    const emp = this.feelWith(source, empathy, att.id, convId, signer);
    return { balance: bal, attunement: att, empathy: emp };
  }

  mentorship(
    mentor: ActorId, mentee: ActorId, basis: string, understanding: string,
    scope: DomainScope, teachingScope: Option<DomainScope>,
    cause: EventId, convId: ConversationId, signer: Signer,
  ): BondMentorshipResult {
    const connection = this.g.subscribe(mentee, mentor, teachingScope, cause, convId, signer);
    const deepening = this.deepen(mentor, mentee, basis, scope, connection.id, convId, signer);
    const attunementEv = this.attune(mentor, understanding, [deepening.id], convId, signer);
    const teaching = this.g.channel(mentor, mentee, teachingScope, attunementEv.id, convId, signer);
    return { connection, deepening, attunement: attunementEv, teaching };
  }

  farewell(
    source: ActorId, departing: ActorId, loss: string, memorial: string,
    gratitudeWeight: Weight, scope: Option<DomainScope>,
    causes: EventId[], convId: ConversationId, signer: Signer,
  ): BondFarewellResult {
    const mourning = this.mourn(source, loss, causes, convId, signer);
    const mem = this.g.emit(source, "memorialize: " + memorial, convId, [mourning.id], signer);
    const gratitude = this.g.endorse(source, mem.id, departing, gratitudeWeight, scope, convId, signer);
    return { mourning, memorial: mem, gratitude };
  }

  forgive(source: ActorId, severEvent: EventId, target: ActorId, scope: Option<DomainScope>, convId: ConversationId, signer: Signer): Event {
    return this.g.forgive(source, severEvent, target, scope, convId, signer);
  }
}

// ── Layer 10: Belonging Grammar (Community) ─────────────────────────────

export interface FestivalResult {
  celebration: Event;
  practice: Event;
  story: Event;
  gift: Event;
}

export interface CommonsGovernanceResult {
  stewardship: Event;
  assessment: Event;
  legislation: Event;
  audit: Event;
}

export interface RenewalResult {
  assessment: Event;
  practice: Event;
  story: Event;
}

export interface OnboardResult {
  inclusion: Event;
  settlement: Event;
  firstPractice: Event;
  contribution: Event;
}

export interface SuccessionResult {
  assessment: Event;
  transfer: Event;
  celebration: Event;
  story: Event;
}

export class BelongingGrammar {
  constructor(private readonly g: Grammar) {}

  settle(source: ActorId, community: ActorId, scope: Option<DomainScope>, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.subscribe(source, community, scope, cause, convId, signer);
  }

  contribute(source: ActorId, contribution: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "contribute: " + contribution, convId, causes, signer);
  }

  include(source: ActorId, action: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "include: " + action, convId, causes, signer);
  }

  practice(source: ActorId, tradition: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "practice: " + tradition, convId, causes, signer);
  }

  steward(source: ActorId, stewardActor: ActorId, scope: DomainScope, weight: Weight, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.delegate(source, stewardActor, scope, weight, cause, convId, signer);
  }

  sustain(source: ActorId, assessment: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "sustain: " + assessment, convId, causes, signer);
  }

  passOn(from: ActorId, to: ActorId, scope: DomainScope, description: string, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.consent(from, to, "pass-on: " + description, scope, cause, convId, signer);
  }

  celebrate(source: ActorId, celebration: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "celebrate: " + celebration, convId, causes, signer);
  }

  tell(source: ActorId, story: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "tell: " + story, convId, causes, signer);
  }

  gift(source: ActorId, giftText: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "gift: " + giftText, convId, causes, signer);
  }

  festival(source: ActorId, celebration: string, tradition: string, story: string, giftText: string, causes: EventId[], convId: ConversationId, signer: Signer): FestivalResult {
    const celebrateEv = this.celebrate(source, celebration, causes, convId, signer);
    const practiceEv = this.practice(source, tradition, [celebrateEv.id], convId, signer);
    const tellEv = this.tell(source, story, [practiceEv.id], convId, signer);
    const giftEv = this.gift(source, giftText, [tellEv.id], convId, signer);
    return { celebration: celebrateEv, practice: practiceEv, story: tellEv, gift: giftEv };
  }

  commonsGovernance(
    source: ActorId, stewardActor: ActorId, scope: DomainScope, weight: Weight,
    assessment: string, rule: string, findings: string,
    cause: EventId, convId: ConversationId, signer: Signer,
  ): CommonsGovernanceResult {
    const stewardship = this.steward(source, stewardActor, scope, weight, cause, convId, signer);
    const sustainEv = this.sustain(stewardActor, assessment, [stewardship.id], convId, signer);
    const legislate = this.g.emit(source, "legislate: " + rule, convId, [sustainEv.id], signer);
    const audit = this.g.annotate(stewardActor, legislate.id, "audit", findings, convId, signer);
    return { stewardship, assessment: sustainEv, legislation: legislate, audit };
  }

  renewal(source: ActorId, assessment: string, evolvedPractice: string, newStory: string, causes: EventId[], convId: ConversationId, signer: Signer): RenewalResult {
    const sustainEv = this.sustain(source, assessment, causes, convId, signer);
    const practiceEv = this.practice(source, evolvedPractice, [sustainEv.id], convId, signer);
    const storyEv = this.tell(source, newStory, [practiceEv.id], convId, signer);
    return { assessment: sustainEv, practice: practiceEv, story: storyEv };
  }

  onboard(
    sponsor: ActorId, newcomer: ActorId, community: ActorId,
    scope: Option<DomainScope>, inclusionAction: string, tradition: string, firstContribution: string,
    cause: EventId, convId: ConversationId, signer: Signer,
  ): OnboardResult {
    const inclusion = this.include(sponsor, inclusionAction, [cause], convId, signer);
    const settlement = this.settle(newcomer, community, scope, inclusion.id, convId, signer);
    const firstPractice = this.practice(newcomer, tradition, [settlement.id], convId, signer);
    const contribution = this.contribute(newcomer, firstContribution, [firstPractice.id], convId, signer);
    return { inclusion, settlement, firstPractice, contribution };
  }

  succession(
    outgoing: ActorId, incoming: ActorId, assessment: string, scope: DomainScope,
    celebration: string, story: string, cause: EventId, convId: ConversationId, signer: Signer,
  ): SuccessionResult {
    const sustainEv = this.sustain(outgoing, assessment, [cause], convId, signer);
    const transfer = this.passOn(outgoing, incoming, scope, "stewardship transfer", sustainEv.id, convId, signer);
    const celebrateEv = this.celebrate(outgoing, celebration, [transfer.id], convId, signer);
    const storyEv = this.tell(outgoing, story, [celebrateEv.id], convId, signer);
    return { assessment: sustainEv, transfer, celebration: celebrateEv, story: storyEv };
  }
}

// ── Layer 11: Meaning Grammar (Culture) ─────────────────────────────────

export interface DesignReviewResult {
  beauty: Event;
  reframe: Event;
  question: Event;
  wisdom: Event;
}

export interface CulturalOnboardingResult {
  translation: Event;
  teaching: Event;
  examination: Event;
}

export interface ForecastResult {
  prophecy: Event;
  examination: Event;
  wisdom: Event;
}

export interface MeaningPostMortemResult {
  examination: Event;
  questions: Event;
  wisdom: Event;
}

export interface MeaningMentorshipResult {
  channel: Event;
  reframing: Event;
  wisdom: Event;
  translation: Event;
}

export class MeaningGrammar {
  constructor(private readonly g: Grammar) {}

  examine(source: ActorId, examination: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "examine: " + examination, convId, causes, signer);
  }

  reframe(source: ActorId, reframing: string, original: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "reframe: " + reframing, original, convId, signer);
  }

  question(source: ActorId, questionText: string, target: EventId, convId: ConversationId, signer: Signer): Event {
    const { disputeFlag } = this.g.challenge(source, "question: " + questionText, target, convId, signer);
    return disputeFlag;
  }

  distill(source: ActorId, wisdom: string, experience: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "distill: " + wisdom, experience, convId, signer);
  }

  beautify(source: ActorId, beauty: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "beautify: " + beauty, convId, causes, signer);
  }

  liken(source: ActorId, metaphor: string, subject: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "liken: " + metaphor, subject, convId, signer);
  }

  lighten(source: ActorId, humour: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "lighten: " + humour, convId, causes, signer);
  }

  teach(source: ActorId, student: ActorId, scope: Option<DomainScope>, cause: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.channel(source, student, scope, cause, convId, signer);
  }

  translate(source: ActorId, translation: string, original: EventId, convId: ConversationId, signer: Signer): Event {
    return this.g.derive(source, "translate: " + translation, original, convId, signer);
  }

  prophesy(source: ActorId, prediction: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "prophesy: " + prediction, convId, causes, signer);
  }

  designReview(source: ActorId, beauty: string, reframing: string, questionText: string, wisdom: string, cause: EventId, convId: ConversationId, signer: Signer): DesignReviewResult {
    const beautifyEv = this.beautify(source, beauty, [cause], convId, signer);
    const reframeEv = this.reframe(source, reframing, beautifyEv.id, convId, signer);
    const q = this.question(source, questionText, reframeEv.id, convId, signer);
    const w = this.distill(source, wisdom, q.id, convId, signer);
    return { beauty: beautifyEv, reframe: reframeEv, question: q, wisdom: w };
  }

  culturalOnboarding(guide: ActorId, newcomer: ActorId, translation: string, teachingScope: Option<DomainScope>, examination: string, cause: EventId, convId: ConversationId, signer: Signer): CulturalOnboardingResult {
    const translateEv = this.translate(guide, translation, cause, convId, signer);
    const teachEv = this.teach(guide, newcomer, teachingScope, translateEv.id, convId, signer);
    const examineEv = this.examine(newcomer, examination, [teachEv.id], convId, signer);
    return { translation: translateEv, teaching: teachEv, examination: examineEv };
  }

  forecast(source: ActorId, prediction: string, assumptions: string, confidence: string, causes: EventId[], convId: ConversationId, signer: Signer): ForecastResult {
    const prophesyEv = this.prophesy(source, prediction, causes, convId, signer);
    const examineEv = this.examine(source, assumptions, [prophesyEv.id], convId, signer);
    const distillEv = this.distill(source, confidence, examineEv.id, convId, signer);
    return { prophecy: prophesyEv, examination: examineEv, wisdom: distillEv };
  }

  postMortem(source: ActorId, examination: string, questionText: string, wisdom: string, cause: EventId, convId: ConversationId, signer: Signer): MeaningPostMortemResult {
    const exam = this.examine(source, examination, [cause], convId, signer);
    const q = this.question(source, questionText, exam.id, convId, signer);
    const w = this.distill(source, wisdom, q.id, convId, signer);
    return { examination: exam, questions: q, wisdom: w };
  }

  mentorship(
    mentor: ActorId, student: ActorId, reframing: string, wisdom: string, translation: string,
    scope: Option<DomainScope>, cause: EventId, convId: ConversationId, signer: Signer,
  ): MeaningMentorshipResult {
    const channelEv = this.teach(mentor, student, scope, cause, convId, signer);
    const reframeEv = this.reframe(mentor, reframing, channelEv.id, convId, signer);
    const distillEv = this.distill(mentor, wisdom, reframeEv.id, convId, signer);
    const translateEv = this.translate(student, translation, distillEv.id, convId, signer);
    return { channel: channelEv, reframing: reframeEv, wisdom: distillEv, translation: translateEv };
  }
}

// ── Layer 12: Evolution Grammar (Emergence) ─────────────────────────────

export interface SelfEvolveResult {
  pattern: Event;
  adaptation: Event;
  selection: Event;
  simplification: Event;
}

export interface HealthCheckResult {
  integrity: Event;
  resilience: Event;
  model: Event;
  purpose: Event;
}

export interface PruneResult {
  pattern: Event;
  simplification: Event;
  verification: Event;
}

export interface PhaseTransitionResult {
  threshold: Event;
  model: Event;
  adaptation: Event;
  selection: Event;
}

export class EvolutionGrammar {
  constructor(private readonly g: Grammar) {}

  detectPattern(source: ActorId, pattern: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "pattern: " + pattern, convId, causes, signer);
  }

  model(source: ActorId, modelText: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "model: " + modelText, convId, causes, signer);
  }

  traceLoop(source: ActorId, loop: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "loop: " + loop, convId, causes, signer);
  }

  watchThreshold(source: ActorId, target: EventId, threshold: string, convId: ConversationId, signer: Signer): Event {
    return this.g.annotate(source, target, "threshold", threshold, convId, signer);
  }

  adapt(source: ActorId, proposal: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "adapt: " + proposal, convId, causes, signer);
  }

  select(source: ActorId, result: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "select: " + result, convId, causes, signer);
  }

  simplify(source: ActorId, simplification: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "simplify: " + simplification, convId, causes, signer);
  }

  checkIntegrity(source: ActorId, assessment: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "integrity: " + assessment, convId, causes, signer);
  }

  assessResilience(source: ActorId, assessment: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "resilience: " + assessment, convId, causes, signer);
  }

  alignPurpose(source: ActorId, alignment: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "purpose: " + alignment, convId, causes, signer);
  }

  selfEvolve(source: ActorId, pattern: string, adaptation: string, selection: string, simplification: string, causes: EventId[], convId: ConversationId, signer: Signer): SelfEvolveResult {
    const pat = this.detectPattern(source, pattern, causes, convId, signer);
    const adaptEv = this.adapt(source, adaptation, [pat.id], convId, signer);
    const sel = this.select(source, selection, [adaptEv.id], convId, signer);
    const simp = this.simplify(source, simplification, [sel.id], convId, signer);
    return { pattern: pat, adaptation: adaptEv, selection: sel, simplification: simp };
  }

  healthCheck(source: ActorId, integrity: string, resilience: string, modelText: string, purpose: string, causes: EventId[], convId: ConversationId, signer: Signer): HealthCheckResult {
    const integ = this.checkIntegrity(source, integrity, causes, convId, signer);
    const resil = this.assessResilience(source, resilience, [integ.id], convId, signer);
    const mod = this.model(source, modelText, [resil.id], convId, signer);
    const purp = this.alignPurpose(source, purpose, [mod.id], convId, signer);
    return { integrity: integ, resilience: resil, model: mod, purpose: purp };
  }

  prune(source: ActorId, unusedPattern: string, simplification: string, verification: string, causes: EventId[], convId: ConversationId, signer: Signer): PruneResult {
    const pattern = this.detectPattern(source, "unused: " + unusedPattern, causes, convId, signer);
    const simp = this.simplify(source, simplification, [pattern.id], convId, signer);
    const verify = this.select(source, verification, [simp.id], convId, signer);
    return { pattern, simplification: simp, verification: verify };
  }

  phaseTransition(source: ActorId, target: EventId, threshold: string, modelText: string, adaptation: string, selection: string, convId: ConversationId, signer: Signer): PhaseTransitionResult {
    const thresh = this.watchThreshold(source, target, threshold, convId, signer);
    const mod = this.model(source, modelText, [thresh.id], convId, signer);
    const adaptEv = this.adapt(source, adaptation, [mod.id], convId, signer);
    const sel = this.select(source, selection, [adaptEv.id], convId, signer);
    return { threshold: thresh, model: mod, adaptation: adaptEv, selection: sel };
  }
}

// ── Layer 13: Being Grammar (Existence) ─────────────────────────────────

export interface BeingFarewellResult {
  acceptance: Event;
  web: Event;
  awe: Event;
  memorial: Event;
}

export interface ContemplationResult {
  change: Event;
  mystery: Event;
  awe: Event;
  wonder: Event;
}

export interface ExistentialAuditResult {
  existence: Event;
  acceptance: Event;
  web: Event;
  purpose: Event;
}

export class BeingGrammar {
  constructor(private readonly g: Grammar) {}

  exist(source: ActorId, observation: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "exist: " + observation, convId, causes, signer);
  }

  accept(source: ActorId, limitation: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "accept: " + limitation, convId, causes, signer);
  }

  observeChange(source: ActorId, observation: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "change: " + observation, convId, causes, signer);
  }

  mapWeb(source: ActorId, mapping: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "web: " + mapping, convId, causes, signer);
  }

  faceMystery(source: ActorId, mystery: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "mystery: " + mystery, convId, causes, signer);
  }

  holdParadox(source: ActorId, paradox: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "paradox: " + paradox, convId, causes, signer);
  }

  marvel(source: ActorId, awe: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "marvel: " + awe, convId, causes, signer);
  }

  askWhy(source: ActorId, question: string, causes: EventId[], convId: ConversationId, signer: Signer): Event {
    return this.g.emit(source, "wonder: " + question, convId, causes, signer);
  }

  farewell(source: ActorId, limitation: string, interconnection: string, awe: string, memorial: string, causes: EventId[], convId: ConversationId, signer: Signer): BeingFarewellResult {
    const acceptEv = this.accept(source, limitation, causes, convId, signer);
    const web = this.mapWeb(source, interconnection, [acceptEv.id], convId, signer);
    const marvelEv = this.marvel(source, awe, [web.id], convId, signer);
    const mem = this.g.emit(source, "memorialize: " + memorial, convId, [marvelEv.id], signer);
    return { acceptance: acceptEv, web, awe: marvelEv, memorial: mem };
  }

  contemplation(source: ActorId, change: string, mystery: string, awe: string, question: string, causes: EventId[], convId: ConversationId, signer: Signer): ContemplationResult {
    const changeEv = this.observeChange(source, change, causes, convId, signer);
    const mysteryEv = this.faceMystery(source, mystery, [changeEv.id], convId, signer);
    const aweEv = this.marvel(source, awe, [mysteryEv.id], convId, signer);
    const wonderEv = this.askWhy(source, question, [aweEv.id], convId, signer);
    return { change: changeEv, mystery: mysteryEv, awe: aweEv, wonder: wonderEv };
  }

  existentialAudit(source: ActorId, existence: string, limitation: string, interconnection: string, purpose: string, causes: EventId[], convId: ConversationId, signer: Signer): ExistentialAuditResult {
    const existEv = this.exist(source, existence, causes, convId, signer);
    const acceptEv = this.accept(source, limitation, [existEv.id], convId, signer);
    const web = this.mapWeb(source, interconnection, [acceptEv.id], convId, signer);
    const purp = this.g.emit(source, "purpose: " + purpose, convId, [web.id], signer);
    return { existence: existEv, acceptance: acceptEv, web, purpose: purp };
  }
}
