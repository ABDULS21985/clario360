'use client';

import type { RiskHeatmapData, CyberSeverity } from '@/types/cyber';

const SEVERITIES: CyberSeverity[] = ['critical', 'high', 'medium', 'low'];

interface HeatmapSummaryTableProps {
  data: RiskHeatmapData;
}

function computeInsights(data: RiskHeatmapData) {
  const totals: Record<string, number> = {};
  const typeTotals: Record<string, number> = {};
  const typeCounts: Record<string, number> = {}; // total assets of type

  let maxCount = 0;
  let maxCell: { assetType: string; severity: CyberSeverity; count: number } | null = null;

  for (const cell of data.cells) {
    totals[cell.asset_type] = (totals[cell.asset_type] ?? 0) + cell.count;
    typeTotals[cell.asset_type] = (typeTotals[cell.asset_type] ?? 0) + cell.count;
    typeCounts[cell.asset_type] = cell.total_assets_of_type;
    if (cell.count > maxCount) {
      maxCount = cell.count;
      maxCell = { assetType: cell.asset_type, severity: cell.severity, count: cell.count };
    }
  }

  const mostVulnerableType = Object.entries(typeTotals).sort((a, b) => b[1] - a[1])[0];

  // Highest vuln-to-asset ratio
  const ratios = Object.entries(typeTotals).map(([type, total]) => ({
    type,
    ratio: typeCounts[type] > 0 ? total / typeCounts[type] : 0,
  }));
  const highestRatio = ratios.sort((a, b) => b.ratio - a.ratio)[0];

  return { maxCell, mostVulnerableType, highestRatio };
}

export function HeatmapSummaryTable({ data }: HeatmapSummaryTableProps) {
  const { maxCell, mostVulnerableType, highestRatio } = computeInsights(data);

  const assetTypes = data.asset_types;
  const severityTotals = SEVERITIES.map((sev) =>
    data.cells.filter((c) => c.severity === sev).reduce((sum, c) => sum + c.count, 0),
  );

  const rowTotals = assetTypes.map((type) =>
    data.cells.filter((c) => c.asset_type === type).reduce((sum, c) => sum + c.count, 0),
  );

  return (
    <div className="space-y-4">
      {/* Summary insight sentences */}
      <div className="rounded-xl border bg-card p-4 space-y-1.5">
        <h4 className="text-sm font-semibold">Key Insights</h4>
        {maxCell && (
          <p className="text-sm text-muted-foreground">
            <span className="font-medium text-foreground">Highest risk:</span>{' '}
            {maxCell.count} <span className="capitalize">{maxCell.severity}</span> vulnerabilities on{' '}
            {maxCell.assetType.replace(/_/g, ' ')} assets.
          </p>
        )}
        {mostVulnerableType && (
          <p className="text-sm text-muted-foreground">
            <span className="font-medium text-foreground">Most vulnerable asset type:</span>{' '}
            {mostVulnerableType[0].replace(/_/g, ' ')} with {mostVulnerableType[1]} open vulnerabilities (
            {data.total_vulnerabilities > 0
              ? ((mostVulnerableType[1] / data.total_vulnerabilities) * 100).toFixed(0)
              : 0}
            % of total).
          </p>
        )}
        {highestRatio && highestRatio.ratio > 0 && (
          <p className="text-sm text-muted-foreground">
            <span className="font-medium text-foreground">Least covered:</span>{' '}
            {highestRatio.type.replace(/_/g, ' ')} assets have the highest vuln-to-asset ratio (
            {highestRatio.ratio.toFixed(1)}).
          </p>
        )}
      </div>

      {/* Totals table */}
      <div className="overflow-x-auto rounded-xl border bg-card">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">Asset Type</th>
              {SEVERITIES.map((s) => (
                <th key={s} className="px-4 py-2 text-center text-xs font-medium capitalize text-muted-foreground">
                  {s}
                </th>
              ))}
              <th className="px-4 py-2 text-center text-xs font-bold text-foreground">Total</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {assetTypes.map((type, i) => (
              <tr key={type} className="hover:bg-muted/30">
                <td className="px-4 py-2 font-medium capitalize">{type.replace(/_/g, ' ')}</td>
                {SEVERITIES.map((sev) => {
                  const cell = data.cells.find(
                    (c) => c.asset_type === type && c.severity === sev,
                  );
                  return (
                    <td key={sev} className="px-4 py-2 text-center tabular-nums">
                      {cell?.count ?? 0}
                    </td>
                  );
                })}
                <td className="px-4 py-2 text-center font-bold tabular-nums">{rowTotals[i]}</td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr className="border-t bg-muted/50">
              <td className="px-4 py-2 font-bold">Total</td>
              {severityTotals.map((t, i) => (
                <td key={i} className="px-4 py-2 text-center font-bold tabular-nums">{t}</td>
              ))}
              <td className="px-4 py-2 text-center font-bold tabular-nums">
                {data.total_vulnerabilities}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  );
}
