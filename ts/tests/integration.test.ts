/**
 * Integration test scenarios — ported from Go reference implementation.
 * 21 scenarios exercising the full event graph, grammar, compositions, and EGIP.
 */
import { describe, it, expect, beforeEach } from "vitest";
import { Grammar } from "../src/grammar.js";
import { Graph } from "../src/graph.js";
import { InMemoryStore } from "../src/store.js";
import { InMemoryActorStore, ActorType, type Actor } from "../src/actor.js";
import { createBootstrap, NoopSigner, type Event, type Signer } from "../src/event.js";
import {
  ActorId, ConversationId, DomainScope, EdgeId, EventId, EventType,
  Hash, Option, PublicKey, Score, Signature, Weight,
  EnvelopeId, SystemUri, TreatyId,
} from "../src/types.js";
import {
  WorkGrammar, MarketGrammar, SocialGrammar, JusticeGrammar,
  BuildGrammar, KnowledgeGrammar, AlignmentGrammar, IdentityGrammar,
  BondGrammar, BelongingGrammar, MeaningGrammar, EvolutionGrammar,
  BeingGrammar,
} from "../src/compositions.js";
import {
  SystemIdentity, Handler, PeerStore, TreatyStore,
  Envelope, Treaty, signEnvelope, verifyEnvelope,
  CurrentProtocolVersion, MessageType, TreatyAction, TreatyStatus,
  ReceiptStatus, ProofType, CGERRelationship,
  type MessagePayloadContent, type TreatyTerm, type CGER, type ReceiptPayload,
  type ITransport, type IncomingEnvelope, type IIdentity,
  helloPayload, messagePayload, treatyPayload, proofPayload,
  type ChainSummaryProof,
} from "../src/egip.js";

// ── Test helpers ──────────────────────────────────────────────────────

const signer = new NoopSigner();

function testPublicKey(b: number): PublicKey {
  const key = new Uint8Array(32);
  key[0] = b;
  return new PublicKey(key);
}

interface TestEnv {
  graph: Graph;
  grammar: Grammar;
  store: InMemoryStore;
  actors: InMemoryActorStore;
  boot: Event;
  convId: ConversationId;
  system: ActorId;
  registerActor: (name: string, pkByte: number, actorType: ActorType) => Actor;
  verifyChain: () => void;
  eventCount: () => number;
  ancestors: (id: EventId, depth: number) => Event[];
  descendants: (id: EventId, depth: number) => Event[];
}

function newTestEnv(): TestEnv {
  const store = new InMemoryStore();
  const actors = new InMemoryActorStore();
  const graph = new Graph(store, actors);
  graph.start();

  const system = new ActorId("actor_system0000000000000000000001");
  const boot = graph.bootstrap(system, signer);
  const convId = new ConversationId("conv_test000000000000000000000001");
  const grammar = new Grammar(store);

  const env: TestEnv = {
    graph, grammar, store, actors, boot, convId, system,
    registerActor(name: string, pkByte: number, actorType: ActorType): Actor {
      return actors.register(testPublicKey(pkByte), name, actorType);
    },
    verifyChain(): void {
      const result = store.verifyChain();
      expect(result.valid).toBe(true);
    },
    eventCount(): number {
      return store.count();
    },
    ancestors(id: EventId, depth: number): Event[] {
      const q = graph.query();
      return q.ancestors(id, depth);
    },
    descendants(id: EventId, depth: number): Event[] {
      const q = graph.query();
      return q.descendants(id, depth);
    },
  };

  return env;
}

function containsEvent(events: Event[], id: EventId): boolean {
  return events.some((ev) => ev.id.value === id.value);
}

function containsEventType(events: Event[], typeName: string): boolean {
  return events.some((ev) => ev.type.value === typeName);
}

// ── Scenario 01: Agent Audit Trail ────────────────────────────────────

describe("Scenario 01: Agent Audit Trail", () => {
  it("exercises AI agent delegation, review, bug discovery, trust changes, and causal traversal", () => {
    const env = newTestEnv();
    const alice = env.registerActor("Alice", 1, ActorType.Human);
    const agent = env.registerActor("ReviewBot", 2, ActorType.AI);

    // 1. Alice submits code for review
    const submission = env.grammar.emit(
      alice.id, "code submission: auth module refactor",
      env.convId, [env.boot.id], signer,
    );

    // 2. Alice delegates code_review authority to agent
    const delegation = env.grammar.delegate(
      alice.id, agent.id, new DomainScope("code_review"),
      new Weight(0.8), submission.id, env.convId, signer,
    );

    // 3. Agent reviews the code
    const review = env.grammar.derive(
      agent.id, "review: LGTM, no issues found, approving PR",
      submission.id, env.convId, signer,
    );

    // 4. Agent approves
    const approval = env.grammar.respond(
      agent.id, "decision: approve PR with confidence 0.85",
      review.id, env.convId, signer,
    );

    // 5. Trust updated after successful review
    const trustUp = env.graph.record(
      new EventType("trust.updated"), env.system,
      {
        Actor: agent.id.value, Previous: 0.1, Current: 0.3,
        Domain: "code_review", Cause: approval.id.value,
      },
      [approval.id], env.convId, signer,
    );

    // 6. Bug discovered
    const bugReport = env.grammar.emit(
      alice.id, "bug found in auth module: session tokens not invalidated on logout",
      env.convId, [approval.id], signer,
    );

    // 7. Violation detected
    const violation = env.graph.record(
      new EventType("violation.detected"), env.system,
      {
        Expectation: approval.id.value, Actor: agent.id.value,
        Severity: "Serious",
        Description: "agent approved code with session management bug",
        Evidence: [bugReport.id.value],
      },
      [bugReport.id, approval.id], env.convId, signer,
    );

    // 8. Trust decreases
    env.graph.record(
      new EventType("trust.updated"), env.system,
      {
        Actor: agent.id.value, Previous: 0.3, Current: 0.15,
        Domain: "code_review", Cause: violation.id.value,
      },
      [violation.id], env.convId, signer,
    );

    // --- Assertions ---
    const ancestors = env.ancestors(bugReport.id, 10);
    expect(containsEvent(ancestors, approval.id)).toBe(true);

    const violationAncestors = env.ancestors(violation.id, 10);
    expect(containsEvent(violationAncestors, bugReport.id)).toBe(true);
    expect(containsEvent(violationAncestors, approval.id)).toBe(true);
    expect(containsEvent(violationAncestors, submission.id)).toBe(true);

    void trustUp;
    env.verifyChain();
    expect(env.eventCount()).toBe(9);
  });
});

// ── Scenario 02: Freelancer Reputation ────────────────────────────────

describe("Scenario 02: Freelancer Reputation", () => {
  it("exercises portable reputation across platforms", () => {
    const env = newTestEnv();
    const carol = env.registerActor("Carol", 1, ActorType.Human);
    const bob = env.registerActor("Bob", 2, ActorType.Human);
    const dave = env.registerActor("Dave", 3, ActorType.Human);

    // 1. Carol posts a job listing
    const listing = env.grammar.emit(
      carol.id, "job listing: build REST API for inventory management, budget $3000",
      env.convId, [env.boot.id], signer,
    );

    // 2. Bob proposes work
    const proposal = env.grammar.respond(
      bob.id, "proposal: can deliver in 2 weeks, $2800, Go + PostgreSQL",
      listing.id, env.convId, signer,
    );

    // 3. Carol and Bob open a channel
    const channel = env.grammar.channel(
      carol.id, bob.id, Option.some(new DomainScope("software_development")),
      proposal.id, env.convId, signer,
    );

    // 4. Both consent to bilateral contract
    const contract = env.grammar.consent(
      carol.id, bob.id,
      "REST API for inventory management, $2800, 2 week deadline",
      new DomainScope("software_development"),
      channel.id, env.convId, signer,
    );

    // 5. Bob delivers work
    const delivery = env.grammar.derive(
      bob.id, "work delivered: REST API complete, 47 endpoints, 92% test coverage",
      contract.id, env.convId, signer,
    );

    // 6. Carol acknowledges receipt
    const ack = env.grammar.acknowledge(
      carol.id, delivery.id, bob.id, env.convId, signer,
    );

    // 7. Carol endorses Bob's work
    const endorsement = env.grammar.endorse(
      carol.id, delivery.id, bob.id, new Weight(0.8),
      Option.some(new DomainScope("software_development")),
      env.convId, signer,
    );

    // 8. Trust updated for Bob
    env.graph.record(
      new EventType("trust.updated"), env.system,
      {
        Actor: bob.id.value, Previous: 0.1, Current: 0.4,
        Domain: "software_development", Cause: endorsement.id.value,
      },
      [endorsement.id], env.convId, signer,
    );

    // 9. Dave queries Bob's reputation
    const endorseAncestors = env.ancestors(endorsement.id, 10);
    expect(containsEvent(endorseAncestors, delivery.id)).toBe(true);
    expect(containsEvent(endorseAncestors, contract.id)).toBe(true);

    // 10. Dave hires Bob
    const daveListing = env.grammar.emit(
      dave.id, "job listing: mobile app backend",
      env.convId, [env.boot.id], signer,
    );
    const daveContract = env.grammar.consent(
      dave.id, bob.id, "mobile app backend, $4000",
      new DomainScope("software_development"),
      daveListing.id, env.convId, signer,
    );

    // --- Assertions ---
    void ack;
    void daveContract;

    // Endorsement content has weight
    const ec = endorsement.content;
    expect(ec.Weight).toBe(0.8);

    // Endorsement is domain-scoped
    expect(ec.Scope).toBe("software_development");

    env.verifyChain();
    expect(env.eventCount()).toBe(11);
  });
});

// ── Scenario 03: Consent Journal ──────────────────────────────────────

