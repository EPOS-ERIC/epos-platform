import os
import unittest
from uuid import uuid4

from auth import get_token
from client import SCHEMA, GatewayClient
from fixtures import assert_problem


class OrganizationsAndContactPointsContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.gateway = GatewayClient(
            SCHEMA,
            os.getenv("GATEWAY_API_URL", "http://localhost:32000/v1"),
            get_token(),
        )

    def test_imported_organization_and_contact_point_reads(self) -> None:
        """Verify imported organization and contact-point records can be read."""

        public_headers = {"Authorization": ""}
        organizations_response = self.gateway.get(
            "/organizations",
            headers=public_headers,
        )
        self.assertEqual(organizations_response.status_code, 200)
        organizations = organizations_response.json()
        self.assertIsInstance(organizations, list)
        organization_matches = [
            organization
            for organization in organizations
            if isinstance(organization, dict)
            and organization.get("name")
            == "INGV - Istituto Nazionale di Geofisica e Vulcanologia"
        ]
        self.assertEqual(len(organization_matches), 1)
        organization = organization_matches[0]
        organization_id = organization["id"]
        self.assertEqual(organization["address"]["street"], "Via di Vigna Murata 605")
        self.assertEqual(organization["address"]["locality"], "Rome")
        self.assertEqual(organization["address"]["postal_code"], "00143")
        self.assertEqual(organization["address"]["country"], "IT")
        identifier_matches = [
            identifier
            for identifier in organization["identifiers"]
            if identifier.get("type") == "pic"
        ]
        self.assertEqual(len(identifier_matches), 1)
        self.assertEqual(identifier_matches[0]["value"], "999472675")

        organization_response = self.gateway.get(
            "/organizations/{organizationId}",
            organizationId=organization_id,
            headers=public_headers,
        )
        self.assertEqual(organization_response.status_code, 200)
        organization_by_id = organization_response.json()
        self.assertEqual(organization_by_id["id"], organization_id)
        self.assertEqual(
            organization_by_id["name"],
            "INGV - Istituto Nazionale di Geofisica e Vulcanologia",
        )

        contact_points_response = self.gateway.get("/contact-points")
        self.assertEqual(contact_points_response.status_code, 200)
        contact_points = contact_points_response.json()
        self.assertIsInstance(contact_points, list)
        contact_matches = [
            contact_point
            for contact_point in contact_points
            if isinstance(contact_point, dict)
            and contact_point.get("email") == "andrea.rovida@ingv.it"
        ]
        self.assertEqual(len(contact_matches), 1)
        contact_point_id = contact_matches[0]["id"]

        contact_point_response = self.gateway.get(
            "/contact-points/{contactPointId}",
            contactPointId=contact_point_id,
        )
        self.assertEqual(contact_point_response.status_code, 200)
        contact_point = contact_point_response.json()
        self.assertEqual(contact_point["id"], contact_point_id)
        self.assertEqual(contact_point["email"], "andrea.rovida@ingv.it")

    def test_organization_and_contact_point_crud(self) -> None:
        """Verify organization hierarchy and contact point CRUD."""

        suffix = str(uuid4())
        contact_payload = {"email": f"contract-{suffix}@example.org"}
        created_contact_response = self.gateway.post(
            "/contact-points",
            json=contact_payload,
        )
        self.assertEqual(created_contact_response.status_code, 201)
        created_contact = created_contact_response.json()
        self.assertEqual(created_contact["email"], contact_payload["email"])
        contact_point_id = created_contact["id"]

        duplicate_contact_response = self.gateway.post(
            "/contact-points",
            json=contact_payload,
        )
        assert_problem(self, duplicate_contact_response, 409, "conflict")

        parent_name = f"contract-parent-{suffix}"
        parent_payload = {
            "name": parent_name,
            "address": {
                "street": "Contract Street 1",
                "locality": "Rome",
                "postal_code": "00100",
                "country": "IT",
            },
            "identifiers": [
                {"type": "contract-test", "value": f"parent-{suffix}"}
            ],
            "parents": [],
        }
        created_parent_response = self.gateway.post(
            "/organizations",
            json=parent_payload,
        )
        self.assertEqual(created_parent_response.status_code, 201)
        created_parent = created_parent_response.json()
        self.assertEqual(created_parent["name"], parent_name)
        self.assertEqual(created_parent["parents"], [])
        parent_id = created_parent["id"]

        duplicate_parent_response = self.gateway.post(
            "/organizations",
            json=parent_payload,
        )
        self.assertEqual(duplicate_parent_response.status_code, 201)
        self.assertNotEqual(duplicate_parent_response.json()["id"], parent_id)

        child_name = f"contract-child-{suffix}"
        child_payload = {
            "name": child_name,
            "address": {
                "street": "Contract Street 2",
                "locality": "Rome",
                "postal_code": "00101",
                "country": "IT",
            },
            "identifiers": [
                {"type": "contract-test", "value": f"child-{suffix}"}
            ],
            "parents": [parent_id],
        }
        created_child_response = self.gateway.post(
            "/organizations",
            json=child_payload,
        )
        self.assertEqual(created_child_response.status_code, 201)
        created_child = created_child_response.json()
        self.assertEqual(created_child["parents"], [parent_id])
        child_id = created_child["id"]

        child_response = self.gateway.get(
            "/organizations/{organizationId}",
            organizationId=child_id,
        )
        self.assertEqual(child_response.status_code, 200)
        child = child_response.json()
        self.assertEqual(child["id"], child_id)
        self.assertEqual(child["parents"], [parent_id])

        updated_contact_payload = {"email": f"updated-{suffix}@example.org"}
        updated_contact_response = self.gateway.put(
            "/contact-points/{contactPointId}",
            contactPointId=contact_point_id,
            json=updated_contact_payload,
        )
        self.assertEqual(updated_contact_response.status_code, 200)
        updated_contact = updated_contact_response.json()
        self.assertEqual(updated_contact["id"], contact_point_id)
        self.assertEqual(updated_contact["email"], updated_contact_payload["email"])

        updated_child_name = f"{child_name}-updated"
        updated_child_payload = {
            "name": updated_child_name,
            "address": child_payload["address"],
            "logo_url": None,
            "page_url": None,
            "email": None,
            "identifiers": child_payload["identifiers"],
            "parents": [parent_id],
        }
        updated_child_response = self.gateway.put(
            "/organizations/{organizationId}",
            organizationId=child_id,
            json=updated_child_payload,
        )
        self.assertEqual(updated_child_response.status_code, 200)
        updated_child = updated_child_response.json()
        self.assertEqual(updated_child["id"], child_id)
        self.assertEqual(updated_child["name"], updated_child_name)
        self.assertEqual(updated_child["parents"], [parent_id])

        deleted_parent_response = self.gateway.delete(
            "/organizations/{organizationId}",
            organizationId=parent_id,
        )
        self.assertEqual(deleted_parent_response.status_code, 204)

        deleted_parent_get_response = self.gateway.get(
            "/organizations/{organizationId}",
            organizationId=parent_id,
        )
        self.assertEqual(deleted_parent_get_response.status_code, 404)
        child_after_parent_response = self.gateway.get(
            "/organizations/{organizationId}",
            organizationId=child_id,
        )
        self.assertEqual(child_after_parent_response.status_code, 200)
        self.assertEqual(child_after_parent_response.json()["parents"], [])

        deleted_contact_response = self.gateway.delete(
            "/contact-points/{contactPointId}",
            contactPointId=contact_point_id,
        )
        self.assertEqual(deleted_contact_response.status_code, 204)

        deleted_contact_get_response = self.gateway.get(
            "/contact-points/{contactPointId}",
            contactPointId=contact_point_id,
        )
        self.assertEqual(deleted_contact_get_response.status_code, 404)

        deleted_child_response = self.gateway.delete(
            "/organizations/{organizationId}",
            organizationId=child_id,
        )
        self.assertEqual(deleted_child_response.status_code, 204)

        deleted_child_get_response = self.gateway.get(
            "/organizations/{organizationId}",
            organizationId=child_id,
        )
        self.assertEqual(deleted_child_get_response.status_code, 404)
