'use client';

import { useMemo } from 'react';
import dagre from 'dagre';
import { type ImpactAnalysis, type LineageGraph, type LineageNode as LineageNodeType } from '@/lib/data-suite';
import { LineageEdge } from '@/app/(dashboard)/data/lineage/_components/lineage-edge';
import { LineageNode, LINEAGE_NODE_HEIGHT, LINEAGE_NODE_WIDTH } from '@/app/(dashboard)/data/lineage/_components/lineage-node';

interface LineageDagProps {
  graph: LineageGraph;
  direction: 'LR' | 'TB';
  selectedNodeId: string | null;
  search: string;
  impact: ImpactAnalysis | null;
  onSelectNode: (node: LineageNodeType) => void;
}

const NODE_COLORS: Record<string, string> = {
  data_source: '#2563eb',
  pipeline: '#f97316',
  data_model: '#16a34a',
  quality_rule: '#0f766e',
  suite_consumer: '#7c3aed',
  report: '#64748b',
};

export function LineageDag({
  graph,
  direction,
  selectedNodeId,
  search,
  impact,
  onSelectNode,
}: LineageDagProps) {
  const layout = useMemo(() => {
    const g = new dagre.graphlib.Graph({ directed: true });
    g.setGraph({
      rankdir: direction,
      ranksep: 120,
      nodesep: 60,
      marginx: 40,
      marginy: 40,
    });
    g.setDefaultEdgeLabel(() => ({}));

    graph.nodes.forEach((node) => {
      g.setNode(node.id, {
        width: LINEAGE_NODE_WIDTH,
        height: LINEAGE_NODE_HEIGHT,
      });
    });
    graph.edges.forEach((edge) => {
      g.setEdge(edge.source, edge.target, {});
    });
    dagre.layout(g);

    const nodes = graph.nodes.map((node) => ({
      ...node,
      position: g.node(node.id) as { x: number; y: number; width: number; height: number },
    }));
    const edges = graph.edges.map((edge) => ({
      ...edge,
      points: (g.edge(edge.source, edge.target)?.points ?? []) as Array<{ x: number; y: number }>,
    }));

    const width = Math.max(...nodes.map((node) => node.position.x + LINEAGE_NODE_WIDTH / 2), 800) + 80;
    const height = Math.max(...nodes.map((node) => node.position.y + LINEAGE_NODE_HEIGHT / 2), 600) + 80;

    return { nodes, edges, width, height };
  }, [direction, graph.edges, graph.nodes]);

  const loweredSearch = search.trim().toLowerCase();
  const searchMatches = new Set(
    graph.nodes
      .filter((node) =>
        loweredSearch
          ? `${node.name} ${node.type} ${node.entity_id}`.toLowerCase().includes(loweredSearch)
          : true,
      )
      .map((node) => node.id),
  );

  const selectedEdgeNodeIds = useMemo(() => {
    if (!selectedNodeId) {
      return new Set<string>();
    }
    const ids = new Set<string>([selectedNodeId]);
    graph.edges.forEach((edge) => {
      if (edge.source === selectedNodeId || edge.target === selectedNodeId) {
        ids.add(edge.source);
        ids.add(edge.target);
      }
    });
    return ids;
  }, [graph.edges, selectedNodeId]);

  const directImpactIds = new Set((impact?.directly_affected ?? []).map((item) => item.node.id));
  const indirectImpactIds = new Set((impact?.indirectly_affected ?? []).map((item) => item.node.id));

  return (
    <div className="overflow-auto rounded-lg border bg-card">
      <svg width={layout.width} height={layout.height} className="min-w-full">
        <defs>
          <marker id="lineage-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M 0 0 L 10 5 L 0 10 z" fill="#94a3b8" />
          </marker>
        </defs>

        {layout.edges.map((edge) => {
          const path = edge.points.reduce((acc, point, index) => {
            return `${acc}${index === 0 ? `M ${point.x} ${point.y}` : ` L ${point.x} ${point.y}`}`;
          }, '');
          const middle = edge.points[Math.floor(edge.points.length / 2)] ?? { x: 0, y: 0 };
          const highlighted = selectedNodeId ? edge.source === selectedNodeId || edge.target === selectedNodeId : false;
          const dimmed =
            (selectedNodeId && !highlighted) ||
            (loweredSearch !== '' && !searchMatches.has(edge.source) && !searchMatches.has(edge.target));
          return (
            <LineageEdge
              key={edge.id}
              edge={edge}
              path={path}
              labelX={middle.x}
              labelY={middle.y - 6}
              highlighted={highlighted}
              dimmed={dimmed}
            />
          );
        })}

        {layout.nodes.map((node) => {
          const selected = node.id === selectedNodeId;
          const dimmed =
            (selectedNodeId && !selectedEdgeNodeIds.has(node.id)) ||
            (loweredSearch !== '' && !searchMatches.has(node.id));

          let fill = NODE_COLORS[node.type] ?? '#334155';
          if (impact) {
            if (node.id === impact.entity.id) {
              fill = '#2563eb';
            } else if (directImpactIds.has(node.id)) {
              fill = '#f97316';
            } else if (indirectImpactIds.has(node.id)) {
              fill = '#eab308';
            }
          }

          return (
            <g
              key={node.id}
              transform={`translate(${node.position.x}, ${node.position.y})`}
              onClick={() => onSelectNode(node)}
              className="cursor-pointer"
            >
              <LineageNode node={node} selected={selected} dimmed={dimmed} fill={fill} />
            </g>
          );
        })}
      </svg>
    </div>
  );
}
