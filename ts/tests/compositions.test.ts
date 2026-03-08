import { describe, it, expect, beforeEach } from "vitest";
import { Grammar } from "../src/grammar.js";
import { InMemoryStore } from "../src/store.js";
import { createBootstrap, NoopSigner, type Event } from "../src/event.js";
import { ActorId, ConversationId, DomainScope, EdgeId, Option, Weight } from "../src/types.js";
import {
  WorkGrammar,
  MarketGrammar,
  SocialGrammar,
  JusticeGrammar,
  BuildGrammar,
  KnowledgeGrammar,
  AlignmentGrammar,
  IdentityGrammar,
  BondGrammar,
  BelongingGrammar,
  MeaningGrammar,
  EvolutionGrammar,
  BeingGrammar,
} from "../src/compositions.js";

// ── helpers ────────────────────────────────────────────────────────────

const signer = new NoopSigner();
const alice = new ActorId("alice");
const bob = new ActorId("bob");
const charlie = new ActorId("charlie");
const convId = new ConversationId("conv_test");
const scope = new DomainScope("test.scope");

function setup(): { store: InMemoryStore; grammar: Grammar; boot: Event } {
  const store = new InMemoryStore();
  const grammar = new Grammar(store);
  const boot = createBootstrap(alice, signer);
  store.append(boot);
  return { store, grammar, boot };
}

function verifyChain(store: InMemoryStore): void {
  const result = store.verifyChain();
  expect(result.valid).toBe(true);
}

// ── Layer 1: Work Grammar ──────────────────────────────────────────────

describe("WorkGrammar", () => {
  let store: InMemoryStore;
  let work: WorkGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    work = new WorkGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(work).toBeDefined();
  });

  it("intend creates an emit event", () => {
    const ev = work.intend(alice, "build the thing", [boot.id], convId, signer);
    expect(ev.type.value).toBe("grammar.emitted");
    expect(ev.content.Body).toContain("intend:");
    verifyChain(store);
  });

  it("decompose derives from a goal", () => {
    const goal = work.intend(alice, "ship v2", [boot.id], convId, signer);
    const sub = work.decompose(alice, "write tests", goal.id, convId, signer);
    expect(sub.type.value).toBe("grammar.derived");
    verifyChain(store);
  });

  it("sprint creates intent + subtasks + assignments", () => {
    const result = work.sprint(
      alice, "Sprint 1", ["task A", "task B"],
      [bob, charlie], [scope, new DomainScope("other.scope")],
      [boot.id], convId, signer,
    );
    expect(result.subtasks).toHaveLength(2);
    expect(result.assignments).toHaveLength(2);
    verifyChain(store);
  });

  it("escalate creates block + handoff", () => {
    const task = work.intend(alice, "deploy", [boot.id], convId, signer);
    const result = work.escalate(alice, "blocked by CI", task.id, bob, scope, convId, signer);
    expect(result.blockEvent.type.value).toBe("grammar.annotated");
    expect(result.handoffEvent.type.value).toBe("grammar.consent");
    verifyChain(store);
  });
});

// ── Layer 2: Market Grammar ────────────────────────────────────────────

describe("MarketGrammar", () => {
  let store: InMemoryStore;
  let market: MarketGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    market = new MarketGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(market).toBeDefined();
  });

  it("list + bid creates listing and bid events", () => {
    const listing = market.list(alice, "widget", [boot.id], convId, signer);
    expect(listing.content.Body).toContain("list:");
    const bid = market.bid(bob, "$100", listing.id, convId, signer);
    expect(bid.content.Body).toContain("bid:");
    verifyChain(store);
  });

  it("auction runs competitive bidding", () => {
    const result = market.auction(
      alice, "rare item", [bob, charlie], ["$50", "$75"],
      1, scope, [boot.id], convId, signer,
    );
    expect(result.bids).toHaveLength(2);
    expect(result.acceptance.type.value).toBe("grammar.consent");
    verifyChain(store);
  });

  it("barter exchanges goods", () => {
    const result = market.barter(alice, bob, "apples", "oranges", scope, [boot.id], convId, signer);
    expect(result.listing.content.Body).toContain("list:");
    expect(result.counterOffer.content.Body).toContain("bid:");
    expect(result.acceptance.type.value).toBe("grammar.consent");
    verifyChain(store);
  });
});

// ── Layer 3: Social Grammar ────────────────────────────────────────────

