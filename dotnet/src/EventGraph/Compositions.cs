namespace EventGraph;

// ── WorkGrammar (Layer 1 — Agency) ─────────────────────────────────────

/// <summary>Layer 1 composition operations for task management.</summary>
public sealed class WorkGrammar
{
    private readonly Grammar _g;
    public WorkGrammar(Grammar g) => _g = g;

    // --- Operations (12) ---

    public Event Intend(ActorId source, string goal, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "intend: " + goal, convId, causes, signer);

    public Event Decompose(ActorId source, string subtask, EventId goal, ConversationId convId, ISigner signer)
        => _g.Derive(source, "decompose: " + subtask, goal, convId, signer);

    public Event Assign(ActorId source, ActorId assignee, DomainScope scope, Weight weight, EventId cause, ConversationId convId, ISigner signer)
        => _g.Delegate(source, assignee, scope, weight, cause, convId, signer);

    public Event Claim(ActorId source, string work, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "claim: " + work, convId, causes, signer);

    public Event Prioritize(ActorId source, EventId target, string priority, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "priority", priority, convId, signer);

    public Event Block(ActorId source, EventId target, string blocker, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "blocked", blocker, convId, signer);

    public Event Unblock(ActorId source, string resolution, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "unblock: " + resolution, convId, causes, signer);

    public Event Progress(ActorId source, string update, EventId previous, ConversationId convId, ISigner signer)
        => _g.Extend(source, "progress: " + update, previous, convId, signer);

    public Event Complete(ActorId source, string summary, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "complete: " + summary, convId, causes, signer);

    public Event Handoff(ActorId from, ActorId to, string description, DomainScope scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Consent(from, to, "handoff: " + description, scope, cause, convId, signer);

    public Event Scope(ActorId source, ActorId target, DomainScope scope, Weight weight, EventId cause, ConversationId convId, ISigner signer)
        => _g.Delegate(source, target, scope, weight, cause, convId, signer);

    public Event Review(ActorId source, string assessment, EventId target, ConversationId convId, ISigner signer)
        => _g.Respond(source, "review: " + assessment, target, convId, signer);

    // --- Named Functions (6) ---

    public sealed record StandupResult(List<Event> Updates, Event Priority);

    public StandupResult Standup(List<ActorId> participants, List<string> updates,
        ActorId lead, string priority, List<EventId> causes, ConversationId convId, ISigner signer)
    {
        if (participants.Count != updates.Count)
            throw new ArgumentException("Standup: participants and updates must have equal length");

        var resultUpdates = new List<Event>();
        EventId lastId = default;
        for (int i = 0; i < participants.Count; i++)
        {
            var prev = i == 0 ? (causes.Count > 0 ? causes[0] : default) : lastId;
            var progress = Progress(participants[i], updates[i], prev, convId, signer);
            resultUpdates.Add(progress);
            lastId = progress.Id;
        }

        var prio = Prioritize(lead, lastId, priority, convId, signer);
        return new StandupResult(resultUpdates, prio);
    }

    public sealed record RetrospectiveResult(List<Event> Reviews, Event Improvement);

    public RetrospectiveResult Retrospective(List<ActorId> reviewers, List<string> assessments,
        ActorId lead, string improvement, EventId target, ConversationId convId, ISigner signer)
    {
        if (reviewers.Count != assessments.Count)
            throw new ArgumentException("Retrospective: reviewers and assessments must have equal length");

        var reviews = new List<Event>();
        var reviewIds = new List<EventId>();
        for (int i = 0; i < reviewers.Count; i++)
        {
            var rev = Review(reviewers[i], assessments[i], target, convId, signer);
            reviews.Add(rev);
            reviewIds.Add(rev.Id);
        }

        var improve = Intend(lead, improvement, reviewIds, convId, signer);
        return new RetrospectiveResult(reviews, improve);
    }

    public sealed record TriageResult(List<Event> Priorities, List<Event> Assignments, List<Event> Scopes);

    public TriageResult Triage(ActorId lead, List<EventId> items, List<string> priorities,
        List<ActorId> assignees, List<DomainScope> scopes, List<Weight> weights,
        ConversationId convId, ISigner signer)
    {
        int n = items.Count;
        if (priorities.Count != n || assignees.Count != n || scopes.Count != n || weights.Count != n)
            throw new ArgumentException("Triage: all lists must have equal length");

        var resultPriorities = new List<Event>();
        var resultAssignments = new List<Event>();
        var resultScopes = new List<Event>();
        for (int i = 0; i < n; i++)
        {
            var prio = Prioritize(lead, items[i], priorities[i], convId, signer);
            resultPriorities.Add(prio);

            var assign = Assign(lead, assignees[i], scopes[i], weights[i], prio.Id, convId, signer);
            resultAssignments.Add(assign);

            var scope = Scope(lead, assignees[i], scopes[i], weights[i], assign.Id, convId, signer);
            resultScopes.Add(scope);
        }
        return new TriageResult(resultPriorities, resultAssignments, resultScopes);
    }

    public sealed record SprintResult(Event Intent, List<Event> Subtasks, List<Event> Assignments);

    public SprintResult Sprint(ActorId source, string goal, List<string> subtasks,
        List<ActorId> assignees, List<DomainScope> scopes, List<EventId> causes,
        ConversationId convId, ISigner signer)
    {
        if (subtasks.Count != assignees.Count || subtasks.Count != scopes.Count)
            throw new ArgumentException("Sprint: subtasks, assignees, and scopes must have equal length");

        var intent = Intend(source, goal, causes, convId, signer);
        var resultSubtasks = new List<Event>();
        var resultAssignments = new List<Event>();
        for (int i = 0; i < subtasks.Count; i++)
        {
            var task = Decompose(source, subtasks[i], intent.Id, convId, signer);
            resultSubtasks.Add(task);

            var assign = Assign(source, assignees[i], scopes[i], new Weight(0.5), task.Id, convId, signer);
            resultAssignments.Add(assign);
        }
        return new SprintResult(intent, resultSubtasks, resultAssignments);
    }

    public sealed record EscalateResult(Event BlockEvent, Event HandoffEvent);

    public EscalateResult Escalate(ActorId source, string blocker, EventId task,
        ActorId authority, DomainScope scope, ConversationId convId, ISigner signer)
    {
        var block = Block(source, task, blocker, convId, signer);
        var handoff = Handoff(source, authority, blocker, scope, block.Id, convId, signer);
        return new EscalateResult(block, handoff);
    }

    public sealed record DelegateAndVerifyResult(Event AssignEvent, Event ScopeEvent);

    public DelegateAndVerifyResult DelegateAndVerify(ActorId source, ActorId assignee,
        DomainScope scope, Weight weight, EventId cause, ConversationId convId, ISigner signer)
    {
        var assign = Assign(source, assignee, scope, weight, cause, convId, signer);
        var scopeEv = Scope(source, assignee, scope, weight, assign.Id, convId, signer);
        return new DelegateAndVerifyResult(assign, scopeEv);
    }
}

// ── MarketGrammar (Layer 2 — Exchange) ─────────────────────────────────

/// <summary>Layer 2 composition operations for trust-based marketplaces.</summary>
public sealed class MarketGrammar
{
    private readonly Grammar _g;
    public MarketGrammar(Grammar g) => _g = g;

    public Event List(ActorId source, string offering, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "list: " + offering, convId, causes, signer);

    public Event Bid(ActorId source, string offer, EventId listing, ConversationId convId, ISigner signer)
        => _g.Respond(source, "bid: " + offer, listing, convId, signer);

    public Event Inquire(ActorId source, string question, EventId listing, ConversationId convId, ISigner signer)
        => _g.Respond(source, "inquire: " + question, listing, convId, signer);

