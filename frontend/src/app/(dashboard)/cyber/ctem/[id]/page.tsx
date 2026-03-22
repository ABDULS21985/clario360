'use client';

import { useRouter } from 'next/navigation';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost, apiDelete } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { ArrowLeft, Target, Trash2, XCircle, FileDown } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useState } from 'react';
import { timeAgo } from '@/lib/utils';
import { showSuccess, showError } from '@/lib/toast';
import { PhaseStepper } from './_components/phase-stepper';
import { FindingTable } from './_components/finding-table';
import { RemediationGroups } from './_components/remediation-groups';
import { AssessmentComparisonView } from './_components/assessment-comparison';
import { normalizeCTEMPhases } from '@/types/cyber';
import type { CTEMAssessment, CTEMFinding } from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';

interface Props {
  params: { id: string };
}

/** Map ALL backend assessment statuses to display styles */
const STATUS_COLORS: Record<string, string> = {
  created: 'text-muted-foreground',
  scoping: 'text-blue-600',
  discovery: 'text-blue-600',
  prioritizing: 'text-blue-600',
  validating: 'text-blue-600',
  mobilizing: 'text-blue-600',
  completed: 'text-green-600',
  failed: 'text-red-600',
  cancelled: 'text-gray-500',
};

/** Statuses that indicate an active/running assessment */
const ACTIVE_STATUSES = ['created', 'scoping', 'discovery', 'prioritizing', 'validating', 'mobilizing'];
const TERMINAL_STATUSES = ['completed', 'failed', 'cancelled'];

