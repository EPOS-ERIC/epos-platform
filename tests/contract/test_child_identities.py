from copy import deepcopy
from typing import Any
from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, child, parameter, web_distribution
from fixtures import create_group


class ChildIdentityContractTest(DistributionContractTestCase):
    def test_identifier_identity_on_replacement(self) -> None:
        """Retain identified identifiers and replace unidentified siblings across resource types."""
        for resource_type, name_key, identifier_key in (
            ("dataset", "title", "persistent_identifiers"),
            ("software-application", "name", "identifiers"),
            ("software-source-code", "name", "identifiers"),
        ):
            with self.subTest(resource_type=resource_type):
                group = create_group(self.gateway, f"identities-{uuid4().hex}")
                metadata = {
                    name_key: f"identities-{uuid4().hex}", "description": "Identifier identities",
                    identifier_key: [
                        {"type": "test", "value": "retained"},
                        {"type": "test", "value": "replaced"},
                        {"type": "test", "value": "removed"},
                    ],
                }
                created = self.gateway.post(
                    "/groups/{groupId}/resources", groupId=group,
                    json={"resource_type": resource_type, "metadata": metadata},
                )
                self.assertEqual(created.status_code, 201)
                resource_id = created.json()["id"]
                versions = self.gateway.get(
                    "/resources/{resourceType}/{id}/versions", resourceType=resource_type, id=resource_id,
                )
                self.assertEqual(versions.status_code, 200)
                draft = next(item for item in versions.json() if item["status"] == "DRAFT")

                def read_detail():
                    response = self.gateway.get(
                        "/resources/{resourceType}/{id}/versions/{versionId}",
                        resourceType=resource_type, id=resource_id, versionId=draft["id"],
                    )
                    self.assertEqual(response.status_code, 200)
                    return response.json()

                before = read_detail()
                originals = {item["value"]: item for item in before["metadata"][identifier_key]}
                metadata[identifier_key] = [
                    {"resource_id": originals["retained"]["resource_id"], "type": "test", "value": "retained-edited"},
                    {"type": "test", "value": "replaced"},
                ]
                response = self.gateway.put(
                    "/resources/{id}/versions/{versionId}/draft", id=resource_id, versionId=draft["id"],
                    json={"resource_type": resource_type, "revision": before["revision"], "metadata": metadata},
                )
                self.assertEqual(response.status_code, 200)
                result = {item["value"]: item for item in read_detail()["metadata"][identifier_key]}
                self.assertEqual(set(result), {"retained-edited", "replaced"})
                self.assertEqual(result["retained-edited"]["resource_id"], originals["retained"]["resource_id"])
                self.assertNotIn(result["replaced"]["resource_id"], {item["resource_id"] for item in originals.values()})

    def test_distribution_tree_identity_on_replacement(self) -> None:
        """Retain or replace each distribution-tree child according to its resource_id."""
        for replaced in ("distribution", "service", "parameter", "nested", "body"):
            with self.subTest(replaced=replaced):
                distribution = web_distribution(
                    [parameter("filter", "TEMPLATE", template="{{.value}}", parameters=[child()])],
                    method="POST", body_content_type="text/plain", body_template="{{.body}}",
                    body_parameters=[child("body")],
                )
                detail = self.create_draft([distribution])
                original = detail["metadata"]["distributions"][0]
                service = original["web_service"]
                config = service
                saved = {
                    "distribution": original, "service": service,
                    "parameter": config["parameters"][0],
                    "nested": config["parameters"][0]["parameters"][0],
                    "body": config["body_parameters"][0],
                }
                request_service = distribution["web_service"]
                request_config = request_service
                inputs = {
                    "distribution": distribution, "service": request_service,
                    "parameter": request_config["parameters"][0],
                    "nested": request_config["parameters"][0]["parameters"][0],
                    "body": request_config["body_parameters"][0],
                }
                # A new parent cannot retain children belonging to the old parent.
                descendants = {
                    "distribution": {"service", "parameter", "nested", "body"},
                    "service": {"parameter", "nested", "body"},
                    "parameter": {"nested"}, "nested": set(), "body": set(),
                }
                new_children = {replaced} | descendants[replaced]
                for name, item in inputs.items():
                    if name not in new_children:
                        item["resource_id"] = saved[name]["resource_id"]
                metadata = {
                    "title": detail["metadata"]["title"], "description": "Identity replacement",
                    "distributions": [distribution],
                }
                self.assertEqual(self.replace_response(detail, metadata).status_code, 200)
                updated = self.details(detail["resource_id"], detail["id"])["metadata"]["distributions"][0]
                updated_service = updated["web_service"]
                updated_config = updated_service
                result = {
                    "distribution": updated, "service": updated_service,
                    "parameter": updated_config["parameters"][0],
                    "nested": updated_config["parameters"][0]["parameters"][0],
                    "body": updated_config["body_parameters"][0],
                }
                for name in saved:
                    with self.subTest(child=name):
                        if name in new_children:
                            self.assertNotEqual(result[name]["resource_id"], saved[name]["resource_id"])
                        else:
                            self.assertEqual(result[name]["resource_id"], saved[name]["resource_id"])

    def test_mixed_parameter_siblings_on_replacement(self) -> None:
        """Retain, replace and remove parameter siblings in the same PUT."""
        distribution = web_distribution([
            parameter("retained"), parameter("replaced", position=1), parameter("removed", position=2),
        ])
        detail = self.create_draft([distribution])
        original = detail["metadata"]["distributions"][0]
        service = original["web_service"]
        before = {item["name"]: item for item in service["parameters"]}
        distribution["resource_id"] = original["resource_id"]
        distribution["web_service"]["resource_id"] = service["resource_id"]
        distribution["web_service"]["parameters"] = [
            parameter("retained-edited", resource_id=before["retained"]["resource_id"]),
            parameter("replaced", position=1),
        ]
        metadata = {
            "title": detail["metadata"]["title"], "description": "Mixed parameter identities",
            "distributions": [distribution],
        }
        self.assertEqual(self.replace_response(detail, metadata).status_code, 200)
        updated = self.details(detail["resource_id"], detail["id"])
        parameters = updated["metadata"]["distributions"][0]["web_service"]["parameters"]
        result = {item["name"]: item for item in parameters}
        self.assertEqual(set(result), {"retained-edited", "replaced"})
        self.assertEqual(result["retained-edited"]["resource_id"], before["retained"]["resource_id"])
        self.assertNotIn(result["replaced"]["resource_id"], {item["resource_id"] for item in before.values()})

    def test_map_layer_identity_on_replacement(self) -> None:
        """Retain, replace and remove WMS and WMTS layer identities independently."""
        for kind in ("WMS", "WMTS"):
            with self.subTest(kind=kind):
                layers: list[dict[str, Any]] = [{"name": name} for name in ("retained", "replaced", "removed")]
                if kind == "WMS":
                    for layer in layers:
                        layer.update({"default_style": "", "styles": [""]})
                configuration = {
                    "type": kind, "capabilities_url": "https://provider.invalid/capabilities",
                    "default_layer": "retained", "layers": layers, "get_feature_info_parameters": [],
                }
                if kind == "WMS":
                    configuration.update({"transparency_editable": False, "default_transparency": False, "get_map_parameters": []})
                else:
                    configuration["get_tile_parameters"] = []
                configuration["visible_on"] = ["MAP"]
                distribution = {
                    "title": kind, "description": "Map service", "type": "WEB_SERVICE",
                    "web_service": configuration,
                }
                detail = self.create_draft([distribution])
                original = detail["metadata"]["distributions"][0]
                service = original["web_service"]
                before = {item["name"]: item for item in service["layers"]}
                distribution["resource_id"] = original["resource_id"]
                distribution["web_service"]["resource_id"] = service["resource_id"]
                retained = deepcopy(layers[0])
                retained.update({"resource_id": before["retained"]["resource_id"], "name": "retained-edited"})
                configuration["default_layer"] = "retained-edited"
                configuration["layers"] = [retained, layers[1]]
                metadata = {
                    "title": detail["metadata"]["title"], "description": "Layer identities",
                    "distributions": [distribution],
                }
                self.assertEqual(self.replace_response(detail, metadata).status_code, 200)
                updated = self.details(detail["resource_id"], detail["id"])
                layers_after = updated["metadata"]["distributions"][0]["web_service"]["layers"]
                result = {item["name"]: item for item in layers_after}
                self.assertEqual(set(result), {"retained-edited", "replaced"})
                self.assertEqual(result["retained-edited"]["resource_id"], before["retained"]["resource_id"])
                self.assertNotIn(result["replaced"]["resource_id"], {item["resource_id"] for item in before.values()})
