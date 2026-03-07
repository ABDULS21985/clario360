'use client';

import Link from 'next/link';
import { StatusBadge } from '@/components/shared/status-badge';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { Badge } from '@/components/ui/badge';
import { ExternalLink, Shield, User, Clock, Crosshair } from 'lucide-react';
import { timeAgo } from '@/lib/utils';
import type { CyberAlert } from '@/types/cyber';

interface AlertContextPanelProps {
  alert: CyberAlert;
}

function Row({ icon: Icon, label, value }: { icon: React.ElementType; label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-start gap-3 py-2.5">
      <Icon className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
      <div className="flex-1 min-w-0">
        <p className="text-xs text-muted-foreground">{label}</p>
        <div className="mt-0.5 text-sm">{value}</div>
      </div>
    </div>
  );
}

export function AlertContextPanel({ alert }: AlertContextPanelProps) {
  return (
    <div className="space-y-4">
      <div className="rounded-xl border divide-y">
        <Row
          icon={Shield}
          label="Severity"
          value={<SeverityIndicator severity={alert.severity} showLabel />}
        />
        <Row
          icon={Shield}
          label="Status"
          value={<StatusBadge status={alert.status} />}
        />
        {alert.asset_name && (
          <Row
            icon={ExternalLink}
            label="Affected Asset"
            value={
              <Link href={`/cyber/assets/${alert.asset_id}`} className="hover:underline font-medium">
                {alert.asset_name}
              </Link>
            }
          />
        )}
        {alert.assigned_to_name && (
          <Row icon={User} label="Assigned To" value={alert.assigned_to_name} />
        )}
        {alert.escalated_to && (
          <Row icon={User} label="Escalated To" value={alert.escalated_to} />
        )}
        <Row
          icon={Clock}
          label="First Seen"
          value={<span title={alert.first_event_at}>{timeAgo(alert.first_event_at)}</span>}
        />
        <Row
          icon={Clock}
          label="Last Seen"
          value={<span title={alert.last_event_at}>{timeAgo(alert.last_event_at)}</span>}
        />
        <Row icon={Clock} label="Events" value={alert.event_count.toLocaleString()} />
        {alert.source && (
          <Row icon={ExternalLink} label="Source" value={alert.source} />
        )}
      </div>

      {/* MITRE */}
      {(alert.mitre_tactic_id || alert.mitre_technique_id) && (
        <div className="rounded-xl border p-3">
          <div className="mb-2 flex items-center gap-2">
            <Crosshair className="h-4 w-4 text-muted-foreground" />
            <span className="text-xs font-semibold">MITRE ATT&CK</span>
          </div>
          {alert.mitre_tactic_name && (
            <div className="mb-1 text-xs text-muted-foreground">
              Tactic: <span className="font-medium text-foreground">{alert.mitre_tactic_name}</span>
              {alert.mitre_tactic_id && <span className="ml-1 font-mono opacity-60">({alert.mitre_tactic_id})</span>}
            </div>
          )}
          {alert.mitre_technique_name && (
            <div className="text-xs text-muted-foreground">
              Technique: <span className="font-medium text-foreground">{alert.mitre_technique_name}</span>
              {alert.mitre_technique_id && <span className="ml-1 font-mono opacity-60">({alert.mitre_technique_id})</span>}
            </div>
          )}
        </div>
      )}

      {/* Tags */}
      {alert.tags?.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {alert.tags.map((tag) => (
            <Badge key={tag} variant="secondary" className="text-xs">{tag}</Badge>
          ))}
        </div>
      )}
    </div>
  );
}
