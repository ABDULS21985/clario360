import { apiDelete, apiGet, apiPost } from '@/lib/api';

export interface NotebookProfile {
  slug: string;
  display_name: string;
  description: string;
  cpu: string;
  memory: string;
  storage: string;
  requires_role?: string[];
  spark_enabled: boolean;
  default: boolean;
}

export interface NotebookTemplate {
  id: string;
  title: string;
  description: string;
  difficulty: 'beginner' | 'intermediate' | 'advanced';
  tags: string[];
  filename: string;
}

export interface NotebookServer {
  id: string;
  profile: string;
  status: 'starting' | 'running' | 'stopping' | 'stopped';
  url: string;
  started_at?: string;
  last_activity?: string;
  cpu_percent: number;
  memory_mb: number;
  memory_limit_mb: number;
}

export interface NotebookServerStatus {
  id: string;
  profile: string;
  status: 'starting' | 'running' | 'stopping' | 'stopped';
  cpu_percent: number;
  memory_mb: number;
  memory_limit_mb: number;
  uptime_seconds: number;
  last_activity?: string;
}

export interface CopiedTemplate {
  template_id: string;
  path: string;
  open_url: string;
}

export const notebookApi = {
  listProfiles: () => apiGet<NotebookProfile[]>('/api/v1/notebooks/profiles'),
  listTemplates: () => apiGet<NotebookTemplate[]>('/api/v1/notebooks/templates'),
  listServers: () => apiGet<NotebookServer[]>('/api/v1/notebooks/servers'),
  startServer: (profile: string) => apiPost<NotebookServer>('/api/v1/notebooks/servers', { profile }),
  stopServer: (id: string) => apiDelete<{ message: string }>(`/api/v1/notebooks/servers/${id}`),
  getStatus: (id: string) => apiGet<NotebookServerStatus>(`/api/v1/notebooks/servers/${id}/status`),
  copyTemplate: (id: string, templateId: string) =>
    apiPost<CopiedTemplate>(`/api/v1/notebooks/servers/${id}/copy-template`, { template_id: templateId }),
};
