import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import create_group

NO_AUTH_HEADERS = {"Authorization": ""}
RESOURCE_TYPES = (
    "dataset",
    "software-application",
    "software-source-code",
)


class DiscoveryFiltersContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def create_user(self, label: str, suffix: str) -> tuple[GatewayClient, str]:
        credentials = register_user(
            name="Discovery",
            surname=f"{label.title()} {suffix}",
            email=f"discovery-{uuid4().hex}@example.org",
            password=uuid4().hex,
        )
        client = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(
                username=credentials.username,
                password=credentials.password,
            ),
        )
        response = client.get("/users/{id}", id="self")
        self.assertEqual(response.status_code, 200)
        return client, response.json()["id"]

    def accept_editor(self, client: GatewayClient, group_id: str) -> str:
        request = client.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json={"roles": []},
        )
        self.assertEqual(request.status_code, 201)
        user_id = request.json()["user"]["id"]
        accepted = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="accept",
        )
        self.assertEqual(accepted.status_code, 200)
        self.assertEqual(accepted.json()["status"], "ACTIVE")
        role = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=user_id,
            role="EDITOR",
        )
        self.assertEqual(role.status_code, 204)
        return user_id

    def create_organization(self, suffix: str) -> str:
        response = self.gateway.post(
            "/organizations",
            json={
                "name": f"Discovery Provider {suffix}",
                "address": {
                    "street": "Discovery Street 1",
                    "locality": "Rome",
                    "postal_code": "00100",
                    "country": "IT",
                },
                "logo_url": None,
                "page_url": None,
                "email": None,
                "lei_code": None,
                "identifiers": [{"type": "discovery", "value": suffix}],
                "parents": [],
                "contact_point_ids": [],
            },
        )
        self.assertEqual(response.status_code, 201)
        return response.json()["id"]

    def create_resource(
        self,
        client: GatewayClient,
        group_id: str,
        resource_type: str,
        metadata: dict[str, Any],
    ) -> tuple[str, str]:
        created = client.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={"resource_type": resource_type, "metadata": metadata},
        )
        self.assertEqual(created.status_code, 201)
        resource_id = created.json()["id"]
        versions = client.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType=resource_type,
            id=resource_id,
        )
        self.assertEqual(versions.status_code, 200)
        drafts = [version for version in versions.json() if version["status"] == "DRAFT"]
        self.assertEqual(len(drafts), 1)
        draft = drafts[0]
        version_id = draft["version_id"]
        replaced = client.put(
            "/resources/{id}/versions/{versionId}/draft",
            id=resource_id,
            versionId=version_id,
            json={
                "resource_type": resource_type,
                "revision": draft["revision"],
                "metadata": metadata,
            },
        )
        self.assertEqual(replaced.status_code, 200)
        return resource_id, version_id

    def publish(self, resource_type: str, resource_id: str, version_id: str) -> None:
        submitted = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType=resource_type,
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(submitted.status_code, 200)
        published = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/publish",
            resourceType=resource_type,
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(published.status_code, 200)

    def search(
        self,
        client: GatewayClient,
        *,
        headers: dict[str, str] | None = None,
        **params: str,
    ) -> list[dict[str, Any]]:
        response = client.get("/search", params=params or None, headers=headers)
        self.assertEqual(response.status_code, 200)
        result = response.json()
        self.assertIsInstance(result, list)
        return result

    def assert_ids(
        self,
        items: list[dict[str, Any]],
        expected: set[str],
    ) -> None:
        self.assertEqual(
            {item["resource_id"] for item in items},
            expected,
            items,
        )

    def details(self, item: dict[str, Any]) -> dict[str, Any]:
        response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType=item["type"],
            id=item["resource_id"],
            versionId=item["version_id"],
        )
        self.assertEqual(response.status_code, 200)
        return response.json()

    def test_discovery_filters_and_visibility(self) -> None:
        """Verify discovery filters and group-scoped creator projection."""

        suffix = uuid4().hex
        data_group = create_group(self.gateway, f"discovery-data-{suffix}")
        software_group = create_group(self.gateway, f"discovery-software-{suffix}", "SOFTWARE")
        provider_id = self.create_organization(suffix)
        editor, editor_id = self.create_user("editor", suffix)
        outsider, _ = self.create_user("outsider", suffix)
        self.accept_editor(editor, data_group)

        dataset_name = f"discovery-dataset-{suffix}"
        dataset_metadata = {
            "title": dataset_name,
            "description": f"dataset description {suffix}",
            "keywords": [f"dataset-keyword-{suffix}", "shared-keyword"],
            "distributions": [
                {
                    "title": f"discovery-service-{suffix}",
                    "description": "Discovery web service with display metadata",
                    "type": "WEB_SERVICE",
                    "web_service": {
                        "name": f"discovery-service-{suffix}",
                        "template_url": "https://example.org/discovery",
                        "method": "GET",
                        "type": "COVJSON",
                        "visible_on": ["MAP", "TABLE"],
                        "format": "application/json",
                        "provider": provider_id,
                    },
                },
                {
                    "title": f"discovery-service-optional-{suffix}",
                    "description": "Discovery web service without optional metadata",
                    "type": "WEB_SERVICE",
                    "web_service": {
                        "name": f"discovery-service-optional-{suffix}",
                        "template_url": "https://example.org/discovery/optional",
                        "method": "GET",
                        "type": "GEOJSON",
                        "format": "application/geo+json",
                    },
                },
                {
                    "title": f"discovery-file-{suffix}",
                    "description": "Discovery downloadable file",
                    "type": "DOWNLOADABLE_FILE",
                    "downloadable_file": {
                        "url": "https://example.org/discovery.csv",
                        "format": "text/csv",
                    },
                },
            ],
            "providers": [provider_id],
            "spatial_coverage": "POINT (12.5 41.9)",
            "temporal_coverage": {
                "start": "2020-01-01T00:00:00Z",
                "end": "2020-01-31T00:00:00Z",
            },
        }
        dataset_id, dataset_version_id = self.create_resource(
            self.gateway, data_group, "dataset", dataset_metadata
        )
        self.publish("dataset", dataset_id, dataset_version_id)
        dataset_details_response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=dataset_id,
            versionId=dataset_version_id,
        )
        self.assertEqual(dataset_details_response.status_code, 200)
        distributions_by_title = {
            distribution["title"]: distribution
            for distribution in dataset_details_response.json()["metadata"]["distributions"]
        }
        service_id = distributions_by_title[f"discovery-service-{suffix}"]["id"]
        optional_service_id = distributions_by_title[f"discovery-service-optional-{suffix}"]["id"]
        file_id = distributions_by_title[f"discovery-file-{suffix}"]["id"]
        for title, format_value in (
            (f"discovery-service-{suffix}", "application/json"),
            (f"discovery-service-optional-{suffix}", "application/geo+json"),
        ):
            distribution = distributions_by_title[title]
            self.assertEqual(
                distribution["available_formats"],
                [{"kind": "ORIGINAL", "output_format": format_value, "label": format_value}],
            )
            self.assertNotIn("available_formats", distribution["web_service"])
        self.assertNotIn(
            "available_formats", distributions_by_title[f"discovery-file-{suffix}"]
        )

        draft_name = f"discovery-draft-{suffix}"
        draft_id, draft_version_id = self.create_resource(
            editor,
            data_group,
            "dataset",
            {
                "title": draft_name,
                "description": f"private draft {suffix}",
                "keywords": [f"draft-keyword-{suffix}"],
            },
        )

        application_name = f"discovery-application-{suffix}"
        application_id, application_version_id = self.create_resource(
            self.gateway,
            software_group,
            "software-application",
            {
                "name": application_name,
                "description": f"application description {suffix}",
                "keywords": [f"software-keyword-{suffix}", "shared-keyword"],
                "creators": [provider_id],
            },
        )
        self.publish("software-application", application_id, application_version_id)

        source_name = f"discovery-source-{suffix}"
        source_id, source_version_id = self.create_resource(
            self.gateway,
            software_group,
            "software-source-code",
            {
                "name": source_name,
                "description": f"source description {suffix}",
                "keywords": [f"source-keyword-{suffix}", "shared-keyword"],
                "creators": [provider_id],
            },
        )
        self.publish("software-source-code", source_id, source_version_id)

        baseline = self.search(self.gateway, headers=NO_AUTH_HEADERS)
        self.assertTrue(baseline)
        self.assertTrue(all(item["status"] == "PUBLISHED" for item in baseline))
        self.assertIn(dataset_id, {item["resource_id"] for item in baseline})
        self.assertTrue(all("created_by" not in item for item in baseline))

        admin_items = self.search(self.gateway, q=suffix)
        self.assert_ids(admin_items, {dataset_id, application_id, source_id})
        self.assertTrue(all("created_by" in item for item in admin_items))
        dataset_distributions = {
            distribution["id"]: distribution
            for distribution in next(
                item for item in admin_items if item["resource_id"] == dataset_id
            )["distributions"]
        }
        self.assertEqual(
            dataset_distributions[service_id],
            {
                "id": service_id,
                "name": f"discovery-service-{suffix}",
                "type": "WEB_SERVICE",
                "web_service_type": "COVJSON",
                "visible_on": ["MAP", "TABLE"],
                "provider": {
                    "id": provider_id,
                    "name": f"Discovery Provider {suffix}",
                },
            },
        )
        self.assertEqual(
            dataset_distributions[optional_service_id],
            {
                "id": optional_service_id,
                "name": f"discovery-service-optional-{suffix}",
                "type": "WEB_SERVICE",
                "web_service_type": "GEOJSON",
            },
        )
        self.assertEqual(
            dataset_distributions[file_id],
            {
                "id": file_id,
                "name": f"discovery-file-{suffix}",
                "type": "DOWNLOADABLE_FILE",
            },
        )

        member_items = self.search(
            editor,
            q=suffix,
            versioningStatus="PUBLISHED,DRAFT",
        )
        self.assert_ids(member_items, {dataset_id, draft_id, application_id, source_id})
        self.assertIn("created_by", next(item for item in member_items if item["resource_id"] == draft_id))
        self.assertIn("created_by", next(item for item in member_items if item["resource_id"] == dataset_id))
        self.assertNotIn("created_by", next(item for item in member_items if item["resource_id"] == application_id))
        self.assertNotIn("created_by", next(item for item in member_items if item["resource_id"] == source_id))

        outsider_items = self.search(outsider, q=suffix)
        self.assert_ids(outsider_items, {dataset_id, application_id, source_id})
        self.assertTrue(all("created_by" not in item for item in outsider_items))

        cases = [
            ({"type": "dataset", "q": suffix}, {dataset_id}),
            ({"type": "software-application,software-source-code", "q": suffix}, {application_id, source_id}),
            ({"concept": "DATA", "q": suffix}, {dataset_id}),
            ({"concept": "SOFTWARE", "q": suffix}, {application_id, source_id}),
            ({"groupId": data_group, "q": suffix}, {dataset_id}),
            ({"editorId": editor_id, "versioningStatus": "DRAFT", "q": suffix}, {draft_id}),
            ({"provider": provider_id, "q": suffix}, {dataset_id, application_id, source_id}),
            ({"q": f"description {suffix}"}, {dataset_id, application_id, source_id}),
            ({"keywords": f"dataset-keyword-{suffix},shared-keyword"}, {dataset_id}),
            ({"dateFrom": "2020-01-31T00:00:00Z", "q": suffix}, {dataset_id}),
            ({"dateTo": "2020-01-01T00:00:00Z", "q": suffix}, {dataset_id}),
            ({"spatialFilter": "POINT (12.5 41.9)", "q": suffix}, {dataset_id}),
            ({"spatialFilter": "POINT (0 0)", "q": suffix}, set()),
            ({"type": "dataset", "concept": "DATA", "groupId": data_group, "q": suffix}, {dataset_id}),
            ({"versioningStatus": "PUBLISHED,DRAFT", "q": suffix}, {dataset_id, draft_id, application_id, source_id}),
        ]
        for params, expected in cases:
            with self.subTest(params=params):
                self.assert_ids(self.search(self.gateway, **params), expected)

        self.assertEqual(self.details(next(item for item in admin_items if item["resource_id"] == dataset_id))["status"], "PUBLISHED")
        self.assertEqual(self.details(next(item for item in member_items if item["resource_id"] == draft_id))["status"], "DRAFT")

        invalid_cases = [
            {"type": "unsupported", "q": suffix},
            {"groupId": "not-a-uuid"},
            {"provider": "not-a-uuid"},
            {"spatialFilter": "not-wkt"},
            {"dateFrom": "not-a-date"},
            {"dateFrom": "2021-01-01T00:00:00Z", "dateTo": "2020-01-01T00:00:00Z"},
        ]
        for params in invalid_cases:
            with self.subTest(params=params):
                response = self.gateway.get("/search", params=params)
                self.assertEqual(response.status_code, 400)
