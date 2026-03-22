'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import { AlertTriangle } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Separator } from '@/components/ui/separator';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { cn } from '@/lib/utils';
import type { VCISORiskEntry } from '@/types/cyber';

interface RiskAcceptanceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  risk: VCISORiskEntry;
  onAccepted: () => void;
}

export function RiskAcceptanceDialog({
  open,
  onOpenChange,
  risk,
  onAccepted,
}: RiskAcceptanceDialogProps) {
  const [rationale, setRationale] = useState('');
  const [expiryDate, setExpiryDate] = useState('');
  const [confirmChecked, setConfirmChecked] = useState(false);

  const acceptMutation = useApiMutation<VCISORiskEntry, Record<string, unknown>>(
    'put',
    `${API_ENDPOINTS.CYBER_VCISO_RISKS}/${risk.id}`,
    {
      successMessage: 'Risk accepted successfully',
      invalidateKeys: ['vciso-risks', API_ENDPOINTS.CYBER_VCISO_RISKS_STATS],
      onSuccess: () => {
        resetForm();
        onOpenChange(false);
        onAccepted();
      },
    },
  );

  const resetForm = () => {
    setRationale('');
    setExpiryDate('');
    setConfirmChecked(false);
  };

  const handleSubmit = () => {
    if (!rationale.trim()) {
      toast.error('Acceptance rationale is required');
      return;
    }
    if (rationale.trim().length < 20) {
      toast.error('Please provide a more detailed rationale (at least 20 characters)');
      return;
    }
    if (!confirmChecked) {
      toast.error('Please confirm that you understand the implications');
      return;
    }

    acceptMutation.mutate({
      // Preserve all existing fields (UpdateRiskRequest requires full DTO)
      title: risk.title,
      description: risk.description,
      category: risk.category,
      department: risk.department,
      inherent_score: risk.inherent_score,
      residual_score: risk.residual_score,
      likelihood: risk.likelihood,
      impact: risk.impact,
      treatment: risk.treatment,
      owner_id: risk.owner_id || undefined,
      owner_name: risk.owner_name,
      review_date: risk.review_date || undefined,
      business_services: risk.business_services,
      controls: risk.controls,
      tags: risk.tags,
      treatment_plan: risk.treatment_plan,
      // Updated fields
      status: 'accepted',
      acceptance_rationale: rationale.trim(),
      acceptance_expiry: expiryDate || undefined,
    });
  };

  const handleOpenChange = (o: boolean) => {
    if (!o) {
      resetForm();
    }
    onOpenChange(o);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-amber-500" />
            Accept Risk
          </DialogTitle>
          <DialogDescription>
            You are about to accept the risk &ldquo;{risk.title}&rdquo;. This means the
            organization acknowledges the risk and chooses not to mitigate it further.
          </DialogDescription>
        </DialogHeader>

        {/* Risk summary */}
        <div className="rounded-xl border bg-muted/30 p-4 space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Risk:</span>
            <span className="font-medium">{risk.title}</span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Residual Score:</span>
            <span
              className={cn(
                'font-bold',
                risk.residual_score <= 30
                  ? 'text-green-600'
                  : risk.residual_score <= 60
                    ? 'text-amber-600'
                    : 'text-red-600',
              )}
            >
              {risk.residual_score}
            </span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Category:</span>
            <span>{risk.category}</span>
          </div>
        </div>

        <Separator />

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="acceptance-rationale">
              Acceptance Rationale <span className="text-destructive">*</span>
            </Label>
            <Textarea
              id="acceptance-rationale"
              value={rationale}
              onChange={(e) => setRationale(e.target.value)}
              placeholder="Explain why this risk is being accepted and the business justification for not mitigating it further..."
              rows={4}
            />
            <p className="text-xs text-muted-foreground">
              Minimum 20 characters. This will be recorded in the audit trail.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="acceptance-expiry">Acceptance Expiry Date (optional)</Label>
            <Input
              id="acceptance-expiry"
              type="date"
              value={expiryDate}
              onChange={(e) => setExpiryDate(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              If set, the risk acceptance will need to be reviewed by this date.
            </p>
          </div>

          <div className="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4">
            <input
              type="checkbox"
              id="accept-confirm"
              checked={confirmChecked}
              onChange={(e) => setConfirmChecked(e.target.checked)}
              className="mt-0.5 h-4 w-4 rounded border-amber-300 text-amber-600 focus:ring-amber-500"
            />
            <label htmlFor="accept-confirm" className="text-sm text-amber-800 leading-relaxed">
              I confirm that I understand the implications of accepting this risk and that
              the residual risk level is within the organization&apos;s risk appetite. This
              decision will be logged for compliance and audit purposes.
            </label>
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={acceptMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={acceptMutation.isPending || !confirmChecked || !rationale.trim()}
            className="bg-amber-600 hover:bg-amber-700 text-white"
          >
            {acceptMutation.isPending ? 'Processing...' : 'Accept Risk'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
