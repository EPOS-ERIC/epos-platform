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


def concept_id(gateway: GatewayClient, name: str) -> str:
    response = gateway.get("/concepts")
    if response.status_code != 200:
        raise AssertionError(f"listing concepts returned {response.status_code}: {response.text}")
    for concept in response.json():
        if concept["name"].casefold() == name.casefold():
            return concept["id"]
    raise AssertionError(f"concept {name!r} not found")


def create_group(
    gateway: GatewayClient,
    name: str,
    concept: str = "DATA",
) -> str:
    response = gateway.post(
        "/groups",
        json={"name": name, "concept_id": concept_id(gateway, concept)},
    )
    if response.status_code != 201:
        raise AssertionError(
            f"group creation returned {response.status_code}, want 201: {response.text}"
        )
    return response.json()["id"]
