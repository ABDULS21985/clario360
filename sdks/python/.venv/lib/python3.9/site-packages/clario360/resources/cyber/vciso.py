from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.common import PaginatedResponse, StringListResponse
from clario360.models.vciso import BriefingHistoryEntry, ConversationSummary, ExecutiveBriefing, PostureSummary, SecurityRecommendation, VCISOReport
from clario360.resources._base import BaseResource


class VCISOResource(BaseResource[ExecutiveBriefing]):
    def __init__(
        self,
        http: HTTPClient,
        async_http: AsyncHTTPClient,
        base_path: str,
        model_class: type[ExecutiveBriefing],
    ) -> None:
        super().__init__(http, async_http, base_path, model_class)
        self._base = "/api/v1/cyber/vciso"

    def briefing(self, *, period_days: int | None = None) -> ExecutiveBriefing:
        query = f"?period_days={period_days}" if period_days is not None else ""
        return self._get_at(f"{self._base}/briefing{query}", ExecutiveBriefing)

    def briefing_history(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[BriefingHistoryEntry]:
        payload = self._http.get(f"{self._base}/briefing/history", params={"page": page, "per_page": per_page})
        return PaginatedResponse.from_payload(self._ensure_mapping(payload), BriefingHistoryEntry)

    def recommendations(self) -> list[SecurityRecommendation]:
        return self._list_models_at(f"{self._base}/recommendations", SecurityRecommendation)

    def report(self, payload: dict[str, object]) -> VCISOReport:
        return self._post_at(f"{self._base}/report", VCISOReport, payload)

    def posture_summary(self) -> PostureSummary:
        return self._get_at(f"{self._base}/posture-summary", PostureSummary)

    def conversations(self) -> list[ConversationSummary]:
        return self._list_models_at(f"{self._base}/conversations", ConversationSummary)

    def suggestions(self) -> StringListResponse:
        payload = self._http.get(f"{self._base}/suggestions")
        data = self._unwrap(payload)
        if isinstance(data, list):
            return StringListResponse.from_payload(data)
        return StringListResponse(values=[])
