from __future__ import annotations

from typing import Any, Dict, Generic, Iterator, List, Mapping, Optional, Sequence, Type, TypeVar

from pydantic import Field

from clario360.models.base import BaseModel
from clario360.utils.dataframe import models_to_dataframe

T = TypeVar("T", bound=BaseModel)


class PaginationMeta(BaseModel):
    page: int = 1
    per_page: int = 0
    total: int = 0
    total_pages: int = 0


class PaginatedResponse(BaseModel, Generic[T]):
    data: List[T] = Field(default_factory=list)
    meta: PaginationMeta = Field(default_factory=PaginationMeta)

    @classmethod
    def from_payload(
        cls,
        payload: Mapping[str, Any],
        item_model: Type[T],
    ) -> "PaginatedResponse[T]":
        raw_items = payload.get("data", [])
        items = [item_model.from_dict(item) for item in raw_items if isinstance(item, dict)]
        meta_payload = payload.get("meta", {})
        meta = PaginationMeta.from_dict(meta_payload if isinstance(meta_payload, dict) else {})
        return cls(data=items, meta=meta)

    @property
    def page(self) -> int:
        return self.meta.page

    @property
    def per_page(self) -> int:
        return self.meta.per_page

    @property
    def total(self) -> int:
        return self.meta.total

    @property
    def total_pages(self) -> int:
        return self.meta.total_pages

    def __iter__(self) -> Iterator[T]:  # type: ignore[override]
        return iter(self.data)

    def __len__(self) -> int:
        return len(self.data)

    def to_dataframe(self) -> Any:
        return models_to_dataframe(self.data)


class MessageResponse(BaseModel):
    message: str


class HealthStatus(BaseModel):
    status: str = "ok"
    service: Optional[str] = None
    version: Optional[str] = None


class MetricsSnapshot(BaseModel):
    metrics: Dict[str, Any] = Field(default_factory=dict)

    @classmethod
    def from_payload(cls, payload: Mapping[str, Any]) -> "MetricsSnapshot":
        return cls(metrics=dict(payload))


class CountSnapshot(BaseModel):
    counts: Dict[str, int] = Field(default_factory=dict)

    @classmethod
    def from_payload(cls, payload: Mapping[str, Any]) -> "CountSnapshot":
        counts: Dict[str, int] = {}
        for key, value in payload.items():
            if isinstance(value, bool):
                continue
            if isinstance(value, int):
                counts[key] = value
        return cls(counts=counts)


class StringListResponse(BaseModel):
    values: List[str] = Field(default_factory=list)

    @classmethod
    def from_payload(cls, payload: Sequence[Any]) -> "StringListResponse":
        return cls(values=[str(item) for item in payload])
