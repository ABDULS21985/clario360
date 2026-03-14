'use client';

import { useState, useMemo, useCallback } from 'react';
import { type ColumnDef } from '@tanstack/react-table';
import {
  ShieldAlert,
  Plus,
  Eye,
  Edit,
  Trash2,
  Power,
  BookOpen,
  PlayCircle,
  Archive,
  Zap,
  ClipboardList,
  AlertTriangle,
  Clock,
} from 'lucide-react';
import { toast } from 'sonner';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { KpiCard } from '@/components/shared/kpi-card';
import { StatusBadge } from '@/components/shared/status-badge';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs';
import { useDataTable } from '@/hooks/use-data-table';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate, titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import { playbookStatusConfig, simulationResultConfig } from '@/lib/status-configs';
import type { PaginatedResponse } from '@/types/api';
import type { FilterConfig, RowAction } from '@/types/table';
import type {
  VCISOEscalationRule,
  VCISOPlaybook,
} from '@/types/cyber';

import { EscalationRuleFormDialog } from './_components/escalation-rule-form-dialog';
import { EscalationRuleDetailPanel } from './_components/escalation-rule-detail-panel';
import { PlaybookFormDialog } from './_components/playbook-form-dialog';
import { PlaybookDetailPanel } from './_components/playbook-detail-panel';

// ── Helpers ──────────────────────────────────────────────────────────────────

function isOverdue(dateStr: string): boolean {
  try {
    return new Date(dateStr) < new Date();
  } catch {
    return false;
  }
}

function triggerTypeColor(type: string): string {
  switch (type) {
    case 'severity':
      return 'bg-red-100 text-red-700';
    case 'time':
      return 'bg-blue-100 text-blue-700';
    case 'count':
      return 'bg-amber-100 text-amber-700';
    case 'custom':
      return 'bg-purple-100 text-purple-700';
    default:
      return 'bg-gray-100 text-gray-700';
  }
}

function targetColor(target: string): string {
  switch (target) {
    case 'management':
      return 'bg-blue-100 text-blue-700';
    case 'legal':
      return 'bg-purple-100 text-purple-700';
    case 'regulator':
      return 'bg-orange-100 text-orange-700';
    case 'board':
      return 'bg-red-100 text-red-700';
    case 'custom':
      return 'bg-teal-100 text-teal-700';
    default:
      return 'bg-gray-100 text-gray-700';
  }
}

// ── Filters ──────────────────────────────────────────────────────────────────

const ESCALATION_FILTERS: FilterConfig[] = [
  {
    key: 'trigger_type',
    label: 'Trigger Type',
    type: 'select',
    options: [
      { label: 'Severity', value: 'severity' },
      { label: 'Time', value: 'time' },
      { label: 'Count', value: 'count' },
      { label: 'Custom', value: 'custom' },
    ],
  },
  {
    key: 'escalation_target',
    label: 'Target',
    type: 'select',
    options: [
      { label: 'Management', value: 'management' },
      { label: 'Legal', value: 'legal' },
      { label: 'Regulator', value: 'regulator' },
      { label: 'Board', value: 'board' },
      { label: 'Custom', value: 'custom' },
    ],
  },
];

const PLAYBOOK_FILTERS: FilterConfig[] = [
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Draft', value: 'draft' },
      { label: 'Approved', value: 'approved' },
      { label: 'Tested', value: 'tested' },
      { label: 'Retired', value: 'retired' },
    ],
  },
];

// ── Escalation Rule Columns ──────────────────────────────────────────────────

