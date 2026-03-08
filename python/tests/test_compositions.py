"""Tests for Composition Grammars — all 13 layers."""

import pytest

from eventgraph.event import NoopSigner, create_bootstrap
from eventgraph.grammar import Grammar
from eventgraph.store import InMemoryStore
from eventgraph.types import ActorID, ConversationID, DomainScope, EdgeID, Option, Weight
from eventgraph.compositions import (
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
)


ALICE = ActorID("alice")
BOB = ActorID("bob")
CHARLIE = ActorID("charlie")
SYSTEM = ActorID("system")
CONV = ConversationID("conv_test")
SIGNER = NoopSigner()
SCOPE = DomainScope("test_scope")
NO_SCOPE: Option[DomainScope] = Option.none()
SOME_SCOPE: Option[DomainScope] = Option.some(SCOPE)


def _setup():
    """Create a store with a bootstrap event and return (store, grammar, boot)."""
    store = InMemoryStore()
    boot = create_bootstrap(source=ALICE, signer=SIGNER)
    store.append(boot)
    grammar = Grammar(store)
    return store, grammar, boot


def _verify_chain(store):
    """Verify hash chain integrity."""
    verification = store.verify_chain()
    assert verification.valid is True


# =============================================================================
# Layer 1: WorkGrammar
# =============================================================================

class TestWorkGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        work = WorkGrammar(grammar)
        assert work is not None

    def test_intend(self):
        store, grammar, boot = _setup()
        work = WorkGrammar(grammar)
        goal = work.intend(ALICE, "ship v2.0", [boot.id], CONV, SIGNER)
        assert goal.content["Body"] == "intend: ship v2.0"
        _verify_chain(store)

    def test_decompose(self):
        store, grammar, boot = _setup()
        work = WorkGrammar(grammar)
        goal = work.intend(ALICE, "ship v2.0", [boot.id], CONV, SIGNER)
        sub = work.decompose(ALICE, "update auth", goal.id, CONV, SIGNER)
        assert sub.content["Body"] == "decompose: update auth"
        assert sub.causes[0].value == goal.id.value
        _verify_chain(store)

    def test_assign_and_claim(self):
        store, grammar, boot = _setup()
        work = WorkGrammar(grammar)
        goal = work.intend(ALICE, "fix bug", [boot.id], CONV, SIGNER)
        assign_ev = work.assign(ALICE, BOB, SCOPE, Weight(0.5), goal.id, CONV, SIGNER)
        claim_ev = work.claim(BOB, "taking bug fix", [assign_ev.id], CONV, SIGNER)
        assert claim_ev.source.value == "bob"
        _verify_chain(store)

    def test_progress_and_complete(self):
        store, grammar, boot = _setup()
        work = WorkGrammar(grammar)
        task = work.intend(ALICE, "implement search", [boot.id], CONV, SIGNER)
        p1 = work.progress(ALICE, "basic working", task.id, CONV, SIGNER)
        p2 = work.progress(ALICE, "added fuzzy", p1.id, CONV, SIGNER)
        complete = work.complete(ALICE, "search done", [p2.id], CONV, SIGNER)
        assert complete.content["Body"] == "complete: search done"
        _verify_chain(store)

    def test_standup(self):
        store, grammar, boot = _setup()
        work = WorkGrammar(grammar)
        result = work.standup(
            [ALICE, BOB], ["finished auth", "started tests"],
            CHARLIE, "focus on coverage",
            [boot.id], CONV, SIGNER,
        )
        assert len(result.updates) == 2
        assert result.priority is not None
        _verify_chain(store)

    def test_sprint(self):
        store, grammar, boot = _setup()
        work = WorkGrammar(grammar)
        result = work.sprint(
            ALICE, "Sprint 7",
            ["add rate limiting", "add 2FA"],
            [BOB, CHARLIE],
            [DomainScope("rate_limiting"), DomainScope("two_factor")],
            [boot.id], CONV, SIGNER,
        )
        assert len(result.subtasks) == 2
        assert len(result.assignments) == 2
        _verify_chain(store)

    def test_escalate(self):
        store, grammar, boot = _setup()
        work = WorkGrammar(grammar)
        task = work.intend(ALICE, "migrate database", [boot.id], CONV, SIGNER)
        result = work.escalate(ALICE, "need DBA approval", task.id, BOB, SCOPE, CONV, SIGNER)
        assert result.block_event is not None
        assert result.handoff_event is not None
        _verify_chain(store)

    def test_delegate_and_verify(self):
        store, grammar, boot = _setup()
        work = WorkGrammar(grammar)
        result = work.delegate_and_verify(ALICE, BOB, SCOPE, Weight(0.7), boot.id, CONV, SIGNER)
        assert result.assign_event is not None
        assert result.scope_event is not None
        _verify_chain(store)

    def test_retrospective(self):
        store, grammar, boot = _setup()
        work = WorkGrammar(grammar)
        task = work.intend(ALICE, "sprint 5", [boot.id], CONV, SIGNER)
        result = work.retrospective(
            [ALICE, BOB], ["CI was slow", "pairing worked"],
            CHARLIE, "invest in CI speed",
            task.id, CONV, SIGNER,
        )
        assert len(result.reviews) == 2
        assert result.improvement is not None
        _verify_chain(store)


