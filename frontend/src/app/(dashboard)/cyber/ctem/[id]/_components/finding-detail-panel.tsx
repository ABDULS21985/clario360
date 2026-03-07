'use client';

import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { Button } from '@/components/ui/button';
import { X, Zap, ArrowRight } from 'lucide-react';
import { useRouter } from 'next/navigation';
import type { CTEMFinding } from '@/types/cyber';

interface FindingDetailPanelProps {
  finding: CTEMFinding;
  onClose: () => void;
}

export function FindingDetailPanel({ finding, onClose }: FindingDetailPanelProps) {
  const router = useRouter();

  return (
    <div className="rounded-xl border bg-card shadow-md">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div className="flex items-center gap-2">
          <SeverityIndicator severity={finding.severity} showLabel />
          {finding.exploit_available && (
            <span className="flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">
              <Zap className="h-3 w-3" /> Exploit
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

        {finding.cvss_score != null && (
          <div className="flex items-center gap-3 rounded-lg border p-3">
            <div className="text-center">
              <p className="text-2xl font-bold tabular-nums">{finding.cvss_score.toFixed(1)}</p>
              <p className="text-xs text-muted-foreground">CVSS</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Priority Score</p>
              <div className="mt-1 flex items-center gap-2">
                <div className="h-1.5 w-24 rounded-full bg-muted">
                  <div className="h-full rounded-full bg-orange-500" style={{ width: `${finding.priority_score}%` }} />
                </div>
                <span className="text-xs font-bold">{finding.priority_score}</span>
              </div>
            </div>
          </div>
        )}

        {finding.asset_name && (
          <div>
            <p className="mb-1 text-xs font-semibold">Affected Asset</p>
            <button
              className="text-sm text-primary hover:underline"
              onClick={() => finding.asset_id && router.push(`/cyber/assets/${finding.asset_id}`)}
            >
              {finding.asset_name}
              <ArrowRight className="ml-1 inline h-3 w-3" />
            </button>
          </div>
        )}

        {(finding.attack_path?.length ?? 0) > 0 && (
          <div>
            <p className="mb-2 text-xs font-semibold">Attack Path</p>
            <ol className="space-y-1">
              {finding.attack_path!.map((step, i) => (
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

        {(finding.remediation_steps?.length ?? 0) > 0 && (
          <div>
            <p className="mb-2 text-xs font-semibold">Remediation Steps</p>
            <ol className="space-y-1">
              {finding.remediation_steps!.map((step, i) => (
                <li key={i} className="flex items-start gap-2 text-xs text-muted-foreground">
                  <span className="flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                    {i + 1}
                  </span>
                  {step}
                </li>
              ))}
            </ol>
          </div>
        )}
      </div>
    </div>
  );
}
