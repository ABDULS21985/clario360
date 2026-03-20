'use client';

import { useState, useMemo } from 'react';
import {
  Plus,
  Scale,
  ShieldCheck,
  GitBranch,
  Eye,
  Edit,
  ClipboardCheck,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { StatusBadge } from '@/components/shared/status-badge';
import { SeverityIndicator, type Severity } from '@/components/shared/severity-indicator';
import {
  obligationStatusConfig,
  controlTestResultConfig,
} from '@/lib/status-configs';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate, titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { ColumnDef } from '@tanstack/react-table';
import type { PaginatedResponse } from '@/types/api';
import type { FilterConfig } from '@/types/table';
import type {
  VCISORegulatoryObligation,
  VCISOControlTest,
  VCISOControlDependency,
  ControlFailureImpact,
} from '@/types/cyber';

import { ObligationFormDialog } from './_components/obligation-form-dialog';
import { ObligationDetailPanel } from './_components/obligation-detail-panel';
import { ControlTestFormDialog } from './_components/control-test-form-dialog';
import { DependencyDetailPanel } from './_components/dependency-detail-panel';

// ─── Failure Impact → Severity mapping ───────────────────────────────────────

const impactToSeverity: Record<ControlFailureImpact, Severity> = {
  critical: 'critical',
  high: 'high',
  medium: 'medium',
  low: 'low',
};

// ─── Regulatory Obligations Tab ──────────────────────────────────────────────

function ObligationsTab({ onCreateObligation }: { onCreateObligation: () => void }) {
  const [detailObligation, setDetailObligation] = useState<VCISORegulatoryObligation | null>(null);
  const [editObligation, setEditObligation] = useState<VCISORegulatoryObligation | null>(null);

  const table = useDataTable<VCISORegulatoryObligation>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISORegulatoryObligation>>(
        API_ENDPOINTS.CYBER_VCISO_OBLIGATIONS,
        params,
      ),
    queryKey: 'vciso-obligations',
    defaultSort: { column: 'name', direction: 'asc' },
    wsTopics: ['vciso.obligations'],
  });

  const filters: FilterConfig[] = [
    {
      key: 'type',
      label: 'Type',
      type: 'select',
      options: [
        { label: 'Legal', value: 'legal' },
        { label: 'Regulatory', value: 'regulatory' },
        { label: 'Contractual', value: 'contractual' },
        { label: 'Industry Standard', value: 'industry_standard' },
      ],
    },
    {
      key: 'status',
      label: 'Status',
      type: 'select',
      options: [
        { label: 'Compliant', value: 'compliant' },
        { label: 'Partially Compliant', value: 'partially_compliant' },
        { label: 'Non-Compliant', value: 'non_compliant' },
        { label: 'Not Assessed', value: 'not_assessed' },
      ],
    },
  ];

  const columns: ColumnDef<VCISORegulatoryObligation>[] = [
    {
      id: 'name',
      header: 'Name',
      accessorKey: 'name',
      enableSorting: true,
      cell: ({ row }) => (
        <button
          className="font-semibold text-sm hover:underline text-left max-w-[180px] sm:max-w-[280px] truncate block"
          onClick={(e) => {
            e.stopPropagation();
            setDetailObligation(row.original);
          }}
        >
          {row.original.name}
        </button>
      ),
    },
    {
      id: 'type',
      header: 'Type',
      accessorKey: 'type',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge variant="outline">{titleCase(row.original.type)}</Badge>
      ),
    },
    {
      id: 'jurisdiction',
      header: 'Jurisdiction',
      accessorKey: 'jurisdiction',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{row.original.jurisdiction}</span>
      ),
    },
    {
      id: 'status',
      header: 'Status',
      accessorKey: 'status',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge
          status={row.original.status}
          config={obligationStatusConfig}
          size="sm"
        />
      ),
    },
    {
      id: 'requirements_met',
      header: 'Requirements Met',
      accessorKey: 'met_requirements',
      enableSorting: false,
      cell: ({ row }) => {
        const { met_requirements, total_requirements } = row.original;
        const percent =
          total_requirements > 0
            ? Math.round((met_requirements / total_requirements) * 100)
            : 0;
        return (
          <div className="flex items-center gap-2 min-w-[140px]">
            <span className="text-sm font-medium whitespace-nowrap">
              {met_requirements}/{total_requirements}
            </span>
            <Progress value={percent} className="h-1.5 flex-1" />
          </div>
        );
      },
    },
    {
      id: 'owner_name',
      header: 'Owner',
      accessorKey: 'owner_name',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.owner_name ?? <span className="text-muted-foreground">Unassigned</span>}
        </span>
      ),
    },
    {
      id: 'review_date',
      header: 'Review Date',
      accessorKey: 'review_date',
      enableSorting: true,
      cell: ({ row }) => {
        const isOverdue = new Date(row.original.review_date) < new Date();
        return (
          <span
            className={cn(
              'text-sm',
              isOverdue && 'text-red-600 font-medium',
            )}
          >
            {formatDate(row.original.review_date)}
          </span>
        );
      },
    },
  ];

  const rowActions = (obligation: VCISORegulatoryObligation) => [
    {
      label: 'View Details',
      icon: Eye,
      onClick: (o: VCISORegulatoryObligation) => setDetailObligation(o),
    },
    {
      label: 'Edit',
      icon: Edit,
      onClick: (o: VCISORegulatoryObligation) => setEditObligation(o),
    },
  ];

  return (
    <>
      <DataTable
        {...table.tableProps}
        columns={columns}
        filters={filters}
        rowActions={rowActions}
        onRowClick={(obligation) => setDetailObligation(obligation)}
        searchPlaceholder="Search obligations..."
        searchSlot={
          <SearchInput
            value={table.tableProps.searchValue ?? ''}
            onChange={table.tableProps.onSearchChange ?? (() => undefined)}
            placeholder="Search obligations..."
            loading={table.tableProps.isLoading}
          />
        }
        emptyState={{
          icon: Scale,
          title: 'No obligations found',
          description: 'Add your first regulatory obligation to start tracking compliance.',
          action: {
            label: 'Add Obligation',
            onClick: onCreateObligation,
            icon: Plus,
          },
        }}
      />

      {/* Detail Panel */}
      {detailObligation && (
        <ObligationDetailPanel
          obligation={detailObligation}
          open={!!detailObligation}
          onClose={() => setDetailObligation(null)}
          onEdit={() => {
            setEditObligation(detailObligation);
            setDetailObligation(null);
          }}
        />
      )}

      {/* Edit Dialog */}
      {editObligation && (
        <ObligationFormDialog
          open={!!editObligation}
          onOpenChange={(o) => !o && setEditObligation(null)}
          obligation={editObligation}
          onSuccess={() => table.refetch()}
        />
      )}
    </>
  );
}

