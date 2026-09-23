import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient


class MembershipInvariantsContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def create_group(self, suffix: str) -> str:
        response = self.gateway.post(
            "/groups",
            json={"name": f"membership-invariants-{suffix}", "concept": "DATA"},
        )
        self.assertEqual(response.status_code, 201)
        return response.json()["id"]

    def create_user(self, label: str, suffix: str) -> tuple[GatewayClient, str]:
        aai_url = os.getenv("AAI_URL", "http://localhost:8110")
        credentials = register_user(
            aai_url,
            name="Membership",
            surname=f"{label.title()} {suffix}",
            email=f"membership-{uuid4().hex}@example.org",
            password=uuid4().hex,
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
        profile = client.get("/users/{id}", id="self")
        self.assertEqual(profile.status_code, 200)
        return client, profile.json()["id"]

    def request_membership(
        self,
        client: GatewayClient,
        group_id: str,
        roles: list[str],
    ) -> dict[str, Any]:
        response = client.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json={"roles": roles},
        )
        self.assertEqual(response.status_code, 201)
        return response.json()

    def decide_membership(
        self,
        group_id: str,
        user_id: str,
        decision: str,
    ) -> dict[str, Any]:
        response = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision=decision,
        )
        self.assertEqual(response.status_code, 200)
        return response.json()

    def get_group_membership(
        self,
        group_id: str,
        user_id: str,
    ) -> dict[str, Any] | None:
        response = self.gateway.get(
            "/groups/{groupId}/memberships",
            groupId=group_id,
        )
        self.assertEqual(response.status_code, 200)
        matches = [item for item in response.json() if item["user"]["id"] == user_id]
        self.assertLessEqual(len(matches), 1)
        return matches[0] if matches else None

    def require_group_membership(self, group_id: str, user_id: str) -> dict[str, Any]:
        membership = self.get_group_membership(group_id, user_id)
        self.assertIsNotNone(membership)
        assert membership is not None
        return membership

    def get_own_membership(
        self,
        client: GatewayClient,
        group_id: str,
    ) -> dict[str, Any] | None:
        response = client.get("/users/{id}", id="self")
        self.assertEqual(response.status_code, 200)
        matches = [
            item
            for item in response.json()["memberships"]
            if item["group"]["id"] == group_id
        ]
        self.assertLessEqual(len(matches), 1)
        return matches[0] if matches else None

    def assert_conflict(self, response: Any) -> None:
        self.assertEqual(response.status_code, 409)
        problem = response.json()
        self.assertEqual(problem["title"], "conflict")
        self.assertEqual(
            problem["detail"],
            "operation conflicts with existing data",
        )

    def assert_empty_access(self, membership: dict[str, Any]) -> None:
        self.assertEqual(membership["roles"], [])
        self.assertEqual(membership["granted_permissions"], [])
        self.assertEqual(membership["role_permissions"], [])
        self.assertEqual(membership["role_requests"], [])

    def test_membership_rejection_and_re_request_cleans_role_requests(self) -> None:
        """Verify membership rejection cascades role requests and resets access."""

        suffix = uuid4().hex
        group_id = self.create_group(suffix)
        applicant, applicant_id = self.create_user("applicant", suffix)

        initial = self.request_membership(
            applicant,
            group_id,
            ["EDITOR", "REVIEWER"],
        )
        self.assertEqual(initial["status"], "PENDING")
        initial_requests = {item["role"]: item for item in initial["role_requests"]}
        self.assertEqual(set(initial_requests), {"EDITOR", "REVIEWER"})
        self.assertTrue(
            all(item["status"] == "PENDING" for item in initial_requests.values())
        )

        self.assert_conflict(
            self.gateway.put(
                "/groups/{groupId}/memberships/{userId}/roles/{role}",
                groupId=group_id,
                userId=applicant_id,
                role="EDITOR",
            )
        )
        self.assert_conflict(
            self.gateway.put(
                "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
                groupId=group_id,
                userId=applicant_id,
                permission="group.resource.create",
            )
        )

        withdrawn = applicant.delete(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
        )
        self.assertEqual(withdrawn.status_code, 204)
        self.assertIsNone(self.get_own_membership(applicant, group_id))

        retry = self.request_membership(
            applicant,
            group_id,
            ["EDITOR", "REVIEWER"],
        )
        retry_requests = {item["role"]: item for item in retry["role_requests"]}
        self.assertEqual(retry["status"], "PENDING")
        self.assertEqual(set(retry_requests), {"EDITOR", "REVIEWER"})
        self.assertTrue(
            all(
                retry_requests[role]["id"] != initial_requests[role]["id"]
                for role in initial_requests
            )
        )

        rejected = self.decide_membership(group_id, applicant_id, "reject")
        self.assertEqual(rejected["status"], "REJECTED")
        self.assertEqual(rejected["role_requests"], [])
        self.assertEqual(rejected["roles"], [])

        self.assert_conflict(
            self.gateway.put(
                "/groups/{groupId}/memberships/{userId}/roles/{role}",
                groupId=group_id,
                userId=applicant_id,
                role="EDITOR",
            )
        )
        self.assert_conflict(
            self.gateway.put(
                "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
                groupId=group_id,
                userId=applicant_id,
                permission="group.resource.create",
            )
        )

        re_requested = self.request_membership(
            applicant,
            group_id,
            ["EDITOR", "REVIEWER"],
        )
        re_requested_requests = {
            item["role"]: item for item in re_requested["role_requests"]
        }
        self.assertEqual(re_requested["status"], "PENDING")
        self.assertEqual(set(re_requested_requests), {"EDITOR", "REVIEWER"})
        self.assertTrue(
            all(
                re_requested_requests[role]["id"] != retry_requests[role]["id"]
                for role in retry_requests
            )
        )

        accepted = self.decide_membership(group_id, applicant_id, "accept")
        self.assertEqual(accepted["status"], "ACTIVE")
        self.assertEqual(accepted["roles"], [])
        accepted_requests = {item["role"]: item for item in accepted["role_requests"]}
        self.assertEqual(set(accepted_requests), {"EDITOR", "REVIEWER"})

        self.assert_conflict(
            self.gateway.put(
                "/groups/{groupId}/memberships/{userId}/roles/{role}",
                groupId=group_id,
                userId=applicant_id,
                role="EDITOR",
            )
        )

        rejected_role = self.gateway.post(
            "/groups/{groupId}/role-requests/{requestId}/{decision}",
            groupId=group_id,
            requestId=accepted_requests["EDITOR"]["id"],
            decision="reject",
        )
        self.assertEqual(rejected_role.status_code, 200)
        rejected_role_membership = rejected_role.json()
        self.assertEqual(rejected_role_membership["status"], "ACTIVE")
        self.assertEqual(rejected_role_membership["roles"], [])
        self.assertEqual(
            [item["role"] for item in rejected_role_membership["role_requests"]],
            ["REVIEWER"],
        )

        accepted_role = self.gateway.post(
            "/groups/{groupId}/role-requests/{requestId}/{decision}",
            groupId=group_id,
            requestId=accepted_requests["REVIEWER"]["id"],
            decision="accept",
        )
        self.assertEqual(accepted_role.status_code, 200)
        accepted_role_membership = accepted_role.json()
        self.assertEqual(accepted_role_membership["status"], "ACTIVE")
        self.assertEqual(accepted_role_membership["roles"], ["REVIEWER"])
        self.assertEqual(accepted_role_membership["role_requests"], [])

        editor_request = applicant.post(
            "/groups/{groupId}/role-requests",
            groupId=group_id,
            json={"role": "EDITOR"},
        )
        self.assertEqual(editor_request.status_code, 201)
        editor_request_id = editor_request.json()["id"]
        self.assert_conflict(
            self.gateway.put(
                "/groups/{groupId}/memberships/{userId}/roles/{role}",
                groupId=group_id,
                userId=applicant_id,
                role="EDITOR",
            )
        )

        withdrawn_role = applicant.delete(
            "/groups/{groupId}/role-requests/{requestId}",
            groupId=group_id,
            requestId=editor_request_id,
        )
        self.assertEqual(withdrawn_role.status_code, 204)

        assigned_editor = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=applicant_id,
            role="EDITOR",
        )
        self.assertEqual(assigned_editor.status_code, 204)
        membership = self.require_group_membership(group_id, applicant_id)
        self.assertCountEqual(membership["roles"], ["EDITOR", "REVIEWER"])
        self.assertCountEqual(
            membership["role_permissions"],
            [
                "group.resource.create",
                "group.resource.edit",
                "group.resource.submit",
                "group.resource.withdraw",
                "group.resource.reopen",
                "group.resource.discard",
                "group.resource.reject",
                "group.resource.publish",
            ],
        )

        removed_editor = self.gateway.delete(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=applicant_id,
            role="EDITOR",
        )
        self.assertEqual(removed_editor.status_code, 204)
        reassigned_editor = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=applicant_id,
            role="EDITOR",
        )
        self.assertEqual(reassigned_editor.status_code, 204)

        granted = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=applicant_id,
            permission="group.category.manage",
        )
        self.assertEqual(granted.status_code, 204)
        membership = self.require_group_membership(group_id, applicant_id)
        self.assertEqual(membership["granted_permissions"], ["group.category.manage"])

        revoked_grant = self.gateway.delete(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=applicant_id,
            permission="group.category.manage",
        )
        self.assertEqual(revoked_grant.status_code, 204)
        membership = self.require_group_membership(group_id, applicant_id)
        self.assertEqual(membership["granted_permissions"], [])

        revoked_membership = applicant.delete(
            "/groups/{groupId}/memberships/{userId}",
            groupId=group_id,
            userId="self",
        )
        self.assertEqual(revoked_membership.status_code, 204)
        self.assertIsNone(self.get_own_membership(applicant, group_id))
        self.assertIsNone(self.get_group_membership(group_id, applicant_id))

        rejoined = self.request_membership(applicant, group_id, [])
        self.assertEqual(rejoined["status"], "PENDING")
        self.assertEqual(rejoined["role_requests"], [])
        restored = self.decide_membership(group_id, applicant_id, "accept")
        self.assertEqual(restored["status"], "ACTIVE")
        self.assert_empty_access(restored)

    def test_membership_revocation_removes_pending_access(self) -> None:
        """Verify revocation removes assignments and pending role requests."""

        suffix = uuid4().hex
        group_id = self.create_group(f"revocation-{suffix}")
        member, member_id = self.create_user("revocation-member", suffix)

        requested = self.request_membership(member, group_id, [])
        self.assertEqual(requested["status"], "PENDING")
        accepted = self.decide_membership(group_id, member_id, "accept")
        self.assertEqual(accepted["status"], "ACTIVE")
        self.assert_empty_access(accepted)

        assigned = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=member_id,
            role="EDITOR",
        )
        self.assertEqual(assigned.status_code, 204)
        granted = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=member_id,
            permission="group.resource.create",
        )
        self.assertEqual(granted.status_code, 204)

        pending_role = member.post(
            "/groups/{groupId}/role-requests",
            groupId=group_id,
            json={"role": "REVIEWER"},
        )
        self.assertEqual(pending_role.status_code, 201)
        pending_role_id = pending_role.json()["id"]
        membership = self.require_group_membership(group_id, member_id)
        self.assertEqual(membership["roles"], ["EDITOR"])
        self.assertEqual(membership["granted_permissions"], ["group.resource.create"])
        self.assertEqual(
            [item["id"] for item in membership["role_requests"]],
            [pending_role_id],
        )

        revoked = member.delete(
            "/groups/{groupId}/memberships/{userId}",
            groupId=group_id,
            userId="self",
        )
        self.assertEqual(revoked.status_code, 204)
        self.assertIsNone(self.get_own_membership(member, group_id))
        self.assertIsNone(self.get_group_membership(group_id, member_id))

        revoked_role = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=member_id,
            role="REVIEWER",
        )
        self.assertEqual(revoked_role.status_code, 404)
        revoked_permission = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=member_id,
            permission="group.resource.create",
        )
        self.assertEqual(revoked_permission.status_code, 404)

        rejoined = self.request_membership(member, group_id, [])
        self.assertEqual(rejoined["status"], "PENDING")
        self.assertEqual(rejoined["role_requests"], [])
        restored = self.decide_membership(group_id, member_id, "accept")
        self.assertEqual(restored["status"], "ACTIVE")
        self.assert_empty_access(restored)

        new_role_request = member.post(
            "/groups/{groupId}/role-requests",
            groupId=group_id,
            json={"role": "REVIEWER"},
        )
        self.assertEqual(new_role_request.status_code, 201)
        self.assertNotEqual(new_role_request.json()["id"], pending_role_id)
