'use client';

import { useEffect } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { Database, Loader2 } from 'lucide-react';
import { FormField } from '@/components/shared/forms/form-field';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { getClassificationBadge, getSourceTypeVisual } from '@/lib/data-suite/utils';
import type { DataSource, DiscoveredSchema } from '@/lib/data-suite';
import {
  pipelineSourceSchema,
  type PipelineSourceValues,
} from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';
import {
  findTable,
  qualifiedTableName,
  tableColumnNames,
} from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-utils';

interface WizardStepSourceProps {
  sources: DataSource[];
  schema: DiscoveredSchema | null;
  schemaLoading: boolean;
  defaultValues: PipelineSourceValues;
  onBack: () => void;
  onSourceChange: (sourceId: string) => void;
  onContinue: (values: PipelineSourceValues) => void;
}

export function WizardStepSource({
  sources,
  schema,
  schemaLoading,
  defaultValues,
  onBack,
  onSourceChange,
  onContinue,
}: WizardStepSourceProps) {
  const form = useForm<PipelineSourceValues>({
    resolver: zodResolver(pipelineSourceSchema),
    mode: 'onChange',
    defaultValues,
  });

  const sourceId = form.watch('source_id');
  const selectedSource = sources.find((source) => source.id === sourceId) ?? null;
  const selectedTable = findTable(schema, form.watch('source_table'));

  useEffect(() => {
    if (sourceId) {
      onSourceChange(sourceId);
    }
  }, [onSourceChange, sourceId]);

  return (
    <FormProvider {...form}>
      <form className="space-y-6" onSubmit={form.handleSubmit(onContinue)}>
        <FormField name="source_id" label="Source" required>
          <Select
            value={sourceId}
            onValueChange={(next) => {
              form.setValue('source_id', next, { shouldValidate: true });
              form.setValue('source_table', '', { shouldValidate: true });
            }}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select a data source" />
            </SelectTrigger>
            <SelectContent>
              {sources.map((source) => {
                const visual = getSourceTypeVisual(source.type);
                return (
                  <SelectItem key={source.id} value={source.id}>
                    {visual.label} • {source.name}
                  </SelectItem>
                );
              })}
            </SelectContent>
          </Select>
        </FormField>

        {selectedSource ? (
          <div className="rounded-xl border bg-muted/10 p-4">
            <div className="flex items-start gap-3">
              <div className="rounded-md border p-2">
                <Database className="h-4 w-4" />
              </div>
              <div className="flex-1">
                <div className="font-medium">{selectedSource.name}</div>
                <div className="text-sm text-muted-foreground">{selectedSource.description || 'Governed data source'}</div>
                <div className="mt-2 flex flex-wrap gap-2">
                  <Badge variant="outline">{getSourceTypeVisual(selectedSource.type).label}</Badge>
                  <Badge variant="outline" className={getClassificationBadge(schema?.highest_classification).className}>
                    {getClassificationBadge(schema?.highest_classification).label}
                  </Badge>
                </div>
              </div>
            </div>
          </div>
        ) : null}

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <FormField name="read_mode" label="Read mode" required>
            <Select
              value={form.watch('read_mode')}
              onValueChange={(next) =>
                form.setValue('read_mode', next as PipelineSourceValues['read_mode'], { shouldValidate: true })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="table">Table</SelectItem>
                <SelectItem value="query">Query</SelectItem>
              </SelectContent>
            </Select>
          </FormField>

          {form.watch('read_mode') === 'table' ? (
            <FormField name="source_table" label="Source table" required>
              <Select
                value={form.watch('source_table')}
                onValueChange={(next) => form.setValue('source_table', next, { shouldValidate: true })}
              >
                <SelectTrigger>
                  <SelectValue placeholder={schemaLoading ? 'Loading schema…' : 'Select table'} />
                </SelectTrigger>
                <SelectContent>
                  {(schema?.tables ?? []).map((table) => (
                    <SelectItem key={qualifiedTableName(table)} value={qualifiedTableName(table)}>
                      {qualifiedTableName(table)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
          ) : (
            <FormField name="source_query" label="Source query" required>
              <Textarea
                {...form.register('source_query')}
                rows={4}
                placeholder="SELECT * FROM public.orders WHERE updated_at >= NOW() - INTERVAL '1 day'"
              />
            </FormField>
          )}
        </div>

        <div className="rounded-xl border p-4">
          <label className="flex items-center gap-3 text-sm font-medium">
            <Checkbox
              checked={form.watch('incremental_enabled')}
              onCheckedChange={(checked) =>
                form.setValue('incremental_enabled', checked === true, { shouldValidate: true })
              }
            />
            Enable incremental extraction
          </label>
          {form.watch('incremental_enabled') ? (
            <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="incremental_field" label="Incremental field" required>
                <Select
                  value={form.watch('incremental_field')}
                  onValueChange={(next) => form.setValue('incremental_field', next, { shouldValidate: true })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select field" />
                  </SelectTrigger>
                  <SelectContent>
                    {tableColumnNames(selectedTable).map((column) => (
                      <SelectItem key={column} value={column}>
                        {column}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="incremental_value" label="Initial value">
                <Input {...form.register('incremental_value')} placeholder="2026-01-01T00:00:00Z" />
              </FormField>
            </div>
          ) : null}
        </div>

        <div className="rounded-xl border bg-muted/10 p-4">
          <div className="mb-3 flex items-center justify-between">
            <div>
              <div className="font-medium">Schema context</div>
              <div className="text-sm text-muted-foreground">
                {schemaLoading ? 'Loading real source schema…' : schema ? `${schema.table_count} tables discovered` : 'Select a source to load schema'}
              </div>
            </div>
            {schemaLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
          </div>
          {selectedTable ? (
            <div className="flex flex-wrap gap-2">
              {selectedTable.columns.map((column) => (
                <Badge key={column.name} variant="outline">
                  {column.name}
                </Badge>
              ))}
            </div>
          ) : null}
        </div>

        <div className="flex justify-between">
          <Button type="button" variant="outline" onClick={onBack}>
            Back
          </Button>
          <Button type="submit" disabled={!form.formState.isValid || schemaLoading}>
            Continue
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}

