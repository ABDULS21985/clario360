from __future__ import annotations

from typing import Any, AsyncIterator, Dict, Generic, Iterator, List, Mapping, Optional, Type, TypeVar, cast

from clario360.http_client import AsyncHTTPClient, HTTPClient
from clario360.models.base import BaseModel
from clario360.models.common import CountSnapshot, MetricsSnapshot, PaginatedResponse

T = TypeVar("T", bound=BaseModel)
R = TypeVar("R", bound=BaseModel)


class BaseResource(Generic[T]):
    def __init__(
        self,
        http: HTTPClient,
        async_http: AsyncHTTPClient,
        base_path: str,
        model_class: Type[T],
    ) -> None:
        self._http = http
        self._async_http = async_http
        self._base = base_path
        self._model = model_class

    def _list(self, *, params: Optional[Mapping[str, Any]] = None) -> PaginatedResponse[T]:
        payload = self._http.get(self._base, params=params)
        return PaginatedResponse.from_payload(self._ensure_mapping(payload), self._model)

    def list_async_params(self, *, params: Optional[Mapping[str, Any]] = None) -> PaginatedResponse[T]:
        """Sync alias for callers that build params dictionaries dynamically."""
        return self._list(params=params)

    async def _alist(self, *, params: Optional[Mapping[str, Any]] = None) -> PaginatedResponse[T]:
        payload = await self._async_http.get(self._base, params=params)
        return PaginatedResponse.from_payload(self._ensure_mapping(payload), self._model)

    async def list_async(self, *, params: Optional[Mapping[str, Any]] = None) -> PaginatedResponse[T]:
        """List resources asynchronously with raw query parameters."""
        return await self._alist(params=params)

    def _list_all(self, *, params: Optional[Mapping[str, Any]] = None) -> Iterator[T]:
        page = 1
        while True:
            merged = dict(params or {})
            merged.setdefault("page", page)
            merged.setdefault("per_page", 200)
            result = self._list(params=merged)
            for item in result.data:
                yield item
            if page >= result.total_pages:
                break
            page += 1

    async def _alist_all(self, *, params: Optional[Mapping[str, Any]] = None) -> AsyncIterator[T]:
        page = 1
        while True:
            merged = dict(params or {})
            merged.setdefault("page", page)
            merged.setdefault("per_page", 200)
            result = await self._alist(params=merged)
            for item in result.data:
                yield item
            if page >= result.total_pages:
                break
            page += 1

    def _get(self, resource_id: str) -> T:
        payload = self._http.get(f"{self._base}/{resource_id}")
        return self._parse_model(payload, self._model)

    async def get_async(self, resource_id: str) -> T:
        """Fetch a resource asynchronously by identifier."""
        return await self._aget(resource_id)

    async def _aget(self, resource_id: str) -> T:
        payload = await self._async_http.get(f"{self._base}/{resource_id}")
        return self._parse_model(payload, self._model)

    def _create(self, data: Mapping[str, Any]) -> T:
        payload = self._http.post(self._base, json=data)
        return self._parse_model(payload, self._model)

    async def create_async(self, data: Mapping[str, Any]) -> T:
        """Create a resource asynchronously."""
        return await self._acreate(data)

    async def _acreate(self, data: Mapping[str, Any]) -> T:
        payload = await self._async_http.post(self._base, json=data)
        return self._parse_model(payload, self._model)

    def _update(self, resource_id: str, data: Mapping[str, Any]) -> T:
        payload = self._http.put(f"{self._base}/{resource_id}", json=data)
        return self._parse_model(payload, self._model)

    async def update_async(self, resource_id: str, data: Mapping[str, Any]) -> T:
        """Update a resource asynchronously."""
        return await self._aupdate(resource_id, data)

    async def _aupdate(self, resource_id: str, data: Mapping[str, Any]) -> T:
        payload = await self._async_http.put(f"{self._base}/{resource_id}", json=data)
        return self._parse_model(payload, self._model)

    def _delete(self, resource_id: str) -> None:
        self._http.delete(f"{self._base}/{resource_id}")

    async def delete_async(self, resource_id: str) -> None:
        """Delete a resource asynchronously."""
        await self._adelete(resource_id)

    async def _adelete(self, resource_id: str) -> None:
        await self._async_http.delete(f"{self._base}/{resource_id}")

    def _get_at(self, path: str, model: Type[R]) -> R:
        payload = self._http.get(path)
        return self._parse_model(payload, model)

    async def _aget_at(self, path: str, model: Type[R]) -> R:
        payload = await self._async_http.get(path)
        return self._parse_model(payload, model)

    def _post_at(self, path: str, model: Type[R], data: Optional[Mapping[str, Any]] = None) -> R:
        payload = self._http.post(path, json=data)
        return self._parse_model(payload, model)

    async def _apost_at(
        self,
        path: str,
        model: Type[R],
        data: Optional[Mapping[str, Any]] = None,
    ) -> R:
        payload = await self._async_http.post(path, json=data)
        return self._parse_model(payload, model)

    def _put_at(self, path: str, model: Type[R], data: Optional[Mapping[str, Any]] = None) -> R:
        payload = self._http.put(path, json=data)
        return self._parse_model(payload, model)

    async def _aput_at(
        self,
        path: str,
        model: Type[R],
        data: Optional[Mapping[str, Any]] = None,
    ) -> R:
        payload = await self._async_http.put(path, json=data)
        return self._parse_model(payload, model)

    def _metrics(self, path: str) -> MetricsSnapshot:
        payload = self._http.get(path)
        return MetricsSnapshot.from_payload(self._unwrap_mapping(payload))

    def _post_metrics_at(self, path: str, data: Optional[Mapping[str, Any]] = None) -> MetricsSnapshot:
        payload = self._http.post(path, json=data)
        return MetricsSnapshot.from_payload(self._unwrap_mapping(payload))

    def _counts(self, path: str) -> CountSnapshot:
        payload = self._http.get(path)
        return CountSnapshot.from_payload(self._unwrap_mapping(payload))

    def _list_models_at(self, path: str, model: Type[R]) -> List[R]:
        payload = self._http.get(path)
        data = self._unwrap(payload)
        if isinstance(data, list):
            return [model.from_dict(item) for item in data if isinstance(item, dict)]
        return []

    def _parse_model(self, payload: Any, model: Type[R]) -> R:
        unwrapped = self._unwrap(payload)
        mapping = self._ensure_mapping(unwrapped)
        return model.from_dict(mapping)

    def _unwrap(self, payload: Any) -> Any:
        if isinstance(payload, dict) and "data" in payload and "access_token" not in payload:
            return payload["data"]
        return payload

    def _unwrap_mapping(self, payload: Any) -> Dict[str, Any]:
        return self._ensure_mapping(self._unwrap(payload))

    def _ensure_mapping(self, payload: Any) -> Dict[str, Any]:
        if isinstance(payload, dict):
            return cast(Dict[str, Any], payload)
        return {}
