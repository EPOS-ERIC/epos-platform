import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token
from client import SCHEMA, GatewayClient


class ResourceLifecycleContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def test_dataset_create_replace_submit_publish(self) -> None:
        """Verify an admin can create, edit, submit, and publish a dataset."""

        groups_response = self.gateway.get("/groups")
        self.assertEqual(groups_response.status_code, 200)
        groups = groups_response.json()
        self.assertIsInstance(groups, list)

        matches = [
            group
            for group in groups
            if isinstance(group, dict) and group.get("name") == "Seismology"
        ]
        self.assertEqual(
            len(matches),
            1,
            "expected exactly one imported Seismology group",
        )
        group = matches[0]
        group_id = group["id"]

        title = f"contract-test-{uuid4()}"
        created = self.gateway.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {"title": title},
            },
        )
        self.assertEqual(created.status_code, 201)
        created_result = created.json()
        self.assertEqual(created_result["resourceType"], "dataset")
        self.assertEqual(created_result["status"], "DRAFT")
        resource_id = created_result["id"]

        def get_version(status: str) -> dict[str, Any]:
            response = self.gateway.get(
                "/resources/{resourceType}/{id}/versions",
                resourceType="dataset",
                id=resource_id,
            )
            self.assertEqual(response.status_code, 200)
            versions = response.json()
            self.assertIsInstance(versions, list)

            matches = [
                version
                for version in versions
                if isinstance(version, dict) and version.get("status") == status
            ]
            self.assertEqual(
                len(matches),
                1,
                f"expected exactly one {status} version",
            )
            return matches[0]

        draft = get_version("DRAFT")
        self.assertEqual(draft["resource_id"], resource_id)
        version_id = draft["version_id"]
        initial_revision = draft["revision"]

        draft_details_response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(draft_details_response.status_code, 200)
        draft_details = draft_details_response.json()
        self.assertEqual(draft_details["resource_id"], resource_id)
        self.assertEqual(draft_details["version_id"], version_id)
        self.assertEqual(draft_details["status"], "DRAFT")
        self.assertEqual(draft_details["group_id"], group_id)
        self.assertEqual(draft_details["metadata"]["resource_type"], "dataset")
        self.assertEqual(draft_details["metadata"]["title"], title)

        updated_title = f"{title}-updated"
        replaced = self.gateway.put(
            "/resources/{id}/versions/{versionId}/draft",
            id=resource_id,
            versionId=version_id,
            json={
                "resource_type": "dataset",
                "revision": initial_revision,
                "metadata": {"title": updated_title},
            },
        )
        self.assertEqual(replaced.status_code, 200)
        replaced_result = replaced.json()
        self.assertEqual(replaced_result["resourceType"], "dataset")
        self.assertEqual(replaced_result["id"], resource_id)
        self.assertEqual(replaced_result["status"], "DRAFT")

        draft = get_version("DRAFT")
        self.assertEqual(draft["resource_id"], resource_id)
        self.assertEqual(draft["revision"], initial_revision + 1)
        version_id = draft["version_id"]

        stale = self.gateway.put(
            "/resources/{id}/versions/{versionId}/draft",
            id=resource_id,
            versionId=version_id,
            json={
                "resource_type": "dataset",
                "revision": initial_revision,
                "metadata": {"title": updated_title},
            },
        )
        self.assertEqual(stale.status_code, 409)
        problem = stale.json()
        self.assertEqual(problem["title"], "conflict")
        self.assertEqual(
            problem["detail"],
            "operation conflicts with existing data",
        )

        submitted = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(submitted.status_code, 200)
        submitted_result = submitted.json()
        self.assertEqual(submitted_result["resourceType"], "dataset")
        self.assertEqual(submitted_result["id"], resource_id)
        self.assertEqual(submitted_result["status"], "SUBMITTED")

        submitted_version = get_version("SUBMITTED")
        self.assertEqual(submitted_version["resource_id"], resource_id)
        self.assertEqual(submitted_version["version_id"], version_id)
        self.assertTrue(submitted_version["submitted_by"])
        self.assertIsNotNone(submitted_version["submitted_at"])

        published = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/publish",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(published.status_code, 200)
        published_result = published.json()
        self.assertEqual(published_result["resourceType"], "dataset")
        self.assertEqual(published_result["id"], resource_id)
        self.assertEqual(published_result["status"], "PUBLISHED")

        published_version = get_version("PUBLISHED")
        self.assertEqual(published_version["resource_id"], resource_id)
        self.assertEqual(published_version["version_id"], version_id)
        self.assertTrue(published_version["published_by"])
        self.assertIsNotNone(published_version["published_at"])

        published_details_response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(published_details_response.status_code, 200)
        published_details = published_details_response.json()
        self.assertEqual(published_details["resource_id"], resource_id)
        self.assertEqual(published_details["version_id"], version_id)
        self.assertEqual(published_details["status"], "PUBLISHED")
        self.assertEqual(published_details["group_id"], group_id)
        self.assertEqual(published_details["metadata"]["resource_type"], "dataset")
        self.assertEqual(published_details["metadata"]["title"], updated_title)
