from __future__ import annotations

import pytest

from clario360.client import Clario360
from clario360.exceptions import ConflictError, GovernanceError, ValidationError
from tests.conftest import MockResponse


def test_400_validation(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add(
        "GET",
        f"{api_url}/invalid",
        MockResponse(status_code=400, payload={"error": {"code": "VALIDATION_ERROR", "message": "bad request", "details": {"field": "severity"}}}),
    )
    with pytest.raises(ValidationError) as exc:
        client._http.get("/invalid")  # noqa: SLF001
    assert exc.value.details["field"] == "severity"
    client.close()


def test_403_governance(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add(
        "GET",
        f"{api_url}/governed",
        MockResponse(status_code=403, payload={"error": {"code": "GOVERNANCE_VIOLATION", "message": "approval required"}}),
    )
    with pytest.raises(GovernanceError):
        client._http.get("/governed")  # noqa: SLF001
    client.close()


def test_409_conflict(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add(
        "GET",
        f"{api_url}/conflict",
        MockResponse(status_code=409, payload={"error": {"code": "STATE_CONFLICT", "message": "duplicate"}}),
    )
    with pytest.raises(ConflictError):
        client._http.get("/conflict")  # noqa: SLF001
    client.close()
