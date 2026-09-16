import os
import unittest
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient


class GroupPermissionContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def test_direct_permission_grant_and_revoke(self) -> None:
        """Verify direct group permissions are granted, read back, and revoked."""

        suffix = str(uuid4())
        group_response = self.gateway.post(
            "/groups",
            json={
                "name": f"contract-permission-group-{suffix}",
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

        manager = create_user("permission-manager")
        target = create_user("permission-target")

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

        manager_id = accept_membership(manager)
        target_id = accept_membership(target)

        grant_manager_permission_response = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=manager_id,
            permission="group.permission.manage",
        )
        self.assertEqual(grant_manager_permission_response.status_code, 204)

        def get_own_membership(client: GatewayClient) -> tuple[dict, dict]:
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

        manager_user, manager_membership = get_own_membership(manager)
        self.assertEqual(manager_membership["status"], "ACTIVE")
        self.assertEqual(manager_membership["roles"], [])
        self.assertEqual(
            manager_membership["permissions"],
            ["group.permission.manage"],
        )
        self.assertEqual(manager_user["global_permissions"], [])

        grant_target_permission_response = manager.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="resource.create",
        )
        self.assertEqual(grant_target_permission_response.status_code, 204)
        repeat_grant_response = manager.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="resource.create",
        )
        self.assertEqual(repeat_grant_response.status_code, 204)

        target_user, target_membership = get_own_membership(target)
        self.assertEqual(target_membership["status"], "ACTIVE")
        self.assertEqual(target_membership["roles"], [])
        self.assertEqual(target_membership["permissions"], ["resource.create"])
        self.assertEqual(target_user["global_permissions"], [])

        revoke_target_permission_response = manager.delete(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="resource.create",
        )
        self.assertEqual(revoke_target_permission_response.status_code, 204)
        repeat_revoke_response = manager.delete(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=target_id,
            permission="resource.create",
        )
        self.assertEqual(repeat_revoke_response.status_code, 204)

        target_user, target_membership = get_own_membership(target)
        self.assertEqual(target_membership["status"], "ACTIVE")
        self.assertEqual(target_membership["roles"], [])
        self.assertEqual(target_membership["permissions"], [])
        self.assertEqual(target_user["global_permissions"], [])
