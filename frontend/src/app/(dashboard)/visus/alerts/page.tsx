'use client';

import { useState } from 'react';
import { type ColumnDef } from '@tanstack/react-table';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Bell } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { enterpriseApi } from '@/lib/enterprise';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { showApiError, showSuccess } from '@/lib/toast';
import type { VisusExecutiveAlert } from '@/types/suites';
import { DismissAlertDialog } from './_components/dismiss-alert-dialog';

export default function VisusAlertsPage() {
  const queryClient = useQueryClient();
  const [dismissTarget, setDismissTarget] = useState<VisusExecutiveAlert | null>(null);
  const { tableProps } = useDataTable<VisusExecutiveAlert>({
    queryKey: 'visus-alerts',
    fetchFn: (params) => enterpriseApi.visus.listAlerts(params),
    defaultPageSize: 25,
    defaultSort: { column: 'created_at', direction: 'desc' },
  });
  const statsQuery = useQuery({
    queryKey: ['visus-alert-stats'],
    queryFn: () => enterpriseApi.visus.getAlertStats(),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, status, dismiss_reason }: { id: string; status: VisusExecutiveAlert['status']; dismiss_reason?: string }) =>
      enterpriseApi.visus.updateAlertStatus(id, { status, dismiss_reason }),
    onSuccess: async () => {
      showSuccess('Alert updated.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['visus-alerts'] }),
        queryClient.invalidateQueries({ queryKey: ['visus-alert-stats'] }),
      ]);
    },
    onError: showApiError,
  });

  const handleDismissConfirm = (id: string, dismissReason?: string) => {
    updateMutation.mutate({ id, status: 'dismissed', dismiss_reason: dismissReason });
    setDismissTarget(null);
  };

  const columns: ColumnDef<VisusExecutiveAlert>[] = [
    {
      id: 'title',
      accessorKey: 'title',
      header: 'Alert',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.title}</p>
          <p className="text-xs text-muted-foreground">{row.original.description}</p>
        </div>
      ),
    },
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      enableSorting: true,
      cell: ({ row }) => <Badge variant={severityVariant(row.original.severity)}>{row.original.severity}</Badge>,
    },
    {
      id: 'category',
      accessorKey: 'category',
      header: 'Category',
      enableSorting: true,
      cell: ({ row }) => <Badge variant="outline">{row.original.category}</Badge>,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <Badge variant="outline">{row.original.status}</Badge>,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={() => updateMutation.mutate({ id: row.original.id, status: 'acknowledged' })}>
            Acknowledge
          </Button>
          <Button variant="ghost" size="sm" onClick={() => setDismissTarget(row.original)}>
            Dismiss
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PermissionRedirect permission="visus:read">
      <div className="space-y-6">
        <PageHeader title="Alerts" description="Executive alerts aggregated across all suites." />
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          {Object.entries(statsQuery.data?.by_severity ?? {}).map(([severity, count]) => (
            <div key={severity} className="rounded-xl border bg-card px-4 py-4">
              <p className="text-sm text-muted-foreground capitalize">{severity}</p>
              <p className="mt-1 text-2xl font-semibold">{count}</p>
            </div>
          ))}
        </div>
        <DataTable
          {...tableProps}
          columns={columns}
          emptyState={{
            icon: Bell,
            title: 'No alerts',
            description: 'No executive alerts are currently open.',
          }}
        />
      </div>
      <DismissAlertDialog
        alert={dismissTarget}
        open={dismissTarget !== null}
        onOpenChange={(open) => { if (!open) setDismissTarget(null); }}
        onConfirm={handleDismissConfirm}
      />
    </PermissionRedirect>
  );
}

function severityVariant(severity: string): 'default' | 'warning' | 'destructive' | 'outline' {
  if (severity === 'critical' || severity === 'high') return 'destructive';
  if (severity === 'medium') return 'warning';
  if (severity === 'low') return 'default';
  return 'outline';
}
