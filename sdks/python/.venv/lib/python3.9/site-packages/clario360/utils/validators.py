from __future__ import annotations

from datetime import date, datetime
from typing import Optional
from uuid import UUID

from clario360.exceptions import ValidationError


def ensure_uuid(value: str, *, field_name: str) -> str:
    try:
        UUID(value)
    except ValueError as exc:
        raise ValidationError(f"{field_name} must be a valid UUID", code="INVALID_UUID") from exc
    return value


def iso_date(value: Optional[date]) -> Optional[str]:
    return value.isoformat() if value is not None else None


def iso_datetime(value: Optional[datetime]) -> Optional[str]:
    return value.isoformat() if value is not None else None
