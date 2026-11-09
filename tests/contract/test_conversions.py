from typing import Any
from uuid import uuid4

from distribution_fixtures import DistributionContractTestCase, web_distribution


class ConversionsContractTest(DistributionContractTestCase):
    def plugin_payload(self) -> dict[str, Any]:
        return {
            "name": f"contract-conversion-{uuid4().hex}",
            "repository": "https://example.invalid/contract-plugin.git",
            "branch": "main",
            "runtime": "python",
            "arguments": ["--contract-test"],
        }

    def create_plugin(self) -> tuple[dict[str, Any], dict[str, Any]]:
        payload = self.plugin_payload()
        response = self.gateway.post("/conversion-plugins", json=payload)
        self.assertEqual(response.status_code, 201)
        return payload, response.json()

    def assert_rejected(self, response: Any) -> None:
        self.assertGreaterEqual(response.status_code, 400)
        self.assertLess(response.status_code, 500)
        self.assertEqual(
            response.headers["Content-Type"].split(";")[0], "application/problem+json"
        )

    def test_plugin_crud(self) -> None:
        """Verify plugin settings persist through creation, replacement, and deletion."""
        payload, created = self.create_plugin()
        plugin_id = created["id"]
        self.assertTrue(created["enabled"])
        for key, value in payload.items():
            self.assertEqual(created[key], value)

        fetched = self.gateway.get("/conversion-plugins/{pluginId}", pluginId=plugin_id)
        self.assertEqual(fetched.status_code, 200)
        self.assertEqual(fetched.json()["name"], payload["name"])
        listed = self.gateway.get("/conversion-plugins")
        self.assertEqual(listed.status_code, 200)
        self.assertIn(plugin_id, [item["id"] for item in listed.json()])

        replacement = {**payload, "name": f"{payload['name']}-updated", "enabled": False}
        updated = self.gateway.put(
            "/conversion-plugins/{pluginId}", pluginId=plugin_id, json=replacement,
        )
        self.assertEqual(updated.status_code, 200)
        self.assertEqual(updated.json()["id"], plugin_id)
        self.assertEqual(updated.json()["created_by"], created["created_by"])
        for key, value in replacement.items():
            self.assertEqual(updated.json()[key], value)
        fetched = self.gateway.get("/conversion-plugins/{pluginId}", pluginId=plugin_id)
        self.assertEqual(fetched.status_code, 200)
        self.assertEqual(fetched.json()["name"], replacement["name"])
        self.assertFalse(fetched.json()["enabled"])

        deleted = self.gateway.delete("/conversion-plugins/{pluginId}", pluginId=plugin_id)
        self.assertEqual(deleted.status_code, 204)
        self.assert_rejected(self.gateway.get("/conversion-plugins/{pluginId}", pluginId=plugin_id))
        listed = self.gateway.get("/conversion-plugins")
        self.assertEqual(listed.status_code, 200)
        self.assertNotIn(plugin_id, [item["id"] for item in listed.json()])

    def test_failed_installation_and_sync(self) -> None:
        """Verify failed installs and syncs preserve plugin settings and safe errors."""
        payload = self.plugin_payload()
        response = self.gateway.post("/conversion-plugins", json=payload)
        self.assertEqual(response.status_code, 201)
        plugin = response.json()
        self.assertFalse(plugin["installed"])
        self.assertIsNone(plugin["installed_commit"])
        plugin_id = plugin["id"]

        sync = self.gateway.post("/conversion-plugins/{pluginId}/sync", pluginId=plugin_id)
        self.assertGreaterEqual(sync.status_code, 400)
        self.assertEqual(sync.headers["Content-Type"].split(";")[0], "application/problem+json")
        problem = sync.json()
        for text in ("rpc error", "dial tcp", "SQLSTATE", payload["repository"]):
            self.assertNotIn(text, problem.get("detail", ""))
        fetched = self.gateway.get("/conversion-plugins/{pluginId}", pluginId=plugin_id)
        self.assertEqual(fetched.status_code, 200)
        self.assertFalse(fetched.json()["installed"])

        settings = {**payload, "enabled": True}
        for field, value in (
            ("repository", "https://another.example.invalid/plugin.git"),
            ("branch", "other-branch"),
            ("runtime", "java"),
        ):
            with self.subTest(field=field):
                settings[field] = value
                updated = self.gateway.put(
                    "/conversion-plugins/{pluginId}", pluginId=plugin_id, json=settings,
                )
                self.assertEqual(updated.status_code, 200)
                self.assertEqual(updated.json()[field], value)
                self.assertFalse(updated.json()["installed"])
                self.assertIsNone(updated.json()["installed_commit"])
                fetched = self.gateway.get("/conversion-plugins/{pluginId}", pluginId=plugin_id)
                self.assertEqual(fetched.status_code, 200)
                self.assertEqual(fetched.json()[field], value)

    def test_plugin_validation_and_missing_ids(self) -> None:
        """Verify repository and branch validation, runtimes, argument order, and missing IDs."""
        payload = self.plugin_payload()
        for field, value in (
            ("repository", "not a URL"),
            ("repository", "https://user:password@example.invalid/plugin.git"),
            ("repository", "file:///contract/plugin.git"),
            ("branch", ""),
            ("branch", "-bad-branch"),
        ):
            with self.subTest(field=field, value=value):
                self.assert_rejected(self.gateway.post(
                    "/conversion-plugins", json={**payload, field: value},
                ))
        for runtime in ("python", "java", "binary"):
            with self.subTest(runtime=runtime):
                args = ["third", "first", "second"]
                created = self.gateway.post("/conversion-plugins", json={
                    **self.plugin_payload(), "runtime": runtime, "arguments": args,
                })
                self.assertEqual(created.status_code, 201)
                self.assertEqual(created.json()["runtime"], runtime)
                self.assertEqual(created.json()["arguments"], args)
                fetched = self.gateway.get(
                    "/conversion-plugins/{pluginId}", pluginId=created.json()["id"],
                )
                self.assertEqual(fetched.status_code, 200)
                self.assertEqual(fetched.json()["arguments"], args)
        missing = str(uuid4())
        for method, path, kwargs in (
            ("GET", "/conversion-plugins/{pluginId}", {}),
            ("PUT", "/conversion-plugins/{pluginId}", {"json": {**payload, "enabled": True}}),
            ("DELETE", "/conversion-plugins/{pluginId}", {}),
            ("POST", "/conversion-plugins/{pluginId}/sync", {}),
        ):
            with self.subTest(method=method, path=path):
                response = self.gateway.request(method, path, pluginId=missing, **kwargs)
                self.assertEqual(response.status_code, 404)
                self.assert_rejected(response)

    def test_relation_crud_and_plugin_deletion(self) -> None:
        """Verify relation CRUD, uniqueness, native format validation, and cascade deletion."""
        _, plugin = self.create_plugin()
        detail = self.create_draft([web_distribution()])
        distribution = detail["metadata"]["distributions"][0]
        payload = {
            "distribution_id": distribution["id"],
            "plugin_id": plugin["id"],
            "input_format": "application/json",
            "output_format": "text/csv",
        }
        created = self.gateway.post("/conversion-relations", json=payload)
        self.assertEqual(created.status_code, 201)
        relation = created.json()
        relation_id = relation["id"]
        for key, value in payload.items():
            self.assertEqual(relation[key], value)
        self.assert_rejected(self.gateway.post("/conversion-relations", json=payload))
        self.assert_rejected(self.gateway.post(
            "/conversion-relations", json={**payload, "input_format": "text/not-native"},
        ))

        fetched = self.gateway.get(
            "/conversion-relations/{conversionId}", conversionId=relation_id,
        )
        self.assertEqual(fetched.status_code, 200)
        self.assertEqual(fetched.json()["output_format"], payload["output_format"])
        listed = self.gateway.get("/conversion-relations")
        self.assertEqual(listed.status_code, 200)
        self.assertIn(relation_id, [item["id"] for item in listed.json()])

        replacement = {**payload, "output_format": "application/xml"}
        updated = self.gateway.put(
            "/conversion-relations/{conversionId}", conversionId=relation_id, json=replacement,
        )
        self.assertEqual(updated.status_code, 200)
        self.assertEqual(updated.json()["id"], relation_id)
        self.assertEqual(updated.json()["created_by"], relation["created_by"])
        for key, value in replacement.items():
            self.assertEqual(updated.json()[key], value)
        fetched = self.gateway.get(
            "/conversion-relations/{conversionId}", conversionId=relation_id,
        )
        self.assertEqual(fetched.status_code, 200)
        self.assertEqual(fetched.json()["output_format"], replacement["output_format"])

        deleted = self.gateway.delete(
            "/conversion-relations/{conversionId}", conversionId=relation_id,
        )
        self.assertEqual(deleted.status_code, 204)
        self.assert_rejected(self.gateway.get(
            "/conversion-relations/{conversionId}", conversionId=relation_id,
        ))
        listed = self.gateway.get("/conversion-relations")
        self.assertEqual(listed.status_code, 200)
        self.assertNotIn(relation_id, [item["id"] for item in listed.json()])

        created = self.gateway.post("/conversion-relations", json=payload)
        self.assertEqual(created.status_code, 201)
        relation_id = created.json()["id"]
        deleted = self.gateway.delete("/conversion-plugins/{pluginId}", pluginId=plugin["id"])
        self.assertEqual(deleted.status_code, 204)
        self.assert_rejected(self.gateway.get(
            "/conversion-relations/{conversionId}", conversionId=relation_id,
        ))
        listed = self.gateway.get("/conversion-relations")
        self.assertEqual(listed.status_code, 200)
        self.assertNotIn(relation_id, [item["id"] for item in listed.json()])

    def test_distribution_plugin_relations(self) -> None:
        """Verify grouped relations, distribution isolation, and dataset read authorization."""
        payload, plugin = self.create_plugin()
        _, other_plugin = self.create_plugin()
        second = web_distribution()
        second["title"] = "other-distribution"
        detail = self.create_draft([web_distribution(), second])
        distributions = {item["title"]: item for item in detail["metadata"]["distributions"]}
        distribution_id = distributions["prepared-service"]["id"]
        other_id = distributions["other-distribution"]["id"]
        path = "/distributions/{distributionId}/conversion-relations"

        empty = self.gateway.get(path, distributionId=distribution_id)
        self.assertEqual(empty.status_code, 200)
        self.assertEqual(empty.json(), [])

        updated = self.gateway.put(
            "/conversion-plugins/{pluginId}", pluginId=plugin["id"],
            json={**payload, "enabled": False},
        )
        self.assertEqual(updated.status_code, 200)
        self.assertFalse(updated.json()["installed"])
        relation_ids = []
        for output_format in ("text/csv", "application/xml"):
            response = self.gateway.post("/conversion-relations", json={
                "distribution_id": distribution_id, "plugin_id": plugin["id"],
                "input_format": "application/json", "output_format": output_format,
            })
            self.assertEqual(response.status_code, 201)
            relation_ids.append(response.json()["id"])
        for target_id, plugin_id in ((distribution_id, other_plugin["id"]), (other_id, plugin["id"])):
            response = self.gateway.post("/conversion-relations", json={
                "distribution_id": target_id, "plugin_id": plugin_id,
                "input_format": "application/json", "output_format": "text/csv",
            })
            self.assertEqual(response.status_code, 201)

        listed = self.gateway.get(path, distributionId=distribution_id)
        self.assertEqual(listed.status_code, 200)
        plugins = {item["id"]: item for item in listed.json()}
        self.assertEqual(set(plugins), {plugin["id"], other_plugin["id"]})
        result = plugins[plugin["id"]]
        self.assertEqual(set(result), {
            "id", "name", "description", "repository", "enabled", "installed",
            "branch", "runtime", "arguments", "relations",
        })
        for key, value in payload.items():
            self.assertEqual(result[key], value)
        self.assertEqual(result["description"], "")
        self.assertFalse(result["enabled"])
        self.assertFalse(result["installed"])
        self.assertCountEqual([item["id"] for item in result["relations"]], relation_ids)
        self.assertCountEqual(
            [item["output_format"] for item in result["relations"]],
            ["text/csv", "application/xml"],
        )
        for relation in result["relations"]:
            self.assertEqual(set(relation), {"id", "distribution_id", "input_format", "output_format"})
            self.assertEqual(relation["distribution_id"], distribution_id)
            self.assertEqual(relation["input_format"], "application/json")
        self.assertEqual(len(plugins[other_plugin["id"]]["relations"]), 1)

        member, member_id = self.create_user("conversion-reader")
        unrelated, _ = self.create_user("conversion-unrelated")
        requested = member.post(
            "/groups/{groupId}/membership-requests", groupId=detail["group_id"], json={"roles": []},
        )
        self.assertEqual(requested.status_code, 201)
        accepted = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}", groupId=detail["group_id"],
            userId=member_id, decision="accept",
        )
        self.assertEqual(accepted.status_code, 200)
        self.assertIn(member.get("/conversion-plugins").status_code, (401, 403))
        visible = member.get(path, distributionId=distribution_id)
        self.assertEqual(visible.status_code, 200)
        self.assertEqual({item["id"] for item in visible.json()}, set(plugins))
        self.assertEqual(unrelated.get(path, distributionId=distribution_id).status_code, 404)
        self.assertEqual(self.gateway.get(
            path, distributionId=distribution_id, headers={"Authorization": None},
        ).status_code, 401)
        self.assertEqual(self.gateway.get(path, distributionId=str(uuid4())).status_code, 404)
        self.assertEqual(self.gateway.get(path, distributionId="invalid-uuid").status_code, 400)

    def test_uninstalled_and_disabled_plugin_formats_are_hidden(self) -> None:
        """Verify relations remain readable but unavailable for uninstalled plugins."""
        payload, plugin = self.create_plugin()
        self.assertFalse(plugin["installed"])
        detail = self.create_draft([web_distribution()])
        distribution = detail["metadata"]["distributions"][0]
        relation = self.gateway.post("/conversion-relations", json={
            "distribution_id": distribution["id"],
            "plugin_id": plugin["id"],
            "input_format": "application/json",
            "output_format": "text/csv",
        })
        self.assertEqual(relation.status_code, 201)
        relation_id = relation.json()["id"]

        for enabled in (True, False):
            with self.subTest(enabled=enabled):
                if not enabled:
                    updated = self.gateway.put(
                        "/conversion-plugins/{pluginId}", pluginId=plugin["id"],
                        json={**payload, "enabled": False},
                    )
                    self.assertEqual(updated.status_code, 200)
                    self.assertFalse(updated.json()["enabled"])
                fetched = self.gateway.get(
                    "/conversion-relations/{conversionId}", conversionId=relation_id,
                )
                self.assertEqual(fetched.status_code, 200)
                refreshed = self.details(detail["resource_id"], detail["id"])
                formats = refreshed["metadata"]["distributions"][0]["available_formats"]
                self.assertTrue(any(
                    item["kind"] == "ORIGINAL" and item["output_format"] == "application/json"
                    for item in formats
                ))
                self.assertFalse(any(
                    item.get("conversion_id") == relation_id for item in formats
                ))

    def test_relation_validation_and_removed_distribution(self) -> None:
        """Verify lifecycle propagation; regression: removed distributions left orphaned relations."""
        plugin_input, plugin = self.create_plugin()
        detail = self.create_draft([web_distribution(), web_distribution(), web_distribution()])
        distribution_id = detail["metadata"]["distributions"][0]["id"]
        retained = detail["metadata"]["distributions"][1]
        retained_other = detail["metadata"]["distributions"][2]
        payload = {
            "distribution_id": distribution_id, "plugin_id": plugin["id"],
            "input_format": "application/json", "output_format": "text/csv",
        }
        for invalid in (
            {**payload, "output_format": "not a mime type"},
            {**payload, "plugin_id": str(uuid4())},
            {**payload, "distribution_id": str(uuid4())},
            {**payload, "input_format": "text/not-native"},
        ):
            with self.subTest(payload=invalid):
                self.assert_rejected(self.gateway.post("/conversion-relations", json=invalid))

        first = self.gateway.post("/conversion-relations", json=payload)
        self.assertEqual(first.status_code, 201)
        self.assert_rejected(self.gateway.post("/conversion-relations", json=payload))
        second = self.gateway.post(
            "/conversion-relations", json={**payload, "output_format": "application/xml"},
        )
        self.assertEqual(second.status_code, 201)
        self.assert_rejected(self.gateway.put(
            "/conversion-relations/{conversionId}", conversionId=second.json()["id"],
            json=payload,
        ))
        unchanged = self.gateway.get(
            "/conversion-relations/{conversionId}", conversionId=second.json()["id"],
        )
        self.assertEqual(unchanged.status_code, 200)
        self.assertEqual(unchanged.json()["output_format"], "application/xml")

        kept_relation = self.gateway.post("/conversion-relations", json={
            **payload, "distribution_id": retained["id"],
        })
        self.assertEqual(kept_relation.status_code, 201)
        extra_relations = []
        for distribution, output_format in (
            (retained, "application/xml"),
            (retained_other, "application/geo+json"),
        ):
            response = self.gateway.post("/conversion-relations", json={
                **payload, "distribution_id": distribution["id"], "output_format": output_format,
            })
            self.assertEqual(response.status_code, 201)
            extra_relations.append(response.json())

        metadata = {
            "title": detail["metadata"]["title"], "description": "Removed distribution",
            "distributions": [
                {**web_distribution(), "resource_id": item["resource_id"]}
                for item in (retained, retained_other)
            ],
        }
        self.assertEqual(self.replace_response(detail, metadata).status_code, 200)
        relation_ids = [first.json()["id"], second.json()["id"]]
        for relation_id in relation_ids:
            fetched = self.gateway.get(
                "/conversion-relations/{conversionId}", conversionId=relation_id,
            )
            self.assertEqual(fetched.status_code, 404)
        self.assertEqual(self.gateway.get(
            "/conversion-relations/{conversionId}", conversionId=kept_relation.json()["id"],
        ).status_code, 200)
        access = self.gateway.post(
            "/distributions/{distributionId}/access", distributionId=distribution_id,
            params={"conversion_id": relation_ids[0]}, json={"type": "HTTP", "parameters": []},
        )
        self.assertEqual(access.status_code, 404)
        self.assert_rejected(access)

        disabled = self.gateway.put(
            "/conversion-plugins/{pluginId}", pluginId=plugin["id"],
            json={**plugin_input, "enabled": False},
        )
        self.assertEqual(disabled.status_code, 200)
        source_relations = [kept_relation.json(), *extra_relations]

        def relation_snapshot(distribution_ids: set[str]) -> set[tuple[str, ...]]:
            response = self.gateway.get("/conversion-relations")
            self.assertEqual(response.status_code, 200)
            return {
                tuple(item[key] for key in ("id", "distribution_id", "plugin_id", "input_format", "output_format"))
                for item in response.json() if item["distribution_id"] in distribution_ids
            }

        source_ids = {retained["id"], retained_other["id"]}
        expected_source = {
            tuple(item[key] for key in ("id", "distribution_id", "plugin_id", "input_format", "output_format"))
            for item in source_relations
        }
        detail = self.details(detail["resource_id"], detail["id"])
        self.transition(detail, "submit")
        self.transition(detail, "publish")
        self.assertEqual(relation_snapshot(source_ids), expected_source)
        source = self.details(detail["resource_id"], detail["id"])
        source_distributions = source["metadata"]["distributions"]
        self.assertEqual({item["id"] for item in source_distributions}, source_ids)

        copied = self.gateway.post(
            "/groups/{groupId}/resources/{resourceType}/{id}/copy",
            groupId=detail["group_id"], resourceType="dataset", id=detail["resource_id"],
        )
        self.assertEqual(copied.status_code, 201)
        copy_id = copied.json()["id"]
        copy_version = self.version(copy_id, "DRAFT")
        copy_distributions = self.details(copy_id, copy_version["id"])["metadata"]["distributions"]
        self.assertTrue({item["resource_id"] for item in copy_distributions}.isdisjoint(
            {item["resource_id"] for item in source_distributions},
        ))

        drafted = self.gateway.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset", id=detail["resource_id"],
        )
        self.assertEqual(drafted.status_code, 201)
        draft = self.version(detail["resource_id"], "DRAFT")
        draft_distributions = self.details(detail["resource_id"], draft["id"])["metadata"]["distributions"]
        listed = self.gateway.get("/conversion-relations")
        self.assertEqual(listed.status_code, 200)
        relations = [item for item in listed.json() if item["plugin_id"] == plugin["id"]]

        def signatures(distribution_id: str) -> set[tuple[str, str, str]]:
            return {
                (item["plugin_id"], item["input_format"], item["output_format"])
                for item in relations if item["distribution_id"] == distribution_id
            }

        expected = {item["resource_id"]: signatures(item["id"]) for item in source_distributions}
        self.assertEqual(len(expected), 2)
        self.assertEqual(sorted(map(len, expected.values())), [1, 2])
        self.assertEqual({item["resource_id"] for item in draft_distributions}, set(expected))
        for item in draft_distributions:
            self.assertEqual(signatures(item["id"]), expected[item["resource_id"]])
        self.assertEqual(len(copy_distributions), 2)
        self.assertCountEqual(
            [signatures(item["id"]) for item in copy_distributions], list(expected.values()),
        )
        cloned_ids = {item["id"] for item in copy_distributions + draft_distributions}
        cloned_relations = [item for item in relations if item["distribution_id"] in cloned_ids]
        self.assertEqual(len(cloned_relations), 6)
        self.assertEqual(len({item["id"] for item in cloned_relations + source_relations}), 9)

        discarded = self.gateway.delete(
            "/resources/{resourceType}/{id}/versions/{versionId}/draft",
            resourceType="dataset", id=detail["resource_id"], versionId=draft["id"],
        )
        self.assertEqual(discarded.status_code, 204)
        for item in cloned_relations:
            if item["distribution_id"] in {distribution["id"] for distribution in draft_distributions}:
                self.assertEqual(self.gateway.get(
                    "/conversion-relations/{conversionId}", conversionId=item["id"],
                ).status_code, 404)
        self.transition(source, "unpublish")
        self.assertEqual(relation_snapshot(source_ids), expected_source)
        archived_draft = self.gateway.post(
            "/resources/{resourceType}/{id}/draft-from-archive",
            resourceType="dataset", id=detail["resource_id"],
        )
        self.assertEqual(archived_draft.status_code, 201)
        archive_version = self.version(detail["resource_id"], "DRAFT")
        archive_distributions = self.details(detail["resource_id"], archive_version["id"])["metadata"]["distributions"]
        listed = self.gateway.get("/conversion-relations")
        self.assertEqual(listed.status_code, 200)
        relations = [item for item in listed.json() if item["plugin_id"] == plugin["id"]]
        self.assertEqual({item["resource_id"] for item in archive_distributions}, set(expected))
        for item in archive_distributions:
            self.assertEqual(signatures(item["id"]), expected[item["resource_id"]])
        archive_relations = [item for item in relations if item["distribution_id"] in {d["id"] for d in archive_distributions}]
        self.assertEqual(len(archive_relations), 3)
        self.assertEqual(len({item["id"] for item in archive_relations + cloned_relations + source_relations}), 12)

        deleted = self.gateway.delete("/conversion-plugins/{pluginId}", pluginId=plugin["id"])
        self.assertEqual(deleted.status_code, 204)
        listed = self.gateway.get("/conversion-relations")
        self.assertEqual(listed.status_code, 200)
        self.assertFalse(set(relation_ids) & {item["id"] for item in listed.json()})
        for relation_id in relation_ids:
            self.assert_rejected(self.gateway.get(
                "/conversion-relations/{conversionId}", conversionId=relation_id,
            ))

    def test_invalid_conversion_requests(self) -> None:
        """Verify invalid plugin and relation request shapes are rejected."""
        payload = self.plugin_payload()
        for invalid in (
            {key: value for key, value in payload.items() if key != "name"},
            {**payload, "name": ""},
            {**payload, "runtime": "unknown"},
            {**payload, "unexpected": True},
        ):
            with self.subTest(payload=invalid):
                self.assert_rejected(self.gateway.post("/conversion-plugins", json=invalid))
        relation = {
            "distribution_id": str(uuid4()), "plugin_id": str(uuid4()),
            "input_format": "application/json", "output_format": "text/csv",
        }
        for invalid in (
            {key: value for key, value in relation.items() if key != "plugin_id"},
            {**relation, "distribution_id": "not-a-uuid"},
            {**relation, "input_format": ""},
            {**relation, "unexpected": True},
        ):
            with self.subTest(payload=invalid):
                self.assert_rejected(self.gateway.post("/conversion-relations", json=invalid))

    def test_conversion_access_selectors_and_permissions(self) -> None:
        """Verify conversion selectors and private distribution access are validated."""
        _, plugin = self.create_plugin()
        second = web_distribution()
        second["title"] = "other-distribution"
        detail = self.create_draft([web_distribution(), second])
        distributions = {item["title"]: item for item in detail["metadata"]["distributions"]}
        distribution = distributions["prepared-service"]
        other = distributions["other-distribution"]
        relation = self.gateway.post("/conversion-relations", json={
            "distribution_id": distribution["id"], "plugin_id": plugin["id"],
            "input_format": "application/json", "output_format": "text/csv",
        })
        self.assertEqual(relation.status_code, 201)
        relation_id = relation.json()["id"]
        body = {"type": "HTTP", "parameters": []}
        for params in (
            {"conversion_id": "not-a-uuid"},
            {"conversion_id": [relation_id, relation_id]},
            {"conversion_id": str(uuid4())},
            {"conversion_id": relation_id, "output_format": "application/json"},
        ):
            with self.subTest(params=params):
                response = self.gateway.post(
                    "/distributions/{distributionId}/access", distributionId=distribution["id"],
                    params=params, json=body,
                )
                self.assert_rejected(response)
        prepared = self.prepare_response(distribution, params={"conversion_id": relation_id})
        self.assert_rejected(prepared)

        foreign = self.gateway.post(
            "/distributions/{distributionId}/access", distributionId=other["id"],
            params={"conversion_id": relation_id}, json=body,
        )
        self.assert_rejected(foreign)
        unrelated, _ = self.create_user("conversion-access-denied")
        for client, headers in ((unrelated, None), (self.gateway, {"Authorization": ""})):
            response = client.post(
                "/distributions/{distributionId}/access", distributionId=distribution["id"],
                params={"conversion_id": relation_id}, json=body, headers=headers,
            )
            self.assertEqual(response.status_code, 404)
            self.assert_rejected(response)

    def test_conversion_authorization(self) -> None:
        """Verify conversion operations require global converter management permission."""
        denied, user_id = self.create_user("conversion-denied")
        payload, plugin = self.create_plugin()
        detail = self.create_draft([web_distribution()])
        relation_payload = {
            "distribution_id": detail["metadata"]["distributions"][0]["id"],
            "plugin_id": plugin["id"],
            "input_format": "application/json",
            "output_format": "text/csv",
        }
        relation = self.gateway.post("/conversion-relations", json=relation_payload)
        self.assertEqual(relation.status_code, 201)
        plugin_id = plugin["id"]
        relation_id = relation.json()["id"]
        calls = [
            ("GET", "/conversion-plugins", {}),
            ("POST", "/conversion-plugins", {"json": payload}),
            ("GET", "/conversion-plugins/{pluginId}", {"pluginId": plugin_id}),
            ("PUT", "/conversion-plugins/{pluginId}", {
                "pluginId": plugin_id, "json": {**payload, "enabled": False},
            }),
            ("DELETE", "/conversion-plugins/{pluginId}", {"pluginId": plugin_id}),
            ("POST", "/conversion-plugins/{pluginId}/sync", {"pluginId": plugin_id}),
            ("GET", "/conversion-relations", {}),
            ("POST", "/conversion-relations", {"json": relation_payload}),
            ("GET", "/conversion-relations/{conversionId}", {"conversionId": relation_id}),
            ("PUT", "/conversion-relations/{conversionId}", {
                "conversionId": relation_id, "json": relation_payload,
            }),
            ("DELETE", "/conversion-relations/{conversionId}", {"conversionId": relation_id}),
        ]
        for method, path, kwargs in calls:
            for client, headers in ((denied, None), (self.gateway, {"Authorization": ""})):
                with self.subTest(method=method, path=path, anonymous=headers is not None):
                    response = client.request(method, path, headers=headers, **kwargs)
                    self.assert_rejected(response)
                    self.assertIn(response.status_code, (401, 403))

        private = denied.get(
            "/resources/{resourceType}/{id}/versions/{versionId}", resourceType="dataset",
            id=detail["resource_id"], versionId=detail["id"],
        )
        self.assertEqual(private.status_code, 404)
        group_id = detail["group_id"]
        membership = denied.post(
            "/groups/{groupId}/membership-requests", groupId=group_id, json={"roles": []},
        )
        self.assertEqual(membership.status_code, 201)
        accepted = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id, userId=user_id, decision="accept",
        )
        self.assertEqual(accepted.status_code, 200)
        assigned = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id, userId=user_id, role="EDITOR",
        )
        self.assertEqual(assigned.status_code, 204)
        self.transition(detail, "submit")
        self.transition(detail, "publish")
        drafted = denied.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset", id=detail["resource_id"],
        )
        self.assertEqual(drafted.status_code, 201)
        draft_version = self.version(detail["resource_id"], "DRAFT")
        draft_distribution = self.details(detail["resource_id"], draft_version["id"])["metadata"]["distributions"][0]
        propagated = self.gateway.get("/conversion-relations")
        self.assertTrue(any(item["distribution_id"] == draft_distribution["id"] and item["plugin_id"] == plugin_id for item in propagated.json()))
        self.assertEqual(denied.delete(
            "/resources/{resourceType}/{id}/versions/{versionId}/draft",
            resourceType="dataset", id=detail["resource_id"], versionId=draft_version["id"],
        ).status_code, 204)
        propagated = self.gateway.get("/conversion-relations")
        self.assertFalse(any(item["distribution_id"] == draft_distribution["id"] for item in propagated.json()))

        granted = self.gateway.put(
            "/users/{id}/global-permissions/{permission}",
            id=user_id, permission="global.converter.manage",
        )
        self.assertEqual(granted.status_code, 204)
        for path in ("/conversion-plugins", "/conversion-relations"):
            with self.subTest(granted=path):
                self.assertEqual(denied.get(path).status_code, 200)

        plugin_update = denied.put(
            "/conversion-plugins/{pluginId}", pluginId=plugin_id,
            json={**payload, "name": f"{payload['name']}-manager", "enabled": False},
        )
        self.assertEqual(plugin_update.status_code, 200)
        self.assertEqual(plugin_update.json()["created_by"], plugin["created_by"])
        self.assertEqual(plugin_update.json()["updated_by"], user_id)
        self.assertNotEqual(plugin_update.json()["created_by"], user_id)

        relation_update = denied.put(
            "/conversion-relations/{conversionId}", conversionId=relation_id,
            json={**relation_payload, "output_format": "application/xml"},
        )
        self.assertEqual(relation_update.status_code, 200)
        self.assertEqual(relation_update.json()["created_by"], relation.json()["created_by"])
        self.assertEqual(relation_update.json()["updated_by"], user_id)
        self.assertNotEqual(relation_update.json()["created_by"], user_id)
        created = denied.post("/conversion-relations", json={
            **relation_payload, "output_format": "application/geo+json",
        })
        self.assertEqual(created.status_code, 201)
        removed = denied.delete(
            "/conversion-relations/{conversionId}", conversionId=created.json()["id"],
        )
        self.assertEqual(removed.status_code, 204)

        revoked = self.gateway.delete(
            "/users/{id}/global-permissions/{permission}",
            id=user_id, permission="global.converter.manage",
        )
        self.assertEqual(revoked.status_code, 204)
        for method, path, kwargs in (
            ("GET", "/conversion-plugins", {}),
            ("PUT", "/conversion-plugins/{pluginId}", {
                "pluginId": plugin_id, "json": {**payload, "enabled": False},
            }),
            ("POST", "/conversion-relations", {"json": relation_payload}),
        ):
            with self.subTest(revoked=path, method=method):
                response = denied.request(method, path, **kwargs)
                self.assert_rejected(response)
                self.assertIn(response.status_code, (401, 403))
