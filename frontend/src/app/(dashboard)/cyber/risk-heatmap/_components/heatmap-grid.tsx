'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import type { RiskHeatmapData, CyberSeverity, RiskHeatmapCell } from '@/types/cyber';
import { HeatmapSvgCell } from './heatmap-cell';

const SEVERITIES: CyberSeverity[] = ['critical', 'high', 'medium', 'low'];
const SEVERITY_LABELS: Record<CyberSeverity, string> = {
  critical: 'Critical',
  high: 'High',
  medium: 'Medium',
  low: 'Low',
  info: 'Info',
};

const CELL_W = 120;
const CELL_H = 56;
const ROW_LABEL_W = 130;
const COL_LABEL_H = 32;
const TOTAL_COL_W = 70;
const PADDING = 8;

interface HeatmapGridProps {
  data: RiskHeatmapData;
}

export function HeatmapGrid({ data }: HeatmapGridProps) {
  const router = useRouter();
  const [sortBySeverity, setSortBySeverity] = useState<CyberSeverity | null>(null);

  // Compute sorted asset types
  const assetTypes = [...data.asset_types].sort((a, b) => {
    if (sortBySeverity) {
      const aVal = data.cells.find((c) => c.asset_type === a && c.severity === sortBySeverity)?.count ?? 0;
      const bVal = data.cells.find((c) => c.asset_type === b && c.severity === sortBySeverity)?.count ?? 0;
      return bVal - aVal;
    }
    // Default: sort by total desc
    const aTotal = data.cells.filter((c) => c.asset_type === a).reduce((s, c) => s + c.count, 0);
    const bTotal = data.cells.filter((c) => c.asset_type === b).reduce((s, c) => s + c.count, 0);
    return bTotal - aTotal;
  });

  // Compute max per severity column (for relative intensity)
  const maxPerSeverity: Record<CyberSeverity, number> = { critical: 0, high: 0, medium: 0, low: 0, info: 0 };
  for (const sev of SEVERITIES) {
    maxPerSeverity[sev] = Math.max(
      ...data.cells.filter((c) => c.severity === sev).map((c) => c.count),
      1,
    );
  }

  // Row totals
  const rowTotals = assetTypes.map((type) =>
    data.cells.filter((c) => c.asset_type === type).reduce((s, c) => s + c.count, 0),
  );
  const colTotals = SEVERITIES.map((sev) =>
    data.cells.filter((c) => c.severity === sev).reduce((s, c) => s + c.count, 0),
  );

  const rows = assetTypes.length;
  const cols = SEVERITIES.length;
  const svgW = ROW_LABEL_W + cols * (CELL_W + PADDING) + TOTAL_COL_W + PADDING;
  const svgH = COL_LABEL_H + rows * (CELL_H + PADDING) + CELL_H + PADDING;

  function getCell(assetType: string, severity: CyberSeverity): RiskHeatmapCell {
    return (
      data.cells.find((c) => c.asset_type === assetType && c.severity === severity) ?? {
        asset_type: assetType,
        severity,
        count: 0,
        affected_asset_count: 0,
        total_assets_of_type: 0,
      }
    );
  }

  return (
    <div className="overflow-x-auto rounded-xl border bg-card p-4">
      <svg
        width={svgW}
        height={svgH}
        viewBox={`0 0 ${svgW} ${svgH}`}
        role="img"
        aria-label="Risk heatmap showing vulnerability distribution"
      >
        {/* Column headers */}
        {SEVERITIES.map((sev, ci) => {
          const x = ROW_LABEL_W + ci * (CELL_W + PADDING);
          return (
            <g
              key={sev}
              role="button"
              tabIndex={0}
              onClick={() => setSortBySeverity(sortBySeverity === sev ? null : sev)}
              onKeyDown={(e) => e.key === 'Enter' && setSortBySeverity(sortBySeverity === sev ? null : sev)}
              style={{ cursor: 'pointer' }}
            >
              <text
                x={x + CELL_W / 2}
                y={COL_LABEL_H / 2 + 5}
                textAnchor="middle"
                dominantBaseline="central"
                fontSize={12}
                fontWeight={sortBySeverity === sev ? 700 : 500}
                fill={sortBySeverity === sev ? '#1f2937' : '#6b7280'}
                className="capitalize"
              >
                {SEVERITY_LABELS[sev]}
              </text>
            </g>
          );
        })}
        {/* Total header */}
        <text
          x={ROW_LABEL_W + cols * (CELL_W + PADDING) + TOTAL_COL_W / 2}
          y={COL_LABEL_H / 2 + 5}
          textAnchor="middle"
          dominantBaseline="central"
          fontSize={12}
          fontWeight={700}
          fill="#1f2937"
        >
          Total
        </text>

        {/* Rows */}
        {assetTypes.map((type, ri) => {
          const y = COL_LABEL_H + ri * (CELL_H + PADDING);
          return (
            <g key={type}>
              {/* Row label */}
              <text
                x={ROW_LABEL_W - PADDING}
                y={y + CELL_H / 2}
                textAnchor="end"
                dominantBaseline="central"
                fontSize={11}
                fill="#374151"
                className="capitalize"
              >
                {type.replace(/_/g, ' ')}
              </text>

              {/* Cells */}
              {SEVERITIES.map((sev, ci) => {
                const x = ROW_LABEL_W + ci * (CELL_W + PADDING);
                const cell = getCell(type, sev);
                return (
                  <HeatmapSvgCell
                    key={sev}
                    cell={cell}
                    maxForSeverity={maxPerSeverity[sev]}
                    x={x}
                    y={y}
                    width={CELL_W}
                    height={CELL_H}
                  />
                );
              })}

              {/* Row total */}
              <text
                x={ROW_LABEL_W + cols * (CELL_W + PADDING) + TOTAL_COL_W / 2}
                y={y + CELL_H / 2}
                textAnchor="middle"
                dominantBaseline="central"
                fontSize={12}
                fontWeight={700}
                fill="#1f2937"
              >
                {rowTotals[ri]}
              </text>
            </g>
          );
        })}

        {/* Column totals row */}
        {(() => {
          const y = COL_LABEL_H + rows * (CELL_H + PADDING);
          return (
            <g>
              <text
                x={ROW_LABEL_W - PADDING}
                y={y + CELL_H / 2}
                textAnchor="end"
                dominantBaseline="central"
                fontSize={12}
                fontWeight={700}
                fill="#1f2937"
              >
                Total
              </text>
              {SEVERITIES.map((sev, ci) => {
                const x = ROW_LABEL_W + ci * (CELL_W + PADDING);
                return (
                  <text
                    key={sev}
                    x={x + CELL_W / 2}
                    y={y + CELL_H / 2}
                    textAnchor="middle"
                    dominantBaseline="central"
                    fontSize={12}
                    fontWeight={700}
                    fill="#1f2937"
                  >
                    {colTotals[ci]}
                  </text>
                );
              })}
              {/* Grand total */}
              <text
                x={ROW_LABEL_W + cols * (CELL_W + PADDING) + TOTAL_COL_W / 2}
                y={y + CELL_H / 2}
                textAnchor="middle"
                dominantBaseline="central"
                fontSize={13}
                fontWeight={700}
                fill="#111827"
              >
                {data.total_vulnerabilities}
              </text>
            </g>
          );
        })()}
      </svg>
    </div>
  );
}