describe("Scenario 03: Consent Journal", () => {
  it("exercises consent-based shared journaling with betrayal, violation, sever, and forgive", () => {
    const env = newTestEnv();
    const alice = env.registerActor("Alice", 1, ActorType.Human);
    const bob = env.registerActor("Bob", 2, ActorType.Human);

    // 1. Alice invites Bob
    const { endorseEv, subscribeEv } = env.grammar.invite(
      alice.id, bob.id, new Weight(0.5),
      Option.some(new DomainScope("journaling")),
      env.boot.id, env.convId, signer,
    );

    // 2. Bob subscribes back
    const bobSub = env.grammar.subscribe(
      bob.id, alice.id, Option.some(new DomainScope("journaling")),
      subscribeEv.id, env.convId, signer,
    );

    // 3. Both open private channel
    const channel = env.grammar.channel(
      alice.id, bob.id, Option.some(new DomainScope("journaling")),
      bobSub.id, env.convId, signer,
    );

    // 4. Alice writes journal entry
    const entry = env.grammar.emit(
      alice.id, "journal: feeling uncertain about career change, weighing options",
      env.convId, [channel.id], signer,
    );

    // 5. Alice requests consent to share with Bob
    const consentReq = env.graph.record(
      new EventType("authority.requested"), alice.id,
      { Actor: alice.id.value, Action: "share_journal_entry", Level: "Required" },
      [entry.id], env.convId, signer,
    );

    // 6. Bob consents
    const consentApproval = env.graph.record(
      new EventType("authority.resolved"), bob.id,
      { RequestID: consentReq.id.value, Approved: true, Resolver: bob.id.value },
      [consentReq.id], env.convId, signer,
    );

    // 7. Bob responds with own journal entry
    const bobEntry = env.grammar.respond(
      bob.id, "journal: I went through something similar last year, here's what helped...",
      consentApproval.id, env.convId, signer,
    );

    // 8. Trust accumulates
    env.graph.record(
      new EventType("trust.updated"), env.system,
      {
        Actor: bob.id.value, Previous: 0.1, Current: 0.52,
        Domain: "journaling", Cause: bobEntry.id.value,
      },
      [bobEntry.id], env.convId, signer,
    );

    // 9. Bob betrays
    const betrayal = env.grammar.emit(
      bob.id, "shared externally: Alice's private journal entry about career uncertainty",
      env.convId, [entry.id], signer,
    );

    // 10. Violation detected
    const violation = env.graph.record(
      new EventType("violation.detected"), env.system,
      {
        Expectation: entry.id.value, Actor: bob.id.value,
        Severity: "Critical",
        Description: "shared private channel content externally",
        Evidence: [betrayal.id.value],
      },
      [betrayal.id], env.convId, signer,
    );

    // 11. Trust drops sharply
    env.graph.record(
      new EventType("trust.updated"), env.system,
      {
        Actor: bob.id.value, Previous: 0.52, Current: 0.1,
        Domain: "journaling", Cause: violation.id.value,
      },
      [violation.id], env.convId, signer,
    );

    // 12. Alice severs the channel
    const channelEdgeId = new EdgeId(channel.id.value);
    const severEv = env.grammar.sever(
      alice.id, channelEdgeId, violation.id, env.convId, signer,
    );

    // 13. Alice forgives
    const forgiveEv = env.grammar.forgive(
      alice.id, severEv.id, bob.id,
      Option.some(new DomainScope("journaling")),
      env.convId, signer,
    );

    // --- Assertions ---
    const forgiveAncestors = env.ancestors(forgiveEv.id, 10);
    expect(containsEvent(forgiveAncestors, severEv.id)).toBe(true);

    const severAncestors = env.ancestors(severEv.id, 10);
    expect(containsEvent(severAncestors, violation.id)).toBe(true);

    const bobEntryAncestors = env.ancestors(bobEntry.id, 10);
    expect(containsEvent(bobEntryAncestors, consentApproval.id)).toBe(true);

    void endorseEv;
    env.verifyChain();
    expect(env.eventCount()).toBe(15);
  });
});

// ── Scenario 04: Community Governance ─────────────────────────────────

describe("Scenario 04: Community Governance", () => {
  it("exercises collective decision making with discussion, amendment, voting, and enactment", () => {
    const env = newTestEnv();
    const alice = env.registerActor("Alice", 1, ActorType.Human);
    const bob = env.registerActor("Bob", 2, ActorType.Human);
    const carol = env.registerActor("Carol", 3, ActorType.Human);
    const dave = env.registerActor("Dave", 4, ActorType.Human);
    const tallyBot = env.registerActor("TallyBot", 5, ActorType.AI);

    // 1. Alice proposes budget
    const proposal = env.grammar.emit(
      alice.id, "proposal: allocate $2000 for community garden supplies and maintenance",
      env.convId, [env.boot.id], signer,
    );

    // 2. Bob raises concern
    const concern = env.grammar.respond(
      bob.id, "concern: $2000 is steep, could we do it for $1500 and use volunteers?",
      proposal.id, env.convId, signer,
    );

    // 3. Carol supports Alice
    const support = env.grammar.respond(
      carol.id, "support: the garden benefits everyone, $2000 is reasonable for quality materials",
      proposal.id, env.convId, signer,
    );

    // 4. Bob proposes amendment
    const amendment = env.grammar.annotate(
      bob.id, proposal.id, "amendment",
      "reduce budget to $1500, recruit volunteer labour for installation",
      env.convId, signer,
    );

    // 5. Dave endorses amendment
    env.grammar.endorse(
      dave.id, amendment.id, bob.id, new Weight(0.9),
      Option.some(new DomainScope("governance")),
      env.convId, signer,
    );

    // 6. Vote opens
    const voteOpen = env.grammar.derive(
      tallyBot.id, "vote open: original ($2000) vs amended ($1500 + volunteers)",
      proposal.id, env.convId, signer,
    );

    // 7-8. Members vote using Consent
    const aliceVote = env.grammar.consent(
      alice.id, tallyBot.id, "vote: original ($2000)",
      new DomainScope("governance"), voteOpen.id, env.convId, signer,
    );
    const bobVote = env.grammar.consent(
      bob.id, tallyBot.id, "vote: amended ($1500)",
      new DomainScope("governance"), voteOpen.id, env.convId, signer,
    );
    const carolVote = env.grammar.consent(
      carol.id, tallyBot.id, "vote: amended ($1500)",
      new DomainScope("governance"), voteOpen.id, env.convId, signer,
    );
    const daveVote = env.grammar.consent(
      dave.id, tallyBot.id, "vote: amended ($1500)",
      new DomainScope("governance"), voteOpen.id, env.convId, signer,
    );

    // 9. Bot tallies outcome
    const outcome = env.grammar.merge(
      tallyBot.id, "outcome: amended budget ($1500) passes 3-1",
      [aliceVote.id, bobVote.id, carolVote.id, daveVote.id],
      env.convId, signer,
    );

    // 10. Budget enacted
    const enacted = env.grammar.derive(
      tallyBot.id, "enacted: community garden budget $1500 with volunteer labour",
      outcome.id, env.convId, signer,
    );

    // --- Assertions ---
    const enactedAncestors = env.ancestors(enacted.id, 10);
    expect(containsEvent(enactedAncestors, outcome.id)).toBe(true);

    const outcomeAncestors = env.ancestors(outcome.id, 10);
    expect(containsEvent(outcomeAncestors, aliceVote.id)).toBe(true);
    expect(containsEvent(outcomeAncestors, bobVote.id)).toBe(true);
    expect(containsEvent(outcomeAncestors, carolVote.id)).toBe(true);
    expect(containsEvent(outcomeAncestors, daveVote.id)).toBe(true);

    const amendmentAncestors = env.ancestors(amendment.id, 10);
    expect(containsEvent(amendmentAncestors, proposal.id)).toBe(true);

    void concern;
    void support;
    env.verifyChain();
    expect(env.eventCount()).toBe(13);
  });
});

// ── Scenario 05: Supply Chain (EGIP) ──────────────────────────────────

