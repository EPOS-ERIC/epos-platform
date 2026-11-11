import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token
from client import SCHEMA, GatewayClient
from fixtures import create_group

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
        """Verify DRAFT resources of each type can be fetched by version."""

        suffix = str(uuid4())
        group_id = create_group(self.gateway, f"discovery-draft-{suffix}")
        for resource_type in RESOURCE_TYPES:
            metadata = {"description": "Discovery test", "keywords": []}
            metadata["title" if resource_type == "dataset" else "name"] = f"draft-{resource_type}-{suffix}"
            created = self.gateway.post(
                "/groups/{groupId}/resources",
                groupId=group_id,
                json={"resource_type": resource_type, "metadata": metadata},
            )
            self.assertEqual(created.status_code, 201)

        response = self.gateway.get(
            "/search",
            params={"versioningStatus": "DRAFT", "q": suffix},
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
                    f"no DRAFT {resource_type} resource found",
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
