from __future__ import annotations

import re
from pathlib import Path
from typing import Any, Mapping
from urllib.parse import quote

import requests
import schemathesis

ROOT = Path(__file__).resolve().parents[2]
SCHEMA = schemathesis.openapi.from_path(ROOT / "services/gateway/docs/openapi.yaml")
DEFAULT_TIMEOUT = 30
PATH_PARAMETER = re.compile(r"{([^{}]+)}")


class GatewayClient:
    def __init__(
        self,
        schema: Any,
        base_url: str,
        token: str,
        *,
        session: requests.Session | None = None,
    ) -> None:
        base_url = base_url.strip().rstrip("/")
        if not base_url:
            raise ValueError("Gateway API URL must not be empty")
        if not token.strip():
            raise ValueError("Gateway bearer token must not be empty")

        self.schema = schema
        self.base_url = base_url
        self.session = session or requests.Session()
        self.session.headers["Authorization"] = f"Bearer {token}"

    def request(
        self,
        method: str,
        path_template: str,
        *,
        params: Mapping[str, Any] | None = None,
        json: Any = None,
        data: Any = None,
        headers: Mapping[str, str] | None = None,
        **path_parameters: object,
    ) -> requests.Response:
        method = method.upper()
        operation = self._operation(path_template, method)
        url = self._url(path_template, path_parameters)
        response = self.session.request(
            method,
            url,
            params=params,
            json=json,
            data=data,
            headers=headers,
            timeout=DEFAULT_TIMEOUT,
        )
        operation.validate_response(response)
        return response

    def get(
        self,
        path_template: str,
        *,
        params: Mapping[str, Any] | None = None,
        headers: Mapping[str, str] | None = None,
        **path_parameters: object,
    ) -> requests.Response:
        return self.request(
            "GET",
            path_template,
            params=params,
            headers=headers,
            **path_parameters,
        )

    def post(
        self,
        path_template: str,
        *,
        params: Mapping[str, Any] | None = None,
        json: Any = None,
        data: Any = None,
        headers: Mapping[str, str] | None = None,
        **path_parameters: object,
    ) -> requests.Response:
        return self.request(
            "POST",
            path_template,
            params=params,
            json=json,
            data=data,
            headers=headers,
            **path_parameters,
        )

    def put(
        self,
        path_template: str,
        *,
        params: Mapping[str, Any] | None = None,
        json: Any = None,
        data: Any = None,
        headers: Mapping[str, str] | None = None,
        **path_parameters: object,
    ) -> requests.Response:
        return self.request(
            "PUT",
            path_template,
            params=params,
            json=json,
            data=data,
            headers=headers,
            **path_parameters,
        )

    def patch(
        self,
        path_template: str,
        *,
        params: Mapping[str, Any] | None = None,
        json: Any = None,
        data: Any = None,
        headers: Mapping[str, str] | None = None,
        **path_parameters: object,
    ) -> requests.Response:
        return self.request(
            "PATCH",
            path_template,
            params=params,
            json=json,
            data=data,
            headers=headers,
            **path_parameters,
        )

    def delete(
        self,
        path_template: str,
        *,
        params: Mapping[str, Any] | None = None,
        json: Any = None,
        data: Any = None,
        headers: Mapping[str, str] | None = None,
        **path_parameters: object,
    ) -> requests.Response:
        return self.request(
            "DELETE",
            path_template,
            params=params,
            json=json,
            data=data,
            headers=headers,
            **path_parameters,
        )

    def _operation(self, path_template: str, method: str) -> Any:
        try:
            path = self.schema[path_template]
        except KeyError as error:
            raise ValueError(
                f"OpenAPI path template not found: {path_template}"
            ) from error

        try:
            return path[method]
        except KeyError as error:
            raise ValueError(
                f"OpenAPI operation not found: {method} {path_template}"
            ) from error

    def _url(self, path_template: str, path_parameters: Mapping[str, object]) -> str:
        names = PATH_PARAMETER.findall(path_template)
        missing = [
            name
            for name in names
            if name not in path_parameters or path_parameters[name] is None
        ]
        if missing:
            raise ValueError(
                f"missing path parameter(s) for {path_template}: {', '.join(missing)}"
            )

        unexpected = sorted(set(path_parameters) - set(names))
        if unexpected:
            raise ValueError(
                f"unexpected path parameter(s) for {path_template}: {', '.join(unexpected)}"
            )

        url = self.base_url + path_template
        for name in names:
            value = path_parameters[name]
            if str(value) == "":
                raise ValueError(
                    f"path parameter {name} for {path_template} must not be empty"
                )
            url = url.replace("{" + name + "}", quote(str(value), safe=""), 1)
        return url
