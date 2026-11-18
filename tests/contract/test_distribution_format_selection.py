from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, parameter, web_distribution


class DistributionFormatSelectionContractTest(DistributionContractTestCase):
    def test_native_format_defaults_and_overrides_at_each_location(self) -> None:
        """Verify native selectors override managed PATH, QUERY and HEADER encoding values."""
        distributions = []
        for location in ("PATH", "QUERY", "HEADER"):
            distribution = web_distribution([parameter(
                "format", location=location, required=location == "PATH", read_only=True,
                property="custom:format-metadata", role="FORMAT", default="xml",
                enum=[{"value": "xml"}, {"value": "json"}],
            )], template_url=(
                "https://provider.invalid/{format}" if location == "PATH"
                else "https://provider.invalid/query"
            ))
            distribution["title"] = location
            distributions.append(distribution)
        detail = self.create_draft(distributions)
        for distribution in detail["metadata"]["distributions"]:
            location = distribution["title"]
            for selected in (None, "json"):
                with self.subTest(location=location, selected=selected):
                    kwargs = {} if selected is None else {"params": {"output_format": selected}}
                    prepared = self.prepare(distribution, **kwargs)
                    value = selected or "xml"
                    if location == "PATH":
                        self.assertEqual(prepared["url"], f"https://provider.invalid/{value}")
                        self.assertEqual(prepared["headers"], {})
                    elif location == "QUERY":
                        self.assertEqual(prepared["url"], f"https://provider.invalid/query?format={value}")
                        self.assertEqual(prepared["headers"], {})
                    else:
                        self.assertEqual(prepared["url"], "https://provider.invalid/query")
                        self.assertEqual(prepared["headers"], {"Format": value})
            identifier = distribution["web_service"]["parameters"][0]["id"]
            for selectors in ({}, {"output_format": "json"}):
                with self.subTest(location=location, bypass=selectors):
                    self.assert_prepare_problem(
                        self.prepare_response(distribution, {identifier: "json"}, params=selectors),
                        target="parameters[0].id",
                    )

    def test_fixed_formats_and_selector_validation(self) -> None:
        """Regression: malformed selectors must identify their field, not return generic Problems."""
        fixed_parameter = web_distribution(
            [
                parameter(
                    "format",
                    read_only=True,
                    property="custom:format-metadata",
                    role="FORMAT",
                    default="xml",
                )
            ]
        )
        fixed_parameter["title"] = "fixed-parameter"
        fixed_service = web_distribution(format="Application/JSON")
        fixed_service["title"] = "fixed-service"
        detail = self.create_draft([fixed_parameter, fixed_service])
        by_title = {item["title"]: item for item in detail["metadata"]["distributions"]}
        distribution = by_title["fixed-parameter"]
        expected = self.prepare(distribution)
        self.assertEqual(self.prepare(distribution, params={"output_format": "xml"}), expected)
        service = by_title["fixed-service"]
        self.assertEqual(
            self.prepare(service, params={"output_format": "Application/JSON"}), self.prepare(service),
        )
        self.assertEqual(self.prepare(service)["headers"], {})
        for target, params in (
            ("output_format", {"output_format": "unsupported"}),
            ("output_format", {"output_format": "XML"}),
            ("output_format", {"output_format": ""}),
            ("output_format", {"output_format": ["xml", "xml"]}),
            ("output_format", {"output_format": ["xml", "json"]}),
            ("conversion_id", {"output_format": "xml", "conversion_id": str(uuid4())}),
            ("conversion_id", {"conversion_id": "invalid-uuid"}),
        ):
            with self.subTest(params=params):
                self.assert_prepare_problem(
                    self.prepare_response(distribution, params=params), target=target,
                )
        self.assert_prepare_problem(
            self.prepare_response(service, params={"output_format": "application/json"}),
            target="output_format",
        )
