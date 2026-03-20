'use client';

import { useState } from 'react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import type { VisusExecutiveAlert } from '@/types/suites';

interface DismissAlertDialogProps {
  alert: VisusExecutiveAlert | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (id: string, dismissReason?: string) => void;
}

export function DismissAlertDialog({ alert, open, onOpenChange, onConfirm }: DismissAlertDialogProps) {
  const [reason, setReason] = useState('');

  const handleConfirm = () => {
    if (!alert) return;
    onConfirm(alert.id, reason.trim() || undefined);
    setReason('');
  };

  const handleOpenChange = (value: boolean) => {
    if (!value) setReason('');
    onOpenChange(value);
  };

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Dismiss Alert</AlertDialogTitle>
          <AlertDialogDescription>
            {alert ? `Dismiss "${alert.title}"? This will remove it from the active alerts view.` : 'Dismiss this alert?'}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="space-y-2 py-2">
          <Label htmlFor="dismiss-reason">Reason (optional)</Label>
          <Textarea
            id="dismiss-reason"
            placeholder="Why is this alert being dismissed?"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={3}
          />
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={handleConfirm}>Dismiss</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
