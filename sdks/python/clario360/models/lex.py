from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class Contract(BaseModel):
    id: str
    title: str
    status: str | None = None
    counterparty: str | None = None
    effective_date: str | None = None
    expiration_date: str | None = None


class Clause(BaseModel):
    id: str
    title: str | None = None
    risk_level: str | None = None
    text: str | None = None


class ComplianceRule(BaseModel):
    id: str
    name: str
    severity: str | None = None
    status: str | None = None


class ComplianceAlert(BaseModel):
    id: str
    title: str
    status: str | None = None
    severity: str | None = None
    created_at: str | None = None


class LegalDocument(BaseModel):
    id: str
    name: str
    status: str | None = None


class LexDashboard(BaseModel):
    summary: Dict[str, Any] = Field(default_factory=dict)
