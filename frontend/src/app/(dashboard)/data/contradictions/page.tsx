'use client';

import { useState } from 'react';
import { AlertTriangle } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { Button } from '@/components/ui/button';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDataTable } from '@/hooks/use-data-table';
import { buildContradictionColumns } from '@/app/(dashboard)/data/contradictions/_components/contradiction-columns';
import { ContradictionDetailPanel } from '@/app/(dashboard)/data/contradictions/_components/contradiction-detail-panel';
import { ContradictionResolveDialog } from '@/app/(dashboard)/data/contradictions/_components/contradiction-resolve-dialog';
import { ContradictionScanDialog } from '@/app/(dashboard)/data/contradictions/_components/contradiction-scan-dialog';
import { ContradictionStatBar } from '@/app/(dashboard)/data/contradictions/_components/contradiction-stat-bar';
import { dataSuiteApi, type Contradiction } from '@/lib/data-suite';
import type { ContradictionResolutionValues } from '@/lib/data-suite/forms';
import { showApiError, showSuccess } from '@/lib/toast';

const CONTRADICTION_FILTERS = [
  {
    key: 'type',
    label: 'Type',
    type: 'multi-select' as const,
    options: [
      { label: 'Logical', value: 'logical' },
      { label: 'Semantic', value: 'semantic' },
      { label: 'Temporal', value: 'temporal' },
      { label: 'Analytical', value: 'analytical' },
    ],
  },
  {
    key: 'severity',
    label: 'Severity',
    type: 'multi-select' as const,
    options: [
      { label: 'Critical', value: 'critical' },
      { label: 'High', value: 'high' },
      { label: 'Medium', value: 'medium' },
      { label: 'Low', value: 'low' },
    ],
  },
];

export default function DataContradictionsPage() {
  const [selected, setSelected] = useState<Contradiction | null>(null);
  const [resolving, setResolving] = useState<Contradiction | null>(null);
  const [scanOpen, setScanOpen] = useState(false);
  const [submittingResolve, setSubmittingResolve] = useState(false);

  const statsQuery = useQuery({
    queryKey: ['data-contradictions-stats'],
    queryFn: () => dataSuiteApi.getContradictionStats(),
  });

  const { tableProps, searchValue, setSearch, refetch } = useDataTable<Contradiction>({
    queryKey: 'data-contradictions',
    fetchFn: (params) => dataSuiteApi.listContradictions(params),
    defaultPageSize: 25,
    defaultSort: { column: 'created_at', direction: 'desc' },
    wsTopics: ['contradiction.detected'],
  });

  const updateStatus = async (contradiction: Contradiction, status: string) => {
    try {
      await dataSuiteApi.updateContradictionStatus(contradiction.id, status);
      showSuccess('Contradiction updated.');
      void refetch();
      void statsQuery.refetch();
    } catch (error) {
      showApiError(error);
    }
  };

  const resolveContradiction = async (values: ContradictionResolutionValues) => {
    if (!resolving) {
      return;
    }
    try {
      setSubmittingResolve(true);
      await dataSuiteApi.resolveContradiction(resolving.id, values);
      showSuccess('Contradiction resolved.');
      setResolving(null);
      setSelected(null);
      void refetch();
      void statsQuery.refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setSubmittingResolve(false);
    }
  };

  if (statsQuery.isLoading) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Contradictions" description="Loading contradiction telemetry and active investigation queue." />
          <LoadingSkeleton variant="card" />
        </div>
      </PermissionRedirect>
    );
  }

  if (statsQuery.error || !statsQuery.data) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message="Failed to load contradiction statistics." onRetry={() => void statsQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Contradictions"
          description="Cross-source inconsistency detection, investigation workflow, and live scan orchestration."
          actions={
            <Button type="button" onClick={() => setScanOpen(true)}>
              Scan now
            </Button>
          }
        />

        <ContradictionStatBar
          stats={statsQuery.data}
          activeStatus={tableProps.activeFilters?.status}
          onFilterStatus={(status) => tableProps.onFilterChange?.('status', status)}
        />

        <DataTable
          {...tableProps}
          columns={buildContradictionColumns({ onOpen: setSelected })}
          filters={CONTRADICTION_FILTERS}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search contradictions..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: AlertTriangle,
            title: 'No contradictions found',
            description: 'No contradictions matched the current filters.',
          }}
        />

        <ContradictionDetailPanel
          open={Boolean(selected)}
          onOpenChange={(open) => {
            if (!open) {
              setSelected(null);
            }
          }}
          contradiction={selected}
          onInvestigate={(item) => void updateStatus(item, 'investigating')}
          onAccept={(item) => void updateStatus(item, 'accepted')}
          onResolve={(item) => setResolving(item)}
          onFalsePositive={(item) => void updateStatus(item, 'false_positive')}
        />

        <ContradictionResolveDialog
          open={Boolean(resolving)}
          onOpenChange={(open) => {
            if (!open) {
              setResolving(null);
            }
          }}
          contradiction={resolving}
          submitting={submittingResolve}
          onSubmit={(values) => void resolveContradiction(values)}
        />

        <ContradictionScanDialog
          open={scanOpen}
          onOpenChange={setScanOpen}
          onComplete={() => {
            void refetch();
            void statsQuery.refetch();
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