function getEscalationColumns(
  onToggleEnabled: (rule: VCISOEscalationRule) => void,
): ColumnDef<VCISOEscalationRule>[] {
  return [
    {
      accessorKey: 'name',
      header: 'Name',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="font-medium text-foreground">{row.original.name}</span>
      ),
    },
    {
      accessorKey: 'trigger_type',
      header: 'Trigger Type',
      enableSorting: true,
      cell: ({ row }) => (
        <span
          className={cn(
            'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
            triggerTypeColor(row.original.trigger_type),
          )}
        >
          {titleCase(row.original.trigger_type)}
        </span>
      ),
    },
    {
      accessorKey: 'trigger_condition',
      header: 'Trigger Condition',
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground max-w-[200px] truncate block">
          {row.original.trigger_condition}
        </span>
      ),
    },
    {
      accessorKey: 'escalation_target',
      header: 'Target',
      enableSorting: true,
      cell: ({ row }) => (
        <span
          className={cn(
            'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
            targetColor(row.original.escalation_target),
          )}
        >
          {titleCase(row.original.escalation_target)}
        </span>
      ),
    },
    {
      accessorKey: 'enabled',
      header: 'Enabled',
      enableSorting: true,
      cell: ({ row }) => (
        <Switch
          checked={row.original.enabled}
          onCheckedChange={() => onToggleEnabled(row.original)}
          aria-label={`Toggle ${row.original.name}`}
        />
      ),
    },
    {
      accessorKey: 'trigger_count',
      header: 'Trigger Count',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm font-medium">{row.original.trigger_count}</span>
      ),
    },
    {
      accessorKey: 'last_triggered_at',
      header: 'Last Triggered',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.last_triggered_at
            ? formatDate(row.original.last_triggered_at)
            : '--'}
        </span>
      ),
    },
    {
      accessorKey: 'created_at',
      header: 'Created',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {formatDate(row.original.created_at)}
        </span>
      ),
    },
  ];
}

// ── Playbook Columns ─────────────────────────────────────────────────────────

function getPlaybookColumns(): ColumnDef<VCISOPlaybook>[] {
  return [
    {
      accessorKey: 'name',
      header: 'Name',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="font-medium text-foreground">{row.original.name}</span>
      ),
    },
    {
      accessorKey: 'scenario',
      header: 'Scenario',
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground max-w-[200px] truncate block">
          {row.original.scenario}
        </span>
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.status} config={playbookStatusConfig} />
      ),
    },
    {
      accessorKey: 'owner_name',
      header: 'Owner',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{row.original.owner_name || 'Unassigned'}</span>
      ),
    },
    {
      accessorKey: 'steps_count',
      header: 'Steps',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm font-medium">{row.original.steps_count}</span>
      ),
    },
    {
      accessorKey: 'rto_hours',
      header: 'RTO (hrs)',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.rto_hours != null ? row.original.rto_hours : '--'}
        </span>
      ),
    },
    {
      accessorKey: 'rpo_hours',
      header: 'RPO (hrs)',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.rpo_hours != null ? row.original.rpo_hours : '--'}
        </span>
      ),
    },
    {
      accessorKey: 'last_tested_at',
      header: 'Last Tested',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.last_tested_at
            ? formatDate(row.original.last_tested_at)
            : '--'}
        </span>
      ),
    },
    {
      accessorKey: 'last_simulation_result',
      header: 'Sim Result',
      enableSorting: true,
      cell: ({ row }) =>
        row.original.last_simulation_result ? (
          <StatusBadge
            status={row.original.last_simulation_result}
            config={simulationResultConfig}
          />
        ) : (
          <span className="text-sm text-muted-foreground">--</span>
        ),
    },
    {
      accessorKey: 'next_test_date',
      header: 'Next Test',
      enableSorting: true,
      cell: ({ row }) => {
        const overdue = isOverdue(row.original.next_test_date);
        return (
          <span
            className={cn(
              'text-sm',
              overdue ? 'text-red-600 font-medium' : 'text-muted-foreground',
            )}
          >
            {formatDate(row.original.next_test_date)}
            {overdue && ' (Overdue)'}
          </span>
        );
      },
    },
  ];
}

// ── Main Page ────────────────────────────────────────────────────────────────

