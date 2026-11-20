import os
import unittest
from urllib.parse import urlparse

from auth import get_token
from client import SCHEMA, GatewayClient
from fixtures import assert_problem


class CapabilitiesContractTest(unittest.TestCase):
    def test_public_instance_capabilities(self) -> None:
        """Verify public capabilities describe the managed instance configuration."""

        gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )
        response = gateway.get("/capabilities", headers={"Authorization": ""})
        self.assertEqual(response.status_code, 200)
        capabilities = response.json()

        self.assertEqual(capabilities["organizations"]["management"], "local")
        auth_root = urlparse(capabilities["authentication"]["root_url"])
        self.assertIn(auth_root.scheme, ("http", "https"))
        self.assertTrue(auth_root.hostname)
        self.assertIsInstance(capabilities["notifications"]["email_enabled"], bool)
        tracker_url = "https://epos-ci.brgm.fr/epos-public/issuetracker/-/work_items"
        self.assertEqual(capabilities["issue_tracker"]["url"], tracker_url)
        self.assertEqual(capabilities["issue_tracker"]["open_new"], f"{tracker_url}/new?type=Issue")

    def test_sync_unavailable_in_managed_mode(self) -> None:
        """Verify manual metadata sync fails when organizations are locally managed."""

        gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )
        capabilities = gateway.get("/capabilities", headers={"Authorization": ""})
        self.assertEqual(capabilities.status_code, 200)
        self.assertEqual(capabilities.json()["organizations"]["management"], "local")

        response = gateway.post("/metadata/sync")
        problem = assert_problem(self, response, 503, "upstream_unavailable")
        self.assertEqual(problem["detail"], "Metadata synchronization is disabled.")
