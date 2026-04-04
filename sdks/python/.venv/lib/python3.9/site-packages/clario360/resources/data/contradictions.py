from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.common import MetricsSnapshot, PaginatedResponse
from clario360.models.quality import Contradiction, ContradictionScan
from clario360.resources._base import BaseResource


class ContradictionsResource(BaseResource[Contradiction]):
    def __init__(
        self,
        http: HTTPClient,
        async_http: AsyncHTTPClient,
        base_path: str,
        model_class: type[Contradiction],
    ) -> None:
        super().__init__(http, async_http, base_path, model_class)
        self._base = "/api/v1/data/contradictions"

    def scan(self, payload: dict[str, object]) -> ContradictionScan:
        return self._post_at("/api/v1/data/contradictions/scan", ContradictionScan, payload)

    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[Contradiction]:
        return self._list(params={"page": page, "per_page": per_page})

    def get(self, contradiction_id: str) -> Contradiction:
        return self._get(contradiction_id)

    def resolve(self, contradiction_id: str, payload: dict[str, object]) -> Contradiction:
        return self._post_at(f"{self._base}/{contradiction_id}/resolve", Contradiction, payload)

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")
