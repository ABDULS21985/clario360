'use client';

import { CheckCircle, XCircle, Clock, SkipForward, Loader2 } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { formatDateTime, formatDuration, titleCase } from '@/lib/format';
import { getStepIcon, formatStepType } from '@/lib/workflow-utils';
import type { StepExecution } from '@/types/models';

const statusIcons: Record<string, React.ElementType> = {
  completed: CheckCircle,
  running: Loader2,
  failed: XCircle,
  pending: Clock,
  skipped: SkipForward,
  cancelled: XCircle,
};

const statusColors: Record<string, string> = {
  completed: 'text-green-600',
  running: 'text-blue-600 animate-spin',
  failed: 'text-red-600',
  pending: 'text-gray-400',
  skipped: 'text-gray-400',
  cancelled: 'text-gray-400',
};

interface StepHistoryProps {
  steps: StepExecution[];
}

export function StepHistory({ steps }: StepHistoryProps) {
  if (steps.length === 0) {
    return null;
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">
          Step Execution History
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="relative">
          {/* Vertical timeline line */}
          <div className="absolute left-4 top-0 bottom-0 w-px bg-border" />

          <div className="space-y-4">
            {steps.map((step, idx) => {
              const StatusIcon = statusIcons[step.status] ?? Clock;
              const StepIcon = getStepIcon(step.step_type);
              const color = statusColors[step.status] ?? 'text-gray-400';

              return (
                <div key={step.id || idx} className="relative pl-10">
                  {/* Timeline dot */}
                  <div className="absolute left-2 top-1">
                    <StatusIcon className={`h-4 w-4 ${color}`} />
                  </div>

                  <div className="border rounded-lg p-3">
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex items-center gap-2 min-w-0">
                        <StepIcon className="h-4 w-4 text-muted-foreground shrink-0" />
                        <span className="text-sm font-medium truncate">
                          {step.step_name ?? step.step_id}
                        </span>
                        <Badge variant="outline" className="text-[10px] shrink-0">
                          {formatStepType(step.step_type)}
                        </Badge>
                      </div>
                      <Badge
                        variant={
                          step.status === 'completed'
                            ? 'default'
                            : step.status === 'failed'
                              ? 'destructive'
                              : 'secondary'
                        }
                        className="text-[10px] shrink-0"
                      >
                        {titleCase(step.status)}
                      </Badge>
                    </div>

                    <div className="mt-2 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-2 text-xs text-muted-foreground">
                      {step.started_at && (
                        <div>
                          <span className="block text-[10px] uppercase font-medium">
                            Started
                          </span>
                          {formatDateTime(step.started_at)}
                        </div>
                      )}
                      {step.completed_at && (
                        <div>
                          <span className="block text-[10px] uppercase font-medium">
                            Completed
                          </span>
                          {formatDateTime(step.completed_at)}
                        </div>
                      )}
                      {step.duration_ms != null && (
                        <div>
                          <span className="block text-[10px] uppercase font-medium">
                            Duration
                          </span>
                          {formatDuration(Math.round(step.duration_ms / 1000))}
                        </div>
                      )}
                      {step.duration_ms == null && step.duration_seconds != null && (
                        <div>
                          <span className="block text-[10px] uppercase font-medium">
                            Duration
                          </span>
                          {formatDuration(step.duration_seconds)}
                        </div>
                      )}
                      {step.assigned_to && (
                        <div>
                          <span className="block text-[10px] uppercase font-medium">
                            Assignee
                          </span>
                          {step.assigned_to}
                        </div>
                      )}
                    </div>

                    {(step.error_message ?? step.error) && (
                      <div className="mt-2 text-xs text-red-600 bg-red-50 rounded p-2">
                        {step.error_message ?? step.error}
                      </div>
                    )}

                    {(step.output_data ?? step.output) &&
                      Object.keys(step.output_data ?? step.output ?? {}).length > 0 && (
                        <details className="mt-2">
                          <summary className="text-xs text-muted-foreground cursor-pointer hover:text-foreground">
                            Output data
                          </summary>
                          <pre className="mt-1 text-xs bg-muted rounded p-2 overflow-x-auto">
                            {JSON.stringify(step.output_data ?? step.output, null, 2)}
                          </pre>
                        </details>
                      )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
