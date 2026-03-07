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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { CyberAsset } from '@/types/cyber';

const schema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters').max(255),
  type: z.enum(['server', 'endpoint', 'cloud_resource', 'network_device', 'iot_device', 'application', 'database', 'container']),
  criticality: z.enum(['critical', 'high', 'medium', 'low']),
  status: z.enum(['active', 'inactive', 'decommissioned', 'unknown']),
  ip_address: z.string().optional().or(z.literal('')),
  hostname: z.string().optional().or(z.literal('')),
  os: z.string().optional().or(z.literal('')),
  owner: z.string().optional().or(z.literal('')),
  department: z.string().optional().or(z.literal('')),
  location: z.string().optional().or(z.literal('')),
});

type FormValues = z.infer<typeof schema>;

interface EditAssetDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  asset: CyberAsset;
  onSuccess?: (asset: CyberAsset) => void;
}

export function EditAssetDialog({ open, onOpenChange, asset, onSuccess }: EditAssetDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: asset.name,
      type: asset.type,
      criticality: asset.criticality,
      status: asset.status,
      ip_address: asset.ip_address ?? '',
      hostname: asset.hostname ?? '',
      os: asset.os ?? '',
      owner: asset.owner ?? '',
      department: asset.department ?? '',
      location: asset.location ?? '',
    },
  });

  useEffect(() => {
    if (open) {
      methods.reset({
        name: asset.name,
        type: asset.type,
        criticality: asset.criticality,
        status: asset.status,
        ip_address: asset.ip_address ?? '',
        hostname: asset.hostname ?? '',
        os: asset.os ?? '',
        owner: asset.owner ?? '',
        department: asset.department ?? '',
        location: asset.location ?? '',
      });
    }
  }, [open, asset, methods]);

  const { mutate, isPending } = useApiMutation<CyberAsset, FormValues>(
    'put',
    `${API_ENDPOINTS.CYBER_ASSETS}/${asset.id}`,
    {
      successMessage: 'Asset updated successfully',
      invalidateKeys: ['cyber-assets', 'cyber-assets-stats', `cyber-asset-${asset.id}`],
      onSuccess: (updated) => {
        onOpenChange(false);
        onSuccess?.(updated);
      },
    },
  );

  const onSubmit = methods.handleSubmit((data) => {
    mutate(data);
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit Asset</DialogTitle>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-4">
            <FormField name="name" label="Name" required>
              <Input placeholder="web-prod-01" {...methods.register('name')} />
            </FormField>

            <div className="grid grid-cols-2 gap-4">
              <FormField name="type" label="Type" required>
                <Select
                  value={methods.watch('type')}
                  onValueChange={(v) => methods.setValue('type', v as FormValues['type'])}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {['server', 'endpoint', 'cloud_resource', 'network_device', 'iot_device', 'application', 'database', 'container'].map((t) => (
                      <SelectItem key={t} value={t}>{t.replace(/_/g, ' ')}</SelectItem>
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

            <FormField name="status" label="Status" required>
              <Select
                value={methods.watch('status')}
                onValueChange={(v) => methods.setValue('status', v as FormValues['status'])}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {['active', 'inactive', 'decommissioned', 'unknown'].map((s) => (
                    <SelectItem key={s} value={s} className="capitalize">{s}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            <div className="grid grid-cols-2 gap-4">
              <FormField name="ip_address" label="IP Address">
                <Input placeholder="192.168.1.1" {...methods.register('ip_address')} />
              </FormField>
              <FormField name="hostname" label="Hostname">
                <Input placeholder="web-prod-01.example.com" {...methods.register('hostname')} />
              </FormField>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <FormField name="os" label="Operating System">
                <Input placeholder="Ubuntu 22.04" {...methods.register('os')} />
              </FormField>
              <FormField name="owner" label="Owner">
                <Input placeholder="Security Team" {...methods.register('owner')} />
              </FormField>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <FormField name="department" label="Department">
                <Input placeholder="Engineering" {...methods.register('department')} />
              </FormField>
              <FormField name="location" label="Location">
                <Input placeholder="us-east-1" {...methods.register('location')} />
              </FormField>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? 'Saving…' : 'Save Changes'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
