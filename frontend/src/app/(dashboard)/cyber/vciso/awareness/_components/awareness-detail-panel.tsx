'use client';

import {
  Users,
  CheckCircle,
  XCircle,
  Calendar,
  Clock,
  BarChart3,
} from 'lucide-react';
import { DetailPanel } from '@/components/shared/detail-panel';
import { StatusBadge } from '@/components/shared/status-badge';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Separator } from '@/components/ui/separator';
import { awarenessStatusConfig } from '@/lib/status-configs';
import { formatDate } from '@/lib/format';
import type { VCISOAwarenessProgram } from '@/types/cyber';

interface AwarenessDetailPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  program: VCISOAwarenessProgram;
}

const TYPE_LABELS: Record<string, string> = {
  training: 'Training',
  phishing_simulation: 'Phishing Simulation',
  policy_attestation: 'Policy Attestation',
};

function rateColor(rate: number): string {
  if (rate >= 80) return 'text-green-600';
  if (rate >= 60) return 'text-amber-600';
  return 'text-red-600';
}

function progressColor(rate: number): string {
  if (rate >= 80) return 'bg-green-500';
  if (rate >= 60) return 'bg-amber-500';
  return 'bg-red-500';
}

export function AwarenessDetailPanel({
  open,
  onOpenChange,
  program,
}: AwarenessDetailPanelProps) {
  const completionPct = Math.round(program.completion_rate * 100);
  const passPct = Math.round(program.pass_rate * 100);
  const pendingUsers = program.total_users - program.completed_users;

  return (
    <DetailPanel
      open={open}
      onOpenChange={onOpenChange}
      title={program.name}
      description={TYPE_LABELS[program.type] ?? program.type}
      width="xl"
    >
      <div className="space-y-6">
        {/* Overview */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Overview
          </h3>
          <div className="flex flex-wrap gap-2">
            <StatusBadge status={program.status} config={awarenessStatusConfig} />
            <Badge
              variant="secondary"
              className="text-xs"
            >
              {TYPE_LABELS[program.type] ?? program.type}
            </Badge>
          </div>
        </div>

        <Separator />

        {/* User Stats */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            User Breakdown
          </h3>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="rounded-xl border p-4 text-center">
              <div className="flex items-center justify-center gap-1.5 mb-1">
                <Users className="h-4 w-4 text-muted-foreground" />
                <p className="text-xs text-muted-foreground">Total Users</p>
              </div>
              <p className="text-2xl font-bold text-foreground">
                {program.total_users.toLocaleString()}
              </p>
            </div>
            <div className="rounded-xl border p-4 text-center">
              <div className="flex items-center justify-center gap-1.5 mb-1">
                <CheckCircle className="h-4 w-4 text-green-600" />
                <p className="text-xs text-muted-foreground">Completed</p>
              </div>
              <p className="text-2xl font-bold text-green-600">
                {program.completed_users.toLocaleString()}
              </p>
            </div>
            <div className="rounded-xl border p-4 text-center">
              <div className="flex items-center justify-center gap-1.5 mb-1">
                <CheckCircle className="h-4 w-4 text-blue-600" />
                <p className="text-xs text-muted-foreground">Passed</p>
              </div>
              <p className="text-2xl font-bold text-blue-600">
                {program.passed_users.toLocaleString()}
              </p>
            </div>
            <div className="rounded-xl border p-4 text-center">
              <div className="flex items-center justify-center gap-1.5 mb-1">
                <XCircle className="h-4 w-4 text-red-600" />
                <p className="text-xs text-muted-foreground">Failed</p>
              </div>
              <p className="text-2xl font-bold text-red-600">
                {program.failed_users.toLocaleString()}
              </p>
            </div>
          </div>

          {pendingUsers > 0 && (
            <div className="rounded-lg border border-amber-200 bg-amber-50 dark:bg-amber-900/10 p-3">
              <p className="text-sm text-amber-800 dark:text-amber-300">
                <span className="font-semibold">{pendingUsers.toLocaleString()}</span> user{pendingUsers !== 1 ? 's' : ''} have not yet completed the program.
              </p>
            </div>
          )}
        </div>

        <Separator />

        {/* Rates */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Completion & Pass Rates
          </h3>

          <div className="space-y-4">
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <BarChart3 className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Completion Rate</span>
                </div>
                <span className={`font-semibold ${rateColor(completionPct)}`}>
                  {completionPct}%
                </span>
              </div>
              <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
                <div
                  className={`h-full rounded-full transition-all ${progressColor(completionPct)}`}
                  style={{ width: `${completionPct}%` }}
                />
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <CheckCircle className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Pass Rate</span>
                </div>
                <span className={`font-semibold ${rateColor(passPct)}`}>
                  {passPct}%
                </span>
              </div>
              <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
                <div
                  className={`h-full rounded-full transition-all ${progressColor(passPct)}`}
                  style={{ width: `${passPct}%` }}
                />
              </div>
            </div>
          </div>
        </div>

        <Separator />

        {/* Timeline */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Timeline
          </h3>
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Start Date:</span>
              <span className="font-medium">{formatDate(program.start_date)}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">End Date:</span>
              <span className="font-medium">{formatDate(program.end_date)}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Created:</span>
              <span className="font-medium">{formatDate(program.created_at)}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Last Updated:</span>
              <span className="font-medium">{formatDate(program.updated_at)}</span>
            </div>
          </div>
        </div>
      </div>
    </DetailPanel>
  );
}
