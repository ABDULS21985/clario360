from __future__ import annotations

from clario360.client import Clario360


def test_list_alerts_with_filters(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add_json(
        "GET",
        f"{api_url}/api/v1/cyber/alerts",
        {
            "data": [{"id": "a1", "title": "Alert A", "severity": "critical", "status": "new", "created_at": "2026-03-09T00:00:00Z"}],
            "meta": {"page": 1, "per_page": 5, "total": 1, "total_pages": 1},
        },
    )
    result = client.cyber.alerts.list(severity="critical", status="new", per_page=5)
    assert result.data[0].id == "a1"
    assert router.requests[0].params == {
        "severity": "critical",
        "status": "new",
        "page": 1,
        "per_page": 5,
        "sort": "created_at",
        "order": "desc",
    }
    client.close()


def test_get_ack_comment_and_close(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add_json(
        "GET",
        f"{api_url}/api/v1/cyber/alerts/a1",
        {"data": {"id": "a1", "title": "Alert A", "severity": "critical", "status": "new", "created_at": "2026-03-09T00:00:00Z"}},
    )
    router.add_json(
        "PUT",
        f"{api_url}/api/v1/cyber/alerts/a1/status",
        {"data": {"id": "a1", "title": "Alert A", "severity": "critical", "status": "acknowledged", "created_at": "2026-03-09T00:00:00Z"}},
    )
    router.add_json(
        "POST",
        f"{api_url}/api/v1/cyber/alerts/a1/comment",
        {"data": {"id": "c1", "body": "Investigating", "created_at": "2026-03-09T00:00:00Z"}},
    )
    router.add_json(
        "PUT",
        f"{api_url}/api/v1/cyber/alerts/a1/status",
        {"data": {"id": "a1", "title": "Alert A", "severity": "critical", "status": "resolved", "created_at": "2026-03-09T00:00:00Z"}},
    )

    alert = client.cyber.alerts.get("a1")
    acknowledged = client.cyber.alerts.acknowledge("a1")
    comment = client.cyber.alerts.add_comment("a1", "Investigating")
    closed = client.cyber.alerts.close("a1")

    assert alert.id == "a1"
    assert acknowledged.status == "acknowledged"
    assert comment.body == "Investigating"
    assert closed.status == "resolved"
    client.close()
