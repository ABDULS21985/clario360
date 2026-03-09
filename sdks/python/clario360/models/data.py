from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class SourceType(BaseModel):
    type: str
    label: str | None = None
    description: str | None = None


class DataSource(BaseModel):
    id: str
    name: str
    type: str
    status: str | None = None
    description: str | None = None
    tags: List[str] = Field(default_factory=list)
    created_at: str | None = None
    updated_at: str | None = None


class DataModel(BaseModel):
    id: str
    name: str
    status: str | None = None
    schema: Dict[str, Any] = Field(default_factory=dict)


class Pipeline(BaseModel):
    id: str
    name: str
    status: str | None = None
    description: str | None = None
    created_at: str | None = None
    updated_at: str | None = None


class PipelineRun(BaseModel):
    id: str
    pipeline_id: str | None = None
    status: str | None = None
    started_at: str | None = None
    finished_at: str | None = None
    logs: List[str] = Field(default_factory=list)


class AnalyticsResult(BaseModel):
    rows: List[Dict[str, Any]] = Field(default_factory=list)
    columns: List[str] = Field(default_factory=list)
    metadata: Dict[str, Any] = Field(default_factory=dict)
