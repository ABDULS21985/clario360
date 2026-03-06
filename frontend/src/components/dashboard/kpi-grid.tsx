'use client';

import { AlertTriangle, GitBranch, CheckSquare, BarChart3 } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { useAuth } from '@/hooks/use-auth';
import { API_ENDPOINTS } from '@/lib/constants';
import { KpiCard } from './kpi-card';

interface AlertCount { count: number; trend?: number }
interface PipelineCount { count: number }
interface TaskCount { pending: number; overdue: number }
interface QualityScore { score: number; trend?: number }

export function KpiGrid() {
  const { hasPermission } = useAuth();
  const hasCyber = hasPermission('cyber:read');
  const hasData = hasPermission('data:read');

  const { data: alertData, isLoading: alertLoading, isError: alertError } = useQuery({
    queryKey: ['kpi', 'alerts'],
    queryFn: () => apiGet<AlertCount>(`${API_ENDPOINTS.CYBER_ALERTS_COUNT}?status=new,acknowledged`),
    refetchInterval: 30000,
    enabled: hasCyber,
  });

  const { data: pipelineData, isLoading: pipelineLoading, isError: pipelineError } = useQuery({
    queryKey: ['kpi', 'pipelines'],
    queryFn: () => apiGet<PipelineCount>('/api/v1/data/pipelines/count?status=failed'),
    refetchInterval: 60000,
    enabled: hasData,
  });

  const { data: taskData, isLoading: taskLoading, isError: taskError } = useQuery({
    queryKey: ['kpi', 'tasks'],
    queryFn: () => apiGet<TaskCount>(API_ENDPOINTS.WORKFLOWS_TASKS_COUNT),
    refetchInterval: 30000,
  });

  const { data: qualityData, isLoading: qualityLoading, isError: qualityError } = useQuery({
    queryKey: ['kpi', 'quality'],
    queryFn: () => apiGet<QualityScore>(API_ENDPOINTS.DATA_QUALITY_SCORE),
    refetchInterval: 60000,
    enabled: hasData,
  });

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {hasCyber && (
        <KpiCard
          title="Open Alerts"
          value={alertData?.count}
          icon={AlertTriangle}
          iconColor="text-destructive"
          href="/cyber/alerts"
          isLoading={alertLoading}
          isError={alertError}
          trend={
            alertData?.trend !== undefined
              ? { value: alertData.trend, label: '24h', direction: alertData.trend > 0 ? 'up' : alertData.trend < 0 ? 'down' : 'neutral', sentiment: alertData.trend > 0 ? 'bad' : 'good' }
              : undefined
          }
        />
      )}
      {hasData && (
        <KpiCard
          title="Failed Pipelines"
          value={pipelineData?.count}
          icon={GitBranch}
          iconColor="text-orange-500"
          href="/data/pipelines"
          isLoading={pipelineLoading}
          isError={pipelineError}
        />
      )}
      <KpiCard
        title="Pending Tasks"
        value={taskData?.pending}
        icon={CheckSquare}
        iconColor="text-blue-500"
        href="/workflows/tasks"
        isLoading={taskLoading}
        isError={taskError}
        trend={
          taskData?.overdue !== undefined && taskData.overdue > 0
            ? { value: taskData.overdue, label: 'overdue', direction: 'up', sentiment: 'bad' }
            : undefined
        }
      />
      {hasData && (
        <KpiCard
          title="Data Quality"
          value={qualityData?.score !== undefined ? qualityData.score.toFixed(1) : undefined}
          unit="%"
          icon={BarChart3}
          iconColor="text-green-600"
          href="/data/quality"
          isLoading={qualityLoading}
          isError={qualityError}
          trend={
            qualityData?.trend !== undefined
              ? { value: qualityData.trend, label: '7d', direction: qualityData.trend > 0 ? 'up' : qualityData.trend < 0 ? 'down' : 'neutral', sentiment: qualityData.trend >= 0 ? 'good' : 'bad' }
              : undefined
          }
        />
      )}
    </div>
  );
}
