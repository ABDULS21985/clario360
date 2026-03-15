'use client';

import type { ReactNode } from 'react';
import { FileJson, Globe, Shield, TerminalSquare } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { formatDateTime } from '@/lib/utils';
import type { AlertEvidence as AlertEvidenceItem, CyberAlert } from '@/types/cyber';

interface AlertEvidenceProps {
  alert: CyberAlert;
}

export function AlertEvidence({ alert }: AlertEvidenceProps) {
  const explanation = alert.explanation;
  const details = explanation.details ?? {};
  const metadata = alert.metadata ?? {};
  const evidence = explanation.evidence ?? [];
  const networkEvidence = evidence.filter((item) => (
    item.field === 'source_ip' || item.field === 'dest_ip' || item.field === 'dest_port'
  ));

  return (
    <div className="space-y-6">
      <Section icon={Shield} title="Structured Evidence">
        {evidence.length > 0 ? (
          <div className="overflow-hidden rounded-2xl border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-left text-xs uppercase tracking-[0.18em] text-muted-foreground">
                <tr>
                  <th className="px-4 py-3">Label</th>
                  <th className="px-4 py-3">Value</th>
                  <th className="px-4 py-3">Description</th>
                </tr>
              </thead>
              <tbody>
                {evidence.map((item, index) => (
                  <tr key={`${item.field}-${index}`} className="border-t">
                    <td className="px-4 py-3 font-medium">{item.label}</td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-900">{formatEvidenceValue(item)}</td>
                    <td className="px-4 py-3 text-muted-foreground">{item.description}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyMessage message="No structured evidence was attached to this alert." />
        )}
      </Section>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <Section icon={Globe} title="Network Context">
          {networkEvidence.length > 0 ? (
            <div className="space-y-3">
              {networkEvidence.map((item, index) => (
                <div key={`${item.field}-${index}`} className="rounded-2xl border bg-background px-4 py-3">
                  <div className="flex items-center justify-between gap-3">
                    <p className="text-sm font-medium text-slate-900">{item.label}</p>
                    <Badge variant="outline">{item.field}</Badge>
                  </div>
                  <p className="mt-2 font-mono text-xs text-slate-700">{formatEvidenceValue(item)}</p>
                </div>
              ))}
            </div>
          ) : (
            <EmptyMessage message="No explicit network pivots were recorded." />
          )}
        </Section>

        <Section icon={TerminalSquare} title="Asset Context">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <ContextItem label="Asset" value={alert.asset_name ?? 'Unknown'} />
            <ContextItem label="Hostname" value={alert.asset_hostname ?? 'Unknown'} />
            <ContextItem label="IP Address" value={alert.asset_ip_address ?? 'Unknown'} />
            <ContextItem label="Operating System" value={alert.asset_os ?? 'Unknown'} />
            <ContextItem label="Owner" value={alert.asset_owner ?? 'Unknown'} />
            <ContextItem label="Criticality" value={alert.asset_criticality ?? 'Unknown'} />
            <ContextItem label="First Event" value={formatDateTime(alert.first_event_at)} />
            <ContextItem label="Last Event" value={formatDateTime(alert.last_event_at)} />
          </div>
        </Section>
      </div>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <Section icon={Shield} title="Indicator Matches">
          {(explanation.indicator_matches?.length ?? 0) > 0 ? (
            <div className="space-y-3">
              {explanation.indicator_matches?.map((match, index) => (
                <div key={`${match.value}-${index}`} className="rounded-2xl border bg-background px-4 py-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant="outline">{match.type}</Badge>
                    <Badge variant="secondary">{match.source}</Badge>
                    <Badge variant="secondary">{Math.round(match.confidence * 100)}%</Badge>
                  </div>
                  <p className="mt-3 break-all font-mono text-xs text-slate-900">{match.value}</p>
                </div>
              ))}
            </div>
          ) : (
            <EmptyMessage message="No threat intelligence matches were attached." />
          )}
        </Section>

        <Section icon={FileJson} title="Detection Payload">
          <div className="space-y-4">
            <JsonBlock title="Explanation Details" value={details} />
            <JsonBlock title="Alert Metadata" value={metadata} />
          </div>
        </Section>
      </div>
    </div>
  );
}

function Section({
  icon: Icon,
  title,
  children,
}: {
  icon: typeof Shield;
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="rounded-[26px] border bg-card p-5 shadow-sm">
      <div className="mb-4 flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-2xl border border-slate-200 bg-slate-50 text-slate-700">
          <Icon className="h-4 w-4" />
        </div>
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            Investigation
          </p>
          <h2 className="text-lg font-semibold tracking-[-0.03em] text-slate-950">{title}</h2>
        </div>
      </div>
      {children}
    </section>
  );
}

function ContextItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border bg-background px-4 py-3">
      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-2 text-sm text-slate-900">{value}</p>
    </div>
  );
}

function JsonBlock({ title, value }: { title: string; value: unknown }) {
  return (
    <div className="rounded-2xl border bg-slate-950 p-4 text-slate-100">
      <p className="mb-3 text-xs font-semibold uppercase tracking-[0.18em] text-slate-300">{title}</p>
      <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs leading-6">
        {JSON.stringify(value ?? {}, null, 2)}
      </pre>
    </div>
  );
}

function formatEvidenceValue(item: AlertEvidenceItem): string {
  if (typeof item.value === 'string') {
    return item.value;
  }
  return JSON.stringify(item.value) ?? 'null';
}

function EmptyMessage({ message }: { message: string }) {
  return <p className="text-sm text-muted-foreground">{message}</p>;
}
