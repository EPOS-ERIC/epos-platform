import os
import unittest
from concurrent.futures import ThreadPoolExecutor
from threading import Barrier
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient


NO_AUTH_HEADERS = {"Authorization": ""}


class SharingContractTest(unittest.TestCase):
    """Personal ownership and permission-based configuration management."""

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

    def test_private_visibility_and_unified_detail(self) -> None:
        """Visibility defaults, ownership, replacement, and private-safe list behavior."""
        owner = self.secondary_user("Owner")
        stranger = self.secondary_user("Stranger")
        created_records = []
        for label, visibility in (
            ("omitted", {}),
            ("null", {"is_public": None}),
            ("false", {"is_public": False}),
        ):
            created = owner.post(
                "/configurations",
                json={
                    "configuration": self.configuration(),
                    "title": f"Private {label} {uuid4().hex}",
                    "description": "Clear on replacement",
                    "list_of_services": ["EPOS"],
                    **visibility,
                },
            )
            self.assertEqual(created.status_code, 201)
            self.assertFalse(created.json()["is_public"])
            self.assertNotIn("owner_id", created.json())
            created_records.append(created)

        created = created_records[0]
        self.assertNotIn("Location", created.headers)
        configuration_id = created.json()["id"]

        owner_list = owner.get("/configurations")
        self.assertEqual(owner_list.status_code, 200)
        self.assertIsInstance(owner_list.json(), list)
        own_summary = next(row for row in owner_list.json() if row["id"] == configuration_id)
        self.assertNotIn("configuration", own_summary)
        self.assertNotIn("owner_id", own_summary)
        self.assertEqual(own_summary["description"], "Clear on replacement")
        self.assertEqual(own_summary["list_of_services"], ["EPOS"])

        self.assertEqual(owner.get("/configurations", headers=NO_AUTH_HEADERS).status_code, 401)
        anonymous_private = owner.get(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            headers=NO_AUTH_HEADERS,
        )
        self.assertEqual(anonymous_private.status_code, 404)
        stranger_get = stranger.get(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(stranger_get.status_code, 404)
        stranger_delete = stranger.delete(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(stranger_delete.status_code, 404)
        stranger_replace = stranger.put(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            json={"configuration": {"hijacked": True}, "title": "Hijacked"},
        )
        self.assertEqual(stranger_replace.status_code, 404)

        owner_get = owner.get("/configurations/{configurationId}", configurationId=configuration_id)
        self.assertEqual(owner_get.status_code, 200)
        self.assertEqual(owner_get.json()["configuration"], self.configuration())
        self.assertNotIn("owner_id", owner_get.json())

        for index, visibility in enumerate(({}, {"is_public": None}, {"is_public": False})):
            made_public = owner.put(
                "/configurations/{configurationId}",
                configurationId=configuration_id,
                json={
                    "configuration": self.configuration(),
                    "title": f"Public before reset {index}",
                    "is_public": True,
                },
            )
            self.assertEqual(made_public.status_code, 200)
            self.assertTrue(made_public.json()["is_public"])
            replaced = owner.put(
                "/configurations/{configurationId}",
                configurationId=configuration_id,
                json={
                    "configuration": self.configuration(),
                    "title": f"Private after reset {index}",
                    **visibility,
                },
            )
            self.assertEqual(replaced.status_code, 200)
            self.assertFalse(replaced.json()["is_public"])
            self.assertNotIn("owner_id", replaced.json())
            if index == 0:
                self.assertIsNone(replaced.json()["description"])
                self.assertEqual(replaced.json()["list_of_services"], [])

        made_public = owner.put(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            json={"configuration": self.configuration(), "title": "Public", "is_public": True},
        )
        self.assertEqual(made_public.status_code, 200)
        self.assertNotIn("owner_id", made_public.json())
        public_get = stranger.get("/configurations/{configurationId}", configurationId=configuration_id)
        self.assertEqual(public_get.status_code, 200)
        self.assertNotIn("owner_id", public_get.json())
        anonymous_public = stranger.get(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            headers=NO_AUTH_HEADERS,
        )
        self.assertEqual(anonymous_public.status_code, 200)
        stranger_list = stranger.get("/configurations")
        self.assertNotIn(configuration_id, [row["id"] for row in stranger_list.json()])

        denied = stranger.put(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            json={"configuration": {}, "title": "No access"},
        )
        self.assertEqual(denied.status_code, 403)
        stranger_delete_public = stranger.delete(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(stranger_delete_public.status_code, 403)
        deleted = owner.delete("/configurations/{configurationId}", configurationId=configuration_id)
        self.assertEqual(deleted.status_code, 204)
        for record in created_records[1:]:
            self.assertEqual(
                owner.delete(
                    "/configurations/{configurationId}",
                    configurationId=record.json()["id"],
                ).status_code,
                204,
            )

        missing_id = str(uuid4())
        self.assertEqual(
            owner.get("/configurations/{configurationId}", configurationId=missing_id).status_code,
            404,
        )
        self.assertEqual(
            owner.put(
                "/configurations/{configurationId}",
                configurationId=missing_id,
                json={"configuration": {}, "title": "Missing"},
            ).status_code,
            404,
        )
        self.assertEqual(
            owner.delete("/configurations/{configurationId}", configurationId=missing_id).status_code,
            404,
        )
        self.assertEqual(
            self.gateway.put(
                "/configurations/{configurationId}/official-status",
                configurationId=missing_id,
                json={"is_official": True},
            ).status_code,
            404,
        )

    def test_official_promotion_demotion_and_public_list(self) -> None:
        """Promotion, official access, manager writes, and demotion preserve lifecycle rules."""
        owner = self.secondary_user("Creator")
        owner_profile = owner.get("/users/{id}", id="self")
        self.assertEqual(owner_profile.status_code, 200)
        owner_id = owner_profile.json()["id"]
        manager = self.secondary_user("ConfigurationManager")
        manager_profile = manager.get("/users/{id}", id="self")
        self.assertEqual(manager_profile.status_code, 200)
        manager_id = manager_profile.json()["id"]
        self.assertNotIn("ADMIN", manager_profile.json()["global_roles"])
        self.assertEqual(
            self.gateway.put(
                "/users/{id}/global-permissions/{permission}",
                id=manager_id,
                permission="global.configuration.manage",
            ).status_code,
            204,
        )
        created = owner.post(
            "/configurations",
            json={"configuration": self.configuration(), "title": f"Lifecycle {uuid4().hex}"},
        )
        self.assertEqual(created.status_code, 201)
        configuration_id = created.json()["id"]
        self.assertFalse(created.json()["is_official"])
        self.assertFalse(created.json()["is_public"])

        official_before_promotion = owner.get("/official-configurations", headers=NO_AUTH_HEADERS)
        self.assertEqual(official_before_promotion.status_code, 200)
        self.assertNotIn(configuration_id, [row["id"] for row in official_before_promotion.json()])

        forbidden_promotion = owner.put(
            "/configurations/{configurationId}/official-status",
            configurationId=configuration_id,
            json={"is_official": True},
        )
        self.assertEqual(forbidden_promotion.status_code, 403)

        promoted = manager.put(
            "/configurations/{configurationId}/official-status",
            configurationId=configuration_id,
            json={"is_official": True},
        )
        self.assertEqual(promoted.status_code, 200)
        self.assertTrue(promoted.json()["is_official"])
        self.assertTrue(promoted.json()["is_public"])
        self.assertEqual(promoted.json()["owner_id"], owner_id)
        self.assertEqual(promoted.json()["configuration"], self.configuration())

        forbidden_demotion = owner.put(
            "/configurations/{configurationId}/official-status",
            configurationId=configuration_id,
            json={"is_official": False},
        )
        self.assertEqual(forbidden_demotion.status_code, 403)

        repeated = manager.put(
            "/configurations/{configurationId}/official-status",
            configurationId=configuration_id,
            json={"is_official": True},
        )
        self.assertEqual(repeated.status_code, 200)
        self.assertEqual(repeated.json()["updated_at"], promoted.json()["updated_at"])

        official_list = owner.get("/official-configurations", headers=NO_AUTH_HEADERS)
        self.assertEqual(official_list.status_code, 200)
        self.assertIsInstance(official_list.json(), list)
        listed = next(row for row in official_list.json() if row["id"] == configuration_id)
        self.assertNotIn("owner_id", listed)
        self.assertNotIn("configuration", listed)
        self.assertTrue(listed["is_official"])
        self.assertTrue(listed["is_public"])

        manager_official_list = manager.get("/official-configurations")
        self.assertEqual(manager_official_list.status_code, 200)
        manager_listed = next(row for row in manager_official_list.json() if row["id"] == configuration_id)
        self.assertNotIn("owner_id", manager_listed)
        self.assertTrue(manager_listed["is_official"])
        self.assertTrue(manager_listed["is_public"])

        anonymous_detail = owner.get(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            headers=NO_AUTH_HEADERS,
        )
        self.assertEqual(anonymous_detail.status_code, 200)
        self.assertEqual(anonymous_detail.json()["configuration"], self.configuration())
        self.assertNotIn("owner_id", anonymous_detail.json())
        creator_list = owner.get("/configurations")
        self.assertIn(configuration_id, [row["id"] for row in creator_list.json()])

        for index, visibility in enumerate(({}, {"is_public": None}, {"is_public": True})):
            manager_replacement = manager.put(
                "/configurations/{configurationId}",
                configurationId=configuration_id,
                json={
                    "configuration": {"manager_edit": index},
                    "title": f"Manager official replacement {index}",
                    **visibility,
                },
            )
            self.assertEqual(manager_replacement.status_code, 200)
            self.assertTrue(manager_replacement.json()["is_official"])
            self.assertTrue(manager_replacement.json()["is_public"])
            self.assertEqual(manager_replacement.json()["owner_id"], owner_id)
            self.assertEqual(manager_replacement.json()["configuration"], {"manager_edit": index})

        forbidden_update = owner.put(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            json={"configuration": {}, "title": "Creator cannot edit official"},
        )
        self.assertEqual(forbidden_update.status_code, 403)
        forbidden_delete = owner.delete("/configurations/{configurationId}", configurationId=configuration_id)
        self.assertEqual(forbidden_delete.status_code, 403)
        private_attempt = manager.put(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            json={"configuration": {}, "title": "Cannot privatize", "is_public": False},
        )
        self.assertEqual(private_attempt.status_code, 400)

        manager_delete_target = owner.post(
            "/configurations",
            json={"configuration": {}, "title": f"Manager delete {uuid4().hex}"},
        )
        self.assertEqual(manager_delete_target.status_code, 201)
        manager_delete_id = manager_delete_target.json()["id"]
        self.assertEqual(
            manager.put(
                "/configurations/{configurationId}/official-status",
                configurationId=manager_delete_id,
                json={"is_official": True},
            ).status_code,
            200,
        )
        self.assertEqual(
            manager.delete("/configurations/{configurationId}", configurationId=manager_delete_id).status_code,
            204,
        )

        demoted = manager.put(
            "/configurations/{configurationId}/official-status",
            configurationId=configuration_id,
            json={"is_official": False},
        )
        self.assertEqual(demoted.status_code, 200)
        self.assertFalse(demoted.json()["is_official"])
        self.assertTrue(demoted.json()["is_public"])
        self.assertEqual(demoted.json()["owner_id"], owner_id)
        self.assertEqual(demoted.json()["configuration"], {"manager_edit": 2})
        repeated_demotion = manager.put(
            "/configurations/{configurationId}/official-status",
            configurationId=configuration_id,
            json={"is_official": False},
        )
        self.assertEqual(repeated_demotion.status_code, 200)
        self.assertEqual(repeated_demotion.json()["updated_at"], demoted.json()["updated_at"])

        official_after_demotion = owner.get("/official-configurations", headers=NO_AUTH_HEADERS)
        self.assertEqual(official_after_demotion.status_code, 200)
        self.assertNotIn(configuration_id, [row["id"] for row in official_after_demotion.json()])

        creator_update = owner.put(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            json={"configuration": {}, "title": "Creator edits after demotion"},
        )
        self.assertEqual(creator_update.status_code, 200)
        self.assertFalse(creator_update.json()["is_public"])
        self.assertNotIn("owner_id", creator_update.json())
        self.assertEqual(
            owner.delete("/configurations/{configurationId}", configurationId=configuration_id).status_code,
            204,
        )
        self.assertEqual(
            self.gateway.delete(
                "/users/{id}/global-permissions/{permission}",
                id=manager_id,
                permission="global.configuration.manage",
            ).status_code,
            204,
        )

    def test_manager_filters_and_cross_owner_management(self) -> None:
        """Verified direct permission enables explicit cross-owner lists and mutations only."""
        manager = self.secondary_user("Manager")
        owner = self.secondary_user("ManagedOwner")
        manager_identity = manager.get("/users/{id}", id="self")
        self.assertEqual(manager_identity.status_code, 200)
        manager_id = manager_identity.json()["id"]
        self.assertNotIn("ADMIN", manager_identity.json()["global_roles"])
        grant = self.gateway.put(
            "/users/{id}/global-permissions/{permission}",
            id=manager_id,
            permission="global.configuration.manage",
        )
        self.assertEqual(grant.status_code, 204)

        created = owner.post(
            "/configurations", json={"configuration": {}, "title": "Managed", "is_public": False}
        )
        self.assertEqual(created.status_code, 201)
        configuration_id = created.json()["id"]
        owner_identity = owner.get("/users/{id}", id="self")
        owner_id = owner_identity.json()["id"]

        own_list = manager.get("/configurations")
        self.assertEqual(own_list.status_code, 200)
        self.assertNotIn(configuration_id, [row["id"] for row in own_list.json()])
        owner_list = manager.get("/configurations", params={"owner_id": owner_id})
        self.assertEqual(owner_list.status_code, 200)
        managed_summary = next(row for row in owner_list.json() if row["id"] == configuration_id)
        self.assertEqual(managed_summary["owner_id"], owner_id)
        self.assertNotIn("configuration", managed_summary)
        self.assertEqual({row["owner_id"] for row in owner_list.json()}, {owner_id})
        all_list = manager.get("/configurations", params={"scope": "all"})
        self.assertEqual(all_list.status_code, 200)
        all_managed_summary = next(row for row in all_list.json() if row["id"] == configuration_id)
        self.assertEqual(all_managed_summary["owner_id"], owner_id)

        conflict = manager.get(
            "/configurations", params={"owner_id": owner_id, "scope": "all"}
        )
        self.assertEqual(conflict.status_code, 400)
        ordinary_filter = owner.get("/configurations", params={"scope": "all"})
        self.assertEqual(ordinary_filter.status_code, 403)
        ordinary_owner_filter = owner.get("/configurations", params={"owner_id": owner_id})
        self.assertEqual(ordinary_owner_filter.status_code, 403)

        manager_detail = manager.get(
            "/configurations/{configurationId}", configurationId=configuration_id
        )
        self.assertEqual(manager_detail.status_code, 200)
        self.assertEqual(manager_detail.json()["owner_id"], owner_id)
        managed_update = manager.put(
            "/configurations/{configurationId}",
            configurationId=configuration_id,
            json={"configuration": {"managed": True}, "title": "Managed replacement"},
        )
        self.assertEqual(managed_update.status_code, 200)
        self.assertEqual(managed_update.json()["owner_id"], owner_id)
        self.assertEqual(
            manager.delete("/configurations/{configurationId}", configurationId=configuration_id).status_code,
            204,
        )

        revoke = self.gateway.delete(
            "/users/{id}/global-permissions/{permission}",
            id=manager_id,
            permission="global.configuration.manage",
        )
        self.assertEqual(revoke.status_code, 204)

    def test_quota_is_atomic_and_demotion_never_blocked(self) -> None:
        """Catch quota races and verify demotion may temporarily raise count above 50."""
        owner = self.secondary_user("Quota")
        token = owner.session.headers["Authorization"].removeprefix("Bearer ")
        created_ids = []
        for index in range(49):
            visibility = {"is_public": True} if index == 0 else {}
            created = owner.post(
                "/configurations",
                json={"configuration": {}, "title": f"Quota {index}", **visibility},
            )
            self.assertEqual(created.status_code, 201)
            if index == 0:
                self.assertTrue(created.json()["is_public"])
            created_ids.append(created.json()["id"])

        start = Barrier(2)

        def create_concurrently(index: int):
            client = GatewayClient(
                SCHEMA,
                os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
                token,
            )
            start.wait(timeout=10)
            return client.post(
                "/configurations",
                json={"configuration": {}, "title": f"Concurrent quota {index}"},
            )

        with ThreadPoolExecutor(max_workers=2) as executor:
            concurrent = list(executor.map(create_concurrently, (1, 2)))
        self.assertCountEqual([response.status_code for response in concurrent], [201, 409])
        created_ids.append(next(response.json()["id"] for response in concurrent if response.status_code == 201))
        at_limit = owner.get("/configurations")
        self.assertEqual(at_limit.status_code, 200)
        self.assertEqual(len(at_limit.json()), 50)

        promoted_id = created_ids[0]
        promoted = self.gateway.put(
            "/configurations/{configurationId}/official-status",
            configurationId=promoted_id,
            json={"is_official": True},
        )
        self.assertEqual(promoted.status_code, 200)
        extra = owner.post(
            "/configurations", json={"configuration": {}, "title": "Back to 50"}
        )
        self.assertEqual(extra.status_code, 201)
        created_ids.append(extra.json()["id"])

        demoted = self.gateway.put(
            "/configurations/{configurationId}/official-status",
            configurationId=promoted_id,
            json={"is_official": False},
        )
        self.assertEqual(demoted.status_code, 200)
        self.assertTrue(demoted.json()["is_public"])
        self.assertEqual(
            owner.post("/configurations", json={"configuration": {}, "title": "Over quota"}).status_code,
            409,
        )

        manager_identity = owner.get("/users/{id}", id="self")
        self.assertEqual(manager_identity.status_code, 200)
        owner_id = manager_identity.json()["id"]
        self.assertEqual(
            self.gateway.put(
                "/users/{id}/global-permissions/{permission}",
                id=owner_id,
                permission="global.configuration.manage",
            ).status_code,
            204,
        )
        self.assertEqual(
            owner.post("/configurations", json={"configuration": {}, "title": "Manager over quota"}).status_code,
            409,
        )

        independent_owner = self.secondary_user("IndependentQuota")
        independent_create = independent_owner.post(
            "/configurations",
            json={"configuration": {}, "title": "Independent quota"},
        )
        self.assertEqual(independent_create.status_code, 201)
        self.assertEqual(
            independent_owner.delete(
                "/configurations/{configurationId}",
                configurationId=independent_create.json()["id"],
            ).status_code,
            204,
        )

        self.assertEqual(
            owner.delete("/configurations/{configurationId}", configurationId=created_ids[0]).status_code,
            204,
        )
        self.assertEqual(
            owner.post("/configurations", json={"configuration": {}, "title": "Still at 50"}).status_code,
            409,
        )
        self.assertEqual(
            owner.delete("/configurations/{configurationId}", configurationId=created_ids[1]).status_code,
            204,
        )
        self.assertEqual(
            owner.post("/configurations", json={"configuration": {}, "title": "Below quota"}).status_code,
            201,
        )

    def test_server_generated_ids_and_request_validation(self) -> None:
        """Reject client-controlled fields and invalid required input."""
        user = self.secondary_user("Validation")
        body = {"configuration": {}, "title": f"Generated {uuid4().hex}"}
        rejected_id = user.post("/configurations", json={**body, "id": str(uuid4())})
        self.assertEqual(rejected_id.status_code, 400)
        rejected_owner = user.post("/configurations", json={**body, "owner_id": "other-user"})
        self.assertEqual(rejected_owner.status_code, 400)
        for is_official in (False, True):
            rejected_status = user.post(
                "/configurations", json={**body, "is_official": is_official}
            )
            self.assertEqual(rejected_status.status_code, 400)
        for title in (None, "", " \t\n", "é" * 501):
            invalid_body: dict[str, object] = {"configuration": {}}
            if title is not None:
                invalid_body["title"] = title
            with self.subTest(title=title):
                self.assertEqual(user.post("/configurations", json=invalid_body).status_code, 400)

        created = user.post("/configurations", json=body)
        self.assertEqual(created.status_code, 201)
        configuration_id = created.json()["id"]
        for invalid_field in (
            {"owner_id": "other-user"},
            {"is_official": False},
            {"is_official": True},
        ):
            with self.subTest(replacement_field=invalid_field):
                response = user.put(
                    "/configurations/{configurationId}",
                    configurationId=configuration_id,
                    json={**body, **invalid_field},
                )
                self.assertEqual(response.status_code, 400)
        self.assertEqual(
            user.get("/configurations/{configurationId}", configurationId=configuration_id).json()["title"],
            body["title"],
        )

        for invalid_status in ({}, {"is_official": None}, {"is_official": "true"}):
            with self.subTest(status_body=invalid_status):
                response = user.put(
                    "/configurations/{configurationId}/official-status",
                    configurationId=configuration_id,
                    json=invalid_status,
                )
                self.assertEqual(response.status_code, 400)
        self.assertEqual(
            user.delete("/configurations/{configurationId}", configurationId=configuration_id).status_code,
            204,
        )

    def test_writes_require_authentication(self) -> None:
        """POST, PUT, DELETE and status changes without a bearer token return 401."""
        created = self.gateway.post(
            "/configurations", json={"configuration": self.configuration(), "title": "Auth"}
        )
        self.assertEqual(created.status_code, 201)
        configuration_id = created.json()["id"]

        for response in (
            self.gateway.post(
                "/configurations",
                json={"configuration": self.configuration(), "title": "Auth"},
                headers=NO_AUTH_HEADERS,
            ),
            self.gateway.put(
                "/configurations/{configurationId}",
                configurationId=configuration_id,
                json={"configuration": self.configuration(), "title": "Auth"},
                headers=NO_AUTH_HEADERS,
            ),
            self.gateway.delete(
                "/configurations/{configurationId}",
                configurationId=configuration_id,
                headers=NO_AUTH_HEADERS,
            ),
            self.gateway.put(
                "/configurations/{configurationId}/official-status",
                configurationId=configuration_id,
                json={"is_official": True},
                headers=NO_AUTH_HEADERS,
            ),
        ):
            self.assertEqual(response.status_code, 401)

        self.assertEqual(
            self.gateway.delete("/configurations/{configurationId}", configurationId=configuration_id).status_code,
            204,
        )


if __name__ == "__main__":
    unittest.main()
