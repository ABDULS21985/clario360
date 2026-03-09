from __future__ import annotations

from typing import Any, Dict, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class ExecutiveBriefing(BaseModel):
    id: Optional[str] = None
    type: Optional[str] = None
    title: Optional[str] = None
    generated_at: Optional[str] = None
    summary: Optional[str] = None
    data: Dict[str, Any] = Field(default_factory=dict)


class BriefingHistoryEntry(BaseModel):
    id: str
    type: Optional[str] = None
    generated_at: Optional[str] = None
    title: Optional[str] = None


class SecurityRecommendation(BaseModel):
    id: Optional[str] = None
    title: str
    priority: Optional[str] = None
    summary: Optional[str] = None


class VCISOReport(BaseModel):
    id: Optional[str] = None
    status: Optional[str] = None
    report_url: Optional[str] = None
    created_at: Optional[str] = None


class PostureSummary(BaseModel):
    overall_score: Optional[float] = None
    summary: Optional[str] = None
    data: Dict[str, Any] = Field(default_factory=dict)


class ConversationSummary(BaseModel):
    id: str
    title: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None
