'use client';

import { useState } from 'react';
import { toast } from 'sonner';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type {
  VCISORiskEntry,
  RiskLikelihood,
  RiskImpact,
  RiskStatus,
  RiskTreatment,
} from '@/types/cyber';

interface RiskFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
}

const LIKELIHOOD_OPTIONS: { label: string; value: RiskLikelihood }[] = [
  { label: 'Rare', value: 'rare' },
  { label: 'Unlikely', value: 'unlikely' },
  { label: 'Possible', value: 'possible' },
  { label: 'Likely', value: 'likely' },
  { label: 'Almost Certain', value: 'almost_certain' },
];

const IMPACT_OPTIONS: { label: string; value: RiskImpact }[] = [
  { label: 'Negligible', value: 'negligible' },
  { label: 'Minor', value: 'minor' },
  { label: 'Moderate', value: 'moderate' },
  { label: 'Major', value: 'major' },
  { label: 'Catastrophic', value: 'catastrophic' },
];

const STATUS_OPTIONS: { label: string; value: RiskStatus }[] = [
  { label: 'Identified', value: 'identified' },
  { label: 'Assessed', value: 'assessed' },
  { label: 'Mitigating', value: 'mitigating' },
];

const TREATMENT_OPTIONS: { label: string; value: RiskTreatment }[] = [
  { label: 'Mitigate', value: 'mitigate' },
  { label: 'Transfer', value: 'transfer' },
  { label: 'Accept', value: 'accept' },
  { label: 'Avoid', value: 'avoid' },
];

interface FormState {
  title: string;
  description: string;
  category: string;
  likelihood: RiskLikelihood;
  impact: RiskImpact;
  status: RiskStatus;
  treatment: RiskTreatment;
  department: string;
  business_services: string;
  treatment_plan: string;
  controls: string;
  review_date: string;
}

const initialFormState: FormState = {
  title: '',
  description: '',
  category: '',
  likelihood: 'possible',
  impact: 'moderate',
  status: 'identified',
  treatment: 'mitigate',
  department: '',
  business_services: '',
  treatment_plan: '',
  controls: '',
  review_date: '',
};

export function RiskFormDialog({ open, onOpenChange, onCreated }: RiskFormDialogProps) {
  const [form, setForm] = useState<FormState>(initialFormState);

  const createMutation = useApiMutation<VCISORiskEntry, Record<string, unknown>>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_RISKS,
    {
      successMessage: 'Risk created successfully',
      invalidateKeys: ['vciso-risks', API_ENDPOINTS.CYBER_VCISO_RISKS_STATS],
      onSuccess: () => {
        setForm(initialFormState);
        onOpenChange(false);
        onCreated();
      },
    },
  );

  const handleSubmit = () => {
    if (!form.title.trim()) {
      toast.error('Title is required');
      return;
    }
    if (!form.category.trim()) {
      toast.error('Category is required');
      return;
    }
    if (!form.description.trim()) {
      toast.error('Description is required');
      return;
    }

    createMutation.mutate({
      title: form.title.trim(),
      description: form.description.trim(),
      category: form.category.trim(),
      likelihood: form.likelihood,
      impact: form.impact,
      status: form.status,
      treatment: form.treatment,
      department: form.department.trim(),
      inherent_score: 0,
      residual_score: 0,
      owner_name: '',
      business_services: form.business_services
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      treatment_plan: form.treatment_plan.trim(),
      controls: form.controls
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      tags: [],
      review_date: form.review_date || undefined,
    });
  };

  const handleOpenChange = (o: boolean) => {
    if (!o) {
      setForm(initialFormState);
    }
    onOpenChange(o);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Add New Risk</DialogTitle>
          <DialogDescription>
            Create a new risk entry in the register. All required fields are marked.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Basic info */}
          <div className="space-y-2">
            <Label htmlFor="risk-title">
              Title <span className="text-destructive">*</span>
            </Label>
            <Input
              id="risk-title"
              value={form.title}
              onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
              placeholder="e.g. Unauthorized access to production database"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="risk-description">
              Description <span className="text-destructive">*</span>
            </Label>
            <Textarea
              id="risk-description"
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              placeholder="Detailed description of the risk, including context and potential consequences"
              rows={3}
            />
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="risk-category">
                Category <span className="text-destructive">*</span>
              </Label>
              <Input
                id="risk-category"
                value={form.category}
                onChange={(e) => setForm((f) => ({ ...f, category: e.target.value }))}
                placeholder="e.g. Operational, Compliance"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="risk-department">Department</Label>
              <Input
                id="risk-department"
                value={form.department}
                onChange={(e) => setForm((f) => ({ ...f, department: e.target.value }))}
                placeholder="e.g. Engineering"
              />
            </div>
          </div>

          <Separator />

          {/* Assessment */}
          <h4 className="text-sm font-semibold text-muted-foreground">Risk Assessment</h4>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Likelihood</Label>
              <Select
                value={form.likelihood}
                onValueChange={(v) => setForm((f) => ({ ...f, likelihood: v as RiskLikelihood }))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {LIKELIHOOD_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Impact</Label>
              <Select
                value={form.impact}
                onValueChange={(v) => setForm((f) => ({ ...f, impact: v as RiskImpact }))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {IMPACT_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Status</Label>
              <Select
                value={form.status}
                onValueChange={(v) => setForm((f) => ({ ...f, status: v as RiskStatus }))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {STATUS_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Treatment</Label>
              <Select
                value={form.treatment}
                onValueChange={(v) => setForm((f) => ({ ...f, treatment: v as RiskTreatment }))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TREATMENT_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <Separator />

          {/* Mitigation */}
          <h4 className="text-sm font-semibold text-muted-foreground">Mitigation Details</h4>

          <div className="space-y-2">
            <Label htmlFor="risk-review-date">Review Date</Label>
            <Input
              id="risk-review-date"
              type="date"
              value={form.review_date}
              onChange={(e) => setForm((f) => ({ ...f, review_date: e.target.value }))}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="risk-treatment-plan">Treatment Plan</Label>
            <Textarea
              id="risk-treatment-plan"
              value={form.treatment_plan}
              onChange={(e) => setForm((f) => ({ ...f, treatment_plan: e.target.value }))}
              placeholder="Describe the planned mitigation steps"
              rows={3}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="risk-controls">Controls (comma-separated)</Label>
            <Input
              id="risk-controls"
              value={form.controls}
              onChange={(e) => setForm((f) => ({ ...f, controls: e.target.value }))}
              placeholder="AC-1, AC-2, SC-7"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="risk-services">Business Services (comma-separated)</Label>
            <Input
              id="risk-services"
              value={form.business_services}
              onChange={(e) => setForm((f) => ({ ...f, business_services: e.target.value }))}
              placeholder="Payment Processing, Customer Portal"
            />
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-4">
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={createMutation.isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={createMutation.isPending}>
            {createMutation.isPending ? 'Creating...' : 'Create Risk'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
