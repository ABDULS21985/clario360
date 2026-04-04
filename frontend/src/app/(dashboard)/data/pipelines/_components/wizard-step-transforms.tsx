'use client';

import { AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { JsonValue } from '@/lib/data-suite';
import type { PipelineTransformDraft } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';
import { validateTransform } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-utils';
import { TransformList } from '@/app/(dashboard)/data/pipelines/_components/transform-builder/transform-list';

interface WizardStepTransformsProps {
  transforms: PipelineTransformDraft[];
  availableColumns: string[];
  previewBeforeRows: Array<Record<string, JsonValue>>;
  previewAfterRows: Array<Record<string, JsonValue>>;
  previewError: string | null;
  onBack: () => void;
  onChange: (transforms: PipelineTransformDraft[]) => void;
  onPreview: () => void;
  onContinue: () => void;
}

export function WizardStepTransforms({
  transforms,
  availableColumns,
  previewBeforeRows,
  previewAfterRows,
  previewError,
  onBack,
  onChange,
  onPreview,
  onContinue,
}: WizardStepTransformsProps) {
  const invalidTransforms = transforms
    .map((transform) => validateTransform(transform))
    .filter((message): message is string => Boolean(message));

  return (
    <div className="space-y-6">
      <div className="rounded-xl border bg-muted/10 p-4">
        <div className="font-medium">Transformation builder</div>
        <div className="mt-1 text-sm text-muted-foreground">
          Order matters. Transformations run sequentially against the selected source rows.
        </div>
      </div>

      <TransformList
        transforms={transforms}
        availableColumns={availableColumns}
        previewBeforeRows={previewBeforeRows}
        previewAfterRows={previewAfterRows}
        previewError={previewError}
        onChange={onChange}
        onPreview={onPreview}
      />

      {invalidTransforms.length > 0 ? (
        <div className="rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-800">
          <div className="mb-2 flex items-center gap-2 font-medium">
            <AlertTriangle className="h-4 w-4" />
            Resolve transform issues before continuing
          </div>
          <ul className="list-disc pl-5">
            {invalidTransforms.map((message) => (
              <li key={message}>{message}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="flex justify-between">
        <Button type="button" variant="outline" onClick={onBack}>
          Back
        </Button>
        <Button type="button" onClick={onContinue} disabled={invalidTransforms.length > 0}>
          Continue
        </Button>
      </div>
    </div>
  );
}

