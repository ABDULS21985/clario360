'use client';

import { useState, useMemo } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { DSPMExceptionType } from '@/types/cyber';

type ExceptionRequestPayload = {
  exception_type: DSPMExceptionType;
  remediation_id?: string;
  data_asset_id?: string;
  policy_id?: string;
  justification: string;
  business_reason?: string;
  compensating_controls?: string;
  risk_score: number;
  expires_at: string;
  review_interval_days: number;
};

interface ExceptionRequestDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: ExceptionRequestPayload) => void;
  isSubmitting?: boolean;
}

const EXCEPTION_TYPES: { value: DSPMExceptionType; label: string }[] = [
  { value: 'posture_finding', label: 'Posture Finding' },
  { value: 'policy_violation', label: 'Policy Violation' },
  { value: 'overprivileged_access', label: 'Overprivileged Access' },
  { value: 'exposure_risk', label: 'Exposure Risk' },
  { value: 'encryption_gap', label: 'Encryption Gap' },
];

const REVIEW_INTERVALS = [
  { value: '30', label: '30 days' },
  { value: '60', label: '60 days' },
  { value: '90', label: '90 days' },
  { value: '180', label: '180 days' },
];

export function ExceptionRequestDialog({
  open,
  onOpenChange,
  onSubmit,
  isSubmitting = false,
}: ExceptionRequestDialogProps) {
  const [exceptionType, setExceptionType] = useState<DSPMExceptionType>('posture_finding');
  const [justification, setJustification] = useState('');
  const [businessReason, setBusinessReason] = useState('');
  const [compensatingControls, setCompensatingControls] = useState('');
  const [riskScore, setRiskScore] = useState<number>(50);
  const [expiresAt, setExpiresAt] = useState('');
  const [reviewIntervalDays, setReviewIntervalDays] = useState(90);
  const [remediationId, setRemediationId] = useState('');
  const [dataAssetId, setDataAssetId] = useState('');
  const [policyId, setPolicyId] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  const maxDate = useMemo(() => {
    const d = new Date();
    d.setDate(d.getDate() + 365);
    return d.toISOString().split('T')[0];
  }, []);

  const today = useMemo(() => new Date().toISOString().split('T')[0], []);

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!justification || justification.trim().length < 20) {
      newErrors.justification = 'Justification must be at least 20 characters';
    }

    if (!expiresAt) {
      newErrors.expires_at = 'Expiration date is required';
    } else {
      const expDate = new Date(expiresAt);
      const maxExpDate = new Date();
      maxExpDate.setDate(maxExpDate.getDate() + 365);
      if (expDate > maxExpDate) {
        newErrors.expires_at = 'Expiration date cannot be more than 365 days from now';
      }
    }

    if (!riskScore || riskScore < 1 || riskScore > 100) {
      newErrors.risk_score = 'Risk score must be between 1 and 100';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    const payload: ExceptionRequestPayload = {
      exception_type: exceptionType,
      justification: justification.trim(),
      risk_score: riskScore,
      expires_at: new Date(expiresAt).toISOString(),
      review_interval_days: reviewIntervalDays,
    };

    if (businessReason.trim()) payload.business_reason = businessReason.trim();
    if (compensatingControls.trim()) payload.compensating_controls = compensatingControls.trim();
    if (remediationId.trim()) payload.remediation_id = remediationId.trim();
    if (dataAssetId.trim()) payload.data_asset_id = dataAssetId.trim();
    if (policyId.trim()) payload.policy_id = policyId.trim();

    onSubmit(payload);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Request Risk Exception</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Exception Type */}
          <div>
            <Label>Exception Type</Label>
            <Select value={exceptionType} onValueChange={(v) => setExceptionType(v as DSPMExceptionType)}>
              <SelectTrigger className="mt-1">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {EXCEPTION_TYPES.map((t) => (
                  <SelectItem key={t.value} value={t.value}>{t.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Justification */}
          <div>
            <Label htmlFor="exc-justification">
              Justification <span className="text-destructive">*</span>
            </Label>
            <Textarea
              id="exc-justification"
              className="mt-1"
              placeholder="Explain why this exception is needed (min 20 characters)..."
              value={justification}
              onChange={(e) => setJustification(e.target.value)}
              rows={3}
            />
            {errors.justification && (
              <p className="mt-1 text-xs text-destructive">{errors.justification}</p>
            )}
          </div>

          {/* Business Reason */}
          <div>
            <Label htmlFor="exc-business-reason">Business Reason</Label>
            <Textarea
              id="exc-business-reason"
              className="mt-1"
              placeholder="Business impact or rationale..."
              value={businessReason}
              onChange={(e) => setBusinessReason(e.target.value)}
              rows={2}
            />
          </div>

          {/* Compensating Controls */}
          <div>
            <Label htmlFor="exc-compensating">Compensating Controls</Label>
            <Textarea
              id="exc-compensating"
              className="mt-1"
              placeholder="Describe any compensating controls in place..."
              value={compensatingControls}
              onChange={(e) => setCompensatingControls(e.target.value)}
              rows={2}
            />
          </div>

          {/* Risk Score and Dates */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <Label htmlFor="exc-risk-score">
                Risk Score (1-100) <span className="text-destructive">*</span>
              </Label>
              <Input
                id="exc-risk-score"
                type="number"
                min={1}
                max={100}
                className="mt-1"
                value={riskScore}
                onChange={(e) => setRiskScore(Number(e.target.value))}
              />
              {errors.risk_score && (
                <p className="mt-1 text-xs text-destructive">{errors.risk_score}</p>
              )}
            </div>
            <div>
              <Label htmlFor="exc-expires">
                Expires At <span className="text-destructive">*</span>
              </Label>
              <Input
                id="exc-expires"
                type="date"
                className="mt-1"
                min={today}
                max={maxDate}
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
              />
              {errors.expires_at && (
                <p className="mt-1 text-xs text-destructive">{errors.expires_at}</p>
              )}
            </div>
          </div>

          {/* Review Interval */}
          <div>
            <Label>Review Interval</Label>
            <Select value={String(reviewIntervalDays)} onValueChange={(v) => setReviewIntervalDays(Number(v))}>
              <SelectTrigger className="mt-1">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {REVIEW_INTERVALS.map((r) => (
                  <SelectItem key={r.value} value={r.value}>{r.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Optional IDs */}
          <div className="space-y-3 rounded-lg border p-3">
            <p className="text-xs font-medium text-muted-foreground">Optional References</p>
            <div>
              <Label htmlFor="exc-remediation-id" className="text-xs">Remediation ID</Label>
              <Input
                id="exc-remediation-id"
                className="mt-1"
                placeholder="Linked remediation ID"
                value={remediationId}
                onChange={(e) => setRemediationId(e.target.value)}
              />
            </div>
            <div>
              <Label htmlFor="exc-asset-id" className="text-xs">Data Asset ID</Label>
              <Input
                id="exc-asset-id"
                className="mt-1"
                placeholder="Linked data asset ID"
                value={dataAssetId}
                onChange={(e) => setDataAssetId(e.target.value)}
              />
            </div>
            <div>
              <Label htmlFor="exc-policy-id" className="text-xs">Policy ID</Label>
              <Input
                id="exc-policy-id"
                className="mt-1"
                placeholder="Linked policy ID"
                value={policyId}
                onChange={(e) => setPolicyId(e.target.value)}
              />
            </div>
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end gap-3 pt-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
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
