'use client';

import { useRouter } from 'next/navigation';
import type { CyberSeverity, RiskHeatmapCell } from '@/types/cyber';

// Per-severity color ramps (Tailwind color values as hex for SVG)
const COLOR_RAMPS: Record<CyberSeverity, string[]> = {
  critical: ['#ffffff', '#fecaca', '#fca5a5', '#f87171', '#b91c1c'],
  high: ['#ffffff', '#fed7aa', '#fdba74', '#fb923c', '#c2410c'],
  medium: ['#ffffff', '#fef08a', '#fde047', '#eab308', '#a16207'],
  low: ['#ffffff', '#bfdbfe', '#93c5fd', '#3b82f6', '#1d4ed8'],
  info: ['#ffffff', '#e0e7ff', '#a5b4fc', '#6366f1', '#3730a3'],
};

function getBucketIndex(count: number, maxCount: number): number {
  if (count === 0) return 0;
  if (maxCount === 0) return 0;
  const ratio = count / maxCount;
  if (ratio <= 0.1) return 1;
  if (ratio <= 0.4) return 2;
  if (ratio <= 0.75) return 3;
  return 4;
}

function getTextColor(bucketIndex: number): string {
  return bucketIndex >= 3 ? '#ffffff' : '#1f2937';
}

interface HeatmapCellProps {
  cell: RiskHeatmapCell;
  maxForSeverity: number;
  x: number;
  y: number;
  width: number;
  height: number;
}

export function HeatmapSvgCell({ cell, maxForSeverity, x, y, width, height }: HeatmapCellProps) {
  const router = useRouter();
  const bucketIdx = getBucketIndex(cell.count, maxForSeverity);
  const fillColor = COLOR_RAMPS[cell.severity]?.[bucketIdx] ?? '#ffffff';
  const textColor = getTextColor(bucketIdx);

  const handleClick = () => {
    const params = new URLSearchParams({
      type: cell.asset_type,
      vulnerability_severity: cell.severity,
      has_vulnerabilities: 'true',
    });
    router.push(`/cyber/assets?${params.toString()}`);
  };

  const tooltipText = [
    `${cell.count} ${cell.severity} vulnerabilities on ${cell.asset_type.replace(/_/g, ' ')} assets`,
    `Affecting ${cell.affected_asset_count} of ${cell.total_assets_of_type} assets`,
    'Click to view →',
  ].join('\n');

  return (
    <g
      role="button"
      tabIndex={0}
      onClick={handleClick}
      onKeyDown={(e) => e.key === 'Enter' && handleClick()}
      style={{ cursor: 'pointer' }}
      aria-label={tooltipText}
    >
      <title>{tooltipText}</title>
      <rect
        x={x}
        y={y}
        width={width}
        height={height}
        fill={fillColor}
        stroke="#e5e7eb"
        strokeWidth={1}
        rx={2}
        className="transition-opacity hover:opacity-80"
      />
      {cell.count > 0 && (
        <text
          x={x + width / 2}
          y={y + height / 2}
          textAnchor="middle"
          dominantBaseline="central"
          fill={textColor}
          fontSize={11}
          fontWeight={600}
          fontFamily="ui-monospace, monospace"
        >
          {cell.count}
        </text>
      )}
    </g>
  );
}
