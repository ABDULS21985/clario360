from __future__ import annotations

from typing import List

from clario360.models.risk import Heatmap, Recommendation, RiskScore, RiskTrendPoint
from clario360.resources._base import BaseResource


class RiskResource(BaseResource[RiskScore]):
    def score(self) -> RiskScore:
        return self._get_at("/api/v1/cyber/risk/score", RiskScore)

    def trend(self) -> List[RiskTrendPoint]:
        return self._list_models_at("/api/v1/cyber/risk/score/trend", RiskTrendPoint)

    def heatmap(self) -> Heatmap:
        payload = self._http.get("/api/v1/cyber/risk/heatmap")
        return Heatmap.from_payload(self._unwrap_mapping(payload))

    def recommendations(self) -> List[Recommendation]:
        return self._list_models_at("/api/v1/cyber/risk/recommendations", Recommendation)
