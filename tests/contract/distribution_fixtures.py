import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, create_group


def parameter(name: str = "value", kind: str = "STRING", **fields: Any) -> dict[str, Any]:
    result = {
        "kind": kind, "position": 0, "name": name, "label": name,
        "required": False, "read_only": False, "location": "QUERY", "explode": False,
    }
    if kind not in ("BOOLEAN", "DATETIME", "TEMPLATE"):
        result["multiple_values"] = False
    result.update(fields)
    return result


def child(name: str = "value", kind: str = "STRING", **fields: Any) -> dict[str, Any]:
    result = parameter(name, kind, **fields)
    for field in ("location", "explode", "multiple_values"):
        result.pop(field, None)
    return result


def web_distribution(
    parameters: list[dict[str, Any]] | None = None, **service: Any,
) -> dict[str, Any]:
    return {
        "title": "prepared-service", "description": "Synthetic preparation contract",
        "type": "WEB_SERVICE",
        "web_service": {
            "name": "prepared-service",
            "configuration": {
                "template_url": "https://provider.invalid/query",
                "method": "GET", "type": "HTTP", "format": "application/json",
                "parameters": parameters or [], **service,
            },
        },
    }


class DistributionContractTestCase(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA, os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"), get_token(),
        )

    def create_draft(
        self, distributions: list[dict[str, Any]],
    ) -> dict[str, Any]:
        group_id = create_group(self.gateway, f"prepare-{uuid4().hex}")
        response = self.gateway.post(
            "/groups/{groupId}/resources", groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {
                    "title": f"prepare-{uuid4().hex}",
                    "description": "Synthetic preparation contract",
                    "distributions": distributions,
                },
            },
        )
        self.assertEqual(response.status_code, 201)
        resource_id = response.json()["id"]
        return self.details(resource_id, self.version(resource_id, "DRAFT")["version_id"])

    def version(self, resource_id: str, status: str) -> dict[str, Any]:
        response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions", resourceType="dataset", id=resource_id,
        )
        self.assertEqual(response.status_code, 200)
        matches = [item for item in response.json() if item["status"] == status]
        self.assertEqual(len(matches), 1)
        return matches[0]

    def details(self, resource_id: str, version_id: str) -> dict[str, Any]:
        response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions/{versionId}", resourceType="dataset",
            id=resource_id, versionId=version_id,
        )
        self.assertEqual(response.status_code, 200)
        return response.json()

    def replace_response(self, detail: dict[str, Any], metadata: dict[str, Any]) -> Any:
        return self.gateway.put(
            "/resources/{id}/versions/{versionId}/draft", id=detail["resource_id"],
            versionId=detail["version_id"],
            json={"resource_type": "dataset", "revision": detail["revision"], "metadata": metadata},
        )

    def transition(self, detail: dict[str, Any], action: str, **kwargs: Any) -> None:
        response = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/" + action,
            resourceType="dataset", id=detail["resource_id"], versionId=detail["version_id"],
            **kwargs,
        )
        self.assertEqual(response.status_code, 200)

    def prepare_response(
        self, distribution: dict[str, Any], values: dict[str, Any] | None = None,
        *, gateway: GatewayClient | None = None, **kwargs: Any,
    ) -> Any:
        return (gateway or self.gateway).post(
            "/distributions/{distributionId}/prepare", distributionId=distribution["id"],
            json={"parameters": [{"id": key, "value": value} for key, value in (values or {}).items()]},
            **kwargs,
        )

    def prepare(self, distribution: dict[str, Any], **kwargs: Any) -> dict[str, Any]:
        response = self.prepare_response(distribution, **kwargs)
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.headers.get("Cache-Control"), "no-store")
        self.assertEqual(response.headers["Content-Type"].split(";")[0], "application/json")
        result = response.json()
        self.assertEqual(set(result), {"method", "url", "headers", "body"})
        return result

    def assert_prepare_problem(
        self, response: Any, status: int = 400, code: str = "validation_failed",
        *, target: str | None = None, secrets: tuple[str, ...] = (),
    ) -> dict[str, Any]:
        problem = assert_problem(self, response, status, code)
        self.assertEqual(response.headers["Content-Type"].split(";")[0], "application/problem+json")
        self.assertTrue(problem.get("instance"))
        self.assertTrue(problem.get("detail"))
        self.assertNotIn("upstream_response", problem)
        self.assertNotIn("retrieval_id", problem)
        if target is not None:
            self.assertTrue(problem.get("target", "").startswith(target), problem)
        for text in ("rpc error", "dial tcp", "SQLSTATE", *secrets):
            self.assertNotIn(text, problem["detail"])
        return problem

    def create_user(self, label: str) -> tuple[GatewayClient, str]:
        credentials = register_user(
            name="Preparation", surname=label, email=f"prepare-{uuid4().hex}@example.org",
            password=uuid4().hex,
        )
        gateway = GatewayClient(
            SCHEMA, self.gateway.base_url,
            get_token(username=credentials.username, password=credentials.password),
        )
        response = gateway.get("/users/{id}", id="self")
        self.assertEqual(response.status_code, 200)
        return gateway, response.json()["id"]
