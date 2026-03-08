"""Tests for Grammar — social grammar vertex operations."""

import pytest

from eventgraph.event import NoopSigner, create_bootstrap, create_event
from eventgraph.grammar import Grammar
from eventgraph.store import InMemoryStore
from eventgraph.types import ActorID, ConversationID, EventID, EventType


ALICE = ActorID("alice")
BOB = ActorID("bob")
CONV = ConversationID("conv_test")
SIGNER = NoopSigner()


def _setup():
    """Create a store with a bootstrap event and return (store, grammar, boot)."""
    store = InMemoryStore()
    boot = create_bootstrap(source=ALICE, signer=SIGNER)
    store.append(boot)
    grammar = Grammar(store)
    return store, grammar, boot


class TestEmit:
    def test_creates_emitted_event(self):
        store, grammar, boot = _setup()
        evt = grammar.emit(ALICE, "hello world", CONV, [boot.id], SIGNER)

        assert evt.type.value == "grammar.emitted"
        assert evt.content["Body"] == "hello world"
        assert evt.source.value == "alice"
        assert boot.id.value in [c.value for c in evt.causes]

    def test_requires_causes(self):
        _, grammar, _ = _setup()
        with pytest.raises(ValueError, match="at least one cause"):
            grammar.emit(ALICE, "hello", CONV, [], SIGNER)

    def test_multiple_causes(self):
        store, grammar, boot = _setup()
        e1 = grammar.emit(ALICE, "first", CONV, [boot.id], SIGNER)
        e2 = grammar.emit(ALICE, "second", CONV, [boot.id, e1.id], SIGNER)

        cause_ids = [c.value for c in e2.causes]
        assert boot.id.value in cause_ids
        assert e1.id.value in cause_ids


class TestRespond:
    def test_creates_responded_event(self):
        store, grammar, boot = _setup()
        evt = grammar.respond(BOB, "I agree", boot.id, CONV, SIGNER)

        assert evt.type.value == "grammar.responded"
        assert evt.content["Body"] == "I agree"
        assert evt.content["Parent"] == boot.id.value
        assert evt.causes[0].value == boot.id.value

    def test_causal_link_to_parent(self):
        store, grammar, boot = _setup()
        e1 = grammar.emit(ALICE, "statement", CONV, [boot.id], SIGNER)
        e2 = grammar.respond(BOB, "reply", e1.id, CONV, SIGNER)

        assert e2.causes[0].value == e1.id.value


class TestDerive:
    def test_creates_derived_event(self):
        store, grammar, boot = _setup()
        e1 = grammar.emit(ALICE, "original", CONV, [boot.id], SIGNER)
        evt = grammar.derive(BOB, "insight from original", e1.id, CONV, SIGNER)

        assert evt.type.value == "grammar.derived"
        assert evt.content["Body"] == "insight from original"
        assert evt.content["Source"] == e1.id.value
        assert evt.causes[0].value == e1.id.value


class TestExtend:
    def test_creates_extended_event(self):
        store, grammar, boot = _setup()
        e1 = grammar.emit(ALICE, "part one", CONV, [boot.id], SIGNER)
        evt = grammar.extend(ALICE, "part two", e1.id, CONV, SIGNER)

        assert evt.type.value == "grammar.extended"
        assert evt.content["Body"] == "part two"
        assert evt.content["Previous"] == e1.id.value
        assert evt.causes[0].value == e1.id.value


class TestRetract:
    def test_creates_retracted_event(self):
        store, grammar, boot = _setup()
        e1 = grammar.emit(ALICE, "mistake", CONV, [boot.id], SIGNER)
        evt = grammar.retract(ALICE, e1.id, "was wrong", CONV, SIGNER)

        assert evt.type.value == "grammar.retracted"
        assert evt.content["Target"] == e1.id.value
        assert evt.content["Reason"] == "was wrong"
        assert evt.causes[0].value == e1.id.value

    def test_validates_authorship(self):
        store, grammar, boot = _setup()
        e1 = grammar.emit(ALICE, "alice said this", CONV, [boot.id], SIGNER)

        with pytest.raises(ValueError, match="cannot retract"):
            grammar.retract(BOB, e1.id, "not mine to retract", CONV, SIGNER)


class TestAnnotate:
    def test_creates_annotated_event(self):
        store, grammar, boot = _setup()
        e1 = grammar.emit(ALICE, "something", CONV, [boot.id], SIGNER)
        evt = grammar.annotate(BOB, e1.id, "sentiment", "positive", CONV, SIGNER)

        assert evt.type.value == "grammar.annotated"
        assert evt.content["Target"] == e1.id.value
        assert evt.content["Key"] == "sentiment"
        assert evt.content["Value"] == "positive"
        assert evt.causes[0].value == e1.id.value


class TestMerge:
    def test_creates_merged_event(self):
        store, grammar, boot = _setup()
        e1 = grammar.emit(ALICE, "point A", CONV, [boot.id], SIGNER)
        e2 = grammar.emit(ALICE, "point B", CONV, [e1.id], SIGNER)
        evt = grammar.merge(ALICE, "synthesis", [e1.id, e2.id], CONV, SIGNER)

        assert evt.type.value == "grammar.merged"
        assert evt.content["Body"] == "synthesis"
        assert evt.content["Sources"] == [e1.id.value, e2.id.value]
        cause_ids = [c.value for c in evt.causes]
        assert e1.id.value in cause_ids
        assert e2.id.value in cause_ids

    def test_requires_at_least_two_sources(self):
        store, grammar, boot = _setup()
        e1 = grammar.emit(ALICE, "only one", CONV, [boot.id], SIGNER)

        with pytest.raises(ValueError, match="at least 2"):
            grammar.merge(ALICE, "not enough", [e1.id], CONV, SIGNER)

    def test_rejects_empty_sources(self):
        _, grammar, _ = _setup()

        with pytest.raises(ValueError, match="at least 2"):
            grammar.merge(ALICE, "nothing", [], CONV, SIGNER)


class TestChainIntegrity:
    def test_chain_valid_after_multiple_operations(self):
        store, grammar, boot = _setup()

        e1 = grammar.emit(ALICE, "first", CONV, [boot.id], SIGNER)
        e2 = grammar.respond(BOB, "reply", e1.id, CONV, SIGNER)
        e3 = grammar.derive(ALICE, "derived", e2.id, CONV, SIGNER)
        e4 = grammar.extend(ALICE, "extended", e3.id, CONV, SIGNER)
        e5 = grammar.annotate(BOB, e4.id, "tag", "important", CONV, SIGNER)
        e6 = grammar.emit(ALICE, "another", CONV, [e5.id], SIGNER)
        e7 = grammar.merge(ALICE, "merged", [e5.id, e6.id], CONV, SIGNER)

        # 1 bootstrap + 7 grammar events
        assert store.count() == 8

        verification = store.verify_chain()
        assert verification.valid is True
        assert verification.length == 8

    def test_prev_hash_links_correctly(self):
        store, grammar, boot = _setup()

        e1 = grammar.emit(ALICE, "first", CONV, [boot.id], SIGNER)
        assert e1.prev_hash.value == boot.hash.value

        e2 = grammar.respond(BOB, "reply", e1.id, CONV, SIGNER)
        assert e2.prev_hash.value == e1.hash.value

        e3 = grammar.retract(ALICE, e1.id, "oops", CONV, SIGNER)
        assert e3.prev_hash.value == e2.hash.value