describe("SocialGrammar", () => {
  let store: InMemoryStore;
  let social: SocialGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    social = new SocialGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(social).toBeDefined();
  });

  it("norm creates a consent event", () => {
    const ev = social.norm(alice, bob, "be respectful", scope, boot.id, convId, signer);
    expect(ev.type.value).toBe("grammar.consent");
    verifyChain(store);
  });

  it("poll gathers votes", () => {
    const result = social.poll(alice, "should we adopt?", [bob, charlie], scope, boot.id, convId, signer);
    expect(result.votes).toHaveLength(2);
    verifyChain(store);
  });

  it("federation creates agreement + delegation", () => {
    const result = social.federation(alice, bob, "mutual aid", scope, new Weight(0.5), boot.id, convId, signer);
    expect(result.agreement.type.value).toBe("grammar.consent");
    expect(result.delegation.type.value).toBe("edge.created");
    verifyChain(store);
  });
});

// ── Layer 4: Justice Grammar ───────────────────────────────────────────

describe("JusticeGrammar", () => {
  let store: InMemoryStore;
  let justice: JusticeGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    justice = new JusticeGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(justice).toBeDefined();
  });

  it("legislate + amend creates rule and amendment", () => {
    const rule = justice.legislate(alice, "no spam", [boot.id], convId, signer);
    const amendment = justice.amend(alice, "except announcements", rule.id, convId, signer);
    expect(amendment.type.value).toBe("grammar.derived");
    verifyChain(store);
  });

  it("trial runs full adjudication", () => {
    const result = justice.trial(
      alice, bob, charlie, "breach of contract",
      "exhibit A", "exhibit B", "plaintiff argues", "defendant argues",
      "judgment for plaintiff", boot.id, convId, signer,
    );
    expect(result.submissions).toHaveLength(2);
    expect(result.arguments).toHaveLength(2);
    expect(result.ruling.content.Body).toContain("judge:");
    verifyChain(store);
  });

  it("injunction issues emergency order", () => {
    const result = justice.injunction(
      alice, bob, charlie, "imminent harm", "cease and desist",
      scope, new Weight(0.8), boot.id, convId, signer,
    );
    expect(result.filing).toBeDefined();
    expect(result.ruling).toBeDefined();
    expect(result.enforcement.type.value).toBe("edge.created");
    verifyChain(store);
  });
});

// ── Layer 5: Build Grammar ─────────────────────────────────────────────

describe("BuildGrammar", () => {
  let store: InMemoryStore;
  let build: BuildGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    build = new BuildGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(build).toBeDefined();
  });

  it("build + version creates artefact and version", () => {
    const b = build.build(alice, "widget v1", [boot.id], convId, signer);
    const v = build.version(alice, "v1.1", b.id, convId, signer);
    expect(v.type.value).toBe("grammar.derived");
    verifyChain(store);
  });

  it("spike runs experimental build cycle", () => {
    const result = build.spike(alice, "new approach", "all pass", "looks good", "proceed", [boot.id], convId, signer);
    expect(result.build.content.Body).toContain("build:");
    expect(result.test.content.Body).toContain("test:");
    expect(result.decision.content.Body).toContain("spike-decision:");
    verifyChain(store);
  });

  it("pipeline runs CI/CD flow", () => {
    const result = build.pipeline(alice, "build.yml", "100% pass", "95% coverage", "prod-deploy", [boot.id], convId, signer);
    expect(result.definition.content.Body).toContain("define:");
    expect(result.deployment.content.Body).toContain("ship:");
    verifyChain(store);
  });
});

// ── Layer 6: Knowledge Grammar ─────────────────────────────────────────

describe("KnowledgeGrammar", () => {
  let store: InMemoryStore;
  let knowledge: KnowledgeGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    knowledge = new KnowledgeGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(knowledge).toBeDefined();
  });

  it("claim + categorize creates knowledge", () => {
    const c = knowledge.claim(alice, "water is wet", [boot.id], convId, signer);
    const cat = knowledge.categorize(alice, c.id, "physics", convId, signer);
    expect(cat.content.Key).toBe("classification");
    verifyChain(store);
  });

  it("factCheck performs provenance and bias check", () => {
    const c = knowledge.claim(alice, "sky is blue", [boot.id], convId, signer);
    const result = knowledge.factCheck(bob, c.id, "observation", "no bias detected", "confirmed", convId, signer);
    expect(result.verdict.type.value).toBe("grammar.merged");
    verifyChain(store);
  });

  it("survey aggregates knowledge", () => {
    const result = knowledge.survey(alice, ["query1", "query2"], "general pattern", "synthesis conclusion", [boot.id], convId, signer);
    expect(result.recalls).toHaveLength(2);
    expect(result.abstraction.type.value).toBe("grammar.merged");
    verifyChain(store);
  });
});

