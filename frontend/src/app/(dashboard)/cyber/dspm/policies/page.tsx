'use client';

import { useState } from 'react';
import { type ColumnDef, type Row } from '@tanstack/react-table';
import {
  BookOpen,
  CheckCircle2,
  Plus,
  ShieldAlert,
  XCircle,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { toast } from 'sonner';
import type {
  DSPMDataPolicy,
  DSPMPolicyViolation,
  DSPMPolicyCategory,
  DSPMPolicyEnforcement,
  CyberSeverity,
} from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';

const CATEGORY_COLORS: Record<string, string> = {
  encryption: 'bg-blue-100 text-blue-700',
  classification: 'bg-purple-100 text-purple-700',
  retention: 'bg-amber-100 text-amber-800',
  exposure: 'bg-red-100 text-red-700',
  pii_protection: 'bg-pink-100 text-pink-700',
  access_review: 'bg-teal-100 text-teal-700',
  backup: 'bg-green-100 text-green-700',
  audit_logging: 'bg-indigo-100 text-indigo-700',
};

const ENFORCEMENT_COLORS: Record<string, string> = {
  alert: 'bg-amber-100 text-amber-800',
  auto_remediate: 'bg-blue-100 text-blue-700',
  block: 'bg-red-100 text-red-700',
};

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'bg-red-100 text-red-700',
  high: 'bg-orange-100 text-orange-700',
  medium: 'bg-amber-100 text-amber-800',
  low: 'bg-blue-100 text-blue-700',
  info: 'bg-gray-100 text-gray-600',
};

const POLICY_CATEGORIES: DSPMPolicyCategory[] = [
  'encryption', 'classification', 'retention', 'exposure',
  'pii_protection', 'access_review', 'backup', 'audit_logging',
];

const ENFORCEMENT_OPTIONS: DSPMPolicyEnforcement[] = ['alert', 'auto_remediate', 'block'];
const SEVERITY_OPTIONS: CyberSeverity[] = ['critical', 'high', 'medium', 'low', 'info'];