export default function CTEMAssessmentDetailPage({ params }: Props) {
  const { id } = params;
  const router = useRouter();
  const queryClient = useQueryClient();
  const [cancelOpen, setCancelOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const { data: envelope, isLoading, error, refetch } = useQuery({
    queryKey: [`ctem-assessment-${id}`],
    queryFn: () => apiGet<{ data: CTEMAssessment }>(API_ENDPOINTS.CYBER_CTEM_ASSESSMENT_DETAIL(id)),
    refetchInterval: (q) => {
      const s = q.state.data?.data.status;
      return s && !TERMINAL_STATUSES.includes(s) ? 10000 : false;
    },
  });

  const assessment = envelope?.data;

  // Fetch findings from dedicated endpoint (backend does NOT embed findings in assessment)
  const findingsQuery = useQuery({
    queryKey: [`ctem-assessment-findings-${id}`],
    queryFn: () =>
      apiGet<PaginatedResponse<CTEMFinding>>(API_ENDPOINTS.CYBER_CTEM_ASSESSMENT_FINDINGS(id), {
        per_page: 200,
        sort: 'priority_score',
        order: 'desc',
      }),
    enabled: !!assessment,
    refetchInterval: (q) => {
      // Poll for findings while assessment is still running
      return assessment && !TERMINAL_STATUSES.includes(assessment.status) ? 15000 : false;
    },
  });

  const findings = findingsQuery.data?.data ?? [];
  const summary = assessment?.findings_summary;

  const isActive = assessment && ACTIVE_STATUSES.includes(assessment.status);
  const canCancel = assessment && !TERMINAL_STATUSES.includes(assessment.status) && assessment.status !== 'created';
  const canDelete = assessment && (assessment.status === 'created' || TERMINAL_STATUSES.includes(assessment.status));

  const [exporting, setExporting] = useState(false);

  const handleExport = async (format: 'pdf' | 'docx') => {
    setExporting(true);
    try {
      await apiPost(API_ENDPOINTS.CYBER_CTEM_ASSESSMENT_REPORT_EXPORT(id), { format });
      showSuccess(`Report export started (${format.toUpperCase()})`);
    } catch {
      showError('Failed to export report');
    } finally {
      setExporting(false);
    }
  };

  const handleCancel = async () => {
    try {
      await apiPost(API_ENDPOINTS.CYBER_CTEM_ASSESSMENT_CANCEL(id));
      showSuccess('Assessment cancelled');
      void refetch();
      void queryClient.invalidateQueries({ queryKey: ['cyber-ctem-assessments'] });
    } catch {
      showError('Failed to cancel assessment');
    }
    setCancelOpen(false);
  };

  const handleDelete = async () => {
    try {
      await apiDelete(`${API_ENDPOINTS.CYBER_CTEM_ASSESSMENTS}/${id}`);
      showSuccess('Assessment deleted');
      void queryClient.invalidateQueries({ queryKey: ['cyber-ctem-assessments'] });
      router.push('/cyber/ctem');
    } catch {
      showError('Failed to delete assessment');
    }
    setDeleteOpen(false);
  };

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {isLoading ? (
          <LoadingSkeleton variant="card" />
        ) : error || !assessment ? (
          <ErrorState message="Failed to load assessment" onRetry={() => refetch()} />
        ) : (
          <>
            <PageHeader
              title={
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => router.back()}
                    className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
                  >
                    <ArrowLeft className="h-4 w-4" />
                  </button>
                  <Target className="h-5 w-5 text-primary" />
                  <span>{assessment.name}</span>
                </div>
              }
              description={
                <div className="flex items-center gap-4 pl-11">
                  <span className={`text-sm font-medium capitalize ${STATUS_COLORS[assessment.status] ?? ''}`}>
                    {assessment.status.replace(/_/g, ' ')}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    Started {timeAgo(assessment.created_at)}
                  </span>
                </div>
              }
              actions={
                <div className="flex items-center gap-2">
                  {assessment.status === 'completed' && (
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="outline" size="sm" disabled={exporting}>
                          <FileDown className="mr-1.5 h-3.5 w-3.5" />
                          {exporting ? 'Exporting...' : 'Export'}
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => handleExport('pdf')}>Export as PDF</DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleExport('docx')}>Export as DOCX</DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  )}
                  {canCancel && (
                    <Button variant="outline" size="sm" onClick={() => setCancelOpen(true)}>
                      <XCircle className="mr-1.5 h-3.5 w-3.5" />
                      Cancel
                    </Button>
                  )}
                  {canDelete && (
                    <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
                      <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                      Delete
                    </Button>
                  )}
                </div>
              }
            />

            {/* Summary KPIs */}
            {summary && (
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
                {[
                  { label: 'Critical', value: summary.critical, color: 'text-red-600' },
                  { label: 'High', value: summary.high, color: 'text-orange-600' },
                  { label: 'Medium', value: summary.medium, color: 'text-yellow-600' },
                  { label: 'Low', value: summary.low, color: 'text-blue-600' },
                  { label: 'Total', value: summary.total, color: 'text-foreground' },
                ].map(({ label, value, color }) => (
                  <div key={label} className="rounded-xl border bg-card p-3 text-center">
                    <p className={`text-2xl font-bold tabular-nums ${color}`}>{value}</p>
                    <p className="mt-0.5 text-xs text-muted-foreground">{label}</p>
                  </div>
                ))}
              </div>
            )}

            {/* Phase stepper */}
            <div className="rounded-xl border bg-card p-4">
              <h3 className="mb-4 text-sm font-semibold">Assessment Progress</h3>
              <PhaseStepper phases={normalizeCTEMPhases(assessment.phases)} currentPhase={assessment.current_phase} />
            </div>

            {/* Findings */}
            <div>
              <h3 className="mb-3 text-sm font-semibold">
                Findings
                {findings.length > 0 && (
                  <span className="ml-2 text-xs font-normal text-muted-foreground">({findings.length})</span>
                )}
              </h3>
              {findingsQuery.isLoading ? (
                <LoadingSkeleton variant="table-row" />
              ) : (
                <FindingTable
                  findings={findings}
                  onStatusUpdated={() => {
                    void findingsQuery.refetch();
                    void refetch();
                  }}
                />
              )}
            </div>

            {/* Remediation Groups */}
            <RemediationGroups assessmentId={id} />

            {/* Assessment Comparison — only for completed assessments */}
            {assessment.status === 'completed' && (
              <AssessmentComparisonView assessmentId={id} />
            )}
          </>
        )}
      </div>

      <ConfirmDialog
        open={cancelOpen}
        onOpenChange={setCancelOpen}
        title="Cancel Assessment"
        description="Are you sure you want to cancel this running assessment? This action cannot be undone."
        confirmLabel="Cancel Assessment"
        variant="destructive"
        onConfirm={handleCancel}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete Assessment"
        description="Are you sure you want to delete this assessment? All findings and data will be permanently removed."
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </PermissionRedirect>
  );
}