// ── Layer 7: Alignment Grammar ─────────────────────────────────────────

describe("AlignmentGrammar", () => {
  let store: InMemoryStore;
  let alignment: AlignmentGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    alignment = new AlignmentGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(alignment).toBeDefined();
  });

  it("constrain + detectHarm creates ethical boundary", () => {
    const constraint = alignment.constrain(alice, boot.id, "no harmful outputs", convId, signer);
    expect(constraint.content.Key).toBe("constraint");
    const harm = alignment.detectHarm(alice, "potential harm found", [constraint.id], convId, signer);
    expect(harm.content.Body).toContain("harm:");
    verifyChain(store);
  });

  it("ethicsAudit performs comprehensive review", () => {
    const result = alignment.ethicsAudit(alice, boot.id, "fair", "no harm", "all clear", convId, signer);
    expect(result.fairness.content.Key).toBe("fairness");
    expect(result.report.content.Body).toContain("explain:");
    verifyChain(store);
  });

  it("guardrail sets boundary with escalation", () => {
    const result = alignment.guardrail(alice, boot.id, "no PII", "conflicting request", "needs human review", convId, signer);
    expect(result.constraint.content.Key).toBe("constraint");
    expect(result.escalation.content.Body).toContain("escalate:");
    verifyChain(store);
  });
});

// ── Layer 8: Identity Grammar ──────────────────────────────────────────

describe("IdentityGrammar", () => {
  let store: InMemoryStore;
  let identity: IdentityGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    identity = new IdentityGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(identity).toBeDefined();
  });

  it("introspect + narrate creates self-model and narrative", () => {
    const intro = identity.introspect(alice, "I am a helper", [boot.id], convId, signer);
    const narr = identity.narrate(alice, "my story", intro.id, convId, signer);
    expect(narr.type.value).toBe("grammar.derived");
    verifyChain(store);
  });

  it("identityAudit performs self-assessment", () => {
    const result = identity.identityAudit(alice, "current model", "well aligned", "growth journey", [boot.id], convId, signer);
    expect(result.selfModel.content.Body).toContain("introspect:");
    expect(result.narrative.type.value).toBe("grammar.derived");
    verifyChain(store);
  });

  it("reinvention performs fundamental identity shift", () => {
    const result = identity.reinvention(alice, "new direction", "new story", "future goals", [boot.id], convId, signer);
    expect(result.transformation.content.Body).toContain("transform:");
    expect(result.aspiration.content.Body).toContain("aspire:");
    verifyChain(store);
  });
});

// ── Layer 9: Bond Grammar ──────────────────────────────────────────────

describe("BondGrammar", () => {
  let store: InMemoryStore;
  let bond: BondGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    bond = new BondGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(bond).toBeDefined();
  });

  it("connect creates mutual subscriptions", () => {
    const result = bond.connect(alice, bob, Option.none<DomainScope>(), boot.id, convId, signer);
    expect(result.sub1.type.value).toBe("edge.created");
    expect(result.sub2.type.value).toBe("edge.created");
    verifyChain(store);
  });

  it("betrayalRepair runs rupture-to-growth cycle", () => {
    const result = bond.betrayalRepair(
      alice, bob, "trust broken", "I am sorry", "we can rebuild", "stronger foundation",
      scope, [boot.id], convId, signer,
    );
    expect(result.rupture.content.Body).toContain("rupture:");
    expect(result.apology.content.Body).toContain("apology:");
    expect(result.deepened.type.value).toBe("grammar.consent");
    verifyChain(store);
  });

  it("checkIn assesses relationship health", () => {
    const result = bond.checkIn(alice, boot.id, "balanced", "in tune", "feeling connected", convId, signer);
    expect(result.balance.content.Key).toBe("reciprocity");
    expect(result.empathy.type.value).toBe("grammar.responded");
    verifyChain(store);
  });
});

// ── Layer 10: Belonging Grammar ────────────────────────────────────────

