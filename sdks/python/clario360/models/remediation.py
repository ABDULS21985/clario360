from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class DryRunResult(BaseModel):
    id: Optional[str] = None
    status: Optional[str] = None
    changes: List[Dict[str, Any]] = Field(default_factory=list)
    warnings: List[str] = Field(default_factory=list)


class ExecutionResult(BaseModel):
    id: Optional[str] = None
    status: Optional[str] = None
    started_at: Optional[str] = None
    finished_at: Optional[str] = None
    output: Dict[str, Any] = Field(default_factory=dict)


class AuditTrailEntry(BaseModel):
    id: Optional[str] = None
    actor: Optional[str] = None
    action: Optional[str] = None
    created_at: Optional[str] = None


class RemediationAction(BaseModel):
    id: str
    title: str
    description: Optional[str] = None
    status: Optional[str] = None
    assigned_to: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None
