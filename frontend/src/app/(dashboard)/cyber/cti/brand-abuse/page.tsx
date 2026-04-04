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
import { MonitoredBrandsManager } from '@/components/cyber/cti/monitored-brands-manager';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { CTIStatusBadge } from '@/components/cyber/cti/status-badge';
import { ExportMenu } from '@/components/cyber/export-menu';
import { DataTable } from '@/components/shared/data-table/data-table';
import { selectColumn } from '@/components/shared/data-table/columns/common-columns';
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
        label: 'Set Monitoring',
        onClick: async (selectedIds) => statusMutation.mutateAsync({ ids: selectedIds, status: 'monitoring' }),
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
}'use client';

import { useEffect, useMemo, useState } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { formatDistanceToNow } from 'date-fns';
import { ShieldAlert } from 'lucide-react';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PageHeader } from '@/components/common/page-header';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { CTIStatusBadge } from '@/components/cyber/cti/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import {
  fetchBrandAbuseIncidents,
  fetchMonitoredBrands,
  flattenBrandAbuseFetchParams,
} from '@/lib/cti-api';
import { CTI_BRAND_ABUSE_TYPE_OPTIONS } from '@/lib/cti-utils';
import type { CTIBrandAbuseIncident, CTIMonitoredBrand } from '@/types/cti';
import type { FilterConfig, FetchParams } from '@/types/table';
import type { PaginatedResponse } from '@/types/api';

function fetchBrandRows(params: FetchParams): Promise<PaginatedResponse<CTIBrandAbuseIncident>> {
  return fetchBrandAbuseIncidents(flattenBrandAbuseFetchParams(params));
}

export default function CTIBrandAbusePage() {
  const [brands, setBrands] = useState<CTIMonitoredBrand[]>([]);

  useEffect(() => {
    void fetchMonitoredBrands().then(setBrands).catch(() => {
      setBrands([]);
    });
  }, []);

  const table = useDataTable<CTIBrandAbuseIncident>({
    fetchFn: fetchBrandRows,
    queryKey: 'cti-brand-abuse',
    defaultPageSize: 25,
    defaultSort: { column: 'first_detected_at', direction: 'desc' },
    wsTopics: [
      'com.clario360.cyber.cti.brand-abuse.detected',
      'com.clario360.cyber.cti.brand-abuse.updated',
      'com.clario360.cyber.cti.brand-abuse.takedown-changed',
    ],
  });

  const columns = useMemo<ColumnDef<CTIBrandAbuseIncident>[]>(
    () => [
      {
        accessorKey: 'risk_level',
        header: 'Risk',
        enableSorting: true,
        cell: ({ row }) => <CTISeverityBadge severity={row.original.risk_level} size="sm" />,
      },
      {
        accessorKey: 'malicious_domain',
        header: 'Domain',
        enableSorting: true,
        size: 300,
        cell: ({ row }) => (
          <div className="space-y-1">
            <p className="font-medium">{row.original.malicious_domain}</p>
            <p className="text-xs text-muted-foreground">{row.original.brand_name}</p>
          </div>
        ),
      },
      {
        accessorKey: 'abuse_type',
        header: 'Abuse Type',
        enableSorting: true,
      },
      {
        accessorKey: 'takedown_status',
        header: 'Takedown Status',
        enableSorting: true,
        cell: ({ row }) => <CTIStatusBadge status={row.original.takedown_status} type="takedown" />,
      },
      {
        accessorKey: 'region_label',
        header: 'Region',
        enableSorting: false,
        cell: ({ row }) => row.original.region_label || 'Unknown',
      },
      {
        accessorKey: 'detection_count',
        header: 'Detections',
        enableSorting: true,
        cell: ({ row }) => <span className="font-medium tabular-nums">{row.original.detection_count}</span>,
      },
      {
        accessorKey: 'first_detected_at',
        header: 'First Detected',
        enableSorting: true,
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {formatDistanceToNow(new Date(row.original.first_detected_at), { addSuffix: true })}
          </span>
        ),
      },
    ],
    [],
  );

  const filters = useMemo<FilterConfig[]>(
    () => [
      {
        key: 'brand_id',
        label: 'Brand',
        type: 'select',
        options: brands.map((brand) => ({ label: brand.brand_name, value: brand.id })),
      },
      {
        key: 'risk_level',
        label: 'Risk Level',
        type: 'multi-select',
        options: [
          { label: 'Critical', value: 'critical' },
          { label: 'High', value: 'high' },
          { label: 'Medium', value: 'medium' },
          { label: 'Low', value: 'low' },
        ],
      },
      {
        key: 'abuse_type',
        label: 'Abuse Type',
        type: 'multi-select',
        options: CTI_BRAND_ABUSE_TYPE_OPTIONS,
      },
      {
        key: 'takedown_status',
        label: 'Takedown Status',
        type: 'multi-select',
        options: [
          { label: 'Detected', value: 'detected' },
          { label: 'Reported', value: 'reported' },
          { label: 'Requested', value: 'takedown_requested' },
          { label: 'Taken Down', value: 'taken_down' },
          { label: 'Monitoring', value: 'monitoring' },
          { label: 'False Positive', value: 'false_positive' },
        ],
      },
    ],
    [brands],
  );

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Brand Abuse"
          description="Track malicious domains, impersonation campaigns, and takedown progress."
        />

        <DataTable
          {...table.tableProps}
          columns={columns}
          filters={filters}
          searchPlaceholder="Search brand abuse incidents…"
          emptyState={{
            icon: ShieldAlert,
            title: 'No brand abuse incidents found',
            description: 'Detected brand impersonation activity will appear here.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
