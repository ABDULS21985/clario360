from __future__ import annotations

from typing import Any, Dict, List

from pydantic import Field

from clario360.models.base import BaseModel


class LineageNode(BaseModel):
    id: str
    entity_type: str | None = None
    name: str | None = None
    metadata: Dict[str, Any] = Field(default_factory=dict)


class LineageEdge(BaseModel):
    id: str | None = None
    source_id: str
    target_id: str
    relation: str | None = None


class LineageGraph(BaseModel):
    nodes: List[LineageNode] = Field(default_factory=list)
    edges: List[LineageEdge] = Field(default_factory=list)


class ImpactAnalysis(BaseModel):
    entity_id: str | None = None
    impacted_nodes: List[LineageNode] = Field(default_factory=list)
    summary: Dict[str, Any] = Field(default_factory=dict)
