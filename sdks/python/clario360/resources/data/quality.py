from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.common import MetricsSnapshot, PaginatedResponse
from clario360.models.quality import QualityResult, QualityRule, QualityScore, QualityTrendPoint
from clario360.resources._base import BaseResource


class QualityResource(BaseResource[QualityRule]):
    def __init__(
        self,
        http: HTTPClient,
        async_http: AsyncHTTPClient,
        base_path: str,
        model_class: type[QualityRule],
    ) -> None:
        super().__init__(http, async_http, base_path, model_class)
        self._base = "/api/v1/data/quality/rules"

    def rules(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[QualityRule]:
        return self._list(params={"page": page, "per_page": per_page})

    def get_rule(self, rule_id: str) -> QualityRule:
        return self._get(rule_id)

    def create_rule(self, payload: dict[str, object]) -> QualityRule:
        return self._create(payload)

    def update_rule(self, rule_id: str, payload: dict[str, object]) -> QualityRule:
        return self._update(rule_id, payload)

    def score(self) -> QualityScore:
        return self._get_at("/api/v1/data/quality/score", QualityScore)

    def trend(self, *, days: int | None = None) -> list[QualityTrendPoint]:
        path = "/api/v1/data/quality/score/trend"
        if days is not None:
            path = f"{path}?days={days}"
        return self._list_models_at(path, QualityTrendPoint)

    def dashboard(self) -> MetricsSnapshot:
        return self._metrics("/api/v1/data/quality/dashboard")

    def results(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[QualityResult]:
        payload = self._http.get("/api/v1/data/quality/results", params={"page": page, "per_page": per_page})
        return PaginatedResponse.from_payload(self._ensure_mapping(payload), QualityResult)
