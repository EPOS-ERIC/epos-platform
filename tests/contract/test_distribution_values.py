from urllib.parse import parse_qs, urlsplit

from distribution_fixtures import DistributionContractTestCase, parameter, web_distribution
from fixtures import assert_problem


class DistributionValuesContractTest(DistributionContractTestCase):
    def test_scalar_serialization_and_explicit_zero_values(self) -> None:
        """Verify scalar types and explicit empty, false and zero values survive rendering."""
        detail = self.create_draft([web_distribution([
            parameter("text", default="default"),
            parameter("flag", "BOOLEAN", position=1, default=True),
            parameter("count", "INTEGER", position=2, default=5),
            parameter("ratio", "FLOAT", position=3, default=1.5),
        ])])
        distribution = detail["metadata"]["distributions"][0]
        identifiers = {item["name"]: item["id"] for item in distribution["web_service"]["parameters"]}
        for values, expected in (
            ({"text": "hello", "flag": True, "count": 42, "ratio": 2.5},
             {"text": ["hello"], "flag": ["true"], "count": ["42"], "ratio": ["2.5"]}),
            ({"text": "", "flag": False, "count": 0, "ratio": 0},
             {"text": [""], "flag": ["false"], "count": ["0"], "ratio": ["0"]}),
        ):
            with self.subTest(values=values):
                prepared = self.prepare(
                    distribution, values={identifiers[name]: value for name, value in values.items()},
                )
                self.assertEqual(parse_qs(urlsplit(prepared["url"]).query, keep_blank_values=True), expected)
        for name, value in (
            ("text", 42), ("flag", "false"), ("flag", 0), ("count", "42"),
            ("count", 1.5), ("count", True), ("ratio", "2.5"), ("ratio", False),
        ):
            with self.subTest(parameter=name, value=value):
                self.assert_prepare_problem(
                    self.prepare_response(distribution, {identifiers[name]: value}),
                    target="parameters[0].value",
                )

    def test_array_shape_and_per_element_validation(self) -> None:
        """Verify array multiplicity, element types and constraints are enforced."""
        detail = self.create_draft([web_distribution([
            parameter("scalar"),
            parameter("strings", position=1, multiple_values=True, explode=True),
            parameter("integers", "INTEGER", position=2, multiple_values=True, explode=True,
                      minimum=0, maximum=3),
            parameter("floats", "FLOAT", position=3, multiple_values=True, explode=True,
                      minimum=0.5, maximum=2.5),
            parameter("patterns", position=4, multiple_values=True, explode=True, pattern="^[A-Z]{2}$",
                      enum=[{"value": "AB"}, {"value": "CD"}]),
            parameter("integer_enum", "INTEGER", position=5, multiple_values=True, explode=True,
                      enum=[{"value": 1}, {"value": 3}]),
            parameter("float_enum", "FLOAT", position=6, multiple_values=True, explode=True,
                      enum=[{"value": 1.5}, {"value": 2.5}]),
        ])])
        distribution = detail["metadata"]["distributions"][0]
        identifiers = {item["name"]: item["id"] for item in distribution["web_service"]["parameters"]}
        values = {
            "strings": ["red", "blue"], "integers": [0, 3], "floats": [0.5, 1, 2.5],
            "patterns": ["AB", "CD"], "integer_enum": [1, 3], "float_enum": [1.5, 2.5],
        }
        prepared = self.prepare(
            distribution, values={identifiers[name]: value for name, value in values.items()},
        )
        query = parse_qs(urlsplit(prepared["url"]).query)
        for name, expected in values.items():
            self.assertEqual(query[name], [str(item) for item in expected])
        for name, value in (
            ("scalar", ["red"]),
            ("strings", "red"),
            ("integers", 1),
            ("strings", ["red", 1]),
            ("integers", [0, 4]), ("integers", [-1, 0]),
            ("integers", [0, 1.5]), ("integers", [0, True]),
            ("floats", [0.5, 2.6]), ("floats", [0.4, 2.5]),
            ("floats", [0.5, "1"]), ("patterns", ["AB", "bad"]),
            ("integer_enum", [1, 2]), ("float_enum", [1.5, 2]),
        ):
            with self.subTest(parameter=name, value=value):
                self.assert_prepare_problem(
                    self.prepare_response(distribution, {identifiers[name]: value}),
                    target="parameters[0].value",
                )

    def test_scalar_constraints(self) -> None:
        """Verify scalar enum, pattern and numeric boundary checks."""
        detail = self.create_draft([web_distribution([
            parameter("enum", enum=[{"value": "red"}, {"value": "blue"}]),
            parameter("pattern", position=1, pattern="^[A-Z]{2}$"),
            parameter("integer", "INTEGER", position=2, minimum=0, maximum=10),
            parameter("float", "FLOAT", position=3, minimum=0.5, maximum=2.5),
            parameter("integer_enum", "INTEGER", position=4, enum=[{"value": 1}, {"value": 3}]),
            parameter("float_enum", "FLOAT", position=5, enum=[{"value": 1.5}, {"value": 2.5}]),
        ])])
        distribution = detail["metadata"]["distributions"][0]
        identifiers = {item["name"]: item["id"] for item in distribution["web_service"]["parameters"]}
        for name, valid, invalid in (
            ("enum", ["red", "blue"], ["green", "RED"]),
            ("pattern", ["AB"], ["bad"]),
            ("integer", [0, 10], [-1, 11]),
            ("float", [0.5, 1, 2.5], [0.4, 2.6]),
            ("integer_enum", [1, 3], [2]),
            ("float_enum", [1.5, 2.5], [2]),
        ):
            for value in valid:
                with self.subTest(parameter=name, accepted=value):
                    prepared = self.prepare(distribution, values={identifiers[name]: value})
                    self.assertEqual(parse_qs(urlsplit(prepared["url"]).query)[name], [str(value)])
            for value in invalid:
                with self.subTest(parameter=name, rejected=value):
                    self.assert_prepare_problem(
                        self.prepare_response(distribution, {identifiers[name]: value}),
                        target="parameters[0].value",
                    )

    def test_defaults_and_overrides(self) -> None:
        """Verify scalar default limits, typed array defaults and overrides."""
        detail = self.create_draft([web_distribution([
            parameter("defaulted", position=1, required=True, default=["default"]),
            parameter("optional", position=2),
            parameter("multiple", position=3, multiple_values=True, explode=True, default=["one", "three"],
                      enum=[{"value": value} for value in ("one", "two", "three")]),
            parameter("joined", position=4, multiple_values=True, default=["one", "three"],
                      enum=[{"value": value} for value in ("one", "two", "three")]),
            parameter("integer_multiple", "INTEGER", position=5, multiple_values=True, explode=True,
                      default=[1, 3], enum=[{"value": value} for value in (1, 2, 3)]),
            parameter("float_joined", "FLOAT", position=6, multiple_values=True,
                      default=[0.5, 2.5], enum=[{"value": value} for value in (0.5, 1.5, 2.5)]),
        ])])
        distribution = detail["metadata"]["distributions"][0]
        identifiers = {item["name"]: item["id"] for item in distribution["web_service"]["parameters"]}
        prepared = self.prepare(distribution)
        query = parse_qs(urlsplit(prepared["url"]).query)
        self.assertEqual(query, {
            "defaulted": ["default"],
            "multiple": ["one", "three"], "joined": ["one,three"],
            "integer_multiple": ["1", "3"], "float_joined": ["0.5,2.5"],
        })
        prepared = self.prepare(distribution, values={
            identifiers["defaulted"]: "override",
            identifiers["multiple"]: ["two", "three"],
        })
        query = parse_qs(urlsplit(prepared["url"]).query)
        self.assertEqual(query["defaulted"], ["override"])
        self.assertEqual(query["multiple"], ["two", "three"])
        self.assertNotIn("optional", query)

        invalid = self.replace_response(detail, {
            "title": detail["metadata"]["title"],
            "description": "Reject multiple defaults for scalar parameters",
            "distributions": [web_distribution([parameter("scalar", default=["one", "two"])])],
        })
        problem = assert_problem(self, invalid, 400, "validation_failed")
        self.assertTrue(
            problem.get("target", "").startswith(
                "metadata.distributions[0].web_service.parameters[0]"
            ),
            problem,
        )
        unchanged = self.details(detail["resource_id"], detail["id"])
        self.assertEqual(unchanged["revision"], detail["revision"])
