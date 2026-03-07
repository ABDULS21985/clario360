'use client';

import { useMemo, useState } from 'react';
import type { LineageLayoutSnapshot, LineageViewportState } from '@/app/(dashboard)/data/lineage/_components/lineage-dag';

interface LineageMinimapProps {
  layout: LineageLayoutSnapshot | null;
  viewport: LineageViewportState | null;
  onNavigate: (x: number, y: number) => void;
}

export function LineageMinimap({
  layout,
  viewport,
  onNavigate,
}: LineageMinimapProps) {
  const [dragging, setDragging] = useState(false);
  const width = 260;
  const height = 180;
  const scale = useMemo(() => {
    if (!layout) {
      return 1;
    }
    return Math.min(width / layout.width, height / layout.height);
  }, [layout]);

  const viewportRect = useMemo(() => {
    if (!layout || !viewport) {
      return null;
    }
    return {
      x: Math.max((-viewport.x / viewport.k) * scale, 0),
      y: Math.max((-viewport.y / viewport.k) * scale, 0),
      width: Math.min((viewport.viewportWidth / viewport.k) * scale, width),
      height: Math.min((viewport.viewportHeight / viewport.k) * scale, height),
    };
  }, [layout, scale, viewport]);

  const navigateFromPointer = (event: React.PointerEvent<SVGSVGElement>) => {
    if (!layout) {
      return;
    }
    const bounds = event.currentTarget.getBoundingClientRect();
    const x = (event.clientX - bounds.left) / scale;
    const y = (event.clientY - bounds.top) / scale;
    onNavigate(x, y);
  };

  return (
    <div className="rounded-lg border bg-card p-3">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">Overview</div>
      <div className="mt-2 text-sm text-muted-foreground">Drag or click the viewport to navigate.</div>
      {layout ? (
        <svg
          width={width}
          height={height}
          className="mt-3 rounded-md border bg-muted/10"
          onPointerDown={(event) => {
            setDragging(true);
            navigateFromPointer(event);
          }}
          onPointerMove={(event) => {
            if (!dragging) {
              return;
            }
            navigateFromPointer(event);
          }}
          onPointerUp={() => setDragging(false)}
          onPointerLeave={() => setDragging(false)}
        >
          {layout.edges.map((edge) => (
            <path
              key={edge.id}
              d={edge.points.reduce(
                (acc, point, index) =>
                  `${acc}${index === 0 ? `M ${point.x * scale} ${point.y * scale}` : ` L ${point.x * scale} ${point.y * scale}`}`,
                '',
              )}
              fill="none"
              stroke="#cbd5e1"
              strokeWidth={1}
            />
          ))}
          {layout.nodes.map((node) => (
            <circle
              key={node.id}
              cx={node.position.x * scale}
              cy={node.position.y * scale}
              r={3}
              fill="#64748b"
            />
          ))}
          {viewportRect ? (
            <rect
              x={viewportRect.x}
              y={viewportRect.y}
              width={viewportRect.width}
              height={viewportRect.height}
              fill="rgba(37,99,235,0.15)"
              stroke="#2563eb"
              strokeWidth={2}
            />
          ) : null}
        </svg>
      ) : null}
    </div>
  );
}
