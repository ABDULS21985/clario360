'use client';

import {
  BookOpen,
  Calendar,
  Clock,
  User,
  ListChecks,
  Timer,
  Database,
  Activity,
  Link2,
} from 'lucide-react';
import { DetailPanel } from '@/components/shared/detail-panel';
import { StatusBadge } from '@/components/shared/status-badge';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { formatDate } from '@/lib/format';
import { cn } from '@/lib/utils';
import { playbookStatusConfig, simulationResultConfig } from '@/lib/status-configs';
import type { VCISOPlaybook } from '@/types/cyber';

interface PlaybookDetailPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  playbook: VCISOPlaybook;
}

function isOverdue(dateStr: string): boolean {
  try {
    return new Date(dateStr) < new Date();
  } catch {
    return false;
  }
}

export function PlaybookDetailPanel({
  open,
  onOpenChange,
  playbook,
}: PlaybookDetailPanelProps) {
  const overdue = isOverdue(playbook.next_test_date);

  return (
    <DetailPanel
      open={open}
      onOpenChange={onOpenChange}
      title={playbook.name}
      description="Crisis Playbook Details"
      width="xl"
    >
      <div className="space-y-6">
        {/* Overview */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Overview
          </h3>
          <div className="flex flex-wrap gap-2 mb-2">
            <StatusBadge status={playbook.status} config={playbookStatusConfig} />
            {playbook.last_simulation_result && (
              <StatusBadge
                status={playbook.last_simulation_result}
                config={simulationResultConfig}
              />
            )}
          </div>
        </div>

        <Separator />

        {/* Scenario */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Scenario
          </h3>
          <p className="text-sm text-foreground leading-relaxed whitespace-pre-wrap">
            {playbook.scenario || 'No scenario description provided.'}
          </p>
        </div>

        <Separator />

        {/* Recovery Objectives */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Recovery Objectives
          </h3>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="rounded-xl border p-4 text-center">
              <div className="flex items-center justify-center gap-1.5 mb-1">
                <Timer className="h-4 w-4 text-muted-foreground" />
                <p className="text-xs text-muted-foreground">RTO</p>
              </div>
              <p className="text-2xl font-bold text-foreground">
                {playbook.rto_hours != null ? `${playbook.rto_hours}h` : '--'}
              </p>
              <p className="text-xs text-muted-foreground mt-1">Recovery Time Objective</p>
            </div>
            <div className="rounded-xl border p-4 text-center">
              <div className="flex items-center justify-center gap-1.5 mb-1">
                <Database className="h-4 w-4 text-muted-foreground" />
                <p className="text-xs text-muted-foreground">RPO</p>
              </div>
              <p className="text-2xl font-bold text-foreground">
                {playbook.rpo_hours != null ? `${playbook.rpo_hours}h` : '--'}
              </p>
              <p className="text-xs text-muted-foreground mt-1">Recovery Point Objective</p>
            </div>
          </div>
        </div>

        <Separator />

        {/* Playbook Details */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Details
          </h3>
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <User className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Owner:</span>
              <span className="font-medium">{playbook.owner_name || 'Unassigned'}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <ListChecks className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Steps:</span>
              <span className="font-medium">{playbook.steps_count}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Activity className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Last Simulation:</span>
              {playbook.last_simulation_result ? (
                <StatusBadge
                  status={playbook.last_simulation_result}
                  config={simulationResultConfig}
                  size="sm"
                />
              ) : (
                <span className="text-muted-foreground">Never tested</span>
              )}
            </div>
          </div>
        </div>

        <Separator />

        {/* Testing Schedule */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Testing Schedule
          </h3>
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Last Tested:</span>
              <span className="font-medium">
                {playbook.last_tested_at ? formatDate(playbook.last_tested_at) : 'Never'}
              </span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Next Test:</span>
              <span
                className={cn(
                  'font-medium',
                  overdue && 'text-red-600',
                )}
              >
                {formatDate(playbook.next_test_date)}
                {overdue && ' (Overdue)'}
              </span>
            </div>
          </div>
        </div>

        {/* Dependencies */}
        {playbook.dependencies.length > 0 && (
          <>
            <Separator />
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Dependencies
              </h3>
              <div className="flex flex-wrap gap-1.5">
                {playbook.dependencies.map((dep) => (
                  <Badge key={dep} variant="secondary" className="text-xs">
                    <Link2 className="mr-1 h-3 w-3" />
                    {dep}
                  </Badge>
                ))}
              </div>
            </div>
          </>
        )}

        <Separator />

        {/* Timestamps */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Timestamps
          </h3>
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Created:</span>
              <span className="font-medium">{formatDate(playbook.created_at)}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Updated:</span>
              <span className="font-medium">{formatDate(playbook.updated_at)}</span>
            </div>
          </div>
        </div>
      </div>
    </DetailPanel>
  );
}
