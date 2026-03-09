from __future__ import annotations

from typing import Any, Dict

from pydantic import Field

from clario360.models.base import BaseModel


class Notification(BaseModel):
    id: str
    title: str
    body: str
    category: str | None = None
    priority: str | None = None
    action_url: str | None = None
    read: bool = False
    read_at: str | None = None
    created_at: str | None = None


class NotificationPreference(BaseModel):
    channels: Dict[str, Any] = Field(default_factory=dict)
    quiet_hours: Dict[str, Any] = Field(default_factory=dict)
