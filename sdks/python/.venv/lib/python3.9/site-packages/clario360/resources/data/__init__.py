from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.data import AnalyticsResult, DataSource, Pipeline
from clario360.models.lineage import LineageGraph
from clario360.models.quality import Contradiction, DarkDataAsset, QualityRule
from clario360.resources.data.analytics import AnalyticsResource
from clario360.resources.data.contradictions import ContradictionsResource
from clario360.resources.data.dark_data import DarkDataResource
from clario360.resources.data.lineage import LineageResource
from clario360.resources.data.pipelines import PipelinesResource
from clario360.resources.data.quality import QualityResource
from clario360.resources.data.sources import SourcesResource


class DataNamespace:
    def __init__(self, http: HTTPClient, async_http: AsyncHTTPClient) -> None:
        self.sources = SourcesResource(http, async_http, "/api/v1/data/sources", DataSource)
        self.pipelines = PipelinesResource(http, async_http, "/api/v1/data/pipelines", Pipeline)
        self.quality = QualityResource(http, async_http, "/api/v1/data/quality/rules", QualityRule)
        self.contradictions = ContradictionsResource(http, async_http, "/api/v1/data/contradictions", Contradiction)
        self.lineage = LineageResource(http, async_http, "/api/v1/data/lineage/graph", LineageGraph)
        self.dark_data = DarkDataResource(http, async_http, "/api/v1/data/dark-data", DarkDataAsset)
        self.analytics = AnalyticsResource(http, async_http, "/api/v1/data/analytics", AnalyticsResult)


__all__ = ["DataNamespace"]
