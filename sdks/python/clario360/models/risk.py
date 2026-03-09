from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class RiskScore(BaseModel):
    overall_score: float = 0.0
    grade: str | None = None
    severity: str | None = None
    last_calculated_at: str | None = None


class RiskTrendPoint(BaseModel):
    recorded_at: str
    overall_score: float


class HeatmapCell(BaseModel):
    likelihood: str | None = None
    impact: str | None = None
    count: int = 0
    details: Dict[str, Any] = Field(default_factory=dict)


class Heatmap(BaseModel):
    cells: List[HeatmapCell] = Field(default_factory=list)

    @classmethod
    def from_payload(cls, payload: Dict[str, Any]) -> "Heatmap":
        raw_cells = payload.get("cells")
        if isinstance(raw_cells, list):
            cells = [HeatmapCell.from_dict(item) for item in raw_cells if isinstance(item, dict)]
            return cls(cells=cells)
        flat_cells = [HeatmapCell.from_dict(item) for item in payload.get("data", []) if isinstance(item, dict)]
        return cls(cells=flat_cells)


class Recommendation(BaseModel):
    id: str | None = None
    title: str
    priority: str | None = None
    summary: str | None = None
