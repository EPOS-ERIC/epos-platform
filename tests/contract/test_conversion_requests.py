import os
import unittest
from uuid import uuid4

from auth import get_token
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, create_group


class ConversionRequestsContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def test_request_requires_native_format(self) -> None:
        """Conversion requests accept draft web services and reject undeclared formats."""
        group_id = create_group(self.gateway, f"conversion-request-{uuid4().hex}")
        created = self.gateway.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {
                    "title": "Conversion request dataset",
                    "description": "Conversion request test",
                    "distributions": [{
                        "title": "Test service",
                        "description": "Test service distribution",
                        "type": "WEB_SERVICE",
                        "web_service": {
                            "name": "Test service",
                            "configuration": {
                                "template_url": "https://example.org/data",
                                "method": "GET",
                                "type": "HTTP",
                                "format": "application/xml",
                                "parameters": [],
                            },
                        },
                    }],
                },
            },
        )
        self.assertEqual(created.status_code, 201)
        resource_id = created.json()["id"]
        versions = self.gateway.get("/resources/{resourceType}/{id}/versions", resourceType="dataset", id=resource_id)
        self.assertEqual(versions.status_code, 200)
        version_id = next(item["version_id"] for item in versions.json() if item["status"] == "DRAFT")
        detail = self.gateway.get("/resources/{resourceType}/{id}/versions/{versionId}", resourceType="dataset", id=resource_id, versionId=version_id)
        self.assertEqual(detail.status_code, 200)
        distribution_id = detail.json()["metadata"]["distributions"][0]["id"]
        path = "/distributions/{distributionId}/conversion-requests"
        request = {"input_format": "application/json", "visualizations": ["MAP", "TABLE", "GRAPH"], "message": "Please convert"}
        invalid = self.gateway.post(path, distributionId=distribution_id, json=request)
        assert_problem(self, invalid, 400, "validation_failed")
        request["input_format"] = "application/xml"
        accepted = self.gateway.post(path, distributionId=distribution_id, json=request)
        self.assertEqual(accepted.status_code, 202)
