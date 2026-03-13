'use client';

import { Bot, Download, RefreshCw } from 'lucide-react';

import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate, formatDateTime } from '@/lib/utils';
import type { VCISOBriefing } from '@/types/cyber';
import { ChatPanel } from './_components/chat-panel';
import { ComplianceStatusSection } from './_components/compliance-status-section';
import { CriticalIssuesCards } from './_components/critical-issues-cards';
import { LLMOpsPanel } from './_components/llm-ops-panel';
import { RecommendationsList } from './_components/recommendations-list';
import { RiskPostureSummary } from './_components/risk-posture-summary';
import { ThreatLandscapeSection } from './_components/threat-landscape-section';

export default function CyberVcisoPage() {
  const {
    data: envelope,
    isLoading,
    error,
    mutate: refetch,
  } = useRealtimeData<{ data: VCISOBriefing }>(API_ENDPOINTS.CYBER_VCISO_BRIEFING, {
    pollInterval: 300000,
  });

  const { mutate: generateReport, isPending: generating } = useApiMutation<{ download_url?: string }, Record<string, never>>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_REPORT,
    {
      successMessage: 'Report generation started',
      onSuccess: (result) => {
        if (result.download_url) {
          window.open(result.download_url, '_blank');
        }
      },
    },
  );

  const briefing = envelope?.data;

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Virtual CISO"
          description="Executive security briefing, hybrid routed chat, and LLM observability in one workspace."
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => void refetch()}>
                <RefreshCw className="mr-1.5 h-4 w-4" />
                Refresh
              </Button>
              <Button size="sm" onClick={() => generateReport({} as Record<string, never>)} disabled={generating}>
                <Download className="mr-1.5 h-4 w-4" />
                {generating ? 'Generating…' : 'Export Report'}
              </Button>
            </div>
          }
        />

        {isLoading ? (
          <div className="grid gap-6 xl:grid-cols-[minmax(0,1.45fr)_minmax(380px,1fr)]">
            <div className="space-y-4">
              <LoadingSkeleton variant="card" />
              <div className="grid gap-4 md:grid-cols-2">
                <LoadingSkeleton variant="card" />
                <LoadingSkeleton variant="card" />
              </div>
              <LoadingSkeleton variant="card" />
            </div>
            <LoadingSkeleton variant="card" className="h-[720px]" />
          </div>
        ) : error || !briefing ? (
          <ErrorState message="Failed to load the Virtual CISO briefing." onRetry={() => void refetch()} />
        ) : (
          <div className="space-y-6">
            <div className="grid gap-6 xl:grid-cols-[minmax(0,1.45fr)_minmax(380px,1fr)]">
              <div className="space-y-6">
                <section className="relative overflow-hidden rounded-[2rem] border bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.16),_transparent_38%),linear-gradient(135deg,#0f172a,#1e293b)] p-6 text-white shadow-xl">
                  <div className="absolute right-6 top-6 opacity-10">
                    <Bot className="h-28 w-28" />
                  </div>
                  <div className="relative space-y-5">
                    <div className="flex flex-wrap items-center gap-3">
                      <Badge className="rounded-full bg-white/15 text-white hover:bg-white/15">Executive Briefing</Badge>
                      <Badge variant="outline" className="rounded-full border-white/20 bg-transparent text-white">
                        {formatDate(briefing.period_start)} - {formatDate(briefing.period_end)}
                      </Badge>
                    </div>
                    <div className="space-y-2">
                      <h2 className="text-2xl font-semibold tracking-tight">Security posture at a glance</h2>
                      <p className="max-w-3xl text-sm leading-6 text-white/80">{briefing.executive_summary}</p>
                    </div>
                    <div className="flex flex-wrap gap-6 text-sm">
                      <div>
                        <p className="text-white/60">Risk score</p>
                        <p className="mt-1 text-3xl font-semibold">{briefing.risk_posture.overall_score}</p>
                      </div>
                      <div>
                        <p className="text-white/60">Grade</p>
                        <p className="mt-1 text-3xl font-semibold">{briefing.risk_posture.grade}</p>
                      </div>
                      <div>
                        <p className="text-white/60">Generated</p>
                        <p className="mt-1 font-medium">{formatDateTime(briefing.generated_at)}</p>
                      </div>
                    </div>
                  </div>
                </section>

                <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
                  <div className="space-y-4">
                    <div>
                      <h3 className="mb-3 text-sm font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                        Critical Issues
                      </h3>
                      <CriticalIssuesCards issues={briefing.critical_issues} />
                    </div>
                    <div>
                      <h3 className="mb-3 text-sm font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                        Recommendations
                      </h3>
                      <RecommendationsList recommendations={briefing.recommendations} />
                    </div>
                  </div>
                  <div className="space-y-4">
                    <RiskPostureSummary posture={briefing.risk_posture} />
                    <ThreatLandscapeSection landscape={briefing.threat_landscape} />
                  </div>
                </div>

                <section>
                  <h3 className="mb-3 text-sm font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                    Compliance Status
                  </h3>
                  <ComplianceStatusSection frameworks={briefing.compliance_status} />
                </section>
              </div>

              <ChatPanel />
            </div>

            <LLMOpsPanel />
          </div>
        )}
      </div>
    </PermissionRedirect>
  );
}