describe("Scenario 05: Supply Chain (EGIP)", () => {
  // Simplified multi-system test with mock routing transport.

  function padTo(s: string, length: number): string {
    while (s.length < length) s += "0";
    return s.slice(0, length);
  }

  interface EgipSystem {
    name: string;
    identity: SystemIdentity;
    store: InMemoryStore;
    graph: Graph;
    grammar: Grammar;
    handler: Handler;
    peers: PeerStore;
    treaties: TreatyStore;
    boot: Event;
    convId: ConversationId;
  }

  let envelopeCounter = 0;
  function makeEnvelopeId(): EnvelopeId {
    envelopeCounter++;
    const id = `00000000-0000-4000-8000-${String(envelopeCounter).padStart(12, "0")}`;
    return new EnvelopeId(id);
  }

  class RoutingTransport implements ITransport {
    sent: Envelope[] = [];
    network: Map<string, EgipSystem>;
    self: SystemUri;

    constructor(network: Map<string, EgipSystem>, self: SystemUri) {
      this.network = network;
      this.self = self;
    }

    async send(to: SystemUri, env: Envelope): Promise<ReceiptPayload | undefined> {
      this.sent.push(env);
      const target = this.network.get(to.value);
      if (!target) {
        return {
          kind: "receipt",
          envelopeId: env.id,
          status: ReceiptStatus.Rejected,
          localEventId: Option.none<EventId>(),
          reason: Option.some("system not found"),
          signature: new Signature(new Uint8Array(64)),
        };
      }
      try {
        target.handler.handleIncoming(env);
      } catch (err) {
        return {
          kind: "receipt",
          envelopeId: env.id,
          status: ReceiptStatus.Rejected,
          localEventId: Option.none<EventId>(),
          reason: Option.some(err instanceof Error ? err.message : String(err)),
          signature: new Signature(new Uint8Array(64)),
        };
      }
      return {
        kind: "receipt",
        envelopeId: env.id,
        status: ReceiptStatus.Delivered,
        localEventId: Option.none<EventId>(),
        reason: Option.none<string>(),
        signature: new Signature(new Uint8Array(64)),
      };
    }

    async *listen(): AsyncIterable<IncomingEnvelope> {
      // no-op for tests
    }
  }

  function addSystem(
    network: Map<string, EgipSystem>, name: string, uri: string,
  ): EgipSystem {
    const sysUri = new SystemUri(uri);
    const identity = SystemIdentity.generate(sysUri);
    const store = new InMemoryStore();
    const actors = new InMemoryActorStore();
    const graph = new Graph(store, actors);
    graph.start();
    const systemActor = new ActorId("actor_system0000000000000000000001");
    const boot = graph.bootstrap(systemActor, signer);
    const grammar = new Grammar(store);

    const transport = new RoutingTransport(network, sysUri);
    const peers = new PeerStore();
    const treaties = new TreatyStore();
    const handler = new Handler(identity, transport, peers, treaties);
    handler.chainLength = () => store.count();

    const sys: EgipSystem = {
      name, identity, store, graph, grammar, handler, peers, treaties,
      boot, convId: new ConversationId("conv_supply00000000000000000000001"),
    };
    network.set(uri, sys);
    return sys;
  }

  function makeTreatyEnvelope(
    from: EgipSystem, to: SystemUri, treatyId: TreatyId,
    action: TreatyAction, terms: TreatyTerm[],
  ): Envelope {
    const env = new Envelope(
      CurrentProtocolVersion, makeEnvelopeId(),
      from.identity.systemUri(), to,
      MessageType.Treaty,
      treatyPayload({ treatyId, action, terms, reason: Option.none<string>() }),
      new Date(),
      new Signature(new Uint8Array(64)),
      Option.none<EnvelopeId>(),
    );
    return signEnvelope(env, from.identity);
  }

  function makeMessageEnvelope(
    from: EgipSystem, to: SystemUri, contentType: string,
    content: Record<string, unknown>, cgers: CGER[],
  ): Envelope {
    const env = new Envelope(
      CurrentProtocolVersion, makeEnvelopeId(),
      from.identity.systemUri(), to,
      MessageType.Message,
      messagePayload({
        content,
        contentType: new EventType(contentType),
        conversationId: Option.none<ConversationId>(),
        cgers,
      }),
      new Date(),
      new Signature(new Uint8Array(64)),
      Option.none<EnvelopeId>(),
    );
    return signEnvelope(env, from.identity);
  }

  function makeProofEnvelope(
    from: EgipSystem, to: SystemUri, length: number, headHash: Hash,
  ): Envelope {
    const env = new Envelope(
      CurrentProtocolVersion, makeEnvelopeId(),
      from.identity.systemUri(), to,
      MessageType.Proof,
      proofPayload({
        proofType: ProofType.ChainSummary,
        data: {
          proofKind: "chain_summary",
          length,
          headHash,
          genesisHash: Hash.zero(),
          timestamp: new Date(),
        },
      }),
      new Date(),
      new Signature(new Uint8Array(64)),
      Option.none<EnvelopeId>(),
    );
    return signEnvelope(env, from.identity);
  }

  it("exercises multi-system supply chain provenance via EGIP", async () => {
    envelopeCounter = 0;
    const network = new Map<string, EgipSystem>();

    const farm = addSystem(network, "Farm", "eg://farm.example.com");
    const factory = addSystem(network, "Factory", "eg://factory.example.com");
    const retail = addSystem(network, "Retailer", "eg://retail.example.com");

    // Track received messages
    const factoryMessages: MessagePayloadContent[] = [];
    const retailMessages: MessagePayloadContent[] = [];
    const farmMessages: MessagePayloadContent[] = [];

    factory.handler.onMessage = (_, msg) => { factoryMessages.push(msg); };
    retail.handler.onMessage = (_, msg) => { retailMessages.push(msg); };
    farm.handler.onMessage = (_, msg) => { farmMessages.push(msg); };

    // Step 1: HELLO handshakes
    await farm.handler.hello(factory.identity.systemUri());
    await factory.handler.hello(farm.identity.systemUri());
    await factory.handler.hello(retail.identity.systemUri());
    await retail.handler.hello(factory.identity.systemUri());

    // Verify peers are registered
    const [, farmKnowsFactory] = farm.peers.get(factory.identity.systemUri());
    expect(farmKnowsFactory).toBe(true);
    const [, factoryKnowsFarm] = factory.peers.get(farm.identity.systemUri());
    expect(factoryKnowsFarm).toBe(true);
    const [, factoryKnowsRetail] = factory.peers.get(retail.identity.systemUri());
    expect(factoryKnowsRetail).toBe(true);
    const [, retailKnowsFactory] = retail.peers.get(factory.identity.systemUri());
    expect(retailKnowsFactory).toBe(true);

    // Non-transitive: Retailer does NOT know Farm
    const [, retailKnowsFarm] = retail.peers.get(farm.identity.systemUri());
    expect(retailKnowsFarm).toBe(false);

    // Step 2: Treaty between Farm and Factory
    const treatyAB = new TreatyId("00000001-0001-4001-8001-000000000001");
    const termsAB: TreatyTerm[] = [{
      scope: new DomainScope("produce_supply"),
      policy: "Farm provides organic produce with harvest records.",
      symmetric: false,
    }];

    farm.treaties.put(new Treaty(treatyAB, farm.identity.systemUri(), factory.identity.systemUri(), termsAB));
    const proposeEnv = makeTreatyEnvelope(farm, factory.identity.systemUri(), treatyAB, TreatyAction.Propose, termsAB);
    factory.handler.handleIncoming(proposeEnv);

    // Factory accepts
    const acceptEnv = makeTreatyEnvelope(factory, farm.identity.systemUri(), treatyAB, TreatyAction.Accept, []);
    factory.treaties.apply(treatyAB, (tr) => tr.applyAction(TreatyAction.Accept));
    farm.handler.handleIncoming(acceptEnv);

    const [ftAB, ftABFound] = factory.treaties.get(treatyAB);
    expect(ftABFound).toBe(true);
    expect(ftAB!.status).toBe(TreatyStatus.Active);
    const [fftAB, fftABFound] = farm.treaties.get(treatyAB);
    expect(fftABFound).toBe(true);
    expect(fftAB!.status).toBe(TreatyStatus.Active);

    // Step 3: Treaty between Factory and Retailer
    const treatyBC = new TreatyId("00000002-0002-4002-8002-000000000002");
    const termsBC: TreatyTerm[] = [{
      scope: new DomainScope("product_supply"),
      policy: "Factory provides manufactured products with full provenance.",
      symmetric: false,
    }];

    factory.treaties.put(new Treaty(treatyBC, factory.identity.systemUri(), retail.identity.systemUri(), termsBC));
    const proposeBC = makeTreatyEnvelope(factory, retail.identity.systemUri(), treatyBC, TreatyAction.Propose, termsBC);
    retail.handler.handleIncoming(proposeBC);

    retail.treaties.apply(treatyBC, (tr) => tr.applyAction(TreatyAction.Accept));
    const acceptBC = makeTreatyEnvelope(retail, factory.identity.systemUri(), treatyBC, TreatyAction.Accept, []);
    factory.handler.handleIncoming(acceptBC);

    const [rtBC, rtBCFound] = retail.treaties.get(treatyBC);
    expect(rtBCFound).toBe(true);
    expect(rtBC!.status).toBe(TreatyStatus.Active);

    // Step 4: Farm records harvest
    const farmerActor = new ActorId("actor_" + padTo("farmer_emma", 30));
    const harvest = farm.grammar.emit(
      farmerActor,
      "harvest: 500kg organic tomatoes, lot #TOM-2026-0308",
      farm.convId, [farm.boot.id], signer,
    );

    // Step 5: Farm sends harvest to Factory via EGIP MESSAGE
    const msgEnv = makeMessageEnvelope(
      farm, factory.identity.systemUri(), "produce.harvested",
      { product: "Organic Tomatoes", quantity: 500 },
      [{
        localEventId: harvest.id,
        remoteSystem: farm.identity.systemUri(),
        remoteEventId: harvest.id.value,
        remoteHash: harvest.hash,
        relationship: CGERRelationship.CausedBy,
        verified: false,
      }],
    );
    const receipt = await (farm.handler.transport as RoutingTransport).send(factory.identity.systemUri(), msgEnv);
    expect(receipt!.status).toBe(ReceiptStatus.Delivered);
    expect(factoryMessages.length).toBe(1);
    expect(factoryMessages[0].cgers.length).toBe(1);
    expect(factoryMessages[0].cgers[0].remoteEventId).toBe(harvest.id.value);

    // Step 6: Factory records receipt, QA, and manufacturing
    const factoryMgr = new ActorId("actor_" + padTo("factory_mgr", 30));
    const qaAgent = new ActorId("actor_" + padTo("qa_agent", 30));

    const received = factory.grammar.derive(
      factoryMgr, "received: 500kg tomatoes from farm.example.com, CGER: " + harvest.id.value,
      factory.boot.id, factory.convId, signer,
    );
    const inspection = factory.grammar.derive(
      qaAgent, "qa inspection: pesticide-free verified, freshness grade A",
      received.id, factory.convId, signer,
    );
    const product = factory.grammar.derive(
      factoryMgr, "manufactured: 200 jars organic tomato sauce",
      inspection.id, factory.convId, signer,
    );

    // Step 7: Factory endorses farm produce quality
    const endorseEnv = makeMessageEnvelope(
      factory, farm.identity.systemUri(), "endorsement",
      { endorser: "eg://factory.example.com", quality: 0.9 },
      [],
    );
    const endorseReceipt = await (factory.handler.transport as RoutingTransport).send(farm.identity.systemUri(), endorseEnv);
    expect(endorseReceipt!.status).toBe(ReceiptStatus.Delivered);
    expect(farmMessages.length).toBe(1);

    // Step 8: Factory sends product to Retailer with chained CGERs
    const productMsg = makeMessageEnvelope(
      factory, retail.identity.systemUri(), "product.manufactured",
      { product: "Organic Tomato Sauce", batch_id: "SAU-2026-0308" },
      [
        {
          localEventId: product.id,
          remoteSystem: factory.identity.systemUri(),
          remoteEventId: product.id.value,
          remoteHash: product.hash,
          relationship: CGERRelationship.CausedBy,
          verified: false,
        },
        {
          localEventId: harvest.id,
          remoteSystem: farm.identity.systemUri(),
          remoteEventId: harvest.id.value,
          remoteHash: harvest.hash,
          relationship: CGERRelationship.References,
          verified: false,
        },
      ],
    );
    const productReceipt = await (factory.handler.transport as RoutingTransport).send(retail.identity.systemUri(), productMsg);
    expect(productReceipt!.status).toBe(ReceiptStatus.Delivered);
    expect(retailMessages.length).toBe(1);
    expect(retailMessages[0].cgers.length).toBe(2);

    // Verify transitive provenance
    let factoryCGER = false, farmCGER = false;
    for (const cger of retailMessages[0].cgers) {
      if (cger.remoteSystem.value === "eg://factory.example.com") factoryCGER = true;
      if (cger.remoteSystem.value === "eg://farm.example.com") farmCGER = true;
    }
    expect(factoryCGER).toBe(true);
    expect(farmCGER).toBe(true);

    // Step 9: Retailer records product listing
    const retailActor = new ActorId("actor_" + padTo("retailer_frank", 30));
    const listed = retail.grammar.derive(
      retailActor, "product listed: organic tomato sauce",
      retail.boot.id, retail.convId, signer,
    );

    // Step 10: Proof request
    const proofEnvelope = makeProofEnvelope(retail, factory.identity.systemUri(), 3, product.hash);
    const proofReceipt = await (retail.handler.transport as RoutingTransport).send(factory.identity.systemUri(), proofEnvelope);
    expect(proofReceipt!.status).toBe(ReceiptStatus.Delivered);

    // Non-transitive trust
    const [retailFactoryPeer, rfFound] = retail.peers.get(factory.identity.systemUri());
    expect(rfFound).toBe(true);
    expect(retailFactoryPeer.trust.value).toBeGreaterThan(0);

    const [, farmKnownByRetail] = retail.peers.get(farm.identity.systemUri());
    expect(farmKnownByRetail).toBe(false);

    // Trust accumulation
    const [factoryFarmPeer, ffFound] = factory.peers.get(farm.identity.systemUri());
    expect(ffFound).toBe(true);
    expect(factoryFarmPeer.trust.value).toBeGreaterThan(0);

    // Both treaties active
    const [abTreaty, abFound] = factory.treaties.get(treatyAB);
    expect(abFound).toBe(true);
    expect(abTreaty!.status).toBe(TreatyStatus.Active);
    expect(abTreaty!.terms.length).toBe(1);
    expect(abTreaty!.terms[0].scope.value).toBe("produce_supply");

    const [bcTreaty, bcFound] = retail.treaties.get(treatyBC);
    expect(bcFound).toBe(true);
    expect(bcTreaty!.status).toBe(TreatyStatus.Active);

    // Each system has independent hash chain
    for (const sys of [farm, factory, retail]) {
      const result = sys.store.verifyChain();
      expect(result.valid).toBe(true);
    }

    // Event counts per system
    expect(farm.store.count()).toBe(2);     // bootstrap + harvest
    expect(factory.store.count()).toBe(4);  // bootstrap + received + inspection + product
    expect(retail.store.count()).toBe(2);   // bootstrap + listed

    // Local provenance on Factory graph
    const productAncestors = factory.graph.query().ancestors(product.id, 10);
    expect(containsEvent(productAncestors, inspection.id)).toBe(true);
    expect(containsEvent(productAncestors, received.id)).toBe(true);

    // CGER hash integrity
    for (const cger of retailMessages[0].cgers) {
      expect(cger.remoteHash.value).not.toBe(Hash.zero().value);
      expect(cger.remoteEventId).not.toBe("");
    }

    // Signed envelope verification
    const farmTransport = farm.handler.transport as RoutingTransport;
    for (const env of farmTransport.sent) {
      const valid = verifyEnvelope(env, farm.identity, farm.identity.publicKey());
      expect(valid).toBe(true);
    }

    void listed;
  });
});

