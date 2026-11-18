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
        configuration["visible_on"] = ["MAP"]
        distribution = {
            "title": "Manual WMS", "description": "Author-supplied WMS configuration",
            "type": "WEB_SERVICE",
            "web_service": configuration,
        }
        detail = self.create_draft([distribution])
        saved = detail["metadata"]["distributions"][0]["web_service"]
        self.assertEqual(saved["visible_on"], ["MAP"])
        actual = saved
        self.assertNotIn("available_formats", saved)
        parameter_fields = ("get_map_parameters", "get_feature_info_parameters")
        for field in configuration.keys() - set(parameter_fields) - {"layers"}:
            self.assertEqual(actual[field], configuration[field], field)
        self.assertEqual(
            [{k: v for k, v in layer.items() if k not in ("id", "resource_id")} for layer in actual["layers"]],
            configuration["layers"],
        )
        common_fields = {"id", "resource_id", "visible_on", "provider", "documentation_uri", "contact_points"}
        self.assertEqual(set(actual) - common_fields, set(configuration) - common_fields)
        for field, default in (("get_map_parameters", "map-value"), ("get_feature_info_parameters", "inspect-value")):
            self.assertEqual(actual[field][0]["name"], "time")
            self.assertEqual(actual[field][0]["default"], [default])
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
            candidate["web_service"] = invalid
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
                            "metadata.distributions[0].web_service."
                        ), problem)
        unchanged = self.details(detail["resource_id"], detail["id"])
        self.assertEqual(unchanged["revision"], detail["revision"])
        self.assertEqual(unchanged["metadata"]["distributions"][0]["web_service"], actual)

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
            clone = self.details(resource_id, version["id"])
            cloned = clone["metadata"]["distributions"][0]["web_service"]
            for field in configuration.keys() - set(parameter_fields) - {"layers"}:
                self.assertEqual(cloned[field], configuration[field], field)
            for source, target in zip(actual["layers"], cloned["layers"], strict=True):
                self.assertNotEqual(source["id"], target["id"])
                if resource_id == detail["resource_id"]:
                    self.assertEqual(source["resource_id"], target["resource_id"])
                else:
                    self.assertNotEqual(source["resource_id"], target["resource_id"])
            for field in parameter_fields:
                for source, target in zip(actual[field], cloned[field], strict=True):
                    self.assertNotEqual(source["id"], target["id"])
                    if resource_id == detail["resource_id"]:
                        self.assertEqual(source["resource_id"], target["resource_id"])
                    else:
                        self.assertNotEqual(source["resource_id"], target["resource_id"])
                    self.assertEqual(source["name"], target["name"])
                    self.assertEqual(source.get("default"), target.get("default"))
                    if source["kind"] == "TEMPLATE":
                        self.assertEqual(source["template"], target["template"])
                        self.assertNotEqual(source["parameters"][0]["id"], target["parameters"][0]["id"])
                        self.assertEqual(target["parameters"][0]["default"], ["map-filter"])
            if resource_id == detail["resource_id"]:
                detail = clone

        current_distribution = detail["metadata"]["distributions"][0]
        current_service = current_distribution["web_service"]
        current_configuration = current_service
        retained = deepcopy(distribution)
        retained["resource_id"] = current_distribution["resource_id"]
        retained["web_service"]["resource_id"] = current_service["resource_id"]
        retained_configuration = retained["web_service"]
        for target, source in zip(retained_configuration["layers"], current_configuration["layers"], strict=True):
            target["resource_id"] = source["resource_id"]
        for field in parameter_fields:
            for target, source in zip(retained_configuration[field], current_configuration[field], strict=True):
                target["resource_id"] = source["resource_id"]
                if target["kind"] == "TEMPLATE":
                    for child_input, child_saved in zip(target["parameters"], source["parameters"], strict=True):
                        child_input["resource_id"] = child_saved["resource_id"]

        moved = deepcopy(retained)
        moved_configuration = moved["web_service"]
        moved_parameter = moved_configuration["get_map_parameters"].pop(0)
        moved_parameter["name"] = "moved-time"
        moved_configuration["get_feature_info_parameters"].append(moved_parameter)
        invalid_move = self.replace_response(detail, {
            "title": detail["metadata"]["title"], "description": "Invalid parameter move",
            "distributions": [moved],
        })
        problem = self.assert_prepare_problem(invalid_move)
        self.assertTrue(problem.get("target", "").endswith("resource_id"), problem)

        retained_configuration["get_map_parameters"][0]["name"] = "renamed-time"
        retained_response = self.replace_response(detail, {
            "title": detail["metadata"]["title"], "description": "Retained parameter identities",
            "distributions": [retained],
        })
        self.assertEqual(retained_response.status_code, 200)
        detail = self.details(detail["resource_id"], detail["id"])
        retained_result = detail["metadata"]["distributions"][0]["web_service"]
        for field in parameter_fields:
            for source, target in zip(current_configuration[field], retained_result[field], strict=True):
                self.assertEqual(source["id"], target["id"])
                self.assertEqual(source["resource_id"], target["resource_id"])
                if source["kind"] == "TEMPLATE":
                    self.assertEqual(source["parameters"][0]["id"], target["parameters"][0]["id"])
                    self.assertEqual(source["parameters"][0]["resource_id"], target["parameters"][0]["resource_id"])

        configuration["get_map_parameters"] = []
        configuration["get_feature_info_parameters"] = []
        configuration["capabilities_url"] = "https://provider.invalid/replaced?tenant=updated"
        previous_distribution = detail["metadata"]["distributions"][0]
        previous_service = previous_distribution["web_service"]
        previous_layer = previous_service["layers"][1]
        distribution["resource_id"] = previous_distribution["resource_id"]
        distribution["web_service"]["resource_id"] = previous_service["resource_id"]
        configuration["default_layer"] = "boundaries-renamed"
        configuration["layers"] = [{
            "resource_id": previous_layer["resource_id"],
            "name": "boundaries-renamed", "default_style": "", "styles": [""],
        }]
        configuration["transparency_editable"] = False
        configuration["default_transparency"] = False
        response = self.replace_response(detail, {
            "title": detail["metadata"]["title"], "description": "Replaced manual WMS",
            "distributions": [distribution],
        })
        self.assertEqual(response.status_code, 200)
        updated = self.details(detail["resource_id"], detail["id"])
        self.assertEqual(updated["revision"], detail["revision"] + 1)
        actual = updated["metadata"]["distributions"][0]["web_service"]
        for field in configuration.keys() - {"layers"}:
            self.assertEqual(actual[field], configuration[field], field)
        self.assertEqual(actual["layers"][0]["id"], previous_layer["id"])
        self.assertEqual(actual["layers"][0]["resource_id"], previous_layer["resource_id"])
        self.assertEqual(actual["layers"][0]["name"], "boundaries-renamed")
