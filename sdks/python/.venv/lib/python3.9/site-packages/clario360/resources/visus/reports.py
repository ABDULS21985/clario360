from __future__ import annotations

from clario360.models.common import PaginatedResponse
from clario360.models.visus import Report
from clario360.resources._base import BaseResource


class ReportsResource(BaseResource[Report]):
    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[Report]:
        return self._list(params={"page": page, "per_page": per_page})

    def get(self, report_id: str) -> Report:
        return self._get(report_id)

    def create(self, payload: dict[str, object]) -> Report:
        return self._create(payload)

    def generate(self, report_id: str) -> Report:
        return self._post_at(f"{self._base}/{report_id}/generate", Report, {})
