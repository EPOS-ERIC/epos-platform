import os
import unittest
from uuid import uuid4

from auth import get_token
from client import SCHEMA, GatewayClient
from fixtures import assert_problem


class GroupAndCategoryContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def test_group_and_category_crud(self) -> None:
        """Verify admin group/category CRUD, normalization, and cascade deletion."""

        suffix = str(uuid4())
        group_name = f"contract-group-{suffix}"
        updated_group_name = f"{group_name}-updated"
        group_description = "Contract test group"
        updated_group_description = "Updated contract test group"

        created_group_response = self.gateway.post(
            "/groups",
            json={
                "name": group_name,
                "description": group_description,
                "concept": "DATA",
            },
        )
        self.assertEqual(created_group_response.status_code, 201)
        created_group = created_group_response.json()
        self.assertEqual(created_group["name"], group_name)
        self.assertEqual(created_group["description"], group_description)
        self.assertEqual(created_group["concept"], "DATA")
        group_id = created_group["id"]

        groups_response = self.gateway.get("/groups")
        self.assertEqual(groups_response.status_code, 200)
        groups = groups_response.json()
        self.assertIsInstance(groups, list)
        matching_groups = [
            group
            for group in groups
            if isinstance(group, dict) and group.get("id") == group_id
        ]
        self.assertEqual(len(matching_groups), 1)
        self.assertEqual(matching_groups[0]["name"], group_name)

        group_response = self.gateway.get(
            "/groups/{groupId}",
            groupId=group_id,
        )
        self.assertEqual(group_response.status_code, 200)
        self.assertEqual(group_response.json()["id"], group_id)
        self.assertEqual(group_response.json()["name"], group_name)

        updated_group_response = self.gateway.put(
            "/groups/{groupId}",
            groupId=group_id,
            json={
                "name": updated_group_name,
                "description": updated_group_description,
                "concept": "DATA",
            },
        )
        self.assertEqual(updated_group_response.status_code, 200)
        updated_group = updated_group_response.json()
        self.assertEqual(updated_group["id"], group_id)
        self.assertEqual(updated_group["name"], updated_group_name)
        self.assertEqual(updated_group["description"], updated_group_description)

        root_name = f"contract-root-{suffix}"
        root_response = self.gateway.post(
            "/groups/{groupId}/categories",
            groupId=group_id,
            json={"name": root_name},
        )
        self.assertEqual(root_response.status_code, 201)
        root = root_response.json()
        self.assertEqual(root["group_id"], group_id)
        self.assertIsNone(root["parent_category_id"])
        self.assertEqual(root["name"], root_name)
        root_id = root["id"]

        child_name = f"contract-child-{suffix}"
        child_response = self.gateway.post(
            "/groups/{groupId}/categories",
            groupId=group_id,
            json={
                "name": child_name,
                "parent_category_id": root_id,
            },
        )
        self.assertEqual(child_response.status_code, 201)
        child = child_response.json()
        self.assertEqual(child["group_id"], group_id)
        self.assertEqual(child["parent_category_id"], root_id)
        self.assertEqual(child["name"], child_name)
        child_id = child["id"]

        categories_response = self.gateway.get(
            "/groups/{groupId}/categories",
            groupId=group_id,
        )
        self.assertEqual(categories_response.status_code, 200)
        categories = categories_response.json()
        self.assertIsInstance(categories, list)
        categories_by_id = {
            category["id"]: category
            for category in categories
            if isinstance(category, dict)
        }
        self.assertEqual(
            set(categories_by_id),
            {root_id, child_id},
        )
        self.assertIsNone(categories_by_id[root_id]["parent_category_id"])
        self.assertEqual(categories_by_id[child_id]["parent_category_id"], root_id)

        for category_id, name, parent_id in (
            (root_id, root_name, None),
            (child_id, child_name, root_id),
        ):
            category_response = self.gateway.get(
                "/groups/{groupId}/categories/{categoryId}",
                groupId=group_id,
                categoryId=category_id,
            )
            self.assertEqual(category_response.status_code, 200)
            category = category_response.json()
            self.assertEqual(category["id"], category_id)
            self.assertEqual(category["group_id"], group_id)
            self.assertEqual(category["parent_category_id"], parent_id)
            self.assertEqual(category["name"], name)

        updated_child_name = f"{child_name}-updated"
        updated_child_response = self.gateway.put(
            "/groups/{groupId}/categories/{categoryId}",
            groupId=group_id,
            categoryId=child_id,
            json={
                "name": updated_child_name,
                "parent_category_id": root_id,
            },
        )
        self.assertEqual(updated_child_response.status_code, 200)
        updated_child = updated_child_response.json()
        self.assertEqual(updated_child["id"], child_id)
        self.assertEqual(updated_child["group_id"], group_id)
        self.assertEqual(updated_child["parent_category_id"], root_id)
        self.assertEqual(updated_child["name"], updated_child_name)

        duplicate_response = self.gateway.post(
            "/groups/{groupId}/categories",
            groupId=group_id,
            json={"name": root_name.upper()},
        )
        assert_problem(self, duplicate_response, 409, "conflict")

        deleted_root_response = self.gateway.delete(
            "/groups/{groupId}/categories/{categoryId}",
            groupId=group_id,
            categoryId=root_id,
        )
        self.assertEqual(deleted_root_response.status_code, 204)

        for category_id in (root_id, child_id):
            deleted_category_response = self.gateway.get(
                "/groups/{groupId}/categories/{categoryId}",
                groupId=group_id,
                categoryId=category_id,
            )
            self.assertEqual(deleted_category_response.status_code, 404)

        empty_categories_response = self.gateway.get(
            "/groups/{groupId}/categories",
            groupId=group_id,
        )
        self.assertEqual(empty_categories_response.status_code, 200)
        self.assertEqual(empty_categories_response.json(), [])

        deleted_group_response = self.gateway.delete(
            "/groups/{groupId}",
            groupId=group_id,
        )
        self.assertEqual(deleted_group_response.status_code, 204)

        deleted_group_get_response = self.gateway.get(
            "/groups/{groupId}",
            groupId=group_id,
        )
        self.assertEqual(deleted_group_get_response.status_code, 404)
