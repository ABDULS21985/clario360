'use client';

import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Server, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useDataTable } from '@/hooks/use-data-table';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { ColumnDef } from '@tanstack/react-table';
import type {
  AIInferenceServer,
  ComputeBackendType,
  InferenceServerStatus,
} from '@/types/ai-governance';
import { ModelCard } from '../_components/model-card';

const BACKEND_TYPES: { label: string; value: ComputeBackendType }[] = [
  { label: 'Inline Go', value: 'inline_go' },
  { label: 'vLLM GPU', value: 'vllm_gpu' },
  { label: 'vLLM CPU', value: 'vllm_cpu' },
  { label: 'llama.cpp CPU', value: 'llamacpp_cpu' },
  { label: 'llama.cpp GPU', value: 'llamacpp_gpu' },
  { label: 'BitNet CPU', value: 'bitnet_cpu' },
  { label: 'ONNX CPU', value: 'onnx_cpu' },
  { label: 'ONNX GPU', value: 'onnx_gpu' },
];

const STATUS_VARIANTS: Record<InferenceServerStatus, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  provisioning: 'outline',
  healthy: 'default',
  degraded: 'secondary',
  offline: 'destructive',
  decommissioned: 'secondary',
};

function backendLabel(type: ComputeBackendType): string {
  return BACKEND_TYPES.find((bt) => bt.value === type)?.label ?? type;
}

