from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, parameter, web_distribution


class DistributionResolutionContractTest(DistributionContractTestCase):
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
        archived = self.details(detail["resource_id"], detail["version_id"])
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
        replaced = self.details(detail["resource_id"], detail["version_id"])
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
        draft = self.details(detail["resource_id"], version["version_id"])
        duplicate = draft["metadata"]["distributions"][0]
        self.assertNotEqual(duplicate["id"], original["id"])
        original_parameter = original["web_service"]["configuration"]["parameters"][0]["id"]
        copied_parameter = duplicate["web_service"]["configuration"]["parameters"][0]["id"]
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
        updated = self.details(draft["resource_id"], draft["version_id"])
        self.assertEqual(
            self.prepare(updated["metadata"]["distributions"][0])["url"],
            "https://provider.invalid/draft?value=draft",
        )
        self.assertEqual(
            self.prepare(original, headers={"Authorization": None})["url"],
            "https://provider.invalid/query?value=published",
        )
