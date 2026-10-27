from typing import Any

from distribution_fixtures import DistributionContractTestCase, parameter, web_distribution


class DistributionMetadataContractTest(DistributionContractTestCase):
    def test_body_required_defaults_and_valid_replacement(self) -> None:
        """Verify body-required defaults, omission and explicit false survive reads and edits."""
        distributions = []
        cases: list[tuple[str, dict[str, Any]]] = [
            ("default", {"body_content_type": "text/plain", "body_template": "default"}),
            ("optional", {"body_content_type": "text/plain", "body_template": "optional", "body_required": False}),
            ("absent", {}),
        ]
        for name, fields in cases:
            distribution = web_distribution(method="POST", **fields)
            distribution["title"] = name
            distributions.append(distribution)
        detail = self.create_draft(distributions)
        by_title = {item["title"]: item for item in detail["metadata"]["distributions"]}
        self.assertIs(by_title["default"]["web_service"]["configuration"]["body_required"], True)
        self.assertIs(by_title["optional"]["web_service"]["configuration"]["body_required"], False)
        self.assertNotIn("body_required", by_title["absent"]["web_service"]["configuration"])
        self.assertEqual(self.prepare(by_title["default"])["body"], "default")
        self.assertEqual(self.prepare(by_title["optional"])["body"], "optional")
        absent = self.prepare(by_title["absent"])
        self.assertIsNone(absent["body"])
        self.assertEqual(absent["headers"], {})

        metadata = {
            "title": detail["metadata"]["title"], "description": "Valid body replacement",
            "distributions": [web_distribution(
                method="PUT", template_url="https://provider.invalid/replaced",
                body_content_type="text/plain", body_template="replacement", body_required=False,
            )],
        }
        self.assertEqual(self.replace_response(detail, metadata).status_code, 200)
        updated = self.details(detail["resource_id"], detail["version_id"])
        self.assertEqual(updated["revision"], detail["revision"] + 1)
        distribution = updated["metadata"]["distributions"][0]
        self.assertIs(distribution["web_service"]["configuration"]["body_required"], False)
        prepared = self.prepare(distribution)
        self.assertEqual(prepared["method"], "PUT")
        self.assertEqual(prepared["url"], "https://provider.invalid/replaced")
        self.assertEqual(prepared["body"], "replacement")

    def test_invalid_body_and_header_metadata_does_not_mutate_draft(self) -> None:
        """Verify POST/PUT reject unsafe metadata and failed edits preserve revision and request."""
        valid = web_distribution([parameter(default="original")])
        detail = self.create_draft([valid])
        original = detail["metadata"]["distributions"][0]
        expected = self.prepare(original)
        cases = []
        for required in (True, False):
            cases.append((f"body-required-without-body-{required}", web_distribution(body_required=required)))
        for method in ("GET", "DELETE"):
            cases.append((f"body-on-{method}", web_distribution(
                method=method, body_content_type="text/plain", body_template="body",
            )))
        cases.extend([
            ("optional-path", web_distribution(
                [parameter("path", location="PATH", required=False)],
                template_url="https://provider.invalid/{path}",
            )),
            ("invalid-header-name", web_distribution([parameter("bad header", location="HEADER")])),
            ("duplicate-header-names", web_distribution([
                parameter("x-custom", location="HEADER"),
                parameter("X-Custom", position=1, location="HEADER"),
            ])),
        ])
        for name in (
            "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer",
            "TE", "Upgrade", "Proxy-Connection", "Keep-Alive",
        ):
            for spelling in (name, name.lower(), name.upper()):
                cases.append((f"forbidden-{spelling}", web_distribution([
                    parameter(spelling, location="HEADER"),
                ])))
        for name, distribution in cases:
            metadata = {
                "title": detail["metadata"]["title"], "description": "Rejected metadata",
                "distributions": [distribution],
            }
            for method in ("POST", "PUT"):
                with self.subTest(case=name, method=method):
                    if method == "POST":
                        response = self.gateway.post(
                            "/groups/{groupId}/resources", groupId=detail["group_id"],
                            json={"resource_type": "dataset", "metadata": metadata},
                        )
                    else:
                        response = self.replace_response(detail, metadata)
                    self.assert_prepare_problem(response, target="metadata.distributions[0].web_service")
                    if method == "PUT":
                        unchanged = self.details(detail["resource_id"], detail["version_id"])
                        self.assertEqual(unchanged["revision"], detail["revision"])
                        self.assertEqual(unchanged["metadata"]["distributions"][0]["id"], original["id"])
                        self.assertEqual(self.prepare(original), expected)
