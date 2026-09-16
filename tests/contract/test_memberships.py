import os
import unittest
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient


class MembershipContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def test_membership_request_and_decision(self) -> None:
        """Verify membership requests and decisions enforce their lifecycle rules."""

        suffix = str(uuid4())
        created_group_response = self.gateway.post(
            "/groups",
            json={
                "name": f"contract-membership-group-{suffix}",
                "concept": "DATA",
            },
        )
        self.assertEqual(created_group_response.status_code, 201)
        group = created_group_response.json()
        group_id = group["id"]

        aai_url = os.getenv("AAI_URL", "http://localhost:8110")
        email = f"contract-applicant-{suffix}@example.org"
        password = uuid4().hex
        credentials = register_user(
            aai_url,
            name="Membership",
            surname=f"Applicant {suffix}",
            email=email,
            password=password,
        )
        applicant = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(
                aai_url=aai_url,
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
        membership = membership_request_response.json()
        self.assertEqual(membership["status"], "PENDING")
        self.assertEqual(membership["roles"], [])
        self.assertEqual(membership["permissions"], [])
        self.assertEqual(membership["role_requests"], [])
        self.assertEqual(membership["user"]["email"], email)
        user_id = membership["user"]["id"]
        self.assertIsInstance(user_id, str)
        self.assertTrue(user_id)

        duplicate_request_response = applicant.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json={"roles": []},
        )
        self.assertEqual(duplicate_request_response.status_code, 409)
        duplicate_problem = duplicate_request_response.json()
        self.assertEqual(duplicate_problem["title"], "conflict")
        self.assertEqual(
            duplicate_problem["detail"],
            "operation conflicts with existing data",
        )

        self_decision_response = applicant.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="accept",
        )
        self.assertEqual(self_decision_response.status_code, 403)
        self_decision_problem = self_decision_response.json()
        self.assertEqual(self_decision_problem["title"], "forbidden")
        self.assertEqual(
            self_decision_problem["detail"],
            "operation is not permitted",
        )

        rejected_response = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="reject",
        )
        self.assertEqual(rejected_response.status_code, 200)
        rejected_membership = rejected_response.json()
        self.assertEqual(rejected_membership["user"]["id"], user_id)
        self.assertEqual(rejected_membership["status"], "REJECTED")

        retry_request_response = applicant.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json={"roles": []},
        )
        self.assertEqual(retry_request_response.status_code, 201)
        retry_membership = retry_request_response.json()
        self.assertEqual(retry_membership["user"]["id"], user_id)
        self.assertEqual(retry_membership["status"], "PENDING")
        self.assertEqual(retry_membership["roles"], [])
        self.assertEqual(retry_membership["permissions"], [])
        self.assertEqual(retry_membership["role_requests"], [])

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
        self.assertEqual(accepted_membership["roles"], [])
        self.assertEqual(accepted_membership["permissions"], [])
        self.assertEqual(accepted_membership["role_requests"], [])

        repeated_accept_response = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="accept",
        )
        self.assertEqual(repeated_accept_response.status_code, 409)
        repeated_accept_problem = repeated_accept_response.json()
        self.assertEqual(repeated_accept_problem["title"], "conflict")
        self.assertEqual(
            repeated_accept_problem["detail"],
            "operation conflicts with existing data",
        )

        user_response = applicant.get(
            "/users/{id}",
            id="self",
        )
        self.assertEqual(user_response.status_code, 200)
        user = user_response.json()
        self.assertEqual(user["id"], user_id)
        self.assertEqual(user["email"], email)
        memberships = user["memberships"]
        self.assertEqual(len(memberships), 1)
        membership_summary = memberships[0]
        self.assertEqual(membership_summary["group"]["id"], group_id)
        self.assertEqual(membership_summary["status"], "ACTIVE")
        self.assertEqual(membership_summary["roles"], [])
        self.assertEqual(membership_summary["permissions"], [])
        self.assertEqual(membership_summary["role_requests"], [])