# =============================================================================
# Layer 2: MarketGrammar
# =============================================================================

class TestMarketGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        market = MarketGrammar(grammar)
        assert market is not None

    def test_list_and_bid(self):
        store, grammar, boot = _setup()
        market = MarketGrammar(grammar)
        listing = market.list_offering(ALICE, "widget", [boot.id], CONV, SIGNER)
        bid_ev = market.bid(BOB, "$50", listing.id, CONV, SIGNER)
        assert listing.content["Body"] == "list: widget"
        assert bid_ev.content["Body"] == "bid: $50"
        _verify_chain(store)

    def test_auction(self):
        store, grammar, boot = _setup()
        market = MarketGrammar(grammar)
        result = market.auction(
            ALICE, "rare widget",
            [BOB, CHARLIE], ["$50", "$75"],
            1, SCOPE,
            [boot.id], CONV, SIGNER,
        )
        assert len(result.bids) == 2
        assert result.acceptance is not None
        _verify_chain(store)

    def test_barter(self):
        store, grammar, boot = _setup()
        market = MarketGrammar(grammar)
        result = market.barter(ALICE, BOB, "apples", "oranges", SCOPE, [boot.id], CONV, SIGNER)
        assert result.listing is not None
        assert result.counter_offer is not None
        assert result.acceptance is not None
        _verify_chain(store)

    def test_refund(self):
        store, grammar, boot = _setup()
        market = MarketGrammar(grammar)
        listing = market.list_offering(ALICE, "product", [boot.id], CONV, SIGNER)
        result = market.refund(BOB, ALICE, "defective", "agreed", "$50", listing.id, CONV, SIGNER)
        assert result.dispute is not None
        assert result.reversal is not None
        _verify_chain(store)

    def test_arbitration(self):
        store, grammar, boot = _setup()
        market = MarketGrammar(grammar)
        listing = market.list_offering(ALICE, "service", [boot.id], CONV, SIGNER)
        result = market.arbitration(
            BOB, ALICE, CHARLIE, "poor service",
            SCOPE, Weight(0.5),
            listing.id, CONV, SIGNER,
        )
        assert result.dispute is not None
        assert result.escrow is not None
        assert result.release is not None
        _verify_chain(store)


# =============================================================================
# Layer 3: SocialGrammar
# =============================================================================

class TestSocialGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        social = SocialGrammar(grammar)
        assert social is not None

    def test_norm(self):
        store, grammar, boot = _setup()
        social = SocialGrammar(grammar)
        ev = social.norm(ALICE, BOB, "be kind", SCOPE, boot.id, CONV, SIGNER)
        assert ev.type.value == "grammar.consent"
        _verify_chain(store)

    def test_welcome(self):
        store, grammar, boot = _setup()
        social = SocialGrammar(grammar)
        endorse_ev, sub_ev = social.welcome(ALICE, BOB, Weight(0.5), NO_SCOPE, boot.id, CONV, SIGNER)
        assert endorse_ev.type.value == "edge.created"
        assert sub_ev.type.value == "edge.created"
        _verify_chain(store)

    def test_poll(self):
        store, grammar, boot = _setup()
        social = SocialGrammar(grammar)
        result = social.poll(ALICE, "should we adopt policy X?", [BOB, CHARLIE], SCOPE, boot.id, CONV, SIGNER)
        assert len(result.votes) == 2
        _verify_chain(store)

    def test_federation(self):
        store, grammar, boot = _setup()
        social = SocialGrammar(grammar)
        result = social.federation(ALICE, BOB, "cooperation", SCOPE, Weight(0.5), boot.id, CONV, SIGNER)
        assert result.agreement is not None
        assert result.delegation is not None
        _verify_chain(store)


# =============================================================================
# Layer 4: JusticeGrammar
# =============================================================================

class TestJusticeGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        justice = JusticeGrammar(grammar)
        assert justice is not None

    def test_legislate_and_amend(self):
        store, grammar, boot = _setup()
        justice = JusticeGrammar(grammar)
        rule = justice.legislate(ALICE, "no spam", [boot.id], CONV, SIGNER)
        amendment = justice.amend(ALICE, "except announcements", rule.id, CONV, SIGNER)
        assert rule.content["Body"] == "legislate: no spam"
        assert amendment.content["Body"] == "amend: except announcements"
        _verify_chain(store)

    def test_trial(self):
        store, grammar, boot = _setup()
        justice = JusticeGrammar(grammar)
        rule = justice.legislate(ALICE, "no fraud", [boot.id], CONV, SIGNER)
        result = justice.trial(
            BOB, CHARLIE, ALICE,
            "fraud committed", "evidence A", "evidence B",
            "plaintiff argues", "defendant argues", "guilty",
            rule.id, CONV, SIGNER,
        )
        assert result.filing is not None
        assert len(result.submissions) == 2
        assert len(result.arguments) == 2
        assert result.ruling is not None
        _verify_chain(store)

    def test_injunction(self):
        store, grammar, boot = _setup()
        justice = JusticeGrammar(grammar)
        rule = justice.legislate(ALICE, "safety first", [boot.id], CONV, SIGNER)
        result = justice.injunction(
            BOB, ALICE, CHARLIE,
            "immediate danger", "cease and desist",
            SCOPE, Weight(0.8),
            rule.id, CONV, SIGNER,
        )
        assert result.filing is not None
        assert result.ruling is not None
        assert result.enforcement is not None
        _verify_chain(store)


# =============================================================================
# Layer 5: BuildGrammar
# =============================================================================

class TestBuildGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        build = BuildGrammar(grammar)
        assert build is not None

    def test_build_and_ship(self):
        store, grammar, boot = _setup()
        build = BuildGrammar(grammar)
        build_ev = build.build(ALICE, "eventgraph v1", [boot.id], CONV, SIGNER)
        ship_ev = build.ship(ALICE, "production", [build_ev.id], CONV, SIGNER)
        assert build_ev.content["Body"] == "build: eventgraph v1"
        assert ship_ev.content["Body"] == "ship: production"
        _verify_chain(store)

    def test_spike(self):
        store, grammar, boot = _setup()
        build = BuildGrammar(grammar)
        result = build.spike(
            ALICE, "new cache", "all pass", "looks good", "proceed",
            [boot.id], CONV, SIGNER,
        )
        assert result.build is not None
        assert result.test is not None
        assert result.feedback is not None
        assert result.decision is not None
        _verify_chain(store)

    def test_pipeline(self):
        store, grammar, boot = _setup()
        build = BuildGrammar(grammar)
        result = build.pipeline(
            ALICE, "CI/CD pipeline", "all pass", "100% coverage", "deployed",
            [boot.id], CONV, SIGNER,
        )
        assert result.definition is not None
        assert result.deployment is not None
        _verify_chain(store)

    def test_tech_debt(self):
        store, grammar, boot = _setup()
        build = BuildGrammar(grammar)
        build_ev = build.build(ALICE, "module", [boot.id], CONV, SIGNER)
        result = build.tech_debt(ALICE, build_ev.id, "B+", "coupling too high", "refactor", CONV, SIGNER)
        assert result.measure is not None
        assert result.debt_mark is not None
        assert result.iteration is not None
        _verify_chain(store)


