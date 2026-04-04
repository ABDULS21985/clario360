'use client';

import { useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import { type ColumnDef } from '@tanstack/react-table';
import { Loader2, MoreHorizontal, Pencil, Play, Plus, Scale, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { DataTable } from '@/components/shared/data-table/data-table';
import { ErrorState } from '@/components/common/error-state';
import { KpiCard } from '@/components/shared/kpi-card';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { RelativeTime } from '@/components/shared/relative-time';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { FormField } from '@/components/shared/forms/form-field';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { SectionCard } from '@/components/suites/section-card';
import { useAuth } from '@/hooks/use-auth';
import { useDataTable } from '@/hooks/use-data-table';
import { API_ENDPOINTS } from '@/lib/constants';
import {
  enterpriseApi,
  lexComplianceRuleSchema,
  type LexComplianceRuleFormValues,
} from '@/lib/enterprise';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { showApiError, showSuccess } from '@/lib/toast';
import type { LexComplianceAlert, LexComplianceRule } from '@/types/suites';

// ---------------------------------------------------------------------------
// Compliance rule form dialog (create + edit)
// ---------------------------------------------------------------------------

const RULE_TYPES = [
  'expiry_warning', 'missing_clause', 'risk_threshold', 'review_overdue',
  'unsigned_contract', 'value_threshold', 'jurisdiction_check',
  'data_protection_required', 'custom',
] as const;

const SEVERITIES = ['critical', 'high', 'medium', 'low'] as const;

interface RuleFormDialogProps {
  open: boolean;
  rule?: LexComplianceRule | null;
  onOpenChange: (open: boolean) => void;
}

function RuleFormDialog({ open, rule, onOpenChange }: RuleFormDialogProps) {
  const isEdit = Boolean(rule);
  const queryClient = useQueryClient();

  const form = useForm<LexComplianceRuleFormValues>({
    resolver: zodResolver(lexComplianceRuleSchema),
    defaultValues: rule
      ? {
          name: rule.name,
          description: rule.description,
          rule_type: rule.rule_type,
          severity: rule.severity,
          config: rule.config ?? {},
          contract_types: rule.contract_types ?? [],
          enabled: rule.enabled,
        }
      : {
          name: '',
          description: '',
          rule_type: 'expiry_warning',
          severity: 'medium',
          config: {},
          contract_types: [],
          enabled: true,
        },
  });

  const saveMutation = useMutation({
    mutationFn: (values: LexComplianceRuleFormValues) =>
      isEdit && rule
        ? enterpriseApi.lex.updateComplianceRule(rule.id, values)
        : enterpriseApi.lex.createComplianceRule(values),
    onSuccess: async () => {
      showSuccess(isEdit ? 'Rule updated.' : 'Rule created.', 'The compliance rule has been saved.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['lex-compliance-rules'] }),
        queryClient.invalidateQueries({ queryKey: ['lex-compliance-dashboard'] }),
        queryClient.invalidateQueries({ queryKey: ['lex-overview'] }),
      ]);
      onOpenChange(false);
    },
    onError: showApiError,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit Rule' : 'Create Compliance Rule'}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Update this compliance rule definition.'
              : 'Add a new rule that will be evaluated during compliance checks.'}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <form className="space-y-4" onSubmit={form.handleSubmit((v) => saveMutation.mutate(v))}>
            <FormField name="name" label="Rule name" required>
              <Input id="name" {...form.register('name')} placeholder="30-day expiry warning" />
            </FormField>
            <div className="grid grid-cols-2 gap-4">
              <FormField name="rule_type" label="Rule type" required>
                <Select
                  value={form.watch('rule_type')}
                  onValueChange={(v) =>
                    form.setValue('rule_type', v as LexComplianceRuleFormValues['rule_type'], { shouldValidate: true })
                  }
                >
                  <SelectTrigger id="rule_type"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {RULE_TYPES.map((t) => (
                      <SelectItem key={t} value={t}>{t.replace(/_/g, ' ')}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="severity" label="Severity" required>
                <Select
                  value={form.watch('severity')}
                  onValueChange={(v) =>
                    form.setValue('severity', v as LexComplianceRuleFormValues['severity'], { shouldValidate: true })
                  }
                >
                  <SelectTrigger id="severity"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {SEVERITIES.map((s) => (
                      <SelectItem key={s} value={s}>{s.charAt(0).toUpperCase() + s.slice(1)}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>
            <FormField name="description" label="Description" required>
              <Textarea
                id="description"
                {...form.register('description')}
                rows={3}
                placeholder="Describe what this rule checks and when it triggers."
              />
            </FormField>
            <div className="rounded-lg border px-4 py-4">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="text-sm font-medium">Enabled</p>
                  <p className="text-xs text-muted-foreground">
                    Disabled rules are skipped during compliance checks.
                  </p>
                </div>
                <Switch
                  checked={form.watch('enabled')}
                  onCheckedChange={(v) => form.setValue('enabled', v, { shouldValidate: true })}
                />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={saveMutation.isPending}>
                {saveMutation.isPending ? <Loader2 className="mr-1.5 h-4 w-4 animate-spin" /> : null}
                {isEdit ? 'Save changes' : 'Create rule'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Alert status update dialog
// ---------------------------------------------------------------------------

const ALERT_STATUSES = ['open', 'acknowledged', 'investigating', 'resolved', 'dismissed'] as const;

interface AlertStatusDialogProps {
  open: boolean;
  alert: LexComplianceAlert | null;
  onOpenChange: (open: boolean) => void;
}

function AlertStatusDialog({ open, alert, onOpenChange }: AlertStatusDialogProps) {
  const queryClient = useQueryClient();
  const [nextStatus, setNextStatus] = useState<string>(alert?.status ?? 'open');
  const [notes, setNotes] = useState('');

  const updateMutation = useMutation({
    mutationFn: () => {
      if (!alert) throw new Error('no alert selected');
      return enterpriseApi.lex.updateComplianceAlertStatus(alert.id, {
        status: nextStatus,
        resolution_notes: notes.trim() || '',
      });
    },
    onSuccess: async () => {
      showSuccess('Alert updated.', 'The compliance alert status has been saved.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['lex-compliance-alerts'] }),
        queryClient.invalidateQueries({ queryKey: ['lex-compliance-dashboard'] }),
        queryClient.invalidateQueries({ queryKey: ['lex-overview'] }),
      ]);
      onOpenChange(false);
    },
    onError: showApiError,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Update Alert Status</DialogTitle>
          <DialogDescription>{alert?.title}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium">New status</label>
            <Select value={nextStatus} onValueChange={setNextStatus}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {ALERT_STATUSES.map((s) => (
                  <SelectItem key={s} value={s}>{s.replace(/_/g, ' ')}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Resolution notes</label>
            <Textarea
              rows={3}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Optional resolution or investigation notes."
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={() => updateMutation.mutate()} disabled={updateMutation.isPending}>
            {updateMutation.isPending ? <Loader2 className="mr-1.5 h-4 w-4 animate-spin" /> : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Main compliance page
// ---------------------------------------------------------------------------

const ALERT_FILTERS = [
  {
    key: 'status',
    label: 'Status',
    type: 'select' as const,
    options: ALERT_STATUSES.map((s) => ({ label: s.replace(/_/g, ' '), value: s })),
  },
  {
    key: 'severity',
    label: 'Severity',
    type: 'select' as const,
    options: SEVERITIES.map((s) => ({ label: s.charAt(0).toUpperCase() + s.slice(1), value: s })),
  },
];

export default function LexCompliancePage() {
  const queryClient = useQueryClient();
  const { hasPermission } = useAuth();
  const canWrite = hasPermission('lex:write');

  const [createRuleOpen, setCreateRuleOpen] = useState(false);
  const [editRuleTarget, setEditRuleTarget] = useState<LexComplianceRule | null>(null);
  const [deleteRuleTarget, setDeleteRuleTarget] = useState<LexComplianceRule | null>(null);
  const [alertTarget, setAlertTarget] = useState<LexComplianceAlert | null>(null);

  const dashboardQuery = useQuery({
    queryKey: ['lex-compliance-dashboard'],
    queryFn: () => enterpriseApi.lex.getComplianceDashboard(),
  });

  const rulesQuery = useQuery({
    queryKey: ['lex-compliance-rules'],
    queryFn: () => enterpriseApi.lex.listComplianceRules({ page: 1, per_page: 100, order: 'asc' }),
  });

  const { tableProps } = useDataTable<LexComplianceAlert>({
    queryKey: 'lex-compliance-alerts',
    fetchFn: (params) => fetchSuitePaginated<LexComplianceAlert>(API_ENDPOINTS.LEX_COMPLIANCE_ALERTS, params),
    defaultPageSize: 25,
    defaultSort: { column: 'created_at', direction: 'desc' },
  });

  const deleteRuleMutation = useMutation({
    mutationFn: (id: string) => enterpriseApi.lex.deleteComplianceRule(id),
    onSuccess: async () => {
      showSuccess('Rule deleted.', 'The compliance rule has been removed.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['lex-compliance-rules'] }),
        queryClient.invalidateQueries({ queryKey: ['lex-compliance-dashboard'] }),
      ]);
      setDeleteRuleTarget(null);
    },
    onError: showApiError,
  });

  const runMutation = useMutation({
    mutationFn: () => enterpriseApi.lex.runCompliance({}),
    onSuccess: async (result) => {
      showSuccess(
        'Compliance check complete.',
        `${result.alerts_created} new alert${result.alerts_created === 1 ? '' : 's'} created. Score: ${Math.round(result.score)}%.`,
      );
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['lex-compliance-alerts'] }),
        queryClient.invalidateQueries({ queryKey: ['lex-compliance-dashboard'] }),
        queryClient.invalidateQueries({ queryKey: ['lex-overview'] }),
      ]);
    },
    onError: showApiError,
  });

  if (dashboardQuery.isLoading && rulesQuery.isLoading) {
    return (
      <PermissionRedirect permission="lex:read">
        <div className="space-y-6">
          <PageHeader title="Compliance" description="Regulatory compliance tracking" />
          <LoadingSkeleton variant="card" count={4} />
        </div>
      </PermissionRedirect>
    );
  }

  if (dashboardQuery.isError && rulesQuery.isError) {
    return (
      <PermissionRedirect permission="lex:read">
        <ErrorState
          message="Failed to load compliance posture."
          onRetry={() => { void dashboardQuery.refetch(); void rulesQuery.refetch(); }}
        />
      </PermissionRedirect>
    );
  }

  const dashboard = dashboardQuery.data;
  const rules = rulesQuery.data?.data ?? [];
  const enabledRules = rules.filter((r) => r.enabled).length;

  const alertColumns: ColumnDef<LexComplianceAlert>[] = [
    {
      id: 'title',
      accessorKey: 'title',
      header: 'Alert',
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.title}</p>
          <p className="text-xs text-muted-foreground line-clamp-1">{row.original.description}</p>
        </div>
      ),
    },
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }) => <SeverityIndicator severity={normalizeSeverity(row.original.severity)} size="sm" />,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => (
        <Badge variant="outline" className="capitalize">
          {row.original.status.replace(/_/g, ' ')}
        </Badge>
      ),
    },
    {
      id: 'created_at',
      accessorKey: 'created_at',
      header: 'Created',
      enableSorting: true,
      cell: ({ row }) => <RelativeTime date={row.original.created_at} />,
    },
    ...(canWrite
      ? [
          {
            id: 'actions',
            header: '',
            cell: ({ row }: { row: { original: LexComplianceAlert } }) => (
              <Button variant="ghost" size="sm" onClick={() => setAlertTarget(row.original)}>
                Update
              </Button>
            ),
          } satisfies ColumnDef<LexComplianceAlert>,
        ]
      : []),
  ];

  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader
          title="Compliance"
          description="Rule coverage, active alerts, and compliance score from the live lex-service compliance engine."
          actions={
            canWrite ? (
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => runMutation.mutate()}
                  disabled={runMutation.isPending}
                >
                  {runMutation.isPending
                    ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
                    : <Play className="mr-1.5 h-3.5 w-3.5" />}
                  Run Compliance Check
                </Button>
                <Button size="sm" onClick={() => setCreateRuleOpen(true)}>
                  <Plus className="mr-1.5 h-4 w-4" />
                  New Rule
                </Button>
              </div>
            ) : undefined
          }
        />

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <KpiCard title="Total Rules" value={rules.length} icon={Scale} iconColor="text-blue-600" />
          <KpiCard title="Enabled Rules" value={enabledRules} icon={Scale} iconColor="text-green-600" />
          <KpiCard title="Open Alerts" value={dashboard?.open_alerts ?? 0} icon={Scale} iconColor="text-red-600" />
          <KpiCard
            title="Compliance Score"
            value={`${Math.round(dashboard?.compliance_score ?? 0)}%`}
            icon={Scale}
            iconColor="text-orange-600"
          />
        </div>

        {/* Rule Library */}
        <SectionCard
          title="Regulation Library"
          description="Compliance rules evaluated during automated and manual compliance checks."
        >
          <div className="space-y-2">
            {rules.length === 0 ? (
              <p className="text-sm text-muted-foreground">No compliance rules have been configured.</p>
            ) : (
              rules.map((rule) => (
                <div
                  key={rule.id}
                  className="flex items-start justify-between gap-3 rounded-lg border px-4 py-3"
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <p className="font-medium">{rule.name}</p>
                      {!rule.enabled && (
                        <Badge variant="secondary" className="text-xs">Disabled</Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground capitalize">
                      {rule.rule_type.replace(/_/g, ' ')}
                    </p>
                    <p className="mt-1 text-sm text-muted-foreground">{rule.description}</p>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <SeverityIndicator severity={normalizeSeverity(rule.severity)} size="sm" />
                    {canWrite ? (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-8 w-8">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => setEditRuleTarget(rule)}>
                            <Pencil className="mr-2 h-4 w-4" />
                            Edit
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            className="text-destructive"
                            onClick={() => setDeleteRuleTarget(rule)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    ) : null}
                  </div>
                </div>
              ))
            )}
          </div>
        </SectionCard>

        {/* Compliance Alerts */}
        <SectionCard
          title="Compliance Alerts"
          description="All alerts generated by compliance rules across the contract portfolio."
        >
          <DataTable
            {...tableProps}
            columns={alertColumns}
            filters={ALERT_FILTERS}
            emptyState={{
              icon: Scale,
              title: 'No compliance alerts',
              description: 'No alerts matched the current filters.',
            }}
          />
        </SectionCard>
      </div>

      {/* Dialogs */}
      <RuleFormDialog open={createRuleOpen} onOpenChange={setCreateRuleOpen} />
      {editRuleTarget ? (
        <RuleFormDialog
          open
          rule={editRuleTarget}
          onOpenChange={(o) => { if (!o) setEditRuleTarget(null); }}
        />
      ) : null}
      <ConfirmDialog
        open={deleteRuleTarget !== null}
        onOpenChange={(o) => { if (!o) setDeleteRuleTarget(null); }}
        title="Delete rule"
        description={`Are you sure you want to delete "${deleteRuleTarget?.name}"? Existing alerts from this rule will remain.`}
        confirmLabel="Delete"
        variant="destructive"
        loading={deleteRuleMutation.isPending}
        onConfirm={() => { if (deleteRuleTarget) deleteRuleMutation.mutate(deleteRuleTarget.id); }}
      />
      <AlertStatusDialog
        open={alertTarget !== null}
        alert={alertTarget}
        onOpenChange={(o) => { if (!o) setAlertTarget(null); }}
      />
    </PermissionRedirect>
  );
}

function normalizeSeverity(value: string): 'critical' | 'high' | 'medium' | 'low' | 'info' {
  switch (value) {
    case 'critical':
    case 'high':
    case 'medium':
    case 'low':
      return value;
    default:
      return 'info';
  }
}
