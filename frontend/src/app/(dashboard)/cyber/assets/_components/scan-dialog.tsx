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
  target: z.string().min(1, 'Target is required'),
  port_range: z.string().optional().or(z.literal('')),
  scan_vuln: z.boolean(),
  scan_config: z.boolean(),
  scan_network: z.boolean(),
});

type FormValues = z.infer<typeof schema>;

interface ScanResult {
  scan_id: string;
  status: string;
  target: string;
  started_at: string;
}

interface ScanDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  defaultTarget?: string;
  onSuccess?: (result: ScanResult) => void;
}

export function ScanDialog({ open, onOpenChange, defaultTarget, onSuccess }: ScanDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      target: defaultTarget ?? '',
      port_range: '',
      scan_vuln: true,
      scan_config: true,
      scan_network: false,
    },
  });

  const { mutate, isPending } = useApiMutation<ScanResult, FormValues>(
    'post',
    API_ENDPOINTS.CYBER_ASSETS_SCAN,
    {
      successMessage: 'Scan started successfully',
      invalidateKeys: ['cyber-assets-scans'],
      onSuccess: (result) => {
        methods.reset();
        onOpenChange(false);
        onSuccess?.(result);
      },
    },
  );

  const onSubmit = methods.handleSubmit((data) => {
    mutate(data);
  });

  const { register, watch, setValue } = methods;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Start Asset Scan</DialogTitle>
          <DialogDescription>
            Scan a target IP, CIDR range, or hostname for vulnerabilities and misconfigurations.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-4">
            <FormField name="target" label="Target" required>
              <Input placeholder="192.168.1.0/24 or hostname" {...register('target')} />
            </FormField>

            <FormField name="port_range" label="Port Range">
              <Input placeholder="1-1024 (default: top 1000)" {...register('port_range')} />
            </FormField>

            <div className="space-y-2">
              <p className="text-sm font-medium">Scan Types</p>
              <div className="space-y-2 rounded-md border p-3">
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="scan_vuln"
                    checked={watch('scan_vuln')}
                    onCheckedChange={(v) => setValue('scan_vuln', !!v)}
                  />
                  <Label htmlFor="scan_vuln" className="cursor-pointer">
                    Vulnerability scan (CVE matching)
                  </Label>
                </div>
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="scan_config"
                    checked={watch('scan_config')}
                    onCheckedChange={(v) => setValue('scan_config', !!v)}
                  />
                  <Label htmlFor="scan_config" className="cursor-pointer">
                    Configuration audit (CIS benchmarks)
                  </Label>
                </div>
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="scan_network"
                    checked={watch('scan_network')}
                    onCheckedChange={(v) => setValue('scan_network', !!v)}
                  />
                  <Label htmlFor="scan_network" className="cursor-pointer">
                    Network topology discovery
                  </Label>
                </div>
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
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
