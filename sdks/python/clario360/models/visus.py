from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class Dashboard(BaseModel):
    id: str
    name: str
    description: str | None = None
    created_at: str | None = None


class Widget(BaseModel):
    id: str
    title: str | None = None
    widget_type: str | None = None
    config: Dict[str, Any] = Field(default_factory=dict)


class KPI(BaseModel):
    id: str
    name: str
    value: float | int | str | None = None
    unit: str | None = None
    updated_at: str | None = None


class Report(BaseModel):
    id: str
    name: str
    status: str | None = None
    created_at: str | None = None


class ExecutiveSummary(BaseModel):
    summary: Dict[str, Any] = Field(default_factory=dict)


class VisusAlert(BaseModel):
    id: str
    title: str
    status: str | None = None
    severity: str | None = None
