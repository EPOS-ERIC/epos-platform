from __future__ import annotations

"""Schemathesis hooks keep fuzzed cases on valid, low-risk inputs."""

# pyright: reportAttributeAccessIssue=false

from datetime import datetime
import os

import schemathesis

# These ops need live auth; skip them when token is missing so runs stay signal-only.
AUTHENTICATED_OPERATION_LABELS = {
    "GET /contact-points",
    "GET /contact-points/{contactPointId}",
    "GET /users",
    "GET /users/{id}",
    "GET /groups/{groupId}/memberships",
    "GET /resources/{resourceType}/{id}/versions/{versionId}",
}

# These ops need concrete group ids from env; skip when missing instead of fuzzing placeholders.
GROUP_TARGET_OPERATION_LABELS = {
    "GET /groups/{groupId}",
    "GET /groups/{groupId}/memberships",
    "GET /groups/{groupId}/categories",
    "GET /groups/{groupId}/categories/{categoryId}",
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


def _configured_group_target() -> tuple[str, str] | None:
    group_id = _env("SCHEMATHESIS_GROUP_ID")
    if group_id is None:
        return None
    return ("groupId", group_id)


def _configured_resource_target() -> tuple[str, str] | None:
    resource_id = _env("SCHEMATHESIS_RESOURCE_ID")
    version_id = _env("SCHEMATHESIS_RESOURCE_VERSION_ID")
    if resource_id is None or version_id is None:
        return None
    return (resource_id, version_id)

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

    if label == "GET /resources/{resourceType}/{id}/versions/{versionId}" and _configured_resource_target() is None:
        return False

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

    if label == "GET /users/{id}":
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

    if label in {"GET /groups/{groupId}/categories/{categoryId}"}:
        path_parameters = dict(case.path_parameters or {})
        category_id = _env("SCHEMATHESIS_CATEGORY_ID")
        if category_id is not None:
            path_parameters["categoryId"] = category_id
        case.path_parameters = path_parameters

    if label == "GET /resources/{resourceType}/{id}/versions/{versionId}":
        target = _configured_resource_target()
        if target is not None:
            path_parameters = dict(case.path_parameters or {})
            path_parameters["resourceType"] = "dataset"
            path_parameters["id"] = target[0]
            path_parameters["versionId"] = target[1]
            case.path_parameters = path_parameters

    if label == "GET /search":
        query = dict(case.query or {})
        if "spatialFilter" in query:
            query["spatialFilter"] = "POINT(12.3 41.8)"
        if "provider" in query:
            provider_id = _env("SCHEMATHESIS_PROVIDER_ID")
            if provider_id is not None:
                query["provider"] = provider_id
            else:
                del query["provider"]
        case.query = query
