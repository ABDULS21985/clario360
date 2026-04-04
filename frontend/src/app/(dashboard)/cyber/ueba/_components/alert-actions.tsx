'use client';

import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
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
import { apiPut, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { UebaAlert, UebaAlertStatus } from './types';

const STATUS_LABELS: Record<UebaAlertStatus, string> = {
  new: 'New',
  acknowledged: 'Acknowledged',
  investigating: 'Investigating',
  resolved: 'Resolved',
  false_positive: 'False Positive',
};

function statusVariant(status: string) {
  if (status === 'resolved' || status === 'false_positive') return 'secondary' as const;
  if (status === 'investigating') return 'warning' as const;
  if (status === 'acknowledged') return 'default' as const;
  return 'outline' as const;
}

export function AlertActions({ alert }: { alert: UebaAlert }) {
  const queryClient = useQueryClient();
  const [loading, setLoading] = useState(false);
  const [fpDialogOpen, setFpDialogOpen] = useState(false);
  const [fpNotes, setFpNotes] = useState('');

  async function updateStatus(newStatus: UebaAlertStatus, notes?: string) {
    setLoading(true);
    try {
      await apiPut(`${API_ENDPOINTS.CYBER_UEBA_ALERTS}/${alert.id}/status`, {
        status: newStatus,
        notes: notes ?? '',
      });
      toast.success(`Alert status updated to ${STATUS_LABELS[newStatus]}`);
      await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-alerts'] });
      await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-entity-alerts'] });
      await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-dashboard'] });
    } catch {
      toast.error('Failed to update alert status');
    } finally {
      setLoading(false);
    }
  }

  async function markFalsePositive() {
    setLoading(true);
    try {
      await apiPost(`${API_ENDPOINTS.CYBER_UEBA_ALERTS}/${alert.id}/false-positive`, {
        notes: fpNotes,
      });
      toast.success('Alert marked as false positive and profile baseline retrained');
      setFpDialogOpen(false);
      setFpNotes('');
      await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-alerts'] });
      await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-entity-alerts'] });
      await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-dashboard'] });
      await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-profile'] });
    } catch {
      toast.error('Failed to mark alert as false positive');
    } finally {
      setLoading(false);
    }
  }

  const isTerminal = alert.status === 'resolved' || alert.status === 'false_positive';

  const transitions: UebaAlertStatus[] = [];
  if (!isTerminal) {
    if (alert.status === 'new') transitions.push('acknowledged');
    if (alert.status !== 'investigating') transitions.push('investigating');
    transitions.push('resolved');
  }

  return (
    <>
      <div className="flex items-center gap-2">
        <Badge variant={statusVariant(alert.status)}>{STATUS_LABELS[alert.status as UebaAlertStatus] ?? alert.status}</Badge>
        {!isTerminal && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" disabled={loading}>
                Actions
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {transitions.map((status) => (
                <DropdownMenuItem key={status} onClick={() => void updateStatus(status)}>
                  {STATUS_LABELS[status]}
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
        )}
      </div>

      <Dialog open={fpDialogOpen} onOpenChange={setFpDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Mark as False Positive</DialogTitle>
            <DialogDescription>
              This will mark the alert as a false positive and retrain the entity&apos;s behavioral
              baseline to prevent similar alerts in the future.
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
            <Button variant="destructive" onClick={() => void markFalsePositive()} disabled={loading}>
              {loading ? 'Processing...' : 'Confirm False Positive'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
