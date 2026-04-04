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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { toast } from 'sonner';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';

const SCAN_INTERVALS = [
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Daily at midnight', value: '0 0 * * *' },
  { label: 'Daily at 6 AM', value: '0 6 * * *' },
  { label: 'Weekly (Sunday midnight)', value: '0 0 * * 0' },
  { label: 'Monthly (1st at midnight)', value: '0 0 1 * *' },
] as const;

const schema = z.object({
  scan_type: z.enum(['network', 'cloud', 'agent']),
  target: z.string().min(1, 'At least one target is required'),
  schedule: z.string().min(1, 'Schedule is required'),
  label: z.string().min(1, 'A descriptive label is required').max(100),
});

type FormValues = z.infer<typeof schema>;

interface ScanScheduleDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ScanScheduleDialog({ open, onOpenChange }: ScanScheduleDialogProps) {
  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      scan_type: 'network',
      target: '',
      schedule: '0 0 * * *',
      label: '',
    },
  });

  const { register, handleSubmit, formState: { errors, isSubmitting }, reset, setValue, watch } = methods;
  const selectedSchedule = watch('schedule');

  const onSubmit = handleSubmit(async (data) => {
    try {
      await apiPost(API_ENDPOINTS.CYBER_ASSETS_SCAN, {
        scan_type: data.scan_type,
        targets: data.target.split(/[,\n]/).map((t: string) => t.trim()).filter(Boolean),
        schedule: data.schedule,
        label: data.label,
      });
      toast.success('Scheduled scan created');
      reset();
      onOpenChange(false);
    } catch {
      toast.error('Failed to create scheduled scan');
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Schedule Recurring Scan</DialogTitle>
          <DialogDescription>
            Configure automated discovery scans that run on a recurring schedule.
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-4">
            <div>
              <Label>Label</Label>
              <Input
                placeholder="e.g., Production network weekly scan"
                {...register('label')}
                className="mt-1"
              />
              {errors.label && <p className="mt-1 text-xs text-destructive">{errors.label.message}</p>}
            </div>

            <div>
              <Label>Scan Type</Label>
              <Select
                value={watch('scan_type')}
                onValueChange={(v) => setValue('scan_type', v as FormValues['scan_type'])}
              >
                <SelectTrigger className="mt-1">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="network">Network Discovery</SelectItem>
                  <SelectItem value="cloud">Cloud Resource Sync</SelectItem>
                  <SelectItem value="agent">Agent-Based Inventory</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div>
              <Label>Targets</Label>
              <Input
                placeholder="IPs, CIDR ranges, or hostnames (comma-separated)"
                {...register('target')}
                className="mt-1"
              />
              {errors.target && <p className="mt-1 text-xs text-destructive">{errors.target.message}</p>}
              <p className="mt-1 text-xs text-muted-foreground">
                Examples: 10.0.0.0/24, 192.168.1.1-100, server-prod-*.example.com
              </p>
            </div>

            <div>
              <Label>Schedule</Label>
              <Select
                value={selectedSchedule}
                onValueChange={(v) => setValue('schedule', v)}
              >
                <SelectTrigger className="mt-1">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SCAN_INTERVALS.map((interval) => (
                    <SelectItem key={interval.value} value={interval.value}>
                      {interval.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="mt-1 text-xs text-muted-foreground">
                Cron expression: <code className="text-[10px] rounded bg-muted px-1">{selectedSchedule}</code>
              </p>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Creating...' : 'Create Schedule'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
