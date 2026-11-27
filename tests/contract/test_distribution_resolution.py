from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, parameter, web_distribution
from test_dynamic_web_services import coordinate, dynamic_distribution
from test_wfs import wfs_distribution


class DistributionResolutionContractTest(DistributionContractTestCase):
    def test_monitoring_prepares_all_service_types(self) -> None:
        """Verify monitoring uses stored defaults for each web-service type."""
        wms = {
            "title": "monitoring-wms", "description": "Synthetic WMS", "type": "WEB_SERVICE",
            "web_service": {
                "type": "WMS", "capabilities_url": "https://provider.invalid/wms",
                "default_layer": "layer", "layers": [{"name": "layer", "default_style": "", "styles": [""]}],
                "transparency_editable": False, "default_transparency": False,
                "get_map_parameters": [], "get_feature_info_parameters": [],
            },
        }
        wmts = {
            "title": "monitoring-wmts", "description": "Synthetic WMTS", "type": "WEB_SERVICE",
            "web_service": {
                "type": "WMTS", "capabilities_url": "https://provider.invalid/wmts",
                "default_layer": "layer", "layers": [{"name": "layer"}],
                "get_tile_parameters": [], "get_feature_info_parameters": [],
            },
        }
        dynamic = dynamic_distribution([
            coordinate("lon", "LONGITUDE", default=12.5),
            coordinate("lat", "LATITUDE", position=1, default=34.5),
        ])
        dynamic["title"] = "monitoring-dynamic"
        detail = self.create_draft([dynamic, wfs_distribution(), wms, wmts])
        ids = {item["title"]: item["id"] for item in detail["metadata"]["distributions"]}
        reloaded = self.details(detail["resource_id"], detail["id"])
        self.assertEqual(
            {item["title"]: item["web_service"]["type"] for item in reloaded["metadata"]["distributions"]},
            {"monitoring-dynamic": "DYNAMIC", "WFS": "WFS", "monitoring-wms": "WMS", "monitoring-wmts": "WMTS"},
        )
        self.transition(detail, "submit")
        self.transition(detail, "publish")
        response = self.gateway.get("/monitoring/distributions")
        self.assertEqual(response.status_code, 200)
        targets = {
            item["version_id"]: item
            for concept in response.json()
            for group in concept["groups"]
            for item in group["distributions"]
        }
        self.assertIn("lon=12.5", targets[ids["monitoring-dynamic"]]["request"]["url"])
        self.assertIn("lat=34.5", targets[ids["monitoring-dynamic"]]["request"]["url"])
        self.assertIn("typeNames=example%3Afaults", targets[ids["WFS"]]["request"]["url"])
        self.assertEqual(targets[ids["monitoring-wms"]]["request"]["url"], "https://provider.invalid/wms")
        self.assertEqual(targets[ids["monitoring-wmts"]]["request"]["url"], "https://provider.invalid/wmts")

    def test_monitoring_lists_only_published_distributions(self) -> None:
        """Verify monitoring targets follow publication and never execute providers."""
        file = {
            "title": "monitoring file", "description": "Synthetic file", "type": "DOWNLOADABLE_FILE",
            "downloadable_file": {"url": "https://provider.invalid/file", "format": "text/plain"},
        }
        detail = self.create_draft([web_distribution(), file])
        distributions = detail["metadata"]["distributions"]
        ids = {distribution["id"] for distribution in distributions}

        def targets() -> dict[str, dict]:
            response = self.gateway.get("/monitoring/distributions")
            self.assertEqual(response.status_code, 200)
            return {
                distribution["version_id"]: distribution
                for concept in response.json()
                for group in concept["groups"]
                for distribution in group["distributions"]
                if distribution["version_id"] in ids
            }

        self.assertEqual(targets(), {})
        self.transition(detail, "submit")
        self.assertEqual(targets(), {})
        self.transition(detail, "publish")
        published = targets()
        self.assertEqual(set(published), ids)
        for distribution in distributions:
            target = published[distribution["id"]]
            self.assertEqual(target["name"], distribution["title"])
            self.assertEqual(target["resource_id"], distribution["resource_id"])
            self.assertEqual(target["request"]["method"], "GET")
            self.assertTrue(target["request"]["url"].startswith("https://provider.invalid/"))
        self.transition(detail, "unpublish")
        self.assertEqual(targets(), {})

    def test_monitoring_replaces_updated_distribution(self) -> None:
        """Catch monitoring targets changing identity or retaining archived distribution versions."""
        file = {
            "title": "original file", "description": "Synthetic file", "type": "DOWNLOADABLE_FILE",
            "downloadable_file": {"url": "https://provider.invalid/original", "format": "text/plain"},
        }
        detail = self.create_draft([file])
        original = detail["metadata"]["distributions"][0]
        stable_id = original["resource_id"]

        def targets() -> list[dict]:
            response = self.gateway.get("/monitoring/distributions")
            self.assertEqual(response.status_code, 200)
            return [
                item
                for concept in response.json()
                for group in concept["groups"]
                for item in group["distributions"]
                if item["resource_id"] == stable_id
            ]

        self.transition(detail, "submit")
        self.transition(detail, "publish")
        published = targets()
        self.assertEqual(len(published), 1)
        self.assertEqual(published[0]["version_id"], original["id"])
        self.assertEqual(published[0]["name"], "original file")
        self.assertNotIn("distribution_id", published[0])

        copied = self.gateway.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset", id=detail["resource_id"],
        )
        self.assertEqual(copied.status_code, 201)
        draft_version = self.version(detail["resource_id"], "DRAFT")
        draft = self.details(detail["resource_id"], draft_version["id"])
        copied_distribution = draft["metadata"]["distributions"][0]
        self.assertEqual(copied_distribution["resource_id"], stable_id)
        self.assertNotEqual(copied_distribution["id"], original["id"])
        updated_file = {
            **file,
            "resource_id": stable_id,
            "title": "updated file",
            "downloadable_file": {"url": "https://provider.invalid/updated", "format": "text/plain"},
        }
        metadata = {
            "title": draft["metadata"]["title"],
            "description": draft["metadata"]["description"],
            "distributions": [updated_file],
        }
        self.assertEqual(self.replace_response(draft, metadata).status_code, 200)
        updated = self.details(detail["resource_id"], draft["id"])
        self.assertEqual(targets()[0]["version_id"], original["id"])
        self.transition(updated, "submit")
        self.transition(updated, "publish")

        published = targets()
        self.assertEqual(len(published), 1)
        self.assertEqual(published[0]["resource_id"], stable_id)
        self.assertEqual(published[0]["version_id"], copied_distribution["id"])
        self.assertEqual(published[0]["name"], "updated file")
        self.assertEqual(published[0]["request"]["url"], "https://provider.invalid/updated")
        self.assertNotEqual(published[0]["version_id"], original["id"])
        self.assertNotIn("distribution_id", published[0])

    def test_monitoring_uses_distribution_owner_contacts(self) -> None:
        """Verify service and file monitoring targets use their respective contacts."""
        suffix = uuid4().hex
        contacts = []
        for owner in ("service", "dataset"):
            response = self.gateway.post(
                "/contact-points", json={"email": f"{owner}-{suffix}@example.org"},
            )
            self.assertEqual(response.status_code, 201)
            contacts.append(response.json()["id"])
        file = {
            "title": "contact file", "description": "Synthetic file", "type": "DOWNLOADABLE_FILE",
            "downloadable_file": {"url": "https://provider.invalid/file", "format": "text/plain"},
        }
        detail = self.create_draft([web_distribution(contact_points=[contacts[0]]), file])
        metadata = {
            "title": detail["metadata"]["title"], "description": "Monitoring contacts",
            "contact_points": [contacts[1]],
            "distributions": [web_distribution(contact_points=[contacts[0]]), file],
        }
        self.assertEqual(self.replace_response(detail, metadata).status_code, 200)
        detail = self.details(detail["resource_id"], detail["id"])
        distributions = {item["type"]: item for item in detail["metadata"]["distributions"]}
        self.transition(detail, "submit")
        self.transition(detail, "publish")
        response = self.gateway.get("/monitoring/distributions")
        self.assertEqual(response.status_code, 200)
        targets = {
            item["version_id"]: item
            for concept in response.json()
            for group in concept["groups"]
            for item in group["distributions"]
        }
        self.assertEqual(
            [item["id"] for item in targets[distributions["WEB_SERVICE"]["id"]]["contact_points"]],
            [contacts[0]],
        )
        self.assertEqual(
            [item["id"] for item in targets[distributions["DOWNLOADABLE_FILE"]["id"]]["contact_points"]],
            [contacts[1]],
        )

    def test_preparation_visibility_follows_version_lifecycle(self) -> None:
        """Verify prepare authorization across draft, review, publication and archive."""
        detail = self.create_draft([web_distribution()])
        distribution = detail["metadata"]["distributions"][0]
        member, member_id = self.create_user("member")
        unrelated, _ = self.create_user("unrelated")
        group_id = detail["group_id"]
        requested = member.post(
            "/groups/{groupId}/membership-requests", groupId=group_id, json={"roles": []},
        )
        self.assertEqual(requested.status_code, 201)
        accepted = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}", groupId=group_id,
            userId=member_id, decision="accept",
        )
        self.assertEqual(accepted.status_code, 200)
        self.assertEqual(accepted.json()["status"], "ACTIVE")

        def assert_private() -> None:
            expected = self.prepare(distribution)
            self.assertEqual(self.prepare(distribution, gateway=member), expected)
            for gateway, headers in (
                (self.gateway, {"Authorization": None}), (unrelated, None),
            ):
                self.assert_prepare_problem(
                    self.prepare_response(distribution, gateway=gateway, headers=headers),
                    404, "not_found",
                )

        assert_private()
        self.transition(detail, "submit")
        assert_private()
        self.transition(detail, "reject", json={"note": "Preparation contract review"})
        assert_private()
        self.transition(detail, "reopen")
        self.transition(detail, "submit")
        self.transition(detail, "publish")
        expected = self.prepare(distribution)
        self.assertEqual(
            self.prepare(distribution, headers={"Authorization": None}), expected,
        )
        self.assertEqual(self.prepare(distribution, gateway=unrelated), expected)
        self.transition(detail, "unpublish")
        archived = self.details(detail["resource_id"], detail["id"])
        self.assertEqual(archived["status"], "ARCHIVED")
        self.assertEqual(archived["metadata"]["distributions"][0]["id"], distribution["id"])
        assert_private()

    def test_distribution_identifiers_and_type(self) -> None:
        """Verify malformed, missing, removed and non-web-service distribution IDs."""
        file = {
            "title": "file", "description": "Synthetic file", "type": "DOWNLOADABLE_FILE",
            "downloadable_file": {"url": "https://provider.invalid/file", "format": "text/plain"},
        }
        detail = self.create_draft([web_distribution(), file])
        by_type = {item["type"]: item for item in detail["metadata"]["distributions"]}
        for identifier, status, code in (
            ("invalid-uuid", 400, "validation_failed"),
            (str(uuid4()), 404, "not_found"),
        ):
            with self.subTest(identifier=identifier):
                self.assert_prepare_problem(
                    self.prepare_response({"id": identifier}), status, code,
                )
        self.assert_prepare_problem(
            self.prepare_response(by_type["DOWNLOADABLE_FILE"]), target="distribution_id",
        )
        original = by_type["WEB_SERVICE"]
        metadata = {
            "title": detail["metadata"]["title"], "description": "Replaced distribution",
            "distributions": [web_distribution(template_url="https://provider.invalid/replacement")],
        }
        self.assertEqual(self.replace_response(detail, metadata).status_code, 200)
        replaced = self.details(detail["resource_id"], detail["id"])
        replacement = replaced["metadata"]["distributions"][0]
        self.assertNotEqual(replacement["id"], original["id"])
        self.assert_prepare_problem(self.prepare_response(original), 404, "not_found")
        self.assertEqual(self.prepare(replacement)["url"], "https://provider.invalid/replacement")

    def test_publication_and_draft_keep_distinct_requests_and_parameter_ids(self) -> None:
        """Verify copied distribution IDs and requests cannot resolve another version."""
        detail = self.create_draft([web_distribution([parameter(default="published")])])
        original = detail["metadata"]["distributions"][0]
        for action in ("submit", "publish"):
            self.transition(detail, action)
        copied = self.gateway.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset", id=detail["resource_id"],
        )
        self.assertEqual(copied.status_code, 201)
        version = self.version(detail["resource_id"], "DRAFT")
        draft = self.details(detail["resource_id"], version["id"])
        duplicate = draft["metadata"]["distributions"][0]
        self.assertNotEqual(duplicate["id"], original["id"])
        original_parameter = original["web_service"]["parameters"][0]["id"]
        copied_parameter = duplicate["web_service"]["parameters"][0]["id"]
        self.assertNotEqual(original_parameter, copied_parameter)
        for distribution, foreign_id in ((original, copied_parameter), (duplicate, original_parameter)):
            with self.subTest(distribution=distribution["id"]):
                self.assert_prepare_problem(
                    self.prepare_response(distribution, {foreign_id: "foreign"}),
                    target="parameters[0].id",
                )
        metadata = {
            "title": draft["metadata"]["title"], "description": "Independent draft",
            "distributions": [web_distribution(
                [parameter(default="draft")], template_url="https://provider.invalid/draft",
            )],
        }
        self.assertEqual(self.replace_response(draft, metadata).status_code, 200)
        updated = self.details(draft["resource_id"], draft["id"])
        self.assertEqual(
            self.prepare(updated["metadata"]["distributions"][0])["url"],
            "https://provider.invalid/draft?value=draft",
        )
        self.assertEqual(
            self.prepare(original, headers={"Authorization": None})["url"],
            "https://provider.invalid/query?value=published",
        )
