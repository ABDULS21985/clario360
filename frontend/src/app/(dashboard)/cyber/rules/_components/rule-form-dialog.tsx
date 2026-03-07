'use client';

import { useEffect } from 'react';
import { useForm, FormProvider } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { DetectionRule } from '@/types/cyber';

const schema = z.object({
  name: z.string().min(2).max(255),
  description: z.string().min(1),
  type: z.enum(['sigma', 'threshold', 'correlation', 'anomaly']),
  severity: z.enum(['critical', 'high', 'medium', 'low', 'info']),
  condition: z.string().optional().or(z.literal('')),
  mitre_technique_ids: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

interface RuleFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rule?: DetectionRule | null;
  onSuccess?: () => void;
}

export function RuleFormDialog({ open, onOpenChange, rule, onSuccess }: RuleFormDialogProps) {
  const isEdit = !!rule;

  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      description: '',
      type: 'sigma',
      severity: 'medium',
      condition: '',
      mitre_technique_ids: '',
    },
  });

  useEffect(() => {
    if (open) {
      methods.reset({
        name: rule?.name ?? '',
        description: rule?.description ?? '',
        type: rule?.type ?? 'sigma',
        severity: rule?.severity ?? 'medium',
        condition: rule?.condition ?? '',
        mitre_technique_ids: (rule?.mitre_technique_ids ?? []).join(', '),
      });
    }
  }, [open, rule, methods]);

  const url = isEdit
    ? `${API_ENDPOINTS.CYBER_RULES}/${rule!.id}`
    : API_ENDPOINTS.CYBER_RULES;

  const { mutate, isPending } = useApiMutation<DetectionRule, FormValues>(
    isEdit ? 'put' : 'post',
    url,
    {
      successMessage: isEdit ? 'Rule updated' : 'Rule created',
      invalidateKeys: ['cyber-rules'],
      onSuccess: () => {
        onOpenChange(false);
        onSuccess?.();
      },
    },
  );

  const onSubmit = methods.handleSubmit((data) => {
    const payload = {
      ...data,
      mitre_technique_ids: data.mitre_technique_ids
        ? data.mitre_technique_ids.split(',').map((s) => s.trim()).filter(Boolean)
        : [],
    };
    mutate(payload as unknown as FormValues);
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit Detection Rule' : 'Create Detection Rule'}</DialogTitle>
        </DialogHeader>
        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-4">
            <FormField name="name" label="Rule Name" required>
              <Input placeholder="Suspicious PowerShell Execution" {...methods.register('name')} />
            </FormField>
            <FormField name="description" label="Description" required>
              <Textarea rows={2} placeholder="Describe what this rule detects…" {...methods.register('description')} />
            </FormField>
            <div className="grid grid-cols-2 gap-4">
              <FormField name="type" label="Type" required>
                <Select
                  value={methods.watch('type')}
                  onValueChange={(v) => methods.setValue('type', v as FormValues['type'])}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {['sigma', 'threshold', 'correlation', 'anomaly'].map((t) => (
                      <SelectItem key={t} value={t} className="capitalize">{t}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="severity" label="Severity" required>
                <Select
                  value={methods.watch('severity')}
                  onValueChange={(v) => methods.setValue('severity', v as FormValues['severity'])}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {['critical', 'high', 'medium', 'low', 'info'].map((s) => (
                      <SelectItem key={s} value={s} className="capitalize">{s}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>
            <FormField name="condition" label="Condition / Query">
              <Textarea rows={3} placeholder="Sigma rule YAML, threshold expression…" className="font-mono text-xs" {...methods.register('condition')} />
            </FormField>
            <FormField name="mitre_technique_ids" label="MITRE Technique IDs (comma separated)">
              <Input placeholder="T1059, T1086" {...methods.register('mitre_technique_ids')} />
            </FormField>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? 'Saving…' : isEdit ? 'Save Changes' : 'Create Rule'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
