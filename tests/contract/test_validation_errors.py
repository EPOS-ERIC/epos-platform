import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, create_group


class ValidationErrorsContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def create_user(self, suffix: str) -> tuple[GatewayClient, str]:
        credentials = register_user(
            name="Validation",
            surname=f"User {suffix}",
            email=f"validation-{uuid4().hex}@example.org",
            password=uuid4().hex,
        )
        client = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(
                username=credentials.username,
                password=credentials.password,
            ),
        )
        profile = client.get("/users/{id}", id="self")
        self.assertEqual(profile.status_code, 200)
        return client, profile.json()["id"]

    def accept_membership(self, client: GatewayClient, group_id: str) -> str:
        requested = client.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json={"roles": []},
        )
        self.assertEqual(requested.status_code, 201)
        user_id = requested.json()["user"]["id"]

        accepted = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="accept",
        )
        self.assertEqual(accepted.status_code, 200)
        self.assertEqual(accepted.json()["status"], "ACTIVE")
        return user_id

    def create_category(self, group_id: str, suffix: str) -> str:
        response = self.gateway.post(
            "/groups/{groupId}/categories",
            groupId=group_id,
            json={"name": f"validation-category-{suffix}"},
        )
        self.assertEqual(response.status_code, 201)
        return response.json()["id"]

    def get_version(self, resource_id: str, status: str) -> dict[str, Any]:
        response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(response.status_code, 200)
        matches = [version for version in response.json() if version["status"] == status]
        self.assertEqual(len(matches), 1)
        return matches[0]

    def create_draft(self, group_id: str, suffix: str) -> tuple[str, dict[str, Any]]:
        response = self.gateway.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {
                    "title": f"validation-draft-{suffix}",
                    "description": "Validation draft description",
                },
            },
        )
        self.assertEqual(response.status_code, 201)
        resource_id = response.json()["id"]
        return resource_id, self.get_version(resource_id, "DRAFT")

    def create_submitted_version(self, group_id: str, suffix: str) -> tuple[str, str]:
        resource_id, draft = self.create_draft(group_id, f"submitted-{suffix}")
        submitted = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType="dataset",
            id=resource_id,
            versionId=draft["id"],
        )
        self.assertEqual(submitted.status_code, 200)
        submitted_version = self.get_version(resource_id, "SUBMITTED")
        return resource_id, submitted_version["id"]

    def assert_bad_request(self, response: Any) -> None:
        assert_problem(self, response, 400, "validation_failed")

    def assert_unsupported_media_type(self, response: Any) -> None:
        assert_problem(self, response, 415, "unsupported_media_type")

    def test_request_shape_validation(self) -> None:
        """Verify representative JSON body and path validation errors."""

        suffix = uuid4().hex
        group_id = create_group(self.gateway, f"validation-{suffix}")
        membership_group_id = create_group(self.gateway, f"membership-{suffix}")
        category_id = self.create_category(group_id, suffix)
        ordinary, ordinary_id = self.create_user(suffix)
        self.assertEqual(self.accept_membership(ordinary, group_id), ordinary_id)
        draft_resource_id, draft = self.create_draft(group_id, suffix)
        submitted_resource_id, submitted_version_id = self.create_submitted_version(
            group_id,
            suffix,
        )

        with self.subTest(operation="createGroup", case="wrong media type"):
            self.assert_unsupported_media_type(
                self.gateway.post(
                    "/groups",
                    data="name=invalid",
                    headers={"Content-Type": "text/plain"},
                )
            )
        with self.subTest(operation="createGroup", case="missing field"):
            self.assert_bad_request(
                self.gateway.post("/groups", json={"concept": "DATA"})
            )
        with self.subTest(operation="createGroup", case="unknown field"):
            self.assert_bad_request(
                self.gateway.post(
                    "/groups",
                    json={
                        "name": f"validation-unknown-{suffix}",
                        "concept": "DATA",
                        "unexpected": True,
                    },
                )
            )
        with self.subTest(operation="createGroup", case="blank name"):
            self.assert_bad_request(
                self.gateway.post(
                    "/groups",
                    json={"name": " ", "concept": "DATA"},
                )
            )
        with self.subTest(operation="createGroup", case="invalid enum"):
            self.assert_bad_request(
                self.gateway.post(
                    "/groups",
                    json={
                        "name": f"validation-enum-{suffix}",
                        "concept": "INVALID",
                    },
                )
            )

        with self.subTest(operation="createGroupCategory", case="malformed UUID"):
            self.assert_bad_request(
                self.gateway.post(
                    "/groups/{groupId}/categories",
                    groupId="not-a-uuid",
                    json={"name": f"validation-path-{suffix}"},
                )
            )
        with self.subTest(operation="createGroupCategory", case="missing field"):
            self.assert_bad_request(
                self.gateway.post(
                    "/groups/{groupId}/categories",
                    groupId=group_id,
                    json={},
                )
            )
        with self.subTest(operation="createGroupCategory", case="unknown field"):
            self.assert_bad_request(
                self.gateway.post(
                    "/groups/{groupId}/categories",
                    groupId=group_id,
                    json={
                        "name": f"validation-unknown-category-{suffix}",
                        "unexpected": True,
                    },
                )
            )
        with self.subTest(operation="createGroupCategory", case="blank name"):
            self.assert_bad_request(
                self.gateway.post(
                    "/groups/{groupId}/categories",
                    groupId=group_id,
                    json={"name": " "},
                )
            )

        with self.subTest(operation="replaceGroupCategory", case="missing field"):
            self.assert_bad_request(
                self.gateway.put(
                    "/groups/{groupId}/categories/{categoryId}",
                    groupId=group_id,
                    categoryId=category_id,
                    json={"name": f"validation-replace-{suffix}"},
                )
            )
        with self.subTest(operation="replaceGroupCategory", case="blank name"):
            self.assert_bad_request(
                self.gateway.put(
                    "/groups/{groupId}/categories/{categoryId}",
                    groupId=group_id,
                    categoryId=category_id,
                    json={"name": " ", "parent_category_id": None},
                )
            )

        with self.subTest(operation="createOrganization", case="unknown field"):
            self.assert_bad_request(
                self.gateway.post(
                    "/organizations",
                    json={"name": f"validation-org-{suffix}", "unexpected": True},
                )
            )
        with self.subTest(operation="createOrganization", case="blank name"):
            self.assert_bad_request(
                self.gateway.post("/organizations", json={"name": " "})
            )
        with self.subTest(operation="createOrganization", case="invalid email"):
            self.assert_bad_request(
                self.gateway.post(
                    "/organizations",
                    json={
                        "name": f"validation-email-{suffix}",
                        "email": "not-an-email",
                    },
                )
            )
        with self.subTest(operation="createOrganization", case="invalid URI"):
            self.assert_bad_request(
                self.gateway.post(
                    "/organizations",
                    json={
                        "name": f"validation-uri-{suffix}",
                        "page_url": "not a uri",
                    },
                )
            )
        with self.subTest(operation="createOrganization", case="malformed UUID"):
            self.assert_bad_request(
                self.gateway.post(
                    "/organizations",
                    json={
                        "name": f"validation-parent-{suffix}",
                        "parents": ["not-a-uuid"],
                    },
                )
            )

        with self.subTest(operation="createContactPoint", case="missing field"):
            self.assert_bad_request(
                self.gateway.post(
                    "/contact-points",
                    json={"email": f"valid-{suffix}@example.org"},
                )
            )
        with self.subTest(operation="createContactPoint", case="unknown field"):
            self.assert_bad_request(
                self.gateway.post(
                    "/contact-points",
                    json={
                        "email": f"unknown-{suffix}@example.org",
                        "role": "validation",
                        "unexpected": True,
                    },
                )
            )
        with self.subTest(operation="createContactPoint", case="blank role"):
            self.assert_bad_request(
                self.gateway.post(
                    "/contact-points",
                    json={
                        "email": f"blank-{suffix}@example.org",
                        "role": " ",
                    },
                )
            )
        with self.subTest(operation="createContactPoint", case="invalid email"):
            self.assert_bad_request(
                self.gateway.post(
                    "/contact-points",
                    json={"email": "not-an-email", "role": "validation"},
                )
            )

        valid_draft_payload = {
            "resource_type": "dataset",
            "revision": draft["revision"],
            "metadata": {
                "title": f"validation-replace-{suffix}",
                "description": "Valid replacement description",
            },
        }
        with self.subTest(operation="replaceResourceDraftVersion", case="missing revision"):
            payload = dict(valid_draft_payload)
            payload.pop("revision")
            self.assert_bad_request(
                self.gateway.put(
                    "/resources/{id}/versions/{versionId}/draft",
                    id=draft_resource_id,
                    versionId=draft["id"],
                    json=payload,
                )
            )
        with self.subTest(operation="replaceResourceDraftVersion", case="unknown field"):
            payload = dict(valid_draft_payload)
            payload["unexpected"] = True
            self.assert_bad_request(
                self.gateway.put(
                    "/resources/{id}/versions/{versionId}/draft",
                    id=draft_resource_id,
                    versionId=draft["id"],
                    json=payload,
                )
            )
        with self.subTest(operation="replaceResourceDraftVersion", case="invalid discriminator"):
            payload = dict(valid_draft_payload)
            payload["resource_type"] = "invalid"
            self.assert_bad_request(
                self.gateway.put(
                    "/resources/{id}/versions/{versionId}/draft",
                    id=draft_resource_id,
                    versionId=draft["id"],
                    json=payload,
                )
            )
        with self.subTest(operation="replaceResourceDraftVersion", case="blank metadata"):
            payload = dict(valid_draft_payload)
            payload["metadata"] = {
                "title": " ",
                "description": "Valid replacement description",
            }
            self.assert_bad_request(
                self.gateway.put(
                    "/resources/{id}/versions/{versionId}/draft",
                    id=draft_resource_id,
                    versionId=draft["id"],
                    json=payload,
                )
            )

        with self.subTest(operation="rejectResourceVersion", case="missing note"):
            self.assert_bad_request(
                self.gateway.post(
                    "/resources/{resourceType}/{id}/versions/{versionId}/reject",
                    resourceType="dataset",
                    id=submitted_resource_id,
                    versionId=submitted_version_id,
                    json={},
                )
            )
        with self.subTest(operation="rejectResourceVersion", case="empty note"):
            self.assert_bad_request(
                self.gateway.post(
                    "/resources/{resourceType}/{id}/versions/{versionId}/reject",
                    resourceType="dataset",
                    id=submitted_resource_id,
                    versionId=submitted_version_id,
                    json={"note": " "},
                )
            )
        with self.subTest(operation="rejectResourceVersion", case="unknown field"):
            self.assert_bad_request(
                self.gateway.post(
                    "/resources/{resourceType}/{id}/versions/{versionId}/reject",
                    resourceType="dataset",
                    id=submitted_resource_id,
                    versionId=submitted_version_id,
                    json={"note": "valid note", "unexpected": True},
                )
            )

        with self.subTest(operation="requestGroupMembership", case="missing field"):
            self.assert_bad_request(
                ordinary.post(
                    "/groups/{groupId}/membership-requests",
                    groupId=membership_group_id,
                    json={},
                )
            )
        with self.subTest(operation="requestGroupMembership", case="unknown field"):
            self.assert_bad_request(
                ordinary.post(
                    "/groups/{groupId}/membership-requests",
                    groupId=membership_group_id,
                    json={"roles": [], "unexpected": True},
                )
            )
        with self.subTest(operation="requestGroupMembership", case="invalid enum"):
            self.assert_bad_request(
                ordinary.post(
                    "/groups/{groupId}/membership-requests",
                    groupId=membership_group_id,
                    json={"roles": ["INVALID"]},
                )
            )

        with self.subTest(operation="requestGroupRole", case="missing field"):
            self.assert_bad_request(
                ordinary.post(
                    "/groups/{groupId}/role-requests",
                    groupId=group_id,
                    json={},
                )
            )
        with self.subTest(operation="requestGroupRole", case="unknown field"):
            self.assert_bad_request(
                ordinary.post(
                    "/groups/{groupId}/role-requests",
                    groupId=group_id,
                    json={"role": "EDITOR", "unexpected": True},
                )
            )
        with self.subTest(operation="requestGroupRole", case="invalid enum"):
            self.assert_bad_request(
                ordinary.post(
                    "/groups/{groupId}/role-requests",
                    groupId=group_id,
                    json={"role": "INVALID"},
                )
            )

    def test_discovery_query_validation(self) -> None:
        """Verify discovery rejects malformed and inconsistent query filters."""

        cases = [
            {"groupId": "not-a-uuid"},
            {"dateFrom": "not-a-date"},
            {
                "dateFrom": "2021-01-01T00:00:00Z",
                "dateTo": "2020-01-01T00:00:00Z",
            },
        ]
        for params in cases:
            with self.subTest(params=params):
                self.assert_bad_request(self.gateway.get("/search", params=params))
