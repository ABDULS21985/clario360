'use client';

import { CheckCircle, Download, ExternalLink, FileText, Calendar, User, Shield } from 'lucide-react';
import { DetailPanel } from '@/components/shared/detail-panel';
import { StatusBadge } from '@/components/shared/status-badge';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { evidenceStatusConfig } from '@/lib/status-configs';
import { formatDate, formatBytes, titleCase } from '@/lib/format';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { VCISOEvidence } from '@/types/cyber';

interface EvidenceDetailPanelProps {
  evidence: VCISOEvidence | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onVerified?: () => void;
}

export function EvidenceDetailPanel({
  evidence,
  open,
  onOpenChange,
  onVerified,
}: EvidenceDetailPanelProps) {
  const { mutate: verify, isPending: verifying } = useApiMutation<unknown, unknown>(
    'put',
    (variables) => `${API_ENDPOINTS.CYBER_VCISO_EVIDENCE}/${(variables as { id: string }).id}/verify`,
    {
      successMessage: 'Evidence marked as verified',
      invalidateKeys: ['vciso-evidence', 'vciso-evidence-stats'],
      onSuccess: () => {
        onVerified?.();
      },
    },
  );

  if (!evidence) return null;

  return (
    <DetailPanel
      open={open}
      onOpenChange={onOpenChange}
      title={evidence.title}
      description="Evidence details and metadata"
      width="lg"
    >
      <div className="space-y-6">
        {/* Status and Type */}
        <div className="flex flex-wrap items-center gap-2">
          <StatusBadge status={evidence.status} config={evidenceStatusConfig} />
          <Badge variant="outline" className="capitalize">
            {titleCase(evidence.type)}
          </Badge>
          <Badge
            variant="secondary"
            className={
              evidence.source === 'automated'
                ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300'
                : ''
            }
          >
            {titleCase(evidence.source)}
          </Badge>
        </div>

        {/* Description */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2">
            Description
          </h4>
          <p className="text-sm leading-relaxed text-foreground">{evidence.description}</p>
        </div>

        <Separator />

        {/* Frameworks */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2">
            <Shield className="inline h-3.5 w-3.5 mr-1 -mt-0.5" />
            Frameworks ({evidence.frameworks.length})
          </h4>
          <div className="flex flex-wrap gap-1.5">
            {evidence.frameworks.length > 0 ? (
              evidence.frameworks.map((fw) => (
                <Badge key={fw} variant="outline" className="text-xs">
                  {fw}
                </Badge>
              ))
            ) : (
              <span className="text-sm text-muted-foreground">No frameworks assigned</span>
            )}
          </div>
        </div>

        {/* Control IDs */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2">
            Control IDs ({evidence.control_ids.length})
          </h4>
          <div className="flex flex-wrap gap-1.5">
            {evidence.control_ids.length > 0 ? (
              evidence.control_ids.map((cid) => (
                <Badge key={cid} variant="secondary" className="text-xs font-mono">
                  {cid}
                </Badge>
              ))
            ) : (
              <span className="text-sm text-muted-foreground">No controls linked</span>
            )}
          </div>
        </div>

        <Separator />

        {/* File Info */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2">
            <FileText className="inline h-3.5 w-3.5 mr-1 -mt-0.5" />
            File Information
          </h4>
          {evidence.file_name ? (
            <div className="space-y-2 rounded-lg border p-3">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium truncate mr-2">{evidence.file_name}</span>
                {evidence.file_size != null && (
                  <span className="text-xs text-muted-foreground shrink-0">
                    {formatBytes(evidence.file_size)}
                  </span>
                )}
              </div>
              {evidence.file_url && (
                <Button variant="outline" size="sm" className="w-full" asChild>
                  <a href={evidence.file_url} target="_blank" rel="noopener noreferrer">
                    <Download className="mr-1.5 h-3.5 w-3.5" />
                    Download File
                    <ExternalLink className="ml-1.5 h-3 w-3" />
                  </a>
                </Button>
              )}
            </div>
          ) : (
            <span className="text-sm text-muted-foreground">No file attached</span>
          )}
        </div>

        <Separator />

        {/* Dates */}
        <div className="grid grid-cols-2 gap-3">
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Calendar className="h-3 w-3" />
              Collected
            </p>
            <p className="text-sm font-medium mt-0.5">{formatDate(evidence.collected_at)}</p>
          </div>
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Calendar className="h-3 w-3" />
              Expires
            </p>
            <p className="text-sm font-medium mt-0.5">
              {evidence.expires_at ? formatDate(evidence.expires_at) : 'No expiry'}
            </p>
          </div>
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Calendar className="h-3 w-3" />
              Created
            </p>
            <p className="text-sm font-medium mt-0.5">{formatDate(evidence.created_at)}</p>
          </div>
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Calendar className="h-3 w-3" />
              Updated
            </p>
            <p className="text-sm font-medium mt-0.5">{formatDate(evidence.updated_at)}</p>
          </div>
        </div>

        {/* Verification Info */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2">
            Verification
          </h4>
          <div className="rounded-lg border p-3 space-y-2">
            {evidence.collector_name && (
              <div className="flex items-center gap-2 text-sm">
                <User className="h-3.5 w-3.5 text-muted-foreground" />
                <span className="text-muted-foreground">Collector:</span>
                <span className="font-medium">{evidence.collector_name}</span>
              </div>
            )}
            <div className="flex items-center gap-2 text-sm">
              <CheckCircle className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-muted-foreground">Last Verified:</span>
              <span className="font-medium">
                {evidence.last_verified_at ? formatDate(evidence.last_verified_at) : 'Never'}
              </span>
            </div>
          </div>
        </div>

        {/* Action Button */}
        <Button
          className="w-full"
          onClick={() => verify({ id: evidence.id })}
          disabled={verifying}
        >
          <CheckCircle className="mr-1.5 h-4 w-4" />
          {verifying ? 'Verifying...' : 'Mark as Verified'}
        </Button>
      </div>
    </DetailPanel>
  );
}
