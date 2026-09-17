import os
import unittest
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient


class UserContractTest(unittest.TestCase):
    def test_user_self_provisioning(self) -> None:
        """Verify a new AAI user is provisioned and can read its own profile."""

        suffix = str(uuid4())
        aai_url = os.getenv("AAI_URL", "http://localhost:8110")
        email = f"contract-user-{suffix}@example.org"
        password = uuid4().hex
        credentials = register_user(
            aai_url,
            name="Contract",
            surname=f"User {suffix}",
            email=email,
            password=password,
        )
        token = get_token(
            aai_url=aai_url,
            username=credentials.username,
            password=credentials.password,
        )

        gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            token,
        )
        response = gateway.get(
            "/users/{id}",
            id="self",
        )

        self.assertEqual(response.status_code, 200)
        user = response.json()
        self.assertIsInstance(user["id"], str)
        self.assertTrue(user["id"])
        self.assertEqual(user["email"], email)
        self.assertEqual(user["given_name"], "Contract")
        self.assertEqual(user["family_name"], f"User {suffix}")
        self.assertEqual(user["status"], "ACTIVE")
        self.assertNotEqual(user["created_at"], "0001-01-01T00:00:00Z")
        self.assertNotEqual(user["updated_at"], "0001-01-01T00:00:00Z")
        self.assertEqual(user["global_roles"], [])
        self.assertEqual(user["global_granted_permissions"], [])
        self.assertEqual(user["global_role_permissions"], [])
        self.assertEqual(user["memberships"], [])
