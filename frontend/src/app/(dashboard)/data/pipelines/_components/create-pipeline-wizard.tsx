'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQueries, useQuery } from '@tanstack/react-query';
import { AlertTriangle } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { dataSuiteApi } from '@/lib/data-suite';
import { showApiError, showSuccess } from '@/lib/toast';
import {
  buildSampleRows,
  createInitialPipelineWizardState,
  describeSchedule,
  findTable,
  runPreview,
  serializePipelinePayload,
  tableColumnNames,
} from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-utils';
import type { DataModel, DataSource } from '@/lib/data-suite';
import type { PipelineScheduleValues, PipelineTargetValues } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';
import { WizardStepBasic } from '@/app/(dashboard)/data/pipelines/_components/wizard-step-basic';
import { WizardStepQuality } from '@/app/(dashboard)/data/pipelines/_components/wizard-step-quality';
import { WizardStepSchedule } from '@/app/(dashboard)/data/pipelines/_components/wizard-step-schedule';
import { WizardStepSource } from '@/app/(dashboard)/data/pipelines/_components/wizard-step-source';
import { WizardStepTarget } from '@/app/(dashboard)/data/pipelines/_components/wizard-step-target';
import { WizardStepTransforms } from '@/app/(dashboard)/data/pipelines/_components/wizard-step-transforms';

interface CreatePipelineWizardProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
}

const STEPS = ['Basic', 'Source', 'Transforms', 'Target', 'Quality', 'Schedule'] as const;

