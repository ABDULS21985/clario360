'use client';

import { useState } from 'react';
import {
  Bot,
  TrendingDown,
  TrendingUp,
  Minus,
  AlertTriangle,
  Shield,
  Target,
  Download,
  RefreshCw,
  ChevronRight,
  CheckCircle,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { VCISOBriefing } from '@/types/cyber';

const COMPLIANCE_STATUS_COLORS: Record<string, string> = {
  compliant: 'bg-green-100 text-green-700',
  partial: 'bg-amber-100 text-amber-800',
  non_compliant: 'bg-red-100 text-red-700',
};

const EFFORT_COLORS: Record<string, string> = {
  low: 'text-green-600',
  medium: 'text-amber-600',
  high: 'text-red-600',
};

export default function CyberVcisoPage() {
  const [expandedRec, setExpandedRec] = useState<string | null>(null);

  const {
    data: envelope,
    isLoading,
    error,
    mutate: refetch,
  } = useRealtimeData<{ data: VCISOBriefing }>(API_ENDPOINTS.CYBER_VCISO_BRIEFING, {
    pollInterval: 300000,
  });

  const { mutate: generateReport, isPending: generating } = useApiMutation<{ download_url: string }, Record<string, never>>(
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

  const trendIcon = briefing?.risk_posture.trend === 'up'
    ? <TrendingUp className="h-4 w-4 text-red-500" />
    : briefing?.risk_posture.trend === 'down'
    ? <TrendingDown className="h-4 w-4 text-green-500" />
    : <Minus className="h-4 w-4 text-muted-foreground" />;

  const gradeColor = (grade: string) => {
    if (['A', 'B'].includes(grade)) return 'text-green-600';
    if (grade === 'C') return 'text-amber-600';
    return 'text-red-600';
  };

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Virtual CISO"
          description="AI-powered executive security briefing and strategic recommendations"
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => void refetch()}>
                <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                Refresh
              </Button>
              <Button size="sm" onClick={() => generateReport({} as Record<string, never>)} disabled={generating}>
                <Download className="mr-1.5 h-3.5 w-3.5" />
                {generating ? 'Generating…' : 'Export Report'}
              </Button>
            </div>
          }
        />

        {isLoading ? (
          <div className="space-y-4">
            <LoadingSkeleton variant="card" />
            <div className="grid grid-cols-3 gap-4">
              <LoadingSkeleton variant="card" />
              <LoadingSkeleton variant="card" />
              <LoadingSkeleton variant="card" />
            </div>
          </div>
        ) : error || !briefing ? (
          <ErrorState message="Failed to load vCISO briefing" onRetry={() => void refetch()} />
        ) : (
          <>
            {/* Executive summary banner */}
            <div className="relative overflow-hidden rounded-2xl border bg-gradient-to-br from-slate-900 to-slate-700 p-6 text-white shadow-lg dark:from-slate-800 dark:to-slate-600">
              <div className="absolute right-4 top-4 opacity-10">
                <Bot className="h-32 w-32" />
              </div>
              <div className="relative">
                <div className="mb-4 flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <Bot className="h-5 w-5 text-primary" />
                      <span className="text-sm font-medium text-white/70">AI Executive Briefing</span>
                    </div>
                    <h2 className="mt-1 text-xl font-semibold">Security Posture Assessment</h2>
                    <p className="text-sm text-white/60">
                      Period: {new Date(briefing.period_start).toLocaleDateString()} – {new Date(briefing.period_end).toLocaleDateString()}
                    </p>
                  </div>
                  <div className="text-right">
                    <div className="flex items-center gap-2">
                      <span className={`text-5xl font-bold ${gradeColor(briefing.risk_posture.grade)}`}>
                        {briefing.risk_posture.grade}
                      </span>
                      <div>
                        <p className="text-sm text-white/70">Risk Grade</p>
                        <div className="flex items-center gap-1">
                          {trendIcon}
                          <span className="text-xs text-white/60">
                            {briefing.risk_posture.trend_delta > 0 ? '+' : ''}
                            {briefing.risk_posture.trend_delta.toFixed(1)} pts
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
                <p className="text-sm leading-relaxed text-white/80">{briefing.executive_summary}</p>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
              {/* Left: critical issues + recommendations */}
              <div className="lg:col-span-2 space-y-4">
                <h3 className="flex items-center gap-2 text-sm font-semibold">
                  <AlertTriangle className="h-4 w-4 text-red-500" />
                  Critical Issues ({briefing.critical_issues.length})
                </h3>
                <div className="space-y-3">
                  {briefing.critical_issues.length === 0 ? (
                    <div className="flex items-center gap-2 rounded-lg border bg-green-50 p-4 text-sm text-green-700 dark:bg-green-950/20 dark:text-green-400">
                      <CheckCircle className="h-5 w-5" />
                      No critical issues identified — excellent security posture!
                    </div>
                  ) : (
                    briefing.critical_issues.map((issue) => (
                      <div key={issue.id} className="rounded-xl border bg-card p-4">
                        <div className="flex items-start justify-between gap-3">
                          <div className="flex-1 min-w-0">
                            <div className="mb-1 flex items-center gap-2">
                              <SeverityIndicator severity={issue.severity} showLabel />
                            </div>
                            <h4 className="font-medium">{issue.title}</h4>
                            <p className="mt-1 text-sm text-muted-foreground">{issue.impact}</p>
                          </div>
                          {issue.link && (
                            <a href={issue.link} className="shrink-0 text-xs text-primary hover:underline">View →</a>
                          )}
                        </div>
                        <div className="mt-3 rounded-lg bg-muted/30 p-3">
                          <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Recommendation</p>
                          <p className="text-sm">{issue.recommendation}</p>
                        </div>
                      </div>
                    ))
                  )}
                </div>

                <h3 className="flex items-center gap-2 pt-2 text-sm font-semibold">
                  <Target className="h-4 w-4 text-blue-500" />
                  Strategic Recommendations
                </h3>
                <div className="space-y-2">
                  {briefing.recommendations.map((rec) => (
                    <div key={rec.id} className="overflow-hidden rounded-xl border bg-card">
                      <button
                        className="flex w-full items-start gap-3 p-4 text-left transition-colors hover:bg-muted/30"
                        onClick={() => setExpandedRec(expandedRec === rec.id ? null : rec.id)}
                      >
                        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                          {rec.priority}
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <p className="text-sm font-medium">{rec.title}</p>
                            <Badge variant="outline" className="text-xs">{rec.category}</Badge>
                          </div>
                          <div className="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
                            <span>Effort: <span className={`font-medium capitalize ${EFFORT_COLORS[rec.effort]}`}>{rec.effort}</span></span>
                            <span>Risk reduction: <span className="font-medium text-green-600">-{rec.estimated_risk_reduction} pts</span></span>
                          </div>
                        </div>
                        <ChevronRight className={`h-4 w-4 shrink-0 text-muted-foreground transition-transform ${expandedRec === rec.id ? 'rotate-90' : ''}`} />
                      </button>
                      {expandedRec === rec.id && (
                        <div className="border-t bg-muted/20 p-4">
                          <p className="mb-3 text-sm text-muted-foreground">{rec.description}</p>
                          <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Actions</p>
                          <ul className="space-y-1.5">
                            {rec.actions.map((action, i) => (
                              <li key={i} className="flex items-start gap-2 text-sm">
                                <CheckCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-primary" />
                                {action}
                              </li>
                            ))}
                          </ul>
                          <p className="mt-3 text-sm italic text-muted-foreground">Impact: {rec.impact}</p>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* Right sidebar */}
              <div className="space-y-4">
                <div className="rounded-xl border bg-card p-4">
                  <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold">
                    <Shield className="h-4 w-4 text-orange-500" />
                    Threat Landscape
                  </h3>
                  <div className="space-y-2.5 text-sm">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Active Threats</span>
                      <span className="font-bold text-orange-600">{briefing.threat_landscape.active_threat_count}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Top Tactic</span>
                      <span className="max-w-[140px] truncate text-right text-xs font-medium">{briefing.threat_landscape.top_tactic}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Top Technique</span>
                      <span className="max-w-[140px] truncate text-right text-xs font-medium">{briefing.threat_landscape.top_technique}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Recent Indicators</span>
                      <span className="font-bold">{briefing.threat_landscape.recent_indicators}</span>
                    </div>
                    {Object.entries(briefing.threat_landscape.threat_by_type).map(([type, count]) => {
                      const maxVal = Math.max(...Object.values(briefing.threat_landscape.threat_by_type));
                      return (
                        <div key={type} className="flex items-center justify-between gap-2">
                          <span className="capitalize text-xs text-muted-foreground">{type}</span>
                          <div className="flex items-center gap-2">
                            <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
                              <div className="h-full rounded-full bg-orange-400" style={{ width: `${Math.min(100, (count / maxVal) * 100)}%` }} />
                            </div>
                            <span className="text-xs font-medium">{count}</span>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>

                <div className="rounded-xl border bg-card p-4">
                  <h3 className="mb-3 text-sm font-semibold">Compliance Status</h3>
                  <div className="space-y-3">
                    {briefing.compliance_status.map((fw) => (
                      <div key={fw.name}>
                        <div className="mb-1 flex items-center justify-between">
                          <span className="text-sm font-medium">{fw.name}</span>
                          <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${COMPLIANCE_STATUS_COLORS[fw.status] ?? 'bg-muted text-muted-foreground'}`}>
                            {fw.status.replace('_', ' ')}
                          </span>
                        </div>
                        <div className="mb-1 h-1.5 overflow-hidden rounded-full bg-muted">
                          <div
                            className={`h-full rounded-full ${fw.coverage_percent >= 80 ? 'bg-green-500' : fw.coverage_percent >= 60 ? 'bg-amber-500' : 'bg-red-500'}`}
                            style={{ width: `${fw.coverage_percent}%` }}
                          />
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {fw.controls_passed}/{fw.controls_total} controls ({fw.coverage_percent.toFixed(0)}%)
                        </p>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="rounded-xl border bg-card p-4">
                  <h3 className="mb-3 text-sm font-semibold">Risk Components</h3>
                  <div className="space-y-2.5">
                    {Object.entries(briefing.risk_posture.components).map(([key, score]) => {
                      const numScore = typeof score === 'number' ? score : 0;
                      return (
                        <div key={key}>
                          <div className="mb-1 flex justify-between text-xs">
                            <span className="capitalize text-muted-foreground">{key.replace(/_/g, ' ')}</span>
                            <span className="font-medium">{numScore.toFixed(0)}</span>
                          </div>
                          <div className="h-1.5 overflow-hidden rounded-full bg-muted">
                            <div
                              className={`h-full rounded-full transition-all ${numScore <= 30 ? 'bg-green-500' : numScore <= 60 ? 'bg-amber-500' : 'bg-red-500'}`}
                              style={{ width: `${numScore}%` }}
                            />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              </div>
            </div>

            <p className="text-right text-xs text-muted-foreground">
              Generated: {new Date(briefing.generated_at).toLocaleString()}
              {briefing.previous_briefing_id && briefing.previous_risk_score !== undefined && (
                <span className="ml-2">vs. previous score: {briefing.previous_risk_score.toFixed(0)}</span>
              )}
            </p>
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
