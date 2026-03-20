from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class Asset(BaseModel):
    id: str
    tenant_id: Optional[str] = None
    name: str
    type: str
    criticality: Optional[str] = None
    status: Optional[str] = None
    ip_address: Optional[str] = None
    hostname: Optional[str] = None
    os: Optional[str] = None
    os_version: Optional[str] = None
    tags: List[str] = Field(default_factory=list)
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class AssetScan(BaseModel):
    id: str
    status: str
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class Vulnerability(BaseModel):
    id: str
    title: Optional[str] = None
    severity: Optional[str] = None
    status: Optional[str] = None
    cve: Optional[str] = None
    cvss_score: Optional[float] = None
    discovered_at: Optional[str] = None


class AlertExplanation(BaseModel):
    summary: str = ""
    confidence_score: Optional[float] = None
    confidence_factors: List[Dict[str, Any]] = Field(default_factory=list)
    recommended_actions: List[str] = Field(default_factory=list)
    false_positive_signals: List[str] = Field(default_factory=list)
    evidence_items: List[Dict[str, Any]] = Field(default_factory=list)


class Alert(BaseModel):
    id: str
    tenant_id: Optional[str] = None
    title: str
    description: Optional[str] = None
    severity: str
    status: str
    source: Optional[str] = None
    confidence_score: float = 0.0
    event_count: int = 0
    assigned_to: Optional[str] = None
    created_at: str
    updated_at: Optional[str] = None
    explanation: Optional[AlertExplanation] = None
    affected_assets: List[Asset] = Field(default_factory=list)


class AlertComment(BaseModel):
    id: str
    body: Optional[str] = None
    author_id: Optional[str] = None
    created_at: Optional[str] = None


class AlertTimelineEntry(BaseModel):
    id: Optional[str] = None
    type: Optional[str] = None
    title: Optional[str] = None
    description: Optional[str] = None
    created_at: Optional[str] = None


class Rule(BaseModel):
    id: str
    name: str
    description: Optional[str] = None
    enabled: Optional[bool] = None
    severity: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class Threat(BaseModel):
    id: str
    name: str
    severity: Optional[str] = None
    status: Optional[str] = None
    description: Optional[str] = None
    created_at: Optional[str] = None


class IndicatorCheckResult(BaseModel):
    matches: List[Dict[str, Any]] = Field(default_factory=list)
    verdict: Optional[str] = None


class MITRETactic(BaseModel):
    id: str
    name: str
    description: Optional[str] = None


class MITRETechnique(BaseModel):
    id: str
    name: str
    tactic: Optional[str] = None
    description: Optional[str] = None


class MITRECoverage(BaseModel):
    covered: Optional[int] = None
    total: Optional[int] = None
    coverage_percent: Optional[float] = None


class DashboardSnapshot(BaseModel):
    kpis: Dict[str, Any] = Field(default_factory=dict)
    alerts_timeline: List[Dict[str, Any]] = Field(default_factory=list)
    severity_distribution: List[Dict[str, Any]] = Field(default_factory=list)