// ─── Control Testing Tab ─────────────────────────────────────────────────────

function ControlTestingTab({ onRecordTest }: { onRecordTest: () => void }) {
  const [detailTest, setDetailTest] = useState<VCISOControlTest | null>(null);
  const [recordTestOpen, setRecordTestOpen] = useState(false);

  const table = useDataTable<VCISOControlTest>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOControlTest>>(
        API_ENDPOINTS.CYBER_VCISO_CONTROL_TESTS,
        params,
      ),
    queryKey: 'vciso-control-tests',
    defaultSort: { column: 'test_date', direction: 'desc' },
    wsTopics: ['vciso.control-tests'],
  });

  const filters: FilterConfig[] = [
    {
      key: 'result',
      label: 'Result',
      type: 'select',
      options: [
        { label: 'Effective', value: 'effective' },
        { label: 'Partially Effective', value: 'partially_effective' },
        { label: 'Ineffective', value: 'ineffective' },
        { label: 'Not Tested', value: 'not_tested' },
      ],
    },
    {
      key: 'test_type',
      label: 'Test Type',
      type: 'select',
      options: [
        { label: 'Design', value: 'design' },
        { label: 'Operating Effectiveness', value: 'operating_effectiveness' },
      ],
    },
  ];

  const columns: ColumnDef<VCISOControlTest>[] = [
    {
      id: 'control_name',
      header: 'Control Name',
      accessorKey: 'control_name',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="font-semibold text-sm max-w-[140px] sm:max-w-[240px] truncate block">
          {row.original.control_name}
        </span>
      ),
    },
    {
      id: 'framework',
      header: 'Framework',
      accessorKey: 'framework',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge variant="outline">{row.original.framework}</Badge>
      ),
    },
    {
      id: 'test_type',
      header: 'Test Type',
      accessorKey: 'test_type',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge variant="secondary">{titleCase(row.original.test_type)}</Badge>
      ),
    },
    {
      id: 'result',
      header: 'Result',
      accessorKey: 'result',
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge
          status={row.original.result}
          config={controlTestResultConfig}
          size="sm"
        />
      ),
    },
    {
      id: 'tester_name',
      header: 'Tester',
      accessorKey: 'tester_name',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">{row.original.tester_name}</span>
      ),
    },
    {
      id: 'test_date',
      header: 'Test Date',
      accessorKey: 'test_date',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {formatDate(row.original.test_date)}
        </span>
      ),
    },
    {
      id: 'next_test_date',
      header: 'Next Test Date',
      accessorKey: 'next_test_date',
      enableSorting: true,
      cell: ({ row }) => {
        const isPast =
          row.original.next_test_date &&
          new Date(row.original.next_test_date) < new Date();
        return (
          <span
            className={cn(
              'text-sm',
              isPast ? 'text-red-600 font-medium' : 'text-muted-foreground',
            )}
          >
            {row.original.next_test_date
              ? formatDate(row.original.next_test_date)
              : '\u2014'}
          </span>
        );
      },
    },
  ];

  const rowActions = (test: VCISOControlTest) => [
    {
      label: 'View Details',
      icon: Eye,
      onClick: (t: VCISOControlTest) => setDetailTest(t),
    },
    {
      label: 'Record New Test',
      icon: ClipboardCheck,
      onClick: () => setRecordTestOpen(true),
    },
  ];

  return (
    <>
      <DataTable
        {...table.tableProps}
        columns={columns}
        filters={filters}
        rowActions={rowActions}
        onRowClick={(test) => setDetailTest(test)}
        searchPlaceholder="Search control tests..."
        searchSlot={
          <SearchInput
            value={table.tableProps.searchValue ?? ''}
            onChange={table.tableProps.onSearchChange ?? (() => undefined)}
            placeholder="Search control tests..."
            loading={table.tableProps.isLoading}
          />
        }
        emptyState={{
          icon: ShieldCheck,
          title: 'No control tests found',
          description: 'Record your first control effectiveness test.',
          action: {
            label: 'Record Test',
            onClick: onRecordTest,
            icon: Plus,
          },
        }}
      />

      {/* Test Detail Panel */}
      {detailTest && (
        <ControlTestDetailView
          test={detailTest}
          open={!!detailTest}
          onClose={() => setDetailTest(null)}
        />
      )}

      {/* Record New Test from row action */}
      <ControlTestFormDialog
        open={recordTestOpen}
        onOpenChange={setRecordTestOpen}
        onSuccess={() => table.refetch()}
      />
    </>
  );
}

