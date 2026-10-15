import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, create_group


NO_AUTH_HEADERS = {"Authorization": ""}


class AuthorizationEdgeCasesContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def create_user(self, suffix: str) -> tuple[GatewayClient, str]:
        credentials = register_user(
            name="Authorization",
            surname=f"Edge {suffix}",
            email=f"authorization-edge-{uuid4().hex}@example.org",
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
        profile = client.get("/users/{id}", id="self")
        self.assertEqual(profile.status_code, 200)
        return client, profile.json()["id"]

    def accept_membership(self, client: GatewayClient, group_id: str) -> str:
        requested = client.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json={"roles": []},
        )
        self.assertEqual(requested.status_code, 201)
        user_id = requested.json()["user"]["id"]

        accepted = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="accept",
        )
        self.assertEqual(accepted.status_code, 200)
        self.assertEqual(accepted.json()["status"], "ACTIVE")
        return user_id

    def get_profile(self, client: GatewayClient, user_id: str) -> dict[str, Any]:
        response = client.get("/users/{id}", id=user_id)
        self.assertEqual(response.status_code, 200)
        return response.json()

    def get_membership(self, profile: dict[str, Any], group_id: str) -> dict[str, Any]:
        matches = [
            membership
            for membership in profile["memberships"]
            if membership["group"]["id"] == group_id
        ]
        self.assertEqual(len(matches), 1)
        return matches[0]

    def create_draft(self, client: GatewayClient, group_id: str, suffix: str) -> tuple[str, str, str]:
        title = f"authorization-edge-draft-{suffix}"
        created = client.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {
                    "title": title,
                    "description": f"Authorization edge draft {suffix}",
                },
            },
        )
        self.assertEqual(created.status_code, 201)
        resource_id = created.json()["id"]

        versions = client.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(versions.status_code, 200)
        drafts = [
            version
            for version in versions.json()
            if version["status"] == "DRAFT"
        ]
        self.assertEqual(len(drafts), 1)
        return title, resource_id, drafts[0]["version_id"]

    def assert_forbidden(self, response: Any) -> None:
        assert_problem(self, response, 403, "permission_denied")

    def assert_conflict(self, response: Any) -> None:
        assert_problem(self, response, 409, "conflict")

    def assert_active_access(
        self,
        client: GatewayClient,
        group_id: str,
        resource_id: str,
        version_id: str,
        draft_title: str,
    ) -> None:
        profile = self.get_profile(client, "self")
        self.assertEqual(profile["status"], "ACTIVE")
        membership = self.get_membership(profile, group_id)
        self.assertEqual(membership["status"], "ACTIVE")
        self.assertEqual(membership["roles"], ["EDITOR"])
        self.assertEqual(membership["granted_permissions"], ["group.category.manage"])

        contact_points = client.get("/contact-points")
        self.assertEqual(contact_points.status_code, 200)

        search = client.get(
            "/search",
            params={"q": draft_title, "versioningStatus": "DRAFT"},
        )
        self.assertEqual(search.status_code, 200)
        self.assertIn(resource_id, {item["resource_id"] for item in search.json()})

        versions = client.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(versions.status_code, 200)
        self.assertIn(version_id, {item["version_id"] for item in versions.json()})

        version = client.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(version.status_code, 200)
        self.assertEqual(version.json()["resource_id"], resource_id)
        self.assertEqual(version.json()["version_id"], version_id)
        self.assertEqual(version.json()["status"], "DRAFT")

    def test_blocked_user_access_restores_and_last_admin_is_protected(self) -> None:
        """Verify blocked access becomes dormant and sole-admin changes conflict."""

        suffix = uuid4().hex
        group_id = create_group(self.gateway, f"authorization-edge-{suffix}")
        ordinary, ordinary_id = self.create_user(suffix)
        accepted_id = self.accept_membership(ordinary, group_id)
        self.assertEqual(accepted_id, ordinary_id)

        assigned_role = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=ordinary_id,
            role="EDITOR",
        )
        self.assertEqual(assigned_role.status_code, 204)
        granted_permission = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=ordinary_id,
            permission="group.category.manage",
        )
        self.assertEqual(granted_permission.status_code, 204)

        draft_title, resource_id, version_id = self.create_draft(
            ordinary,
            group_id,
            suffix,
        )
        self.assert_active_access(
            ordinary,
            group_id,
            resource_id,
            version_id,
            draft_title,
        )

        block = self.gateway.post(
            "/users/{id}/block",
            id=ordinary_id,
            json={"reason": "authorization edge contract test"},
        )
        self.assertEqual(block.status_code, 204)

        blocked_profile = self.get_profile(self.gateway, ordinary_id)
        self.assertEqual(blocked_profile["status"], "BLOCKED")
        blocked_membership = self.get_membership(blocked_profile, group_id)
        self.assertEqual(blocked_membership["status"], "ACTIVE")
        self.assertEqual(blocked_membership["roles"], ["EDITOR"])
        self.assertEqual(
            blocked_membership["granted_permissions"],
            ["group.category.manage"],
        )

        self.assert_forbidden(ordinary.get("/contact-points"))
        self.assert_forbidden(
            ordinary.get(
                "/search",
                params={"q": draft_title, "versioningStatus": "DRAFT"},
            )
        )
        self.assert_forbidden(
            ordinary.get(
                "/resources/{resourceType}/{id}/versions",
                resourceType="dataset",
                id=resource_id,
            )
        )
        self.assert_forbidden(
            ordinary.get(
                "/resources/{resourceType}/{id}/versions/{versionId}",
                resourceType="dataset",
                id=resource_id,
                versionId=version_id,
            )
        )

        blocked_group = ordinary.get("/groups/{groupId}", groupId=group_id)
        anonymous_group = ordinary.get(
            "/groups/{groupId}",
            groupId=group_id,
            headers=NO_AUTH_HEADERS,
        )
        for response in (blocked_group, anonymous_group):
            self.assertEqual(response.status_code, 200)
            self.assertEqual(response.json()["id"], group_id)

        blocked_organizations = ordinary.get("/organizations")
        anonymous_organizations = ordinary.get(
            "/organizations",
            headers=NO_AUTH_HEADERS,
        )
        for response in (blocked_organizations, anonymous_organizations):
            self.assertEqual(response.status_code, 200)
            self.assertIsInstance(response.json(), list)

        unblock = self.gateway.post(
            "/users/{id}/unblock",
            id=ordinary_id,
        )
        self.assertEqual(unblock.status_code, 204)
        self.assert_active_access(
            ordinary,
            group_id,
            resource_id,
            version_id,
            draft_title,
        )

        admin_profile = self.get_profile(self.gateway, "self")
        admin_id = admin_profile["id"]
        self.assertEqual(admin_profile["status"], "ACTIVE")
        self.assertEqual(admin_profile["global_roles"], ["ADMIN"])

        self.assert_conflict(
            self.gateway.post(
                "/users/{id}/block",
                id=admin_id,
                json={"reason": "sole-admin protection contract test"},
            )
        )
        self.assert_conflict(
            self.gateway.delete(
                "/users/{id}/global-roles/{role}",
                id=admin_id,
                role="ADMIN",
            )
        )

        final_admin_profile = self.get_profile(self.gateway, "self")
        self.assertEqual(final_admin_profile["status"], "ACTIVE")
        self.assertEqual(final_admin_profile["global_roles"], ["ADMIN"])
