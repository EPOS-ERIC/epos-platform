import os
import unittest
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import assert_problem


class GroupRoleContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def test_direct_role_assignment_and_group_admin_boundary(self) -> None:
        """Verify role managers and global role managers have distinct authority."""

        suffix = str(uuid4())
        group_response = self.gateway.post(
            "/groups",
            json={
                "name": f"contract-role-management-group-{suffix}",
                "concept": "DATA",
            },
        )
        self.assertEqual(group_response.status_code, 201)
        group_id = group_response.json()["id"]

        aai_url = os.getenv("AAI_URL", "http://localhost:8110")

        def create_user(label: str) -> GatewayClient:
            email = f"contract-{label}-{suffix}@example.org"
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

        target = create_user("role-target")
        group_role_manager = create_user("group-role-manager")
        global_role_manager = create_user("global-role-manager")

        def accept_membership(client: GatewayClient) -> str:
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

        target_id = accept_membership(target)
        manager_id = accept_membership(group_role_manager)

        manager_admin_response = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=manager_id,
            role="GROUP_ADMIN",
        )
        self.assertEqual(manager_admin_response.status_code, 204)

        global_manager_profile_response = global_role_manager.get(
            "/users/{id}",
            id="self",
        )
        self.assertEqual(global_manager_profile_response.status_code, 200)
        global_manager_profile = global_manager_profile_response.json()
        global_manager_id = global_manager_profile["id"]
        self.assertEqual(global_manager_profile["memberships"], [])

        grant_global_role_permission_response = self.gateway.put(
            "/users/{id}/global-permissions/{permission}",
            id=global_manager_id,
            permission="global.role.manage",
        )
        self.assertEqual(grant_global_role_permission_response.status_code, 204)

        global_manager_profile_response = global_role_manager.get(
            "/users/{id}",
            id="self",
        )
        self.assertEqual(global_manager_profile_response.status_code, 200)
        self.assertIn(
            "global.role.manage",
            global_manager_profile_response.json()["global_granted_permissions"],
        )

        manager_profile_response = group_role_manager.get(
            "/users/{id}",
            id="self",
        )
        self.assertEqual(manager_profile_response.status_code, 200)
        manager_memberships = manager_profile_response.json()["memberships"]
        self.assertEqual(len(manager_memberships), 1)
        self.assertEqual(manager_memberships[0]["group"]["id"], group_id)
        self.assertEqual(manager_memberships[0]["roles"], ["GROUP_ADMIN"])

        assign_editor_response = group_role_manager.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=target_id,
            role="EDITOR",
        )
        self.assertEqual(assign_editor_response.status_code, 204)
        repeat_assign_editor_response = group_role_manager.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=target_id,
            role="EDITOR",
        )
        self.assertEqual(repeat_assign_editor_response.status_code, 204)

        target_profile_response = target.get(
            "/users/{id}",
            id="self",
        )
        self.assertEqual(target_profile_response.status_code, 200)
        target_profile = target_profile_response.json()
        target_memberships = target_profile["memberships"]
        self.assertEqual(len(target_memberships), 1)
        target_membership = target_memberships[0]
        self.assertEqual(target_membership["group"]["id"], group_id)
        self.assertEqual(target_membership["roles"], ["EDITOR"])
        self.assertCountEqual(
            target_membership["granted_permissions"],
            [],
        )
        self.assertCountEqual(
            target_membership["role_permissions"],
            [
                "group.resource.create",
                "group.resource.edit",
                "group.resource.submit",
                "group.resource.withdraw",
                "group.resource.reopen",
                "group.resource.discard",
            ],
        )

        remove_editor_response = group_role_manager.delete(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=target_id,
            role="EDITOR",
        )
        self.assertEqual(remove_editor_response.status_code, 204)
        repeat_remove_editor_response = group_role_manager.delete(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=target_id,
            role="EDITOR",
        )
        self.assertEqual(repeat_remove_editor_response.status_code, 204)

        target_profile_response = target.get(
            "/users/{id}",
            id="self",
        )
        self.assertEqual(target_profile_response.status_code, 200)
        target_membership = target_profile_response.json()["memberships"][0]
        self.assertEqual(target_membership["roles"], [])
        self.assertEqual(target_membership["granted_permissions"], [])
        self.assertEqual(target_membership["role_permissions"], [])

        group_manager_admin_response = group_role_manager.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=target_id,
            role="GROUP_ADMIN",
        )
        assert_problem(self, group_manager_admin_response, 403, "permission_denied")

        global_manager_admin_response = global_role_manager.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=target_id,
            role="GROUP_ADMIN",
        )
        self.assertEqual(global_manager_admin_response.status_code, 204)

        target_profile_response = target.get(
            "/users/{id}",
            id="self",
        )
        self.assertEqual(target_profile_response.status_code, 200)
        target_membership = target_profile_response.json()["memberships"][0]
        self.assertEqual(target_membership["roles"], ["GROUP_ADMIN"])
        self.assertCountEqual(
            target_membership["granted_permissions"],
            [],
        )
        self.assertCountEqual(
            target_membership["role_permissions"],
            [
                "group.resource.create",
                "group.resource.edit",
                "group.resource.submit",
                "group.resource.withdraw",
                "group.resource.reopen",
                "group.resource.discard",
                "group.resource.reject",
                "group.resource.publish",
                "group.resource.unpublish",
                "group.membership.manage",
                "group.role.manage",
                "group.permission.manage",
                "group.notification.manage",
                "group.category.manage",
            ],
        )

        remove_group_admin_response = global_role_manager.delete(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=target_id,
            role="GROUP_ADMIN",
        )
        self.assertEqual(remove_group_admin_response.status_code, 204)

        target_profile_response = target.get(
            "/users/{id}",
            id="self",
        )
        self.assertEqual(target_profile_response.status_code, 200)
        target_membership = target_profile_response.json()["memberships"][0]
        self.assertEqual(target_membership["roles"], [])
        self.assertEqual(target_membership["granted_permissions"], [])
        self.assertEqual(target_membership["role_permissions"], [])
