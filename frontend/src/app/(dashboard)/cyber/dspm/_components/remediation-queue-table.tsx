'use client';

import { Badge } from '@/components/ui/badge';
import { AlertTriangle, Clock } from 'lucide-react';
import type { DSPMRemediation, DSPMRemediationStatus, DSPMFindingType } from '@/types/cyber';

interface RemediationQueueTableProps {
  remediations: DSPMRemediation[];
  onRowClick: (id: string) => void;
}

const STATUS_COLORS: Record<DSPMRemediationStatus, string> = {
  open: 'bg-blue-100 text-blue-700',
  in_progress: 'bg-amber-100 text-amber-800',
  awaiting_approval: 'bg-purple-100 text-purple-800',
  completed: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
  cancelled: 'bg-gray-100 text-gray-700',
  rolled_back: 'bg-orange-100 text-orange-800',
  exception_granted: 'bg-teal-100 text-teal-800',
};

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'bg-red-100 text-red-700',
  high: 'bg-orange-100 text-orange-700',
  medium: 'bg-amber-100 text-amber-800',
  low: 'bg-blue-100 text-blue-700',
  info: 'bg-gray-100 text-gray-700',
};

function formatFindingType(type: DSPMFindingType): string {
  return type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

function formatStatusLabel(status: DSPMRemediationStatus): string {
  return status.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

function formatTimeRemaining(dueAt: string): { text: string; color: string } {
  const now = new Date();
  const due = new Date(dueAt);
  const diffMs = due.getTime() - now.getTime();

  if (diffMs <= 0) {
    return { text: 'Overdue', color: 'text-red-600' };
  }

  const totalMinutes = Math.floor(diffMs / 60000);
  const totalHours = Math.floor(totalMinutes / 60);
  const totalDays = Math.floor(totalHours / 24);

  let text: string;
  if (totalDays > 0) {
    text = `${totalDays}d`;
  } else if (totalHours > 0) {
    const remainingMinutes = totalMinutes % 60;
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

export function RemediationQueueTable({ remediations, onRowClick }: RemediationQueueTableProps) {
  if (remediations.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed p-12 text-center">
        <p className="text-sm text-muted-foreground">No remediations found.</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/50">
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Title</th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Severity</th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Asset</th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Assignee</th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">SLA</th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Progress</th>
          </tr>
        </thead>
        <tbody>
          {remediations.map((rem) => {
            const progressPct = rem.total_steps > 0
              ? Math.round((rem.current_step / rem.total_steps) * 100)
              : 0;

            return (
              <tr
                key={rem.id}
                onClick={() => onRowClick(rem.id)}
                className="cursor-pointer border-b transition-colors hover:bg-muted/30"
              >
                <td className="px-4 py-3">
                  <div>
                    <p className="font-medium">{rem.title}</p>
                    <Badge variant="outline" className="mt-1 text-xs capitalize">
                      {formatFindingType(rem.finding_type)}
                    </Badge>
                  </div>
                </td>
                <td className="px-4 py-3">
                  <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${SEVERITY_COLORS[rem.severity] ?? 'bg-gray-100 text-gray-700'}`}>
                    {rem.severity}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span className="text-sm">{rem.data_asset_name ?? '---'}</span>
                </td>
                <td className="px-4 py-3">
                  <span className="text-sm">{rem.assigned_to ?? rem.assigned_team ?? '---'}</span>
                </td>
                <td className="px-4 py-3">
                  <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_COLORS[rem.status] ?? 'bg-gray-100 text-gray-700'}`}>
                    {formatStatusLabel(rem.status)}
                  </span>
                </td>
                <td className="px-4 py-3">
                  {rem.sla_breached ? (
                    <div className="flex items-center gap-1.5">
                      <AlertTriangle className="h-3.5 w-3.5 text-red-600" />
                      <span className="text-xs font-semibold text-red-600">SLA Breached</span>
                    </div>
                  ) : rem.sla_due_at ? (
                    (() => {
                      const { text, color } = formatTimeRemaining(rem.sla_due_at);
                      return (
                        <div className="flex items-center gap-1.5">
                          <Clock className={`h-3.5 w-3.5 ${color}`} />
                          <span className={`text-xs font-medium ${color}`}>{text}</span>
                        </div>
                      );
                    })()
                  ) : (
                    <span className="text-xs text-muted-foreground">---</span>
                  )}
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full rounded-full bg-primary transition-all"
                        style={{ width: `${progressPct}%` }}
                      />
                    </div>
                    <span className="text-xs tabular-nums text-muted-foreground">
                      {rem.current_step}/{rem.total_steps} ({progressPct}%)
                    </span>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
