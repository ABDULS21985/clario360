'use client';

import ReactMarkdown from 'react-markdown';
import { Badge } from '@/components/ui/badge';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { threatStatusConfig } from '@/lib/status-configs';
import { formatDateTime } from '@/lib/utils';
import { getThreatTypeLabel } from '@/lib/cyber-threats';
import type { Threat } from '@/types/cyber';

interface ThreatOverviewProps {
  threat: Threat;
}

export function ThreatOverview({ threat }: ThreatOverviewProps) {
  const daysActive = calculateDaysActive(threat.first_seen_at, threat.contained_at);

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-[2fr,1fr]">
        <section className="rounded-[24px] border bg-card p-5">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline">{getThreatTypeLabel(threat.type)}</Badge>
            <SeverityIndicator severity={threat.severity} showLabel />
            <StatusBadge status={threat.status} config={threatStatusConfig} />
            {threat.threat_actor && <Badge variant="secondary">{threat.threat_actor}</Badge>}
            {threat.campaign && <Badge variant="secondary">{threat.campaign}</Badge>}
          </div>

          <div className="prose prose-sm mt-4 max-w-none text-slate-700">
            <ReactMarkdown>{threat.description || 'No description provided.'}</ReactMarkdown>
          </div>

          {(threat.tags?.length ?? 0) > 0 && (
            <div className="mt-4 flex flex-wrap gap-2">
              {threat.tags.map((tag) => (
                <Badge key={tag} variant="outline" className="text-xs">
                  {tag}
                </Badge>
              ))}
            </div>
          )}
        </section>

        <section className="rounded-[24px] border bg-card p-5">
          <h3 className="text-sm font-semibold">Lifecycle</h3>
          <dl className="mt-4 space-y-3 text-sm">
            <Field label="First Seen" value={formatDateTime(threat.first_seen_at)} />
            <Field label="Last Seen" value={formatDateTime(threat.last_seen_at)} />
            <Field label="Contained At" value={formatDateTime(threat.contained_at)} />
            <Field label="Days Active" value={String(daysActive)} />
          </dl>
        </section>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard label="Indicators" value={String(threat.indicator_count)} />
        <MetricCard label="Affected Assets" value={String(threat.affected_asset_count)} />
        <MetricCard label="Linked Alerts" value={String(threat.alert_count)} />
        <MetricCard label="MITRE Techniques" value={String(threat.mitre_technique_ids.length)} />
      </div>
    </div>
  );
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[22px] border bg-card p-4">
      <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-2 text-3xl font-semibold tracking-[-0.04em] text-slate-950">{value}</p>
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{label}</dt>
      <dd className="mt-1 font-medium text-slate-900">{value}</dd>
    </div>
  );
}

function calculateDaysActive(firstSeenAt: string, containedAt?: string): number {
  const start = new Date(firstSeenAt).getTime();
  const end = containedAt ? new Date(containedAt).getTime() : Date.now();
  return Math.max(1, Math.ceil((end - start) / (1000 * 60 * 60 * 24)));
}