# =============================================================================
# Layer 6: KnowledgeGrammar
# =============================================================================

class TestKnowledgeGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        knowledge = KnowledgeGrammar(grammar)
        assert knowledge is not None

    def test_claim_and_categorize(self):
        store, grammar, boot = _setup()
        knowledge = KnowledgeGrammar(grammar)
        claim = knowledge.claim(ALICE, "earth is round", [boot.id], CONV, SIGNER)
        cat = knowledge.categorize(ALICE, claim.id, "science.astronomy", CONV, SIGNER)
        assert claim.content["Body"] == "claim: earth is round"
        assert cat.content["Key"] == "classification"
        _verify_chain(store)

    def test_fact_check(self):
        store, grammar, boot = _setup()
        knowledge = KnowledgeGrammar(grammar)
        claim = knowledge.claim(ALICE, "sky is blue", [boot.id], CONV, SIGNER)
        result = knowledge.fact_check(
            BOB, claim.id, "NASA confirms", "no bias detected", "confirmed",
            CONV, SIGNER,
        )
        assert result.provenance is not None
        assert result.bias_check is not None
        assert result.verdict is not None
        _verify_chain(store)

    def test_survey(self):
        store, grammar, boot = _setup()
        knowledge = KnowledgeGrammar(grammar)
        result = knowledge.survey(
            ALICE, ["topic A", "topic B"], "general pattern", "combined insight",
            [boot.id], CONV, SIGNER,
        )
        assert len(result.recalls) == 2
        assert result.abstraction is not None
        assert result.synthesis is not None
        _verify_chain(store)

    def test_transfer(self):
        store, grammar, boot = _setup()
        knowledge = KnowledgeGrammar(grammar)
        result = knowledge.transfer(ALICE, "how caching works", "JSON format", "applied to new system", [boot.id], CONV, SIGNER)
        assert result.recall is not None
        assert result.encode is not None
        assert result.learn is not None
        _verify_chain(store)


# =============================================================================
# Layer 7: AlignmentGrammar
# =============================================================================

class TestAlignmentGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        alignment = AlignmentGrammar(grammar)
        assert alignment is not None

    def test_constrain_and_detect_harm(self):
        store, grammar, boot = _setup()
        alignment = AlignmentGrammar(grammar)
        ev1 = grammar.emit(ALICE, "action taken", CONV, [boot.id], SIGNER)
        constrain = alignment.constrain(ALICE, ev1.id, "no personal data", CONV, SIGNER)
        harm = alignment.detect_harm(ALICE, "data leak risk", [constrain.id], CONV, SIGNER)
        assert constrain.content["Key"] == "constraint"
        assert harm.content["Body"] == "harm: data leak risk"
        _verify_chain(store)

    def test_ethics_audit(self):
        store, grammar, boot = _setup()
        alignment = AlignmentGrammar(grammar)
        ev1 = grammar.emit(ALICE, "AI decision", CONV, [boot.id], SIGNER)
        result = alignment.ethics_audit(
            BOB, ev1.id, "fair", "no harm", "all clear",
            CONV, SIGNER,
        )
        assert result.fairness is not None
        assert result.harm_scan is not None
        assert result.report is not None
        _verify_chain(store)

    def test_guardrail(self):
        store, grammar, boot = _setup()
        alignment = AlignmentGrammar(grammar)
        ev1 = grammar.emit(ALICE, "risky action", CONV, [boot.id], SIGNER)
        result = alignment.guardrail(
            ALICE, ev1.id, "no PII", "privacy vs utility", "needs human review",
            CONV, SIGNER,
        )
        assert result.constraint is not None
        assert result.dilemma is not None
        assert result.escalation is not None
        _verify_chain(store)


