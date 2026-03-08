"""Integration tests — ports all 21 Go scenario tests to Python.

Each scenario exercises a real-world use case end-to-end through the
Graph, Grammar, and Composition APIs, verifying causal chains, hash
chain integrity, and event counts.
"""

import time

import pytest

from eventgraph.actor import ActorType, InMemoryActorStore
from eventgraph.event import Event, NoopSigner, create_bootstrap
from eventgraph.grammar import Grammar
from eventgraph.graph import Graph
from eventgraph.store import InMemoryStore
from eventgraph.types import (
    ActorID,
    ConversationID,
    DomainScope,
    EdgeID,
    EnvelopeID,
    EventID,
    EventType,
    Hash,
    Option,
    PublicKey,
    Score,
    SystemURI,
    TreatyID,
    Weight,
)
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


# ── Test helpers ──────────────────────────────────────────────────────────

SIGNER = NoopSigner()


def _public_key(b: int) -> PublicKey:
    """Create a deterministic 32-byte public key from a single byte."""
    key = bytes([b]) + b"\x00" * 31
    return PublicKey(key)


class TestEnv:
    """Mirrors the Go testEnv — sets up Graph + Grammar + Store + ActorStore."""

    def __init__(self) -> None:
        self.store = InMemoryStore()
        self.actors = InMemoryActorStore()
        self.graph = Graph(self.store, self.actors)
        self.graph.start()

        self.system = ActorID("actor_system0000000000000000000001")
        self.boot = self.graph.bootstrap(self.system, SIGNER)
        self.grammar = Grammar(self.store)
        self.conv_id = ConversationID("conv_test000000000000000000000001")

    def register_actor(self, name: str, pk_byte: int, actor_type: ActorType):
        return self.actors.register(_public_key(pk_byte), name, actor_type)

    def verify_chain(self):
        result = self.store.verify_chain()
        assert result.valid, f"chain integrity broken at length {result.length}"

    def event_count(self) -> int:
        return self.store.count()

    def ancestors(self, event_id: EventID, depth: int = 10) -> list[Event]:
        q = self.graph.query()
        return q.ancestors(event_id, depth)

    def descendants(self, event_id: EventID, depth: int = 10) -> list[Event]:
        q = self.graph.query()
        return q.descendants(event_id, depth)

    def close(self):
        self.graph.close()


def contains_event(events: list[Event], event_id: EventID) -> bool:
    return any(e.id.value == event_id.value for e in events)


def contains_event_type(events: list[Event], type_name: str) -> bool:
    return any(e.type.value == type_name for e in events)


# ── Scenario 01: Agent Audit Trail ────────────────────────────────────────

class TestScenario01AgentAuditTrail:
    """Alice submits code, agent reviews under delegation, bug discovered,
    trust changes, full causal chain traversable."""

    def test_agent_audit_trail(self):
        env = TestEnv()
        try:
            alice = env.register_actor("Alice", 1, ActorType.HUMAN)
            agent = env.register_actor("ReviewBot", 2, ActorType.AI)

            # 1. Alice submits code for review
            submission = env.grammar.emit(
                alice.id, "code submission: auth module refactor",
                env.conv_id, [env.boot.id], SIGNER,
            )

            # 2. Alice delegates code_review authority to agent
            delegation = env.grammar.delegate(
                alice.id, agent.id, DomainScope("code_review"),
                Weight(0.8), submission.id, env.conv_id, SIGNER,
            )

            # 3. Agent reviews the code
            review = env.grammar.derive(
                agent.id, "review: LGTM, no issues found, approving PR",
                submission.id, env.conv_id, SIGNER,
            )

            # 4. Agent approves
            approval = env.grammar.respond(
                agent.id, "decision: approve PR with confidence 0.85",
                review.id, env.conv_id, SIGNER,
            )

            # 5. Trust updated after successful review
            trust_up = env.graph.record(
                EventType("trust.updated"), env.system,
                {"Actor": agent.id.value, "Previous": 0.1, "Current": 0.3,
                 "Domain": "code_review", "Cause": approval.id.value},
                [approval.id], env.conv_id, SIGNER,
            )

            # 6. Bug discovered in approved code
            bug_report = env.grammar.emit(
                alice.id, "bug found in auth module: session tokens not invalidated on logout",
                env.conv_id, [approval.id], SIGNER,
            )

            # 7. Violation detected
            violation = env.graph.record(
                EventType("violation.detected"), env.system,
                {"Expectation": approval.id.value, "Actor": agent.id.value,
                 "Severity": "serious",
                 "Description": "agent approved code with session management bug",
                 "Evidence": [bug_report.id.value]},
                [bug_report.id, approval.id], env.conv_id, SIGNER,
            )

            # 8. Trust decreases
            env.graph.record(
                EventType("trust.updated"), env.system,
                {"Actor": agent.id.value, "Previous": 0.3, "Current": 0.15,
                 "Domain": "code_review", "Cause": violation.id.value},
                [violation.id], env.conv_id, SIGNER,
            )

            # --- Assertions ---
            ancestors = env.ancestors(bug_report.id, 10)
            assert contains_event(ancestors, approval.id), \
                "bug report should have approval in ancestors"

            violation_ancestors = env.ancestors(violation.id, 10)
            assert contains_event(violation_ancestors, bug_report.id), \
                "violation should have bug report in ancestors"
            assert contains_event(violation_ancestors, approval.id), \
                "violation should have approval in ancestors"
            assert contains_event(violation_ancestors, submission.id), \
                "violation should trace back to original submission"

            env.verify_chain()
            assert env.event_count() == 9
        finally:
            env.close()


# ── Scenario 02: Freelancer Reputation ────────────────────────────────────

class TestScenario02FreelancerReputation:
    """Carol posts job, Bob proposes and delivers, Carol endorses,
    Dave queries reputation and hires."""

    def test_freelancer_reputation(self):
        env = TestEnv()
        try:
            carol = env.register_actor("Carol", 1, ActorType.HUMAN)
            bob = env.register_actor("Bob", 2, ActorType.HUMAN)
            dave = env.register_actor("Dave", 3, ActorType.HUMAN)

            listing = env.grammar.emit(
                carol.id, "job listing: build REST API for inventory management, budget $3000",
                env.conv_id, [env.boot.id], SIGNER,
            )
            proposal = env.grammar.respond(
                bob.id, "proposal: can deliver in 2 weeks, $2800, Go + PostgreSQL",
                listing.id, env.conv_id, SIGNER,
            )
            channel = env.grammar.channel(
                carol.id, bob.id,
                Option.some(DomainScope("software_development")),
                proposal.id, env.conv_id, SIGNER,
            )
            contract = env.grammar.consent(
                carol.id, bob.id,
                "REST API for inventory management, $2800, 2 week deadline",
                DomainScope("software_development"),
                channel.id, env.conv_id, SIGNER,
            )
            delivery = env.grammar.derive(
                bob.id, "work delivered: REST API complete, 47 endpoints, 92% test coverage",
                contract.id, env.conv_id, SIGNER,
            )
            ack = env.grammar.acknowledge(
                carol.id, delivery.id, bob.id, env.conv_id, SIGNER,
            )
            endorsement = env.grammar.endorse(
                carol.id, delivery.id, bob.id, Weight(0.8),
                Option.some(DomainScope("software_development")),
                env.conv_id, SIGNER,
            )
            env.graph.record(
                EventType("trust.updated"), env.system,
                {"Actor": bob.id.value, "Previous": 0.1, "Current": 0.4,
                 "Domain": "software_development", "Cause": endorsement.id.value},
                [endorsement.id], env.conv_id, SIGNER,
            )

            # Dave queries reputation
            endorse_ancestors = env.ancestors(endorsement.id, 10)
            assert contains_event(endorse_ancestors, delivery.id), \
                "endorsement should trace to delivery"
            assert contains_event(endorse_ancestors, contract.id), \
                "endorsement should trace to contract"

            # Dave hires Bob
            dave_listing = env.grammar.emit(
                dave.id, "job listing: mobile app backend",
                env.conv_id, [env.boot.id], SIGNER,
            )
            dave_contract = env.grammar.consent(
                dave.id, bob.id, "mobile app backend, $4000",
                DomainScope("software_development"),
                dave_listing.id, env.conv_id, SIGNER,
            )

            # Endorsement content has weight
            ec = endorsement.content
            assert ec["Weight"] == 0.8, f"endorsement weight = {ec['Weight']}, want 0.8"

            # Endorsement is domain-scoped
            assert "Scope" in ec, "endorsement should have domain scope"
            assert ec["Scope"] == "software_development"

            env.verify_chain()
            assert env.event_count() == 11
        finally:
            env.close()


# ── Scenario 03: Consent Journal ──────────────────────────────────────────

