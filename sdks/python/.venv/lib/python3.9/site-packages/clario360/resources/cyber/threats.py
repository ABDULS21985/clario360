from __future__ import annotations

from typing import List, Optional

from clario360.models.common import MetricsSnapshot, PaginatedResponse
from clario360.models.cyber import IndicatorCheckResult, Threat
from clario360.resources._base import BaseResource


class ThreatsResource(BaseResource[Threat]):
    def list(self, *, status: Optional[str] = None, page: int = 1, per_page: int = 50) -> PaginatedResponse[Threat]:
        return self._list(params={"status": status, "page": page, "per_page": per_page})

    def get(self, threat_id: str) -> Threat:
        return self._get(threat_id)

    def check_indicators(self, indicators: List[str]) -> IndicatorCheckResult:
        return self._post_at("/api/v1/cyber/indicators/check", IndicatorCheckResult, {"indicators": indicators})

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")
