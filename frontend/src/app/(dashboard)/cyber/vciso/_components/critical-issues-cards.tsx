'use client';
import { CheckCircle } from 'lucide-react';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import type { VCISOCriticalIssue } from '@/types/cyber';

export function CriticalIssuesCards({ issues }: { issues: VCISOCriticalIssue[] }) {
  if (issues.length === 0) {
    return (
      <div className="flex items-center gap-3 rounded-lg border border-green-200 bg-green-50 p-4">
        <CheckCircle className="h-5 w-5 flex-shrink-0 text-green-600" />
        <p className="text-sm font-medium text-green-700">
          No critical issues — your security posture is in good standing.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {issues.map((issue) => (
        <div
          key={issue.id}
          className="rounded-lg border bg-white p-4 shadow-sm space-y-3"
        >
          {/* Header: severity badge + title */}
          <div className="flex items-start gap-3">
            <SeverityIndicator severity={issue.severity} />
            <h3 className="text-sm font-semibold text-foreground leading-snug">
              {issue.title}
            </h3>
          </div>

          {/* Impact */}
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-0.5">
              Impact
            </p>
            <p className="text-sm text-muted-foreground">{issue.impact}</p>
          </div>

          {/* Recommendation */}
          <div className="rounded-md border border-blue-100 bg-blue-50 p-3">
            <p className="text-xs font-semibold uppercase tracking-wide text-blue-700 mb-0.5">
              Recommendation
            </p>
            <p className="text-sm text-blue-800">{issue.recommendation}</p>
          </div>

          {/* Optional link */}
          {issue.link && (
            <a
              href={issue.link}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-block text-xs font-medium text-primary underline underline-offset-2 hover:opacity-80 transition-opacity"
            >
              View details →
            </a>
          )}
        </div>
      ))}
    </div>
  );
}
