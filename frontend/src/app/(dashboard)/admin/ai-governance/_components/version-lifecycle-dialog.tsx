'use client';

import { useEffect, useMemo, useState } from 'react';
import { showApiError, showSuccess } from '@/lib/toast';
import { enterpriseApi } from '@/lib/enterprise';
import type { AIModelVersion, AIRegisteredModel } from '@/types/ai-governance';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';

export type VersionLifecycleAction = 'retire' | 'fail' | 'stop_shadow';

interface VersionLifecycleDialogProps {
  action: VersionLifecycleAction | null;
  model: AIRegisteredModel | null;
  version: AIModelVersion | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}

export function VersionLifecycleDialog({
  action,
  model,
  version,
  open,
  onOpenChange,
  onSaved,
}: VersionLifecycleDialogProps) {
  const [reason, setReason] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) {
      setReason('');
    }
  }, [open]);

  const copy = useMemo(() => getActionCopy(action), [action]);

  const submit = async () => {
    if (!action || !model || !version || !reason.trim()) {
      return;
    }

    setSaving(true);
    try {
      switch (action) {
        case 'retire':
          await enterpriseApi.ai.retire(model.id, version.id, { reason: reason.trim() });
          showSuccess('Version retired.', `${model.slug} v${version.version_number} was retired.`);
          break;
        case 'fail':
          await enterpriseApi.ai.failVersion(model.id, version.id, { reason: reason.trim() });
          showSuccess('Version marked failed.', `${model.slug} v${version.version_number} was moved to failed state.`);
          break;
        case 'stop_shadow':
          await enterpriseApi.ai.stopShadow(model.id, { version_id: version.id, reason: reason.trim() });
          showSuccess('Shadow mode stopped.', `${model.slug} v${version.version_number} returned to staging.`);
          break;
      }

      onOpenChange(false);
      onSaved();
    } catch (error) {
      showApiError(error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{copy.title}</DialogTitle>
          <DialogDescription>{copy.description}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-lg bg-muted/30 p-4 text-sm">
            <div className="font-medium">{model?.name}</div>
            <div className="mt-1 text-muted-foreground">
              {model?.slug} • version {version?.version_number} • currently {version?.status}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="ai-lifecycle-reason">{copy.reasonLabel}</Label>
            <Textarea
              id="ai-lifecycle-reason"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder={copy.placeholder}
              className="min-h-28"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant={action === 'stop_shadow' ? 'default' : 'destructive'} onClick={() => void submit()} disabled={saving || !reason.trim()}>
            {saving ? copy.pendingLabel : copy.confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function getActionCopy(action: VersionLifecycleAction | null) {
  switch (action) {
    case 'retire':
      return {
        title: 'Retire Model Version',
        description: 'Retirement keeps the audit trail intact while removing this version from active lifecycle use.',
        reasonLabel: 'Retirement reason',
        placeholder: 'Explain why this version is being retired.',
        confirmLabel: 'Retire Version',
        pendingLabel: 'Retiring…',
      };
    case 'fail':
      return {
        title: 'Mark Version as Failed',
        description: 'Use this when validation, shadow, or operational review found a blocking issue in the candidate version.',
        reasonLabel: 'Failure reason',
        placeholder: 'Describe the validation failure, regression, or governance blocker.',
        confirmLabel: 'Mark Failed',
        pendingLabel: 'Saving failure…',
      };
    case 'stop_shadow':
      return {
        title: 'Stop Shadow Mode',
        description: 'Stopping shadow mode returns the version to staging without promoting it into production.',
        reasonLabel: 'Stop reason',
        placeholder: 'Explain why shadow traffic should stop for this version.',
        confirmLabel: 'Stop Shadow',
        pendingLabel: 'Stopping shadow…',
      };
    default:
      return {
        title: 'Lifecycle Action',
        description: 'Record the lifecycle transition and keep the backend audit trail aligned with the frontend workflow.',
        reasonLabel: 'Reason',
        placeholder: 'Enter a reason.',
        confirmLabel: 'Save',
        pendingLabel: 'Saving…',
      };
  }
}
