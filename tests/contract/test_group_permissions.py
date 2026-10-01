import os
import unittest
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, create_group


class GroupPermissionContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def create_user(self, label: str, suffix: str) -> GatewayClient:
        aai_url = os.getenv("AAI_URL", "http://localhost:8110")
        email = f"contract-{uuid4().hex}@example.org"
        password = uuid4().hex
        credentials = register_user(
            aai_url,
            name="Contract",
            surname=f"{label.title()} {suffix}",
            email=email,
            password=password,
        )
        return GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(
                aai_url=aai_url,
                username=credentials.username,
                password=credentials.password,
            ),
        )

    def accept_membership(self, client: GatewayClient, group_id: str) -> str:
        request_response = client.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json={"roles": []},
        )
        self.assertEqual(request_response.status_code, 201)
        membership = request_response.json()
        user_id = membership["user"]["id"]

        accepted_response = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="accept",
        )
        self.assertEqual(accepted_response.status_code, 200)
        accepted_membership = accepted_response.json()
        self.assertEqual(accepted_membership["user"]["id"], user_id)
        self.assertEqual(accepted_membership["status"], "ACTIVE")
        return user_id

    def get_own_membership(
        self, client: GatewayClient, group_id: str
    ) -> tuple[dict, dict]:
        response = client.get(
            "/users/{id}",
            id="self",
        )
        self.assertEqual(response.status_code, 200)
        user = response.json()
        memberships = [
            membership
            for membership in user["memberships"]
            if membership["group"]["id"] == group_id
        ]
        self.assertEqual(len(memberships), 1)
        return user, memberships[0]

    def test_direct_permission_grant_and_revoke(self) -> None:
        """Verify direct group permissions are granted, read back, and revoked."""

        suffix = str(uuid4())
        group_id = create_group(self.gateway, f"contract-permission-group-{suffix}")
        manager = self.create_user("permission-manager", suffix)
        target = self.create_user("permission-target", suffix)
        manager_id = self.accept_membership(manager, group_id)
        target_id = self.accept_membership(target, group_id)

        grant_manager_permission_response = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=manager_id,
            permission="group.permission.manage",
        )
        self.assertEqual(grant_manager_permission_response.status_code, 204)

        manager_user, manager_membership = self.get_own_membership(manager, group_id)
        self.assertEqual(manager_membership["status"], "ACTIVE")
        self.assertEqual(manager_membership["roles"], [])
        self.assertEqual(
            manager_membership["granted_permissions"],
            ["group.permission.manage"],
        )
        self.assertEqual(manager_membership["role_permissions"], [])
        self.assertEqual(manager_user["global_granted_permissions"], [])
        self.assertEqual(manager_user["global_role_permissions"], [])

        grant_target_permission_response = manager.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="group.resource.create",
        )
        self.assertEqual(grant_target_permission_response.status_code, 204)
        repeat_grant_response = manager.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="group.resource.create",
        )
        self.assertEqual(repeat_grant_response.status_code, 204)

        target_user, target_membership = self.get_own_membership(target, group_id)
        self.assertEqual(target_membership["status"], "ACTIVE")
        self.assertEqual(target_membership["roles"], [])
        self.assertEqual(target_membership["granted_permissions"], ["group.resource.create"])
        self.assertEqual(target_membership["role_permissions"], [])
        self.assertEqual(target_user["global_granted_permissions"], [])
        self.assertEqual(target_user["global_role_permissions"], [])

        revoke_target_permission_response = manager.delete(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="group.resource.create",
        )
        self.assertEqual(revoke_target_permission_response.status_code, 204)
        repeat_revoke_response = manager.delete(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="group.resource.create",
        )
        self.assertEqual(repeat_revoke_response.status_code, 204)

        target_user, target_membership = self.get_own_membership(target, group_id)
        self.assertEqual(target_membership["status"], "ACTIVE")
        self.assertEqual(target_membership["roles"], [])
        self.assertEqual(target_membership["granted_permissions"], [])
        self.assertEqual(target_membership["role_permissions"], [])
        self.assertEqual(target_user["global_granted_permissions"], [])
        self.assertEqual(target_user["global_role_permissions"], [])

    def test_revoke_role_permission_conflict(self) -> None:
        """Regression: revoking a role-derived permission returned 204 instead of 409."""

        suffix = str(uuid4())
        group_id = create_group(self.gateway, f"contract-role-permission-revoke-{suffix}")
        manager = self.create_user("role-permission-manager", suffix)
        target = self.create_user("role-permission-target", suffix)
        manager_id = self.accept_membership(manager, group_id)
        target_id = self.accept_membership(target, group_id)

        grant_manager_permission_response = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=manager_id,
            permission="group.permission.manage",
        )
        self.assertEqual(grant_manager_permission_response.status_code, 204)

        assign_role_response = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=target_id,
            role="EDITOR",
        )
        self.assertEqual(assign_role_response.status_code, 204)

        _, target_membership = self.get_own_membership(target, group_id)
        self.assertEqual(target_membership["granted_permissions"], [])
        self.assertIn("group.resource.create", target_membership["role_permissions"])

        grant_target_permission_response = manager.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="group.resource.create",
        )
        self.assertEqual(grant_target_permission_response.status_code, 204)
        _, target_membership = self.get_own_membership(target, group_id)
        self.assertEqual(target_membership["granted_permissions"], ["group.resource.create"])
        self.assertIn("group.resource.create", target_membership["role_permissions"])

        revoke_granted_permission_response = manager.delete(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="group.resource.create",
        )
        self.assertEqual(revoke_granted_permission_response.status_code, 204)
        _, target_membership = self.get_own_membership(target, group_id)
        self.assertEqual(target_membership["granted_permissions"], [])
        self.assertIn("group.resource.create", target_membership["role_permissions"])

        revoke_response = manager.delete(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="group.resource.create",
        )
        assert_problem(self, revoke_response, 409, "conflict")

        admin_revoke_response = self.gateway.delete(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="group.resource.create",
        )
        self.assertEqual(admin_revoke_response.status_code, 409)
