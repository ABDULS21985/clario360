from __future__ import annotations

from typing import AsyncIterator, Iterator, Optional

from clario360.models.common import CountSnapshot, MetricsSnapshot, PaginatedResponse
from clario360.models.cyber import Alert, AlertComment, AlertTimelineEntry
from clario360.resources._base import BaseResource


class AlertsResource(BaseResource[Alert]):
    def list(
        self,
        *,
        severity: Optional[str] = None,
        status: Optional[str] = None,
        search: Optional[str] = None,
        page: int = 1,
        per_page: int = 50,
        sort: str = "created_at",
        order: str = "desc",
    ) -> PaginatedResponse[Alert]:
        return self._list(
            params={
                "severity": severity,
                "status": status,
                "search": search,
                "page": page,
                "per_page": per_page,
                "sort": sort,
                "order": order,
            }
        )

    async def alist(
        self,
        *,
        severity: Optional[str] = None,
        status: Optional[str] = None,
        search: Optional[str] = None,
        page: int = 1,
        per_page: int = 50,
        sort: str = "created_at",
        order: str = "desc",
    ) -> PaginatedResponse[Alert]:
        return await self._alist(
            params={
                "severity": severity,
                "status": status,
                "search": search,
                "page": page,
                "per_page": per_page,
                "sort": sort,
                "order": order,
            }
        )

    def list_all(self, *, severity: Optional[str] = None, status: Optional[str] = None) -> Iterator[Alert]:
        return self._list_all(params={"severity": severity, "status": status})

    async def alist_all(self, *, severity: Optional[str] = None, status: Optional[str] = None) -> AsyncIterator[Alert]:
        async for item in self._alist_all(params={"severity": severity, "status": status}):
            yield item

    def get(self, alert_id: str) -> Alert:
        return self._get(alert_id)

    def acknowledge(self, alert_id: str) -> Alert:
        return self._put_at(f"{self._base}/{alert_id}/status", Alert, {"status": "acknowledged"})

    def assign(self, alert_id: str, assigned_to: str) -> Alert:
        return self._put_at(f"{self._base}/{alert_id}/assign", Alert, {"assigned_to": assigned_to})

    def close(self, alert_id: str, resolution: str = "resolved") -> Alert:
        return self._put_at(f"{self._base}/{alert_id}/status", Alert, {"status": resolution})

    def add_comment(self, alert_id: str, comment: str) -> AlertComment:
        return self._post_at(f"{self._base}/{alert_id}/comment", AlertComment, {"comment": comment})

    def comments(self, alert_id: str) -> list[AlertComment]:
        return self._list_models_at(f"{self._base}/{alert_id}/comments", AlertComment)

    def timeline(self, alert_id: str) -> list[AlertTimelineEntry]:
        return self._list_models_at(f"{self._base}/{alert_id}/timeline", AlertTimelineEntry)

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")

    def count(self, *, status: Optional[str] = None, severity: Optional[str] = None) -> CountSnapshot:
        suffix = []
        if status:
            suffix.append(f"status={status}")
        if severity:
            suffix.append(f"severity={severity}")
        path = f"{self._base}/count"
        if suffix:
            path = f"{path}?{'&'.join(suffix)}"
        return self._counts(path)
