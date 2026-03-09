from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class Committee(BaseModel):
    id: str
    name: str
    description: Optional[str] = None
    status: Optional[str] = None
    members: List[Dict[str, Any]] = Field(default_factory=list)


class Meeting(BaseModel):
    id: str
    title: str
    status: Optional[str] = None
    scheduled_at: Optional[str] = None
    committee_id: Optional[str] = None
    created_at: Optional[str] = None


class AgendaItem(BaseModel):
    id: str
    title: str
    status: Optional[str] = None
    notes: Optional[str] = None


class ActionItem(BaseModel):
    id: str
    title: str
    status: Optional[str] = None
    due_at: Optional[str] = None
    assigned_to: Optional[str] = None


class Minutes(BaseModel):
    id: Optional[str] = None
    status: Optional[str] = None
    content: Optional[str] = None
    created_at: Optional[str] = None
