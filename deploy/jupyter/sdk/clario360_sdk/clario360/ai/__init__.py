from .models import ModelResource
from .predictions import PredictionResource
from .lifecycle import LifecycleResource


class AINamespace:
    def __init__(self, client):
        self.models = ModelResource(client)
        self.predictions = PredictionResource(client)
        self.lifecycle = LifecycleResource(client)
