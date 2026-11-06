import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, create_group


class CanonicalReferenceIntegrityContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def create_contact_point(self, suffix: str) -> str:
        response = self.gateway.post(
            "/contact-points",
            json={
                "email": f"canonical-integrity-{suffix}@example.org",
            },
        )
        self.assertEqual(response.status_code, 201)
        return response.json()["id"]

    def organization_payload(
        self,
        suffix: str,
        *,
        parents: list[str],
        contact_point_ids: list[str],
    ) -> dict[str, Any]:
        return {
            "name": f"Canonical Organization {suffix}",
            "address": {
                "street": "Canonical Street 1",
                "locality": "Rome",
                "postal_code": "00100",
                "country": "IT",
            },
            "logo_url": None,
            "page_url": None,
            "email": None,
            "lei_code": None,
            "identifiers": [{"type": "canonical", "value": suffix}],
            "parents": parents,
            "contact_point_ids": contact_point_ids,
        }

    def create_organization(
        self,
        suffix: str,
        *,
        parents: list[str],
        contact_point_ids: list[str],
    ) -> str:
        payload = self.organization_payload(
            suffix,
            parents=parents,
            contact_point_ids=contact_point_ids,
        )
        response = self.gateway.post("/organizations", json=payload)
        self.assertEqual(response.status_code, 201)
        return response.json()["id"]

    def create_category(self, group_id: str, name: str, parent_id: str | None = None) -> str:
        payload: dict[str, Any] = {"name": name}
        if parent_id is not None:
            payload["parent_category_id"] = parent_id
        response = self.gateway.post(
            "/groups/{groupId}/categories",
            groupId=group_id,
            json=payload,
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

    def version(self, resource_type: str, resource_id: str, status: str) -> dict[str, Any]:
        matches = [
            version
            for version in self.versions(resource_type, resource_id)
            if version["status"] == status
        ]
        self.assertEqual(len(matches), 1)
        return matches[0]

    def details(
        self,
        resource_type: str,
        resource_id: str,
        version_id: str,
    ) -> dict[str, Any]:
        response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType=resource_type,
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(response.status_code, 200)
        return response.json()

    def create_published_and_archived_versions(
        self,
        group_id: str,
        resource_type: str,
        metadata: dict[str, Any],
    ) -> tuple[str, dict[str, Any], dict[str, Any]]:
        created = self.gateway.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={"resource_type": resource_type, "metadata": metadata},
        )
        self.assertEqual(created.status_code, 201)
        resource_id = created.json()["id"]

        first_draft = self.version(resource_type, resource_id, "DRAFT")
        submitted = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType=resource_type,
            id=resource_id,
            versionId=first_draft["id"],
        )
        self.assertEqual(submitted.status_code, 200)
        published = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/publish",
            resourceType=resource_type,
            id=resource_id,
            versionId=first_draft["id"],
        )
        self.assertEqual(published.status_code, 200)

        drafted = self.gateway.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType=resource_type,
            id=resource_id,
        )
        self.assertEqual(drafted.status_code, 201)
        second_draft = self.version(resource_type, resource_id, "DRAFT")
        submitted = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType=resource_type,
            id=resource_id,
            versionId=second_draft["id"],
        )
        self.assertEqual(submitted.status_code, 200)
        published = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/publish",
            resourceType=resource_type,
            id=resource_id,
            versionId=second_draft["id"],
        )
        self.assertEqual(published.status_code, 200)

        return (
            resource_id,
            self.version(resource_type, resource_id, "PUBLISHED"),
            self.version(resource_type, resource_id, "ARCHIVED"),
        )

    def resource_metadata(
        self,
        suffix: str,
        organization_id: str,
        contact_point_id: str,
        category_id: str,
    ) -> dict[str, dict[str, Any]]:
        return {
            "dataset": {
                "title": f"Canonical Dataset {suffix}",
                "description": "Canonical reference integrity dataset",
                "keywords": ["canonical", "dataset"],
                "distributions": [
                    {
                        "title": "Canonical service",
                        "description": "Canonical reference integrity service",
                        "type": "WEB_SERVICE",
                        "web_service": {
                            "configuration": {
                                "type": "WMS",
                                "capabilities_url": "https://example.org/canonical",
                                "default_layer": "canonical",
                                "layers": [{"name": "canonical", "default_style": "", "styles": [""]}],
                                "transparency_editable": False,
                                "default_transparency": False,
                                "get_map_parameters": [],
                                "get_feature_info_parameters": [],
                            },
                            "provider": organization_id,
                            "contact_points": [contact_point_id],
                        },
                    }
                ],
                "persistent_identifiers": [
                    {"type": "dataset", "value": f"dataset-{suffix}"}
                ],
                "contact_points": [contact_point_id],
                "providers": [organization_id],
                "categories": [category_id],
            },
            "software-application": {
                "name": f"Canonical Application {suffix}",
                "description": "Canonical reference integrity application",
                "keywords": ["canonical", "application"],
                "download_url": "https://example.org/canonical-application.zip",
                "main_entity_of_page": "https://example.org/canonical-application",
                "creators": [organization_id],
                "identifiers": [
                    {"type": "application", "value": f"application-{suffix}"}
                ],
                "contact_points": [contact_point_id],
                "categories": [category_id],
            },
            "software-source-code": {
                "name": f"Canonical Source {suffix}",
                "description": "Canonical reference integrity source code",
                "keywords": ["canonical", "source"],
                "repository": "https://example.org/canonical-source.git",
                "programming_language": "Go",
                "main_entity_of_page": "https://example.org/canonical-source",
                "creators": [organization_id],
                "identifiers": [{"type": "source", "value": f"source-{suffix}"}],
                "contact_points": [contact_point_id],
                "categories": [category_id],
            },
        }

    def assert_reference_presence(
        self,
        resource_type: str,
        details: dict[str, Any],
        organization_id: str,
        contact_point_id: str,
        category_id: str,
    ) -> None:
        metadata = details["metadata"]
        self.assertEqual(metadata["contact_points"], [contact_point_id])
        self.assertEqual([item["id"] for item in metadata["categories"]], [category_id])
        if resource_type == "dataset":
            self.assertEqual([item["id"] for item in metadata["providers"]], [organization_id])
            web_service = metadata["distributions"][0]["web_service"]
            self.assertEqual(web_service["provider"]["id"], organization_id)
            self.assertEqual(web_service["contact_points"], [contact_point_id])
        else:
            self.assertEqual([item["id"] for item in metadata["creators"]], [organization_id])

    def delete_problem(self, response: Any) -> None:
        assert_problem(self, response, 409, "conflict")

    def test_canonical_deletion_unlinks_all_resource_references(self) -> None:
        """Verify canonical deletion unlinks published and archived references."""

        suffix = uuid4().hex
        group_id = create_group(self.gateway, f"canonical-integrity-{suffix}")
        contact_point_id = self.create_contact_point(suffix)
        parent_id = self.create_organization(
            f"parent-{suffix}", parents=[], contact_point_ids=[contact_point_id]
        )
        child_id = self.create_organization(
            f"child-{suffix}", parents=[parent_id], contact_point_ids=[contact_point_id]
        )
        root_category_id = self.create_category(group_id, f"root-{suffix}")
        child_category_id = self.create_category(
            group_id, f"child-{suffix}", root_category_id
        )
        metadata_by_type = self.resource_metadata(
            suffix, parent_id, contact_point_id, child_category_id
        )

        resources = {
            resource_type: self.create_published_and_archived_versions(
                group_id, resource_type, metadata
            )
            for resource_type, metadata in metadata_by_type.items()
        }
        identifiers_by_version: dict[tuple[str, str], list[dict[str, Any]]] = {}
        for resource_type, (resource_id, published, archived) in resources.items():
            for version in (published, archived):
                details = self.details(resource_type, resource_id, version["id"])
                self.assert_reference_presence(
                    resource_type,
                    details,
                    parent_id,
                    contact_point_id,
                    child_category_id,
                )
                identifier_key = (
                    "persistent_identifiers"
                    if resource_type == "dataset"
                    else "identifiers"
                )
                identifiers_by_version[(resource_type, version["id"])] = details[
                    "metadata"
                ][identifier_key]

        deleted_organization = self.gateway.delete(
            "/organizations/{organizationId}", organizationId=parent_id
        )
        self.assertEqual(deleted_organization.status_code, 204)
        self.assertEqual(
            self.gateway.get(
                "/organizations/{organizationId}", organizationId=parent_id
            ).status_code,
            404,
        )
        child = self.gateway.get(
            "/organizations/{organizationId}", organizationId=child_id
        )
        self.assertEqual(child.status_code, 200)
        self.assertEqual(child.json()["parents"], [])
        for resource_type, (resource_id, published, archived) in resources.items():
            for version in (published, archived):
                details = self.details(resource_type, resource_id, version["id"])
                metadata = details["metadata"]
                self.assertEqual(metadata["contact_points"], [contact_point_id])
                if resource_type == "dataset":
                    self.assertEqual(metadata["providers"], [])
                    self.assertNotIn("provider", metadata["distributions"][0]["web_service"])
                else:
                    self.assertEqual(metadata["creators"], [])
                identifier_key = (
                    "persistent_identifiers"
                    if resource_type == "dataset"
                    else "identifiers"
                )
                self.assertEqual(
                    metadata[identifier_key],
                    identifiers_by_version[(resource_type, version["id"])],
                )

        deleted_contact_point = self.gateway.delete(
            "/contact-points/{contactPointId}", contactPointId=contact_point_id
        )
        self.assertEqual(deleted_contact_point.status_code, 204)
        self.assertEqual(
            self.gateway.get(
                "/contact-points/{contactPointId}", contactPointId=contact_point_id
            ).status_code,
            404,
        )
        child = self.gateway.get(
            "/organizations/{organizationId}", organizationId=child_id
        )
        self.assertEqual(child.status_code, 200)
        self.assertEqual(child.json()["contact_point_ids"], [])
        for resource_type, (resource_id, published, archived) in resources.items():
            for version in (published, archived):
                details = self.details(resource_type, resource_id, version["id"])
                self.assertFalse(details["metadata"]["contact_points"])
                if resource_type == "dataset":
                    self.assertEqual(
                        details["metadata"]["distributions"][0]["web_service"]["contact_points"],
                        [],
                    )

        deleted_category = self.gateway.delete(
            "/groups/{groupId}/categories/{categoryId}",
            groupId=group_id,
            categoryId=root_category_id,
        )
        self.assertEqual(deleted_category.status_code, 204)
        for category_id in (root_category_id, child_category_id):
            self.assertEqual(
                self.gateway.get(
                    "/groups/{groupId}/categories/{categoryId}",
                    groupId=group_id,
                    categoryId=category_id,
                ).status_code,
                404,
            )
        for resource_type, (resource_id, published, archived) in resources.items():
            for version in (published, archived):
                details = self.details(resource_type, resource_id, version["id"])
                self.assertEqual(details["metadata"]["categories"], [])

        self.delete_problem(self.gateway.delete("/groups/{groupId}", groupId=group_id))

    def test_hierarchy_self_links_and_cycles_are_invalid(self) -> None:
        """Verify hierarchy self-links and cycles return HTTP 400."""

        suffix = uuid4().hex
        group_id = create_group(self.gateway, f"canonical-integrity-cycles-{suffix}")
        root_category_id = self.create_category(group_id, f"root-{suffix}")
        child_category_id = self.create_category(
            group_id, f"child-{suffix}", root_category_id
        )

        category_self_link = self.gateway.put(
            "/groups/{groupId}/categories/{categoryId}",
            groupId=group_id,
            categoryId=root_category_id,
            json={"name": f"root-{suffix}", "parent_category_id": root_category_id},
        )
        self.assertEqual(category_self_link.status_code, 400)
        category_cycle = self.gateway.put(
            "/groups/{groupId}/categories/{categoryId}",
            groupId=group_id,
            categoryId=root_category_id,
            json={"name": f"root-{suffix}", "parent_category_id": child_category_id},
        )
        self.assertEqual(category_cycle.status_code, 400)
        root = self.gateway.get(
            "/groups/{groupId}/categories/{categoryId}",
            groupId=group_id,
            categoryId=root_category_id,
        )
        self.assertEqual(root.status_code, 200)
        self.assertIsNone(root.json()["parent_category_id"])

        standalone_id = self.create_organization(
            f"standalone-{suffix}", parents=[], contact_point_ids=[]
        )
        standalone_payload = self.organization_payload(
            f"standalone-{suffix}", parents=[standalone_id], contact_point_ids=[]
        )
        organization_self_link = self.gateway.put(
            "/organizations/{organizationId}",
            organizationId=standalone_id,
            json=standalone_payload,
        )
        self.assertEqual(organization_self_link.status_code, 400)

        parent_id = self.create_organization(
            f"parent-{suffix}", parents=[], contact_point_ids=[]
        )
        child_id = self.create_organization(
            f"child-{suffix}", parents=[parent_id], contact_point_ids=[]
        )
        parent_payload = self.organization_payload(
            f"parent-{suffix}", parents=[child_id], contact_point_ids=[]
        )
        organization_cycle = self.gateway.put(
            "/organizations/{organizationId}",
            organizationId=parent_id,
            json=parent_payload,
        )
        self.assertEqual(organization_cycle.status_code, 400)
        parent = self.gateway.get(
            "/organizations/{organizationId}", organizationId=parent_id
        )
        self.assertEqual(parent.status_code, 200)
        self.assertEqual(parent.json()["parents"], [])
        child = self.gateway.get(
            "/organizations/{organizationId}", organizationId=child_id
        )
        self.assertEqual(child.status_code, 200)
        self.assertEqual(child.json()["parents"], [parent_id])