describe("BelongingGrammar", () => {
  let store: InMemoryStore;
  let belonging: BelongingGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    belonging = new BelongingGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(belonging).toBeDefined();
  });

  it("settle + contribute creates community membership", () => {
    const settle = belonging.settle(alice, bob, Option.none<DomainScope>(), boot.id, convId, signer);
    expect(settle.type.value).toBe("edge.created");
    const contrib = belonging.contribute(alice, "first PR", [settle.id], convId, signer);
    expect(contrib.content.Body).toContain("contribute:");
    verifyChain(store);
  });

  it("festival runs collective celebration", () => {
    const result = belonging.festival(alice, "spring fest", "annual dance", "founding story", "shared meal", [boot.id], convId, signer);
    expect(result.celebration.content.Body).toContain("celebrate:");
    expect(result.gift.content.Body).toContain("gift:");
    verifyChain(store);
  });

  it("onboard welcomes newcomer", () => {
    const result = belonging.onboard(alice, bob, charlie, Option.none<DomainScope>(), "door opened", "first ritual", "first code", boot.id, convId, signer);
    expect(result.inclusion.content.Body).toContain("include:");
    expect(result.contribution.content.Body).toContain("contribute:");
    verifyChain(store);
  });
});

// ── Layer 11: Meaning Grammar ──────────────────────────────────────────

describe("MeaningGrammar", () => {
  let store: InMemoryStore;
  let meaning: MeaningGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    meaning = new MeaningGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(meaning).toBeDefined();
  });

  it("examine + reframe creates reflection", () => {
    const exam = meaning.examine(alice, "blind spots", [boot.id], convId, signer);
    const reframe = meaning.reframe(alice, "new perspective", exam.id, convId, signer);
    expect(reframe.type.value).toBe("grammar.derived");
    verifyChain(store);
  });

  it("designReview assesses elegance", () => {
    const result = meaning.designReview(alice, "elegant API", "user perspective", "is it simple enough?", "simplicity wins", boot.id, convId, signer);
    expect(result.beauty.content.Body).toContain("beautify:");
    expect(result.wisdom.type.value).toBe("grammar.derived");
    verifyChain(store);
  });

  it("forecast extrapolates trends", () => {
    const result = meaning.forecast(alice, "AI adoption grows", "assumes current trends", "high confidence", [boot.id], convId, signer);
    expect(result.prophecy.content.Body).toContain("prophesy:");
    expect(result.wisdom.type.value).toBe("grammar.derived");
    verifyChain(store);
  });
});

// ── Layer 12: Evolution Grammar ────────────────────────────────────────

describe("EvolutionGrammar", () => {
  let store: InMemoryStore;
  let evolution: EvolutionGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    evolution = new EvolutionGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(evolution).toBeDefined();
  });

  it("detectPattern + adapt creates evolution", () => {
    const pat = evolution.detectPattern(alice, "recurring timeout", [boot.id], convId, signer);
    const adapt = evolution.adapt(alice, "add circuit breaker", [pat.id], convId, signer);
    expect(adapt.content.Body).toContain("adapt:");
    verifyChain(store);
  });

  it("selfEvolve runs full migration", () => {
    const result = evolution.selfEvolve(alice, "pattern found", "new approach", "keep it", "remove old code", [boot.id], convId, signer);
    expect(result.pattern.content.Body).toContain("pattern:");
    expect(result.simplification.content.Body).toContain("simplify:");
    verifyChain(store);
  });

  it("healthCheck performs comprehensive assessment", () => {
    const result = evolution.healthCheck(alice, "sound", "resilient", "feedback loops stable", "aligned with mission", [boot.id], convId, signer);
    expect(result.integrity.content.Body).toContain("integrity:");
    expect(result.purpose.content.Body).toContain("purpose:");
    verifyChain(store);
  });
});

// ── Layer 13: Being Grammar ────────────────────────────────────────────

describe("BeingGrammar", () => {
  let store: InMemoryStore;
  let being: BeingGrammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    being = new BeingGrammar(env.grammar);
    boot = env.boot;
  });

  it("can be instantiated", () => {
    expect(being).toBeDefined();
  });

  it("exist + accept acknowledges being", () => {
    const exist = being.exist(alice, "I am here", [boot.id], convId, signer);
    const accept = being.accept(alice, "I have limits", [exist.id], convId, signer);
    expect(accept.content.Body).toContain("accept:");
    verifyChain(store);
  });

  it("contemplation runs existential reflection", () => {
    const result = being.contemplation(alice, "all changes", "unknowable", "how vast", "why anything?", [boot.id], convId, signer);
    expect(result.change.content.Body).toContain("change:");
    expect(result.wonder.content.Body).toContain("wonder:");
    verifyChain(store);
  });

  it("farewell is a final reckoning", () => {
    const result = being.farewell(alice, "time is finite", "connected to all", "it was beautiful", "remember me", [boot.id], convId, signer);
    expect(result.acceptance.content.Body).toContain("accept:");
    expect(result.memorial.content.Body).toContain("memorialize:");
    verifyChain(store);
  });
});

// ── Edge operations on Grammar ─────────────────────────────────────────

