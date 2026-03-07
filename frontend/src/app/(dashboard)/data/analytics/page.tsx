'use client';

import { useMemo, useState } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { FormField } from '@/components/shared/forms/form-field';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { ModelBrowser } from '@/app/(dashboard)/data/analytics/_components/model-browser';
import { QueryBuilder, type QueryOrderRowState } from '@/app/(dashboard)/data/analytics/_components/query-builder';
import { type QueryAggregationRowState } from '@/app/(dashboard)/data/analytics/_components/query-aggregation-builder';
import { QueryExecutionStatus } from '@/app/(dashboard)/data/analytics/_components/query-execution-status';
import { type QueryFilterRowState } from '@/app/(dashboard)/data/analytics/_components/query-filter-builder';
import { QueryResultsTable } from '@/app/(dashboard)/data/analytics/_components/query-results-table';
import { SavedQueriesList } from '@/app/(dashboard)/data/analytics/_components/saved-queries-list';
import { dataSuiteApi, saveQuerySchema, type AnalyticsAggregation, type AnalyticsFilter, type AnalyticsOrder, type AnalyticsQuery, type QueryResult, type SavedQuery, type SaveQueryValues } from '@/lib/data-suite';
import { showApiError, showSuccess } from '@/lib/toast';

export default function DataAnalyticsPage() {
  const [browserSearch, setBrowserSearch] = useState('');
  const [tab, setTab] = useState<'builder' | 'saved'>('builder');
  const [selectedModelId, setSelectedModelId] = useState<string | null>(null);
  const [selectedColumns, setSelectedColumns] = useState<string[]>([]);
  const [filters, setFilters] = useState<QueryFilterRowState[]>([]);
  const [aggregations, setAggregations] = useState<QueryAggregationRowState[]>([]);
  const [groupBy, setGroupBy] = useState<string[]>([]);
  const [orders, setOrders] = useState<QueryOrderRowState[]>([]);
  const [limit, setLimit] = useState(100);
  const [executionState, setExecutionState] = useState<'idle' | 'running' | 'success' | 'error'>('idle');
  const [executionMessage, setExecutionMessage] = useState<string | undefined>();
  const [result, setResult] = useState<QueryResult | null>(null);
  const [saveOpen, setSaveOpen] = useState(false);
  const [editingSaved, setEditingSaved] = useState<SavedQuery | null>(null);

  const modelsQuery = useQuery({
    queryKey: ['data-analytics-models'],
    queryFn: () =>
      dataSuiteApi.listModels({
        page: 1,
        per_page: 200,
        sort: 'updated_at',
        order: 'desc',
      }),
  });

  const savedQueryQuery = useQuery({
    queryKey: ['data-analytics-saved'],
    queryFn: () =>
      dataSuiteApi.listSavedQueries({
        page: 1,
        per_page: 100,
        sort: 'updated_at',
        order: 'desc',
      }),
  });

  const saveForm = useForm<SaveQueryValues>({
    resolver: zodResolver(saveQuerySchema),
    defaultValues: {
      name: '',
      description: '',
      visibility: 'private',
    },
  });

  const models = modelsQuery.data?.data ?? [];
  const selectedModel = models.find((model) => model.id === selectedModelId) ?? null;
  const modelNames = useMemo<Record<string, string>>(
    () => Object.fromEntries(models.map((model) => [model.id, model.display_name || model.name])),
    [models],
  );

  const buildAnalyticsQuery = (): AnalyticsQuery => {
    const normalizedFilters: AnalyticsFilter[] = filters
      .filter((filter) => filter.column && filter.operator)
      .map((filter) => {
        let value: AnalyticsFilter['value'];
        if (filter.operator === 'between') {
          value = [filter.value, filter.secondaryValue];
        } else if (filter.operator === 'in' || filter.operator === 'not_in') {
          value = filter.value.split(',').map((item) => item.trim()).filter(Boolean);
        } else if (filter.operator === 'is_null' || filter.operator === 'is_not_null') {
          value = undefined;
        } else {
          value = filter.value;
        }
        return {
          column: filter.column,
          operator: filter.operator,
          value,
        };
      });

    const normalizedAggregations: AnalyticsAggregation[] = aggregations
      .filter((aggregation) => aggregation.column && aggregation.func && aggregation.alias)
      .map((aggregation) => ({
        column: aggregation.column,
        function: aggregation.func,
        alias: aggregation.alias,
      }));

    const normalizedOrders: AnalyticsOrder[] = orders
      .filter((order) => order.column)
      .map((order) => ({
        column: order.column,
        direction: order.direction,
      }));

    return {
      columns: selectedColumns,
      filters: normalizedFilters.length > 0 ? normalizedFilters : undefined,
      group_by: groupBy.length > 0 ? groupBy : undefined,
      aggregations: normalizedAggregations.length > 0 ? normalizedAggregations : undefined,
      order_by: normalizedOrders.length > 0 ? normalizedOrders : undefined,
      limit,
    };
  };

  const clearBuilder = () => {
    setSelectedColumns([]);
    setFilters([]);
    setAggregations([]);
    setGroupBy([]);
    setOrders([]);
    setLimit(100);
    setExecutionState('idle');
    setExecutionMessage(undefined);
    setResult(null);
    setEditingSaved(null);
  };

  const runQuery = async (queryOverride?: AnalyticsQuery, modelIdOverride?: string) => {
    const modelId = modelIdOverride ?? selectedModelId;
    if (!modelId) {
      return;
    }
    try {
      setExecutionState('running');
      setExecutionMessage('Executing analytics query...');
      const executionResult = await dataSuiteApi.executeAnalyticsQuery({
        model_id: modelId,
        query: queryOverride ?? buildAnalyticsQuery(),
      });
      setResult(executionResult);
      setExecutionState('success');
      setExecutionMessage(`${executionResult.row_count} row(s) returned.`);
    } catch (error) {
      setExecutionState('error');
      setExecutionMessage(error instanceof Error ? error.message : 'Query execution failed.');
      showApiError(error);
    }
  };

  const onSaveSubmit = async (values: SaveQueryValues) => {
    if (!selectedModelId) {
      return;
    }
    try {
      const payload = {
        name: values.name,
        description: values.description ?? '',
        model_id: selectedModelId,
        query_definition: buildAnalyticsQuery(),
        visibility: values.visibility,
      };
      if (editingSaved) {
        await dataSuiteApi.updateSavedQuery(editingSaved.id, {
          description: values.description ?? '',
          query_definition: payload.query_definition,
          visibility: values.visibility,
        });
      } else {
        await dataSuiteApi.createSavedQuery(payload);
      }
      showSuccess(editingSaved ? 'Saved query updated.' : 'Saved query created.');
      setSaveOpen(false);
      setEditingSaved(null);
      await savedQueryQuery.refetch();
    } catch (error) {
      showApiError(error);
    }
  };

  const loadSavedQueryIntoBuilder = (saved: SavedQuery) => {
    setSelectedModelId(saved.model_id);
    setSelectedColumns(saved.query_definition.columns ?? []);
    setFilters(
      (saved.query_definition.filters ?? []).map((filter) => ({
        id: crypto.randomUUID(),
        column: filter.column,
        operator: filter.operator,
        value: Array.isArray(filter.value) ? `${filter.value[0] ?? ''}` : `${filter.value ?? ''}`,
        secondaryValue: Array.isArray(filter.value) ? `${filter.value[1] ?? ''}` : '',
      })),
    );
    setAggregations(
      (saved.query_definition.aggregations ?? []).map((aggregation) => ({
        id: crypto.randomUUID(),
        column: aggregation.column,
        func: aggregation.function,
        alias: aggregation.alias,
      })),
    );
    setGroupBy(saved.query_definition.group_by ?? []);
    setOrders(
      (saved.query_definition.order_by ?? []).map((order) => ({
        id: crypto.randomUUID(),
        column: order.column,
        direction: order.direction,
      })),
    );
    setLimit(saved.query_definition.limit ?? 100);
    setTab('builder');
    setEditingSaved(saved);
  };

  if (modelsQuery.isLoading || savedQueryQuery.isLoading) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Analytics" description="Loading governed models and saved queries." />
          <LoadingSkeleton variant="card" />
        </div>
      </PermissionRedirect>
    );
  }

  if (modelsQuery.error || savedQueryQuery.error) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message="Failed to load analytics workspace." onRetry={() => {
          void modelsQuery.refetch();
          void savedQueryQuery.refetch();
        }} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Analytics"
          description="Governed query builder for data models with saved query execution and PII-aware result rendering."
        />

        <div className="grid gap-4 xl:grid-cols-[0.3fr_0.7fr]">
          <ModelBrowser
            models={models}
            selectedModelId={selectedModelId}
            search={browserSearch}
            onSearch={setBrowserSearch}
            onSelectModel={(modelId) => {
              setSelectedModelId(modelId);
              clearBuilder();
              setSelectedModelId(modelId);
            }}
            onAddColumn={(columnName) => {
              setSelectedColumns((current) => Array.from(new Set([...current, columnName])));
            }}
          />

          <div className="space-y-4">
            <Tabs value={tab} onValueChange={(value) => setTab(value as 'builder' | 'saved')}>
              <TabsList>
                <TabsTrigger value="builder">Query Builder</TabsTrigger>
                <TabsTrigger value="saved">Saved Queries</TabsTrigger>
              </TabsList>

              <TabsContent value="builder" className="space-y-4">
                <QueryBuilder
                  models={models}
                  selectedModelId={selectedModelId}
                  selectedColumns={selectedColumns}
                  filters={filters}
                  aggregations={aggregations}
                  groupBy={groupBy}
                  orders={orders}
                  limit={limit}
                  running={executionState === 'running'}
                  onSelectModel={(modelId) => {
                    setSelectedModelId(modelId);
                    clearBuilder();
                    setSelectedModelId(modelId);
                  }}
                  onToggleColumn={(column) => {
                    setSelectedColumns((current) =>
                      current.includes(column) ? current.filter((item) => item !== column) : [...current, column],
                    );
                  }}
                  onSelectAllColumns={() => {
                    if (!selectedModel) {
                      return;
                    }
                    setSelectedColumns(selectedModel.schema_definition.map((field) => field.name));
                  }}
                  onClearColumns={() => setSelectedColumns([])}
                  onChangeFilters={setFilters}
                  onChangeAggregations={setAggregations}
                  onChangeGroupBy={setGroupBy}
                  onChangeOrders={setOrders}
                  onChangeLimit={setLimit}
                  onRun={() => void runQuery()}
                  onSave={() => {
                    saveForm.reset({
                      name: editingSaved?.name ?? '',
                      description: editingSaved?.description ?? '',
                      visibility: editingSaved?.visibility ?? 'private',
                    });
                    setSaveOpen(true);
                  }}
                  onClear={clearBuilder}
                />

                <QueryExecutionStatus state={executionState} message={executionMessage} />
                <QueryResultsTable result={result} />
              </TabsContent>

              <TabsContent value="saved" className="space-y-4">
                <SavedQueriesList
                  queries={savedQueryQuery.data?.data ?? []}
                  modelNames={modelNames}
                  onRun={(query) => void runQuery(query.query_definition, query.model_id)}
                  onEdit={loadSavedQueryIntoBuilder}
                  onDelete={async (query) => {
                    try {
                      await dataSuiteApi.deleteSavedQuery(query.id);
                      showSuccess('Saved query deleted.');
                      await savedQueryQuery.refetch();
                    } catch (error) {
                      showApiError(error);
                    }
                  }}
                />
              </TabsContent>
            </Tabs>
          </div>
        </div>

        <Dialog open={saveOpen} onOpenChange={setSaveOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{editingSaved ? 'Edit Saved Query' : 'Save Query'}</DialogTitle>
            </DialogHeader>
            <FormProvider {...saveForm}>
              <form className="space-y-4" onSubmit={saveForm.handleSubmit((values) => void onSaveSubmit(values))}>
                <FormField name="name" label="Name" required>
                  <Input {...saveForm.register('name')} />
                </FormField>
                <FormField name="description" label="Description">
                  <Textarea rows={4} {...saveForm.register('description')} />
                </FormField>
                <FormField name="visibility" label="Visibility" required>
                  <Select value={saveForm.watch('visibility')} onValueChange={(value) => saveForm.setValue('visibility', value as SaveQueryValues['visibility'], { shouldValidate: true })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="private">Private</SelectItem>
                      <SelectItem value="team">Team</SelectItem>
                      <SelectItem value="organization">Organization</SelectItem>
                    </SelectContent>
                  </Select>
                </FormField>
                <DialogFooter>
                  <Button type="button" variant="outline" onClick={() => setSaveOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="submit">Save</Button>
                </DialogFooter>
              </form>
            </FormProvider>
          </DialogContent>
        </Dialog>
      </div>
    </PermissionRedirect>
  );
}
