from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Any, Iterable, Iterator, Optional
from urllib.parse import urljoin

import requests

from .auth import TokenManager
from .utils.dataframe import records_to_dataframe


class APIError(RuntimeError):
    def __init__(self, status_code: int, message: str, payload: Optional[dict[str, Any]] = None):
        super().__init__(message)
        self.status_code = status_code
        self.payload = payload or {}


class AttrDict(dict):
    def __getattr__(self, item: str) -> Any:
        try:
            return self[item]
        except KeyError as exc:
            raise AttributeError(item) from exc


def wrap(value: Any) -> Any:
    if isinstance(value, dict):
        return AttrDict({key: wrap(val) for key, val in value.items()})
    if isinstance(value, list):
        return [wrap(item) for item in value]
    return value


@dataclass
class PaginationMeta:
    page: int = 1
    per_page: int = 0
    total: int = 0
    last_page: int = 1


class ListResult(Iterable[Any]):
    def __init__(self, client: "Client", path: str, params: Optional[dict[str, Any]], items: list[Any], meta: PaginationMeta):
        self._client = client
        self._path = path
        self._params = params or {}
        self.data = items
        self.meta = meta

    def __iter__(self) -> Iterator[Any]:
        return iter(self.data)

    def to_dataframe(self):
        return records_to_dataframe(self.data)

    def auto_paginate(self) -> Iterator[Any]:
        current_page = self.meta.page or 1
        last_page = self.meta.last_page or current_page
        for item in self.data:
            yield item
        while current_page < last_page:
            current_page += 1
            params = dict(self._params)
            params["page"] = current_page
            next_page = self._client.list(self._path, params=params)
            for item in next_page.data:
                yield item


class Client:
    def __init__(self, api_url: str, token: str, refresh_token: Optional[str] = None, timeout: int = 30):
        self.api_url = api_url.rstrip("/")
        self.timeout = timeout
        self.session = requests.Session()
        self.tokens = TokenManager(self.api_url, token, refresh_token)
        self._whoami_cache: Optional[AttrDict] = None

        from .cyber import CyberNamespace
        from .data import DataNamespace
        from .ai import AINamespace

        self.cyber = CyberNamespace(self)
        self.data = DataNamespace(self)
        self.ai = AINamespace(self)

    @classmethod
    def from_env(cls) -> "Client":
        api_url = os.environ["CLARIO360_API_URL"]
        access_token = os.environ["CLARIO360_ACCESS_TOKEN"]
        refresh_token = os.environ.get("CLARIO360_REFRESH_TOKEN")
        return cls(api_url=api_url, token=access_token, refresh_token=refresh_token)

    @property
    def tenant_id(self) -> str:
        return str(self.whoami()["tenant_id"])

    def whoami(self) -> AttrDict:
        if self._whoami_cache is None:
            self._whoami_cache = self.get("/api/v1/users/me")
        return self._whoami_cache

    def get(self, path: str, params: Optional[dict[str, Any]] = None) -> AttrDict:
        payload = self._request("GET", path, params=params)
        return self._unwrap(payload)

    def post(self, path: str, json: Optional[dict[str, Any]] = None) -> AttrDict:
        payload = self._request("POST", path, json=json)
        return self._unwrap(payload)

    def put(self, path: str, json: Optional[dict[str, Any]] = None) -> AttrDict:
        payload = self._request("PUT", path, json=json)
        return self._unwrap(payload)

    def patch(self, path: str, json: Optional[dict[str, Any]] = None) -> AttrDict:
        payload = self._request("PATCH", path, json=json)
        return self._unwrap(payload)

    def delete(self, path: str) -> AttrDict:
        payload = self._request("DELETE", path)
        return self._unwrap(payload)

    def list(self, path: str, params: Optional[dict[str, Any]] = None) -> ListResult:
        payload = self._request("GET", path, params=params)
        items, meta = self._unwrap_list(payload)
        return ListResult(self, path, params, items, meta)

    def record_data_query(self, source: str, description: str, metadata: Optional[dict[str, Any]] = None) -> None:
        self._record_activity(
            {
                "kind": "data_query",
                "source": source,
                "description": description,
                "metadata": metadata or {},
            }
        )

    def record_spark_job(self, status: str, description: str, metadata: Optional[dict[str, Any]] = None) -> None:
        self._record_activity(
            {
                "kind": "spark_job",
                "status": status,
                "description": description,
                "metadata": metadata or {},
            }
        )

    def _request(
        self,
        method: str,
        path: str,
        *,
        params: Optional[dict[str, Any]] = None,
        json: Optional[dict[str, Any]] = None,
        allow_refresh: bool = True,
        suppress_activity: bool = False,
    ) -> dict[str, Any]:
        url = urljoin(f"{self.api_url}/", path.lstrip("/"))
        headers = self.tokens.apply({"Accept": "application/json"})
        response = self.session.request(method, url, params=params, json=json, headers=headers, timeout=self.timeout)

        if response.status_code == 401 and allow_refresh and self.tokens.refresh_token:
            self.tokens.refresh(self.session)
            return self._request(method, path, params=params, json=json, allow_refresh=False, suppress_activity=suppress_activity)

        if not suppress_activity and path != "/api/v1/notebooks/activity":
            self._record_activity({"kind": "sdk_api", "endpoint": path, "status": str(response.status_code)})

        if response.status_code >= 400:
            try:
                payload = response.json()
            except ValueError:
                payload = {"error": response.text}
            raise APIError(response.status_code, payload.get("message") or payload.get("error") or "request failed", payload)

        if response.content:
            return response.json()
        return {}

    def _unwrap(self, payload: Any) -> AttrDict:
        if isinstance(payload, dict) and "data" in payload and isinstance(payload["data"], dict):
            return wrap(payload["data"])
        if isinstance(payload, dict):
            return wrap(payload)
        raise APIError(500, "unexpected response payload", {"payload": payload})

    def _unwrap_list(self, payload: Any) -> tuple[list[Any], PaginationMeta]:
        if isinstance(payload, dict) and "data" in payload and isinstance(payload["data"], list):
            pagination = payload.get("pagination") or payload.get("meta") or {}
            meta = PaginationMeta(
                page=int(pagination.get("page", 1) or 1),
                per_page=int(pagination.get("per_page", 0) or 0),
                total=int(pagination.get("total", len(payload["data"])) or len(payload["data"])),
                last_page=int(pagination.get("last_page", 1) or 1),
            )
            return wrap(payload["data"]), meta
        if isinstance(payload, list):
            return wrap(payload), PaginationMeta(total=len(payload), last_page=1)
        if isinstance(payload, dict):
            maybe_data = payload.get("data")
            if isinstance(maybe_data, list):
                return wrap(maybe_data), PaginationMeta(total=len(maybe_data), last_page=1)
        raise APIError(500, "unexpected list payload", {"payload": payload})

    def _record_activity(self, payload: dict[str, Any]) -> None:
        try:
            self._request("POST", "/api/v1/notebooks/activity", json=payload, suppress_activity=True, allow_refresh=False)
        except Exception:
            return
