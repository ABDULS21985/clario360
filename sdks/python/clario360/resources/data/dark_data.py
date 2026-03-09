from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.common import MetricsSnapshot, PaginatedResponse
from clario360.models.quality import DarkDataAsset
from clario360.resources._base import BaseResource


class DarkDataResource(BaseResource[DarkDataAsset]):
    def __init__(
        self,
        http: HTTPClient,
        async_http: AsyncHTTPClient,
        base_path: str,
        model_class: type[DarkDataAsset],
    ) -> None:
        super().__init__(http, async_http, base_path, model_class)
        self._base = "/api/v1/data/dark-data"

    def scan(self, payload: dict[str, object]) -> MetricsSnapshot:
        return self._post_at(f"{self._base}/scan", MetricsSnapshot, payload)

    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[DarkDataAsset]:
        return self._list(params={"page": page, "per_page": per_page})

    def get(self, asset_id: str) -> DarkDataAsset:
        return self._get(asset_id)

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")
