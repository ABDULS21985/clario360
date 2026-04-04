from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.common import PaginatedResponse
from clario360.models.ctem import Assessment, ExposureScore, Finding, RemediationGroup
from clario360.resources._base import BaseResource


class CTEMResource(BaseResource[Assessment]):
    def __init__(
        self,
        http: HTTPClient,
        async_http: AsyncHTTPClient,
        base_path: str,
        model_class: type[Assessment],
    ) -> None:
        super().__init__(http, async_http, base_path, model_class)
        self._base = "/api/v1/cyber/ctem/assessments"

    def list(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[Assessment]:
        return self._list(params={"page": page, "per_page": per_page})

    def create(self, payload: dict[str, object]) -> Assessment:
        return self._create(payload)

    def get(self, assessment_id: str) -> Assessment:
        return self._get(assessment_id)

    def exposure_score(self) -> ExposureScore:
        return self._get_at("/api/v1/cyber/ctem/exposure-score", ExposureScore)

    def findings(self, assessment_id: str, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[Finding]:
        payload = self._http.get(f"{self._base}/{assessment_id}/findings", params={"page": page, "per_page": per_page})
        return PaginatedResponse.from_payload(self._ensure_mapping(payload), Finding)

    def remediation_groups(self, assessment_id: str, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[RemediationGroup]:
        payload = self._http.get(f"{self._base}/{assessment_id}/remediation-groups", params={"page": page, "per_page": per_page})
        return PaginatedResponse.from_payload(self._ensure_mapping(payload), RemediationGroup)
