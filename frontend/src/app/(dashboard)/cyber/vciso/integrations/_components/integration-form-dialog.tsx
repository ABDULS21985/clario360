'use client';

import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { VCISOIntegration, CyberIntegrationType } from '@/types/cyber';

// ─── Constants ───────────────────────────────────────────────────────────────

const INTEGRATION_TYPES: { label: string; value: CyberIntegrationType }[] = [
  { label: 'Asset Management', value: 'asset_management' },
  { label: 'Ticketing', value: 'ticketing' },
  { label: 'Cloud Security', value: 'cloud_security' },
  { label: 'Data Protection', value: 'data_protection' },
  { label: 'SIEM', value: 'siem' },
  { label: 'IAM', value: 'iam' },
];

const SYNC_FREQUENCIES: { label: string; value: string }[] = [
  { label: 'Every 5 minutes', value: 'every_5m' },
  { label: 'Every 15 minutes', value: 'every_15m' },
  { label: 'Every hour', value: 'every_hour' },
  { label: 'Every 6 hours', value: 'every_6h' },
  { label: 'Daily', value: 'daily' },
];

// ─── Props ───────────────────────────────────────────────────────────────────

interface IntegrationFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  integration?: VCISOIntegration | null;
}

// ─── Form State ──────────────────────────────────────────────────────────────

interface IntegrationFormData {
  name: string;
  type: CyberIntegrationType;
  provider: string;
  sync_frequency: string;
  config_json: string;
}

function getDefaultForm(integration?: VCISOIntegration | null): IntegrationFormData {
  if (integration) {
    return {
      name: integration.name,
      type: integration.type,
      provider: integration.provider,
      sync_frequency: integration.sync_frequency,
      config_json: JSON.stringify(integration.config, null, 2),
    };
  }
  return {
    name: '',
    type: 'asset_management',
    provider: '',
    sync_frequency: 'every_hour',
    config_json: '{}',
  };
}

function isValidJSON(str: string): boolean {
  try {
    JSON.parse(str);
    return true;
  } catch {
    return false;
  }
}

// ─── Component ───────────────────────────────────────────────────────────────

export function IntegrationFormDialog({
  open,
  onOpenChange,
  integration,
}: IntegrationFormDialogProps) {
  const isEdit = !!integration;
  const [form, setForm] = useState<IntegrationFormData>(() => getDefaultForm(integration));

  useEffect(() => {
    if (open) {
      setForm(getDefaultForm(integration));
    }
  }, [open, integration]);

  const { mutate: createIntegration, isPending: creating } = useApiMutation<
    VCISOIntegration,
    Record<string, unknown>
  >('post', API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS, {
    successMessage: 'Integration added successfully',
    invalidateKeys: [API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS],
    onSuccess: () => {
      onOpenChange(false);
    },
  });

  const { mutate: updateIntegration, isPending: updating } = useApiMutation<
    VCISOIntegration,
    Record<string, unknown>
  >(
    'put',
    () => `${API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS}/${integration?.id}`,
    {
      successMessage: 'Integration updated successfully',
      invalidateKeys: [API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS],
      onSuccess: () => {
        onOpenChange(false);
      },
    },
  );

  const isPending = creating || updating;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    let config: Record<string, unknown> = {};
    try {
      config = JSON.parse(form.config_json) as Record<string, unknown>;
    } catch {
      return;
    }

    const payload: Record<string, unknown> = {
      name: form.name.trim(),
      type: form.type,
      provider: form.provider.trim(),
      status: isEdit ? (integration?.status ?? 'pending') : 'pending',
      sync_frequency: form.sync_frequency,
      config,
    };

    if (isEdit) {
      updateIntegration(payload);
    } else {
      createIntegration(payload);
    }
  }

  function updateField<K extends keyof IntegrationFormData>(key: K, value: IntegrationFormData[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  const isValid =
    form.name.trim().length > 0 &&
    form.provider.trim().length > 0 &&
    isValidJSON(form.config_json);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit Integration' : 'Add Integration'}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Update the integration configuration.'
              : 'Connect a new external service to your security platform.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Name */}
          <div className="space-y-2">
            <Label htmlFor="integration-name">Name *</Label>
            <Input
              id="integration-name"
              placeholder="e.g., Jira Cloud, AWS Security Hub"
              value={form.name}
              onChange={(e) => updateField('name', e.target.value)}
              required
            />
          </div>

          {/* Type & Provider */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Type *</Label>
              <Select
                value={form.type}
                onValueChange={(v) => updateField('type', v as CyberIntegrationType)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select type" />
                </SelectTrigger>
                <SelectContent>
                  {INTEGRATION_TYPES.map((t) => (
                    <SelectItem key={t.value} value={t.value}>
                      {t.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="integration-provider">Provider *</Label>
              <Input
                id="integration-provider"
                placeholder="e.g., Atlassian, AWS, CrowdStrike"
                value={form.provider}
                onChange={(e) => updateField('provider', e.target.value)}
                required
              />
            </div>
          </div>

          {/* Sync Frequency */}
          <div className="space-y-2">
            <Label>Sync Frequency</Label>
            <Select
              value={form.sync_frequency}
              onValueChange={(v) => updateField('sync_frequency', v)}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select frequency" />
              </SelectTrigger>
              <SelectContent>
                {SYNC_FREQUENCIES.map((f) => (
                  <SelectItem key={f.value} value={f.value}>
                    {f.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Configuration JSON */}
          <div className="space-y-2">
            <Label htmlFor="integration-config">Configuration (JSON)</Label>
            <Textarea
              id="integration-config"
              placeholder='{"api_key": "...", "base_url": "https://..."}'
              value={form.config_json}
              onChange={(e) => updateField('config_json', e.target.value)}
              rows={5}
              className="font-mono text-xs"
            />
            {!isValidJSON(form.config_json) && form.config_json.trim().length > 0 && (
              <p className="text-xs text-destructive">Invalid JSON format</p>
            )}
            <p className="text-xs text-muted-foreground">
              Enter connection configuration as a JSON object. Sensitive values will be encrypted.
            </p>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending || !isValid}>
              {isPending
                ? isEdit
                  ? 'Saving...'
                  : 'Adding...'
                : isEdit
                  ? 'Save Changes'
                  : 'Add Integration'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
