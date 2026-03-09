from __future__ import annotations

import pytest

from clario360.client import Clario360
from clario360.exceptions import ConfigurationError
from clario360.models.common import PaginatedResponse
from clario360.models.cyber import Alert


def test_paginated_response_parses_meta() -> None:
    response = PaginatedResponse.from_payload(
        {
            "data": [{"id": "a1", "title": "Alert", "severity": "high", "status": "new", "created_at": "2026-03-09T00:00:00Z"}],
            "meta": {"page": 2, "per_page": 10, "total": 30, "total_pages": 3},
        },
        Alert,
    )
    assert response.page == 2
    assert response.total_pages == 3
    assert len(response) == 1


def test_list_all_multi_page(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add_json(
        "GET",
        f"{api_url}/api/v1/cyber/alerts",
        {
            "data": [{"id": "a1", "title": "A1", "severity": "high", "status": "new", "created_at": "2026-03-09T00:00:00Z"}],
            "meta": {"page": 1, "per_page": 200, "total": 2, "total_pages": 2},
        },
    )
    router.add_json(
        "GET",
        f"{api_url}/api/v1/cyber/alerts",
        {
            "data": [{"id": "a2", "title": "A2", "severity": "critical", "status": "new", "created_at": "2026-03-09T00:00:00Z"}],
            "meta": {"page": 2, "per_page": 200, "total": 2, "total_pages": 2},
        },
    )
    alerts = list(client.cyber.alerts.list_all())
    assert [alert.id for alert in alerts] == ["a1", "a2"]
    client.close()


def test_to_dataframe_requires_pandas() -> None:
    response = PaginatedResponse.from_payload(
        {
            "data": [{"id": "a1", "title": "Alert", "severity": "high", "status": "new", "created_at": "2026-03-09T00:00:00Z"}],
            "meta": {"page": 1, "per_page": 10, "total": 1, "total_pages": 1},
        },
        Alert,
    )
    try:
        import pandas  # noqa: F401
    except ImportError:
        with pytest.raises(ConfigurationError):
            response.to_dataframe()
    else:
        dataframe = response.to_dataframe()
        assert list(dataframe["id"]) == ["a1"]
