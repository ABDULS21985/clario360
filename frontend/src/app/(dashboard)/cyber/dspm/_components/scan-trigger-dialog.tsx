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
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';

const schema = z.object({
  scope: z.array(z.string()).min(1, 'Select at least one scope item'),
  asset_types: z.string().optional(),
  full_scan: z.boolean(),
});

type FormValues = z.infer<typeof schema>;

interface ScanResult {
  scan_id: string;
  status: string;
  started_at: string;
}

interface ScanTriggerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (result: ScanResult) => void;
}

const SCOPE_OPTIONS = [
  { id: 'databases', label: 'Databases' },
  { id: 'cloud_storage', label: 'Cloud Storage' },
  { id: 'file_servers', label: 'File Servers' },
  { id: 'api_endpoints', label: 'API Endpoints' },
];

export function ScanTriggerDialog({ open, onOpenChange, onSuccess }: ScanTriggerDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { scope: ['databases'], asset_types: '', full_scan: false },
  });

  const { mutate, isPending } = useApiMutation<ScanResult, FormValues>(
    'post',
    API_ENDPOINTS.CYBER_DSPM_SCAN,
    {
      successMessage: 'DSPM scan started',
      invalidateKeys: ['cyber-dspm'],
      onSuccess: (result) => {
        methods.reset();
        onOpenChange(false);
        onSuccess?.(result);
      },
    },
  );

  const { watch, setValue } = methods;
  const scope = watch('scope');

  const toggleScope = (id: string) => {
    const current = scope ?? [];
    if (current.includes(id)) {
      setValue('scope', current.filter((s) => s !== id));
    } else {
      setValue('scope', [...current, id]);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Trigger DSPM Scan</DialogTitle>
          <DialogDescription>
            Scan your data infrastructure for classification, risk, and compliance posture.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit((d) => mutate(d))} className="space-y-4">
            <div className="space-y-2">
              <p className="text-sm font-medium">Scan Scope <span className="text-destructive">*</span></p>
              <div className="space-y-2 rounded-md border p-3">
                {SCOPE_OPTIONS.map(({ id, label }) => (
                  <div key={id} className="flex items-center gap-2">
                    <Checkbox
                      id={id}
                      checked={(scope ?? []).includes(id)}
                      onCheckedChange={() => toggleScope(id)}
                    />
                    <Label htmlFor={id} className="cursor-pointer">{label}</Label>
                  </div>
                ))}
              </div>
              {methods.formState.errors.scope && (
                <p className="text-xs text-destructive">{methods.formState.errors.scope.message}</p>
              )}
            </div>

            <FormField name="asset_types" label="Asset Type Filter">
              <Input placeholder="e.g. postgresql,mysql (blank = all)" {...methods.register('asset_types')} />
            </FormField>

            <div className="flex items-center gap-2">
              <Checkbox
                id="full_scan"
                checked={watch('full_scan')}
                onCheckedChange={(v) => setValue('full_scan', !!v)}
              />
              <Label htmlFor="full_scan" className="cursor-pointer">
                Full re-scan (slower, overrides cached results)
              </Label>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? 'Starting…' : 'Start Scan'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
