'use client';

import { useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, BarChart3, Play, GitCompare } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type {
  AIBenchmarkRun,
  AIBenchmarkComparison,
  BenchmarkRunStatus,
} from '@/types/ai-governance';
import { ModelCard } from '../../_components/model-card';

const RUN_STATUS_VARIANTS: Record<BenchmarkRunStatus, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  pending: 'outline',
  running: 'secondary',
  completed: 'default',
  failed: 'destructive',
  cancelled: 'secondary',
};

export default function BenchmarkSuiteDetailPage() {
  const router = useRouter();
  const params = useParams();
  const suiteId = params?.suiteId as string;

  const [runDialogOpen, setRunDialogOpen] = useState(false);
  const [selectedServer, setSelectedServer] = useState('');
  const [running, setRunning] = useState(false);
  const [compareIds, setCompareIds] = useState<Set<string>>(new Set());
  const [comparison, setComparison] = useState<AIBenchmarkComparison | null>(null);
  const [comparing, setComparing] = useState(false);

  const suiteQuery = useQuery({
    queryKey: ['ai-benchmark-suite', suiteId],
    queryFn: () => enterpriseApi.ai.getBenchmarkSuite(suiteId),
    enabled: Boolean(suiteId),
  });

  const runsQuery = useQuery({
    queryKey: ['ai-benchmark-runs', suiteId],
    queryFn: () => enterpriseApi.ai.listBenchmarkRuns({ page: 1, per_page: 100, filters: { suite_id: suiteId } }),
    enabled: Boolean(suiteId),
  });

  const serversQuery = useQuery({
    queryKey: ['ai-inference-servers-all'],
    queryFn: () => enterpriseApi.ai.listServers({ page: 1, per_page: 200 }),
  });

  const suite = suiteQuery.data;
  const runs = runsQuery.data?.data ?? [];
  const servers = serversQuery.data?.data ?? [];
  const healthyServers = servers.filter((s) => s.status === 'healthy');
  const completedRuns = runs.filter((r) => r.status === 'completed');

  const handleRunBenchmark = async () => {
    if (!selectedServer) return;
    try {
      setRunning(true);
      await enterpriseApi.ai.runBenchmark(suiteId, { server_id: selectedServer });
      showSuccess('Benchmark started.');
      setRunDialogOpen(false);
      setSelectedServer('');
      await runsQuery.refetch();
    } catch (error) {
      showApiError(error);
    } finally {
      setRunning(false);
    }
  };

  const toggleCompare = (runId: string) => {
    setCompareIds((prev) => {
      const next = new Set(prev);
      if (next.has(runId)) next.delete(runId);
      else next.add(runId);
      return next;
    });
  };

  const handleCompare = async () => {
    if (compareIds.size < 2) return;
    try {
      setComparing(true);
      const result = await enterpriseApi.ai.compareRuns({ run_ids: Array.from(compareIds) });
      setComparison(result);
    } catch (error) {
      showApiError(error);
    } finally {
      setComparing(false);
    }
  };

  const bestLatency = completedRuns.length > 0
    ? Math.min(...completedRuns.map((r) => r.p50_latency_ms ?? Infinity))
    : null;
  const bestThroughput = completedRuns.length > 0
    ? Math.max(...completedRuns.map((r) => r.tokens_per_second ?? 0))
    : null;

  return (
    <PermissionRedirect permission="users:read">
      <div className="space-y-6">
        <PageHeader
          title={suite?.name ?? 'Benchmark Suite'}
          description={suite?.description ?? 'Loading suite details…'}
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" onClick={() => router.push('/admin/ai-governance/benchmarks')}>
                <ArrowLeft className="mr-1.5 h-4 w-4" />
                Back
              </Button>
              <Button onClick={() => setRunDialogOpen(true)}>
                <Play className="mr-1.5 h-4 w-4" />
                Run Benchmark
              </Button>
              {compareIds.size >= 2 && (
                <Button variant="secondary" onClick={handleCompare} disabled={comparing}>
                  <GitCompare className="mr-1.5 h-4 w-4" />
                  {comparing ? 'Comparing…' : `Compare (${compareIds.size})`}
                </Button>
              )}
            </div>
          }
        />

        {suite && (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">Suite Configuration</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">Model:</span>{' '}
                  <Badge variant="outline">{suite.model_slug}</Badge>
                </div>
                <div><span className="text-muted-foreground">Warmup:</span> {suite.warmup_count}</div>
                <div><span className="text-muted-foreground">Iterations:</span> {suite.iteration_count}</div>
                <div><span className="text-muted-foreground">Concurrency:</span> {suite.concurrency}</div>
                <div><span className="text-muted-foreground">Timeout:</span> {suite.timeout_seconds}s</div>
                <div><span className="text-muted-foreground">Prompts:</span> {suite.dataset_size}</div>
              </div>
            </CardContent>
          </Card>
        )}

        <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <ModelCard label="Total Runs" value={runs.length} helper="Benchmark runs for this suite." />
          <ModelCard label="Completed" value={completedRuns.length} helper="Runs with full results." />
          <ModelCard
            label="Best p50 Latency"
            value={bestLatency != null && bestLatency < Infinity ? `${bestLatency.toFixed(0)} ms` : '—'}
            helper="Lowest median latency achieved."
          />
          <ModelCard
            label="Best Throughput"
            value={bestThroughput != null && bestThroughput > 0 ? `${bestThroughput.toFixed(1)} tok/s` : '—'}
            helper="Highest tokens/sec achieved."
          />
        </section>

        {/* Comparison Results */}
        {comparison && (
          <Card className="border-primary/30 bg-primary/5">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <GitCompare className="h-5 w-5" />
                Comparison Results
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-3">
                <div>
                  <p className="text-sm text-muted-foreground">Monthly Cost Delta</p>
                  <p className={`text-2xl font-semibold ${comparison.cost_delta_monthly_usd < 0 ? 'text-green-600' : 'text-red-600'}`}>
                    {comparison.cost_delta_monthly_usd < 0 ? '−' : '+'}${Math.abs(comparison.cost_delta_monthly_usd).toFixed(2)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Latency Delta</p>
                  <p className="text-2xl font-semibold">
                    {comparison.latency_delta_percent > 0 ? '+' : ''}{comparison.latency_delta_percent.toFixed(1)}%
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Recommendation</p>
                  <Badge
                    variant={comparison.recommendation === 'cpu_viable' ? 'default' : comparison.recommendation === 'gpu_required' ? 'destructive' : 'secondary'}
                    className="mt-1 text-sm"
                  >
                    {comparison.recommendation.replace(/_/g, ' ')}
                  </Badge>
                  <p className="mt-1 text-sm text-muted-foreground">{comparison.recommendation_reason}</p>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Runs Table */}
        <div className="rounded-3xl border border-border/70 bg-[radial-gradient(circle_at_top_left,_rgba(15,118,110,0.14),_transparent_38%),radial-gradient(circle_at_bottom_right,_rgba(180,83,9,0.12),_transparent_34%)] p-6">
          <div className="mb-4 flex items-center gap-3">
            <div className="rounded-2xl bg-primary/10 p-3 text-primary">
              <BarChart3 className="h-6 w-6" />
            </div>
            <div>
              <h2 className="text-xl font-semibold">Benchmark Runs</h2>
              <p className="text-sm text-muted-foreground">
                Select two or more completed runs to compare CPU vs GPU performance.
              </p>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-muted-foreground">
                  <th className="p-2 w-10"></th>
                  <th className="p-2">Backend</th>
                  <th className="p-2">Model</th>
                  <th className="p-2">Status</th>
                  <th className="p-2">p50</th>
                  <th className="p-2">p95</th>
                  <th className="p-2">p99</th>
                  <th className="p-2">Tok/s</th>
                  <th className="p-2">Req/s</th>
                  <th className="p-2">CPU %</th>
                  <th className="p-2">Mem (MB)</th>
                  <th className="p-2">$/hr</th>
                  <th className="p-2">Date</th>
                </tr>
              </thead>
              <tbody>
                {runs.map((run) => (
                  <tr key={run.id} className="border-b hover:bg-muted/30">
                    <td className="p-2">
                      {run.status === 'completed' && (
                        <Checkbox
                          checked={compareIds.has(run.id)}
                          onCheckedChange={() => toggleCompare(run.id)}
                        />
                      )}
                    </td>
                    <td className="p-2"><Badge variant="outline">{run.backend_type}</Badge></td>
                    <td className="p-2">
                      {run.model_name}
                      {run.quantization && <span className="ml-1 text-muted-foreground">({run.quantization})</span>}
                    </td>
                    <td className="p-2"><Badge variant={RUN_STATUS_VARIANTS[run.status]}>{run.status}</Badge></td>
                    <td className="p-2 font-mono">{run.p50_latency_ms?.toFixed(0) ?? '—'}</td>
                    <td className="p-2 font-mono">{run.p95_latency_ms?.toFixed(0) ?? '—'}</td>
                    <td className="p-2 font-mono">{run.p99_latency_ms?.toFixed(0) ?? '—'}</td>
                    <td className="p-2 font-mono">{run.tokens_per_second?.toFixed(1) ?? '—'}</td>
                    <td className="p-2 font-mono">{run.requests_per_second?.toFixed(1) ?? '—'}</td>
                    <td className="p-2 font-mono">{run.peak_cpu_percent?.toFixed(0) ?? '—'}</td>
                    <td className="p-2 font-mono">{run.peak_memory_mb ?? '—'}</td>
                    <td className="p-2 font-mono">{run.estimated_hourly_cost_usd?.toFixed(2) ?? '—'}</td>
                    <td className="p-2 text-muted-foreground">{new Date(run.created_at).toLocaleDateString()}</td>
                  </tr>
                ))}
                {runs.length === 0 && (
                  <tr>
                    <td colSpan={13} className="p-8 text-center text-muted-foreground">
                      No benchmark runs yet. Click &quot;Run Benchmark&quot; to start.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Run Benchmark Dialog */}
      <Dialog open={runDialogOpen} onOpenChange={(open) => { if (!open) setSelectedServer(''); setRunDialogOpen(open); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Run Benchmark</DialogTitle>
            <DialogDescription>
              Execute this suite against an inference server.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label>Select Server</Label>
              <Select value={selectedServer} onValueChange={setSelectedServer}>
                <SelectTrigger><SelectValue placeholder="Choose an inference server" /></SelectTrigger>
                <SelectContent>
                  {healthyServers.map((s) => (
                    <SelectItem key={s.id} value={s.id}>
                      {s.name} ({s.backend_type})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setRunDialogOpen(false); setSelectedServer(''); }}>Cancel</Button>
            <Button onClick={handleRunBenchmark} disabled={running || !selectedServer}>
              {running ? 'Starting…' : 'Start Benchmark'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PermissionRedirect>
  );
}
