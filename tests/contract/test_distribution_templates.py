from urllib.parse import parse_qs, urlsplit

from distribution_fixtures import DistributionContractTestCase, child, parameter, web_distribution


class DistributionTemplatesContractTest(DistributionContractTestCase):
    def test_template_activation_and_required_children(self) -> None:
        """Verify the same child/default/branch rules for parameter and body templates."""
        cases = [
            ("required", True, "{{.value}}", [child()], {"value": "supplied"}, "supplied"),
            ("skipped", True, "{{if .enabled}}{{.value}}{{else}}fallback{{end}}",
             [child("enabled", "BOOLEAN", default=False), child(position=1)], {}, "fallback"),
            ("evaluated-missing", True, "{{if .enabled}}{{.value}}{{end}}",
             [child("enabled", "BOOLEAN", default=True), child(position=1)], {}, 400),
            ("required-skipped", True, "{{if .enabled}}{{.value}}{{end}}",
             [child("enabled", "BOOLEAN", default=False), child(position=1, required=True)], {}, 400),
            ("optional-defaults", False, "{{.left}}/{{.right}}",
             [child("left", default="a"), child("right", position=1, default="b")], {}, "a/b"),
            ("optional-missing", False, "{{.left}}/{{.right}}",
             [child("left"), child("right", position=1)], {}, None),
            ("optional-partial", False, "{{.left}}/{{.right}}",
             [child("left"), child("right", position=1)], {"left": "a"}, 400),
            ("optional-full", False, "{{.left}}/{{.right}}",
             [child("left"), child("right", position=1)], {"left": "a", "right": "b"}, "a/b"),
            ("optional-literal", False, "literal", [], {}, "literal"),
            ("required-literal", True, "literal", [], {}, "literal"),
            ("empty-render", True, "{{if .enabled}}value{{end}}",
             [child("enabled", "BOOLEAN", default=False)], {}, ""),
        ]
        distributions = []
        for location in ("parameter", "body"):
            for name, required, source, children, _, _ in cases:
                if location == "parameter":
                    distribution = web_distribution([parameter(
                        "rendered", "TEMPLATE", required=required, template=source, parameters=children,
                    )])
                else:
                    distribution = web_distribution(
                        method="POST", body_content_type="text/plain", body_template=source,
                        body_required=required, body_parameters=children,
                    )
                distribution["title"] = f"{location}-{name}"
                distributions.append(distribution)
        detail = self.create_draft(distributions)
        by_title = {item["title"]: item for item in detail["metadata"]["distributions"]}
        for location in ("parameter", "body"):
            for name, _, _, _, supplied, expected in cases:
                with self.subTest(location=location, case=name):
                    distribution = by_title[f"{location}-{name}"]
                    service = distribution["web_service"]
                    children = (
                        service["parameters"][0]["parameters"] if location == "parameter"
                        else service.get("body_parameters", [])
                    )
                    identifiers = {item["name"]: item["id"] for item in children}
                    values = {identifiers[key]: value for key, value in supplied.items()}
                    if expected == 400:
                        self.assert_prepare_problem(
                            self.prepare_response(distribution, values),
                            target="parameters",
                        )
                        continue
                    prepared = self.prepare(distribution, values=values)
                    if location == "body":
                        self.assertEqual(prepared["body"], expected)
                        self.assertEqual(prepared["method"], "POST")
                    else:
                        query = parse_qs(urlsplit(prepared["url"]).query, keep_blank_values=True)
                        if expected is None:
                            self.assertNotIn("rendered", query)
                        else:
                            self.assertEqual(query["rendered"], [expected])

    def test_typed_children_and_literal_interpolation(self) -> None:
        """Verify typed conditions, datetime formatting and unescaped template text."""
        source = '{{if and .enabled (gt .count 2) (lt .ratio 2.0)}}{{.when}}|{{.text}}{{else}}skipped{{end}}'
        children = [
            child("enabled", "BOOLEAN"), child("count", "INTEGER", position=1),
            child("ratio", "FLOAT", position=2),
            child("when", "DATETIME", position=3, layout="YYYY/MM/DD hh:mm:ss"),
            child("text", position=4),
        ]
        query_distribution = web_distribution([parameter(
            "rendered", "TEMPLATE", required=True, template=source, parameters=children,
        )])
        body_distribution = web_distribution(
            method="PUT", body_content_type="application/json", body_template=source,
            body_parameters=children,
        )
        query_distribution["title"] = "query-template"
        body_distribution["title"] = "body-template"
        detail = self.create_draft([query_distribution, body_distribution])
        by_title = {item["title"]: item for item in detail["metadata"]["distributions"]}
        text = 'quote"\\backslash\nnext line é'
        for location in ("query", "body"):
            distribution = by_title[f"{location}-template"]
            service = distribution["web_service"]
            children = service["parameters"][0]["parameters"] if location == "query" else service["body_parameters"]
            identifiers = {item["name"]: item["id"] for item in children}
            for enabled in (True, False):
                with self.subTest(location=location, enabled=enabled):
                    values = {
                        "enabled": enabled, "count": 3, "ratio": 1.5,
                        "when": "2026-10-21T12:30:00+02:00", "text": text,
                    }
                    prepared = self.prepare(
                        distribution, values={identifiers[key]: value for key, value in values.items()},
                    )
                    expected = "2026/10/21 10:30:00|" + text if enabled else "skipped"
                    if location == "query":
                        self.assertEqual(parse_qs(urlsplit(prepared["url"]).query)["rendered"], [expected])
                    else:
                        self.assertEqual(prepared["body"], expected)
                        self.assertEqual(prepared["body"].encode("utf-8"), expected.encode("utf-8"))
                        self.assertEqual(prepared["method"], "PUT")
                        self.assertNotIn("Content-Type", prepared["headers"])
