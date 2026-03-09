from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.ai import LifecycleEvent, Prediction, RegisteredModel
from clario360.resources.ai.lifecycle import LifecycleResource
from clario360.resources.ai.models import ModelsResource
from clario360.resources.ai.predictions import PredictionsResource


class AINamespace:
    def __init__(self, http: HTTPClient, async_http: AsyncHTTPClient) -> None:
        self.models = ModelsResource(http, async_http, "/api/v1/ai/models", RegisteredModel)
        self.predictions = PredictionsResource(http, async_http, "/api/v1/ai/predictions", Prediction)
        self.lifecycle = LifecycleResource(http, async_http, "/api/v1/ai/models", LifecycleEvent)


__all__ = ["AINamespace"]
