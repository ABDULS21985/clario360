'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ColumnDef } from '@tanstack/react-table';
import { Plus, Shield } from 'lucide-react';
import { toast } from 'sonner';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PermissionGate } from '@/components/auth/permission-gate';
import { BrandAbuseFormDialog } from '@/components/cyber/cti/brand-abuse-form-dialog';
import { IOCValueDisplay } from '@/components/cyber/cti/ioc-value-display';
import { CTIKPIStatCard } from '@/components/cyber/cti/kpi-stat-card';
import { MonitoredBrandsManager } from '@/components/cyber/cti/monitored-brands-manager';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { CTIStatusBadge } from '@/components/cyber/cti/status-badge';
import { ExportMenu } from '@/components/cyber/export-menu';
import { DataTable } from '@/components/shared/data-table/data-table';
import { selectColumn } from '@/components/shared/data-table/columns/common-columns';
import { Button } from '@/components/ui/button';
import { useDataTable } from '@/hooks/use-data-table';
import { useAuth } from '@/hooks/use-auth';
import {
  fetchBrandAbuseIncidents,
  fetchMonitoredBrands,
  flattenBrandAbuseFetchParams,
  updateTakedownStatus,
} from '@/lib/cti-api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import {
  CTI_RISK_LEVEL_OPTIONS,
  CTI_TAKEDOWN_STATUS_OPTIONS,
} from '@/lib/cti-utils';
import { timeAgo } from '@/lib/utils';
import type { CTIBrandAbuseIncident } from '@/types/cti';
import type { BulkAction, FetchParams, FilterConfig, RowAction } from '@/types/table';

function fetchBrandAbuseRows(params: FetchParams) {
  return fetchBrandAbuseIncidents(flattenBrandAbuseFetchParams(params));
}