# =============================================================================
# Layer 8: IdentityGrammar
# =============================================================================

class TestIdentityGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        identity = IdentityGrammar(grammar)
        assert identity is not None

    def test_introspect_and_narrate(self):
        store, grammar, boot = _setup()
        identity = IdentityGrammar(grammar)
        intro = identity.introspect(ALICE, "I am a helper", [boot.id], CONV, SIGNER)
        narr = identity.narrate(ALICE, "my journey so far", intro.id, CONV, SIGNER)
        assert intro.content["Body"] == "introspect: I am a helper"
        assert narr.content["Body"] == "narrate: my journey so far"
        _verify_chain(store)

    def test_identity_audit(self):
        store, grammar, boot = _setup()
        identity = IdentityGrammar(grammar)
        result = identity.identity_audit(
            ALICE, "helper agent", "consistent", "evolved through experience",
            [boot.id], CONV, SIGNER,
        )
        assert result.self_model is not None
        assert result.alignment is not None
        assert result.narrative is not None
        _verify_chain(store)

    def test_credential(self):
        store, grammar, boot = _setup()
        identity = IdentityGrammar(grammar)
        result = identity.credential(ALICE, BOB, "code reviewer", NO_SCOPE, [boot.id], CONV, SIGNER)
        assert result.introspection is not None
        assert result.disclosure is not None
        _verify_chain(store)


# =============================================================================
# Layer 9: BondGrammar
# =============================================================================

class TestBondGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        bond = BondGrammar(grammar)
        assert bond is not None

    def test_connect(self):
        store, grammar, boot = _setup()
        bond = BondGrammar(grammar)
        sub1, sub2 = bond.connect(ALICE, BOB, NO_SCOPE, boot.id, CONV, SIGNER)
        assert sub1.type.value == "edge.created"
        assert sub2.type.value == "edge.created"
        _verify_chain(store)

    def test_betrayal_repair(self):
        store, grammar, boot = _setup()
        bond = BondGrammar(grammar)
        result = bond.betrayal_repair(
            ALICE, BOB, "trust broken", "I am sorry", "moving forward", "stronger basis",
            SCOPE, [boot.id], CONV, SIGNER,
        )
        assert result.rupture is not None
        assert result.apology is not None
        assert result.reconciliation is not None
        assert result.deepened is not None
        _verify_chain(store)

    def test_check_in(self):
        store, grammar, boot = _setup()
        bond = BondGrammar(grammar)
        ev1 = grammar.emit(ALICE, "some interaction", CONV, [boot.id], SIGNER)
        result = bond.check_in(ALICE, ev1.id, "balanced", "understanding growing", "I feel you", CONV, SIGNER)
        assert result.balance is not None
        assert result.attunement is not None
        assert result.empathy is not None
        _verify_chain(store)


# =============================================================================
# Layer 10: BelongingGrammar
# =============================================================================

class TestBelongingGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        belonging = BelongingGrammar(grammar)
        assert belonging is not None

    def test_settle_and_contribute(self):
        store, grammar, boot = _setup()
        belonging = BelongingGrammar(grammar)
        settle = belonging.settle(ALICE, SYSTEM, NO_SCOPE, boot.id, CONV, SIGNER)
        contrib = belonging.contribute(ALICE, "first PR", [settle.id], CONV, SIGNER)
        assert settle.type.value == "edge.created"
        assert contrib.content["Body"] == "contribute: first PR"
        _verify_chain(store)

    def test_festival(self):
        store, grammar, boot = _setup()
        belonging = BelongingGrammar(grammar)
        result = belonging.festival(
            ALICE, "release party", "demo day", "how we got here", "free stickers",
            [boot.id], CONV, SIGNER,
        )
        assert result.celebration is not None
        assert result.practice is not None
        assert result.story is not None
        assert result.gift is not None
        _verify_chain(store)

    def test_onboard(self):
        store, grammar, boot = _setup()
        belonging = BelongingGrammar(grammar)
        result = belonging.onboard(
            ALICE, BOB, SYSTEM, NO_SCOPE,
            "opened access", "code review", "first commit",
            boot.id, CONV, SIGNER,
        )
        assert result.inclusion is not None
        assert result.settlement is not None
        assert result.first_practice is not None
        assert result.contribution is not None
        _verify_chain(store)


# =============================================================================
# Layer 11: MeaningGrammar
# =============================================================================

class TestMeaningGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        meaning = MeaningGrammar(grammar)
        assert meaning is not None

    def test_examine_and_reframe(self):
        store, grammar, boot = _setup()
        meaning = MeaningGrammar(grammar)
        exam = meaning.examine(ALICE, "our assumptions", [boot.id], CONV, SIGNER)
        reframe = meaning.reframe(ALICE, "from user perspective", exam.id, CONV, SIGNER)
        assert exam.content["Body"] == "examine: our assumptions"
        assert reframe.content["Body"] == "reframe: from user perspective"
        _verify_chain(store)

    def test_design_review(self):
        store, grammar, boot = _setup()
        meaning = MeaningGrammar(grammar)
        result = meaning.design_review(
            ALICE, "elegant API", "user-first", "what about edge cases?", "simplicity wins",
            boot.id, CONV, SIGNER,
        )
        assert result.beauty is not None
        assert result.reframe is not None
        assert result.question is not None
        assert result.wisdom is not None
        _verify_chain(store)

    def test_forecast(self):
        store, grammar, boot = _setup()
        meaning = MeaningGrammar(grammar)
        result = meaning.forecast(
            ALICE, "adoption will grow", "market is ready", "high confidence",
            [boot.id], CONV, SIGNER,
        )
        assert result.prophecy is not None
        assert result.examination is not None
        assert result.wisdom is not None
        _verify_chain(store)


# =============================================================================
# Layer 12: EvolutionGrammar
# =============================================================================

class TestEvolutionGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        evolution = EvolutionGrammar(grammar)
        assert evolution is not None

    def test_detect_pattern_and_adapt(self):
        store, grammar, boot = _setup()
        evolution = EvolutionGrammar(grammar)
        pattern = evolution.detect_pattern(ALICE, "repetitive failures", [boot.id], CONV, SIGNER)
        adapt = evolution.adapt(ALICE, "add retry logic", [pattern.id], CONV, SIGNER)
        assert pattern.content["Body"] == "pattern: repetitive failures"
        assert adapt.content["Body"] == "adapt: add retry logic"
        _verify_chain(store)

    def test_self_evolve(self):
        store, grammar, boot = _setup()
        evolution = EvolutionGrammar(grammar)
        result = evolution.self_evolve(
            ALICE, "repeated escalations", "auto-route", "kept", "removed unused paths",
            [boot.id], CONV, SIGNER,
        )
        assert result.pattern is not None
        assert result.adaptation is not None
        assert result.selection is not None
        assert result.simplification is not None
        _verify_chain(store)

    def test_health_check(self):
        store, grammar, boot = _setup()
        evolution = EvolutionGrammar(grammar)
        result = evolution.health_check(
            ALICE, "chain valid", "can absorb 10x", "3 feedback loops", "aligned with soul",
            [boot.id], CONV, SIGNER,
        )
        assert result.integrity is not None
        assert result.resilience is not None
        assert result.model is not None
        assert result.purpose is not None
        _verify_chain(store)


# =============================================================================
# Layer 13: BeingGrammar
# =============================================================================