const policyColumns: ColumnDef<DSPMDataPolicy>[] = [
  {
    id: 'name',
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }: { row: Row<DSPMDataPolicy> }) => (
      <div>
        <p className="text-sm font-medium">{row.original.name}</p>
        {row.original.description && (
          <p className="mt-0.5 text-xs text-muted-foreground line-clamp-1">{row.original.description}</p>
        )}
      </div>
    ),
    enableSorting: true,
  },
  {
    id: 'category',
    accessorKey: 'category',
    header: 'Category',
    cell: ({ row }: { row: Row<DSPMDataPolicy> }) => {
      const cat = row.original.category;
      return (
        <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${CATEGORY_COLORS[cat] ?? 'bg-muted text-muted-foreground'}`}>
          {cat.replace(/_/g, ' ')}
        </span>
      );
    },
    enableSorting: true,
  },
  {
    id: 'enforcement',
    accessorKey: 'enforcement',
    header: 'Enforcement',
    cell: ({ row }: { row: Row<DSPMDataPolicy> }) => {
      const enf = row.original.enforcement;
      return (
        <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${ENFORCEMENT_COLORS[enf] ?? 'bg-muted text-muted-foreground'}`}>
          {enf.replace(/_/g, ' ')}
        </span>
      );
    },
    enableSorting: true,
  },
  {
    id: 'severity',
    accessorKey: 'severity',
    header: 'Severity',
    cell: ({ row }: { row: Row<DSPMDataPolicy> }) => {
      const sev = row.original.severity;
      return (
        <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${SEVERITY_COLORS[sev] ?? 'bg-muted text-muted-foreground'}`}>
          {sev}
        </span>
      );
    },
    enableSorting: true,
  },
  {
    id: 'scope',
    header: 'Scope',
    cell: ({ row }: { row: Row<DSPMDataPolicy> }) => {
      const scopes = row.original.scope_classification ?? [];
      if (scopes.length === 0) return <span className="text-xs text-muted-foreground">All</span>;
      return (
        <div className="flex flex-wrap gap-1">
          {scopes.slice(0, 2).map((s) => (
            <Badge key={s} variant="outline" className="text-xs capitalize px-1.5 py-0">{s}</Badge>
          ))}
          {scopes.length > 2 && <Badge variant="outline" className="text-xs px-1.5 py-0">+{scopes.length - 2}</Badge>}
        </div>
      );
    },
  },
  {
    id: 'enabled',
    accessorKey: 'enabled',
    header: 'Enabled',
    cell: ({ row }: { row: Row<DSPMDataPolicy> }) => (
      row.original.enabled
        ? <CheckCircle2 className="h-4 w-4 text-green-500" />
        : <XCircle className="h-4 w-4 text-muted-foreground" />
    ),
  },
  {
    id: 'violation_count',
    accessorKey: 'violation_count',
    header: 'Violations',
    cell: ({ row }: { row: Row<DSPMDataPolicy> }) => {
      const count = row.original.violation_count;
      return (
        <span className={`text-sm font-medium tabular-nums ${count > 0 ? 'text-red-600' : 'text-green-600'}`}>
          {count}
        </span>
      );
    },
    enableSorting: true,
  },
  {
    id: 'last_evaluated_at',
    accessorKey: 'last_evaluated_at',
    header: 'Last Evaluated',
    cell: ({ row }: { row: Row<DSPMDataPolicy> }) => {
      const dt = row.original.last_evaluated_at;
      return (
        <span className="text-xs text-muted-foreground">
          {dt ? new Date(dt).toLocaleDateString() : 'Never'}
        </span>
      );
    },
    enableSorting: true,
  },
];

interface CreatePolicyForm {
  name: string;
  description: string;
  category: DSPMPolicyCategory;
  enforcement: DSPMPolicyEnforcement;
  severity: CyberSeverity;
  enabled: boolean;
}

const INITIAL_FORM: CreatePolicyForm = {
  name: '',
  description: '',
  category: 'encryption',
  enforcement: 'alert',
  severity: 'medium',
  enabled: true,
};

export default function DataPoliciesPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState<CreatePolicyForm>(INITIAL_FORM);
  const [creating, setCreating] = useState(false);

  const { tableProps, refetch } = useDataTable<DSPMDataPolicy>({
    queryKey: 'cyber-dspm-policies',
    fetchFn: (params: FetchParams) =>
      apiGet<PaginatedResponse<DSPMDataPolicy>>(API_ENDPOINTS.CYBER_DSPM_DATA_POLICIES, params as unknown as Record<string, unknown>),
    defaultSort: { column: 'name', direction: 'asc' },
  });

  const {
    data: violationsEnvelope,
    isLoading: violationsLoading,
    error: violationsError,
    mutate: refetchViolations,
  } = useRealtimeData<{ data: DSPMPolicyViolation[] }>(API_ENDPOINTS.CYBER_DSPM_POLICY_VIOLATIONS, {
    pollInterval: 120000,
  });

  const violations = violationsEnvelope?.data ?? [];
  const totalPolicies = tableProps.totalRows;
  const enabledCount = tableProps.data?.filter((p) => p.enabled).length ?? 0;
  const totalViolations = violations.length;

  const filters = [
    {
      key: 'category',
      label: 'Category',
      type: 'multi-select' as const,
      options: POLICY_CATEGORIES.map((c) => ({
        label: c.replace(/_/g, ' ').replace(/\b\w/g, (x) => x.toUpperCase()),
        value: c,
      })),
    },
    {
      key: 'enforcement',
      label: 'Enforcement',
      type: 'multi-select' as const,
      options: ENFORCEMENT_OPTIONS.map((e) => ({
        label: e.replace(/_/g, ' ').replace(/\b\w/g, (x) => x.toUpperCase()),
        value: e,
      })),
    },
    {
      key: 'enabled',
      label: 'Enabled',
      type: 'multi-select' as const,
      options: [
        { label: 'Enabled', value: 'true' },
        { label: 'Disabled', value: 'false' },
      ],
    },
  ];

  async function handleCreatePolicy() {
    if (!form.name.trim()) {
      toast.error('Policy name is required');
      return;
    }
    setCreating(true);
    try {
      await apiPost(API_ENDPOINTS.CYBER_DSPM_DATA_POLICIES, {
        name: form.name,
        description: form.description,
        category: form.category,
        enforcement: form.enforcement,
        severity: form.severity,
        enabled: form.enabled,
        rule: {},
      });
      toast.success('Policy created');
      setCreateOpen(false);
      setForm(INITIAL_FORM);
      refetch();
    } catch {
      toast.error('Failed to create policy');
    } finally {
      setCreating(false);
    }
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Data Policies"
          description="Define and enforce data security policies across your organization"
          actions={
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              Create Policy
            </Button>
          }
        />

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Card>
            <CardContent className="flex items-center gap-4 p-5">
              <BookOpen className="h-5 w-5 text-blue-600" />
              <div>
                <p className="text-xs text-muted-foreground">Total Policies</p>
                <p className="text-2xl font-bold tabular-nums">{totalPolicies}</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-4 p-5">
              <CheckCircle2 className="h-5 w-5 text-green-600" />
              <div>
                <p className="text-xs text-muted-foreground">Enabled</p>
                <p className="text-2xl font-bold tabular-nums">{enabledCount}</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-4 p-5">
              <ShieldAlert className="h-5 w-5 text-red-600" />
              <div>
                <p className="text-xs text-muted-foreground">Active Violations</p>
                <p className="text-2xl font-bold tabular-nums">{totalViolations}</p>
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="rounded-xl border bg-card">
          <div className="border-b px-5 py-4">
            <h3 className="text-sm font-semibold">Policy Catalog</h3>
            <p className="text-xs text-muted-foreground">All data security policies with their enforcement configuration</p>
          </div>
          <div className="p-5">
            {tableProps.isLoading ? (
              <LoadingSkeleton variant="table-row" count={6} />
            ) : tableProps.error ? (
              <ErrorState message="Failed to load policies" onRetry={refetch} />
            ) : (
              <DataTable
                {...tableProps}
                columns={policyColumns}
                filters={filters}
                onSortChange={() => undefined}
                searchPlaceholder="Search policies..."
                emptyState={{
                  icon: BookOpen,
                  title: 'No policies defined',
                  description: 'Create your first data security policy to start enforcing controls.',
                  action: { label: 'Create Policy', onClick: () => setCreateOpen(true) },
                }}
              />
            )}
          </div>
        </div>

        <div className="rounded-xl border bg-card">
          <div className="border-b px-5 py-4">
            <h3 className="text-sm font-semibold">Current Violations</h3>
            <p className="text-xs text-muted-foreground">Active policy violations across data assets</p>
          </div>
          <div className="p-5">
            {violationsLoading ? (
              <LoadingSkeleton variant="table-row" count={4} />
            ) : violationsError ? (
              <ErrorState message="Failed to load violations" onRetry={() => void refetchViolations()} />
            ) : violations.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <CheckCircle2 className="mb-3 h-8 w-8 text-green-500" />
                <p className="text-sm font-medium">No Active Violations</p>
                <p className="text-xs text-muted-foreground">All data assets are compliant with defined policies.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {violations.slice(0, 20).map((v, idx) => (
                  <div key={`${v.policy_id}-${v.asset_id}-${idx}`} className="flex items-start justify-between rounded-lg border p-3">
                    <div className="space-y-1">
                      <p className="text-sm font-medium">{v.policy_name}</p>
                      <p className="text-xs text-muted-foreground">{v.description}</p>
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className="text-xs">{v.asset_name}</Badge>
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize ${CATEGORY_COLORS[v.category] ?? 'bg-muted text-muted-foreground'}`}>
                          {v.category.replace(/_/g, ' ')}
                        </span>
                      </div>
                    </div>
                    <div className="flex flex-col items-end gap-1">
                      <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${SEVERITY_COLORS[v.severity] ?? 'bg-muted text-muted-foreground'}`}>
                        {v.severity}
                      </span>
                      <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize ${ENFORCEMENT_COLORS[v.enforcement] ?? 'bg-muted text-muted-foreground'}`}>
                        {v.enforcement.replace(/_/g, ' ')}
                      </span>
                    </div>
                  </div>
                ))}
                {violations.length > 20 && (
                  <p className="text-center text-xs text-muted-foreground">
                    Showing 20 of {violations.length} violations
                  </p>
                )}
              </div>
            )}
          </div>
        </div>

        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>Create Data Policy</DialogTitle>
              <DialogDescription>
                Define a new data security policy with enforcement rules.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-2">
              <div className="space-y-2">
                <Label htmlFor="policy-name">Name</Label>
                <Input
                  id="policy-name"
                  placeholder="e.g. Require encryption at rest"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="policy-desc">Description</Label>
                <Input
                  id="policy-desc"
                  placeholder="What does this policy enforce?"
                  value={form.description}
                  onChange={(e) => setForm({ ...form, description: e.target.value })}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Category</Label>
                  <Select value={form.category} onValueChange={(v) => setForm({ ...form, category: v as DSPMPolicyCategory })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {POLICY_CATEGORIES.map((c) => (
                        <SelectItem key={c} value={c}>
                          {c.replace(/_/g, ' ').replace(/\b\w/g, (x) => x.toUpperCase())}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>Enforcement</Label>
                  <Select value={form.enforcement} onValueChange={(v) => setForm({ ...form, enforcement: v as DSPMPolicyEnforcement })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {ENFORCEMENT_OPTIONS.map((e) => (
                        <SelectItem key={e} value={e}>
                          {e.replace(/_/g, ' ').replace(/\b\w/g, (x) => x.toUpperCase())}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Severity</Label>
                  <Select value={form.severity} onValueChange={(v) => setForm({ ...form, severity: v as CyberSeverity })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {SEVERITY_OPTIONS.map((s) => (
                        <SelectItem key={s} value={s}>
                          {s.charAt(0).toUpperCase() + s.slice(1)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex items-end gap-3 pb-1">
                  <Label htmlFor="policy-enabled" className="pb-0.5">Enabled</Label>
                  <Switch
                    id="policy-enabled"
                    checked={form.enabled}
                    onCheckedChange={(checked) => setForm({ ...form, enabled: checked })}
                  />
                </div>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setCreateOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreatePolicy} disabled={creating}>
                {creating ? 'Creating...' : 'Create Policy'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </PermissionRedirect>
  );
}
