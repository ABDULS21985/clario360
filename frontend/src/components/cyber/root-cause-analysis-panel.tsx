'use client';

import { AlertTriangle, GitBranch, ShieldAlert } from 'lucide-react';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import type { RootCauseAnalysis } from '@/types/cyber';

interface RootCauseAnalysisPanelProps {
  analysis: RootCauseAnalysis | null | undefined;
  isLoading: boolean;
  error?: string | null;
  onAnalyze?: () => void;
  analyzeLabel?: string;
  emptyTitle: string;
  emptyDescription: string;
  disabledReason?: string;
}

function formatTimestamp(value?: string) {
  if (!value) return 'Unknown';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function formatPercent(value?: number) {
  if (typeof value !== 'number') return '—';
  return `${Math.round(value * 100)}%`;
}

export function RootCauseAnalysisPanel({
  analysis,
  isLoading,
  error,
  onAnalyze,
  analyzeLabel = 'Analyze',
  emptyTitle,
  emptyDescription,
  disabledReason,
}: RootCauseAnalysisPanelProps) {
  if (isLoading) {
    return <LoadingSkeleton variant="card" />;
  }

  if (error) {
    return <ErrorState message={error} onRetry={onAnalyze} />;
  }

  if (!analysis) {
    return (
      <div className="rounded-xl border bg-card p-6">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-base font-semibold">{emptyTitle}</h3>
            <p className="mt-2 max-w-2xl text-sm text-muted-foreground">{emptyDescription}</p>
            {disabledReason ? <p className="mt-3 text-xs text-muted-foreground">{disabledReason}</p> : null}
          </div>
          {onAnalyze ? (
            <Button size="sm" onClick={onAnalyze} disabled={Boolean(disabledReason)}>
              <GitBranch className="mr-1.5 h-3.5 w-3.5" />
              {analyzeLabel}
            </Button>
          ) : null}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="rounded-xl border bg-card p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <ShieldAlert className="h-4 w-4 text-amber-600" />
              <h3 className="text-base font-semibold">Root Cause Analysis</h3>
            </div>
            <p className="mt-2 text-sm text-muted-foreground">{analysis.summary}</p>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant="outline">Confidence {formatPercent(analysis.confidence)}</Badge>
            <Badge variant="outline">{analysis.duration_ms} ms</Badge>
            {onAnalyze ? (
              <Button size="sm" variant="outline" onClick={onAnalyze}>
                {analyzeLabel}
              </Button>
            ) : null}
          </div>
        </div>

        {analysis.root_cause ? (
          <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50/60 p-4">
            <div className="flex items-center gap-2 text-amber-800">
              <AlertTriangle className="h-4 w-4" />
              <span className="text-sm font-semibold">Identified Root Cause</span>
            </div>
            <p className="mt-2 text-sm text-slate-800">{analysis.root_cause.description}</p>
            <p className="mt-2 text-xs text-slate-600">{formatTimestamp(analysis.root_cause.timestamp)}</p>
          </div>
        ) : null}
      </div>

      <div className="grid gap-4 xl:grid-cols-[1.5fr_1fr]">
        <div className="space-y-4">
          <div className="rounded-xl border bg-card p-5">
            <h3 className="text-sm font-semibold">Causal Chain</h3>
            <div className="mt-4 space-y-3">
              {analysis.causal_chain.map((step) => (
                <div
                  key={`${step.event_id}-${step.order}`}
                  className={`rounded-lg border p-4 ${step.is_root_cause ? 'border-amber-300 bg-amber-50/50' : 'bg-background'}`}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium">{step.description}</p>
                      <p className="mt-1 text-xs text-muted-foreground">{formatTimestamp(step.timestamp)}</p>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant="outline">#{step.order}</Badge>
                      {step.mitre_phase ? <Badge variant="outline">{step.mitre_phase}</Badge> : null}
                      {step.severity ? <Badge variant="outline">{step.severity}</Badge> : null}
                    </div>
                  </div>
                  {step.evidence.length > 0 ? (
                    <div className="mt-3 space-y-2">
                      {step.evidence.slice(0, 3).map((evidence) => (
                        <div key={`${step.event_id}-${evidence.field}`} className="text-xs text-muted-foreground">
                          <span className="font-medium text-foreground">{evidence.label}:</span> {String(evidence.value)}
                        </div>
                      ))}
                    </div>
                  ) : null}
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-xl border bg-card p-5">
            <h3 className="text-sm font-semibold">Event Timeline</h3>
            <div className="mt-4 space-y-3">
              {analysis.timeline.slice(0, 8).map((event) => (
                <div key={event.id} className="rounded-lg border bg-background p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium">{event.summary}</p>
                      <p className="mt-1 text-xs text-muted-foreground">{event.source}</p>
                    </div>
                    <div className="text-right text-xs text-muted-foreground">
                      <div>{formatTimestamp(event.timestamp)}</div>
                      {event.mitre_phase ? <div className="mt-1">{event.mitre_phase}</div> : null}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div className="rounded-xl border bg-card p-5">
            <h3 className="text-sm font-semibold">Impact</h3>
            {analysis.impact ? (
              <div className="mt-4 space-y-3">
                <MetricRow label="Business Impact" value={analysis.impact.business_impact} />
                <MetricRow label="Assets Affected" value={String(analysis.impact.total_affected)} />
                <MetricRow label="Users At Risk" value={String(analysis.impact.users_at_risk)} />
                <MetricRow label="Data At Risk" value={String(analysis.impact.data_at_risk.length)} />
                <p className="text-xs text-muted-foreground">{analysis.impact.summary}</p>
              </div>
            ) : (
              <p className="mt-4 text-sm text-muted-foreground">No impact assessment is available for this incident.</p>
            )}
          </div>

          <div className="rounded-xl border bg-card p-5">
            <h3 className="text-sm font-semibold">Recommendations</h3>
            <div className="mt-4 space-y-3">
              {analysis.recommendations.length > 0 ? (
                analysis.recommendations.map((item) => (
                  <div key={`${item.priority}-${item.action}`} className="rounded-lg border bg-background p-4">
                    <div className="flex items-center justify-between gap-3">
                      <p className="text-sm font-medium">{item.action}</p>
                      <Badge variant="outline">P{item.priority}</Badge>
                    </div>
                    <p className="mt-2 text-xs text-muted-foreground">{item.rationale}</p>
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">No recommendations are available yet.</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function MetricRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium capitalize">{value}</span>
    </div>
  );
}
