'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ArrowDown, ArrowUp, Minus, BarChart3 } from 'lucide-react';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import type { PaginatedResponse } from '@/types/api';
import type { CTEMAssessment } from '@/types/cyber';

interface ComparisonSide {
  id: string;
  name: string;
  exposure_score?: number;
  findings: Record<string, number>;
}

interface FindingSummary {
  id: string;
  title: string;
  type: string;
  severity: string;
  priority_score: number;
}

interface ComparisonDelta {
  score_change: number;
  score_direction: string;
  findings_new: number;
  findings_resolved: number;
  findings_unchanged: number;
  findings_worsened: number;
  new_findings: FindingSummary[];
  resolved_findings: FindingSummary[];
}

interface AssessmentComparison {
  current: ComparisonSide;
  previous: ComparisonSide;
  delta: ComparisonDelta;
}

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'text-red-600',
  high: 'text-orange-600',
  medium: 'text-yellow-600',
  low: 'text-blue-600',
};

interface AssessmentComparisonViewProps {
  assessmentId: string;
}

export function AssessmentComparisonView({ assessmentId }: AssessmentComparisonViewProps) {
  const [otherId, setOtherId] = useState<string | null>(null);

  // Fetch completed assessments to compare against
  const { data: assessmentsData } = useQuery({
    queryKey: ['cyber-ctem-assessments-completed'],
    queryFn: () =>
      apiGet<PaginatedResponse<CTEMAssessment>>(API_ENDPOINTS.CYBER_CTEM_ASSESSMENTS, {
        per_page: 50,
        status: 'completed',
        sort: 'created_at',
        order: 'desc',
      }),
  });

  const otherAssessments = (assessmentsData?.data ?? []).filter((a) => a.id !== assessmentId);

  const { data: comparisonData, isLoading: comparing } = useQuery({
    queryKey: [`ctem-compare-${assessmentId}-${otherId}`],
    queryFn: () =>
      apiGet<{ data: AssessmentComparison }>(API_ENDPOINTS.CYBER_CTEM_ASSESSMENT_COMPARE(assessmentId, otherId!)),
    enabled: !!otherId,
  });

  const comparison = comparisonData?.data;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <BarChart3 className="h-4 w-4 text-muted-foreground" />
        <h3 className="text-base font-semibold">Compare with Previous Assessment</h3>
      </div>

      <div className="flex items-center gap-3">
        <Select value={otherId ?? ''} onValueChange={(v) => setOtherId(v || null)}>
          <SelectTrigger className="w-72">
            <SelectValue placeholder="Select an assessment to compare" />
          </SelectTrigger>
          <SelectContent>
            {otherAssessments.map((a) => (
              <SelectItem key={a.id} value={a.id}>
                {a.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {comparing && <LoadingSkeleton variant="card" />}

      {comparison && (
        <div className="space-y-4">
          {/* Score Comparison */}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <ScoreCard label={comparison.current.name} score={comparison.current.exposure_score} variant="current" />
            <DeltaCard delta={comparison.delta} />
            <ScoreCard label={comparison.previous.name} score={comparison.previous.exposure_score} variant="previous" />
          </div>

          {/* Findings Summary */}
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <StatCard label="New Findings" value={comparison.delta.findings_new} color="text-red-600" />
            <StatCard label="Resolved" value={comparison.delta.findings_resolved} color="text-green-600" />
            <StatCard label="Unchanged" value={comparison.delta.findings_unchanged} color="text-muted-foreground" />
            <StatCard label="Worsened" value={comparison.delta.findings_worsened} color="text-orange-600" />
          </div>

          {/* Findings by Severity */}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <SeverityBreakdown label="Current" findings={comparison.current.findings} />
            <SeverityBreakdown label="Previous" findings={comparison.previous.findings} />
          </div>

          {/* New Findings List */}
          {comparison.delta.new_findings.length > 0 && (
            <FindingsList title="New Findings" findings={comparison.delta.new_findings} />
          )}

          {/* Resolved Findings List */}
          {comparison.delta.resolved_findings.length > 0 && (
            <FindingsList title="Resolved Findings" findings={comparison.delta.resolved_findings} />
          )}
        </div>
      )}

      {!otherId && otherAssessments.length === 0 && (
        <p className="py-4 text-center text-sm text-muted-foreground">
          No other completed assessments available for comparison.
        </p>
      )}
    </div>
  );
}

function ScoreCard({ label, score, variant }: { label: string; score?: number; variant: 'current' | 'previous' }) {
  return (
    <div className="rounded-xl border bg-card p-4 text-center">
      <p className="text-xs font-medium text-muted-foreground">{variant === 'current' ? 'Current' : 'Previous'}</p>
      <p className="mt-0.5 text-xs text-muted-foreground line-clamp-1">{label}</p>
      <p className="mt-2 text-3xl font-bold tabular-nums">{score != null ? Math.round(score) : '—'}</p>
      <p className="text-xs text-muted-foreground">Exposure Score</p>
    </div>
  );
}

function DeltaCard({ delta }: { delta: ComparisonDelta }) {
  const Icon = delta.score_direction === 'worsened' ? ArrowUp : delta.score_direction === 'improved' ? ArrowDown : Minus;
  const color = delta.score_direction === 'worsened' ? 'text-red-600' : delta.score_direction === 'improved' ? 'text-green-600' : 'text-muted-foreground';

  return (
    <div className="flex flex-col items-center justify-center rounded-xl border bg-card p-4">
      <Icon className={`h-6 w-6 ${color}`} />
      <p className={`mt-1 text-2xl font-bold tabular-nums ${color}`}>
        {delta.score_change > 0 ? '+' : ''}{delta.score_change.toFixed(1)}
      </p>
      <p className="text-xs capitalize text-muted-foreground">{delta.score_direction}</p>
    </div>
  );
}

function StatCard({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="rounded-xl border bg-card p-3 text-center">
      <p className={`text-2xl font-bold tabular-nums ${color}`}>{value}</p>
      <p className="mt-0.5 text-xs text-muted-foreground">{label}</p>
    </div>
  );
}

function SeverityBreakdown({ label, findings }: { label: string; findings: Record<string, number> }) {
  const severities = ['critical', 'high', 'medium', 'low'];
  return (
    <div className="rounded-xl border bg-card p-4">
      <p className="mb-2 text-xs font-semibold text-muted-foreground">{label}</p>
      <div className="flex items-center gap-4">
        {severities.map((sev) => (
          <div key={sev} className="text-center">
            <p className={`text-lg font-bold tabular-nums ${SEVERITY_COLORS[sev] ?? ''}`}>{findings[sev] ?? 0}</p>
            <p className="text-xs capitalize text-muted-foreground">{sev}</p>
          </div>
        ))}
      </div>
    </div>
  );
}

function FindingsList({ title, findings }: { title: string; findings: FindingSummary[] }) {
  return (
    <div className="rounded-xl border bg-card p-4">
      <p className="mb-2 text-sm font-semibold">{title}</p>
      <div className="space-y-1">
        {findings.map((f) => (
          <div key={f.id} className="flex items-center justify-between rounded-lg px-2 py-1.5 hover:bg-muted/20">
            <div className="flex items-center gap-2">
              <span className={`text-xs font-bold capitalize ${SEVERITY_COLORS[f.severity] ?? ''}`}>
                {f.severity.charAt(0).toUpperCase()}
              </span>
              <span className="text-sm">{f.title}</span>
            </div>
            <Badge variant="outline" className="text-xs capitalize">{f.type.replace(/_/g, ' ')}</Badge>
          </div>
        ))}
      </div>
    </div>
  );
}
