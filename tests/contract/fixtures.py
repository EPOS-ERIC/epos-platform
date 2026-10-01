import unittest
from typing import Any

from client import GatewayClient


def assert_problem(
    test_case: unittest.TestCase,
    response: Any,
    status: int,
    code: str,
) -> dict[str, Any]:
    test_case.assertEqual(response.status_code, status)
    problem = response.json()
    test_case.assertEqual(problem["status"], status)
    test_case.assertEqual(problem["type"], code)
    return problem


def create_group(
    gateway: GatewayClient,
    name: str,
    concept: str = "DATA",
) -> str:
    response = gateway.post(
        "/groups",
        json={"name": name, "concept": concept},
    )
    if response.status_code != 201:
        raise AssertionError(
            f"group creation returned {response.status_code}, want 201: {response.text}"
        )
    return response.json()["id"]
