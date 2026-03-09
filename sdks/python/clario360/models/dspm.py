from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class DataAsset(BaseModel):
    id: str
    name: str
    type: str | None = None
    classification: str | None = None
    status: str | None = None
    owner: str | None = None


class DSPMScan(BaseModel):
    id: str
    status: str
    created_at: str | None = None


class Classification(BaseModel):
    labels: List[str] = Field(default_factory=list)
    counts: Dict[str, int] = Field(default_factory=dict)


class PostureFinding(BaseModel):
    id: str | None = None
    title: str | None = None
    severity: str | None = None
    status: str | None = None


class DSPMDashboard(BaseModel):
    summary: Dict[str, Any] = Field(default_factory=dict)