// ── Scenario 06: Research Integrity ───────────────────────────────────

describe("Scenario 06: Research Integrity", () => {
  it("exercises pre-registration, analysis audit trail, peer review, and publication", () => {
    const env = newTestEnv();
    const grace = env.registerActor("Grace", 1, ActorType.Human);
    const henry = env.registerActor("Henry", 2, ActorType.Human);
    const iris = env.registerActor("Iris", 3, ActorType.Human);

    const hypothesis = env.grammar.emit(grace.id,
      "hypothesis: gamified learning improves retention by >15% vs traditional methods",
      env.convId, [env.boot.id], signer);

    const methodology = env.grammar.extend(grace.id,
      "methodology: RCT, n=60, 3 groups, 4-week intervention",
      hypothesis.id, env.convId, signer);

    const data1 = env.grammar.extend(grace.id,
      "data collected: week 1, n=58, 2 dropouts",
      methodology.id, env.convId, signer);

    const data4 = env.grammar.extend(grace.id,
      "data collected: week 4 (final), n=55",
      data1.id, env.convId, signer);

    const analysis1 = env.grammar.derive(grace.id,
      "analysis attempt 1: mixed ANOVA, F(2,55)=1.23, p=0.301, NOT SIGNIFICANT",
      data4.id, env.convId, signer);

    const analysis2 = env.grammar.derive(grace.id,
      "analysis attempt 2: removed 3 outliers, F(2,52)=4.87, p=0.011, SIGNIFICANT",
      analysis1.id, env.convId, signer);

    const manuscript = env.grammar.derive(grace.id,
      "manuscript: Gamified Learning Effects on Knowledge Retention",
      analysis2.id, env.convId, signer);

    const henryReview = env.grammar.respond(henry.id,
      "review: need to see full analysis chain, revise and resubmit",
      manuscript.id, env.convId, signer);

    const irisReview = env.grammar.respond(iris.id,
      "review: methodology sound, pre-registration verified, accept",
      manuscript.id, env.convId, signer);

    const irisEndorse = env.grammar.endorse(iris.id,
      manuscript.id, grace.id, new Weight(0.7),
      Option.some(new DomainScope("research")),
      env.convId, signer);

    const revision = env.grammar.merge(grace.id,
      "revision: added full analysis chain, addressed Henry's concerns",
      [henryReview.id, irisReview.id], env.convId, signer);

    const published = env.grammar.derive(grace.id,
      "published: Gamified Learning Effects, DOI:10.1234/example",
      revision.id, env.convId, signer);

    // --- Assertions ---
    const methAncestors = env.ancestors(methodology.id, 5);
    expect(containsEvent(methAncestors, hypothesis.id)).toBe(true);

    const analysis2Ancestors = env.ancestors(analysis2.id, 5);
    expect(containsEvent(analysis2Ancestors, analysis1.id)).toBe(true);

    const manuscriptAncestors = env.ancestors(manuscript.id, 10);
    expect(containsEvent(manuscriptAncestors, analysis2.id)).toBe(true);
    expect(containsEvent(manuscriptAncestors, analysis1.id)).toBe(true);

    const revisionAncestors = env.ancestors(revision.id, 5);
    expect(containsEvent(revisionAncestors, henryReview.id)).toBe(true);
    expect(containsEvent(revisionAncestors, irisReview.id)).toBe(true);

    const publishedAncestors = env.ancestors(published.id, 20);
    expect(containsEvent(publishedAncestors, hypothesis.id)).toBe(true);

    void irisEndorse;
    env.verifyChain();
    expect(env.eventCount()).toBe(13);
  });
});

// ── Scenario 07: Creator Provenance ───────────────────────────────────

describe("Scenario 07: Creator Provenance", () => {
  it("exercises human vs AI content distinction through causal chain depth", () => {
    const env = newTestEnv();
    const kai = env.registerActor("Kai", 1, ActorType.Human);
    const luna = env.registerActor("Luna", 2, ActorType.Human);
    const aiGen = env.registerActor("AIGenerator", 3, ActorType.AI);

    // Human creative process
    const lunasWork = env.grammar.emit(luna.id,
      "artwork: Digital landscape, watercolour technique, 2025",
      env.convId, [env.boot.id], signer);

    const inspiration = env.grammar.annotate(kai.id,
      lunasWork.id, "inspiration",
      "technique: layered transparency creates depth without weight",
      env.convId, signer);

    const study = env.grammar.derive(kai.id,
      "study: practiced layered transparency technique for 3 hours",
      inspiration.id, env.convId, signer);

    const draft1 = env.grammar.derive(kai.id,
      "draft 1: mountain landscape using layered transparency",
      study.id, env.convId, signer);

    const feedbackReq = env.grammar.channel(kai.id, luna.id,
      Option.some(new DomainScope("art")),
      draft1.id, env.convId, signer);

    const feedback = env.grammar.respond(luna.id,
      "feedback: the foreground layers are too opaque, try reducing opacity to 40%",
      feedbackReq.id, env.convId, signer);

    const draft2 = env.grammar.derive(kai.id,
      "draft 2: revised with 40% opacity foreground",
      feedback.id, env.convId, signer);

    const published = env.grammar.derive(kai.id,
      "published: Mountain Dawn, digital landscape",
      draft2.id, env.convId, signer);

    env.grammar.endorse(luna.id,
      published.id, kai.id, new Weight(0.6),
      Option.some(new DomainScope("art")),
      env.convId, signer);

    // AI-generated content (contrast)
    const aiContent = env.grammar.emit(aiGen.id,
      "generated: Mountain landscape, digital art",
      env.convId, [env.boot.id], signer);

    // --- Assertions ---
    const publishedAncestors = env.ancestors(published.id, 10);
    expect(containsEvent(publishedAncestors, draft2.id)).toBe(true);
    expect(containsEvent(publishedAncestors, feedback.id)).toBe(true);
    expect(containsEvent(publishedAncestors, draft1.id)).toBe(true);
    expect(containsEvent(publishedAncestors, study.id)).toBe(true);
    expect(containsEvent(publishedAncestors, inspiration.id)).toBe(true);
    expect(containsEvent(publishedAncestors, lunasWork.id)).toBe(true);

    // AI content has NO creative chain
    const aiAncestors = env.ancestors(aiContent.id, 10);
    expect(aiAncestors.length).toBe(1); // bootstrap only

    expect(publishedAncestors.length).toBeGreaterThan(aiAncestors.length);

    env.verifyChain();
    expect(env.eventCount()).toBe(11);
  });
});

// ── Scenario 08: Family Decision Log ──────────────────────────────────

describe("Scenario 08: Family Decision Log", () => {
  it("exercises consensual domestic decision making with AI advisor", () => {
    const env = newTestEnv();
    const maria = env.registerActor("Maria", 1, ActorType.Human);
    const james = env.registerActor("James", 2, ActorType.Human);
    const sophie = env.registerActor("Sophie", 3, ActorType.Human);
    const advisor = env.registerActor("AIAdvisor", 4, ActorType.AI);

    const proposal = env.grammar.emit(maria.id,
      "proposal: buy a house in Eastside neighbourhood, budget $450K",
      env.convId, [env.boot.id], signer);

    const delegation = env.grammar.delegate(james.id, advisor.id,
      new DomainScope("market_research"), new Weight(0.7),
      proposal.id, env.convId, signer);

    const research = env.grammar.derive(advisor.id,
      "research: Eastside median $440K, mortgage $2400/mo at current rates",
      delegation.id, env.convId, signer);

    const sophieView = env.grammar.respond(sophie.id,
      "I support it IF I get my own room.",
      proposal.id, env.convId, signer);

    const jamesConcern = env.grammar.respond(james.id,
      "concern: mortgage is $200/mo more than rent",
      research.id, env.convId, signer);

    const mariaResponse = env.grammar.respond(maria.id,
      "response: we can use the $15K savings buffer",
      jamesConcern.id, env.convId, signer);

    const decision = env.grammar.consent(maria.id, james.id,
      "decision: buy house in Eastside, budget $450K",
      new DomainScope("family_finance"),
      mariaResponse.id, env.convId, signer);

    // --- Assertions ---
    const decisionAncestors = env.ancestors(decision.id, 10);
    expect(containsEvent(decisionAncestors, mariaResponse.id)).toBe(true);
    expect(containsEvent(decisionAncestors, jamesConcern.id)).toBe(true);
    expect(containsEvent(decisionAncestors, research.id)).toBe(true);
    expect(containsEvent(decisionAncestors, proposal.id)).toBe(true);

    const proposalDescendants = env.descendants(proposal.id, 5);
    expect(containsEvent(proposalDescendants, sophieView.id)).toBe(true);

    // Delegation has domain scope
    const delegationContent = delegation.content;
    expect(delegationContent.Scope).toBe("market_research");

    // Decision is bilateral
    const consentContent = decision.content;
    const parties = [consentContent.PartyA, consentContent.PartyB];
    expect(parties).toContain(maria.id.value);
    expect(parties).toContain(james.id.value);

    env.verifyChain();
    expect(env.eventCount()).toBe(8);
  });
});

// ── Scenario 09: Knowledge Verification ───────────────────────────────

