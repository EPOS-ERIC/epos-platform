import copy
import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, create_group


class DistributionFormatsContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def format_parameter(self, **overrides: Any) -> dict[str, Any]:
        return {
            "kind": "STRING",
            "position": 0,
            "name": "format",
            "label": "Format",
            "required": False,
            "read_only": True,
            "location": "QUERY",
            "explode": False,
            "multiple_values": False,
            "property": "schema:encodingFormat",
            "role": "FORMAT",
            "default": "xml_dc",
            **overrides,
        }

    def distribution(
        self, title: str, format_value: str, parameters: list[dict[str, Any]]
    ) -> dict[str, Any]:
        return {
            "title": title,
            "description": "Contract format discovery",
            "type": "WEB_SERVICE",
            "web_service": {
                "name": title,
                "configuration": {
                    "template_url": "https://example.org/formats",
                    "method": "GET",
                    "type": "HTTP",
                    "format": format_value,
                    "parameters": parameters,
                },
            },
        }

    def metadata(self, distributions: list[dict[str, Any]]) -> dict[str, Any]:
        return {
            "title": f"formats-{uuid4().hex}",
            "description": "Contract format discovery dataset",
            "distributions": distributions,
        }

    def version(self, resource_id: str, status: str) -> dict[str, Any]:
        response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(response.status_code, 200)
        matches = [item for item in response.json() if item["status"] == status]
        self.assertEqual(len(matches), 1)
        return matches[0]

    def details(self, resource_id: str, version_id: str) -> dict[str, Any]:
        response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(response.status_code, 200)
        return response.json()

    def create_draft(
        self, group_id: str, metadata: dict[str, Any]
    ) -> tuple[str, dict[str, Any]]:
        response = self.gateway.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={"resource_type": "dataset", "metadata": metadata},
        )
        self.assertEqual(response.status_code, 201)
        resource_id = response.json()["id"]
        version = self.version(resource_id, "DRAFT")
        return resource_id, self.details(resource_id, version["id"])

    def assert_formats(
        self, distribution: dict[str, Any], expected: list[tuple[str, str]]
    ) -> None:
        self.assertNotIn("available_formats", distribution["web_service"])
        self.assertCountEqual(
            distribution["available_formats"],
            [
                {"kind": "ORIGINAL", "output_format": value, "label": label}
                for value, label in expected
            ],
        )

    def replace_draft(
        self, detail: dict[str, Any], metadata: dict[str, Any]
    ) -> dict[str, Any]:
        response = self.gateway.put(
            "/resources/{id}/versions/{versionId}/draft",
            id=detail["resource_id"],
            versionId=detail["id"],
            json={
                "resource_type": "dataset", "revision": detail["revision"],
                "metadata": metadata,
            },
        )
        self.assertEqual(response.status_code, 200)
        return self.details(detail["resource_id"], detail["id"])

    def test_native_format_discovery(self) -> None:
        """Verify explicit format roles, labels, generic properties, and file exclusion."""
        group_id = create_group(self.gateway, f"formats-{uuid4().hex}")
        fixed = self.distribution(
            "fixed", "unused-format", [self.format_parameter(location="PATH", required=True)]
        )
        fixed["web_service"]["configuration"]["template_url"] += "/{format}"
        distributions = [
            self.distribution("fallback", "Application/JSON", []),
            fixed,
            self.distribution(
                "enum",
                "unused-format",
                [self.format_parameter(enum=[
                    {"value": "GeoJSON", "label": "GeoJSON document"},
                    {"value": "xml_dc"},
                ])],
            ),
            self.distribution(
                "header", "unused-format",
                [self.format_parameter(location="HEADER", default="header-format")],
            ),
            self.distribution(
                "other-property", "fallback-format",
                [self.format_parameter(property="schema:encodingFormat", role=None)],
            ),
            {
                "title": "file", "description": "Contract downloadable file",
                "type": "DOWNLOADABLE_FILE",
                "downloadable_file": {
                    "url": "https://example.org/formats.csv", "format": "text/csv",
                },
            },
        ]
        _, detail = self.create_draft(group_id, self.metadata(distributions))
        by_title = {item["title"]: item for item in detail["metadata"]["distributions"]}
        self.assertEqual(set(by_title), {item["title"] for item in distributions})
        expected = {
            "fallback": [("Application/JSON", "Application/JSON")],
            "fixed": [("xml_dc", "xml_dc")],
            "enum": [("GeoJSON", "GeoJSON document"), ("xml_dc", "xml_dc")],
            "header": [("header-format", "header-format")],
            "other-property": [("fallback-format", "fallback-format")],
        }
        for title, formats in expected.items():
            with self.subTest(distribution=title):
                self.assert_formats(by_title[title], formats)
        self.assertNotIn("available_formats", by_title["file"])
        self.assertEqual(by_title["file"]["downloadable_file"]["format"], "text/csv")

    def test_formats_follow_edits_and_selected_version(self) -> None:
        """Verify formats recompute after edits without leaking across versions."""
        group_id = create_group(self.gateway, f"format-versions-{uuid4().hex}")
        metadata = self.metadata([self.distribution("versioned", "format-A", [])])
        resource_id, draft = self.create_draft(group_id, metadata)
        self.assert_formats(draft["metadata"]["distributions"][0], [("format-A", "format-A")])

        metadata["distributions"][0]["web_service"]["configuration"]["parameters"] = [
            self.format_parameter(default="format-B")
        ]
        draft = self.replace_draft(draft, metadata)
        self.assert_formats(draft["metadata"]["distributions"][0], [("format-B", "format-B")])
        published_version_id = draft["id"]
        for action in ("submit", "publish"):
            response = self.gateway.post(
                "/resources/{resourceType}/{id}/versions/{versionId}/" + action,
                resourceType="dataset", id=resource_id, versionId=published_version_id,
            )
            self.assertEqual(response.status_code, 200)

        response = self.gateway.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset", id=resource_id,
        )
        self.assertEqual(response.status_code, 201)
        version = self.version(resource_id, "DRAFT")
        self.assertNotEqual(version["id"], published_version_id)
        draft = self.details(resource_id, version["id"])
        copied_parameter = draft["metadata"]["distributions"][0]["web_service"]["configuration"]["parameters"][0]
        self.assertEqual(copied_parameter["role"], "FORMAT")
        self.assertEqual(copied_parameter["property"], "schema:encodingFormat")
        metadata["distributions"][0]["web_service"]["configuration"]["parameters"] = [
            self.format_parameter(
                default="format-C",
                enum=[{"value": "format-C", "label": "New format"}],
            )
        ]
        draft = self.replace_draft(draft, metadata)
        self.assert_formats(draft["metadata"]["distributions"][0], [("format-C", "New format")])
        published = self.details(resource_id, published_version_id)
        self.assert_formats(published["metadata"]["distributions"][0], [("format-B", "format-B")])

    def test_invalid_format_metadata_is_rejected(self) -> None:
        """Regression: reject empty/invalid format enums without mutating drafts."""
        group_id = create_group(self.gateway, f"format-validation-{uuid4().hex}")
        metadata = self.metadata([
            self.distribution("valid", "unused-format", [self.format_parameter()])
        ])
        resource_id, draft = self.create_draft(group_id, metadata)
        boolean = self.format_parameter(kind="BOOLEAN", default=True)
        boolean.pop("multiple_values")
        template = self.format_parameter(kind="TEMPLATE", template="fixed", parameters=[])
        template.pop("multiple_values")
        template.pop("default")
        missing_default = self.format_parameter()
        missing_default.pop("default")
        cases = [
            ("multiple parameters", [
                self.format_parameter(),
                self.format_parameter(name="other_format", position=1, location="HEADER"),
            ]),
            ("wrong value type", [boolean]),
            ("template", [template]),
            ("multiple values", [self.format_parameter(
                multiple_values=True, enum=[{"value": "xml_dc"}],
            )]),
            ("not read only", [self.format_parameter(read_only=False)]),
            ("missing default", [missing_default]),
            ("empty enum", [self.format_parameter(enum=[])]),
            ("default outside enum", [self.format_parameter(enum=[{"value": "GeoJSON"}])]),
            ("default violates pattern", [self.format_parameter(pattern="^GeoJSON$")]),
        ]
        for name, parameters in cases:
            invalid = copy.deepcopy(metadata)
            invalid["distributions"][0]["web_service"]["configuration"]["parameters"] = parameters
            for method in ("POST", "PUT"):
                with self.subTest(case=name, method=method):
                    if method == "POST":
                        response = self.gateway.post(
                            "/groups/{groupId}/resources",
                            groupId=group_id,
                            json={"resource_type": "dataset", "metadata": invalid},
                        )
                    else:
                        response = self.gateway.put(
                            "/resources/{id}/versions/{versionId}/draft",
                            id=resource_id, versionId=draft["id"],
                            json={
                                "resource_type": "dataset", "revision": draft["revision"],
                                "metadata": invalid,
                            },
                        )
                    problem = assert_problem(self, response, 400, "validation_failed")
                    self.assertTrue(
                        problem.get("target", "").startswith(
                            "metadata.distributions[0].web_service.configuration.parameters["
                        ),
                        problem,
                    )
                    if method == "PUT":
                        unchanged = self.details(resource_id, draft["id"])
                        self.assertEqual(unchanged["revision"], draft["revision"])
                        self.assert_formats(
                            unchanged["metadata"]["distributions"][0], [("xml_dc", "xml_dc")]
                        )
