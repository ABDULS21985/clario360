from __future__ import annotations

from clario360.models.common import MetricsSnapshot, PaginatedResponse
from clario360.models.visus import KPI
from clario360.resources._base import BaseResource


class KPIsResource(BaseResource[KPI]):
    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[KPI]:
        return self._list(params={"page": page, "per_page": per_page})

    def get(self, kpi_id: str) -> KPI:
        return self._get(kpi_id)

    def summary(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/summary")