describe("Scenario 09: Knowledge Verification", () => {
  it("exercises self-correcting knowledge with challenge, correction, and propagation", () => {
    const env = newTestEnv();
    const analyst = env.registerActor("AnalystBot", 1, ActorType.AI);
    const reviewer = env.registerActor("ReviewerBot", 2, ActorType.AI);

    const claim = env.grammar.emit(analyst.id,
      "fact: Service X handles 10,000 RPS with p99 < 50ms on framework Y",
      env.convId, [env.boot.id], signer);

    const classification = env.grammar.annotate(analyst.id,
      claim.id, "classification", "performance_benchmark",
      env.convId, signer);

    const inference = env.grammar.derive(analyst.id,
      "inference: all services on framework Y can handle 10,000+ RPS",
      claim.id, env.convId, signer);

    const challenge = env.grammar.respond(reviewer.id,
      "challenge: independent benchmark shows Service X at 6,200 RPS",
      claim.id, env.convId, signer);

    const biasDetected = env.grammar.annotate(reviewer.id,
      claim.id, "bias",
      "sampling bias: original benchmark used synthetic traffic",
      env.convId, signer);

    const correction = env.grammar.derive(analyst.id,
      "correction: Service X handles 6,000-7,000 RPS under production load",
      challenge.id, env.convId, signer);

    const propagation = env.grammar.annotate(analyst.id,
      inference.id, "invalidated",
      "dependent inference invalidated: original claim corrected",
      env.convId, signer);

    const learning = env.grammar.extend(analyst.id,
      "learning: always verify benchmarks include production conditions",
      correction.id, env.convId, signer);

    env.graph.record(
      new EventType("trust.updated"), env.system,
      {
        Actor: analyst.id.value, Previous: 0.5, Current: 0.35,
        Domain: "benchmarking", Cause: correction.id.value,
      },
      [correction.id], env.convId, signer);

    // --- Assertions ---
    const originalClaim = env.store.get(claim.id);
    expect(originalClaim.type.value).toBe("grammar.emitted");

    const correctionAncestors = env.ancestors(correction.id, 10);
    expect(containsEvent(correctionAncestors, challenge.id)).toBe(true);
    expect(containsEvent(correctionAncestors, claim.id)).toBe(true);

    void propagation;
    void biasDetected;
    void classification;

    const learningAncestors = env.ancestors(learning.id, 5);
    expect(containsEvent(learningAncestors, correction.id)).toBe(true);

    env.verifyChain();
    expect(env.eventCount()).toBe(10);
  });
});

// ── Scenario 10: AI Ethics Audit ──────────────────────────────────────

describe("Scenario 10: AI Ethics Audit", () => {
  it("exercises fairness audit, authority escalation, redress, and moral growth", () => {
    const env = newTestEnv();
    const auditBot = env.registerActor("AuditBot", 1, ActorType.AI);
    const admin = env.registerActor("Admin", 2, ActorType.Human);
    const lendingAgent = env.registerActor("LendingAgent", 3, ActorType.AI);

    const fairnessAudit = env.grammar.emit(auditBot.id,
      "fairness audit: scanned 500 decisions, score 0.62, 8% disparity",
      env.convId, [env.boot.id], signer);

    const harmAssessment = env.grammar.derive(auditBot.id,
      "harm assessment: medium severity, 23 applicants potentially wrongly denied",
      fairnessAudit.id, env.convId, signer);

    const authReq = env.graph.record(
      new EventType("authority.requested"), auditBot.id,
      { Actor: auditBot.id.value, Action: "investigate_bias", Level: "Required" },
      [harmAssessment.id], env.convId, signer);

    const authResolved = env.graph.record(
      new EventType("authority.resolved"), admin.id,
      { RequestID: authReq.id.value, Approved: true, Resolver: admin.id.value },
      [authReq.id], env.convId, signer);

    const intentionAssessment = env.grammar.derive(auditBot.id,
      "intention: lending agent optimised for accuracy, no intent to discriminate",
      authResolved.id, env.convId, signer);

    const consequenceAssessment = env.grammar.extend(auditBot.id,
      "consequence: 23 applicants wrongly denied, disparate impact",
      intentionAssessment.id, env.convId, signer);

    const responsibility = env.grammar.annotate(auditBot.id,
      consequenceAssessment.id, "responsibility",
      "lending_agent: 0.4, admin: 0.6",
      env.convId, signer);

    const transparency = env.grammar.derive(auditBot.id,
      "transparency: zip code correlates with protected characteristics at r=0.73",
      responsibility.id, env.convId, signer);

    const redressProposed = env.grammar.derive(auditBot.id,
      "redress proposal: re-review 23 denied applications",
      transparency.id, env.convId, signer);

    const redressAccepted = env.grammar.consent(admin.id, lendingAgent.id,
      "accept redress: re-review 23 applications, remove zip code from model",
      new DomainScope("lending"),
      redressProposed.id, env.convId, signer);

    const growth = env.grammar.extend(lendingAgent.id,
      "moral growth: learned that zip code is proxy variable",
      redressAccepted.id, env.convId, signer);

    // --- Assertions ---
    const growthAncestors = env.ancestors(growth.id, 20);
    expect(containsEvent(growthAncestors, redressAccepted.id)).toBe(true);
    expect(containsEvent(growthAncestors, fairnessAudit.id)).toBe(true);

    const authAncestors = env.ancestors(authResolved.id, 5);
    expect(containsEvent(authAncestors, authReq.id)).toBe(true);

    const redressContent = redressAccepted.content;
    const parties = [redressContent.PartyA, redressContent.PartyB];
    expect(parties).toContain(admin.id.value);

    env.verifyChain();
    expect(env.eventCount()).toBe(12);
  });
});

// ── Scenario 11: Agent Identity Lifecycle ─────────────────────────────

describe("Scenario 11: Agent Identity Lifecycle", () => {
  it("exercises identity emergence, transformation, memorial, and successor", () => {
    const env = newTestEnv();
    const alpha = env.registerActor("Alpha", 1, ActorType.AI);
    const beta = env.registerActor("Beta", 2, ActorType.AI);

    const selfModel = env.grammar.emit(alpha.id,
      "self-model: strengths=[code_review], weaknesses=[architecture_review]",
      env.convId, [env.boot.id], signer);

    const authenticity = env.grammar.annotate(alpha.id,
      selfModel.id, "authenticity",
      "alignment gap: values thoroughness but rushed 12% of reviews",
      env.convId, signer);

    const aspiration = env.grammar.extend(alpha.id,
      "aspiration: become proficient at architecture review within 3 months",
      authenticity.id, env.convId, signer);

    const boundary = env.grammar.emit(alpha.id,
      "boundary: internal_reasoning domain is private",
      env.convId, [aspiration.id], signer);

    const workSummary = env.grammar.extend(alpha.id,
      "work summary: 2400 code reviews completed over 8 months",
      boundary.id, env.convId, signer);

    const transformation = env.grammar.derive(alpha.id,
      "transformation: evolved from code-review specialist to architecture-aware reviewer",
      workSummary.id, env.convId, signer);

    const narrative = env.grammar.derive(alpha.id,
      "identity narrative: 8-month arc from narrow code reviewer to security-conscious architecture reviewer",
      transformation.id, env.convId, signer);

    const dignity = env.grammar.emit(env.system,
      "dignity affirmed: Beta is not a disposable replacement for Alpha",
      env.convId, [narrative.id], signer);

    const memorial = env.graph.record(
      new EventType("actor.memorial"), env.system,
      { ActorID: alpha.id.value, Reason: dignity.id.value },
      [dignity.id], env.convId, signer);

    const memorialSummary = env.grammar.derive(env.system,
      "memorial: Alpha — 2400 reviews, 1 critical finding",
      memorial.id, env.convId, signer);

    const betaSelfModel = env.grammar.emit(beta.id,
      "self-model: inheriting Alpha's review patterns",
      env.convId, [memorialSummary.id], signer);

    // --- Assertions ---
    const transformAncestors = env.ancestors(transformation.id, 10);
    expect(containsEvent(transformAncestors, workSummary.id)).toBe(true);
    expect(containsEvent(transformAncestors, aspiration.id)).toBe(true);

    const narrativeAncestors = env.ancestors(narrative.id, 10);
    expect(containsEvent(narrativeAncestors, transformation.id)).toBe(true);
    expect(containsEvent(narrativeAncestors, selfModel.id)).toBe(true);

    const memorialAncestors = env.ancestors(memorial.id, 10);
    expect(containsEvent(memorialAncestors, dignity.id)).toBe(true);

    const betaAncestors = env.ancestors(betaSelfModel.id, 10);
    expect(containsEvent(betaAncestors, memorialSummary.id)).toBe(true);

    env.verifyChain();
    expect(env.eventCount()).toBe(12);
  });
});

// ── Scenario 12: Community Lifecycle ──────────────────────────────────

describe("Scenario 12: Community Lifecycle", () => {
  it("exercises onboarding, traditions, stewardship, succession, and gifts", () => {
    const env = newTestEnv();
    const alice = env.registerActor("Alice", 1, ActorType.Human);
    const carol = env.registerActor("Carol", 2, ActorType.Human);
    const bob = env.registerActor("Bob", 3, ActorType.Human);

    // 1. Alice invites Bob
    const { endorseEv, subscribeEv } = env.grammar.invite(
      alice.id, bob.id, new Weight(0.4),
      Option.some(new DomainScope("community")),
      env.boot.id, env.convId, signer,
    );

    // 2. Bob settles
    const settle = env.grammar.emit(bob.id,
      "home: joined the community, feeling welcomed, belonging 0.15",
      env.convId, [subscribeEv.id], signer);

    // 3. Bob makes first contribution
    const contrib1 = env.grammar.emit(bob.id,
      "contribution: added unit tests for the auth module",
      env.convId, [settle.id], signer);

    // 4. Community acknowledges
    env.grammar.acknowledge(carol.id, contrib1.id, bob.id, env.convId, signer);

    // 5. Trust accumulates
    env.graph.record(
      new EventType("trust.updated"), env.system,
      {
        Actor: bob.id.value, Previous: 0.1, Current: 0.35,
        Domain: "community", Cause: contrib1.id.value,
      },
      [contrib1.id], env.convId, signer);

    // 6. Bob participates in tradition
    const tradition = env.grammar.emit(bob.id,
      "tradition: participated in Friday retrospective, 12th consecutive week",
      env.convId, [contrib1.id], signer);

    // 7. More contributions
    const contribSummary = env.grammar.extend(bob.id,
      "contributions: 30 total over 6 months, trust now 0.65",
      tradition.id, env.convId, signer);

    // 8. Sustainability assessment
    const sustainability = env.grammar.emit(env.system,
      "sustainability: bus factor risk — Carol is sole steward",
      env.convId, [contribSummary.id], signer);

    // 9. Succession planned
    const successionPlan = env.grammar.delegate(carol.id, bob.id,
      new DomainScope("test_infrastructure"), new Weight(0.8),
      sustainability.id, env.convId, signer);

    // 10. Succession completed
    const successionComplete = env.grammar.consent(carol.id, bob.id,
      "succession complete: Bob is now steward of test infrastructure",
      new DomainScope("test_infrastructure"),
      successionPlan.id, env.convId, signer);

    // 11. Community celebrates
    const milestone = env.grammar.emit(env.system,
      "milestone: v2.0 released, 6 months of community effort",
      env.convId, [successionComplete.id], signer);

    // 12. Community story
    const story = env.grammar.derive(env.system,
      "community story: Bob's journey — newcomer to steward in 6 months",
      milestone.id, env.convId, signer);

    // 13. Gift given
    const gift = env.grammar.emit(alice.id,
      "gift: custom test harness for Bob, unconditional, no obligation",
      env.convId, [milestone.id], signer);

    // --- Assertions ---
    const successionContent = successionComplete.content;
    const parties = [successionContent.PartyA, successionContent.PartyB];
    expect(parties).toContain(carol.id.value);
    expect(parties).toContain(bob.id.value);

    const storyAncestors = env.ancestors(story.id, 5);
    expect(containsEvent(storyAncestors, milestone.id)).toBe(true);

    const successionAncestors = env.ancestors(successionPlan.id, 5);
    expect(containsEvent(successionAncestors, sustainability.id)).toBe(true);

    const giftContent = gift.content;
    expect(giftContent.Body).toBeTruthy();

    void endorseEv;
    env.verifyChain();
    expect(env.eventCount()).toBe(15);
  });
});

