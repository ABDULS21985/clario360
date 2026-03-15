'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { type ColumnDef } from '@tanstack/react-table';
import { LineChart, Line, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { TrendingUp } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { enterpriseApi } from '@/lib/enterprise';
import { SectionCard } from '@/components/suites/section-card';
import { Badge } from '@/components/ui/badge';
import { KpiCard } from '@/components/shared/kpi-card';
import type { VisusKPIDefinition } from '@/types/suites';

export default function VisusKpisPage() {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const { tableProps, data } = useDataTable<VisusKPIDefinition>({
    queryKey: 'visus-kpis',
    fetchFn: (params) => enterpriseApi.visus.listKpis(params),
    defaultPageSize: 25,
    defaultSort: { column: 'name', direction: 'asc' },
  });
  const selected = selectedId ?? data[0]?.id ?? null;
  const detailQuery = useQuery({
    queryKey: ['visus-kpi-detail', selected],
    queryFn: () => enterpriseApi.visus.getKpi(selected!),
    enabled: Boolean(selected),
  });

  const columns: ColumnDef<VisusKPIDefinition>[] = [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'KPI',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.name}</p>
          <p className="text-xs text-muted-foreground">{row.original.description}</p>
        </div>
      ),
    },
    {
      id: 'suite',
      accessorKey: 'suite',
      header: 'Suite',
      cell: ({ row }) => <Badge variant="outline">{row.original.suite}</Badge>,
    },
    {
      id: 'last_value',
      accessorKey: 'last_value',
      header: 'Latest',
      cell: ({ row }) => <span className="text-sm">{row.original.last_value ?? '—'}</span>,
    },
    {
      id: 'last_status',
      accessorKey: 'last_status',
      header: 'Status',
      cell: ({ row }) => <Badge variant={statusVariant(row.original.last_status)}>{row.original.last_status ?? 'unknown'}</Badge>,
    },
  ];

  const history = detailQuery.data?.history ?? [];
  const definition = detailQuery.data?.definition;

  return (
    <PermissionRedirect permission="visus:read">
      <div className="space-y-6">
        <PageHeader title="KPIs" description="Executive KPI catalogue, latest values, and history." />
        <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
          <DataTable
            {...tableProps}
            columns={columns}
            onRowClick={(row) => setSelectedId(row.id)}
            emptyState={{
              icon: TrendingUp,
              title: 'No KPIs found',
              description: 'No KPI definitions are configured for this tenant.',
            }}
          />
          <SectionCard title={definition?.name ?? 'KPI detail'} description={definition?.description ?? 'Select a KPI to inspect its latest history.'}>
            {definition ? (
              <div className="space-y-4">
                <div className="grid gap-4 sm:grid-cols-2">
                  <KpiCard title="Latest Value" value={definition.last_value ?? 0} />
                  <KpiCard title="Target" value={definition.target_value ?? '—'} />
                </div>
                <div className="rounded-xl border p-4">
                  <p className="mb-3 text-sm font-medium">History</p>
                  <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={history.map((point) => ({ at: point.created_at.slice(5, 10), value: point.value }))}>
                        <XAxis dataKey="at" />
                        <YAxis />
                        <Tooltip />
                        <Line type="monotone" dataKey="value" stroke="#0f766e" strokeWidth={2} dot={false} />
                      </LineChart>
                    </ResponsiveContainer>
                  </div>
                </div>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">Select a KPI to inspect its current status.</p>
            )}
          </SectionCard>
        </div>
      </div>
    </PermissionRedirect>
  );
}

function statusVariant(status: string | null | undefined): 'default' | 'warning' | 'destructive' | 'outline' {
  if (status === 'warning') return 'warning';
  if (status === 'critical') return 'destructive';
  if (status === 'normal') return 'default';
  return 'outline';
}
