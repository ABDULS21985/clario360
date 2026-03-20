'use client';

import { useForm, FormProvider } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { CheckCircle, XCircle } from 'lucide-react';
import type { RemediationAction } from '@/types/cyber';

const schema = z.object({ notes: z.string().optional() });
type FormValues = z.infer<typeof schema>;

interface RemediationApproveDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  action: RemediationAction;
  mode: 'approve' | 'reject';
  onSuccess?: () => void;
}

export function RemediationApproveDialog({ open, onOpenChange, action, mode, onSuccess }: RemediationApproveDialogProps) {
  const methods = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { notes: '' } });
  const isApprove = mode === 'approve';

  const { mutate, isPending } = useApiMutation<RemediationAction, FormValues>(
    'post',
    `${API_ENDPOINTS.CYBER_REMEDIATION}/${action.id}/${isApprove ? 'approve' : 'reject'}`,
    {
      successMessage: isApprove ? 'Action approved' : 'Action rejected',
      invalidateKeys: ['cyber-remediation', `cyber-remediation-${action.id}`],
      onSuccess: () => {
        methods.reset();
        onOpenChange(false);
        onSuccess?.();
      },
    },
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className={`flex items-center gap-2 ${isApprove ? 'text-green-600' : 'text-destructive'}`}>
            {isApprove ? <CheckCircle className="h-5 w-5" /> : <XCircle className="h-5 w-5" />}
            {isApprove ? 'Approve' : 'Reject'} Remediation
          </DialogTitle>
          <DialogDescription>
            {isApprove ? 'Approve' : 'Reject'}: <strong>{action.title}</strong>
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit((d) => mutate(d))} className="space-y-4">
            <FormField name="notes" label={isApprove ? 'Approval Notes' : 'Rejection Reason'}>
              <Textarea rows={3} placeholder={isApprove ? 'Any conditions or notes…' : 'Why is this being rejected?'} {...methods.register('notes')} />
            </FormField>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
              <Button
                type="submit"
                disabled={isPending}
                variant={isApprove ? 'default' : 'destructive'}
              >
                {isPending ? (isApprove ? 'Approving…' : 'Rejecting…') : (isApprove ? 'Approve' : 'Reject')}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
