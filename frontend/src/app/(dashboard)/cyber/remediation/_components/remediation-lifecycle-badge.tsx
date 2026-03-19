'use client';

import { cn } from '@/lib/utils';
import type { RemediationStatus } from '@/types/cyber';

const STATUS_CONFIG: Record<RemediationStatus, { label: string; color: string }> = {
  draft: { label: 'Draft', color: 'bg-gray-100 text-gray-700' },
  pending_approval: { label: 'Pending Approval', color: 'bg-amber-100 text-amber-800' },
  approved: { label: 'Approved', color: 'bg-blue-100 text-blue-800' },
  rejected: { label: 'Rejected', color: 'bg-red-100 text-red-800' },
  revision_requested: { label: 'Revision Requested', color: 'bg-orange-100 text-orange-800' },
  dry_run_running: { label: 'Dry Run…', color: 'bg-purple-100 text-purple-800 animate-pulse' },
  dry_run_completed: { label: 'Dry Run OK', color: 'bg-purple-100 text-purple-700' },
  dry_run_failed: { label: 'Dry Run Failed', color: 'bg-red-100 text-red-800' },
  execution_pending: { label: 'Execution Pending', color: 'bg-blue-100 text-blue-800' },
  executing: { label: 'Executing…', color: 'bg-blue-100 text-blue-800 animate-pulse' },
  executed: { label: 'Executed', color: 'bg-teal-100 text-teal-800' },
  execution_failed: { label: 'Execution Failed', color: 'bg-red-100 text-red-800' },
  verification_pending: { label: 'Verifying…', color: 'bg-teal-100 text-teal-700 animate-pulse' },
  verified: { label: 'Verified', color: 'bg-green-100 text-green-800' },
  verification_failed: { label: 'Verification Failed', color: 'bg-red-100 text-red-800' },
  rollback_pending: { label: 'Rollback Pending', color: 'bg-orange-100 text-orange-800' },
  rolling_back: { label: 'Rolling Back…', color: 'bg-orange-100 text-orange-800 animate-pulse' },
  rolled_back: { label: 'Rolled Back', color: 'bg-gray-100 text-gray-700' },
  rollback_failed: { label: 'Rollback Failed', color: 'bg-red-100 text-red-800' },
  closed: { label: 'Closed', color: 'bg-green-100 text-green-700' },
};

interface RemediationLifecycleBadgeProps {
  status: RemediationStatus;
  className?: string;
}

export function RemediationLifecycleBadge({ status, className }: RemediationLifecycleBadgeProps) {
  const cfg = STATUS_CONFIG[status] ?? { label: status, color: 'bg-muted text-muted-foreground' };
  return (
    <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium', cfg.color, className)}>
      {cfg.label}
    </span>
  );
}
