from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class Assessment(BaseModel):
    id: str
    name: str
    status: Optional[str] = None
    scope: Dict[str, Any] = Field(default_factory=dict)
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class Finding(BaseModel):
    id: str
    title: str
    severity: Optional[str] = None
    status: Optional[str] = None
    recommendation: Optional[str] = None


class ExposureScore(BaseModel):
    overall_score: float = 0.0
    trend_direction: Optional[str] = None
    updated_at: Optional[str] = None


class RemediationGroup(BaseModel):
    id: str
    status: Optional[str] = None
    title: Optional[str] = None
    findings: List[Finding] = Field(default_factory=list)
