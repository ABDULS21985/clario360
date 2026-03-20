'use client';

import { Badge } from '@/components/ui/badge';
import { getStatusTone } from '@/lib/data-suite/utils';
import type { Pipeline, PipelineRunStatus, PipelineStatus } from '@/lib/data-suite';

interface PipelineStatusIndicatorProps {
  status: PipelineStatus;
  lastRunStatus?: PipelineRunStatus | null;
  compact?: boolean;
}

export function PipelineStatusIndicator({
  status,
  lastRunStatus,
  compact = false,
}: PipelineStatusIndicatorProps) {
  const dotClass = getStatusTone(lastRunStatus === 'failed' ? lastRunStatus : status);

  return (
    <div className={`flex items-center gap-2 ${compact ? 'text-xs' : 'text-sm'}`}>
      <span className={`h-2.5 w-2.5 rounded-full ${dotClass}`} />
      <Badge variant="outline" className="capitalize">
        {status}
      </Badge>
      {lastRunStatus ? (
        <span className="text-muted-foreground">
          last run {lastRunStatus}
        </span>
      ) : null}
    </div>
  );
}

export function pipelineCanResume(pipeline: Pipeline): boolean {
  return pipeline.status === 'paused';
}

