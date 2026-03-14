'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { FileUp, Fingerprint, Plus, Search } from 'lucide-react';
import { toast } from 'sonner';
import { PermissionGate } from '@/components/auth/permission-gate';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { useAuth } from '@/hooks/use-auth';
import { apiDelete, apiGet, apiPut } from '@/lib/api';
import {
  exportIndicatorsAsCsv,
  exportIndicatorsAsJson,
  exportIndicatorsAsStix,
  INDICATOR_SOURCE_OPTIONS,
} from '@/lib/cyber-indicators';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { INDICATOR_TYPE_OPTIONS } from '@/lib/cyber-threats';
import type { PaginatedResponse } from '@/types/api';
import type { BulkAction, FetchParams, FilterConfig } from '@/types/table';
import type { IndicatorStats as IndicatorStatsType, ThreatIndicator } from '@/types/cyber';
import { Button } from '@/components/ui/button';

import { AddIndicatorDialog } from './_components/add-indicator-dialog';
import { BulkImportDialog } from './_components/bulk-import-dialog';
import { getIndicatorColumns } from './_components/indicator-columns';
import { IndicatorDetailPanel } from './_components/indicator-detail-panel';
import { IndicatorStats } from './_components/indicator-stats';
import { IndicatorCheckDialog } from '../threats/_components/indicator-check-dialog';

const INDICATOR_FILTERS: FilterConfig[] = [
  {
    key: 'type',
    label: 'Type',
    type: 'multi-select',
    options: INDICATOR_TYPE_OPTIONS.map((option) => ({
      label: option.label,
      value: option.value,
    })),
  },
  {
    key: 'source',
    label: 'Source',
    type: 'multi-select',
    options: INDICATOR_SOURCE_OPTIONS.map((option) => ({
      label: option.label,
      value: option.value,
    })),
  },
  {
    key: 'severity',
    label: 'Severity',
    type: 'multi-select',
    options: [
      { label: 'Critical', value: 'critical' },
      { label: 'High', value: 'high' },
      { label: 'Medium', value: 'medium' },
      { label: 'Low', value: 'low' },
    ],
  },
  {
    key: 'active',
    label: 'Active',
    type: 'select',
    options: [
      { label: 'Active Only', value: 'true' },
      { label: 'Inactive Only', value: 'false' },
    ],
  },
  {
    key: 'linked',
    label: 'Threat Link',
    type: 'select',
    options: [
      { label: 'Linked', value: 'true' },
      { label: 'Unlinked', value: 'false' },
    ],
  },
  {
    key: 'confidence_range',
    label: 'Confidence',
    type: 'range',
    min: 0,
    max: 100,
    step: 5,
    valueSuffix: '%',
  },
];

async function fetchIndicators(params: FetchParams): Promise<PaginatedResponse<ThreatIndicator>> {
  return apiGet<PaginatedResponse<ThreatIndicator>>(
    API_ENDPOINTS.CYBER_INDICATORS,
    flattenIndicatorParams(params),
  );
}

