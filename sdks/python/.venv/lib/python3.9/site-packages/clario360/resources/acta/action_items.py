from __future__ import annotations

from clario360.models.acta import ActionItem
from clario360.models.common import MetricsSnapshot, PaginatedResponse
from clario360.resources._base import BaseResource


class ActionItemsResource(BaseResource[ActionItem]):
    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[ActionItem]:
        return self._list(params={"page": page, "per_page": per_page})

    def get(self, action_item_id: str) -> ActionItem:
        return self._get(action_item_id)

    def create(self, payload: dict[str, object]) -> ActionItem:
        return self._create(payload)

    def update_status(self, action_item_id: str, status: str) -> ActionItem:
        return self._put_at(f"{self._base}/{action_item_id}/status", ActionItem, {"status": status})

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")
