from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.ctem import Assessment
from clario360.models.cyber import Alert, Asset, DashboardSnapshot, MITRETechnique, Rule, Threat
from clario360.models.dspm import DataAsset
from clario360.models.remediation import RemediationAction
from clario360.models.risk import RiskScore
from clario360.models.ueba import BehavioralProfile
from clario360.models.vciso import ExecutiveBriefing
from clario360.resources.cyber.alerts import AlertsResource
from clario360.resources.cyber.assets import AssetsResource
from clario360.resources.cyber.ctem import CTEMResource
from clario360.resources.cyber.dashboard import DashboardResource
from clario360.resources.cyber.dspm import DSPMResource
from clario360.resources.cyber.mitre import MITREResource
from clario360.resources.cyber.remediation import RemediationResource
from clario360.resources.cyber.risk import RiskResource
from clario360.resources.cyber.rules import RulesResource
from clario360.resources.cyber.threats import ThreatsResource
from clario360.resources.cyber.ueba import UEBAResource
from clario360.resources.cyber.vciso import VCISOResource
from clario360.resources._base import BaseResource


class CyberNamespace:
    def __init__(self, http: HTTPClient, async_http: AsyncHTTPClient) -> None:
        self.assets = AssetsResource(http, async_http, "/api/v1/cyber/assets", Asset)
        self.alerts = AlertsResource(http, async_http, "/api/v1/cyber/alerts", Alert)
        self.threats = ThreatsResource(http, async_http, "/api/v1/cyber/threats", Threat)
        self.rules = RulesResource(http, async_http, "/api/v1/cyber/rules", Rule)
        self.ctem = CTEMResource(http, async_http, "/api/v1/cyber/ctem/assessments", Assessment)
        self.risk = RiskResource(http, async_http, "/api/v1/cyber/risk", RiskScore)
        self.remediation = RemediationResource(http, async_http, "/api/v1/cyber/remediation", RemediationAction)
        self.dspm = DSPMResource(http, async_http, "/api/v1/cyber/dspm/data-assets", DataAsset)
        self.ueba = UEBAResource(http, async_http, "/api/v1/cyber/ueba/profiles", BehavioralProfile)
        self.vciso = VCISOResource(http, async_http, "/api/v1/cyber/vciso", ExecutiveBriefing)
        self.mitre = MITREResource(http, async_http, "/api/v1/cyber/mitre/techniques", MITRETechnique)
        self.dashboard = DashboardResource(http, async_http, "/api/v1/cyber/dashboard", DashboardSnapshot)


__all__ = ["CyberNamespace"]