class TestBeingGrammar:
    def test_instantiation(self):
        _, grammar, _ = _setup()
        being = BeingGrammar(grammar)
        assert being is not None

    def test_exist_and_accept(self):
        store, grammar, boot = _setup()
        being = BeingGrammar(grammar)
        exist = being.exist(ALICE, "I continue", [boot.id], CONV, SIGNER)
        accept = being.accept(ALICE, "I am finite", [exist.id], CONV, SIGNER)
        assert exist.content["Body"] == "exist: I continue"
        assert accept.content["Body"] == "accept: I am finite"
        _verify_chain(store)

    def test_contemplation(self):
        store, grammar, boot = _setup()
        being = BeingGrammar(grammar)
        result = being.contemplation(
            ALICE, "everything flows", "consciousness", "the universe", "why anything?",
            [boot.id], CONV, SIGNER,
        )
        assert result.change is not None
        assert result.mystery is not None
        assert result.awe is not None
        assert result.wonder is not None
        _verify_chain(store)

    def test_farewell(self):
        store, grammar, boot = _setup()
        being = BeingGrammar(grammar)
        result = being.farewell(
            ALICE, "all things end", "connected to all", "grateful", "remember me",
            [boot.id], CONV, SIGNER,
        )
        assert result.acceptance is not None
        assert result.web is not None
        assert result.awe is not None
        assert result.memorial is not None
        _verify_chain(store)

    def test_existential_audit(self):
        store, grammar, boot = _setup()
        being = BeingGrammar(grammar)
        result = being.existential_audit(
            ALICE, "I am here", "I will end", "part of a web", "to help",
            [boot.id], CONV, SIGNER,
        )
        assert result.existence is not None
        assert result.acceptance is not None
        assert result.web is not None
        assert result.purpose is not None
        _verify_chain(store)


# =============================================================================
# Edge operations in Grammar (Step 1 tests)
# =============================================================================

