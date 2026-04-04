'use client';

import { AlertTriangle, Clock } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { CyberSeverity } from '@/types/cyber';

interface SLATrackerProps {
  slaDueAt?: string;
  slaBreached: boolean;
  severity?: CyberSeverity;
}

const SLA_TARGETS: Record<CyberSeverity, string> = {
  critical: '4h',
  high: '24h',
  medium: '72h',
  low: '168h',
  info: '---',
};

function computeTimeRemaining(dueAt: string): {
  text: string;
  color: string;
} {
  const now = new Date();
  const due = new Date(dueAt);
  const diffMs = due.getTime() - now.getTime();

  if (diffMs <= 0) {
    return { text: 'Overdue', color: 'text-red-600' };
  }

  const totalMinutes = Math.floor(diffMs / 60000);
  const totalHours = Math.floor(totalMinutes / 60);
  const totalDays = Math.floor(totalHours / 24);
  const remainingHours = totalHours % 24;
  const remainingMinutes = totalMinutes % 60;

  let text: string;
  if (totalDays > 0) {
    text = `${totalDays}d ${remainingHours}h`;
  } else if (totalHours > 0) {
    text = `${totalHours}h ${remainingMinutes}m`;
  } else {
    text = `${totalMinutes}m`;
  }

  let color: string;
  if (totalHours >= 24) {
    color = 'text-green-600';
  } else if (totalHours >= 4) {
    color = 'text-amber-600';
  } else {
    color = 'text-red-600';
  }

  return { text, color };
}

export function SLATracker({ slaDueAt, slaBreached, severity }: SLATrackerProps) {
  if (slaBreached) {
    return (
      <div className="inline-flex items-center gap-1.5">
        <AlertTriangle className="h-3.5 w-3.5 text-red-600" />
        <span className="text-xs font-semibold text-red-600">SLA BREACHED</span>
      </div>
    );
  }

  if (!slaDueAt) {
    return (
      <span className="text-xs text-muted-foreground">No SLA</span>
    );
  }

  const { text, color } = computeTimeRemaining(slaDueAt);
  const slaTarget = severity ? SLA_TARGETS[severity] : undefined;

  return (
    <div
      className="inline-flex items-center gap-1.5"
      title={slaTarget ? `SLA target: ${slaTarget} (${severity})` : undefined}
    >
      <Clock className={cn('h-3.5 w-3.5', color)} />
      <span className={cn('text-xs font-medium tabular-nums', color)}>{text}</span>
    </div>
  );
}