export function CreatePipelineWizard({
  open,
  onOpenChange,
  onCreated,
}: CreatePipelineWizardProps) {
  const router = useRouter();
  const [state, setState] = useState(createInitialPipelineWizardState);
  const [submitting, setSubmitting] = useState(false);

  const [sourcesQuery, modelsQuery] = useQueries({
    queries: [
      {
        queryKey: ['pipeline-wizard-sources'],
        queryFn: () =>
          dataSuiteApi.listSources({
            page: 1,
            per_page: 200,
            sort: 'name',
            order: 'asc',
          }),
      },
      {
        queryKey: ['pipeline-wizard-models'],
        queryFn: () =>
          dataSuiteApi.listModels({
            page: 1,
            per_page: 200,
            sort: 'name',
            order: 'asc',
          }),
      },
    ],
  });

  const schemaQuery = useQuery({
    queryKey: ['pipeline-wizard-source-schema', state.source.source_id],
    queryFn: () => dataSuiteApi.getSourceSchema(state.source.source_id),
    enabled: open && Boolean(state.source.source_id),
    staleTime: 60_000,
  });

  const sources = sourcesQuery.data?.data ?? [];
  const models = modelsQuery.data?.data ?? [];
  const selectedTable = useMemo(
    () => findTable(schemaQuery.data ?? state.sourceSchema, state.source.source_table),
    [schemaQuery.data, state.source.source_table, state.sourceSchema],
  );
  const availableColumns = useMemo(() => tableColumnNames(selectedTable), [selectedTable]);

  useEffect(() => {
    if (!open) {
      setState(createInitialPipelineWizardState());
      setSubmitting(false);
    }
  }, [open]);

  useEffect(() => {
    if (!schemaQuery.data) {
      return;
    }
    setState((current) => ({
      ...current,
      sourceSchema: schemaQuery.data,
    }));
  }, [schemaQuery.data]);

  useEffect(() => {
    setState((current) => ({
      ...current,
      selectedSource: sources.find((source) => source.id === current.source.source_id) ?? null,
      selectedModel: models.find((model) => model.id === current.target.target_model_id) ?? null,
    }));
  }, [models, sources]);

  const isLoading = sourcesQuery.isLoading || modelsQuery.isLoading;
  const loadError = sourcesQuery.error ?? modelsQuery.error ?? null;

  const closeWizard = () => {
    const dirty =
      state.basic.name ||
      state.basic.description ||
      state.source.source_id ||
      state.transforms.length > 0 ||
      state.quality.quality_gates.length > 0;
    if (dirty && !window.confirm('Discard pipeline wizard changes?')) {
      return;
    }
    onOpenChange(false);
  };

  const goToCompletedStep = (step: number) => {
    if (step < state.step) {
      setState((current) => ({ ...current, step }));
    }
  };

  const startPreview = () => {
    const beforeRows = buildSampleRows(selectedTable, 5);
    const preview = runPreview(beforeRows, state.transforms);
    setState((current) => ({
      ...current,
      previewBeforeRows: beforeRows,
      previewAfterRows: preview.rows,
      previewError: preview.error,
    }));
  };

  const submitPipeline = async (schedule: PipelineScheduleValues) => {
    setSubmitting(true);
    try {
      const nextState = { ...state, schedule };
      const pipeline = await dataSuiteApi.createPipeline(serializePipelinePayload(nextState));
      showSuccess('Pipeline created successfully.', `${pipeline.name} is ready to run.`);
      onCreated?.();
      onOpenChange(false);
      router.push(`/data/pipelines/${pipeline.id}`);
    } catch (error) {
      showApiError(error);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          closeWizard();
          return;
        }
        onOpenChange(next);
      }}
    >
      <DialogContent className="max-w-6xl">
        <DialogHeader>
          <DialogTitle>Create Pipeline</DialogTitle>
          <DialogDescription>
            Define the source, transformation flow, quality gates, and schedule for a governed data pipeline.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          <div className="grid grid-cols-6 gap-2">
            {STEPS.map((label, index) => {
              const step = index + 1;
              const complete = state.step > step;
              const current = state.step === step;
              return (
                <button
                  key={label}
                  type="button"
                  className="flex items-center gap-3 rounded-lg border p-3 text-left"
                  onClick={() => goToCompletedStep(step)}
                >
                  <span
                    className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium ${
                      complete
                        ? 'bg-emerald-500 text-white'
                        : current
                          ? 'bg-primary text-primary-foreground'
                          : 'border border-muted-foreground/30 text-muted-foreground'
                    }`}
                  >
                    {complete ? '✓' : step}
                  </span>
                  <span className="text-sm font-medium">{label}</span>
                </button>
              );
            })}
          </div>

          <div className="rounded-xl border bg-muted/10 p-4">
            <div className="font-medium">{state.basic.name || 'New pipeline'}</div>
            <div className="mt-1 text-sm text-muted-foreground">
              {state.selectedSource ? `${state.selectedSource.name} → ${state.target.target_table || state.selectedModel?.display_name || 'target pending'}` : 'Select a source to begin.'}
            </div>
            <div className="mt-2 text-xs text-muted-foreground">
              Schedule: {describeSchedule(state.schedule.schedule_mode, state.schedule.schedule_preset, state.schedule.custom_cron || '')}
            </div>
          </div>

          {isLoading ? (
            <LoadingSkeleton variant="card" />
          ) : loadError ? (
            <ErrorState
              message={loadError instanceof Error ? loadError.message : 'Failed to load pipeline wizard data.'}
              onRetry={() => {
                void sourcesQuery.refetch();
                void modelsQuery.refetch();
              }}
            />
          ) : (
            <>
              {state.step === 1 ? (
                <WizardStepBasic
                  defaultValues={state.basic}
                  onContinue={(basic) => setState((current) => ({ ...current, basic, step: 2 }))}
                />
              ) : null}

              {state.step === 2 ? (
                <WizardStepSource
                  sources={sources}
                  schema={schemaQuery.data ?? state.sourceSchema}
                  schemaLoading={schemaQuery.isFetching}
                  defaultValues={state.source}
                  onBack={() => setState((current) => ({ ...current, step: 1 }))}
                  onSourceChange={(sourceId) =>
                    setState((current) => ({
                      ...current,
                      source: {
                        ...current.source,
                        source_id: sourceId,
                      },
                      sourceSchema: current.source.source_id === sourceId ? current.sourceSchema : null,
                      previewBeforeRows: [],
                      previewAfterRows: [],
                      previewError: null,
                    }))
                  }
                  onContinue={(source) =>
                    setState((current) => ({
                      ...current,
                      source,
                      step: 3,
                      selectedSource: sources.find((item) => item.id === source.source_id) ?? null,
                      sourceSchema: schemaQuery.data ?? current.sourceSchema,
                      previewBeforeRows: [],
                      previewAfterRows: [],
                      previewError: null,
                    }))
                  }
                />
              ) : null}

              {state.step === 3 ? (
                <WizardStepTransforms
                  transforms={state.transforms}
                  availableColumns={availableColumns}
                  previewBeforeRows={state.previewBeforeRows}
                  previewAfterRows={state.previewAfterRows}
                  previewError={state.previewError}
                  onBack={() => setState((current) => ({ ...current, step: 2 }))}
                  onChange={(transforms) => setState((current) => ({ ...current, transforms }))}
                  onPreview={startPreview}
                  onContinue={() => setState((current) => ({ ...current, step: 4 }))}
                />
              ) : null}

              {state.step === 4 ? (
                <WizardStepTarget
                  defaultValues={state.target}
                  sources={sources}
                  models={models}
                  availableColumns={availableColumns}
                  onBack={() => setState((current) => ({ ...current, step: 3 }))}
                  onContinue={(target: PipelineTargetValues) =>
                    setState((current) => ({
                      ...current,
                      target,
                      step: 5,
                      selectedModel: models.find((model) => model.id === target.target_model_id) ?? null,
                    }))
                  }
                />
              ) : null}

              {state.step === 5 ? (
                <WizardStepQuality
                  value={state.quality}
                  availableColumns={availableColumns}
                  onBack={() => setState((current) => ({ ...current, step: 4 }))}
                  onChange={(quality) => setState((current) => ({ ...current, quality }))}
                  onContinue={() => setState((current) => ({ ...current, step: 6 }))}
                />
              ) : null}

              {state.step === 6 ? (
                <WizardStepSchedule
                  defaultValues={state.schedule}
                  onBack={() => setState((current) => ({ ...current, step: 5 }))}
                  onSubmit={(schedule) => {
                    setState((current) => ({ ...current, schedule }));
                    void submitPipeline(schedule);
                  }}
                  submitting={submitting}
                />
              ) : null}
            </>
          )}

          {state.step > 1 && schemaQuery.error ? (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertTitle>Schema load failed</AlertTitle>
              <AlertDescription>
                {schemaQuery.error instanceof Error ? schemaQuery.error.message : 'Could not load source schema.'}
              </AlertDescription>
            </Alert>
          ) : null}

          <div className="flex justify-start">
            <Button type="button" variant="ghost" onClick={closeWizard}>
              Cancel
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
