'use client';

import { useForm, FormProvider } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { VCISOApprovalRequest, ApprovalRequestType } from '@/types/cyber';

// ── Schema ────────────────────────────────────────────────────────────────────

const schema = z.object({
  type: z.string().min(1, 'Type is required'),
  title: z.string().min(2, 'Title must be at least 2 characters').max(255),
  description: z.string().optional().default(''),
  approver_id: z.string().uuid('Must be a valid UUID'),
  approver_name: z.string().min(1, 'Approver name is required'),
  priority: z.string().min(1, 'Priority is required'),
  deadline: z.string().min(1, 'Deadline is required'),
  linked_entity_type: z.string().optional().default(''),
  linked_entity_id: z.string().optional().default(''),
});

type FormValues = z.infer<typeof schema>;

// ── Constants ─────────────────────────────────────────────────────────────────

const TYPE_OPTIONS: { label: string; value: ApprovalRequestType }[] = [
  { label: 'Risk Acceptance', value: 'risk_acceptance' },
  { label: 'Policy Exception', value: 'policy_exception' },
  { label: 'Remediation', value: 'remediation' },
  { label: 'Budget', value: 'budget' },
  { label: 'Vendor Onboarding', value: 'vendor_onboarding' },
];

const PRIORITY_OPTIONS = [
  { label: 'Critical', value: 'critical' },
  { label: 'High', value: 'high' },
  { label: 'Medium', value: 'medium' },
  { label: 'Low', value: 'low' },
];

// ── Props ─────────────────────────────────────────────────────────────────────

interface CreateApprovalDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

// ── Component ─────────────────────────────────────────────────────────────────

export function CreateApprovalDialog({
  open,
  onOpenChange,
  onSuccess,
}: CreateApprovalDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      type: '',
      title: '',
      description: '',
      approver_id: '',
      approver_name: '',
      priority: '',
      deadline: '',
      linked_entity_type: '',
      linked_entity_id: '',
    },
  });

  const { register, handleSubmit, setValue, watch, reset } = methods;

  const mutation = useApiMutation<VCISOApprovalRequest, FormValues>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_APPROVALS,
    {
      successMessage: 'Approval request created',
      invalidateKeys: ['vciso-approvals'],
      onSuccess: () => {
        reset();
        onOpenChange(false);
        onSuccess();
      },
    },
  );

  const handleOpenChange = (o: boolean) => {
    if (!o) reset();
    onOpenChange(o);
  };

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  const typeValue = watch('type');
  const priorityValue = watch('priority');

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>New Approval Request</DialogTitle>
          <DialogDescription>
            Submit a new item to the approval queue for governance review.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 py-2">
            {/* Type */}
            <FormField name="type" label="Type" required>
              <Select
                value={typeValue}
                onValueChange={(v) => setValue('type', v, { shouldValidate: true })}
                disabled={mutation.isPending}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select type..." />
                </SelectTrigger>
                <SelectContent>
                  {TYPE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            {/* Title */}
            <FormField name="title" label="Title" required>
              <Input
                {...register('title')}
                id="title"
                placeholder="Brief description of the request"
                disabled={mutation.isPending}
              />
            </FormField>

            {/* Description */}
            <FormField name="description" label="Description">
              <Textarea
                {...register('description')}
                id="description"
                placeholder="Detailed context and justification..."
                rows={3}
                disabled={mutation.isPending}
              />
            </FormField>

            {/* Priority */}
            <FormField name="priority" label="Priority" required>
              <Select
                value={priorityValue}
                onValueChange={(v) => setValue('priority', v, { shouldValidate: true })}
                disabled={mutation.isPending}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select priority..." />
                </SelectTrigger>
                <SelectContent>
                  {PRIORITY_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            {/* Deadline */}
            <FormField name="deadline" label="Deadline" required>
              <Input
                {...register('deadline')}
                id="deadline"
                type="date"
                disabled={mutation.isPending}
              />
            </FormField>

            {/* Approver ID */}
            <FormField name="approver_id" label="Approver ID" required>
              <Input
                {...register('approver_id')}
                id="approver_id"
                placeholder="UUID of the approver"
                disabled={mutation.isPending}
              />
            </FormField>

            {/* Approver Name */}
            <FormField name="approver_name" label="Approver Name" required>
              <Input
                {...register('approver_name')}
                id="approver_name"
                placeholder="Display name of the approver"
                disabled={mutation.isPending}
              />
            </FormField>

            {/* Linked Entity (optional) */}
            <FormField name="linked_entity_type" label="Linked Entity Type">
              <Input
                {...register('linked_entity_type')}
                id="linked_entity_type"
                placeholder="e.g. risk, policy, asset (optional)"
                disabled={mutation.isPending}
              />
            </FormField>

            <FormField name="linked_entity_id" label="Linked Entity ID">
              <Input
                {...register('linked_entity_id')}
                id="linked_entity_id"
                placeholder="UUID of the linked entity (optional)"
                disabled={mutation.isPending}
              />
            </FormField>

            <div className="flex justify-end gap-2 pt-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
                disabled={mutation.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? 'Creating...' : 'Create Request'}
              </Button>
            </div>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