// ─── Control Test Detail View (inline) ───────────────────────────────────────

import { DetailPanel } from '@/components/shared/detail-panel';
import { Separator } from '@/components/ui/separator';

function ControlTestDetailView({
  test,
  open,
  onClose,
}: {
  test: VCISOControlTest;
  open: boolean;
  onClose: () => void;
}) {
  const isPastDue =
    test.next_test_date && new Date(test.next_test_date) < new Date();

  return (
    <DetailPanel
      open={open}
      onOpenChange={(o) => !o && onClose()}
      title={test.control_name}
      description={`Framework: ${test.framework}`}
      width="xl"
    >
      <div className="space-y-6">
        {/* Result */}
        <div className="flex items-center justify-between">
          <StatusBadge
            status={test.result}
            config={controlTestResultConfig}
            size="lg"
          />
          <Badge variant="secondary">{titleCase(test.test_type)}</Badge>
        </div>

        <Separator />

        {/* Metadata */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Framework
            </p>
            <Badge variant="outline">{test.framework}</Badge>
          </div>

          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Test Type
            </p>
            <p className="text-sm">{titleCase(test.test_type)}</p>
          </div>

          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Tester
            </p>
            <p className="text-sm">{test.tester_name}</p>
          </div>

          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Test Date
            </p>
            <p className="text-sm">{formatDate(test.test_date)}</p>
          </div>

          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Next Test Date
            </p>
            <p
              className={cn(
                'text-sm',
                isPastDue && 'text-red-600 font-medium',
              )}
            >
              {test.next_test_date
                ? formatDate(test.next_test_date)
                : '\u2014'}
              {isPastDue && ' (Overdue)'}
            </p>
          </div>

          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Evidence Items
            </p>
            <p className="text-sm">{test.evidence_ids?.length ?? 0}</p>
          </div>
        </div>

        <Separator />

        {/* Findings */}
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-foreground">Findings</h3>
          {test.findings ? (
            <div className="rounded-lg border border-border bg-muted/30 p-4">
              <p className="text-sm whitespace-pre-wrap">{test.findings}</p>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              No findings recorded.
            </p>
          )}
        </div>
      </div>
    </DetailPanel>
  );
}

// ─── Control Dependencies Tab ────────────────────────────────────────────────

function ControlDependenciesTab() {
  const [detailDependency, setDetailDependency] = useState<VCISOControlDependency | null>(null);

  const table = useDataTable<VCISOControlDependency>({
    fetchFn: (params) =>
      apiGet<PaginatedResponse<VCISOControlDependency>>(
        API_ENDPOINTS.CYBER_VCISO_CONTROL_DEPENDENCIES,
        params,
      ),
    queryKey: 'vciso-control-dependencies',
    defaultSort: { column: 'control_name', direction: 'asc' },
    wsTopics: ['vciso.control-dependencies'],
  });

  const columns: ColumnDef<VCISOControlDependency>[] = [
    {
      id: 'control_name',
      header: 'Control Name',
      accessorKey: 'control_name',
      enableSorting: true,
      cell: ({ row }) => (
        <button
          className="font-semibold text-sm hover:underline text-left max-w-[160px] sm:max-w-[260px] truncate block"
          onClick={(e) => {
            e.stopPropagation();
            setDetailDependency(row.original);
          }}
        >
          {row.original.control_name}
        </button>
      ),
    },
    {
      id: 'framework',
      header: 'Framework',
      accessorKey: 'framework',
      enableSorting: true,
      cell: ({ row }) => (
        <Badge variant="outline">{row.original.framework}</Badge>
      ),
    },
    {
      id: 'failure_impact',
      header: 'Failure Impact',
      accessorKey: 'failure_impact',
      enableSorting: true,
      cell: ({ row }) => (
        <SeverityIndicator
          severity={impactToSeverity[row.original.failure_impact]}
          size="sm"
        />
      ),
    },
    {
      id: 'depends_on',
      header: 'Depends On',
      accessorKey: 'depends_on',
      enableSorting: false,
      cell: ({ row }) => (
        <Badge variant="secondary">
          {row.original.depends_on.length}
        </Badge>
      ),
    },
    {
      id: 'depended_by',
      header: 'Depended By',
      accessorKey: 'depended_by',
      enableSorting: false,
      cell: ({ row }) => (
        <Badge variant="secondary">
          {row.original.depended_by.length}
        </Badge>
      ),
    },
    {
      id: 'risk_domains',
      header: 'Risk Domains',
      accessorKey: 'risk_domains',
      enableSorting: false,
      cell: ({ row }) => {
        const domains = row.original.risk_domains;
        const displayed = domains.slice(0, 2);
        const remaining = domains.length - displayed.length;
        return (
          <div className="flex items-center gap-1">
            {displayed.map((d) => (
              <Badge key={d} variant="secondary" className="text-xs">
                {titleCase(d)}
              </Badge>
            ))}
            {remaining > 0 && (
              <span className="text-xs text-muted-foreground">
                +{remaining}
              </span>
            )}
          </div>
        );
      },
    },
    {
      id: 'compliance_domains',
      header: 'Compliance Domains',
      accessorKey: 'compliance_domains',
      enableSorting: false,
      cell: ({ row }) => {
        const domains = row.original.compliance_domains;
        const displayed = domains.slice(0, 2);
        const remaining = domains.length - displayed.length;
        return (
          <div className="flex items-center gap-1">
            {displayed.map((d) => (
              <Badge key={d} variant="outline" className="text-xs">
                {titleCase(d)}
              </Badge>
            ))}
            {remaining > 0 && (
              <span className="text-xs text-muted-foreground">
                +{remaining}
              </span>
            )}
          </div>
        );
      },
    },
  ];

  return (
    <>
      <DataTable
        {...table.tableProps}
        columns={columns}
        onRowClick={(dep) => setDetailDependency(dep)}
        searchPlaceholder="Search dependencies..."
        searchSlot={
          <SearchInput
            value={table.tableProps.searchValue ?? ''}
            onChange={table.tableProps.onSearchChange ?? (() => undefined)}
            placeholder="Search dependencies..."
            loading={table.tableProps.isLoading}
          />
        }
        getRowId={(row) => row.control_id}
        emptyState={{
          icon: GitBranch,
          title: 'No control dependencies found',
          description: 'Control dependency mappings will appear here once configured.',
        }}
      />

      {/* Detail Panel */}
      {detailDependency && (
        <DependencyDetailPanel
          dependency={detailDependency}
          open={!!detailDependency}
          onClose={() => setDetailDependency(null)}
        />
      )}
    </>
  );
}

// ─── Main Page ───────────────────────────────────────────────────────────────

export default function VCISOCompliancePage() {
  const [activeTab, setActiveTab] = useState('obligations');
  const [createObligationOpen, setCreateObligationOpen] = useState(false);
  const [createTestOpen, setCreateTestOpen] = useState(false);

  const headerActions = useMemo(() => {
    if (activeTab === 'obligations') {
      return (
        <Button onClick={() => setCreateObligationOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add Obligation
        </Button>
      );
    }
    if (activeTab === 'testing') {
      return (
        <Button onClick={() => setCreateTestOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Record Test
        </Button>
      );
    }
    return null;
  }, [activeTab]);

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Compliance Management"
          description="Track regulatory obligations, test control effectiveness, and map control dependencies across your compliance program."
          actions={headerActions}
        />

        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="obligations" className="gap-1.5">
              <Scale className="h-4 w-4" />
              Regulatory Obligations
            </TabsTrigger>
            <TabsTrigger value="testing" className="gap-1.5">
              <ShieldCheck className="h-4 w-4" />
              Control Testing
            </TabsTrigger>
            <TabsTrigger value="dependencies" className="gap-1.5">
              <GitBranch className="h-4 w-4" />
              Control Dependencies
            </TabsTrigger>
          </TabsList>

          <TabsContent value="obligations" className="mt-6">
            <ObligationsTab
              onCreateObligation={() => setCreateObligationOpen(true)}
            />
          </TabsContent>

          <TabsContent value="testing" className="mt-6">
            <ControlTestingTab
              onRecordTest={() => setCreateTestOpen(true)}
            />
          </TabsContent>

          <TabsContent value="dependencies" className="mt-6">
            <ControlDependenciesTab />
          </TabsContent>
        </Tabs>

        {/* Create Obligation Dialog */}
        <ObligationFormDialog
          open={createObligationOpen}
          onOpenChange={setCreateObligationOpen}
          onSuccess={() => {}}
        />

        {/* Create Test Dialog */}
        <ControlTestFormDialog
          open={createTestOpen}
          onOpenChange={setCreateTestOpen}
          onSuccess={() => {}}
        />
      </div>
    </PermissionRedirect>
  );
}
