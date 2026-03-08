namespace EventGraph.Tests;

public class CompositionsTests
{
    private static readonly ActorId Alice = new("alice");
    private static readonly ActorId Bob = new("bob");
    private static readonly ActorId Charlie = new("charlie");
    private static readonly ConversationId ConvId = new("conv_1");
    private static readonly ISigner Signer = new NoopSigner();
    private static readonly DomainScope TestScope = new("test.scope");

    private static (Grammar grammar, IStore store, Event bootstrap) Setup()
    {
        var store = new InMemoryStore();
        var boot = EventFactory.CreateBootstrap(Alice, Signer);
        store.Append(boot);
        var grammar = new Grammar(store);
        return (grammar, store, boot);
    }

    // ── Grammar Edge Operations ────────────────────────────────────────

    [Fact]
    public void Acknowledge_CreatesEdgeEvent()
    {
        var (g, store, boot) = Setup();
        var emitEv = g.Emit(Alice, "content", ConvId, new List<EventId> { boot.Id }, Signer);
        var ack = g.Acknowledge(Alice, emitEv.Id, Bob, ConvId, Signer);

        Assert.Equal("edge.created", ack.Type.Value);
        Assert.Equal("acknowledgement", ack.Content["EdgeType"]);
        Assert.Equal("centripetal", ack.Content["Direction"]);
        Assert.Equal(Alice.Value, ack.Content["From"]);
        Assert.Equal(Bob.Value, ack.Content["To"]);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void Propagate_CreatesEdgeEvent()
    {
        var (g, _, boot) = Setup();
        var emitEv = g.Emit(Alice, "content", ConvId, new List<EventId> { boot.Id }, Signer);
        var prop = g.Propagate(Alice, emitEv.Id, Bob, ConvId, Signer);

        Assert.Equal("edge.created", prop.Type.Value);
        Assert.Equal("reference", prop.Content["EdgeType"]);
        Assert.Equal("centrifugal", prop.Content["Direction"]);
    }

    [Fact]
    public void Endorse_CreatesEdgeWithWeight()
    {
        var (g, _, boot) = Setup();
        var emitEv = g.Emit(Alice, "content", ConvId, new List<EventId> { boot.Id }, Signer);
        var endorsement = g.Endorse(Alice, emitEv.Id, Bob, new Weight(0.7), Option<DomainScope>.Some(TestScope), ConvId, Signer);

        Assert.Equal("edge.created", endorsement.Type.Value);
        Assert.Equal("endorsement", endorsement.Content["EdgeType"]);
        Assert.Equal(0.7, endorsement.Content["Weight"]);
        Assert.Equal(TestScope.Value, endorsement.Content["Scope"]);
    }

    [Fact]
    public void Subscribe_CreatesEdgeEvent()
    {
        var (g, _, boot) = Setup();
        var sub = g.Subscribe(Alice, Bob, Option<DomainScope>.None(), boot.Id, ConvId, Signer);

        Assert.Equal("edge.created", sub.Type.Value);
        Assert.Equal("subscription", sub.Content["EdgeType"]);
        Assert.Equal("centripetal", sub.Content["Direction"]);
        Assert.Null(sub.Content["Scope"]);
    }

    [Fact]
    public void Channel_CreatesEdgeEvent()
    {
        var (g, _, boot) = Setup();
        var ch = g.Channel(Alice, Bob, Option<DomainScope>.None(), boot.Id, ConvId, Signer);

        Assert.Equal("edge.created", ch.Type.Value);
        Assert.Equal("channel", ch.Content["EdgeType"]);
    }

    [Fact]
    public void Delegate_CreatesEdgeEvent()
    {
        var (g, _, boot) = Setup();
        var del = g.Delegate(Alice, Bob, TestScope, new Weight(0.5), boot.Id, ConvId, Signer);

        Assert.Equal("edge.created", del.Type.Value);
        Assert.Equal("delegation", del.Content["EdgeType"]);
        Assert.Equal("centrifugal", del.Content["Direction"]);
        Assert.Equal(0.5, del.Content["Weight"]);
        Assert.Equal(TestScope.Value, del.Content["Scope"]);
    }

    [Fact]
    public void Consent_CreatesConsentEvent()
    {
        var (g, _, boot) = Setup();
        var consent = g.Consent(Alice, Bob, "agreement", TestScope, boot.Id, ConvId, Signer);

        Assert.Equal("grammar.consented", consent.Type.Value);
        Assert.Equal(Alice.Value, consent.Content["PartyA"]);
        Assert.Equal(Bob.Value, consent.Content["PartyB"]);
        Assert.Equal("agreement", consent.Content["Agreement"]);
        Assert.Equal(TestScope.Value, consent.Content["Scope"]);
    }

    [Fact]
    public void Sever_SupersedesEdge()
    {
        var (g, store, boot) = Setup();
        var sub = g.Subscribe(Alice, Bob, Option<DomainScope>.None(), boot.Id, ConvId, Signer);
        var edgeId = new EdgeId(sub.Id.Value);
        var sever = g.Sever(Alice, edgeId, boot.Id, ConvId, Signer);

        Assert.Equal("edge.superseded", sever.Type.Value);
        Assert.Equal(edgeId.Value, sever.Content["PreviousEdge"]);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void Sever_ThrowsForNonParty()
    {
        var (g, _, boot) = Setup();
        var sub = g.Subscribe(Alice, Bob, Option<DomainScope>.None(), boot.Id, ConvId, Signer);
        var edgeId = new EdgeId(sub.Id.Value);

        Assert.Throws<InvalidOperationException>(() =>
            g.Sever(Charlie, edgeId, boot.Id, ConvId, Signer));
    }

    [Fact]
    public void Sever_ThrowsForNonSeverableEdge()
    {
        var (g, _, boot) = Setup();
        var emitEv = g.Emit(Alice, "content", ConvId, new List<EventId> { boot.Id }, Signer);
        var ack = g.Acknowledge(Alice, emitEv.Id, Bob, ConvId, Signer);
        var edgeId = new EdgeId(ack.Id.Value);

        Assert.Throws<InvalidOperationException>(() =>
            g.Sever(Alice, edgeId, boot.Id, ConvId, Signer));
    }

    // ── Grammar Named Functions ────────────────────────────────────────

    [Fact]
    public void Challenge_CreatesResponseAndFlag()
    {
        var (g, store, boot) = Setup();
        var emitEv = g.Emit(Alice, "claim", ConvId, new List<EventId> { boot.Id }, Signer);
        var (response, flag) = g.Challenge(Bob, "I disagree", emitEv.Id, ConvId, Signer);

        Assert.Equal("grammar.responded", response.Type.Value);
        Assert.Equal("grammar.annotated", flag.Type.Value);
        Assert.Equal("dispute", flag.Content["Key"]);
        Assert.Equal("challenged", flag.Content["Value"]);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void Recommend_CreatesPropagateAndChannel()
    {
        var (g, store, boot) = Setup();
        var emitEv = g.Emit(Alice, "good content", ConvId, new List<EventId> { boot.Id }, Signer);
        var (prop, ch) = g.Recommend(Alice, emitEv.Id, Bob, ConvId, Signer);

        Assert.Equal("edge.created", prop.Type.Value);
        Assert.Equal("reference", prop.Content["EdgeType"]);
        Assert.Equal("edge.created", ch.Type.Value);
        Assert.Equal("channel", ch.Content["EdgeType"]);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void Invite_CreatesEndorseAndSubscribe()
    {
        var (g, store, boot) = Setup();
        var (endorseEv, subEv) = g.Invite(Alice, Bob, new Weight(0.5), Option<DomainScope>.None(), boot.Id, ConvId, Signer);

        Assert.Equal("edge.created", endorseEv.Type.Value);
        Assert.Equal("endorsement", endorseEv.Content["EdgeType"]);
        Assert.Equal("edge.created", subEv.Type.Value);
        Assert.Equal("subscription", subEv.Content["EdgeType"]);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void Forgive_CreatesSubscription()
    {
        var (g, store, boot) = Setup();
        var sub = g.Subscribe(Alice, Bob, Option<DomainScope>.None(), boot.Id, ConvId, Signer);
        var edgeId = new EdgeId(sub.Id.Value);
        var sever = g.Sever(Alice, edgeId, boot.Id, ConvId, Signer);
        var forgive = g.Forgive(Alice, sever.Id, Bob, Option<DomainScope>.None(), ConvId, Signer);

        Assert.Equal("edge.created", forgive.Type.Value);
        Assert.Equal("subscription", forgive.Content["EdgeType"]);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── Edge operations maintain chain integrity ───────────────────────

    [Fact]
    public void AllEdgeOperations_MaintainChainIntegrity()
    {
        var (g, store, boot) = Setup();

        var emitEv = g.Emit(Alice, "content", ConvId, new List<EventId> { boot.Id }, Signer);
        g.Acknowledge(Alice, emitEv.Id, Bob, ConvId, Signer);
        g.Propagate(Alice, emitEv.Id, Bob, ConvId, Signer);
        g.Endorse(Alice, emitEv.Id, Bob, new Weight(0.5), Option<DomainScope>.None(), ConvId, Signer);
        var sub = g.Subscribe(Alice, Bob, Option<DomainScope>.None(), boot.Id, ConvId, Signer);
        g.Channel(Alice, Bob, Option<DomainScope>.None(), boot.Id, ConvId, Signer);
        g.Delegate(Alice, Bob, TestScope, new Weight(0.3), boot.Id, ConvId, Signer);
        g.Consent(Alice, Bob, "agree", TestScope, boot.Id, ConvId, Signer);
        g.Sever(Alice, new EdgeId(sub.Id.Value), boot.Id, ConvId, Signer);

        var v = store.VerifyChain();
        Assert.True(v.Valid);
    }

    // ── WorkGrammar ────────────────────────────────────────────────────

    [Fact]
    public void WorkGrammar_IntendAndDecompose()
    {
        var (g, store, boot) = Setup();
        var wg = new WorkGrammar(g);

        var intent = wg.Intend(Alice, "build feature", new List<EventId> { boot.Id }, ConvId, Signer);
        Assert.Contains("intend:", (string?)intent.Content["Body"]);

        var task = wg.Decompose(Alice, "sub-task 1", intent.Id, ConvId, Signer);
        Assert.Contains("decompose:", (string?)task.Content["Body"]);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void WorkGrammar_Sprint()
    {
        var (g, store, boot) = Setup();
        var wg = new WorkGrammar(g);

        var result = wg.Sprint(Alice, "sprint goal",
            new List<string> { "task1", "task2" },
            new List<ActorId> { Bob, Charlie },
            new List<DomainScope> { TestScope, TestScope },
            new List<EventId> { boot.Id },
            ConvId, Signer);

        Assert.NotNull(result.Intent);
        Assert.Equal(2, result.Subtasks.Count);
        Assert.Equal(2, result.Assignments.Count);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void WorkGrammar_Escalate()
    {
        var (g, store, boot) = Setup();
        var wg = new WorkGrammar(g);

        var task = wg.Intend(Alice, "task", new List<EventId> { boot.Id }, ConvId, Signer);
        var result = wg.Escalate(Alice, "dependency missing", task.Id, Bob, TestScope, ConvId, Signer);

        Assert.NotNull(result.BlockEvent);
        Assert.NotNull(result.HandoffEvent);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void WorkGrammar_Standup()
    {
        var (g, store, boot) = Setup();
        var wg = new WorkGrammar(g);

        var task = wg.Intend(Alice, "task", new List<EventId> { boot.Id }, ConvId, Signer);
        var result = wg.Standup(
            new List<ActorId> { Alice, Bob },
            new List<string> { "done thing A", "done thing B" },
            Alice, "focus on X",
            new List<EventId> { task.Id },
            ConvId, Signer);

        Assert.Equal(2, result.Updates.Count);
        Assert.NotNull(result.Priority);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── MarketGrammar ──────────────────────────────────────────────────

    [Fact]
    public void MarketGrammar_ListBidAccept()
    {
        var (g, store, boot) = Setup();
        var mg = new MarketGrammar(g);

        var listing = mg.List(Alice, "widget", new List<EventId> { boot.Id }, ConvId, Signer);
        var bid = mg.Bid(Bob, "$10", listing.Id, ConvId, Signer);
        var accept = mg.Accept(Bob, Alice, "deal at $10", TestScope, bid.Id, ConvId, Signer);

        Assert.Contains("list:", (string?)listing.Content["Body"]);
        Assert.Contains("bid:", (string?)bid.Content["Body"]);
        Assert.Equal("grammar.consented", accept.Type.Value);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void MarketGrammar_Auction()
    {
        var (g, store, boot) = Setup();
        var mg = new MarketGrammar(g);

        var result = mg.Auction(Alice, "rare item",
            new List<ActorId> { Bob, Charlie },
            new List<string> { "$50", "$75" },
            1, TestScope, new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Listing);
        Assert.Equal(2, result.Bids.Count);
        Assert.NotNull(result.Acceptance);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void MarketGrammar_Barter()
    {
        var (g, store, boot) = Setup();
        var mg = new MarketGrammar(g);

        var result = mg.Barter(Alice, Bob, "apples", "oranges", TestScope,
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Listing);
        Assert.NotNull(result.CounterOffer);
        Assert.NotNull(result.Acceptance);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── SocialGrammar ──────────────────────────────────────────────────

    [Fact]
    public void SocialGrammar_NormAndModerate()
    {
        var (g, store, boot) = Setup();
        var sg = new SocialGrammar(g);

        var norm = sg.Norm(Alice, Bob, "be kind", TestScope, boot.Id, ConvId, Signer);
        Assert.Equal("grammar.consented", norm.Type.Value);

        var content = g.Emit(Charlie, "rude post", ConvId, new List<EventId> { boot.Id }, Signer);
        var mod = sg.Moderate(Alice, content.Id, "warning", ConvId, Signer);
        Assert.Equal("moderation", mod.Content["Key"]);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void SocialGrammar_Welcome()
    {
        var (g, store, boot) = Setup();
        var sg = new SocialGrammar(g);

        var (endorseEv, subEv) = sg.Welcome(Alice, Bob, new Weight(0.3), Option<DomainScope>.None(), boot.Id, ConvId, Signer);
        Assert.Equal("endorsement", endorseEv.Content["EdgeType"]);
        Assert.Equal("subscription", subEv.Content["EdgeType"]);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void SocialGrammar_Poll()
    {
        var (g, store, boot) = Setup();
        var sg = new SocialGrammar(g);

        var result = sg.Poll(Alice, "should we do X?", new List<ActorId> { Bob, Charlie },
            TestScope, boot.Id, ConvId, Signer);

        Assert.NotNull(result.Proposal);
        Assert.Equal(2, result.Votes.Count);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── JusticeGrammar ─────────────────────────────────────────────────

    [Fact]
    public void JusticeGrammar_Trial()
    {
        var (g, store, boot) = Setup();
        var jg = new JusticeGrammar(g);

        var target = g.Emit(Alice, "disputed action", ConvId, new List<EventId> { boot.Id }, Signer);
        var result = jg.Trial(Alice, Bob, Charlie,
            "breach of norm", "plaintiff evidence", "defendant evidence",
            "plaintiff argument", "defendant argument", "guilty",
            target.Id, ConvId, Signer);

        Assert.NotNull(result.Filing);
        Assert.Equal(2, result.Submissions.Count);
        Assert.Equal(2, result.Arguments.Count);
        Assert.NotNull(result.Ruling);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void JusticeGrammar_ConstitutionalAmendment()
    {
        var (g, store, boot) = Setup();
        var jg = new JusticeGrammar(g);

        var target = g.Emit(Alice, "existing law", ConvId, new List<EventId> { boot.Id }, Signer);
        var result = jg.ConstitutionalAmendment(Alice, "proposal", "new legislation", "rights ok", target.Id, ConvId, Signer);

        Assert.NotNull(result.Reform);
        Assert.NotNull(result.Legislation);
        Assert.NotNull(result.RightsCheck);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── BuildGrammar ───────────────────────────────────────────────────

    [Fact]
    public void BuildGrammar_Pipeline()
    {
        var (g, store, boot) = Setup();
        var bg = new BuildGrammar(g);

        var result = bg.Pipeline(Alice, "CI/CD", "all pass", "coverage 95%", "prod",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Definition);
        Assert.NotNull(result.TestResult);
        Assert.NotNull(result.Metrics);
        Assert.NotNull(result.Deployment);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void BuildGrammar_Spike()
    {
        var (g, store, boot) = Setup();
        var bg = new BuildGrammar(g);

        var result = bg.Spike(Alice, "try new approach", "tests pass", "looks good", "adopt",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Build);
        Assert.NotNull(result.Test);
        Assert.NotNull(result.Feedback);
        Assert.NotNull(result.Decision);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void BuildGrammar_Migration()
    {
        var (g, store, boot) = Setup();
        var bg = new BuildGrammar(g);

        var artefact = bg.Build(Alice, "old system", new List<EventId> { boot.Id }, ConvId, Signer);
        var result = bg.Migration(Alice, artefact.Id, "use new API", "v2.0", "prod deploy", "all pass", ConvId, Signer);

        Assert.NotNull(result.Sunset);
        Assert.NotNull(result.Version);
        Assert.NotNull(result.Ship);
        Assert.NotNull(result.Test);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── KnowledgeGrammar ───────────────────────────────────────────────

    [Fact]
    public void KnowledgeGrammar_FactCheck()
    {
        var (g, store, boot) = Setup();
        var kg = new KnowledgeGrammar(g);

        var claim = kg.Claim(Alice, "the sky is blue", new List<EventId> { boot.Id }, ConvId, Signer);
        var result = kg.FactCheck(Bob, claim.Id, "primary source", "no bias", "confirmed", ConvId, Signer);

        Assert.NotNull(result.Provenance);
        Assert.NotNull(result.BiasCheck);
        Assert.NotNull(result.Verdict);
        Assert.Equal("grammar.merged", result.Verdict.Type.Value);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void KnowledgeGrammar_Survey()
    {
        var (g, store, boot) = Setup();
        var kg = new KnowledgeGrammar(g);

        var result = kg.Survey(Alice, new List<string> { "topic A", "topic B" },
            "general pattern", "synthesis result",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.Equal(2, result.Recalls.Count);
        Assert.NotNull(result.Abstraction);
        Assert.NotNull(result.Synthesis);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void KnowledgeGrammar_Transfer()
    {
        var (g, store, boot) = Setup();
        var kg = new KnowledgeGrammar(g);

        var result = kg.Transfer(Alice, "old knowledge", "new format", "applied insight",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Recall);
        Assert.NotNull(result.Encode);
        Assert.NotNull(result.Learn);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── AlignmentGrammar ───────────────────────────────────────────────

    [Fact]
    public void AlignmentGrammar_EthicsAudit()
    {
        var (g, store, boot) = Setup();
        var ag = new AlignmentGrammar(g);

        var target = g.Emit(Alice, "decision", ConvId, new List<EventId> { boot.Id }, Signer);
        var result = ag.EthicsAudit(Bob, target.Id, "fair", "no harm", "all clear", ConvId, Signer);

        Assert.NotNull(result.Fairness);
        Assert.NotNull(result.HarmScan);
        Assert.NotNull(result.Report);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void AlignmentGrammar_RestorativeJustice()
    {
        var (g, store, boot) = Setup();
        var ag = new AlignmentGrammar(g);

        var result = ag.RestorativeJustice(Alice, Bob, Charlie,
            "caused harm", "responsible", "will fix", "learned lesson",
            TestScope, boot.Id, ConvId, Signer);

        Assert.NotNull(result.HarmDetection);
        Assert.NotNull(result.Responsibility);
        Assert.NotNull(result.Redress);
        Assert.NotNull(result.Growth);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void AlignmentGrammar_Guardrail()
    {
        var (g, store, boot) = Setup();
        var ag = new AlignmentGrammar(g);

        var target = g.Emit(Alice, "action", ConvId, new List<EventId> { boot.Id }, Signer);
        var result = ag.Guardrail(Alice, target.Id, "limit X", "values conflict", "needs human review", ConvId, Signer);

        Assert.NotNull(result.Constraint);
        Assert.NotNull(result.Dilemma);
        Assert.NotNull(result.Escalation);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── IdentityGrammar ────────────────────────────────────────────────

    [Fact]
    public void IdentityGrammar_IdentityAudit()
    {
        var (g, store, boot) = Setup();
        var ig = new IdentityGrammar(g);

        var result = ig.IdentityAudit(Alice, "who I am", "aligned", "my story",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.SelfModel);
        Assert.NotNull(result.Alignment);
        Assert.NotNull(result.Narrative);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void IdentityGrammar_Reinvention()
    {
        var (g, store, boot) = Setup();
        var ig = new IdentityGrammar(g);

        var result = ig.Reinvention(Alice, "new me", "new story", "new goal",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Transformation);
        Assert.NotNull(result.Narrative);
        Assert.NotNull(result.Aspiration);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void IdentityGrammar_Credential()
    {
        var (g, store, boot) = Setup();
        var ig = new IdentityGrammar(g);

        var result = ig.Credential(Alice, Bob, "I am qualified",
            Option<DomainScope>.None(), new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Introspection);
        Assert.NotNull(result.Disclosure);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── BondGrammar ────────────────────────────────────────────────────

    [Fact]
    public void BondGrammar_Connect()
    {
        var (g, store, boot) = Setup();
        var bg = new BondGrammar(g);

        var (sub1, sub2) = bg.Connect(Alice, Bob, Option<DomainScope>.None(), boot.Id, ConvId, Signer);
        Assert.Equal("subscription", sub1.Content["EdgeType"]);
        Assert.Equal("subscription", sub2.Content["EdgeType"]);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void BondGrammar_BetrayalRepair()
    {
        var (g, store, boot) = Setup();
        var bg = new BondGrammar(g);

        var result = bg.BetrayalRepair(Alice, Bob, "broken trust", "I am sorry",
            "rebuilding", "stronger foundation", TestScope,
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Rupture);
        Assert.NotNull(result.Apology);
        Assert.NotNull(result.Reconciliation);
        Assert.NotNull(result.Deepened);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void BondGrammar_CheckIn()
    {
        var (g, store, boot) = Setup();
        var bg = new BondGrammar(g);

        var target = g.Emit(Alice, "shared event", ConvId, new List<EventId> { boot.Id }, Signer);
        var result = bg.CheckIn(Alice, target.Id, "balanced", "attuned to you", "I understand", ConvId, Signer);

        Assert.NotNull(result.Balance);
        Assert.NotNull(result.Attunement);
        Assert.NotNull(result.Empathy);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void BondGrammar_Farewell()
    {
        var (g, store, boot) = Setup();
        var bg = new BondGrammar(g);

        var result = bg.Farewell(Alice, Bob, "goodbye", "wonderful memories", new Weight(0.9),
            Option<DomainScope>.None(), new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Mourning);
        Assert.NotNull(result.Memorial);
        Assert.NotNull(result.Gratitude);
        Assert.Equal("endorsement", result.Gratitude.Content["EdgeType"]);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── BelongingGrammar ───────────────────────────────────────────────

    [Fact]
    public void BelongingGrammar_Festival()
    {
        var (g, store, boot) = Setup();
        var bg = new BelongingGrammar(g);

        var result = bg.Festival(Alice, "harvest fest", "drumming", "the founding", "seeds",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Celebration);
        Assert.NotNull(result.Practice);
        Assert.NotNull(result.Story);
        Assert.NotNull(result.Gift);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void BelongingGrammar_Onboard()
    {
        var (g, store, boot) = Setup();
        var bg = new BelongingGrammar(g);

        var result = bg.Onboard(Alice, Bob, Charlie, Option<DomainScope>.None(),
            "open door", "daily standup", "first PR", boot.Id, ConvId, Signer);

        Assert.NotNull(result.Inclusion);
        Assert.NotNull(result.Settlement);
        Assert.NotNull(result.FirstPractice);
        Assert.NotNull(result.Contribution);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void BelongingGrammar_Succession()
    {
        var (g, store, boot) = Setup();
        var bg = new BelongingGrammar(g);

        var result = bg.Succession(Alice, Bob, "healthy", TestScope, "well done", "new chapter",
            boot.Id, ConvId, Signer);

        Assert.NotNull(result.Assessment);
        Assert.NotNull(result.Transfer);
        Assert.NotNull(result.Celebration);
        Assert.NotNull(result.Story);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── MeaningGrammar ─────────────────────────────────────────────────

    [Fact]
    public void MeaningGrammar_DesignReview()
    {
        var (g, store, boot) = Setup();
        var mg = new MeaningGrammar(g);

        var target = g.Emit(Alice, "design", ConvId, new List<EventId> { boot.Id }, Signer);
        var result = mg.DesignReview(Alice, "elegant", "new perspective", "why this way?", "simplicity matters",
            target.Id, ConvId, Signer);

        Assert.NotNull(result.Beauty);
        Assert.NotNull(result.Reframe);
        Assert.NotNull(result.Question);
        Assert.NotNull(result.Wisdom);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void MeaningGrammar_Forecast()
    {
        var (g, store, boot) = Setup();
        var mg = new MeaningGrammar(g);

        var result = mg.Forecast(Alice, "trend continues", "assumes growth", "high confidence",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Prophecy);
        Assert.NotNull(result.Examination);
        Assert.NotNull(result.Wisdom);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void MeaningGrammar_Mentorship()
    {
        var (g, store, boot) = Setup();
        var mg = new MeaningGrammar(g);

        var result = mg.Mentorship(Alice, Bob, "see it differently", "core lesson",
            "in my words", Option<DomainScope>.None(), boot.Id, ConvId, Signer);

        Assert.NotNull(result.Channel);
        Assert.NotNull(result.Reframing);
        Assert.NotNull(result.Wisdom);
        Assert.NotNull(result.Translation);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── EvolutionGrammar ───────────────────────────────────────────────

    [Fact]
    public void EvolutionGrammar_SelfEvolve()
    {
        var (g, store, boot) = Setup();
        var eg = new EvolutionGrammar(g);

        var result = eg.SelfEvolve(Alice, "repeated pattern", "try caching", "kept", "removed wrapper",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Pattern);
        Assert.NotNull(result.Adaptation);
        Assert.NotNull(result.Selection);
        Assert.NotNull(result.Simplification);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void EvolutionGrammar_HealthCheck()
    {
        var (g, store, boot) = Setup();
        var eg = new EvolutionGrammar(g);

        var result = eg.HealthCheck(Alice, "sound", "resilient", "feedback loops ok", "aligned",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Integrity);
        Assert.NotNull(result.Resilience);
        Assert.NotNull(result.Model);
        Assert.NotNull(result.Purpose);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void EvolutionGrammar_PhaseTransition()
    {
        var (g, store, boot) = Setup();
        var eg = new EvolutionGrammar(g);

        var target = g.Emit(Alice, "system state", ConvId, new List<EventId> { boot.Id }, Signer);
        var result = eg.PhaseTransition(Alice, target.Id,
            "approaching limit", "new dynamics", "restructure", "verified", ConvId, Signer);

        Assert.NotNull(result.Threshold);
        Assert.NotNull(result.Model);
        Assert.NotNull(result.Adaptation);
        Assert.NotNull(result.Selection);
        Assert.True(store.VerifyChain().Valid);
    }

    // ── BeingGrammar ───────────────────────────────────────────────────

    [Fact]
    public void BeingGrammar_Contemplation()
    {
        var (g, store, boot) = Setup();
        var bg = new BeingGrammar(g);

        var result = bg.Contemplation(Alice, "everything changes", "cannot know all",
            "how vast", "why anything at all?",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Change);
        Assert.NotNull(result.Mystery);
        Assert.NotNull(result.Awe);
        Assert.NotNull(result.Wonder);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void BeingGrammar_Farewell()
    {
        var (g, store, boot) = Setup();
        var bg = new BeingGrammar(g);

        var result = bg.Farewell(Alice, "finite", "connected to all", "so beautiful", "thank you",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Acceptance);
        Assert.NotNull(result.Web);
        Assert.NotNull(result.Awe);
        Assert.NotNull(result.Memorial);
        Assert.True(store.VerifyChain().Valid);
    }

    [Fact]
    public void BeingGrammar_ExistentialAudit()
    {
        var (g, store, boot) = Setup();
        var bg = new BeingGrammar(g);

        var result = bg.ExistentialAudit(Alice, "I am here", "I am limited", "connected", "to serve",
            new List<EventId> { boot.Id }, ConvId, Signer);

        Assert.NotNull(result.Existence);
        Assert.NotNull(result.Acceptance);
        Assert.NotNull(result.Web);
        Assert.NotNull(result.Purpose);
        Assert.True(store.VerifyChain().Valid);
    }
}
