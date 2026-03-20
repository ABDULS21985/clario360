from __future__ import annotations

import pytest

from clario360.client import Clario360
from clario360.exceptions import ConfigurationError
from tests.conftest import make_jwt


def test_init_with_api_key(api_url: str) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    assert client.auth_state.api_key == "clario360_ak_test"
    client.close()


def test_init_with_email_password(api_url: str, router) -> None:
    router.add_json(
        "POST",
        f"{api_url}/api/v1/auth/login",
        {
            "access_token": "token-1",
            "refresh_token": "refresh-1",
            "user": {
                "id": "user-1",
                "tenant_id": "tenant-1",
                "email": "analyst@example.com",
                "first_name": "Clario",
                "last_name": "Analyst",
                "status": "active",
            },
        },
    )
    client = Clario360(api_url=api_url, email="analyst@example.com", password="secret")
    assert client.auth_state.access_token == "token-1"
    assert client.auth_state.refresh_token == "refresh-1"
    client.close()


def test_init_from_env(monkeypatch: pytest.MonkeyPatch, api_url: str) -> None:
    monkeypatch.setenv("CLARIO360_API_URL", api_url)
    monkeypatch.setenv("CLARIO360_API_KEY", "clario360_ak_env")
    client = Clario360()
    assert client.auth_state.api_key == "clario360_ak_env"
    client.close()


def test_init_no_auth(monkeypatch: pytest.MonkeyPatch, api_url: str) -> None:
    monkeypatch.setenv("CLARIO360_API_URL", api_url)
    for name in [
        "CLARIO360_API_KEY",
        "CLARIO360_ACCESS_TOKEN",
        "CLARIO360_REFRESH_TOKEN",
        "CLARIO360_EMAIL",
        "CLARIO360_PASSWORD",
    ]:
        monkeypatch.delenv(name, raising=False)
    with pytest.raises(ConfigurationError):
        Clario360()


def test_whoami_and_health(api_url: str, router, api_key_client: Clario360) -> None:
    router.add_json("GET", f"{api_url}/healthz", {"status": "ok", "service": "gateway"})
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

    health = api_key_client.health()
    user = api_key_client.whoami()

    assert health.status == "ok"
    assert user.email == "admin@example.com"
    api_key_client.close()


def test_tenant_id_from_jwt(api_url: str) -> None:
    token = make_jwt({"tid": "tenant-jwt"})
    client = Clario360(api_url=api_url, access_token=token)
    assert client.tenant_id == "tenant-jwt"
    client.close()
