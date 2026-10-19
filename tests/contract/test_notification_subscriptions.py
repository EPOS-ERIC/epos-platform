import os
import unittest
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, create_group


class NotificationSubscriptionsContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def create_user(self, label: str, suffix: str) -> tuple[GatewayClient, str]:
        credentials = register_user(
            name="Notification",
            surname=f"{label.title()} {suffix}",
            email=f"notification-{uuid4().hex}@example.org",
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
        return user_id

    def test_group_managers_only_see_subscriptions_in_managed_groups(self) -> None:
        """Regression: group-only managers must not see global or other-group subscriptions."""

        suffix = uuid4().hex
        direct_group_id = create_group(self.gateway, f"contract-notification-direct-{suffix}")
        role_group_id = create_group(self.gateway, f"contract-notification-role-{suffix}")
        hidden_group_id = create_group(self.gateway, f"contract-notification-hidden-{suffix}")
        target, target_id = self.create_user("target", suffix)
        manager, manager_id = self.create_user("manager", suffix)
        denied, _ = self.create_user("denied", suffix)
        global_manager, global_manager_id = self.create_user("global-manager", suffix)

        for group_id in (direct_group_id, role_group_id, hidden_group_id):
            self.accept_membership(target, group_id)
        self.accept_membership(manager, direct_group_id)
        self.accept_membership(manager, role_group_id)

        direct_permission = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=direct_group_id,
            userId=manager_id,
            permission="group.notification.manage",
        )
        self.assertEqual(direct_permission.status_code, 204)
        role_permission = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=role_group_id,
            userId=manager_id,
            role="GROUP_ADMIN",
        )
        self.assertEqual(role_permission.status_code, 204)
        global_permission = self.gateway.put(
            "/users/{id}/global-permissions/{permission}",
            id=global_manager_id,
            permission="global.notification.manage",
        )
        self.assertEqual(global_permission.status_code, 204)

        global_subscription = target.put(
            "/users/{userId}/notification-subscriptions/{topicKey}",
            userId="self",
            topicKey="global.membership.requested",
        )
        self.assertEqual(global_subscription.status_code, 204)
        for group_id in (direct_group_id, role_group_id, hidden_group_id):
            subscription = target.put(
                "/groups/{groupId}/notification-subscriptions/{userId}/{topicKey}",
                groupId=group_id,
                userId="self",
                topicKey="group.resource.published",
            )
            self.assertEqual(subscription.status_code, 204)

        manager_response = manager.get(
            "/users/{userId}/notification-subscriptions",
            userId=target_id,
        )
        self.assertEqual(manager_response.status_code, 200)
        manager_subscriptions = manager_response.json()
        self.assertCountEqual(
            [item.get("group_id") for item in manager_subscriptions],
            [direct_group_id, role_group_id],
        )
        self.assertTrue(
            all(item["user_id"] == target_id for item in manager_subscriptions)
        )

        denied_response = denied.get(
            "/users/{userId}/notification-subscriptions",
            userId=target_id,
        )
        assert_problem(self, denied_response, 403, "permission_denied")

        self_response = target.get(
            "/users/{userId}/notification-subscriptions",
            userId="self",
        )
        self.assertEqual(self_response.status_code, 200)
        self_subscriptions = self_response.json()
        self.assertIn(
            (None, "global.membership.requested"),
            [(item.get("group_id"), item["topic_key"]) for item in self_subscriptions],
        )
        self.assertCountEqual(
            [item["group_id"] for item in self_subscriptions if "group_id" in item],
            [direct_group_id, role_group_id, hidden_group_id],
        )

        global_manager_response = global_manager.get(
            "/users/{userId}/notification-subscriptions",
            userId=target_id,
        )
        self.assertEqual(global_manager_response.status_code, 200)
        self.assertCountEqual(
            [
                (item.get("group_id"), item["topic_key"])
                for item in global_manager_response.json()
            ],
            [
                (item.get("group_id"), item["topic_key"])
                for item in self_subscriptions
            ],
        )
