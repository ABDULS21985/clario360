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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { VCISOAwarenessProgram, AwarenessProgramType } from '@/types/cyber';

interface AwarenessFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
  program?: VCISOAwarenessProgram | null;
}

const TYPE_OPTIONS: { label: string; value: AwarenessProgramType }[] = [
  { label: 'Training', value: 'training' },
  { label: 'Phishing Simulation', value: 'phishing_simulation' },
  { label: 'Policy Attestation', value: 'policy_attestation' },
];

interface FormState {
  name: string;
  type: AwarenessProgramType;
  total_users: string;
  start_date: string;
  end_date: string;
}

const initialFormState: FormState = {
  name: '',
  type: 'training',
  total_users: '',
  start_date: '',
  end_date: '',
};

export function AwarenessFormDialog({
  open,
  onOpenChange,
  onCreated,
  program,
}: AwarenessFormDialogProps) {
  const isEdit = !!program;

  const [form, setForm] = useState<FormState>(() =>
    program
      ? {
          name: program.name,
          type: program.type,
          total_users: String(program.total_users),
          start_date: program.start_date ? program.start_date.split('T')[0] : '',
          end_date: program.end_date ? program.end_date.split('T')[0] : '',
        }
      : initialFormState,
  );

  const createMutation = useApiMutation<VCISOAwarenessProgram, Record<string, unknown>>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_AWARENESS,
    {
      successMessage: 'Program created successfully',
      invalidateKeys: ['vciso-awareness'],
      onSuccess: () => {
        setForm(initialFormState);
        onOpenChange(false);
        onCreated();
      },
    },
  );

  const updateMutation = useApiMutation<VCISOAwarenessProgram, Record<string, unknown>>(
    'put',
    program ? `${API_ENDPOINTS.CYBER_VCISO_AWARENESS}/${program.id}` : '',
    {
      successMessage: 'Program updated successfully',
      invalidateKeys: ['vciso-awareness'],
      onSuccess: () => {
        onOpenChange(false);
        onCreated();
      },
    },
  );

  const handleSubmit = () => {
    if (!form.name.trim()) {
      toast.error('Name is required');
      return;
    }
    if (!form.total_users || parseInt(form.total_users, 10) <= 0) {
      toast.error('Total users must be a positive number');
      return;
    }
    if (!form.start_date) {
      toast.error('Start date is required');
      return;
    }
    if (!form.end_date) {
      toast.error('End date is required');
      return;
    }

    const payload: Record<string, unknown> = {
      name: form.name.trim(),
      type: form.type,
      total_users: parseInt(form.total_users, 10),
      start_date: form.start_date,
      end_date: form.end_date,
    };

    if (isEdit) {
      updateMutation.mutate(payload);
    } else {
      createMutation.mutate(payload);
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  const handleOpenChange = (o: boolean) => {
    if (!o && !isEdit) {
      setForm(initialFormState);
    }
    onOpenChange(o);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit Program' : 'Create Awareness Program'}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Update the awareness program details.'
              : 'Set up a new security awareness program for your organization.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="program-name">
              Name <span className="text-destructive">*</span>
            </Label>
            <Input
              id="program-name"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="e.g. Q1 2026 Security Training"
            />
          </div>

          <div className="space-y-2">
            <Label>
              Type <span className="text-destructive">*</span>
            </Label>
            <Select
              value={form.type}
              onValueChange={(v) => setForm((f) => ({ ...f, type: v as AwarenessProgramType }))}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {TYPE_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="program-total-users">
              Total Users <span className="text-destructive">*</span>
            </Label>
            <Input
              id="program-total-users"
              type="number"
              min={1}
              value={form.total_users}
              onChange={(e) => setForm((f) => ({ ...f, total_users: e.target.value }))}
              placeholder="e.g. 250"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="program-start-date">
                Start Date <span className="text-destructive">*</span>
              </Label>
              <Input
                id="program-start-date"
                type="date"
                value={form.start_date}
                onChange={(e) => setForm((f) => ({ ...f, start_date: e.target.value }))}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="program-end-date">
                End Date <span className="text-destructive">*</span>
              </Label>
              <Input
                id="program-end-date"
                type="date"
                value={form.end_date}
                onChange={(e) => setForm((f) => ({ ...f, end_date: e.target.value }))}
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-4">
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isPending}>
            {isPending ? (isEdit ? 'Saving...' : 'Creating...') : (isEdit ? 'Save Changes' : 'Create Program')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