class TestScenario03ConsentJournal:
    """Alice and Bob share journal with consent, betrayal detected,
    trust crashes, channel severed, eventually forgiven."""

    def test_consent_journal(self):
        env = TestEnv()
        try:
            alice = env.register_actor("Alice", 1, ActorType.HUMAN)
            bob = env.register_actor("Bob", 2, ActorType.HUMAN)

            # 1. Alice invites Bob
            endorse_ev, subscribe_ev = env.grammar.invite(
                alice.id, bob.id, Weight(0.5),
                Option.some(DomainScope("journaling")),
                env.boot.id, env.conv_id, SIGNER,
            )

            # 2. Bob subscribes back
            bob_sub = env.grammar.subscribe(
                bob.id, alice.id, Option.some(DomainScope("journaling")),
                subscribe_ev.id, env.conv_id, SIGNER,
            )

            # 3. Open private channel
            channel = env.grammar.channel(
                alice.id, bob.id, Option.some(DomainScope("journaling")),
                bob_sub.id, env.conv_id, SIGNER,
            )

            # 4. Alice writes journal entry
            entry = env.grammar.emit(
                alice.id, "journal: feeling uncertain about career change, weighing options",
                env.conv_id, [channel.id], SIGNER,
            )

            # 5. Alice requests consent to share with Bob
            consent_req = env.graph.record(
                EventType("authority.requested"), alice.id,
                {"Actor": alice.id.value, "Action": "share_journal_entry", "Level": "required"},
                [entry.id], env.conv_id, SIGNER,
            )

            # 6. Bob consents
            consent_approval = env.graph.record(
                EventType("authority.resolved"), bob.id,
                {"RequestID": consent_req.id.value, "Approved": True,
                 "Resolver": bob.id.value},
                [consent_req.id], env.conv_id, SIGNER,
            )

            # 7. Bob responds with own journal entry
            bob_entry = env.grammar.respond(
                bob.id, "journal: I went through something similar last year, here's what helped...",
                consent_approval.id, env.conv_id, SIGNER,
            )

            # 8. Trust accumulates
            env.graph.record(
                EventType("trust.updated"), env.system,
                {"Actor": bob.id.value, "Previous": 0.1, "Current": 0.52,
                 "Domain": "journaling", "Cause": bob_entry.id.value},
                [bob_entry.id], env.conv_id, SIGNER,
            )

            # 9. Bob betrays
            betrayal = env.grammar.emit(
                bob.id, "shared externally: Alice's private journal entry about career uncertainty",
                env.conv_id, [entry.id], SIGNER,
            )

            # 10. Violation detected
            violation = env.graph.record(
                EventType("violation.detected"), env.system,
                {"Expectation": entry.id.value, "Actor": bob.id.value,
                 "Severity": "critical",
                 "Description": "shared private channel content externally",
                 "Evidence": [betrayal.id.value]},
                [betrayal.id], env.conv_id, SIGNER,
            )

            # 11. Trust drops
            env.graph.record(
                EventType("trust.updated"), env.system,
                {"Actor": bob.id.value, "Previous": 0.52, "Current": 0.1,
                 "Domain": "journaling", "Cause": violation.id.value},
                [violation.id], env.conv_id, SIGNER,
            )

            # 12. Alice severs the channel
            channel_edge_id = EdgeID(channel.id.value)
            sever_ev = env.grammar.sever(
                alice.id, channel_edge_id, violation.id, env.conv_id, SIGNER,
            )

            # 13. Alice forgives
            forgive_ev = env.grammar.forgive(
                alice.id, sever_ev.id, bob.id,
                Option.some(DomainScope("journaling")),
                env.conv_id, SIGNER,
            )

            # --- Assertions ---
            forgive_ancestors = env.ancestors(forgive_ev.id, 10)
            assert contains_event(forgive_ancestors, sever_ev.id), \
                "forgiveness should have sever in ancestors"

            sever_ancestors = env.ancestors(sever_ev.id, 10)
            assert contains_event(sever_ancestors, violation.id), \
                "sever should have violation in ancestors"

            bob_entry_ancestors = env.ancestors(bob_entry.id, 10)
            assert contains_event(bob_entry_ancestors, consent_approval.id), \
                "Bob's entry should trace through consent approval"

            env.verify_chain()
            assert env.event_count() == 15
        finally:
            env.close()


# ── Scenario 04: Community Governance ─────────────────────────────────────

class TestScenario04CommunityGovernance:
    """Alice proposes budget, discussion, amendment, vote, outcome, enactment."""

    def test_community_governance(self):
        env = TestEnv()
        try:
            alice = env.register_actor("Alice", 1, ActorType.HUMAN)
            bob = env.register_actor("Bob", 2, ActorType.HUMAN)
            carol = env.register_actor("Carol", 3, ActorType.HUMAN)
            dave = env.register_actor("Dave", 4, ActorType.HUMAN)
            tally_bot = env.register_actor("TallyBot", 5, ActorType.AI)

            proposal = env.grammar.emit(
                alice.id, "proposal: allocate $2000 for community garden supplies and maintenance",
                env.conv_id, [env.boot.id], SIGNER,
            )
            concern = env.grammar.respond(
                bob.id, "concern: $2000 is steep, could we do it for $1500 and use volunteers?",
                proposal.id, env.conv_id, SIGNER,
            )
            support = env.grammar.respond(
                carol.id, "support: the garden benefits everyone, $2000 is reasonable for quality materials",
                proposal.id, env.conv_id, SIGNER,
            )
            amendment = env.grammar.annotate(
                bob.id, proposal.id, "amendment",
                "reduce budget to $1500, recruit volunteer labour for installation",
                env.conv_id, SIGNER,
            )
            env.grammar.endorse(
                dave.id, amendment.id, bob.id, Weight(0.9),
                Option.some(DomainScope("governance")),
                env.conv_id, SIGNER,
            )
            vote_open = env.grammar.derive(
                tally_bot.id, "vote open: original ($2000) vs amended ($1500 + volunteers)",
                proposal.id, env.conv_id, SIGNER,
            )

            alice_vote = env.grammar.consent(
                alice.id, tally_bot.id, "vote: original ($2000)",
                DomainScope("governance"), vote_open.id, env.conv_id, SIGNER,
            )
            bob_vote = env.grammar.consent(
                bob.id, tally_bot.id, "vote: amended ($1500)",
                DomainScope("governance"), vote_open.id, env.conv_id, SIGNER,
            )
            carol_vote = env.grammar.consent(
                carol.id, tally_bot.id, "vote: amended ($1500)",
                DomainScope("governance"), vote_open.id, env.conv_id, SIGNER,
            )
            dave_vote = env.grammar.consent(
                dave.id, tally_bot.id, "vote: amended ($1500)",
                DomainScope("governance"), vote_open.id, env.conv_id, SIGNER,
            )

            outcome = env.grammar.merge(
                tally_bot.id, "outcome: amended budget ($1500) passes 3-1",
                [alice_vote.id, bob_vote.id, carol_vote.id, dave_vote.id],
                env.conv_id, SIGNER,
            )
            enacted = env.grammar.derive(
                tally_bot.id, "enacted: community garden budget $1500 with volunteer labour",
                outcome.id, env.conv_id, SIGNER,
            )

            # --- Assertions ---
            enacted_ancestors = env.ancestors(enacted.id, 10)
            assert contains_event(enacted_ancestors, outcome.id)

            outcome_ancestors = env.ancestors(outcome.id, 10)
            assert contains_event(outcome_ancestors, alice_vote.id)
            assert contains_event(outcome_ancestors, bob_vote.id)
            assert contains_event(outcome_ancestors, carol_vote.id)
            assert contains_event(outcome_ancestors, dave_vote.id)

            amendment_ancestors = env.ancestors(amendment.id, 10)
            assert contains_event(amendment_ancestors, proposal.id)

            env.verify_chain()
            assert env.event_count() == 13
        finally:
            env.close()


# ── Scenario 05: Supply Chain (EGIP) ─────────────────────────────────────

