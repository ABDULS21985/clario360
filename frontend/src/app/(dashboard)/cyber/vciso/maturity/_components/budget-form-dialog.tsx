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
import type { VCISOBudgetItem, BudgetItemType } from '@/types/cyber';

interface BudgetFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
}

interface FormState {
  title: string;
  category: string;
  type: BudgetItemType;
  amount: string;
  currency: string;
  priority: string;
  justification: string;
  fiscal_year: string;
  quarter: string;
  linked_risk_ids: string;
  linked_recommendation_ids: string;
}

const initialFormState: FormState = {
  title: '',
  category: '',
  type: 'opex',
  amount: '',
  currency: 'USD',
  priority: '3',
  justification: '',
  fiscal_year: new Date().getFullYear().toString(),
  quarter: '',
  linked_risk_ids: '',
  linked_recommendation_ids: '',
};

const TYPE_OPTIONS: { label: string; value: BudgetItemType }[] = [
  { label: 'Capital Expenditure (CapEx)', value: 'capex' },
  { label: 'Operating Expenditure (OpEx)', value: 'opex' },
];

const QUARTER_OPTIONS = [
  { label: 'Not specified', value: '' },
  { label: 'Q1', value: 'Q1' },
  { label: 'Q2', value: 'Q2' },
  { label: 'Q3', value: 'Q3' },
  { label: 'Q4', value: 'Q4' },
];

const CURRENCY_OPTIONS = ['USD', 'EUR', 'GBP', 'CAD', 'AUD', 'JPY'];

const CATEGORY_OPTIONS = [
  'Identity & Access Management',
  'Endpoint Security',
  'Network Security',
  'Cloud Security',
  'Security Operations',
  'Compliance & Governance',
  'Training & Awareness',
  'Incident Response',
  'Data Protection',
  'Application Security',
  'Third-Party Risk',
  'Other',
];

export function BudgetFormDialog({
  open,
  onOpenChange,
  onCreated,
}: BudgetFormDialogProps) {
  const [form, setForm] = useState<FormState>(initialFormState);

  const createMutation = useApiMutation<VCISOBudgetItem, Record<string, unknown>>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_BUDGET,
    {
      successMessage: 'Budget item created successfully',
      invalidateKeys: [
        'vciso-budget',
        API_ENDPOINTS.CYBER_VCISO_BUDGET_SUMMARY,
      ],
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
    if (!form.amount || Number(form.amount) <= 0) {
      toast.error('A valid amount is required');
      return;
    }
    if (!form.justification.trim()) {
      toast.error('Justification is required');
      return;
    }

    createMutation.mutate({
      title: form.title.trim(),
      category: form.category.trim(),
      type: form.type,
      amount: Number(form.amount),
      currency: form.currency,
      priority: Number(form.priority),
      justification: form.justification.trim(),
      fiscal_year: form.fiscal_year.trim(),
      quarter: form.quarter || undefined,
      linked_risk_ids: form.linked_risk_ids
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      linked_recommendation_ids: form.linked_recommendation_ids
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      status: 'proposed',
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
          <DialogTitle>Add Budget Item</DialogTitle>
          <DialogDescription>
            Create a new security budget item for investment prioritization. Required fields are marked.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Basic Info */}
          <div className="space-y-2">
            <Label htmlFor="budget-title">
              Title <span className="text-destructive">*</span>
            </Label>
            <Input
              id="budget-title"
              value={form.title}
              onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
              placeholder="e.g. SIEM Platform Upgrade"
            />
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>
                Category <span className="text-destructive">*</span>
              </Label>
              <Select
                value={form.category}
                onValueChange={(v) => setForm((f) => ({ ...f, category: v }))}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select category" />
                </SelectTrigger>
                <SelectContent>
                  {CATEGORY_OPTIONS.map((cat) => (
                    <SelectItem key={cat} value={cat}>
                      {cat}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Type</Label>
              <Select
                value={form.type}
                onValueChange={(v) =>
                  setForm((f) => ({ ...f, type: v as BudgetItemType }))
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TYPE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <div className="space-y-2">
              <Label htmlFor="budget-amount">
                Amount <span className="text-destructive">*</span>
              </Label>
              <Input
                id="budget-amount"
                type="number"
                min="0"
                step="0.01"
                value={form.amount}
                onChange={(e) =>
                  setForm((f) => ({ ...f, amount: e.target.value }))
                }
                placeholder="50000"
              />
            </div>
            <div className="space-y-2">
              <Label>Currency</Label>
              <Select
                value={form.currency}
                onValueChange={(v) => setForm((f) => ({ ...f, currency: v }))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {CURRENCY_OPTIONS.map((c) => (
                    <SelectItem key={c} value={c}>
                      {c}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="budget-priority">Priority (1-5)</Label>
              <Input
                id="budget-priority"
                type="number"
                min="1"
                max="5"
                value={form.priority}
                onChange={(e) =>
                  setForm((f) => ({ ...f, priority: e.target.value }))
                }
              />
            </div>
          </div>

          <Separator />

          {/* Timeline */}
          <h4 className="text-sm font-semibold text-muted-foreground">
            Timeline
          </h4>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="budget-fiscal-year">Fiscal Year</Label>
              <Input
                id="budget-fiscal-year"
                value={form.fiscal_year}
                onChange={(e) =>
                  setForm((f) => ({ ...f, fiscal_year: e.target.value }))
                }
                placeholder="2026"
              />
            </div>
            <div className="space-y-2">
              <Label>Quarter</Label>
              <Select
                value={form.quarter}
                onValueChange={(v) => setForm((f) => ({ ...f, quarter: v }))}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select quarter" />
                </SelectTrigger>
                <SelectContent>
                  {QUARTER_OPTIONS.map((q) => (
                    <SelectItem key={q.value || 'none'} value={q.value || 'none'}>
                      {q.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <Separator />

          {/* Justification */}
          <h4 className="text-sm font-semibold text-muted-foreground">
            Business Case
          </h4>

          <div className="space-y-2">
            <Label htmlFor="budget-justification">
              Justification <span className="text-destructive">*</span>
            </Label>
            <Textarea
              id="budget-justification"
              value={form.justification}
              onChange={(e) =>
                setForm((f) => ({ ...f, justification: e.target.value }))
              }
              placeholder="Provide a business justification for this investment, including expected outcomes and risk reduction impact"
              rows={4}
            />
          </div>

          <Separator />

          {/* Linked Entities */}
          <h4 className="text-sm font-semibold text-muted-foreground">
            Linked Entities
          </h4>

          <div className="space-y-2">
            <Label htmlFor="budget-risk-ids">
              Linked Risk IDs (comma-separated)
            </Label>
            <Input
              id="budget-risk-ids"
              value={form.linked_risk_ids}
              onChange={(e) =>
                setForm((f) => ({ ...f, linked_risk_ids: e.target.value }))
              }
              placeholder="risk-001, risk-002"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="budget-rec-ids">
              Linked Recommendation IDs (comma-separated)
            </Label>
            <Input
              id="budget-rec-ids"
              value={form.linked_recommendation_ids}
              onChange={(e) =>
                setForm((f) => ({
                  ...f,
                  linked_recommendation_ids: e.target.value,
                }))
              }
              placeholder="rec-001, rec-002"
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
            {createMutation.isPending ? 'Creating...' : 'Create Budget Item'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
