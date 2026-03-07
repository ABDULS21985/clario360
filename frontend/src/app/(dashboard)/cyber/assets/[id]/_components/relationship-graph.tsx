'use client';

import { useEffect, useRef, useCallback } from 'react';
import type { AssetRelationship, AssetType } from '@/types/cyber';

interface GraphNode {
  id: string;
  name: string;
  type: AssetType;
  criticality: string;
  isCurrent: boolean;
  x?: number;
  y?: number;
  vx?: number;
  vy?: number;
  fx?: number | null;
  fy?: number | null;
}

interface GraphLink {
  source: string | GraphNode;
  target: string | GraphNode;
  relationship_type: string;
}

interface RelationshipGraphProps {
  assetId: string;
  assetName: string;
  assetType: AssetType;
  relationships: AssetRelationship[];
  width?: number;
  height?: number;
}

const TYPE_COLORS: Record<string, string> = {
  server: '#3b82f6',
  endpoint: '#8b5cf6',
  cloud_resource: '#06b6d4',
  network_device: '#f59e0b',
  iot_device: '#84cc16',
  application: '#f97316',
  database: '#ec4899',
  container: '#14b8a6',
};

const CRITICALITY_STROKE: Record<string, string> = {
  critical: '#ef4444',
  high: '#f97316',
  medium: '#eab308',
  low: '#22c55e',
};