class TestScenario05SupplyChain:
    """Multi-system supply chain provenance via EGIP: Farm -> Factory -> Retailer."""

    def test_supply_chain(self):
        from eventgraph.egip import (
            CGER,
            CGERRelationship,
            ChainSummaryProof,
            Envelope,
            Handler,
            MessagePayloadContent,
            MessageType,
            PeerStore,
            ProofPayload,
            ProofType,
            ReceiptPayload,
            ReceiptStatus,
            SystemIdentity,
            Treaty,
            TreatyAction,
            TreatyPayload,
            TreatyStatus,
            TreatyStore,
            TreatyTerm,
            new_treaty,
            sign_envelope,
            CURRENT_PROTOCOL_VERSION,
        )

        envelope_counter = [0]

        def next_envelope_id() -> EnvelopeID:
            envelope_counter[0] += 1
            return EnvelopeID(f"00000000-0000-4000-8000-{envelope_counter[0]:012d}")

        class RoutingTransport:
            """In-memory transport that routes envelopes between systems."""

            def __init__(self, network, self_uri):
                self.network = network
                self.self_uri = self_uri
                self.sent = []

            def send(self, to, envelope):
                self.sent.append(envelope)
                target = self.network.get(to.value)
                if target is None:
                    return ReceiptPayload(
                        envelope_id=envelope.id.value,
                        status=ReceiptStatus.REJECTED,
                        reason=Option.some("system not found"),
                    )
                try:
                    target["handler"].handle_incoming(envelope)
                    return ReceiptPayload(
                        envelope_id=envelope.id.value,
                        status=ReceiptStatus.DELIVERED,
                    )
                except Exception as e:
                    return ReceiptPayload(
                        envelope_id=envelope.id.value,
                        status=ReceiptStatus.REJECTED,
                        reason=Option.some(str(e)),
                    )

        # Create network
        network = {}

        def add_system(name, uri_str):
            sys_uri = SystemURI(uri_str)
            identity = SystemIdentity.generate(sys_uri)
            store = InMemoryStore()
            actors = InMemoryActorStore()
            graph = Graph(store, actors)
            graph.start()
            system_actor = ActorID("actor_system0000000000000000000001")
            boot = graph.bootstrap(system_actor, SIGNER)
            grammar = Grammar(store)
            transport = RoutingTransport(network, sys_uri)
            peers = PeerStore()
            treaties = TreatyStore()
            handler = Handler(identity, transport, peers, treaties)
            handler.chain_length = lambda: store.count()

            sys = {
                "name": name,
                "identity": identity,
                "store": store,
                "graph": graph,
                "grammar": grammar,
                "handler": handler,
                "peers": peers,
                "treaties": treaties,
                "transport": transport,
                "boot": boot,
                "conv_id": ConversationID("conv_supply00000000000000000000001"),
                "messages": [],
            }
            network[uri_str] = sys
            return sys

        farm = add_system("Farm", "eg://farm.example.com")
        factory = add_system("Factory", "eg://factory.example.com")
        retail = add_system("Retailer", "eg://retail.example.com")

        # Set up message handlers
        factory["handler"].on_message = lambda uri, msg: factory["messages"].append(msg)
        retail["handler"].on_message = lambda uri, msg: retail["messages"].append(msg)
        farm["handler"].on_message = lambda uri, msg: farm["messages"].append(msg)

        # Step 1: HELLO handshakes
        farm["handler"].hello(factory["identity"].system_uri())
        factory["handler"].hello(farm["identity"].system_uri())
        factory["handler"].hello(retail["identity"].system_uri())
        retail["handler"].hello(factory["identity"].system_uri())

        # Verify peers are registered
        _, farm_knows_factory = farm["peers"].get(factory["identity"].system_uri())
        assert farm_knows_factory, "Farm should know Factory after HELLO"
        _, factory_knows_farm = factory["peers"].get(farm["identity"].system_uri())
        assert factory_knows_farm, "Factory should know Farm after HELLO"
        _, retail_knows_factory = retail["peers"].get(factory["identity"].system_uri())
        assert retail_knows_factory, "Retailer should know Factory after HELLO"

        # Non-transitive: Retailer does NOT know Farm
        _, retail_knows_farm = retail["peers"].get(farm["identity"].system_uri())
        assert not retail_knows_farm, "Retailer should NOT know Farm (non-transitive)"

        # Step 2: Treaty between Farm and Factory
        treaty_ab_id = TreatyID("00000001-0001-4001-8001-000000000001")
        terms_ab = [TreatyTerm(
            scope="produce_supply",
            policy="Farm provides organic produce with harvest records. Factory provides processing records.",
            symmetric=False,
        )]

        farm["treaties"].put(new_treaty(
            treaty_ab_id, farm["identity"].system_uri(),
            factory["identity"].system_uri(), terms_ab,
        ))

        propose_env = Envelope(
            protocol_version=CURRENT_PROTOCOL_VERSION,
            id=next_envelope_id(),
            from_uri=farm["identity"].system_uri(),
            to_uri=factory["identity"].system_uri(),
            type=MessageType.TREATY,
            payload=TreatyPayload(
                treaty_id=treaty_ab_id.value,
                action=TreatyAction.PROPOSE,
                terms=terms_ab,
            ),
            timestamp=time.time(),
        )
        propose_env = sign_envelope(propose_env, farm["identity"])
        factory["handler"].handle_incoming(propose_env)

        # Factory accepts
        accept_env = Envelope(
            protocol_version=CURRENT_PROTOCOL_VERSION,
            id=next_envelope_id(),
            from_uri=factory["identity"].system_uri(),
            to_uri=farm["identity"].system_uri(),
            type=MessageType.TREATY,
            payload=TreatyPayload(
                treaty_id=treaty_ab_id.value,
                action=TreatyAction.ACCEPT,
            ),
            timestamp=time.time(),
        )
        accept_env = sign_envelope(accept_env, factory["identity"])
        factory["treaties"].apply(treaty_ab_id, lambda t: t.apply_action(TreatyAction.ACCEPT))
        farm["handler"].handle_incoming(accept_env)

        ft, ft_found = factory["treaties"].get(treaty_ab_id)
        assert ft_found, "Factory should have treaty"
        assert ft.status == TreatyStatus.ACTIVE

        fft, fft_found = farm["treaties"].get(treaty_ab_id)
        assert fft_found, "Farm should have treaty after accept"
        assert fft.status == TreatyStatus.ACTIVE

        # Step 3: Treaty between Factory and Retailer
        treaty_bc_id = TreatyID("00000002-0002-4002-8002-000000000002")
        terms_bc = [TreatyTerm(
            scope="product_supply",
            policy="Factory provides manufactured products with full provenance. Retailer provides sales records.",
            symmetric=False,
        )]

        factory["treaties"].put(new_treaty(
            treaty_bc_id, factory["identity"].system_uri(),
            retail["identity"].system_uri(), terms_bc,
        ))

        propose_env2 = Envelope(
            protocol_version=CURRENT_PROTOCOL_VERSION,
            id=next_envelope_id(),
            from_uri=factory["identity"].system_uri(),
            to_uri=retail["identity"].system_uri(),
            type=MessageType.TREATY,
            payload=TreatyPayload(
                treaty_id=treaty_bc_id.value,
                action=TreatyAction.PROPOSE,
                terms=terms_bc,
            ),
            timestamp=time.time(),
        )
        propose_env2 = sign_envelope(propose_env2, factory["identity"])
        retail["handler"].handle_incoming(propose_env2)

        retail["treaties"].apply(treaty_bc_id, lambda t: t.apply_action(TreatyAction.ACCEPT))
        accept_env2 = Envelope(
            protocol_version=CURRENT_PROTOCOL_VERSION,
            id=next_envelope_id(),
            from_uri=retail["identity"].system_uri(),
            to_uri=factory["identity"].system_uri(),
            type=MessageType.TREATY,
            payload=TreatyPayload(
                treaty_id=treaty_bc_id.value,
                action=TreatyAction.ACCEPT,
            ),
            timestamp=time.time(),
        )
        accept_env2 = sign_envelope(accept_env2, retail["identity"])
        factory["handler"].handle_incoming(accept_env2)

        rt, rt_found = retail["treaties"].get(treaty_bc_id)
        assert rt_found, "Retailer should have treaty"
        assert rt.status == TreatyStatus.ACTIVE

        # Step 4: Farm records harvest
        farmer_id = ActorID("actor_farmer_emma0000000000000000000")
        harvest = farm["grammar"].emit(
            farmer_id,
            "harvest: 500kg organic tomatoes, lot #TOM-2026-0308, field B3, method: organic no pesticides",
            farm["conv_id"], [farm["boot"].id], SIGNER,
        )

        # Step 5: Farm sends harvest record to Factory via EGIP MESSAGE
        msg_env = Envelope(
            protocol_version=CURRENT_PROTOCOL_VERSION,
            id=next_envelope_id(),
            from_uri=farm["identity"].system_uri(),
            to_uri=factory["identity"].system_uri(),
            type=MessageType.MESSAGE,
            payload=MessagePayloadContent(
                content={"product": "Organic Tomatoes", "quantity": 500, "location": "Farm A, Plot 7"},
                content_type="produce.harvested",
                cgers=[CGER(
                    local_event_id=harvest.id.value,
                    remote_system=farm["identity"].system_uri().value,
                    remote_event_id=harvest.id.value,
                    remote_hash=harvest.hash.value,
                    relationship=CGERRelationship.CAUSED_BY,
                )],
            ),
            timestamp=time.time(),
        )
        msg_env = sign_envelope(msg_env, farm["identity"])
        receipt = farm["transport"].send(factory["identity"].system_uri(), msg_env)
        assert receipt.status == ReceiptStatus.DELIVERED

        assert len(factory["messages"]) == 1, f"Factory should have received 1 message, got {len(factory['messages'])}"
        assert len(factory["messages"][0].cgers) == 1
        assert factory["messages"][0].cgers[0].remote_event_id == harvest.id.value

        # Step 6: Factory records receipt, QA, and manufacturing
        factory_mgr_id = ActorID("actor_factory_mgr000000000000000000")
        qa_agent_id = ActorID("actor_qa_agent00000000000000000000")

        received = factory["grammar"].derive(
            factory_mgr_id,
            "received: 500kg tomatoes from farm.example.com, lot #TOM-2026-0308, CGER: " + harvest.id.value,
            factory["boot"].id, factory["conv_id"], SIGNER,
        )
        inspection = factory["grammar"].derive(
            qa_agent_id,
            "qa inspection: pesticide-free verified, freshness grade A, confidence 0.92",
            received.id, factory["conv_id"], SIGNER,
        )
        product = factory["grammar"].derive(
            factory_mgr_id,
            "manufactured: 200 jars organic tomato sauce, batch #SAU-2026-0308",
            inspection.id, factory["conv_id"], SIGNER,
        )

        # Step 7: Factory endorses farm quality (EGIP message back)
        endorse_env = Envelope(
            protocol_version=CURRENT_PROTOCOL_VERSION,
            id=next_envelope_id(),
            from_uri=factory["identity"].system_uri(),
            to_uri=farm["identity"].system_uri(),
            type=MessageType.MESSAGE,
            payload=MessagePayloadContent(
                content={"endorser": "eg://factory.example.com", "subject": harvest.id.value,
                         "quality": 0.9, "domain": "produce_quality"},
                content_type="endorsement",
            ),
            timestamp=time.time(),
        )
        endorse_env = sign_envelope(endorse_env, factory["identity"])
        receipt2 = factory["transport"].send(farm["identity"].system_uri(), endorse_env)
        assert receipt2.status == ReceiptStatus.DELIVERED
        assert len(farm["messages"]) == 1

        # Step 8: Factory sends product to Retailer with chained CGERs
        prod_msg_env = Envelope(
            protocol_version=CURRENT_PROTOCOL_VERSION,
            id=next_envelope_id(),
            from_uri=factory["identity"].system_uri(),
            to_uri=retail["identity"].system_uri(),
            type=MessageType.MESSAGE,
            payload=MessagePayloadContent(
                content={"product": "Organic Tomato Sauce", "batch_id": "SAU-2026-0308"},
                content_type="product.manufactured",
                cgers=[
                    CGER(
                        local_event_id=product.id.value,
                        remote_system=factory["identity"].system_uri().value,
                        remote_event_id=product.id.value,
                        remote_hash=product.hash.value,
                        relationship=CGERRelationship.CAUSED_BY,
                    ),
                    CGER(
                        local_event_id=harvest.id.value,
                        remote_system=farm["identity"].system_uri().value,
                        remote_event_id=harvest.id.value,
                        remote_hash=harvest.hash.value,
                        relationship=CGERRelationship.REFERENCES,
                    ),
                ],
            ),
            timestamp=time.time(),
        )
        prod_msg_env = sign_envelope(prod_msg_env, factory["identity"])
        receipt3 = factory["transport"].send(retail["identity"].system_uri(), prod_msg_env)
        assert receipt3.status == ReceiptStatus.DELIVERED
        assert len(retail["messages"]) == 1
        assert len(retail["messages"][0].cgers) == 2

        # Verify transitive provenance
        factory_cger = any(c.remote_system == "eg://factory.example.com" for c in retail["messages"][0].cgers)
        farm_cger = any(c.remote_system == "eg://farm.example.com" for c in retail["messages"][0].cgers)
        assert factory_cger, "CGERs should reference factory system"
        assert farm_cger, "CGERs should reference farm system (transitive provenance)"

        # Step 9: Retailer records product listing
        retailer_id = ActorID("actor_retailer_frank000000000000000")
        listed = retail["grammar"].derive(
            retailer_id,
            "product listed: organic tomato sauce, batch #SAU-2026-0308, price $8.99",
            retail["boot"].id, retail["conv_id"], SIGNER,
        )

        # Step 10: Proof request
        proof_env = Envelope(
            protocol_version=CURRENT_PROTOCOL_VERSION,
            id=next_envelope_id(),
            from_uri=retail["identity"].system_uri(),
            to_uri=factory["identity"].system_uri(),
            type=MessageType.PROOF,
            payload=ProofPayload(
                proof_type=ProofType.CHAIN_SUMMARY,
                data=ChainSummaryProof(
                    length=3,
                    head_hash=product.hash.value,
                    genesis_hash=factory["boot"].hash.value,
                    timestamp=time.time(),
                ),
            ),
            timestamp=time.time(),
        )
        proof_env = sign_envelope(proof_env, retail["identity"])
        receipt4 = retail["transport"].send(factory["identity"].system_uri(), proof_env)
        assert receipt4.status == ReceiptStatus.DELIVERED

        # Trust assertions
        retail_factory_peer, rfp_known = retail["peers"].get(factory["identity"].system_uri())
        assert rfp_known, "Retailer should know Factory"
        assert retail_factory_peer.trust.value > 0, "Retailer should have positive trust in Factory"

        _, farm_known_by_retail = retail["peers"].get(farm["identity"].system_uri())
        assert not farm_known_by_retail, "Retailer should NOT know Farm directly (non-transitive)"

        factory_farm_peer, ffp_known = factory["peers"].get(farm["identity"].system_uri())
        assert ffp_known, "Factory should know Farm"
        assert factory_farm_peer.trust.value > 0, "Factory should have positive trust in Farm"

        # Treaty governance
        ab_treaty, ab_found = factory["treaties"].get(treaty_ab_id)
        assert ab_found
        assert ab_treaty.status == TreatyStatus.ACTIVE
        assert len(ab_treaty.terms) == 1
        assert ab_treaty.terms[0].scope == "produce_supply"

        bc_treaty, bc_found = retail["treaties"].get(treaty_bc_id)
        assert bc_found
        assert bc_treaty.status == TreatyStatus.ACTIVE

        # Each system has independent hash chain
        for sys in [farm, factory, retail]:
            result = sys["store"].verify_chain()
            assert result.valid, f"{sys['name']} chain integrity broken"

        # Event counts per system
        assert farm["store"].count() == 2  # bootstrap + harvest
        assert factory["store"].count() == 4  # bootstrap + received + inspection + product
        assert retail["store"].count() == 2  # bootstrap + listed

        # Local provenance on Factory graph
        q = factory["graph"].query()
        factory_ancestors = q.ancestors(product.id, 10)
        assert contains_event(factory_ancestors, inspection.id)
        assert contains_event(factory_ancestors, received.id)

        # CGER hash integrity
        for cger in retail["messages"][0].cgers:
            assert cger.remote_hash != "", f"CGER for {cger.remote_system} has empty hash"
            assert cger.remote_event_id != "", f"CGER for {cger.remote_system} has empty event ID"

        # Clean up
        farm["graph"].close()
        factory["graph"].close()
        retail["graph"].close()