export default function ComputePage() {
  const [formOpen, setFormOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<AIInferenceServer | null>(null);
  const [deleting, setDeleting] = useState(false);

  // Form state
  const [name, setName] = useState('');
  const [backendType, setBackendType] = useState<ComputeBackendType>('llamacpp_cpu');
  const [baseUrl, setBaseUrl] = useState('');
  const [healthEndpoint, setHealthEndpoint] = useState('/health');
  const [modelName, setModelName] = useState('');
  const [quantization, setQuantization] = useState('');
  const [cpuCores, setCpuCores] = useState('');
  const [memoryMb, setMemoryMb] = useState('');
  const [gpuType, setGpuType] = useState('');
  const [gpuCount, setGpuCount] = useState('0');
  const [maxConcurrent, setMaxConcurrent] = useState('4');

  const { tableProps, refetch } = useDataTable<AIInferenceServer>({
    queryKey: 'ai-inference-servers',
    fetchFn: (params) => enterpriseApi.ai.listServers(params),
    defaultPageSize: 20,
    defaultSort: { column: 'name', direction: 'asc' },
  });

  const serversQuery = useQuery({
    queryKey: ['ai-inference-servers-all'],
    queryFn: () => enterpriseApi.ai.listServers({ page: 1, per_page: 200 }),
  });

  const allServers = serversQuery.data?.data ?? [];
  const healthyCount = allServers.filter((s) => s.status === 'healthy').length;
  const cpuCount = allServers.filter((s) =>
    ['llamacpp_cpu', 'bitnet_cpu', 'onnx_cpu', 'vllm_cpu'].includes(s.backend_type),
  ).length;
  const gpuCountTotal = allServers.filter((s) =>
    ['vllm_gpu', 'llamacpp_gpu', 'onnx_gpu'].includes(s.backend_type),
  ).length;

  const resetForm = () => {
    setName('');
    setBackendType('llamacpp_cpu');
    setBaseUrl('');
    setHealthEndpoint('/health');
    setModelName('');
    setQuantization('');
    setCpuCores('');
    setMemoryMb('');
    setGpuType('');
    setGpuCount('0');
    setMaxConcurrent('4');
  };

  const handleCreate = async () => {
    try {
      setSaving(true);
      await enterpriseApi.ai.createServer({
        name,
        backend_type: backendType,
        base_url: baseUrl,
        health_endpoint: healthEndpoint,
        model_name: modelName || null,
        quantization: quantization || null,
        cpu_cores: cpuCores ? Number(cpuCores) : null,
        memory_mb: memoryMb ? Number(memoryMb) : null,
        gpu_type: gpuType || null,
        gpu_count: Number(gpuCount) || 0,
        max_concurrent: Number(maxConcurrent) || 4,
        metadata: {},
      });
      showSuccess('Inference server registered.');
      setFormOpen(false);
      resetForm();
      await Promise.all([refetch(), serversQuery.refetch()]);
    } catch (error) {
      showApiError(error);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      setDeleting(true);
      await enterpriseApi.ai.deleteServer(deleteTarget.id);
      showSuccess('Inference server decommissioned.');
      setDeleteTarget(null);
      await Promise.all([refetch(), serversQuery.refetch()]);
    } catch (error) {
      showApiError(error);
    } finally {
      setDeleting(false);
    }
  };

  const handleStatusChange = async (server: AIInferenceServer, status: string) => {
    try {
      await enterpriseApi.ai.updateServerStatus(server.id, { status });
      showSuccess(`Server status updated to ${status}.`);
      await Promise.all([refetch(), serversQuery.refetch()]);
    } catch (error) {
      showApiError(error);
    }
  };

  const columns: ColumnDef<AIInferenceServer>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: ({ row }) => (
          <div>
            <p className="font-medium">{row.original.name}</p>
            <p className="text-xs text-muted-foreground">{row.original.base_url}</p>
          </div>
        ),
      },
      {
        accessorKey: 'backend_type',
        header: 'Backend',
        cell: ({ row }) => (
          <Badge variant="outline">{backendLabel(row.original.backend_type)}</Badge>
        ),
      },
      {
        accessorKey: 'model_name',
        header: 'Model',
        cell: ({ row }) => (
          <div className="text-sm">
            <p>{row.original.model_name ?? '—'}</p>
            {row.original.quantization && (
              <p className="text-xs text-muted-foreground">{row.original.quantization}</p>
            )}
          </div>
        ),
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: ({ row }) => (
          <Badge variant={STATUS_VARIANTS[row.original.status]}>
            {row.original.status}
          </Badge>
        ),
      },
      {
        id: 'resources',
        header: 'Resources',
        cell: ({ row }) => {
          const s = row.original;
          const parts: string[] = [];
          if (s.cpu_cores) parts.push(`${s.cpu_cores} CPU`);
          if (s.memory_mb) parts.push(`${s.memory_mb} MB`);
          if (s.gpu_count > 0) parts.push(`${s.gpu_count}× ${s.gpu_type ?? 'GPU'}`);
          return <span className="text-sm text-muted-foreground">{parts.join(' · ') || '—'}</span>;
        },
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => (
          <div className="flex items-center gap-1">
            {row.original.status !== 'healthy' && row.original.status !== 'decommissioned' && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleStatusChange(row.original, 'healthy')}
              >
                <RefreshCw className="mr-1 h-3 w-3" />
                Mark Healthy
              </Button>
            )}
            {row.original.status !== 'decommissioned' && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setDeleteTarget(row.original)}
              >
                <Trash2 className="h-3 w-3 text-destructive" />
              </Button>
            )}
          </div>
        ),
      },
    ],
    [],
  );

  return (
    <PermissionRedirect permission="users:read">
      <div className="space-y-6">
        <PageHeader
          title="Compute Infrastructure"
          description="Manage inference servers for CPU and GPU model serving. Register llama.cpp, BitNet, vLLM, and ONNX endpoints."
          actions={
            <div className="flex items-center gap-2">
              <Button onClick={() => setFormOpen(true)}>
                <Plus className="mr-1.5 h-4 w-4" />
                Add Server
              </Button>
              <Button variant="outline" onClick={() => void Promise.all([refetch(), serversQuery.refetch()])}>
                Refresh
              </Button>
            </div>
          }
        />

        <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <ModelCard label="Total Servers" value={allServers.length} helper="Registered inference endpoints." />
          <ModelCard label="Healthy" value={healthyCount} helper="Endpoints responding to health checks." />
          <ModelCard label="CPU Backends" value={cpuCount} helper="llama.cpp, BitNet, ONNX, vLLM CPU." />
          <ModelCard label="GPU Backends" value={gpuCountTotal} helper="vLLM GPU, llama.cpp GPU, ONNX GPU." />
        </section>

        <div className="rounded-3xl border border-border/70 bg-[radial-gradient(circle_at_top_left,_rgba(15,118,110,0.14),_transparent_38%),radial-gradient(circle_at_bottom_right,_rgba(180,83,9,0.12),_transparent_34%)] p-6">
          <div className="mb-4 flex items-center gap-3">
            <div className="rounded-2xl bg-primary/10 p-3 text-primary">
              <Server className="h-6 w-6" />
            </div>
            <div>
              <h2 className="text-xl font-semibold">Inference Servers</h2>
              <p className="text-sm text-muted-foreground">
                Each server exposes an OpenAI-compatible /v1/chat/completions endpoint used for benchmarking and production inference.
              </p>
            </div>
          </div>
          <DataTable
            {...tableProps}
            columns={columns}
            onSortChange={() => undefined}
            emptyState={{
              icon: Server,
              title: 'No inference servers registered',
              description: 'Add a llama.cpp, BitNet, or vLLM server to start benchmarking CPU vs GPU inference.',
            }}
          />
        </div>
      </div>

      {/* Register Server Dialog */}
      <Dialog open={formOpen} onOpenChange={(open) => { if (!open) resetForm(); setFormOpen(open); }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Register Inference Server</DialogTitle>
            <DialogDescription>
              Add a new inference endpoint for benchmarking and production routing.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="server-name">Name</Label>
              <Input id="server-name" placeholder="e.g. llamacpp-cpu-prod-01" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label>Backend Type</Label>
              <Select value={backendType} onValueChange={(v) => setBackendType(v as ComputeBackendType)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {BACKEND_TYPES.map((bt) => (
                    <SelectItem key={bt.value} value={bt.value}>{bt.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="base-url">Base URL</Label>
              <Input id="base-url" placeholder="http://localhost:8081/v1" value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="health-endpoint">Health Endpoint</Label>
              <Input id="health-endpoint" value={healthEndpoint} onChange={(e) => setHealthEndpoint(e.target.value)} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="model-name">Model Name</Label>
                <Input id="model-name" placeholder="llama-3.1-8b-instruct" value={modelName} onChange={(e) => setModelName(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="quantization">Quantization</Label>
                <Input id="quantization" placeholder="Q4_0" value={quantization} onChange={(e) => setQuantization(e.target.value)} />
              </div>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="cpu-cores">CPU Cores</Label>
                <Input id="cpu-cores" type="number" value={cpuCores} onChange={(e) => setCpuCores(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="memory-mb">Memory (MB)</Label>
                <Input id="memory-mb" type="number" value={memoryMb} onChange={(e) => setMemoryMb(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="max-concurrent">Max Concurrent</Label>
                <Input id="max-concurrent" type="number" value={maxConcurrent} onChange={(e) => setMaxConcurrent(e.target.value)} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="gpu-type">GPU Type</Label>
                <Input id="gpu-type" placeholder="A100" value={gpuType} onChange={(e) => setGpuType(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="gpu-count">GPU Count</Label>
                <Input id="gpu-count" type="number" value={gpuCount} onChange={(e) => setGpuCount(e.target.value)} />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setFormOpen(false); resetForm(); }}>Cancel</Button>
            <Button onClick={handleCreate} disabled={saving || !name || !baseUrl}>
              {saving ? 'Registering…' : 'Register Server'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <Dialog open={Boolean(deleteTarget)} onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Decommission Server</DialogTitle>
            <DialogDescription>
              This will mark <strong>{deleteTarget?.name}</strong> as decommissioned. It will no longer receive traffic.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>Cancel</Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? 'Decommissioning…' : 'Decommission'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PermissionRedirect>
  );
}