    public Event Negotiate(ActorId source, ActorId counterparty, Option<DomainScope> scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Channel(source, counterparty, scope, cause, convId, signer);

    public Event Accept(ActorId buyer, ActorId seller, string terms, DomainScope scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Consent(buyer, seller, "accept: " + terms, scope, cause, convId, signer);

    public Event Decline(ActorId source, string reason, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "decline: " + reason, convId, causes, signer);

    public Event Invoice(ActorId source, string description, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "invoice: " + description, convId, causes, signer);

    public Event Pay(ActorId source, string description, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "pay: " + description, convId, causes, signer);

    public Event Deliver(ActorId source, string description, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "deliver: " + description, convId, causes, signer);

    public Event Confirm(ActorId source, string confirmation, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "confirm: " + confirmation, convId, causes, signer);

    public Event Rate(ActorId source, EventId target, ActorId targetActor, Weight weight, Option<DomainScope> scope, ConversationId convId, ISigner signer)
        => _g.Endorse(source, target, targetActor, weight, scope, convId, signer);

    public Event Dispute(ActorId source, string complaint, EventId target, ConversationId convId, ISigner signer)
    {
        var (_, flag) = _g.Challenge(source, "dispute: " + complaint, target, convId, signer);
        return flag;
    }

    public Event Escrow(ActorId source, ActorId escrowActor, DomainScope scope, Weight weight, EventId cause, ConversationId convId, ISigner signer)
        => _g.Delegate(source, escrowActor, scope, weight, cause, convId, signer);

    public Event Release(ActorId partyA, ActorId partyB, string terms, DomainScope scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Consent(partyA, partyB, "release: " + terms, scope, cause, convId, signer);

    // Named Functions

    public sealed record AuctionResult(Event Listing, List<Event> Bids, Event Acceptance);

    public AuctionResult Auction(ActorId seller, string offering, List<ActorId> bidders, List<string> bids,
        int winnerIdx, DomainScope scope, List<EventId> causes, ConversationId convId, ISigner signer)
    {
        if (bidders.Count != bids.Count)
            throw new ArgumentException("Auction: bidders and bids must have equal length");
        if (winnerIdx < 0 || winnerIdx >= bidders.Count)
            throw new ArgumentOutOfRangeException(nameof(winnerIdx));

        var listing = List(seller, offering, causes, convId, signer);
        var resultBids = new List<Event>();
        for (int i = 0; i < bidders.Count; i++)
        {
            resultBids.Add(Bid(bidders[i], bids[i], listing.Id, convId, signer));
        }

        var acceptance = Accept(bidders[winnerIdx], seller, "auction won: " + bids[winnerIdx], scope, resultBids[winnerIdx].Id, convId, signer);
        return new AuctionResult(listing, resultBids, acceptance);
    }

    public sealed record MilestoneResult(Event Acceptance, List<Event> Deliveries, List<Event> Payments);

    public MilestoneResult Milestone(ActorId buyer, ActorId seller, string terms,
        List<string> milestones, List<string> payments, DomainScope scope,
        EventId cause, ConversationId convId, ISigner signer)
    {
        if (milestones.Count != payments.Count)
            throw new ArgumentException("Milestone: milestones and payments must have equal length");

        var acceptance = Accept(buyer, seller, terms, scope, cause, convId, signer);
        var deliveries = new List<Event>();
        var paymentEvs = new List<Event>();
        var prev = acceptance.Id;
        for (int i = 0; i < milestones.Count; i++)
        {
            var delivery = Deliver(seller, milestones[i], new List<EventId> { prev }, convId, signer);
            deliveries.Add(delivery);

            var payment = Pay(buyer, payments[i], new List<EventId> { delivery.Id }, convId, signer);
            paymentEvs.Add(payment);
            prev = payment.Id;
        }
        return new MilestoneResult(acceptance, deliveries, paymentEvs);
    }

    public sealed record BarterResult(Event Listing, Event CounterOffer, Event Acceptance);

    public BarterResult Barter(ActorId partyA, ActorId partyB, string offerA, string offerB,
        DomainScope scope, List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var listing = List(partyA, offerA, causes, convId, signer);
        var counter = Bid(partyB, offerB, listing.Id, convId, signer);
        var acceptance = Accept(partyA, partyB, "barter: " + offerA + " for " + offerB, scope, counter.Id, convId, signer);
        return new BarterResult(listing, counter, acceptance);
    }

    public sealed record SubscriptionResult(Event Acceptance, List<Event> Payments, List<Event> Deliveries);

    public SubscriptionResult Subscription(ActorId subscriber, ActorId provider, string terms,
        List<string> periods, List<string> deliveries, DomainScope scope,
        EventId cause, ConversationId convId, ISigner signer)
    {
        if (periods.Count != deliveries.Count)
            throw new ArgumentException("Subscription: periods and deliveries must have equal length");

        var acceptance = Accept(subscriber, provider, terms, scope, cause, convId, signer);
        var paymentEvs = new List<Event>();
        var deliveryEvs = new List<Event>();
        var prev = acceptance.Id;
        for (int i = 0; i < periods.Count; i++)
        {
            var payment = Pay(subscriber, periods[i], new List<EventId> { prev }, convId, signer);
            paymentEvs.Add(payment);

            var delivery = Deliver(provider, deliveries[i], new List<EventId> { payment.Id }, convId, signer);
            deliveryEvs.Add(delivery);
            prev = delivery.Id;
        }
        return new SubscriptionResult(acceptance, paymentEvs, deliveryEvs);
    }

    public sealed record RefundResult(Event Dispute, Event Resolution, Event Reversal);

    public RefundResult Refund(ActorId buyer, ActorId seller, string complaint, string resolution,
        string refundAmount, EventId target, ConversationId convId, ISigner signer)
    {
        var dispute = Dispute(buyer, complaint, target, convId, signer);
        var resolutionEv = _g.Emit(seller, "resolution: " + resolution, convId, new List<EventId> { dispute.Id }, signer);
        var reversal = Pay(seller, "refund: " + refundAmount, new List<EventId> { resolutionEv.Id }, convId, signer);
        return new RefundResult(dispute, resolutionEv, reversal);
    }

    public sealed record ReputationTransferResult(List<Event> Ratings);

    public ReputationTransferResult ReputationTransfer(List<ActorId> raters, List<EventId> targets,
        ActorId targetActor, List<Weight> weights, Option<DomainScope> scope,
        ConversationId convId, ISigner signer)
    {
        if (raters.Count != targets.Count || raters.Count != weights.Count)
            throw new ArgumentException("ReputationTransfer: raters, targets, and weights must have equal length");

        var ratings = new List<Event>();
        for (int i = 0; i < raters.Count; i++)
            ratings.Add(Rate(raters[i], targets[i], targetActor, weights[i], scope, convId, signer));
        return new ReputationTransferResult(ratings);
    }

    public sealed record ArbitrationResult(Event Dispute, Event Escrow, Event Release);

    public ArbitrationResult Arbitration(ActorId plaintiff, ActorId defendant, ActorId arbiter,
        string complaint, DomainScope scope, Weight weight,
        EventId target, ConversationId convId, ISigner signer)
    {
        var dispute = Dispute(plaintiff, complaint, target, convId, signer);
        var escrow = this.Escrow(defendant, arbiter, scope, weight, dispute.Id, convId, signer);
        var release = Release(arbiter, plaintiff, "arbitration resolved", scope, escrow.Id, convId, signer);
        return new ArbitrationResult(dispute, escrow, release);
    }
}

// ── SocialGrammar (Layer 3 — Society) ──────────────────────────────────

/// <summary>Layer 3 composition operations for user-owned social platforms.</summary>
public sealed class SocialGrammar
{
    private readonly Grammar _g;
    public SocialGrammar(Grammar g) => _g = g;

    public Event Norm(ActorId proposer, ActorId supporter, string norm, DomainScope scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Consent(proposer, supporter, "norm: " + norm, scope, cause, convId, signer);

    public Event Moderate(ActorId moderator, EventId target, string action, ConversationId convId, ISigner signer)
        => _g.Annotate(moderator, target, "moderation", action, convId, signer);

    public Event Elect(ActorId community, ActorId elected, string role, DomainScope scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Consent(community, elected, "elect: " + role, scope, cause, convId, signer);

    public (Event EndorseEv, Event SubscribeEv) Welcome(ActorId sponsor, ActorId newcomer,
        Weight weight, Option<DomainScope> scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Invite(sponsor, newcomer, weight, scope, cause, convId, signer);

    public sealed record ExileResult(Event Exclusion, Event Sever, Event Sanction);

    public ExileResult Exile(ActorId moderator, EdgeId edge, string reason, EventId cause, ConversationId convId, ISigner signer)
    {
        var exclusion = _g.Emit(moderator, "exile: " + reason, convId, new List<EventId> { cause }, signer);
        var sever = _g.Sever(moderator, edge, exclusion.Id, convId, signer);
        var sanction = _g.Annotate(moderator, sever.Id, "sanction", reason, convId, signer);
        return new ExileResult(exclusion, sever, sanction);
    }

    // Named Functions

    public sealed record PollResult(Event Proposal, List<Event> Votes);

    public PollResult Poll(ActorId proposer, string question, List<ActorId> voters, DomainScope scope, EventId cause, ConversationId convId, ISigner signer)
    {
        var proposal = _g.Emit(proposer, "poll: " + question, convId, new List<EventId> { cause }, signer);
        var votes = new List<Event>();
        for (int i = 0; i < voters.Count; i++)
            votes.Add(_g.Consent(voters[i], proposer, "vote: " + question, scope, proposal.Id, convId, signer));
        return new PollResult(proposal, votes);
    }

    public sealed record FederationResult(Event Agreement, Event Delegation);

    public FederationResult Federation(ActorId communityA, ActorId communityB, string terms,
        DomainScope scope, Weight weight, EventId cause, ConversationId convId, ISigner signer)
    {
        var agreement = _g.Consent(communityA, communityB, "federation: " + terms, scope, cause, convId, signer);
        var delegation = _g.Delegate(communityA, communityB, scope, weight, agreement.Id, convId, signer);
        return new FederationResult(agreement, delegation);
    }

    public sealed record SchismResult(Event ConflictingNorm, ExileResult Exile, Event NewCommunity);

    public SchismResult Schism(ActorId faction, ActorId moderator, string conflictingNorm,
        DomainScope scope, EdgeId edge, string reason, EventId cause, ConversationId convId, ISigner signer)
    {
        var norm = _g.Emit(faction, "conflicting-norm: " + conflictingNorm, convId, new List<EventId> { cause }, signer);
        var exile = Exile(moderator, edge, reason, norm.Id, convId, signer);
        var community = _g.Emit(faction, "new-community: split over " + conflictingNorm, convId, new List<EventId> { exile.Sanction.Id }, signer);
        return new SchismResult(norm, exile, community);
    }
}

// ── JusticeGrammar (Layer 4 — Legal) ───────────────────────────────────

/// <summary>Layer 4 composition operations for transparent dispute resolution.</summary>
public sealed class JusticeGrammar
{
    private readonly Grammar _g;
    public JusticeGrammar(Grammar g) => _g = g;

    public Event Legislate(ActorId source, string rule, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "legislate: " + rule, convId, causes, signer);

    public Event Amend(ActorId source, string amendment, EventId rule, ConversationId convId, ISigner signer)
        => _g.Derive(source, "amend: " + amendment, rule, convId, signer);

    public Event Repeal(ActorId source, EventId rule, string reason, ConversationId convId, ISigner signer)
        => _g.Retract(source, rule, reason, convId, signer);

    public Event File(ActorId source, string complaint, EventId target, ConversationId convId, ISigner signer)
    {
        var (_, flag) = _g.Challenge(source, "file: " + complaint, target, convId, signer);
        return flag;
    }

    public Event Submit(ActorId source, EventId target, string evidence, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "evidence", evidence, convId, signer);

    public Event Argue(ActorId source, string argument, EventId target, ConversationId convId, ISigner signer)
        => _g.Respond(source, "argue: " + argument, target, convId, signer);

    public Event Judge(ActorId source, string ruling, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "judge: " + ruling, convId, causes, signer);

    public Event Appeal(ActorId source, string grounds, EventId ruling, ConversationId convId, ISigner signer)
    {
        var (_, flag) = _g.Challenge(source, "appeal: " + grounds, ruling, convId, signer);
        return flag;
    }

    public Event Enforce(ActorId source, ActorId executor, DomainScope scope, Weight weight, EventId cause, ConversationId convId, ISigner signer)
        => _g.Delegate(source, executor, scope, weight, cause, convId, signer);

    public Event Audit(ActorId source, EventId target, string findings, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "audit", findings, convId, signer);

    public Event Pardon(ActorId authority, ActorId pardoned, string terms, DomainScope scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Consent(authority, pardoned, "pardon: " + terms, scope, cause, convId, signer);

    public Event Reform(ActorId source, string proposal, EventId precedent, ConversationId convId, ISigner signer)
        => _g.Derive(source, "reform: " + proposal, precedent, convId, signer);

    // Named Functions

    public sealed record TrialResult(Event Filing, List<Event> Submissions, List<Event> Arguments, Event Ruling);

    public TrialResult Trial(ActorId plaintiff, ActorId defendant, ActorId judge,
        string complaint, string plaintiffEvidence, string defendantEvidence,
        string plaintiffArgument, string defendantArgument, string ruling,
        EventId target, ConversationId convId, ISigner signer)
    {
        var filing = File(plaintiff, complaint, target, convId, signer);
        var sub1 = Submit(plaintiff, filing.Id, plaintiffEvidence, convId, signer);
        var sub2 = Submit(defendant, filing.Id, defendantEvidence, convId, signer);
        var arg1 = Argue(plaintiff, plaintiffArgument, sub1.Id, convId, signer);
        var arg2 = Argue(defendant, defendantArgument, sub2.Id, convId, signer);
        var verdict = Judge(judge, ruling, new List<EventId> { arg1.Id, arg2.Id }, convId, signer);
        return new TrialResult(filing, new List<Event> { sub1, sub2 }, new List<Event> { arg1, arg2 }, verdict);
    }

    public sealed record ConstitutionalAmendmentResult(Event Reform, Event Legislation, Event RightsCheck);

    public ConstitutionalAmendmentResult ConstitutionalAmendment(ActorId proposer,
        string proposal, string legislation, string rightsAssessment,
        EventId precedent, ConversationId convId, ISigner signer)
    {
        var reform = Reform(proposer, proposal, precedent, convId, signer);
        var legislate = Legislate(proposer, legislation, new List<EventId> { reform.Id }, convId, signer);
        var rights = Audit(proposer, legislate.Id, rightsAssessment, convId, signer);
        return new ConstitutionalAmendmentResult(reform, legislate, rights);
    }

    public sealed record InjunctionResult(Event Filing, Event Ruling, Event Enforcement);

    public InjunctionResult Injunction(ActorId petitioner, ActorId judge, ActorId executor,
        string complaint, string ruling, DomainScope scope, Weight weight,
        EventId target, ConversationId convId, ISigner signer)
    {
        var filing = File(petitioner, complaint, target, convId, signer);
        var verdict = Judge(judge, "emergency: " + ruling, new List<EventId> { filing.Id }, convId, signer);
        var enforce = Enforce(judge, executor, scope, weight, verdict.Id, convId, signer);
        return new InjunctionResult(filing, verdict, enforce);
    }

    public sealed record PleaResult(Event Filing, Event Acceptance, Event Enforcement);

    public PleaResult Plea(ActorId defendant, ActorId prosecutor, ActorId executor,
        string complaint, string deal, DomainScope scope, Weight weight,
        EventId target, ConversationId convId, ISigner signer)
    {
        var filing = File(prosecutor, complaint, target, convId, signer);
        var acceptance = Pardon(prosecutor, defendant, deal, scope, filing.Id, convId, signer);
        var enforce = Enforce(prosecutor, executor, scope, weight, acceptance.Id, convId, signer);
        return new PleaResult(filing, acceptance, enforce);
    }

    public sealed record ClassActionResult(List<Event> Filings, Event Merged, TrialResult Trial);

    public ClassActionResult ClassAction(List<ActorId> plaintiffs, ActorId defendant, ActorId judge,
        List<string> complaints, string evidence, string argument,
        string defenseEvidence, string defenseArgument, string ruling,
        EventId target, ConversationId convId, ISigner signer)
    {
        if (plaintiffs.Count != complaints.Count)
            throw new ArgumentException("ClassAction: plaintiffs and complaints must have equal length");

        var filings = new List<Event>();
        var filingIds = new List<EventId>();
        for (int i = 0; i < plaintiffs.Count; i++)
        {
            var filing = File(plaintiffs[i], complaints[i], target, convId, signer);
            filings.Add(filing);
            filingIds.Add(filing.Id);
        }

        var merged = _g.Merge(plaintiffs[0], "class-action: merged complaints", filingIds, convId, signer);
        var trial = Trial(plaintiffs[0], defendant, judge, "class-action",
            evidence, defenseEvidence, argument, defenseArgument, ruling,
            merged.Id, convId, signer);
        return new ClassActionResult(filings, merged, trial);
    }

    public sealed record RecallResult(Event Audit, Event Filing, Event Consent, Event Revocation);

    public RecallResult Recall(ActorId auditor, ActorId community, ActorId official,
        string findings, string complaint, DomainScope scope,
        EventId target, ConversationId convId, ISigner signer)
    {
        var audit = this.Audit(auditor, target, findings, convId, signer);
        var filing = File(auditor, complaint, audit.Id, convId, signer);
        var consent = _g.Consent(community, official, "recall: " + complaint, scope, filing.Id, convId, signer);
        var revocation = _g.Emit(community, "role-revoked: " + complaint, convId, new List<EventId> { consent.Id }, signer);
        return new RecallResult(audit, filing, consent, revocation);
    }
}

// ── BuildGrammar (Layer 5 — Technology) ────────────────────────────────

/// <summary>Layer 5 composition operations for development, CI/CD, and artefact lifecycle.</summary>
public sealed class BuildGrammar
{
    private readonly Grammar _g;
    public BuildGrammar(Grammar g) => _g = g;

    public Event Build(ActorId source, string artefact, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "build: " + artefact, convId, causes, signer);

    public Event Version(ActorId source, string version, EventId previous, ConversationId convId, ISigner signer)
        => _g.Derive(source, "version: " + version, previous, convId, signer);

    public Event Ship(ActorId source, string deployment, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "ship: " + deployment, convId, causes, signer);

    public Event Sunset(ActorId source, EventId target, string migration, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "deprecated", migration, convId, signer);

    public Event Define(ActorId source, string workflow, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "define: " + workflow, convId, causes, signer);

    public Event Automate(ActorId source, string automation, EventId workflow, ConversationId convId, ISigner signer)
        => _g.Derive(source, "automate: " + automation, workflow, convId, signer);

    public Event Test(ActorId source, string results, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "test: " + results, convId, causes, signer);

    public Event Review(ActorId source, string assessment, EventId target, ConversationId convId, ISigner signer)
        => _g.Respond(source, "review: " + assessment, target, convId, signer);

    public Event Measure(ActorId source, EventId target, string scores, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "quality", scores, convId, signer);

    public Event Feedback(ActorId source, string feedback, EventId target, ConversationId convId, ISigner signer)
        => _g.Respond(source, "feedback: " + feedback, target, convId, signer);

    public Event Iterate(ActorId source, string improvement, EventId previous, ConversationId convId, ISigner signer)
        => _g.Derive(source, "iterate: " + improvement, previous, convId, signer);

    public Event Innovate(ActorId source, string innovation, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "innovate: " + innovation, convId, causes, signer);

    // Named Functions

    public sealed record SpikeResult(Event Build, Event Test, Event Feedback, Event Decision);

    public SpikeResult Spike(ActorId source, string experiment, string testResults,
        string feedback, string decision, List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var build = Build(source, "spike: " + experiment, causes, convId, signer);
        var test = Test(source, testResults, new List<EventId> { build.Id }, convId, signer);
        var fb = Feedback(source, feedback, test.Id, convId, signer);
        var dec = _g.Emit(source, "spike-decision: " + decision, convId, new List<EventId> { fb.Id }, signer);
        return new SpikeResult(build, test, fb, dec);
    }

    public sealed record MigrationResult(Event Sunset, Event Version, Event Ship, Event Test);

    public MigrationResult Migration(ActorId source, EventId deprecatedTarget, string migrationPath,
        string newVersion, string deployment, string testResults, ConversationId convId, ISigner signer)
    {
        var sunset = Sunset(source, deprecatedTarget, migrationPath, convId, signer);
        var version = Version(source, newVersion, sunset.Id, convId, signer);
        var ship = Ship(source, deployment, new List<EventId> { version.Id }, convId, signer);
        var test = Test(source, testResults, new List<EventId> { ship.Id }, convId, signer);
        return new MigrationResult(sunset, version, ship, test);
    }

    public sealed record TechDebtResult(Event Measure, Event DebtMark, Event Iteration);

    public TechDebtResult TechDebt(ActorId source, EventId target, string scores,
        string debtDescription, string plan, ConversationId convId, ISigner signer)
    {
        var measure = Measure(source, target, scores, convId, signer);
        var debt = _g.Annotate(source, measure.Id, "tech_debt", debtDescription, convId, signer);
        var iterate = Iterate(source, plan, debt.Id, convId, signer);
        return new TechDebtResult(measure, debt, iterate);
    }

    public sealed record PipelineResult(Event Definition, Event TestResult, Event Metrics, Event Deployment);

    public PipelineResult Pipeline(ActorId source, string workflow, string testResults,
        string metrics, string deployment, List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var def = Define(source, workflow, causes, convId, signer);
        var test = Test(source, testResults, new List<EventId> { def.Id }, convId, signer);
        var measure = Measure(source, test.Id, metrics, convId, signer);
        var ship = Ship(source, deployment, new List<EventId> { measure.Id }, convId, signer);
        return new PipelineResult(def, test, measure, ship);
    }

    public sealed record PostMortemResult(List<Event> Feedback, Event Analysis, Event Improvements);

    public PostMortemResult PostMortem(ActorId lead, List<ActorId> contributors, List<string> feedbacks,
        string analysis, string improvements, EventId incident, ConversationId convId, ISigner signer)
    {
        if (contributors.Count != feedbacks.Count)
            throw new ArgumentException("PostMortem: contributors and feedbacks must have equal length");

        var fbList = new List<Event>();
        var fbIds = new List<EventId>();
        for (int i = 0; i < contributors.Count; i++)
        {
            var fb = Feedback(contributors[i], feedbacks[i], incident, convId, signer);
            fbList.Add(fb);
            fbIds.Add(fb.Id);
        }

        var analysisEv = Measure(lead, fbIds[^1], "post-mortem: " + analysis, convId, signer);
        var improve = Define(lead, improvements, new List<EventId> { analysisEv.Id }, convId, signer);
        return new PostMortemResult(fbList, analysisEv, improve);
    }
}

// ── KnowledgeGrammar (Layer 6 — Information) ───────────────────────────

/// <summary>Layer 6 composition operations for verified, provenanced knowledge.</summary>
public sealed class KnowledgeGrammar
{
    private readonly Grammar _g;
    public KnowledgeGrammar(Grammar g) => _g = g;

    public Event Claim(ActorId source, string claim, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "claim: " + claim, convId, causes, signer);

    public Event Categorize(ActorId source, EventId target, string taxonomy, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "classification", taxonomy, convId, signer);

    public Event Abstract(ActorId source, string generalization, List<EventId> instances, ConversationId convId, ISigner signer)
    {
        if (instances.Count < 2)
            throw new ArgumentException("Abstract: requires at least two instances");
        return _g.Merge(source, "abstract: " + generalization, instances, convId, signer);
    }

    public Event Encode(ActorId source, string encoding, EventId original, ConversationId convId, ISigner signer)
        => _g.Derive(source, "encode: " + encoding, original, convId, signer);

    public Event Infer(ActorId source, string conclusion, EventId premise, ConversationId convId, ISigner signer)
        => _g.Derive(source, "infer: " + conclusion, premise, convId, signer);

    public Event Remember(ActorId source, string knowledge, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "remember: " + knowledge, convId, causes, signer);

    public Event Challenge(ActorId source, string counterEvidence, EventId claim, ConversationId convId, ISigner signer)
    {
        var (_, flag) = _g.Challenge(source, "challenge: " + counterEvidence, claim, convId, signer);
        return flag;
    }

    public Event DetectBias(ActorId source, EventId target, string bias, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "bias", bias, convId, signer);

    public Event Correct(ActorId source, string correction, EventId original, ConversationId convId, ISigner signer)
        => _g.Derive(source, "correct: " + correction, original, convId, signer);

    public Event Trace(ActorId source, EventId target, string provenance, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "provenance", provenance, convId, signer);

    public Event Recall(ActorId source, string query, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "recall: " + query, convId, causes, signer);

    public Event Learn(ActorId source, string learning, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "learn: " + learning, convId, causes, signer);

    // Named Functions

    public Event Retract(ActorId source, EventId claim, string reason, ConversationId convId, ISigner signer)
        => _g.Retract(source, claim, reason, convId, signer);

    public sealed record FactCheckResult(Event Provenance, Event BiasCheck, Event Verdict);

    public FactCheckResult FactCheck(ActorId checker, EventId claim, string provenance,
        string biasAnalysis, string verdict, ConversationId convId, ISigner signer)
    {
        var trace = Trace(checker, claim, provenance, convId, signer);
        var bias = DetectBias(checker, claim, biasAnalysis, convId, signer);
        var verdictEv = _g.Merge(checker, "fact-check: " + verdict,
            new List<EventId> { trace.Id, bias.Id }, convId, signer);
        return new FactCheckResult(trace, bias, verdictEv);
    }

    public sealed record VerifyResult(Event Claim, Event Provenance, Event Corroboration);

    public VerifyResult Verify(ActorId source, string claim, string provenance, string corroboration,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var claimEv = Claim(source, claim, causes, convId, signer);
        var trace = Trace(source, claimEv.Id, provenance, convId, signer);
        var corroborate = Claim(source, "corroborate: " + corroboration, new List<EventId> { trace.Id }, convId, signer);
        return new VerifyResult(claimEv, trace, corroborate);
    }

    public sealed record SurveyResult(List<Event> Recalls, Event Abstraction, Event Synthesis);

    public SurveyResult Survey(ActorId source, List<string> queries, string generalization, string synthesis,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        if (queries.Count < 2)
            throw new ArgumentException("Survey: requires at least two queries");

        var recalls = new List<Event>();
        var recallIds = new List<EventId>();
        for (int i = 0; i < queries.Count; i++)
        {
            var recall = Recall(source, queries[i], causes, convId, signer);
            recalls.Add(recall);
            recallIds.Add(recall.Id);
        }

        var abs = Abstract(source, generalization, recallIds, convId, signer);
        var synthesisClaim = Claim(source, "synthesis: " + synthesis, new List<EventId> { abs.Id }, convId, signer);
        return new SurveyResult(recalls, abs, synthesisClaim);
    }

    public sealed record KnowledgeBaseResult(List<Event> Claims, List<Event> Categories, Event Memory);

    public KnowledgeBaseResult KnowledgeBase(ActorId source, List<string> claims, List<string> taxonomies,
        string memoryLabel, List<EventId> causes, ConversationId convId, ISigner signer)
    {
        if (claims.Count != taxonomies.Count)
            throw new ArgumentException("KnowledgeBase: claims and taxonomies must have equal length");

        var claimEvs = new List<Event>();
        var catEvs = new List<Event>();
        var claimIds = new List<EventId>();
        for (int i = 0; i < claims.Count; i++)
        {
            var claimEv = Claim(source, claims[i], causes, convId, signer);
            claimEvs.Add(claimEv);

            var cat = Categorize(source, claimEv.Id, taxonomies[i], convId, signer);
            catEvs.Add(cat);
            claimIds.Add(cat.Id);
        }

        var memory = Remember(source, memoryLabel, claimIds, convId, signer);
        return new KnowledgeBaseResult(claimEvs, catEvs, memory);
    }

    public sealed record TransferResult(Event Recall, Event Encode, Event Learn);

    public TransferResult Transfer(ActorId source, string query, string encoding, string learning,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var recall = Recall(source, query, causes, convId, signer);
        var encode = Encode(source, encoding, recall.Id, convId, signer);
        var learn = Learn(source, learning, new List<EventId> { encode.Id }, convId, signer);
        return new TransferResult(recall, encode, learn);
    }
}

// ── AlignmentGrammar (Layer 7 — Ethics) ────────────────────────────────

/// <summary>Layer 7 composition operations for AI accountability.</summary>
public sealed class AlignmentGrammar
{
    private readonly Grammar _g;
    public AlignmentGrammar(Grammar g) => _g = g;

    public Event Constrain(ActorId source, EventId target, string constraint, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "constraint", constraint, convId, signer);

    public Event DetectHarm(ActorId source, string harm, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "harm: " + harm, convId, causes, signer);

    public Event AssessFairness(ActorId source, EventId target, string assessment, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "fairness", assessment, convId, signer);

    public Event FlagDilemma(ActorId source, string dilemma, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "dilemma: " + dilemma, convId, causes, signer);

    public Event Weigh(ActorId source, string weighing, EventId decision, ConversationId convId, ISigner signer)
        => _g.Derive(source, "weigh: " + weighing, decision, convId, signer);

    public Event Explain(ActorId source, string explanation, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "explain: " + explanation, convId, causes, signer);

    public Event Assign(ActorId source, EventId target, string responsibility, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "responsibility", responsibility, convId, signer);

    public Event Repair(ActorId source, ActorId affected, string redress, DomainScope scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Consent(source, affected, "repair: " + redress, scope, cause, convId, signer);

    public Event Care(ActorId source, string care, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "care: " + care, convId, causes, signer);

    public Event Grow(ActorId source, string growth, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "grow: " + growth, convId, causes, signer);

    // Named Functions

    public sealed record EthicsAuditResult(Event Fairness, Event HarmScan, Event Report);

    public EthicsAuditResult EthicsAudit(ActorId auditor, EventId target,
        string fairnessAssessment, string harmScan, string summary, ConversationId convId, ISigner signer)
    {
        var fairness = AssessFairness(auditor, target, fairnessAssessment, convId, signer);
        var harm = DetectHarm(auditor, harmScan, new List<EventId> { fairness.Id }, convId, signer);
        var report = Explain(auditor, summary, new List<EventId> { fairness.Id, harm.Id }, convId, signer);
        return new EthicsAuditResult(fairness, harm, report);
    }

    public sealed record RestorativeJusticeResult(Event HarmDetection, Event Responsibility, Event Redress, Event Growth);

    public RestorativeJusticeResult RestorativeJustice(ActorId auditor, ActorId agent, ActorId affected,
        string harm, string responsibility, string redress, string growth, DomainScope scope,
        EventId cause, ConversationId convId, ISigner signer)
    {
        var harmEv = DetectHarm(auditor, harm, new List<EventId> { cause }, convId, signer);
        var assign = Assign(auditor, harmEv.Id, responsibility, convId, signer);
        var repair = Repair(auditor, affected, redress, scope, assign.Id, convId, signer);
        var growEv = Grow(agent, growth, new List<EventId> { repair.Id }, convId, signer);
        return new RestorativeJusticeResult(harmEv, assign, repair, growEv);
    }

    public sealed record GuardrailResult(Event Constraint, Event Dilemma, Event Escalation);

    public GuardrailResult Guardrail(ActorId source, EventId target,
        string constraint, string dilemma, string escalation, ConversationId convId, ISigner signer)
    {
        var constrain = Constrain(source, target, constraint, convId, signer);
        var dilemmaEv = FlagDilemma(source, dilemma, new List<EventId> { constrain.Id }, convId, signer);
        var escalate = _g.Emit(source, "escalate: " + escalation, convId, new List<EventId> { dilemmaEv.Id }, signer);
        return new GuardrailResult(constrain, dilemmaEv, escalate);
    }

    public sealed record ImpactAssessmentResult(Event Weighing, Event Fairness, Event Explanation);

    public ImpactAssessmentResult ImpactAssessment(ActorId source, EventId decision,
        string weighing, string fairness, string explanation, ConversationId convId, ISigner signer)
    {
        var weigh = Weigh(source, weighing, decision, convId, signer);
        var fair = AssessFairness(source, weigh.Id, fairness, convId, signer);
        var explain = Explain(source, explanation, new List<EventId> { weigh.Id, fair.Id }, convId, signer);
        return new ImpactAssessmentResult(weigh, fair, explain);
    }

    public sealed record WhistleblowResult(Event Harm, Event Explanation, Event Escalation);

    public WhistleblowResult Whistleblow(ActorId source, string harm, string explanation, string escalation,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var harmEv = DetectHarm(source, harm, causes, convId, signer);
        var explain = Explain(source, explanation, new List<EventId> { harmEv.Id }, convId, signer);
        var escalate = _g.Emit(source, "escalate-external: " + escalation, convId, new List<EventId> { explain.Id }, signer);
        return new WhistleblowResult(harmEv, explain, escalate);
    }
}

// ── IdentityGrammar (Layer 8 — Identity) ───────────────────────────────

/// <summary>Layer 8 composition operations for self-sovereign identity.</summary>
public sealed class IdentityGrammar
{
    private readonly Grammar _g;
    public IdentityGrammar(Grammar g) => _g = g;

    public Event Introspect(ActorId source, string selfModel, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "introspect: " + selfModel, convId, causes, signer);

    public Event Narrate(ActorId source, string narrative, EventId basis, ConversationId convId, ISigner signer)
        => _g.Derive(source, "narrate: " + narrative, basis, convId, signer);

    public Event Align(ActorId source, EventId target, string alignment, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "alignment", alignment, convId, signer);

    public Event Bound(ActorId source, string boundary, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "bound: " + boundary, convId, causes, signer);

    public Event Aspire(ActorId source, string aspiration, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "aspire: " + aspiration, convId, causes, signer);

    public Event Transform(ActorId source, string transformation, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "transform: " + transformation, convId, causes, signer);

    public Event Disclose(ActorId source, ActorId target, Option<DomainScope> scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Channel(source, target, scope, cause, convId, signer);

    public Event Recognize(ActorId source, string recognition, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "recognize: " + recognition, convId, causes, signer);

    public Event Distinguish(ActorId source, EventId target, string uniqueness, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "uniqueness", uniqueness, convId, signer);

    public Event Memorialize(ActorId source, string memorial, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "memorialize: " + memorial, convId, causes, signer);

    // Named Functions

    public sealed record IdentityAuditResult(Event SelfModel, Event Alignment, Event Narrative);

    public IdentityAuditResult IdentityAudit(ActorId source, string selfModel, string alignment, string narrative,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var intro = Introspect(source, selfModel, causes, convId, signer);
        var align = Align(source, intro.Id, alignment, convId, signer);
        var narr = Narrate(source, narrative, align.Id, convId, signer);
        return new IdentityAuditResult(intro, align, narr);
    }

    public sealed record RetirementResult(Event Memorial, Event Transfer, Event Archive);

    public RetirementResult Retirement(ActorId system, ActorId departing, ActorId successor,
        string memorial, DomainScope scope, Weight weight,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var mem = Memorialize(system, $"retirement of {departing.Value}: {memorial}", causes, convId, signer);
        var transfer = _g.Delegate(system, successor, scope, weight, mem.Id, convId, signer);
        var archive = _g.Emit(system, $"archive: contributions of {departing.Value}", convId, new List<EventId> { transfer.Id }, signer);
        return new RetirementResult(mem, transfer, archive);
    }

    public sealed record CredentialResult(Event Introspection, Event Disclosure);

    public CredentialResult Credential(ActorId source, ActorId verifier, string selfModel,
        Option<DomainScope> scope, List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var intro = Introspect(source, selfModel, causes, convId, signer);
        var disclose = Disclose(source, verifier, scope, intro.Id, convId, signer);
        return new CredentialResult(intro, disclose);
    }

    public sealed record ReinventionResult(Event Transformation, Event Narrative, Event Aspiration);

    public ReinventionResult Reinvention(ActorId source, string transformation, string narrative, string aspiration,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var transform = Transform(source, transformation, causes, convId, signer);
        var narr = Narrate(source, narrative, transform.Id, convId, signer);
        var aspire = Aspire(source, aspiration, new List<EventId> { narr.Id }, convId, signer);
        return new ReinventionResult(transform, narr, aspire);
    }

    public sealed record IntroductionResult(Event Disclosure, Event Narrative);

    public IntroductionResult Introduction(ActorId source, ActorId target,
        Option<DomainScope> scope, string narrative, EventId cause, ConversationId convId, ISigner signer)
    {
        var disclose = Disclose(source, target, scope, cause, convId, signer);
        var narr = Narrate(source, narrative, disclose.Id, convId, signer);
        return new IntroductionResult(disclose, narr);
    }
}

// ── BondGrammar (Layer 9 — Relationship) ───────────────────────────────

/// <summary>Layer 9 composition operations for deep relational bonds.</summary>
public sealed class BondGrammar
{
    private readonly Grammar _g;
    public BondGrammar(Grammar g) => _g = g;

    public (Event Sub1, Event Sub2) Connect(ActorId source, ActorId target,
        Option<DomainScope> scope, EventId cause, ConversationId convId, ISigner signer)
    {
        var sub1 = _g.Subscribe(source, target, scope, cause, convId, signer);
        var sub2 = _g.Subscribe(target, source, scope, sub1.Id, convId, signer);
        return (sub1, sub2);
    }

    public Event Balance(ActorId source, EventId target, string assessment, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "reciprocity", assessment, convId, signer);

    public Event Deepen(ActorId source, ActorId other, string basis, DomainScope scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Consent(source, other, "deepen: " + basis, scope, cause, convId, signer);

    public Event Open(ActorId source, ActorId target, Option<DomainScope> scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Channel(source, target, scope, cause, convId, signer);

    public Event Attune(ActorId source, string understanding, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "attune: " + understanding, convId, causes, signer);

    public Event FeelWith(ActorId source, string empathy, EventId target, ConversationId convId, ISigner signer)
        => _g.Respond(source, "empathy: " + empathy, target, convId, signer);

    public Event Break(ActorId source, string rupture, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "rupture: " + rupture, convId, causes, signer);

    public Event Apologize(ActorId source, string apology, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "apology: " + apology, convId, causes, signer);

    public Event Reconcile(ActorId source, ActorId other, string progress, DomainScope scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Consent(source, other, "reconcile: " + progress, scope, cause, convId, signer);

    public Event Mourn(ActorId source, string loss, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "mourn: " + loss, convId, causes, signer);

    // Named Functions

    public sealed record BetrayalRepairResult(Event Rupture, Event Apology, Event Reconciliation, Event Deepened);

    public BetrayalRepairResult BetrayalRepair(ActorId injured, ActorId offender,
        string rupture, string apology, string reconciliation, string newBasis,
        DomainScope scope, List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var ruptureEv = Break(injured, rupture, causes, convId, signer);
        var apologyEv = Apologize(offender, apology, new List<EventId> { ruptureEv.Id }, convId, signer);
        var reconcileEv = Reconcile(injured, offender, reconciliation, scope, apologyEv.Id, convId, signer);
        var deepen = Deepen(injured, offender, newBasis, scope, reconcileEv.Id, convId, signer);
        return new BetrayalRepairResult(ruptureEv, apologyEv, reconcileEv, deepen);
    }

    public sealed record CheckInResult(Event Balance, Event Attunement, Event Empathy);

    public CheckInResult CheckIn(ActorId source, EventId balanceTarget, string assessment,
        string attunement, string empathy, ConversationId convId, ISigner signer)
    {
        var bal = Balance(source, balanceTarget, assessment, convId, signer);
        var att = Attune(source, attunement, new List<EventId> { bal.Id }, convId, signer);
        var emp = FeelWith(source, empathy, att.Id, convId, signer);
        return new CheckInResult(bal, att, emp);
    }

    public sealed record BondMentorshipResult(Event Connection, Event Deepening, Event Attunement, Event Teaching);

    public BondMentorshipResult Mentorship(ActorId mentor, ActorId mentee,
        string basis, string understanding, DomainScope scope, Option<DomainScope> teachingScope,
        EventId cause, ConversationId convId, ISigner signer)
    {
        var connect = _g.Subscribe(mentee, mentor, teachingScope, cause, convId, signer);
        var deepen = Deepen(mentor, mentee, basis, scope, connect.Id, convId, signer);
        var attune = Attune(mentor, understanding, new List<EventId> { deepen.Id }, convId, signer);
        var teach = _g.Channel(mentor, mentee, teachingScope, attune.Id, convId, signer);
        return new BondMentorshipResult(connect, deepen, attune, teach);
    }

    public sealed record BondFarewellResult(Event Mourning, Event Memorial, Event Gratitude);

    public BondFarewellResult Farewell(ActorId source, ActorId departing,
        string loss, string memorial, Weight gratitudeWeight,
        Option<DomainScope> scope, List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var mourn = Mourn(source, loss, causes, convId, signer);
        var mem = _g.Emit(source, "memorialize: " + memorial, convId, new List<EventId> { mourn.Id }, signer);
        var gratitude = _g.Endorse(source, mem.Id, departing, gratitudeWeight, scope, convId, signer);
        return new BondFarewellResult(mourn, mem, gratitude);
    }

    public Event Forgive(ActorId source, EventId severEvent, ActorId target,
        Option<DomainScope> scope, ConversationId convId, ISigner signer)
        => _g.Forgive(source, severEvent, target, scope, convId, signer);
}

// ── BelongingGrammar (Layer 10 — Community) ────────────────────────────

/// <summary>Layer 10 composition operations for communities with shared resources.</summary>
public sealed class BelongingGrammar
{
    private readonly Grammar _g;
    public BelongingGrammar(Grammar g) => _g = g;

    public Event Settle(ActorId source, ActorId community, Option<DomainScope> scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Subscribe(source, community, scope, cause, convId, signer);

    public Event Contribute(ActorId source, string contribution, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "contribute: " + contribution, convId, causes, signer);

    public Event Include(ActorId source, string action, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "include: " + action, convId, causes, signer);

    public Event Practice(ActorId source, string tradition, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "practice: " + tradition, convId, causes, signer);

    public Event Steward(ActorId source, ActorId steward, DomainScope scope, Weight weight, EventId cause, ConversationId convId, ISigner signer)
        => _g.Delegate(source, steward, scope, weight, cause, convId, signer);

    public Event Sustain(ActorId source, string assessment, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "sustain: " + assessment, convId, causes, signer);

    public Event PassOn(ActorId from, ActorId to, DomainScope scope, string description, EventId cause, ConversationId convId, ISigner signer)
        => _g.Consent(from, to, "pass-on: " + description, scope, cause, convId, signer);

    public Event Celebrate(ActorId source, string celebration, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "celebrate: " + celebration, convId, causes, signer);

    public Event Tell(ActorId source, string story, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "tell: " + story, convId, causes, signer);

    public Event Gift(ActorId source, string gift, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "gift: " + gift, convId, causes, signer);

    // Named Functions

    public sealed record FestivalResult(Event Celebration, Event Practice, Event Story, Event Gift);

    public FestivalResult Festival(ActorId source, string celebration, string tradition, string story, string gift,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var celebrate = Celebrate(source, celebration, causes, convId, signer);
        var practice = Practice(source, tradition, new List<EventId> { celebrate.Id }, convId, signer);
        var tell = Tell(source, story, new List<EventId> { practice.Id }, convId, signer);
        var giftEv = Gift(source, gift, new List<EventId> { tell.Id }, convId, signer);
        return new FestivalResult(celebrate, practice, tell, giftEv);
    }

    public sealed record CommonsGovernanceResult(Event Stewardship, Event Assessment, Event Legislation, Event Audit);

    public CommonsGovernanceResult CommonsGovernance(ActorId source, ActorId steward,
        DomainScope scope, Weight weight, string assessment, string rule, string findings,
        EventId cause, ConversationId convId, ISigner signer)
    {
        var stewardship = Steward(source, steward, scope, weight, cause, convId, signer);
        var sustain = Sustain(steward, assessment, new List<EventId> { stewardship.Id }, convId, signer);
        var legislate = _g.Emit(source, "legislate: " + rule, convId, new List<EventId> { sustain.Id }, signer);
        var audit = _g.Annotate(steward, legislate.Id, "audit", findings, convId, signer);
        return new CommonsGovernanceResult(stewardship, sustain, legislate, audit);
    }

    public sealed record RenewalResult(Event Assessment, Event Practice, Event Story);

    public RenewalResult Renewal(ActorId source, string assessment, string evolvedPractice, string newStory,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var sustain = Sustain(source, assessment, causes, convId, signer);
        var practice = Practice(source, evolvedPractice, new List<EventId> { sustain.Id }, convId, signer);
        var story = Tell(source, newStory, new List<EventId> { practice.Id }, convId, signer);
        return new RenewalResult(sustain, practice, story);
    }

    public sealed record OnboardResult(Event Inclusion, Event Settlement, Event FirstPractice, Event Contribution);

    public OnboardResult Onboard(ActorId sponsor, ActorId newcomer, ActorId community,
        Option<DomainScope> scope, string inclusionAction, string tradition, string firstContribution,
        EventId cause, ConversationId convId, ISigner signer)
    {
        var inclusion = Include(sponsor, inclusionAction, new List<EventId> { cause }, convId, signer);
        var settle = Settle(newcomer, community, scope, inclusion.Id, convId, signer);
        var practice = Practice(newcomer, tradition, new List<EventId> { settle.Id }, convId, signer);
        var contrib = Contribute(newcomer, firstContribution, new List<EventId> { practice.Id }, convId, signer);
        return new OnboardResult(inclusion, settle, practice, contrib);
    }

    public sealed record SuccessionResult(Event Assessment, Event Transfer, Event Celebration, Event Story);

    public SuccessionResult Succession(ActorId outgoing, ActorId incoming,
        string assessment, DomainScope scope, string celebration, string story,
        EventId cause, ConversationId convId, ISigner signer)
    {
        var sustain = Sustain(outgoing, assessment, new List<EventId> { cause }, convId, signer);
        var transfer = PassOn(outgoing, incoming, scope, "stewardship transfer", sustain.Id, convId, signer);
        var celebrate = Celebrate(outgoing, celebration, new List<EventId> { transfer.Id }, convId, signer);
        var tell = Tell(outgoing, story, new List<EventId> { celebrate.Id }, convId, signer);
        return new SuccessionResult(sustain, transfer, celebrate, tell);
    }
}

// ── MeaningGrammar (Layer 11 — Culture) ────────────────────────────────

/// <summary>Layer 11 composition operations for cross-cultural communication.</summary>
public sealed class MeaningGrammar
{
    private readonly Grammar _g;
    public MeaningGrammar(Grammar g) => _g = g;

    public Event Examine(ActorId source, string examination, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "examine: " + examination, convId, causes, signer);

    public Event Reframe(ActorId source, string reframing, EventId original, ConversationId convId, ISigner signer)
        => _g.Derive(source, "reframe: " + reframing, original, convId, signer);

    public Event Question(ActorId source, string question, EventId target, ConversationId convId, ISigner signer)
    {
        var (_, flag) = _g.Challenge(source, "question: " + question, target, convId, signer);
        return flag;
    }

    public Event Distill(ActorId source, string wisdom, EventId experience, ConversationId convId, ISigner signer)
        => _g.Derive(source, "distill: " + wisdom, experience, convId, signer);

    public Event Beautify(ActorId source, string beauty, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "beautify: " + beauty, convId, causes, signer);

    public Event Liken(ActorId source, string metaphor, EventId subject, ConversationId convId, ISigner signer)
        => _g.Derive(source, "liken: " + metaphor, subject, convId, signer);

    public Event Lighten(ActorId source, string humour, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "lighten: " + humour, convId, causes, signer);

    public Event Teach(ActorId source, ActorId student, Option<DomainScope> scope, EventId cause, ConversationId convId, ISigner signer)
        => _g.Channel(source, student, scope, cause, convId, signer);

    public Event Translate(ActorId source, string translation, EventId original, ConversationId convId, ISigner signer)
        => _g.Derive(source, "translate: " + translation, original, convId, signer);

    public Event Prophesy(ActorId source, string prediction, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "prophesy: " + prediction, convId, causes, signer);

    // Named Functions

    public sealed record DesignReviewResult(Event Beauty, Event Reframe, Event Question, Event Wisdom);

    public DesignReviewResult DesignReview(ActorId source, string beauty, string reframing, string question, string wisdom,
        EventId cause, ConversationId convId, ISigner signer)
    {
        var beautify = Beautify(source, beauty, new List<EventId> { cause }, convId, signer);
        var reframe = Reframe(source, reframing, beautify.Id, convId, signer);
        var q = Question(source, question, reframe.Id, convId, signer);
        var w = Distill(source, wisdom, q.Id, convId, signer);
        return new DesignReviewResult(beautify, reframe, q, w);
    }

    public sealed record CulturalOnboardingResult(Event Translation, Event Teaching, Event Examination);

    public CulturalOnboardingResult CulturalOnboarding(ActorId guide, ActorId newcomer,
        string translation, Option<DomainScope> teachingScope, string examination,
        EventId cause, ConversationId convId, ISigner signer)
    {
        var translate = Translate(guide, translation, cause, convId, signer);
        var teach = Teach(guide, newcomer, teachingScope, translate.Id, convId, signer);
        var examine = Examine(newcomer, examination, new List<EventId> { teach.Id }, convId, signer);
        return new CulturalOnboardingResult(translate, teach, examine);
    }

    public sealed record ForecastResult(Event Prophecy, Event Examination, Event Wisdom);

    public ForecastResult Forecast(ActorId source, string prediction, string assumptions, string confidence,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var prophesy = Prophesy(source, prediction, causes, convId, signer);
        var examine = Examine(source, assumptions, new List<EventId> { prophesy.Id }, convId, signer);
        var distill = Distill(source, confidence, examine.Id, convId, signer);
        return new ForecastResult(prophesy, examine, distill);
    }

    public sealed record MeaningPostMortemResult(Event Examination, Event Questions, Event Wisdom);

    public MeaningPostMortemResult PostMortem(ActorId source, string examination, string question, string wisdom,
        EventId cause, ConversationId convId, ISigner signer)
    {
        var exam = Examine(source, examination, new List<EventId> { cause }, convId, signer);
        var q = Question(source, question, exam.Id, convId, signer);
        var w = Distill(source, wisdom, q.Id, convId, signer);
        return new MeaningPostMortemResult(exam, q, w);
    }

    public sealed record MeaningMentorshipResult(Event Channel, Event Reframing, Event Wisdom, Event Translation);

    public MeaningMentorshipResult Mentorship(ActorId mentor, ActorId student,
        string reframing, string wisdom, string translation,
        Option<DomainScope> scope, EventId cause, ConversationId convId, ISigner signer)
    {
        var channel = Teach(mentor, student, scope, cause, convId, signer);
        var reframe = Reframe(mentor, reframing, channel.Id, convId, signer);
        var distill = Distill(mentor, wisdom, reframe.Id, convId, signer);
        var translate = Translate(student, translation, distill.Id, convId, signer);
        return new MeaningMentorshipResult(channel, reframe, distill, translate);
    }
}

// ── EvolutionGrammar (Layer 12 — Emergence) ────────────────────────────

/// <summary>Layer 12 composition operations for system self-awareness and evolution.</summary>
public sealed class EvolutionGrammar
{
    private readonly Grammar _g;
    public EvolutionGrammar(Grammar g) => _g = g;

    public Event DetectPattern(ActorId source, string pattern, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "pattern: " + pattern, convId, causes, signer);

    public Event Model(ActorId source, string model, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "model: " + model, convId, causes, signer);

    public Event TraceLoop(ActorId source, string loop, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "loop: " + loop, convId, causes, signer);

    public Event WatchThreshold(ActorId source, EventId target, string threshold, ConversationId convId, ISigner signer)
        => _g.Annotate(source, target, "threshold", threshold, convId, signer);

    public Event Adapt(ActorId source, string proposal, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "adapt: " + proposal, convId, causes, signer);

    public Event Select(ActorId source, string result, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "select: " + result, convId, causes, signer);

    public Event Simplify(ActorId source, string simplification, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "simplify: " + simplification, convId, causes, signer);

    public Event CheckIntegrity(ActorId source, string assessment, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "integrity: " + assessment, convId, causes, signer);

    public Event AssessResilience(ActorId source, string assessment, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "resilience: " + assessment, convId, causes, signer);

    public Event AlignPurpose(ActorId source, string alignment, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "purpose: " + alignment, convId, causes, signer);

    // Named Functions

    public sealed record SelfEvolveResult(Event Pattern, Event Adaptation, Event Selection, Event Simplification);

    public SelfEvolveResult SelfEvolve(ActorId source, string pattern, string adaptation, string selection, string simplification,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var pat = DetectPattern(source, pattern, causes, convId, signer);
        var adapt = Adapt(source, adaptation, new List<EventId> { pat.Id }, convId, signer);
        var sel = Select(source, selection, new List<EventId> { adapt.Id }, convId, signer);
        var simp = Simplify(source, simplification, new List<EventId> { sel.Id }, convId, signer);
        return new SelfEvolveResult(pat, adapt, sel, simp);
    }

    public sealed record HealthCheckResult(Event Integrity, Event Resilience, Event Model, Event Purpose);

    public HealthCheckResult HealthCheck(ActorId source, string integrity, string resilience, string model, string purpose,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var integ = CheckIntegrity(source, integrity, causes, convId, signer);
        var resil = AssessResilience(source, resilience, new List<EventId> { integ.Id }, convId, signer);
        var mod = Model(source, model, new List<EventId> { resil.Id }, convId, signer);
        var purp = AlignPurpose(source, purpose, new List<EventId> { mod.Id }, convId, signer);
        return new HealthCheckResult(integ, resil, mod, purp);
    }

    public sealed record PruneResult(Event Pattern, Event Simplification, Event Verification);

    public PruneResult Prune(ActorId source, string unusedPattern, string simplification, string verification,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var pattern = DetectPattern(source, "unused: " + unusedPattern, causes, convId, signer);
        var simplify = Simplify(source, simplification, new List<EventId> { pattern.Id }, convId, signer);
        var verify = Select(source, verification, new List<EventId> { simplify.Id }, convId, signer);
        return new PruneResult(pattern, simplify, verify);
    }

    public sealed record PhaseTransitionResult(Event Threshold, Event Model, Event Adaptation, Event Selection);

    public PhaseTransitionResult PhaseTransition(ActorId source, EventId target,
        string threshold, string model, string adaptation, string selection,
        ConversationId convId, ISigner signer)
    {
        var thresh = WatchThreshold(source, target, threshold, convId, signer);
        var mod = Model(source, model, new List<EventId> { thresh.Id }, convId, signer);
        var adapt = Adapt(source, adaptation, new List<EventId> { mod.Id }, convId, signer);
        var sel = Select(source, selection, new List<EventId> { adapt.Id }, convId, signer);
        return new PhaseTransitionResult(thresh, mod, adapt, sel);
    }
}

// ── BeingGrammar (Layer 13 — Existence) ────────────────────────────────

/// <summary>Layer 13 composition operations for the system's relationship with its own existence.</summary>
public sealed class BeingGrammar
{
    private readonly Grammar _g;
    public BeingGrammar(Grammar g) => _g = g;

    public Event Exist(ActorId source, string observation, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "exist: " + observation, convId, causes, signer);

    public Event Accept(ActorId source, string limitation, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "accept: " + limitation, convId, causes, signer);

    public Event ObserveChange(ActorId source, string observation, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "change: " + observation, convId, causes, signer);

    public Event MapWeb(ActorId source, string mapping, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "web: " + mapping, convId, causes, signer);

    public Event FaceMystery(ActorId source, string mystery, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "mystery: " + mystery, convId, causes, signer);

    public Event HoldParadox(ActorId source, string paradox, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "paradox: " + paradox, convId, causes, signer);

    public Event Marvel(ActorId source, string awe, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "marvel: " + awe, convId, causes, signer);

    public Event AskWhy(ActorId source, string question, List<EventId> causes, ConversationId convId, ISigner signer)
        => _g.Emit(source, "wonder: " + question, convId, causes, signer);

    // Named Functions

    public sealed record BeingFarewellResult(Event Acceptance, Event Web, Event Awe, Event Memorial);

    public BeingFarewellResult Farewell(ActorId source, string limitation, string interconnection, string awe, string memorial,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var accept = Accept(source, limitation, causes, convId, signer);
        var web = MapWeb(source, interconnection, new List<EventId> { accept.Id }, convId, signer);
        var marvel = Marvel(source, awe, new List<EventId> { web.Id }, convId, signer);
        var mem = _g.Emit(source, "memorialize: " + memorial, convId, new List<EventId> { marvel.Id }, signer);
        return new BeingFarewellResult(accept, web, marvel, mem);
    }

    public sealed record ContemplationResult(Event Change, Event Mystery, Event Awe, Event Wonder);

    public ContemplationResult Contemplation(ActorId source, string change, string mystery, string awe, string question,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var changeEv = ObserveChange(source, change, causes, convId, signer);
        var mysteryEv = FaceMystery(source, mystery, new List<EventId> { changeEv.Id }, convId, signer);
        var aweEv = Marvel(source, awe, new List<EventId> { mysteryEv.Id }, convId, signer);
        var wonderEv = AskWhy(source, question, new List<EventId> { aweEv.Id }, convId, signer);
        return new ContemplationResult(changeEv, mysteryEv, aweEv, wonderEv);
    }

    public sealed record ExistentialAuditResult(Event Existence, Event Acceptance, Event Web, Event Purpose);

    public ExistentialAuditResult ExistentialAudit(ActorId source,
        string existence, string limitation, string interconnection, string purpose,
        List<EventId> causes, ConversationId convId, ISigner signer)
    {
        var exist = Exist(source, existence, causes, convId, signer);
        var accept = Accept(source, limitation, new List<EventId> { exist.Id }, convId, signer);
        var web = MapWeb(source, interconnection, new List<EventId> { accept.Id }, convId, signer);
        var purp = _g.Emit(source, "purpose: " + purpose, convId, new List<EventId> { web.Id }, signer);
        return new ExistentialAuditResult(exist, accept, web, purp);
    }
}
