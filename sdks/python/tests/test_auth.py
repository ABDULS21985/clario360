from __future__ import annotations

import pytest

from clario360.client import Clario360
from clario360.exceptions import AuthenticationError
from tests.conftest import MockResponse


def test_api_key_header(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add_json("GET", f"{api_url}/healthz", {"status": "ok"})
    client.health()
    assert router.requests[-1].headers["X-API-Key"] == "clario360_ak_test"
    client.close()


def test_jwt_header(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, access_token="jwt-token")
    router.add_json("GET", f"{api_url}/healthz", {"status": "ok"})
    client.health()
    assert router.requests[-1].headers["Authorization"] == "Bearer jwt-token"
    client.close()


def test_token_refresh_on_401(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, access_token="expired-token", refresh_token="refresh-token")
    router.add("GET", f"{api_url}/api/v1/users/me", MockResponse(status_code=401, payload={"error": {"code": "AUTH_REQUIRED", "message": "expired"}}))
    router.add_json(
        "POST",
        f"{api_url}/api/v1/auth/refresh",
        {
            "access_token": "fresh-token",
            "refresh_token": "refresh-token-2",
            "user": {
                "id": "user-1",
                "tenant_id": "tenant-1",
                "email": "admin@example.com",
                "first_name": "Admin",
                "last_name": "User",
                "status": "active",
            },
        },
    )
    router.add_json(
        "GET",
        f"{api_url}/api/v1/users/me",
        {
            "data": {
                "id": "user-1",
                "tenant_id": "tenant-1",
                "email": "admin@example.com",
                "first_name": "Admin",
                "last_name": "User",
                "status": "active",
            }
        },
    )

    user = client.whoami()

    assert user.email == "admin@example.com"
    assert router.requests[2].headers["Authorization"] == "Bearer fresh-token"
    client.close()


def test_api_key_no_refresh_on_401(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add("GET", f"{api_url}/api/v1/users/me", MockResponse(status_code=401, payload={"error": {"code": "AUTH_REQUIRED", "message": "denied"}}))
    with pytest.raises(AuthenticationError):
        client.whoami()
    assert len(router.requests) == 1
    client.close()
