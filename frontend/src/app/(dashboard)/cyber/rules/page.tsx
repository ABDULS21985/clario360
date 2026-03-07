'use client';

import { useState } from 'react';
import { Plus, ShieldCheck, LayoutGrid } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { ExportMenu } from '@/components/cyber/export-menu';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams, FilterConfig } from '@/types/table';
import type { DetectionRule, RuleTemplate } from '@/types/cyber';

import { getRuleColumns } from './_components/rule-columns';
import { RuleFormDialog } from './_components/rule-form-dialog';
import { RuleTestDialog } from './_components/rule-test-dialog';
import { RuleTemplateGallery } from './_components/rule-template-gallery';

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
  return apiGet<PaginatedResponse<DetectionRule>>(
    API_ENDPOINTS.CYBER_RULES,
    params as unknown as Record<string, unknown>,
  );
}

export default function DetectionRulesPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<DetectionRule | null>(null);
  const [testTarget, setTestTarget] = useState<DetectionRule | null>(null);
  const [galleryOpen, setGalleryOpen] = useState(false);
  const [activatedTemplateIds, setActivatedTemplateIds] = useState<string[]>([]);

  const { tableProps, refetch, totalRows, activeFilters } = useDataTable<DetectionRule>({
    fetchFn: fetchRules,
    queryKey: 'cyber-rules',
    defaultPageSize: 25,
    defaultSort: { column: 'trigger_count', direction: 'desc' },
    wsTopics: ['rule.created', 'rule.updated', 'rule.deleted'],
  });

  // Stats
  const { data: statsEnvelope } = useQuery({
    queryKey: ['cyber-rules-stats'],
    queryFn: () =>
      apiGet<{ data: { total: number; active: number } }>(
        `${API_ENDPOINTS.CYBER_RULES}/stats`,
      ),
  });

  const stats = statsEnvelope?.data;

  const { mutate: toggleRule } = useApiMutation<DetectionRule, void>(
    'put',
    (rule: unknown) => `${API_ENDPOINTS.CYBER_RULES}/${(rule as DetectionRule).id}/toggle`,
    {
      invalidateKeys: ['cyber-rules'],
    },
  );

  const { mutate: duplicateRule } = useApiMutation<DetectionRule, { source_id: string; name: string }>(
    'post',
    `${API_ENDPOINTS.CYBER_RULES}/duplicate`,
    {
      successMessage: 'Rule duplicated',
      invalidateKeys: ['cyber-rules'],
    },
  );

  const { mutate: deleteRule } = useApiMutation<void, { id: string }>(
    'delete',
    (v: unknown) => `${API_ENDPOINTS.CYBER_RULES}/${(v as { id: string }).id}`,
    {
      successMessage: 'Rule deleted',
      invalidateKeys: ['cyber-rules'],
    },
  );

  const columns = getRuleColumns({
    onToggle: (rule) => toggleRule(rule as unknown as void),
    onEdit: setEditTarget,
    onTest: setTestTarget,
    onDuplicate: (rule) =>
      duplicateRule({ source_id: rule.id, name: `Copy of ${rule.name}` }),
    onDelete: (rule) => deleteRule({ id: rule.id } as unknown as { id: string }),
    onViewAlerts: (rule) => {
      window.location.href = `/cyber/alerts?rule_id=${rule.id}`;
    },
  });

  function handleTemplateActivate(template: RuleTemplate) {
    setActivatedTemplateIds((ids) => [...ids, template.id]);
    setGalleryOpen(false);
    setEditTarget({
      id: '',
      tenant_id: '',
      name: template.name,
      description: template.description,
      type: template.type as DetectionRule['type'],
      severity: template.severity,
      enabled: false,
      mitre_technique_ids: template.mitre_technique_ids,
      mitre_tactic_ids: [],
      trigger_count: 0,
      false_positive_rate: 0,
      is_template: false,
      tags: [],
      created_at: '',
      updated_at: '',
    });
    setCreateOpen(true);
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Detection Rules"
          description="Manage threat detection logic and monitor rule performance"
          actions={
            <div className="flex items-center gap-2">
              <ExportMenu
                entityType="rules"
                baseUrl={API_ENDPOINTS.CYBER_RULES}
                currentFilters={activeFilters as Record<string, string>}
                totalCount={totalRows}
                enabledFormats={['csv', 'json']}
              />
              <Button size="sm" variant="outline" onClick={() => setGalleryOpen(true)}>
                <LayoutGrid className="mr-1.5 h-3.5 w-3.5" />
                Templates
              </Button>
              <Button size="sm" onClick={() => setCreateOpen(true)}>
                <Plus className="mr-1.5 h-3.5 w-3.5" />
                Create Rule
              </Button>
            </div>
          }
        />

        {/* Stats bar */}
        {stats && (
          <div className="flex flex-wrap gap-3">
            <Badge variant="secondary" className="text-xs">Total: {stats.total}</Badge>
            <Badge className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300 text-xs">
              Active: {stats.active}
            </Badge>
          </div>
        )}

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
        onSuccess={() => { setCreateOpen(false); refetch(); }}
      />
      {editTarget && (
        <RuleFormDialog
          open={!!editTarget}
          onOpenChange={(o) => {
            if (!o) setEditTarget(null);
          }}
          rule={editTarget.id ? editTarget : null}
          initialTechniqueId={null}
          onSuccess={() => {
            setEditTarget(null);
            refetch();
          }}
        />
      )}
      {testTarget && (
        <RuleTestDialog
          open={!!testTarget}
          onOpenChange={(o) => {
            if (!o) setTestTarget(null);
          }}
          rule={testTarget}
        />
      )}
      <RuleTemplateGallery
        open={galleryOpen}
        onOpenChange={setGalleryOpen}
        activatedTemplateIds={activatedTemplateIds}
        onActivate={handleTemplateActivate}
      />
    </PermissionRedirect>
  );
}
