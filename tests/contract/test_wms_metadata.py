from copy import deepcopy
from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, child, parameter
from fixtures import create_group


class WMSMetadataContractTest(DistributionContractTestCase):
    def test_manual_configuration_survives_create_and_replace(self) -> None:
        """Catch loss of WMS author choices on writes/copies; reject invalid intent without provider calls."""
        configuration = {
            "type": "WMS",
            "capabilities_url": "https://provider.invalid/wms?SERVICE=WMS&REQUEST=GetCapabilities&tenant=a%2Fb&tag=one&tag=two",
            "default_layer": "zoning",
            "layers": [
                {
                    "name": "zoning", "default_style": "highlight",
                    "styles": ["highlight", ""],
                },
                {
                    "name": "boundaries", "default_style": "", "styles": ["", "outline"],
                },
            ],
            "transparency_editable": True, "default_transparency": True,
            "get_map_parameters": [
                parameter("time", default="map-value", required=True),
                parameter("filter", "TEMPLATE", position=1, required=True,
                          template="{{.value}}", parameters=[child(default="map-filter", required=True)]),
            ],
            "get_feature_info_parameters": [parameter("time", default="inspect-value", required=True)],
        }
        distribution = {
            "title": "Manual WMS", "description": "Author-supplied WMS configuration",
            "type": "WEB_SERVICE",
            "web_service": {
                "name": "Manual WMS", "visible_on": ["MAP"], "configuration": configuration,
            },
        }
        detail = self.create_draft([distribution])
        saved = detail["metadata"]["distributions"][0]["web_service"]
        self.assertEqual(saved["visible_on"], ["MAP"])
        actual = saved["configuration"]
        parameter_fields = ("get_map_parameters", "get_feature_info_parameters")
        for field in configuration.keys() - set(parameter_fields):
            self.assertEqual(actual[field], configuration[field], field)
        self.assertEqual(set(actual), set(configuration))
        for field, default in (("get_map_parameters", "map-value"), ("get_feature_info_parameters", "inspect-value")):
            self.assertEqual(actual[field][0]["name"], "time")
            self.assertEqual(actual[field][0]["default"], default)
            self.assertTrue(actual[field][0]["id"])

        invalid_cases = [
            ("missing URL", {key: value for key, value in configuration.items() if key != "capabilities_url"}),
            ("empty URL", {**configuration, "capabilities_url": ""}),
            ("relative URL", {**configuration, "capabilities_url": "/wms"}),
            ("non-HTTP URL", {**configuration, "capabilities_url": "ftp://provider.invalid/wms"}),
            ("URL fragment", {**configuration, "capabilities_url": "https://provider.invalid/wms#fragment"}),
            ("empty layers", {**configuration, "layers": []}),
            ("unknown layer", {**configuration, "default_layer": "missing"}),
            ("duplicate layers", {**configuration, "layers": [configuration["layers"][0]] * 2}),
        ]
        for label, fields in (
            ("empty styles", {"styles": []}),
            ("duplicate styles", {"styles": ["", ""]}),
            ("null style", {"styles": [None]}),
            ("unknown style", {"default_style": "missing"}),
        ):
            invalid_cases.append((label, {
                **configuration, "layers": [{**configuration["layers"][0], **fields}],
            }))
        for field in parameter_fields:
            for name in ("service", "VERSION", "Request", "LAYERS", "styles", "crs", "BBOX", "width", "height", "FORMAT", "TRANSPARENT"):
                invalid_cases.append((f"{field} reserved {name}", {
                    **configuration, field: [parameter(name, default="invalid")],
                }))
            for label, parameters in (
                ("wrong location", [parameter("custom", location="HEADER")]),
                ("missing default", [parameter("custom", required=True)]),
                ("duplicate names", [parameter("custom"), parameter("custom", position=1)]),
            ):
                invalid_cases.append((f"{field} {label}", {**configuration, field: parameters}))
        invalid_cases.append(("reserved info parameter", {
            **configuration, "get_feature_info_parameters": [parameter("INFO_FORMAT", default="text/plain")],
        }))
        for label, invalid in invalid_cases:
            candidate = deepcopy(distribution)
            candidate["web_service"]["configuration"] = invalid
            metadata = {
                "title": detail["metadata"]["title"], "description": "Invalid author configuration",
                "distributions": [candidate],
            }
            for method in ("POST", "PUT"):
                with self.subTest(case=label, method=method):
                    if method == "POST":
                        response = self.gateway.post(
                            "/groups/{groupId}/resources", groupId=detail["group_id"],
                            json={"resource_type": "dataset", "metadata": metadata},
                        )
                    else:
                        response = self.replace_response(detail, metadata)
                    problem = self.assert_prepare_problem(response)
                    if label == "missing URL":
                        self.assertEqual(problem.get("target"), "capabilities_url")
                    elif label != "null style":
                        self.assertTrue(problem.get("target", "").startswith(
                            "metadata.distributions[0].web_service.configuration"
                        ), problem)
        unchanged = self.details(detail["resource_id"], detail["version_id"])
        self.assertEqual(unchanged["revision"], detail["revision"])
        self.assertEqual(unchanged["metadata"]["distributions"][0]["web_service"]["configuration"], actual)

        for action in ("submit", "publish"):
            self.transition(detail, action)
        target_group = create_group(self.gateway, f"wms-copy-{uuid4().hex}")
        copied = self.gateway.post(
            "/groups/{groupId}/resources/{resourceType}/{id}/copy",
            groupId=target_group, resourceType="dataset", id=detail["resource_id"],
        )
        self.assertEqual(copied.status_code, 201)
        drafted = self.gateway.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset", id=detail["resource_id"],
        )
        self.assertEqual(drafted.status_code, 201)
        for resource_id in (copied.json()["id"], detail["resource_id"]):
            version = self.version(resource_id, "DRAFT")
            clone = self.details(resource_id, version["version_id"])
            cloned = clone["metadata"]["distributions"][0]["web_service"]["configuration"]
            for field in configuration.keys() - set(parameter_fields):
                self.assertEqual(cloned[field], configuration[field], field)
            for field in parameter_fields:
                for source, target in zip(actual[field], cloned[field], strict=True):
                    self.assertNotEqual(source["id"], target["id"])
                    self.assertEqual(source["name"], target["name"])
                    self.assertEqual(source.get("default"), target.get("default"))
                    if source["kind"] == "TEMPLATE":
                        self.assertEqual(source["template"], target["template"])
                        self.assertNotEqual(source["parameters"][0]["id"], target["parameters"][0]["id"])
                        self.assertEqual(target["parameters"][0]["default"], "map-filter")
            if resource_id == detail["resource_id"]:
                detail = clone

        configuration["get_map_parameters"] = []
        configuration["get_feature_info_parameters"] = []
        configuration["capabilities_url"] = "https://provider.invalid/replaced?tenant=updated"
        configuration["default_layer"] = "boundaries"
        configuration["layers"] = [{"name": "boundaries", "default_style": "", "styles": [""]}]
        configuration["transparency_editable"] = False
        configuration["default_transparency"] = False
        response = self.replace_response(detail, {
            "title": detail["metadata"]["title"], "description": "Replaced manual WMS",
            "distributions": [distribution],
        })
        self.assertEqual(response.status_code, 200)
        updated = self.details(detail["resource_id"], detail["version_id"])
        self.assertEqual(updated["revision"], detail["revision"] + 1)
        actual = updated["metadata"]["distributions"][0]["web_service"]["configuration"]
        for field in configuration:
            self.assertEqual(actual[field], configuration[field], field)
