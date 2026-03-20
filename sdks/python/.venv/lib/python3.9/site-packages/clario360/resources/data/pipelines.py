from __future__ import annotations

from clario360.models.common import CountSnapshot, MetricsSnapshot, PaginatedResponse
from clario360.models.data import Pipeline, PipelineRun
from clario360.resources._base import BaseResource


class PipelinesResource(BaseResource[Pipeline]):
    def list(self, *, page: int = 1, per_page: int = 50, search: str | None = None) -> PaginatedResponse[Pipeline]:
        return self._list(params={"page": page, "per_page": per_page, "search": search})

    def get(self, pipeline_id: str) -> Pipeline:
        return self._get(pipeline_id)

    def create(self, payload: dict[str, object]) -> Pipeline:
        return self._create(payload)

    def update(self, pipeline_id: str, payload: dict[str, object]) -> Pipeline:
        return self._update(pipeline_id, payload)

    def delete(self, pipeline_id: str) -> None:
        self._delete(pipeline_id)

    def run(self, pipeline_id: str) -> PipelineRun:
        return self._post_at(f"{self._base}/{pipeline_id}/run", PipelineRun, {})

    def runs(self, pipeline_id: str, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[PipelineRun]:
        payload = self._http.get(f"{self._base}/{pipeline_id}/runs", params={"page": page, "per_page": per_page})
        return PaginatedResponse.from_payload(self._ensure_mapping(payload), PipelineRun)

    def run_status(self, pipeline_id: str, run_id: str) -> PipelineRun:
        return self._get_at(f"{self._base}/{pipeline_id}/runs/{run_id}", PipelineRun)

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")

    def count(self) -> CountSnapshot:
        return self._counts(f"{self._base}/count")
