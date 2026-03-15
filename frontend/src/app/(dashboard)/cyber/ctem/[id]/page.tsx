'use client';


import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Button } from '@/components/ui/button';
import { ArrowLeft, Target } from 'lucide-react';
import { timeAgo } from '@/lib/utils';
import { PhaseStepper } from './_components/phase-stepper';
import { FindingTable } from './_components/finding-table';
import type { CTEMAssessment } from '@/types/cyber';

interface Props {
  params: { id: string };
}

const STATUS_COLORS: Record<string, string> = {
  draft: 'text-muted-foreground',
  running: 'text-blue-600',
  completed: 'text-green-600',
  failed: 'text-red-600',
  cancelled: 'text-gray-500',
};

export default function CTEMAssessmentDetailPage({ params }: Props) {
  const { id } = params;
  const router = useRouter();

  const { data: envelope, isLoading, error, refetch } = useQuery({
    queryKey: [`ctem-assessment-${id}`],
    queryFn: () => apiGet<{ data: CTEMAssessment }>(`${API_ENDPOINTS.CYBER_CTEM_ASSESSMENTS}/${id}`),
    refetchInterval: (q) => q.state.data?.data.status === 'running' ? 10000 : false,
  });

  const assessment = envelope?.data;
  const summary = assessment?.findings_summary;

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
                    {assessment.status}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    Started {timeAgo(assessment.created_at)}
                  </span>
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
              <PhaseStepper phases={assessment.phases ?? []} currentPhase={assessment.current_phase} />
            </div>

            {/* Findings */}
            <div>
              <h3 className="mb-3 text-sm font-semibold">Findings</h3>
              <FindingTable findings={assessment.findings ?? []} />
            </div>
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
