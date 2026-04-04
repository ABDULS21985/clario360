'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import * as d3 from 'd3';
import dagre from 'dagre';
import {
  type ImpactAnalysis,
  type LineageEdge as LineageEdgeType,
  type LineageGraph,
  type LineageNode as LineageNodeType,
} from '@/lib/data-suite';
import { LineageEdge } from '@/app/(dashboard)/data/lineage/_components/lineage-edge';
import { LineageNode, LINEAGE_NODE_HEIGHT, LINEAGE_NODE_WIDTH } from '@/app/(dashboard)/data/lineage/_components/lineage-node';

interface LineageDagProps {
  graph: LineageGraph;
  direction: 'LR' | 'TB';
  selectedNodeId: string | null;
  search: string;
  impact: ImpactAnalysis | null;
  onSelectNode: (node: LineageNodeType) => void;
  onReady?: (api: LineageDagApi) => void;
  onLayoutChange?: (layout: LineageLayoutSnapshot) => void;
  onViewportChange?: (viewport: LineageViewportState) => void;
}

const NODE_COLORS: Record<string, string> = {
  data_source: '#2563eb',
  pipeline: '#f97316',
  data_model: '#16a34a',
  quality_rule: '#0f766e',
  suite_consumer: '#7c3aed',
  report: '#64748b',
};

interface PositionedNode extends LineageNodeType {
  position: { x: number; y: number; width: number; height: number };
}

interface PositionedEdge extends LineageEdgeType {
  points: Array<{ x: number; y: number }>;
}

export interface LineageLayoutSnapshot {
  nodes: PositionedNode[];
  edges: PositionedEdge[];
  width: number;
  height: number;
}

export interface LineageViewportState {
  x: number;
  y: number;
  k: number;
  viewportWidth: number;
  viewportHeight: number;
}

export interface LineageDagApi {
  fitToScreen: () => void;
  reset: () => void;
  zoomIn: () => void;
  zoomOut: () => void;
  fullscreen: () => void;
  centerOn: (x: number, y: number) => void;
}

export function LineageDag({
  graph,
  direction,
  selectedNodeId,
  search,
  impact,
  onSelectNode,
  onReady,
  onLayoutChange,
  onViewportChange,
}: LineageDagProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const viewportRef = useRef<SVGGElement | null>(null);
  const zoomRef = useRef<d3.ZoomBehavior<SVGSVGElement, unknown> | null>(null);
  const [containerSize, setContainerSize] = useState({ width: 960, height: 720 });
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

  const [viewport, setViewport] = useState<LineageViewportState>({
    x: 0,
    y: 0,
    k: 1,
    viewportWidth: containerSize.width,
    viewportHeight: containerSize.height,
  });

  useEffect(() => {
    onLayoutChange?.(layout);
  }, [layout, onLayoutChange]);

  useEffect(() => {
    onViewportChange?.(viewport);
  }, [onViewportChange, viewport]);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (!entry) {
        return;
      }
      setContainerSize({
        width: Math.max(entry.contentRect.width, 640),
        height: Math.max(entry.contentRect.height, 520),
      });
    });
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, []);

  const applyTransform = (transform: d3.ZoomTransform) => {
    if (!svgRef.current || !zoomRef.current) {
      return;
    }
    d3.select(svgRef.current)
      .transition()
      .duration(250)
      .call(zoomRef.current.transform, transform);
  };

  const fitToScreen = () => {
    const scale = Math.min(
      (containerSize.width - 80) / layout.width,
      (containerSize.height - 80) / layout.height,
      1,
    );
    const transform = d3.zoomIdentity
      .translate(
        (containerSize.width - layout.width * scale) / 2,
        (containerSize.height - layout.height * scale) / 2,
      )
      .scale(scale);
    applyTransform(transform);
  };

  const reset = () => {
    applyTransform(d3.zoomIdentity.translate(24, 24).scale(1));
  };

  const zoomIn = () => {
    if (!svgRef.current || !zoomRef.current) {
      return;
    }
    d3.select(svgRef.current).transition().duration(180).call(zoomRef.current.scaleBy, 1.2);
  };

  const zoomOut = () => {
    if (!svgRef.current || !zoomRef.current) {
      return;
    }
    d3.select(svgRef.current).transition().duration(180).call(zoomRef.current.scaleBy, 0.85);
  };

  const centerOn = (x: number, y: number) => {
    const transform = d3.zoomIdentity
      .translate(containerSize.width / 2 - x * viewport.k, containerSize.height / 2 - y * viewport.k)
      .scale(viewport.k);
    applyTransform(transform);
  };

  const fullscreen = () => {
    if (!containerRef.current || !containerRef.current.requestFullscreen) {
      return;
    }
    void containerRef.current.requestFullscreen();
  };

  useEffect(() => {
    if (!svgRef.current || !viewportRef.current) {
      return;
    }

    const selection = d3.select(svgRef.current);
    const zoom = d3
      .zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.3, 2.8])
      .on('zoom', (event) => {
        const transform = event.transform;
        d3.select(viewportRef.current).attr(
          'transform',
          `translate(${transform.x}, ${transform.y}) scale(${transform.k})`,
        );
        setViewport({
          x: transform.x,
          y: transform.y,
          k: transform.k,
          viewportWidth: containerSize.width,
          viewportHeight: containerSize.height,
        });
      });
    selection.call(zoom);
    zoomRef.current = zoom;
    return () => {
      selection.on('.zoom', null);
    };
  }, [containerSize.height, containerSize.width]);

  useEffect(() => {
    fitToScreen();
    // Re-fit when the graph layout changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [layout.height, layout.width, direction]);

  useEffect(() => {
    onReady?.({
      fitToScreen,
      reset,
      zoomIn,
      zoomOut,
      fullscreen,
      centerOn,
    });
  }, [onReady, containerSize.height, containerSize.width, layout.height, layout.width, viewport.k]);

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
    <div ref={containerRef} className="relative h-[720px] overflow-hidden rounded-lg border bg-card">
      <svg ref={svgRef} width="100%" height="100%">
        <defs>
          <marker id="lineage-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M 0 0 L 10 5 L 0 10 z" fill="#94a3b8" />
          </marker>
        </defs>
        <g ref={viewportRef}>
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
                data-testid={`lineage-node-${node.id}`}
              >
                <LineageNode node={node} selected={selected} dimmed={dimmed} fill={fill} />
              </g>
            );
          })}
        </g>
      </svg>
    </div>
  );
}
