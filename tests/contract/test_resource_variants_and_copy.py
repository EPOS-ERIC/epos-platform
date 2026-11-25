import copy
import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from distribution_fixtures import child, parameter, web_distribution
from fixtures import create_group


class ResourceVariantsAndCopyContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def create_user(self, label: str, suffix: str) -> tuple[GatewayClient, str]:
        credentials = register_user(
            name="Variants",
            surname=f"{label.title()} {suffix}",
            email=f"variants-{uuid4().hex}@example.org",
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

    def add_membership(self, client: GatewayClient, group_id: str, role: str) -> str:
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
        assigned = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=user_id,
            role=role,
        )
        self.assertEqual(assigned.status_code, 204)
        return user_id

    def create_contact_point(self, suffix: str) -> str:
        response = self.gateway.post(
            "/contact-points",
            json={"email": f"variants-{suffix}@example.org"},
        )
        self.assertEqual(response.status_code, 201)
        return response.json()["id"]

    def create_organization(self, suffix: str) -> str:
        response = self.gateway.post(
            "/organizations",
            json={
                "name": f"Variants Organization {suffix}",
                "address": {
                    "street": "Variants Street 1",
                    "locality": "Rome",
                    "postal_code": "00100",
                    "country": "IT",
                },
                "identifiers": [{"type": "variants", "value": suffix}],
                "parents": [],
            },
        )
        self.assertEqual(response.status_code, 201)
        return response.json()["id"]

    def create_category(self, group_id: str, name: str) -> str:
        response = self.gateway.post(
            "/groups/{groupId}/categories",
            groupId=group_id,
            json={"name": name},
        )
        self.assertEqual(response.status_code, 201)
        return response.json()["id"]

    def versions(self, resource_type: str, resource_id: str) -> list[dict[str, Any]]:
        response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType=resource_type,
            id=resource_id,
        )
        self.assertEqual(response.status_code, 200)
        return response.json()

    def version(
        self, resource_type: str, resource_id: str, status: str
    ) -> dict[str, Any]:
        matches = [
            version
            for version in self.versions(resource_type, resource_id)
            if version["status"] == status
        ]
        self.assertEqual(len(matches), 1)
        return matches[0]

    def details(
        self, resource_type: str, resource_id: str, version_id: str
    ) -> dict[str, Any]:
        response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType=resource_type,
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(response.status_code, 200)
        return response.json()

    def create_and_publish(
        self, group_id: str, resource_type: str, metadata: dict[str, Any]
    ) -> str:
        created = self.gateway.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={"resource_type": resource_type, "metadata": metadata},
        )
        self.assertEqual(created.status_code, 201)
        resource_id = created.json()["id"]
        draft = self.version(resource_type, resource_id, "DRAFT")
        submitted = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType=resource_type,
            id=resource_id,
            versionId=draft["id"],
        )
        self.assertEqual(submitted.status_code, 200)
        published = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/publish",
            resourceType=resource_type,
            id=resource_id,
            versionId=draft["id"],
        )
        self.assertEqual(published.status_code, 200)
        return resource_id

    def assert_identifier_clones(
        self,
        source_metadata: dict[str, Any],
        copy_metadata: dict[str, Any],
        identifier_key: str,
        same_resource: bool = False,
    ) -> None:
        source_identifiers = source_metadata[identifier_key]
        copied_identifiers = copy_metadata[identifier_key]
        self.assertEqual(
            {(item["type"], item["value"]) for item in copied_identifiers},
            {(item["type"], item["value"]) for item in source_identifiers},
        )
        self.assertNotEqual(
            {item["id"] for item in copied_identifiers},
            {item["id"] for item in source_metadata[identifier_key]},
        )
        source_ids = {item["resource_id"] for item in source_identifiers}
        copied_ids = {item["resource_id"] for item in copied_identifiers}
        if same_resource:
            self.assertEqual(copied_ids, source_ids)
        else:
            self.assertTrue(copied_ids.isdisjoint(source_ids))

    def assert_link_ids(
        self,
        source_metadata: dict[str, Any],
        copied_metadata: dict[str, Any],
        key: str,
    ) -> None:
        self.assertEqual(
            [
                item["id"] if isinstance(item, dict) else item
                for item in copied_metadata[key]
            ],
            [
                item["id"] if isinstance(item, dict) else item
                for item in source_metadata[key]
            ],
        )

    def assert_web_service_roles(self, metadata: dict[str, Any]) -> None:
        distribution = next(
            item
            for item in metadata["distributions"]
            if item["title"].startswith("Role metadata")
        )
        self.assertEqual(
            {
                parameter["name"]: parameter.get("role")
                for parameter in distribution["web_service"]["parameters"]
            },
            {
                "start": "START_DATE",
                "west": "MIN_LON",
            },
        )

    def test_publication_draft_clones_web_service_body_parameters(self) -> None:
        """Catch clone assigning body parameters both service and body owners."""

        suffix = uuid4().hex
        group_id = create_group(self.gateway, f"body-parameter-{suffix}")
        category_id = self.create_category(group_id, f"service-{suffix}")
        metadata = {
            "title": f"Dataset {suffix}",
            "description": "Dataset with web-service body parameter",
            "conforms_to": "standard",
            "provenance": "source",
            "type": "GEOSPATIAL",
            "version": "1.0",
            "spatial_resolution_in_meters": 0.25,
            "temporal_resolution": 0,
            "categories": [category_id],
            "distributions": [
                web_distribution(
                    method="POST",
                    body_content_type="text/plain",
                    body_template="{{.payload}}",
                    body_parameters=[child("payload")],
                    keywords=["service"],
                )
            ],
        }
        metadata["distributions"][0]["byte_size"] = 3_000_000_000
        metadata["distributions"][0]["license"] = "license"
        resource_id = self.create_and_publish(group_id, "dataset", metadata)
        published_version = self.version("dataset", resource_id, "PUBLISHED")
        published_details = self.details(
            "dataset", resource_id, published_version["id"]
        )
        source_distribution = published_details["metadata"]["distributions"][0]
        source_service = source_distribution["web_service"]
        source_parameter = source_service["body_parameters"][0]

        drafted = self.gateway.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(drafted.status_code, 201)
        self.assertEqual(drafted.json()["id"], resource_id)
        self.assertEqual(drafted.json()["status"], "DRAFT")

        draft_version = self.version("dataset", resource_id, "DRAFT")
        draft_details = self.details("dataset", resource_id, draft_version["id"])
        draft_distribution = draft_details["metadata"]["distributions"][0]
        draft_service = draft_distribution["web_service"]
        draft_parameter = draft_service["body_parameters"][0]
        for field in (
            "conforms_to",
            "provenance",
            "type",
            "version",
            "spatial_resolution_in_meters",
            "temporal_resolution",
        ):
            self.assertEqual(draft_details["metadata"][field], metadata[field])
        for distribution in (source_distribution, draft_distribution):
            service = distribution["web_service"]
            self.assertEqual(distribution["byte_size"], 3_000_000_000)
            self.assertEqual(distribution["license"], "license")
            self.assertEqual(service["keywords"], ["service"])
            for field in ("access_rights", "license", "categories"):
                self.assertNotIn(field, service)
        self.assertEqual(draft_parameter["name"], source_parameter["name"])
        self.assertNotEqual(draft_parameter["id"], source_parameter["id"])
        self.assertEqual(
            draft_service["body_template"], source_service["body_template"]
        )
        self.assertEqual(
            [category["id"] for category in draft_details["metadata"]["categories"]],
            [category_id],
        )

        copied = self.gateway.post(
            "/groups/{groupId}/resources/{resourceType}/{id}/copy",
            groupId=group_id,
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(copied.status_code, 201)
        copied_id = copied.json()["id"]
        copied_version = self.version("dataset", copied_id, "DRAFT")
        copied_metadata = self.details("dataset", copied_id, copied_version["id"])[
            "metadata"
        ]
        self.assertEqual(copied_metadata["categories"], [])

    def test_resource_variants_and_copy(self) -> None:
        """Verify resource variants, copied ownership, and publication drafts."""

        suffix = uuid4().hex
        source_group_id = create_group(self.gateway, f"variants-source-{suffix}")
        target_group_id = create_group(self.gateway, f"variants-target-{suffix}")
        contact_point_id = self.create_contact_point(suffix)
        organization_id = self.create_organization(suffix)
        source_category_id = self.create_category(source_group_id, f"source-{suffix}")
        target_category_id = self.create_category(target_group_id, f"target-{suffix}")

        editor, _ = self.create_user("editor", suffix)
        source_only, _ = self.create_user("source-only", suffix)
        self.add_membership(editor, source_group_id, "EDITOR")
        self.add_membership(editor, target_group_id, "EDITOR")
        self.add_membership(source_only, source_group_id, "EDITOR")

        dataset_metadata = {
            "title": f"Dataset {suffix}",
            "description": "Representative dataset metadata",
            "keywords": ["variants", "dataset"],
            "geological_temporal_coverage": {
                "start_mya": 251.902,
                "end_mya": 201.4,
            },
            "distributions": [
                {
                    "title": "Dataset download",
                    "description": "Representative download",
                    "type": "DOWNLOADABLE_FILE",
                    "downloadable_file": {
                        "url": "https://example.org/dataset.csv",
                        "format": "text/csv",
                    },
                }
            ],
            "persistent_identifiers": [{"type": "doi", "value": f"10.0000/{suffix}"}],
            "contact_points": [contact_point_id],
            "providers": [organization_id],
            "categories": [source_category_id],
        }
        role_distribution = web_distribution(
            [
                parameter("start", "DATETIME", role="START_DATE"),
                parameter("west", "FLOAT", position=1, role="MIN_LON"),
            ]
        )
        role_distribution["title"] = f"Role metadata {suffix}"
        dataset_metadata["distributions"].append(role_distribution)
        application_metadata = {
            "name": f"Application {suffix}",
            "description": "Representative application metadata",
            "keywords": ["variants", "application"],
            "download_url": "https://example.org/application.zip",
            "main_entity_of_page": "https://example.org/application",
            "creators": [organization_id],
            "identifiers": [{"type": "application", "value": suffix}],
            "contact_points": [contact_point_id],
            "categories": [source_category_id],
        }
        source_code_metadata = {
            "name": f"Source code {suffix}",
            "description": "Representative source-code metadata",
            "keywords": ["variants", "source"],
            "repository": "https://example.org/source.git",
            "programming_language": "Go",
            "main_entity_of_page": "https://example.org/source",
            "creators": [organization_id],
            "identifiers": [{"type": "source", "value": suffix}],
            "contact_points": [contact_point_id],
            "categories": [source_category_id],
        }
        specs = [
            ("dataset", dataset_metadata, "persistent_identifiers", "title"),
            ("software-application", application_metadata, "identifiers", "name"),
            ("software-source-code", source_code_metadata, "identifiers", "name"),
        ]

        source_ids = {
            resource_type: self.create_and_publish(
                source_group_id, resource_type, metadata
            )
            for resource_type, metadata, _, _ in specs
        }

        mismatch = self.gateway.post(
            "/groups/{groupId}/resources",
            groupId=source_group_id,
            json={
                "resource_type": "dataset",
                "metadata": application_metadata,
            },
        )
        self.assertEqual(mismatch.status_code, 400)

        for resource_type, source_metadata, identifier_key, name_key in specs:
            source_id = source_ids[resource_type]
            source_version = self.version(resource_type, source_id, "PUBLISHED")
            source_details = self.details(
                resource_type, source_id, source_version["id"]
            )
            source_result_metadata = source_details["metadata"]
            self.assertEqual(source_result_metadata["resource_type"], resource_type)
            self.assertEqual(
                source_result_metadata[name_key], source_metadata[name_key]
            )

            copied = editor.post(
                "/groups/{groupId}/resources/{resourceType}/{id}/copy",
                groupId=target_group_id,
                resourceType=resource_type,
                id=source_id,
            )
            self.assertEqual(copied.status_code, 201)
            copied_result = copied.json()
            self.assertEqual(copied_result["resourceType"], resource_type)
            self.assertEqual(copied_result["status"], "DRAFT")
            copied_id = copied_result["id"]
            self.assertNotEqual(copied_id, source_id)

            copied_version = self.version(resource_type, copied_id, "DRAFT")
            copied_details = self.details(
                resource_type, copied_id, copied_version["id"]
            )
            copied_metadata = copied_details["metadata"]
            self.assertEqual(copied_details["group_id"], target_group_id)
            self.assertEqual(
                copied_metadata[name_key], source_result_metadata[name_key]
            )
            self.assertEqual(
                copied_metadata["description"], source_result_metadata["description"]
            )
            self.assertEqual(
                copied_metadata["keywords"], source_result_metadata["keywords"]
            )
            self.assertEqual(copied_metadata["categories"], [])
            if resource_type == "dataset":
                self.assert_web_service_roles(source_result_metadata)
                source_download = next(
                    item
                    for item in source_result_metadata["distributions"]
                    if item["title"] == "Dataset download"
                )
                copied_download = next(
                    item
                    for item in copied_metadata["distributions"]
                    if item["title"] == "Dataset download"
                )
                self.assertEqual(
                    copied_download["title"],
                    source_download["title"],
                )
                self.assertEqual(
                    copied_download["downloadable_file"]["url"],
                    source_download["downloadable_file"]["url"],
                )
                self.assertEqual(
                    copied_metadata["geological_temporal_coverage"],
                    source_result_metadata["geological_temporal_coverage"],
                )
                self.assert_web_service_roles(copied_metadata)
            self.assert_identifier_clones(
                source_result_metadata, copied_metadata, identifier_key
            )
            for link_key in ("contact_points", "providers", "creators"):
                if link_key in source_result_metadata:
                    self.assert_link_ids(
                        source_result_metadata, copied_metadata, link_key
                    )

            invalid_category_metadata = copy.deepcopy(source_metadata)
            invalid_category_metadata["categories"] = [source_category_id]
            invalid_category = editor.put(
                "/resources/{id}/versions/{versionId}/draft",
                id=copied_id,
                versionId=copied_version["id"],
                json={
                    "resource_type": resource_type,
                    "revision": copied_version["revision"],
                    "metadata": invalid_category_metadata,
                },
            )
            self.assertEqual(invalid_category.status_code, 400)

            replacement = copy.deepcopy(source_metadata)
            replacement[identifier_key] = [
                {
                    "resource_id": item["resource_id"],
                    "type": item["type"],
                    "value": item["value"],
                }
                for item in copied_metadata[identifier_key]
            ]
            replacement["categories"] = [target_category_id]
            replacement[name_key] = f"{source_metadata[name_key]} replaced"
            replaced = editor.put(
                "/resources/{id}/versions/{versionId}/draft",
                id=copied_id,
                versionId=copied_version["id"],
                json={
                    "resource_type": resource_type,
                    "revision": copied_version["revision"],
                    "metadata": replacement,
                },
            )
            self.assertEqual(replaced.status_code, 200)
            replaced_details = self.details(
                resource_type,
                copied_id,
                self.version(resource_type, copied_id, "DRAFT")["id"],
            )
            self.assertEqual(
                replaced_details["metadata"][name_key], replacement[name_key]
            )
            self.assertEqual(
                replaced_details["metadata"][identifier_key],
                copied_metadata[identifier_key],
            )
            self.assertEqual(
                [item["id"] for item in replaced_details["metadata"]["categories"]],
                [target_category_id],
            )

            drafted = editor.post(
                "/resources/{resourceType}/{id}/draft-from-publication",
                resourceType=resource_type,
                id=source_id,
            )
            self.assertEqual(drafted.status_code, 201)
            self.assertEqual(drafted.json()["id"], source_id)
            self.assertEqual(drafted.json()["status"], "DRAFT")
            publication_draft = self.version(resource_type, source_id, "DRAFT")
            publication_draft_details = self.details(
                resource_type, source_id, publication_draft["id"]
            )
            publication_metadata = publication_draft_details["metadata"]
            self.assertEqual(
                publication_metadata[name_key], source_result_metadata[name_key]
            )
            self.assertEqual(
                publication_metadata["description"],
                source_result_metadata["description"],
            )
            self.assertEqual(
                publication_metadata["keywords"], source_result_metadata["keywords"]
            )
            self.assert_identifier_clones(
                source_result_metadata,
                publication_metadata,
                identifier_key,
                same_resource=True,
            )
            self.assertEqual(
                [item["id"] for item in publication_metadata["categories"]],
                [source_category_id],
            )
            if resource_type == "dataset":
                self.assertEqual(
                    publication_metadata["geological_temporal_coverage"],
                    source_result_metadata["geological_temporal_coverage"],
                )
                self.assert_web_service_roles(publication_metadata)

            unauthorized = source_only.post(
                "/groups/{groupId}/resources/{resourceType}/{id}/copy",
                groupId=target_group_id,
                resourceType=resource_type,
                id=source_id,
            )
            self.assertEqual(unauthorized.status_code, 403)
