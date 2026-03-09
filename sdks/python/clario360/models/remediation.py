from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class DryRunResult(BaseModel):
    id: str | None = None
    status: str | None = None
    changes: List[Dict[str, Any]] = Field(default_factory=list)
    warnings: List[str] = Field(default_factory=list)


class ExecutionResult(BaseModel):
    id: str | None = None
    status: str | None = None
    started_at: str | None = None
    finished_at: str | None = None
    output: Dict[str, Any] = Field(default_factory=dict)


class AuditTrailEntry(BaseModel):
    id: str | None = None
    actor: str | None = None
    action: str | None = None
    created_at: str | None = None


class RemediationAction(BaseModel):
    id: str
    title: str
    description: str | None = None
    status: str | None = None
    assigned_to: str | None = None
    created_at: str | None = None
    updated_at: str | None = None
