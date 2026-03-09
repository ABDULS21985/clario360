from __future__ import annotations

from clario360.models.ai import ModelVersion, RegisteredModel
from clario360.models.common import PaginatedResponse
from clario360.resources._base import BaseResource


class ModelsResource(BaseResource[RegisteredModel]):
    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[RegisteredModel]:
        return self._list(params={"page": page, "per_page": per_page})

    def get(self, model_id: str) -> RegisteredModel:
        return self._get(model_id)

    def create(self, payload: dict[str, object]) -> RegisteredModel:
        return self._create(payload)

    def create_version(self, model_id: str, payload: dict[str, object]) -> ModelVersion:
        return self._post_at(f"{self._base}/{model_id}/versions", ModelVersion, payload)

    def versions(self, model_id: str) -> list[ModelVersion]:
        return self._list_models_at(f"{self._base}/{model_id}/versions", ModelVersion)
