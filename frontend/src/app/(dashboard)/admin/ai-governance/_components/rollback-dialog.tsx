'use client';

import { useState } from 'react';
import { showApiError, showSuccess } from '@/lib/toast';
import { enterpriseApi } from '@/lib/enterprise';
import type { AIRegisteredModel } from '@/types/ai-governance';
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

interface RollbackDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  model: AIRegisteredModel | null;
  onSaved: () => void;
}

export function RollbackDialog({ open, onOpenChange, model, onSaved }: RollbackDialogProps) {
  const [reason, setReason] = useState('');
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    if (!model || !reason.trim()) {
      return;
    }
    try {
      setSaving(true);
      await enterpriseApi.ai.rollback(model.id, { reason });
      showSuccess('Model rolled back.', `${model.slug} reverted to the previous production version.`);
      setReason('');
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
          <DialogTitle>Rollback Production Version</DialogTitle>
          <DialogDescription>
            Record the operational or governance reason for restoring the previous production version.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="rollback-reason">Rollback reason</Label>
          <Textarea
            id="rollback-reason"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder="Explain the regression, drift issue, or governance exception."
            className="min-h-28"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={() => void submit()} disabled={saving || !reason.trim()}>
            {saving ? 'Rolling back…' : 'Rollback'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
