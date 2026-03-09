from __future__ import annotations

from typing import Optional

from clario360.models.common import MetricsSnapshot, PaginatedResponse
from clario360.models.cyber import Rule
from clario360.resources._base import BaseResource


class RulesResource(BaseResource[Rule]):
    def list(self, *, enabled: Optional[bool] = None, page: int = 1, per_page: int = 50) -> PaginatedResponse[Rule]:
        return self._list(params={"enabled": enabled, "page": page, "per_page": per_page})

    def get(self, rule_id: str) -> Rule:
        return self._get(rule_id)

    def create(self, payload: dict[str, object]) -> Rule:
        return self._create(payload)

    def update(self, rule_id: str, payload: dict[str, object]) -> Rule:
        return self._update(rule_id, payload)

    def toggle(self, rule_id: str) -> Rule:
        return self._put_at(f"{self._base}/{rule_id}/toggle", Rule, {})

    def test(self, rule_id: str, payload: dict[str, object]) -> Rule:
        return self._post_at(f"{self._base}/{rule_id}/test", Rule, payload)

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")
