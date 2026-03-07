'use client';

import { useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import { CheckCircle2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { FileUpload } from '@/components/shared/forms/file-upload';
import { FormField } from '@/components/shared/forms/form-field';
import { actionStatusSchema, enterpriseApi, type ActionStatusFormValues } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { ActaActionItem } from '@/types/suites';

interface CompleteActionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  item: ActaActionItem | null;
}

export function CompleteActionDialog({
  open,
  onOpenChange,
  item,
}: CompleteActionDialogProps) {
  const queryClient = useQueryClient();
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [evidence, setEvidence] = useState<string[]>([]);
  const form = useForm<ActionStatusFormValues>({
    resolver: zodResolver(actionStatusSchema),
    defaultValues: {
      status: 'completed',
      completion_notes: '',
      completion_evidence: [],
    },
  });

  const completeMutation = useMutation({
    mutationFn: (payload: ActionStatusFormValues) =>
      item ? enterpriseApi.acta.updateActionItemStatus(item.id, payload) : Promise.reject(new Error('No action item selected.')),
    onSuccess: async () => {
      showSuccess('Action item completed.', 'Completion notes and evidence have been saved.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['acta-action-items'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-dashboard'] }),
      ]);
      onOpenChange(false);
      setEvidence([]);
      form.reset({ status: 'completed', completion_notes: '', completion_evidence: [] });
    },
    onError: showApiError,
  });

  const handleUpload = async (files: File[]) => {
    if (!item) {
      return;
    }
    setUploading(true);
    try {
      const ids: string[] = [];
      for (const file of files) {
        const uploaded = await enterpriseApi.files.upload(
          file,
          { suite: 'acta', entity_type: 'action_item_evidence', entity_id: item.id },
          setProgress,
        );
        ids.push(uploaded.id);
      }
      const nextEvidence = [...evidence, ...ids];
      setEvidence(nextEvidence);
      form.setValue('completion_evidence', nextEvidence, { shouldValidate: true });
    } catch (error) {
      showApiError(error);
    } finally {
      setUploading(false);
      setProgress(0);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>Complete Action Item</DialogTitle>
          <DialogDescription>
            Capture closure notes and optional evidence for the completed follow-up.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form
            className="space-y-4"
            onSubmit={form.handleSubmit((values) =>
              completeMutation.mutate({ ...values, completion_evidence: evidence }),
            )}
          >
            <FormField name="completion_notes" label="Completion notes" required>
              <Textarea {...form.register('completion_notes')} rows={4} />
            </FormField>

            <div className="space-y-2">
              <p className="text-sm font-medium">Completion evidence</p>
              <FileUpload onUpload={handleUpload} uploading={uploading} progress={progress} multiple />
              {evidence.length > 0 ? (
                <p className="text-xs text-muted-foreground">
                  {evidence.length} evidence file{evidence.length === 1 ? '' : 's'} attached.
                </p>
              ) : null}
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={completeMutation.isPending}>
                <CheckCircle2 className="mr-1.5 h-4 w-4" />
                {completeMutation.isPending ? 'Saving…' : 'Complete action'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
