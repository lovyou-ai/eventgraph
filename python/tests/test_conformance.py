"""Conformance tests against Go reference implementation values.

The Go conformance tests in go/pkg/event/conformance_test.go are the authority.
The canonical form format is: version|prev_hash|causes|id|type|source|conversation_id|timestamp_nanos|content_json
"""

import json
import os

import pytest

from eventgraph.event import canonical_content_json, canonical_form, compute_hash
from eventgraph.types import (
    Activation,
    Cadence,
    Hash,
    Layer,
    Score,
    Weight,
)
from eventgraph.errors import OutOfRangeError, InvalidFormatError

# Load vectors for type validation
_VECTORS_PATH = os.path.join(
    os.path.dirname(__file__), "..", "..", "docs", "conformance", "canonical-vectors.json"
)

with open(_VECTORS_PATH) as f:
    VECTORS = json.load(f)


class TestCanonicalFormConformance:
    """Tests matching Go reference: go/pkg/event/conformance_test.go"""

    def test_bootstrap_canonical_form(self):
        """Matches TestConformanceBootstrapCanonicalForm in Go."""
        content = {
            "ActorID": "actor_00000000000000000000000000000001",
            "ChainGenesis": "0000000000000000000000000000000000000000000000000000000000000000",
            "Timestamp": "2023-11-14T22:13:20Z",
        }
        content_json = canonical_content_json(content)
        canon = canonical_form(
            version=1,
            prev_hash="",
            causes=[],  # bootstrap: empty causes
            event_id="019462a0-0000-7000-8000-000000000001",
            event_type="system.bootstrapped",
            source="actor_00000000000000000000000000000001",
            conversation_id="conv_00000000000000000000000000000001",
            timestamp_nanos=1700000000000000000,
            content_json=content_json,
        )

        # Go reference: starts with "1|||" (empty prev_hash AND empty causes)
        assert canon.startswith("1|||")

        expected_canonical = (
            '1|||019462a0-0000-7000-8000-000000000001|system.bootstrapped'
            '|actor_00000000000000000000000000000001'
            '|conv_00000000000000000000000000000001'
            '|1700000000000000000'
            '|{"ActorID":"actor_00000000000000000000000000000001",'
            '"ChainGenesis":"0000000000000000000000000000000000000000000000000000000000000000",'
            '"Timestamp":"2023-11-14T22:13:20Z"}'
        )
        assert canon == expected_canonical

        h = compute_hash(canon)
        # Go reference hash from TestConformanceBootstrapCanonicalForm
        assert h.value == "f7cae7ae11c1232a932c64f2302432c0e304dffce80f3935e688980dfbafeb75"

    def test_trust_updated_hash(self):
        """Matches TestConformanceTrustUpdatedCanonicalForm in Go."""
        content = {
            "Actor": "actor_00000000000000000000000000000002",
            "Cause": "019462a0-0000-7000-8000-000000000001",
            "Current": 0.85,
            "Domain": "code_review",
            "Previous": 0.8,
        }
        content_json = canonical_content_json(content)
        causes = ["019462a0-0000-7000-8000-000000000001"]
        canon = canonical_form(
            version=1,
            prev_hash="a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
            causes=causes,
            event_id="019462a0-0000-7000-8000-000000000002",
            event_type="trust.updated",
            source="actor_00000000000000000000000000000001",
            conversation_id="conv_00000000000000000000000000000001",
            timestamp_nanos=1700000001000000000,
            content_json=content_json,
        )

        h = compute_hash(canon)
        # Go reference hash from TestConformanceTrustUpdatedCanonicalForm
        assert h.value == "b2fbcd2684868f0b0d07d2f5136b52f14b8e749da7b4b7bae2a22f67147152b7"

    def test_edge_created_key_ordering_hash(self):
        """Matches TestConformanceEdgeCreatedKeyOrdering in Go."""
        content = {
            "Weight": 0.5,
            "From": "actor_00000000000000000000000000000001",
            "To": "actor_00000000000000000000000000000002",
            "EdgeType": "Trust",
            "Direction": "Centripetal",
        }
        content_json = canonical_content_json(content)
        # Verify key ordering: Direction < EdgeType < From < To < Weight
        assert content_json.index("Direction") < content_json.index("EdgeType")
        assert content_json.index("EdgeType") < content_json.index("From")
        assert content_json.index("From") < content_json.index("To")
        assert content_json.index("To") < content_json.index("Weight")

        causes = ["019462a0-0000-7000-8000-000000000001"]
        canon = canonical_form(
            version=1,
            prev_hash="b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3",
            causes=causes,
            event_id="019462a0-0000-7000-8000-000000000003",
            event_type="edge.created",
            source="actor_00000000000000000000000000000001",
            conversation_id="conv_00000000000000000000000000000001",
            timestamp_nanos=1700000002000000000,
            content_json=content_json,
        )

        h = compute_hash(canon)
        # Go reference hash from TestConformanceEdgeCreatedKeyOrdering
        assert h.value == "4e5c6710ca9325676663b4a66d2e82114fcd8fb49dbe5705795051e0b0be374c"

    def test_integer_float_formatting(self):
        """Go outputs 1 for 1.0, not 1.0 — all languages must match."""
        content = {"count": 1.0, "rate": 0.5}
        json_str = canonical_content_json(content)
        assert json_str == '{"count":1,"rate":0.5}'

    def test_null_omission(self):
        """Null/None fields must be omitted entirely, not included as null."""
        content = {"Actor": "actor_1", "Scope": None, "Reason": None}
        json_str = canonical_content_json(content)
        assert json_str == '{"Actor":"actor_1"}'

    def test_number_formatting_rules(self):
        """Verify specific number formatting from conformance vectors."""
        cases = [(1.0, "1"), (0.5, "0.5"), (0.1, "0.1"), (100.0, "100"), (0.001, "0.001")]
        for value, expected in cases:
            json_str = canonical_content_json({"v": value})
            assert json_str == '{"v":' + expected + '}', f"Failed for {value}"

    def test_hash_determinism(self):
        """Matches TestConformanceHashDeterminism in Go."""
        canonical = "1||test|system.bootstrapped|actor_test|conv_test|1000|{}"
        h1 = compute_hash(canonical)
        h2 = compute_hash(canonical)
        assert h1.value == h2.value


class TestTypeValidationConformance:
    """Tests against type_validation vectors in canonical-vectors.json."""

    @pytest.mark.parametrize("case", VECTORS["type_validation"]["invalid"],
                             ids=lambda c: f"{c['type']}_{c['reason']}")
    def test_invalid_rejected(self, case):
        type_map = {"Score": Score, "Weight": Weight, "Activation": Activation,
                     "Layer": Layer, "Cadence": Cadence, "Hash": Hash}
        cls = type_map[case["type"]]
        with pytest.raises((OutOfRangeError, InvalidFormatError)):
            cls(case["value"])

    @pytest.mark.parametrize("case", VECTORS["type_validation"]["valid"],
                             ids=lambda c: f"{c['type']}_{c['value']}")
    def test_valid_accepted(self, case):
        type_map = {"Score": Score, "Weight": Weight, "Activation": Activation,
                     "Layer": Layer, "Cadence": Cadence}
        cls = type_map[case["type"]]
        obj = cls(case["value"])
        assert obj.value == case["value"]
