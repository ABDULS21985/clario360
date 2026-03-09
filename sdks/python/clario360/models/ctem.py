from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class Assessment(BaseModel):
    id: str
    name: str
    status: str | None = None
    scope: Dict[str, Any] = Field(default_factory=dict)
    created_at: str | None = None
    updated_at: str | None = None


class Finding(BaseModel):
    id: str
    title: str
    severity: str | None = None
    status: str | None = None
    recommendation: str | None = None


class ExposureScore(BaseModel):
    overall_score: float = 0.0
    trend_direction: str | None = None
    updated_at: str | None = None


class RemediationGroup(BaseModel):
    id: str
    status: str | None = None
    title: str | None = None
    findings: List[Finding] = Field(default_factory=list)
