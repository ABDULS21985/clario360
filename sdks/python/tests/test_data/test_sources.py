from __future__ import annotations

from clario360.client import Clario360


def test_list_sources(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add_json(
        "GET",
        f"{api_url}/api/v1/data/sources",
        {
            "data": [{"id": "s1", "name": "Postgres", "type": "postgres", "status": "active"}],
            "meta": {"page": 1, "per_page": 25, "total": 1, "total_pages": 1},
        },
    )
    result = client.data.sources.list(per_page=25)
    assert result.data[0].name == "Postgres"
    client.close()


def test_test_source(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add_json("POST", f"{api_url}/api/v1/data/sources/source-1/test", {"data": {"success": True, "latency_ms": 22}})
    result = client.data.sources.test("source-1")
    assert result.metrics["success"] is True
    client.close()
