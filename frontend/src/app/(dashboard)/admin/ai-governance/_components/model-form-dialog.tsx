'use client';

import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { AlertCircle } from 'lucide-react';
import { showApiError, showSuccess } from '@/lib/toast';
import { enterpriseApi } from '@/lib/enterprise';
import type {
  AIModelStatus,
  AIModelSuite,
  AIModelType,
  AIRiskTier,
  AIRegisteredModel,
  AIRegisterModelPayload,
  AIUpdateModelPayload,
  JsonValue,
} from '@/types/ai-governance';
import type { UserDirectoryEntry } from '@/types/suites';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';

interface ModelFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  model?: AIRegisteredModel | null;
  onSaved: () => void;
}

interface ModelFormState {
  name: string;
  slug: string;
  description: string;
  modelType: AIModelType;
  suite: AIModelSuite;
  riskTier: AIRiskTier;
  status: AIModelStatus;
  ownerUserId: string;
  ownerTeam: string;
  tags: string;
  metadata: string;
}

const MODEL_TYPES: AIModelType[] = [
  'rule_based',
  'statistical',
  'ml_classifier',
  'ml_regressor',
  'nlp_extractor',
  'anomaly_detector',
  'scorer',
  'recommender',
  'llm_agentic',
];

const MODEL_SUITES: AIModelSuite[] = ['cyber', 'data', 'acta', 'lex', 'visus', 'platform'];
const RISK_TIERS: AIRiskTier[] = ['low', 'medium', 'high', 'critical'];
const MODEL_STATUSES: AIModelStatus[] = ['active', 'deprecated', 'retired'];

const EMPTY_STATE: ModelFormState = {
  name: '',
  slug: '',
  description: '',
  modelType: 'rule_based',
  suite: 'cyber',
  riskTier: 'medium',
  status: 'active',
  ownerUserId: 'unassigned',
  ownerTeam: '',
  tags: '',
  metadata: '{}',
};

