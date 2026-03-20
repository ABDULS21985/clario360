from __future__ import annotations

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.common import PaginatedResponse
from clario360.models.dspm import Classification, DSPMDashboard, DSPMScan, DataAsset
from clario360.resources._base import BaseResource


class DSPMResource(BaseResource[DataAsset]):
    def __init__(
        self,
        http: HTTPClient,
        async_http: AsyncHTTPClient,
        base_path: str,
        model_class: type[DataAsset],
    ) -> None:
        super().__init__(http, async_http, base_path, model_class)
        self._base = "/api/v1/cyber/dspm/data-assets"

    def list_data_assets(self, *, page: int = 1, per_page: int = 50) -> PaginatedResponse[DataAsset]:
        return self._list(params={"page": page, "per_page": per_page})

    def get_data_asset(self, asset_id: str) -> DataAsset:
        return self._get(asset_id)

    def scan(self, payload: dict[str, object]) -> DSPMScan:
        return self._post_at("/api/v1/cyber/dspm/scan", DSPMScan, payload)

    def classification(self) -> Classification:
        return self._get_at("/api/v1/cyber/dspm/classification", Classification)

    def dashboard(self) -> DSPMDashboard:
        return self._get_at("/api/v1/cyber/dspm/dashboard", DSPMDashboard)
