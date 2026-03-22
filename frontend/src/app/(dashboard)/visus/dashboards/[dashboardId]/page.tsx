'use client';

import Link from 'next/link';
import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, LayoutTemplate, Plus, Sparkles, Trash2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { EmptyState } from '@/components/common/empty-state';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { RelativeTime } from '@/components/shared/relative-time';
import { enterpriseApi } from '@/lib/enterprise';
import { safeJsonPreview, sortWidgetsByLayout } from '@/lib/enterprise/utils';
import { showApiError, showSuccess } from '@/lib/toast';
import type { VisusKPIDefinition, VisusWidget } from '@/types/suites';
import { compactWidgetPositions } from '../../_components/form-utils';
import { WidgetFormDialog } from './_components/widget-form-dialog';
import { WidgetPreviewDialog } from './_components/widget-preview-dialog';

interface DashboardDetailPageProps {
  params: {
    dashboardId: string;
  };
}

export default function VisusDashboardDetailPage({ params }: DashboardDetailPageProps) {
  const queryClient = useQueryClient();
  const dashboardId = params.dashboardId;
  const [widgetOpen, setWidgetOpen] = useState(false);
  const [editingWidget, setEditingWidget] = useState<VisusWidget | null>(null);
  const [previewWidget, setPreviewWidget] = useState<VisusWidget | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<VisusWidget | null>(null);

  const dashboardQuery = useQuery({
    queryKey: ['visus-dashboard-detail', dashboardId],
    queryFn: () => enterpriseApi.visus.getDashboard(dashboardId),
  });
  const widgetTypesQuery = useQuery({
    queryKey: ['visus-widget-types'],
    queryFn: () => enterpriseApi.visus.listWidgetTypes(),
  });
  const kpisQuery = useQuery({
    queryKey: ['visus-widget-kpis'],
    queryFn: () => enterpriseApi.visus.listKpis({ page: 1, per_page: 200, sort: 'name', order: 'asc' }),
  });

  const createMutation = useMutation({
    mutationFn: (payload: unknown) => enterpriseApi.visus.createWidget(dashboardId, payload),
    onSuccess: async () => {
      showSuccess('Widget created.');
      await queryClient.invalidateQueries({ queryKey: ['visus-dashboard-detail', dashboardId] });
      setWidgetOpen(false);
    },
    onError: showApiError,
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: unknown }) => enterpriseApi.visus.updateWidget(dashboardId, id, payload),
    onSuccess: async () => {
      showSuccess('Widget updated.');
      await queryClient.invalidateQueries({ queryKey: ['visus-dashboard-detail', dashboardId] });
      setEditingWidget(null);
      setWidgetOpen(false);
    },
    onError: showApiError,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => enterpriseApi.visus.deleteWidget(dashboardId, id),
    onSuccess: async () => {
      showSuccess('Widget deleted.');
      await queryClient.invalidateQueries({ queryKey: ['visus-dashboard-detail', dashboardId] });
      setDeleteTarget(null);
    },
    onError: showApiError,
  });

  const arrangeMutation = useMutation({
    mutationFn: (widgets: VisusWidget[]) => enterpriseApi.visus.updateWidgetLayout(dashboardId, compactWidgetPositions(widgets)),
    onSuccess: async () => {
      showSuccess('Layout normalized.');
      await queryClient.invalidateQueries({ queryKey: ['visus-dashboard-detail', dashboardId] });
    },
    onError: showApiError,
  });

  const dashboard = dashboardQuery.data;
  const widgets = sortWidgetsByLayout(dashboard?.widgets ?? []);
  const widgetTypes = widgetTypesQuery.data ?? [];
  const kpis = (kpisQuery.data?.data ?? []) as VisusKPIDefinition[];

  if (dashboardQuery.isLoading) {
    return (
      <PermissionRedirect permission="visus:read">
        <div className="space-y-6">
          <LoadingSkeleton variant="card" count={2} />
        </div>
      </PermissionRedirect>
    );
  }

  if (dashboardQuery.isError || !dashboard) {
    return (
      <PermissionRedirect permission="visus:read">
        <ErrorState
          title="Unable to load dashboard"
          message="The requested dashboard could not be loaded."
          onRetry={() => void dashboardQuery.refetch()}
        />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="visus:read">
      <div className="space-y-6">
        <PageHeader
          title={dashboard.name}
          description={dashboard.description}
          actions={
            <>
              <Button variant="outline" size="sm" asChild>
                <Link href="/visus/dashboards">
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back to dashboards
                </Link>
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={widgets.length < 2 || arrangeMutation.isPending}
                onClick={() => arrangeMutation.mutate(widgets)}
              >
                <Sparkles className="mr-2 h-4 w-4" />
                {arrangeMutation.isPending ? 'Arranging...' : 'Auto-arrange'}
              </Button>
              <Button
                size="sm"
                onClick={() => {
                  setEditingWidget(null);
                  setWidgetOpen(true);
                }}
              >
                <Plus className="mr-2 h-4 w-4" />
                Add Widget
              </Button>
            </>
          }
        />

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-4">
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Visibility</CardDescription>
              <CardTitle className="text-lg capitalize">{dashboard.visibility}</CardTitle>
            </CardHeader>
            <CardContent className="pt-0">
              <div className="flex flex-wrap gap-2">
                {dashboard.is_default ? <Badge variant="secondary">Default</Badge> : null}
                {dashboard.is_system ? <Badge variant="outline">System</Badge> : null}
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Widgets</CardDescription>
              <CardTitle className="text-lg">{widgets.length}</CardTitle>
            </CardHeader>
            <CardContent className="pt-0 text-sm text-muted-foreground">
              Grid columns: {dashboard.grid_columns}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Updated</CardDescription>
              <CardTitle className="text-lg">
                <RelativeTime date={dashboard.updated_at} />
              </CardTitle>
            </CardHeader>
            <CardContent className="pt-0 text-sm text-muted-foreground">
              Created <RelativeTime date={dashboard.created_at} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Tags</CardDescription>
              <CardTitle className="text-lg">{dashboard.tags.length}</CardTitle>
            </CardHeader>
            <CardContent className="pt-0">
              <div className="flex flex-wrap gap-2">
                {dashboard.tags.length > 0 ? dashboard.tags.map((tag) => <Badge key={tag} variant="outline">{tag}</Badge>) : <span className="text-sm text-muted-foreground">No tags</span>}
              </div>
            </CardContent>
          </Card>
        </div>

        {widgets.length === 0 ? (
          <EmptyState
            icon={LayoutTemplate}
            title="No widgets configured"
            description="Add widgets to make this dashboard useful to executive viewers."
            action={{
              label: 'Add Widget',
              onClick: () => {
                setEditingWidget(null);
                setWidgetOpen(true);
              },
            }}
          />
        ) : (
          <div className="grid auto-rows-[84px] grid-cols-12 gap-4">
            {widgets.map((widget) => (
              <Card
                key={widget.id}
                style={{
                  gridColumn: `span ${widget.position.w} / span ${widget.position.w}`,
                  gridRow: `span ${widget.position.h} / span ${widget.position.h}`,
                }}
                className="overflow-hidden"
              >
                <CardHeader className="space-y-2 pb-2">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <CardTitle className="truncate text-base">{widget.title}</CardTitle>
                      {widget.subtitle ? <CardDescription>{widget.subtitle}</CardDescription> : null}
                    </div>
                    <Badge variant="outline">{widget.type.replace(/_/g, ' ')}</Badge>
                  </div>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                    <span>
                      x:{widget.position.x} y:{widget.position.y}
                    </span>
                    <span>
                      {widget.position.w}x{widget.position.h}
                    </span>
                    <span>{widget.refresh_interval_seconds}s refresh</span>
                  </div>
                  <pre className="max-h-36 overflow-auto rounded-lg border bg-muted/40 p-3 text-[11px] leading-5">
                    {safeJsonPreview(widget.config)}
                  </pre>
                  <div className="flex flex-wrap gap-2">
                    <Button variant="outline" size="sm" onClick={() => setPreviewWidget(widget)}>
                      Preview Data
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setEditingWidget(widget);
                        setWidgetOpen(true);
                      }}
                    >
                      Edit
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(widget)}>
                      <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                      Delete
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}

        <WidgetFormDialog
          open={widgetOpen}
          onOpenChange={(open) => {
            setWidgetOpen(open);
            if (!open) {
              setEditingWidget(null);
            }
          }}
          dashboard={dashboard}
          widget={editingWidget}
          widgetTypes={widgetTypes}
          kpis={kpis}
          pending={createMutation.isPending || updateMutation.isPending}
          onSubmit={async (payload) => {
            if (editingWidget) {
              // Strip `type` — backend UpdateWidgetRequest does not accept it
              // (DecodeJSON uses DisallowUnknownFields).
              const { type: _type, ...updatePayload } = payload as Record<string, unknown>;
              await updateMutation.mutateAsync({ id: editingWidget.id, payload: updatePayload });
              return;
            }
            await createMutation.mutateAsync(payload);
          }}
        />

        <WidgetPreviewDialog
          dashboardId={dashboardId}
          widget={previewWidget}
          open={Boolean(previewWidget)}
          onOpenChange={(open) => {
            if (!open) setPreviewWidget(null);
          }}
        />

        <ConfirmDialog
          open={Boolean(deleteTarget)}
          onOpenChange={(open) => {
            if (!open) setDeleteTarget(null);
          }}
          title="Delete Widget"
          description={`Delete "${deleteTarget?.title}" from this dashboard?`}
          confirmLabel="Delete Widget"
          variant="destructive"
          loading={deleteMutation.isPending}
          onConfirm={async () => {
            if (!deleteTarget) return;
            await deleteMutation.mutateAsync(deleteTarget.id);
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