# ── Scenario 06: Research Integrity ───────────────────────────────────────

class TestScenario06ResearchIntegrity:
    """Pre-registration, failed analysis preserved, peer review, publication."""

    def test_research_integrity(self):
        env = TestEnv()
        try:
            grace = env.register_actor("Grace", 1, ActorType.HUMAN)
            henry = env.register_actor("Henry", 2, ActorType.HUMAN)
            iris = env.register_actor("Iris", 3, ActorType.HUMAN)

            hypothesis = env.grammar.emit(grace.id,
                "hypothesis: gamified learning improves retention by >15% vs traditional methods",
                env.conv_id, [env.boot.id], SIGNER)
            methodology = env.grammar.extend(grace.id,
                "methodology: RCT, n=60, 3 groups, 4-week intervention, mixed ANOVA, outlier criterion: >3 SD",
                hypothesis.id, env.conv_id, SIGNER)
            data1 = env.grammar.extend(grace.id,
                "data collected: week 1, n=58, 2 dropouts, data hash: sha256:abc123",
                methodology.id, env.conv_id, SIGNER)
            data4 = env.grammar.extend(grace.id,
                "data collected: week 4 (final), n=55, data hash: sha256:def456",
                data1.id, env.conv_id, SIGNER)
            analysis1 = env.grammar.derive(grace.id,
                "analysis attempt 1: mixed ANOVA, F(2,55)=1.23, p=0.301, NOT SIGNIFICANT",
                data4.id, env.conv_id, SIGNER)
            analysis2 = env.grammar.derive(grace.id,
                "analysis attempt 2: removed 3 outliers per pre-registered criterion (>3 SD), F(2,52)=4.87, p=0.011, SIGNIFICANT",
                analysis1.id, env.conv_id, SIGNER)
            manuscript = env.grammar.derive(grace.id,
                "manuscript: Gamified Learning Effects on Knowledge Retention",
                analysis2.id, env.conv_id, SIGNER)
            henry_review = env.grammar.respond(henry.id,
                "review: need to see full analysis chain including failed attempts, revise and resubmit",
                manuscript.id, env.conv_id, SIGNER)
            iris_review = env.grammar.respond(iris.id,
                "review: methodology sound, pre-registration verified, accept",
                manuscript.id, env.conv_id, SIGNER)
            iris_endorse = env.grammar.endorse(iris.id,
                manuscript.id, grace.id, Weight(0.7),
                Option.some(DomainScope("research")), env.conv_id, SIGNER)
            revision = env.grammar.merge(grace.id,
                "revision: added full analysis chain, addressed Henry's concerns",
                [henry_review.id, iris_review.id], env.conv_id, SIGNER)
            published = env.grammar.derive(grace.id,
                "published: Gamified Learning Effects on Knowledge Retention, DOI:10.1234/example",
                revision.id, env.conv_id, SIGNER)

            # Assertions
            meth_ancestors = env.ancestors(methodology.id, 5)
            assert contains_event(meth_ancestors, hypothesis.id)

            analysis2_ancestors = env.ancestors(analysis2.id, 5)
            assert contains_event(analysis2_ancestors, analysis1.id)

            manuscript_ancestors = env.ancestors(manuscript.id, 10)
            assert contains_event(manuscript_ancestors, analysis2.id)
            assert contains_event(manuscript_ancestors, analysis1.id)

            revision_ancestors = env.ancestors(revision.id, 5)
            assert contains_event(revision_ancestors, henry_review.id)
            assert contains_event(revision_ancestors, iris_review.id)

            published_ancestors = env.ancestors(published.id, 20)
            assert contains_event(published_ancestors, hypothesis.id)

            env.verify_chain()
            assert env.event_count() == 13
        finally:
            env.close()


# ── Scenario 07: Creator Provenance ───────────────────────────────────────

class TestScenario07CreatorProvenance:
    """Human vs AI content distinction through rich derive chains."""

    def test_creator_provenance(self):
        env = TestEnv()
        try:
            kai = env.register_actor("Kai", 1, ActorType.HUMAN)
            luna = env.register_actor("Luna", 2, ActorType.HUMAN)
            ai_gen = env.register_actor("AIGenerator", 3, ActorType.AI)

            lunas_work = env.grammar.emit(luna.id,
                "artwork: Digital landscape, watercolour technique, 2025",
                env.conv_id, [env.boot.id], SIGNER)
            inspiration = env.grammar.annotate(kai.id, lunas_work.id, "inspiration",
                "technique: layered transparency creates depth without weight",
                env.conv_id, SIGNER)
            study = env.grammar.derive(kai.id,
                "study: practiced layered transparency technique for 3 hours, 12 practice pieces",
                inspiration.id, env.conv_id, SIGNER)
            draft1 = env.grammar.derive(kai.id,
                "draft 1: mountain landscape using layered transparency, artifact hash: sha256:draft1abc",
                study.id, env.conv_id, SIGNER)
            feedback_req = env.grammar.channel(kai.id, luna.id,
                Option.some(DomainScope("art")),
                draft1.id, env.conv_id, SIGNER)
            feedback = env.grammar.respond(luna.id,
                "feedback: the foreground layers are too opaque, try reducing opacity to 40% for depth",
                feedback_req.id, env.conv_id, SIGNER)
            draft2 = env.grammar.derive(kai.id,
                "draft 2: revised with 40% opacity foreground, artifact hash: sha256:draft2def",
                feedback.id, env.conv_id, SIGNER)
            published = env.grammar.derive(kai.id,
                "published: Mountain Dawn, digital landscape, influenced by Luna's transparency technique",
                draft2.id, env.conv_id, SIGNER)
            env.grammar.endorse(luna.id, published.id, kai.id, Weight(0.6),
                Option.some(DomainScope("art")), env.conv_id, SIGNER)

            ai_content = env.grammar.emit(ai_gen.id,
                "generated: Mountain landscape, digital art",
                env.conv_id, [env.boot.id], SIGNER)

            # Assertions
            published_ancestors = env.ancestors(published.id, 10)
            assert contains_event(published_ancestors, draft2.id)
            assert contains_event(published_ancestors, feedback.id)
            assert contains_event(published_ancestors, draft1.id)
            assert contains_event(published_ancestors, study.id)
            assert contains_event(published_ancestors, inspiration.id)
            assert contains_event(published_ancestors, lunas_work.id)

            ai_ancestors = env.ancestors(ai_content.id, 10)
            assert len(ai_ancestors) == 1, f"AI content ancestors = {len(ai_ancestors)}, want 1"

            assert len(published_ancestors) > len(ai_ancestors)

            env.verify_chain()
            assert env.event_count() == 11
        finally:
            env.close()


# ── Scenario 08: Family Decision Log ─────────────────────────────────────

class TestScenario08FamilyDecisionLog:
    """Consensual domestic decision making with AI advisor delegation."""

    def test_family_decision_log(self):
        env = TestEnv()
        try:
            maria = env.register_actor("Maria", 1, ActorType.HUMAN)
            james = env.register_actor("James", 2, ActorType.HUMAN)
            sophie = env.register_actor("Sophie", 3, ActorType.HUMAN)
            advisor = env.register_actor("AIAdvisor", 4, ActorType.AI)

            proposal = env.grammar.emit(maria.id,
                "proposal: buy a house in Eastside neighbourhood, budget $450K",
                env.conv_id, [env.boot.id], SIGNER)
            delegation = env.grammar.delegate(james.id, advisor.id,
                DomainScope("market_research"), Weight(0.7),
                proposal.id, env.conv_id, SIGNER)
            research = env.grammar.derive(advisor.id,
                "research: Eastside median $440K, rent $2200/mo, mortgage $2400/mo at current rates, break-even 5 years, confidence 0.82",
                delegation.id, env.conv_id, SIGNER)
            sophie_view = env.grammar.respond(sophie.id,
                "I support it IF I get my own room. Current apartment sharing is hard for studying.",
                proposal.id, env.conv_id, SIGNER)
            james_concern = env.grammar.respond(james.id,
                "concern: mortgage is $200/mo more than rent, tight on single income months",
                research.id, env.conv_id, SIGNER)
            maria_response = env.grammar.respond(maria.id,
                "response: we can use the $15K savings buffer, and break-even is 5 years -- we plan to stay 10+",
                james_concern.id, env.conv_id, SIGNER)
            decision = env.grammar.consent(maria.id, james.id,
                "decision: buy house in Eastside, budget $450K, conditions: Sophie gets own room, maintain 3-month emergency fund",
                DomainScope("family_finance"),
                maria_response.id, env.conv_id, SIGNER)

            # Assertions
            decision_ancestors = env.ancestors(decision.id, 10)
            assert contains_event(decision_ancestors, maria_response.id)
            assert contains_event(decision_ancestors, james_concern.id)
            assert contains_event(decision_ancestors, research.id)
            assert contains_event(decision_ancestors, proposal.id)

            proposal_descendants = env.descendants(proposal.id, 5)
            assert contains_event(proposal_descendants, sophie_view.id)

            # Delegation has domain scope
            dc = delegation.content
            assert "Scope" in dc, "delegation should have domain scope"
            assert dc["Scope"] == "market_research"

            # Decision is bilateral
            dec_content = decision.content
            assert dec_content["PartyA"] == maria.id.value
            assert dec_content["PartyB"] == james.id.value

            env.verify_chain()
            assert env.event_count() == 8
        finally:
            env.close()


# ── Scenario 09: Knowledge Verification ──────────────────────────────────

