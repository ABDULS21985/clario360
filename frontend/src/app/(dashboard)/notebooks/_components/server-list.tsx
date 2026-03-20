'use client';

import { formatDistanceToNow } from 'date-fns';
import { ExternalLink, Square } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { ResourceUsage } from './resource-usage';
import type { NotebookServer } from '@/lib/notebooks';

function formatProfileLabel(profile: string): string {
  return profile
    .split('-')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

interface ServerListProps {
  servers: NotebookServer[];
  busyServerId: string | null;
  onStop: (server: NotebookServer) => void;
}

export function ServerList({ servers, busyServerId, onStop }: ServerListProps) {
  if (servers.length === 0) {
    return (
      <Card className="border-dashed">
        <CardContent className="py-10 text-sm text-muted-foreground">
          No active notebook server. Launch one to open JupyterLab and copy a template into your workspace.
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {servers.map((server) => (
        <Card key={server.id} className="overflow-hidden border-border/70">
          <CardHeader className="flex flex-row items-start justify-between gap-4">
            <div className="space-y-2">
              <div className="flex items-center gap-3">
                <CardTitle className="text-lg">{formatProfileLabel(server.profile || 'notebook-server')}</CardTitle>
                <Badge variant={server.status === 'running' ? 'default' : 'secondary'}>{server.status}</Badge>
              </div>
              <p className="text-sm text-muted-foreground">
                Started {server.started_at ? formatDistanceToNow(new Date(server.started_at), { addSuffix: true }) : 'recently'}
                {' · '}
                Last activity {server.last_activity ? formatDistanceToNow(new Date(server.last_activity), { addSuffix: true }) : 'unknown'}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Button asChild variant="outline" size="sm">
                <a href={server.url} target="_blank" rel="noreferrer">
                  Open JupyterLab
                  <ExternalLink className="ml-2 h-4 w-4" />
                </a>
              </Button>
              <Button
                variant="destructive"
                size="sm"
                disabled={busyServerId === server.id}
                onClick={() => onStop(server)}
              >
                <Square className="mr-2 h-4 w-4" />
                Stop Server
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <ResourceUsage
              cpuPercent={server.cpu_percent}
              memoryMB={server.memory_mb}
              memoryLimitMB={server.memory_limit_mb}
            />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
