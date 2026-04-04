from __future__ import annotations

from clario360.models.ai import Prediction
from clario360.models.common import PaginatedResponse
from clario360.resources._base import BaseResource


class PredictionsResource(BaseResource[Prediction]):
    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[Prediction]:
        return self._list(params={"page": page, "per_page": per_page})

    def get(self, prediction_id: str) -> Prediction:
        return self._get(prediction_id)
