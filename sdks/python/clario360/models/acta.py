from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class Committee(BaseModel):
    id: str
    name: str
    description: str | None = None
    status: str | None = None
    members: List[Dict[str, Any]] = Field(default_factory=list)


class Meeting(BaseModel):
    id: str
    title: str
    status: str | None = None
    scheduled_at: str | None = None
    committee_id: str | None = None
    created_at: str | None = None


class AgendaItem(BaseModel):
    id: str
    title: str
    status: str | None = None
    notes: str | None = None


class ActionItem(BaseModel):
    id: str
    title: str
    status: str | None = None
    due_at: str | None = None
    assigned_to: str | None = None


class Minutes(BaseModel):
    id: str | None = None
    status: str | None = None
    content: str | None = None
    created_at: str | None = None
