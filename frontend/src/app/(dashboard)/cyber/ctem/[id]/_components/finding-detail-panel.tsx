'use client';

import { useState } from 'react';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { X, Zap, Pencil } from 'lucide-react';
import { FindingStatusDialog } from './finding-status-dialog';
import type { CTEMFinding } from '@/types/cyber';

interface FindingDetailPanelProps {
  finding: CTEMFinding;
  onClose: () => void;
  onStatusUpdated?: () => void;
  assetNames?: Record<string, string>;
}

/** Derive "exploit available" from exploitability_score (backend has no explicit field) */
function isExploitAvailable(finding: CTEMFinding): boolean {
  return finding.exploitability_score >= 0.7;
}

export function FindingDetailPanel({ finding, onClose, onStatusUpdated, assetNames = {} }: FindingDetailPanelProps) {
  const [statusDialogOpen, setStatusDialogOpen] = useState(false);

  return (
    <>
    <FindingStatusDialog
      finding={finding}
      open={statusDialogOpen}
      onOpenChange={setStatusDialogOpen}
      onSuccess={() => onStatusUpdated?.()}
    />
    <div className="rounded-xl border bg-card shadow-md">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div className="flex items-center gap-2">
          <SeverityIndicator severity={finding.severity} showLabel />
          {isExploitAvailable(finding) && (
            <span className="flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700 dark:bg-red-950/30 dark:text-red-400">
              <Zap className="h-3 w-3" /> High Exploit Risk
            </span>
          )}
        </div>
        <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <div className="space-y-4 p-4">
        <div>
          <h4 className="font-semibold">{finding.title}</h4>
          <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{finding.description}</p>
        </div>

        {/* Type & Category badges */}
        <div className="flex flex-wrap gap-2">
          <Badge variant="outline" className="text-xs capitalize">{finding.type.replace(/_/g, ' ')}</Badge>
          <Badge variant="secondary" className="text-xs capitalize">{finding.category}</Badge>
          {finding.validation_status !== 'pending' && (
            <Badge variant="outline" className="text-xs capitalize">{finding.validation_status.replace(/_/g, ' ')}</Badge>
          )}
        </div>

        {/* Scores — use real backend fields: exploitability_score, business_impact_score, priority_score */}
        <div className="flex items-center gap-3 rounded-lg border p-3">
          <div className="text-center">
            <p className="text-2xl font-bold tabular-nums">{finding.exploitability_score.toFixed(1)}</p>
            <p className="text-xs text-muted-foreground">Exploitability</p>
          </div>
          <div className="text-center">
            <p className="text-2xl font-bold tabular-nums">{finding.business_impact_score.toFixed(1)}</p>
            <p className="text-xs text-muted-foreground">Impact</p>
          </div>
          <div className="flex-1">
            <p className="text-xs text-muted-foreground">Priority Score</p>
            <div className="mt-1 flex items-center gap-2">
              <div className="h-1.5 w-full rounded-full bg-muted">
                <div className="h-full rounded-full bg-orange-500" style={{ width: `${finding.priority_score}%` }} />
              </div>
              <span className="text-xs font-bold tabular-nums">{Math.round(finding.priority_score)}</span>
            </div>
          </div>
        </div>

        {/* Affected Assets — resolve names from asset IDs when available */}
        {(finding.affected_asset_count > 0 || finding.primary_asset_id) && (
          <div>
            <p className="mb-1 text-xs font-semibold">Affected Assets</p>
            {finding.primary_asset_id && assetNames[finding.primary_asset_id] ? (
              <div className="space-y-1">
                <p className="text-sm font-medium">{assetNames[finding.primary_asset_id]}</p>
                {finding.affected_asset_count > 1 && (
                  <p className="text-xs text-muted-foreground">
                    +{finding.affected_asset_count - 1} other asset{finding.affected_asset_count > 2 ? 's' : ''}
                  </p>
                )}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                {finding.affected_asset_count} asset{finding.affected_asset_count > 1 ? 's' : ''} affected
              </p>
            )}
          </div>
        )}

        {/* CVEs */}
        {(finding.cve_ids?.length ?? 0) > 0 && (
          <div>
            <p className="mb-1 text-xs font-semibold">CVEs</p>
            <div className="flex flex-wrap gap-1">
              {finding.cve_ids!.map((cve) => (
                <Badge key={cve} variant="outline" className="text-xs">{cve}</Badge>
              ))}
            </div>
          </div>
        )}

        {/* Attack Path */}
        {Array.isArray(finding.attack_path) && finding.attack_path.length > 0 && (
          <div>
            <p className="mb-2 text-xs font-semibold">Attack Path</p>
            <ol className="space-y-1">
              {(finding.attack_path as string[]).map((step, i) => (
                <li key={i} className="flex items-start gap-2 text-xs">
                  <span className="flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-bold">
                    {i + 1}
                  </span>
                  {step}
                </li>
              ))}
            </ol>
          </div>
        )}

        {/* Remediation — use real backend field: remediation_description (not legacy remediation_steps) */}
        {finding.remediation_description && (
          <div>
            <p className="mb-2 text-xs font-semibold">Remediation</p>
            <p className="text-sm text-muted-foreground">{finding.remediation_description}</p>
            {finding.remediation_type && (
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <Badge variant="outline" className="text-xs capitalize">{finding.remediation_type.replace(/_/g, ' ')}</Badge>
                {finding.remediation_effort && (
                  <Badge variant="secondary" className="text-xs capitalize">{finding.remediation_effort} effort</Badge>
                )}
                {finding.estimated_days != null && (
                  <span className="text-xs text-muted-foreground">~{finding.estimated_days}d</span>
                )}
              </div>
            )}
          </div>
        )}

        {/* Status info */}
        <div className="rounded-lg border bg-muted/20 p-3">
          <div className="flex items-center justify-between">
            <div>
              <span className="text-xs text-muted-foreground">Status</span>
              <p className="text-sm font-medium capitalize">{finding.status.replace(/_/g, ' ')}</p>
            </div>
            <Button variant="ghost" size="sm" className="h-7 gap-1 text-xs" onClick={() => setStatusDialogOpen(true)}>
              <Pencil className="h-3 w-3" />
              Update
            </Button>
          </div>
          {finding.status_notes && (
            <p className="mt-1 text-xs text-muted-foreground">{finding.status_notes}</p>
          )}
        </div>
      </div>
    </div>
    </>
  );
}
