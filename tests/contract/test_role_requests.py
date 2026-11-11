import os
import unittest
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, concept_id


class RoleRequestContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def test_role_request_rejection_and_acceptance(self) -> None:
        """Verify role requests can be rejected, retried, and accepted."""

        suffix = str(uuid4())
        group_response = self.gateway.post(
            "/groups",
            json={
                "name": f"contract-role-group-{suffix}",
                "concept_id": concept_id(self.gateway, "Data"),
            },
        )
        self.assertEqual(group_response.status_code, 201)
        group_id = group_response.json()["id"]

        email = f"contract-role-applicant-{suffix}@example.org"
        password = uuid4().hex
        credentials = register_user(
            name="Role",
            surname=f"Applicant {suffix}",
            email=email,
            password=password,
        )
        applicant = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(
                username=credentials.username,
                password=credentials.password,
            ),
        )

        membership_request_response = applicant.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json={"roles": []},
        )
        self.assertEqual(membership_request_response.status_code, 201)
        membership_request = membership_request_response.json()
        self.assertEqual(membership_request["status"], "PENDING")
        user_id = membership_request["user"]["id"]

        membership_accept_response = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="accept",
        )
        self.assertEqual(membership_accept_response.status_code, 200)
        accepted_membership = membership_accept_response.json()
        self.assertEqual(accepted_membership["user"]["id"], user_id)
        self.assertEqual(accepted_membership["status"], "ACTIVE")
        self.assertEqual(accepted_membership["roles"], [])

        role_request_response = applicant.post(
            "/groups/{groupId}/role-requests",
            groupId=group_id,
            json={"role": "EDITOR"},
        )
        self.assertEqual(role_request_response.status_code, 201)
        role_request = role_request_response.json()
        self.assertEqual(role_request["role"], "EDITOR")
        self.assertEqual(role_request["status"], "PENDING")
        role_request_id = role_request["id"]

        duplicate_role_request_response = applicant.post(
            "/groups/{groupId}/role-requests",
            groupId=group_id,
            json={"role": "EDITOR"},
        )
        assert_problem(self, duplicate_role_request_response, 409, "conflict")

        rejected_response = self.gateway.post(
            "/groups/{groupId}/role-requests/{requestId}/{decision}",
            groupId=group_id,
            requestId=role_request_id,
            decision="reject",
        )
        self.assertEqual(rejected_response.status_code, 200)
        rejected_membership = rejected_response.json()
        self.assertEqual(rejected_membership["user"]["id"], user_id)
        self.assertEqual(rejected_membership["status"], "ACTIVE")
        self.assertEqual(rejected_membership["roles"], [])

        retry_role_request_response = applicant.post(
            "/groups/{groupId}/role-requests",
            groupId=group_id,
            json={"role": "EDITOR"},
        )
        self.assertEqual(retry_role_request_response.status_code, 201)
        retry_role_request = retry_role_request_response.json()
        self.assertEqual(retry_role_request["role"], "EDITOR")
        self.assertEqual(retry_role_request["status"], "PENDING")
        retry_role_request_id = retry_role_request["id"]

        accepted_response = self.gateway.post(
            "/groups/{groupId}/role-requests/{requestId}/{decision}",
            groupId=group_id,
            requestId=retry_role_request_id,
            decision="accept",
        )
        self.assertEqual(accepted_response.status_code, 200)
        accepted_membership = accepted_response.json()
        self.assertEqual(accepted_membership["user"]["id"], user_id)
        self.assertEqual(accepted_membership["status"], "ACTIVE")
        self.assertEqual(accepted_membership["roles"], ["EDITOR"])
        self.assertEqual(accepted_membership["role_requests"], [])

        user_response = applicant.get(
            "/users/{id}",
            id="self",
        )
        self.assertEqual(user_response.status_code, 200)
        user = user_response.json()
        self.assertEqual(user["id"], user_id)
        self.assertEqual(user["email"], email)
        memberships = [
            membership
            for membership in user["memberships"]
            if membership["group"]["id"] == group_id
        ]
        self.assertEqual(len(memberships), 1)
        membership_summary = memberships[0]
        self.assertEqual(membership_summary["status"], "ACTIVE")
        self.assertEqual(membership_summary["roles"], ["EDITOR"])
        self.assertCountEqual(
            membership_summary["granted_permissions"],
            [],
        )
        self.assertCountEqual(
            membership_summary["role_permissions"],
            [
                "group.resource.create",
                "group.resource.edit",
                "group.resource.submit",
                "group.resource.withdraw",
                "group.resource.reopen",
                "group.resource.discard",
            ],
        )
        self.assertEqual(membership_summary["role_requests"], [])
