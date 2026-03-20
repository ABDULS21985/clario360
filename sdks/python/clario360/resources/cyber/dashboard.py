from __future__ import annotations

from clario360.models.common import MetricsSnapshot
from clario360.models.cyber import DashboardSnapshot
from clario360.resources._base import BaseResource


class DashboardResource(BaseResource[DashboardSnapshot]):
    def summary(self) -> DashboardSnapshot:
        return self._get_at("/api/v1/cyber/dashboard", DashboardSnapshot)

    def kpis(self) -> MetricsSnapshot:
        return self._metrics("/api/v1/cyber/dashboard/kpis")
