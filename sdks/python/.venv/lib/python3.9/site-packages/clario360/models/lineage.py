from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import Field

from clario360.models.base import BaseModel


class LineageNode(BaseModel):
    id: str
    entity_type: Optional[str] = None
    name: Optional[str] = None
    metadata: Dict[str, Any] = Field(default_factory=dict)


class LineageEdge(BaseModel):
    id: Optional[str] = None
    source_id: str
    target_id: str
    relation: Optional[str] = None


class LineageGraph(BaseModel):
    nodes: List[LineageNode] = Field(default_factory=list)
    edges: List[LineageEdge] = Field(default_factory=list)


class ImpactAnalysis(BaseModel):
    entity_id: Optional[str] = None
    impacted_nodes: List[LineageNode] = Field(default_factory=list)
    summary: Dict[str, Any] = Field(default_factory=dict)
