'use client';

import { useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { cn } from '@/lib/utils';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { Shield } from 'lucide-react';
import type { MITREHeatmapData } from '@/types/cyber';

interface MitreHeatmapWidgetProps {
  data?: MITREHeatmapData;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
}

function cellColor(count: number, max: number): string {
  if (count === 0) return 'bg-muted hover:bg-muted/80';
  const ratio = count / Math.max(max, 1);
  if (ratio > 0.6) return 'bg-red-700 text-white hover:bg-red-600';
  if (ratio > 0.3) return 'bg-red-500 text-white hover:bg-red-400';
  if (ratio > 0.1) return 'bg-red-200 text-red-900 hover:bg-red-300';
  return 'bg-red-100 text-red-800 hover:bg-red-200';
}

export function MitreHeatmapWidget({ data, loading, error, onRetry }: MitreHeatmapWidgetProps) {
  const router = useRouter();

  const { tactics, techniquesByTactic } = useMemo(() => {
    if (!data?.cells) return { tactics: [], techniquesByTactic: new Map<string, MITREHeatmapData['cells']>() };

    const tacticMap = new Map<string, { id: string; name: string }>();
    const byTactic = new Map<string, MITREHeatmapData['cells']>();

    for (const cell of data.cells) {
      if (!tacticMap.has(cell.tactic_id)) {
        tacticMap.set(cell.tactic_id, { id: cell.tactic_id, name: cell.tactic_name });
        byTactic.set(cell.tactic_id, []);
      }
      byTactic.get(cell.tactic_id)!.push(cell);
    }

    return {
      tactics: Array.from(tacticMap.values()),
      techniquesByTactic: byTactic,
    };
  }, [data]);

  if (loading) return <LoadingSkeleton variant="table-row" count={4} />;
  if (error) return <ErrorState message={error} onRetry={onRetry} />;
  if (!data?.cells || data.cells.length === 0) {
    return (
      <EmptyState
        icon={Shield}
        title="No MITRE data"
        description="No MITRE ATT&CK detections recorded yet."
      />
    );
  }

  return (
    <div className="overflow-x-auto">
      <div className="min-w-max space-y-1">
        <div className="flex gap-1">
          {tactics.map((tactic) => (
            <div
              key={tactic.id}
              className="w-24 truncate text-center text-[10px] font-medium text-muted-foreground"
              title={tactic.name}
            >
              {tactic.name.length > 12 ? tactic.name.slice(0, 10) + '…' : tactic.name}
            </div>
          ))}
        </div>

        {/* Show top techniques per tactic as a grid row */}
        <div className="flex gap-1">
          {tactics.map((tactic) => {
            const cells = techniquesByTactic.get(tactic.id) ?? [];
            return (
              <div key={tactic.id} className="flex flex-col gap-0.5">
                {cells.slice(0, 5).map((cell) => (
                  <button
                    key={cell.technique_id}
                    title={`${cell.technique_name}\nAlerts: ${cell.alert_count}`}
                    onClick={() => router.push(`/cyber/alerts?mitre_technique_id=${cell.technique_id}`)}
                    className={cn(
                      'h-5 w-24 rounded text-[10px] truncate px-1 transition-colors',
                      cellColor(cell.alert_count, data.max_count),
                    )}
                  >
                    {cell.alert_count > 0 ? `${cell.technique_id}: ${cell.alert_count}` : cell.technique_id}
                  </button>
                ))}
              </div>
            );
          })}
        </div>
      </div>
      <p className="mt-2 text-xs text-muted-foreground">
        Showing top 5 techniques per tactic · {data.cells.length} total detections
      </p>
    </div>
  );
}
