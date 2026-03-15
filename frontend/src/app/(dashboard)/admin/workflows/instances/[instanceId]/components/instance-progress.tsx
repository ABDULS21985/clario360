'use client';

import { Card, CardContent } from '@/components/ui/card';
import { formatDuration } from '@/lib/format';
import type { WorkflowInstance } from '@/types/models';

interface InstanceProgressProps {
  instance: WorkflowInstance;
}

export function InstanceProgress({ instance }: InstanceProgressProps) {
  const totalSteps = instance.total_steps ?? 0;
  const completedSteps = instance.completed_steps ?? 0;
  const percent =
    totalSteps > 0
      ? Math.round((completedSteps / totalSteps) * 100)
      : 0;

  const startTime = new Date(instance.started_at).getTime();
  const endTime = instance.completed_at
    ? new Date(instance.completed_at).getTime()
    : Date.now();
  const durationSec = Math.floor((endTime - startTime) / 1000);

  return (
    <Card>
      <CardContent className="py-4">
        <div className="flex items-center justify-between text-sm mb-2">
          <span className="text-muted-foreground">
            Step {completedSteps} of {totalSteps}
          </span>
          <span className="font-medium">{percent}%</span>
        </div>
        <div className="h-2 rounded-full bg-muted overflow-hidden">
          <div
            className="h-full rounded-full bg-primary transition-all duration-500"
            style={{ width: `${percent}%` }}
          />
        </div>
        <div className="flex items-center justify-between text-xs text-muted-foreground mt-2">
          <span>
            {instance.current_step_name
              ? `Current: ${instance.current_step_name}`
              : instance.status === 'completed'
                ? 'All steps completed'
                : '—'}
          </span>
          <span>Duration: {formatDuration(durationSec)}</span>
        </div>
      </CardContent>
    </Card>
  );
}
