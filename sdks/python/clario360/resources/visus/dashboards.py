from __future__ import annotations

from clario360.models.common import PaginatedResponse
from clario360.models.visus import Dashboard
from clario360.resources._base import BaseResource


class DashboardsResource(BaseResource[Dashboard]):
    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[Dashboard]:
        return self._list(params={"page": page, "per_page": per_page})

    def get(self, dashboard_id: str) -> Dashboard:
        return self._get(dashboard_id)

    def create(self, payload: dict[str, object]) -> Dashboard:
        return self._create(payload)
