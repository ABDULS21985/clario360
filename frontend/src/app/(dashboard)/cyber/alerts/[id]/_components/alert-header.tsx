'use client';

import Link from 'next/link';
import { BadgeAlert, Radar, Server } from 'lucide-react';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { StatusBadge } from '@/components/shared/status-badge';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { ROUTES } from '@/lib/constants';
import {
  ALERT_STATUS_CONFIG,
  alertConfidencePercent,
  getAlertStatusVariant,
} from '@/lib/cyber-alerts';
import { formatDateTime } from '@/lib/utils';
import { getAvatarColor, getInitials } from '@/lib/format';
import type { CyberAlert } from '@/types/cyber';

import { AlertActions } from './alert-actions';
import { ConfidenceGauge } from './confidence-gauge';

interface AlertHeaderProps {
  alert: CyberAlert;
  onUpdated: () => void;
}

export function AlertHeader({ alert, onUpdated }: AlertHeaderProps) {
  const assignedName = alert.assigned_to_name ?? '';
  const [firstName, ...rest] = assignedName.split(' ');
  const lastName = rest.join(' ');
  const initials = getInitials(firstName || '?', lastName || '');
  const avatarTone = getAvatarColor(assignedName || alert.assigned_to_email || 'analyst');
  const assetLabel = alert.asset_name ?? alert.asset_hostname ?? alert.asset_ip_address;

  return (
    <div className="rounded-[30px] border border-[color:var(--panel-border)] bg-[linear-gradient(140deg,rgba(255,255,255,0.95),rgba(245,250,247,0.88))] p-6 shadow-[var(--card-shadow)]">
      <div className="flex flex-col gap-6 xl:flex-row xl:items-start xl:justify-between">
        <div className="space-y-5">
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <SeverityIndicator severity={alert.severity} showLabel />
              <StatusBadge
                status={alert.status}
                config={ALERT_STATUS_CONFIG}
                variant={getAlertStatusVariant(alert.status)}
              />
              {alert.mitre_technique_id && (
                <Badge variant="outline" className="font-mono text-[11px]">
                  {alert.mitre_technique_id}
                </Badge>
              )}
              {alert.mitre_tactic_name && (
                <Badge variant="secondary" className="text-[11px]">
                  {alert.mitre_tactic_name}
                </Badge>
              )}
            </div>

            <div className="space-y-2">
              <h1 className="text-3xl font-semibold tracking-[-0.05em] text-slate-950">
                {alert.title}
              </h1>
              <p className="max-w-4xl text-sm leading-7 text-slate-600">
                {alert.description || 'No analyst description was provided for this alert.'}
              </p>
            </div>
          </div>

          <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
            <HeaderFact
              icon={BadgeAlert}
              label="Events Correlated"
              value={String(alert.event_count)}
              caption={`Source: ${alert.source}`}
            />
            <HeaderFact
              icon={Radar}
              label="First Seen"
              value={formatDateTime(alert.first_event_at)}
              caption={`Last seen ${formatDateTime(alert.last_event_at)}`}
            />
            <HeaderFact
              icon={Server}
              label="Affected Asset"
              value={assetLabel ?? 'No linked asset'}
              caption={alert.asset_criticality ? `Criticality: ${alert.asset_criticality}` : 'Asset context unavailable'}
              href={alert.asset_id ? `${ROUTES.CYBER_ASSETS}/${alert.asset_id}` : undefined}
            />
            <div className="rounded-2xl border bg-white/75 p-4">
              <div className="flex items-center gap-3">
                <Avatar className="h-10 w-10">
                  <AvatarFallback className={`${avatarTone} text-sm font-semibold text-white`}>
                    {initials}
                  </AvatarFallback>
                </Avatar>
                <div className="min-w-0">
                  <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                    Assigned Analyst
                  </p>
                  <p className="truncate text-sm font-medium text-slate-900">
                    {alert.assigned_to_name ?? 'Unassigned'}
                  </p>
                  <p className="truncate text-xs text-muted-foreground">
                    {alert.assigned_to_email ?? 'No analyst attached yet'}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="flex w-full max-w-sm flex-col gap-4">
          <ConfidenceGauge score={alertConfidencePercent(alert.confidence_score)} size="lg" />
          <div className="rounded-2xl border bg-white/75 p-4">
            <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
              Analyst Actions
            </p>
            <div className="mt-3">
              <AlertActions alert={alert} onUpdated={onUpdated} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

interface HeaderFactProps {
  icon: typeof BadgeAlert;
  label: string;
  value: string;
  caption: string;
  href?: string;
}

function HeaderFact({ icon: Icon, label, value, caption, href }: HeaderFactProps) {
  const body = (
    <div className="rounded-2xl border bg-white/75 p-4">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-2xl border border-emerald-200 bg-emerald-50 text-emerald-700">
          <Icon className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
            {label}
          </p>
          <p className="truncate text-sm font-medium text-slate-900">{value}</p>
          <p className="truncate text-xs text-muted-foreground">{caption}</p>
        </div>
      </div>
    </div>
  );

  if (!href) {
    return body;
  }

  return (
    <Link href={href} className="block transition-transform hover:-translate-y-0.5">
      {body}
    </Link>
  );
}
