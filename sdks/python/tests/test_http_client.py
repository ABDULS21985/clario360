from __future__ import annotations

import pytest
import requests

from clario360.client import Clario360
from clario360.exceptions import Clario360Error, NotFoundError, RateLimitError, ServerError
from tests.conftest import MockResponse


def test_retry_on_500(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test", max_retries=2)
    router.add("GET", f"{api_url}/healthz", MockResponse(status_code=500, payload={"error": {"code": "SERVER", "message": "boom"}}))
    router.add_json("GET", f"{api_url}/healthz", {"status": "ok"})
    assert client.health().status == "ok"
    assert len(router.requests) == 2
    client.close()


def test_retry_on_429(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test", max_retries=2)
    router.add(
        "GET",
        f"{api_url}/healthz",
        MockResponse(
            status_code=429,
            payload={"error": {"code": "RATE_LIMITED", "message": "slow down"}},
            headers={"Content-Type": "application/json", "Retry-After": "1"},
        ),
    )
    router.add_json("GET", f"{api_url}/healthz", {"status": "ok"})
    assert client.health().status == "ok"
    assert len(router.requests) == 2
    client.close()


def test_no_retry_on_404(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add("GET", f"{api_url}/missing", MockResponse(status_code=404, payload={"error": {"code": "NOT_FOUND", "message": "gone"}}))
    with pytest.raises(NotFoundError):
        client._http.get("/missing")  # noqa: SLF001
    client.close()


def test_max_retries_exceeded(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test", max_retries=1)
    router.add("GET", f"{api_url}/healthz", MockResponse(status_code=500, payload={"error": {"code": "SERVER", "message": "boom"}}))
    router.add("GET", f"{api_url}/healthz", MockResponse(status_code=500, payload={"error": {"code": "SERVER", "message": "boom again"}}))
    with pytest.raises(ServerError):
        client.health()
    client.close()


def test_connection_error_retry(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test", max_retries=1)
    router.add("GET", f"{api_url}/healthz", requests.RequestException("network down"))
    router.add_json("GET", f"{api_url}/healthz", {"status": "ok"})
    assert client.health().status == "ok"
    client.close()


def test_timeout_exhaustion(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test", max_retries=0)
    router.add("GET", f"{api_url}/healthz", requests.Timeout("timed out"))
    with pytest.raises(Clario360Error):
        client.health()
    client.close()


def test_rate_limit_error_raises_when_retries_exhausted(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test", max_retries=0)
    router.add(
        "GET",
        f"{api_url}/healthz",
        MockResponse(
            status_code=429,
            payload={"error": {"code": "RATE_LIMITED", "message": "slow down"}},
            headers={"Content-Type": "application/json", "Retry-After": "3"},
        ),
    )
    with pytest.raises(RateLimitError) as exc:
        client.health()
    assert exc.value.retry_after == 3
    client.close()
