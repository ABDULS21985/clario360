'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { EmptyState } from '@/components/common/empty-state';
import { ErrorState } from '@/components/common/error-state';
import { Button } from '@/components/ui/button';
import { Plus, Target } from 'lucide-react';
import type { PaginatedResponse } from '@/types/api';
import type { CTEMAssessment } from '@/types/cyber';

import { ExposureScoreGauge } from './_components/exposure-score-gauge';
import { AssessmentCard } from './_components/assessment-card';
import { CreateAssessmentDialog } from './_components/create-assessment-dialog';

export default function CyberCtemPage() {
  const [createOpen, setCreateOpen] = useState(false);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['cyber-ctem-assessments'],
    queryFn: () =>
      apiGet<PaginatedResponse<CTEMAssessment>>(API_ENDPOINTS.CYBER_CTEM_ASSESSMENTS, {
        per_page: 20,
        sort: 'created_at',
        order: 'desc',
      }),
    refetchInterval: 30000,
  });

  const assessments = data?.data ?? [];

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="CTEM Assessments"
          description="Continuous Threat Exposure Management — quantify and reduce your attack surface"
          actions={
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              New Assessment
            </Button>
          }
        />

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-4">
          {/* Exposure score sidebar */}
          <div className="lg:col-span-1">
            <ExposureScoreGauge />
          </div>

          {/* Assessment list */}
          <div className="lg:col-span-3">
            {isLoading ? (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                {Array.from({ length: 4 }).map((_, i) => (
                  <LoadingSkeleton key={i} variant="card" />
                ))}
              </div>
            ) : isError ? (
              <ErrorState message="Failed to load assessments" onRetry={() => void refetch()} />
            ) : assessments.length === 0 ? (
              <EmptyState
                icon={Target}
                title="No assessments"
                description="Launch your first CTEM assessment to quantify your threat exposure."
                action={{ label: 'New Assessment', onClick: () => setCreateOpen(true) }}
              />
            ) : (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                {assessments.map((a) => (
                  <AssessmentCard key={a.id} assessment={a} />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      <CreateAssessmentDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={() => refetch()}
      />
    </PermissionRedirect>
  );
}
