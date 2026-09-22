import os
import unittest
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient


NO_AUTH_HEADERS = {"Authorization": ""}


class SharingContractTest(unittest.TestCase):
    """Visibility, ownership and official-flag policy for configurations."""

    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def secondary_user(self, label: str) -> GatewayClient:
        """Register a throwaway non-admin user and return its client."""
        aai_url = os.getenv("AAI_URL", "http://localhost:8110")
        email = f"sharing-{uuid4().hex}@example.org"
        password = uuid4().hex
        credentials = register_user(
            aai_url,
            name="Sharing",
            surname=label,
            email=email,
            password=password,
        )
        return GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(
                aai_url=aai_url,
                username=credentials.username,
                password=credentials.password,
            ),
        )

    def configuration(self) -> dict:
        return {
            "configurables": [{"type": "mapZoom", "value": "11"}],
            "dataSearchConfigurables": [],
        }

    def test_private_visibility_lifecycle(self) -> None:
        """Private shares are visible to the owner and admins only."""
        owner = self.secondary_user("Owner")
        stranger = self.secondary_user("Stranger")
        suffix = uuid4().hex[:8]

        created = owner.post(
            "/configurations",
            json={
                "configuration": self.configuration(),
                "title": f"Private sharing {suffix}",
                "is_public": False,
            },
        )
        self.assertEqual(created.status_code, 201)
        configuration_id = created.json()["id"]
        self.assertFalse(created.json()["is_public"])

        anonymous = owner.get(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            headers=NO_AUTH_HEADERS,
        )
        self.assertEqual(anonymous.status_code, 404)

        anonymous_list = owner.get("/configurations", headers=NO_AUTH_HEADERS)
        self.assertEqual(anonymous_list.status_code, 200)
        self.assertNotIn(
            configuration_id,
            [share["id"] for share in anonymous_list.json()["configurations"]],
        )

        stranger_get = stranger.get(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(stranger_get.status_code, 404)

        stranger_replace = stranger.put(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            json={"configuration": {"hijacked": True}},
        )
        self.assertEqual(stranger_replace.status_code, 404)

        owner_get = owner.get(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(owner_get.status_code, 200)

        admin_get = self.gateway.get(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(admin_get.status_code, 200)

        replaced = owner.put(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            json={"configuration": {"hello": "world"}, "is_public": True},
        )
        self.assertEqual(replaced.status_code, 200)
        self.assertTrue(replaced.json()["is_public"])

        now_public = stranger.get(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(now_public.status_code, 200)

        stranger_replace_public = stranger.put(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            json={"configuration": {"hijacked": True}},
        )
        self.assertEqual(stranger_replace_public.status_code, 403)

        deleted = owner.delete(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(deleted.status_code, 204)

    def test_official_requires_admin(self) -> None:
        """Only administrators may create official configurations."""
        user = self.secondary_user("Unofficial")
        suffix = uuid4().hex[:8]

        rejected = user.post(
            "/configurations",
            json={
                "configuration": self.configuration(),
                "title": f"Fake official {suffix}",
                "is_official": True,
            },
        )
        self.assertEqual(rejected.status_code, 403)

        created = self.gateway.post(
            "/configurations",
            json={
                "configuration": self.configuration(),
                "title": f"Official sharing {suffix}",
                "is_official": True,
            },
        )
        self.assertEqual(created.status_code, 201)
        configuration_id = created.json()["id"]
        self.assertTrue(created.json()["is_official"])
        self.assertTrue(created.json()["is_public"])

        official_only = self.gateway.get(
            "/configurations", params={"official": True}
        )
        self.assertEqual(official_only.status_code, 200)
        self.assertIn(
            configuration_id,
            [share["id"] for share in official_only.json()["configurations"]],
        )

        unofficial_only = self.gateway.get(
            "/configurations", params={"official": False}
        )
        self.assertEqual(unofficial_only.status_code, 200)
        self.assertNotIn(
            configuration_id,
            [share["id"] for share in unofficial_only.json()["configurations"]],
        )

        deleted = self.gateway.delete(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(deleted.status_code, 204)

    def test_official_cannot_be_private(self) -> None:
        """Official configurations must stay public."""
        response = self.gateway.post(
            "/configurations",
            json={
                "configuration": self.configuration(),
                "is_official": True,
                "is_public": False,
            },
        )
        self.assertEqual(response.status_code, 400)

    def test_duplicate_configuration_id_conflicts(self) -> None:
        """Creating two configurations with the same id returns 409."""
        configuration_id = str(uuid4())
        first = self.gateway.post(
            "/configurations",
            json={"id": configuration_id, "configuration": {"a": 1}},
        )
        self.assertEqual(first.status_code, 201)

        duplicate = self.gateway.post(
            "/configurations",
            json={"id": configuration_id, "configuration": {"a": 1}},
        )
        self.assertEqual(duplicate.status_code, 409)

        cleanup = self.gateway.delete(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(cleanup.status_code, 204)

    def test_writes_require_authentication(self) -> None:
        """POST, PUT and DELETE without a token return 401."""
        created = self.gateway.post(
            "/configurations", json={"configuration": self.configuration()}
        )
        self.assertEqual(created.status_code, 201)
        configuration_id = created.json()["id"]

        for response in (
            self.gateway.post(
                "/configurations",
                json={"configuration": self.configuration()},
                headers=NO_AUTH_HEADERS,
            ),
            self.gateway.put(
                "/configurations/{configurationId}",
                configurationId=configuration_id,
                json={"configuration": self.configuration()},
                headers=NO_AUTH_HEADERS,
            ),
            self.gateway.delete(
                "/configurations/{configurationId}",
                configurationId=configuration_id,
                headers=NO_AUTH_HEADERS,
            ),
        ):
            self.assertEqual(response.status_code, 401)

        cleanup = self.gateway.delete(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(cleanup.status_code, 204)


if __name__ == "__main__":
    unittest.main()
