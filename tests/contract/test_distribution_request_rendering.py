from distribution_fixtures import DistributionContractTestCase, child, parameter, web_distribution


class DistributionRequestRenderingContractTest(DistributionContractTestCase):
    def test_configured_methods_and_absent_body(self) -> None:
        """Verify preparation preserves each configured method without inventing headers or body."""
        distributions = []
        for method in ("GET", "POST", "PUT", "DELETE"):
            distribution = web_distribution(method=method)
            distribution["title"] = method
            distributions.append(distribution)
        detail = self.create_draft(distributions)
        for distribution in detail["metadata"]["distributions"]:
            with self.subTest(method=distribution["title"]):
                prepared = self.prepare(distribution)
                self.assertEqual(prepared["method"], distribution["title"])
                self.assertEqual(prepared["url"], "https://provider.invalid/query")
                self.assertEqual(prepared["headers"], {})
                self.assertIsNone(prepared["body"])

    def test_raw_url_path_and_query_encoding(self) -> None:
        """Verify path substitution and query escaping preserve serialized URL structure."""
        choices = [{"value": "a,b"}, {"value": "c/d"}]
        detail = self.create_draft([web_distribution([
            parameter("path", location="PATH", required=True),
            parameter("another", position=1, location="PATH", required=True),
            parameter("q name/&", position=2),
            parameter("repeat", position=3, multiple_values=True, explode=True, enum=choices),
            parameter("joined", position=4, multiple_values=True, enum=choices),
        ], template_url="https://provider.invalid/base%2Fkept/{path}/{path}/{another}?existing=a%2Fb&existing=two")])
        distribution = detail["metadata"]["distributions"][0]
        identifiers = {item["name"]: item["id"] for item in distribution["web_service"]["parameters"]}
        values = {
            "path": "one/ space%é,{another}", "another": "done", "q name/&": "one/two &%",
            "repeat": ["a,b", "c/d"], "joined": ["a,b", "c/d"],
        }
        prepared = self.prepare(
            distribution, values={identifiers[key]: value for key, value in values.items()},
        )
        escaped = "one%2F%20space%25%C3%A9%2C%7Banother%7D"
        self.assertEqual(
            prepared["url"],
            f"https://provider.invalid/base%2Fkept/{escaped}/{escaped}/done"
            "?existing=a%2Fb&existing=two&q+name%2F%26=one%2Ftwo+%26%25"
            "&repeat=a%2Cb&repeat=c%2Fd&joined=a%2Cb,c%2Fd",
        )

    def test_headers_body_and_authentication_are_independent(self) -> None:
        """Verify literal header arrays, explicit upstream auth and body content type behavior."""
        detail = self.create_draft([web_distribution([
            parameter("X-Values", location="HEADER", multiple_values=True,
                      enum=[{"value": "a/b"}, {"value": "c d"}]),
            parameter("Authorization", position=1, location="HEADER"),
            parameter("Content-Type", position=2, location="HEADER"),
        ], method="POST", body_content_type="application/json", body_template='{"message":"{{.text}}"}',
            body_parameters=[child("text")])])
        distribution = detail["metadata"]["distributions"][0]
        service = distribution["web_service"]
        identifiers = {item["name"]: item["id"] for item in service["parameters"]}
        body_identifier = service["body_parameters"][0]["id"]
        text = 'raw"\\text\nline é\x00'
        prepared = self.prepare(distribution, values={
            identifiers["X-Values"]: ["a/b", "c d"], body_identifier: text,
        })
        self.assertEqual(prepared["headers"], {"X-Values": "a/b,c d"})
        self.assertEqual(prepared["body"], '{"message":"' + text + '"}')
        self.assertEqual(prepared["url"], "https://provider.invalid/query")
        self.assertNotIn("Authorization", prepared["headers"])
        self.assertNotIn("Content-Type", prepared["headers"])
        explicit = self.prepare(distribution, values={
            identifiers["Authorization"]: "Bearer provider-only",
            identifiers["Content-Type"]: "text/plain", body_identifier: "body",
        })
        self.assertEqual(explicit["headers"], {
            "Authorization": "Bearer provider-only", "Content-Type": "text/plain",
        })
        for value in explicit["headers"].values():
            self.assertFalse(
                value == self.gateway.session.headers["Authorization"],
                "Platform credentials leaked into prepared headers",
            )

    def test_invalid_header_values_include_template_output(self) -> None:
        """Verify direct and rendered header values reject CR, LF, NUL and DEL safely."""
        detail = self.create_draft([web_distribution([
            parameter("X-Direct", location="HEADER"),
            parameter("X-Rendered", "TEMPLATE", position=1, location="HEADER",
                      template="{{.value}}", parameters=[child()]),
        ])])
        distribution = detail["metadata"]["distributions"][0]
        by_name = {item["name"]: item for item in distribution["web_service"]["parameters"]}
        identifiers = (
            by_name["X-Direct"]["id"], by_name["X-Rendered"]["parameters"][0]["id"],
        )
        for identifier in identifiers:
            for control in ("\r", "\n", "\r\n", "\x00", "\x7f"):
                with self.subTest(parameter=identifier, control=repr(control)):
                    secret = "provider-credential-canary"
                    self.assert_prepare_problem(
                        self.prepare_response(distribution, {identifier: secret + control + "injected"}),
                        target="headers", secrets=(secret,),
                    )
