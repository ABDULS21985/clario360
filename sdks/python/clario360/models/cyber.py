from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class Asset(BaseModel):
    id: str
    tenant_id: str | None = None
    name: str
    type: str
    criticality: str | None = None
    status: str | None = None
    ip_address: str | None = None
    hostname: str | None = None
    os: str | None = None
    os_version: str | None = None
    tags: List[str] = Field(default_factory=list)
    created_at: str | None = None
    updated_at: str | None = None


class AssetScan(BaseModel):
    id: str
    status: str
    created_at: str | None = None
    updated_at: str | None = None


class Vulnerability(BaseModel):
    id: str
    title: str | None = None
    severity: str | None = None
    status: str | None = None
    cve: str | None = None
    cvss_score: float | None = None
    discovered_at: str | None = None


class AlertExplanation(BaseModel):
    summary: str = ""
    confidence_score: float | None = None
    confidence_factors: List[Dict[str, Any]] = Field(default_factory=list)
    recommended_actions: List[str] = Field(default_factory=list)
    false_positive_signals: List[str] = Field(default_factory=list)
    evidence_items: List[Dict[str, Any]] = Field(default_factory=list)


class Alert(BaseModel):
    id: str
    tenant_id: str | None = None
    title: str
    description: str | None = None
    severity: str
    status: str
    source: str | None = None
    confidence_score: float = 0.0
    event_count: int = 0
    assigned_to: str | None = None
    created_at: str
    updated_at: str | None = None
    explanation: AlertExplanation | None = None
    affected_assets: List[Asset] = Field(default_factory=list)


class AlertComment(BaseModel):
    id: str
    body: str | None = None
    author_id: str | None = None
    created_at: str | None = None


class AlertTimelineEntry(BaseModel):
    id: str | None = None
    type: str | None = None
    title: str | None = None
    description: str | None = None
    created_at: str | None = None


class Rule(BaseModel):
    id: str
    name: str
    description: str | None = None
    enabled: bool | None = None
    severity: str | None = None
    created_at: str | None = None
    updated_at: str | None = None


class Threat(BaseModel):
    id: str
    name: str
    severity: str | None = None
    status: str | None = None
    description: str | None = None
    created_at: str | None = None


class IndicatorCheckResult(BaseModel):
    matches: List[Dict[str, Any]] = Field(default_factory=list)
    verdict: str | None = None


class MITRETactic(BaseModel):
    id: str
    name: str
    description: str | None = None


class MITRETechnique(BaseModel):
    id: str
    name: str
    tactic: str | None = None
    description: str | None = None


class MITRECoverage(BaseModel):
    covered: int | None = None
    total: int | None = None
    coverage_percent: float | None = None


class DashboardSnapshot(BaseModel):
    kpis: Dict[str, Any] = Field(default_factory=dict)
    alerts_timeline: List[Dict[str, Any]] = Field(default_factory=list)
    severity_distribution: List[Dict[str, Any]] = Field(default_factory=list)
