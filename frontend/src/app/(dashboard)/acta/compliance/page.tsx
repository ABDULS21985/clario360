'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, Shield } from 'lucide-react';
import { KpiCard } from '@/components/shared/kpi-card';
import { DataTable } from '@/components/shared/data-table/data-table';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { SectionCard } from '@/components/suites/section-card';
import { useDataTable } from '@/hooks/use-data-table';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import { complianceStatusConfig } from '@/lib/status-configs';
import { StatusBadge } from '@/components/shared/status-badge';
import type { ActaComplianceCheck } from '@/types/suites';
import { ActaComplianceBars } from '../_components/acta-compliance-bars';

export default function ActaCompliancePage() {
  const queryClient = useQueryClient();
  const reportQuery = useQuery({
    queryKey: ['acta-compliance-report'],
    queryFn: () => enterpriseApi.acta.getComplianceReport(),
  });
  const scoreQuery = useQuery({
    queryKey: ['acta-compliance-score'],
    queryFn: () => enterpriseApi.acta.getComplianceScore(),
  });
  const { tableProps } = useDataTable<ActaComplianceCheck>({
    queryKey: 'acta-compliance-results',
    fetchFn: (params) => enterpriseApi.acta.listComplianceResults(params),
    defaultPageSize: 25,
    defaultSort: { column: 'checked_at', direction: 'desc' },
  });

  const runMutation = useMutation({
    mutationFn: () => enterpriseApi.acta.runCompliance(),
    onSuccess: async () => {
      showSuccess('Compliance checks completed.', 'Stored findings and scorecards have been refreshed.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['acta-compliance-report'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-compliance-score'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-compliance-results'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-dashboard'] }),
      ]);
    },
    onError: showApiError,
  });

  const columns: ColumnDef<ActaComplianceCheck>[] = [
    {
      id: 'check_name',
      accessorKey: 'check_name',
      header: 'Check',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.check_name}</p>
          <p className="text-xs text-muted-foreground">{row.original.description}</p>
        </div>
      ),
    },
    {
      id: 'check_type',
      accessorKey: 'check_type',
      header: 'Type',
      cell: ({ row }) => (
        <Badge variant="outline" className="capitalize">
          {row.original.check_type.replace(/_/g, ' ')}
        </Badge>
      ),
    },
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }) => <Badge variant={row.original.severity === 'critical' ? 'destructive' : 'outline'}>{row.original.severity}</Badge>,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => <StatusBadge status={row.original.status} config={complianceStatusConfig} size="sm" />,
    },
    {
      id: 'finding',
      accessorKey: 'finding',
      header: 'Finding',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.finding ?? 'No exception found'}
        </span>
      ),
    },
  ];

  const report = reportQuery.data;

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader
          title="Compliance"
          description="Automated governance checks, committee scorecards, and auditable findings."
          actions={
            <Button onClick={() => runMutation.mutate()} disabled={runMutation.isPending}>
              {runMutation.isPending ? 'Running checks…' : 'Run checks'}
            </Button>
          }
        />

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-4">
          <KpiCard title="Compliance Score" value={`${Math.round(scoreQuery.data?.score ?? 0)}%`} icon={Shield} iconColor="text-emerald-600" />
          <KpiCard title="Non-Compliant" value={report?.non_compliant_count ?? 0} icon={AlertTriangle} iconColor="text-rose-600" />
          <KpiCard title="Warnings" value={report?.warning_count ?? 0} icon={AlertTriangle} iconColor="text-amber-600" />
          <KpiCard title="Checks Logged" value={report?.results.length ?? 0} icon={Shield} iconColor="text-sky-600" />
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1fr_1.1fr]">
          <SectionCard title="Latest Scorecard" description="Current committee-level compliance distribution.">
            <ActaComplianceBars items={report?.by_committee ?? []} />
          </SectionCard>
          <SectionCard title="Check Distribution" description="Counts by compliance status from the last report run.">
            <div className="space-y-4">
              {Object.entries(report?.by_status ?? {}).map(([status, count]) => (
                <div key={status}>
                  <div className="mb-1 flex items-center justify-between text-sm">
                    <span className="capitalize text-muted-foreground">{status.replace(/_/g, ' ')}</span>
                    <span className="font-medium">{count}</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-primary"
                      style={{
                        width: `${Math.min((count / Math.max(report?.results.length ?? 1, 1)) * 100, 100)}%`,
                      }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </SectionCard>
        </div>

        <DataTable
          {...tableProps}
          columns={columns}
          emptyState={{
            icon: Shield,
            title: 'No compliance findings',
            description: 'Run compliance checks to populate auditable results.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