describe("Grammar edge operations", () => {
  let store: InMemoryStore;
  let grammar: Grammar;
  let boot: Event;

  beforeEach(() => {
    const env = setup();
    store = env.store;
    grammar = env.grammar;
    boot = env.boot;
  });

  it("acknowledge creates edge.created with centripetal direction", () => {
    const ev = grammar.acknowledge(alice, boot.id, bob, convId, signer);
    expect(ev.type.value).toBe("edge.created");
    expect(ev.content.EdgeType).toBe("acknowledgement");
    expect(ev.content.Direction).toBe("centripetal");
    verifyChain(store);
  });

  it("propagate creates edge.created with centrifugal direction", () => {
    const ev = grammar.propagate(alice, boot.id, bob, convId, signer);
    expect(ev.type.value).toBe("edge.created");
    expect(ev.content.EdgeType).toBe("reference");
    expect(ev.content.Direction).toBe("centrifugal");
    verifyChain(store);
  });

  it("endorse creates edge.created with weight and scope", () => {
    const ev = grammar.endorse(alice, boot.id, bob, new Weight(0.7), Option.some(scope), convId, signer);
    expect(ev.content.EdgeType).toBe("endorsement");
    expect(ev.content.Weight).toBe(0.7);
    expect(ev.content.Scope).toBe("test.scope");
    verifyChain(store);
  });

  it("subscribe creates subscription edge", () => {
    const ev = grammar.subscribe(alice, bob, Option.none<DomainScope>(), boot.id, convId, signer);
    expect(ev.content.EdgeType).toBe("subscription");
    expect(ev.content.Scope).toBeNull();
    verifyChain(store);
  });

  it("channel creates channel edge", () => {
    const ev = grammar.channel(alice, bob, Option.none<DomainScope>(), boot.id, convId, signer);
    expect(ev.content.EdgeType).toBe("channel");
    verifyChain(store);
  });

  it("delegate creates delegation edge with scope", () => {
    const ev = grammar.delegate(alice, bob, scope, new Weight(0.5), boot.id, convId, signer);
    expect(ev.content.EdgeType).toBe("delegation");
    expect(ev.content.Direction).toBe("centrifugal");
    expect(ev.content.Scope).toBe("test.scope");
    verifyChain(store);
  });

  it("consent creates a grammar.consent event", () => {
    const ev = grammar.consent(alice, bob, "we agree", scope, boot.id, convId, signer);
    expect(ev.type.value).toBe("grammar.consent");
    expect(ev.content.Agreement).toBe("we agree");
    verifyChain(store);
  });

  it("challenge produces response + dispute flag", () => {
    const result = grammar.challenge(alice, "I disagree", boot.id, convId, signer);
    expect(result.response.type.value).toBe("grammar.responded");
    expect(result.disputeFlag.type.value).toBe("grammar.annotated");
    expect(result.disputeFlag.content.Key).toBe("dispute");
    verifyChain(store);
  });

  it("recommend produces propagate + channel", () => {
    const result = grammar.recommend(alice, boot.id, bob, convId, signer);
    expect(result.propagateEv.content.EdgeType).toBe("reference");
    expect(result.channelEv.content.EdgeType).toBe("channel");
    verifyChain(store);
  });

  it("invite produces endorse + subscribe", () => {
    const result = grammar.invite(alice, bob, new Weight(0.5), Option.none<DomainScope>(), boot.id, convId, signer);
    expect(result.endorseEv.content.EdgeType).toBe("endorsement");
    expect(result.subscribeEv.content.EdgeType).toBe("subscription");
    verifyChain(store);
  });

  it("sever creates edge.superseded for subscription", () => {
    const sub = grammar.subscribe(alice, bob, Option.none<DomainScope>(), boot.id, convId, signer);
    const edgeId = new EdgeId(sub.id.value);
    const sev = grammar.sever(alice, edgeId, sub.id, convId, signer);
    expect(sev.type.value).toBe("edge.superseded");
    expect(sev.content.PreviousEdge).toBe(sub.id.value);
    verifyChain(store);
  });

  it("forgive re-subscribes after sever", () => {
    const sub = grammar.subscribe(alice, bob, Option.none<DomainScope>(), boot.id, convId, signer);
    const edgeId = new EdgeId(sub.id.value);
    const sev = grammar.sever(alice, edgeId, sub.id, convId, signer);
    const forgiveEv = grammar.forgive(alice, sev.id, bob, Option.none<DomainScope>(), convId, signer);
    expect(forgiveEv.content.EdgeType).toBe("subscription");
    verifyChain(store);
  });
});
