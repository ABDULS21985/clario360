from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class RegisteredModel(BaseModel):
    id: str
    name: str
    description: Optional[str] = None
    status: Optional[str] = None
    created_at: Optional[str] = None


class ModelVersion(BaseModel):
    id: str
    version: Optional[str] = None
    stage: Optional[str] = None
    created_at: Optional[str] = None


class Prediction(BaseModel):
    id: str
    model_id: Optional[str] = None
    created_at: Optional[str] = None
    input: Dict[str, Any] = Field(default_factory=dict)
    output: Dict[str, Any] = Field(default_factory=dict)


class Explanation(BaseModel):
    prediction_id: Optional[str] = None
    summary: Optional[str] = None
    factors: List[Dict[str, Any]] = Field(default_factory=list)


class ShadowComparison(BaseModel):
    id: Optional[str] = None
    status: Optional[str] = None
    metrics: Dict[str, Any] = Field(default_factory=dict)


class DriftAlert(BaseModel):
    id: Optional[str] = None
    status: Optional[str] = None
    metrics: Dict[str, Any] = Field(default_factory=dict)


class LifecycleEvent(BaseModel):
    id: Optional[str] = None
    action: Optional[str] = None
    created_at: Optional[str] = None


class AIDashboard(BaseModel):
    summary: Dict[str, Any] = Field(default_factory=dict)
