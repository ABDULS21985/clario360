'use client';

import Link from 'next/link';
import type { ColumnDef } from '@tanstack/react-table';
import { Bell } from 'lucide-react';

import { DataTable } from '@/components/shared/data-table/data-table';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import { ALERT_STATUS_CONFIG, getAlertStatusVariant } from '@/lib/cyber-alerts';
import { API_ENDPOINTS } from '@/lib/constants';
import { timeAgo } from '@/lib/utils';
import type { PaginatedResponse } from '@/types/api';
import type { CyberAlert } from '@/types/cyber';
import type { FetchParams } from '@/types/table';

const ALERT_COLUMNS: ColumnDef<CyberAlert>[] = [
  {
    id: 'title',
    accessorKey: 'title',
    header: 'Alert',
    cell: ({ row }) => (
      <div className="space-y-1">
        <Link href={`/cyber/alerts/${row.original.id}`} className="font-medium hover:text-emerald-700 hover:underline">
          {row.original.title}
        </Link>
        <p className="text-xs text-muted-foreground">{row.original.rule_name ?? row.original.source}</p>
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
    cell: ({ row }) => (
      <StatusBadge status={row.original.status} config={ALERT_STATUS_CONFIG} variant={getAlertStatusVariant(row.original.status)} />
    ),
  },
  {
    id: 'confidence',
    accessorKey: 'confidence_score',
    header: 'Confidence',
    cell: ({ row }) => <span className="tabular-nums text-sm">{Math.round(row.original.confidence_score * 100)}%</span>,
  },
  {
    id: 'asset',
    header: 'Asset',
    cell: ({ row }) => <span className="text-sm">{row.original.asset_name ?? row.original.asset_ip_address ?? 'Unknown asset'}</span>,
  },
  {
    id: 'created_at',
    accessorKey: 'created_at',
    header: 'Created',
    cell: ({ row }) => <span className="text-sm text-muted-foreground">{timeAgo(row.original.created_at)}</span>,
  },
];

interface RuleAlertsTabProps {
  ruleId: string;
}

export function RuleAlertsTab({ ruleId }: RuleAlertsTabProps) {
  const { tableProps } = useDataTable<CyberAlert>({
    fetchFn: async (params: FetchParams) =>
      apiGet<PaginatedResponse<CyberAlert>>(API_ENDPOINTS.CYBER_ALERTS, {
        rule_id: ruleId,
        page: params.page,
        per_page: params.per_page,
        search: params.search,
        ...(params.filters ?? {}),
      }),
    queryKey: `cyber-rule-alerts-${ruleId}`,
    defaultPageSize: 10,
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['cyber.alert.created', 'cyber.alert.status_changed', 'cyber.alert.escalated'],
  });

  return (
    <DataTable
      columns={ALERT_COLUMNS}
      searchPlaceholder="Search related alerts"
      emptyState={{
        icon: Bell,
        title: 'No related alerts',
        description: 'This rule has not generated any alerts yet.',
      }}
      {...tableProps}
    />
  );
}
