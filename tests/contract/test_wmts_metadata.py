from copy import deepcopy
from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, child, parameter
from fixtures import create_group


class WMTSMetadataContractTest(DistributionContractTestCase):
    def test_configuration_lifecycle(self) -> None:
        """Catch lost WMTS choices on replacement and copy, and reject invalid intent without provider calls."""
        configuration = {
            "type": "WMTS",
            "capabilities_url": "https://provider.invalid/wmts?SERVICE=WMTS&REQUEST=GetCapabilities&tenant=a%2Fb&tag=one&tag=two#fragment",
            "default_layer": "displacement",
            "layers": [
                {
                    "name": "displacement",
                    "default_style": "default",
                    "styles": ["default", "alternate"],
                },
                {
                    "name": "boundaries",
                    "default_style": "outline",
                    "styles": ["outline"],
                },
                {"name": "provider-default"},
                {"name": "approved-default", "styles": ["default", "alternate"]},
            ],
            "get_tile_parameters": [
                parameter("Time", default="tile-time", required=True),
                parameter(
                    "filter",
                    "TEMPLATE",
                    position=1,
                    required=True,
                    template="{{.value}}",
                    parameters=[child(default="tile-filter", required=True)],
                ),
            ],
            "get_feature_info_parameters": [
                parameter("Time", default="info-time", required=True)
            ],
        }
        configuration["visible_on"] = ["MAP"]
        distribution = {
            "title": "Manual WMTS",
            "description": "Author-supplied WMTS configuration",
            "type": "WEB_SERVICE",
            "web_service": configuration,
        }
        detail = self.create_draft([distribution])
        saved_distribution = detail["metadata"]["distributions"][0]
        actual = saved_distribution["web_service"]
        fields = ("get_tile_parameters", "get_feature_info_parameters")
        self.assertEqual(
            actual["capabilities_url"], configuration["capabilities_url"].split("#")[0]
        )
        for field in (
            configuration.keys() - set(fields) - {"layers", "capabilities_url"}
        ):
            self.assertEqual(actual[field], configuration[field], field)
        self.assertEqual(
            [
                {k: v for k, v in layer.items() if k not in ("id", "resource_id")}
                for layer in actual["layers"][:2]
            ],
            configuration["layers"][:2],
        )
        self.assertEqual(actual["layers"][2]["name"], "provider-default")
        self.assertNotIn("default_style", actual["layers"][2])
        self.assertFalse(actual["layers"][2].get("styles"))
        self.assertNotIn("default_style", actual["layers"][3])
        self.assertEqual(actual["layers"][3]["styles"], ["default", "alternate"])
        common_fields = {
            "id",
            "resource_id",
            "visible_on",
            "provider",
            "documentation_uri",
            "contact_points",
        }
        self.assertEqual(
            set(actual) - common_fields, set(configuration) - common_fields
        )
        self.assertEqual(actual[fields[0]][0]["default"], ["tile-time"])
        self.assertEqual(actual[fields[1]][0]["default"], ["info-time"])
        self.assertNotEqual(actual[fields[0]][0]["id"], actual[fields[1]][0]["id"])
        self.assertNotIn("available_formats", actual)

        for endpoint in ("prepare", "access"):
            response = self.gateway.post(
                "/distributions/{distributionId}/" + endpoint,
                distributionId=saved_distribution["id"],
                json={"type": "HTTP", "parameters": []},
            )
            self.assert_prepare_problem(response)

        invalid = [
            {
                key: value
                for key, value in configuration.items()
                if key != "capabilities_url"
            },
            {**configuration, "capabilities_url": ""},
            {**configuration, "capabilities_url": "/wmts"},
            {**configuration, "capabilities_url": "ftp://provider.invalid/wmts"},
            {**configuration, "layers": []},
            {**configuration, "default_layer": "missing"},
            {**configuration, "layers": [configuration["layers"][0]] * 2},
        ]
        for overrides in (
            {"styles": []},
            {"styles": ["default", "default"]},
            {"styles": [""]},
            {"styles": ["   "]},
            {"styles": [None]},
            {"default_style": "missing"},
            {"default_style": ""},
        ):
            invalid.append(
                {
                    **configuration,
                    "layers": [{**configuration["layers"][0], **overrides}],
                }
            )
        for field in fields:
            for name in (
                "service",
                "VERSION",
                "Request",
                "layer",
                "STYLE",
                "format",
                "tilematrixset",
                "tilematrix",
                "tilerow",
                "tilecol",
            ):
                invalid.append(
                    {**configuration, field: [parameter(name, default="invalid")]}
                )
            invalid.extend(
                [
                    {**configuration, field: [parameter("custom", required=True)]},
                    {**configuration, field: [parameter(
                        "custom", "TEMPLATE", template="{{eq .value 1}}", parameters=[child(default="text")],
                    )]},
                    {**configuration, field: [parameter("custom", location="HEADER")]},
                    {
                        **configuration,
                        field: [parameter("custom"), parameter("custom", position=1)],
                    },
                ]
            )
        for name in ("infoformat", "i", "J"):
            invalid.append(
                {**configuration, fields[1]: [parameter(name, default="invalid")]}
            )
        for candidate_configuration in invalid:
            candidate = deepcopy(distribution)
            candidate["web_service"] = candidate_configuration
            metadata = {
                "title": detail["metadata"]["title"],
                "description": "Invalid WMTS",
                "distributions": [candidate],
            }
            for method in ("POST", "PUT"):
                with self.subTest(configuration=candidate_configuration, method=method):
                    response = (
                        self.gateway.post(
                            "/groups/{groupId}/resources",
                            groupId=detail["group_id"],
                            json={"resource_type": "dataset", "metadata": metadata},
                        )
                        if method == "POST"
                        else self.replace_response(detail, metadata)
                    )
                    problem = self.assert_prepare_problem(response)
                    if candidate_configuration.get("capabilities_url") == "":
                        self.assertEqual(
                            problem.get("target"),
                            "metadata.distributions[0].web_service.capabilities_url",
                        )
        unchanged = self.details(detail["resource_id"], detail["id"])
        self.assertEqual(unchanged["revision"], detail["revision"])
        self.assertEqual(
            unchanged["metadata"]["distributions"][0]["web_service"], actual
        )

        for action in ("submit", "publish"):
            self.transition(detail, action)
        group = create_group(self.gateway, f"wmts-copy-{uuid4().hex}")
        copied = self.gateway.post(
            "/groups/{groupId}/resources/{resourceType}/{id}/copy",
            groupId=group,
            resourceType="dataset",
            id=detail["resource_id"],
        )
        self.assertEqual(copied.status_code, 201)
        drafted = self.gateway.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset",
            id=detail["resource_id"],
        )
        self.assertEqual(drafted.status_code, 201)
        for resource_id in (copied.json()["id"], detail["resource_id"]):
            clone = self.details(resource_id, self.version(resource_id, "DRAFT")["id"])
            cloned = clone["metadata"]["distributions"][0]["web_service"]
            for field in configuration.keys() - set(fields) - {"layers"}:
                self.assertEqual(cloned[field], actual[field], field)
            for source, target in zip(actual["layers"], cloned["layers"], strict=True):
                self.assertNotEqual(source["id"], target["id"])
                if resource_id == detail["resource_id"]:
                    self.assertEqual(source["resource_id"], target["resource_id"])
                else:
                    self.assertNotEqual(source["resource_id"], target["resource_id"])
            for field in fields:
                for source, target in zip(actual[field], cloned[field], strict=True):
                    self.assertNotEqual(source["id"], target["id"])
                    self.assertEqual(source["name"], target["name"])
                    self.assertEqual(source.get("default"), target.get("default"))
                    if source["kind"] == "TEMPLATE":
                        self.assertEqual(source["template"], target["template"])
                        self.assertNotEqual(
                            source["parameters"][0]["id"], target["parameters"][0]["id"]
                        )
                        self.assertEqual(
                            target["parameters"][0]["default"], ["tile-filter"]
                        )
            if resource_id == detail["resource_id"]:
                detail = clone

        previous_distribution = detail["metadata"]["distributions"][0]
        previous_service = previous_distribution["web_service"]
        previous_layer = previous_service["layers"][1]
        distribution["resource_id"] = previous_distribution["resource_id"]
        distribution["web_service"]["resource_id"] = previous_service["resource_id"]
        configuration.update(
            {
                "capabilities_url": "https://provider.invalid/replaced?tenant=updated#fragment",
                "default_layer": "boundaries-renamed",
                "layers": [
                    {
                        "resource_id": previous_layer["resource_id"],
                        "name": "boundaries-renamed",
                        "default_style": "outline",
                        "styles": ["outline"],
                    }
                ],
                "get_tile_parameters": [],
                "get_feature_info_parameters": [],
            }
        )
        response = self.replace_response(
            detail,
            {
                "title": detail["metadata"]["title"],
                "description": "Replaced WMTS",
                "distributions": [distribution],
            },
        )
        self.assertEqual(response.status_code, 200)
        updated = self.details(detail["resource_id"], detail["id"])
        self.assertEqual(updated["revision"], detail["revision"] + 1)
        updated_configuration = updated["metadata"]["distributions"][0]["web_service"]
        self.assertEqual(
            updated_configuration["capabilities_url"],
            configuration["capabilities_url"].split("#")[0],
        )
        for field in configuration.keys() - {"layers", "capabilities_url"}:
            self.assertEqual(updated_configuration[field], configuration[field], field)
        self.assertEqual(updated_configuration["layers"][0]["id"], previous_layer["id"])
        self.assertEqual(
            updated_configuration["layers"][0]["resource_id"],
            previous_layer["resource_id"],
        )
        self.assertEqual(
            updated_configuration["layers"][0]["name"], "boundaries-renamed"
        )
