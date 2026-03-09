from __future__ import annotations

from clario360.models.lex import ComplianceAlert, ComplianceRule, LexDashboard
from clario360.resources._base import BaseResource


class ComplianceResource(BaseResource[ComplianceRule]):
    def list_rules(self) -> list[ComplianceRule]:
        return self._list_models_at("/api/v1/lex/compliance/rules", ComplianceRule)

    def create_rule(self, payload: dict[str, object]) -> ComplianceRule:
        return self._post_at("/api/v1/lex/compliance/rules", ComplianceRule, payload)

    def alerts(self) -> list[ComplianceAlert]:
        return self._list_models_at("/api/v1/lex/compliance/alerts", ComplianceAlert)

    def dashboard(self) -> LexDashboard:
        return self._get_at("/api/v1/lex/compliance/dashboard", LexDashboard)
