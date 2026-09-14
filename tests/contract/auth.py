from __future__ import annotations

import os
from dataclasses import dataclass
from html.parser import HTMLParser

import requests

DEFAULT_AAI_URL = "http://localhost:8110"
DEFAULT_TIMEOUT = 30


@dataclass(frozen=True)
class UserCredentials:
    username: str
    password: str


class _CSRFTokenParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.token: str | None = None

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if tag != "input":
            return

        attributes = dict(attrs)
        if attributes.get("name") == "csrf_token":
            self.token = attributes.get("value")


def get_token(
    *,
    aai_url: str | None = None,
    username: str | None = None,
    password: str | None = None,
    client_id: str | None = None,
    client_secret: str | None = None,
    timeout: int = DEFAULT_TIMEOUT,
) -> str:
    aai_url = _setting(aai_url, "AAI_URL", DEFAULT_AAI_URL).rstrip("/")
    username = _setting(username, "AUTH_USERNAME", "admin@admin.org")
    password = _setting(password, "AUTH_PASSWORD", "adminadmin", strip=False)
    client_id = _setting(client_id, "AUTH_CLIENT_ID", "local-dev-client")
    client_secret = _setting(
        client_secret,
        "AUTH_CLIENT_SECRET",
        "dev-secret",
        strip=False,
    )

    response = requests.post(
        f"{aai_url}/oauth/token",
        auth=(client_id, client_secret),
        data={
            "grant_type": "password",
            "username": username,
            "password": password,
        },
        timeout=timeout,
    )
    response.raise_for_status()

    try:
        payload = response.json()
    except (TypeError, ValueError) as error:
        raise RuntimeError("AAI token response was not valid JSON") from error
    if not isinstance(payload, dict):
        raise RuntimeError("AAI token response was not a JSON object")

    token = payload.get("access_token")
    if not token:
        raise RuntimeError("AAI token response did not contain access_token")
    return str(token)


def register_user(
    aai_url: str,
    *,
    name: str,
    surname: str,
    email: str,
    password: str,
    timeout: int = DEFAULT_TIMEOUT,
) -> UserCredentials:
    aai_url = aai_url.strip().rstrip("/")
    if not aai_url:
        raise ValueError("AAI URL must not be empty")
    if not name.strip() or not surname.strip() or not email.strip() or not password:
        raise ValueError(
            "AAI user registration requires name, surname, email, and password"
        )

    email = email.strip().lower()
    register_url = f"{aai_url}/register"
    with requests.Session() as session:
        page = session.get(register_url, timeout=timeout)
        page.raise_for_status()

        parser = _CSRFTokenParser()
        parser.feed(page.text)
        if not parser.token:
            raise RuntimeError("AAI registration page did not contain csrf_token")

        response = session.post(
            register_url,
            data={
                "name": name,
                "surname": surname,
                "email": email,
                "password": password,
                "csrf_token": parser.token,
            },
            allow_redirects=False,
            timeout=timeout,
        )
        if response.status_code != 303 or not response.headers.get("Location"):
            raise RuntimeError(
                "AAI user registration did not return the expected success redirect "
                f"(status {response.status_code})"
            )

    return UserCredentials(username=email, password=password)


def _setting(
    value: str | None,
    environment_name: str,
    default: str,
    *,
    strip: bool = True,
) -> str:
    result = value if value is not None else os.getenv(environment_name, default)
    if strip:
        result = result.strip()
    if not result:
        raise RuntimeError(f"{environment_name} must not be empty")
    return result
