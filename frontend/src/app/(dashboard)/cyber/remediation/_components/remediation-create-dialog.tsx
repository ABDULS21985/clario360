'use client';

import { useForm, FormProvider, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { Plus, Trash2 } from 'lucide-react';
import type { RemediationAction } from '@/types/cyber';

const stepSchema = z.object({
  number: z.number(),
  action: z.string().min(1, 'Action is required'),
  description: z.string().optional(),
  target: z.string().optional(),
});

const schema = z.object({
  title: z.string().min(3, 'Title must be at least 3 characters'),
  description: z.string().min(10, 'Description must be at least 10 characters'),
  type: z.enum(['patch', 'config_change', 'block_ip', 'isolate_asset', 'firewall_rule', 'access_revoke', 'certificate_renew', 'custom']),
  severity: z.enum(['critical', 'high', 'medium', 'low']),
  execution_mode: z.enum(['automated', 'manual', 'semi_automated']),
  requires_approval_from: z.enum(['security_manager', 'ciso', 'tenant_admin']).default('security_manager'),
  steps: z.array(stepSchema).min(1, 'At least one step is required'),
  alert_id: z.string().optional().or(z.literal('')),
  vulnerability_id: z.string().optional().or(z.literal('')),
});

type FormValues = z.infer<typeof schema>;

interface RemediationCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (action: RemediationAction) => void;
  defaultAlertId?: string;
  defaultVulnId?: string;
}

export function RemediationCreateDialog({
  open,
  onOpenChange,
  onSuccess,
  defaultAlertId,
  defaultVulnId,
}: RemediationCreateDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      type: 'patch',
      severity: 'medium',
      execution_mode: 'manual' as const,
      requires_approval_from: 'security_manager' as const,
      alert_id: defaultAlertId ?? '',
      vulnerability_id: defaultVulnId ?? '',
      steps: [{ number: 1, action: '', description: '', target: '' }],
    },
  });

  const { fields, append, remove } = useFieldArray({ control: methods.control, name: 'steps' });

  const { mutate, isPending } = useApiMutation<RemediationAction, FormValues>(
    'post',
    API_ENDPOINTS.CYBER_REMEDIATION,
    {
      successMessage: 'Remediation action created',
      invalidateKeys: ['cyber-remediation', 'cyber-remediation-stats'],
      onSuccess: (action) => {
        methods.reset();
        onOpenChange(false);
        onSuccess?.(action);
      },
    },
  );

  const onSubmit = methods.handleSubmit((data) => {
    const { steps, ...rest } = data;
    const payload = {
      ...rest,
      alert_id: data.alert_id || undefined,
      vulnerability_id: data.vulnerability_id || undefined,
      affected_asset_ids: [],
      plan: {
        steps: steps.map((s, i) => ({ ...s, number: i + 1 })),
        reversible: data.execution_mode !== 'automated',
        risk_level: data.severity,
      },
    };
    mutate(payload as unknown as FormValues);
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Remediation Action</DialogTitle>
          <DialogDescription>Define a structured remediation plan with step-by-step execution.</DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-5">
            <FormField name="title" label="Title" required>
              <Input placeholder="Apply security patch CVE-2024-1234" {...methods.register('title')} />
            </FormField>

            <FormField name="description" label="Description" required>
              <Textarea rows={2} placeholder="What will this remediation accomplish?" {...methods.register('description')} />
            </FormField>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <FormField name="type" label="Type" required>
                <Select value={methods.watch('type')} onValueChange={(v) => methods.setValue('type', v as FormValues['type'])}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {['patch', 'config_change', 'block_ip', 'isolate_asset', 'firewall_rule', 'access_revoke', 'certificate_renew', 'custom'].map((t) => (
                      <SelectItem key={t} value={t} className="capitalize">{t.replace(/_/g, ' ')}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="severity" label="Severity" required>
                <Select value={methods.watch('severity')} onValueChange={(v) => methods.setValue('severity', v as FormValues['severity'])}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {['critical', 'high', 'medium', 'low'].map((s) => (
                      <SelectItem key={s} value={s} className="capitalize">{s}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="execution_mode" label="Execution Mode" required>
                <Select value={methods.watch('execution_mode')} onValueChange={(v) => methods.setValue('execution_mode', v as FormValues['execution_mode'])}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="manual">Manual</SelectItem>
                    <SelectItem value="semi_automated">Semi-Automated</SelectItem>
                    <SelectItem value="automated">Automated</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <FormField name="requires_approval_from" label="Requires Approval From">
              <Select
                value={methods.watch('requires_approval_from')}
                onValueChange={(v) => methods.setValue('requires_approval_from', v as FormValues['requires_approval_from'])}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="security_manager">Security Manager</SelectItem>
                  <SelectItem value="ciso">CISO</SelectItem>
                  <SelectItem value="tenant_admin">Tenant Admin</SelectItem>
                </SelectContent>
              </Select>
            </FormField>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <FormField name="alert_id" label="Linked Alert ID">
                <Input placeholder="Optional alert UUID" {...methods.register('alert_id')} />
              </FormField>
              <FormField name="vulnerability_id" label="Linked Vulnerability ID">
                <Input placeholder="Optional vuln UUID" {...methods.register('vulnerability_id')} />
              </FormField>
            </div>

            {/* Steps */}
            <div>
              <div className="mb-2 flex items-center justify-between">
                <p className="text-sm font-medium">Remediation Steps</p>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => append({ number: fields.length + 1, action: '', description: '', target: '' })}
                >
                  <Plus className="mr-1 h-3.5 w-3.5" /> Add Step
                </Button>
              </div>
              <div className="space-y-3">
                {fields.map((field, idx) => (
                  <div key={field.id} className="rounded-lg border bg-muted/20 p-3">
                    <div className="mb-2 flex items-center justify-between">
                      <span className="text-xs font-semibold text-muted-foreground">Step {idx + 1}</span>
                      {fields.length > 1 && (
                        <button type="button" onClick={() => remove(idx)} className="text-muted-foreground hover:text-destructive">
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      )}
                    </div>
                    <div className="grid gap-2">
                      <Input
                        placeholder="Action (e.g. Run apt-get upgrade)"
                        {...methods.register(`steps.${idx}.action`)}
                      />
                      <Input
                        placeholder="Target host or resource (optional)"
                        {...methods.register(`steps.${idx}.target`)}
                      />
                      <Input
                        placeholder="Additional description (optional)"
                        {...methods.register(`steps.${idx}.description`)}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? 'Creating…' : 'Create Action'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
