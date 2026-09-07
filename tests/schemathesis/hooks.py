from __future__ import annotations

"""Schemathesis hooks provide authentication and existing-resource IDs."""

# pyright: reportAttributeAccessIssue=false

import os

import schemathesis

RESOURCE_TARGETS = {
    "GET /resources/{resourceType}/{id}/versions/{versionId}": "DRAFT",
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


def _configured_resource_target(status: str) -> tuple[str, str] | None:
    resource_id = _env(f"SCHEMATHESIS_{status}_RESOURCE_ID")
    version_id = _env(f"SCHEMATHESIS_{status}_RESOURCE_VERSION_ID")
    if resource_id is None or version_id is None:
        return None
    return (resource_id, version_id)

@schemathesis.hook("filter_case")
def general_filter_case(context, case):
    label = _operation_label(context, case)

    if _env("SCHEMATHESIS_BEARER_TOKEN") is None:
        return False

    if "{groupId}" in label and _env("SCHEMATHESIS_GROUP_ID") is None:
        return False

    return True


@schemathesis.hook
def before_call(context, case, kwargs):
    label = _operation_label(context, case)

    token = _env("SCHEMATHESIS_BEARER_TOKEN")
    if token is not None:
        headers = dict(case.headers or {})
        headers["Authorization"] = f"Bearer {token}"
        case.headers = headers

    path_parameters = dict(case.path_parameters or {})

    group_id = _env("SCHEMATHESIS_GROUP_ID")
    if group_id is not None and "groupId" in path_parameters:
        path_parameters["groupId"] = group_id

    category_id = _env("SCHEMATHESIS_CATEGORY_ID")
    if category_id is not None and "categoryId" in path_parameters:
        path_parameters["categoryId"] = category_id

    contact_point_id = _env("SCHEMATHESIS_CONTACT_POINT_ID")
    if contact_point_id is not None and "contactPointId" in path_parameters:
        path_parameters["contactPointId"] = contact_point_id

    organization_id = _env("SCHEMATHESIS_ORGANIZATION_ID")
    if organization_id is not None and "organizationId" in path_parameters:
        path_parameters["organizationId"] = organization_id

    membership_user_id = _env("SCHEMATHESIS_MEMBERSHIP_USER_ID")
    if membership_user_id is not None and "userId" in path_parameters:
        path_parameters["userId"] = membership_user_id

    role_request_id = _env("SCHEMATHESIS_ROLE_REQUEST_ID")
    if role_request_id is not None and "requestId" in path_parameters:
        path_parameters["requestId"] = role_request_id

    if label == "GET /users/{id}":
        # "self" is the only stable user id for these endpoints.
        path_parameters["id"] = "self"
    elif "/users/{id}/" in label:
        user_id = _env("SCHEMATHESIS_USER_ID")
        if user_id is not None:
            path_parameters["id"] = user_id

    resource_status = RESOURCE_TARGETS.get(label)
    if resource_status is not None:
        target = _configured_resource_target(resource_status)
        if target is not None:
            path_parameters["resourceType"] = "dataset"
            path_parameters["id"] = target[0]
            if "versionId" in path_parameters:
                path_parameters["versionId"] = target[1]

    case.path_parameters = path_parameters

    if label == "POST /groups/{groupId}/categories":
        category_id = _env("SCHEMATHESIS_CATEGORY_ID")
        body = case.body
        if category_id is not None and isinstance(body, dict) and body.get("parent_category_id") is not None:
            body = dict(body)
            body["parent_category_id"] = category_id
            case.body = body
