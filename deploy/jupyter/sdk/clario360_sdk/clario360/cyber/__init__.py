from .assets import AssetResource
from .alerts import AlertResource
from .threats import ThreatResource
from .rules import RuleResource
from .ctem import CTEMResource
from .risk import RiskResource
from .remediation import RemediationResource
from .dspm import DSPMResource
from .vciso import VCISOResource
from .dashboard import DashboardResource


class CyberNamespace:
    def __init__(self, client):
        self.assets = AssetResource(client)
        self.alerts = AlertResource(client)
        self.threats = ThreatResource(client)
        self.rules = RuleResource(client)
        self.ctem = CTEMResource(client)
        self.risk = RiskResource(client)
        self.remediation = RemediationResource(client)
        self.dspm = DSPMResource(client)
        self.vciso = VCISOResource(client)
        self.dashboard = DashboardResource(client)
