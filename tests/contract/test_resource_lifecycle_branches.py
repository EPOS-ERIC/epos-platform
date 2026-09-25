import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import create_group


class ResourceLifecycleBranchesContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def create_user(self, label: str, suffix: str) -> tuple[GatewayClient, str]:
        aai_url = os.getenv("AAI_URL", "http://localhost:8110")
        email = f"rl-{uuid4().hex}@example.org"
        password = uuid4().hex
        credentials = register_user(
            aai_url,
            name="Lifecycle",
            surname=f"{label.title()} {suffix}",
            email=email,
            password=password,
        )
        client = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(
                aai_url=aai_url,
                username=credentials.username,
                password=credentials.password,
            ),
        )
        response = client.get("/users/{id}", id="self")
        self.assertEqual(response.status_code, 200)
        return client, response.json()["id"]

    def accept_membership(self, client: GatewayClient, group_id: str) -> str:
        request = client.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json={"roles": []},
        )
        self.assertEqual(request.status_code, 201)
        user_id = request.json()["user"]["id"]
        response = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="accept",
        )
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["status"], "ACTIVE")
        return user_id

    def assign_role(self, group_id: str, user_id: str, role: str) -> None:
        response = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=user_id,
            role=role,
        )
        self.assertEqual(response.status_code, 204)

    def get_version(self, resource_id: str, status: str) -> dict[str, Any]:
        response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(response.status_code, 200)
        matches = [version for version in response.json() if version["status"] == status]
        self.assertEqual(len(matches), 1)
        return matches[0]

    def assert_conflict(self, response: Any) -> None:
        self.assertEqual(response.status_code, 409)
        problem = response.json()
        self.assertEqual(problem["title"], "conflict")
        self.assertEqual(problem["detail"], "operation conflicts with existing data")

    def test_resource_lifecycle_branches(self) -> None:
        """Verify lifecycle branches, version invariants, and discarded changes."""

        suffix = str(uuid4())
        group_id = create_group(self.gateway, f"contract-lifecycle-branches-{suffix}")
        editor, editor_id = self.create_user("editor", suffix)
        reviewer, reviewer_id = self.create_user("reviewer", suffix)
        self.accept_membership(editor, group_id)
        self.accept_membership(reviewer, group_id)
        self.assign_role(group_id, editor_id, "EDITOR")
        self.assign_role(group_id, reviewer_id, "REVIEWER")
        unpublish_permission = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=reviewer_id,
            permission="group.resource.unpublish",
        )
        self.assertEqual(unpublish_permission.status_code, 204)

        title = f"contract-lifecycle-branches-{suffix}"
        created = editor.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {
                    "title": title,
                    "description": "Lifecycle branch description",
                },
            },
        )
        self.assertEqual(created.status_code, 201)
        resource_id = created.json()["id"]
        draft = self.get_version(resource_id, "DRAFT")
        version_id = draft["version_id"]
        self.assertEqual(draft["resource_id"], resource_id)
        self.assertEqual(draft["revision"], 1)

        draft_details = editor.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(draft_details.status_code, 200)
        self.assertEqual(draft_details.json()["group_id"], group_id)
        self.assertEqual(draft_details.json()["metadata"]["title"], title)

        updated_title = f"{title}-updated"
        replace_payload = {
            "resource_type": "dataset",
            "revision": draft["revision"],
            "metadata": {
                "title": updated_title,
                "description": "Lifecycle branch description",
                "keywords": ["lifecycle", "branches"],
            },
        }
        replaced = editor.put(
            "/resources/{id}/versions/{versionId}/draft",
            id=resource_id,
            versionId=version_id,
            json=replace_payload,
        )
        self.assertEqual(replaced.status_code, 200)
        draft = self.get_version(resource_id, "DRAFT")
        self.assertEqual(draft["revision"], 2)
        version_id = draft["version_id"]
        draft_details = editor.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        ).json()
        self.assertEqual(draft_details["metadata"]["title"], updated_title)
        self.assertEqual(
            draft_details["metadata"]["description"],
            "Lifecycle branch description",
        )
        self.assertEqual(
            draft_details["metadata"]["keywords"],
            ["lifecycle", "branches"],
        )

        submitted = editor.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(submitted.status_code, 200)
        submitted_version = self.get_version(resource_id, "SUBMITTED")
        self.assertEqual(submitted_version["version_id"], version_id)
        self.assertEqual(submitted_version["submitted_by"], editor_id)
        self.assertIsNotNone(submitted_version["submitted_at"])

        withdrawn = editor.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/withdraw",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(withdrawn.status_code, 200)
        self.assertEqual(self.get_version(resource_id, "DRAFT")["version_id"], version_id)

        self.assertEqual(
            editor.post(
                "/resources/{resourceType}/{id}/versions/{versionId}/submit",
                resourceType="dataset",
                id=resource_id,
                versionId=version_id,
            ).status_code,
            200,
        )
        rejection_note = "Please add the missing provenance."
        rejected = reviewer.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/reject",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
            json={"note": rejection_note},
        )
        self.assertEqual(rejected.status_code, 200)
        rejected_version = self.get_version(resource_id, "REJECTED")
        self.assertEqual(rejected_version["version_id"], version_id)
        self.assertEqual(rejected_version["rejected_by"], reviewer_id)
        self.assertIsNotNone(rejected_version["rejected_at"])
        self.assertEqual(rejected_version["rejection_note"], rejection_note)

        reopened = editor.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/reopen",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(reopened.status_code, 200)
        reopened_version = self.get_version(resource_id, "DRAFT")
        self.assertEqual(reopened_version["version_id"], version_id)
        self.assertEqual(reopened_version["rejection_note"], rejection_note)

        self.assertEqual(
            editor.post(
                "/resources/{resourceType}/{id}/versions/{versionId}/submit",
                resourceType="dataset",
                id=resource_id,
                versionId=version_id,
            ).status_code,
            200,
        )
        published = reviewer.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/publish",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(published.status_code, 200)
        published_version = self.get_version(resource_id, "PUBLISHED")
        self.assertEqual(published_version["version_id"], version_id)
        self.assertEqual(published_version["published_by"], reviewer_id)
        self.assertIsNotNone(published_version["published_at"])
        self.assertIsNone(published_version["rejected_by"])
        self.assertIsNone(published_version["rejected_at"])
        self.assertIsNone(published_version["rejection_note"])

        immutable_edit = editor.put(
            "/resources/{id}/versions/{versionId}/draft",
            id=resource_id,
            versionId=version_id,
            json={
                "resource_type": "dataset",
                "revision": published_version["revision"],
                "metadata": {
                    "title": "must remain immutable",
                    "description": "Immutable edit description",
                },
            },
        )
        self.assert_conflict(immutable_edit)

        from_publication = editor.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(from_publication.status_code, 201)
        self.assertEqual(from_publication.json()["id"], resource_id)
        self.assertEqual(from_publication.json()["status"], "DRAFT")
        publication_draft = self.get_version(resource_id, "DRAFT")
        self.assertNotEqual(publication_draft["version_id"], version_id)
        self.assertEqual(publication_draft["revision"], 1)
        publication_draft_details = editor.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=publication_draft["version_id"],
        ).json()
        self.assertEqual(
            publication_draft_details["metadata"]["title"],
            updated_title,
        )
        self.assertEqual(
            publication_draft_details["metadata"]["description"],
            "Lifecycle branch description",
        )
        repeated_publication_draft = editor.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset",
            id=resource_id,
        )
        self.assert_conflict(repeated_publication_draft)
        discarded_publication_draft = editor.delete(
            "/resources/{resourceType}/{id}/versions/{versionId}/draft",
            resourceType="dataset",
            id=resource_id,
            versionId=publication_draft["version_id"],
        )
        self.assertEqual(discarded_publication_draft.status_code, 204)

        unpublished = reviewer.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/unpublish",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(unpublished.status_code, 200)
        archived_version = self.get_version(resource_id, "ARCHIVED")
        self.assertEqual(archived_version["version_id"], version_id)
        self.assertEqual(archived_version["archived_by"], reviewer_id)
        self.assertIsNotNone(archived_version["archived_at"])

        from_archive = editor.post(
            "/resources/{resourceType}/{id}/draft-from-archive",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(from_archive.status_code, 201)
        self.assertEqual(from_archive.json()["id"], resource_id)
        self.assertEqual(from_archive.json()["status"], "DRAFT")
        archive_draft = self.get_version(resource_id, "DRAFT")
        self.assertNotEqual(archive_draft["version_id"], version_id)
        archive_draft_details = editor.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=archive_draft["version_id"],
        ).json()
        self.assertEqual(archive_draft_details["metadata"]["title"], updated_title)

        rejected_resource = editor.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {
                    "title": f"rejected-discard-{suffix}",
                    "description": "Rejected discard description",
                },
            },
        )
        self.assertEqual(rejected_resource.status_code, 201)
        rejected_id = rejected_resource.json()["id"]
        rejected_version_id = self.get_version(rejected_id, "DRAFT")["version_id"]
        self.assertEqual(
            editor.post(
                "/resources/{resourceType}/{id}/versions/{versionId}/submit",
                resourceType="dataset",
                id=rejected_id,
                versionId=rejected_version_id,
            ).status_code,
            200,
        )
        self.assertEqual(
            reviewer.post(
                "/resources/{resourceType}/{id}/versions/{versionId}/reject",
                resourceType="dataset",
                id=rejected_id,
                versionId=rejected_version_id,
                json={"note": "discard rejected change"},
            ).status_code,
            200,
        )
        discarded_rejected = editor.delete(
            "/resources/{resourceType}/{id}/versions/{versionId}/draft",
            resourceType="dataset",
            id=rejected_id,
            versionId=rejected_version_id,
        )
        self.assertEqual(discarded_rejected.status_code, 204)
        deleted_rejected = editor.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=rejected_id,
            versionId=rejected_version_id,
        )
        self.assertEqual(deleted_rejected.status_code, 404)

        draft_resource = editor.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {
                    "title": f"draft-discard-{suffix}",
                    "description": "Draft discard description",
                },
            },
        )
        self.assertEqual(draft_resource.status_code, 201)
        draft_resource_id = draft_resource.json()["id"]
        draft_resource_version_id = self.get_version(draft_resource_id, "DRAFT")["version_id"]
        discarded_draft = editor.delete(
            "/resources/{resourceType}/{id}/versions/{versionId}/draft",
            resourceType="dataset",
            id=draft_resource_id,
            versionId=draft_resource_version_id,
        )
        self.assertEqual(discarded_draft.status_code, 204)
        deleted_draft = editor.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=draft_resource_id,
            versionId=draft_resource_version_id,
        )
        self.assertEqual(deleted_draft.status_code, 404)
