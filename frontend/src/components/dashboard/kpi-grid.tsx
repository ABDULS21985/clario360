'use client';

import { useEffect, useRef, useState, useMemo } from 'react';
import { AlertTriangle, GitBranch, CheckSquare, BarChart3 } from 'lucide-react';
import { useAuth } from '@/hooks/use-auth';
import { API_ENDPOINTS } from '@/lib/constants';
import { KpiCard } from './kpi-card';
import { SparkLine } from './spark-line';
import { useRealtimeData } from '@/hooks/use-realtime-data';

interface AlertCount { count: number; trend?: number; history?: number[] }
interface PipelineCount { count: number; history?: number[] }
interface TaskCount { pending: number; overdue: number; history?: number[] }
interface QualityScore { score: number; trend?: number; history?: number[] }

export function KpiGrid() {
  const { hasPermission } = useAuth();
  const hasCyber = hasPermission('cyber:read');
  const hasData = hasPermission('data:read');

  const { data: alertData, isLoading: alertLoading, error: alertError, lastUpdate: alertUpdate } =
    useRealtimeData<AlertCount>(API_ENDPOINTS.CYBER_ALERTS_COUNT, {
      params: { status: 'new,acknowledged' },
      wsTopics: ['alert.created', 'alert.escalated', 'alert.resolved'],
      enabled: hasCyber,
    });

  const {
    data: pipelineData,
    isLoading: pipelineLoading,
    error: pipelineError,
    lastUpdate: pipelineUpdate,
  } = useRealtimeData<PipelineCount>('/api/v1/data/pipelines/count', {
    params: { status: 'failed' },
    wsTopics: ['pipeline.failed', 'pipeline.completed'],
    enabled: hasData,
  });

  const { data: taskData, isLoading: taskLoading, error: taskError, lastUpdate: taskUpdate } =
    useRealtimeData<TaskCount>(API_ENDPOINTS.WORKFLOWS_TASKS_COUNT, {
      wsTopics: [
        'task.assigned',
        'task.completed',
        'task.escalated',
        'workflow.task.created',
        'workflow.task.completed',
        'workflow.task.escalated',
      ],
    });

  const {
    data: qualityData,
    isLoading: qualityLoading,
    error: qualityError,
    lastUpdate: qualityUpdate,
  } = useRealtimeData<QualityScore>(API_ENDPOINTS.DATA_QUALITY_SCORE, {
    wsTopics: ['data_quality.issue_detected'],
    enabled: hasData,
  });

  const alertDelta = useLiveDelta(alertData?.count);
  const pipelineDelta = useLiveDelta(pipelineData?.count);
  const taskDelta = useLiveDelta(taskData?.pending);
  const qualityDelta = useLiveDelta(qualityData?.score);

  // Build sparkline history from data (use provided history or build from live deltas)
  const alertHistory = useSparkHistory(alertData?.count, alertData?.history);
  const pipelineHistory = useSparkHistory(pipelineData?.count, pipelineData?.history);
  const taskHistory = useSparkHistory(taskData?.pending, taskData?.history);
  const qualityHistory = useSparkHistory(qualityData?.score, qualityData?.history);

  let cardIndex = 0;

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
          isError={Boolean(alertError)}
          highlightKey={alertUpdate?.getTime() ?? null}
          liveDelta={alertDelta}
          index={cardIndex++}
          trend={
            alertData?.trend !== undefined
              ? { value: alertData.trend, label: '24h', direction: alertData.trend > 0 ? 'up' : alertData.trend < 0 ? 'down' : 'neutral', sentiment: alertData.trend > 0 ? 'bad' : 'good' }
              : undefined
          }
        >
          {alertHistory.length >= 2 && (
            <SparkLine data={alertHistory} color="#EF4444" />
          )}
        </KpiCard>
      )}
      {hasData && (
        <KpiCard
          title="Failed Pipelines"
          value={pipelineData?.count}
          icon={GitBranch}
          iconColor="text-orange-500"
          href="/data/pipelines"
          isLoading={pipelineLoading}
          isError={Boolean(pipelineError)}
          highlightKey={pipelineUpdate?.getTime() ?? null}
          liveDelta={pipelineDelta}
          index={cardIndex++}
        >
          {pipelineHistory.length >= 2 && (
            <SparkLine data={pipelineHistory} color="#F97316" />
          )}
        </KpiCard>
      )}
      <KpiCard
        title="Pending Tasks"
        value={taskData?.pending}
        icon={CheckSquare}
        iconColor="text-blue-500"
        href="/workflows/tasks"
        isLoading={taskLoading}
        isError={Boolean(taskError)}
        highlightKey={taskUpdate?.getTime() ?? null}
        liveDelta={taskDelta}
        index={cardIndex++}
        trend={
          taskData?.overdue !== undefined && taskData.overdue > 0
            ? { value: taskData.overdue, label: 'overdue', direction: 'up', sentiment: 'bad' }
            : undefined
        }
      >
        {taskHistory.length >= 2 && (
          <SparkLine data={taskHistory} color="#3B82F6" />
        )}
      </KpiCard>
      {hasData && (
        <KpiCard
          title="Data Quality"
          value={qualityData?.score !== undefined ? qualityData.score.toFixed(1) : undefined}
          unit="%"
          icon={BarChart3}
          iconColor="text-green-600"
          href="/data/quality"
          isLoading={qualityLoading}
          isError={Boolean(qualityError)}
          highlightKey={qualityUpdate?.getTime() ?? null}
          liveDelta={qualityDelta}
          index={cardIndex++}
          trend={
            qualityData?.trend !== undefined
              ? { value: qualityData.trend, label: '7d', direction: qualityData.trend > 0 ? 'up' : qualityData.trend < 0 ? 'down' : 'neutral', sentiment: qualityData.trend >= 0 ? 'good' : 'bad' }
              : undefined
          }
        >
          {qualityHistory.length >= 2 && (
            <SparkLine data={qualityHistory} color="#22C55E" />
          )}
        </KpiCard>
      )}
    </div>
  );
}

function useLiveDelta(value: number | undefined): number | null {
  const previousValue = useRef<number | undefined>(value);
  const [delta, setDelta] = useState<number | null>(null);

  useEffect(() => {
    if (value === undefined || previousValue.current === undefined) {
      previousValue.current = value;
      return;
    }

    const nextDelta = value - previousValue.current;
    previousValue.current = value;
    if (nextDelta !== 0) {
      setDelta(nextDelta);
      const timeout = window.setTimeout(() => setDelta(null), 3000);
      return () => window.clearTimeout(timeout);
    }
  }, [value]);

  return delta;
}

/** Builds a rolling sparkline history from live value changes or initial history array */
function useSparkHistory(currentValue: number | undefined, serverHistory?: number[]): number[] {
  const historyRef = useRef<number[]>([]);
  const [history, setHistory] = useState<number[]>([]);

  useEffect(() => {
    // If server provides history, use it as the base
    if (serverHistory && serverHistory.length > 0) {
      historyRef.current = [...serverHistory];
      setHistory(historyRef.current);
      return;
    }

    // Otherwise build incrementally from live values
    if (currentValue !== undefined) {
      const next = [...historyRef.current, currentValue].slice(-12);
      historyRef.current = next;
      setHistory(next);
    }
  }, [currentValue, serverHistory]);

  return history;
}
