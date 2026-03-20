'use client';

import { useEffect, useState } from 'react';
import { useAuth } from '@/hooks/use-auth';
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
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';

interface PromoteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  model: AIRegisteredModel | null;
  version: AIModelVersion | null;
  onSaved: () => void;
}

export function PromoteDialog({ open, onOpenChange, model, version, onSaved }: PromoteDialogProps) {
  const { user } = useAuth();
  const [override, setOverride] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) {
      setOverride(false);
    }
  }, [open]);

  const submit = async () => {
    if (!model || !version) {
      return;
    }
    try {
      setSaving(true);
      await enterpriseApi.ai.promote(model.id, version.id, {
        approved_by: user?.id,
        override,
      });
      showSuccess('Model version promoted.', `${model.slug} v${version.version_number} moved forward in the lifecycle.`);
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
          <DialogTitle>Promote Model Version</DialogTitle>
          <DialogDescription>
            Promotion enforces lifecycle gates, shadow recommendations, and critical-tier approval requirements.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="rounded-lg bg-muted/30 p-4 text-sm">
            <div className="font-medium">{model?.name}</div>
            <div className="mt-1 text-muted-foreground">
              {model?.slug} • version {version?.version_number} • currently {version?.status}
            </div>
          </div>
          <div className="flex items-center justify-between rounded-lg border border-border/70 p-4">
            <div>
              <Label htmlFor="promotion-override">Manual override</Label>
              <p className="text-sm text-muted-foreground">
                Use when a shadow recommendation is `keep_shadow` and an approved promotion is still required.
              </p>
            </div>
            <Switch id="promotion-override" checked={override} onCheckedChange={setOverride} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={() => void submit()} disabled={saving}>
            {saving ? 'Promoting…' : 'Promote'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