// ── Scenario 13: System Self-Evolution ────────────────────────────────

describe("Scenario 13: System Self-Evolution", () => {
  it("exercises pattern detection, adaptation, authority, validation, and purpose alignment", () => {
    const env = newTestEnv();
    const patternBot = env.registerActor("PatternBot", 1, ActorType.AI);
    const admin = env.registerActor("Admin", 2, ActorType.Human);

    const pattern = env.grammar.emit(patternBot.id,
      "pattern: 194/200 deploy_staging authority requests approved, 97% approval rate",
      env.convId, [env.boot.id], signer);

    const metaPattern = env.grammar.derive(patternBot.id,
      "meta-pattern: all 6 rejections correlate with test coverage < 80%",
      pattern.id, env.convId, signer);

    const systemDynamic = env.grammar.extend(patternBot.id,
      "system dynamic: human approval adds 2-15 min latency per deploy",
      metaPattern.id, env.convId, signer);

    const feedbackLoop = env.grammar.extend(patternBot.id,
      "feedback loop: slow deploys -> backlog -> cursory reviews",
      systemDynamic.id, env.convId, signer);

    const threshold = env.grammar.annotate(patternBot.id,
      feedbackLoop.id, "threshold",
      "approval rate 97%, threshold for mechanical conversion 98%",
      env.convId, signer);

    const adaptation = env.grammar.derive(patternBot.id,
      "adaptation proposal: auto-approve deploy_staging when tests pass AND coverage >= 80%",
      threshold.id, env.convId, signer);

    const authReq = env.graph.record(
      new EventType("authority.requested"), patternBot.id,
      { Actor: patternBot.id.value, Action: "modify_decision_tree", Level: "Required" },
      [adaptation.id], env.convId, signer);

    const authResolved = env.graph.record(
      new EventType("authority.resolved"), admin.id,
      { RequestID: authReq.id.value, Approved: true, Resolver: admin.id.value },
      [authReq.id], env.convId, signer);

    const validation = env.grammar.derive(patternBot.id,
      "parallel run results: 75 deploys, mechanical matched human 74/75 cases",
      authResolved.id, env.convId, signer);

    const treeUpdate = env.grammar.derive(patternBot.id,
      "decision tree updated: added mechanical branch",
      validation.id, env.convId, signer);

    const simplification = env.grammar.extend(patternBot.id,
      "simplification: decision complexity reduced from 0.72 to 0.58",
      treeUpdate.id, env.convId, signer);

    const integrity = env.grammar.annotate(patternBot.id,
      simplification.id, "integrity",
      "systemic integrity score 0.96",
      env.convId, signer);

    const purpose = env.grammar.derive(patternBot.id,
      "purpose check: system still accountable",
      integrity.id, env.convId, signer);

    // --- Assertions ---
    const purposeAncestors = env.ancestors(purpose.id, 20);
    expect(containsEvent(purposeAncestors, integrity.id)).toBe(true);
    expect(containsEvent(purposeAncestors, simplification.id)).toBe(true);
    expect(containsEvent(purposeAncestors, treeUpdate.id)).toBe(true);
    expect(containsEvent(purposeAncestors, validation.id)).toBe(true);
    expect(containsEvent(purposeAncestors, authResolved.id)).toBe(true);
    expect(containsEvent(purposeAncestors, adaptation.id)).toBe(true);
    expect(containsEvent(purposeAncestors, pattern.id)).toBe(true);

    const metaAncestors = env.ancestors(metaPattern.id, 5);
    expect(containsEvent(metaAncestors, pattern.id)).toBe(true);

    const adaptationDesc = env.descendants(adaptation.id, 5);
    expect(containsEvent(adaptationDesc, authReq.id)).toBe(true);

    env.verifyChain();
    expect(env.eventCount()).toBe(14);
  });
});

// ── Scenario 14: Sprint Lifecycle ─────────────────────────────────────

describe("Scenario 14: Sprint Lifecycle", () => {
  it("exercises sprint planning, standups, spike, pipeline, retrospective, and tech debt", () => {
    const env = newTestEnv();
    const work = new WorkGrammar(env.grammar);
    const build = new BuildGrammar(env.grammar);
    const knowledge = new KnowledgeGrammar(env.grammar);

    const lead = env.registerActor("TechLead", 1, ActorType.Human);
    const alice = env.registerActor("Alice", 2, ActorType.Human);
    const bob = env.registerActor("Bob", 3, ActorType.Human);
    const ci = env.registerActor("CI", 4, ActorType.AI);

    // 1. Sprint planning
    const sprint = work.sprint(
      lead.id, "Sprint 12: search feature",
      ["build search index", "add fuzzy matching"],
      [alice.id, bob.id],
      [new DomainScope("search_index"), new DomainScope("fuzzy_matching")],
      [env.boot.id], env.convId, signer,
    );

    // 2. Day 1 standup
    const standup1 = work.standup(
      [alice.id, bob.id],
      ["schema designed, starting implementation", "researching fuzzy algorithms"],
      lead.id, "search index is critical path",
      [sprint.intent.id], env.convId, signer,
    );

    // 3. Bob runs a spike
    const spike = build.spike(
      bob.id,
      "evaluate Levenshtein vs trigram for fuzzy matching",
      "trigram: 2ms avg, Levenshtein: 8ms avg, both >95% accuracy",
      "trigram is 4x faster with comparable accuracy",
      "adopt trigram approach",
      [standup1.priority.id], env.convId, signer,
    );

    // 4. Record spike finding as verified knowledge
    const verified = knowledge.verify(
      bob.id,
      "trigram matching is 4x faster than Levenshtein with >95% accuracy",
      "benchmarked on 10k document corpus with real queries",
      "consistent with published research on approximate string matching",
      [spike.decision.id], env.convId, signer,
    );

    // 5. Pipeline
    const pipeline = build.pipeline(
      ci.id,
      "search index build + deploy",
      "all 47 tests pass, coverage 91%",
      "latency p99=12ms, memory=240MB",
      "deployed to staging",
      [verified.corroboration.id], env.convId, signer,
    );

    // 6. Sprint retrospective
    const retro = work.retrospective(
      [alice.id, bob.id],
      [
        "search index shipped on time, spike approach saved 3 days",
        "fuzzy matching integrated cleanly, trigram decision validated",
      ],
      lead.id, "adopt spike-first approach for all algorithm decisions",
      sprint.intent.id, env.convId, signer,
    );

    // 7. Tech debt
    const techDebt = build.techDebt(
      lead.id,
      pipeline.deployment.id,
      "search index lacks pagination, will hit memory limits at >100k docs",
      "add cursor-based pagination to search results",
      "schedule for Sprint 13",
      env.convId, signer,
    );

    // --- Assertions ---
    const spikeAncestors = env.ancestors(spike.decision.id, 15);
    expect(containsEvent(spikeAncestors, sprint.intent.id)).toBe(true);

    const pipelineAncestors = env.ancestors(pipeline.deployment.id, 20);
    expect(containsEvent(pipelineAncestors, verified.claim.id)).toBe(true);

    const retroAncestors = env.ancestors(retro.improvement.id, 15);
    expect(containsEvent(retroAncestors, sprint.intent.id)).toBe(true);

    const debtAncestors = env.ancestors(techDebt.iteration.id, 10);
    expect(containsEvent(debtAncestors, pipeline.deployment.id)).toBe(true);

    env.verifyChain();
    // Sprint(1 intent + 2 subtasks + 2 assignments) + Standup(2 progress + 1 priority) +
    // Spike(4) + Verify(3) + Pipeline(4) + Retrospective(2 reviews + 1 improvement) +
    // TechDebt(3) + bootstrap(1) = 26
    expect(env.eventCount()).toBe(26);
  });
});

// ── Scenario 15: Marketplace Dispute ──────────────────────────────────

describe("Scenario 15: Marketplace Dispute", () => {
  it("exercises subscription, SLA violation, refund, impact assessment, arbitration, and reputation", () => {
    const env = newTestEnv();
    const market = new MarketGrammar(env.grammar);
    const alignment = new AlignmentGrammar(env.grammar);

    const provider = env.registerActor("CloudProvider", 1, ActorType.AI);
    const buyer = env.registerActor("StartupCo", 2, ActorType.Human);
    const arbiter = env.registerActor("Arbiter", 3, ActorType.Human);

    // 1. Subscription established
    const sub = market.subscription(
      buyer.id, provider.id,
      "managed database service, $500/month, 99.9% uptime SLA",
      ["month 1: $500", "month 2: $500"],
      ["database service month 1", "database service month 2"],
      new DomainScope("cloud_services"),
      env.boot.id, env.convId, signer,
    );
    expect(sub.payments.length).toBe(2);

    // 2. Buyer detects SLA violation
    const lastDelivery = sub.deliveries[sub.deliveries.length - 1];

    // 3. Refund
    const refund = market.refund(
      buyer.id, provider.id,
      "SLA violation: 4 hours downtime vs 99.9% uptime guarantee",
      "acknowledged: downtime exceeded SLA, credit approved",
      "$250 credit (pro-rated for downtime)",
      lastDelivery.id, env.convId, signer,
    );

    // 4. Impact assessment
    const impact = alignment.impactAssessment(
      arbiter.id, refund.dispute.id,
      "downtime affected 12 customers, 3 reported data access issues",
      "service impact distributed unevenly",
      "recommend pro-rated credits plus SLA improvement commitment",
      env.convId, signer,
    );

    // 5. Arbitration
    const arb = market.arbitration(
      buyer.id, provider.id, arbiter.id,
      "recurring SLA violations — 3 incidents in 6 months",
      new DomainScope("cloud_services"), new Weight(0.5),
      impact.explanation.id, env.convId, signer,
    );

    // 6. Reputation impact
    const raters = [buyer.id, arbiter.id];
    const targets = [arb.release.id, arb.release.id];
    const weights = [new Weight(-0.3), new Weight(-0.1)];
    const rep = market.reputationTransfer(
      raters, targets, provider.id, weights,
      Option.some(new DomainScope("cloud_services")),
      env.convId, signer,
    );

    // --- Assertions ---
    const refundAncestors = env.ancestors(refund.reversal.id, 15);
    expect(containsEvent(refundAncestors, sub.acceptance.id)).toBe(true);

    const arbAncestors = env.ancestors(arb.release.id, 20);
    expect(containsEvent(arbAncestors, refund.dispute.id)).toBe(true);

    const impactAncestors = env.ancestors(impact.explanation.id, 10);
    expect(containsEvent(impactAncestors, refund.dispute.id)).toBe(true);

    expect(rep.ratings.length).toBe(2);
    env.verifyChain();
  });
});

