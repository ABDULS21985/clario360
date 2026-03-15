'use client';

import { useEffect, useState } from 'react';
import { AlertCircle } from 'lucide-react';
import { showApiError, showSuccess } from '@/lib/toast';
import { enterpriseApi } from '@/lib/enterprise';
import type {
  AIArtifactType,
  AICreateVersionPayload,
  AIExplainabilityType,
  AIRegisteredModel,
  JsonValue,
} from '@/types/ai-governance';
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

interface VersionFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  model: AIRegisteredModel | null;
  onSaved: () => void;
}

interface VersionFormState {
  description: string;
  artifactType: AIArtifactType;
  artifactConfig: string;
  explainabilityType: AIExplainabilityType;
  explanationTemplate: string;
  trainingDataDesc: string;
  trainingDataHash: string;
  trainingMetrics: string;
}

const EMPTY_STATE: VersionFormState = {
  description: '',
  artifactType: 'serialized_model',
  artifactConfig: '{}',
  explainabilityType: 'feature_importance',
  explanationTemplate: '',
  trainingDataDesc: '',
  trainingDataHash: '',
  trainingMetrics: '{}',
};

const ARTIFACT_TYPES: AIArtifactType[] = [
  'go_function',
  'rule_set',
  'statistical_config',
  'template_config',
  'serialized_model',
  'gguf_model',
  'bitnet_model',
  'onnx_model',
];

const EXPLAINABILITY_TYPES: AIExplainabilityType[] = [
  'rule_trace',
  'feature_importance',
  'statistical_deviation',
  'template_based',
  'reasoning_trace',
];

export function VersionFormDialog({ open, onOpenChange, model, onSaved }: VersionFormDialogProps) {
  const [state, setState] = useState<VersionFormState>(EMPTY_STATE);
  const [saving, setSaving] = useState(false);
  const [jsonError, setJsonError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setState(EMPTY_STATE);
      setJsonError(null);
    }
  }, [open]);

  const submit = async () => {
    if (!model) {
      return;
    }

    const parsedArtifactConfig = parseJsonObject(state.artifactConfig, 'Artifact config');
    if (!parsedArtifactConfig.ok) {
      setJsonError(parsedArtifactConfig.error);
      return;
    }

    const parsedTrainingMetrics = parseJsonObject(state.trainingMetrics, 'Training metrics');
    if (!parsedTrainingMetrics.ok) {
      setJsonError(parsedTrainingMetrics.error);
      return;
    }

    const payload: AICreateVersionPayload = {
      description: state.description.trim(),
      artifact_type: state.artifactType,
      artifact_config: parsedArtifactConfig.value,
      explainability_type: state.explainabilityType,
      training_metrics: parsedTrainingMetrics.value,
    };

    if (state.explanationTemplate.trim()) {
      payload.explanation_template = state.explanationTemplate.trim();
    }
    if (state.trainingDataDesc.trim()) {
      payload.training_data_desc = state.trainingDataDesc.trim();
    }
    if (state.trainingDataHash.trim()) {
      payload.training_data_hash = state.trainingDataHash.trim();
    }

    setJsonError(null);
    setSaving(true);
    try {
      const version = await enterpriseApi.ai.createVersion(model.id, payload);
      showSuccess('Version created.', `${model.slug} v${version.version_number} is now ready for validation and promotion.`);
      onOpenChange(false);
      onSaved();
    } catch (error) {
      showApiError(error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-4xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Model Version</DialogTitle>
          <DialogDescription>
            Register the next governed version for {model?.name ?? 'this model'} so it can move through validation, shadow, and lifecycle review.
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

          <div className="space-y-2">
            <Label htmlFor="version-description">Version Description</Label>
            <Textarea
              id="version-description"
              value={state.description}
              onChange={(event) => setState((current) => ({ ...current, description: event.target.value }))}
              placeholder="What changed in this model version and why it should be evaluated."
              className="min-h-24"
            />
          </div>

          <section className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label>Artifact Type</Label>
              <Select
                value={state.artifactType}
                onValueChange={(value) => setState((current) => ({ ...current, artifactType: value as AIArtifactType }))}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select artifact type" />
                </SelectTrigger>
                <SelectContent>
                  {ARTIFACT_TYPES.map((type) => (
                    <SelectItem key={type} value={type}>
                      {humanizeEnum(type)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Explainability Type</Label>
              <Select
                value={state.explainabilityType}
                onValueChange={(value) =>
                  setState((current) => ({ ...current, explainabilityType: value as AIExplainabilityType }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select explainability type" />
                </SelectTrigger>
                <SelectContent>
                  {EXPLAINABILITY_TYPES.map((type) => (
                    <SelectItem key={type} value={type}>
                      {humanizeEnum(type)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </section>

          <div className="space-y-2">
            <Label htmlFor="artifact-config">Artifact Config (JSON)</Label>
            <Textarea
              id="artifact-config"
              value={state.artifactConfig}
              onChange={(event) => setState((current) => ({ ...current, artifactConfig: event.target.value }))}
              className="min-h-44 font-mono text-xs"
              placeholder='{"model_uri":"s3://models/score.pkl","threshold":0.72}'
            />
          </div>

          <section className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="explanation-template">Explanation Template</Label>
              <Textarea
                id="explanation-template"
                value={state.explanationTemplate}
                onChange={(event) => setState((current) => ({ ...current, explanationTemplate: event.target.value }))}
                placeholder="Optional human-readable explanation template."
                className="min-h-24"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="training-data-desc">Training Data Description</Label>
              <Textarea
                id="training-data-desc"
                value={state.trainingDataDesc}
                onChange={(event) => setState((current) => ({ ...current, trainingDataDesc: event.target.value }))}
                placeholder="Dataset provenance, scope, and labeling notes."
                className="min-h-24"
              />
            </div>
          </section>

          <section className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="training-data-hash">Training Data Hash</Label>
              <Input
                id="training-data-hash"
                value={state.trainingDataHash}
                onChange={(event) => setState((current) => ({ ...current, trainingDataHash: event.target.value }))}
                placeholder="sha256:..."
              />
            </div>
          </section>

          <div className="space-y-2">
            <Label htmlFor="training-metrics">Training Metrics (JSON)</Label>
            <Textarea
              id="training-metrics"
              value={state.trainingMetrics}
              onChange={(event) => setState((current) => ({ ...current, trainingMetrics: event.target.value }))}
              className="min-h-36 font-mono text-xs"
              placeholder='{"accuracy":0.94,"auc":0.97,"precision":0.92,"recall":0.9}'
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={() => void submit()} disabled={saving || !state.description.trim()}>
            {saving ? 'Creating…' : 'Create Version'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function parseJsonObject(raw: string, label: string): { ok: true; value: JsonValue } | { ok: false; error: string } {
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
