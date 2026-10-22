from typing import Any
from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, parameter, web_distribution
from fixtures import assert_problem, create_group


class DistributionParameterRolesContractTest(DistributionContractTestCase):
    def test_all_roles_round_trip_independently_of_property(self) -> None:
        """Verify each explicit role and generic property survive create, read, and replace."""
        parameters = [
            parameter("format", role="FORMAT", read_only=True, default="json"),
            parameter("start", "DATETIME", position=1, role="START_DATE", property="custom:start"),
            parameter("end", "DATETIME", position=2, role="END_DATE", property="custom:end"),
            parameter("west", "FLOAT", position=3, role="MIN_LON", property="custom:west"),
            parameter("east", "INTEGER", position=4, role="MAX_LON", property="custom:east"),
            parameter("south", "FLOAT", position=5, role="MIN_LAT", property="custom:south"),
            parameter("north", "INTEGER", position=6, role="MAX_LAT", property="custom:north"),
        ]
        distribution = web_distribution(parameters)
        detail = self.create_draft([distribution])
        expected = {
            "format": ("FORMAT", None),
            "start": ("START_DATE", "custom:start"),
            "end": ("END_DATE", "custom:end"),
            "west": ("MIN_LON", "custom:west"),
            "east": ("MAX_LON", "custom:east"),
            "south": ("MIN_LAT", "custom:south"),
            "north": ("MAX_LAT", "custom:north"),
        }
        self.assert_role_metadata(detail, expected)

        response = self.replace_response(
            detail,
            {
                "title": detail["metadata"]["title"],
                "description": detail["metadata"]["description"],
                "distributions": [distribution],
            },
        )
        self.assertEqual(response.status_code, 200)
        updated = self.details(detail["resource_id"], detail["version_id"])
        self.assert_role_metadata(updated, expected)

    def test_role_can_be_omitted_or_cleared(self) -> None:
        """Verify omitted and null roles remain unset, including after replacement."""
        role_parameter = parameter(
            "date", "DATETIME", role="START_DATE", property="schema:startDate",
        )
        distribution = web_distribution([
            role_parameter,
            parameter("omitted", property="schema:encodingFormat"),
            parameter("null", "FLOAT", position=2, role=None, property="epos:westernmostLongitude"),
        ])
        detail = self.create_draft([distribution])
        params = self.parameters(detail)
        self.assertEqual(params["date"].get("role"), "START_DATE")
        self.assertIsNone(params["omitted"].get("role"))
        self.assertIsNone(params["null"].get("role"))

        role_parameter.pop("role")
        response = self.replace_response(
            detail,
            {
                "title": detail["metadata"]["title"],
                "description": detail["metadata"]["description"],
                "distributions": [distribution],
            },
        )
        self.assertEqual(response.status_code, 200)
        updated = self.details(detail["resource_id"], detail["version_id"])
        params = self.parameters(updated)
        self.assertIsNone(params["date"].get("role"))
        self.assertEqual(params["date"]["property"], "schema:startDate")

    def test_partial_roles_and_same_role_on_separate_services_are_valid(self) -> None:
        """Verify incomplete role sets and repeated roles across services are accepted."""
        distributions = []
        for title, role, kind in (
            ("first", "START_DATE", "DATETIME"),
            ("second", "START_DATE", "DATETIME"),
            ("west-only", "MIN_LON", "FLOAT"),
        ):
            distribution = web_distribution([
                parameter("value", kind, role=role),
            ])
            distribution["title"] = title
            distribution["web_service"]["name"] = title
            distributions.append(distribution)

        detail = self.create_draft(distributions)
        roles = {
            item["title"]: item["web_service"]["parameters"][0]["role"]
            for item in detail["metadata"]["distributions"]
        }
        self.assertEqual(roles, {
            "first": "START_DATE",
            "second": "START_DATE",
            "west-only": "MIN_LON",
        })

    def test_invalid_role_values_and_schema_combinations_are_rejected(self) -> None:
        """Verify unknown, duplicate, and schema-incompatible roles fail validation."""
        group_id = create_group(self.gateway, f"parameter-roles-{uuid4().hex}")
        duplicate = [
            parameter("start", "DATETIME", role="START_DATE"),
            parameter("other-start", "DATETIME", position=1, role="START_DATE"),
        ]
        cases = [
            ("unknown role", [parameter("unknown", role="UNSUPPORTED")]),
            ("duplicate role", duplicate),
            ("date role requires datetime", [parameter("start", role="START_DATE")]),
            ("coordinate role requires numeric", [parameter("west", "BOOLEAN", role="MIN_LON")]),
            (
                "coordinate role is scalar",
                [parameter("west", "FLOAT", role="MIN_LON", multiple_values=True, enum=[{"value": 1.0}])],
            ),
            (
                "template cannot have role",
                [parameter("template", "TEMPLATE", role="START_DATE", template="fixed", parameters=[])],
            ),
        ]
        for name, parameters in cases:
            with self.subTest(case=name):
                response = self.gateway.post(
                    "/groups/{groupId}/resources",
                    groupId=group_id,
                    json={
                        "resource_type": "dataset",
                        "metadata": {
                            "title": f"{name}-{uuid4().hex}",
                            "description": "Parameter role validation",
                            "distributions": [web_distribution(parameters)],
                        },
                    },
                )
                assert_problem(self, response, 400, "validation_failed")

    def assert_role_metadata(
        self, detail: dict[str, Any], expected: dict[str, tuple[str, str | None]],
    ) -> None:
        self.assertEqual(
            {
                item["name"]: (item.get("role"), item.get("property"))
                for item in self.parameters(detail).values()
            },
            expected,
        )

    def parameters(self, detail: dict[str, Any]) -> dict[str, dict[str, Any]]:
        return {
            item["name"]: item
            for item in detail["metadata"]["distributions"][0]["web_service"]["parameters"]
        }
