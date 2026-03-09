from __future__ import annotations

from clario360.models.common import MetricsSnapshot, PaginatedResponse
from clario360.models.data import DataSource, SourceType
from clario360.resources._base import BaseResource


class SourcesResource(BaseResource[DataSource]):
    def list(self, *, page: int = 1, per_page: int = 50, search: str | None = None) -> PaginatedResponse[DataSource]:
        return self._list(params={"page": page, "per_page": per_page, "search": search})

    def get(self, source_id: str) -> DataSource:
        return self._get(source_id)

    def create(self, payload: dict[str, object]) -> DataSource:
        return self._create(payload)

    def update(self, source_id: str, payload: dict[str, object]) -> DataSource:
        return self._update(source_id, payload)

    def delete(self, source_id: str) -> None:
        self._delete(source_id)

    def test(self, source_id: str) -> MetricsSnapshot:
        return self._post_at(f"{self._base}/{source_id}/test", MetricsSnapshot, {})

    def discover(self, source_id: str) -> MetricsSnapshot:
        return self._post_at(f"{self._base}/{source_id}/discover", MetricsSnapshot, {})

    def sync(self, source_id: str) -> MetricsSnapshot:
        return self._post_at(f"{self._base}/{source_id}/sync", MetricsSnapshot, {})

    def source_types(self) -> list[SourceType]:
        return self._list_models_at("/api/v1/data/source-types", SourceType)

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")
