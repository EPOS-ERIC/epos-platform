from urllib.parse import parse_qs, urlsplit

from distribution_fixtures import DistributionContractTestCase, parameter, web_distribution


class DistributionDatetimesContractTest(DistributionContractTestCase):
    def test_datetime_normalization_layouts_and_defaults(self) -> None:
        """Verify datetime inputs and defaults use UTC and configured layouts."""
        detail = self.create_draft([web_distribution([
            parameter("instant", "DATETIME"),
            parameter("formatted", "DATETIME", position=1, layout="YYYY/MM/DD hh:mm:ss"),
            parameter("defaulted", "DATETIME", position=2, default="2026-10-21T12:30:00+02:00"),
            parameter("formatted_default", "DATETIME", position=3,
                      default="2026-10-21T05:30:00-05:00", layout="YYYY-MM-DD hh:mm:ss"),
        ])])
        distribution = detail["metadata"]["distributions"][0]
        identifiers = {item["name"]: item["id"] for item in distribution["web_service"]["parameters"]}
        for instant in ("2026-10-21T12:30:00+02:00", "2026-10-21T05:30:00-05:00", "2026-10-21T10:30:00Z"):
            with self.subTest(instant=instant):
                prepared = self.prepare(distribution, values={
                    identifiers["instant"]: instant, identifiers["formatted"]: instant,
                })
                query = parse_qs(urlsplit(prepared["url"]).query)
                self.assertEqual(query["instant"], ["2026-10-21T10:30:00Z"])
                self.assertEqual(query["formatted"], ["2026/10/21 10:30:00"])
                self.assertEqual(query["defaulted"], ["2026-10-21T10:30:00Z"])
                self.assertEqual(query["formatted_default"], ["2026-10-21 10:30:00"])
        for value in ("not-a-date", "2026-10-21", "2026-13-21T00:00:00Z", "2026-10-21T10:30:00", 42):
            with self.subTest(invalid=value):
                self.assert_prepare_problem(
                    self.prepare_response(distribution, {identifiers["instant"]: value}),
                    target="parameters[0].value", secrets=(str(value),),
                )

    def test_datetime_ranges_and_safe_bound_errors(self) -> None:
        """Verify inclusive one-sided and two-sided datetime ranges without input leaks."""
        minimum = "2026-10-20T00:00:00Z"
        maximum = "2026-10-22T00:00:00Z"
        detail = self.create_draft([web_distribution([
            parameter("minimum", "DATETIME", minimum=minimum),
            parameter("maximum", "DATETIME", position=1, maximum=maximum),
            parameter("both", "DATETIME", position=2, minimum=minimum, maximum=maximum),
        ])])
        distribution = detail["metadata"]["distributions"][0]
        identifiers = {item["name"]: item["id"] for item in distribution["web_service"]["parameters"]}
        for name, accepted, rejected in (
            ("minimum", [minimum, maximum], ["2026-10-19T23:59:59Z"]),
            ("maximum", [minimum, maximum], ["2026-10-22T00:00:01Z"]),
            ("both", [minimum, "2026-10-21T00:00:00Z", maximum],
             ["2026-10-19T23:59:59Z", "2026-10-22T00:00:01Z"]),
        ):
            for value in accepted:
                with self.subTest(parameter=name, accepted=value):
                    prepared = self.prepare(distribution, values={identifiers[name]: value})
                    self.assertEqual(parse_qs(urlsplit(prepared["url"]).query)[name], [value])
            for value in rejected:
                with self.subTest(parameter=name, rejected=value):
                    self.assert_prepare_problem(
                        self.prepare_response(distribution, {identifiers[name]: value}),
                        target="parameters[0].value", secrets=(value,),
                    )
