from typing import Any
from urllib.parse import parse_qs, urlsplit

from distribution_fixtures import DistributionContractTestCase, parameter, web_distribution


def wfs_distribution(**configuration: Any) -> dict[str, Any]:
    return {
        "type": "WEB_SERVICE", "title": "WFS", "description": "Synthetic WFS contract",
        "web_service": {
            "name": "WFS",
            "configuration": {
                "type": "WFS", "get_feature_template_url": "https://provider.invalid/ows",
                "version": "2.0.0", "feature_types": ["example:faults", "example:quakes"],
                "default_feature_type": "example:faults", "output_formats": ["GEOJSON", "csv"],
                "default_output_format": "GEOJSON", "result_limit_editable": False,
                "parameters": [], **configuration,
            },
        },
    }


class WFSContractTest(DistributionContractTestCase):
    def wfs_prepare(
        self, distribution: dict[str, Any], *, output_format: str | None = None, **fields: Any,
    ) -> dict[str, Any]:
        response = self.gateway.post(
            "/distributions/{distributionId}/prepare", distributionId=distribution["id"],
            json={"type": "WFS", "parameters": [], **fields},
            params={} if output_format is None else {"output_format": output_format},
        )
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.headers.get("Cache-Control"), "no-store")
        prepared = response.json()
        self.assertEqual(prepared["method"], "GET")
        self.assertEqual(prepared["headers"], {})
        self.assertIsNone(prepared["body"])
        return prepared

    def test_metadata_create_read_replace_and_optional_limit(self) -> None:
        """Regression: WFS details and replacement preserve metadata without requiring count."""
        source = wfs_distribution(result_limit=50, result_limit_editable=True)
        detail = self.create_draft([source])
        distribution = detail["metadata"]["distributions"][0]
        stored = distribution["web_service"]["configuration"]
        for field, expected in source["web_service"]["configuration"].items():
            with self.subTest(field=field):
                self.assertEqual(stored[field], expected)
        self.assertCountEqual(
            [(item["kind"], item["output_format"], item["label"])
             for item in distribution["available_formats"]],
            [("ORIGINAL", "GEOJSON", "GEOJSON"), ("ORIGINAL", "csv", "csv")],
        )
        replacement = wfs_distribution(default_feature_type="example:quakes", default_output_format="csv")
        metadata = {
            "title": detail["metadata"]["title"], "description": "Replace WFS defaults",
            "distributions": [replacement],
        }
        self.assertEqual(self.replace_response(detail, metadata).status_code, 200)
        updated = self.details(detail["resource_id"], detail["id"])
        self.assertEqual(updated["revision"], detail["revision"] + 1)
        distribution = updated["metadata"]["distributions"][0]
        stored = distribution["web_service"]["configuration"]
        self.assertIsNone(stored.get("result_limit"))
        self.assertFalse(stored["result_limit_editable"])
        self.assertEqual(stored["default_feature_type"], "example:quakes")
        self.assertEqual(stored["default_output_format"], "csv")
        query = parse_qs(urlsplit(self.wfs_prepare(distribution)["url"]).query)
        self.assertEqual(query["typeNames"], ["example:quakes"])
        self.assertEqual(query["outputFormat"], ["csv"])
        self.assertNotIn("count", query)
        self.assertNotIn("maxFeatures", query)

    def test_versions_defaults_selections_and_encoding_without_provider_call(self) -> None:
        """Verify both WFS versions prepare defaults, CRS84 bounds and escaped custom values offline."""
        distributions = []
        for version in ("1.1.0", "2.0.0"):
            distribution = wfs_distribution(
                version=version,
                get_feature_template_url="https://provider.invalid/{workspace}/ows?existing=a%2Fb&existing=two",
                parameters=[
                    parameter("workspace", location="PATH", required=True, default="a/b"),
                    parameter("custom", position=1, default="a & b"),
                ],
            )
            distribution["title"] = version
            distributions.append(distribution)
        detail = self.create_draft(distributions)
        for distribution in detail["metadata"]["distributions"]:
            version = distribution["title"]
            feature_key = "typeName" if version == "1.1.0" else "typeNames"
            other_key = "typeNames" if version == "1.1.0" else "typeName"
            for explicit in (False, True):
                with self.subTest(version=version, explicit=explicit):
                    fields = {
                        "feature_type": "example:quakes", "output_format": "csv",
                        "bbox": [-10.0, 20.0, 30.0, 40.0],
                    } if explicit else {}
                    url = urlsplit(self.wfs_prepare(distribution, **fields)["url"])
                    query = parse_qs(url.query)
                    self.assertEqual(url.hostname, "provider.invalid")
                    self.assertEqual(url.path, "/a%2Fb/ows")
                    self.assertTrue(url.query.startswith("existing=a%2Fb&existing=two&"))
                    self.assertEqual(query["existing"], ["a/b", "two"])
                    self.assertEqual(query["custom"], ["a & b"])
                    self.assertEqual(query["SERVICE"], ["WFS"])
                    self.assertEqual(query["REQUEST"], ["GetFeature"])
                    self.assertEqual(query["VERSION"], [version])
                    self.assertEqual(query[feature_key], ["example:quakes" if explicit else "example:faults"])
                    self.assertNotIn(other_key, query)
                    self.assertEqual(query["outputFormat"], ["csv" if explicit else "GEOJSON"])
                    self.assertEqual(query["srsName"], ["urn:ogc:def:crs:OGC:1.3:CRS84"])
                    self.assertNotIn("count", query)
                    self.assertNotIn("maxFeatures", query)
                    if explicit:
                        self.assertEqual(query["BBOX"], ["-10,20,30,40,urn:ogc:def:crs:OGC:1.3:CRS84"])
                    else:
                        self.assertNotIn("BBOX", query)

    def test_fixed_and_editable_optional_limits(self) -> None:
        """Regression: absent limits omit count; fixed limits reject overrides but accept equal values."""
        distributions = []
        for version in ("1.1.0", "2.0.0"):
            for limit in (None, 50):
                for editable in (False, True):
                    distribution = wfs_distribution(
                        version=version, result_limit=limit, result_limit_editable=editable,
                    )
                    distribution["title"] = f"{version}-{limit}-{editable}"
                    distributions.append(distribution)
        detail = self.create_draft(distributions)
        for distribution in detail["metadata"]["distributions"]:
            config = distribution["web_service"]["configuration"]
            limit, editable = config.get("result_limit"), config["result_limit_editable"]
            key = "maxFeatures" if config["version"] == "1.1.0" else "count"
            for supplied in (None, 50, 75, 0):
                with self.subTest(service=distribution["title"], supplied=supplied):
                    body = {"type": "WFS", "parameters": []}
                    if supplied is not None:
                        body["result_limit"] = supplied
                    response = self.gateway.post(
                        "/distributions/{distributionId}/prepare", distributionId=distribution["id"], json=body,
                    )
                    if supplied == 0 or (supplied is not None and not editable and supplied != limit):
                        self.assert_prepare_problem(response, target="result_limit")
                        continue
                    self.assertEqual(response.status_code, 200)
                    query = parse_qs(urlsplit(response.json()["url"]).query)
                    effective = supplied if supplied is not None else limit
                    if effective is None:
                        self.assertNotIn(key, query)
                    else:
                        self.assertEqual(query[key], [str(effective)])
                    self.assertNotIn("count" if key == "maxFeatures" else "maxFeatures", query)

    def test_request_discriminator_and_invalid_selections(self) -> None:
        """Verify typed requests reject missing/mismatched types and undeclared WFS selections."""
        detail = self.create_draft([wfs_distribution(), web_distribution()])
        by_title = {item["title"]: item for item in detail["metadata"]["distributions"]}
        for title, body, params, target in (
            ("WFS", {"parameters": []}, {}, None),
            ("WFS", {"type": "HTTP", "parameters": []}, {}, "type"),
            ("prepared-service", {"type": "WFS", "parameters": []}, {}, "type"),
            ("WFS", {"type": "WFS", "feature_type": "unknown", "parameters": []}, {}, "feature_type"),
            ("WFS", {"type": "WFS", "parameters": []}, {"output_format": "unknown"}, "output_format"),
        ):
            with self.subTest(title=title, body=body, params=params):
                response = self.gateway.post(
                    "/distributions/{distributionId}/prepare", distributionId=by_title[title]["id"],
                    json=body, params=params,
                )
                self.assert_prepare_problem(response, target=target)
