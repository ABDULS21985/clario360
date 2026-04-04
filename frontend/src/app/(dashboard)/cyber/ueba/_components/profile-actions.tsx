'use client';

import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
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
import type { UebaProfile, UebaProfileStatus, UebaEntityType } from './types';

const STATUS_LABELS: Record<UebaProfileStatus, string> = {
  active: 'Active',
  inactive: 'Inactive',
  suppressed: 'Suppressed',
  whitelisted: 'Whitelisted',
};

export function ProfileActions({ profile }: { profile: UebaProfile }) {
  const queryClient = useQueryClient();
  const [loading, setLoading] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [targetStatus, setTargetStatus] = useState<UebaProfileStatus | null>(null);
  const [reason, setReason] = useState('');

  function openStatusDialog(status: UebaProfileStatus) {
    setTargetStatus(status);
    setReason('');
    setDialogOpen(true);
  }

  async function confirmStatusChange() {
    if (!targetStatus) return;
    setLoading(true);
    try {
      const payload: {
        entity_type: UebaEntityType;
        status: UebaProfileStatus;
        reason?: string;
        suppressed_until?: string;
      } = {
        entity_type: profile.entity_type as UebaEntityType,
        status: targetStatus,
        reason,
      };
      if (targetStatus === 'suppressed') {
        const until = new Date();
        until.setDate(until.getDate() + 30);
        payload.suppressed_until = until.toISOString();
      }
      await apiPut(
        `${API_ENDPOINTS.CYBER_UEBA_PROFILES}/${encodeURIComponent(profile.entity_id)}/status`,
        payload,
      );
      toast.success(`Profile status updated to ${STATUS_LABELS[targetStatus]}`);
      setDialogOpen(false);
      await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-profile'] });
      await queryClient.invalidateQueries({ queryKey: ['cyber-ueba-dashboard'] });
    } catch {
      toast.error('Failed to update profile status');
    } finally {
      setLoading(false);
    }
  }

  const transitions: UebaProfileStatus[] = (['active', 'inactive', 'suppressed', 'whitelisted'] as UebaProfileStatus[]).filter(
    (s) => s !== profile.status,
  );

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" disabled={loading}>
            Change Status
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {transitions.map((status) => (
            <DropdownMenuItem key={status} onClick={() => openStatusDialog(status)}>
              {STATUS_LABELS[status]}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Change Profile Status</DialogTitle>
            <DialogDescription>
              {targetStatus === 'suppressed'
                ? 'Suppressing this profile will stop alerting for 30 days. The profile will continue learning in the background.'
                : targetStatus === 'whitelisted'
                  ? 'Whitelisting this profile will permanently suppress all alerts for this entity until reactivated.'
                  : targetStatus === 'inactive'
                    ? 'Marking inactive stops detection for this entity. Reactivate to resume monitoring.'
                    : 'Reactivating will resume normal detection and alerting for this entity.'}
            </DialogDescription>
          </DialogHeader>
          <Textarea
            placeholder="Reason for status change (optional)"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={3}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => void confirmStatusChange()} disabled={loading}>
              {loading ? 'Updating...' : `Set ${targetStatus ? STATUS_LABELS[targetStatus] : ''}`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
