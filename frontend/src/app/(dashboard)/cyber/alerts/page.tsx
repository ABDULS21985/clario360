'use client';

import { useState } from 'react';
import { Bell } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';
import type { CyberAlert } from '@/types/cyber';

import { getAlertColumns } from './_components/alert-columns';
import { ALERT_FILTERS } from './_components/alert-filters';
import { AlertStatBar } from './_components/alert-stat-bar';
import { AlertAssignDialog } from './_components/alert-assign-dialog';
import { AlertStatusDialog } from './_components/alert-status-dialog';
import { AlertEscalateDialog } from './_components/alert-escalate-dialog';

function fetchAlerts(params: FetchParams): Promise<PaginatedResponse<CyberAlert>> {
  return apiGet<PaginatedResponse<CyberAlert>>(API_ENDPOINTS.CYBER_ALERTS, params as unknown as Record<string, unknown>);
}

export default function CyberAlertsPage() {
  const [assignTarget, setAssignTarget] = useState<CyberAlert | null>(null);
  const [statusTarget, setStatusTarget] = useState<CyberAlert | null>(null);
  const [escalateTarget, setEscalateTarget] = useState<CyberAlert | null>(null);

  const { tableProps, setFilter, refetch } = useDataTable<CyberAlert>({
    fetchFn: fetchAlerts,
    queryKey: 'cyber-alerts',
    defaultPageSize: 25,
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['alert.created', 'alert.status_changed', 'alert.assigned', 'alert.escalated'],
  });

  const columns = getAlertColumns({
    onAssign: setAssignTarget,
    onChangeStatus: setStatusTarget,
    onEscalate: setEscalateTarget,
  });

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Security Alerts"
          description="Monitor, investigate, and resolve security alerts"
        />

        <AlertStatBar onFilterBySeverity={(sev) => setFilter('severity', sev)} />

        <DataTable
          columns={columns}
          filters={ALERT_FILTERS}
          searchPlaceholder="Search alerts…"
          emptyState={{
            icon: Bell,
            title: 'No alerts',
            description: 'No security alerts match the current filter.',
          }}
          getRowId={(row) => row.id}
          enableColumnToggle
          stickyHeader
          {...tableProps}
        />
      </div>

      {assignTarget && (
        <AlertAssignDialog
          open={!!assignTarget}
          onOpenChange={(o) => { if (!o) setAssignTarget(null); }}
          alert={assignTarget}
          onSuccess={() => { setAssignTarget(null); refetch(); }}
        />
      )}
      {statusTarget && (
        <AlertStatusDialog
          open={!!statusTarget}
          onOpenChange={(o) => { if (!o) setStatusTarget(null); }}
          alert={statusTarget}
          onSuccess={() => { setStatusTarget(null); refetch(); }}
        />
      )}
      {escalateTarget && (
        <AlertEscalateDialog
          open={!!escalateTarget}
          onOpenChange={(o) => { if (!o) setEscalateTarget(null); }}
          alert={escalateTarget}
          onSuccess={() => { setEscalateTarget(null); refetch(); }}
        />
      )}
    </PermissionRedirect>
  );
}
