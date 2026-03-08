'use client';

import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { BookOpenText } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Card, CardContent } from '@/components/ui/card';
import { LaunchButton } from './_components/launch-button';
import { ProfileSelector } from './_components/profile-selector';
import { ServerList } from './_components/server-list';
import { TemplateGallery } from './_components/template-gallery';
import { notebookApi, type NotebookProfile, type NotebookServer, type NotebookTemplate } from '@/lib/notebooks';
import { showApiError, showSuccess } from '@/lib/toast';

export default function NotebookWorkspacePage() {
  const queryClient = useQueryClient();
  const [selectorOpen, setSelectorOpen] = useState(false);
  const [busyServerId, setBusyServerId] = useState<string | null>(null);
  const [busyTemplateId, setBusyTemplateId] = useState<string | null>(null);

  const profilesQuery = useQuery({
    queryKey: ['notebook-profiles'],
    queryFn: notebookApi.listProfiles,
  });
  const templatesQuery = useQuery({
    queryKey: ['notebook-templates'],
    queryFn: notebookApi.listTemplates,
  });
  const serversQuery = useQuery({
    queryKey: ['notebook-servers'],
    queryFn: notebookApi.listServers,
    refetchInterval: 30000,
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
    mutationFn: async (server: NotebookServer) => {
      setBusyServerId(server.id);
      return notebookApi.stopServer(server.id);
    },
    onSuccess: async () => {
      showSuccess('Notebook server stopped');
      await queryClient.invalidateQueries({ queryKey: ['notebook-servers'] });
    },
    onError: showApiError,
    onSettled: () => setBusyServerId(null),
  });

  const copyMutation = useMutation({
    mutationFn: async (template: NotebookTemplate) => {
      if (!activeServer) {
        throw new Error('Launch a notebook server before opening a template.');
      }
      setBusyTemplateId(template.id);
      return notebookApi.copyTemplate(activeServer.id, template.id);
    },
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
        <PageHeader
          title="Notebook Workspace"
          description="Secure Jupyter-based analysis for SOC investigation, model validation, and Spark-scale threat research."
          actions={<LaunchButton disabled={Boolean(activeServer)} onClick={() => setSelectorOpen(true)} />}
        />

        <Card className="overflow-hidden border-border/70 bg-[radial-gradient(circle_at_top_left,_rgba(15,118,110,0.14),_transparent_38%),radial-gradient(circle_at_bottom_right,_rgba(180,83,9,0.12),_transparent_34%)]">
          <CardContent className="space-y-6 p-6">
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
