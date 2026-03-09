from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class RegisteredModel(BaseModel):
    id: str
    name: str
    description: str | None = None
    status: str | None = None
    created_at: str | None = None


class ModelVersion(BaseModel):
    id: str
    version: str | None = None
    stage: str | None = None
    created_at: str | None = None


class Prediction(BaseModel):
    id: str
    model_id: str | None = None
    created_at: str | None = None
    input: Dict[str, Any] = Field(default_factory=dict)
    output: Dict[str, Any] = Field(default_factory=dict)


class Explanation(BaseModel):
    prediction_id: str | None = None
    summary: str | None = None
    factors: List[Dict[str, Any]] = Field(default_factory=list)


class ShadowComparison(BaseModel):
    id: str | None = None
    status: str | None = None
    metrics: Dict[str, Any] = Field(default_factory=dict)


class DriftAlert(BaseModel):
    id: str | None = None
    status: str | None = None
    metrics: Dict[str, Any] = Field(default_factory=dict)


class LifecycleEvent(BaseModel):
    id: str | None = None
    action: str | None = None
    created_at: str | None = None


class AIDashboard(BaseModel):
    summary: Dict[str, Any] = Field(default_factory=dict)