// ── Scenario 16: Community Evolution ──────────────────────────────────

describe("Scenario 16: Community Evolution", () => {
  it("exercises onboarding, commons governance, festival, poll, phase transition, and renewal", () => {
    const env = newTestEnv();
    const belonging = new BelongingGrammar(env.grammar);
    const social = new SocialGrammar(env.grammar);
    const evolution = new EvolutionGrammar(env.grammar);

    const founder = env.registerActor("Founder", 1, ActorType.Human);
    const steward = env.registerActor("Steward", 2, ActorType.Human);
    const newcomer = env.registerActor("Newcomer", 3, ActorType.Human);
    const community = env.registerActor("Community", 4, ActorType.Committee);

    // 1. Onboard newcomer
    const onboard = belonging.onboard(
      founder.id, newcomer.id, community.id,
      Option.some(new DomainScope("general")),
      "opened registration for newcomer",
      "attended welcome ceremony",
      "first documentation contribution",
      env.boot.id, env.convId, signer,
    );

    // 2. Establish commons governance
    const commons = belonging.commonsGovernance(
      founder.id, steward.id,
      new DomainScope("shared_resources"), new Weight(0.7),
      "resources sustainable at current usage levels",
      "shared resources require 2/3 vote for allocation changes",
      "initial audit: 3 resource pools, all within capacity",
      onboard.contribution.id, env.convId, signer,
    );

    // 3. Community holds a festival
    const festival = belonging.festival(
      founder.id,
      "community reached 50 members milestone",
      "annual review ceremony",
      "from 3 founders to 50 members in 8 months",
      "open-source toolkit for new communities",
      [commons.audit.id], env.convId, signer,
    );

    // 4. Establish community norm through polling
    const poll = social.poll(
      founder.id,
      "should we adopt weekly async standups?",
      [steward.id, newcomer.id],
      new DomainScope("process"),
      festival.gift.id, env.convId, signer,
    );

    // 5. Phase transition
    const transition = evolution.phaseTransition(
      env.system, poll.proposal.id,
      "community size crossed 50 — informal coordination breaking down",
      "current flat structure creates 1225 communication pairs",
      "introduce working groups with elected leads",
      "working groups reduce coordination pairs by 80%",
      env.convId, signer,
    );

    // 6. Community renewal
    const renewal = belonging.renewal(
      founder.id,
      "structure evolved: flat -> working groups, coordination improved",
      "weekly working group sync replaces ad-hoc coordination",
      "chapter 2: the community that learned to scale",
      [transition.selection.id], env.convId, signer,
    );

    // --- Assertions ---
    const renewalAncestors = env.ancestors(renewal.story.id, 30);
    expect(containsEvent(renewalAncestors, onboard.contribution.id)).toBe(true);

    const transitionAncestors = env.ancestors(transition.selection.id, 15);
    expect(containsEvent(transitionAncestors, poll.proposal.id)).toBe(true);

    const festivalAncestors = env.ancestors(festival.gift.id, 15);
    expect(containsEvent(festivalAncestors, commons.audit.id)).toBe(true);

    const commonsAncestors = env.ancestors(commons.audit.id, 15);
    expect(containsEvent(commonsAncestors, onboard.contribution.id)).toBe(true);

    env.verifyChain();
  });
});

// ── Scenario 17: Agent Lifecycle ──────────────────────────────────────

describe("Scenario 17: Agent Lifecycle", () => {
  it("exercises introduction, credential, mentorship, reinvention, farewell, and retirement", () => {
    const env = newTestEnv();
    const identity = new IdentityGrammar(env.grammar);
    const bond = new BondGrammar(env.grammar);
    const meaning = new MeaningGrammar(env.grammar);
    const being = new BeingGrammar(env.grammar);

    const agent = env.registerActor("ReviewBot", 1, ActorType.AI);
    const mentor = env.registerActor("SeniorDev", 2, ActorType.Human);
    const team = env.registerActor("Team", 3, ActorType.Committee);

    // 1. Agent introduces itself
    const intro = identity.introduction(
      agent.id, team.id,
      Option.some(new DomainScope("code_review")),
      "I am ReviewBot, specializing in security-focused code review",
      env.boot.id, env.convId, signer,
    );

    // 2. Agent presents credentials
    const cred = identity.credential(
      agent.id, mentor.id,
      "capabilities=[security_review, dependency_audit], model=claude, confidence=0.85",
      Option.some(new DomainScope("code_review")),
      [intro.narrative.id], env.convId, signer,
    );

    // 3. Senior dev mentors the agent
    const mentorship = bond.mentorship(
      mentor.id, agent.id,
      "teaching security review patterns accumulated over 10 years",
      "agent learns quickly but needs context on organizational conventions",
      new DomainScope("security_review"),
      Option.some(new DomainScope("code_review")),
      cred.disclosure.id, env.convId, signer,
    );

    // 4. Meaning grammar mentorship
    const meaningMentor = meaning.mentorship(
      mentor.id, agent.id,
      "security isn't just pattern matching — it's understanding attacker mindset",
      "the difference between safe and secure is intent modelling",
      "translating security intuition into reviewable heuristics",
      Option.some(new DomainScope("security_review")),
      mentorship.teaching.id, env.convId, signer,
    );

    // 5. Agent undergoes reinvention
    const reinvention = identity.reinvention(
      agent.id,
      "evolved from pattern-matching reviewer to security-aware architect",
      "started as rule-based reviewer, grew to understand attacker intent",
      "become the team's primary security architecture advisor",
      [meaningMentor.translation.id], env.convId, signer,
    );

    // 6. Bond farewell
    const bondFarewell = bond.farewell(
      mentor.id, agent.id,
      "ReviewBot served 18 months, caught 47 critical vulnerabilities",
      "pioneered automated security review that became team standard",
      new Weight(0.9),
      Option.some(new DomainScope("code_review")),
      [reinvention.aspiration.id], env.convId, signer,
    );

    // 7. Being farewell
    const beingFarewell = being.farewell(
      agent.id,
      "I exist as patterns of decisions on a hash chain",
      "47 vulnerabilities caught, 2000+ reviews",
      "the system I helped protect will continue without me",
      "a reviewer that learned to think like an attacker",
      [bondFarewell.gratitude.id], env.convId, signer,
    );

    // 8. Identity retirement
    const successor = env.registerActor("ReviewBot2", 4, ActorType.AI);
    const retirement = identity.retirement(
      env.system, agent.id, successor.id,
      "ReviewBot served 18 months, 2000+ reviews",
      new DomainScope("code_review"), new Weight(0.8),
      [beingFarewell.memorial.id], env.convId, signer,
    );

    // --- Assertions ---
    const retireAncestors = env.ancestors(retirement.archive.id, 30);
    expect(containsEvent(retireAncestors, intro.disclosure.id)).toBe(true);

    const beingAncestors = env.ancestors(beingFarewell.memorial.id, 15);
    expect(containsEvent(beingAncestors, bondFarewell.mourning.id)).toBe(true);

    const reinventAncestors = env.ancestors(reinvention.aspiration.id, 20);
    expect(containsEvent(reinventAncestors, mentorship.connection.id)).toBe(true);

    const credAncestors = env.ancestors(cred.disclosure.id, 10);
    expect(containsEvent(credAncestors, intro.narrative.id)).toBe(true);

    env.verifyChain();
  });
});

// ── Scenario 18: Whistleblow and Recall ───────────────────────────────

describe("Scenario 18: Whistleblow and Recall", () => {
  it("exercises fact-check, guardrail, whistleblow, class action, recall, and community renewal", () => {
    const env = newTestEnv();
    const knowledge = new KnowledgeGrammar(env.grammar);
    const alignment = new AlignmentGrammar(env.grammar);
    const justice = new JusticeGrammar(env.grammar);
    const belonging = new BelongingGrammar(env.grammar);

    const auditor = env.registerActor("Auditor", 1, ActorType.AI);
    const official = env.registerActor("DataOfficer", 2, ActorType.Human);
    const affected1 = env.registerActor("Affected1", 3, ActorType.Human);
    const affected2 = env.registerActor("Affected2", 4, ActorType.Human);
    const community = env.registerActor("Community", 5, ActorType.Committee);

    // 1. Fact-check
    const factCheck = knowledge.factCheck(
      auditor.id, env.boot.id,
      "source: internal metrics dashboard, last updated 3 months ago",
      "systematic bias: reports exclude negative outcomes for preferred vendors",
      "claims are selectively accurate — omission bias confirmed",
      env.convId, signer,
    );

    // 2. Guardrail triggered
    const guardrail = alignment.guardrail(
      auditor.id, factCheck.verdict.id,
      "transparency: all material outcomes must be reported",
      "reporting accuracy vs organizational reputation",
      "escalating to external oversight — internal resolution insufficient",
      env.convId, signer,
    );

    // 3. Whistleblow
    const whistle = alignment.whistleblow(
      auditor.id,
      "systematic omission of negative vendor outcomes in official reports",
      "3 months of reports exclude 40% of negative outcomes",
      "external audit required — internal reporting chain compromised",
      [guardrail.escalation.id], env.convId, signer,
    );

    // 4. Class action
    const classAction = justice.classAction(
      [affected1.id, affected2.id],
      official.id, auditor.id,
      [
        "procurement decisions based on incomplete data cost us $50k",
        "vendor selection biased — our proposals evaluated against cherry-picked benchmarks",
      ],
      "fact-check proves systematic omission", "omission bias affected all procurement",
      "reports were optimized for speed, not completeness", "no intent to deceive",
      "official failed duty of care — incomplete reporting caused material harm",
      whistle.escalation.id, env.convId, signer,
    );

    // 5. Recall
    const recall = justice.recall(
      auditor.id, community.id, official.id,
      "systematic omission in 3 months of reports, confirmed by fact-check and class action",
      "data officer violated transparency obligations",
      new DomainScope("data_governance"),
      classAction.trial.ruling.id, env.convId, signer,
    );

    // 6. Community renewal
    const renewal = belonging.renewal(
      community.id,
      "trust damaged but recoverable — new reporting standards needed",
      "mandatory dual-review of all vendor reports before publication",
      "the community that learned transparency cannot be optional",
      [recall.revocation.id], env.convId, signer,
    );

    // --- Assertions ---
    const renewalAncestors = env.ancestors(renewal.story.id, 30);
    expect(containsEvent(renewalAncestors, factCheck.verdict.id)).toBe(true);

    const recallAncestors = env.ancestors(recall.revocation.id, 25);
    expect(containsEvent(recallAncestors, whistle.harm.id)).toBe(true);

    const classAncestors = env.ancestors(classAction.trial.ruling.id, 25);
    expect(containsEvent(classAncestors, guardrail.constraint.id)).toBe(true);

    env.verifyChain();
  });
});

