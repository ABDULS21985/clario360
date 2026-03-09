from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class BehavioralProfile(BaseModel):
    id: str | None = None
    entity_id: str | None = None
    entity_name: str | None = None
    status: str | None = None
    risk_score: float | None = None
    attributes: Dict[str, Any] = Field(default_factory=dict)


class UEBAAlert(BaseModel):
    id: str
    title: str
    severity: str | None = None
    status: str | None = None
    risk_score: float | None = None
    created_at: str | None = None


class UEBATimelineEntry(BaseModel):
    id: str | None = None
    type: str | None = None
    description: str | None = None
    created_at: str | None = None


class UEBADashboard(BaseModel):
    summary: Dict[str, Any] = Field(default_factory=dict)


class UEBARiskRankingEntry(BaseModel):
    entity_id: str
    entity_name: str | None = None
    score: float | None = None


class UEBAConfig(BaseModel):
    enabled: bool | None = None
    settings: Dict[str, Any] = Field(default_factory=dict)
