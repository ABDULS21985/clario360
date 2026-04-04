'use client';

import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { showSuccess, showError } from '@/lib/toast';
import type { CTEMFinding } from '@/types/cyber';

const STATUS_OPTIONS: { value: CTEMFinding['status']; label: string }[] = [
  { value: 'open', label: 'Open' },
  { value: 'in_remediation', label: 'In Remediation' },
  { value: 'remediated', label: 'Remediated' },
  { value: 'accepted_risk', label: 'Accepted Risk' },
  { value: 'false_positive', label: 'False Positive' },
  { value: 'deferred', label: 'Deferred' },
];

interface FindingStatusDialogProps {
  finding: CTEMFinding;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function FindingStatusDialog({ finding, open, onOpenChange, onSuccess }: FindingStatusDialogProps) {
  const [status, setStatus] = useState<CTEMFinding['status']>(finding.status);
  const [notes, setNotes] = useState(finding.status_notes ?? '');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    setSubmitting(true);
    try {
      await apiPut(API_ENDPOINTS.CYBER_CTEM_FINDING_STATUS(finding.id), {
        status,
        notes: notes.trim() || undefined,
      });
      showSuccess('Finding status updated');
      onSuccess();
      onOpenChange(false);
    } catch {
      showError('Failed to update finding status');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Update Finding Status</DialogTitle>
          <DialogDescription className="line-clamp-2">{finding.title}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="finding-status">Status</Label>
            <Select value={status} onValueChange={(v) => setStatus(v as CTEMFinding['status'])}>
              <SelectTrigger id="finding-status">
                <SelectValue placeholder="Select status" />
              </SelectTrigger>
              <SelectContent>
                {STATUS_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="finding-notes">Notes (optional)</Label>
            <Textarea
              id="finding-notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Reason for status change..."
              rows={3}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={submitting || status === finding.status}>
            {submitting ? 'Updating...' : 'Update Status'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
