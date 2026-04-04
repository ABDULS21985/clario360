'use client';

import { format } from 'date-fns';
import { useQuery } from '@tanstack/react-query';

import { Badge } from '@/components/ui/badge';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { getRuleTypeLabel } from '@/lib/cyber-rules';
import type { DetectionRule } from '@/types/cyber';

interface UserMinimal {
  full_name: string;
  email: string;
}

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/**
 * Resolves a creator field to a human-readable display label.
 * - null / undefined / empty → "System"
 * - UUID string → fetches the user from IAM and returns full_name or email
 * - Any other string → returned as-is (already human-readable)
 */
function useCreatorLabel(createdBy: string | null | undefined): string {
  const isUUID = Boolean(createdBy && UUID_PATTERN.test(createdBy));

  const { data } = useQuery<UserMinimal>({
    queryKey: ['user-mini', createdBy],
    queryFn: () => apiGet<UserMinimal>(API_ENDPOINTS.USER_DETAIL(createdBy!)),
    enabled: isUUID,
    staleTime: 10 * 60_000, // user profiles rarely change
    retry: false,
  });

  if (!createdBy) return 'System';
  if (!isUUID) return createdBy;
  return data?.full_name || data?.email || 'System';
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[22px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-4 shadow-[var(--card-shadow)]">
      <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-2 text-xl font-semibold text-slate-950">{value}</p>
    </div>
  );
}

export function RuleOverview({ rule }: { rule: DetectionRule }) {
  const creatorLabel = useCreatorLabel(rule.created_by ?? null);

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard label="Rule Type" value={getRuleTypeLabel(rule.rule_type)} />
        <MetricCard label="Trigger Count" value={rule.trigger_count.toLocaleString()} />
        <MetricCard label="Mapped Techniques" value={String(rule.mitre_technique_ids.length)} />
        <MetricCard label="Confidence" value={`${Math.round(rule.base_confidence * 100)}%`} />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1.3fr_1fr]">
        <div className="rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
          <p className="text-sm font-medium">Description</p>
          <p className="mt-3 text-sm leading-7 text-slate-700">
            {rule.description || 'No description provided for this rule.'}
          </p>

          <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Created</p>
              <p className="mt-2 text-sm text-slate-900">{rule.created_at ? format(new Date(rule.created_at), 'PPP p') : 'Unknown'}</p>
            </div>
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Last Updated</p>
              <p className="mt-2 text-sm text-slate-900">{rule.updated_at ? format(new Date(rule.updated_at), 'PPP p') : 'Unknown'}</p>
            </div>
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Created By</p>
              <p className="mt-2 text-sm text-slate-900">{creatorLabel}</p>
            </div>
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Last Triggered</p>
              <p className="mt-2 text-sm text-slate-900">{rule.last_triggered_at ? format(new Date(rule.last_triggered_at), 'PPP p') : 'Never'}</p>
            </div>
          </div>
        </div>

        <div className="space-y-6">
          <div className="rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
            <p className="text-sm font-medium">MITRE Mapping</p>
            <div className="mt-4 flex flex-wrap gap-2">
              {rule.mitre_tactic_ids.length > 0 ? (
                rule.mitre_tactic_ids.map((tacticId) => (
                  <Badge key={tacticId} variant="outline" className="font-mono">
                    {tacticId}
                  </Badge>
                ))
              ) : (
                <span className="text-sm text-muted-foreground">No tactics mapped.</span>
              )}
            </div>
            <div className="mt-4 flex flex-wrap gap-2">
              {rule.mitre_technique_ids.length > 0 ? (
                rule.mitre_technique_ids.map((techniqueId) => (
                  <Badge key={techniqueId} variant="secondary" className="font-mono">
                    {techniqueId}
                  </Badge>
                ))
              ) : (
                <span className="text-sm text-muted-foreground">No techniques mapped.</span>
              )}
            </div>
          </div>

          <div className="rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
            <p className="text-sm font-medium">Tags</p>
            <div className="mt-4 flex flex-wrap gap-2">
              {rule.tags.length > 0 ? (
                rule.tags.map((tag) => <Badge key={tag} variant="outline">{tag}</Badge>)
              ) : (
                <span className="text-sm text-muted-foreground">No tags applied.</span>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
