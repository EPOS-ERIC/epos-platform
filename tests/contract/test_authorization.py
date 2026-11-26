import os
import unittest
from typing import Any
from uuid import uuid4

from auth import get_token, register_user
from client import SCHEMA, GatewayClient
from fixtures import assert_problem, create_group, concept_id


NO_AUTH_HEADERS = {"Authorization": ""}


class AuthorizationContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def create_user(self, label: str, suffix: str) -> tuple[GatewayClient, str]:
        email = f"ca-{uuid4().hex}@example.org"
        password = uuid4().hex
        credentials = register_user(
            name="Authorization",
            surname=f"{label.title()} {suffix}",
            email=email,
            password=password,
        )
        client = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(
                username=credentials.username,
                password=credentials.password,
            ),
        )
        response = client.get("/users/{id}", id="self")
        self.assertEqual(response.status_code, 200)
        return client, response.json()["id"]

    def grant_global_permission(self, user_id: str, permission: str) -> None:
        response = self.gateway.put(
            "/users/{id}/global-permissions/{permission}",
            id=user_id,
            permission=permission,
        )
        self.assertEqual(response.status_code, 204)

    def grant_group_permission(
        self,
        group_id: str,
        user_id: str,
        permission: str,
    ) -> None:
        response = self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=user_id,
            permission=permission,
        )
        self.assertEqual(response.status_code, 204)

    def accept_membership(self, client: GatewayClient, group_id: str) -> str:
        request = client.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json={"roles": []},
        )
        self.assertEqual(request.status_code, 201)
        user_id = request.json()["user"]["id"]
        response = self.gateway.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=user_id,
            decision="accept",
        )
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["status"], "ACTIVE")
        return user_id

    def create_dataset(self, group_id: str, title: str) -> tuple[str, str]:
        response = self.gateway.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {
                    "title": title,
                    "description": f"{title} description",
                },
            },
        )
        self.assertEqual(response.status_code, 201)
        resource_id = response.json()["id"]
        versions_response = self.gateway.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(versions_response.status_code, 200)
        drafts = [
            version
            for version in versions_response.json()
            if version["status"] == "DRAFT"
        ]
        self.assertEqual(len(drafts), 1)
        return resource_id, drafts[0]["id"]

    def assert_forbidden(self, response: Any) -> None:
        assert_problem(self, response, 403, "permission_denied")

    def assert_unauthorized(self, response: Any) -> None:
        assert_problem(self, response, 401, "unauthenticated")

    def assert_invalid_token(self, response: Any) -> None:
        assert_problem(self, response, 401, "unauthenticated")

    def test_public_permission_catalog(self) -> None:
        """Verify the public permission catalog contains every fixed permission."""

        expected_scopes = {
            "group.resource.create": "GROUP",
            "global.resource.create": "GLOBAL",
            "group.resource.edit": "GROUP",
            "global.resource.edit": "GLOBAL",
            "group.resource.submit": "GROUP",
            "global.resource.submit": "GLOBAL",
            "group.resource.withdraw": "GROUP",
            "global.resource.withdraw": "GLOBAL",
            "group.resource.reopen": "GROUP",
            "global.resource.reopen": "GLOBAL",
            "group.resource.discard": "GROUP",
            "global.resource.discard": "GLOBAL",
            "group.resource.reject": "GROUP",
            "global.resource.reject": "GLOBAL",
            "group.resource.publish": "GROUP",
            "global.resource.publish": "GLOBAL",
            "group.resource.unpublish": "GROUP",
            "global.resource.unpublish": "GLOBAL",
            "group.membership.manage": "GROUP",
            "group.role.manage": "GROUP",
            "group.permission.manage": "GROUP",
            "group.notification.manage": "GROUP",
            "global.membership.manage": "GLOBAL",
            "global.role.manage": "GLOBAL",
            "global.permission.manage": "GLOBAL",
            "global.user_api_token.revoke": "GLOBAL",
            "global.notification.manage": "GLOBAL",
            "global.organization.manage": "GLOBAL",
            "global.contact_point.manage": "GLOBAL",
            "global.group.manage": "GLOBAL",
            "global.concept.manage": "GLOBAL",
            "global.configuration.manage": "GLOBAL",
            "group.category.manage": "GROUP",
            "global.category.manage": "GLOBAL",
            "global.converter.manage": "GLOBAL",
        }

        response = self.gateway.get("/permissions", headers=NO_AUTH_HEADERS)
        self.assertEqual(response.status_code, 200)
        permissions = response.json()

        self.assertEqual(len(permissions), len(expected_scopes))
        self.assertEqual(
            {permission["key"] for permission in permissions},
            set(expected_scopes),
        )
        for permission in permissions:
            self.assertIn(permission["key"], expected_scopes)
            self.assertEqual(permission["scope"], expected_scopes[permission["key"]])
            self.assertTrue(permission["label"])
            self.assertTrue(permission["description"])

    def test_public_and_optional_read_authorization(self) -> None:
        """Verify public reads ignore tokens and optional reads enforce them."""

        suffix = str(uuid4())
        group_id = create_group(self.gateway, f"contract-auth-public-{suffix}")
        category_response = self.gateway.post(
            "/groups/{groupId}/categories",
            groupId=group_id,
            json={"name": f"contract-auth-category-{suffix}"},
        )
        self.assertEqual(category_response.status_code, 201)
        category_id = category_response.json()["id"]

        member, _ = self.create_user("member", suffix)
        self.accept_membership(member, group_id)
        resource_id, version_id = self.create_dataset(
            group_id,
            f"contract-auth-dataset-{suffix}",
        )
        submit_response = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(submit_response.status_code, 200)
        publish_response = self.gateway.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/publish",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(publish_response.status_code, 200)

        blocked, blocked_id = self.create_user("blocked", suffix)
        block_response = self.gateway.post(
            "/users/{id}/block",
            id=blocked_id,
            json={"reason": "authorization contract test"},
        )
        self.assertEqual(block_response.status_code, 204)

        for client in (self.gateway, blocked):
            groups_response = client.get(
                "/groups",
                headers=NO_AUTH_HEADERS if client is self.gateway else None,
            )
            self.assertEqual(groups_response.status_code, 200)
            group_response = client.get(
                "/groups/{groupId}",
                groupId=group_id,
                headers=NO_AUTH_HEADERS if client is self.gateway else None,
            )
            self.assertEqual(group_response.status_code, 200)
            categories_response = client.get(
                "/groups/{groupId}/categories",
                groupId=group_id,
                headers=NO_AUTH_HEADERS if client is self.gateway else None,
            )
            self.assertEqual(categories_response.status_code, 200)
            category_response = client.get(
                "/groups/{groupId}/categories/{categoryId}",
                groupId=group_id,
                categoryId=category_id,
                headers=NO_AUTH_HEADERS if client is self.gateway else None,
            )
            self.assertEqual(category_response.status_code, 200)

        organizations_response = self.gateway.get(
            "/organizations",
            headers=NO_AUTH_HEADERS,
        )
        self.assertEqual(organizations_response.status_code, 200)
        organizations = organizations_response.json()
        imported_organizations = [
            organization
            for organization in organizations
            if organization.get("name")
            == "INGV - Istituto Nazionale di Geofisica e Vulcanologia"
        ]
        self.assertEqual(len(imported_organizations), 1)
        organization_id = imported_organizations[0]["id"]
        for client in (self.gateway, blocked):
            organization_response = client.get(
                "/organizations/{organizationId}",
                organizationId=organization_id,
                headers=NO_AUTH_HEADERS if client is self.gateway else None,
            )
            self.assertEqual(organization_response.status_code, 200)

        anonymous_search = self.gateway.get(
            "/search",
            params={"versioningStatus": "DRAFT"},
            headers=NO_AUTH_HEADERS,
        )
        self.assertEqual(anonymous_search.status_code, 200)
        member_search = member.get(
            "/search",
            params={"versioningStatus": "DRAFT"},
        )
        self.assertEqual(member_search.status_code, 200)
        blocked_search = blocked.get(
            "/search",
            params={"versioningStatus": "DRAFT"},
        )
        self.assert_forbidden(blocked_search)
        invalid_client = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            "invalid-contract-token",
        )
        invalid_search = invalid_client.get(
            "/search",
            params={"versioningStatus": "DRAFT"},
        )
        self.assert_invalid_token(invalid_search)

        anonymous_published = self.gateway.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
            headers=NO_AUTH_HEADERS,
        )
        self.assertEqual(anonymous_published.status_code, 200)
        blocked_published = blocked.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assert_forbidden(blocked_published)

    def test_user_directory_authorization(self) -> None:
        """Verify global user access permissions and admin-only user actions."""

        suffix = str(uuid4())
        directory_manager, _ = self.create_user("directory-manager", suffix)
        role_manager, _ = self.create_user("role-manager", suffix)
        permission_manager, _ = self.create_user("permission-manager", suffix)
        denied, _ = self.create_user("directory-denied", suffix)
        target, target_id = self.create_user("directory-target", suffix)

        self.grant_global_permission(
            self.user_id(directory_manager),
            "global.membership.manage",
        )
        self.grant_global_permission(
            self.user_id(permission_manager),
            "global.permission.manage",
        )
        self.grant_global_permission(
            self.user_id(role_manager),
            "global.role.manage",
        )

        missing_token_response = self.gateway.get(
            "/users",
            headers=NO_AUTH_HEADERS,
        )
        self.assert_unauthorized(missing_token_response)

        target_profile = target.get("/users/{id}", id="self").json()
        for manager in (directory_manager, role_manager, permission_manager):
            users_response = manager.get("/users")
            self.assertEqual(users_response.status_code, 200)
            listed_target = next(
                user for user in users_response.json() if user["id"] == target_id
            )
            self.assertEqual(listed_target["created_at"], target_profile["created_at"])
            self.assertEqual(listed_target["updated_at"], target_profile["updated_at"])

            target_response = manager.get(
                "/users/{id}",
                id=target_id,
            )
            self.assertEqual(target_response.status_code, 200)
            self.assertEqual(target_response.json()["id"], target_id)

        admin_profile_response = self.gateway.get("/users/{id}", id="self")
        self.assertEqual(admin_profile_response.status_code, 200)
        admin_profile = admin_profile_response.json()
        self.assertEqual(admin_profile["global_roles"], ["ADMIN"])
        self.assertCountEqual(
            admin_profile["global_role_permissions"],
            [
                "global.membership.manage",
                "global.role.manage",
                "global.permission.manage",
                "global.notification.manage",
                "global.organization.manage",
                "global.contact_point.manage",
                "global.group.manage",
                "global.concept.manage",
                "global.configuration.manage",
                "global.resource.create",
                "global.resource.edit",
                "global.resource.submit",
                "global.resource.withdraw",
                "global.resource.reopen",
                "global.resource.discard",
                "global.resource.reject",
                "global.resource.publish",
                "global.resource.unpublish",
                "global.category.manage",
                "global.converter.manage",
            ],
        )

        denied_target_response = denied.get(
            "/users/{id}",
            id=target_id,
        )
        self.assert_forbidden(denied_target_response)
        self.assert_forbidden(denied.get("/users"))
        self.assertEqual(denied.get("/users/{id}", id="self").status_code, 200)

        permission_manager_id = self.user_id(permission_manager)
        global_permission_response = permission_manager.put(
            "/users/{id}/global-permissions/{permission}",
            id=target_id,
            permission="global.organization.manage",
        )
        self.assertEqual(global_permission_response.status_code, 204)
        target_profile = target.get("/users/{id}", id="self").json()
        self.assertIn("global.organization.manage", target_profile["global_granted_permissions"])
        self.assertEqual(target_profile["global_role_permissions"], [])

        denied_global_permission_response = denied.put(
            "/users/{id}/global-permissions/{permission}",
            id=target_id,
            permission="global.contact_point.manage",
        )
        self.assert_forbidden(denied_global_permission_response)

        revoke_global_permission_response = permission_manager.delete(
            "/users/{id}/global-permissions/{permission}",
            id=target_id,
            permission="global.organization.manage",
        )
        self.assertEqual(revoke_global_permission_response.status_code, 204)
        target_profile = target.get("/users/{id}", id="self").json()
        self.assertNotIn("global.organization.manage", target_profile["global_granted_permissions"])

        denied_block_response = denied.post(
            "/users/{id}/block",
            id=target_id,
            json={"reason": "authorization contract test"},
        )
        self.assert_forbidden(denied_block_response)
        block_response = self.gateway.post(
            "/users/{id}/block",
            id=target_id,
            json={"reason": "authorization contract test"},
        )
        self.assertEqual(block_response.status_code, 204)
        denied_unblock_response = denied.post(
            "/users/{id}/unblock",
            id=target_id,
        )
        self.assert_forbidden(denied_unblock_response)
        unblock_response = self.gateway.post(
            "/users/{id}/unblock",
            id=target_id,
        )
        self.assertEqual(unblock_response.status_code, 204)

        denied_role_response = denied.put(
            "/users/{id}/global-roles/{role}",
            id=target_id,
            role="ADMIN",
        )
        self.assert_forbidden(denied_role_response)
        assign_role_response = self.gateway.put(
            "/users/{id}/global-roles/{role}",
            id=target_id,
            role="ADMIN",
        )
        self.assertEqual(assign_role_response.status_code, 204)
        remove_role_response = self.gateway.delete(
            "/users/{id}/global-roles/{role}",
            id=target_id,
            role="ADMIN",
        )
        self.assertEqual(remove_role_response.status_code, 204)

    def test_canonical_entity_authorization(self) -> None:
        """Verify scoped canonical writes and public canonical reads."""

        suffix = str(uuid4())
        group_id = create_group(self.gateway, f"contract-auth-canonical-{suffix}")
        category_manager, category_manager_id = self.create_user(
            "category-manager",
            suffix,
        )
        contact_manager, contact_manager_id = self.create_user(
            "contact-manager",
            suffix,
        )
        scheme_manager, scheme_manager_id = self.create_user(
            "scheme-manager",
            suffix,
        )
        organization_manager, organization_manager_id = self.create_user(
            "organization-manager",
            suffix,
        )
        denied, _ = self.create_user("canonical-denied", suffix)

        self.accept_membership(category_manager, group_id)
        self.accept_membership(contact_manager, group_id)
        self.grant_group_permission(
            group_id,
            category_manager_id,
            "group.category.manage",
        )
        self.grant_global_permission(scheme_manager_id, "global.group.manage")
        self.grant_global_permission(
            organization_manager_id,
            "global.organization.manage",
        )
        self.grant_global_permission(contact_manager_id, "global.contact_point.manage")

        group_payload = {
            "name": f"contract-auth-canonical-updated-{suffix}",
            "concept_id": concept_id(self.gateway, "Data"),
        }
        denied_group_replace = denied.put(
            "/groups/{groupId}",
            groupId=group_id,
            json=group_payload,
        )
        self.assert_forbidden(denied_group_replace)
        group_replace = scheme_manager.put(
            "/groups/{groupId}",
            groupId=group_id,
            json=group_payload,
        )
        self.assertEqual(group_replace.status_code, 200)
        self.assertEqual(group_replace.json()["name"], group_payload["name"])

        category_payload = {"name": f"contract-auth-category-{suffix}"}
        denied_category_create = contact_manager.post(
            "/groups/{groupId}/categories",
            groupId=group_id,
            json=category_payload,
        )
        self.assert_forbidden(denied_category_create)
        category_create = category_manager.post(
            "/groups/{groupId}/categories",
            groupId=group_id,
            json=category_payload,
        )
        self.assertEqual(category_create.status_code, 201)
        category_id = category_create.json()["id"]

        updated_category_payload = {
            "name": f"contract-auth-category-updated-{suffix}",
            "parent_category_id": None,
        }
        denied_category_replace = contact_manager.put(
            "/groups/{groupId}/categories/{categoryId}",
            groupId=group_id,
            categoryId=category_id,
            json=updated_category_payload,
        )
        self.assert_forbidden(denied_category_replace)
        category_replace = category_manager.put(
            "/groups/{groupId}/categories/{categoryId}",
            groupId=group_id,
            categoryId=category_id,
            json=updated_category_payload,
        )
        self.assertEqual(category_replace.status_code, 200)
        self.assertEqual(category_replace.json()["name"], updated_category_payload["name"])
        denied_category_delete = contact_manager.delete(
            "/groups/{groupId}/categories/{categoryId}",
            groupId=group_id,
            categoryId=category_id,
        )
        self.assert_forbidden(denied_category_delete)
        category_delete = category_manager.delete(
            "/groups/{groupId}/categories/{categoryId}",
            groupId=group_id,
            categoryId=category_id,
        )
        self.assertEqual(category_delete.status_code, 204)

        contact_payload = {
            "email": f"cp-{suffix}@example.org",
        }
        denied_contact_create = denied.post(
            "/contact-points",
            json=contact_payload,
        )
        self.assert_forbidden(denied_contact_create)
        contact_create = contact_manager.post(
            "/contact-points",
            json=contact_payload,
        )
        self.assertEqual(contact_create.status_code, 201)
        contact_id = contact_create.json()["id"]

        contact_list = contact_manager.get("/contact-points")
        self.assertEqual(contact_list.status_code, 200)
        denied_contact_list = denied.get("/contact-points")
        self.assert_forbidden(denied_contact_list)
        contact_get = contact_manager.get(
            "/contact-points/{contactPointId}",
            contactPointId=contact_id,
        )
        self.assertEqual(contact_get.status_code, 200)
        denied_contact_get = denied.get(
            "/contact-points/{contactPointId}",
            contactPointId=contact_id,
        )
        self.assert_forbidden(denied_contact_get)

        updated_contact_payload = {
            "email": f"cp-updated-{suffix}@example.org",
        }
        denied_contact_replace = denied.put(
            "/contact-points/{contactPointId}",
            contactPointId=contact_id,
            json=updated_contact_payload,
        )
        self.assert_forbidden(denied_contact_replace)
        contact_replace = contact_manager.put(
            "/contact-points/{contactPointId}",
            contactPointId=contact_id,
            json=updated_contact_payload,
        )
        self.assertEqual(contact_replace.status_code, 200)
        denied_contact_delete = denied.delete(
            "/contact-points/{contactPointId}",
            contactPointId=contact_id,
        )
        self.assert_forbidden(denied_contact_delete)
        contact_delete = contact_manager.delete(
            "/contact-points/{contactPointId}",
            contactPointId=contact_id,
        )
        self.assertEqual(contact_delete.status_code, 204)

        organization_payload = {
            "name": f"contract-auth-organization-{suffix}",
            "address": {
                "street": "Authorization Street 1",
                "locality": "Rome",
                "postal_code": "00100",
                "country": "IT",
            },
            "identifiers": [
                {"type": "contract-test", "value": f"organization-{suffix}"}
            ],
            "parents": [],
        }
        denied_organization_create = denied.post(
            "/organizations",
            json=organization_payload,
        )
        self.assert_forbidden(denied_organization_create)
        organization_create = organization_manager.post(
            "/organizations",
            json=organization_payload,
        )
        self.assertEqual(organization_create.status_code, 201)
        organization_id = organization_create.json()["id"]

        updated_organization_payload = {
            **organization_payload,
            "name": f"contract-auth-organization-updated-{suffix}",
            "logo_url": None,
            "page_url": None,
            "email": None,
        }
        denied_organization_replace = denied.put(
            "/organizations/{organizationId}",
            organizationId=organization_id,
            json=updated_organization_payload,
        )
        self.assert_forbidden(denied_organization_replace)
        organization_replace = organization_manager.put(
            "/organizations/{organizationId}",
            organizationId=organization_id,
            json=updated_organization_payload,
        )
        self.assertEqual(organization_replace.status_code, 200)
        denied_organization_delete = denied.delete(
            "/organizations/{organizationId}",
            organizationId=organization_id,
        )
        self.assert_forbidden(denied_organization_delete)
        organization_delete = organization_manager.delete(
            "/organizations/{organizationId}",
            organizationId=organization_id,
        )
        self.assertEqual(organization_delete.status_code, 204)

        denied_group_create = denied.post(
            "/groups",
            json={"name": f"contract-auth-denied-group-{suffix}", "concept_id": concept_id(self.gateway, "Data")},
        )
        self.assert_forbidden(denied_group_create)
        manager_group = scheme_manager.post(
            "/groups",
            json={"name": f"contract-auth-managed-group-{suffix}", "concept_id": concept_id(self.gateway, "Data")},
        )
        self.assertEqual(manager_group.status_code, 201)
        self.assertEqual(scheme_manager.delete("/groups/{groupId}", groupId=manager_group.json()["id"]).status_code, 204)
        denied_concept = scheme_manager.post("/concepts", json={"name": f"contract-auth-concept-{suffix}"})
        self.assert_forbidden(denied_concept)
        denied_group_delete = denied.delete(
            "/groups/{groupId}",
            groupId=group_id,
        )
        self.assert_forbidden(denied_group_delete)
        group_delete = self.gateway.delete(
            "/groups/{groupId}",
            groupId=group_id,
        )
        self.assertEqual(group_delete.status_code, 204)

    def test_concept_permission(self) -> None:
        """Verify top concept management requires its dedicated global permission."""
        suffix = str(uuid4())
        manager, manager_id = self.create_user("concept-manager", suffix)
        self.grant_global_permission(manager_id, "global.concept.manage")
        created = manager.post("/concepts", json={"name": f"contract-concept-{suffix}"})
        self.assertEqual(created.status_code, 201)
        concept_id = created.json()["id"]
        self.assertEqual(manager.put("/concepts/{conceptId}", conceptId=concept_id, json={"name": f"contract-concept-updated-{suffix}"}).status_code, 200)
        self.assertEqual(manager.delete("/concepts/{conceptId}", conceptId=concept_id).status_code, 204)

    def test_membership_and_access_authorization(self) -> None:
        """Verify membership, role, and group permission authorization."""

        suffix = str(uuid4())
        group_id = create_group(self.gateway, f"contract-auth-membership-{suffix}")
        membership_manager, membership_manager_id = self.create_user(
            "membership-manager",
            suffix,
        )
        role_manager, role_manager_id = self.create_user("role-manager", suffix)
        permission_manager, permission_manager_id = self.create_user(
            "permission-manager",
            suffix,
        )
        target, target_id = self.create_user("membership-target", suffix)
        candidate, candidate_id = self.create_user("membership-candidate", suffix)
        denied, _ = self.create_user("membership-denied", suffix)

        self.accept_membership(membership_manager, group_id)
        self.accept_membership(role_manager, group_id)
        self.accept_membership(permission_manager, group_id)
        self.accept_membership(target, group_id)
        self.grant_group_permission(
            group_id,
            membership_manager_id,
            "group.membership.manage",
        )
        self.grant_group_permission(group_id, role_manager_id, "group.role.manage")
        self.grant_group_permission(
            group_id,
            permission_manager_id,
            "group.permission.manage",
        )

        roster = membership_manager.get(
            "/groups/{groupId}/memberships",
            groupId=group_id,
        )
        self.assertEqual(roster.status_code, 200)
        self.assertTrue(any(item["user"]["id"] == target_id for item in roster.json()))
        denied_roster = denied.get(
            "/groups/{groupId}/memberships",
            groupId=group_id,
        )
        self.assert_forbidden(denied_roster)
        missing_roster = self.gateway.get(
            "/groups/{groupId}/memberships",
            groupId=group_id,
            headers=NO_AUTH_HEADERS,
        )
        self.assert_unauthorized(missing_roster)

        membership_body = {"roles": []}
        missing_request = self.gateway.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json=membership_body,
            headers=NO_AUTH_HEADERS,
        )
        self.assert_unauthorized(missing_request)
        requested = candidate.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json=membership_body,
        )
        self.assertEqual(requested.status_code, 201)
        self.assertEqual(requested.json()["user"]["id"], candidate_id)
        withdrawn = candidate.delete(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
        )
        self.assertEqual(withdrawn.status_code, 204)
        requested = candidate.post(
            "/groups/{groupId}/membership-requests",
            groupId=group_id,
            json=membership_body,
        )
        self.assertEqual(requested.status_code, 201)

        denied_decision = denied.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=candidate_id,
            decision="accept",
        )
        self.assert_forbidden(denied_decision)
        accepted = membership_manager.post(
            "/groups/{groupId}/memberships/{userId}/{decision}",
            groupId=group_id,
            userId=candidate_id,
            decision="accept",
        )
        self.assertEqual(accepted.status_code, 200)
        self.assertEqual(accepted.json()["status"], "ACTIVE")

        denied_delete = denied.delete(
            "/groups/{groupId}/memberships/{userId}",
            groupId=group_id,
            userId=target_id,
        )
        self.assert_forbidden(denied_delete)
        self_delete = target.delete(
            "/groups/{groupId}/memberships/{userId}",
            groupId=group_id,
            userId="self",
        )
        self.assertEqual(self_delete.status_code, 204)
        manager_delete = membership_manager.delete(
            "/groups/{groupId}/memberships/{userId}",
            groupId=group_id,
            userId=candidate_id,
        )
        self.assertEqual(manager_delete.status_code, 204)

        assign_role = role_manager.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=permission_manager_id,
            role="EDITOR",
        )
        self.assertEqual(assign_role.status_code, 204)
        denied_assign_role = denied.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=permission_manager_id,
            role="REVIEWER",
        )
        self.assert_forbidden(denied_assign_role)
        remove_role = role_manager.delete(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=permission_manager_id,
            role="EDITOR",
        )
        self.assertEqual(remove_role.status_code, 204)
        denied_remove_role = denied.delete(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=permission_manager_id,
            role="REVIEWER",
        )
        self.assert_forbidden(denied_remove_role)

        role_request = permission_manager.post(
            "/groups/{groupId}/role-requests",
            groupId=group_id,
            json={"role": "EDITOR"},
        )
        self.assertEqual(role_request.status_code, 201)
        role_request_id = role_request.json()["id"]
        denied_role_request = denied.post(
            "/groups/{groupId}/role-requests",
            groupId=group_id,
            json={"role": "EDITOR"},
        )
        self.assert_forbidden(denied_role_request)
        withdrawn_role_request = permission_manager.delete(
            "/groups/{groupId}/role-requests/{requestId}",
            groupId=group_id,
            requestId=role_request_id,
        )
        self.assertEqual(withdrawn_role_request.status_code, 204)
        role_request = permission_manager.post(
            "/groups/{groupId}/role-requests",
            groupId=group_id,
            json={"role": "EDITOR"},
        )
        self.assertEqual(role_request.status_code, 201)
        role_request_id = role_request.json()["id"]
        denied_role_decision = denied.post(
            "/groups/{groupId}/role-requests/{requestId}/{decision}",
            groupId=group_id,
            requestId=role_request_id,
            decision="accept",
        )
        self.assert_forbidden(denied_role_decision)
        role_decision = role_manager.post(
            "/groups/{groupId}/role-requests/{requestId}/{decision}",
            groupId=group_id,
            requestId=role_request_id,
            decision="accept",
        )
        self.assertEqual(role_decision.status_code, 200)
        self.assertEqual(role_decision.json()["roles"], ["EDITOR"])

        grant_permission = permission_manager.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=permission_manager_id,
            permission="group.resource.create",
        )
        self.assertEqual(grant_permission.status_code, 204)
        denied_grant_permission = denied.put(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=permission_manager_id,
            permission="group.resource.edit",
        )
        self.assert_forbidden(denied_grant_permission)
        denied_revoke_permission = denied.delete(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=permission_manager_id,
            permission="group.resource.create",
        )
        self.assert_forbidden(denied_revoke_permission)
        revoke_permission = permission_manager.delete(
            "/groups/{groupId}/memberships/{userId}/permissions/{permission}",
            groupId=group_id,
            userId=permission_manager_id,
            permission="group.resource.create",
        )
        self.assertEqual(revoke_permission.status_code, 204)

    def test_global_resource_permissions_work_without_membership(self) -> None:
        """Regression: global resource permissions must not require group membership."""

        suffix = str(uuid4())
        group_id = create_group(self.gateway, f"contract-auth-global-resource-{suffix}")
        operator, operator_id = self.create_user("global-resource-operator", suffix)
        self.grant_global_permission(operator_id, "global.resource.create")

        created = operator.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json={
                "resource_type": "dataset",
                "metadata": {
                    "title": f"contract-auth-global-resource-{suffix}",
                    "description": "Global resource permission authorization test",
                },
            },
        )
        self.assertEqual(created.status_code, 201)
        self.assertEqual(operator.get("/users/{id}", id="self").json()["memberships"], [])

        resource_id = created.json()["id"]
        versions = self.gateway.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType="dataset",
            id=resource_id,
        ).json()
        drafts = [version for version in versions if version["status"] == "DRAFT"]
        self.assertEqual(len(drafts), 1)
        draft = drafts[0]
        self.grant_global_permission(operator_id, "global.resource.edit")
        replaced = operator.put(
            "/resources/{id}/versions/{versionId}/draft",
            id=resource_id,
            versionId=draft["id"],
            json={
                "resource_type": "dataset",
                "revision": draft["revision"],
                "metadata": {
                    "title": f"contract-auth-global-resource-edited-{suffix}",
                    "description": "Global resource edit authorization test",
                },
            },
        )
        self.assertEqual(replaced.status_code, 200)

    def test_resource_authorization(self) -> None:
        """Verify resource access and lifecycle operations require group access."""

        suffix = str(uuid4())
        group_id = create_group(self.gateway, f"contract-auth-resource-{suffix}")
        target_group_id = create_group(self.gateway, f"contract-auth-resource-target-{suffix}")
        editor, _ = self.create_user("resource-editor", suffix)
        reviewer, reviewer_id = self.create_user("resource-reviewer", suffix)
        denied, _ = self.create_user("resource-denied", suffix)
        self.accept_membership(editor, group_id)
        self.accept_membership(editor, target_group_id)
        self.accept_membership(reviewer, group_id)
        editor_id = self.user_id(editor)
        self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=editor_id,
            role="EDITOR",
        )
        self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=target_group_id,
            userId=editor_id,
            role="EDITOR",
        )
        self.gateway.put(
            "/groups/{groupId}/memberships/{userId}/roles/{role}",
            groupId=group_id,
            userId=reviewer_id,
            role="REVIEWER",
        )
        self.grant_group_permission(group_id, reviewer_id, "group.resource.unpublish")

        def get_version(resource_id: str, status: str) -> dict[str, Any]:
            response = self.gateway.get(
                "/resources/{resourceType}/{id}/versions",
                resourceType="dataset",
                id=resource_id,
            )
            self.assertEqual(response.status_code, 200)
            matches = [item for item in response.json() if item["status"] == status]
            self.assertEqual(len(matches), 1)
            return matches[0]

        def dataset_payload(title: str) -> dict[str, Any]:
            return {
                "resource_type": "dataset",
                "metadata": {
                    "title": title,
                    "description": f"{title} description",
                },
            }

        draft_resource_id, draft_version_id = self.create_dataset(
            group_id,
            f"contract-auth-draft-{suffix}",
        )
        draft = self.gateway.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=draft_resource_id,
            versionId=draft_version_id,
        ).json()
        replace_payload = {
            "resource_type": "dataset",
            "revision": draft["revision"],
            "metadata": {
                "title": f"contract-auth-draft-updated-{suffix}",
                "description": "Authorization lifecycle description",
            },
        }
        denied_replace = denied.put(
            "/resources/{id}/versions/{versionId}/draft",
            id=draft_resource_id,
            versionId=draft_version_id,
            json=replace_payload,
        )
        self.assert_forbidden(denied_replace)
        replaced = editor.put(
            "/resources/{id}/versions/{versionId}/draft",
            id=draft_resource_id,
            versionId=draft_version_id,
            json=replace_payload,
        )
        self.assertEqual(replaced.status_code, 200)
        draft_version_id = get_version(draft_resource_id, "DRAFT")["id"]
        denied_discard = denied.delete(
            "/resources/{resourceType}/{id}/versions/{versionId}/draft",
            resourceType="dataset",
            id=draft_resource_id,
            versionId=draft_version_id,
        )
        self.assert_forbidden(denied_discard)
        discarded = editor.delete(
            "/resources/{resourceType}/{id}/versions/{versionId}/draft",
            resourceType="dataset",
            id=draft_resource_id,
            versionId=draft_version_id,
        )
        self.assertEqual(discarded.status_code, 204)

        created = editor.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json=dataset_payload(f"contract-auth-lifecycle-{suffix}"),
        )
        self.assertEqual(created.status_code, 201)
        resource_id = created.json()["id"]
        draft = get_version(resource_id, "DRAFT")
        version_id = draft["id"]

        denied_get = denied.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        assert_problem(self, denied_get, 404, "not_found")
        editor_get = editor.get(
            "/resources/{resourceType}/{id}/versions/{versionId}",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(editor_get.status_code, 200)
        denied_versions = denied.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType="dataset",
            id=resource_id,
        )
        self.assert_forbidden(denied_versions)
        editor_versions = editor.get(
            "/resources/{resourceType}/{id}/versions",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(editor_versions.status_code, 200)

        denied_submit = denied.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assert_forbidden(denied_submit)
        submitted = editor.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(submitted.status_code, 200)
        denied_withdraw = denied.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/withdraw",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assert_forbidden(denied_withdraw)
        withdrawn = editor.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/withdraw",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(withdrawn.status_code, 200)
        submitted = editor.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(submitted.status_code, 200)

        denied_reject = denied.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/reject",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
            json={"note": "authorization contract rejection"},
        )
        self.assert_forbidden(denied_reject)
        rejected = reviewer.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/reject",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
            json={"note": "authorization contract rejection"},
        )
        self.assertEqual(rejected.status_code, 200)
        denied_reopen = denied.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/reopen",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assert_forbidden(denied_reopen)
        reopened = editor.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/reopen",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(reopened.status_code, 200)
        submitted = editor.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/submit",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(submitted.status_code, 200)
        denied_publish = denied.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/publish",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assert_forbidden(denied_publish)
        published = reviewer.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/publish",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(published.status_code, 200)

        denied_copy = denied.post(
            "/groups/{groupId}/resources/{resourceType}/{id}/copy",
            groupId=target_group_id,
            resourceType="dataset",
            id=resource_id,
        )
        self.assert_forbidden(denied_copy)
        copied = editor.post(
            "/groups/{groupId}/resources/{resourceType}/{id}/copy",
            groupId=target_group_id,
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(copied.status_code, 201)

        denied_unpublish = denied.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/unpublish",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assert_forbidden(denied_unpublish)
        unpublished = reviewer.post(
            "/resources/{resourceType}/{id}/versions/{versionId}/unpublish",
            resourceType="dataset",
            id=resource_id,
            versionId=version_id,
        )
        self.assertEqual(unpublished.status_code, 200)
        denied_archive_draft = denied.post(
            "/resources/{resourceType}/{id}/draft-from-archive",
            resourceType="dataset",
            id=resource_id,
        )
        self.assert_forbidden(denied_archive_draft)
        archive_draft = editor.post(
            "/resources/{resourceType}/{id}/draft-from-archive",
            resourceType="dataset",
            id=resource_id,
        )
        self.assertEqual(archive_draft.status_code, 201)

        publication_resource = editor.post(
            "/groups/{groupId}/resources",
            groupId=group_id,
            json=dataset_payload(f"contract-auth-publication-{suffix}"),
        )
        self.assertEqual(publication_resource.status_code, 201)
        publication_id = publication_resource.json()["id"]
        publication_version_id = get_version(publication_id, "DRAFT")["id"]
        self.assertEqual(
            editor.post(
                "/resources/{resourceType}/{id}/versions/{versionId}/submit",
                resourceType="dataset",
                id=publication_id,
                versionId=publication_version_id,
            ).status_code,
            200,
        )
        self.assertEqual(
            reviewer.post(
                "/resources/{resourceType}/{id}/versions/{versionId}/publish",
                resourceType="dataset",
                id=publication_id,
                versionId=publication_version_id,
            ).status_code,
            200,
        )
        denied_publication_draft = denied.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset",
            id=publication_id,
        )
        self.assert_forbidden(denied_publication_draft)
        publication_draft = editor.post(
            "/resources/{resourceType}/{id}/draft-from-publication",
            resourceType="dataset",
            id=publication_id,
        )
        self.assertEqual(publication_draft.status_code, 201)

    def test_turtle_import_authorization(self) -> None:
        """Verify Turtle import is restricted to global administrators."""

        suffix = str(uuid4())
        denied, _ = self.create_user("import-denied", suffix)
        turtle = f"""
@prefix dcat: <http://www.w3.org/ns/dcat#> .
@prefix dct: <http://purl.org/dc/terms/> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .

<category:auth-scheme-{suffix}> a skos:ConceptScheme ;
    dct:title "Authorization import scheme {suffix}" ;
    skos:hasTopConcept <category:facets/dataset-theme> .
<category:auth-category-{suffix}> a skos:Concept ;
    skos:inScheme <category:auth-scheme-{suffix}> ;
    skos:prefLabel "Authorization import category {suffix}" .
<dataset:auth-{suffix}> a dcat:Dataset ;
    dct:title "Authorization import dataset {suffix}" ;
    dcat:theme <category:auth-category-{suffix}> .
""".strip()

        denied_import = denied.post(
            "/imports/turtle",
            data=turtle,
            headers={"Content-Type": "text/turtle"},
        )
        self.assert_forbidden(denied_import)
        imported = self.gateway.post(
            "/imports/turtle",
            data=turtle,
            headers={"Content-Type": "text/turtle"},
        )
        self.assertEqual(imported.status_code, 200)
        result = imported.json()
        self.assertGreaterEqual(result["datasets"], 1)

    def user_id(self, client: GatewayClient) -> str:
        response = client.get("/users/{id}", id="self")
        self.assertEqual(response.status_code, 200)
        return response.json()["id"]
