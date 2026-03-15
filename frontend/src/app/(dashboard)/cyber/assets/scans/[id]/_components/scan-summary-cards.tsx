'use client';

import type { AssetScan } from '@/types/cyber';

interface Props {
  scan: AssetScan;
}

function computeDuration(scan: AssetScan): string {
  if (!scan.started_at) return '—';
  const end = scan.completed_at ? new Date(scan.completed_at) : new Date();
  const start = new Date(scan.started_at);
  const diffMs = end.getTime() - start.getTime();
  if (diffMs < 0) return '—';
  const mins = Math.floor(diffMs / 60000);
  const secs = Math.floor((diffMs % 60000) / 1000);
  if (mins === 0) return `${secs}s`;
  return `${mins}m ${secs}s`;
}

interface StatCardProps {
  value: string;
  label: string;
  valueClassName?: string;
}

function StatCard({ value, label, valueClassName = 'text-foreground' }: StatCardProps) {
  return (
    <div className="rounded-xl border bg-card p-4 text-center">
      <p className={`text-2xl font-bold tabular-nums ${valueClassName}`}>{value}</p>
      <p className="mt-0.5 text-xs text-muted-foreground">{label}</p>
    </div>
  );
}

export function ScanSummaryCards({ scan }: Props) {
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        value={scan.assets_found.toLocaleString()}
        label="Assets Found"
        valueClassName="text-green-600 dark:text-green-400"
      />
      <StatCard
        value={scan.assets_updated.toLocaleString()}
        label="Assets Updated"
        valueClassName="text-blue-600 dark:text-blue-400"
      />
      <StatCard
        value={computeDuration(scan)}
        label="Duration"
      />
      <div className="rounded-xl border bg-card p-4 text-center">
        <p className="truncate text-sm font-semibold text-foreground">{scan.target ?? '—'}</p>
        <p className="mt-0.5 text-xs text-muted-foreground">Target</p>
      </div>
    </div>
  );
}
