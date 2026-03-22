'use client';

import { FileText } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { formatDate } from '@/lib/utils';
import type { CTEMAssessment, CTEMFinding, CyberSeverity } from '@/types/cyber';

const SEVERITY_ORDER: CyberSeverity[] = ['critical', 'high', 'medium', 'low'];

const exposureScoreColor = (score: number): string => {
  if (score >= 80) return 'text-red-600 dark:text-red-400';
  if (score >= 60) return 'text-orange-600 dark:text-orange-400';
  if (score >= 40) return 'text-yellow-600 dark:text-yellow-400';
  return 'text-green-600 dark:text-green-400';
};

const statusVariant = (
  status: CTEMFinding['status'],
): 'default' | 'secondary' | 'outline' | 'destructive' => {
  switch (status) {
    case 'open':
      return 'destructive';
    case 'in_remediation':
      return 'default';
    case 'remediated':
      return 'secondary';
    default:
      return 'outline';
  }
};

const statusLabel: Record<CTEMFinding['status'], string> = {
  open: 'Open',
  in_remediation: 'In Remediation',
  remediated: 'Remediated',
  accepted_risk: 'Accepted Risk',
  false_positive: 'False Positive',
  deferred: 'Deferred',
};

interface AssessmentReportViewProps {
  assessment: CTEMAssessment;
  findings: CTEMFinding[];
}

export function AssessmentReportView({ assessment, findings }: AssessmentReportViewProps) {
  const isCompleted = assessment.status === 'completed';

  if (!isCompleted) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-14 text-center">
        <FileText className="h-10 w-10 text-muted-foreground/40" aria-hidden />
        <p className="text-sm font-medium text-muted-foreground">
          Report available once assessment is completed
        </p>
      </div>
    );
  }

  const summary = assessment.findings_summary;

  const groupedFindings = SEVERITY_ORDER.reduce<Record<CyberSeverity, CTEMFinding[]>>(
    (acc, sev) => {
      acc[sev] = findings.filter((f) => f.severity === sev);
      return acc;
    },
    { critical: [], high: [], medium: [], low: [], info: [] },
  );

  return (
    <div className="space-y-6">
      {/* Header Card */}
      <div className="rounded-xl border bg-card p-5">
        <div className="flex items-start gap-3">
          <FileText className="mt-0.5 h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
          <div className="flex-1 min-w-0">
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Assessment Report
            </p>
            <h2 className="mt-0.5 text-lg font-semibold">{assessment.name}</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              {formatDate(assessment.created_at)}
              {assessment.completed_at ? ` — ${formatDate(assessment.completed_at)}` : ''}
            </p>
          </div>
        </div>
      </div>

      {/* Executive Summary */}
      <div className="rounded-xl border bg-card p-5 space-y-4">
        <h3 className="text-sm font-semibold">Executive Summary</h3>

        {assessment.exposure_score != null && (
          <div className="flex items-center gap-3">
            <span className="text-xs text-muted-foreground">Exposure Score</span>
            <span className={`text-3xl font-bold tabular-nums ${exposureScoreColor(assessment.exposure_score)}`}>
              {assessment.exposure_score}
            </span>
            <span className="text-xs text-muted-foreground">/ 100</span>
          </div>
        )}

        {summary && (
          <div className="overflow-hidden rounded-lg border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40">
                  <th className="px-3 py-2 text-left text-xs font-semibold">Severity</th>
                  <th className="px-3 py-2 text-right text-xs font-semibold">Count</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {SEVERITY_ORDER.map((sev) => (
                  <tr key={sev} className="hover:bg-muted/20">
                    <td className="px-3 py-2">
                      <SeverityIndicator severity={sev} showLabel size="sm" />
                    </td>
                    <td className="px-3 py-2 text-right font-medium tabular-nums">
                      {summary[sev as keyof typeof summary] ?? 0}
                    </td>
                  </tr>
                ))}
                <tr className="border-t bg-muted/20 font-semibold">
                  <td className="px-3 py-2 text-xs">Total</td>
                  <td className="px-3 py-2 text-right tabular-nums">{summary.total}</td>
                </tr>
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Findings by Severity — use real backend fields */}
      {findings.length > 0 && (
        <div className="rounded-xl border bg-card p-5 space-y-5">
          <h3 className="text-sm font-semibold">Findings by Severity</h3>
          {SEVERITY_ORDER.map((sev) => {
            const group = groupedFindings[sev];
            if (group.length === 0) return null;
            return (
              <div key={sev} className="space-y-2">
                <div className="flex items-center gap-2">
                  <SeverityIndicator severity={sev} showLabel size="sm" />
                  <span className="text-xs text-muted-foreground tabular-nums">({group.length})</span>
                </div>
                <div className="overflow-hidden rounded-lg border">
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="border-b bg-muted/40">
                        <th className="px-3 py-2 text-left font-semibold">Finding</th>
                        <th className="px-3 py-2 text-left font-semibold">Status</th>
                        <th className="px-3 py-2 text-right font-semibold">Priority</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y">
                      {group.map((finding) => (
                        <tr key={finding.id} className="hover:bg-muted/20">
                          <td className="px-3 py-2 font-medium">{finding.title}</td>
                          <td className="px-3 py-2">
                            <Badge variant={statusVariant(finding.status)} className="text-xs">
                              {statusLabel[finding.status]}
                            </Badge>
                          </td>
                          <td className="px-3 py-2 text-right tabular-nums text-muted-foreground">
                            {Math.round(finding.priority_score)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
