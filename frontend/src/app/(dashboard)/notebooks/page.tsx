'use client';

import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, BookOpenText } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Card, CardContent } from '@/components/ui/card';
import { LaunchButton } from './_components/launch-button';
import { ProfileSelector } from './_components/profile-selector';
import { ServerList } from './_components/server-list';
import { TemplateGallery } from './_components/template-gallery';
import { notebookApi, type NotebookProfile, type NotebookServer, type NotebookTemplate } from '@/lib/notebooks';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { showApiError, showSuccess } from '@/lib/toast';

/** Poll every 5 s while any server is transitioning; every 30 s otherwise. */
const POLL_FAST_MS = 5_000;
const POLL_SLOW_MS = 30_000;

function hasTransitioningServer(servers: NotebookServer[] | undefined): boolean {
  return servers?.some((s) => s.status === 'starting' || s.status === 'stopping') ?? false;
}

export default function NotebookWorkspacePage() {
  const queryClient = useQueryClient();
  const [selectorOpen, setSelectorOpen] = useState(false);
  const [busyServerId, setBusyServerId] = useState<string | null>(null);
  const [busyTemplateId, setBusyTemplateId] = useState<string | null>(null);

  const healthQuery = useQuery({
    queryKey: ['notebook-health'],
    queryFn: notebookApi.checkHealth,
    refetchInterval: 60_000,
  });

  const profilesQuery = useQuery({
    queryKey: ['notebook-profiles'],
    queryFn: notebookApi.listProfiles,
  });
  const templatesQuery = useQuery({
    queryKey: ['notebook-templates'],
    queryFn: notebookApi.listTemplates,
  });

  // Adaptive polling: faster while a server is starting or stopping so the
  // UI reflects the state transition promptly without hammering the API at rest.
  const serversQuery = useQuery({
    queryKey: ['notebook-servers'],
    queryFn: notebookApi.listServers,
    refetchInterval: (query) =>
      hasTransitioningServer(query.state.data) ? POLL_FAST_MS : POLL_SLOW_MS,
  });

  const activeServer = useMemo<NotebookServer | null>(
    () => serversQuery.data?.find((server) => server.status === 'running' || server.status === 'starting') ?? null,
    [serversQuery.data],
  );

  const startMutation = useMutation({
    mutationFn: (profile: NotebookProfile) => notebookApi.startServer(profile.slug),
    onSuccess: async (server) => {
      showSuccess('Notebook server requested', `${server.profile} is starting now.`);
      setSelectorOpen(false);
      await queryClient.invalidateQueries({ queryKey: ['notebook-servers'] });
    },
    onError: showApiError,
  });

  const stopMutation = useMutation({
    mutationFn: (server: NotebookServer) => notebookApi.stopServer(server.id),
    // Set busy state immediately on mutation start so the button disables
    // before the async mutationFn even begins (prevents duplicate clicks).
    onMutate: (server) => setBusyServerId(server.id),
    onSuccess: async () => {
      showSuccess('Notebook server stopped');
      await queryClient.invalidateQueries({ queryKey: ['notebook-servers'] });
    },
    onError: showApiError,
    onSettled: () => setBusyServerId(null),
  });

  const copyMutation = useMutation({
    mutationFn: (template: NotebookTemplate) => {
      if (!activeServer) {
        throw new Error('Launch a notebook server before opening a template.');
      }
      return notebookApi.copyTemplate(activeServer.id, template.id);
    },
    onMutate: (template) => setBusyTemplateId(template.id),
    onSuccess: async (result) => {
      showSuccess('Template copied', 'Opening JupyterLab in a new tab.');
      window.open(result.open_url, '_blank', 'noopener,noreferrer');
      await queryClient.invalidateQueries({ queryKey: ['notebook-servers'] });
    },
    onError: showApiError,
    onSettled: () => setBusyTemplateId(null),
  });

  return (
    <PermissionRedirect permission="*:read">
      <div className="space-y-6">
        {healthQuery.data?.status === 'degraded' && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              JupyterHub is currently unreachable. Notebook servers may not start or respond correctly.
              {healthQuery.data.jupyterhub.error ? ` (${healthQuery.data.jupyterhub.error})` : ''}
            </AlertDescription>
          </Alert>
        )}
        <PageHeader
          title="Notebook Workspace"
          description="Secure Jupyter-based analysis for SOC investigation, model validation, and Spark-scale threat research."
          actions={<LaunchButton disabled={Boolean(activeServer)} onClick={() => setSelectorOpen(true)} />}
        />

        <Card className="overflow-hidden border-border/70 bg-[radial-gradient(circle_at_top_left,_rgba(15,118,110,0.14),_transparent_38%),radial-gradient(circle_at_bottom_right,_rgba(180,83,9,0.12),_transparent_34%)]">
          <CardContent className="space-y-6 p-4 sm:p-6">
            <div className="flex items-center gap-3">
              <div className="rounded-2xl bg-primary/10 p-3 text-primary">
                <BookOpenText className="h-6 w-6" />
              </div>
              <div>
                <h2 className="text-xl font-semibold">Active Servers</h2>
                <p className="text-sm text-muted-foreground">
                  {activeServer?.last_activity
                    ? `Current activity updated ${formatDistanceToNow(new Date(activeServer.last_activity), { addSuffix: true })}.`
                    : 'Launch an isolated notebook pod with governed data access and JupyterHub SSO.'}
                </p>
              </div>
            </div>
            <ServerList
              servers={serversQuery.data ?? []}
              busyServerId={busyServerId}
              onStop={(server) => stopMutation.mutate(server)}
            />
          </CardContent>
        </Card>

        <section className="space-y-4">
          <div>
            <h2 className="text-xl font-semibold">Notebook Templates</h2>
            <p className="text-sm text-muted-foreground">
              Copy one of the governed starter notebooks into your personal workspace and open it directly in JupyterLab.
            </p>
          </div>
          <TemplateGallery
            templates={templatesQuery.data ?? []}
            activeServer={activeServer}
            busyTemplateId={busyTemplateId}
            onOpenTemplate={(template) => copyMutation.mutate(template)}
          />
        </section>

        <ProfileSelector
          open={selectorOpen}
          onOpenChange={setSelectorOpen}
          profiles={profilesQuery.data ?? []}
          busy={startMutation.isPending}
          onSelect={(profile) => startMutation.mutate(profile)}
        />
      </div>
    </PermissionRedirect>
  );
}