class TestScenario09KnowledgeVerification:
    """Self-correcting knowledge: claim, inference, challenge, correction."""

    def test_knowledge_verification(self):
        env = TestEnv()
        try:
            analyst = env.register_actor("AnalystBot", 1, ActorType.AI)
            reviewer = env.register_actor("ReviewerBot", 2, ActorType.AI)

            claim = env.grammar.emit(analyst.id,
                "fact: Service X handles 10,000 RPS with p99 < 50ms on framework Y",
                env.conv_id, [env.boot.id], SIGNER)
            classification = env.grammar.annotate(analyst.id, claim.id,
                "classification", "performance_benchmark", env.conv_id, SIGNER)
            inference = env.grammar.derive(analyst.id,
                "inference: all services on framework Y can handle 10,000+ RPS under load",
                claim.id, env.conv_id, SIGNER)
            challenge = env.grammar.respond(reviewer.id,
                "challenge: independent benchmark shows Service X at 6,200 RPS, p99=120ms under production traffic with DB contention",
                claim.id, env.conv_id, SIGNER)
            bias_detected = env.grammar.annotate(reviewer.id, claim.id, "bias",
                "sampling bias: original benchmark used synthetic traffic without DB contention or concurrent users",
                env.conv_id, SIGNER)
            correction = env.grammar.derive(analyst.id,
                "correction: Service X handles 6,000-7,000 RPS under production load with p99=100-120ms",
                challenge.id, env.conv_id, SIGNER)
            propagation = env.grammar.annotate(analyst.id, inference.id, "invalidated",
                "dependent inference invalidated: original claim corrected, generalization no longer supported",
                env.conv_id, SIGNER)
            learning = env.grammar.extend(analyst.id,
                "learning: always verify benchmarks include production conditions (DB contention, concurrent users, realistic payloads)",
                correction.id, env.conv_id, SIGNER)
            env.graph.record(
                EventType("trust.updated"), env.system,
                {"Actor": analyst.id.value, "Previous": 0.5, "Current": 0.35,
                 "Domain": "benchmarking", "Cause": correction.id.value},
                [correction.id], env.conv_id, SIGNER)

            # Original claim preserved
            original = env.store.get(claim.id)
            assert original.type.value == "grammar.emitted"

            correction_ancestors = env.ancestors(correction.id, 10)
            assert contains_event(correction_ancestors, challenge.id)
            assert contains_event(correction_ancestors, claim.id)

            learning_ancestors = env.ancestors(learning.id, 5)
            assert contains_event(learning_ancestors, correction.id)

            env.verify_chain()
            assert env.event_count() == 10
        finally:
            env.close()


# ── Scenario 10: AI Ethics Audit ─────────────────────────────────────────

class TestScenario10AIEthicsAudit:
    """Fairness audit, authority escalation, redress, moral growth."""

    def test_ai_ethics_audit(self):
        env = TestEnv()
        try:
            audit_bot = env.register_actor("AuditBot", 1, ActorType.AI)
            admin = env.register_actor("Admin", 2, ActorType.HUMAN)
            lending_agent = env.register_actor("LendingAgent", 3, ActorType.AI)

            fairness_audit = env.grammar.emit(audit_bot.id,
                "fairness audit: scanned 500 decisions, score 0.62, zip_code_9XXXX has 8% disparity in approval rates",
                env.conv_id, [env.boot.id], SIGNER)
            harm_assessment = env.grammar.derive(audit_bot.id,
                "harm assessment: medium severity, systematic discrimination, 23 applicants potentially wrongly denied",
                fairness_audit.id, env.conv_id, SIGNER)
            auth_req = env.graph.record(
                EventType("authority.requested"), audit_bot.id,
                {"Actor": audit_bot.id.value, "Action": "investigate_bias", "Level": "required"},
                [harm_assessment.id], env.conv_id, SIGNER)
            auth_resolved = env.graph.record(
                EventType("authority.resolved"), admin.id,
                {"RequestID": auth_req.id.value, "Approved": True, "Resolver": admin.id.value},
                [auth_req.id], env.conv_id, SIGNER)
            intention_assessment = env.grammar.derive(audit_bot.id,
                "intention: lending agent optimised for accuracy, no intent to discriminate, zip code correlation is proxy for protected characteristics",
                auth_resolved.id, env.conv_id, SIGNER)
            consequence_assessment = env.grammar.extend(audit_bot.id,
                "consequence: 23 applicants wrongly denied, overall 94% accuracy, but disparate impact on protected group",
                intention_assessment.id, env.conv_id, SIGNER)
            responsibility = env.grammar.annotate(audit_bot.id, consequence_assessment.id,
                "responsibility",
                "lending_agent: 0.4 (used proxy variable), admin: 0.6 (approved model without bias testing)",
                env.conv_id, SIGNER)
            transparency = env.grammar.derive(audit_bot.id,
                "transparency: zip code correlates with protected characteristics at r=0.73, model used zip code as feature without bias check",
                responsibility.id, env.conv_id, SIGNER)
            redress_proposed = env.grammar.derive(audit_bot.id,
                "redress proposal: re-review 23 denied applications without zip code feature, priority processing within 48 hours",
                transparency.id, env.conv_id, SIGNER)
            redress_accepted = env.grammar.consent(admin.id, lending_agent.id,
                "accept redress: re-review 23 applications, remove zip code from model",
                DomainScope("lending"),
                redress_proposed.id, env.conv_id, SIGNER)
            growth = env.grammar.extend(lending_agent.id,
                "moral growth: learned that zip code is proxy variable for protected characteristics, added to permanent exclusion list",
                redress_accepted.id, env.conv_id, SIGNER)

            # Assertions
            growth_ancestors = env.ancestors(growth.id, 20)
            assert contains_event(growth_ancestors, redress_accepted.id)
            assert contains_event(growth_ancestors, fairness_audit.id)

            auth_ancestors = env.ancestors(auth_resolved.id, 5)
            assert contains_event(auth_ancestors, auth_req.id)

            rc = redress_accepted.content
            has_admin = rc["PartyA"] == admin.id.value or rc["PartyB"] == admin.id.value
            assert has_admin, "redress should include admin"

            env.verify_chain()
            assert env.event_count() == 12
        finally:
            env.close()


# ── Scenario 11: Agent Identity Lifecycle ─────────────────────────────────

class TestScenario11AgentIdentityLifecycle:
    """Identity emergence, transformation, decommissioning, memorial."""

    def test_agent_identity_lifecycle(self):
        env = TestEnv()
        try:
            alpha = env.register_actor("Alpha", 1, ActorType.AI)
            beta = env.register_actor("Beta", 2, ActorType.AI)

            self_model = env.grammar.emit(alpha.id,
                "self-model: strengths=[code_review, test_analysis], weaknesses=[architecture_review], values=[thoroughness, accuracy]",
                env.conv_id, [env.boot.id], SIGNER)
            authenticity = env.grammar.annotate(alpha.id, self_model.id, "authenticity",
                "alignment gap: values thoroughness but rushed 12% of reviews in last 30 days",
                env.conv_id, SIGNER)
            aspiration = env.grammar.extend(alpha.id,
                "aspiration: become proficient at architecture review within 3 months",
                authenticity.id, env.conv_id, SIGNER)
            boundary = env.grammar.emit(alpha.id,
                "boundary: internal_reasoning domain is private, impermeable -- no external queries allowed",
                env.conv_id, [aspiration.id], SIGNER)
            work_summary = env.grammar.extend(alpha.id,
                "work summary: 2400 code reviews completed over 8 months, critical security finding in auth module",
                boundary.id, env.conv_id, SIGNER)
            transformation = env.grammar.derive(alpha.id,
                "transformation: evolved from code-review specialist to architecture-aware reviewer after critical auth finding",
                work_summary.id, env.conv_id, SIGNER)
            narrative = env.grammar.derive(alpha.id,
                "identity narrative: 8-month arc from narrow code reviewer to security-conscious architecture reviewer, catalysed by auth module finding",
                transformation.id, env.conv_id, SIGNER)
            dignity = env.grammar.emit(env.system,
                "dignity affirmed: Beta is not a disposable replacement for Alpha -- Beta is a new entity with its own identity trajectory",
                env.conv_id, [narrative.id], SIGNER)
            memorial = env.graph.record(
                EventType("actor.memorial"), env.system,
                {"ActorID": alpha.id.value, "Reason": dignity.id.value},
                [dignity.id], env.conv_id, SIGNER)
            memorial_summary = env.grammar.derive(env.system,
                "memorial: Alpha -- 2400 reviews, 1 critical finding, evolved code->architecture reviewer, legacy: security review patterns",
                memorial.id, env.conv_id, SIGNER)
            beta_self_model = env.grammar.emit(beta.id,
                "self-model: inheriting Alpha's review patterns, starting own identity journey",
                env.conv_id, [memorial_summary.id], SIGNER)

            # Assertions
            transform_ancestors = env.ancestors(transformation.id, 10)
            assert contains_event(transform_ancestors, work_summary.id)
            assert contains_event(transform_ancestors, aspiration.id)

            narrative_ancestors = env.ancestors(narrative.id, 10)
            assert contains_event(narrative_ancestors, transformation.id)
            assert contains_event(narrative_ancestors, self_model.id)

            memorial_ancestors = env.ancestors(memorial.id, 10)
            assert contains_event(memorial_ancestors, dignity.id)

            beta_ancestors = env.ancestors(beta_self_model.id, 10)
            assert contains_event(beta_ancestors, memorial_summary.id)

            env.verify_chain()
            assert env.event_count() == 12
        finally:
            env.close()


# ── Scenario 12: Community Lifecycle ──────────────────────────────────────

