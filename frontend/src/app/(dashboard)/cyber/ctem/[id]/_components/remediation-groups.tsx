'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronRight, Play, Clock, CheckCircle2, PauseCircle, ShieldCheck } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiGet, apiPut, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { showSuccess, showError } from '@/lib/toast';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import type { CTEMRemediationGroup, CTEMRemediationGroupStatus } from '@/types/cyber';

const GROUP_STATUS_CONFIG: Record<CTEMRemediationGroupStatus, { label: string; color: string; icon: typeof Clock }> = {
  planned: { label: 'Planned', color: 'bg-blue-100 text-blue-800 dark:bg-blue-950/30 dark:text-blue-400', icon: Clock },
  in_progress: { label: 'In Progress', color: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-950/30 dark:text-yellow-400', icon: Play },
  completed: { label: 'Completed', color: 'bg-green-100 text-green-800 dark:bg-green-950/30 dark:text-green-400', icon: CheckCircle2 },
  deferred: { label: 'Deferred', color: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300', icon: PauseCircle },
  accepted: { label: 'Accepted', color: 'bg-purple-100 text-purple-800 dark:bg-purple-950/30 dark:text-purple-400', icon: ShieldCheck },
};

const EFFORT_COLORS: Record<string, string> = {
  low: 'text-green-600',
  medium: 'text-yellow-600',
  high: 'text-red-600',
};

function GroupCard({
  group,
  onRefresh,
}: {
  group: CTEMRemediationGroup;
  onRefresh: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const [updating, setUpdating] = useState(false);
  const config = GROUP_STATUS_CONFIG[group.status] ?? GROUP_STATUS_CONFIG.planned;
  const StatusIcon = config.icon;

  const handleStatusChange = async (newStatus: CTEMRemediationGroupStatus) => {
    setUpdating(true);
    try {
      await apiPut(API_ENDPOINTS.CYBER_CTEM_REMEDIATION_GROUP_STATUS(group.id), { status: newStatus });
      showSuccess('Group status updated');
      onRefresh();
    } catch {
      showError('Failed to update group status');
    } finally {
      setUpdating(false);
    }
  };

  const handleExecute = async () => {
    setUpdating(true);
    try {
      await apiPost(API_ENDPOINTS.CYBER_CTEM_REMEDIATION_GROUP_EXECUTE(group.id));
      showSuccess('Remediation group execution started');
      onRefresh();
    } catch {
      showError('Failed to execute remediation group');
    } finally {
      setUpdating(false);
    }
  };

  return (
    <div className="rounded-xl border bg-card">
      <button
        type="button"
        className="flex w-full items-start justify-between gap-3 px-4 py-3 text-left"
        onClick={() => setExpanded((v) => !v)}
        aria-expanded={expanded}
      >
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-semibold">{group.title}</span>
            <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${config.color}`}>
              <StatusIcon className="h-3 w-3" />
              {config.label}
            </span>
          </div>
          <p className="text-xs text-muted-foreground line-clamp-1">{group.description}</p>
        </div>
        <div className="flex shrink-0 items-center gap-3">
          <div className="text-right">
            <p className="text-xs tabular-nums text-muted-foreground">{group.finding_count} finding{group.finding_count !== 1 ? 's' : ''}</p>
            {group.affected_asset_count > 0 && (
              <p className="text-xs tabular-nums text-muted-foreground">{group.affected_asset_count} asset{group.affected_asset_count !== 1 ? 's' : ''}</p>
            )}
          </div>
          <ChevronRight
            className={`h-4 w-4 shrink-0 text-muted-foreground transition-transform duration-150 ${expanded ? 'rotate-90' : ''}`}
            aria-hidden
          />
        </div>
      </button>

      {expanded && (
        <div className="space-y-3 border-t px-4 pb-4 pt-3">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <div>
              <p className="text-xs text-muted-foreground">Type</p>
              <p className="text-sm font-medium capitalize">{group.type.replace(/_/g, ' ')}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Effort</p>
              <p className={`text-sm font-medium capitalize ${EFFORT_COLORS[group.effort] ?? ''}`}>{group.effort}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Priority Group</p>
              <p className="text-sm font-medium tabular-nums">P{group.priority_group}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Max Priority</p>
              <p className="text-sm font-medium tabular-nums">{Math.round(group.max_priority_score)}</p>
            </div>
          </div>

          {group.estimated_days != null && (
            <p className="text-xs text-muted-foreground">Estimated: ~{group.estimated_days} day{group.estimated_days !== 1 ? 's' : ''}</p>
          )}

          {group.score_reduction != null && (
            <p className="text-xs text-muted-foreground">
              Score reduction: <span className="font-medium text-green-600">-{group.score_reduction.toFixed(1)}</span>
            </p>
          )}

          {(group.cve_ids?.length ?? 0) > 0 && (
            <div className="flex flex-wrap gap-1">
              {group.cve_ids.map((cve) => (
                <Badge key={cve} variant="outline" className="text-xs">{cve}</Badge>
              ))}
            </div>
          )}

          <div className="flex flex-wrap items-center gap-2 pt-1">
            <Select
              value={group.status}
              onValueChange={(v) => handleStatusChange(v as CTEMRemediationGroupStatus)}
              disabled={updating}
            >
              <SelectTrigger className="h-8 w-40 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(GROUP_STATUS_CONFIG).map(([value, cfg]) => (
                  <SelectItem key={value} value={value}>{cfg.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>

            {(group.status === 'planned' || group.status === 'in_progress') && (
              <Button size="sm" variant="outline" className="h-8 gap-1 text-xs" onClick={handleExecute} disabled={updating}>
                <Play className="h-3 w-3" />
                Execute
              </Button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

interface RemediationGroupsProps {
  assessmentId: string;
}

export function RemediationGroups({ assessmentId }: RemediationGroupsProps) {
  const { data, isLoading, refetch } = useQuery({
    queryKey: [`ctem-remediation-groups-${assessmentId}`],
    queryFn: () =>
      apiGet<{ data: CTEMRemediationGroup[] }>(API_ENDPOINTS.CYBER_CTEM_ASSESSMENT_REMEDIATION_GROUPS(assessmentId)),
  });

  const groups = data?.data ?? [];

  if (isLoading) return <LoadingSkeleton variant="card" />;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <h3 className="text-base font-semibold">Remediation Groups</h3>
        <Badge variant="secondary" className="tabular-nums">{groups.length}</Badge>
      </div>

      {groups.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted-foreground">
          No remediation groups available. Groups are generated during the mobilization phase.
        </p>
      ) : (
        <div className="space-y-3">
          {groups.map((group) => (
            <GroupCard key={group.id} group={group} onRefresh={() => void refetch()} />
          ))}
        </div>
      )}
    </div>
  );
}