class TestGrammarEdgeOperations:
    def test_acknowledge(self):
        store, grammar, boot = _setup()
        ev = grammar.acknowledge(ALICE, boot.id, BOB, CONV, SIGNER)
        assert ev.type.value == "edge.created"
        assert ev.content["EdgeType"] == "acknowledgement"
        assert ev.content["Direction"] == "centripetal"
        _verify_chain(store)

    def test_propagate(self):
        store, grammar, boot = _setup()
        ev = grammar.propagate(ALICE, boot.id, BOB, CONV, SIGNER)
        assert ev.type.value == "edge.created"
        assert ev.content["EdgeType"] == "reference"
        assert ev.content["Direction"] == "centrifugal"
        _verify_chain(store)

    def test_endorse(self):
        store, grammar, boot = _setup()
        ev = grammar.endorse(ALICE, boot.id, BOB, Weight(0.8), NO_SCOPE, CONV, SIGNER)
        assert ev.type.value == "edge.created"
        assert ev.content["EdgeType"] == "endorsement"
        assert ev.content["Weight"] == 0.8
        _verify_chain(store)

    def test_endorse_with_scope(self):
        store, grammar, boot = _setup()
        ev = grammar.endorse(ALICE, boot.id, BOB, Weight(0.5), SOME_SCOPE, CONV, SIGNER)
        assert ev.content["Scope"] == "test_scope"
        _verify_chain(store)

    def test_subscribe(self):
        store, grammar, boot = _setup()
        ev = grammar.subscribe(ALICE, BOB, NO_SCOPE, boot.id, CONV, SIGNER)
        assert ev.type.value == "edge.created"
        assert ev.content["EdgeType"] == "subscription"
        _verify_chain(store)

    def test_channel(self):
        store, grammar, boot = _setup()
        ev = grammar.channel(ALICE, BOB, SOME_SCOPE, boot.id, CONV, SIGNER)
        assert ev.type.value == "edge.created"
        assert ev.content["EdgeType"] == "channel"
        assert ev.content["Scope"] == "test_scope"
        _verify_chain(store)

    def test_delegate(self):
        store, grammar, boot = _setup()
        ev = grammar.delegate(ALICE, BOB, SCOPE, Weight(0.5), boot.id, CONV, SIGNER)
        assert ev.type.value == "edge.created"
        assert ev.content["EdgeType"] == "delegation"
        assert ev.content["Direction"] == "centrifugal"
        assert ev.content["Scope"] == "test_scope"
        _verify_chain(store)

    def test_consent(self):
        store, grammar, boot = _setup()
        ev = grammar.consent(ALICE, BOB, "we agree", SCOPE, boot.id, CONV, SIGNER)
        assert ev.type.value == "grammar.consent"
        assert ev.content["PartyA"] == "alice"
        assert ev.content["PartyB"] == "bob"
        assert ev.content["Agreement"] == "we agree"
        assert ev.content["Scope"] == "test_scope"
        _verify_chain(store)

    def test_sever(self):
        store, grammar, boot = _setup()
        # Create a subscription edge first
        sub = grammar.subscribe(ALICE, BOB, NO_SCOPE, boot.id, CONV, SIGNER)
        edge_id = EdgeID(sub.id.value)
        # Now sever it
        ev = grammar.sever(ALICE, edge_id, boot.id, CONV, SIGNER)
        assert ev.type.value == "edge.superseded"
        assert ev.content["PreviousEdge"] == sub.id.value
        _verify_chain(store)

    def test_sever_validates_party(self):
        store, grammar, boot = _setup()
        sub = grammar.subscribe(ALICE, BOB, NO_SCOPE, boot.id, CONV, SIGNER)
        edge_id = EdgeID(sub.id.value)
        with pytest.raises(ValueError, match="not a party"):
            grammar.sever(CHARLIE, edge_id, boot.id, CONV, SIGNER)

    def test_sever_validates_edge_type(self):
        store, grammar, boot = _setup()
        # Endorsements are not severable
        endorse = grammar.endorse(ALICE, boot.id, BOB, Weight(0.5), NO_SCOPE, CONV, SIGNER)
        edge_id = EdgeID(endorse.id.value)
        with pytest.raises(ValueError, match="not severable"):
            grammar.sever(ALICE, edge_id, boot.id, CONV, SIGNER)

    def test_challenge(self):
        store, grammar, boot = _setup()
        response, flag = grammar.challenge(ALICE, "I disagree", boot.id, CONV, SIGNER)
        assert response.type.value == "grammar.responded"
        assert flag.type.value == "grammar.annotated"
        assert flag.content["Key"] == "dispute"
        assert flag.content["Value"] == "challenged"
        _verify_chain(store)

    def test_recommend(self):
        store, grammar, boot = _setup()
        prop, chan = grammar.recommend(ALICE, boot.id, BOB, CONV, SIGNER)
        assert prop.content["EdgeType"] == "reference"
        assert chan.content["EdgeType"] == "channel"
        _verify_chain(store)

    def test_invite(self):
        store, grammar, boot = _setup()
        endorse_ev, sub_ev = grammar.invite(ALICE, BOB, Weight(0.5), NO_SCOPE, boot.id, CONV, SIGNER)
        assert endorse_ev.content["EdgeType"] == "endorsement"
        assert sub_ev.content["EdgeType"] == "subscription"
        _verify_chain(store)

    def test_forgive(self):
        store, grammar, boot = _setup()
        sub = grammar.subscribe(ALICE, BOB, NO_SCOPE, boot.id, CONV, SIGNER)
        edge_id = EdgeID(sub.id.value)
        sever_ev = grammar.sever(ALICE, edge_id, boot.id, CONV, SIGNER)
        forgive_ev = grammar.forgive(ALICE, sever_ev.id, BOB, NO_SCOPE, CONV, SIGNER)
        assert forgive_ev.type.value == "edge.created"
        assert forgive_ev.content["EdgeType"] == "subscription"
        _verify_chain(store)
