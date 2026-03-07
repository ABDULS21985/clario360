'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ExternalLink, Shield, AlertTriangle, Plus, FileText, ToggleLeft } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { DetailPanel } from '@/components/shared/detail-panel';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { timeAgo } from '@/lib/utils';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { toast } from 'sonner';
import type { MITRETechniqueCoverage, CyberAlert, DetectionRule } from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';

interface MitreTechniquePanelProps {
  technique: MITRETechniqueCoverage | null;
  onClose: () => void;
  onCreateRule?: (techniqueId: string) => void;
}

function CoverageStatusBadge({ state }: { state: string }) {
  if (state === 'active') return <Badge className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300">Active</Badge>;
  if (state === 'passive') return <Badge className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300">Passive</Badge>;
  return <Badge variant="destructive">Gap</Badge>;
}

export function MitreTechniquePanel({ technique, onClose, onCreateRule }: MitreTechniquePanelProps) {
  const [togglingRuleId, setTogglingRuleId] = useState<string | null>(null);

  const coverageState =
    technique && technique.rule_count > 0 && technique.alert_count > 0
      ? 'active'
      : technique && technique.rule_count > 0
      ? 'passive'
      : 'gap';

  // Fetch technique detail (description, platforms)
  const { data: detailEnvelope } = useQuery({
    queryKey: [`mitre-technique-${technique?.technique_id}`],
    queryFn: () =>
      apiGet<{ data: MITRETechniqueCoverage }>(
        `${API_ENDPOINTS.CYBER_MITRE_TECHNIQUES}/${technique!.technique_id}`,
      ),
    enabled: !!technique,
    staleTime: 300000,
  });

  // Fetch detection rules for this technique
  const { data: rulesEnvelope, refetch: refetchRules } = useQuery({
    queryKey: [`mitre-rules-${technique?.technique_id}`],
    queryFn: () =>
      apiGet<PaginatedResponse<DetectionRule>>(API_ENDPOINTS.CYBER_RULES, {
        mitre_technique_id: technique!.technique_id,
        per_page: 20,
      }),
    enabled: !!technique,
  });

  // Fetch recent alerts
  const { data: alertsEnvelope } = useQuery({
    queryKey: [`mitre-alerts-${technique?.technique_id}`],
    queryFn: () =>
      apiGet<PaginatedResponse<CyberAlert>>(API_ENDPOINTS.CYBER_ALERTS, {
        mitre_technique_id: technique!.technique_id,
        per_page: 10,
      }),
    enabled: !!technique,
  });

  // Fetch template if any
  const { data: templateEnvelope } = useQuery({
    queryKey: [`mitre-template-${technique?.technique_id}`],
    queryFn: () =>
      apiGet<{ data: { template_id?: string; has_template: boolean } }>(
        `${API_ENDPOINTS.CYBER_RULE_TEMPLATES}/check`,
        { mitre_technique_id: technique!.technique_id },
      ),
    enabled: !!technique && coverageState === 'gap',
  });

  const detail = detailEnvelope?.data ?? technique;
  const rules = rulesEnvelope?.data ?? [];
  const alerts = alertsEnvelope?.data ?? [];
  const hasTemplate = templateEnvelope?.data?.has_template ?? false;

  const handleToggleRule = async (rule: DetectionRule) => {
    setTogglingRuleId(rule.id);
    try {
      await apiPost(`${API_ENDPOINTS.CYBER_RULES}/${rule.id}/toggle`);
      toast.success(`Rule ${rule.enabled ? 'disabled' : 'enabled'}`);
      void refetchRules();
    } catch {
      toast.error('Failed to toggle rule');
    } finally {
      setTogglingRuleId(null);
    }
  };

  return (
    <DetailPanel
      open={!!technique}
      onOpenChange={(o) => { if (!o) onClose(); }}
      title={technique?.technique_name ?? ''}
      width="lg"
    >
      {technique && (
        <div className="space-y-5">
          {/* Header badges */}
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline" className="font-mono">{technique.technique_id}</Badge>
            <CoverageStatusBadge state={coverageState} />
            <Badge variant="secondary">{technique.tactic_name}</Badge>
          </div>

          {/* Description */}
          <section>
            <h4 className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Description
            </h4>
            {detail?.description ? (
              <p className="text-sm text-foreground">{detail.description}</p>
            ) : (
              <p className="text-sm text-muted-foreground">
                Technique {technique.technique_id} from the MITRE ATT&CK framework.
              </p>
            )}
            {detail?.platforms && detail.platforms.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1">
                {detail.platforms.map((p) => (
                  <Badge key={p} variant="secondary" className="text-xs">{p}</Badge>
                ))}
              </div>
            )}
            <a
              href={`https://attack.mitre.org/techniques/${technique.technique_id}/`}
              target="_blank"
              rel="noopener noreferrer"
              className="mt-2 inline-flex items-center gap-1 text-xs text-primary hover:underline"
            >
              View on ATT&CK
              <ExternalLink className="h-3 w-3" />
            </a>
          </section>

          {/* Detection Rules */}
          <section>
            <div className="mb-2 flex items-center justify-between">
              <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Detection Rules
              </h4>
              {rules.length > 0 && (
                <a
                  href={`/cyber/rules?mitre_technique_id=${technique.technique_id}`}
                  className="text-xs text-primary hover:underline"
                >
                  View all →
                </a>
              )}
            </div>
            {rules.length === 0 ? (
              <div className="rounded-lg border bg-muted/30 p-3 text-center">
                <p className="text-sm text-muted-foreground">No detection rules for this technique.</p>
                <Button
                  size="sm"
                  variant="outline"
                  className="mt-2"
                  onClick={() => onCreateRule?.(technique.technique_id)}
                >
                  <Plus className="mr-1.5 h-3.5 w-3.5" />
                  Create Detection Rule
                </Button>
              </div>
            ) : (
              <div className="space-y-2">
                {rules.map((rule) => (
                  <div key={rule.id} className="flex items-center justify-between rounded-lg border p-2.5">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="truncate text-sm font-medium">{rule.name}</span>
                        <Badge
                          variant="secondary"
                          className="shrink-0 text-[10px] capitalize"
                        >
                          {rule.type}
                        </Badge>
                        <SeverityIndicator severity={rule.severity} />
                      </div>
                    </div>
                    <Switch
                      checked={rule.enabled}
                      disabled={togglingRuleId === rule.id}
                      onCheckedChange={() => void handleToggleRule(rule)}
                      aria-label={`Toggle ${rule.name}`}
                    />
                  </div>
                ))}
              </div>
            )}
          </section>

          {/* Recent Alerts */}
          <section>
            <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Recent Alerts
            </h4>
            {alerts.length === 0 ? (
              <p className="text-sm text-muted-foreground">No alerts from this technique.</p>
            ) : (
              <div className="space-y-1.5">
                {alerts.map((alert) => (
                  <a
                    key={alert.id}
                    href={`/cyber/alerts/${alert.id}`}
                    className="flex items-start gap-2 rounded-lg border p-2 text-sm transition-colors hover:bg-muted/30"
                  >
                    <SeverityIndicator severity={alert.severity} />
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-medium">{alert.title}</p>
                      <p className="text-xs text-muted-foreground">
                        {Math.round(alert.confidence_score * 100)}% confidence · {timeAgo(alert.created_at)}
                      </p>
                    </div>
                  </a>
                ))}
              </div>
            )}
          </section>

          {/* Coverage Recommendation */}
          {coverageState === 'gap' && (
            <section className="rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20">
              <div className="flex items-center gap-2 text-red-800 dark:text-red-400">
                <AlertTriangle className="h-4 w-4 shrink-0" />
                <p className="text-sm font-semibold">Detection Gap</p>
              </div>
              <p className="mt-1 text-xs text-red-700 dark:text-red-500">
                This technique is a detection gap. No detection rules are configured for{' '}
                {technique.technique_name}.
              </p>
              {hasTemplate && (
                <p className="mt-2 text-xs text-red-700 dark:text-red-500">
                  A pre-built template is available.{' '}
                  <Button
                    size="sm"
                    variant="outline"
                    className="ml-1 h-5 border-red-300 px-2 text-[10px] text-red-700 hover:bg-red-100 dark:text-red-400"
                    onClick={() => onCreateRule?.(technique.technique_id)}
                  >
                    Use Template
                  </Button>
                </p>
              )}
            </section>
          )}
          {coverageState === 'passive' && (
            <section className="rounded-lg border border-yellow-200 bg-yellow-50 p-3 dark:border-yellow-800 dark:bg-yellow-950/20">
              <div className="flex items-center gap-2 text-yellow-800 dark:text-yellow-400">
                <ToggleLeft className="h-4 w-4 shrink-0" />
                <p className="text-sm font-semibold">Rules Active — No Alerts</p>
              </div>
              <p className="mt-1 text-xs text-yellow-700 dark:text-yellow-500">
                Rules are active but have not triggered.
              </p>
              <ul className="mt-1 list-inside list-decimal text-xs text-yellow-700 dark:text-yellow-500">
                <li>Good — no attack attempts using this technique.</li>
                <li>Review — rule conditions may be too strict or data sources missing.</li>
              </ul>
            </section>
          )}

          {/* Coverage data */}
          <section>
            <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Coverage Details
            </h4>
            <div className="grid grid-cols-2 gap-2">
              <div className="flex items-center justify-between rounded-lg border p-2.5">
                <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
                  <Shield className="h-3.5 w-3.5" /> Rules
                </span>
                <span className="font-semibold tabular-nums">{technique.rule_count}</span>
              </div>
              <div className="flex items-center justify-between rounded-lg border p-2.5">
                <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
                  <FileText className="h-3.5 w-3.5" /> Alerts
                </span>
                <span className={`font-semibold tabular-nums ${technique.alert_count > 0 ? 'text-orange-600' : 'text-green-600'}`}>
                  {technique.alert_count}
                </span>
              </div>
            </div>
            {technique.last_alert && (
              <p className="mt-2 text-xs text-muted-foreground">
                Last alert: {timeAgo(technique.last_alert)}
              </p>
            )}
          </section>
        </div>
      )}
    </DetailPanel>
  );
}
