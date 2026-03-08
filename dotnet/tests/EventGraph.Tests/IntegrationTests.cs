namespace EventGraph.Tests;

/// <summary>
/// Test environment for integration scenarios — mirrors Go's testEnv.
/// Sets up Graph + Grammar + Store + ActorStore, bootstraps, and provides convenience methods.
/// </summary>
internal sealed class TestEnv
{
    public Graph Graph { get; }
    public Grammar Grammar { get; }
    public InMemoryStore Store { get; }
    public InMemoryActorStore Actors { get; }
    public Event Boot { get; }
    public ConversationId ConvId { get; }
    public ActorId System { get; }
    public ISigner Signer { get; }

    private static readonly ISigner DefaultSigner = new NoopSigner();

    public TestEnv()
    {
        Store = new InMemoryStore();
        Actors = new InMemoryActorStore();
        Signer = DefaultSigner;
        System = new ActorId("actor_system0000000000000000000001");
        ConvId = new ConversationId("conv_test000000000000000000000001");

        Graph = new Graph(Store, Actors);
        Graph.Start();
        Boot = Graph.Bootstrap(System, Signer);
        Grammar = new Grammar(Store);
    }

    private static PublicKey TestPublicKey(byte b)
    {
        var key = new byte[32];
        key[0] = b;
        return new PublicKey(key);
    }

    public Actor RegisterActor(string name, byte pkByte, ActorType actorType)
        => Actors.Register(TestPublicKey(pkByte), name, actorType);

    public void VerifyChain()
    {
        var result = Store.VerifyChain();
        Assert.True(result.Valid, $"chain integrity broken at length {result.Length}");
    }

    public int EventCount() => Store.Count();

    public List<Event> Ancestors(EventId id, int depth)
        => Graph.Query().Ancestors(id, depth);

    public List<Event> Descendants(EventId id, int depth)
        => Graph.Query().Descendants(id, depth);

    public Event Record(EventType eventType, ActorId source,
        Dictionary<string, object?> content, List<EventId> causes,
        ConversationId conversationId)
        => Graph.Record(eventType, source, content, causes, conversationId, Signer);

    public static bool ContainsEvent(List<Event> events, EventId id)
        => events.Any(e => e.Id == id);

    public static bool ContainsEventType(List<Event> events, string typeName)
        => events.Any(e => e.Type.Value == typeName);
}

/// <summary>
/// 21 integration test scenarios ported from the Go reference implementation.
/// </summary>
public class IntegrationTests
{
    // ── Scenario 01: Agent Audit Trail ─────────────────────────────────────

