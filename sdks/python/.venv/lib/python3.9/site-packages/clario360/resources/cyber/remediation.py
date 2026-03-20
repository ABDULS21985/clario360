from __future__ import annotations

from typing import List

from clario360.models.common import MetricsSnapshot, PaginatedResponse
from clario360.models.remediation import AuditTrailEntry, DryRunResult, ExecutionResult, RemediationAction
from clario360.resources._base import BaseResource


class RemediationResource(BaseResource[RemediationAction]):
    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[RemediationAction]:
        return self._list(params={"page": page, "per_page": per_page})

    def create(self, payload: dict[str, object]) -> RemediationAction:
        return self._create(payload)

    def get(self, remediation_id: str) -> RemediationAction:
        return self._get(remediation_id)

    def approve(self, remediation_id: str) -> RemediationAction:
        return self._post_at(f"{self._base}/{remediation_id}/approve", RemediationAction, {})

    def dry_run(self, remediation_id: str) -> DryRunResult:
        return self._post_at(f"{self._base}/{remediation_id}/dry-run", DryRunResult, {})

    def execute(self, remediation_id: str) -> ExecutionResult:
        return self._post_at(f"{self._base}/{remediation_id}/execute", ExecutionResult, {})

    def verify(self, remediation_id: str) -> ExecutionResult:
        return self._post_at(f"{self._base}/{remediation_id}/verify", ExecutionResult, {})

    def rollback(self, remediation_id: str) -> ExecutionResult:
        return self._post_at(f"{self._base}/{remediation_id}/rollback", ExecutionResult, {})

    def audit_trail(self, remediation_id: str) -> List[AuditTrailEntry]:
        return self._list_models_at(f"{self._base}/{remediation_id}/audit-trail", AuditTrailEntry)

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")
