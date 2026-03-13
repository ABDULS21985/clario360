'use client';

import { CheckCircle, XCircle, AlertCircle, ChevronRight } from 'lucide-react';
import { ConfidenceGauge } from './confidence-gauge';
import { ConfidenceFactors } from './confidence-factors';
import type { AlertExplanation } from '@/types/cyber';

interface AlertExplanationPanelProps {
  explanation: AlertExplanation;
  confidenceScore: number;
}

export function AlertExplanationPanel({ explanation, confidenceScore }: AlertExplanationPanelProps) {
  return (
    <div className="space-y-6">
      {/* AI Summary */}
      <div className="rounded-xl border bg-gradient-to-br from-slate-50 to-blue-50/30 p-5 dark:from-slate-900 dark:to-blue-950/20">
        <div className="flex items-start gap-4">
          <ConfidenceGauge score={confidenceScore} size="md" />
          <div className="flex-1">
            <p className="text-sm font-semibold text-foreground">AI Analysis</p>
            <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{explanation.summary}</p>
          </div>
        </div>
      </div>

      {/* Detection Reason */}
      <div>
        <h4 className="mb-2 text-sm font-semibold">Why was this alert triggered?</h4>
        <p className="text-sm leading-relaxed text-muted-foreground">{explanation.reason}</p>
      </div>

      {/* Matched Conditions */}
      {(explanation.matched_conditions?.length ?? 0) > 0 && (
        <div>
          <h4 className="mb-2 text-sm font-semibold">Matched Conditions</h4>
          <div className="space-y-1.5">
            {explanation.matched_conditions?.map((cond, i) => (
              <div key={i} className="flex items-start gap-2">
                <CheckCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-red-500" />
                <span className="text-sm">{cond}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Confidence Factors */}
      {(explanation.confidence_factors?.length ?? 0) > 0 && (
        <div>
          <h4 className="mb-2 text-sm font-semibold">Confidence Factors</h4>
          <ConfidenceFactors factors={explanation.confidence_factors} />
        </div>
      )}

      {/* Recommended Actions */}
      {(explanation.recommended_actions?.length ?? 0) > 0 && (
        <div>
          <h4 className="mb-2 text-sm font-semibold">Recommended Actions</h4>
          <ol className="space-y-1.5">
            {explanation.recommended_actions?.map((action, i) => (
              <li key={i} className="flex items-start gap-2 rounded-md border border-primary/20 bg-primary/5 p-2.5">
                <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                  {i + 1}
                </span>
                <span className="text-sm">{action}</span>
              </li>
            ))}
          </ol>
        </div>
      )}

      {/* False Positive Indicators */}
      {(explanation.false_positive_indicators?.length ?? 0) > 0 && (
        <div>
          <h4 className="mb-2 text-sm font-semibold text-green-700">False Positive Indicators</h4>
          <div className="space-y-1.5 rounded-lg border border-green-200 bg-green-50/50 p-3 dark:border-green-900 dark:bg-green-950/20">
            {explanation.false_positive_indicators?.map((fp, i) => (
              <div key={i} className="flex items-start gap-2">
                <XCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-green-600" />
                <span className="text-sm text-green-800 dark:text-green-300">{fp}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Indicator Matches */}
      {(explanation.indicator_matches?.length ?? 0) > 0 && (
        <div>
          <h4 className="mb-2 text-sm font-semibold">Threat Intelligence Matches</h4>
          <div className="space-y-1.5">
            {explanation.indicator_matches!.map((ind, i) => (
              <div key={i} className="flex items-center gap-3 rounded-md border bg-amber-50/50 p-2.5 dark:bg-amber-950/20">
                <AlertCircle className="h-4 w-4 shrink-0 text-amber-600" />
                <div className="min-w-0 flex-1">
                  <span className="font-mono text-xs font-medium">{ind.value}</span>
                  <span className="ml-2 text-xs text-muted-foreground">{ind.type} · {ind.source}</span>
                </div>
                <span className="text-xs font-medium">{Math.round(ind.confidence * 100)}%</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
