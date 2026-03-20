from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.visus import Dashboard, KPI, Report
from clario360.resources.visus.dashboards import DashboardsResource
from clario360.resources.visus.kpis import KPIsResource
from clario360.resources.visus.reports import ReportsResource


class VisusNamespace:
    def __init__(self, http: HTTPClient, async_http: AsyncHTTPClient) -> None:
        self.dashboards = DashboardsResource(http, async_http, "/api/v1/visus/dashboards", Dashboard)
        self.kpis = KPIsResource(http, async_http, "/api/v1/visus/kpis", KPI)
        self.reports = ReportsResource(http, async_http, "/api/v1/visus/reports", Report)


__all__ = ["VisusNamespace"]
