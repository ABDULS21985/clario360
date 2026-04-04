'use client';

import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, RefreshCw } from 'lucide-react';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { CyberAlert } from '@/types/cyber';

import { AlertComments } from './_components/alert-comments';
import { AlertEvidence } from './_components/alert-evidence';
import { AlertExplanation } from './_components/alert-explanation';
import { AlertHeader } from './_components/alert-header';
import { AlertRelated } from './_components/alert-related';
import { AlertTimeline } from './_components/alert-timeline';

interface AlertDetailPageProps {
  params: { id: string };
}

export default function AlertDetailPage({ params }: AlertDetailPageProps) {
  const router = useRouter();

  const alertQuery = useQuery({
    queryKey: ['cyber-alert', params.id],
    queryFn: () => apiGet<{ data: CyberAlert }>(API_ENDPOINTS.CYBER_ALERT_DETAIL(params.id)),
    refetchInterval: 30000,
  });

  const alert = alertQuery.data?.data;

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Alert Investigation Workspace"
          description="Inspect the explanation payload, review supporting evidence, collaborate with analysts, and pivot into related detections."
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => router.push('/cyber/alerts')}>
                <ArrowLeft className="mr-1.5 h-4 w-4" />
                Back to Alerts
              </Button>
              <Button variant="outline" size="sm" onClick={() => void alertQuery.refetch()}>
                <RefreshCw className="mr-1.5 h-4 w-4" />
                Refresh
              </Button>
            </div>
          }
        />

        {alertQuery.isLoading ? (
          <LoadingSkeleton variant="card" />
        ) : alertQuery.error || !alert ? (
          <ErrorState message="Failed to load alert details" onRetry={() => void alertQuery.refetch()} />
        ) : (
          <>
            <AlertHeader alert={alert} onUpdated={() => void alertQuery.refetch()} />

            {(alert.tags?.length ?? 0) > 0 && (
              <div className="flex flex-wrap gap-2">
                {alert.tags.map((tag) => (
                  <Badge key={tag} variant="secondary">
                    {tag}
                  </Badge>
                ))}
              </div>
            )}

            <Tabs defaultValue="explanation" className="space-y-4">
              <TabsList className="w-full justify-start overflow-x-auto">
                <TabsTrigger value="explanation">AI Explanation</TabsTrigger>
                <TabsTrigger value="evidence">Evidence</TabsTrigger>
                <TabsTrigger value="comments">Comments</TabsTrigger>
                <TabsTrigger value="timeline">Timeline</TabsTrigger>
                <TabsTrigger value="related">Related Alerts</TabsTrigger>
              </TabsList>

              <TabsContent value="explanation">
                <AlertExplanation alert={alert} />
              </TabsContent>

              <TabsContent value="evidence">
                <AlertEvidence alert={alert} />
              </TabsContent>

              <TabsContent value="comments">
                <AlertComments alertId={alert.id} />
              </TabsContent>

              <TabsContent value="timeline">
                <AlertTimeline alertId={alert.id} />
              </TabsContent>

              <TabsContent value="related">
                <AlertRelated alert={alert} />
              </TabsContent>
            </Tabs>
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
