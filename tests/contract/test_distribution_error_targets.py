from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, child, parameter, web_distribution
from fixtures import assert_problem, create_group


class DistributionErrorTargetsContractTest(DistributionContractTestCase):
    def assert_target(self, response, target: str) -> None:
        problem = assert_problem(self, response, 400, "validation_failed")
        self.assertEqual(problem.get("target"), target, problem)

    def create_response(self, group_id: str, distribution: dict):
        return self.gateway.post(
            "/groups/{groupId}/resources", groupId=group_id,
            json={"resource_type": "dataset", "metadata": {
                "title": f"error-target-{uuid4().hex}", "description": "Error target regression",
                "distributions": [distribution],
            }},
        )

    def test_parameter_schema_and_template_targets(self) -> None:
        """Regression: schema and template errors must use flat Gateway fields, not domain schema paths."""
        group_id = create_group(self.gateway, f"error-target-{uuid4().hex}")
        cases = (
            (parameter(pattern="["), "parameters[0].pattern"),
            (parameter("filter", "TEMPLATE", template="{{", parameters=[]), "parameters[0].template"),
            (parameter("filter", "TEMPLATE", template="{{.value}}", parameters=[
                child("bad-name"),
            ]), "parameters[0].parameters[0].name"),
        )
        for invalid, suffix in cases:
            with self.subTest(target=suffix):
                self.assert_target(
                    self.create_response(group_id, web_distribution([invalid])),
                    f"metadata.distributions[0].web_service.{suffix}",
                )

    def test_sorted_parameter_role_targets_original_index(self) -> None:
        """Regression: role validation must not report an index from the sorted parameter list."""
        group_id = create_group(self.gateway, f"error-target-{uuid4().hex}")
        distribution = web_distribution([
            parameter("invalid", position=10, role="START_DATE"),
            parameter("valid", position=0),
        ])
        self.assert_target(
            self.create_response(group_id, distribution),
            "metadata.distributions[0].web_service.parameters[0].kind",
        )

    def test_nested_parameter_identity_targets_original_request(self) -> None:
        """Regression: nested identity errors must omit template and retain original child index."""
        distribution = web_distribution([
            parameter("filter", "TEMPLATE", template="{{.first}} {{.second}}", parameters=[
                child("first", position=10), child("second", position=0),
            ]),
        ])
        detail = self.create_draft([distribution])
        original = detail["metadata"]["distributions"][0]
        distribution["resource_id"] = original["resource_id"]
        distribution["web_service"]["resource_id"] = original["web_service"]["resource_id"]
        distribution["web_service"]["parameters"][0]["resource_id"] = (
            original["web_service"]["parameters"][0]["resource_id"]
        )
        distribution["web_service"]["parameters"][0]["parameters"][0]["resource_id"] = str(uuid4())
        metadata = {
            "title": detail["metadata"]["title"], "description": "Invalid nested identity",
            "distributions": [distribution],
        }
        self.assert_target(
            self.replace_response(detail, metadata),
            "metadata.distributions[0].web_service.parameters[0].parameters[0].resource_id",
        )

    def test_wms_template_default_target(self) -> None:
        """Regression: missing WMS template defaults must not include a nonexistent schema field."""
        group_id = create_group(self.gateway, f"error-target-{uuid4().hex}")
        distribution = {
            "title": "WMS", "description": "WMS default target", "type": "WEB_SERVICE",
            "web_service": {
                "type": "WMS", "visible_on": ["MAP"],
                "capabilities_url": "https://provider.invalid/wms",
                "default_layer": "layer", "layers": [{"name": "layer", "default_style": "", "styles": [""]}],
                "transparency_editable": False, "default_transparency": False,
                "get_map_parameters": [parameter(
                    "filter", "TEMPLATE", required=True, template="{{.value}}",
                    parameters=[child(required=True)],
                )],
                "get_feature_info_parameters": [],
            },
        }
        self.assert_target(
            self.create_response(group_id, distribution),
            "metadata.distributions[0].web_service.get_map_parameters[0].parameters[0].default",
        )

    def test_access_template_targets_parameters(self) -> None:
        """Regression: query and body template errors must not invent nested access-request fields."""
        distributions = [
            web_distribution([parameter(
                "filter", "TEMPLATE", template="{{.left}} {{.right}}",
                parameters=[child("left"), child("right", position=1)],
            )]),
            web_distribution(
                method="POST", body_content_type="text/plain", body_required=False,
                body_template="{{.left}} {{.right}}",
                body_parameters=[child("left"), child("right", position=1)],
            ),
        ]
        distributions[1]["title"] = "body"
        detail = self.create_draft(distributions)
        for distribution in detail["metadata"]["distributions"]:
            with self.subTest(distribution=distribution["title"]):
                service = distribution["web_service"]
                children = (service["body_parameters"] if distribution["title"] == "body"
                            else service["parameters"][0]["parameters"])
                left = next(item for item in children if item["name"] == "left")
                self.assert_target(self.prepare_response(distribution, {left["id"]: "value"}), "parameters")

    def test_access_array_element_target(self) -> None:
        """Regression: invalid array values must identify the failing element, not only the array."""
        detail = self.create_draft([web_distribution([
            parameter("values", multiple_values=True, pattern="^ok$"),
        ])])
        distribution = detail["metadata"]["distributions"][0]
        identifier = distribution["web_service"]["parameters"][0]["id"]
        self.assert_target(
            self.prepare_response(distribution, {identifier: ["ok", "bad"]}),
            "parameters[0].value[1]",
        )

    def test_invalid_template_url_rejected_on_write(self) -> None:
        """Regression: invalid stored URLs must not fail later against an unrelated access request."""
        group_id = create_group(self.gateway, f"error-target-{uuid4().hex}")
        distribution = web_distribution(template_url="/relative")
        self.assert_target(
            self.create_response(group_id, distribution),
            "metadata.distributions[0].web_service.template_url",
        )

    def test_dynamic_coordinate_targets(self) -> None:
        """Regression: synthetic coordinate parameters must not leak into access-request targets."""
        distribution = web_distribution([
            parameter("lon", "FLOAT", role="LONGITUDE", required=True, minimum=-180, maximum=180),
            parameter("lat", "FLOAT", position=1, role="LATITUDE", required=True, minimum=-90, maximum=90),
        ], type="DYNAMIC")
        detail = self.create_draft([distribution])
        saved = detail["metadata"]["distributions"][0]
        for field, value in (("longitude", 181), ("latitude", -91)):
            with self.subTest(field=field):
                body = {"type": "DYNAMIC", "longitude": 0, "latitude": 0, "parameters": []}
                body[field] = value
                response = self.gateway.post(
                    "/distributions/{distributionId}/prepare", distributionId=saved["id"], json=body,
                )
                self.assert_target(response, field)
