'use client';

import { CheckCircle, XCircle } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import type { VerificationResult } from '@/types/cyber';

export function VerificationResultsPanel({ result }: { result: VerificationResult }) {
  return (
    <div className="space-y-5">
      {/* Header */}
      <div
        className={`flex items-start gap-3 rounded-xl border p-4 ${
          result.verified
            ? 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950/20'
            : 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/20'
        }`}
      >
        {result.verified
          ? <CheckCircle className="mt-0.5 h-5 w-5 shrink-0 text-green-600 dark:text-green-400" aria-hidden />
          : <XCircle className="mt-0.5 h-5 w-5 shrink-0 text-red-600 dark:text-red-400" aria-hidden />
        }
        <div className="flex-1 min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <p className={`font-semibold ${result.verified ? 'text-green-800 dark:text-green-300' : 'text-red-800 dark:text-red-300'}`}>
              Verification Results
            </p>
            <Badge
              variant={result.verified ? 'default' : 'destructive'}
              className="text-xs"
            >
              {result.verified ? 'Verified' : 'Failed'}
            </Badge>
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Completed in {(result.duration_ms / 1000).toFixed(2)}s
          </p>
        </div>
      </div>

      {/* Checks */}
      {result.checks.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-semibold text-muted-foreground">
            Checks ({result.checks.filter((c) => c.passed).length}/{result.checks.length} passed)
          </p>
          {result.checks.map((check, idx) => (
            <div key={idx} className="rounded-lg border bg-card p-3 space-y-2">
              <div className="flex items-center gap-2">
                {check.passed
                  ? <CheckCircle className="h-4 w-4 shrink-0 text-green-600 dark:text-green-400" aria-hidden />
                  : <XCircle className="h-4 w-4 shrink-0 text-red-600 dark:text-red-400" aria-hidden />
                }
                <span className="text-sm font-medium">{check.name}</span>
              </div>

              <div className="grid grid-cols-1 gap-2 pl-6 sm:grid-cols-2">
                <div>
                  <p className="mb-0.5 text-xs text-muted-foreground">Expected</p>
                  <p className="rounded bg-muted px-2 py-1 text-xs font-mono">{check.expected}</p>
                </div>
                <div>
                  <p className="mb-0.5 text-xs text-muted-foreground">Actual</p>
                  <p
                    className={`rounded px-2 py-1 text-xs font-mono ${
                      check.passed
                        ? 'bg-muted'
                        : 'bg-red-50 text-red-700 dark:bg-red-950/30 dark:text-red-400'
                    }`}
                  >
                    {check.actual}
                  </p>
                </div>
              </div>

              {check.notes && (
                <p className="pl-6 text-xs text-muted-foreground">{check.notes}</p>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Failure reason */}
      {!result.verified && result.failure_reason && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 dark:border-red-800 dark:bg-red-950/20">
          <p className="text-xs font-semibold text-red-700 dark:text-red-400">Failure Reason</p>
          <p className="mt-1 text-sm text-red-700 dark:text-red-300">{result.failure_reason}</p>
        </div>
      )}
    </div>
  );
}
