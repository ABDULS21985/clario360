'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, CheckCheck, GitMerge, ShieldAlert, UserCheck } from 'lucide-react';
import { toast } from 'sonner';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PermissionGate } from '@/components/auth/permission-gate';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { useAuth } from '@/hooks/use-auth';
import { apiGet, apiPut } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import type { PaginatedResponse } from '@/types/api';
import type { BulkAction, FetchParams } from '@/types/table';
import type { CyberAlert, MITRETacticItem } from '@/types/cyber';

import { AlertAssignDialog } from './_components/alert-assign-dialog';
import { getAlertColumns } from './_components/alert-columns';
import { AlertEscalateDialog } from './_components/alert-escalate-dialog';
import { AlertFalsePositiveDialog } from './_components/alert-false-positive-dialog';
import { buildAlertFilters, flattenAlertFetchParams } from './_components/alert-filters';
import { AlertMergeDialog } from './_components/alert-merge-dialog';
import { AlertStatsBar } from './_components/alert-stats-bar';

function fetchAlerts(params: FetchParams): Promise<PaginatedResponse<CyberAlert>> {
  return apiGet<PaginatedResponse<CyberAlert>>(
    API_ENDPOINTS.CYBER_ALERTS,
    flattenAlertFetchParams(params),
  );
}

