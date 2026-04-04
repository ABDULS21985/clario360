from __future__ import annotations

from typing import Optional


class ModelResource:
    def __init__(self, client):
        self.client = client

    def list(self, **params):
        return self.client.list("/api/v1/ai/models", params=params)

    def get(self, model_id: str):
        return self.client.get(f"/api/v1/ai/models/{model_id}")

    def get_by_name(self, name_or_slug: str):
        models = self.list(per_page=100).auto_paginate()
        for item in models:
            if item.get("slug") == name_or_slug or item.get("name") == name_or_slug:
                return item
            model = item.get("model")
            if model and (model.get("slug") == name_or_slug or model.get("name") == name_or_slug):
                return model
        raise LookupError(f"model {name_or_slug} not found")

    def create_version(self, model_id: str, config: dict, metrics: Optional[dict] = None, lifecycle_stage: str = "development"):
        payload = {
            "artifact_config": config,
            "training_metrics": metrics or {},
            "status": lifecycle_stage,
            "artifact_type": "rule_set",
            "explainability_type": "rule_trace",
            "description": "Notebook-created version",
        }
        return self.client.post(f"/api/v1/ai/models/{model_id}/versions", json=payload)
