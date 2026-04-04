from __future__ import annotations

from typing import Any, Dict, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class Notification(BaseModel):
    id: str
    title: str
    body: str
    category: Optional[str] = None
    priority: Optional[str] = None
    action_url: Optional[str] = None
    read: bool = False
    read_at: Optional[str] = None
    created_at: Optional[str] = None


class NotificationPreference(BaseModel):
    channels: Dict[str, Any] = Field(default_factory=dict)
    quiet_hours: Dict[str, Any] = Field(default_factory=dict)
