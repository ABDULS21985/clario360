from __future__ import annotations

from typing import Any, Dict, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class QualityScore(BaseModel):
    overall_score: float = 0.0
    grade: Optional[str] = None
    dimensions: Dict[str, Any] = Field(default_factory=dict)


class QualityTrendPoint(BaseModel):
    date: Optional[str] = None
    score: float = 0.0


class QualityRule(BaseModel):
    id: str
    name: str
    rule_type: Optional[str] = None
    status: Optional[str] = None
    created_at: Optional[str] = None


class QualityResult(BaseModel):
    id: str
    rule_id: Optional[str] = None
    status: Optional[str] = None
    score: Optional[float] = None
    created_at: Optional[str] = None


class Contradiction(BaseModel):
    id: str
    title: Optional[str] = None
    status: Optional[str] = None
    severity: Optional[str] = None
    created_at: Optional[str] = None


class ContradictionScan(BaseModel):
    id: str
    status: Optional[str] = None
    created_at: Optional[str] = None


class DarkDataAsset(BaseModel):
    id: str
    name: str
    status: Optional[str] = None
    owner: Optional[str] = None