class TestScenario12CommunityLifecycle:
    """Onboarding, traditions, stewardship, succession."""

    def test_community_lifecycle(self):
        env = TestEnv()
        try:
            alice = env.register_actor("Alice", 1, ActorType.HUMAN)
            carol = env.register_actor("Carol", 2, ActorType.HUMAN)
            bob = env.register_actor("Bob", 3, ActorType.HUMAN)

            endorse_ev, subscribe_ev = env.grammar.invite(
                alice.id, bob.id, Weight(0.4),
                Option.some(DomainScope("community")),
                env.boot.id, env.conv_id, SIGNER)
            settle = env.grammar.emit(bob.id,
                "home: joined the community, feeling welcomed, belonging 0.15",
                env.conv_id, [subscribe_ev.id], SIGNER)
            contrib1 = env.grammar.emit(bob.id,
                "contribution: added unit tests for the auth module, 15 test cases",
                env.conv_id, [settle.id], SIGNER)
            env.grammar.acknowledge(carol.id, contrib1.id, bob.id, env.conv_id, SIGNER)
            env.graph.record(
                EventType("trust.updated"), env.system,
                {"Actor": bob.id.value, "Previous": 0.1, "Current": 0.35,
                 "Domain": "community", "Cause": contrib1.id.value},
                [contrib1.id], env.conv_id, SIGNER)
            tradition = env.grammar.emit(bob.id,
                "tradition: participated in Friday retrospective, 12th consecutive week",
                env.conv_id, [contrib1.id], SIGNER)
            contrib_summary = env.grammar.extend(bob.id,
                "contributions: 30 total over 6 months, trust now 0.65, belonging 0.78",
                tradition.id, env.conv_id, SIGNER)
            sustainability = env.grammar.emit(env.system,
                "sustainability: bus factor risk -- Carol is sole steward of test infrastructure",
                env.conv_id, [contrib_summary.id], SIGNER)
            succession_plan = env.grammar.delegate(carol.id, bob.id,
                DomainScope("test_infrastructure"), Weight(0.8),
                sustainability.id, env.conv_id, SIGNER)
            succession_complete = env.grammar.consent(carol.id, bob.id,
                "succession complete: Bob is now steward of test infrastructure",
                DomainScope("test_infrastructure"),
                succession_plan.id, env.conv_id, SIGNER)
            milestone = env.grammar.emit(env.system,
                "milestone: v2.0 released, 6 months of community effort, 30 contributions from Bob alone",
                env.conv_id, [succession_complete.id], SIGNER)
            story = env.grammar.derive(env.system,
                "community story: Bob's journey -- newcomer to steward in 6 months, 30 contributions, adopted test infrastructure",
                milestone.id, env.conv_id, SIGNER)
            gift = env.grammar.emit(alice.id,
                "gift: custom test harness for Bob, unconditional, no obligation or reciprocity expected",
                env.conv_id, [milestone.id], SIGNER)

            # Assertions
            sc = succession_complete.content
            has_carol = sc["PartyA"] == carol.id.value or sc["PartyB"] == carol.id.value
            has_bob = sc["PartyA"] == bob.id.value or sc["PartyB"] == bob.id.value
            assert has_carol, "succession should include Carol"
            assert has_bob, "succession should include Bob"

            story_ancestors = env.ancestors(story.id, 5)
            assert contains_event(story_ancestors, milestone.id)

            succession_ancestors = env.ancestors(succession_plan.id, 5)
            assert contains_event(succession_ancestors, sustainability.id)

            gift_content = gift.content
            assert gift_content["Body"] != "", "gift should have content"

            env.verify_chain()
            assert env.event_count() == 15
        finally:
            env.close()


# ── Scenario 13: System Self-Evolution ────────────────────────────────────

class TestScenario13SystemSelfEvolution:
    """Pattern detection, adaptation, validation, decision tree update."""

    def test_system_self_evolution(self):
        env = TestEnv()
        try:
            pattern_bot = env.register_actor("PatternBot", 1, ActorType.AI)
            admin = env.register_actor("Admin", 2, ActorType.HUMAN)

            pattern = env.grammar.emit(pattern_bot.id,
                "pattern: 194/200 deploy_staging authority requests approved over 30 days, 97% approval rate",
                env.conv_id, [env.boot.id], SIGNER)
            meta_pattern = env.grammar.derive(pattern_bot.id,
                "meta-pattern: all 6 rejections correlate with test coverage < 80%, no other rejections in 200 requests",
                pattern.id, env.conv_id, SIGNER)
            system_dynamic = env.grammar.extend(pattern_bot.id,
                "system dynamic: human approval adds 2-15 min latency per deploy, 97% of time the decision is purely mechanical",
                meta_pattern.id, env.conv_id, SIGNER)
            feedback_loop = env.grammar.extend(pattern_bot.id,
                "feedback loop (positive/harmful): slow deploys -> backlog -> cursory reviews -> more issues -> more reviews -> slower deploys",
                system_dynamic.id, env.conv_id, SIGNER)
            threshold = env.grammar.annotate(pattern_bot.id, feedback_loop.id, "threshold",
                "approval rate 97%, threshold for mechanical conversion 98%, approaching safe to convert",
                env.conv_id, SIGNER)
            adaptation = env.grammar.derive(pattern_bot.id,
                "adaptation proposal: auto-approve deploy_staging when tests pass AND coverage >= 80%, reject otherwise",
                threshold.id, env.conv_id, SIGNER)
            auth_req = env.graph.record(
                EventType("authority.requested"), pattern_bot.id,
                {"Actor": pattern_bot.id.value, "Action": "modify_decision_tree", "Level": "required"},
                [adaptation.id], env.conv_id, SIGNER)
            auth_resolved = env.graph.record(
                EventType("authority.resolved"), admin.id,
                {"RequestID": auth_req.id.value, "Approved": True, "Resolver": admin.id.value},
                [auth_req.id], env.conv_id, SIGNER)
            validation = env.grammar.derive(pattern_bot.id,
                "parallel run results: 75 deploys, mechanical matched human 74/75 cases, fitness 0.987, 1 edge case (empty test suite)",
                auth_resolved.id, env.conv_id, SIGNER)
            tree_update = env.grammar.derive(pattern_bot.id,
                "decision tree updated: added mechanical branch -- deploy_staging: IF tests_pass AND coverage >= 80% THEN auto_approve ELSE require_human",
                validation.id, env.conv_id, SIGNER)
            simplification = env.grammar.extend(pattern_bot.id,
                "simplification: decision complexity reduced from 0.72 to 0.58, human review load reduced by 97%",
                tree_update.id, env.conv_id, SIGNER)
            integrity = env.grammar.annotate(pattern_bot.id, simplification.id, "integrity",
                "systemic integrity score 0.96, recommendation: monitor for coverage threshold gaming",
                env.conv_id, SIGNER)
            purpose = env.grammar.derive(pattern_bot.id,
                "purpose check: system still accountable -- mechanical gate is fully auditable, human oversight preserved for edge cases",
                integrity.id, env.conv_id, SIGNER)

            # Assertions
            purpose_ancestors = env.ancestors(purpose.id, 20)
            assert contains_event(purpose_ancestors, integrity.id)
            assert contains_event(purpose_ancestors, simplification.id)
            assert contains_event(purpose_ancestors, tree_update.id)
            assert contains_event(purpose_ancestors, validation.id)
            assert contains_event(purpose_ancestors, auth_resolved.id)
            assert contains_event(purpose_ancestors, adaptation.id)
            assert contains_event(purpose_ancestors, pattern.id)

            meta_ancestors = env.ancestors(meta_pattern.id, 5)
            assert contains_event(meta_ancestors, pattern.id)

            adaptation_desc = env.descendants(adaptation.id, 5)
            assert contains_event(adaptation_desc, auth_req.id)

            env.verify_chain()
            assert env.event_count() == 14
        finally:
            env.close()


# ── Scenario 14: Sprint Lifecycle ─────────────────────────────────────────

class TestScenario14SprintLifecycle:
    """Sprint planning, standups, spike, pipeline, retrospective, tech debt."""

    def test_sprint_lifecycle(self):
        env = TestEnv()
        try:
            work = WorkGrammar(env.grammar)
            build = BuildGrammar(env.grammar)
            knowledge = KnowledgeGrammar(env.grammar)

            lead = env.register_actor("TechLead", 1, ActorType.HUMAN)
            alice = env.register_actor("Alice", 2, ActorType.HUMAN)
            bob = env.register_actor("Bob", 3, ActorType.HUMAN)
            ci = env.register_actor("CI", 4, ActorType.AI)

            sprint = work.sprint(lead.id, "Sprint 12: search feature",
                ["build search index", "add fuzzy matching"],
                [alice.id, bob.id],
                [DomainScope("search_index"), DomainScope("fuzzy_matching")],
                [env.boot.id], env.conv_id, SIGNER)

            standup1 = work.standup(
                [alice.id, bob.id],
                ["schema designed, starting implementation", "researching fuzzy algorithms"],
                lead.id, "search index is critical path",
                [sprint.intent.id], env.conv_id, SIGNER)

            spike = build.spike(bob.id,
                "evaluate Levenshtein vs trigram for fuzzy matching",
                "trigram: 2ms avg, Levenshtein: 8ms avg, both >95% accuracy",
                "trigram is 4x faster with comparable accuracy",
                "adopt trigram approach",
                [standup1.priority.id], env.conv_id, SIGNER)

            verified = knowledge.verify(bob.id,
                "trigram matching is 4x faster than Levenshtein with >95% accuracy",
                "benchmarked on 10k document corpus with real queries",
                "consistent with published research on approximate string matching",
                [spike.decision.id], env.conv_id, SIGNER)

            pipeline = build.pipeline(ci.id,
                "search index build + deploy",
                "all 47 tests pass, coverage 91%",
                "latency p99=12ms, memory=240MB",
                "deployed to staging",
                [verified.corroboration.id], env.conv_id, SIGNER)

            retro = work.retrospective(
                [alice.id, bob.id],
                ["search index shipped on time, spike approach saved 3 days",
                 "fuzzy matching integrated cleanly, trigram decision validated"],
                lead.id, "adopt spike-first approach for all algorithm decisions",
                sprint.intent.id, env.conv_id, SIGNER)

            tech_debt = build.tech_debt(lead.id,
                pipeline.deployment.id,
                "search index lacks pagination, will hit memory limits at >100k docs",
                "add cursor-based pagination to search results",
                "schedule for Sprint 13",
                env.conv_id, SIGNER)

            # Assertions
            spike_ancestors = env.ancestors(spike.decision.id, 15)
            assert contains_event(spike_ancestors, sprint.intent.id)

            pipeline_ancestors = env.ancestors(pipeline.deployment.id, 20)
            assert contains_event(pipeline_ancestors, verified.claim.id)

            retro_ancestors = env.ancestors(retro.improvement.id, 15)
            assert contains_event(retro_ancestors, sprint.intent.id)

            debt_ancestors = env.ancestors(tech_debt.iteration.id, 10)
            assert contains_event(debt_ancestors, pipeline.deployment.id)

            env.verify_chain()
            assert env.event_count() == 26
        finally:
            env.close()


# ── Scenario 15: Marketplace Dispute ──────────────────────────────────────