export default function CTIBrandAbusePage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { hasPermission } = useAuth();
  const canWrite = hasPermission('cyber:write');
  const [formOpen, setFormOpen] = useState(false);
  const [brandsManagerOpen, setBrandsManagerOpen] = useState(false);
  const [editingIncident, setEditingIncident] = useState<CTIBrandAbuseIncident | null>(null);

  const { tableProps, refetch } = useDataTable<CTIBrandAbuseIncident>({
    fetchFn: fetchBrandAbuseRows,
    queryKey: 'cti-brand-abuse',
    defaultPageSize: 25,
    defaultSort: { column: 'last_detected_at', direction: 'desc' },
  });

  const brandsQuery = useQuery({
    queryKey: ['cti-brand-abuse-brands'],
    queryFn: fetchMonitoredBrands,
  });
  const kpisQuery = useQuery({
    queryKey: ['cti-brand-abuse-kpis'],
    queryFn: async () => {
      const [critical, total, pending, takenDown] = await Promise.all([
        fetchBrandAbuseIncidents({ page: 1, per_page: 1, risk_level: 'critical' }),
        fetchBrandAbuseIncidents({ page: 1, per_page: 1 }),
        fetchBrandAbuseIncidents({ page: 1, per_page: 1, takedown_status: ['reported', 'takedown_requested'] }),
        fetchBrandAbuseIncidents({ page: 1, per_page: 1, takedown_status: 'taken_down' }),
      ]);

      return {
        monitoredBrands: brandsQuery.data?.length ?? 0,
        criticalAlerts: critical.meta.total,
        totalAlerts: total.meta.total,
        pendingTakedowns: pending.meta.total,
        takenDown: takenDown.meta.total,
      };
    },
    enabled: brandsQuery.isSuccess,
  });

  const filters = useMemo<FilterConfig[]>(() => [
    {
      key: 'brand_id',
      label: 'Brand',
      type: 'select',
      options: (brandsQuery.data ?? []).map((brand) => ({ label: brand.brand_name, value: brand.id })),
    },
    {
      key: 'risk_level',
      label: 'Risk',
      type: 'multi-select',
      options: CTI_RISK_LEVEL_OPTIONS,
    },
    {
      key: 'takedown_status',
      label: 'Takedown',
      type: 'multi-select',
      options: CTI_TAKEDOWN_STATUS_OPTIONS,
    },
    {
      key: 'abuse_type',
      label: 'Abuse Type',
      type: 'text',
      placeholder: 'Filter by abuse type',
    },
  ], [brandsQuery.data]);

  const statusMutation = useMutation({
    mutationFn: async ({ ids, status }: { ids: string[]; status: string }) => {
      await Promise.all(ids.map((id) => updateTakedownStatus(id, status)));
    },
    onSuccess: async (_, variables) => {
      await queryClient.invalidateQueries({ queryKey: ['cti-brand-abuse'] });
      toast.success(`${variables.ids.length} incident${variables.ids.length === 1 ? '' : 's'} moved to ${variables.status}`);
      refetch();
    },
    onError: () => toast.error('Failed to update takedown status'),
  });

  const bulkActions = useMemo<BulkAction[]>(() => {
    if (!canWrite) {
      return [];
    }

    return [
      {
        label: 'Mark Reported',
        onClick: async (selectedIds) => statusMutation.mutateAsync({ ids: selectedIds, status: 'reported' }),
      },
      {
        label: 'Request Takedown',
        onClick: async (selectedIds) => statusMutation.mutateAsync({ ids: selectedIds, status: 'takedown_requested' }),
      },
      {
        label: 'Mark Taken Down',
        onClick: async (selectedIds) => statusMutation.mutateAsync({ ids: selectedIds, status: 'taken_down' }),
      },
      {
        label: 'Set Monitoring',
        onClick: async (selectedIds) => statusMutation.mutateAsync({ ids: selectedIds, status: 'monitoring' }),
      },
      {
        label: 'Mark False Positive',
        onClick: async (selectedIds) => statusMutation.mutateAsync({ ids: selectedIds, status: 'false_positive' }),
      },
    ];
  }, [canWrite, statusMutation]);

  const rowActions = useMemo<RowAction<CTIBrandAbuseIncident>[]>(() => {
    const baseActions: RowAction<CTIBrandAbuseIncident>[] = [
      {
        label: 'View Incident',
        onClick: (incident) => router.push(`${ROUTES.CYBER_CTI_BRAND_ABUSE}/${incident.id}`),
      },
    ];

    if (!canWrite) {
      return baseActions;
    }

    return [
      ...baseActions,
      {
        label: 'Edit Incident',
        onClick: (incident) => setEditingIncident(incident),
      },
      {
        label: 'Mark Taken Down',
        onClick: (incident) => statusMutation.mutate({ ids: [incident.id], status: 'taken_down' }),
        hidden: (incident) => incident.takedown_status === 'taken_down',
      },
      {
        label: 'Request Takedown',
        onClick: (incident) => statusMutation.mutate({ ids: [incident.id], status: 'takedown_requested' }),
        hidden: (incident) => incident.takedown_status === 'takedown_requested' || incident.takedown_status === 'taken_down',
      },
      {
        label: 'Mark False Positive',
        onClick: (incident) => statusMutation.mutate({ ids: [incident.id], status: 'false_positive' }),
        hidden: (incident) => incident.takedown_status === 'false_positive',
      },
    ];
  }, [canWrite, router, statusMutation]);

  const columns = useMemo<ColumnDef<CTIBrandAbuseIncident>[]>(() => {
    const baseColumns: ColumnDef<CTIBrandAbuseIncident>[] = [
      {
        accessorKey: 'brand_name',
        header: 'Brand',
        cell: ({ row }) => (
          <button className="min-w-0 text-left" onClick={() => router.push(`${ROUTES.CYBER_CTI_BRAND_ABUSE}/${row.original.id}`)}>
            <p className="truncate font-medium text-foreground hover:underline">{row.original.brand_name}</p>
            <p className="text-xs text-muted-foreground">{row.original.abuse_type.replaceAll('_', ' ')}</p>
          </button>
        ),
        size: 200,
      },
      {
        accessorKey: 'malicious_domain',
        header: 'Domain',
        cell: ({ row }) => <IOCValueDisplay type="domain" value={row.original.malicious_domain} className="border-0 bg-transparent p-0" copyable={false} />,
        size: 280,
      },
      {
        accessorKey: 'risk_level',
        header: 'Risk',
        cell: ({ row }) => <CTISeverityBadge severity={row.original.risk_level} />,
        size: 110,
      },
      {
        accessorKey: 'takedown_status',
        header: 'Takedown',
        cell: ({ row }) => <CTIStatusBadge status={row.original.takedown_status} type="takedown" />,
        size: 160,
      },
      {
        accessorKey: 'detection_count',
        header: 'Detections',
        cell: ({ row }) => <span className="text-sm tabular-nums">{row.original.detection_count.toLocaleString()}</span>,
        size: 100,
      },
      {
        accessorKey: 'last_detected_at',
        header: 'Last Detected',
        cell: ({ row }) => <span className="text-sm text-muted-foreground">{timeAgo(row.original.last_detected_at)}</span>,
        size: 140,
      },
    ];

    return canWrite ? [selectColumn<CTIBrandAbuseIncident>(), ...baseColumns] : baseColumns;
  }, [canWrite, router]);

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Brand Abuse Monitoring"
          description="Track malicious domains, takedown progress, and the monitored-brand catalogue that powers executive CTI reporting."
          actions={
            <div className="flex flex-wrap items-center gap-2">
              <ExportMenu
                entityType="cti-brand-abuse"
                baseUrl={API_ENDPOINTS.CTI_BRAND_ABUSE}
                currentFilters={{ ...tableProps.activeFilters, search: tableProps.searchValue ?? '' }}
                totalCount={tableProps.totalRows}
                enabledFormats={['csv', 'json']}
              />
              <PermissionGate permission="cyber:write">
                <Button variant="outline" size="sm" onClick={() => setBrandsManagerOpen(true)}>
                  Manage Brands
                </Button>
                <Button size="sm" onClick={() => setFormOpen(true)}>
                  <Plus className="mr-1.5 h-3.5 w-3.5" />
                  New Incident
                </Button>
              </PermissionGate>
            </div>
          }
        />

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
          <CTIKPIStatCard label="Monitored Brands" value={kpisQuery.data?.monitoredBrands ?? 0} subtitle="Protected brands" />
          <CTIKPIStatCard label="Critical Alerts" value={kpisQuery.data?.criticalAlerts ?? 0} subtitle="Immediate action" color="#FF3B5C" />
          <CTIKPIStatCard label="Total Incidents" value={kpisQuery.data?.totalAlerts ?? 0} subtitle="Tracked abuse cases" />
          <CTIKPIStatCard label="Pending Takedowns" value={kpisQuery.data?.pendingTakedowns ?? 0} subtitle="Reported or requested" />
          <CTIKPIStatCard label="Taken Down" value={kpisQuery.data?.takenDown ?? 0} subtitle="Remediated incidents" color="#4ADE80" />
        </div>

        <DataTable
          {...tableProps}
          columns={columns}
          filters={filters}
          enableSelection={canWrite}
          bulkActions={bulkActions}
          rowActions={rowActions}
          getRowId={(row) => row.id}
          searchPlaceholder="Search incidents by brand or malicious domain"
          emptyState={{
            icon: Shield,
            title: 'No brand abuse incidents found',
            description: 'No incidents match the current filters.',
            action: canWrite
              ? { label: 'Create Incident', onClick: () => setFormOpen(true), icon: Plus }
              : undefined,
          }}
          onRowClick={(row) => router.push(`${ROUTES.CYBER_CTI_BRAND_ABUSE}/${row.id}`)}
        />
      </div>

      <BrandAbuseFormDialog
        open={formOpen || Boolean(editingIncident)}
        onOpenChange={(open) => {
          if (!open) {
            setFormOpen(false);
            setEditingIncident(null);
          }
        }}
        incident={editingIncident}
        onSuccess={(incident) => {
          setFormOpen(false);
          setEditingIncident(null);
          if (incident) {
            router.push(`${ROUTES.CYBER_CTI_BRAND_ABUSE}/${incident.id}`);
          }
          refetch();
        }}
      />

      <MonitoredBrandsManager open={brandsManagerOpen} onOpenChange={setBrandsManagerOpen} onUpdated={() => void brandsQuery.refetch()} />
    </PermissionRedirect>
  );
}
