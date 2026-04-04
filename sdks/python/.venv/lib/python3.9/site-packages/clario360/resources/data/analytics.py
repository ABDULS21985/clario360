from __future__ import annotations

from clario360.models.common import MetricsSnapshot
from clario360.models.data import AnalyticsResult
from clario360.resources._base import BaseResource


class AnalyticsResource(BaseResource[AnalyticsResult]):
    def query(self, payload: dict[str, object]) -> AnalyticsResult:
        return self._post_at("/api/v1/data/analytics/query", AnalyticsResult, payload)

    def saved_queries(self) -> MetricsSnapshot:
        return self._metrics("/api/v1/data/analytics/saved")
