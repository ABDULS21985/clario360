from __future__ import annotations

from clario360.models.ai import DriftAlert, LifecycleEvent, ShadowComparison
from clario360.resources._base import BaseResource


class LifecycleResource(BaseResource[LifecycleEvent]):
    def promote(self, model_id: str, version_id: str) -> LifecycleEvent:
        return self._post_at(f"/api/v1/ai/models/{model_id}/versions/{version_id}/promote", LifecycleEvent, {})

    def rollback(self, model_id: str, payload: dict[str, object]) -> LifecycleEvent:
        return self._post_at(f"/api/v1/ai/models/{model_id}/rollback", LifecycleEvent, payload)

    def shadow_start(self, model_id: str, payload: dict[str, object]) -> LifecycleEvent:
        return self._post_at(f"/api/v1/ai/models/{model_id}/shadow/start", LifecycleEvent, payload)

    def shadow_comparison(self, model_id: str) -> ShadowComparison:
        return self._get_at(f"/api/v1/ai/models/{model_id}/shadow/comparison", ShadowComparison)

    def drift(self, model_id: str) -> DriftAlert:
        return self._get_at(f"/api/v1/ai/models/{model_id}/drift", DriftAlert)
