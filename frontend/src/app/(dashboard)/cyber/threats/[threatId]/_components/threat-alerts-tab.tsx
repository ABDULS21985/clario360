'use client';

import { useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import type { ColumnDef } from '@tanstack/react-table';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDateTime } from '@/lib/utils';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { alertStatusConfig } from '@/lib/status-configs';
import { AlertTriangle } from 'lucide-react';
import type { CyberAlert } from '@/types/cyber';

interface ThreatAlertsTabProps {
  threatId: string;
}

export function ThreatAlertsTab({ threatId }: ThreatAlertsTabProps) {
  const router = useRouter();
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['threat-alerts', threatId],
    queryFn: () => apiGet<{ data: CyberAlert[] }>(API_ENDPOINTS.CYBER_THREAT_ALERTS(threatId)),
  });

  const alerts = data?.data ?? [];
  const columns = useMemo<ColumnDef<CyberAlert>[]>(() => [
    {
      id: 'title',
      accessorKey: 'title',
      header: 'Alert',
      cell: ({ row }) => (
        <div className="space-y-1">
          <p className="font-medium">{row.original.title}</p>
          <p className="text-xs text-muted-foreground">{row.original.asset_name ?? row.original.source}</p>
        </div>
      ),
    },
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }) => <SeverityIndicator severity={row.original.severity} showLabel />,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => <StatusBadge status={row.original.status} config={alertStatusConfig} />,
    },
    {
      id: 'confidence_score',
      accessorKey: 'confidence_score',
      header: 'Confidence',
      cell: ({ row }) => (
        <span className="tabular-nums text-sm text-muted-foreground">
          {Math.round((row.original.confidence_score ?? 0) * 100)}%
        </span>
      ),
    },
    {
      id: 'mitre_technique_name',
      accessorKey: 'mitre_technique_name',
      header: 'MITRE Technique',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.mitre_technique_name ?? row.original.mitre_technique_id ?? '—'}
        </span>
      ),
    },
    {
      id: 'created_at',
      accessorKey: 'created_at',
      header: 'Created',
      cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatDateTime(row.original.created_at)}</span>,
    },
  ], []);

  if (isLoading) return <LoadingSkeleton variant="card" />;
  if (error) return <ErrorState message="Failed to load related alerts" onRetry={() => void refetch()} />;

  if (alerts.length === 0) {
    return (
      <EmptyState
        icon={AlertTriangle}
        title="No related alerts"
        description="No alerts currently map back to this threat’s indicators or MITRE techniques."
      />
    );
  }

  return (
    <DataTable
      columns={columns}
      data={alerts}
      totalRows={alerts.length}
      page={1}
      pageSize={Math.max(alerts.length, 1)}
      onPageChange={() => undefined}
      onPageSizeChange={() => undefined}
      onSortChange={() => undefined}
      onRowClick={(row) => router.push(`/cyber/alerts/${row.id}`)}
      enableColumnToggle={false}
    />
  );
}
