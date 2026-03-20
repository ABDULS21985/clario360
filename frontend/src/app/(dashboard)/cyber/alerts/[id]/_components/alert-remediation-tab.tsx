'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { CheckCircle, ArrowRight, ListChecks } from 'lucide-react';
import type { AlertExplanation } from '@/types/cyber';

interface AlertRemediationTabProps {
  alertId: string;
  explanation: AlertExplanation;
}

export function AlertRemediationTab({ alertId, explanation }: AlertRemediationTabProps) {
  const router = useRouter();
  const actions = explanation.recommended_actions ?? [];

  return (
    <div className="space-y-6">
      {/* Quick actions */}
      {actions.length > 0 && (
        <div>
          <h4 className="mb-3 text-sm font-semibold">Recommended Actions</h4>
          <ol className="space-y-2">
            {actions.map((action, i) => (
              <li key={i} className="flex items-start gap-3 rounded-xl border bg-card p-3.5">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                  {i + 1}
                </div>
                <div className="flex-1">
                  <p className="text-sm">{action}</p>
                </div>
                <CheckCircle className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground opacity-30" />
              </li>
            ))}
          </ol>
        </div>
      )}

      {/* Create remediation task */}
      <div className="rounded-xl border border-primary/30 bg-primary/5 p-5">
        <div className="flex items-start gap-3">
          <ListChecks className="mt-0.5 h-5 w-5 text-primary" />
          <div className="flex-1">
            <p className="font-semibold text-sm">Create Remediation Action</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Formalize this alert's remediation with an auditable workflow — approval gates, dry-run testing, rollback support.
            </p>
          </div>
        </div>
        <div className="mt-4 flex gap-2">
          <Button size="sm" onClick={() => router.push(`/cyber/remediation?source_alert=${alertId}`)}>
            Create Remediation
            <ArrowRight className="ml-1.5 h-3.5 w-3.5" />
          </Button>
          <Button size="sm" variant="outline" onClick={() => router.push('/cyber/remediation')}>
            View All Remediations
          </Button>
        </div>
      </div>
    </div>
  );
}
