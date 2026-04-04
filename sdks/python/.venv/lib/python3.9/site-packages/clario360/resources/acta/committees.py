from __future__ import annotations

from clario360.models.acta import Committee
from clario360.models.common import PaginatedResponse
from clario360.resources._base import BaseResource


class CommitteesResource(BaseResource[Committee]):
    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[Committee]:
        return self._list(params={"page": page, "per_page": per_page})

    def get(self, committee_id: str) -> Committee:
        return self._get(committee_id)

    def create(self, payload: dict[str, object]) -> Committee:
        return self._create(payload)

    def update(self, committee_id: str, payload: dict[str, object]) -> Committee:
        return self._update(committee_id, payload)
