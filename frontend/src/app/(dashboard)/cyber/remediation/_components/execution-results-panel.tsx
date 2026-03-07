'use client';

import { useState } from 'react';
import { CheckCircle, XCircle, Minus, ChevronDown } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import type { ExecutionResult, StepResult } from '@/types/cyber';

function stepStatusIcon(status: StepResult['status']) {
  switch (status) {
    case 'success':
      return <CheckCircle className="h-4 w-4 shrink-0 text-green-600 dark:text-green-400" aria-hidden />;
    case 'failure':
      return <XCircle className="h-4 w-4 shrink-0 text-red-600 dark:text-red-400" aria-hidden />;
    case 'skipped':
      return <Minus className="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden />;
  }
}

function StepRow({ step }: { step: StepResult }) {
  const [expanded, setExpanded] = useState(false);
  const hasOutput = Boolean(step.output);
  const hasTruncated = hasOutput && step.output!.length > 200;

  return (
    <div className="rounded-lg border bg-card">
      <div className="flex items-start gap-3 px-3 py-2.5">
        <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border bg-muted text-xs font-bold tabular-nums">
          {step.step_number}
        </div>
        {stepStatusIcon(step.status)}
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <span className="text-xs font-medium capitalize">{step.status}</span>
            <span className="text-xs tabular-nums text-muted-foreground">{step.duration_ms} ms</span>
          </div>

          {step.error && (
            <p className="mt-1 text-xs text-red-600 dark:text-red-400">{step.error}</p>
          )}

          {hasOutput && (
            <div className="mt-1.5">
              <pre className="rounded bg-muted px-2 py-1.5 text-xs font-mono leading-relaxed whitespace-pre-wrap break-all">
                {hasTruncated && !expanded
                  ? `${step.output!.slice(0, 200)}…`
                  : step.output}
              </pre>
              {hasTruncated && (
                <button
                  type="button"
                  onClick={() => setExpanded((v) => !v)}
                  className="mt-1 flex items-center gap-1 text-xs text-primary hover:underline"
                >
                  <ChevronDown
                    className={`h-3 w-3 transition-transform duration-150 ${expanded ? 'rotate-180' : ''}`}
                    aria-hidden
                  />
                  {expanded ? 'Show less' : 'Show more'}
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export function ExecutionResultsPanel({ result }: { result: ExecutionResult }) {
  return (
    <div className="space-y-5">
      {/* Header */}
      <div
        className={`flex items-start gap-3 rounded-xl border p-4 ${
          result.success
            ? 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950/20'
            : 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/20'
        }`}
      >
        {result.success
          ? <CheckCircle className="mt-0.5 h-5 w-5 shrink-0 text-green-600 dark:text-green-400" aria-hidden />
          : <XCircle className="mt-0.5 h-5 w-5 shrink-0 text-red-600 dark:text-red-400" aria-hidden />
        }
        <div className="flex-1 min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <p className={`font-semibold ${result.success ? 'text-green-800 dark:text-green-300' : 'text-red-800 dark:text-red-300'}`}>
              Execution Results
            </p>
            <Badge
              variant={result.success ? 'default' : 'destructive'}
              className="text-xs"
            >
              {result.success ? 'Success' : 'Failed'}
            </Badge>
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Completed in {(result.duration_ms / 1000).toFixed(2)}s
          </p>
        </div>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-2 gap-3">
        <div className="rounded-lg border p-3 text-center">
          <p className="text-2xl font-bold tabular-nums">
            {result.steps_executed}
            <span className="text-base font-normal text-muted-foreground">/{result.steps_total}</span>
          </p>
          <p className="text-xs text-muted-foreground">Steps Executed</p>
        </div>
        <div className="rounded-lg border p-3 text-center">
          <p className="text-2xl font-bold tabular-nums">{result.changes_applied.length}</p>
          <p className="text-xs text-muted-foreground">Changes Applied</p>
        </div>
      </div>

      {/* Step-by-step list */}
      {result.step_results.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-semibold text-muted-foreground">Steps</p>
          {result.step_results.map((step) => (
            <StepRow key={step.step_number} step={step} />
          ))}
        </div>
      )}

      {/* Changes Applied */}
      {result.changes_applied.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-semibold text-muted-foreground">Changes Applied</p>
          {result.changes_applied.map((change, idx) => (
            <div key={idx} className="rounded-lg border bg-card p-3 space-y-1.5">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-xs capitalize">
                    {change.change_type}
                  </Badge>
                  <span className="text-xs font-medium text-muted-foreground">{change.asset_id}</span>
                </div>
              </div>
              <p className="text-xs">{change.description}</p>
              {(change.old_value || change.new_value) && (
                <div className="grid grid-cols-2 gap-2 pt-1">
                  <div>
                    <p className="mb-0.5 text-xs text-muted-foreground">Before</p>
                    <code className="block rounded bg-red-50 px-2 py-1 text-xs font-mono text-red-700 dark:bg-red-950/30 dark:text-red-400">
                      {change.old_value ?? '—'}
                    </code>
                  </div>
                  <div>
                    <p className="mb-0.5 text-xs text-muted-foreground">After</p>
                    <code className="block rounded bg-green-50 px-2 py-1 text-xs font-mono text-green-700 dark:bg-green-950/30 dark:text-green-400">
                      {change.new_value ?? '—'}
                    </code>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
