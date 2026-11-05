import os
import unittest
from typing import Any

from auth import get_token
from client import SCHEMA, GatewayClient

RESOURCE_TYPES = ("dataset", "software-application", "software-source-code")


class DiscoveryContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def test_draft_discovery_and_details(self) -> None:
        """Verify imported DRAFT resources can be fetched by version."""

        response = self.gateway.get(
            "/search",
            params={"versioningStatus": "DRAFT"},
        )
        self.assertEqual(response.status_code, 200)
        items = response.json()
        self.assertIsInstance(items, list)

        items_by_type: dict[str, list[dict[str, Any]]] = {
            resource_type: [] for resource_type in RESOURCE_TYPES
        }
        for item in items:
            if isinstance(item, dict) and item.get("type") in RESOURCE_TYPES:
                resource_type = item["type"]
                items_by_type[resource_type].append(item)

        for resource_type in RESOURCE_TYPES:
            with self.subTest(resource_type=resource_type):
                candidates = items_by_type[resource_type]
                self.assertTrue(
                    candidates,
                    f"no DRAFT {resource_type} resource found; run make populate-metadatone",
                )
                item = min(
                    candidates,
                    key=lambda candidate: (
                        str(candidate.get("name", "")),
                        str(candidate["resource_id"]),
                        str(candidate["version_id"]),
                    ),
                )

                details = self.gateway.get(
                    "/resources/{resourceType}/{id}/versions/{versionId}",
                    resourceType=item["type"],
                    id=item["resource_id"],
                    versionId=item["version_id"],
                ).json()

                self.assertEqual(details["resource_id"], item["resource_id"])
                self.assertEqual(details["id"], item["version_id"])
                self.assertEqual(details["status"], "DRAFT")
                self.assertEqual(details["metadata"]["resource_type"], resource_type)