export function RelationshipGraph({
  assetId,
  assetName,
  assetType,
  relationships,
  width = 700,
  height = 450,
}: RelationshipGraphProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const simulationRef = useRef<import('d3').Simulation<GraphNode, GraphLink> | null>(null);

  const draw = useCallback(async () => {
    if (!svgRef.current) return;

    const d3 = await import('d3');
    const svg = d3.select(svgRef.current);
    svg.selectAll('*').remove();

    // Build graph data
    const nodeMap = new Map<string, GraphNode>();
    nodeMap.set(assetId, {
      id: assetId,
      name: assetName,
      type: assetType,
      criticality: 'high',
      isCurrent: true,
    });

    relationships.forEach((rel) => {
      if (!nodeMap.has(rel.source_asset_id)) {
        nodeMap.set(rel.source_asset_id, {
          id: rel.source_asset_id,
          name: rel.source_asset_name,
          type: rel.source_asset_type as AssetType,
          criticality: rel.source_criticality,
          isCurrent: false,
        });
      }
      if (!nodeMap.has(rel.target_asset_id)) {
        nodeMap.set(rel.target_asset_id, {
          id: rel.target_asset_id,
          name: rel.target_asset_name,
          type: rel.target_asset_type as AssetType,
          criticality: rel.target_criticality,
          isCurrent: false,
        });
      }
    });

    const nodes: GraphNode[] = Array.from(nodeMap.values());
    const links: GraphLink[] = relationships.map((rel) => ({
      source: rel.source_asset_id,
      target: rel.target_asset_id,
      relationship_type: rel.relationship_type,
    }));

    // Set up SVG
    const g = svg.append('g');

    // Zoom behavior
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.3, 3])
      .on('zoom', (event) => {
        g.attr('transform', event.transform as unknown as string);
      });
    svg.call(zoom);

    // Defs for arrowhead
    svg.append('defs').append('marker')
      .attr('id', 'arrowhead')
      .attr('viewBox', '-0 -5 10 10')
      .attr('refX', 22)
      .attr('refY', 0)
      .attr('orient', 'auto')
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .append('svg:path')
      .attr('d', 'M 0,-5 L 10 ,0 L 0,5')
      .attr('fill', '#94a3b8')
      .style('stroke', 'none');

    // Links
    const link = g.append('g')
      .attr('class', 'links')
      .selectAll('g')
      .data(links)
      .join('g');

    const linkLine = link.append('line')
      .attr('stroke', '#94a3b8')
      .attr('stroke-width', 1.5)
      .attr('stroke-opacity', 0.6)
      .attr('marker-end', 'url(#arrowhead)');

    const linkLabel = link.append('text')
      .attr('text-anchor', 'middle')
      .attr('font-size', 9)
      .attr('fill', '#94a3b8')
      .attr('dy', -4)
      .text((d) => (d.relationship_type ?? '').replace(/_/g, ' '));

    // Nodes
    const node = g.append('g')
      .attr('class', 'nodes')
      .selectAll('g')
      .data(nodes)
      .join('g')
      .attr('cursor', 'pointer')
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      .call(d3.drag<SVGGElement, GraphNode>()
          .on('start', (event, d) => {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            d.fx = d.x;
            d.fy = d.y;
          })
          .on('drag', (event, d) => {
            d.fx = event.x;
            d.fy = event.y;
          })
          .on('end', (event, d) => {
            if (!event.active) simulation.alphaTarget(0);
            d.fx = null;
            d.fy = null;
          }) as never);

    // Node circles
    node.append('circle')
      .attr('r', (d) => d.isCurrent ? 22 : 16)
      .attr('fill', (d) => TYPE_COLORS[d.type] ?? '#6b7280')
      .attr('fill-opacity', (d) => d.isCurrent ? 1 : 0.8)
      .attr('stroke', (d) => CRITICALITY_STROKE[d.criticality] ?? '#6b7280')
      .attr('stroke-width', (d) => d.isCurrent ? 3 : 2);

    // Node icons (text emoji stand-ins)
    node.append('text')
      .attr('text-anchor', 'middle')
      .attr('dy', '0.35em')
      .attr('font-size', (d) => d.isCurrent ? 14 : 10)
      .attr('fill', 'white')
      .attr('pointer-events', 'none')
      .text((d) => {
        const icons: Record<string, string> = {
          server: '⬛', endpoint: '💻', cloud_resource: '☁', network_device: '🔌',
          iot_device: '📡', application: '⚙', database: '🗄', container: '📦',
        };
        return icons[d.type] ?? '●';
      });

    // Node labels
    node.append('text')
      .attr('text-anchor', 'middle')
      .attr('dy', (d) => d.isCurrent ? 36 : 28)
      .attr('font-size', 10)
      .attr('fill', 'currentColor')
      .attr('pointer-events', 'none')
      .text((d) => d.name.length > 18 ? d.name.slice(0, 15) + '…' : d.name);

    // Simulation
    const simulation = d3.forceSimulation<GraphNode>(nodes)
      .force('link', d3.forceLink<GraphNode, GraphLink>(links)
        .id((d) => d.id)
        .distance(120))
      .force('charge', d3.forceManyBody().strength(-300))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide(40))
      .on('tick', () => {
        linkLine
          .attr('x1', (d) => (d.source as GraphNode).x ?? 0)
          .attr('y1', (d) => (d.source as GraphNode).y ?? 0)
          .attr('x2', (d) => (d.target as GraphNode).x ?? 0)
          .attr('y2', (d) => (d.target as GraphNode).y ?? 0);

        linkLabel
          .attr('x', (d) => (((d.source as GraphNode).x ?? 0) + ((d.target as GraphNode).x ?? 0)) / 2)
          .attr('y', (d) => (((d.source as GraphNode).y ?? 0) + ((d.target as GraphNode).y ?? 0)) / 2);

        node.attr('transform', (d) => `translate(${d.x ?? 0},${d.y ?? 0})`);
      });

    simulationRef.current = simulation as import('d3').Simulation<GraphNode, GraphLink>;
  }, [assetId, assetName, assetType, relationships, width, height]);

  useEffect(() => {
    void draw();
    return () => {
      simulationRef.current?.stop();
    };
  }, [draw]);

  return (
    <div className="overflow-hidden rounded-lg border bg-background">
      <svg
        ref={svgRef}
        width={width}
        height={height}
        className="w-full"
        style={{ minHeight: height }}
      />
      {/* Legend */}
      <div className="flex flex-wrap gap-3 border-t px-4 py-2">
        {Object.entries(TYPE_COLORS).map(([type, color]) => (
          <div key={type} className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <div className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: color }} />
            {type.replace(/_/g, ' ')}
          </div>
        ))}
      </div>
    </div>
  );
}
