from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class SourceType(BaseModel):
    type: str
    label: Optional[str] = None
    description: Optional[str] = None


class DataSource(BaseModel):
    id: str
    name: str
    type: str
    status: Optional[str] = None
    description: Optional[str] = None
    tags: List[str] = Field(default_factory=list)
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class DataModel(BaseModel):
    id: str
    name: str
    status: Optional[str] = None
    schema_definition: Dict[str, Any] = Field(default_factory=dict, alias="schema")


class Pipeline(BaseModel):
    id: str
    name: str
    status: Optional[str] = None
    description: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class PipelineRun(BaseModel):
    id: str
    pipeline_id: Optional[str] = None
    status: Optional[str] = None
    started_at: Optional[str] = None
    finished_at: Optional[str] = None
    logs: List[str] = Field(default_factory=list)


class AnalyticsResult(BaseModel):
    rows: List[Dict[str, Any]] = Field(default_factory=list)
    columns: List[str] = Field(default_factory=list)
    metadata: Dict[str, Any] = Field(default_factory=dict)
