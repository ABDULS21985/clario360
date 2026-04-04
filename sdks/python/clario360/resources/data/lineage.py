from __future__ import annotations

from clario360.models.common import MetricsSnapshot
from clario360.models.lineage import ImpactAnalysis, LineageGraph
from clario360.resources._base import BaseResource


class LineageResource(BaseResource[LineageGraph]):
    def graph(self) -> LineageGraph:
        return self._get_at("/api/v1/data/lineage/graph", LineageGraph)

    def entity(self, entity_type: str, entity_id: str) -> LineageGraph:
        return self._get_at(f"/api/v1/data/lineage/graph/{entity_type}/{entity_id}", LineageGraph)

    def impact(self, entity_type: str, entity_id: str) -> ImpactAnalysis:
        return self._get_at(f"/api/v1/data/lineage/impact/{entity_type}/{entity_id}", ImpactAnalysis)

    def stats(self) -> MetricsSnapshot:
        return self._metrics("/api/v1/data/lineage/stats")
