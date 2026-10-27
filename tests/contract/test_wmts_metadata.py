from copy import deepcopy
from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, child, parameter
from fixtures import create_group


class WMTSMetadataContractTest(DistributionContractTestCase):
    def test_configuration_lifecycle(self) -> None:
        """Catch lost WMTS choices on replacement and copy, and reject invalid intent without provider calls."""
        configuration = {
            "type": "WMTS",
            "capabilities_url": "https://provider.invalid/wmts?SERVICE=WMTS&REQUEST=GetCapabilities&tenant=a%2Fb&tag=one&tag=two",
            "default_layer": "displacement",
            "layers": [
                {"name": "displacement", "default_style": "default", "styles": ["default", "alternate"]},
                {"name": "boundaries", "default_style": "outline", "styles": ["outline"]},
                {"name": "provider-default"},
                {"name": "approved-default", "styles": ["default", "alternate"]},
            ],
            "get_tile_parameters": [
                parameter("Time", default="tile-time", required=True),
                parameter("filter", "TEMPLATE", position=1, required=True,
                          template="{{.value}}", parameters=[child(default="tile-filter", required=True)]),
            ],
            "get_feature_info_parameters": [parameter("Time", default="info-time", required=True)],
        }
        distribution = {
            "title": "Manual WMTS", "description": "Author-supplied WMTS configuration",
            "type": "WEB_SERVICE",
            "web_service": {"name": "Manual WMTS", "visible_on": ["MAP"], "configuration": configuration},
        }
        detail = self.create_draft([distribution])
        saved_distribution = detail["metadata"]["distributions"][0]
        actual = saved_distribution["web_service"]["configuration"]
        fields = ("get_tile_parameters", "get_feature_info_parameters")
        for field in configuration.keys() - set(fields) - {"layers"}:
            self.assertEqual(actual[field], configuration[field], field)
        self.assertEqual(actual["layers"][:2], configuration["layers"][:2])
        self.assertEqual(actual["layers"][2]["name"], "provider-default")
        self.assertNotIn("default_style", actual["layers"][2])
        self.assertFalse(actual["layers"][2].get("styles"))
        self.assertNotIn("default_style", actual["layers"][3])
        self.assertEqual(actual["layers"][3]["styles"], ["default", "alternate"])
        self.assertEqual(set(actual), set(configuration))
        self.assertEqual(actual[fields[0]][0]["default"], "tile-time")
        self.assertEqual(actual[fields[1]][0]["default"], "info-time")
        self.assertNotEqual(actual[fields[0]][0]["id"], actual[fields[1]][0]["id"])
        self.assertIn("image/png", [item["output_format"] for item in saved_distribution["available_formats"]])

        for endpoint in ("prepare", "access"):
            response = self.gateway.post(
                "/distributions/{distributionId}/" + endpoint,
                distributionId=saved_distribution["id"], json={"type": "HTTP", "parameters": []},
            )
            self.assert_prepare_problem(response)

        invalid = [
            {key: value for key, value in configuration.items() if key != "capabilities_url"},
            {**configuration, "capabilities_url": ""},
            {**configuration, "capabilities_url": "/wmts"},
            {**configuration, "capabilities_url": "ftp://provider.invalid/wmts"},
            {**configuration, "capabilities_url": "https://provider.invalid/wmts#fragment"},
            {**configuration, "layers": []},
            {**configuration, "default_layer": "missing"},
            {**configuration, "layers": [configuration["layers"][0]] * 2},
        ]
        for overrides in (
            {"styles": []}, {"styles": ["default", "default"]},
            {"styles": [""]}, {"styles": ["   "]}, {"styles": [None]},
            {"default_style": "missing"}, {"default_style": ""},
        ):
            invalid.append({**configuration, "layers": [{**configuration["layers"][0], **overrides}]})
        for field in fields:
            for name in ("service", "VERSION", "Request", "layer", "STYLE", "format", "tilematrixset", "tilematrix", "tilerow", "tilecol"):
                invalid.append({**configuration, field: [parameter(name, default="invalid")]})
            invalid.extend([
                {**configuration, field: [parameter("custom", required=True)]},
                {**configuration, field: [parameter("custom", location="HEADER")]},
                {**configuration, field: [parameter("custom"), parameter("custom", position=1)]},
            ])
        for name in ("infoformat", "i", "J"):
            invalid.append({**configuration, fields[1]: [parameter(name, default="invalid")]})
        for candidate_configuration in invalid:
            candidate = deepcopy(distribution)
            candidate["web_service"]["configuration"] = candidate_configuration
            metadata = {"title": detail["metadata"]["title"], "description": "Invalid WMTS", "distributions": [candidate]}
            for method in ("POST", "PUT"):
                with self.subTest(configuration=candidate_configuration, method=method):
                    response = (self.gateway.post(
                        "/groups/{groupId}/resources", groupId=detail["group_id"],
                        json={"resource_type": "dataset", "metadata": metadata},
                    ) if method == "POST" else self.replace_response(detail, metadata))
                    self.assert_prepare_problem(response)
        unchanged = self.details(detail["resource_id"], detail["version_id"])
        self.assertEqual(unchanged["revision"], detail["revision"])
        self.assertEqual(unchanged["metadata"]["distributions"][0]["web_service"]["configuration"], actual)

        for action in ("submit", "publish"):
            self.transition(detail, action)
        group = create_group(self.gateway, f"wmts-copy-{uuid4().hex}")
        copied = self.gateway.post(
            "/groups/{groupId}/resources/{resourceType}/{id}/copy",
            groupId=group, resourceType="dataset", id=detail["resource_id"],
        )
        self.assertEqual(copied.status_code, 201)
        drafted = self.gateway.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset", id=detail["resource_id"],
        )
        self.assertEqual(drafted.status_code, 201)
        for resource_id in (copied.json()["id"], detail["resource_id"]):
            clone = self.details(resource_id, self.version(resource_id, "DRAFT")["version_id"])
            cloned = clone["metadata"]["distributions"][0]["web_service"]["configuration"]
            for field in configuration.keys() - set(fields):
                self.assertEqual(cloned[field], actual[field], field)
            for field in fields:
                for source, target in zip(actual[field], cloned[field], strict=True):
                    self.assertNotEqual(source["id"], target["id"])
                    self.assertEqual(source["name"], target["name"])
                    self.assertEqual(source.get("default"), target.get("default"))
                    if source["kind"] == "TEMPLATE":
                        self.assertEqual(source["template"], target["template"])
                        self.assertNotEqual(source["parameters"][0]["id"], target["parameters"][0]["id"])
                        self.assertEqual(target["parameters"][0]["default"], "tile-filter")
            if resource_id == detail["resource_id"]:
                detail = clone

        configuration.update({
            "capabilities_url": "https://provider.invalid/replaced?tenant=updated",
            "default_layer": "boundaries",
            "layers": [{"name": "boundaries", "default_style": "outline", "styles": ["outline"]}],
            "get_tile_parameters": [], "get_feature_info_parameters": [],
        })
        response = self.replace_response(detail, {
            "title": detail["metadata"]["title"], "description": "Replaced WMTS", "distributions": [distribution],
        })
        self.assertEqual(response.status_code, 200)
        updated = self.details(detail["resource_id"], detail["version_id"])
        self.assertEqual(updated["revision"], detail["revision"] + 1)
        self.assertEqual(updated["metadata"]["distributions"][0]["web_service"]["configuration"], configuration)