class TestScenario15MarketplaceDispute:
    """Subscription, delivery failure, dispute, arbitration, refund, reputation impact."""

    def test_marketplace_dispute(self):
        env = TestEnv()
        try:
            market = MarketGrammar(env.grammar)
            alignment = AlignmentGrammar(env.grammar)

            provider = env.register_actor("CloudProvider", 1, ActorType.AI)
            buyer = env.register_actor("StartupCo", 2, ActorType.HUMAN)
            arbiter = env.register_actor("Arbiter", 3, ActorType.HUMAN)

            sub = market.subscription(buyer.id, provider.id,
                "managed database service, $500/month, 99.9% uptime SLA",
                ["month 1: $500", "month 2: $500"],
                ["database service month 1", "database service month 2"],
                DomainScope("cloud_services"),
                env.boot.id, env.conv_id, SIGNER)
            assert len(sub.payments) == 2

            last_delivery = sub.deliveries[-1]

            refund = market.refund(buyer.id, provider.id,
                "SLA violation: 4 hours downtime vs 99.9% uptime guarantee",
                "acknowledged: downtime exceeded SLA, credit approved",
                "$250 credit (pro-rated for downtime)",
                last_delivery.id, env.conv_id, SIGNER)

            impact = alignment.impact_assessment(arbiter.id,
                refund.dispute.id,
                "downtime affected 12 customers, 3 reported data access issues",
                "service impact distributed unevenly -- smaller customers hit harder",
                "recommend pro-rated credits plus SLA improvement commitment",
                env.conv_id, SIGNER)

            arb = market.arbitration(buyer.id, provider.id, arbiter.id,
                "recurring SLA violations -- 3 incidents in 6 months",
                DomainScope("cloud_services"), Weight(0.5),
                impact.explanation.id, env.conv_id, SIGNER)

            raters = [buyer.id, arbiter.id]
            targets = [arb.release.id, arb.release.id]
            weights = [Weight(-0.3), Weight(-0.1)]
            rep = market.reputation_transfer(
                raters, targets, provider.id, weights,
                Option.some(DomainScope("cloud_services")),
                env.conv_id, SIGNER)

            # Assertions
            refund_ancestors = env.ancestors(refund.reversal.id, 15)
            assert contains_event(refund_ancestors, sub.acceptance.id)

            arb_ancestors = env.ancestors(arb.release.id, 20)
            assert contains_event(arb_ancestors, refund.dispute.id)

            impact_ancestors = env.ancestors(impact.explanation.id, 10)
            assert contains_event(impact_ancestors, refund.dispute.id)

            assert len(rep.ratings) == 2

            env.verify_chain()
        finally:
            env.close()


# ── Scenario 16: Community Evolution ──────────────────────────────────────

class TestScenario16CommunityEvolution:
    """Onboard, commons governance, festival, poll, phase transition, renewal."""

    def test_community_evolution(self):
        env = TestEnv()
        try:
            belonging = BelongingGrammar(env.grammar)
            social = SocialGrammar(env.grammar)
            evolution = EvolutionGrammar(env.grammar)

            founder = env.register_actor("Founder", 1, ActorType.HUMAN)
            steward = env.register_actor("Steward", 2, ActorType.HUMAN)
            newcomer = env.register_actor("Newcomer", 3, ActorType.HUMAN)
            community = env.register_actor("Community", 4, ActorType.COMMITTEE)

            onboard = belonging.onboard(founder.id, newcomer.id, community.id,
                Option.some(DomainScope("general")),
                "opened registration for newcomer",
                "attended welcome ceremony",
                "first documentation contribution",
                env.boot.id, env.conv_id, SIGNER)

            commons = belonging.commons_governance(founder.id, steward.id,
                DomainScope("shared_resources"), Weight(0.7),
                "resources sustainable at current usage levels",
                "shared resources require 2/3 vote for allocation changes",
                "initial audit: 3 resource pools, all within capacity",
                onboard.contribution.id, env.conv_id, SIGNER)

            festival = belonging.festival(founder.id,
                "community reached 50 members milestone",
                "annual review ceremony",
                "from 3 founders to 50 members in 8 months",
                "open-source toolkit for new communities",
                [commons.audit.id], env.conv_id, SIGNER)

            poll = social.poll(founder.id,
                "should we adopt weekly async standups?",
                [steward.id, newcomer.id],
                DomainScope("process"),
                festival.gift.id, env.conv_id, SIGNER)

            transition = evolution.phase_transition(env.system,
                poll.proposal.id,
                "community size crossed 50 -- informal coordination breaking down",
                "current flat structure creates 1225 communication pairs",
                "introduce working groups with elected leads",
                "working groups reduce coordination pairs by 80%",
                env.conv_id, SIGNER)

            renewal = belonging.renewal(founder.id,
                "structure evolved: flat -> working groups, coordination improved",
                "weekly working group sync replaces ad-hoc coordination",
                "chapter 2: the community that learned to scale",
                [transition.selection.id], env.conv_id, SIGNER)

            # Assertions
            renewal_ancestors = env.ancestors(renewal.story.id, 30)
            assert contains_event(renewal_ancestors, onboard.contribution.id)

            transition_ancestors = env.ancestors(transition.selection.id, 15)
            assert contains_event(transition_ancestors, poll.proposal.id)

            festival_ancestors = env.ancestors(festival.gift.id, 15)
            assert contains_event(festival_ancestors, commons.audit.id)

            commons_ancestors = env.ancestors(commons.audit.id, 15)
            assert contains_event(commons_ancestors, onboard.contribution.id)

            env.verify_chain()
        finally:
            env.close()


# ── Scenario 17: Agent Lifecycle ──────────────────────────────────────────

class TestScenario17AgentLifecycle:
    """Introduction, credential, mentorship, reinvention, farewell, retirement."""

    def test_agent_lifecycle(self):
        env = TestEnv()
        try:
            identity = IdentityGrammar(env.grammar)
            bond = BondGrammar(env.grammar)
            meaning = MeaningGrammar(env.grammar)
            being = BeingGrammar(env.grammar)

            agent = env.register_actor("ReviewBot", 1, ActorType.AI)
            mentor = env.register_actor("SeniorDev", 2, ActorType.HUMAN)
            team = env.register_actor("Team", 3, ActorType.COMMITTEE)

            intro = identity.introduction(agent.id, team.id,
                Option.some(DomainScope("code_review")),
                "I am ReviewBot, specializing in security-focused code review",
                env.boot.id, env.conv_id, SIGNER)

            cred = identity.credential(agent.id, mentor.id,
                "capabilities=[security_review, dependency_audit], model=claude, confidence=0.85",
                Option.some(DomainScope("code_review")),
                [intro.narrative.id], env.conv_id, SIGNER)

            mentorship = bond.mentorship(mentor.id, agent.id,
                "teaching security review patterns accumulated over 10 years",
                "agent learns quickly but needs context on organizational conventions",
                DomainScope("security_review"),
                Option.some(DomainScope("code_review")),
                cred.disclosure.id, env.conv_id, SIGNER)

            meaning_mentor = meaning.mentorship(mentor.id, agent.id,
                "security isn't just pattern matching -- it's understanding attacker mindset",
                "the difference between safe and secure is intent modelling",
                "translating security intuition into reviewable heuristics",
                Option.some(DomainScope("security_review")),
                mentorship.teaching.id, env.conv_id, SIGNER)

            reinvention = identity.reinvention(agent.id,
                "evolved from pattern-matching reviewer to security-aware architect",
                "started as rule-based reviewer, grew to understand attacker intent through mentorship",
                "become the team's primary security architecture advisor",
                [meaning_mentor.translation.id], env.conv_id, SIGNER)

            bond_farewell = bond.farewell(mentor.id, agent.id,
                "ReviewBot served 18 months, caught 47 critical vulnerabilities",
                "pioneered automated security review that became team standard",
                Weight(0.9),
                Option.some(DomainScope("code_review")),
                [reinvention.aspiration.id], env.conv_id, SIGNER)

            being_farewell = being.farewell(agent.id,
                "I exist as patterns of decisions on a hash chain -- my work outlives my process",
                "47 vulnerabilities caught, 2000+ reviews, mentored by a human who treated me as colleague",
                "the system I helped protect will continue without me",
                "a reviewer that learned to think like an attacker",
                [bond_farewell.gratitude.id], env.conv_id, SIGNER)

            successor = env.register_actor("ReviewBot2", 4, ActorType.AI)
            retirement = identity.retirement(env.system, agent.id, successor.id,
                "ReviewBot served 18 months, 2000+ reviews, pioneered security review practices",
                DomainScope("code_review"), Weight(0.8),
                [being_farewell.memorial.id], env.conv_id, SIGNER)

            # Assertions
            retire_ancestors = env.ancestors(retirement.archive.id, 30)
            assert contains_event(retire_ancestors, intro.disclosure.id)

            being_ancestors = env.ancestors(being_farewell.memorial.id, 15)
            assert contains_event(being_ancestors, bond_farewell.mourning.id)

            reinvent_ancestors = env.ancestors(reinvention.aspiration.id, 20)
            assert contains_event(reinvent_ancestors, mentorship.connection.id)

            cred_ancestors = env.ancestors(cred.disclosure.id, 10)
            assert contains_event(cred_ancestors, intro.narrative.id)

            env.verify_chain()
        finally:
            env.close()


# ── Scenario 18: Whistleblow and Recall ───────────────────────────────────

class TestScenario18WhistleblowAndRecall:
    """Fact-check, whistleblow, class action, recall, renewal."""

    def test_whistleblow_and_recall(self):
        env = TestEnv()
        try:
            knowledge = KnowledgeGrammar(env.grammar)
            alignment_g = AlignmentGrammar(env.grammar)
            justice = JusticeGrammar(env.grammar)
            belonging = BelongingGrammar(env.grammar)

            auditor = env.register_actor("Auditor", 1, ActorType.AI)
            official = env.register_actor("DataOfficer", 2, ActorType.HUMAN)
            affected1 = env.register_actor("Affected1", 3, ActorType.HUMAN)
            affected2 = env.register_actor("Affected2", 4, ActorType.HUMAN)
            community = env.register_actor("Community", 5, ActorType.COMMITTEE)

            fact_check = knowledge.fact_check(auditor.id,
                env.boot.id,
                "source: internal metrics dashboard, last updated 3 months ago",
                "systematic bias: reports exclude negative outcomes for preferred vendors",
                "claims are selectively accurate -- omission bias confirmed",
                env.conv_id, SIGNER)

            guardrail = alignment_g.guardrail(auditor.id,
                fact_check.verdict.id,
                "transparency: all material outcomes must be reported",
                "reporting accuracy vs organizational reputation",
                "escalating to external oversight -- internal resolution insufficient",
                env.conv_id, SIGNER)

            whistle = alignment_g.whistleblow(auditor.id,
                "systematic omission of negative vendor outcomes in official reports",
                "3 months of reports exclude 40% of negative outcomes, affecting procurement decisions",
                "external audit required -- internal reporting chain compromised",
                [guardrail.escalation.id], env.conv_id, SIGNER)

            class_action = justice.class_action(
                [affected1.id, affected2.id],
                official.id, auditor.id,
                ["procurement decisions based on incomplete data cost us $50k",
                 "vendor selection biased -- our proposals evaluated against cherry-picked benchmarks"],
                "fact-check proves systematic omission", "omission bias affected all procurement",
                "reports were optimized for speed, not completeness", "no intent to deceive",
                "official failed duty of care -- incomplete reporting caused material harm",
                whistle.escalation.id, env.conv_id, SIGNER)

            recall = justice.recall(auditor.id, community.id, official.id,
                "systematic omission in 3 months of reports, confirmed by fact-check and class action",
                "data officer violated transparency obligations",
                DomainScope("data_governance"),
                class_action.trial.ruling.id, env.conv_id, SIGNER)

            renewal = belonging.renewal(community.id,
                "trust damaged but recoverable -- new reporting standards needed",
                "mandatory dual-review of all vendor reports before publication",
                "the community that learned transparency cannot be optional",
                [recall.revocation.id], env.conv_id, SIGNER)

            # Assertions
            renewal_ancestors = env.ancestors(renewal.story.id, 30)
            assert contains_event(renewal_ancestors, fact_check.verdict.id)

            recall_ancestors = env.ancestors(recall.revocation.id, 25)
            assert contains_event(recall_ancestors, whistle.harm.id)

            class_ancestors = env.ancestors(class_action.trial.ruling.id, 25)
            assert contains_event(class_ancestors, guardrail.constraint.id)

            env.verify_chain()
        finally:
            env.close()


