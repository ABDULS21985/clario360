from __future__ import annotations

from typing import Any, Dict, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class Contract(BaseModel):
    id: str
    title: str
    status: Optional[str] = None
    counterparty: Optional[str] = None
    effective_date: Optional[str] = None
    expiration_date: Optional[str] = None


class Clause(BaseModel):
    id: str
    title: Optional[str] = None
    risk_level: Optional[str] = None
    text: Optional[str] = None


class ComplianceRule(BaseModel):
    id: str
    name: str
    severity: Optional[str] = None
    status: Optional[str] = None


class ComplianceAlert(BaseModel):
    id: str
    title: str
    status: Optional[str] = None
    severity: Optional[str] = None
    created_at: Optional[str] = None


class LegalDocument(BaseModel):
    id: str
    name: str
    status: Optional[str] = None


class LexDashboard(BaseModel):
    summary: Dict[str, Any] = Field(default_factory=dict)