export default function IncidentReadinessPage() {
  // ── Escalation Rules state ──────────────────────────
  const [selectedRule, setSelectedRule] = useState<VCISOEscalationRule | null>(null);
  const [showRuleDialog, setShowRuleDialog] = useState(false);
  const [editingRule, setEditingRule] = useState<VCISOEscalationRule | null>(null);
  const [deleteRuleTarget, setDeleteRuleTarget] = useState<VCISOEscalationRule | null>(null);

  // ── Playbooks state ─────────────────────────────────
  const [selectedPlaybook, setSelectedPlaybook] = useState<VCISOPlaybook | null>(null);
  const [showPlaybookDialog, setShowPlaybookDialog] = useState(false);
  const [editingPlaybook, setEditingPlaybook] = useState<VCISOPlaybook | null>(null);
  const [retireTarget, setRetireTarget] = useState<VCISOPlaybook | null>(null);
  const [simulateTarget, setSimulateTarget] = useState<VCISOPlaybook | null>(null);

  // ── Escalation Rules Table ──────────────────────────
  const {
    data: rulesData,
    totalRows: rulesTotal,
    tableProps: rulesTableProps,
    refetch: refetchRules,
  } = useDataTable<VCISOEscalationRule>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOEscalationRule>>(
        API_ENDPOINTS.CYBER_VCISO_ESCALATION_RULES,
        params as unknown as Record<string, unknown>,
      ),
    queryKey: 'vciso-escalation-rules',
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['vciso.escalation-rules'],
  });

  // ── Playbooks Table ─────────────────────────────────
  const {
    data: playbooksData,
    totalRows: playbooksTotal,
    tableProps: playbooksTableProps,
    refetch: refetchPlaybooks,
  } = useDataTable<VCISOPlaybook>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOPlaybook>>(
        API_ENDPOINTS.CYBER_VCISO_PLAYBOOKS,
        params as unknown as Record<string, unknown>,
      ),
    queryKey: 'vciso-playbooks',
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['vciso.playbooks'],
  });

  // ── Toggle enabled mutation ─────────────────────────
  const toggleMutation = useApiMutation<VCISOEscalationRule, Record<string, unknown>>(
    'put',
    (variables) =>
      `${API_ENDPOINTS.CYBER_VCISO_ESCALATION_RULES}/${(variables as Record<string, string>).id}`,
    {
      invalidateKeys: ['vciso-escalation-rules'],
      onSuccess: (data) => {
        toast.success(`Rule ${data.enabled ? 'enabled' : 'disabled'} successfully`);
        refetchRules();
      },
    },
  );

  // ── Delete rule mutation ────────────────────────────
  const deleteRuleMutation = useApiMutation<void, Record<string, unknown>>(
    'delete',
    deleteRuleTarget
      ? `${API_ENDPOINTS.CYBER_VCISO_ESCALATION_RULES}/${deleteRuleTarget.id}`
      : '',
    {
      successMessage: 'Escalation rule deleted successfully',
      invalidateKeys: ['vciso-escalation-rules'],
      onSuccess: () => {
        setDeleteRuleTarget(null);
        refetchRules();
      },
    },
  );

  // ── Retire playbook mutation ────────────────────────
  const retireMutation = useApiMutation<VCISOPlaybook, Record<string, unknown>>(
    'put',
    retireTarget
      ? `${API_ENDPOINTS.CYBER_VCISO_PLAYBOOKS}/${retireTarget.id}`
      : '',
    {
      successMessage: 'Playbook retired successfully',
      invalidateKeys: ['vciso-playbooks'],
      onSuccess: () => {
        setRetireTarget(null);
        refetchPlaybooks();
      },
    },
  );

  // ── Simulate playbook mutation ──────────────────────
  const simulateMutation = useApiMutation<VCISOPlaybook, Record<string, unknown>>(
    'post',
    simulateTarget
      ? `${API_ENDPOINTS.CYBER_VCISO_PLAYBOOKS}/${simulateTarget.id}`
      : '',
    {
      successMessage: 'Simulation started successfully',
      invalidateKeys: ['vciso-playbooks'],
      onSuccess: () => {
        setSimulateTarget(null);
        refetchPlaybooks();
      },
    },
  );

  // ── Toggle enabled handler ──────────────────────────
  const handleToggleEnabled = useCallback(
    (rule: VCISOEscalationRule) => {
      toggleMutation.mutate({
        id: rule.id,
        enabled: !rule.enabled,
      });
    },
    [toggleMutation],
  );

  // ── KPI stats computed from loaded data ─────────────
  const enabledRulesCount = useMemo(
    () => rulesData.filter((r) => r.enabled).length,
    [rulesData],
  );

  const totalTriggerCount = useMemo(
    () => rulesData.reduce((sum, r) => sum + r.trigger_count, 0),
    [rulesData],
  );

  const testedPlaybooksCount = useMemo(
    () => playbooksData.filter((p) => p.status === 'tested').length,
    [playbooksData],
  );

  const overduePlaybooksCount = useMemo(
    () => playbooksData.filter((p) => isOverdue(p.next_test_date) && p.status !== 'retired').length,
    [playbooksData],
  );

  // ── Columns ─────────────────────────────────────────
  const escalationColumns = useMemo(
    () => getEscalationColumns(handleToggleEnabled),
    [handleToggleEnabled],
  );
  const playbookColumns = useMemo(() => getPlaybookColumns(), []);

  // ── Escalation row actions ──────────────────────────
  const escalationRowActions: RowAction<VCISOEscalationRule>[] = useMemo(
    () => [
      {
        label: 'View Details',
        icon: Eye,
        onClick: (row) => setSelectedRule(row),
      },
      {
        label: 'Edit',
        icon: Edit,
        onClick: (row) => {
          setEditingRule(row);
          setShowRuleDialog(true);
        },
      },
      {
        label: 'Delete',
        icon: Trash2,
        variant: 'destructive' as const,
        onClick: (row) => setDeleteRuleTarget(row),
      },
      {
        label: 'Toggle Enable',
        icon: Power,
        onClick: (row) => handleToggleEnabled(row),
      },
    ],
    [handleToggleEnabled],
  );

  // ── Playbook row actions ────────────────────────────
  const playbookRowActions: RowAction<VCISOPlaybook>[] = useMemo(
    () => [
      {
        label: 'View Details',
        icon: Eye,
        onClick: (row) => setSelectedPlaybook(row),
      },
      {
        label: 'Edit',
        icon: Edit,
        onClick: (row) => {
          setEditingPlaybook(row);
          setShowPlaybookDialog(true);
        },
        hidden: (row) => row.status === 'retired',
      },
      {
        label: 'Run Simulation',
        icon: PlayCircle,
        onClick: (row) => setSimulateTarget(row),
        hidden: (row) => row.status === 'retired',
      },
      {
        label: 'Retire',
        icon: Archive,
        variant: 'destructive' as const,
        onClick: (row) => setRetireTarget(row),
        hidden: (row) => row.status === 'retired',
      },
    ],
    [],
  );

  // ── Event handlers ──────────────────────────────────
  const handleRuleDialogOpen = () => {
    setEditingRule(null);
    setShowRuleDialog(true);
  };

  const handleRuleDialogChange = (open: boolean) => {
    if (!open) {
      setEditingRule(null);
    }
    setShowRuleDialog(open);
  };

  const handlePlaybookDialogOpen = () => {
    setEditingPlaybook(null);
    setShowPlaybookDialog(true);
  };

  const handlePlaybookDialogChange = (open: boolean) => {
    if (!open) {
      setEditingPlaybook(null);
    }
    setShowPlaybookDialog(open);
  };

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {/* Page Header */}
        <PageHeader
          title="Incident Readiness"
          description="Manage escalation rules and crisis playbooks to ensure rapid, coordinated incident response."
        />

        {/* KPI Stats Row */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <KpiCard
            title="Escalation Rules"
            value={rulesTotal}
            icon={Zap}
            iconColor="text-amber-600"
            description={`${enabledRulesCount} enabled`}
          />
          <KpiCard
            title="Total Triggers"
            value={totalTriggerCount}
            icon={ShieldAlert}
            iconColor="text-red-600"
            description="Across all rules"
          />
          <KpiCard
            title="Tested Playbooks"
            value={testedPlaybooksCount}
            icon={ClipboardList}
            iconColor="text-green-600"
            description={`of ${playbooksTotal} total`}
          />
          <KpiCard
            title="Overdue Tests"
            value={overduePlaybooksCount}
            icon={AlertTriangle}
            iconColor="text-red-600"
            description="Past next test date"
          />
        </div>

        {/* Tabs */}
        <Tabs defaultValue="escalation" className="space-y-4">
          <TabsList>
            <TabsTrigger value="escalation">Escalation Rules</TabsTrigger>
            <TabsTrigger value="playbooks">Crisis Playbooks</TabsTrigger>
          </TabsList>

          {/* ── Escalation Rules Tab ───────────────────────────── */}
          <TabsContent value="escalation" className="space-y-4">
            <div className="flex justify-end">
              <Button onClick={handleRuleDialogOpen}>
                <Plus className="mr-2 h-4 w-4" />
                Add Rule
              </Button>
            </div>
            <DataTable
              columns={escalationColumns}
              filters={ESCALATION_FILTERS}
              rowActions={escalationRowActions}
              searchPlaceholder="Search escalation rules..."
              emptyState={{
                icon: Zap,
                title: 'No escalation rules found',
                description:
                  'No escalation rules match the current filters or none have been created yet.',
                action: {
                  label: 'Add Rule',
                  onClick: handleRuleDialogOpen,
                  icon: Plus,
                },
              }}
              onRowClick={(row) => setSelectedRule(row)}
              getRowId={(row) => row.id}
              enableColumnToggle
              stickyHeader
              {...rulesTableProps}
            />
          </TabsContent>

          {/* ── Crisis Playbooks Tab ───────────────────────────── */}
          <TabsContent value="playbooks" className="space-y-4">
            <div className="flex justify-end">
              <Button onClick={handlePlaybookDialogOpen}>
                <Plus className="mr-2 h-4 w-4" />
                Add Playbook
              </Button>
            </div>
            <DataTable
              columns={playbookColumns}
              filters={PLAYBOOK_FILTERS}
              rowActions={playbookRowActions}
              searchPlaceholder="Search playbooks..."
              emptyState={{
                icon: BookOpen,
                title: 'No playbooks found',
                description:
                  'No crisis playbooks match the current filters or none have been created yet.',
                action: {
                  label: 'Add Playbook',
                  onClick: handlePlaybookDialogOpen,
                  icon: Plus,
                },
              }}
              onRowClick={(row) => setSelectedPlaybook(row)}
              getRowId={(row) => row.id}
              enableColumnToggle
              stickyHeader
              {...playbooksTableProps}
            />
          </TabsContent>
        </Tabs>
      </div>

      {/* ── Escalation Rule Detail Panel ───────────────────── */}
      {selectedRule && (
        <EscalationRuleDetailPanel
          open={!!selectedRule}
          onOpenChange={(o) => {
            if (!o) setSelectedRule(null);
          }}
          rule={selectedRule}
        />
      )}

      {/* ── Escalation Rule Form Dialog ────────────────────── */}
      <EscalationRuleFormDialog
        open={showRuleDialog}
        onOpenChange={handleRuleDialogChange}
        onSaved={refetchRules}
        editRule={editingRule}
      />

      {/* ── Delete Rule Confirm ────────────────────────────── */}
      <ConfirmDialog
        open={!!deleteRuleTarget}
        onOpenChange={(o) => {
          if (!o) setDeleteRuleTarget(null);
        }}
        title="Delete Escalation Rule"
        description={`Are you sure you want to delete the escalation rule "${deleteRuleTarget?.name ?? ''}"? This action cannot be undone.`}
        confirmLabel="Delete Rule"
        variant="destructive"
        loading={deleteRuleMutation.isPending}
        onConfirm={async () => {
          if (deleteRuleTarget) {
            deleteRuleMutation.mutate({});
          }
        }}
      />

      {/* ── Playbook Detail Panel ──────────────────────────── */}
      {selectedPlaybook && (
        <PlaybookDetailPanel
          open={!!selectedPlaybook}
          onOpenChange={(o) => {
            if (!o) setSelectedPlaybook(null);
          }}
          playbook={selectedPlaybook}
        />
      )}

      {/* ── Playbook Form Dialog ───────────────────────────── */}
      <PlaybookFormDialog
        open={showPlaybookDialog}
        onOpenChange={handlePlaybookDialogChange}
        onSaved={refetchPlaybooks}
        editPlaybook={editingPlaybook}
      />

      {/* ── Retire Playbook Confirm ────────────────────────── */}
      <ConfirmDialog
        open={!!retireTarget}
        onOpenChange={(o) => {
          if (!o) setRetireTarget(null);
        }}
        title="Retire Playbook"
        description={`Are you sure you want to retire the playbook "${retireTarget?.name ?? ''}"? Retired playbooks are archived and will no longer be scheduled for testing.`}
        confirmLabel="Retire Playbook"
        variant="destructive"
        loading={retireMutation.isPending}
        onConfirm={async () => {
          if (retireTarget) {
            retireMutation.mutate({
              status: 'retired',
            });
          }
        }}
      />

      {/* ── Run Simulation Confirm ─────────────────────────── */}
      <ConfirmDialog
        open={!!simulateTarget}
        onOpenChange={(o) => {
          if (!o) setSimulateTarget(null);
        }}
        title="Run Simulation"
        description={`Are you sure you want to run a simulation for the playbook "${simulateTarget?.name ?? ''}"? This will initiate a tabletop exercise and record the results.`}
        confirmLabel="Start Simulation"
        loading={simulateMutation.isPending}
        onConfirm={async () => {
          if (simulateTarget) {
            simulateMutation.mutate({
              action: 'simulate',
            });
          }
        }}
      />
    </PermissionRedirect>
  );
}
