from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class ExecutiveBriefing(BaseModel):
    id: str | None = None
    type: str | None = None
    title: str | None = None
    generated_at: str | None = None
    summary: str | None = None
    data: Dict[str, Any] = Field(default_factory=dict)


class BriefingHistoryEntry(BaseModel):
    id: str
    type: str | None = None
    generated_at: str | None = None
    title: str | None = None


class SecurityRecommendation(BaseModel):
    id: str | None = None
    title: str
    priority: str | None = None
    summary: str | None = None


class VCISOReport(BaseModel):
    id: str | None = None
    status: str | None = None
    report_url: str | None = None
    created_at: str | None = None


class PostureSummary(BaseModel):
    overall_score: float | None = None
    summary: str | None = None
    data: Dict[str, Any] = Field(default_factory=dict)


class ConversationSummary(BaseModel):
    id: str
    title: str | None = None
    created_at: str | None = None
    updated_at: str | None = None