export default function CyberAlertsPage() {
  const router = useRouter();
  const { hasPermission } = useAuth();
  const canWrite = hasPermission('cyber:write');

  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [tableResetKey, setTableResetKey] = useState(0);
  const [assignTarget, setAssignTarget] = useState<CyberAlert | null>(null);
  const [assignBulkIds, setAssignBulkIds] = useState<string[]>([]);
  const [escalateTarget, setEscalateTarget] = useState<CyberAlert | null>(null);
  const [falsePositiveIds, setFalsePositiveIds] = useState<string[]>([]);
  const [mergeIds, setMergeIds] = useState<string[]>([]);
  const [ackTarget, setAckTarget] = useState<CyberAlert | null>(null);

  const { tableProps, setFilter, refetch } = useDataTable<CyberAlert>({
    fetchFn: fetchAlerts,
    queryKey: 'cyber-alerts',
    defaultPageSize: 25,
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['alert.created', 'alert.updated', 'alert.status_changed', 'alert.assigned', 'alert.escalated'],
  });

  const tacticsQuery = useQuery({
    queryKey: ['cyber-mitre-tactics'],
    queryFn: () => apiGet<{ data: MITRETacticItem[] }>(API_ENDPOINTS.CYBER_MITRE_TACTICS),
  });

  const filters = useMemo(
    () => buildAlertFilters(tacticsQuery.data?.data ?? []),
    [tacticsQuery.data?.data],
  );

  const currentAlerts = tableProps.data;
  const mergeAlerts = currentAlerts.filter((alert) => mergeIds.includes(alert.id));

  const columns = useMemo(
    () => getAlertColumns({
      includeSelection: canWrite,
      onAssign: canWrite ? setAssignTarget : undefined,
      onEscalate: canWrite ? setEscalateTarget : undefined,
      onAcknowledge: canWrite ? setAckTarget : undefined,
    }),
    [canWrite],
  );

  const bulkActions = useMemo<BulkAction[]>(() => {
    if (!canWrite) {
      return [];
    }

    return [
      {
        label: 'Acknowledge Selected',
        icon: CheckCheck,
        onClick: async (ids) => {
          if (ids.length === 0) {
            toast.error('Select at least one alert');
            return;
          }
          await Promise.all(ids.map((id) => (
            apiPut(API_ENDPOINTS.CYBER_ALERT_STATUS(id), { status: 'acknowledged' })
          )));
          toast.success(`${ids.length} alerts acknowledged`);
          await handleMutationComplete();
        },
      },
      {
        label: 'Assign to Analyst',
        icon: UserCheck,
        onClick: async (ids) => {
          if (ids.length === 0) {
            toast.error('Select at least one alert');
            return;
          }
          setAssignBulkIds(ids);
        },
      },
      {
        label: 'Mark False Positive',
        icon: ShieldAlert,
        onClick: async (ids) => {
          if (ids.length === 0) {
            toast.error('Select at least one alert');
            return;
          }
          setFalsePositiveIds(ids);
        },
      },
      {
        label: 'Merge Selected Alerts',
        icon: GitMerge,
        onClick: async (ids) => {
          if (ids.length < 2) {
            toast.error('Select at least two alerts to merge');
            return;
          }
          setMergeIds(ids);
        },
      },
    ];
  }, [canWrite]);

  async function handleMutationComplete() {
    setSelectedIds([]);
    setTableResetKey((value) => value + 1);
    await refetch();
    void tacticsQuery.refetch();
  }

  async function handleAcknowledge(alert: CyberAlert) {
    await apiPut(API_ENDPOINTS.CYBER_ALERT_STATUS(alert.id), { status: 'acknowledged' });
    toast.success('Alert acknowledged');
    await handleMutationComplete();
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Alert Management"
          description="Triages new detections, route investigations to analysts, and pivot from MITRE techniques into evidence, comments, and correlated alerts."
        />

        <AlertStatsBar onFilterByStatus={(status) => setFilter('status', status)} />

        <DataTable
          key={tableResetKey}
          {...tableProps}
          columns={columns}
          filters={filters}
          searchPlaceholder="Search alerts, rules, assets, or investigation context…"
          getRowId={(row) => row.id}
          onRowClick={(row) => router.push(`${ROUTES.CYBER_ALERTS}/${row.id}`)}
          enableSelection={canWrite}
          onSelectionChange={setSelectedIds}
          bulkActions={bulkActions}
          emptyState={{
            icon: AlertTriangle,
            title: 'No alerts found',
            description: 'No alerts match the current filters.',
          }}
        />
      </div>

      <PermissionGate permission="cyber:write">
        <AlertAssignDialog
          open={Boolean(assignTarget) || assignBulkIds.length > 0}
          onOpenChange={(open) => {
            if (!open) {
              setAssignTarget(null);
              setAssignBulkIds([]);
            }
          }}
          alert={assignTarget}
          alertIds={assignBulkIds}
          onSuccess={() => {
            setAssignTarget(null);
            setAssignBulkIds([]);
            void handleMutationComplete();
          }}
        />

        {escalateTarget && (
          <AlertEscalateDialog
            open={Boolean(escalateTarget)}
            onOpenChange={(open) => {
              if (!open) {
                setEscalateTarget(null);
              }
            }}
            alert={escalateTarget}
            onSuccess={() => {
              setEscalateTarget(null);
              void handleMutationComplete();
            }}
          />
        )}

        <AlertFalsePositiveDialog
          open={falsePositiveIds.length > 0}
          onOpenChange={(open) => {
            if (!open) {
              setFalsePositiveIds([]);
            }
          }}
          alertIds={falsePositiveIds}
          onSuccess={() => {
            setFalsePositiveIds([]);
            void handleMutationComplete();
          }}
        />

        <AlertMergeDialog
          open={mergeIds.length > 0}
          onOpenChange={(open) => {
            if (!open) {
              setMergeIds([]);
            }
          }}
          alerts={mergeAlerts}
          onSuccess={() => {
            setMergeIds([]);
            void handleMutationComplete();
          }}
        />

        {ackTarget && (
          <ConfirmDialog
            open={Boolean(ackTarget)}
            onOpenChange={(open) => {
              if (!open) {
                setAckTarget(null);
              }
            }}
            title="Acknowledge Alert"
            description={`This will move ${ackTarget.title} into the acknowledged state and auto-assign it to you if it is currently unowned.`}
            confirmLabel="Acknowledge"
            onConfirm={async () => {
              if (ackTarget) {
                await handleAcknowledge(ackTarget);
                setAckTarget(null);
              }
            }}
          />
        )}
      </PermissionGate>
    </PermissionRedirect>
  );
}