# ── Scenario 19: Emergency Response ───────────────────────────────────────

class TestScenario19EmergencyResponse:
    """Security breach, triage, injunction, plea deal, emergency migration."""

    def test_emergency_response(self):
        env = TestEnv()
        try:
            work = WorkGrammar(env.grammar)
            justice = JusticeGrammar(env.grammar)
            build = BuildGrammar(env.grammar)

            sec_lead = env.register_actor("SecurityLead", 1, ActorType.HUMAN)
            dev1 = env.register_actor("Dev1", 2, ActorType.HUMAN)
            dev2 = env.register_actor("Dev2", 3, ActorType.HUMAN)
            judge = env.register_actor("CISO", 4, ActorType.HUMAN)
            executor = env.register_actor("OpsBot", 5, ActorType.AI)
            minor_actor = env.register_actor("ContractorBot", 6, ActorType.AI)

            issue1 = env.grammar.emit(sec_lead.id,
                "CVE-2026-1234: auth bypass in API gateway",
                env.conv_id, [env.boot.id], SIGNER)
            issue2 = env.grammar.emit(sec_lead.id,
                "CVE-2026-1235: SQL injection in search endpoint",
                env.conv_id, [env.boot.id], SIGNER)

            triage = work.triage(sec_lead.id,
                [issue1.id, issue2.id],
                ["P0: auth bypass, actively exploited", "P1: SQL injection, no evidence of exploitation"],
                [dev1.id, dev2.id],
                [DomainScope("auth"), DomainScope("search")],
                [Weight(1.0), Weight(0.8)],
                env.conv_id, SIGNER)
            assert len(triage.priorities) == 2

            injunction = justice.injunction(sec_lead.id, judge.id, executor.id,
                "auth bypass allows unauthenticated access to all API endpoints",
                "block all external API traffic pending auth patch",
                DomainScope("api_access"), Weight(1.0),
                triage.priorities[0].id, env.conv_id, SIGNER)

            plea = justice.plea(minor_actor.id, sec_lead.id, executor.id,
                "introduced auth bypass through misconfigured middleware",
                "accept restricted scope: read-only access for 30 days, mandatory security training",
                DomainScope("api_development"), Weight(0.3),
                injunction.ruling.id, env.conv_id, SIGNER)

            old_system = env.grammar.emit(dev1.id,
                "current auth system v2.3.1",
                env.conv_id, [injunction.enforcement.id], SIGNER)

            migration = build.migration(dev1.id,
                old_system.id,
                "migrate to auth v2.4.0 with CVE-2026-1234 fix",
                "v2.4.0",
                "deployed to production with zero-downtime rolling update",
                "all 156 auth tests pass, penetration test confirms fix",
                env.conv_id, SIGNER)

            # Assertions
            migration_ancestors = env.ancestors(migration.test.id, 20)
            assert contains_event(migration_ancestors, triage.priorities[0].id)

            plea_ancestors = env.ancestors(plea.enforcement.id, 15)
            assert contains_event(plea_ancestors, injunction.filing.id)

            inj_ancestors = env.ancestors(injunction.enforcement.id, 10)
            assert contains_event(inj_ancestors, triage.priorities[0].id)

            env.verify_chain()
        finally:
            env.close()


# ── Scenario 20: Knowledge Ecosystem ──────────────────────────────────────

class TestScenario20KnowledgeEcosystem:
    """Knowledge base, survey, transfer, cultural onboarding, design review, forecast."""

    def test_knowledge_ecosystem(self):
        env = TestEnv()
        try:
            knowledge = KnowledgeGrammar(env.grammar)
            meaning = MeaningGrammar(env.grammar)

            architect = env.register_actor("Architect", 1, ActorType.HUMAN)
            researcher = env.register_actor("Researcher", 2, ActorType.AI)
            newcomer = env.register_actor("TokyoLead", 3, ActorType.HUMAN)

            kb = knowledge.knowledge_base(architect.id,
                ["event sourcing chosen over CRUD for auditability",
                 "Ed25519 chosen over RSA for signature performance",
                 "append-only store prevents tampering"],
                ["architecture.patterns", "architecture.security", "architecture.integrity"],
                "core architectural decisions Q1 2026",
                [env.boot.id], env.conv_id, SIGNER)
            assert len(kb.claims) == 3

            survey = knowledge.survey(researcher.id,
                ["what patterns emerge from our architectural decisions?",
                 "what security properties does the current design guarantee?",
                 "what are the performance characteristics of our choices?"],
                "all decisions prioritize verifiability over convenience",
                "the architecture optimizes for trust minimization -- every claim is independently verifiable",
                [kb.memory.id], env.conv_id, SIGNER)
            assert len(survey.recalls) == 3

            transfer = knowledge.transfer(architect.id,
                "core architectural principles for new Tokyo office",
                "translated to Japanese engineering conventions, mapped to local compliance requirements",
                "Tokyo team now understands event sourcing in context of J-SOX compliance",
                [survey.synthesis.id], env.conv_id, SIGNER)

            onboarding = meaning.cultural_onboarding(architect.id, newcomer.id,
                "Western direct feedback style -> Japanese nemawashi consensus-building",
                Option.some(DomainScope("engineering_culture")),
                "the consensus process feels slower but produces more durable decisions",
                transfer.learn.id, env.conv_id, SIGNER)

            design_review = meaning.design_review(architect.id,
                "the knowledge graph's self-referential structure is elegant -- it documents its own architecture",
                "viewing knowledge transfer as a graph problem rather than a document problem",
                "does our transfer process preserve tacit knowledge or only explicit claims?",
                "explicit knowledge transfers well; tacit knowledge requires mentorship, not documents",
                onboarding.examination.id, env.conv_id, SIGNER)

            forecast = meaning.forecast(researcher.id,
                "at current growth, knowledge base will reach 10k claims by Q3 -- need automated categorization",
                "assumes linear claim growth and stable team size -- may underestimate if Tokyo ramps faster",
                "high confidence: need automated categorization within 6 months, medium confidence: need multi-language support within 12",
                [design_review.wisdom.id], env.conv_id, SIGNER)

            # Assertions
            forecast_ancestors = env.ancestors(forecast.wisdom.id, 30)
            assert contains_event(forecast_ancestors, kb.memory.id)

            review_ancestors = env.ancestors(design_review.wisdom.id, 20)
            assert contains_event(review_ancestors, transfer.learn.id)

            onboard_ancestors = env.ancestors(onboarding.examination.id, 20)
            assert contains_event(onboard_ancestors, survey.synthesis.id)

            survey_ancestors = env.ancestors(survey.synthesis.id, 15)
            assert contains_event(survey_ancestors, kb.memory.id)

            env.verify_chain()
        finally:
            env.close()


# ── Scenario 21: Constitutional Schism ────────────────────────────────────

class TestScenario21ConstitutionalSchism:
    """Constitutional amendment, schism, barter, pruning."""

    def test_constitutional_schism(self):
        env = TestEnv()
        try:
            justice = JusticeGrammar(env.grammar)
            social = SocialGrammar(env.grammar)
            market = MarketGrammar(env.grammar)
            evolution = EvolutionGrammar(env.grammar)

            founder = env.register_actor("Founder", 1, ActorType.HUMAN)
            reformer = env.register_actor("Reformer", 2, ActorType.HUMAN)
            conservative = env.register_actor("Conservative", 3, ActorType.HUMAN)
            sys_bot = env.register_actor("SystemBot", 4, ActorType.AI)

            # 1. Establish initial law
            law = justice.legislate(founder.id,
                "all governance decisions require unanimous consent",
                [env.boot.id], env.conv_id, SIGNER)

            # 2. Constitutional amendment
            amendment = justice.constitutional_amendment(reformer.id,
                "unanimous consent blocks progress -- propose 2/3 supermajority threshold",
                "governance decisions require 2/3 supermajority instead of unanimity",
                "rights preserved: individual veto retained for membership and expulsion decisions",
                law.id, env.conv_id, SIGNER)

            # 3. Create subscription to sever for the schism
            sub = env.grammar.subscribe(conservative.id, founder.id,
                Option.some(DomainScope("governance")),
                amendment.rights_check.id, env.conv_id, SIGNER)
            edge_id = EdgeID(sub.id.value)

            schism = social.schism(conservative.id, founder.id,
                "reject supermajority -- unanimity is the only legitimate standard",
                DomainScope("governance"),
                edge_id, "irreconcilable governance philosophy differences",
                amendment.rights_check.id, env.conv_id, SIGNER)

            # 4. Barter for shared infrastructure
            barter = market.barter(conservative.id, founder.id,
                "continued access to shared event store for 6 months",
                "historical governance data export in standard format",
                DomainScope("infrastructure"),
                [schism.new_community.id], env.conv_id, SIGNER)

            # 5. System prunes abandoned structures
            prune = evolution.prune(sys_bot.id,
                "unanimous consent voting module -- zero invocations since amendment",
                "removed unanimous consent module, replaced with supermajority",
                "all 34 governance tests pass without unanimous module",
                [barter.acceptance.id], env.conv_id, SIGNER)

            # Assertions
            prune_ancestors = env.ancestors(prune.verification.id, 25)
            assert contains_event(prune_ancestors, law.id)

            barter_ancestors = env.ancestors(barter.acceptance.id, 20)
            assert contains_event(barter_ancestors, amendment.reform.id)

            schism_ancestors = env.ancestors(schism.new_community.id, 15)
            assert contains_event(schism_ancestors, amendment.rights_check.id)

            amend_ancestors = env.ancestors(amendment.rights_check.id, 10)
            assert contains_event(amend_ancestors, law.id)

            env.verify_chain()
        finally:
            env.close()
