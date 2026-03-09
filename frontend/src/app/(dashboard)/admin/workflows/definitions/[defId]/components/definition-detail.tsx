'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import {
  ArrowLeft,
  Pencil,
  Upload,
  Archive,
  Copy,
  Calendar,
  Globe,
  MousePointerClick,
  Webhook,
  Loader2,
} from 'lucide-react';
import { useQueryClient } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { StatusBadge } from '@/components/shared/status-badge';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { DataTable } from '@/components/shared/data-table/data-table';
import { getWorkflowInstanceColumns } from '@/components/workflows/workflow-instance-columns';
import { workflowDefinitionStatusConfig, workflowStatusConfig } from '@/lib/status-configs';
import { workflowInstanceFilters } from '@/components/workflows/workflow-instance-filters';
import { SearchInput } from '@/components/shared/forms/search-input';
import { formatDateTime, titleCase } from '@/lib/format';
import { useDataTable } from '@/hooks/use-data-table';
import {
  useWorkflowDefinition,
  useWorkflowDefinitionVersions,
  usePublishWorkflowDefinition,
  useArchiveWorkflowDefinition,
  useCloneWorkflowDefinition,
} from '@/hooks/use-workflow-definitions';
import { WorkflowCanvas } from '../designer/components/workflow-canvas';
import { apiGet } from '@/lib/api';
import type { WorkflowInstance } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';

const triggerIcons: Record<string, React.ElementType> = {
  manual: MousePointerClick,
  event: Globe,
  schedule: Calendar,
  webhook: Webhook,
};

