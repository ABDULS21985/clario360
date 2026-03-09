from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class QualityScore(BaseModel):
    overall_score: float = 0.0
    grade: str | None = None
    dimensions: Dict[str, Any] = Field(default_factory=dict)


class QualityTrendPoint(BaseModel):
    date: str | None = None
    score: float = 0.0


class QualityRule(BaseModel):
    id: str
    name: str
    rule_type: str | None = None
    status: str | None = None
    created_at: str | None = None


class QualityResult(BaseModel):
    id: str
    rule_id: str | None = None
    status: str | None = None
    score: float | None = None
    created_at: str | None = None


class Contradiction(BaseModel):
    id: str
    title: str | None = None
    status: str | None = None
    severity: str | None = None
    created_at: str | None = None


class ContradictionScan(BaseModel):
    id: str
    status: str | None = None
    created_at: str | None = None


class DarkDataAsset(BaseModel):
    id: str
    name: str
    status: str | None = None
    owner: str | None = None
