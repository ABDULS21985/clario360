from __future__ import annotations

from clario360.client import Clario360


def test_list_and_run_pipeline(api_url: str, router) -> None:
    client = Clario360(api_url=api_url, api_key="clario360_ak_test")
    router.add_json(
        "GET",
        f"{api_url}/api/v1/data/pipelines",
        {
            "data": [{"id": "p1", "name": "Ingest CRM", "status": "idle"}],
            "meta": {"page": 1, "per_page": 25, "total": 1, "total_pages": 1},
        },
    )
    router.add_json(
        "POST",
        f"{api_url}/api/v1/data/pipelines/p1/run",
        {"data": {"id": "run-1", "pipeline_id": "p1", "status": "running"}},
    )
    pipelines = client.data.pipelines.list(per_page=25)
    run = client.data.pipelines.run("p1")
    assert pipelines.data[0].id == "p1"
    assert run.status == "running"
    client.close()
