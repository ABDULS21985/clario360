from __future__ import annotations

import json
from typing import Any, Mapping, Sequence

from clario360.models.base import BaseModel
from clario360.models.common import PaginatedResponse


def serialize_value(value: Any) -> Any:
    if isinstance(value, PaginatedResponse):
        return {
            "data": [serialize_value(item) for item in value.data],
            "meta": value.meta.to_dict(),
        }
    if isinstance(value, BaseModel):
        return value.to_dict()
    if isinstance(value, Mapping):
        return {str(key): serialize_value(item) for key, item in value.items()}
    if isinstance(value, Sequence) and not isinstance(value, (str, bytes, bytearray)):
        return [serialize_value(item) for item in value]
    return value


def render_json(value: Any) -> str:
    return json.dumps(serialize_value(value), indent=2, sort_keys=True, default=str)
