from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.common import PaginatedResponse
from clario360.models.ueba import BehavioralProfile, UEBAAlert, UEBAConfig, UEBADashboard, UEBARiskRankingEntry, UEBATimelineEntry
from clario360.resources._base import BaseResource


class UEBAResource(BaseResource[BehavioralProfile]):
    def __init__(
        self,
        http: HTTPClient,
        async_http: AsyncHTTPClient,
        base_path: str,
        model_class: type[BehavioralProfile],
    ) -> None:
        super().__init__(http, async_http, base_path, model_class)
        self._base = "/api/v1/cyber/ueba/profiles"

    def profiles(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[BehavioralProfile]:
        return self._list(params={"page": page, "per_page": per_page})

    def profile(self, entity_id: str) -> BehavioralProfile:
        return self._get(entity_id)

    def timeline(self, entity_id: str, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[UEBATimelineEntry]:
        payload = self._http.get(f"{self._base}/{entity_id}/timeline", params={"page": page, "per_page": per_page})
        return PaginatedResponse.from_payload(self._ensure_mapping(payload), UEBATimelineEntry)

    def alerts(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[UEBAAlert]:
        payload = self._http.get("/api/v1/cyber/ueba/alerts", params={"page": page, "per_page": per_page})
        return PaginatedResponse.from_payload(self._ensure_mapping(payload), UEBAAlert)

    def dashboard(self) -> UEBADashboard:
        return self._get_at("/api/v1/cyber/ueba/dashboard", UEBADashboard)

    def risk_ranking(self) -> list[UEBARiskRankingEntry]:
        return self._list_models_at("/api/v1/cyber/ueba/risk-ranking", UEBARiskRankingEntry)

    def config(self) -> UEBAConfig:
        return self._get_at("/api/v1/cyber/ueba/config", UEBAConfig)
