'use client';

import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { LayoutGrid, Plus, ShieldCheck } from 'lucide-react';

import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ExportMenu } from '@/components/cyber/export-menu';
import { DataTable } from '@/components/shared/data-table/data-table';
import { Button } from '@/components/ui/button';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import {
  buildRuleQueryParams,
  normalizeRule,
  normalizeRuleList,
  normalizeRuleTemplate,
} from '@/lib/cyber-rules';
import { API_ENDPOINTS } from '@/lib/constants';
import type { PaginatedResponse } from '@/types/api';
import type { DetectionRule, DetectionRuleStats, RuleTemplate } from '@/types/cyber';
import type { FetchParams, FilterConfig } from '@/types/table';

import { getRuleColumns } from './_components/rule-columns';
import { RuleStats } from './_components/rule-stats';
import { RuleTemplateGallery } from './_components/rule-template-gallery';
import { RuleTestDialog } from './_components/rule-test-dialog';
import { RuleWizard } from './_components/rule-wizard';

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
      { label: 'Info', value: 'info' },
    ],
  },
  {
    key: 'mitre_tactic_id',
    label: 'MITRE Tactic',
    type: 'select',
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

async function fetchRules(params: FetchParams): Promise<PaginatedResponse<DetectionRule>> {
  const response = await apiGet<PaginatedResponse<DetectionRule>>(API_ENDPOINTS.CYBER_RULES, buildRuleQueryParams(params));
  return {
    ...response,
    data: normalizeRuleList(response.data),
  };
}

export default function DetectionRulesPage() {
  const searchParams = useSearchParams();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<DetectionRule | null>(null);
  const [testRule, setTestRule] = useState<DetectionRule | null>(null);
  const [templateGalleryOpen, setTemplateGalleryOpen] = useState(false);
  const [activatedTemplateIds, setActivatedTemplateIds] = useState<string[]>([]);
  const initialTechniqueId = searchParams?.get('mitre_technique_id');
  const createRequested = searchParams?.get('create') === '1';

  const { tableProps, refetch, totalRows, activeFilters } = useDataTable<DetectionRule>({
    fetchFn: fetchRules,
    queryKey: 'cyber-rules',
    defaultPageSize: 25,
    defaultSort: { column: 'last_triggered_at', direction: 'desc' },
    wsTopics: ['cyber.rule.created', 'cyber.rule.updated', 'cyber.rule.toggled'],
  });

  const { data: statsEnvelope, isLoading: statsLoading } = useQuery({
    queryKey: ['cyber-rules-stats'],
    queryFn: () => apiGet<{ data: DetectionRuleStats }>(API_ENDPOINTS.CYBER_RULE_STATS),
  });

  const { data: tacticsEnvelope } = useQuery({
    queryKey: ['cyber-rule-filter-tactics'],
    queryFn: () => apiGet<{ data: Array<{ id: string; name: string }> }>(API_ENDPOINTS.CYBER_MITRE_TACTICS),
    staleTime: 300_000,
  });

  const filterConfig = useMemo(() => {
    return RULE_FILTERS.map((filter) =>
      filter.key === 'mitre_tactic_id'
        ? {
            ...filter,
            options: (tacticsEnvelope?.data ?? []).map((tactic) => ({
              label: tactic.name,
              value: tactic.id,
            })),
          }
        : filter,
    );
  }, [tacticsEnvelope?.data]);

  const toggleMutation = useApiMutation<DetectionRule, { id: string; enabled: boolean }>(
    'put',
    (variables) => API_ENDPOINTS.CYBER_RULE_TOGGLE(variables.id),
    {
      successMessage: 'Rule status updated',
      invalidateKeys: ['cyber-rules', 'cyber-rules-stats', 'cyber-mitre-coverage'],
      onSuccess: () => refetch(),
    },
  );

  const deleteMutation = useApiMutation<void, { id: string }>('delete', (variables) => API_ENDPOINTS.CYBER_RULE_DETAIL(variables.id), {
    successMessage: 'Detection rule deleted',
    invalidateKeys: ['cyber-rules', 'cyber-rules-stats', 'cyber-mitre-coverage'],
    onSuccess: () => refetch(),
  });

  const columns = getRuleColumns({
    onToggle: (rule) =>
      toggleMutation.mutate({
        id: rule.id,
        enabled: !rule.enabled,
      }),
    onEdit: (rule) => {
      setEditingRule(normalizeRule(rule));
      setWizardOpen(true);
    },
    onDuplicate: (rule) => {
      setEditingRule({
        ...normalizeRule(rule),
        id: '',
        name: `Copy of ${rule.name}`,
      });
      setWizardOpen(true);
    },
    onDelete: (rule) => deleteMutation.mutate({ id: rule.id }),
    onTest: (rule) => setTestRule(normalizeRule(rule)),
  });

  useEffect(() => {
    if (createRequested && !wizardOpen) {
      setEditingRule(null);
      setWizardOpen(true);
    }
  }, [createRequested, wizardOpen]);

  function handleCreateFromTemplate(template: RuleTemplate) {
    const normalized = normalizeRuleTemplate(template);
    setActivatedTemplateIds((current) => [...current, template.id]);
    setEditingRule({
      id: '',
      name: normalized.name,
      description: normalized.description,
      rule_type: normalized.rule_type,
      type: normalized.rule_type,
      severity: normalized.severity,
      enabled: true,
      mitre_tactic_ids: normalized.mitre_tactic_ids ?? [],
      mitre_technique_ids: normalized.mitre_technique_ids,
      trigger_count: 0,
      false_positive_count: 0,
      true_positive_count: 0,
      rule_content: normalized.rule_content ?? {},
      base_confidence: 0.7,
      is_template: false,
      tags: normalized.tags ?? [],
      created_at: '',
      updated_at: '',
    });
    setTemplateGalleryOpen(false);
    // Defer wizard open until after the Sheet's exit animation starts to avoid
    // Radix Presence ref-callback infinite loop when two overlays transition simultaneously.
    setTimeout(() => setWizardOpen(true), 0);
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Detection Rules"
          description="Manage Sigma, threshold, correlation, and anomaly rules against the live MITRE coverage model."
          actions={
            <div className="flex flex-wrap items-center gap-2">
              <ExportMenu
                entityType="rules"
                baseUrl={API_ENDPOINTS.CYBER_RULES}
                currentFilters={activeFilters as Record<string, string>}
                totalCount={totalRows}
                enabledFormats={['csv', 'json']}
              />
              <Button variant="outline" onClick={() => setTemplateGalleryOpen(true)}>
                <LayoutGrid className="mr-2 h-4 w-4" />
                Templates
              </Button>
              <Button
                onClick={() => {
                  setEditingRule(null);
                  setWizardOpen(true);
                }}
              >
                <Plus className="mr-2 h-4 w-4" />
                Create Rule
              </Button>
            </div>
          }
        />

        <RuleStats stats={statsEnvelope?.data} loading={statsLoading} />

        <DataTable
          columns={columns}
          filters={filterConfig}
          searchPlaceholder="Search rules by name or description"
          emptyState={{
            icon: ShieldCheck,
            title: 'No detection rules',
            description: 'Create a rule or activate a template to start detecting activity.',
            action: {
              label: 'Create Rule',
              onClick: () => {
                setEditingRule(null);
                setWizardOpen(true);
              },
            },
          }}
          getRowId={(row) => row.id || row.name}
          enableColumnToggle
          {...tableProps}
        />
      </div>

      <RuleWizard
        open={wizardOpen}
        onOpenChange={(open) => {
          setWizardOpen(open);
          if (!open) {
            setEditingRule(null);
          }
        }}
        rule={editingRule}
        initialTechniqueId={editingRule ? null : initialTechniqueId}
        onSuccess={() => {
          setEditingRule(null);
          refetch();
        }}
      />

      <RuleTestDialog
        open={Boolean(testRule)}
        onOpenChange={(open) => {
          if (!open) {
            setTestRule(null);
          }
        }}
        rule={testRule}
      />

      <RuleTemplateGallery
        open={templateGalleryOpen}
        onOpenChange={setTemplateGalleryOpen}
        activatedTemplateIds={activatedTemplateIds}
        onActivate={handleCreateFromTemplate}
      />
    </PermissionRedirect>
  );
}
