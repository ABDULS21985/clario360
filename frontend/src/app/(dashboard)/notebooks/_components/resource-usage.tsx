'use client';

import { Progress } from '@/components/ui/progress';
import { cn } from '@/lib/utils';

interface ResourceUsageProps {
  cpuPercent: number;
  memoryMB: number;
  memoryLimitMB: number;
}

export function ResourceUsage({ cpuPercent, memoryMB, memoryLimitMB }: ResourceUsageProps) {
  const memoryPercent = memoryLimitMB > 0 ? (memoryMB / memoryLimitMB) * 100 : 0;
  const tone = memoryPercent > 95 ? 'bg-red-500' : memoryPercent > 80 ? 'bg-amber-500' : 'bg-emerald-500';

  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span>CPU</span>
          <span>{cpuPercent.toFixed(0)}%</span>
        </div>
        <Progress value={Math.max(0, Math.min(100, cpuPercent))} className="h-2" />
      </div>
      <div className="space-y-1">
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span>Memory</span>
          <span>
            {memoryMB} / {memoryLimitMB || '?'} MB
          </span>
        </div>
        <Progress
          value={Math.max(0, Math.min(100, memoryPercent))}
          className="h-2"
          indicatorClassName={cn(tone)}
        />
      </div>
    </div>
  );
}
