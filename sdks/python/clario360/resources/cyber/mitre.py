from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.cyber import MITRECoverage, MITRETactic, MITRETechnique
from clario360.resources._base import BaseResource


class MITREResource(BaseResource[MITRETechnique]):
    def __init__(
        self,
        http: HTTPClient,
        async_http: AsyncHTTPClient,
        base_path: str,
        model_class: type[MITRETechnique],
    ) -> None:
        super().__init__(http, async_http, base_path, model_class)
        self._base = "/api/v1/cyber/mitre/techniques"

    def tactics(self) -> list[MITRETactic]:
        return self._list_models_at("/api/v1/cyber/mitre/tactics", MITRETactic)

    def techniques(self) -> list[MITRETechnique]:
        return self._list_models_at(self._base, MITRETechnique)

    def technique(self, technique_id: str) -> MITRETechnique:
        return self._get(technique_id)

    def coverage(self) -> MITRECoverage:
        return self._get_at("/api/v1/cyber/mitre/coverage", MITRECoverage)
