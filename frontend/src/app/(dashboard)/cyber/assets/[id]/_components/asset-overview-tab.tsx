'use client';

import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { Badge } from '@/components/ui/badge';
import { timeAgo } from '@/lib/utils';
import { TYPE_ICONS, TYPE_LABELS } from '../../_components/asset-columns';
import type { CyberAsset } from '@/types/cyber';

interface AssetOverviewTabProps {
  asset: CyberAsset;
}

function Field({ label, value }: { label: string; value?: React.ReactNode }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-0.5 text-sm font-medium">{value ?? '—'}</p>
    </div>
  );
}

export function AssetOverviewTab({ asset }: AssetOverviewTabProps) {
  const Icon = TYPE_ICONS[asset.type] ?? TYPE_ICONS.server;

  return (
    <div className="space-y-6">
      {/* Identity */}
      <div className="rounded-lg border p-4">
        <h3 className="mb-4 text-sm font-semibold">Identity</h3>
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          <Field label="Type" value={
            <span className="flex items-center gap-1.5">
              <Icon className="h-3.5 w-3.5 text-muted-foreground" />
              {TYPE_LABELS[asset.type]}
            </span>
          } />
          <Field label="Criticality" value={<SeverityIndicator severity={asset.criticality} showLabel />} />
          <Field label="Status" value={<StatusBadge status={asset.status} />} />
          <Field label="Discovery Source" value={asset.discovery_source} />
        </div>
      </div>

      {/* Network */}
      <div className="rounded-lg border p-4">
        <h3 className="mb-4 text-sm font-semibold">Network</h3>
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Field label="IP Address" value={asset.ip_address ? <span className="font-mono">{asset.ip_address}</span> : undefined} />
          <Field label="Hostname" value={asset.hostname ? <span className="font-mono text-xs">{asset.hostname}</span> : undefined} />
          <Field label="MAC Address" value={asset.mac_address ? <span className="font-mono text-xs">{asset.mac_address}</span> : undefined} />
        </div>
      </div>

      {/* System */}
      <div className="rounded-lg border p-4">
        <h3 className="mb-4 text-sm font-semibold">System</h3>
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Field label="Operating System" value={asset.os} />
          <Field label="OS Version" value={asset.os_version} />
          <Field label="Location" value={asset.location} />
        </div>
      </div>

      {/* Ownership */}
      <div className="rounded-lg border p-4">
        <h3 className="mb-4 text-sm font-semibold">Ownership</h3>
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Field label="Owner" value={asset.owner} />
          <Field label="Department" value={asset.department} />
        </div>
      </div>

      {/* Security */}
      <div className="rounded-lg border p-4">
        <h3 className="mb-4 text-sm font-semibold">Security Posture</h3>
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <Field label="Total Vulnerabilities" value={
            <span className={(asset.vulnerability_count ?? 0) > 0 ? 'text-orange-600' : 'text-green-600'}>
              {asset.vulnerability_count ?? 0}
            </span>
          } />
          <Field label="Critical Vulns" value={
            <span className={(asset.critical_vuln_count ?? 0) > 0 ? 'text-red-600 font-semibold' : ''}>
              {asset.critical_vuln_count ?? 0}
            </span>
          } />
          <Field label="High Vulns" value={
            <span className={(asset.high_vuln_count ?? 0) > 0 ? 'text-orange-600' : ''}>
              {asset.high_vuln_count ?? 0}
            </span>
          } />
          <Field label="Open Alerts" value={asset.alert_count ?? 0} />
        </div>
      </div>

      {/* Tags */}
      {(asset.tags?.length ?? 0) > 0 && (
        <div className="rounded-lg border p-4">
          <h3 className="mb-3 text-sm font-semibold">Tags</h3>
          <div className="flex flex-wrap gap-1.5">
            {asset.tags.map((tag) => (
              <Badge key={tag} variant="secondary">{tag}</Badge>
            ))}
          </div>
        </div>
      )}

      {/* Timestamps */}
      <div className="rounded-lg border p-4">
        <h3 className="mb-4 text-sm font-semibold">Timeline</h3>
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Field label="Discovered" value={asset.discovered_at ? timeAgo(asset.discovered_at) : undefined} />
          <Field label="Last Seen" value={asset.last_seen_at ? timeAgo(asset.last_seen_at) : undefined} />
          <Field label="Last Updated" value={timeAgo(asset.updated_at)} />
          <Field label="Created" value={timeAgo(asset.created_at)} />
        </div>
      </div>
    </div>
  );
}
