from __future__ import annotations

# pyright: reportAttributeAccessIssue=false

from datetime import datetime

import schemathesis

_SEARCH_TYPES = {
    "organization",
    "softwareapplication",
    "softwaresourcecode",
}


def _parse_datetime(value: object) -> datetime | None:
    if not isinstance(value, str) or value == "":
        return None

    normalized = value.replace("Z", "+00:00")
    try:
        return datetime.fromisoformat(normalized)
    except ValueError:
        return None


@schemathesis.hook.apply_to(operation_id="searchMetadata")
def filter_case(ctx, case):
    query = case.query or {}
    type_value = query.get("type")

    if not isinstance(type_value, str) or type_value == "":
        return False

    allowed_types = {item for item in type_value.split(",") if item}
    if not allowed_types or any(item not in _SEARCH_TYPES for item in allowed_types):
        return False

    date_from = _parse_datetime(query.get("dateFrom"))
    date_to = _parse_datetime(query.get("dateTo"))

    # Gateway rejects inverted temporal ranges with 400. Skip those cases.
    if date_from is not None and date_to is not None and date_to < date_from:
        return False

    return True
