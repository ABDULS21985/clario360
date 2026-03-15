'use client';

import { Button } from '@/components/ui/button';
import { type ContradictionStats } from '@/lib/data-suite';

interface ContradictionStatBarProps {
  stats: ContradictionStats;
  activeStatus?: string | string[];
  onFilterStatus: (status?: string) => void;
}

const STATUS_ORDER = ['detected', 'investigating', 'resolved', 'accepted', 'false_positive'] as const;

export function ContradictionStatBar({
  stats,
  activeStatus,
  onFilterStatus,
}: ContradictionStatBarProps) {
  const active = Array.isArray(activeStatus) ? activeStatus[0] : activeStatus;

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-5">
      {STATUS_ORDER.map((status) => (
        <button
          key={status}
          type="button"
          className={`rounded-lg border p-4 text-left transition-colors ${active === status ? 'border-primary bg-primary/5' : 'bg-card'}`}
          onClick={() => onFilterStatus(active === status ? undefined : status)}
        >
          <div className="text-xs uppercase tracking-wide text-muted-foreground">{status.replace(/_/g, ' ')}</div>
          <div className="mt-1 text-2xl font-semibold">{stats.by_status[status] ?? 0}</div>
        </button>
      ))}
    </div>
  );
}
