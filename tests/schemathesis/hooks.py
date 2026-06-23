from __future__ import annotations

"""Schemathesis hooks keep fuzzed cases on valid, low-risk inputs."""

# pyright: reportAttributeAccessIssue=false

from datetime import datetime
import os

import schemathesis

# These ops need live auth; skip them when token is missing so runs stay signal-only.
AUTHENTICATED_OPERATION_LABELS = {
    "GET /users",
    "GET /users/{id}",
    "PATCH /users/{id}",
    "GET /groups",
    "GET /groups/{groupId}",
    "GET /groups/{groupId}/memberships",
    "PATCH /groups/{groupId}",
    "PATCH /groups/{groupId}/memberships/{userId}",
}

# These ops need concrete group ids from env; skip when missing instead of fuzzing placeholders.
GROUP_TARGET_OPERATION_LABELS = {
    "GET /groups/{groupId}",
    "GET /groups/{groupId}/memberships",
    "PATCH /groups/{groupId}",
    "PATCH /groups/{groupId}/memberships/{userId}",
}

# These writes need concrete user + group ids from env.
MEMBERSHIP_TARGET_OPERATION_LABELS = {
    "PATCH /groups/{groupId}/memberships/{userId}",
}

# Fixed ids keep resource-path coverage working in local dev and CI.
DEFAULT_RESOURCE_IDS = {
    "concept": "category%3Asoftware_jup_conc_multi%2Fv1.0",
    "conceptscheme": "category%3Asoftware_jup%2Fv1.0",
    "dataset": "https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FSeismology%2FDataset%2FAHEAD%2Fevents%2Fv1.0",
    "distribution": "https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FSeismology%2FDataset%2FAHEAD%2Fevents%2Fdistribution%2Frestful%2Fv1.0",
    "organization": "ROR%3A048a87296%2Fv1.0",
    "softwareapplication": "https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2FCatalogToXls%2Fv1.0",
    "softwaresourcecode": "https%3A%2F%2Fdoi.org%2F10.5281%2Fzenodo.17598403",
    "webservice": "https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FSeismology%2FWebService%2FAHEAD%2Frestful%2Ffdsnws-event%2Fv1.0",
}

# Patch bodies need stable, harmless values; generated payloads can fail validation or mutate too much.
SAFE_PATCH_BODIES = {
    "PATCH /groups/{groupId}": {
        "name": "ALL",
        "description": "Basic group",
        "is_system": True,
    },
    "PATCH /groups/{groupId}/memberships/{userId}": {
        "role": "VIEWER",
        "status": "ACTIVE",
    },
    "PATCH /users/{id}": {
        "email": "admin@admin.org",
        "given_name": "Admin",
        "family_name": "Admin",
    },
}


def _operation_label(context: object, case: object) -> str:
    operation = getattr(context, "operation", None) or getattr(case, "operation", None)
    return str(getattr(operation, "label", ""))


def _env(name: str) -> str | None:
    value = os.getenv(name)
    if value is None:
        return None

    value = value.strip()
    if value == "":
        return None
    return value


def _resource_id_for_type(resource_type: object) -> str | None:
    if not isinstance(resource_type, str) or resource_type == "":
        return None

    return DEFAULT_RESOURCE_IDS.get(resource_type)


def _configured_group_target() -> tuple[str, str] | None:
    group_id = _env("SCHEMATHESIS_GROUP_ID")
    if group_id is None:
        return None
    return ("groupId", group_id)


def _configured_membership_target() -> tuple[tuple[str, str], tuple[str, str]] | None:
    group_id = _env("SCHEMATHESIS_GROUP_ID")
    user_id = _env("SCHEMATHESIS_MEMBERSHIP_USER_ID")
    if group_id is None or user_id is None:
        return None
    return (("groupId", group_id), ("userId", user_id))

def _parse_datetime(value: object) -> datetime | None:
    if not isinstance(value, str) or value == "":
        return None

    normalized = value.replace("Z", "+00:00")
    try:
        return datetime.fromisoformat(normalized)
    except ValueError:
        return None


@schemathesis.hook("filter_case")
def general_filter_case(context, case):
    label = _operation_label(context, case)

    if label == "GET /search":
        query = case.query or {}
        type_value = query.get("type")

        if not isinstance(type_value, str) or type_value == "":
            return False

        date_from = _parse_datetime(query.get("dateFrom"))
        date_to = _parse_datetime(query.get("dateTo"))

        # Gateway rejects inverted temporal ranges with 400. Skip those cases.
        if date_from is not None and date_to is not None and date_to < date_from:
            return False

    if label in AUTHENTICATED_OPERATION_LABELS and _env("SCHEMATHESIS_BEARER_TOKEN") is None:
        return False

    # Skip group-targeted cases until env provides a real target.
    if label in GROUP_TARGET_OPERATION_LABELS and _configured_group_target() is None:
        return False

    # Membership tests need both ids or they only generate 404 noise.
    if label in MEMBERSHIP_TARGET_OPERATION_LABELS and _configured_membership_target() is None:
        return False

    if label == "GET /resources/{resourceType}/{id}":
        path_parameters = case.path_parameters or {}
        resource_type = path_parameters.get("resourceType")
        return _resource_id_for_type(resource_type) is not None

    return True


@schemathesis.hook
def before_call(context, case, kwargs):
    label = _operation_label(context, case)

    if label in AUTHENTICATED_OPERATION_LABELS:
        token = _env("SCHEMATHESIS_BEARER_TOKEN")
        if token is not None:
            headers = dict(case.headers or {})
            headers["Authorization"] = f"Bearer {token}"
            case.headers = headers

    if label in {"GET /users/{id}", "PATCH /users/{id}"}:
        # "self" is the only stable user id for these endpoints.
        path_parameters = dict(case.path_parameters or {})
        path_parameters["id"] = "self"
        case.path_parameters = path_parameters

    if label in GROUP_TARGET_OPERATION_LABELS:
        target = _configured_group_target()
        if target is not None:
            path_parameters = dict(case.path_parameters or {})
            path_parameters[target[0]] = target[1]
            case.path_parameters = path_parameters

    if label in MEMBERSHIP_TARGET_OPERATION_LABELS:
        target = _configured_membership_target()
        if target is not None:
            path_parameters = dict(case.path_parameters or {})
            for key, value in target:
                path_parameters[key] = value
            case.path_parameters = path_parameters

    if label == "GET /resources/{resourceType}/{id}":
        # Resource ids come from fixture data; generated ids are usually invalid here.
        path_parameters = dict(case.path_parameters or {})
        resource_type = path_parameters.get("resourceType")
        resource_id = _resource_id_for_type(resource_type)
        if resource_id is not None:
            path_parameters["id"] = resource_id
            case.path_parameters = path_parameters

    safe_body = SAFE_PATCH_BODIES.get(label)
    if safe_body is not None:
        case.body = dict(safe_body)
