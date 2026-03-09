from __future__ import annotations

from typing import Any, Dict, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class BehavioralProfile(BaseModel):
    id: Optional[str] = None
    entity_id: Optional[str] = None
    entity_name: Optional[str] = None
    status: Optional[str] = None
    risk_score: Optional[float] = None
    attributes: Dict[str, Any] = Field(default_factory=dict)


class UEBAAlert(BaseModel):
    id: str
    title: str
    severity: Optional[str] = None
    status: Optional[str] = None
    risk_score: Optional[float] = None
    created_at: Optional[str] = None


class UEBATimelineEntry(BaseModel):
    id: Optional[str] = None
    type: Optional[str] = None
    description: Optional[str] = None
    created_at: Optional[str] = None


class UEBADashboard(BaseModel):
    summary: Dict[str, Any] = Field(default_factory=dict)


class UEBARiskRankingEntry(BaseModel):
    entity_id: str
    entity_name: Optional[str] = None
    score: Optional[float] = None


class UEBAConfig(BaseModel):
    enabled: Optional[bool] = None
    settings: Dict[str, Any] = Field(default_factory=dict)
