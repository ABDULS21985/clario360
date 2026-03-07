'use client';

import type { MITREHeatmapData } from '@/types/cyber';

interface MitreMiniHeatmapProps {
  data: MITREHeatmapData;
  maxTactics?: number;
  maxTechniques?: number;
}

function heatColor(count: number, max: number): string {
  if (count === 0 || max === 0) return '#f1f5f9'; // slate-100
  const intensity = Math.min(count / max, 1);
  if (intensity < 0.25) return '#fef9c3'; // yellow-100
  if (intensity < 0.5) return '#fde047';  // yellow-400
  if (intensity < 0.75) return '#f97316'; // orange-500
  return '#dc2626'; // red-600
}

export function MitreMiniHeatmap({
  data,
  maxTactics = 5,
  maxTechniques = 8,
}: MitreMiniHeatmapProps) {
  if (!data.cells || data.cells.length === 0) {
    return (
      <div className="flex h-20 items-center justify-center rounded-lg border bg-muted/30 text-xs text-muted-foreground">
        No MITRE data
      </div>
    );
  }

  // Group by tactic, pick top N tactics by alert count
  const tacticMap = new Map<string, { name: string; cells: typeof data.cells; total: number }>();
  for (const cell of data.cells) {
    if (!tacticMap.has(cell.tactic_id)) {
      tacticMap.set(cell.tactic_id, { name: cell.tactic_name, cells: [], total: 0 });
    }
    const entry = tacticMap.get(cell.tactic_id)!;
    entry.cells.push(cell);
    entry.total += cell.alert_count;
  }

  const topTactics = Array.from(tacticMap.entries())
    .sort((a, b) => b[1].total - a[1].total)
    .slice(0, maxTactics);

  const max = data.max_count || 1;

  return (
    <div className="flex gap-1 overflow-x-auto">
      {topTactics.map(([tacticId, tactic]) => {
        const topTechniques = tactic.cells
          .sort((a, b) => b.alert_count - a.alert_count)
          .slice(0, maxTechniques);

        return (
          <div key={tacticId} className="flex flex-col gap-0.5">
            <p className="truncate text-[9px] font-semibold text-muted-foreground" title={tactic.name}>
              {tactic.name.slice(0, 8)}
            </p>
            {topTechniques.map((cell) => (
              <div
                key={cell.technique_id}
                className="h-4 w-10 rounded-sm"
                style={{ backgroundColor: heatColor(cell.alert_count, max) }}
                title={`${cell.technique_id}: ${cell.alert_count} alerts`}
              />
            ))}
          </div>
        );
      })}
    </div>
  );
}
