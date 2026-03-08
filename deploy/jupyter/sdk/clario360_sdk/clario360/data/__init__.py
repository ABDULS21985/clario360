from .sources import SourceResource
from .pipelines import PipelineResource
from .quality import QualityResource
from .analytics import AnalyticsResource


class DataNamespace:
    def __init__(self, client):
        self.sources = SourceResource(client)
        self.pipelines = PipelineResource(client)
        self.quality = QualityResource(client)
        self.analytics = AnalyticsResource(client)