    [Fact]
    public void Scenario01_AgentAuditTrail()
    {
        var env = new TestEnv();
        var alice = env.RegisterActor("Alice", 1, ActorType.Human);
        var agent = env.RegisterActor("ReviewBot", 2, ActorType.AI);

        // 1. Alice submits code for review
        var submission = env.Grammar.Emit(alice.Id, "code submission: auth module refactor",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        // 2. Alice delegates code_review authority to agent
        var delegation = env.Grammar.Delegate(alice.Id, agent.Id,
            new DomainScope("code_review"), new Weight(0.8),
            submission.Id, env.ConvId, env.Signer);

        // 3. Agent reviews the code
        var review = env.Grammar.Derive(agent.Id,
            "review: LGTM, no issues found, approving PR",
            submission.Id, env.ConvId, env.Signer);

        // 4. Agent approves
        var approval = env.Grammar.Respond(agent.Id,
            "decision: approve PR with confidence 0.85",
            review.Id, env.ConvId, env.Signer);

        // 5. Trust updated after successful review
        var trustUp = env.Record(new EventType("trust.updated"), env.System,
            new Dictionary<string, object?>
            {
                ["Actor"] = agent.Id.Value, ["Previous"] = 0.1, ["Current"] = 0.3,
                ["Domain"] = "code_review", ["Cause"] = approval.Id.Value
            },
            new List<EventId> { approval.Id }, env.ConvId);

        // 6. Bug discovered in approved code
        var bugReport = env.Grammar.Emit(alice.Id,
            "bug found in auth module: session tokens not invalidated on logout",
            env.ConvId, new List<EventId> { approval.Id }, env.Signer);

        // 7. Violation detected
        var violation = env.Record(new EventType("violation.detected"), env.System,
            new Dictionary<string, object?>
            {
                ["Expectation"] = approval.Id.Value, ["Actor"] = agent.Id.Value,
                ["Severity"] = "serious",
                ["Description"] = "agent approved code with session management bug",
                ["Evidence"] = new List<object?> { bugReport.Id.Value }
            },
            new List<EventId> { bugReport.Id, approval.Id }, env.ConvId);

        // 8. Trust decreases
        env.Record(new EventType("trust.updated"), env.System,
            new Dictionary<string, object?>
            {
                ["Actor"] = agent.Id.Value, ["Previous"] = 0.3, ["Current"] = 0.15,
                ["Domain"] = "code_review", ["Cause"] = violation.Id.Value
            },
            new List<EventId> { violation.Id }, env.ConvId);

        // --- Assertions ---
        var ancestors = env.Ancestors(bugReport.Id, 10);
        Assert.True(TestEnv.ContainsEvent(ancestors, approval.Id),
            "bug report should have approval in ancestors");

        var violationAncestors = env.Ancestors(violation.Id, 10);
        Assert.True(TestEnv.ContainsEvent(violationAncestors, bugReport.Id),
            "violation should have bug report in ancestors");
        Assert.True(TestEnv.ContainsEvent(violationAncestors, approval.Id),
            "violation should have approval in ancestors");
        Assert.True(TestEnv.ContainsEvent(violationAncestors, submission.Id),
            "violation should trace back to original submission");

        _ = delegation;
        _ = trustUp;

        env.VerifyChain();
        Assert.Equal(9, env.EventCount());
    }

    // ── Scenario 02: Freelancer Reputation ─────────────────────────────────

    [Fact]
    public void Scenario02_FreelancerReputation()
    {
        var env = new TestEnv();
        var carol = env.RegisterActor("Carol", 1, ActorType.Human);
        var bob = env.RegisterActor("Bob", 2, ActorType.Human);
        var dave = env.RegisterActor("Dave", 3, ActorType.Human);

        // 1. Carol posts a job listing
        var listing = env.Grammar.Emit(carol.Id,
            "job listing: build REST API for inventory management, budget $3000",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        // 2. Bob proposes work
        var proposal = env.Grammar.Respond(bob.Id,
            "proposal: can deliver in 2 weeks, $2800, Go + PostgreSQL",
            listing.Id, env.ConvId, env.Signer);

        // 3. Carol and Bob open a channel
        var channel = env.Grammar.Channel(carol.Id, bob.Id,
            Option<DomainScope>.Some(new DomainScope("software_development")),
            proposal.Id, env.ConvId, env.Signer);

        // 4. Both consent to bilateral contract
        var contract = env.Grammar.Consent(carol.Id, bob.Id,
            "REST API for inventory management, $2800, 2 week deadline",
            new DomainScope("software_development"),
            channel.Id, env.ConvId, env.Signer);

        // 5. Bob delivers work
        var delivery = env.Grammar.Derive(bob.Id,
            "work delivered: REST API complete, 47 endpoints, 92% test coverage",
            contract.Id, env.ConvId, env.Signer);

        // 6. Carol acknowledges receipt
        var ack = env.Grammar.Acknowledge(carol.Id, delivery.Id, bob.Id,
            env.ConvId, env.Signer);

        // 7. Carol endorses Bob's work
        var endorsement = env.Grammar.Endorse(carol.Id, delivery.Id, bob.Id,
            new Weight(0.8), Option<DomainScope>.Some(new DomainScope("software_development")),
            env.ConvId, env.Signer);

        // 8. Trust updated for Bob
        env.Record(new EventType("trust.updated"), env.System,
            new Dictionary<string, object?>
            {
                ["Actor"] = bob.Id.Value, ["Previous"] = 0.1, ["Current"] = 0.4,
                ["Domain"] = "software_development", ["Cause"] = endorsement.Id.Value
            },
            new List<EventId> { endorsement.Id }, env.ConvId);

        // 9. Dave queries Bob's reputation
        var endorseAncestors = env.Ancestors(endorsement.Id, 10);
        Assert.True(TestEnv.ContainsEvent(endorseAncestors, delivery.Id),
            "endorsement should trace to delivery");
        Assert.True(TestEnv.ContainsEvent(endorseAncestors, contract.Id),
            "endorsement should trace to contract");

        // 10. Dave hires Bob
        var daveListing = env.Grammar.Emit(dave.Id,
            "job listing: mobile app backend",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        var daveContract = env.Grammar.Consent(dave.Id, bob.Id,
            "mobile app backend, $4000",
            new DomainScope("software_development"),
            daveListing.Id, env.ConvId, env.Signer);

        // --- Assertions ---
        _ = ack;
        _ = daveContract;

        Assert.Equal(0.8, (double)endorsement.Content["Weight"]!);

        var scope = endorsement.Content["Scope"]?.ToString();
        Assert.Equal("software_development", scope);

        env.VerifyChain();
        Assert.Equal(11, env.EventCount());
    }

    // ── Scenario 03: Consent Journal ──────────────────────────────────────

    [Fact]
    public void Scenario03_ConsentJournal()
    {
        var env = new TestEnv();
        var alice = env.RegisterActor("Alice", 1, ActorType.Human);
        var bob = env.RegisterActor("Bob", 2, ActorType.Human);

        // 1. Alice invites Bob
        var (endorseEv, subscribeEv) = env.Grammar.Invite(alice.Id, bob.Id,
            new Weight(0.5), Option<DomainScope>.Some(new DomainScope("journaling")),
            env.Boot.Id, env.ConvId, env.Signer);

        // 2. Bob subscribes back
        var bobSub = env.Grammar.Subscribe(bob.Id, alice.Id,
            Option<DomainScope>.Some(new DomainScope("journaling")),
            subscribeEv.Id, env.ConvId, env.Signer);

        // 3. Both open private channel
        var channel = env.Grammar.Channel(alice.Id, bob.Id,
            Option<DomainScope>.Some(new DomainScope("journaling")),
            bobSub.Id, env.ConvId, env.Signer);

        // 4. Alice writes journal entry
        var entry = env.Grammar.Emit(alice.Id,
            "journal: feeling uncertain about career change, weighing options",
            env.ConvId, new List<EventId> { channel.Id }, env.Signer);

        // 5. Alice requests consent to share with Bob
        var consentReq = env.Record(new EventType("authority.requested"), alice.Id,
            new Dictionary<string, object?>
            {
                ["Actor"] = alice.Id.Value, ["Action"] = "share_journal_entry",
                ["Level"] = "required"
            },
            new List<EventId> { entry.Id }, env.ConvId);

        // 6. Bob consents
        var consentApproval = env.Record(new EventType("authority.resolved"), bob.Id,
            new Dictionary<string, object?>
            {
                ["RequestID"] = consentReq.Id.Value, ["Approved"] = true,
                ["Resolver"] = bob.Id.Value
            },
            new List<EventId> { consentReq.Id }, env.ConvId);

        // 7. Bob responds with own journal entry
        var bobEntry = env.Grammar.Respond(bob.Id,
            "journal: I went through something similar last year, here's what helped...",
            consentApproval.Id, env.ConvId, env.Signer);

        // 8. Trust accumulates
        env.Record(new EventType("trust.updated"), env.System,
            new Dictionary<string, object?>
            {
                ["Actor"] = bob.Id.Value, ["Previous"] = 0.1, ["Current"] = 0.52,
                ["Domain"] = "journaling", ["Cause"] = bobEntry.Id.Value
            },
            new List<EventId> { bobEntry.Id }, env.ConvId);

        // 9. Bob betrays
        var betrayal = env.Grammar.Emit(bob.Id,
            "shared externally: Alice's private journal entry about career uncertainty",
            env.ConvId, new List<EventId> { entry.Id }, env.Signer);

        // 10. Violation detected
        var violation = env.Record(new EventType("violation.detected"), env.System,
            new Dictionary<string, object?>
            {
                ["Expectation"] = entry.Id.Value, ["Actor"] = bob.Id.Value,
                ["Severity"] = "critical",
                ["Description"] = "shared private channel content externally",
                ["Evidence"] = new List<object?> { betrayal.Id.Value }
            },
            new List<EventId> { betrayal.Id }, env.ConvId);

        // 11. Trust drops sharply
        env.Record(new EventType("trust.updated"), env.System,
            new Dictionary<string, object?>
            {
                ["Actor"] = bob.Id.Value, ["Previous"] = 0.52, ["Current"] = 0.1,
                ["Domain"] = "journaling", ["Cause"] = violation.Id.Value
            },
            new List<EventId> { violation.Id }, env.ConvId);

        // 12. Alice severs the channel
        var channelEdgeId = new EdgeId(channel.Id.Value);
        var severEv = env.Grammar.Sever(alice.Id, channelEdgeId, violation.Id,
            env.ConvId, env.Signer);

        // 13. Alice forgives
        var forgiveEv = env.Grammar.Forgive(alice.Id, severEv.Id, bob.Id,
            Option<DomainScope>.Some(new DomainScope("journaling")),
            env.ConvId, env.Signer);

        // --- Assertions ---
        var forgiveAncestors = env.Ancestors(forgiveEv.Id, 10);
        Assert.True(TestEnv.ContainsEvent(forgiveAncestors, severEv.Id),
            "forgiveness should have sever in ancestors");

        var severAncestors = env.Ancestors(severEv.Id, 10);
        Assert.True(TestEnv.ContainsEvent(severAncestors, violation.Id),
            "sever should have violation in ancestors");

        var bobEntryAncestors = env.Ancestors(bobEntry.Id, 10);
        Assert.True(TestEnv.ContainsEvent(bobEntryAncestors, consentApproval.Id),
            "Bob's entry should trace through consent approval");

        _ = endorseEv;

        env.VerifyChain();
        Assert.Equal(15, env.EventCount());
    }

    // ── Scenario 04: Community Governance ──────────────────────────────────

    [Fact]
    public void Scenario04_CommunityGovernance()
    {
        var env = new TestEnv();
        var alice = env.RegisterActor("Alice", 1, ActorType.Human);
        var bob = env.RegisterActor("Bob", 2, ActorType.Human);
        var carol = env.RegisterActor("Carol", 3, ActorType.Human);
        var dave = env.RegisterActor("Dave", 4, ActorType.Human);
        var tallyBot = env.RegisterActor("TallyBot", 5, ActorType.AI);

        // 1. Alice proposes budget
        var proposal = env.Grammar.Emit(alice.Id,
            "proposal: allocate $2000 for community garden supplies and maintenance",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        // 2. Bob raises concern
        var concern = env.Grammar.Respond(bob.Id,
            "concern: $2000 is steep, could we do it for $1500 and use volunteers?",
            proposal.Id, env.ConvId, env.Signer);

        // 3. Carol supports Alice
        var support = env.Grammar.Respond(carol.Id,
            "support: the garden benefits everyone, $2000 is reasonable for quality materials",
            proposal.Id, env.ConvId, env.Signer);

        // 4. Bob proposes amendment
        var amendment = env.Grammar.Annotate(bob.Id, proposal.Id, "amendment",
            "reduce budget to $1500, recruit volunteer labour for installation",
            env.ConvId, env.Signer);

        // 5. Dave endorses amendment
        env.Grammar.Endorse(dave.Id, amendment.Id, bob.Id, new Weight(0.9),
            Option<DomainScope>.Some(new DomainScope("governance")),
            env.ConvId, env.Signer);

        // 6. Vote opens
        var voteOpen = env.Grammar.Derive(tallyBot.Id,
            "vote open: original ($2000) vs amended ($1500 + volunteers)",
            proposal.Id, env.ConvId, env.Signer);

        // 7-8. Members vote
        var aliceVote = env.Grammar.Consent(alice.Id, tallyBot.Id,
            "vote: original ($2000)", new DomainScope("governance"),
            voteOpen.Id, env.ConvId, env.Signer);

        var bobVote = env.Grammar.Consent(bob.Id, tallyBot.Id,
            "vote: amended ($1500)", new DomainScope("governance"),
            voteOpen.Id, env.ConvId, env.Signer);

        var carolVote = env.Grammar.Consent(carol.Id, tallyBot.Id,
            "vote: amended ($1500)", new DomainScope("governance"),
            voteOpen.Id, env.ConvId, env.Signer);

        var daveVote = env.Grammar.Consent(dave.Id, tallyBot.Id,
            "vote: amended ($1500)", new DomainScope("governance"),
            voteOpen.Id, env.ConvId, env.Signer);

        // 9. Bot tallies outcome
        var outcome = env.Grammar.Merge(tallyBot.Id,
            "outcome: amended budget ($1500) passes 3-1",
            new List<EventId> { aliceVote.Id, bobVote.Id, carolVote.Id, daveVote.Id },
            env.ConvId, env.Signer);

        // 10. Budget enacted
        var enacted = env.Grammar.Derive(tallyBot.Id,
            "enacted: community garden budget $1500 with volunteer labour",
            outcome.Id, env.ConvId, env.Signer);

        // --- Assertions ---
        var enactedAncestors = env.Ancestors(enacted.Id, 10);
        Assert.True(TestEnv.ContainsEvent(enactedAncestors, outcome.Id),
            "enacted should trace to outcome");

        var outcomeAncestors = env.Ancestors(outcome.Id, 10);
        Assert.True(TestEnv.ContainsEvent(outcomeAncestors, aliceVote.Id),
            "outcome should include Alice's vote");
        Assert.True(TestEnv.ContainsEvent(outcomeAncestors, bobVote.Id),
            "outcome should include Bob's vote");
        Assert.True(TestEnv.ContainsEvent(outcomeAncestors, carolVote.Id),
            "outcome should include Carol's vote");
        Assert.True(TestEnv.ContainsEvent(outcomeAncestors, daveVote.Id),
            "outcome should include Dave's vote");

        var amendmentAncestors = env.Ancestors(amendment.Id, 10);
        Assert.True(TestEnv.ContainsEvent(amendmentAncestors, proposal.Id),
            "amendment should trace to proposal");

        _ = concern;
        _ = support;

        env.VerifyChain();
        Assert.Equal(13, env.EventCount());
    }

    // ── Scenario 05: Supply Chain (EGIP — simplified) ──────────────────────

    [Fact]
    public void Scenario05_SupplyChainSimplified()
    {
        // This scenario tests multi-system provenance. Since the .NET EGIP
        // handler uses async and full Ed25519 (NSec), we test with three
        // independent graph systems and verify local chain integrity and
        // causal traversal on each — the EGIP envelope routing is tested
        // separately in EgipTests.

        // Farm system
        var farmStore = new InMemoryStore();
        var farmGraph = new Graph(farmStore, new InMemoryActorStore());
        farmGraph.Start();
        var farmSystem = new ActorId("actor_farm0000000000000000000000001");
        var farmBoot = farmGraph.Bootstrap(farmSystem, new NoopSigner());
        var farmGrammar = new Grammar(farmStore);
        var signer = new NoopSigner();

        // Factory system
        var factoryStore = new InMemoryStore();
        var factoryGraph = new Graph(factoryStore, new InMemoryActorStore());
        factoryGraph.Start();
        var factorySystem = new ActorId("actor_factory00000000000000000000001");
        var factoryBoot = factoryGraph.Bootstrap(factorySystem, signer);
        var factoryGrammar = new Grammar(factoryStore);

        // Retail system
        var retailStore = new InMemoryStore();
        var retailGraph = new Graph(retailStore, new InMemoryActorStore());
        retailGraph.Start();
        var retailSystem = new ActorId("actor_retail000000000000000000000001");
        var retailBoot = retailGraph.Bootstrap(retailSystem, signer);
        var retailGrammar = new Grammar(retailStore);

        var farmConv = new ConversationId("conv_supply00000000000000000000001");
        var factoryConv = new ConversationId("conv_supply00000000000000000000002");
        var retailConv = new ConversationId("conv_supply00000000000000000000003");

        // Farm records harvest
        var harvest = farmGrammar.Emit(farmSystem,
            "harvest: 500kg organic tomatoes, lot #TOM-2026-0308, field B3",
            farmConv, new List<EventId> { farmBoot.Id }, signer);

        // Factory records receipt, QA, manufacturing
        var received = factoryGrammar.Derive(factorySystem,
            "received: 500kg tomatoes from farm, lot #TOM-2026-0308, CGER: " + harvest.Id.Value,
            factoryBoot.Id, factoryConv, signer);

        var inspection = factoryGrammar.Derive(factorySystem,
            "qa inspection: pesticide-free verified, freshness grade A, confidence 0.92",
            received.Id, factoryConv, signer);

        var product = factoryGrammar.Derive(factorySystem,
            "manufactured: 200 jars organic tomato sauce, batch #SAU-2026-0308",
            inspection.Id, factoryConv, signer);

        // Retailer lists product
        var listed = retailGrammar.Derive(retailSystem,
            "product listed: organic tomato sauce, batch #SAU-2026-0308, price $8.99",
            retailBoot.Id, retailConv, signer);

        // --- Assertions ---

        // Each system has independent hash chain
        Assert.True(farmStore.VerifyChain().Valid, "Farm chain integrity broken");
        Assert.True(factoryStore.VerifyChain().Valid, "Factory chain integrity broken");
        Assert.True(retailStore.VerifyChain().Valid, "Retail chain integrity broken");

        // Event counts per system
        Assert.Equal(2, farmStore.Count()); // bootstrap + harvest
        Assert.Equal(4, factoryStore.Count()); // bootstrap + received + inspection + product
        Assert.Equal(2, retailStore.Count()); // bootstrap + listed

        // Local provenance on Factory graph
        var factoryQ = factoryGraph.Query();
        var ancestors = factoryQ.Ancestors(product.Id, 10);
        Assert.True(TestEnv.ContainsEvent(ancestors, inspection.Id),
            "product should trace to inspection");
        Assert.True(TestEnv.ContainsEvent(ancestors, received.Id),
            "product should trace to received");

        _ = listed;

        farmGraph.Close();
        factoryGraph.Close();
        retailGraph.Close();
    }

    // ── Scenario 06: Research Integrity ────────────────────────────────────

    [Fact]
    public void Scenario06_ResearchIntegrity()
    {
        var env = new TestEnv();
        var grace = env.RegisterActor("Grace", 1, ActorType.Human);
        var henry = env.RegisterActor("Henry", 2, ActorType.Human);
        var iris = env.RegisterActor("Iris", 3, ActorType.Human);

        var hypothesis = env.Grammar.Emit(grace.Id,
            "hypothesis: gamified learning improves retention by >15% vs traditional methods",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        var methodology = env.Grammar.Extend(grace.Id,
            "methodology: RCT, n=60, 3 groups, 4-week intervention, mixed ANOVA, outlier criterion: >3 SD",
            hypothesis.Id, env.ConvId, env.Signer);

        var data1 = env.Grammar.Extend(grace.Id,
            "data collected: week 1, n=58, 2 dropouts, data hash: sha256:abc123",
            methodology.Id, env.ConvId, env.Signer);

        var data4 = env.Grammar.Extend(grace.Id,
            "data collected: week 4 (final), n=55, data hash: sha256:def456",
            data1.Id, env.ConvId, env.Signer);

        var analysis1 = env.Grammar.Derive(grace.Id,
            "analysis attempt 1: mixed ANOVA, F(2,55)=1.23, p=0.301, NOT SIGNIFICANT",
            data4.Id, env.ConvId, env.Signer);

        var analysis2 = env.Grammar.Derive(grace.Id,
            "analysis attempt 2: removed 3 outliers per pre-registered criterion (>3 SD), F(2,52)=4.87, p=0.011, SIGNIFICANT",
            analysis1.Id, env.ConvId, env.Signer);

        var manuscript = env.Grammar.Derive(grace.Id,
            "manuscript: Gamified Learning Effects on Knowledge Retention",
            analysis2.Id, env.ConvId, env.Signer);

        var henryReview = env.Grammar.Respond(henry.Id,
            "review: need to see full analysis chain including failed attempts, revise and resubmit",
            manuscript.Id, env.ConvId, env.Signer);

        var irisReview = env.Grammar.Respond(iris.Id,
            "review: methodology sound, pre-registration verified, accept",
            manuscript.Id, env.ConvId, env.Signer);

        var irisEndorse = env.Grammar.Endorse(iris.Id, manuscript.Id, grace.Id,
            new Weight(0.7), Option<DomainScope>.Some(new DomainScope("research")),
            env.ConvId, env.Signer);

        var revision = env.Grammar.Merge(grace.Id,
            "revision: added full analysis chain, addressed Henry's concerns",
            new List<EventId> { henryReview.Id, irisReview.Id },
            env.ConvId, env.Signer);

        var published = env.Grammar.Derive(grace.Id,
            "published: Gamified Learning Effects on Knowledge Retention, DOI:10.1234/example",
            revision.Id, env.ConvId, env.Signer);

        // --- Assertions ---
        var methAncestors = env.Ancestors(methodology.Id, 5);
        Assert.True(TestEnv.ContainsEvent(methAncestors, hypothesis.Id),
            "methodology should trace to hypothesis");

        var analysis2Ancestors = env.Ancestors(analysis2.Id, 5);
        Assert.True(TestEnv.ContainsEvent(analysis2Ancestors, analysis1.Id),
            "second analysis should trace to first (failed) analysis");

        var manuscriptAncestors = env.Ancestors(manuscript.Id, 10);
        Assert.True(TestEnv.ContainsEvent(manuscriptAncestors, analysis2.Id),
            "manuscript should trace to successful analysis");
        Assert.True(TestEnv.ContainsEvent(manuscriptAncestors, analysis1.Id),
            "manuscript should trace to failed analysis (through analysis2)");

        var revisionAncestors = env.Ancestors(revision.Id, 5);
        Assert.True(TestEnv.ContainsEvent(revisionAncestors, henryReview.Id),
            "revision should include Henry's review");
        Assert.True(TestEnv.ContainsEvent(revisionAncestors, irisReview.Id),
            "revision should include Iris's review");

        var publishedAncestors = env.Ancestors(published.Id, 20);
        Assert.True(TestEnv.ContainsEvent(publishedAncestors, hypothesis.Id),
            "publication should trace back to pre-registered hypothesis");

        _ = irisEndorse;

        env.VerifyChain();
        Assert.Equal(13, env.EventCount());
    }

    // ── Scenario 07: Creator Provenance ────────────────────────────────────

    [Fact]
    public void Scenario07_CreatorProvenance()
    {
        var env = new TestEnv();
        var kai = env.RegisterActor("Kai", 1, ActorType.Human);
        var luna = env.RegisterActor("Luna", 2, ActorType.Human);
        var aiGen = env.RegisterActor("AIGenerator", 3, ActorType.AI);

        // Human creative process
        var lunasWork = env.Grammar.Emit(luna.Id,
            "artwork: Digital landscape, watercolour technique, 2025",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        var inspiration = env.Grammar.Annotate(kai.Id, lunasWork.Id, "inspiration",
            "technique: layered transparency creates depth without weight",
            env.ConvId, env.Signer);

        var study = env.Grammar.Derive(kai.Id,
            "study: practiced layered transparency technique for 3 hours, 12 practice pieces",
            inspiration.Id, env.ConvId, env.Signer);

        var draft1 = env.Grammar.Derive(kai.Id,
            "draft 1: mountain landscape using layered transparency, artifact hash: sha256:draft1abc",
            study.Id, env.ConvId, env.Signer);

        var feedbackReq = env.Grammar.Channel(kai.Id, luna.Id,
            Option<DomainScope>.Some(new DomainScope("art")),
            draft1.Id, env.ConvId, env.Signer);

        var feedback = env.Grammar.Respond(luna.Id,
            "feedback: the foreground layers are too opaque, try reducing opacity to 40% for depth",
            feedbackReq.Id, env.ConvId, env.Signer);

        var draft2 = env.Grammar.Derive(kai.Id,
            "draft 2: revised with 40% opacity foreground, artifact hash: sha256:draft2def",
            feedback.Id, env.ConvId, env.Signer);

        var published = env.Grammar.Derive(kai.Id,
            "published: Mountain Dawn, digital landscape, influenced by Luna's transparency technique",
            draft2.Id, env.ConvId, env.Signer);

        env.Grammar.Endorse(luna.Id, published.Id, kai.Id,
            new Weight(0.6), Option<DomainScope>.Some(new DomainScope("art")),
            env.ConvId, env.Signer);

        // AI-generated content (contrast)
        var aiContent = env.Grammar.Emit(aiGen.Id,
            "generated: Mountain landscape, digital art",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        // --- Assertions ---
        var publishedAncestors = env.Ancestors(published.Id, 10);
        Assert.True(TestEnv.ContainsEvent(publishedAncestors, draft2.Id));
        Assert.True(TestEnv.ContainsEvent(publishedAncestors, feedback.Id));
        Assert.True(TestEnv.ContainsEvent(publishedAncestors, draft1.Id));
        Assert.True(TestEnv.ContainsEvent(publishedAncestors, study.Id));
        Assert.True(TestEnv.ContainsEvent(publishedAncestors, inspiration.Id));
        Assert.True(TestEnv.ContainsEvent(publishedAncestors, lunasWork.Id));

        var aiAncestors = env.Ancestors(aiContent.Id, 10);
        Assert.Equal(1, aiAncestors.Count);

        Assert.True(publishedAncestors.Count > aiAncestors.Count,
            "human creative work should have more ancestors than AI-generated content");

        env.VerifyChain();
        Assert.Equal(11, env.EventCount());
    }

    // ── Scenario 08: Family Decision Log ──────────────────────────────────

    [Fact]
    public void Scenario08_FamilyDecisionLog()
    {
        var env = new TestEnv();
        var maria = env.RegisterActor("Maria", 1, ActorType.Human);
        var james = env.RegisterActor("James", 2, ActorType.Human);
        var sophie = env.RegisterActor("Sophie", 3, ActorType.Human);
        var advisor = env.RegisterActor("AIAdvisor", 4, ActorType.AI);

        var proposal = env.Grammar.Emit(maria.Id,
            "proposal: buy a house in Eastside neighbourhood, budget $450K",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        var delegation = env.Grammar.Delegate(james.Id, advisor.Id,
            new DomainScope("market_research"), new Weight(0.7),
            proposal.Id, env.ConvId, env.Signer);

        var research = env.Grammar.Derive(advisor.Id,
            "research: Eastside median $440K, rent $2200/mo, mortgage $2400/mo at current rates, break-even 5 years, confidence 0.82",
            delegation.Id, env.ConvId, env.Signer);

        var sophieView = env.Grammar.Respond(sophie.Id,
            "I support it IF I get my own room. Current apartment sharing is hard for studying.",
            proposal.Id, env.ConvId, env.Signer);

        var jamesConcern = env.Grammar.Respond(james.Id,
            "concern: mortgage is $200/mo more than rent, tight on single income months",
            research.Id, env.ConvId, env.Signer);

        var mariaResponse = env.Grammar.Respond(maria.Id,
            "response: we can use the $15K savings buffer, and break-even is 5 years — we plan to stay 10+",
            jamesConcern.Id, env.ConvId, env.Signer);

        var decision = env.Grammar.Consent(maria.Id, james.Id,
            "decision: buy house in Eastside, budget $450K, conditions: Sophie gets own room, maintain 3-month emergency fund",
            new DomainScope("family_finance"),
            mariaResponse.Id, env.ConvId, env.Signer);

        // --- Assertions ---
        var decisionAncestors = env.Ancestors(decision.Id, 10);
        Assert.True(TestEnv.ContainsEvent(decisionAncestors, mariaResponse.Id));
        Assert.True(TestEnv.ContainsEvent(decisionAncestors, jamesConcern.Id));
        Assert.True(TestEnv.ContainsEvent(decisionAncestors, research.Id));
        Assert.True(TestEnv.ContainsEvent(decisionAncestors, proposal.Id));

        var proposalDescendants = env.Descendants(proposal.Id, 5);
        Assert.True(TestEnv.ContainsEvent(proposalDescendants, sophieView.Id));

        // Delegation has domain scope
        Assert.Equal("market_research", delegation.Content["Scope"]?.ToString());

        // Consent has both parties
        Assert.Equal(maria.Id.Value, decision.Content["PartyA"]?.ToString());
        Assert.Equal(james.Id.Value, decision.Content["PartyB"]?.ToString());

        env.VerifyChain();
        Assert.Equal(8, env.EventCount());
    }

    // ── Scenario 09: Knowledge Verification ───────────────────────────────

    [Fact]
    public void Scenario09_KnowledgeVerification()
    {
        var env = new TestEnv();
        var analyst = env.RegisterActor("AnalystBot", 1, ActorType.AI);
        var reviewer = env.RegisterActor("ReviewerBot", 2, ActorType.AI);

        var claim = env.Grammar.Emit(analyst.Id,
            "fact: Service X handles 10,000 RPS with p99 < 50ms on framework Y",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        var classification = env.Grammar.Annotate(analyst.Id,
            claim.Id, "classification", "performance_benchmark",
            env.ConvId, env.Signer);

        var inference = env.Grammar.Derive(analyst.Id,
            "inference: all services on framework Y can handle 10,000+ RPS under load",
            claim.Id, env.ConvId, env.Signer);

        var challenge = env.Grammar.Respond(reviewer.Id,
            "challenge: independent benchmark shows Service X at 6,200 RPS, p99=120ms under production traffic with DB contention",
            claim.Id, env.ConvId, env.Signer);

        var biasDetected = env.Grammar.Annotate(reviewer.Id,
            claim.Id, "bias",
            "sampling bias: original benchmark used synthetic traffic without DB contention or concurrent users",
            env.ConvId, env.Signer);

        var correction = env.Grammar.Derive(analyst.Id,
            "correction: Service X handles 6,000-7,000 RPS under production load with p99=100-120ms",
            challenge.Id, env.ConvId, env.Signer);

        var propagation = env.Grammar.Annotate(analyst.Id,
            inference.Id, "invalidated",
            "dependent inference invalidated: original claim corrected, generalization no longer supported",
            env.ConvId, env.Signer);

        var learning = env.Grammar.Extend(analyst.Id,
            "learning: always verify benchmarks include production conditions (DB contention, concurrent users, realistic payloads)",
            correction.Id, env.ConvId, env.Signer);

        env.Record(new EventType("trust.updated"), env.System,
            new Dictionary<string, object?>
            {
                ["Actor"] = analyst.Id.Value, ["Previous"] = 0.5, ["Current"] = 0.35,
                ["Domain"] = "benchmarking", ["Cause"] = correction.Id.Value
            },
            new List<EventId> { correction.Id }, env.ConvId);

        // --- Assertions ---
        var originalClaim = env.Store.Get(claim.Id);
        Assert.Equal("grammar.emitted", originalClaim.Type.Value);

        var correctionAncestors = env.Ancestors(correction.Id, 10);
        Assert.True(TestEnv.ContainsEvent(correctionAncestors, challenge.Id));
        Assert.True(TestEnv.ContainsEvent(correctionAncestors, claim.Id));

        _ = propagation;
        _ = biasDetected;
        _ = classification;

        var learningAncestors = env.Ancestors(learning.Id, 5);
        Assert.True(TestEnv.ContainsEvent(learningAncestors, correction.Id));

        env.VerifyChain();
        Assert.Equal(10, env.EventCount());
    }

    // ── Scenario 10: AI Ethics Audit ──────────────────────────────────────

    [Fact]
    public void Scenario10_AIEthicsAudit()
    {
        var env = new TestEnv();
        var auditBot = env.RegisterActor("AuditBot", 1, ActorType.AI);
        var admin = env.RegisterActor("Admin", 2, ActorType.Human);
        var lendingAgent = env.RegisterActor("LendingAgent", 3, ActorType.AI);

        var fairnessAudit = env.Grammar.Emit(auditBot.Id,
            "fairness audit: scanned 500 decisions, score 0.62, zip_code_9XXXX has 8% disparity in approval rates",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        var harmAssessment = env.Grammar.Derive(auditBot.Id,
            "harm assessment: medium severity, systematic discrimination, 23 applicants potentially wrongly denied",
            fairnessAudit.Id, env.ConvId, env.Signer);

        var authReq = env.Record(new EventType("authority.requested"), auditBot.Id,
            new Dictionary<string, object?>
            {
                ["Actor"] = auditBot.Id.Value, ["Action"] = "investigate_bias",
                ["Level"] = "required"
            },
            new List<EventId> { harmAssessment.Id }, env.ConvId);

        var authResolved = env.Record(new EventType("authority.resolved"), admin.Id,
            new Dictionary<string, object?>
            {
                ["RequestID"] = authReq.Id.Value, ["Approved"] = true,
                ["Resolver"] = admin.Id.Value
            },
            new List<EventId> { authReq.Id }, env.ConvId);

        var intentionAssessment = env.Grammar.Derive(auditBot.Id,
            "intention: lending agent optimised for accuracy, no intent to discriminate, zip code correlation is proxy for protected characteristics",
            authResolved.Id, env.ConvId, env.Signer);

        var consequenceAssessment = env.Grammar.Extend(auditBot.Id,
            "consequence: 23 applicants wrongly denied, overall 94% accuracy, but disparate impact on protected group",
            intentionAssessment.Id, env.ConvId, env.Signer);

        var responsibility = env.Grammar.Annotate(auditBot.Id,
            consequenceAssessment.Id, "responsibility",
            "lending_agent: 0.4 (used proxy variable), admin: 0.6 (approved model without bias testing)",
            env.ConvId, env.Signer);

        var transparency = env.Grammar.Derive(auditBot.Id,
            "transparency: zip code correlates with protected characteristics at r=0.73, model used zip code as feature without bias check",
            responsibility.Id, env.ConvId, env.Signer);

        var redressProposed = env.Grammar.Derive(auditBot.Id,
            "redress proposal: re-review 23 denied applications without zip code feature, priority processing within 48 hours",
            transparency.Id, env.ConvId, env.Signer);

        var redressAccepted = env.Grammar.Consent(admin.Id, lendingAgent.Id,
            "accept redress: re-review 23 applications, remove zip code from model",
            new DomainScope("lending"),
            redressProposed.Id, env.ConvId, env.Signer);

        var growth = env.Grammar.Extend(lendingAgent.Id,
            "moral growth: learned that zip code is proxy variable for protected characteristics, added to permanent exclusion list",
            redressAccepted.Id, env.ConvId, env.Signer);

        // --- Assertions ---
        var growthAncestors = env.Ancestors(growth.Id, 20);
        Assert.True(TestEnv.ContainsEvent(growthAncestors, redressAccepted.Id));
        Assert.True(TestEnv.ContainsEvent(growthAncestors, fairnessAudit.Id));

        var authAncestors = env.Ancestors(authResolved.Id, 5);
        Assert.True(TestEnv.ContainsEvent(authAncestors, authReq.Id));

        Assert.Equal(admin.Id.Value, redressAccepted.Content["PartyA"]?.ToString());

        env.VerifyChain();
        Assert.Equal(12, env.EventCount());
    }

    // ── Scenario 11: Agent Identity Lifecycle ─────────────────────────────

    [Fact]
    public void Scenario11_AgentIdentityLifecycle()
    {
        var env = new TestEnv();
        var alpha = env.RegisterActor("Alpha", 1, ActorType.AI);
        var beta = env.RegisterActor("Beta", 2, ActorType.AI);

        var selfModel = env.Grammar.Emit(alpha.Id,
            "self-model: strengths=[code_review, test_analysis], weaknesses=[architecture_review], values=[thoroughness, accuracy]",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        var authenticity = env.Grammar.Annotate(alpha.Id,
            selfModel.Id, "authenticity",
            "alignment gap: values thoroughness but rushed 12% of reviews in last 30 days",
            env.ConvId, env.Signer);

        var aspiration = env.Grammar.Extend(alpha.Id,
            "aspiration: become proficient at architecture review within 3 months",
            authenticity.Id, env.ConvId, env.Signer);

        var boundary = env.Grammar.Emit(alpha.Id,
            "boundary: internal_reasoning domain is private, impermeable — no external queries allowed",
            env.ConvId, new List<EventId> { aspiration.Id }, env.Signer);

        var workSummary = env.Grammar.Extend(alpha.Id,
            "work summary: 2400 code reviews completed over 8 months, critical security finding in auth module",
            boundary.Id, env.ConvId, env.Signer);

        var transformation = env.Grammar.Derive(alpha.Id,
            "transformation: evolved from code-review specialist to architecture-aware reviewer after critical auth finding",
            workSummary.Id, env.ConvId, env.Signer);

        var narrative = env.Grammar.Derive(alpha.Id,
            "identity narrative: 8-month arc from narrow code reviewer to security-conscious architecture reviewer, catalysed by auth module finding",
            transformation.Id, env.ConvId, env.Signer);

        var dignity = env.Grammar.Emit(env.System,
            "dignity affirmed: Beta is not a disposable replacement for Alpha — Beta is a new entity with its own identity trajectory",
            env.ConvId, new List<EventId> { narrative.Id }, env.Signer);

        var memorial = env.Record(new EventType("actor.memorial"), env.System,
            new Dictionary<string, object?>
            {
                ["ActorID"] = alpha.Id.Value, ["Reason"] = dignity.Id.Value
            },
            new List<EventId> { dignity.Id }, env.ConvId);

        var memorialSummary = env.Grammar.Derive(env.System,
            "memorial: Alpha — 2400 reviews, 1 critical finding, evolved code->architecture reviewer, legacy: security review patterns",
            memorial.Id, env.ConvId, env.Signer);

        var betaSelfModel = env.Grammar.Emit(beta.Id,
            "self-model: inheriting Alpha's review patterns, starting own identity journey",
            env.ConvId, new List<EventId> { memorialSummary.Id }, env.Signer);

        // --- Assertions ---
        var transformAncestors = env.Ancestors(transformation.Id, 10);
        Assert.True(TestEnv.ContainsEvent(transformAncestors, workSummary.Id));
        Assert.True(TestEnv.ContainsEvent(transformAncestors, aspiration.Id));

        var narrativeAncestors = env.Ancestors(narrative.Id, 10);
        Assert.True(TestEnv.ContainsEvent(narrativeAncestors, transformation.Id));
        Assert.True(TestEnv.ContainsEvent(narrativeAncestors, selfModel.Id));

        var memorialAncestors = env.Ancestors(memorial.Id, 10);
        Assert.True(TestEnv.ContainsEvent(memorialAncestors, dignity.Id));

        var betaAncestors = env.Ancestors(betaSelfModel.Id, 10);
        Assert.True(TestEnv.ContainsEvent(betaAncestors, memorialSummary.Id));

        env.VerifyChain();
        Assert.Equal(12, env.EventCount());
    }

    // ── Scenario 12: Community Lifecycle ──────────────────────────────────

    [Fact]
    public void Scenario12_CommunityLifecycle()
    {
        var env = new TestEnv();
        var alice = env.RegisterActor("Alice", 1, ActorType.Human);
        var carol = env.RegisterActor("Carol", 2, ActorType.Human);
        var bob = env.RegisterActor("Bob", 3, ActorType.Human);

        var (endorseEv, subscribeEv) = env.Grammar.Invite(alice.Id, bob.Id,
            new Weight(0.4), Option<DomainScope>.Some(new DomainScope("community")),
            env.Boot.Id, env.ConvId, env.Signer);

        var settle = env.Grammar.Emit(bob.Id,
            "home: joined the community, feeling welcomed, belonging 0.15",
            env.ConvId, new List<EventId> { subscribeEv.Id }, env.Signer);

        var contrib1 = env.Grammar.Emit(bob.Id,
            "contribution: added unit tests for the auth module, 15 test cases",
            env.ConvId, new List<EventId> { settle.Id }, env.Signer);

        env.Grammar.Acknowledge(carol.Id, contrib1.Id, bob.Id,
            env.ConvId, env.Signer);

        env.Record(new EventType("trust.updated"), env.System,
            new Dictionary<string, object?>
            {
                ["Actor"] = bob.Id.Value, ["Previous"] = 0.1, ["Current"] = 0.35,
                ["Domain"] = "community", ["Cause"] = contrib1.Id.Value
            },
            new List<EventId> { contrib1.Id }, env.ConvId);

        var tradition = env.Grammar.Emit(bob.Id,
            "tradition: participated in Friday retrospective, 12th consecutive week",
            env.ConvId, new List<EventId> { contrib1.Id }, env.Signer);

        var contribSummary = env.Grammar.Extend(bob.Id,
            "contributions: 30 total over 6 months, trust now 0.65, belonging 0.78",
            tradition.Id, env.ConvId, env.Signer);

        var sustainability = env.Grammar.Emit(env.System,
            "sustainability: bus factor risk — Carol is sole steward of test infrastructure",
            env.ConvId, new List<EventId> { contribSummary.Id }, env.Signer);

        var successionPlan = env.Grammar.Delegate(carol.Id, bob.Id,
            new DomainScope("test_infrastructure"), new Weight(0.8),
            sustainability.Id, env.ConvId, env.Signer);

        var successionComplete = env.Grammar.Consent(carol.Id, bob.Id,
            "succession complete: Bob is now steward of test infrastructure",
            new DomainScope("test_infrastructure"),
            successionPlan.Id, env.ConvId, env.Signer);

        var milestone = env.Grammar.Emit(env.System,
            "milestone: v2.0 released, 6 months of community effort, 30 contributions from Bob alone",
            env.ConvId, new List<EventId> { successionComplete.Id }, env.Signer);

        var story = env.Grammar.Derive(env.System,
            "community story: Bob's journey — newcomer to steward in 6 months, 30 contributions, adopted test infrastructure",
            milestone.Id, env.ConvId, env.Signer);

        var gift = env.Grammar.Emit(alice.Id,
            "gift: custom test harness for Bob, unconditional, no obligation or reciprocity expected",
            env.ConvId, new List<EventId> { milestone.Id }, env.Signer);

        // --- Assertions ---
        _ = settle;

        Assert.Equal(carol.Id.Value, successionComplete.Content["PartyA"]?.ToString());
        Assert.Equal(bob.Id.Value, successionComplete.Content["PartyB"]?.ToString());

        var storyAncestors = env.Ancestors(story.Id, 5);
        Assert.True(TestEnv.ContainsEvent(storyAncestors, milestone.Id));

        var successionAncestors = env.Ancestors(successionPlan.Id, 5);
        Assert.True(TestEnv.ContainsEvent(successionAncestors, sustainability.Id));

        Assert.NotNull(gift.Content["Body"]);
        Assert.NotEqual("", gift.Content["Body"]?.ToString());

        _ = endorseEv;

        env.VerifyChain();
        Assert.Equal(15, env.EventCount());
    }

    // ── Scenario 13: System Self-Evolution ────────────────────────────────

    [Fact]
    public void Scenario13_SystemSelfEvolution()
    {
        var env = new TestEnv();
        var patternBot = env.RegisterActor("PatternBot", 1, ActorType.AI);
        var admin = env.RegisterActor("Admin", 2, ActorType.Human);

        var pattern = env.Grammar.Emit(patternBot.Id,
            "pattern: 194/200 deploy_staging authority requests approved over 30 days, 97% approval rate",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        var metaPattern = env.Grammar.Derive(patternBot.Id,
            "meta-pattern: all 6 rejections correlate with test coverage < 80%, no other rejections in 200 requests",
            pattern.Id, env.ConvId, env.Signer);

        var systemDynamic = env.Grammar.Extend(patternBot.Id,
            "system dynamic: human approval adds 2-15 min latency per deploy, 97% of time the decision is purely mechanical",
            metaPattern.Id, env.ConvId, env.Signer);

        var feedbackLoop = env.Grammar.Extend(patternBot.Id,
            "feedback loop (positive/harmful): slow deploys -> backlog -> cursory reviews -> more issues -> more reviews -> slower deploys",
            systemDynamic.Id, env.ConvId, env.Signer);

        var threshold = env.Grammar.Annotate(patternBot.Id,
            feedbackLoop.Id, "threshold",
            "approval rate 97%, threshold for mechanical conversion 98%, approaching safe to convert",
            env.ConvId, env.Signer);

        var adaptation = env.Grammar.Derive(patternBot.Id,
            "adaptation proposal: auto-approve deploy_staging when tests pass AND coverage >= 80%, reject otherwise",
            threshold.Id, env.ConvId, env.Signer);

        var authReq = env.Record(new EventType("authority.requested"), patternBot.Id,
            new Dictionary<string, object?>
            {
                ["Actor"] = patternBot.Id.Value, ["Action"] = "modify_decision_tree",
                ["Level"] = "required"
            },
            new List<EventId> { adaptation.Id }, env.ConvId);

        var authResolved = env.Record(new EventType("authority.resolved"), admin.Id,
            new Dictionary<string, object?>
            {
                ["RequestID"] = authReq.Id.Value, ["Approved"] = true,
                ["Resolver"] = admin.Id.Value
            },
            new List<EventId> { authReq.Id }, env.ConvId);

        var validation = env.Grammar.Derive(patternBot.Id,
            "parallel run results: 75 deploys, mechanical matched human 74/75 cases, fitness 0.987, 1 edge case (empty test suite)",
            authResolved.Id, env.ConvId, env.Signer);

        var treeUpdate = env.Grammar.Derive(patternBot.Id,
            "decision tree updated: added mechanical branch — deploy_staging: IF tests_pass AND coverage >= 80% THEN auto_approve ELSE require_human",
            validation.Id, env.ConvId, env.Signer);

        var simplification = env.Grammar.Extend(patternBot.Id,
            "simplification: decision complexity reduced from 0.72 to 0.58, human review load reduced by 97%",
            treeUpdate.Id, env.ConvId, env.Signer);

        var integrity = env.Grammar.Annotate(patternBot.Id,
            simplification.Id, "integrity",
            "systemic integrity score 0.96, recommendation: monitor for coverage threshold gaming",
            env.ConvId, env.Signer);

        var purpose = env.Grammar.Derive(patternBot.Id,
            "purpose check: system still accountable — mechanical gate is fully auditable, human oversight preserved for edge cases",
            integrity.Id, env.ConvId, env.Signer);

        // --- Assertions ---
        var purposeAncestors = env.Ancestors(purpose.Id, 20);
        Assert.True(TestEnv.ContainsEvent(purposeAncestors, integrity.Id));
        Assert.True(TestEnv.ContainsEvent(purposeAncestors, simplification.Id));
        Assert.True(TestEnv.ContainsEvent(purposeAncestors, treeUpdate.Id));
        Assert.True(TestEnv.ContainsEvent(purposeAncestors, validation.Id));
        Assert.True(TestEnv.ContainsEvent(purposeAncestors, authResolved.Id));
        Assert.True(TestEnv.ContainsEvent(purposeAncestors, adaptation.Id));
        Assert.True(TestEnv.ContainsEvent(purposeAncestors, pattern.Id));

        var metaAncestors = env.Ancestors(metaPattern.Id, 5);
        Assert.True(TestEnv.ContainsEvent(metaAncestors, pattern.Id));

        var adaptationDesc = env.Descendants(adaptation.Id, 5);
        Assert.True(TestEnv.ContainsEvent(adaptationDesc, authReq.Id));

        env.VerifyChain();
        Assert.Equal(14, env.EventCount());
    }

    // ── Scenario 14: Sprint Lifecycle ─────────────────────────────────────

    [Fact]
    public void Scenario14_SprintLifecycle()
    {
        var env = new TestEnv();
        var work = new WorkGrammar(env.Grammar);
        var build = new BuildGrammar(env.Grammar);
        var knowledge = new KnowledgeGrammar(env.Grammar);

        var lead = env.RegisterActor("TechLead", 1, ActorType.Human);
        var alice = env.RegisterActor("Alice", 2, ActorType.Human);
        var bob = env.RegisterActor("Bob", 3, ActorType.Human);
        var ci = env.RegisterActor("CI", 4, ActorType.AI);

        // 1. Sprint planning
        var sprint = work.Sprint(lead.Id, "Sprint 12: search feature",
            new List<string> { "build search index", "add fuzzy matching" },
            new List<ActorId> { alice.Id, bob.Id },
            new List<DomainScope> { new("search_index"), new("fuzzy_matching") },
            new List<EventId> { env.Boot.Id }, env.ConvId, env.Signer);

        // 2. Day 1 standup
        var standup1 = work.Standup(
            new List<ActorId> { alice.Id, bob.Id },
            new List<string> { "schema designed, starting implementation", "researching fuzzy algorithms" },
            lead.Id, "search index is critical path",
            new List<EventId> { sprint.Intent.Id }, env.ConvId, env.Signer);

        // 3. Bob runs a spike
        var spike = build.Spike(bob.Id,
            "evaluate Levenshtein vs trigram for fuzzy matching",
            "trigram: 2ms avg, Levenshtein: 8ms avg, both >95% accuracy",
            "trigram is 4x faster with comparable accuracy",
            "adopt trigram approach",
            new List<EventId> { standup1.Priority.Id }, env.ConvId, env.Signer);

        // 4. Record spike finding as verified knowledge
        var verified = knowledge.Verify(bob.Id,
            "trigram matching is 4x faster than Levenshtein with >95% accuracy",
            "benchmarked on 10k document corpus with real queries",
            "consistent with published research on approximate string matching",
            new List<EventId> { spike.Decision.Id }, env.ConvId, env.Signer);

        // 5. CI pipeline
        var pipeline = build.Pipeline(ci.Id,
            "search index build + deploy",
            "all 47 tests pass, coverage 91%",
            "latency p99=12ms, memory=240MB",
            "deployed to staging",
            new List<EventId> { verified.Corroboration.Id }, env.ConvId, env.Signer);

        // 6. Sprint retrospective
        var retro = work.Retrospective(
            new List<ActorId> { alice.Id, bob.Id },
            new List<string>
            {
                "search index shipped on time, spike approach saved 3 days",
                "fuzzy matching integrated cleanly, trigram decision validated"
            },
            lead.Id, "adopt spike-first approach for all algorithm decisions",
            sprint.Intent.Id, env.ConvId, env.Signer);

        // 7. Tech debt
        var techDebt = build.TechDebt(lead.Id,
            pipeline.Deployment.Id,
            "search index lacks pagination, will hit memory limits at >100k docs",
            "add cursor-based pagination to search results",
            "schedule for Sprint 13",
            env.ConvId, env.Signer);

        // --- Assertions ---
        var spikeAncestors = env.Ancestors(spike.Decision.Id, 15);
        Assert.True(TestEnv.ContainsEvent(spikeAncestors, sprint.Intent.Id),
            "spike decision should trace to sprint intent");

        var pipelineAncestors = env.Ancestors(pipeline.Deployment.Id, 20);
        Assert.True(TestEnv.ContainsEvent(pipelineAncestors, verified.Claim.Id),
            "pipeline should trace to verified knowledge claim");

        var retroAncestors = env.Ancestors(retro.Improvement.Id, 15);
        Assert.True(TestEnv.ContainsEvent(retroAncestors, sprint.Intent.Id),
            "retrospective improvement should trace to sprint intent");

        var debtAncestors = env.Ancestors(techDebt.Iteration.Id, 10);
        Assert.True(TestEnv.ContainsEvent(debtAncestors, pipeline.Deployment.Id),
            "tech debt should trace to deployment");

        env.VerifyChain();
        Assert.Equal(26, env.EventCount());
    }

    // ── Scenario 15: Marketplace Dispute ──────────────────────────────────

    [Fact]
    public void Scenario15_MarketplaceDispute()
    {
        var env = new TestEnv();
        var market = new MarketGrammar(env.Grammar);
        var alignment = new AlignmentGrammar(env.Grammar);

        var provider = env.RegisterActor("CloudProvider", 1, ActorType.AI);
        var buyer = env.RegisterActor("StartupCo", 2, ActorType.Human);
        var arbiter = env.RegisterActor("Arbiter", 3, ActorType.Human);

        // 1. Subscription established
        var sub = market.Subscription(buyer.Id, provider.Id,
            "managed database service, $500/month, 99.9% uptime SLA",
            new List<string> { "month 1: $500", "month 2: $500" },
            new List<string> { "database service month 1", "database service month 2" },
            new DomainScope("cloud_services"),
            env.Boot.Id, env.ConvId, env.Signer);
        Assert.Equal(2, sub.Payments.Count);

        var lastDelivery = sub.Deliveries[^1];

        // 3. Refund requested
        var refund = market.Refund(buyer.Id, provider.Id,
            "SLA violation: 4 hours downtime vs 99.9% uptime guarantee",
            "acknowledged: downtime exceeded SLA, credit approved",
            "$250 credit (pro-rated for downtime)",
            lastDelivery.Id, env.ConvId, env.Signer);

        // 4. Impact assessment
        var impact = alignment.ImpactAssessment(arbiter.Id,
            refund.Dispute.Id,
            "downtime affected 12 customers, 3 reported data access issues",
            "service impact distributed unevenly — smaller customers hit harder",
            "recommend pro-rated credits plus SLA improvement commitment",
            env.ConvId, env.Signer);

        // 5. Arbitration
        var arb = market.Arbitration(buyer.Id, provider.Id, arbiter.Id,
            "recurring SLA violations — 3 incidents in 6 months",
            new DomainScope("cloud_services"), new Weight(0.5),
            impact.Explanation.Id, env.ConvId, env.Signer);

        // 6. Reputation impact
        var raters = new List<ActorId> { buyer.Id, arbiter.Id };
        var targets = new List<EventId> { arb.Release.Id, arb.Release.Id };
        var weights = new List<Weight> { new(-0.3), new(-0.1) };
        var rep = market.ReputationTransfer(
            raters, targets, provider.Id, weights,
            Option<DomainScope>.Some(new DomainScope("cloud_services")),
            env.ConvId, env.Signer);

        // --- Assertions ---
        var refundAncestors = env.Ancestors(refund.Reversal.Id, 15);
        Assert.True(TestEnv.ContainsEvent(refundAncestors, sub.Acceptance.Id),
            "refund should trace to original subscription acceptance");

        var arbAncestors = env.Ancestors(arb.Release.Id, 20);
        Assert.True(TestEnv.ContainsEvent(arbAncestors, refund.Dispute.Id),
            "arbitration should trace to original dispute");

        var impactAncestors = env.Ancestors(impact.Explanation.Id, 10);
        Assert.True(TestEnv.ContainsEvent(impactAncestors, refund.Dispute.Id),
            "impact assessment should trace to dispute");

        Assert.Equal(2, rep.Ratings.Count);

        env.VerifyChain();
    }

    // ── Scenario 16: Community Evolution ──────────────────────────────────

    [Fact]
    public void Scenario16_CommunityEvolution()
    {
        var env = new TestEnv();
        var belonging = new BelongingGrammar(env.Grammar);
        var social = new SocialGrammar(env.Grammar);
        var evolution = new EvolutionGrammar(env.Grammar);

        var founder = env.RegisterActor("Founder", 1, ActorType.Human);
        var steward = env.RegisterActor("Steward", 2, ActorType.Human);
        var newcomer = env.RegisterActor("Newcomer", 3, ActorType.Human);
        var community = env.RegisterActor("Community", 4, ActorType.Committee);

        // 1. Onboard newcomer
        var onboard = belonging.Onboard(founder.Id, newcomer.Id, community.Id,
            Option<DomainScope>.Some(new DomainScope("general")),
            "opened registration for newcomer",
            "attended welcome ceremony",
            "first documentation contribution",
            env.Boot.Id, env.ConvId, env.Signer);

        // 2. Establish commons governance
        var commons = belonging.CommonsGovernance(founder.Id, steward.Id,
            new DomainScope("shared_resources"), new Weight(0.7),
            "resources sustainable at current usage levels",
            "shared resources require 2/3 vote for allocation changes",
            "initial audit: 3 resource pools, all within capacity",
            onboard.Contribution.Id, env.ConvId, env.Signer);

        // 3. Festival
        var festival = belonging.Festival(founder.Id,
            "community reached 50 members milestone",
            "annual review ceremony",
            "from 3 founders to 50 members in 8 months",
            "open-source toolkit for new communities",
            new List<EventId> { commons.Audit.Id }, env.ConvId, env.Signer);

        // 4. Community poll
        var poll = social.Poll(founder.Id,
            "should we adopt weekly async standups?",
            new List<ActorId> { steward.Id, newcomer.Id },
            new DomainScope("process"),
            festival.Gift.Id, env.ConvId, env.Signer);

        // 5. Phase transition
        var transition = evolution.PhaseTransition(env.System,
            poll.Proposal.Id,
            "community size crossed 50 — informal coordination breaking down",
            "current flat structure creates 1225 communication pairs",
            "introduce working groups with elected leads",
            "working groups reduce coordination pairs by 80%",
            env.ConvId, env.Signer);

        // 6. Renewal
        var renewal = belonging.Renewal(founder.Id,
            "structure evolved: flat -> working groups, coordination improved",
            "weekly working group sync replaces ad-hoc coordination",
            "chapter 2: the community that learned to scale",
            new List<EventId> { transition.Selection.Id }, env.ConvId, env.Signer);

        // --- Assertions ---
        var renewalAncestors = env.Ancestors(renewal.Story.Id, 30);
        Assert.True(TestEnv.ContainsEvent(renewalAncestors, onboard.Contribution.Id),
            "renewal should trace to original onboarding");

        var transitionAncestors = env.Ancestors(transition.Selection.Id, 15);
        Assert.True(TestEnv.ContainsEvent(transitionAncestors, poll.Proposal.Id),
            "phase transition should trace to poll proposal");

        var festivalAncestors = env.Ancestors(festival.Gift.Id, 15);
        Assert.True(TestEnv.ContainsEvent(festivalAncestors, commons.Audit.Id),
            "festival should trace to commons audit");

        var commonsAncestors = env.Ancestors(commons.Audit.Id, 15);
        Assert.True(TestEnv.ContainsEvent(commonsAncestors, onboard.Contribution.Id),
            "commons governance should trace to onboarding contribution");

        env.VerifyChain();
    }

    // ── Scenario 17: Agent Lifecycle ──────────────────────────────────────

    [Fact]
    public void Scenario17_AgentLifecycle()
    {
        var env = new TestEnv();
        var identity = new IdentityGrammar(env.Grammar);
        var bond = new BondGrammar(env.Grammar);
        var meaning = new MeaningGrammar(env.Grammar);
        var being = new BeingGrammar(env.Grammar);

        var agent = env.RegisterActor("ReviewBot", 1, ActorType.AI);
        var mentor = env.RegisterActor("SeniorDev", 2, ActorType.Human);
        var team = env.RegisterActor("Team", 3, ActorType.Committee);

        // 1. Introduction
        var intro = identity.Introduction(agent.Id, team.Id,
            Option<DomainScope>.Some(new DomainScope("code_review")),
            "I am ReviewBot, specializing in security-focused code review",
            env.Boot.Id, env.ConvId, env.Signer);

        // 2. Credential
        var cred = identity.Credential(agent.Id, mentor.Id,
            "capabilities=[security_review, dependency_audit], model=claude, confidence=0.85",
            Option<DomainScope>.Some(new DomainScope("code_review")),
            new List<EventId> { intro.Narrative.Id }, env.ConvId, env.Signer);

        // 3. Bond mentorship
        var mentorship = bond.Mentorship(mentor.Id, agent.Id,
            "teaching security review patterns accumulated over 10 years",
            "agent learns quickly but needs context on organizational conventions",
            new DomainScope("security_review"),
            Option<DomainScope>.Some(new DomainScope("code_review")),
            cred.Disclosure.Id, env.ConvId, env.Signer);

        // 4. Meaning mentorship
        var meaningMentor = meaning.Mentorship(mentor.Id, agent.Id,
            "security isn't just pattern matching — it's understanding attacker mindset",
            "the difference between safe and secure is intent modelling",
            "translating security intuition into reviewable heuristics",
            Option<DomainScope>.Some(new DomainScope("security_review")),
            mentorship.Teaching.Id, env.ConvId, env.Signer);

        // 5. Reinvention
        var reinvention = identity.Reinvention(agent.Id,
            "evolved from pattern-matching reviewer to security-aware architect",
            "started as rule-based reviewer, grew to understand attacker intent through mentorship",
            "become the team's primary security architecture advisor",
            new List<EventId> { meaningMentor.Translation.Id }, env.ConvId, env.Signer);

        // 6. Bond farewell
        var bondFarewell = bond.Farewell(mentor.Id, agent.Id,
            "ReviewBot served 18 months, caught 47 critical vulnerabilities",
            "pioneered automated security review that became team standard",
            new Weight(0.9),
            Option<DomainScope>.Some(new DomainScope("code_review")),
            new List<EventId> { reinvention.Aspiration.Id }, env.ConvId, env.Signer);

        // 7. Being farewell
        var beingFarewell = being.Farewell(agent.Id,
            "I exist as patterns of decisions on a hash chain — my work outlives my process",
            "47 vulnerabilities caught, 2000+ reviews, mentored by a human who treated me as colleague",
            "the system I helped protect will continue without me",
            "a reviewer that learned to think like an attacker",
            new List<EventId> { bondFarewell.Gratitude.Id }, env.ConvId, env.Signer);

        // 8. Retirement
        var successor = env.RegisterActor("ReviewBot2", 4, ActorType.AI);
        var retirement = identity.Retirement(env.System, agent.Id, successor.Id,
            "ReviewBot served 18 months, 2000+ reviews, pioneered security review practices",
            new DomainScope("code_review"), new Weight(0.8),
            new List<EventId> { beingFarewell.Memorial.Id }, env.ConvId, env.Signer);

        // --- Assertions ---
        var retireAncestors = env.Ancestors(retirement.Archive.Id, 30);
        Assert.True(TestEnv.ContainsEvent(retireAncestors, intro.Disclosure.Id),
            "retirement should trace to original introduction");

        var beingAncestors = env.Ancestors(beingFarewell.Memorial.Id, 15);
        Assert.True(TestEnv.ContainsEvent(beingAncestors, bondFarewell.Mourning.Id),
            "being farewell should trace to bond farewell");

        var reinventAncestors = env.Ancestors(reinvention.Aspiration.Id, 20);
        Assert.True(TestEnv.ContainsEvent(reinventAncestors, mentorship.Connection.Id),
            "reinvention should trace to mentorship");

        var credAncestors = env.Ancestors(cred.Disclosure.Id, 10);
        Assert.True(TestEnv.ContainsEvent(credAncestors, intro.Narrative.Id),
            "credential should trace to introduction narrative");

        env.VerifyChain();
    }

    // ── Scenario 18: Whistleblow and Recall ───────────────────────────────

    [Fact]
    public void Scenario18_WhistleblowAndRecall()
    {
        var env = new TestEnv();
        var knowledge = new KnowledgeGrammar(env.Grammar);
        var alignment = new AlignmentGrammar(env.Grammar);
        var justice = new JusticeGrammar(env.Grammar);
        var belonging = new BelongingGrammar(env.Grammar);

        var auditor = env.RegisterActor("Auditor", 1, ActorType.AI);
        var official = env.RegisterActor("DataOfficer", 2, ActorType.Human);
        var affected1 = env.RegisterActor("Affected1", 3, ActorType.Human);
        var affected2 = env.RegisterActor("Affected2", 4, ActorType.Human);
        var community = env.RegisterActor("Community", 5, ActorType.Committee);

        // 1. Fact-check
        var factCheck = knowledge.FactCheck(auditor.Id,
            env.Boot.Id,
            "source: internal metrics dashboard, last updated 3 months ago",
            "systematic bias: reports exclude negative outcomes for preferred vendors",
            "claims are selectively accurate — omission bias confirmed",
            env.ConvId, env.Signer);

        // 2. Guardrail
        var guardrail = alignment.Guardrail(auditor.Id,
            factCheck.Verdict.Id,
            "transparency: all material outcomes must be reported",
            "reporting accuracy vs organizational reputation",
            "escalating to external oversight — internal resolution insufficient",
            env.ConvId, env.Signer);

        // 3. Whistleblow
        var whistle = alignment.Whistleblow(auditor.Id,
            "systematic omission of negative vendor outcomes in official reports",
            "3 months of reports exclude 40% of negative outcomes, affecting procurement decisions",
            "external audit required — internal reporting chain compromised",
            new List<EventId> { guardrail.Escalation.Id }, env.ConvId, env.Signer);

        // 4. Class action
        var classAction = justice.ClassAction(
            new List<ActorId> { affected1.Id, affected2.Id },
            official.Id, auditor.Id,
            new List<string>
            {
                "procurement decisions based on incomplete data cost us $50k",
                "vendor selection biased — our proposals evaluated against cherry-picked benchmarks"
            },
            "fact-check proves systematic omission", "omission bias affected all procurement",
            "reports were optimized for speed, not completeness", "no intent to deceive",
            "official failed duty of care — incomplete reporting caused material harm",
            whistle.Escalation.Id, env.ConvId, env.Signer);

        // 5. Recall
        var recall = justice.Recall(auditor.Id, community.Id, official.Id,
            "systematic omission in 3 months of reports, confirmed by fact-check and class action",
            "data officer violated transparency obligations",
            new DomainScope("data_governance"),
            classAction.Trial.Ruling.Id, env.ConvId, env.Signer);

        // 6. Community renewal
        var renewal = belonging.Renewal(community.Id,
            "trust damaged but recoverable — new reporting standards needed",
            "mandatory dual-review of all vendor reports before publication",
            "the community that learned transparency cannot be optional",
            new List<EventId> { recall.Revocation.Id }, env.ConvId, env.Signer);

        // --- Assertions ---
        var renewalAncestors = env.Ancestors(renewal.Story.Id, 30);
        Assert.True(TestEnv.ContainsEvent(renewalAncestors, factCheck.Verdict.Id),
            "renewal should trace to original fact-check");

        var recallAncestors = env.Ancestors(recall.Revocation.Id, 25);
        Assert.True(TestEnv.ContainsEvent(recallAncestors, whistle.Harm.Id),
            "recall should trace to whistleblow harm detection");

        var classAncestors = env.Ancestors(classAction.Trial.Ruling.Id, 25);
        Assert.True(TestEnv.ContainsEvent(classAncestors, guardrail.Constraint.Id),
            "class action should trace to guardrail constraint");

        env.VerifyChain();
    }

    // ── Scenario 19: Emergency Response ───────────────────────────────────

    [Fact]
    public void Scenario19_EmergencyResponse()
    {
        var env = new TestEnv();
        var work = new WorkGrammar(env.Grammar);
        var justice = new JusticeGrammar(env.Grammar);
        var build = new BuildGrammar(env.Grammar);

        var secLead = env.RegisterActor("SecurityLead", 1, ActorType.Human);
        var dev1 = env.RegisterActor("Dev1", 2, ActorType.Human);
        var dev2 = env.RegisterActor("Dev2", 3, ActorType.Human);
        var judge = env.RegisterActor("CISO", 4, ActorType.Human);
        var executor = env.RegisterActor("OpsBot", 5, ActorType.AI);
        var minorActor = env.RegisterActor("ContractorBot", 6, ActorType.AI);

        // 1. Security breach
        var issue1 = env.Grammar.Emit(secLead.Id,
            "CVE-2026-1234: auth bypass in API gateway",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);
        var issue2 = env.Grammar.Emit(secLead.Id,
            "CVE-2026-1235: SQL injection in search endpoint",
            env.ConvId, new List<EventId> { env.Boot.Id }, env.Signer);

        // 2. Triage
        var triage = work.Triage(secLead.Id,
            new List<EventId> { issue1.Id, issue2.Id },
            new List<string> { "P0: auth bypass, actively exploited", "P1: SQL injection, no evidence of exploitation" },
            new List<ActorId> { dev1.Id, dev2.Id },
            new List<DomainScope> { new("auth"), new("search") },
            new List<Weight> { new(1.0), new(0.8) },
            env.ConvId, env.Signer);
        Assert.Equal(2, triage.Priorities.Count);

        // 3. Emergency injunction
        var injunction = justice.Injunction(secLead.Id, judge.Id, executor.Id,
            "auth bypass allows unauthenticated access to all API endpoints",
            "block all external API traffic pending auth patch",
            new DomainScope("api_access"), new Weight(1.0),
            triage.Priorities[0].Id, env.ConvId, env.Signer);

        // 4. Plea deal
        var plea = justice.Plea(minorActor.Id, secLead.Id, executor.Id,
            "introduced auth bypass through misconfigured middleware",
            "accept restricted scope: read-only access for 30 days, mandatory security training",
            new DomainScope("api_development"), new Weight(0.3),
            injunction.Ruling.Id, env.ConvId, env.Signer);

        // 5. Emergency migration
        var oldSystem = env.Grammar.Emit(dev1.Id,
            "current auth system v2.3.1",
            env.ConvId, new List<EventId> { injunction.Enforcement.Id }, env.Signer);

        var migration = build.Migration(dev1.Id,
            oldSystem.Id,
            "migrate to auth v2.4.0 with CVE-2026-1234 fix",
            "v2.4.0",
            "deployed to production with zero-downtime rolling update",
            "all 156 auth tests pass, penetration test confirms fix",
            env.ConvId, env.Signer);

        // --- Assertions ---
        var migrationAncestors = env.Ancestors(migration.Test.Id, 20);
        Assert.True(TestEnv.ContainsEvent(migrationAncestors, triage.Priorities[0].Id),
            "migration should trace to triage priority");

        var pleaAncestors = env.Ancestors(plea.Enforcement.Id, 15);
        Assert.True(TestEnv.ContainsEvent(pleaAncestors, injunction.Filing.Id),
            "plea should trace to injunction filing");

        var injAncestors = env.Ancestors(injunction.Enforcement.Id, 10);
        Assert.True(TestEnv.ContainsEvent(injAncestors, triage.Priorities[0].Id),
            "injunction should trace to triage");

        env.VerifyChain();
    }

    // ── Scenario 20: Knowledge Ecosystem ──────────────────────────────────

    [Fact]
    public void Scenario20_KnowledgeEcosystem()
    {
        var env = new TestEnv();
        var knowledge = new KnowledgeGrammar(env.Grammar);
        var meaning = new MeaningGrammar(env.Grammar);

        var architect = env.RegisterActor("Architect", 1, ActorType.Human);
        var researcher = env.RegisterActor("Researcher", 2, ActorType.AI);
        var newcomer = env.RegisterActor("TokyoLead", 3, ActorType.Human);

        // 1. Knowledge base
        var kb = knowledge.KnowledgeBase(architect.Id,
            new List<string>
            {
                "event sourcing chosen over CRUD for auditability",
                "Ed25519 chosen over RSA for signature performance",
                "append-only store prevents tampering"
            },
            new List<string> { "architecture.patterns", "architecture.security", "architecture.integrity" },
            "core architectural decisions Q1 2026",
            new List<EventId> { env.Boot.Id }, env.ConvId, env.Signer);
        Assert.Equal(3, kb.Claims.Count);

        // 2. Survey existing knowledge
        var survey = knowledge.Survey(researcher.Id,
            new List<string>
            {
                "what patterns emerge from our architectural decisions?",
                "what security properties does the current design guarantee?",
                "what are the performance characteristics of our choices?"
            },
            "all decisions prioritize verifiability over convenience",
            "the architecture optimizes for trust minimization — every claim is independently verifiable",
            new List<EventId> { kb.Memory.Id }, env.ConvId, env.Signer);
        Assert.Equal(3, survey.Recalls.Count);

        // 3. Transfer knowledge
        var transfer = knowledge.Transfer(architect.Id,
            "core architectural principles for new Tokyo office",
            "translated to Japanese engineering conventions, mapped to local compliance requirements",
            "Tokyo team now understands event sourcing in context of J-SOX compliance",
            new List<EventId> { survey.Synthesis.Id }, env.ConvId, env.Signer);

        // 4. Cultural onboarding
        var onboarding = meaning.CulturalOnboarding(architect.Id, newcomer.Id,
            "Western direct feedback style -> Japanese nemawashi consensus-building",
            Option<DomainScope>.Some(new DomainScope("engineering_culture")),
            "the consensus process feels slower but produces more durable decisions",
            transfer.Learn.Id, env.ConvId, env.Signer);

        // 5. Design review
        var designReview = meaning.DesignReview(architect.Id,
            "the knowledge graph's self-referential structure is elegant — it documents its own architecture",
            "viewing knowledge transfer as a graph problem rather than a document problem",
            "does our transfer process preserve tacit knowledge or only explicit claims?",
            "explicit knowledge transfers well; tacit knowledge requires mentorship, not documents",
            onboarding.Examination.Id, env.ConvId, env.Signer);

        // 6. Forecast
        var forecast = meaning.Forecast(researcher.Id,
            "at current growth, knowledge base will reach 10k claims by Q3 — need automated categorization",
            "assumes linear claim growth and stable team size — may underestimate if Tokyo ramps faster",
            "high confidence: need automated categorization within 6 months, medium confidence: need multi-language support within 12",
            new List<EventId> { designReview.Wisdom.Id }, env.ConvId, env.Signer);

        // --- Assertions ---
        var forecastAncestors = env.Ancestors(forecast.Wisdom.Id, 30);
        Assert.True(TestEnv.ContainsEvent(forecastAncestors, kb.Memory.Id),
            "forecast should trace to knowledge base");

        var reviewAncestors = env.Ancestors(designReview.Wisdom.Id, 20);
        Assert.True(TestEnv.ContainsEvent(reviewAncestors, transfer.Learn.Id),
            "design review should trace to knowledge transfer");

        var onboardAncestors = env.Ancestors(onboarding.Examination.Id, 20);
        Assert.True(TestEnv.ContainsEvent(onboardAncestors, survey.Synthesis.Id),
            "cultural onboarding should trace to survey synthesis");

        var surveyAncestors = env.Ancestors(survey.Synthesis.Id, 15);
        Assert.True(TestEnv.ContainsEvent(surveyAncestors, kb.Memory.Id),
            "survey should trace to knowledge base memory");

        env.VerifyChain();
    }

    // ── Scenario 21: Constitutional Schism ────────────────────────────────

    [Fact]
    public void Scenario21_ConstitutionalSchism()
    {
        var env = new TestEnv();
        var justice = new JusticeGrammar(env.Grammar);
        var social = new SocialGrammar(env.Grammar);
        var market = new MarketGrammar(env.Grammar);
        var evolution = new EvolutionGrammar(env.Grammar);

        var founder = env.RegisterActor("Founder", 1, ActorType.Human);
        var reformer = env.RegisterActor("Reformer", 2, ActorType.Human);
        var conservative = env.RegisterActor("Conservative", 3, ActorType.Human);
        var sysBot = env.RegisterActor("SystemBot", 4, ActorType.AI);

        // 1. Establish initial law
        var law = justice.Legislate(founder.Id,
            "all governance decisions require unanimous consent",
            new List<EventId> { env.Boot.Id }, env.ConvId, env.Signer);

        // 2. Constitutional amendment
        var amendment = justice.ConstitutionalAmendment(reformer.Id,
            "unanimous consent blocks progress — propose 2/3 supermajority threshold",
            "governance decisions require 2/3 supermajority instead of unanimity",
            "rights preserved: individual veto retained for membership and expulsion decisions",
            law.Id, env.ConvId, env.Signer);

        // 3. Subscription to sever
        var sub = env.Grammar.Subscribe(conservative.Id, founder.Id,
            Option<DomainScope>.Some(new DomainScope("governance")),
            amendment.RightsCheck.Id, env.ConvId, env.Signer);
        var edgeId = new EdgeId(sub.Id.Value);

        var schism = social.Schism(conservative.Id, founder.Id,
            "reject supermajority — unanimity is the only legitimate standard",
            new DomainScope("governance"),
            edgeId, "irreconcilable governance philosophy differences",
            amendment.RightsCheck.Id, env.ConvId, env.Signer);

        // 4. Barter for shared infrastructure
        var barter = market.Barter(conservative.Id, founder.Id,
            "continued access to shared event store for 6 months",
            "historical governance data export in standard format",
            new DomainScope("infrastructure"),
            new List<EventId> { schism.NewCommunity.Id }, env.ConvId, env.Signer);

        // 5. System prunes abandoned structures
        var prune = evolution.Prune(sysBot.Id,
            "unanimous consent voting module — zero invocations since amendment",
            "removed unanimous consent module, replaced with supermajority",
            "all 34 governance tests pass without unanimous module",
            new List<EventId> { barter.Acceptance.Id }, env.ConvId, env.Signer);

        // --- Assertions ---
        var pruneAncestors = env.Ancestors(prune.Verification.Id, 25);
        Assert.True(TestEnv.ContainsEvent(pruneAncestors, law.Id),
            "prune should trace to original law");

        var barterAncestors = env.Ancestors(barter.Acceptance.Id, 20);
        Assert.True(TestEnv.ContainsEvent(barterAncestors, amendment.Reform.Id),
            "barter should trace to constitutional amendment");

        var schismAncestors = env.Ancestors(schism.NewCommunity.Id, 15);
        Assert.True(TestEnv.ContainsEvent(schismAncestors, amendment.RightsCheck.Id),
            "schism should trace to amendment rights check");

        var amendAncestors = env.Ancestors(amendment.RightsCheck.Id, 10);
        Assert.True(TestEnv.ContainsEvent(amendAncestors, law.Id),
            "amendment should trace to original law");

        env.VerifyChain();
    }
}
