from __future__ import annotations

from typing import Iterator, Optional

from clario360.models.common import CountSnapshot, MetricsSnapshot, PaginatedResponse
from clario360.models.cyber import Asset, AssetScan, Vulnerability
from clario360.resources._base import BaseResource


class AssetsResource(BaseResource[Asset]):
    def list(
        self,
        *,
        asset_type: Optional[str] = None,
        criticality: Optional[str] = None,
        status: Optional[str] = None,
        search: Optional[str] = None,
        page: int = 1,
        per_page: int = 50,
    ) -> PaginatedResponse[Asset]:
        return self._list(
            params={
                "type": asset_type,
                "criticality": criticality,
                "status": status,
                "search": search,
                "page": page,
                "per_page": per_page,
            }
        )

    def list_all(self, *, asset_type: Optional[str] = None, criticality: Optional[str] = None) -> Iterator[Asset]:
        return self._list_all(params={"type": asset_type, "criticality": criticality})

    def get(self, asset_id: str) -> Asset:
        return self._get(asset_id)

    def create(self, payload: dict[str, object]) -> Asset:
        return self._create(payload)

    def update(self, asset_id: str, payload: dict[str, object]) -> Asset:
        return self._update(asset_id, payload)

    def delete(self, asset_id: str) -> None:
        self._delete(asset_id)

    def bulk_create(self, assets: list[dict[str, object]]) -> list[Asset]:
        payload = self._http.post(f"{self._base}/bulk", json={"assets": assets})
        data = self._unwrap(payload)
        if isinstance(data, list):
            return [Asset.from_dict(item) for item in data if isinstance(item, dict)]
        return []

    def scan(self, payload: dict[str, object]) -> AssetScan:
        return self._post_at(f"{self._base}/scan", AssetScan, payload)

    def list_scans(self) -> list[AssetScan]:
        return self._list_models_at(f"{self._base}/scans", AssetScan)

    def get_scan(self, scan_id: str) -> AssetScan:
        return self._get_at(f"{self._base}/scans/{scan_id}", AssetScan)

    def vulnerabilities(self, asset_id: str) -> list[Vulnerability]:
        return self._list_models_at(f"{self._base}/{asset_id}/vulnerabilities", Vulnerability)

    def stats(self) -> MetricsSnapshot:
        return self._metrics(f"{self._base}/stats")

    def count(self) -> CountSnapshot:
        return self._counts(f"{self._base}/count")