export function ModelFormDialog({ open, onOpenChange, model, onSaved }: ModelFormDialogProps) {
  const [state, setState] = useState<ModelFormState>(EMPTY_STATE);
  const [saving, setSaving] = useState(false);
  const [jsonError, setJsonError] = useState<string | null>(null);

  const usersQuery = useQuery({
    queryKey: ['ai-governance-users'],
    queryFn: () =>
      enterpriseApi.users.list({
        page: 1,
        per_page: 100,
        sort: 'email',
        order: 'asc',
      }),
    enabled: open,
  });

  useEffect(() => {
    if (!open) {
      return;
    }

    if (!model) {
      setState(EMPTY_STATE);
      setJsonError(null);
      return;
    }

    setState({
      name: model.name,
      slug: model.slug,
      description: model.description,
      modelType: model.model_type,
      suite: model.suite,
      riskTier: model.risk_tier,
      status: model.status,
      ownerUserId: model.owner_user_id ?? 'unassigned',
      ownerTeam: model.owner_team ?? '',
      tags: model.tags.join(', '),
      metadata: JSON.stringify(model.metadata ?? {}, null, 2),
    });
    setJsonError(null);
  }, [open, model]);

  const users = usersQuery.data?.data ?? [];
  const ownerLabel = useMemo(
    () => users.find((item) => item.id === state.ownerUserId) ?? null,
    [users, state.ownerUserId],
  );

  const submit = async () => {
    const parsedMetadata = parseJson(state.metadata, 'Metadata');
    if (!parsedMetadata.ok) {
      setJsonError(parsedMetadata.error);
      return;
    }

    setJsonError(null);
    setSaving(true);
    try {
      if (model) {
        const payload: AIUpdateModelPayload = {
          name: state.name.trim(),
          description: state.description.trim(),
          owner_team: state.ownerTeam.trim(),
          risk_tier: state.riskTier,
          status: state.status,
          tags: parseTags(state.tags),
          metadata: parsedMetadata.value,
        };

        if (state.ownerUserId !== 'unassigned') {
          payload.owner_user_id = state.ownerUserId;
        }

        await enterpriseApi.ai.updateModel(model.id, payload);
        showSuccess('Model updated.', `${state.slug} metadata and governance settings were saved.`);
      } else {
        const payload: AIRegisterModelPayload = {
          name: state.name.trim(),
          slug: normalizeSlug(state.slug || state.name),
          description: state.description.trim(),
          model_type: state.modelType,
          suite: state.suite,
          risk_tier: state.riskTier,
          tags: parseTags(state.tags),
          metadata: parsedMetadata.value,
        };

        if (state.ownerUserId !== 'unassigned') {
          payload.owner_user_id = state.ownerUserId;
        }
        if (state.ownerTeam.trim()) {
          payload.owner_team = state.ownerTeam.trim();
        }

        await enterpriseApi.ai.createModel(payload);
        showSuccess('Model registered.', `${payload.slug} is now part of the governed registry.`);
      }

      onOpenChange(false);
      onSaved();
    } catch (error) {
      showApiError(error);
    } finally {
      setSaving(false);
    }
  };

  const canSubmit = state.name.trim() && (model || state.slug.trim() || state.name.trim());

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{model ? 'Edit Governed Model' : 'Register Model'}</DialogTitle>
          <DialogDescription>
            {model
              ? 'Update ownership, risk tier, tags, and registry metadata for this governed model.'
              : 'Add a backend-registered model to the AI governance frontend so lifecycle, shadow, drift, and validation controls become accessible.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {jsonError ? (
            <Alert variant="warning">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Invalid JSON</AlertTitle>
              <AlertDescription>{jsonError}</AlertDescription>
            </Alert>
          ) : null}

          <section className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="model-name">Name</Label>
              <Input
                id="model-name"
                value={state.name}
                onChange={(event) => {
                  const nextName = event.target.value;
                  setState((current) => ({
                    ...current,
                    name: nextName,
                    slug: model || current.slug ? current.slug : normalizeSlug(nextName),
                  }));
                }}
                placeholder="Threat scoring classifier"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="model-slug">Slug</Label>
              <Input
                id="model-slug"
                value={state.slug}
                disabled={Boolean(model)}
                onChange={(event) => setState((current) => ({ ...current, slug: normalizeSlug(event.target.value) }))}
                placeholder="threat-scoring-classifier"
              />
            </div>

            <div className="space-y-2">
              <Label>Suite</Label>
              <Select
                value={state.suite}
                disabled={Boolean(model)}
                onValueChange={(value) => setState((current) => ({ ...current, suite: value as AIModelSuite }))}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select suite" />
                </SelectTrigger>
                <SelectContent>
                  {MODEL_SUITES.map((suite) => (
                    <SelectItem key={suite} value={suite}>
                      {humanizeEnum(suite)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Model Type</Label>
              <Select
                value={state.modelType}
                disabled={Boolean(model)}
                onValueChange={(value) => setState((current) => ({ ...current, modelType: value as AIModelType }))}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select model type" />
                </SelectTrigger>
                <SelectContent>
                  {MODEL_TYPES.map((type) => (
                    <SelectItem key={type} value={type}>
                      {humanizeEnum(type)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Risk Tier</Label>
              <Select
                value={state.riskTier}
                onValueChange={(value) => setState((current) => ({ ...current, riskTier: value as AIRiskTier }))}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select risk tier" />
                </SelectTrigger>
                <SelectContent>
                  {RISK_TIERS.map((riskTier) => (
                    <SelectItem key={riskTier} value={riskTier}>
                      {humanizeEnum(riskTier)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {model ? (
              <div className="space-y-2">
                <Label>Status</Label>
                <Select
                  value={state.status}
                  onValueChange={(value) => setState((current) => ({ ...current, status: value as AIModelStatus }))}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select status" />
                  </SelectTrigger>
                  <SelectContent>
                    {MODEL_STATUSES.map((status) => (
                      <SelectItem key={status} value={status}>
                        {humanizeEnum(status)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : null}
          </section>

          <div className="space-y-2">
            <Label htmlFor="model-description">Description</Label>
            <Textarea
              id="model-description"
              value={state.description}
              onChange={(event) => setState((current) => ({ ...current, description: event.target.value }))}
              placeholder="Explain the decision domain, intended use, and governance expectations."
              className="min-h-24"
            />
          </div>

          <section className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label>Owner User</Label>
              <Select
                value={state.ownerUserId}
                onValueChange={(value) => setState((current) => ({ ...current, ownerUserId: value }))}
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={
                      ownerLabel
                        ? `${ownerLabel.first_name} ${ownerLabel.last_name}`.trim() || ownerLabel.email
                        : 'Unassigned'
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="unassigned">Unassigned</SelectItem>
                  {users.map((user: UserDirectoryEntry) => (
                    <SelectItem key={user.id} value={user.id}>
                      {`${user.first_name} ${user.last_name}`.trim() || user.email} ({user.email})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="model-owner-team">Owner Team</Label>
              <Input
                id="model-owner-team"
                value={state.ownerTeam}
                onChange={(event) => setState((current) => ({ ...current, ownerTeam: event.target.value }))}
                placeholder="Security Analytics"
              />
            </div>

            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="model-tags">Tags</Label>
              <Input
                id="model-tags"
                value={state.tags}
                onChange={(event) => setState((current) => ({ ...current, tags: event.target.value }))}
                placeholder="threat, production, explainable"
              />
            </div>
          </section>

          <div className="space-y-2">
            <Label htmlFor="model-metadata">Registry Metadata (JSON)</Label>
            <Textarea
              id="model-metadata"
              value={state.metadata}
              onChange={(event) => setState((current) => ({ ...current, metadata: event.target.value }))}
              className="min-h-40 font-mono text-xs"
              placeholder='{"owner":"security-analytics","business_impact":"high"}'
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={() => void submit()} disabled={saving || !canSubmit}>
            {saving ? (model ? 'Saving…' : 'Registering…') : model ? 'Save Changes' : 'Register Model'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function parseTags(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function normalizeSlug(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

function parseJson(raw: string, label: string): { ok: true; value: JsonValue } | { ok: false; error: string } {
  try {
    const parsed = JSON.parse(raw || '{}') as unknown;
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return { ok: false, error: `${label} must be a JSON object.` };
    }
    return { ok: true, value: parsed as JsonValue };
  } catch (error) {
    return { ok: false, error: `${label} is not valid JSON: ${(error as Error).message}` };
  }
}

function humanizeEnum(value: string): string {
  return value.replaceAll('_', ' ');
}
