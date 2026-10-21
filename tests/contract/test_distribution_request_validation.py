import json
from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, parameter, web_distribution


class DistributionRequestValidationContractTest(DistributionContractTestCase):
    def test_request_shape_and_parameter_assignment_validation(self) -> None:
        """Verify JSON shape and assignment errors are rejected before rendering."""
        foreign_service = web_distribution([parameter("foreign")])
        foreign_service["title"] = "foreign"
        detail = self.create_draft([
            web_distribution([
                parameter(required=True, default="default"),
                parameter("locked", position=1, read_only=True, default="fixed"),
                parameter("rendered", "TEMPLATE", position=2, template="fixed", parameters=[]),
            ]),
            foreign_service,
        ])
        by_title = {item["title"]: item for item in detail["metadata"]["distributions"]}
        distribution, other = by_title["prepared-service"], by_title["foreign"]
        parameters = {item["name"]: item for item in distribution["web_service"]["parameters"]}
        identifier = parameters["value"]["id"]
        self.assertIn("value=default", self.prepare(distribution)["url"])
        malformed = [
            ("missing body", ""), ("invalid JSON", "{"), ("null body", "null"),
            ("array body", "[]"), ("missing parameters", "{}"),
            ("wrong parameters type", '{"parameters":"wrong"}'),
            ("null parameters", '{"parameters":null}'),
            ("object parameters", '{"parameters":{}}'),
            ("unknown property", '{"parameters":[],"extra":true}'),
        ]
        invalid_assignments = [
            ("missing ID", [{"value": "value"}]),
            ("invalid ID", [{"id": "bad", "value": "value"}]),
            ("null ID", [{"id": None, "value": "value"}]),
            ("null assignment", [None]),
            ("non-object assignment", ["value"]),
            ("missing value", [{"id": identifier}]),
            ("null", [{"id": identifier, "value": None}]),
            ("object", [{"id": identifier, "value": {"key": "value"}}]),
            ("nested array", [{"id": identifier, "value": [["value"]]}]),
            ("empty array", [{"id": identifier, "value": []}]),
            ("wrong ID type", [{"id": 123, "value": "value"}]),
            ("unknown assignment property", [{"id": identifier, "value": "value", "extra": 1}]),
        ]
        malformed.extend(
            (name, json.dumps({"parameters": assignments}))
            for name, assignments in invalid_assignments
        )
        for name, body in malformed:
            with self.subTest(case=name):
                response = self.gateway.post(
                    "/distributions/{distributionId}/prepare", distributionId=distribution["id"],
                    data=body, headers={"Content-Type": "application/json"},
                )
                self.assert_prepare_problem(response)

        foreign = other["web_service"]["parameters"][0]["id"]
        for name, assignments, target in (
            ("unknown ID", [{"id": str(uuid4()), "value": "value"}], "parameters[0].id"),
            ("foreign ID", [{"id": foreign, "value": "value"}], "parameters[0].id"),
            ("duplicate", [{"id": identifier, "value": "a"}, {"id": identifier, "value": "b"}],
             "parameters[1].id"),
            ("read only", [{"id": parameters["locked"]["id"], "value": "changed"}], "parameters[0].id"),
            ("template", [{"id": parameters["rendered"]["id"], "value": "changed"}], "parameters[0].id"),
        ):
            with self.subTest(case=name):
                response = self.gateway.post(
                    "/distributions/{distributionId}/prepare", distributionId=distribution["id"],
                    json={"parameters": assignments},
                )
                self.assert_prepare_problem(response, target=target)

    def test_invalid_bearer_token_is_rejected_even_for_public_distribution(self) -> None:
        """Verify an invalid supplied token cannot fall back to anonymous preparation."""
        detail = self.create_draft([web_distribution()])
        distribution = detail["metadata"]["distributions"][0]
        for state in ("DRAFT", "PUBLISHED"):
            with self.subTest(state=state):
                if state == "PUBLISHED":
                    for action in ("submit", "publish"):
                        self.transition(detail, action)
                self.assert_prepare_problem(
                    self.prepare_response(
                        distribution, headers={"Authorization": "Bearer invalid-contract-token"},
                    ),
                    401, "unauthenticated", secrets=("invalid-contract-token",),
                )
