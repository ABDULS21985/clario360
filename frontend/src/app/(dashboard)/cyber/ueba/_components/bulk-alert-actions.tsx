'use client';

import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { UebaAlertStatus } from './types';

interface BulkAlertActionsProps {
  selectedIds: string[];
  onComplete: () => void;
}

interface BulkResponse {
  updated: number;
  failed: number;
  errors?: string[];
}

const BULK_STATUSES: { value: UebaAlertStatus; label: string }[] = [
  { value: 'acknowledged', label: 'Acknowledge' },
  { value: 'investigating', label: 'Investigate' },
  { value: 'resolved', label: 'Resolve' },
];

export function BulkAlertActions({ selectedIds, onComplete }: BulkAlertActionsProps) {
  const queryClient = useQueryClient();
  const [loading, setLoading] = useState(false);
  const [fpDialogOpen, setFpDialogOpen] = useState(false);
  const [fpNotes, setFpNotes] = useState('');

  async function invalidateAll() {
    await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-alerts'] });
    await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-entity-alerts'] });
    await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-dashboard'] });
    await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-profile'] });
  }

  async function bulkUpdate(status: UebaAlertStatus) {
    setLoading(true);
    try {
      const res = await apiPut<BulkResponse>(API_ENDPOINTS.CYBER_UEBA_ALERTS_BULK_STATUS, {
        alert_ids: selectedIds,
        status,
      });
      if (res.failed > 0) {
        toast.warning(`${res.updated} updated, ${res.failed} failed`);
      } else {
        toast.success(`${res.updated} alert${res.updated !== 1 ? 's' : ''} updated`);
      }
      await invalidateAll();
      onComplete();
    } catch {
      toast.error('Bulk update failed');
    } finally {
      setLoading(false);
    }
  }

  async function bulkFalsePositive() {
    setLoading(true);
    try {
      const res = await apiPut<BulkResponse>(API_ENDPOINTS.CYBER_UEBA_ALERTS_BULK_STATUS, {
        alert_ids: selectedIds,
        false_positive: true,
        notes: fpNotes,
      });
      if (res.failed > 0) {
        toast.warning(`${res.updated} marked false positive, ${res.failed} failed`);
      } else {
        toast.success(`${res.updated} alert${res.updated !== 1 ? 's' : ''} marked as false positive`);
      }
      setFpDialogOpen(false);
      setFpNotes('');
      await invalidateAll();
      onComplete();
    } catch {
      toast.error('Bulk false positive failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <div className="flex items-center gap-2 rounded-md border bg-muted/50 px-3 py-2">
        <span className="text-sm font-medium">
          {selectedIds.length} selected
        </span>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" disabled={loading}>
              {loading ? 'Updating...' : 'Bulk Actions'}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            {BULK_STATUSES.map(({ value, label }) => (
              <DropdownMenuItem key={value} onClick={() => void bulkUpdate(value)}>
                {label}
              </DropdownMenuItem>
            ))}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-destructive"
              onClick={() => setFpDialogOpen(true)}
            >
              Mark as False Positive
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <Dialog open={fpDialogOpen} onOpenChange={setFpDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Bulk Mark as False Positive</DialogTitle>
            <DialogDescription>
              This will mark {selectedIds.length} alert{selectedIds.length !== 1 ? 's' : ''} as
              false positive and retrain affected entity baselines.
            </DialogDescription>
          </DialogHeader>
          <Textarea
            placeholder="Reason for marking as false positive (optional)"
            value={fpNotes}
            onChange={(e) => setFpNotes(e.target.value)}
            rows={3}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setFpDialogOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={() => void bulkFalsePositive()} disabled={loading}>
              {loading ? 'Processing...' : `Confirm (${selectedIds.length})`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