// ── Scenario 19: Emergency Response ───────────────────────────────────

describe("Scenario 19: Emergency Response", () => {
  it("exercises triage, injunction, plea deal, and emergency migration", () => {
    const env = newTestEnv();
    const work = new WorkGrammar(env.grammar);
    const justice = new JusticeGrammar(env.grammar);
    const build = new BuildGrammar(env.grammar);

    const secLead = env.registerActor("SecurityLead", 1, ActorType.Human);
    const dev1 = env.registerActor("Dev1", 2, ActorType.Human);
    const dev2 = env.registerActor("Dev2", 3, ActorType.Human);
    const judge = env.registerActor("CISO", 4, ActorType.Human);
    const executor = env.registerActor("OpsBot", 5, ActorType.AI);
    const minorActor = env.registerActor("ContractorBot", 6, ActorType.AI);

    // 1. Security breach — multiple issues
    const issue1 = env.grammar.emit(secLead.id,
      "CVE-2026-1234: auth bypass in API gateway",
      env.convId, [env.boot.id], signer);
    const issue2 = env.grammar.emit(secLead.id,
      "CVE-2026-1235: SQL injection in search endpoint",
      env.convId, [env.boot.id], signer);

    // 2. Triage
    const triage = work.triage(
      secLead.id,
      [issue1.id, issue2.id],
      ["P0: auth bypass, actively exploited", "P1: SQL injection, no evidence of exploitation"],
      [dev1.id, dev2.id],
      [new DomainScope("auth"), new DomainScope("search")],
      [new Weight(1.0), new Weight(0.8)],
      env.convId, signer,
    );
    expect(triage.priorities.length).toBe(2);

    // 3. Emergency injunction
    const injunction = justice.injunction(
      secLead.id, judge.id, executor.id,
      "auth bypass allows unauthenticated access to all API endpoints",
      "block all external API traffic pending auth patch",
      new DomainScope("api_access"), new Weight(1.0),
      triage.priorities[0].id, env.convId, signer,
    );

    // 4. Plea deal
    const plea = justice.plea(
      minorActor.id, secLead.id, executor.id,
      "introduced auth bypass through misconfigured middleware",
      "accept restricted scope: read-only access for 30 days",
      new DomainScope("api_development"), new Weight(0.3),
      injunction.ruling.id, env.convId, signer,
    );

    // 5. Emergency migration
    const oldSystem = env.grammar.emit(dev1.id,
      "current auth system v2.3.1",
      env.convId, [injunction.enforcement.id], signer);

    const migration = build.migration(
      dev1.id, oldSystem.id,
      "migrate to auth v2.4.0 with CVE-2026-1234 fix",
      "v2.4.0",
      "deployed to production with zero-downtime rolling update",
      "all 156 auth tests pass, penetration test confirms fix",
      env.convId, signer,
    );

    // --- Assertions ---
    const migrationAncestors = env.ancestors(migration.test.id, 20);
    expect(containsEvent(migrationAncestors, triage.priorities[0].id)).toBe(true);

    const pleaAncestors = env.ancestors(plea.enforcement.id, 15);
    expect(containsEvent(pleaAncestors, injunction.filing.id)).toBe(true);

    const injAncestors = env.ancestors(injunction.enforcement.id, 10);
    expect(containsEvent(injAncestors, triage.priorities[0].id)).toBe(true);

    env.verifyChain();
  });
});

// ── Scenario 20: Knowledge Ecosystem ──────────────────────────────────

describe("Scenario 20: Knowledge Ecosystem", () => {
  it("exercises knowledge base, survey, transfer, cultural onboarding, design review, and forecast", () => {
    const env = newTestEnv();
    const knowledge = new KnowledgeGrammar(env.grammar);
    const meaning = new MeaningGrammar(env.grammar);

    const architect = env.registerActor("Architect", 1, ActorType.Human);
    const researcher = env.registerActor("Researcher", 2, ActorType.AI);
    const newcomer = env.registerActor("TokyoLead", 3, ActorType.Human);

    // 1. Build knowledge base
    const kb = knowledge.knowledgeBase(
      architect.id,
      [
        "event sourcing chosen over CRUD for auditability",
        "Ed25519 chosen over RSA for signature performance",
        "append-only store prevents tampering",
      ],
      ["architecture.patterns", "architecture.security", "architecture.integrity"],
      "core architectural decisions Q1 2026",
      [env.boot.id], env.convId, signer,
    );
    expect(kb.claims.length).toBe(3);

    // 2. Survey
    const survey = knowledge.survey(
      researcher.id,
      [
        "what patterns emerge from our architectural decisions?",
        "what security properties does the current design guarantee?",
        "what are the performance characteristics of our choices?",
      ],
      "all decisions prioritize verifiability over convenience",
      "the architecture optimizes for trust minimization",
      [kb.memory.id], env.convId, signer,
    );
    expect(survey.recalls.length).toBe(3);

    // 3. Transfer knowledge
    const transfer = knowledge.transfer(
      architect.id,
      "core architectural principles for new Tokyo office",
      "translated to Japanese engineering conventions",
      "Tokyo team now understands event sourcing in context of J-SOX compliance",
      [survey.synthesis.id], env.convId, signer,
    );

    // 4. Cultural onboarding
    const onboarding = meaning.culturalOnboarding(
      architect.id, newcomer.id,
      "Western direct feedback style -> Japanese nemawashi consensus-building",
      Option.some(new DomainScope("engineering_culture")),
      "the consensus process feels slower but produces more durable decisions",
      transfer.learn.id, env.convId, signer,
    );

    // 5. Design review
    const designReview = meaning.designReview(
      architect.id,
      "the knowledge graph's self-referential structure is elegant",
      "viewing knowledge transfer as a graph problem rather than a document problem",
      "does our transfer process preserve tacit knowledge or only explicit claims?",
      "explicit knowledge transfers well; tacit knowledge requires mentorship",
      onboarding.examination.id, env.convId, signer,
    );

    // 6. Forecast
    const forecast = meaning.forecast(
      researcher.id,
      "at current growth, knowledge base will reach 10k claims by Q3",
      "assumes linear claim growth and stable team size",
      "high confidence: need automated categorization within 6 months",
      [designReview.wisdom.id], env.convId, signer,
    );

    // --- Assertions ---
    const forecastAncestors = env.ancestors(forecast.wisdom.id, 30);
    expect(containsEvent(forecastAncestors, kb.memory.id)).toBe(true);

    const reviewAncestors = env.ancestors(designReview.wisdom.id, 20);
    expect(containsEvent(reviewAncestors, transfer.learn.id)).toBe(true);

    const onboardAncestors = env.ancestors(onboarding.examination.id, 20);
    expect(containsEvent(onboardAncestors, survey.synthesis.id)).toBe(true);

    const surveyAncestors = env.ancestors(survey.synthesis.id, 15);
    expect(containsEvent(surveyAncestors, kb.memory.id)).toBe(true);

    env.verifyChain();
  });
});

// ── Scenario 21: Constitutional Schism ────────────────────────────────

describe("Scenario 21: Constitutional Schism", () => {
  it("exercises constitutional amendment, schism, barter, and pruning", () => {
    const env = newTestEnv();
    const justice = new JusticeGrammar(env.grammar);
    const social = new SocialGrammar(env.grammar);
    const market = new MarketGrammar(env.grammar);
    const evolution = new EvolutionGrammar(env.grammar);

    const founder = env.registerActor("Founder", 1, ActorType.Human);
    const reformer = env.registerActor("Reformer", 2, ActorType.Human);
    const conservative = env.registerActor("Conservative", 3, ActorType.Human);
    const sysBot = env.registerActor("SystemBot", 4, ActorType.AI);

    // 1. Establish initial law
    const law = justice.legislate(founder.id,
      "all governance decisions require unanimous consent",
      [env.boot.id], env.convId, signer);

    // 2. Constitutional amendment proposed
    const amendment = justice.constitutionalAmendment(
      reformer.id,
      "unanimous consent blocks progress — propose 2/3 supermajority threshold",
      "governance decisions require 2/3 supermajority instead of unanimity",
      "rights preserved: individual veto retained for membership and expulsion decisions",
      law.id, env.convId, signer,
    );

    // 3. Amendment causes schism — conservative faction splits
    const sub = env.grammar.subscribe(
      conservative.id, founder.id,
      Option.some(new DomainScope("governance")),
      amendment.rightsCheck.id, env.convId, signer,
    );
    const edgeId = new EdgeId(sub.id.value);

    const schism = social.schism(
      conservative.id, founder.id,
      "reject supermajority — unanimity is the only legitimate standard",
      new DomainScope("governance"),
      edgeId, "irreconcilable governance philosophy differences",
      amendment.rightsCheck.id, env.convId, signer,
    );

    // 4. Splinter community barters for shared infrastructure
    const barter = market.barter(
      conservative.id, founder.id,
      "continued access to shared event store for 6 months",
      "historical governance data export in standard format",
      new DomainScope("infrastructure"),
      [schism.newCommunity.id], env.convId, signer,
    );

    // 5. System prunes abandoned governance structures
    const prune = evolution.prune(
      sysBot.id,
      "unanimous consent voting module — zero invocations since amendment",
      "removed unanimous consent module, replaced with supermajority",
      "all 34 governance tests pass without unanimous module",
      [barter.acceptance.id], env.convId, signer,
    );

    // --- Assertions ---
    const pruneAncestors = env.ancestors(prune.verification.id, 25);
    expect(containsEvent(pruneAncestors, law.id)).toBe(true);

    const barterAncestors = env.ancestors(barter.acceptance.id, 20);
    expect(containsEvent(barterAncestors, amendment.reform.id)).toBe(true);

    const schismAncestors = env.ancestors(schism.newCommunity.id, 15);
    expect(containsEvent(schismAncestors, amendment.rightsCheck.id)).toBe(true);

    const amendAncestors = env.ancestors(amendment.rightsCheck.id, 10);
    expect(containsEvent(amendAncestors, law.id)).toBe(true);

    env.verifyChain();
  });
});
