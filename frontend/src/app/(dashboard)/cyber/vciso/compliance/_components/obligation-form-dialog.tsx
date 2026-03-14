'use client';

import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { VCISORegulatoryObligation, ObligationType } from '@/types/cyber';

const OBLIGATION_TYPES: { label: string; value: ObligationType }[] = [
  { label: 'Legal', value: 'legal' },
  { label: 'Regulatory', value: 'regulatory' },
  { label: 'Contractual', value: 'contractual' },
  { label: 'Industry Standard', value: 'industry_standard' },
];

interface ObligationFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  obligation?: VCISORegulatoryObligation | null;
  onSuccess: () => void;
}

export function ObligationFormDialog({
  open,
  onOpenChange,
  obligation,
  onSuccess,
}: ObligationFormDialogProps) {
  const isEditing = !!obligation;

  const [name, setName] = useState('');
  const [type, setType] = useState<ObligationType | ''>('');
  const [jurisdiction, setJurisdiction] = useState('');
  const [description, setDescription] = useState('');
  const [effectiveDate, setEffectiveDate] = useState('');
  const [reviewDate, setReviewDate] = useState('');

  useEffect(() => {
    if (open) {
      if (obligation) {
        setName(obligation.name);
        setType(obligation.type);
        setJurisdiction(obligation.jurisdiction);
        setDescription(obligation.description);
        setEffectiveDate(obligation.effective_date?.slice(0, 10) ?? '');
        setReviewDate(obligation.review_date?.slice(0, 10) ?? '');
      } else {
        setName('');
        setType('');
        setJurisdiction('');
        setDescription('');
        setEffectiveDate('');
        setReviewDate('');
      }
    }
  }, [open, obligation]);

  const createMutation = useApiMutation<VCISORegulatoryObligation, Record<string, unknown>>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_OBLIGATIONS,
    {
      invalidateKeys: ['vciso-obligations'],
      successMessage: 'Obligation created successfully',
      onSuccess: () => {
        onOpenChange(false);
        onSuccess();
      },
    },
  );

  const updateMutation = useApiMutation<VCISORegulatoryObligation, Record<string, unknown>>(
    'put',
    () => `${API_ENDPOINTS.CYBER_VCISO_OBLIGATIONS}/${obligation?.id}`,
    {
      invalidateKeys: ['vciso-obligations'],
      successMessage: 'Obligation updated successfully',
      onSuccess: () => {
        onOpenChange(false);
        onSuccess();
      },
    },
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) {
      toast.error('Name is required');
      return;
    }
    if (!type) {
      toast.error('Type is required');
      return;
    }
    if (!jurisdiction.trim()) {
      toast.error('Jurisdiction is required');
      return;
    }
    if (!description.trim()) {
      toast.error('Description is required');
      return;
    }

    const payload = {
      name: name.trim(),
      type,
      jurisdiction: jurisdiction.trim(),
      description: description.trim(),
      effective_date: effectiveDate || undefined,
      review_date: reviewDate || undefined,
    };

    if (isEditing) {
      updateMutation.mutate(payload);
    } else {
      createMutation.mutate(payload);
    }
  };

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? 'Edit Obligation' : 'Add Obligation'}
          </DialogTitle>
          <DialogDescription>
            {isEditing
              ? 'Update the regulatory obligation details below.'
              : 'Add a new regulatory obligation to your compliance library.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div className="space-y-2">
            <Label htmlFor="obligation-name">Name</Label>
            <Input
              id="obligation-name"
              placeholder="e.g., GDPR Data Processing Requirements"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={isSubmitting}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="obligation-type">Type</Label>
              <Select
                value={type}
                onValueChange={(v) => setType(v as ObligationType)}
                disabled={isSubmitting}
              >
                <SelectTrigger id="obligation-type">
                  <SelectValue placeholder="Select type" />
                </SelectTrigger>
                <SelectContent>
                  {OBLIGATION_TYPES.map((t) => (
                    <SelectItem key={t.value} value={t.value}>
                      {t.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="obligation-jurisdiction">Jurisdiction</Label>
              <Input
                id="obligation-jurisdiction"
                placeholder="e.g., European Union"
                value={jurisdiction}
                onChange={(e) => setJurisdiction(e.target.value)}
                disabled={isSubmitting}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="obligation-description">Description</Label>
            <Textarea
              id="obligation-description"
              placeholder="Describe the regulatory obligation..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={isSubmitting}
              className="min-h-[120px]"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="obligation-effective-date">Effective Date</Label>
              <Input
                id="obligation-effective-date"
                type="date"
                value={effectiveDate}
                onChange={(e) => setEffectiveDate(e.target.value)}
                disabled={isSubmitting}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="obligation-review-date">Review Date</Label>
              <Input
                id="obligation-review-date"
                type="date"
                value={reviewDate}
                onChange={(e) => setReviewDate(e.target.value)}
                disabled={isSubmitting}
              />
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting
                ? isEditing
                  ? 'Updating...'
                  : 'Creating...'
                : isEditing
                  ? 'Update Obligation'
                  : 'Add Obligation'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
