'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { AlertTriangle } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { useAuth } from '@/hooks/use-auth';
import { dataSuiteApi, deriveSourceName } from '@/lib/data-suite';
import type { SourceConfigureValues } from '@/lib/data-suite/forms';
import { showApiError, showSuccess } from '@/lib/toast';
import { createInitialSourceWizardState, type SourceWizardState } from '@/app/(dashboard)/data/sources/_components/source-wizard-types';
import { WizardStepConnection } from '@/app/(dashboard)/data/sources/_components/wizard-step-connection';
import { WizardStepSchema } from '@/app/(dashboard)/data/sources/_components/wizard-step-schema';
import { WizardStepSync } from '@/app/(dashboard)/data/sources/_components/wizard-step-sync';
import { WizardStepTest } from '@/app/(dashboard)/data/sources/_components/wizard-step-test';
import { WizardStepType } from '@/app/(dashboard)/data/sources/_components/wizard-step-type';

interface CreateSourceWizardProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
}

const STEPS = ['Type', 'Connection', 'Test', 'Schema', 'Configure'] as const;
const CONFIGURE_FORM_ID = 'create-source-configure-form';

export function CreateSourceWizard({
  open,
  onOpenChange,
  onCreated,
}: CreateSourceWizardProps) {
  const router = useRouter();
  const { hasPermission } = useAuth();
  const [state, setState] = useState<SourceWizardState>(createInitialSourceWizardState);
  const [testing, setTesting] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) {
      setState(createInitialSourceWizardState());
      setTesting(false);
      setDiscovering(false);
      setSubmitting(false);
    }
  }, [open]);

  const connectionLabel = useMemo(() => {
    const host = typeof state.connectionConfig.host === 'string' ? state.connectionConfig.host : typeof state.connectionConfig.base_url === 'string' ? state.connectionConfig.base_url : typeof state.connectionConfig.bucket === 'string' ? state.connectionConfig.bucket : 'source';
    const port = typeof state.connectionConfig.port === 'number' ? `:${state.connectionConfig.port}` : '';
    const database = typeof state.connectionConfig.database === 'string' ? `/${state.connectionConfig.database}` : '';
    return `${host}${port}${database}`;
  }, [state.connectionConfig]);

  async function cleanupProvisionalSource() {
    if (!state.createdSource?.id) {
      return;
    }
    try {
      await dataSuiteApi.deleteSource(state.createdSource.id);
    } catch {
      // Best effort cleanup only.
    }
  }

  async function ensurePersistedSource() {
    if (!state.sourceType) {
      throw new Error('Select a source type first.');
    }

    const signature = JSON.stringify(state.connectionConfig);
    if (!state.createdSource) {
      const created = await dataSuiteApi.createSource({
        name: deriveSourceName(state.sourceType, state.connectionConfig),
        description: '',
        type: state.sourceType,
        connection_config: state.connectionConfig,
        sync_frequency: null,
        tags: [],
        metadata: {},
      });
      setState((current) => ({
        ...current,
        createdSource: created,
        persistedConfigSignature: signature,
        configuration: {
          ...current.configuration,
          name: current.configuration.name || created.name,
        },
      }));
      return created;
    }

    if (state.persistedConfigSignature !== signature) {
      const updated = await dataSuiteApi.updateSource(state.createdSource.id, {
        connection_config: state.connectionConfig,
      });
      setState((current) => ({
        ...current,
        createdSource: updated,
        persistedConfigSignature: signature,
      }));
      return updated;
    }

    return state.createdSource;
  }

  async function runConnectionVerification() {
    if (!state.sourceType) {
      return;
    }
    setTesting(true);
    try {
      const result = await dataSuiteApi.testSourceConfig({
        type: state.sourceType,
        connection_config: state.connectionConfig,
      });
      setState((current) => ({
        ...current,
        testResult: result,
        testError: null,
        skippedVerificationDetails: false,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Connection test failed';
      setState((current) => ({
        ...current,
        testError: message,
        testResult: null,
      }));
    } finally {
      setTesting(false);
    }
  }

  async function discoverSchema() {
    setDiscovering(true);
    try {
      const source = await ensurePersistedSource();
      const schema = await dataSuiteApi.discoverSource(source.id);
      setState((current) => ({
        ...current,
        schema,
        testError: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Schema discovery failed';
      setState((current) => ({
        ...current,
        testError: message,
      }));
    } finally {
      setDiscovering(false);
    }
  }

  useEffect(() => {
    if (!open || state.step !== 3) {
      return;
    }
    void runConnectionVerification();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, state.step, state.persistedConfigSignature]);

  useEffect(() => {
    if (!open || state.step !== 4) {
      return;
    }
    if (state.schema) {
      return;
    }
    void discoverSchema();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, state.step, state.schema]);

  const closeWizard = async () => {
    const dirty = state.sourceType || Object.keys(state.connectionConfig).length > 0 || state.createdSource;
    if (dirty && !window.confirm('Discard changes?')) {
      return;
    }
    await cleanupProvisionalSource();
    onOpenChange(false);
  };

  const finalizeSource = async (configuration: SourceConfigureValues) => {
    if (!state.createdSource) {
      return;
    }
    setSubmitting(true);
    try {
      const updated = await dataSuiteApi.updateSource(state.createdSource.id, {
        name: configuration.name,
        description: configuration.description,
        sync_frequency: configuration.sync_frequency,
        tags: configuration.tags,
      });
      showSuccess('Source created successfully.', `${updated.name} is ready for use.`);
      onCreated?.();
      onOpenChange(false);
      router.push(`/data/sources/${updated.id}`);
    } catch (error) {
      showApiError(error);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) { void closeWizard(); } else { onOpenChange(next); } }}>
      <DialogContent className="max-w-5xl">
        <DialogHeader>
          <DialogTitle>Create Source</DialogTitle>
          <DialogDescription>
            Add a governed source with connection verification, schema discovery, and sync configuration.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          <div className="grid grid-cols-5 gap-2">
            {STEPS.map((label, index) => {
              const step = index + 1;
              const complete = state.step > step;
              const current = state.step === step;
              return (
                <button
                  key={label}
                  type="button"
                  className="flex items-center gap-3 rounded-lg border p-3 text-left"
                  onClick={() => {
                    if (complete) {
                      setState((currentState) => ({ ...currentState, step }));
                    }
                  }}
                >
                  <span className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium ${complete ? 'bg-emerald-500 text-white' : current ? 'bg-primary text-primary-foreground' : 'border border-muted-foreground/30 text-muted-foreground'}`}>
                    {complete ? '✓' : step}
                  </span>
                  <span className="text-sm font-medium">{label}</span>
                </button>
              );
            })}
          </div>

          {state.step === 1 ? (
            <WizardStepType
              value={state.sourceType}
              onSelect={(sourceType) => {
                if (state.createdSource && state.sourceType && state.sourceType !== sourceType) {
                  void dataSuiteApi.deleteSource(state.createdSource.id).catch(() => undefined);
                }
                setState((current) => ({
                  ...current,
                  sourceType,
                  step: 2,
                  connectionConfig: current.sourceType === sourceType ? current.connectionConfig : {},
                  createdSource: current.sourceType === sourceType ? current.createdSource : null,
                  persistedConfigSignature: current.sourceType === sourceType ? current.persistedConfigSignature : null,
                }));
              }}
            />
          ) : null}

          {state.step === 2 && state.sourceType ? (
            <WizardStepConnection
              sourceType={state.sourceType}
              defaultValues={state.connectionConfig}
              onSave={(connectionConfig) =>
                setState((current) => ({
                  ...current,
                  connectionConfig,
                  step: 3,
                  testResult: null,
                  testError: null,
                  schema: null,
                }))
              }
            />
          ) : null}

          {state.step === 3 ? (
            <WizardStepTest
              loading={testing}
              connectionLabel={connectionLabel}
              result={state.testResult}
              error={state.testError}
              onEditConnection={() => setState((current) => ({ ...current, step: 2 }))}
              onRetry={() => void runConnectionVerification()}
              onContinueWithoutDetails={() => setState((current) => ({ ...current, step: 4, skippedVerificationDetails: true }))}
            />
          ) : null}

          {state.step === 4 ? (
            <WizardStepSchema
              schema={state.schema}
              loading={discovering}
              error={state.testError}
              reviewed={state.schemaReviewed}
              onReviewedChange={(value) => setState((current) => ({ ...current, schemaReviewed: value }))}
              onRetry={() => void discoverSchema()}
              canViewPii={hasPermission('data:pii')}
            />
          ) : null}

          {state.step === 5 ? (
            <WizardStepSync
              defaultValues={state.configuration}
              formId={CONFIGURE_FORM_ID}
              onSubmit={(configuration) => {
                setState((current) => ({ ...current, configuration }));
                void finalizeSource(configuration);
              }}
            />
          ) : null}

          {state.step === 3 && !testing && state.testResult?.success ? (
            <div className="flex justify-between">
              <Button type="button" variant="outline" onClick={() => setState((current) => ({ ...current, step: 2 }))}>
                Back
              </Button>
              <Button type="button" onClick={() => setState((current) => ({ ...current, step: 4 }))}>
                Continue
              </Button>
            </div>
          ) : null}

          {state.step === 4 ? (
            <div className="flex justify-between">
              <Button type="button" variant="outline" onClick={() => setState((current) => ({ ...current, step: 3 }))}>
                Back
              </Button>
              <Button
                type="button"
                onClick={() => setState((current) => ({ ...current, step: 5 }))}
                disabled={!state.schemaReviewed}
              >
                Continue
              </Button>
            </div>
          ) : null}

          {state.step === 5 ? (
            <div className="space-y-3">
              {state.skippedVerificationDetails ? (
                <Alert className="border-amber-200 bg-amber-50">
                  <AlertTriangle className="h-4 w-4 text-amber-600" />
                  <AlertTitle className="text-amber-700">Verification details were skipped</AlertTitle>
                  <AlertDescription className="text-amber-700">
                    The source will be created during schema discovery or final submission, so no connection health record was persisted yet.
                  </AlertDescription>
                </Alert>
              ) : null}
              <div className="flex justify-between">
                <Button type="button" variant="outline" onClick={() => setState((current) => ({ ...current, step: 4 }))}>
                  Back
                </Button>
                <Button type="submit" form={CONFIGURE_FORM_ID} disabled={submitting}>
                  {submitting ? 'Creating…' : 'Create Source'}
                </Button>
              </div>
            </div>
          ) : null}

          {state.step > 1 ? (
            <div className="flex justify-start">
              <Button type="button" variant="ghost" onClick={() => void closeWizard()}>
                Cancel
              </Button>
            </div>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}
