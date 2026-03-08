// Minimal EventGraph example -- bootstrap a graph, record two events, verify the chain.
//
// To run:
//   1. Create a new console project:  dotnet new console -n MinimalExample
//   2. Add a reference to the EventGraph library:
//      dotnet add reference ../../dotnet/src/EventGraph/EventGraph.csproj
//   3. Replace Program.cs with this file and run:  dotnet run

using EventGraph;

// 1. Create in-memory store, actor store, and graph.
var store = new InMemoryStore();
var actorStore = new InMemoryActorStore();
var graph = new Graph(store, actorStore);
graph.Start();

// 2. Bootstrap the hash chain.
var systemActor = new ActorId("actor_system");
var boot = graph.Bootstrap(systemActor);
Console.WriteLine($"Bootstrap: {boot.Id.Value} (hash: {boot.Hash.Value[..16]}...)");

// 3. Record events using social grammar.
//    Grammar operates directly on the store (IStore).
var gr = new Grammar(store);
var conv = new ConversationId("conv_example");
var alice = new ActorId("actor_alice");

// Emit -- creates independent content with at least one cause.
var ev1 = gr.Emit(alice, "Hello, EventGraph!", conv, new List<EventId> { boot.Id }, new NoopSigner());
Console.WriteLine($"Event 1:   {ev1.Id.Value} (type: {ev1.Type.Value})");

// Derive -- creates causally dependent but independent content.
var ev2 = gr.Derive(alice, "A derived thought", ev1.Id, conv, new NoopSigner());
Console.WriteLine($"Event 2:   {ev2.Id.Value} (type: {ev2.Type.Value})");

// 4. Verify chain integrity.
var result = store.VerifyChain();
Console.WriteLine($"Chain:     {result.Length} events, valid={result.Valid}");

// 5. Clean up.
graph.Close();
