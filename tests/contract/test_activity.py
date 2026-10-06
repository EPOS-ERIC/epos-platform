import os
import unittest
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import assert_problem


NO_AUTH_HEADERS = {"Authorization": ""}


class ActivityContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def test_activity_requires_global_activity_permission(self) -> None:
        """Verify admins can read activity and other identities cannot."""
        response = self.gateway.get("/activity")
        self.assertEqual(response.status_code, 200)
        events = response.json()
        self.assertIsInstance(events, list)
        for event in events:
            self.assertIn("action", event)
            self.assertIn("occurred_at", event)

        invalid_group = self.gateway.get(
            "/activity",
            params={"groupId": "00000000-0000-0000-0000-000000000000"},
        )
        group_problem = assert_problem(self, invalid_group, 400, "validation_failed")
        self.assertEqual(group_problem.get("target"), "groupId")
        self.assertEqual(group_problem["detail"], "Must not be the nil UUID.")

        invalid_date_range = self.gateway.get(
            "/activity",
            params={
                "dateFrom": "2026-01-02T00:00:00Z",
                "dateTo": "2026-01-01T00:00:00Z",
            },
        )
        date_problem = assert_problem(self, invalid_date_range, 400, "validation_failed")
        self.assertEqual(date_problem.get("target"), "dateFrom")
        self.assertEqual(date_problem["detail"], "Must be on or before `dateTo`.")

        unauthenticated = self.gateway.get("/activity", headers=NO_AUTH_HEADERS)
        assert_problem(self, unauthenticated, 401, "unauthenticated")

        suffix = uuid4().hex
        aai_url = os.getenv("AAI_URL", "http://localhost:8110")
        credentials = register_user(
            aai_url,
            name="Activity",
            surname=f"Reader {suffix}",
            email=f"activity-{suffix}@example.org",
            password=uuid4().hex,
        )
        unauthorized_user = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(
                aai_url=aai_url,
                username=credentials.username,
                password=credentials.password,
            ),
        )
        denied = unauthorized_user.get("/activity")
        assert_problem(self, denied, 403, "permission_denied")
