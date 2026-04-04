from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class DataAsset(BaseModel):
    id: str
    name: str
    type: Optional[str] = None
    classification: Optional[str] = None
    status: Optional[str] = None
    owner: Optional[str] = None


class DSPMScan(BaseModel):
    id: str
    status: str
    created_at: Optional[str] = None


class Classification(BaseModel):
    labels: List[str] = Field(default_factory=list)
    counts: Dict[str, int] = Field(default_factory=dict)


class PostureFinding(BaseModel):
    id: Optional[str] = None
    title: Optional[str] = None
    severity: Optional[str] = None
    status: Optional[str] = None


class DSPMDashboard(BaseModel):
    summary: Dict[str, Any] = Field(default_factory=dict)
