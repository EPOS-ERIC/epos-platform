from typing import Any
from urllib.parse import parse_qs, urlsplit
from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, child, parameter, web_distribution
from fixtures import assert_problem, create_group
from test_wfs import wfs_distribution


def coordinate(name: str, role: str, **fields: Any) -> dict[str, Any]:
    return parameter(name, "FLOAT", role=role, required=True, **fields)


def dynamic_distribution(parameters: list[dict[str, Any]], **configuration: Any) -> dict[str, Any]:
    return web_distribution(parameters, type="DYNAMIC", **configuration)


class DynamicWebServiceContractTest(DistributionContractTestCase):
    def request(self, distribution: dict[str, Any], endpoint: str, body: dict[str, Any]) -> Any:
        return self.gateway.post(
            "/distributions/{distributionId}/" + endpoint,
            distributionId=distribution["id"], json=body,
        )

    def test_configuration_validation(self) -> None:
        """Verify coordinate roles, uniqueness, schema restrictions and service-type exclusivity."""
        group_id = create_group(self.gateway, f"dynamic-{uuid4().hex}")
        lon = coordinate("lon", "LONGITUDE")
        lat = coordinate("lat", "LATITUDE")
        cases = [
            ("missing both", dynamic_distribution([])),
            ("missing latitude", dynamic_distribution([lon])),
            ("missing longitude", dynamic_distribution([lat])),
            ("duplicate longitude", dynamic_distribution([lon, lat, coordinate("other", "LONGITUDE")])),
            ("duplicate nested role", dynamic_distribution([
                lon, lat, parameter("point", "TEMPLATE", template="{{.other}}", parameters=[
                    child("other", "FLOAT", required=True, role="LONGITUDE"),
                ]),
            ])),
            ("integer", dynamic_distribution([parameter("lon", "INTEGER", role="LONGITUDE", required=True), lat])),
            ("optional", dynamic_distribution([parameter("lon", "FLOAT", role="LONGITUDE"), lat])),
            ("read only", dynamic_distribution([coordinate("lon", "LONGITUDE", read_only=True), lat])),
            ("default", dynamic_distribution([coordinate("lon", "LONGITUDE", default=1.0), lat])),
            ("enum", dynamic_distribution([coordinate("lon", "LONGITUDE", enum=[{"value": 1.0}]), lat])),
            ("multiple", dynamic_distribution([coordinate("lon", "LONGITUDE", multiple_values=True,
                                                           enum=[{"value": 1.0}]), lat])),
            ("explode", dynamic_distribution([coordinate("lon", "LONGITUDE", explode=True), lat])),
            ("http role", web_distribution([lon, lat])),
            ("wfs role", wfs_distribution(parameters=[lon, lat])),
        ]
        for label, distribution in cases:
            with self.subTest(case=label):
                response = self.gateway.post(
                    "/groups/{groupId}/resources", groupId=group_id,
                    json={"resource_type": "dataset", "metadata": {
                        "title": f"dynamic-{uuid4().hex}", "description": "Dynamic validation",
                        "distributions": [distribution],
                    }},
                )
                assert_problem(self, response, 400, "validation_failed")

    def test_prepare_and_access_request_validation(self) -> None:
        """Verify typed coordinates, bounds, duplicate assignments and request-type matching."""
        source = dynamic_distribution([
            coordinate("lon", "LONGITUDE", minimum=-180.0, maximum=180.0),
            coordinate("lat", "LATITUDE", position=1, minimum=-90.0, maximum=90.0),
            parameter("other", position=2, default="ok"),
            parameter("west", "FLOAT", position=3, role="MIN_LON", default=-1.0),
        ])
        detail = self.create_draft([source, web_distribution()])
        dynamic = next(item for item in detail["metadata"]["distributions"]
                       if item["web_service"]["configuration"]["type"] == "DYNAMIC")
        http = next(item for item in detail["metadata"]["distributions"]
                    if item["web_service"]["configuration"]["type"] == "HTTP")
        config = dynamic["web_service"]["configuration"]
        params = {item["name"]: item for item in config["parameters"]}
        self.assertEqual(params["lon"]["role"], "LONGITUDE")
        self.assertEqual(params["lat"]["role"], "LATITUDE")
        body = {"type": "DYNAMIC", "longitude": 12.5, "latitude": -42.0, "parameters": []}
        response = self.request(dynamic, "prepare", body)
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.headers.get("Cache-Control"), "no-store")
        self.assertEqual(parse_qs(urlsplit(response.json()["url"]).query), {
            "lon": ["12.5"], "lat": ["-42"], "other": ["ok"], "west": ["-1"],
        })
        boundary = self.request(dynamic, "prepare", {**body, "longitude": -180, "latitude": 90})
        self.assertEqual(boundary.status_code, 200)
        zero = self.request(dynamic, "prepare", {**body, "longitude": 0, "latitude": 0})
        self.assertEqual(zero.status_code, 200)
        self.assertEqual(parse_qs(urlsplit(zero.json()["url"]).query)["lon"], ["0"])
        self.assertEqual(parse_qs(urlsplit(zero.json()["url"]).query)["lat"], ["0"])
        cases = [
            (dynamic, {"type": "DYNAMIC", "latitude": 0, "parameters": []}),
            (dynamic, {"type": "DYNAMIC", "longitude": 0, "parameters": []}),
            (dynamic, {**body, "longitude": 181}),
            (dynamic, {**body, "latitude": -91}),
            (dynamic, {**body, "parameters": [{"id": params["lon"]["id"], "value": 12.5}]}),
            (dynamic, {**body, "parameters": [{"id": params["lat"]["id"], "value": -42}]}),
            (dynamic, {"type": "HTTP", "parameters": []}),
            (http, body),
        ]
        for endpoint in ("prepare", "access"):
            for distribution, request in cases:
                with self.subTest(endpoint=endpoint, body=request):
                    self.assert_prepare_problem(self.request(distribution, endpoint, request))

        replacement = {
            "title": detail["metadata"]["title"], "description": "Replace dynamic configuration",
            "distributions": [source],
        }
        self.assertEqual(self.replace_response(detail, replacement).status_code, 200)
        updated = self.details(detail["resource_id"], detail["version_id"])
        replaced = updated["metadata"]["distributions"][0]
        self.assertEqual(replaced["web_service"]["configuration"]["type"], "DYNAMIC")
        self.assertEqual(self.request(replaced, "prepare", body).status_code, 200)

    def test_clone_nested_coordinate_roles(self) -> None:
        """Regression: both clone flows preserve template-child roles and properties."""
        source = dynamic_distribution([
            parameter("point", "TEMPLATE", template="{{.lon}},{{.lat}}", parameters=[
                child("lon", "FLOAT", required=True, role="LONGITUDE", property="geo:longitude"),
                child("lat", "FLOAT", position=1, required=True, role="LATITUDE", property="geo:latitude"),
            ]),
        ])
        detail = self.create_draft([source])
        for action in ("submit", "publish"):
            self.transition(detail, action)
        target_group = create_group(self.gateway, f"dynamic-copy-{uuid4().hex}")
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
            with self.subTest(resource_id=resource_id):
                version = self.version(resource_id, "DRAFT")
                clone = self.details(resource_id, version["version_id"])
                distribution = clone["metadata"]["distributions"][0]
                parameters = distribution["web_service"]["configuration"]["parameters"][0]["parameters"]
                self.assertEqual(
                    {item["name"]: (item["role"], item["property"]) for item in parameters},
                    {"lon": ("LONGITUDE", "geo:longitude"), "lat": ("LATITUDE", "geo:latitude")},
                )
                response = self.request(distribution, "prepare", {
                    "type": "DYNAMIC", "longitude": 12.5, "latitude": -42, "parameters": [],
                })
                self.assertEqual(response.status_code, 200)
                self.assertEqual(parse_qs(urlsplit(response.json()["url"]).query)["point"], ["12.5,-42"])

    def test_coordinate_locations_and_template_activation(self) -> None:
        """Verify coordinates render in path, header, query templates and optional bodies."""
        path_header = dynamic_distribution([
            coordinate("lon", "LONGITUDE", location="PATH"),
            coordinate("X-Latitude", "LATITUDE", location="HEADER", position=1),
        ], template_url="https://provider.invalid/{lon}")
        template = dynamic_distribution([
            parameter("point", "TEMPLATE", template="{{.lon}},{{.lat}}", parameters=[
                child("lon", "FLOAT", required=True, role="LONGITUDE"),
                child("lat", "FLOAT", required=True, role="LATITUDE"),
                child("extra"),
            ]),
        ])
        template["title"] = "template"
        body_service = dynamic_distribution([], method="POST", body_content_type="text/plain",
                                            body_required=False, body_template="{{.lon}},{{.lat}},{{.extra}}",
                                            body_parameters=[
                                                child("lon", "FLOAT", required=True, role="LONGITUDE"),
                                                child("lat", "FLOAT", required=True, role="LATITUDE"),
                                                child("extra"),
                                            ])
        body_service["title"] = "body"
        detail = self.create_draft([path_header, template, body_service])
        distributions = {item["title"]: item for item in detail["metadata"]["distributions"]}
        request = {"type": "DYNAMIC", "longitude": 12.5, "latitude": -42, "parameters": []}
        path = self.request(distributions["prepared-service"], "prepare", request)
        self.assertEqual(path.status_code, 200)
        self.assertEqual(path.json()["url"], "https://provider.invalid/12.5")
        self.assertEqual(path.json()["headers"]["X-Latitude"], "-42")
        for title in ("template", "body"):
            with self.subTest(location=title):
                distribution = distributions[title]
                self.assert_prepare_problem(self.request(distribution, "prepare", request))
                config = distribution["web_service"]["configuration"]
                children = config["parameters"][0]["parameters"] if title == "template" else config["body_parameters"]
                extra = next(item for item in children if item["name"] == "extra")
                response = self.request(distribution, "prepare", {
                    **request, "parameters": [{"id": extra["id"], "value": "ok"}],
                })
                self.assertEqual(response.status_code, 200)
                if title == "template":
                    self.assertEqual(parse_qs(urlsplit(response.json()["url"]).query)["point"], ["12.5,-42"])
                else:
                    self.assertEqual(response.json()["body"], "12.5,-42,ok")
