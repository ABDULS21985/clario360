'use client';

import { useEffect, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Bot, HeartPulse, Loader2, Settings2, Waypoints } from 'lucide-react';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { Textarea } from '@/components/ui/textarea';
import { useAuth } from '@/hooks/use-auth';
import { apiGet, apiPost, apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatCompactNumber, formatCurrency, parseApiError } from '@/lib/format';
import { cn, formatDateTime } from '@/lib/utils';
import type {
  VCISOLLMConfigRequest,
  VCISOLLMConfigResponse,
  VCISOLLMHealth,
  VCISOLLMPromptVersion,
  VCISOLLMPromptVersionRequest,
  VCISOLLMUsage,
} from '@/types/cyber';

const DEFAULT_TEMPERATURE = '0.1';

export function LLMOpsPanel() {
  const { hasPermission } = useAuth();
  const canAdmin = hasPermission('vciso:llm:admin') || hasPermission('admin:*') || hasPermission('*');

  const [provider, setProvider] = useState('openai');
  const [model, setModel] = useState('');
  const [temperature, setTemperature] = useState(DEFAULT_TEMPERATURE);
  const [promptVersion, setPromptVersion] = useState('');
  const [promptDescription, setPromptDescription] = useState('');
  const [promptText, setPromptText] = useState('');

  const healthQuery = useQuery({
    queryKey: ['vciso-llm-health'],
    queryFn: () => apiGet<VCISOLLMHealth>(API_ENDPOINTS.CYBER_VCISO_LLM_HEALTH),
    staleTime: 30_000,
    refetchInterval: 30_000,
  });

  const usageQuery = useQuery({
    queryKey: ['vciso-llm-usage'],
    queryFn: () => apiGet<VCISOLLMUsage>(API_ENDPOINTS.CYBER_VCISO_LLM_USAGE),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const promptsQuery = useQuery({
    queryKey: ['vciso-llm-prompts'],
    queryFn: () => apiGet<VCISOLLMPromptVersion[]>(API_ENDPOINTS.CYBER_VCISO_LLM_PROMPTS),
    enabled: canAdmin,
    staleTime: 30_000,
  });

  useEffect(() => {
    if (!healthQuery.data) {
      return;
    }
    if (!model) {
      setProvider(healthQuery.data.provider);
      setModel(healthQuery.data.model);
    }
  }, [healthQuery.data, model]);

  const configMutation = useMutation({
    mutationFn: (payload: VCISOLLMConfigRequest) =>
      apiPut<VCISOLLMConfigResponse>(API_ENDPOINTS.CYBER_VCISO_LLM_CONFIG, payload),
    onSuccess: (response) => {
      toast.success('LLM provider settings updated');
      setProvider(response.provider);
      setModel(response.model);
      setTemperature(String(response.temperature));
      void healthQuery.refetch();
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });

  const createPromptMutation = useMutation({
    mutationFn: (payload: VCISOLLMPromptVersionRequest) =>
      apiPost<VCISOLLMPromptVersion>(API_ENDPOINTS.CYBER_VCISO_LLM_PROMPTS, payload),
    onSuccess: () => {
      toast.success('Prompt version created');
      setPromptVersion('');
      setPromptDescription('');
      setPromptText('');
      void promptsQuery.refetch();
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });

  const activatePromptMutation = useMutation({
    mutationFn: (version: string) =>
      apiPut<void>(`${API_ENDPOINTS.CYBER_VCISO_LLM_PROMPTS}/${encodeURIComponent(version)}/activate`),
    onSuccess: () => {
      toast.success('Prompt version activated');
      void promptsQuery.refetch();
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });

  const health = healthQuery.data;
  const usage = usageQuery.data;
  const promptVersions = promptsQuery.data ?? [];

  function handleSaveConfig() {
    const parsedTemperature = Number.parseFloat(temperature);
    if (!provider.trim() || !model.trim() || Number.isNaN(parsedTemperature)) {
      toast.error('Provider, model, and a valid temperature are required.');
      return;
    }
    configMutation.mutate({
      provider: provider.trim(),
      model: model.trim(),
      temperature: parsedTemperature,
    });
  }

  function handleCreatePrompt() {
    if (!promptVersion.trim() || !promptText.trim()) {
      toast.error('Version and prompt text are required.');
      return;
    }
    createPromptMutation.mutate({
      version: promptVersion.trim(),
      description: promptDescription.trim(),
      prompt_text: promptText.trim(),
    });
  }

  return (
    <section className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h3 className="text-lg font-semibold tracking-tight">LLM Operations</h3>
          <p className="text-sm text-muted-foreground">
            Surface provider health, token usage, and prompt governance from the existing vCISO LLM backend.
          </p>
        </div>
        <Badge variant="outline" className="w-fit rounded-full">
          Audit traces open from assistant messages
        </Badge>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          icon={HeartPulse}
          title="Provider Health"
          value={healthQuery.isLoading ? 'Loading...' : health?.status ?? 'Unavailable'}
          detail={health ? `${health.provider} · ${health.model}` : 'Health endpoint pending'}
          badge={health ? { label: health.status, className: statusClass(health.status) } : undefined}
        />
        <MetricCard
          icon={Bot}
          title="Latency"
          value={health ? `${health.latency_ms}ms` : '—'}
          detail={health ? `${formatCompactNumber(Math.max(health.rate_limit_remaining, 0))} rate limit remaining` : 'No provider telemetry yet'}
        />
        <MetricCard
          icon={Waypoints}
          title="Usage Today"
          value={usage ? formatCompactNumber(usage.tokens_today) : '—'}
          detail={usage ? `${formatCompactNumber(usage.calls_today)} calls · ${formatCurrency(usage.cost_today)}` : 'Usage endpoint pending'}
        />
        <MetricCard
          icon={Settings2}
          title="Usage This Month"
          value={usage ? formatCurrency(usage.cost_this_month) : '—'}
          detail={usage ? `${formatCompactNumber(usage.calls_this_month)} routed calls` : 'Monthly totals unavailable'}
        />
      </div>

      {!canAdmin ? (
        <Card className="border-dashed">
          <CardContent className="p-4 text-sm text-muted-foreground sm:p-6">
            LLM admin controls require the `vciso:llm:admin` permission. Health, usage, engine selection, and message-level traces remain visible here.
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
          <Card className="border-border/70">
            <CardHeader>
              <CardTitle>Provider Configuration</CardTitle>
              <CardDescription>
                Update the active provider override used by the tenant-level LLM manager.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="llm-provider">Provider</Label>
                <Select value={provider} onValueChange={setProvider}>
                  <SelectTrigger id="llm-provider">
                    <SelectValue placeholder="Select provider" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="openai">OpenAI</SelectItem>
                    <SelectItem value="anthropic">Anthropic</SelectItem>
                    <SelectItem value="azure">Azure OpenAI</SelectItem>
                    <SelectItem value="local">Local</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="llm-model">Model</Label>
                <Input
                  id="llm-model"
                  value={model}
                  onChange={(event) => setModel(event.target.value)}
                  placeholder="gpt-4o"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="llm-temperature">Temperature</Label>
                <Input
                  id="llm-temperature"
                  type="number"
                  step="0.1"
                  min="0"
                  max="2"
                  value={temperature}
                  onChange={(event) => setTemperature(event.target.value)}
                />
              </div>

              <div className="rounded-2xl border border-dashed bg-slate-50/80 p-3 text-xs text-muted-foreground">
                Current health check: {health ? `${health.provider} / ${health.model} at ${health.latency_ms}ms` : 'waiting for provider telemetry'}.
              </div>

              <Button
                type="button"
                onClick={handleSaveConfig}
                disabled={configMutation.isPending}
              >
                {configMutation.isPending && <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />}
                Save Provider Override
              </Button>
            </CardContent>
          </Card>

          <Card className="border-border/70">
            <CardHeader>
              <CardTitle>Prompt Versions</CardTitle>
              <CardDescription>
                Review prompt versions, activate a candidate, or register a new system prompt.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="space-y-3">
                {promptsQuery.isLoading ? (
                  <div className="flex items-center gap-3 rounded-2xl border bg-slate-50 px-4 py-3 text-sm text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Loading prompt versions...
                  </div>
                ) : promptVersions.length === 0 ? (
                  <div className="rounded-2xl border border-dashed px-4 py-3 text-sm text-muted-foreground">
                    No prompt versions registered yet.
                  </div>
                ) : (
                  promptVersions
                    .slice()
                    .sort((left, right) => right.created_at.localeCompare(left.created_at))
                    .map((prompt) => (
                      <div key={prompt.id} className="rounded-2xl border p-4">
                        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                          <div className="space-y-1">
                            <div className="flex flex-wrap items-center gap-2">
                              <p className="text-sm font-semibold">{prompt.version}</p>
                              {prompt.active && (
                                <Badge className="rounded-full bg-emerald-600 text-white hover:bg-emerald-600">
                                  Active
                                </Badge>
                              )}
                            </div>
                            <p className="text-sm text-muted-foreground">
                              {prompt.description?.trim() || 'No description provided.'}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              Created by {prompt.created_by} on {formatDateTime(prompt.created_at)}
                            </p>
                          </div>
                          {!prompt.active && (
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              onClick={() => activatePromptMutation.mutate(prompt.version)}
                              disabled={activatePromptMutation.isPending}
                            >
                              Activate
                            </Button>
                          )}
                        </div>
                      </div>
                    ))
                )}
              </div>

              <Separator />

              <div className="space-y-4">
                <div>
                  <h4 className="text-sm font-semibold">Create Prompt Version</h4>
                  <p className="text-sm text-muted-foreground">
                    Register a new version; activation remains explicit.
                  </p>
                </div>
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="prompt-version">Version</Label>
                    <Input
                      id="prompt-version"
                      value={promptVersion}
                      onChange={(event) => setPromptVersion(event.target.value)}
                      placeholder="v1.1"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="prompt-description">Description</Label>
                    <Input
                      id="prompt-description"
                      value={promptDescription}
                      onChange={(event) => setPromptDescription(event.target.value)}
                      placeholder="Executive routing adjustments"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="prompt-text">Prompt Text</Label>
                  <Textarea
                    id="prompt-text"
                    value={promptText}
                    onChange={(event) => setPromptText(event.target.value.slice(0, 100000))}
                    placeholder="You are the vCISO assistant..."
                    className="min-h-[220px]"
                  />
                  <p className="text-xs text-muted-foreground">{promptText.length}/100000</p>
                </div>
                <Button
                  type="button"
                  onClick={handleCreatePrompt}
                  disabled={createPromptMutation.isPending}
                >
                  {createPromptMutation.isPending && <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />}
                  Create Prompt Version
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </section>
  );
}

function MetricCard({
  icon: Icon,
  title,
  value,
  detail,
  badge,
}: {
  icon: typeof Bot;
  title: string;
  value: string;
  detail: string;
  badge?: { label: string; className?: string };
}) {
  return (
    <Card className="border-border/70">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-sm text-muted-foreground">
          <Icon className="h-4 w-4" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-start justify-between gap-3">
          <p className="text-2xl font-semibold tracking-tight">{value}</p>
          {badge && (
            <Badge variant="outline" className={cn('rounded-full', badge.className)}>
              {badge.label}
            </Badge>
          )}
        </div>
        <p className="mt-2 text-sm text-muted-foreground">{detail}</p>
      </CardContent>
    </Card>
  );
}

function statusClass(status: string): string {
  switch (status.toLowerCase()) {
    case 'healthy':
    case 'ok':
      return 'border-emerald-200 text-emerald-700';
    case 'degraded':
      return 'border-amber-200 text-amber-700';
    case 'unavailable':
    case 'down':
      return 'border-rose-200 text-rose-700';
    default:
      return 'border-border text-foreground';
  }
}
