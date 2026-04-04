'use client';

import { Progress } from '@/components/ui/progress';
import { attendeeCounts, quorumProgress } from '@/lib/enterprise';
import { cn } from '@/lib/utils';
import type { ActaAttendee } from '@/types/suites';

interface QuorumIndicatorProps {
  attendance: ActaAttendee[];
  quorumRequired: number;
}

export function QuorumIndicator({ attendance, quorumRequired }: QuorumIndicatorProps) {
  const counts = attendeeCounts(attendance);
  const progress = quorumProgress(attendance, quorumRequired);

  return (
    <div className="rounded-xl border bg-card p-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm font-medium">Attendance</p>
          <p className="text-sm text-muted-foreground">
            {counts.countedForQuorum} of {counts.total} members counted for quorum
          </p>
        </div>
        <div
          className={cn(
            'rounded-full px-2.5 py-1 text-xs font-medium',
            progress.met ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700',
          )}
        >
          {progress.met ? 'Quorum Met' : 'Quorum Not Met'}
        </div>
      </div>
      <Progress
        value={progress.percent}
        indicatorClassName={progress.met ? 'bg-emerald-500' : 'bg-rose-500'}
        className="mt-4 h-2"
      />
      <div className="mt-3 flex flex-wrap gap-4 text-xs text-muted-foreground">
        <span>{counts.present} present</span>
        <span>{counts.proxy} proxy</span>
        <span>{counts.absent} absent</span>
        <span>{counts.excused} excused</span>
        <span>{quorumRequired} required</span>
      </div>
    </div>
  );
}
