'use client';

import { useState } from 'react';
import { Plus, ShieldCheck } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams, FilterConfig } from '@/types/table';
import type { DetectionRule } from '@/types/cyber';

import { getRuleColumns } from './_components/rule-columns';
import { RuleFormDialog } from './_components/rule-form-dialog';
import { RuleTestDialog } from './_components/rule-test-dialog';

const RULE_FILTERS: FilterConfig[] = [
  {
    key: 'type',
    label: 'Type',
    type: 'multi-select',
    options: [
      { label: 'Sigma', value: 'sigma' },
      { label: 'Threshold', value: 'threshold' },
      { label: 'Correlation', value: 'correlation' },
      { label: 'Anomaly', value: 'anomaly' },
    ],
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
    key: 'enabled',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Enabled', value: 'true' },
      { label: 'Disabled', value: 'false' },
    ],
  },
];

function fetchRules(params: FetchParams): Promise<PaginatedResponse<DetectionRule>> {
  return apiGet<PaginatedResponse<DetectionRule>>(API_ENDPOINTS.CYBER_RULES, params as unknown as Record<string, unknown>);
}

export default function DetectionRulesPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<DetectionRule | null>(null);
  const [testTarget, setTestTarget] = useState<DetectionRule | null>(null);

  const { tableProps, refetch } = useDataTable<DetectionRule>({
    fetchFn: fetchRules,
    queryKey: 'cyber-rules',
    defaultPageSize: 25,
    defaultSort: { column: 'trigger_count', direction: 'desc' },
  });

  const { mutate: toggleRule } = useApiMutation<DetectionRule, void>(
    'put',
    (rule: unknown) => `${API_ENDPOINTS.CYBER_RULES}/${(rule as DetectionRule).id}/toggle`,
    {
      invalidateKeys: ['cyber-rules'],
    },
  );

  const columns = getRuleColumns({
    onToggle: (rule) => toggleRule(rule as unknown as void),
    onEdit: setEditTarget,
    onTest: setTestTarget,
  });

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Detection Rules"
          description="Manage SIEM detection rules and configure alert thresholds"
          actions={
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              New Rule
            </Button>
          }
        />

        <DataTable
          columns={columns}
          filters={RULE_FILTERS}
          searchPlaceholder="Search rules…"
          emptyState={{
            icon: ShieldCheck,
            title: 'No detection rules',
            description: 'Create your first detection rule to start monitoring for threats.',
            action: { label: 'Create Rule', onClick: () => setCreateOpen(true) },
          }}
          getRowId={(row) => row.id}
          enableColumnToggle
          {...tableProps}
        />
      </div>

      <RuleFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={() => refetch()}
      />
      {editTarget && (
        <RuleFormDialog
          open={!!editTarget}
          onOpenChange={(o) => { if (!o) setEditTarget(null); }}
          rule={editTarget}
          onSuccess={() => { setEditTarget(null); refetch(); }}
        />
      )}
      {testTarget && (
        <RuleTestDialog
          open={!!testTarget}
          onOpenChange={(o) => { if (!o) setTestTarget(null); }}
          rule={testTarget}
        />
      )}
    </PermissionRedirect>
  );
}
