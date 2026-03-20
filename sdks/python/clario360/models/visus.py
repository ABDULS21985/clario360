from __future__ import annotations

from typing import Any, Dict, Optional, Union

from pydantic import Field

from clario360.models.base import BaseModel


class Dashboard(BaseModel):
    id: str
    name: str
    description: Optional[str] = None
    created_at: Optional[str] = None


class Widget(BaseModel):
    id: str
    title: Optional[str] = None
    widget_type: Optional[str] = None
    config: Dict[str, Any] = Field(default_factory=dict)


class KPI(BaseModel):
    id: str
    name: str
    value: Optional[Union[float, int, str]] = None
    unit: Optional[str] = None
    updated_at: Optional[str] = None


class Report(BaseModel):
    id: str
    name: str
    status: Optional[str] = None
    created_at: Optional[str] = None


class ExecutiveSummary(BaseModel):
    summary: Dict[str, Any] = Field(default_factory=dict)


class VisusAlert(BaseModel):
    id: str
    title: str
    status: Optional[str] = None
    severity: Optional[str] = None