export default function CyberIndicatorsPage() {
  const router = useRouter();
  const { hasPermission } = useAuth();
  const canWrite = hasPermission('cyber:write');

  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [tableResetKey, setTableResetKey] = useState(0);
  const [detailIndicator, setDetailIndicator] = useState<ThreatIndicator | null>(null);
  const [editorIndicator, setEditorIndicator] = useState<ThreatIndicator | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [bulkImportOpen, setBulkImportOpen] = useState(false);
  const [indicatorCheckOpen, setIndicatorCheckOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<ThreatIndicator | null>(null);

  const { tableProps, refetch } = useDataTable<ThreatIndicator>({
    fetchFn: fetchIndicators,
    queryKey: 'cyber-indicators',
    defaultPageSize: 25,
    defaultSort: { column: 'last_seen_at', direction: 'desc' },
    wsTopics: ['cyber.indicator.created', 'cyber.indicator.updated'],
  });

  const statsQuery = useQuery({
    queryKey: ['cyber-indicator-stats'],
    queryFn: () => apiGet<{ data: IndicatorStatsType }>(API_ENDPOINTS.CYBER_INDICATORS_STATS),
  });

  const selectedIndicators = useMemo(
    () => tableProps.data.filter((item) => selectedIds.includes(item.id)),
    [selectedIds, tableProps.data],
  );

  const columns = useMemo(
    () => getIndicatorColumns({
      canWrite,
      onView: setDetailIndicator,
      onEdit: (indicator) => {
        setEditorIndicator(indicator);
        setEditorOpen(true);
      },
      onDelete: setDeleteTarget,
      onToggleActive: async (indicator, active) => {
        try {
          await apiPut(API_ENDPOINTS.CYBER_INDICATOR_STATUS(indicator.id), { active });
          toast.success(active ? 'Indicator activated' : 'Indicator deactivated');
          await handleMutationComplete();
        } catch (error) {
          toast.error(error instanceof Error ? error.message : 'Unable to update indicator');
        }
      },
      onOpenThreat: (indicator) => {
        if (indicator.threat_id) {
          router.push(`${ROUTES.CYBER_THREATS}/${indicator.threat_id}`);
        }
      },
    }),
    [canWrite, router, tableProps.data],
  );

  const bulkActions = useMemo<BulkAction[]>(() => {
    if (!canWrite) {
      return [];
    }

    return [
      {
        label: 'Activate Selected',
        onClick: async (ids) => {
          await Promise.all(ids.map((id) => apiPut(API_ENDPOINTS.CYBER_INDICATOR_STATUS(id), { active: true })));
          toast.success(`${ids.length} indicators activated`);
          await handleMutationComplete();
        },
      },
      {
        label: 'Deactivate Selected',
        onClick: async (ids) => {
          await Promise.all(ids.map((id) => apiPut(API_ENDPOINTS.CYBER_INDICATOR_STATUS(id), { active: false })));
          toast.success(`${ids.length} indicators deactivated`);
          await handleMutationComplete();
        },
      },
      {
        label: 'Export CSV',
        onClick: async () => {
          exportIndicatorsAsCsv(selectedIndicators);
        },
      },
      {
        label: 'Export JSON',
        onClick: async () => {
          exportIndicatorsAsJson(selectedIndicators);
        },
      },
      {
        label: 'Export STIX',
        onClick: async () => {
          exportIndicatorsAsStix(selectedIndicators);
        },
      },
      {
        label: 'Delete Selected',
        variant: 'destructive',
        onClick: async (ids) => {
          await Promise.all(ids.map((id) => apiDelete(API_ENDPOINTS.CYBER_INDICATOR_DETAIL(id))));
          toast.success(`${ids.length} indicators deleted`);
          await handleMutationComplete();
        },
      },
    ];
  }, [canWrite, selectedIndicators]);

  async function handleMutationComplete() {
    setSelectedIds([]);
    setTableResetKey((value) => value + 1);
    await refetch();
    void statsQuery.refetch();
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="IOC Management"
          description="Validate, enrich, and operationalize indicators across threat hunting, detections, and threat intelligence feed ingestion."
          actions={(
            <div className="flex flex-wrap items-center gap-2">
              <Button variant="outline" onClick={() => setIndicatorCheckOpen(true)}>
                <Search className="mr-2 h-4 w-4" />
                Check Indicators
              </Button>
              <PermissionGate permission="cyber:write">
                <Button variant="outline" onClick={() => setBulkImportOpen(true)}>
                  <FileUp className="mr-2 h-4 w-4" />
                  Bulk Import
                </Button>
                <Button
                  onClick={() => {
                    setEditorIndicator(null);
                    setEditorOpen(true);
                  }}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add Indicator
                </Button>
              </PermissionGate>
            </div>
          )}
        />

        <IndicatorStats stats={statsQuery.data?.data} loading={statsQuery.isLoading} />

        <DataTable
          key={tableResetKey}
          {...tableProps}
          columns={columns}
          filters={INDICATOR_FILTERS}
          searchPlaceholder="Search IOC values, tags, or linked threat context…"
          getRowId={(row) => row.id}
          onRowClick={(row) => setDetailIndicator(row)}
          enableSelection={canWrite}
          onSelectionChange={setSelectedIds}
          bulkActions={bulkActions}
          emptyState={{
            icon: Fingerprint,
            title: 'No indicators found',
            description: 'No indicators match the current filters.',
          }}
        />
      </div>

      <AddIndicatorDialog
        open={editorOpen}
        onOpenChange={setEditorOpen}
        indicator={editorIndicator}
        onSuccess={(indicator) => {
          setEditorIndicator(indicator);
          setDetailIndicator(indicator);
          void handleMutationComplete();
        }}
      />

      <BulkImportDialog
        open={bulkImportOpen}
        onOpenChange={setBulkImportOpen}
        onSuccess={() => {
          void handleMutationComplete();
        }}
      />

      <IndicatorCheckDialog
        open={indicatorCheckOpen}
        onOpenChange={setIndicatorCheckOpen}
      />

      <IndicatorDetailPanel
        open={Boolean(detailIndicator)}
        onOpenChange={(open) => {
          if (!open) {
            setDetailIndicator(null);
          }
        }}
        indicator={detailIndicator}
        onEdit={(indicator) => {
          setEditorIndicator(indicator);
          setEditorOpen(true);
        }}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteTarget(null);
          }
        }}
        title="Delete indicator?"
        description="This removes the indicator from the tenant and stops future matches against it."
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={async () => {
          if (!deleteTarget) {
            return;
          }
          await apiDelete(API_ENDPOINTS.CYBER_INDICATOR_DETAIL(deleteTarget.id));
          toast.success('Indicator deleted');
          if (detailIndicator?.id === deleteTarget.id) {
            setDetailIndicator(null);
          }
          setDeleteTarget(null);
          await handleMutationComplete();
        }}
      />
    </PermissionRedirect>
  );
}

function flattenIndicatorParams(params: FetchParams): Record<string, unknown> {
  const flat: Record<string, unknown> = {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    search: params.search,
  };

  for (const [key, value] of Object.entries(params.filters ?? {})) {
    if (!value) {
      continue;
    }
    if (key === 'confidence_range' && typeof value === 'string') {
      const [minRaw, maxRaw] = value.split(',');
      const min = Number(minRaw);
      const max = Number(maxRaw);
      if (Number.isFinite(min)) {
        flat.min_confidence = min / 100;
      }
      if (Number.isFinite(max)) {
        flat.max_confidence = max / 100;
      }
      continue;
    }
    flat[key] = Array.isArray(value) ? value.join(',') : value;
  }

  return flat;
}
