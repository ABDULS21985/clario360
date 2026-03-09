from __future__ import annotations

from typing import List

from clario360.models.common import MetricsSnapshot, PaginatedResponse
from clario360.models.lex import Clause, Contract
from clario360.resources._base import BaseResource


class ContractsResource(BaseResource[Contract]):
    def list(self, *, page: int = 1, per_page: int = 50, search: str | None = None) -> PaginatedResponse[Contract]:
        return self._list(params={"page": page, "per_page": per_page, "search": search})

    def get(self, contract_id: str) -> Contract:
        return self._get(contract_id)

    def create(self, payload: dict[str, object]) -> Contract:
        return self._create(payload)

    def update(self, contract_id: str, payload: dict[str, object]) -> Contract:
        return self._update(contract_id, payload)

    def clauses(self, contract_id: str) -> List[Clause]:
        return self._list_models_at(f"{self._base}/{contract_id}/clauses", Clause)

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")
