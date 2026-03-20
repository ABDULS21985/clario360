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
import { Label } from '@/components/ui/label';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';

const schema = z.object({
  scan_type: z.enum(['network', 'cloud', 'agent'], {
    required_error: 'Scan type is required',
  }),
  target: z.string().min(1, 'At least one target is required'),
  ports: z.string().optional().or(z.literal('')),
  include_vuln: z.boolean(),
  include_config: z.boolean(),
});

type FormValues = z.infer<typeof schema>;

interface ScanTriggerPayload {
  scan_type: string;
  targets: string[];
  ports?: number[];
  options?: Record<string, unknown>;
}

interface ScanResult {
  scan_id: string;
  status: string;
  message: string;
}

interface ScanDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  defaultTarget?: string;
  onSuccess?: (result: ScanResult) => void;
}

function parsePorts(raw: string): number[] | undefined {
  if (!raw.trim()) return undefined;
  const ports: number[] = [];
  for (const part of raw.split(',')) {
    const trimmed = part.trim();
    if (trimmed.includes('-')) {
      const [start, end] = trimmed.split('-').map(Number);
      if (!isNaN(start) && !isNaN(end)) {
        for (let p = start; p <= end && ports.length < 1000; p++) {
          if (p >= 1 && p <= 65535) ports.push(p);
        }
      }
    } else {
      const p = Number(trimmed);
      if (!isNaN(p) && p >= 1 && p <= 65535) ports.push(p);
    }
    if (ports.length >= 1000) break;
  }
  return ports.length > 0 ? ports.slice(0, 1000) : undefined;
}

export function ScanDialog({ open, onOpenChange, defaultTarget, onSuccess }: ScanDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      scan_type: 'network',
      target: defaultTarget ?? '',
      ports: '',
      include_vuln: true,
      include_config: true,
    },
  });

  const { mutate, isPending } = useApiMutation<ScanResult, ScanTriggerPayload>(
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
    const targets = data.target.split(',').map((t) => t.trim()).filter(Boolean);
    const ports = parsePorts(data.ports ?? '');
    const options: Record<string, unknown> = {};
    if (data.include_vuln) options['vuln_scan'] = true;
    if (data.include_config) options['config_audit'] = true;

    mutate({
      scan_type: data.scan_type,
      targets,
      ...(ports ? { ports } : {}),
      ...(Object.keys(options).length > 0 ? { options } : {}),
    });
  });

  const { register, watch, setValue } = methods;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Start Asset Scan</DialogTitle>
          <DialogDescription>
            Scan targets for vulnerabilities, misconfigurations, and network topology.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="scan_type">Scan Type</Label>
              <select
                id="scan_type"
                {...register('scan_type')}
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <option value="network">Network — topology &amp; port discovery</option>
                <option value="cloud">Cloud — cloud asset enumeration</option>
                <option value="agent">Agent — agent-based host scan</option>
              </select>
            </div>

            <FormField name="target" label="Targets" required>
              <Input
                placeholder="10.0.0.1, 192.168.1.0/24, host.example.com"
                {...register('target')}
              />
              <p className="text-xs text-muted-foreground">Comma-separated IPs, CIDR ranges, or hostnames.</p>
            </FormField>

            <FormField name="ports" label="Ports">
              <Input placeholder="80,443,8080 or 1-1024 (optional)" {...register('ports')} />
              <p className="text-xs text-muted-foreground">Leave blank to scan the top 1,000 ports.</p>
            </FormField>

            <div className="space-y-2">
              <p className="text-sm font-medium">Additional checks</p>
              <div className="space-y-2 rounded-md border p-3">
                <label className="flex cursor-pointer items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    className="h-4 w-4 rounded border-gray-300"
                    checked={watch('include_vuln')}
                    onChange={(e) => setValue('include_vuln', e.target.checked)}
                  />
                  Vulnerability matching (CVE lookup)
                </label>
                <label className="flex cursor-pointer items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    className="h-4 w-4 rounded border-gray-300"
                    checked={watch('include_config')}
                    onChange={(e) => setValue('include_config', e.target.checked)}
                  />
                  Configuration audit (CIS benchmarks)
                </label>
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
