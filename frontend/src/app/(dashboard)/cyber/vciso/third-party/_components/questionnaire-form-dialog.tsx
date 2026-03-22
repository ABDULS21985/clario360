'use client';

import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
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
import type { VCISOQuestionnaire, QuestionnaireType } from '@/types/cyber';

const QUESTIONNAIRE_TYPE_OPTIONS: { label: string; value: QuestionnaireType }[] = [
  { label: 'Vendor Assessment', value: 'vendor' },
  { label: 'Customer Due Diligence', value: 'customer' },
  { label: 'Audit Questionnaire', value: 'audit' },
  { label: 'Internal Review', value: 'internal' },
];

interface QuestionnaireFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  questionnaire?: VCISOQuestionnaire | null;
  onSuccess: () => void;
  defaultVendorId?: string;
}

export function QuestionnaireFormDialog({
  open,
  onOpenChange,
  questionnaire,
  onSuccess,
  defaultVendorId,
}: QuestionnaireFormDialogProps) {
  const isEditing = !!questionnaire;

  const [title, setTitle] = useState('');
  const [type, setType] = useState<QuestionnaireType>('vendor');
  const [vendorId, setVendorId] = useState('');
  const [vendorName, setVendorName] = useState('');
  const [totalQuestions, setTotalQuestions] = useState('');
  const [dueDate, setDueDate] = useState('');
  const [assignedTo, setAssignedTo] = useState('');
  const [assignedToName, setAssignedToName] = useState('');

  useEffect(() => {
    if (open) {
      if (questionnaire) {
        setTitle(questionnaire.title);
        setType(questionnaire.type);
        setVendorId(questionnaire.vendor_id ?? '');
        setVendorName(questionnaire.vendor_name ?? '');
        setTotalQuestions(String(questionnaire.total_questions));
        setDueDate(questionnaire.due_date ? questionnaire.due_date.split('T')[0] : '');
        setAssignedTo(questionnaire.assigned_to ?? '');
        setAssignedToName(questionnaire.assigned_to_name ?? '');
      } else {
        setTitle('');
        setType('vendor');
        setVendorId(defaultVendorId ?? '');
        setVendorName('');
        setTotalQuestions('');
        setDueDate('');
        setAssignedTo('');
        setAssignedToName('');
      }
    }
  }, [open, questionnaire, defaultVendorId]);

  const createMutation = useApiMutation<VCISOQuestionnaire, Record<string, unknown>>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_QUESTIONNAIRES,
    {
      invalidateKeys: ['vciso-questionnaires'],
      successMessage: 'Questionnaire created successfully',
      onSuccess: () => {
        onOpenChange(false);
        onSuccess();
      },
    },
  );

  const updateMutation = useApiMutation<VCISOQuestionnaire, Record<string, unknown>>(
    'put',
    () => `${API_ENDPOINTS.CYBER_VCISO_QUESTIONNAIRES}/${questionnaire?.id}`,
    {
      invalidateKeys: ['vciso-questionnaires'],
      successMessage: 'Questionnaire updated successfully',
      onSuccess: () => {
        onOpenChange(false);
        onSuccess();
      },
    },
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!title.trim()) {
      toast.error('Title is required');
      return;
    }
    if (!totalQuestions || parseInt(totalQuestions, 10) < 1) {
      toast.error('Total questions must be at least 1');
      return;
    }
    if (!dueDate) {
      toast.error('Due date is required');
      return;
    }

    const payload: Record<string, unknown> = {
      title: title.trim(),
      type,
      status: isEditing ? (questionnaire?.status ?? 'draft') : 'draft',
      total_questions: parseInt(totalQuestions, 10),
      due_date: dueDate,
      vendor_id: vendorId.trim() || undefined,
      vendor_name: vendorName.trim() || undefined,
      assigned_to: assignedTo.trim() || undefined,
      assigned_to_name: assignedToName.trim() || undefined,
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
      <DialogContent className="sm:max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? 'Edit Questionnaire' : 'Create Questionnaire'}
          </DialogTitle>
          <DialogDescription>
            {isEditing
              ? 'Update the questionnaire details below.'
              : 'Create a new security assessment questionnaire.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div className="space-y-2">
            <Label htmlFor="q-title">Title</Label>
            <Input
              id="q-title"
              placeholder="e.g., SOC 2 Vendor Assessment"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={isSubmitting}
            />
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="q-type">Type</Label>
              <Select
                value={type}
                onValueChange={(v) => setType(v as QuestionnaireType)}
                disabled={isSubmitting}
              >
                <SelectTrigger id="q-type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {QUESTIONNAIRE_TYPE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="q-questions">Total Questions</Label>
              <Input
                id="q-questions"
                type="number"
                min={1}
                placeholder="e.g., 50"
                value={totalQuestions}
                onChange={(e) => setTotalQuestions(e.target.value)}
                disabled={isSubmitting}
              />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="q-vendor-id">Vendor ID (optional)</Label>
              <Input
                id="q-vendor-id"
                placeholder="UUID of associated vendor"
                value={vendorId}
                onChange={(e) => setVendorId(e.target.value)}
                disabled={isSubmitting}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="q-vendor-name">Vendor Name (optional)</Label>
              <Input
                id="q-vendor-name"
                placeholder="e.g., Acme Corp"
                value={vendorName}
                onChange={(e) => setVendorName(e.target.value)}
                disabled={isSubmitting}
              />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="q-due-date">Due Date</Label>
              <Input
                id="q-due-date"
                type="date"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
                disabled={isSubmitting}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="q-assigned-to">Assigned To (optional)</Label>
              <Input
                id="q-assigned-to"
                placeholder="User UUID"
                value={assignedTo}
                onChange={(e) => setAssignedTo(e.target.value)}
                disabled={isSubmitting}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="q-assigned-to-name">Assignee Name (optional)</Label>
            <Input
              id="q-assigned-to-name"
              placeholder="e.g., Jane Smith"
              value={assignedToName}
              onChange={(e) => setAssignedToName(e.target.value)}
              disabled={isSubmitting}
            />
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
                  ? 'Update Questionnaire'
                  : 'Create Questionnaire'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
