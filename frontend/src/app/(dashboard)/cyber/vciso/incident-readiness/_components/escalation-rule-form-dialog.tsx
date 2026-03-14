'use client';

import { useState, useEffect } from 'react';
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
import { Checkbox } from '@/components/ui/checkbox';
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
  VCISOEscalationRule,
  EscalationTriggerType,
  EscalationTarget,
} from '@/types/cyber';

interface EscalationRuleFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
  editRule?: VCISOEscalationRule | null;
}

const TRIGGER_TYPE_OPTIONS: { label: string; value: EscalationTriggerType }[] = [
  { label: 'Severity', value: 'severity' },
  { label: 'Time', value: 'time' },
  { label: 'Count', value: 'count' },
  { label: 'Custom', value: 'custom' },
];

const TARGET_OPTIONS: { label: string; value: EscalationTarget }[] = [
  { label: 'Management', value: 'management' },
  { label: 'Legal', value: 'legal' },
  { label: 'Regulator', value: 'regulator' },
  { label: 'Board', value: 'board' },
  { label: 'Custom', value: 'custom' },
];

const NOTIFICATION_CHANNELS = ['email', 'sms', 'slack', 'webhook'] as const;

interface FormState {
  name: string;
  description: string;
  trigger_type: EscalationTriggerType;
  trigger_condition: string;
  escalation_target: EscalationTarget;
  target_contacts: string;
  notification_channels: string[];
}

const initialFormState: FormState = {
  name: '',
  description: '',
  trigger_type: 'severity',
  trigger_condition: '',
  escalation_target: 'management',
  target_contacts: '',
  notification_channels: ['email'],
};

function formStateFromRule(rule: VCISOEscalationRule): FormState {
  return {
    name: rule.name,
    description: rule.description,
    trigger_type: rule.trigger_type,
    trigger_condition: rule.trigger_condition,
    escalation_target: rule.escalation_target,
    target_contacts: rule.target_contacts.join(', '),
    notification_channels: [...rule.notification_channels],
  };
}

export function EscalationRuleFormDialog({
  open,
  onOpenChange,
  onSaved,
  editRule,
}: EscalationRuleFormDialogProps) {
  const [form, setForm] = useState<FormState>(initialFormState);
  const isEditing = !!editRule;

  useEffect(() => {
    if (open && editRule) {
      setForm(formStateFromRule(editRule));
    } else if (open && !editRule) {
      setForm(initialFormState);
    }
  }, [open, editRule]);

  const createMutation = useApiMutation<VCISOEscalationRule, Record<string, unknown>>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_ESCALATION_RULES,
    {
      successMessage: 'Escalation rule created successfully',
      invalidateKeys: ['vciso-escalation-rules'],
      onSuccess: () => {
        setForm(initialFormState);
        onOpenChange(false);
        onSaved();
      },
    },
  );

  const updateMutation = useApiMutation<VCISOEscalationRule, Record<string, unknown>>(
    'put',
    editRule ? `${API_ENDPOINTS.CYBER_VCISO_ESCALATION_RULES}/${editRule.id}` : '',
    {
      successMessage: 'Escalation rule updated successfully',
      invalidateKeys: ['vciso-escalation-rules'],
      onSuccess: () => {
        setForm(initialFormState);
        onOpenChange(false);
        onSaved();
      },
    },
  );

  const handleSubmit = () => {
    if (!form.name.trim()) {
      toast.error('Name is required');
      return;
    }
    if (!form.trigger_condition.trim()) {
      toast.error('Trigger condition is required');
      return;
    }
    if (form.notification_channels.length === 0) {
      toast.error('At least one notification channel is required');
      return;
    }

    const payload = {
      name: form.name.trim(),
      description: form.description.trim(),
      trigger_type: form.trigger_type,
      trigger_condition: form.trigger_condition.trim(),
      escalation_target: form.escalation_target,
      target_contacts: form.target_contacts
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      notification_channels: form.notification_channels,
    };

    if (isEditing) {
      updateMutation.mutate(payload);
    } else {
      createMutation.mutate(payload);
    }
  };

  const handleChannelToggle = (channel: string, checked: boolean) => {
    setForm((f) => ({
      ...f,
      notification_channels: checked
        ? [...f.notification_channels, channel]
        : f.notification_channels.filter((c) => c !== channel),
    }));
  };

  const handleOpenChange = (o: boolean) => {
    if (!o) {
      setForm(initialFormState);
    }
    onOpenChange(o);
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit Escalation Rule' : 'Add Escalation Rule'}</DialogTitle>
          <DialogDescription>
            {isEditing
              ? 'Update the escalation rule configuration.'
              : 'Create a new escalation rule to define when and how incidents are escalated.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Basic info */}
          <div className="space-y-2">
            <Label htmlFor="rule-name">
              Name <span className="text-destructive">*</span>
            </Label>
            <Input
              id="rule-name"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="e.g. Critical Alert Escalation"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="rule-description">Description</Label>
            <Textarea
              id="rule-description"
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              placeholder="Describe when and why this escalation rule triggers"
              rows={3}
            />
          </div>

          <Separator />

          {/* Trigger configuration */}
          <h4 className="text-sm font-semibold text-muted-foreground">Trigger Configuration</h4>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Trigger Type</Label>
              <Select
                value={form.trigger_type}
                onValueChange={(v) =>
                  setForm((f) => ({ ...f, trigger_type: v as EscalationTriggerType }))
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TRIGGER_TYPE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Escalation Target</Label>
              <Select
                value={form.escalation_target}
                onValueChange={(v) =>
                  setForm((f) => ({ ...f, escalation_target: v as EscalationTarget }))
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TARGET_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="rule-condition">
              Trigger Condition <span className="text-destructive">*</span>
            </Label>
            <Input
              id="rule-condition"
              value={form.trigger_condition}
              onChange={(e) => setForm((f) => ({ ...f, trigger_condition: e.target.value }))}
              placeholder="e.g. severity >= critical AND unresolved > 30m"
            />
            <p className="text-xs text-muted-foreground">
              Define the condition that triggers this escalation rule.
            </p>
          </div>

          <Separator />

          {/* Contact & Notification */}
          <h4 className="text-sm font-semibold text-muted-foreground">
            Contacts & Notifications
          </h4>

          <div className="space-y-2">
            <Label htmlFor="rule-contacts">Target Contacts (comma-separated)</Label>
            <Input
              id="rule-contacts"
              value={form.target_contacts}
              onChange={(e) => setForm((f) => ({ ...f, target_contacts: e.target.value }))}
              placeholder="ciso@company.com, security-team@company.com"
            />
          </div>

          <div className="space-y-2">
            <Label>
              Notification Channels <span className="text-destructive">*</span>
            </Label>
            <div className="flex flex-wrap gap-4 pt-1">
              {NOTIFICATION_CHANNELS.map((channel) => (
                <label
                  key={channel}
                  className="flex items-center gap-2 text-sm cursor-pointer"
                >
                  <Checkbox
                    checked={form.notification_channels.includes(channel)}
                    onCheckedChange={(checked) =>
                      handleChannelToggle(channel, checked === true)
                    }
                  />
                  <span className="capitalize">{channel}</span>
                </label>
              ))}
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-4">
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isPending}>
            {isPending
              ? isEditing
                ? 'Saving...'
                : 'Creating...'
              : isEditing
                ? 'Save Changes'
                : 'Create Rule'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