export function DefinitionDetailClient() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const defId = (params?.defId as string | undefined) ?? '';
  const [activeTab, setActiveTab] = useState('overview');

  const { data: definition, isLoading, isError, refetch } = useWorkflowDefinition(defId);
  const { data: versionsData } = useWorkflowDefinitionVersions(defId);
  const publishMutation = usePublishWorkflowDefinition();
  const archiveMutation = useArchiveWorkflowDefinition();
  const cloneMutation = useCloneWorkflowDefinition();

  // Instances tab data table
  const instancesTable = useDataTable<WorkflowInstance>({
    queryKey: `definition-${defId}-instances`,
    defaultPageSize: 10,
    defaultSort: { column: 'started_at', direction: 'desc' },
    fetchFn: (p) =>
      apiGet<PaginatedResponse<WorkflowInstance>>('/api/v1/workflows/instances', {
        page: p.page,
        per_page: p.per_page,
        sort: p.sort ?? 'started_at',
        order: p.order ?? 'desc',
        search: p.search,
        definition_id: defId,
        ...(p.filters?.status
          ? {
              status: Array.isArray(p.filters.status)
                ? p.filters.status.join(',')
                : p.filters.status,
            }
          : {}),
      }),
  });

  const instanceColumns = getWorkflowInstanceColumns({
    onView: (inst) => router.push(`/workflows/${inst.id}`),
    onCancel: () => undefined,
    onRetry: () => undefined,
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <LoadingSkeleton variant="card" count={3} />
      </div>
    );
  }

  if (isError || !definition) {
    return (
      <ErrorState
        message="Failed to load workflow definition"
        onRetry={() => refetch()}
      />
    );
  }

  const TriggerIcon = triggerIcons[definition.trigger.type] ?? Globe;
  const versions = versionsData?.versions ?? [];

  return (
    <div className="space-y-6">
      {/* Back button */}
      <button
        onClick={() => router.push('/admin/workflows/definitions')}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        type="button"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Definitions
      </button>

      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">{definition.name}</h1>
          <div className="mt-1.5 flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
            <StatusBadge
              status={definition.status}
              config={workflowDefinitionStatusConfig}
            />
            <span>v{definition.version}</span>
            <Badge variant="secondary" className="text-xs">
              {titleCase(definition.category)}
            </Badge>
          </div>
          {definition.description && (
            <p className="mt-2 text-sm text-muted-foreground max-w-xl">
              {definition.description}
            </p>
          )}
        </div>

        <div className="flex items-center gap-2">
          {definition.status === 'draft' && (
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  router.push(
                    `/admin/workflows/definitions/${defId}/designer`,
                  )
                }
              >
                <Pencil className="mr-1 h-3.5 w-3.5" />
                Edit
              </Button>
              <Button
                size="sm"
                onClick={() => publishMutation.mutate(defId)}
                disabled={publishMutation.isPending}
              >
                {publishMutation.isPending ? (
                  <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Upload className="mr-1 h-3.5 w-3.5" />
                )}
                Publish
              </Button>
            </>
          )}
          {definition.status === 'active' && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => archiveMutation.mutate(defId)}
              disabled={archiveMutation.isPending}
            >
              {archiveMutation.isPending ? (
                <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
              ) : (
                <Archive className="mr-1 h-3.5 w-3.5" />
              )}
              Archive
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => cloneMutation.mutate(defId)}
            disabled={cloneMutation.isPending}
          >
            <Copy className="mr-1 h-3.5 w-3.5" />
            Clone
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="designer">Designer</TabsTrigger>
          <TabsTrigger value="versions">Versions</TabsTrigger>
          <TabsTrigger value="instances">Instances</TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="space-y-4 mt-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {/* Trigger card */}
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Trigger</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex items-center gap-2">
                  <TriggerIcon className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium">
                    {titleCase(definition.trigger.type)}
                  </span>
                </div>
                {definition.trigger.event_type && (
                  <p className="text-xs text-muted-foreground mt-1">
                    Event: {definition.trigger.event_type}
                  </p>
                )}
                {definition.trigger.schedule_cron && (
                  <p className="text-xs text-muted-foreground mt-1 font-mono">
                    {definition.trigger.schedule_cron}
                  </p>
                )}
              </CardContent>
            </Card>

            {/* Stats card */}
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Statistics</CardTitle>
              </CardHeader>
              <CardContent className="space-y-1">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Steps</span>
                  <span className="font-medium">{definition.steps.length}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Instances</span>
                  <span className="font-medium">{definition.instance_count}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Variables</span>
                  <span className="font-medium">{definition.variables.length}</span>
                </div>
              </CardContent>
            </Card>

            {/* Dates card */}
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Timeline</CardTitle>
              </CardHeader>
              <CardContent className="space-y-1">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Created</span>
                  <span>{formatDateTime(definition.created_at)}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Updated</span>
                  <span>{formatDateTime(definition.updated_at)}</span>
                </div>
                {definition.published_at && (
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Published</span>
                    <span>{formatDateTime(definition.published_at)}</span>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Variables */}
          {definition.variables.length > 0 && (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Variables</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>Required</TableHead>
                      <TableHead>Default</TableHead>
                      <TableHead>Description</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {definition.variables.map((v) => (
                      <TableRow key={v.name}>
                        <TableCell className="font-mono text-xs">
                          {v.name}
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline" className="text-xs">
                            {v.type}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {v.required ? (
                            <Badge variant="default" className="text-xs">
                              Required
                            </Badge>
                          ) : (
                            <span className="text-xs text-muted-foreground">
                              Optional
                            </span>
                          )}
                        </TableCell>
                        <TableCell className="text-xs font-mono">
                          {v.default_value !== undefined
                            ? String(v.default_value)
                            : '—'}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {v.description || '—'}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Designer Tab (read-only) */}
        <TabsContent value="designer" className="mt-4">
          <Card className="overflow-hidden">
            <div className="h-[500px]">
              <WorkflowCanvas
                definition={definition}
                readOnly
                isSaving={false}
                isPublishing={false}
                onSave={() => undefined}
                onPublish={() => undefined}
              />
            </div>
          </Card>
        </TabsContent>

        {/* Versions Tab */}
        <TabsContent value="versions" className="mt-4">
          <Card>
            <CardContent className="pt-6">
              {versions.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-8">
                  No version history available.
                </p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Version</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Published At</TableHead>
                      <TableHead>Published By</TableHead>
                      <TableHead>Change Summary</TableHead>
                      <TableHead>Steps</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {versions.map((v) => (
                      <TableRow key={v.version}>
                        <TableCell className="font-medium">
                          v{v.version}
                        </TableCell>
                        <TableCell>
                          <StatusBadge
                            status={v.status}
                            config={workflowDefinitionStatusConfig}
                          />
                        </TableCell>
                        <TableCell>
                          {v.published_at
                            ? formatDateTime(v.published_at)
                            : '—'}
                        </TableCell>
                        <TableCell>{v.published_by ?? '—'}</TableCell>
                        <TableCell className="text-sm max-w-xs truncate">
                          {v.change_summary || '—'}
                        </TableCell>
                        <TableCell>{v.step_count}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Instances Tab */}
        <TabsContent value="instances" className="mt-4">
          <DataTable
            columns={instanceColumns}
            filters={workflowInstanceFilters}
            searchSlot={
              <SearchInput
                value={instancesTable.searchValue}
                onChange={instancesTable.setSearch}
                placeholder="Search instances..."
              />
            }
            {...instancesTable.tableProps}
            onRowClick={(row) => router.push(`/workflows/${row.id}`)}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}
