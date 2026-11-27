from urllib.parse import parse_qs, urlsplit

from distribution_fixtures import DistributionContractTestCase, child, parameter, web_distribution


class DistributionNestedParameterRolesContractTest(DistributionContractTestCase):
    def test_wfs_filter_template_preserves_roles_and_renders_values(self) -> None:
        """Regression: roles on filter children survive and render correctly."""
        children = [
            child("minlongitude", "FLOAT", role="MIN_LON", default=-180),
            child("minlatitude", "FLOAT", position=1, role="MIN_LAT", default=-90),
            child("maxlongitude", "FLOAT", position=2, role="MAX_LON", default=180),
            child("maxlatitude", "FLOAT", position=3, role="MAX_LAT", default=90),
            child("minmagnitude", "FLOAT", position=4, default=0),
            child("startdate", "DATETIME", position=5, role="START_DATE", layout="YYYY-MM-DD",
                  default="1970-01-01T00:00:00Z"),
            child("enddate", "DATETIME", position=6, role="END_DATE", layout="YYYY-MM-DD",
                  default="2020-01-01T00:00:00Z"),
        ]
        template = (
            "<ogc:Filter><ogc:And><ogc:BBOX><ogc:PropertyName>the_geom</ogc:PropertyName>"
            "<gml:Envelope srsName='EPSG:4326'><gml:lowerCorner>{{.minlongitude}} {{.minlatitude}}"
            "</gml:lowerCorner><gml:upperCorner>{{.maxlongitude}} {{.maxlatitude}}</gml:upperCorner>"
            "</gml:Envelope></ogc:BBOX><ogc:PropertyIsGreaterThanOrEqualTo>"
            "<ogc:PropertyName>magnitude</ogc:PropertyName><ogc:Literal>{{.minmagnitude}}</ogc:Literal>"
            "</ogc:PropertyIsGreaterThanOrEqualTo><ogc:PropertyIsGreaterThanOrEqualTo>"
            "<ogc:PropertyName>date</ogc:PropertyName><ogc:Literal>{{.startdate}}</ogc:Literal>"
            "</ogc:PropertyIsGreaterThanOrEqualTo><ogc:PropertyIsLessThanOrEqualTo>"
            "<ogc:PropertyName>date</ogc:PropertyName><ogc:Literal>{{.enddate}}</ogc:Literal>"
            "</ogc:PropertyIsLessThanOrEqualTo></ogc:And></ogc:Filter>"
        )
        detail = self.create_draft([web_distribution([
            parameter("service", default="WFS"),
            parameter("request", position=1, default="GetFeature"),
            parameter("filter", "TEMPLATE", position=2, required=True, template=template, parameters=children),
        ])])
        distribution = detail["metadata"]["distributions"][0]
        params = {item["name"]: item for item in distribution["web_service"]["parameters"]}
        stored = {item["name"]: item for item in params["filter"]["parameters"]}
        for source in children:
            with self.subTest(parameter=source["name"]):
                self.assertEqual(stored[source["name"]].get("role"), source.get("role"))
                self.assertNotIn("property", stored[source["name"]])

        values = {
            "minlongitude": -26, "minlatitude": 30, "maxlongitude": 46, "maxlatitude": 90,
            "minmagnitude": 1.5, "startdate": "2000-01-01T00:00:00Z", "enddate": "2014-12-31T00:00:00Z",
        }
        prepared = self.prepare(distribution, values={stored[name]["id"]: value for name, value in values.items()})
        query = parse_qs(urlsplit(prepared["url"]).query)
        self.assertEqual(query["service"], ["WFS"])
        self.assertEqual(query["request"], ["GetFeature"])
        expected = (template.replace("{{.minlongitude}}", "-26").replace("{{.minlatitude}}", "30")
                    .replace("{{.maxlongitude}}", "46").replace("{{.maxlatitude}}", "90")
                    .replace("{{.minmagnitude}}", "1.5").replace("{{.startdate}}", "2000-01-01")
                    .replace("{{.enddate}}", "2014-12-31"))
        self.assertEqual(query["filter"], [expected])

    def test_format_selection_in_template_and_body(self) -> None:
        """Regression: nested format roles select output in query templates and request bodies."""
        format_child = child("format", role="FORMAT", read_only=True,
                             default="xml", enum=[{"value": "xml"}, {"value": "json"}])
        query = web_distribution([parameter(
            "filter", "TEMPLATE", required=True, template="format={{.format}}", parameters=[format_child],
        )])
        query["title"] = "query"
        body = web_distribution(method="POST", body_content_type="text/plain", body_required=True,
                                body_template="format={{.format}}", body_parameters=[format_child])
        body["title"] = "body"
        detail = self.create_draft([query, body])
        for distribution in detail["metadata"]["distributions"]:
            config = distribution["web_service"]
            stored = (config["parameters"][0]["parameters"][0] if distribution["title"] == "query"
                      else config["body_parameters"][0])
            self.assertEqual(stored["role"], "FORMAT")
            self.assertNotIn("property", stored)
            for selected in (None, "json"):
                with self.subTest(location=distribution["title"], selected=selected):
                    kwargs = {} if selected is None else {"params": {"output_format": selected}}
                    prepared = self.prepare(distribution, **kwargs)
                    expected = "format=" + (selected or "xml")
                    if distribution["title"] == "query":
                        self.assertEqual(parse_qs(urlsplit(prepared["url"]).query)["filter"], [expected])
                    else:
                        self.assertEqual(prepared["body"], expected)
