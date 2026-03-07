'use client';

import { type LineageEdge as LineageEdgeType } from '@/lib/data-suite';

interface LineageEdgeProps {
  edge: LineageEdgeType;
  path: string;
  labelX: number;
  labelY: number;
  highlighted: boolean;
  dimmed: boolean;
}

export function LineageEdge({
  edge,
  path,
  labelX,
  labelY,
  highlighted,
  dimmed,
}: LineageEdgeProps) {
  return (
    <g opacity={dimmed ? 0.2 : 1}>
      <path
        d={path}
        fill="none"
        stroke={highlighted ? '#2563eb' : '#94a3b8'}
        strokeWidth={highlighted ? 3 : 2}
        strokeDasharray={edge.active ? undefined : '6 4'}
        markerEnd="url(#lineage-arrow)"
      />
      <text x={labelX} y={labelY} fontSize="10" fill="#475569" textAnchor="middle">
        {edge.relationship}
      </text>
    </g>
  );
}
