// Social Grammar example -- demonstrates the 15 social grammar operations
// and 4 named functions on an event graph.
//
// To run:
//   1. Create a new console project:  dotnet new console -n SocialExample
//   2. Add a reference to the EventGraph library:
//      dotnet add reference ../../dotnet/src/EventGraph/EventGraph.csproj
//   3. Replace Program.cs with this file and run:  dotnet run

using EventGraph;

// --- Setup ---

var store = new InMemoryStore();
var actorStore = new InMemoryActorStore();
var graph = new Graph(store, actorStore);
graph.Start();

var systemActor = new ActorId("actor_system");
var boot = graph.Bootstrap(systemActor);

// Grammar operates directly on the store.
var gr = new Grammar(store);
var sign = new NoopSigner();
var conv = new ConversationId("conv_social_demo");

var alice = new ActorId("actor_alice");
var bob = new ActorId("actor_bob");

// --- 15 Social Grammar Operations ---

// 1. Emit -- Alice starts a conversation (independent content, requires causes).
var emit = gr.Emit(alice, "I have an idea for a new project", conv,
    new List<EventId> { boot.Id }, sign);
Show("Emit", emit);

// 2. Respond -- Bob replies (causally dependent, subordinate content).
var respond = gr.Respond(bob, "That sounds great, tell me more", emit.Id, conv, sign);
Show("Respond", respond);

// 3. Derive -- Alice builds on her idea (causally dependent, independent content).
var derive = gr.Derive(alice, "Here's the detailed spec", emit.Id, conv, sign);
Show("Derive", derive);

// 4. Extend -- Bob adds to Alice's spec (sequential content from same thread).
var extend = gr.Extend(bob, "Adding performance requirements", derive.Id, conv, sign);
Show("Extend", extend);

// 5. Retract -- Alice retracts her initial emit (only author can retract).
var retract = gr.Retract(alice, emit.Id, "Superseded by the spec", conv, sign);
Show("Retract", retract);

// 6. Annotate -- Alice adds metadata to the spec.
var annotate = gr.Annotate(alice, derive.Id, "priority", "high", conv, sign);
Show("Annotate", annotate);

// 7. Acknowledge -- Bob acknowledges Alice's spec (content-free centripetal edge).
var ack = gr.Acknowledge(bob, derive.Id, alice, conv, sign);
Show("Acknowledge", ack);

// 8. Propagate -- Alice propagates the spec to Bob (centrifugal reference edge).
var propagate = gr.Propagate(alice, derive.Id, bob, conv, sign);
Show("Propagate", propagate);

// 9. Endorse -- Bob endorses Alice's spec (reputation-staked centripetal edge).
var endorse = gr.Endorse(bob, derive.Id, alice,
    new Weight(0.9), Option<DomainScope>.Some(new DomainScope("innovation")), conv, sign);
Show("Endorse", endorse);

// 10. Subscribe -- Bob subscribes to Alice's updates (persistent future-oriented edge).
var subscribe = gr.Subscribe(bob, alice,
    Option<DomainScope>.Some(new DomainScope("projects")), boot.Id, conv, sign);
Show("Subscribe", subscribe);

// 11. Channel -- Alice creates a private channel with Bob (bidirectional edge).
var channel = gr.Channel(alice, bob,
    Option<DomainScope>.Some(new DomainScope("project_alpha")), emit.Id, conv, sign);
Show("Channel", channel);

// 12. Delegate -- Alice delegates a task to Bob (authority-granting centrifugal edge).
var delegateEv = gr.Delegate(alice, bob,
    new DomainScope("engineering"), new Weight(0.8), derive.Id, conv, sign);
Show("Delegate", delegateEv);

// 13. Consent -- Alice and Bob establish mutual consent.
var consent = gr.Consent(alice, bob,
    "Both parties agree to collaborate on project alpha",
    new DomainScope("collaboration"), derive.Id, conv, sign);
Show("Consent", consent);

// 14. Sever -- Alice severs the delegation (only parties can sever).
var delegateEdgeId = new EdgeId(delegateEv.Id.Value);
var sever = gr.Sever(alice, delegateEdgeId, consent.Id, conv, sign);
Show("Sever", sever);

// 15. Merge -- Alice merges her spec with Bob's extension (joins subtrees).
var merge = gr.Merge(alice, "Final spec with all requirements",
    new List<EventId> { derive.Id, extend.Id }, conv, sign);
Show("Merge", merge);

// --- Named Functions (compositions of base operations) ---

Console.WriteLine("\n--- Named Functions ---");

// Challenge = Respond + dispute annotation.
var (challengeResp, challengeFlag) = gr.Challenge(
    bob, "I disagree with this approach", derive.Id, conv, sign);
Show("Challenge (resp)", challengeResp);
Show("Challenge (flag)", challengeFlag);

// Recommend = Propagate + Channel: directed sharing.
var (recProp, recChan) = gr.Recommend(alice, derive.Id, bob, conv, sign);
Show("Recommend (prop)", recProp);
Show("Recommend (chan)", recChan);

// Invite = Endorse + Subscribe: trust-staked introduction.
var (invEndorse, invSubscribe) = gr.Invite(alice, bob,
    new Weight(0.7), Option<DomainScope>.Some(new DomainScope("engineering")),
    derive.Id, conv, sign);
Show("Invite (endorse)", invEndorse);
Show("Invite (sub)", invSubscribe);

// Forgive = Subscribe after Sever: reconciliation with history.
var forgive = gr.Forgive(alice, sever.Id, bob,
    Option<DomainScope>.Some(new DomainScope("engineering")), conv, sign);
Show("Forgive", forgive);

// --- Verify integrity ---

var result = store.VerifyChain();
var count = store.Count();
Console.WriteLine($"\nChain: {count} events, valid={result.Valid}");

graph.Close();

// --- Helper ---

static void Show(string op, Event ev)
{
    Console.WriteLine($"{op,-20} -> {ev.Type.Value} ({ev.Id.Value[..13]}...)");
}
