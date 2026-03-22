'use client';

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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { CyberAsset } from '@/types/cyber';

const schema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters').max(255),
  type: z.enum(['server', 'endpoint', 'cloud_resource', 'network_device', 'iot_device', 'application', 'database', 'container']),
  criticality: z.enum(['critical', 'high', 'medium', 'low']),
  ip_address: z.string().optional().or(z.literal('')),
  hostname: z.string().optional().or(z.literal('')),
  os: z.string().optional().or(z.literal('')),
  owner: z.string().optional().or(z.literal('')),
  department: z.string().optional().or(z.literal('')),
});

type FormValues = z.infer<typeof schema>;

interface CreateAssetDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (asset: CyberAsset) => void;
}

export function CreateAssetDialog({ open, onOpenChange, onSuccess }: CreateAssetDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { type: 'server', criticality: 'medium' },
  });

  const { mutate, isPending } = useApiMutation<CyberAsset, FormValues>(
    'post',
    API_ENDPOINTS.CYBER_ASSETS,
    {
      successMessage: 'Asset created successfully',
      invalidateKeys: ['cyber-assets', 'cyber-assets-stats'],
      onSuccess: (asset) => {
        methods.reset();
        onOpenChange(false);
        onSuccess?.(asset);
      },
    },
  );

  const onSubmit = methods.handleSubmit((data) => {
    // Strip empty strings → undefined so backend *string fields stay nil
    // (Go validates `omitempty,ip` — a pointer to "" fails IP validation)
    const cleaned = Object.fromEntries(
      Object.entries(data).map(([k, v]) => [k, v === '' ? undefined : v]),
    ) as FormValues;
    mutate(cleaned);
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Create Asset</DialogTitle>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-4">
            <FormField name="name" label="Name" required>
              <Input placeholder="web-prod-01" {...methods.register('name')} />
            </FormField>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <FormField name="type" label="Type" required>
                <Select
                  value={methods.watch('type')}
                  onValueChange={(v) => methods.setValue('type', v as FormValues['type'])}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {['server', 'endpoint', 'cloud_resource', 'network_device', 'iot_device', 'application', 'database', 'container'].map((t) => (
                      <SelectItem key={t} value={t}>{t.replace('_', ' ')}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="criticality" label="Criticality" required>
                <Select
                  value={methods.watch('criticality')}
                  onValueChange={(v) => methods.setValue('criticality', v as FormValues['criticality'])}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {['critical', 'high', 'medium', 'low'].map((c) => (
                      <SelectItem key={c} value={c} className="capitalize">{c}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <FormField name="ip_address" label="IP Address">
                <Input placeholder="192.168.1.1" {...methods.register('ip_address')} />
              </FormField>
              <FormField name="hostname" label="Hostname">
                <Input placeholder="web-prod-01.example.com" {...methods.register('hostname')} />
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <FormField name="os" label="Operating System">
                <Input placeholder="Ubuntu 22.04" {...methods.register('os')} />
              </FormField>
              <FormField name="owner" label="Owner">
                <Input placeholder="Security Team" {...methods.register('owner')} />
              </FormField>
            </div>

            <FormField name="department" label="Department">
              <Input placeholder="Engineering" {...methods.register('department')} />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? 'Creating…' : 'Create Asset'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
