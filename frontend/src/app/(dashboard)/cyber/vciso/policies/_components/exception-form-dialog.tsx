'use client';

import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { VCISOPolicy, VCISOPolicyException } from '@/types/cyber';

interface ExceptionFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  policies: VCISOPolicy[];
  onSuccess: () => void;
  preselectedPolicyId?: string;
}

export function ExceptionFormDialog({
  open,
  onOpenChange,
  policies,
  onSuccess,
  preselectedPolicyId,
}: ExceptionFormDialogProps) {
  const [policyId, setPolicyId] = useState('');
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [justification, setJustification] = useState('');
  const [compensatingControls, setCompensatingControls] = useState('');
  const [expiresAt, setExpiresAt] = useState('');

  useEffect(() => {
    if (open) {
      setPolicyId(preselectedPolicyId ?? '');
      setTitle('');
      setDescription('');
      setJustification('');
      setCompensatingControls('');
      setExpiresAt('');
    }
  }, [open, preselectedPolicyId]);

  const createMutation = useApiMutation<VCISOPolicyException, Record<string, unknown>>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_POLICY_EXCEPTIONS,
    {
      invalidateKeys: ['vciso-policy-exceptions', 'vciso-policies'],
      successMessage: 'Exception request submitted',
      onSuccess: () => {
        onOpenChange(false);
        onSuccess();
      },
    },
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!policyId) {
      toast.error('Please select a policy');
      return;
    }
    if (!title.trim()) {
      toast.error('Title is required');
      return;
    }
    if (!description.trim()) {
      toast.error('Description is required');
      return;
    }
    if (!justification.trim()) {
      toast.error('Justification is required');
      return;
    }
    if (!compensatingControls.trim()) {
      toast.error('Compensating controls are required');
      return;
    }
    if (!expiresAt) {
      toast.error('Expiration date is required');
      return;
    }

    createMutation.mutate({
      policy_id: policyId,
      title: title.trim(),
      description: description.trim(),
      justification: justification.trim(),
      compensating_controls: compensatingControls.trim(),
      expires_at: new Date(expiresAt).toISOString(),
    });
  };

  const isSubmitting = createMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Request Policy Exception</DialogTitle>
          <DialogDescription>
            Submit a request for a temporary exception to an existing policy. Include
            compensating controls and a justification.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div className="space-y-2">
            <Label htmlFor="exception-policy">Policy</Label>
            <Select
              value={policyId}
              onValueChange={setPolicyId}
              disabled={isSubmitting}
            >
              <SelectTrigger id="exception-policy">
                <SelectValue placeholder="Select a policy" />
              </SelectTrigger>
              <SelectContent>
                {policies.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    {p.title}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="exception-title">Title</Label>
            <Input
              id="exception-title"
              placeholder="e.g., Temporary access exception for Project X"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={isSubmitting}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="exception-description">Description</Label>
            <Textarea
              id="exception-description"
              placeholder="Describe the exception being requested..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={isSubmitting}
              className="min-h-[80px]"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="exception-justification">Justification</Label>
            <Textarea
              id="exception-justification"
              placeholder="Explain why this exception is necessary..."
              value={justification}
              onChange={(e) => setJustification(e.target.value)}
              disabled={isSubmitting}
              className="min-h-[80px]"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="exception-controls">Compensating Controls</Label>
            <Textarea
              id="exception-controls"
              placeholder="Describe the compensating controls in place..."
              value={compensatingControls}
              onChange={(e) => setCompensatingControls(e.target.value)}
              disabled={isSubmitting}
              className="min-h-[80px]"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="exception-expires">Expires At</Label>
            <Input
              id="exception-expires"
              type="date"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
              disabled={isSubmitting}
              min={new Date().toISOString().split('T')[0]}
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? 'Submitting...' : 'Submit Exception Request'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
