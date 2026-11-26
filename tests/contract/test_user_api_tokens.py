import os
import unittest
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, create_group


class UserAPITokenContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.base_url = os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1")
        cls.admin = GatewayClient(SCHEMA, cls.base_url, get_token())

    def create_user(self) -> tuple[GatewayClient, str]:
        credentials = register_user(
            name="Token",
            surname="Contract",
            email=f"contract-token-{uuid4().hex}@example.org",
            password=uuid4().hex,
        )
        client = GatewayClient(
            SCHEMA,
            self.base_url,
            get_token(username=credentials.username, password=credentials.password),
        )
        profile = client.get("/users/{id}", id="self")
        self.assertEqual(profile.status_code, 200)
        return client, profile.json()["id"]

    def issue_token(self, client: GatewayClient) -> str:
        response = client.post("/users/self/api-token")
        self.assertEqual(response.status_code, 201)
        self.assertEqual(response.headers.get("Cache-Control"), "no-store")
        token = response.json()["token"]
        self.assertTrue(token.startswith("epos_pat_"))
        return token

    def token_client(self, token: str) -> GatewayClient:
        return GatewayClient(SCHEMA, self.base_url, token)

    def test_token_lifecycle(self) -> None:
        """Verify issue, rotation, and revocation of a user's token."""

        oauth, user_id = self.create_user()
        assert_problem(self, oauth.post("/users/self/api-token/rotate"), 404, "not_found")

        token = self.issue_token(oauth)
        bearer = self.token_client(token)
        self.assertEqual(bearer.get("/users/{id}", id="self").json()["id"], user_id)
        assert_problem(self, oauth.post("/users/self/api-token"), 409, "conflict")

        rotated = oauth.post("/users/self/api-token/rotate")
        self.assertEqual(rotated.status_code, 200)
        self.assertEqual(rotated.headers.get("Cache-Control"), "no-store")
        new_token = rotated.json()["token"]
        self.assertTrue(new_token.startswith("epos_pat_"))
        self.assertTrue(token != new_token)
        assert_problem(self, bearer.get("/users/{id}", id="self"), 401, "unauthenticated")
        new_bearer = self.token_client(new_token)
        self.assertEqual(new_bearer.get("/users/{id}", id="self").json()["id"], user_id)

        self.assertEqual(oauth.delete("/users/self/api-token").status_code, 204)
        assert_problem(self, new_bearer.get("/users/{id}", id="self"), 401, "unauthenticated")
        self.assertEqual(oauth.delete("/users/self/api-token").status_code, 204)
        self.assertEqual(oauth.get("/users/{id}", id="self").status_code, 200)

    def test_credential_boundaries(self) -> None:
        """Verify token-management restrictions and secret-free user responses."""

        oauth, user_id = self.create_user()
        _, other_id = self.create_user()
        token = self.issue_token(oauth)
        bearer = self.token_client(token)

        assert_problem(self, bearer.post("/users/self/api-token"), 403, "permission_denied")
        assert_problem(self, bearer.post("/users/self/api-token/rotate"), 403, "permission_denied")
        assert_problem(self, bearer.delete("/users/self/api-token"), 403, "permission_denied")
        assert_problem(
            self,
            bearer.delete("/users/{userId}/api-token", userId=other_id),
            403,
            "permission_denied",
        )

        profile = oauth.get("/users/{id}", id="self")
        listing = self.admin.get("/users")
        self.assertEqual(profile.status_code, 200)
        self.assertEqual(listing.status_code, 200)
        self.assertEqual(profile.json()["id"], user_id)
        for response in (profile, listing):
            self.assertFalse(token in response.text)
            self.assertFalse("api_token_digest" in response.text)
        self.assertNotIn("api_token", profile.json())
        for user in listing.json():
            self.assertNotIn("api_token", user)

        assert_problem(
            self,
            oauth.get("/users/{id}", id="self", headers={"Authorization": ""}),
            401,
            "unauthenticated",
        )
        for invalid in ("epos_pat_invalid", f"epos_pat_{uuid4().hex}"):
            response = self.token_client(invalid).get("/users/{id}", id="self")
            assert_problem(self, response, 401, "unauthenticated")

        self.assertEqual(oauth.delete("/users/self/api-token").status_code, 204)
        assert_problem(self, bearer.get("/users/{id}", id="self"), 401, "unauthenticated")

    def test_cross_user_revocation(self) -> None:
        """Verify cross-user revocation requires the global revoke permission."""

        operator, operator_id = self.create_user()
        target, target_id = self.create_user()
        token = self.issue_token(target)
        bearer = self.token_client(token)
        path = "/users/{userId}/api-token"

        assert_problem(
            self, operator.delete(path, userId=target_id), 403, "permission_denied"
        )
        self.assertEqual(bearer.get("/users/{id}", id="self").status_code, 200)

        permission = "global.user_api_token.revoke"
        catalog = self.admin.get("/permissions")
        self.assertEqual(catalog.status_code, 200)
        self.assertTrue(
            any(item["key"] == permission and item["scope"] == "GLOBAL" for item in catalog.json())
        )
        granted = self.admin.put(
            "/users/{id}/global-permissions/{permission}",
            id=operator_id,
            permission=permission,
        )
        self.assertEqual(granted.status_code, 204)
        profile = operator.get("/users/{id}", id="self")
        self.assertEqual(profile.status_code, 200)
        self.assertIn(permission, profile.json()["global_granted_permissions"])
        users = self.admin.get("/users")
        self.assertEqual(users.status_code, 200)
        listed_operator = next(user for user in users.json() if user["id"] == operator_id)
        self.assertIn(permission, listed_operator["global_granted_permissions"])

        self.assertEqual(operator.delete(path, userId=target_id).status_code, 204)
        assert_problem(self, bearer.get("/users/{id}", id="self"), 401, "unauthenticated")
        self.assertEqual(operator.delete(path, userId=target_id).status_code, 204)
        assert_problem(self, operator.delete(path, userId=str(uuid4())), 404, "not_found")

    def test_block_and_unblock_revokes_token(self) -> None:
        """Verify blocking permanently revokes the user's existing token."""

        oauth, user_id = self.create_user()
        token = self.issue_token(oauth)
        bearer = self.token_client(token)

        blocked = self.admin.post(
            "/users/{id}/block", id=user_id, json={"reason": "API token contract test"}
        )
        self.assertEqual(blocked.status_code, 204)
        assert_problem(self, bearer.get("/users/{id}", id="self"), 401, "unauthenticated")
        self.assertEqual(self.admin.post("/users/{id}/unblock", id=user_id).status_code, 204)
        assert_problem(self, bearer.get("/users/{id}", id="self"), 401, "unauthenticated")
        self.assertEqual(oauth.get("/users/{id}", id="self").status_code, 200)
        new_token = self.issue_token(oauth)
        self.assertTrue(new_token != token)
        self.assertEqual(
            self.token_client(new_token).get("/users/{id}", id="self").status_code, 200
        )

    def test_token_uses_current_permissions_for_resource_workflow(self) -> None:
        """Verify token requests follow current group permissions and workflow rules."""

        oauth, user_id = self.create_user()
        group_id = create_group(self.admin, f"contract-token-group-{uuid4()}")
        requested = oauth.post(
            "/groups/{groupId}/membership-requests", groupId=group_id, json={"roles": []}
        )
        self.assertEqual(requested.status_code, 201)
        accepted = self.admin.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="accept",
        )
        self.assertEqual(accepted.status_code, 200)

        permission_path = "/groups/{groupId}/memberships/{userId}/permissions/{permission}"
        for permission in ("group.resource.create", "group.resource.edit", "group.resource.submit"):
            response = self.admin.put(
                permission_path, groupId=group_id, userId=user_id, permission=permission
            )
            self.assertEqual(response.status_code, 204)

        bearer = self.token_client(self.issue_token(oauth))
        created = bearer.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {"title": f"contract-token-{uuid4()}", "description": "Token workflow"},
            },
        )
        self.assertEqual(created.status_code, 201)
        resource_id = created.json()["id"]
        versions = bearer.get(
            "/resources/{resourceType}/{id}/versions", resourceType="dataset", id=resource_id
        )
        self.assertEqual(versions.status_code, 200)
        drafts = [version for version in versions.json() if version["status"] == "DRAFT"]
        self.assertEqual(len(drafts), 1)
        version = drafts[0]

        edited = bearer.put(
            "/resources/{id}/versions/{versionId}/draft",
            id=resource_id,
            versionId=version["id"],
            json={
                "resource_type": "dataset",
                "revision": version["revision"],
                "metadata": {"title": "Contract token edited", "description": "Token workflow"},
            },
        )
        self.assertEqual(edited.status_code, 200)
        permission = "group.resource.submit"
        removed = self.admin.delete(
            permission_path, groupId=group_id, userId=user_id, permission=permission
        )
        self.assertEqual(removed.status_code, 204)
        submit_path = "/resources/{resourceType}/{id}/versions/{versionId}/submit"
        submit_args = {"resourceType": "dataset", "id": resource_id, "versionId": version["id"]}
        assert_problem(self, bearer.post(submit_path, **submit_args), 403, "permission_denied")
        restored = self.admin.put(
            permission_path, groupId=group_id, userId=user_id, permission=permission
        )
        self.assertEqual(restored.status_code, 204)
        submitted = bearer.post(submit_path, **submit_args)
        self.assertEqual(submitted.status_code, 200)
        self.assertEqual(submitted.json()["status"], "SUBMITTED")
