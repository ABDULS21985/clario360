'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { VCISOControlOwnership } from '@/types/cyber';

interface OwnershipFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  ownership?: VCISOControlOwnership;
}

interface FormState {
  control_id: string;
  control_name: string;
  framework: string;
  owner_id: string;
  owner_name: string;
  delegate_id: string;
  delegate_name: string;
  next_review_date: string;
}

const initialFormState: FormState = {
  control_id: '',
  control_name: '',
  framework: '',
  owner_id: '',
  owner_name: '',
  delegate_id: '',
  delegate_name: '',
  next_review_date: '',
};

function formStateFromOwnership(ownership: VCISOControlOwnership): FormState {
  return {
    control_id: ownership.control_id,
    control_name: ownership.control_name,
    framework: ownership.framework,
    owner_id: ownership.owner_id,
    owner_name: ownership.owner_name,
    delegate_id: ownership.delegate_id ?? '',
    delegate_name: ownership.delegate_name ?? '',
    next_review_date: ownership.next_review_date
      ? ownership.next_review_date.split('T')[0]
      : '',
  };
}

export function OwnershipFormDialog({
  open,
  onOpenChange,
  onSuccess,
  ownership,
}: OwnershipFormDialogProps) {
  const isEdit = !!ownership;
  const [form, setForm] = useState<FormState>(
    ownership ? formStateFromOwnership(ownership) : initialFormState,
  );

  const createMutation = useApiMutation<VCISOControlOwnership, Record<string, unknown>>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_CONTROL_OWNERSHIP,
    {
      successMessage: 'Ownership assigned successfully',
      invalidateKeys: ['vciso-control-ownership'],
      onSuccess: () => {
        setForm(initialFormState);
        onOpenChange(false);
        onSuccess();
      },
    },
  );

  const updateMutation = useApiMutation<VCISOControlOwnership, Record<string, unknown>>(
    'put',
    `${API_ENDPOINTS.CYBER_VCISO_CONTROL_OWNERSHIP}/${ownership?.id ?? ''}`,
    {
      successMessage: 'Ownership updated successfully',
      invalidateKeys: ['vciso-control-ownership'],
      onSuccess: () => {
        onOpenChange(false);
        onSuccess();
      },
    },
  );

  const mutation = isEdit ? updateMutation : createMutation;

  const handleSubmit = () => {
    if (!form.control_id.trim()) {
      toast.error('Control ID is required');
      return;
    }
    if (!form.control_name.trim()) {
      toast.error('Control Name is required');
      return;
    }
    if (!form.framework.trim()) {
      toast.error('Framework is required');
      return;
    }
    if (!form.owner_id.trim()) {
      toast.error('Owner ID is required');
      return;
    }
    if (!form.owner_name.trim()) {
      toast.error('Owner Name is required');
      return;
    }
    if (!form.next_review_date) {
      toast.error('Next Review Date is required');
      return;
    }

    const payload: Record<string, unknown> = {
      control_id: form.control_id.trim(),
      control_name: form.control_name.trim(),
      framework: form.framework.trim(),
      owner_id: form.owner_id.trim(),
      owner_name: form.owner_name.trim(),
      status: isEdit ? (ownership?.status ?? 'assigned') : 'assigned',
      next_review_date: form.next_review_date,
    };

    if (form.delegate_id.trim()) {
      payload.delegate_id = form.delegate_id.trim();
      payload.delegate_name = form.delegate_name.trim();
    }

    mutation.mutate(payload);
  };

  const handleOpenChange = (o: boolean) => {
    if (!o && !isEdit) {
      setForm(initialFormState);
    }
    onOpenChange(o);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Reassign Ownership' : 'Assign Control Ownership'}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Update the owner, delegate, or review schedule for this control.'
              : 'Assign an owner to a security control. Required fields are marked.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Control Info */}
          <h4 className="text-sm font-semibold text-muted-foreground">Control Information</h4>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="ownership-control-id">
                Control ID <span className="text-destructive">*</span>
              </Label>
              <Input
                id="ownership-control-id"
                value={form.control_id}
                onChange={(e) => setForm((f) => ({ ...f, control_id: e.target.value }))}
                placeholder="e.g. AC-1"
                disabled={isEdit}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="ownership-framework">
                Framework <span className="text-destructive">*</span>
              </Label>
              <Input
                id="ownership-framework"
                value={form.framework}
                onChange={(e) => setForm((f) => ({ ...f, framework: e.target.value }))}
                placeholder="e.g. NIST 800-53"
                disabled={isEdit}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="ownership-control-name">
              Control Name <span className="text-destructive">*</span>
            </Label>
            <Input
              id="ownership-control-name"
              value={form.control_name}
              onChange={(e) => setForm((f) => ({ ...f, control_name: e.target.value }))}
              placeholder="e.g. Access Control Policy and Procedures"
              disabled={isEdit}
            />
          </div>

          <Separator />

          {/* Owner Info */}
          <h4 className="text-sm font-semibold text-muted-foreground">Owner Assignment</h4>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="ownership-owner-id">
                Owner ID <span className="text-destructive">*</span>
              </Label>
              <Input
                id="ownership-owner-id"
                value={form.owner_id}
                onChange={(e) => setForm((f) => ({ ...f, owner_id: e.target.value }))}
                placeholder="User ID of the owner"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="ownership-owner-name">
                Owner Name <span className="text-destructive">*</span>
              </Label>
              <Input
                id="ownership-owner-name"
                value={form.owner_name}
                onChange={(e) => setForm((f) => ({ ...f, owner_name: e.target.value }))}
                placeholder="e.g. John Smith"
              />
            </div>
          </div>

          <Separator />

          {/* Delegate Info */}
          <h4 className="text-sm font-semibold text-muted-foreground">Delegate (Optional)</h4>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="ownership-delegate-id">Delegate ID</Label>
              <Input
                id="ownership-delegate-id"
                value={form.delegate_id}
                onChange={(e) => setForm((f) => ({ ...f, delegate_id: e.target.value }))}
                placeholder="User ID of the delegate"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="ownership-delegate-name">Delegate Name</Label>
              <Input
                id="ownership-delegate-name"
                value={form.delegate_name}
                onChange={(e) => setForm((f) => ({ ...f, delegate_name: e.target.value }))}
                placeholder="e.g. Jane Doe"
              />
            </div>
          </div>

          <Separator />

          {/* Schedule */}
          <div className="space-y-2">
            <Label htmlFor="ownership-review-date">
              Next Review Date <span className="text-destructive">*</span>
            </Label>
            <Input
              id="ownership-review-date"
              type="date"
              value={form.next_review_date}
              onChange={(e) => setForm((f) => ({ ...f, next_review_date: e.target.value }))}
            />
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-4">
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={mutation.isPending}>
            {mutation.isPending
              ? isEdit
                ? 'Updating...'
                : 'Assigning...'
              : isEdit
                ? 'Update Ownership'
                : 'Assign Ownership'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
